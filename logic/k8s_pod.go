package logic

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"

	"github.com/chihqiang/infra-go/logger"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

// K8sPodItem Pod 列表项。
type K8sPodItem struct {
	Name       string             `json:"name"`
	Namespace  string             `json:"namespace"`
	Status     string             `json:"status"` // Pending / Running / Succeeded / Failed / Unknown
	Ready      string             `json:"ready"`  // 如 "1/2"
	Restarts   int32              `json:"restarts"`
	NodeName   string             `json:"node_name"`
	IP         string             `json:"ip"`
	Image      string             `json:"image"`
	CreatedAt  string             `json:"created_at"`
	Labels     map[string]string  `json:"labels,omitempty"`
	QoSClass   string             `json:"qos_class,omitempty"`
	Containers []K8sContainerItem `json:"containers,omitempty"`
}

// K8sContainerItem 容器信息。
type K8sContainerItem struct {
	Name     string `json:"name"`
	Image    string `json:"image"`
	State    string `json:"state"` // running / waiting / terminated
	Ready    bool   `json:"ready"`
	Restarts int32  `json:"restarts"`
	Reason   string `json:"reason,omitempty"`
}

// ListPods Pod 列表；namespace 为空时使用默认。
func (l *K8sLogic) ListPods(ctx context.Context, namespace string) ([]K8sPodItem, error) {
	return l.ListPodsWithOptions(ctx, K8sListOptions{Namespace: namespace})
}

// ListPodsWithOptions Pod 列表（支持标签/字段过滤）。
func (l *K8sLogic) ListPodsWithOptions(ctx context.Context, opts K8sListOptions) ([]K8sPodItem, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	ns := l.namespace()
	if opts.Namespace != "" {
		ns = opts.Namespace
	}

	podList, err := cli.CoreV1().Pods(ns).List(ctx, opts.toListOptions())
	if err != nil {
		return nil, err
	}

	items := make([]K8sPodItem, 0, len(podList.Items))
	for i := range podList.Items {
		items = append(items, toK8sPodItem(&podList.Items[i]))
	}
	return items, nil
}

// InspectPod Pod 详情（原始对象）。
func (l *K8sLogic) InspectPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	ns := l.namespace()
	if namespace != "" {
		ns = namespace
	}
	return cli.CoreV1().Pods(ns).Get(ctx, name, metav1.GetOptions{})
}

// DeletePod 删除 Pod。
func (l *K8sLogic) DeletePod(ctx context.Context, namespace, name string) error {
	cli, err := l.newClient()
	if err != nil {
		return err
	}
	ns := l.namespace()
	if namespace != "" {
		ns = namespace
	}
	err = cli.CoreV1().Pods(ns).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		logger.ErrorCtx(ctx, "k8s delete pod failed", logger.String("ns", ns), logger.String("name", name), logger.Err(err))
	}
	return err
}

// PodLogsOptions Pod 日志选项。
type PodLogsOptions struct {
	Namespace    string
	Name         string
	Container    string // 容器名（单容器时可空）
	Follow       bool
	TailLines    *int64
	SinceSeconds *int64       // 相对时间（秒），仅返回 N 秒内的日志
	SinceTime    *metav1.Time // 绝对时间（RFC3339），仅返回此时间之后的日志
	Previous     bool
	Timestamps   bool
}

// StreamPodLogs 获取 Pod 日志流（调用方负责关闭）。
func (l *K8sLogic) StreamPodLogs(ctx context.Context, opts PodLogsOptions) (io.ReadCloser, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	ns := l.namespace()
	if opts.Namespace != "" {
		ns = opts.Namespace
	}

	podLogOpts := &corev1.PodLogOptions{
		Container:  opts.Container,
		Follow:     opts.Follow,
		Previous:   opts.Previous,
		Timestamps: opts.Timestamps,
	}
	if opts.TailLines != nil {
		podLogOpts.TailLines = opts.TailLines
	}
	if opts.SinceSeconds != nil {
		podLogOpts.SinceSeconds = opts.SinceSeconds
	}
	if opts.SinceTime != nil {
		podLogOpts.SinceTime = opts.SinceTime
	}

	req := cli.CoreV1().Pods(ns).GetLogs(opts.Name, podLogOpts)
	stream, err := req.Stream(ctx)
	if err != nil {
		return nil, err
	}
	return stream, nil
}

// PodExecOptions Pod exec 选项。
type PodExecOptions struct {
	Namespace string
	Name      string
	Container string
	Command   []string
	TTY       bool
}

// PodExecResult exec 执行结果。
type PodExecResult struct {
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr,omitempty"`
}

// ExecPod 在 Pod 中执行命令（一次性，非交互式）。
func (l *K8sLogic) ExecPod(ctx context.Context, opts PodExecOptions) (*PodExecResult, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	ns := l.namespace()
	if opts.Namespace != "" {
		ns = opts.Namespace
	}

	req := cli.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(ns).
		Name(opts.Name).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: opts.Container,
			Command:   opts.Command,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       opts.TTY,
		}, scheme.ParameterCodec)

	restCfg, err := l.restConfig()
	if err != nil {
		return nil, err
	}
	executor, err := remotecommand.NewSPDYExecutor(restCfg, "POST", req.URL())
	if err != nil {
		return nil, err
	}

	var stdout, stderr bytes.Buffer
	err = executor.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	result := &PodExecResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}
	if err != nil {
		return result, fmt.Errorf("exec failed: %w", err)
	}
	return result, nil
}

// toK8sPodItem corev1.Pod → 列表项。
func toK8sPodItem(p *corev1.Pod) K8sPodItem {
	item := K8sPodItem{
		Name:      p.Name,
		Namespace: p.Namespace,
		Status:    string(p.Status.Phase),
		NodeName:  p.Spec.NodeName,
		IP:        p.Status.PodIP,
		CreatedAt: p.CreationTimestamp.Format("2006-01-02 15:04:05"),
		Labels:    p.Labels,
		QoSClass:  string(p.Status.QOSClass),
	}

	// 容器信息。
	totalContainers := len(p.Spec.Containers)
	readyContainers := 0
	totalRestarts := int32(0)

	for _, cs := range p.Status.ContainerStatuses {
		item.Containers = append(item.Containers, K8sContainerItem{
			Name:     cs.Name,
			Image:    cs.Image,
			State:    getContainerState(cs.State),
			Ready:    cs.Ready,
			Restarts: cs.RestartCount,
			Reason:   getContainerReason(cs.State),
		})
		totalRestarts += cs.RestartCount
		if cs.Ready {
			readyContainers++
		}
	}

	item.Restarts = totalRestarts
	item.Ready = fmt.Sprintf("%d/%d", readyContainers, totalContainers)

	// 主镜像（取第一个容器）。
	if len(p.Spec.Containers) > 0 {
		item.Image = p.Spec.Containers[0].Image
	}

	return item
}

// getContainerState 获取容器状态字符串。
func getContainerState(s corev1.ContainerState) string {
	switch {
	case s.Running != nil:
		return "running"
	case s.Waiting != nil:
		return "waiting"
	case s.Terminated != nil:
		return "terminated"
	default:
		return "unknown"
	}
}

// getContainerReason 获取容器状态原因。
func getContainerReason(s corev1.ContainerState) string {
	switch {
	case s.Waiting != nil:
		return s.Waiting.Reason
	case s.Terminated != nil:
		return s.Terminated.Reason
	default:
		return ""
	}
}

// Ensure url import is used (URL method returns *url.URL).
var _ = (*url.URL)(nil)
