package logic

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/chihqiang/infra-go/logger"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// K8sDeploymentItem Deployment 列表项。
type K8sDeploymentItem struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Ready     string            `json:"ready"` // 如 "2/3"
	UpToDate  int32             `json:"up_to_date"`
	Available int32             `json:"available"`
	Replicas  int32             `json:"replicas"`
	Desired   int32             `json:"desired"`
	Image     string            `json:"image"`
	CreatedAt string            `json:"created_at"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// ListDeployments Deployment 列表。
func (l *K8sLogic) ListDeployments(ctx context.Context, namespace string) ([]K8sDeploymentItem, error) {
	return l.ListDeploymentsWithOptions(ctx, K8sListOptions{Namespace: namespace})
}

// ListDeploymentsWithOptions Deployment 列表（支持标签/字段过滤）。
func (l *K8sLogic) ListDeploymentsWithOptions(ctx context.Context, opts K8sListOptions) ([]K8sDeploymentItem, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	ns := l.resolveNamespace(opts.Namespace)

	depList, err := cli.AppsV1().Deployments(ns).List(ctx, opts.toListOptions())
	if err != nil {
		return nil, err
	}

	items := make([]K8sDeploymentItem, 0, len(depList.Items))
	for i := range depList.Items {
		items = append(items, toK8sDeploymentItem(&depList.Items[i]))
	}
	return items, nil
}

// InspectDeployment Deployment 详情（原始对象）。
func (l *K8sLogic) InspectDeployment(ctx context.Context, namespace, name string) (*appsv1.Deployment, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	ns := l.namespace()
	if namespace != "" {
		ns = namespace
	}
	return cli.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
}

// DeleteDeployment 删除 Deployment。
func (l *K8sLogic) DeleteDeployment(ctx context.Context, namespace, name string) error {
	cli, err := l.newClient()
	if err != nil {
		return err
	}
	ns := l.namespace()
	if namespace != "" {
		ns = namespace
	}
	err = cli.AppsV1().Deployments(ns).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		logger.ErrorCtx(ctx, "k8s delete deployment failed", logger.String("ns", ns), logger.String("name", name), logger.Err(err))
	}
	return err
}

// ScaleDeployment 伸缩 Deployment 副本数。
func (l *K8sLogic) ScaleDeployment(ctx context.Context, namespace, name string, replicas int32) error {
	cli, err := l.newClient()
	if err != nil {
		return err
	}
	ns := l.namespace()
	if namespace != "" {
		ns = namespace
	}

	_, err = cli.AppsV1().Deployments(ns).UpdateScale(ctx, name, &autoscalingv1.Scale{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       autoscalingv1.ScaleSpec{Replicas: replicas},
	}, metav1.UpdateOptions{})
	if err != nil {
		logger.ErrorCtx(ctx, "k8s scale deployment failed", logger.String("ns", ns), logger.String("name", name), logger.Int("replicas", int(replicas)), logger.Err(err))
	}
	return err
}

// RestartDeployment 重启 Deployment（通过注入 restart 注解触发滚动更新）。
func (l *K8sLogic) RestartDeployment(ctx context.Context, namespace, name string) error {
	cli, err := l.newClient()
	if err != nil {
		return err
	}
	ns := l.namespace()
	if namespace != "" {
		ns = namespace
	}

	patch := map[string]any{
		"spec": map[string]any{
			"template": map[string]any{
				"metadata": map[string]any{
					"annotations": map[string]any{
						"kubectl.kubernetes.io/restartedAt": metav1.Now().Format("2006-01-02T15:04:05Z07:00"),
					},
				},
			},
		},
	}
	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return err
	}

	_, err = cli.AppsV1().Deployments(ns).Patch(ctx, name, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		logger.ErrorCtx(ctx, "k8s restart deployment failed", logger.String("ns", ns), logger.String("name", name), logger.Err(err))
	}
	return err
}

// K8sStatefulSetItem StatefulSet 列表项。
type K8sStatefulSetItem struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Ready     string            `json:"ready"` // 如 "2/3"
	Replicas  int32             `json:"replicas"`
	Image     string            `json:"image"`
	CreatedAt string            `json:"created_at"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// ListStatefulSets StatefulSet 列表。
func (l *K8sLogic) ListStatefulSets(ctx context.Context, namespace string) ([]K8sStatefulSetItem, error) {
	return l.ListStatefulSetsWithOptions(ctx, K8sListOptions{Namespace: namespace})
}

// ListStatefulSetsWithOptions StatefulSet 列表（支持标签/字段过滤）。
func (l *K8sLogic) ListStatefulSetsWithOptions(ctx context.Context, opts K8sListOptions) ([]K8sStatefulSetItem, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	ns := l.resolveNamespace(opts.Namespace)

	stsList, err := cli.AppsV1().StatefulSets(ns).List(ctx, opts.toListOptions())
	if err != nil {
		return nil, err
	}

	items := make([]K8sStatefulSetItem, 0, len(stsList.Items))
	for i := range stsList.Items {
		sts := &stsList.Items[i]
		item := K8sStatefulSetItem{
			Name:      sts.Name,
			Namespace: sts.Namespace,
			Ready:     fmt.Sprintf("%d/%d", sts.Status.ReadyReplicas, sts.Status.Replicas),
			Replicas:  sts.Status.Replicas,
			CreatedAt: sts.CreationTimestamp.Format("2006-01-02 15:04:05"),
			Labels:    sts.Labels,
		}
		if len(sts.Spec.Template.Spec.Containers) > 0 {
			item.Image = sts.Spec.Template.Spec.Containers[0].Image
		}
		items = append(items, item)
	}
	return items, nil
}

// InspectStatefulSet StatefulSet 详情。
func (l *K8sLogic) InspectStatefulSet(ctx context.Context, namespace, name string) (*appsv1.StatefulSet, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	ns := l.namespace()
	if namespace != "" {
		ns = namespace
	}
	return cli.AppsV1().StatefulSets(ns).Get(ctx, name, metav1.GetOptions{})
}

// DeleteStatefulSet 删除 StatefulSet。
func (l *K8sLogic) DeleteStatefulSet(ctx context.Context, namespace, name string) error {
	cli, err := l.newClient()
	if err != nil {
		return err
	}
	ns := l.namespace()
	if namespace != "" {
		ns = namespace
	}
	return cli.AppsV1().StatefulSets(ns).Delete(ctx, name, metav1.DeleteOptions{})
}

// ScaleStatefulSet 伸缩 StatefulSet 副本数。
func (l *K8sLogic) ScaleStatefulSet(ctx context.Context, namespace, name string, replicas int32) error {
	cli, err := l.newClient()
	if err != nil {
		return err
	}
	ns := l.namespace()
	if namespace != "" {
		ns = namespace
	}
	_, err = cli.AppsV1().StatefulSets(ns).UpdateScale(ctx, name, &autoscalingv1.Scale{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       autoscalingv1.ScaleSpec{Replicas: replicas},
	}, metav1.UpdateOptions{})
	if err != nil {
		logger.ErrorCtx(ctx, "k8s scale statefulset failed",
			logger.String("ns", ns), logger.String("name", name), logger.Int("replicas", int(replicas)), logger.Err(err))
	}
	return err
}

// RestartStatefulSet 重启 StatefulSet（注入 restart 注解）。
func (l *K8sLogic) RestartStatefulSet(ctx context.Context, namespace, name string) error {
	cli, err := l.newClient()
	if err != nil {
		return err
	}
	ns := l.namespace()
	if namespace != "" {
		ns = namespace
	}
	patch := map[string]any{
		"spec": map[string]any{
			"template": map[string]any{
				"metadata": map[string]any{
					"annotations": map[string]any{
						"kubectl.kubernetes.io/restartedAt": metav1.Now().Format("2006-01-02T15:04:05Z07:00"),
					},
				},
			},
		},
	}
	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return err
	}
	_, err = cli.AppsV1().StatefulSets(ns).Patch(ctx, name, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		logger.ErrorCtx(ctx, "k8s restart statefulset failed", logger.String("ns", ns), logger.String("name", name), logger.Err(err))
	}
	return err
}

// K8sDaemonSetItem DaemonSet 列表项。
type K8sDaemonSetItem struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Desired   int32             `json:"desired"`
	Current   int32             `json:"current"`
	Ready     int32             `json:"ready"`
	Available int32             `json:"available"`
	Image     string            `json:"image"`
	CreatedAt string            `json:"created_at"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// ListDaemonSets DaemonSet 列表。
func (l *K8sLogic) ListDaemonSets(ctx context.Context, namespace string) ([]K8sDaemonSetItem, error) {
	return l.ListDaemonSetsWithOptions(ctx, K8sListOptions{Namespace: namespace})
}

// ListDaemonSetsWithOptions DaemonSet 列表（支持标签/字段过滤）。
func (l *K8sLogic) ListDaemonSetsWithOptions(ctx context.Context, opts K8sListOptions) ([]K8sDaemonSetItem, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	ns := l.resolveNamespace(opts.Namespace)

	dsList, err := cli.AppsV1().DaemonSets(ns).List(ctx, opts.toListOptions())
	if err != nil {
		return nil, err
	}

	items := make([]K8sDaemonSetItem, 0, len(dsList.Items))
	for i := range dsList.Items {
		ds := &dsList.Items[i]
		item := K8sDaemonSetItem{
			Name:      ds.Name,
			Namespace: ds.Namespace,
			Desired:   ds.Status.DesiredNumberScheduled,
			Current:   ds.Status.CurrentNumberScheduled,
			Ready:     ds.Status.NumberReady,
			Available: ds.Status.NumberAvailable,
			CreatedAt: ds.CreationTimestamp.Format("2006-01-02 15:04:05"),
			Labels:    ds.Labels,
		}
		if len(ds.Spec.Template.Spec.Containers) > 0 {
			item.Image = ds.Spec.Template.Spec.Containers[0].Image
		}
		items = append(items, item)
	}
	return items, nil
}

// InspectDaemonSet DaemonSet 详情。
func (l *K8sLogic) InspectDaemonSet(ctx context.Context, namespace, name string) (*appsv1.DaemonSet, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	ns := l.namespace()
	if namespace != "" {
		ns = namespace
	}
	return cli.AppsV1().DaemonSets(ns).Get(ctx, name, metav1.GetOptions{})
}

// DeleteDaemonSet 删除 DaemonSet。
func (l *K8sLogic) DeleteDaemonSet(ctx context.Context, namespace, name string) error {
	cli, err := l.newClient()
	if err != nil {
		return err
	}
	ns := l.namespace()
	if namespace != "" {
		ns = namespace
	}
	return cli.AppsV1().DaemonSets(ns).Delete(ctx, name, metav1.DeleteOptions{})
}

// RestartDaemonSet 重启 DaemonSet（注入 restart 注解）。
func (l *K8sLogic) RestartDaemonSet(ctx context.Context, namespace, name string) error {
	cli, err := l.newClient()
	if err != nil {
		return err
	}
	ns := l.namespace()
	if namespace != "" {
		ns = namespace
	}
	patch := map[string]any{
		"spec": map[string]any{
			"template": map[string]any{
				"metadata": map[string]any{
					"annotations": map[string]any{
						"kubectl.kubernetes.io/restartedAt": metav1.Now().Format("2006-01-02T15:04:05Z07:00"),
					},
				},
			},
		},
	}
	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return err
	}
	_, err = cli.AppsV1().DaemonSets(ns).Patch(ctx, name, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		logger.ErrorCtx(ctx, "k8s restart daemonset failed", logger.String("ns", ns), logger.String("name", name), logger.Err(err))
	}
	return err
}

// toK8sDeploymentItem appsv1.Deployment → 列表项。
func toK8sDeploymentItem(d *appsv1.Deployment) K8sDeploymentItem {
	item := K8sDeploymentItem{
		Name:      d.Name,
		Namespace: d.Namespace,
		Ready:     fmt.Sprintf("%d/%d", d.Status.ReadyReplicas, d.Status.Replicas),
		Replicas:  d.Status.Replicas,
		Desired:   *d.Spec.Replicas,
		UpToDate:  d.Status.UpdatedReplicas,
		Available: d.Status.AvailableReplicas,
		CreatedAt: d.CreationTimestamp.Format("2006-01-02 15:04:05"),
		Labels:    d.Labels,
	}
	if len(d.Spec.Template.Spec.Containers) > 0 {
		item.Image = d.Spec.Template.Spec.Containers[0].Image
	}
	return item
}
