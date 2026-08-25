package logic

import (
	"context"
	"strconv"
	"time"

	"github.com/chihqiang/infra-go/logger"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// K8sJobItem Job 列表项。
type K8sJobItem struct {
	Name        string `json:"name"`
	Namespace   string `json:"namespace"`
	Completions string `json:"completions"` // 如 "1/1"
	Duration    string `json:"duration"`   // 执行时长
	Status      string `json:"status"`     // Running / Complete / Failed
	Parallelism int32  `json:"parallelism"`
	Image       string `json:"image"`
	CreatedAt   string `json:"created_at"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// K8sCronJobItem CronJob 列表项。
type K8sCronJobItem struct {
	Name        string `json:"name"`
	Namespace   string `json:"namespace"`
	Schedule    string `json:"schedule"`
	Suspend     bool   `json:"suspend"`
	Active      int32  `json:"active"`
	LastSchedule string `json:"last_schedule"`
	Image       string `json:"image"`
	CreatedAt   string `json:"created_at"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// ListJobs Job 列表。
func (l *K8sLogic) ListJobs(ctx context.Context, namespace string) ([]K8sJobItem, error) {
	return l.ListJobsWithOptions(ctx, K8sListOptions{Namespace: namespace})
}

// ListJobsWithOptions Job 列表（支持标签/字段过滤）。
func (l *K8sLogic) ListJobsWithOptions(ctx context.Context, opts K8sListOptions) ([]K8sJobItem, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	ns := l.resolveNamespace(opts.Namespace)

	jobList, err := cli.BatchV1().Jobs(ns).List(ctx, opts.toListOptions())
	if err != nil {
		return nil, err
	}

	items := make([]K8sJobItem, 0, len(jobList.Items))
	for i := range jobList.Items {
		items = append(items, toK8sJobItem(&jobList.Items[i]))
	}
	return items, nil
}

// InspectJob Job 详情（原始对象）。
func (l *K8sLogic) InspectJob(ctx context.Context, namespace, name string) (*batchv1.Job, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	ns := l.namespace()
	if namespace != "" {
		ns = namespace
	}
	return cli.BatchV1().Jobs(ns).Get(ctx, name, metav1.GetOptions{})
}

// DeleteJob 删除 Job。
func (l *K8sLogic) DeleteJob(ctx context.Context, namespace, name string) error {
	cli, err := l.newClient()
	if err != nil {
		return err
	}
	ns := l.namespace()
	if namespace != "" {
		ns = namespace
	}
	err = cli.BatchV1().Jobs(ns).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		logger.ErrorCtx(ctx, "k8s delete job failed", logger.String("ns", ns), logger.String("name", name), logger.Err(err))
	}
	return err
}

// ListCronJobs CronJob 列表。
func (l *K8sLogic) ListCronJobs(ctx context.Context, namespace string) ([]K8sCronJobItem, error) {
	return l.ListCronJobsWithOptions(ctx, K8sListOptions{Namespace: namespace})
}

// ListCronJobsWithOptions CronJob 列表（支持标签/字段过滤）。
func (l *K8sLogic) ListCronJobsWithOptions(ctx context.Context, opts K8sListOptions) ([]K8sCronJobItem, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	ns := l.resolveNamespace(opts.Namespace)

	cronJobList, err := cli.BatchV1().CronJobs(ns).List(ctx, opts.toListOptions())
	if err != nil {
		return nil, err
	}

	items := make([]K8sCronJobItem, 0, len(cronJobList.Items))
	for i := range cronJobList.Items {
		items = append(items, toK8sCronJobItem(&cronJobList.Items[i]))
	}
	return items, nil
}

// InspectCronJob CronJob 详情（原始对象）。
func (l *K8sLogic) InspectCronJob(ctx context.Context, namespace, name string) (*batchv1.CronJob, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	ns := l.namespace()
	if namespace != "" {
		ns = namespace
	}
	return cli.BatchV1().CronJobs(ns).Get(ctx, name, metav1.GetOptions{})
}

// DeleteCronJob 删除 CronJob。
func (l *K8sLogic) DeleteCronJob(ctx context.Context, namespace, name string) error {
	cli, err := l.newClient()
	if err != nil {
		return err
	}
	ns := l.namespace()
	if namespace != "" {
		ns = namespace
	}
	err = cli.BatchV1().CronJobs(ns).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		logger.ErrorCtx(ctx, "k8s delete cronjob failed", logger.String("ns", ns), logger.String("name", name), logger.Err(err))
	}
	return err
}

// toK8sJobItem batchv1.Job → 列表项。
func toK8sJobItem(job *batchv1.Job) K8sJobItem {
	item := K8sJobItem{
		Name:        job.Name,
		Namespace:   job.Namespace,
		Completions: "",
		Duration:    "",
		Status:      "Running",
		Parallelism: 0,
		Image:       "",
		CreatedAt:   job.CreationTimestamp.Format("2006-01-02 15:04:05"),
		Labels:      job.Labels,
	}

	if job.Spec.Parallelism != nil {
		item.Parallelism = *job.Spec.Parallelism
	}

	// 完成度。
	succeeded := job.Status.Succeeded
	desired := int32(0)
	if job.Spec.Completions != nil {
		desired = *job.Spec.Completions
	}
	if desired > 0 {
		item.Completions = formatInt(succeeded) + "/" + formatInt(desired)
	} else {
		item.Completions = formatInt(succeeded) + "/1"
	}

	// 状态判定。
	if len(job.Status.Conditions) > 0 {
		for _, cond := range job.Status.Conditions {
			if cond.Status == "True" {
				if cond.Type == batchv1.JobComplete {
					item.Status = "Complete"
					break
				}
				if cond.Type == batchv1.JobFailed {
					item.Status = "Failed"
					break
				}
			}
		}
	}

	// 执行时长。
	if job.Status.StartTime != nil && job.Status.CompletionTime != nil {
		dur := job.Status.CompletionTime.Sub(job.Status.StartTime.Time)
		item.Duration = dur.Truncate(time.Second).String()
	} else if job.Status.StartTime != nil {
		dur := time.Since(job.Status.StartTime.Time)
		item.Duration = dur.Truncate(time.Second).String()
	}

	// 主镜像。
	if len(job.Spec.Template.Spec.Containers) > 0 {
		item.Image = job.Spec.Template.Spec.Containers[0].Image
	}

	return item
}

// toK8sCronJobItem batchv1.CronJob → 列表项。
func toK8sCronJobItem(cj *batchv1.CronJob) K8sCronJobItem {
	item := K8sCronJobItem{
		Name:         cj.Name,
		Namespace:    cj.Namespace,
		Schedule:     cj.Spec.Schedule,
		Suspend:      false,
		Active:       int32(len(cj.Status.Active)),
		LastSchedule: "",
		Image:        "",
		CreatedAt:    cj.CreationTimestamp.Format("2006-01-02 15:04:05"),
		Labels:       cj.Labels,
	}

	if cj.Spec.Suspend != nil {
		item.Suspend = *cj.Spec.Suspend
	}

	if cj.Status.LastScheduleTime != nil {
		item.LastSchedule = cj.Status.LastScheduleTime.Format("2006-01-02 15:04:05")
	}

	// 主镜像。
	if len(cj.Spec.JobTemplate.Spec.Template.Spec.Containers) > 0 {
		item.Image = cj.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image
	}

	return item
}

// formatInt int32 → string。
func formatInt(n int32) string {
	return strconv.FormatInt(int64(n), 10)
}
