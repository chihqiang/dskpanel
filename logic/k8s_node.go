package logic

import (
	"context"
	"fmt"

	"github.com/chihqiang/infra-go/logger"

	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// K8sNodeItem 节点列表项。
type K8sNodeItem struct {
	Name             string            `json:"name"`
	Role             string            `json:"role"` // master / worker
	Ready            bool              `json:"ready"`
	Status           string            `json:"status"`  // Ready / NotReady
	Version          string            `json:"version"` // kubelet 版本
	OS               string            `json:"os"`
	Arch             string            `json:"arch"`
	KernelVersion    string            `json:"kernel_version"`
	ContainerRuntime string            `json:"container_runtime"`
	InternalIP       string            `json:"internal_ip"`
	ExternalIP       string            `json:"external_ip"`
	CPU              string            `json:"cpu"`
	Memory           string            `json:"memory"`
	PodsCapacity     int64             `json:"pods_capacity"`
	Labels           map[string]string `json:"labels"`
	Taints           []K8sTaint        `json:"taints,omitempty"`
	CreatedAt        string            `json:"created_at"`
}

// K8sTaint 污点信息。
type K8sTaint struct {
	Key    string `json:"key"`
	Value  string `json:"value,omitempty"`
	Effect string `json:"effect"`
}

// ListNodes 节点列表（支持标签过滤）。
func (l *K8sLogic) ListNodes(ctx context.Context, opts K8sListOptions) ([]K8sNodeItem, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}

	nodeList, err := cli.CoreV1().Nodes().List(ctx, opts.toListOptions())
	if err != nil {
		return nil, err
	}

	items := make([]K8sNodeItem, 0, len(nodeList.Items))
	for i := range nodeList.Items {
		items = append(items, toK8sNodeItem(&nodeList.Items[i]))
	}
	return items, nil
}

// InspectNode 节点详情（原始对象）。
func (l *K8sLogic) InspectNode(ctx context.Context, name string) (*corev1.Node, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	return cli.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
}

// CordonNode 将节点标记为不可调度（设 Unschedulable=true）。
func (l *K8sLogic) CordonNode(ctx context.Context, name string) error {
	cli, err := l.newClient()
	if err != nil {
		return err
	}
	node, err := cli.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	node.Spec.Unschedulable = true
	_, err = cli.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		logger.ErrorCtx(ctx, "k8s cordon node failed", logger.String("name", name), logger.Err(err))
	}
	return err
}

// UncordonNode 将节点恢复调度（设 Unschedulable=false）。
func (l *K8sLogic) UncordonNode(ctx context.Context, name string) error {
	cli, err := l.newClient()
	if err != nil {
		return err
	}
	node, err := cli.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	node.Spec.Unschedulable = false
	_, err = cli.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		logger.ErrorCtx(ctx, "k8s uncordon node failed", logger.String("name", name), logger.Err(err))
	}
	return err
}

// DrainNode 驱逐节点上的所有 Pod（排除 DaemonSet Pod）。
// 实际 Drain 需逐个 Pod 调用 Eviction API，此处实现简化版。
func (l *K8sLogic) DrainNode(ctx context.Context, name string) error {
	cli, err := l.newClient()
	if err != nil {
		return err
	}
	// 先 Cordon。
	node, err := cli.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	node.Spec.Unschedulable = true
	if _, err = cli.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{}); err != nil {
		return err
	}

	// 列出该节点上所有 Pod（排除 kube-system 中的 DaemonSet）。
	podList, err := cli.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", name),
	})
	if err != nil {
		return err
	}

	gracePeriod := int64(30)
	for i := range podList.Items {
		pod := &podList.Items[i]
		// 跳过 DaemonSet Pod（通过 OwnerReference 判断）。
		if isDaemonSetPod(pod) {
			continue
		}
		// 跳过已终止的 Pod。
		if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
			continue
		}
		deleteOpts := metav1.DeleteOptions{GracePeriodSeconds: &gracePeriod}
		err = cli.CoreV1().Pods(pod.Namespace).EvictV1(ctx, &policyv1.Eviction{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pod.Name,
				Namespace: pod.Namespace,
			},
			DeleteOptions: &deleteOpts,
		})
		if err != nil {
			logger.WarnCtx(ctx, "k8s drain pod failed",
				logger.String("pod", pod.Name), logger.String("ns", pod.Namespace), logger.Err(err))
		}
	}
	return nil
}

// isDaemonSetPod 判断 Pod 是否由 DaemonSet 管理。
func isDaemonSetPod(pod *corev1.Pod) bool {
	for _, ref := range pod.OwnerReferences {
		if ref.Kind == "DaemonSet" {
			return true
		}
	}
	return false
}

// K8sNodeUsage 节点资源使用率。
type K8sNodeUsage struct {
	Name       string `json:"name"`
	CPUUsed    string `json:"cpu_used"`
	CPUTotal   string `json:"cpu_total"`
	CPUPercent string `json:"cpu_percent"`
	MemUsed    string `json:"mem_used"`
	MemTotal   string `json:"mem_total"`
	MemPercent string `json:"mem_percent"`
	PodsUsed   int    `json:"pods_used"`
	PodsTotal  int64  `json:"pods_total"`
}

// NodeUsage 获取节点资源使用率（基于 Pod requests 聚合估算）。
func (l *K8sLogic) NodeUsage(ctx context.Context, name string) (*K8sNodeUsage, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	node, err := cli.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	usage := &K8sNodeUsage{Name: name}

	// 可分配资源。
	allocatableCPU := resource.Quantity{}
	allocatableMem := resource.Quantity{}
	if v, ok := node.Status.Allocatable[corev1.ResourceCPU]; ok {
		allocatableCPU = v
		usage.CPUTotal = v.String()
	}
	if v, ok := node.Status.Allocatable[corev1.ResourceMemory]; ok {
		allocatableMem = v
		usage.MemTotal = formatResource(v)
	}
	if v, ok := node.Status.Allocatable[corev1.ResourcePods]; ok {
		usage.PodsTotal = v.Value()
	}

	// 已用资源：列出该节点上所有 Running Pod 的 requests 之和。
	podList, err := cli.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", name),
	})
	if err == nil {
		var cpuReq, memReq resource.Quantity
		podCount := 0
		for i := range podList.Items {
			pod := &podList.Items[i]
			if pod.Status.Phase != corev1.PodRunning {
				continue
			}
			podCount++
			for _, cs := range pod.Spec.Containers {
				if r, ok := cs.Resources.Requests[corev1.ResourceCPU]; ok {
					cpuReq.Add(r)
				}
				if r, ok := cs.Resources.Requests[corev1.ResourceMemory]; ok {
					memReq.Add(r)
				}
			}
		}
		usage.CPUUsed = cpuReq.String()
		usage.MemUsed = formatResource(memReq)
		usage.PodsUsed = podCount

		if !allocatableCPU.IsZero() {
			pct := float64(cpuReq.MilliValue()) / float64(allocatableCPU.MilliValue()) * 100
			usage.CPUPercent = fmt.Sprintf("%.1f%%", pct)
		}
		if !allocatableMem.IsZero() {
			pct := float64(memReq.Value()) / float64(allocatableMem.Value()) * 100
			usage.MemPercent = fmt.Sprintf("%.1f%%", pct)
		}
	}

	return usage, nil
}

// toK8sNodeItem corev1.Node → 列表项。
func toK8sNodeItem(n *corev1.Node) K8sNodeItem {
	item := K8sNodeItem{
		Name:             n.Name,
		Labels:           n.Labels,
		CreatedAt:        n.CreationTimestamp.Format("2006-01-02 15:04:05"),
		KernelVersion:    n.Status.NodeInfo.KernelVersion,
		ContainerRuntime: n.Status.NodeInfo.ContainerRuntimeVersion,
		OS:               n.Status.NodeInfo.OperatingSystem,
		Arch:             n.Status.NodeInfo.Architecture,
		Version:          n.Status.NodeInfo.KubeletVersion,
	}

	// 角色：从 labels 读取 node-role.kubernetes.io/<role>。
	for k := range n.Labels {
		if k == "node-role.kubernetes.io/master" || k == "node-role.kubernetes.io/control-plane" {
			item.Role = "master"
		}
	}
	if item.Role == "" {
		item.Role = "worker"
	}

	// Ready 状态。
	for _, cond := range n.Status.Conditions {
		if cond.Type == corev1.NodeReady {
			if cond.Status == corev1.ConditionTrue {
				item.Ready = true
				item.Status = "Ready"
			} else {
				item.Ready = false
				item.Status = "NotReady"
			}
			break
		}
	}

	// IP 地址。
	for _, addr := range n.Status.Addresses {
		switch addr.Type {
		case corev1.NodeInternalIP:
			item.InternalIP = addr.Address
		case corev1.NodeExternalIP:
			item.ExternalIP = addr.Address
		}
	}

	// 可分配资源。
	if val, ok := n.Status.Allocatable[corev1.ResourceCPU]; ok {
		item.CPU = val.String()
	}
	if val, ok := n.Status.Allocatable[corev1.ResourceMemory]; ok {
		item.Memory = formatResource(val)
	}
	if val, ok := n.Status.Allocatable[corev1.ResourcePods]; ok {
		item.PodsCapacity = val.Value()
	}

	// 污点。
	for _, taint := range n.Spec.Taints {
		item.Taints = append(item.Taints, K8sTaint{
			Key:    taint.Key,
			Value:  taint.Value,
			Effect: string(taint.Effect),
		})
	}

	return item
}
