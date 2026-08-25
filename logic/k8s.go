package logic

import (
	"context"
	"fmt"

	"chihqiang/dskpanel/config"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// ErrK8sNotAvailable K8s 集群不可用。
var ErrK8sNotAvailable = fmt.Errorf("kubernetes not available")

// K8sListOptions 统一列表查询参数。
type K8sListOptions struct {
	Namespace     string // 命名空间（为空则用默认）
	LabelSelector string // 标签选择器（如 app=nginx）
	FieldSelector string // 字段选择器（如 status.phase=Running）
	Limit         int64  // 分页限制（0 表示不分页）
}

// toListOptions 将 K8sListOptions 转为 metav1.ListOptions。
func (o K8sListOptions) toListOptions() metav1.ListOptions {
	opts := metav1.ListOptions{
		LabelSelector: o.LabelSelector,
		FieldSelector: o.FieldSelector,
	}
	if o.Limit > 0 {
		opts.Limit = o.Limit
	}
	return opts
}

// K8sLogic Kubernetes 集群管理逻辑。
// 连接目标由 config.K8s 决定：Kubeconfig 为空则自动检测（InCluster 或默认 kubeconfig），
// 否则使用指定的 kubeconfig 内容连接远程集群。
type K8sLogic struct {
	cfg config.K8s
}

// NewK8sLogic 创建 K8s 管理逻辑。
func NewK8sLogic(cfg config.K8s) *K8sLogic {
	return &K8sLogic{cfg: cfg}
}

// namespace 返回默认命名空间（空则 default）。
func (l *K8sLogic) namespace() string {
	if l.cfg.Namespace != "" {
		return l.cfg.Namespace
	}
	return "default"
}

// restConfig 构建 rest.Config（复用于 clientset 和 SPDY executor）。
func (l *K8sLogic) restConfig() (*rest.Config, error) {
	// 1. 配置了 kubeconfig 内容 → 从内容加载。
	if l.cfg.Kubeconfig != "" {
		loader, err := clientcmd.NewClientConfigFromBytes([]byte(l.cfg.Kubeconfig))
		if err != nil {
			return nil, fmt.Errorf("parse kubeconfig failed: %w", err)
		}
		cfg, err := loader.ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("build rest config from kubeconfig failed: %w", err)
		}
		// 覆盖 master URL。
		if l.cfg.Master != "" {
			cfg.Host = l.cfg.Master
		}
		return cfg, nil
	}

	// 2. 未配置 kubeconfig → 尝试 InCluster 模式。
	if cfg, err := rest.InClusterConfig(); err == nil {
		return cfg, nil
	}

	// 3. 回退到默认 kubeconfig 文件路径（~/.kube/config）。
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		&clientcmd.ConfigOverrides{ClusterInfo: api.Cluster{Server: l.cfg.Master}},
	)
	cfg, err := loader.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("build rest config from default kubeconfig failed: %w", err)
	}
	return cfg, nil
}

// newClient 创建 K8s clientset。
func (l *K8sLogic) newClient() (kubernetes.Interface, error) {
	cfg, err := l.restConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(cfg)
}

// K8sStatus 集群状态摘要。
type K8sStatus struct {
	Available  bool   `json:"available"`
	Error      string `json:"error,omitempty"`
	Version    string `json:"version,omitempty"`
	Platform   string `json:"platform,omitempty"`
	GitVersion string `json:"git_version,omitempty"`
	BuildDate  string `json:"build_date,omitempty"`
	GoVersion  string `json:"go_version,omitempty"`
}

// Detect 检测 K8s 集群可用性。
func (l *K8sLogic) Detect(ctx context.Context) *K8sStatus {
	cli, err := l.newClient()
	if err != nil {
		return &K8sStatus{Available: false, Error: err.Error()}
	}

	// Discovery.ServerVersion() 不接受 context，此处忽略 ctx。
	_ = ctx
	version, err := cli.Discovery().ServerVersion()
	if err != nil {
		return &K8sStatus{Available: false, Error: err.Error()}
	}

	return &K8sStatus{
		Available:  true,
		Version:    version.GitVersion,
		Platform:   version.Platform,
		GitVersion: version.GitVersion,
		BuildDate:  version.BuildDate,
		GoVersion:  version.GoVersion,
	}
}

// K8sOverview 集群概览。
type K8sOverview struct {
	Status       *K8sStatus    `json:"status"`
	Nodes        []K8sNodeItem `json:"nodes"`
	Namespaces   int           `json:"namespaces"`
	Pods         int           `json:"pods"`
	Services     int           `json:"services"`
	Deployments  int           `json:"deployments"`
	StatefulSets int           `json:"statefulsets"`
	DaemonSets   int           `json:"daemonsets"`
	Summary      K8sSummary    `json:"summary"`
}

// K8sSummary 概览统计。
type K8sSummary struct {
	NodeCount        int            `json:"node_count"`
	MasterCount      int            `json:"master_count"`
	WorkerCount      int            `json:"worker_count"`
	NodesReady       int            `json:"nodes_ready"`
	PodCount         int            `json:"pod_count"`
	PodsByPhase      map[string]int `json:"pods_by_phase"`
	ServiceCount     int            `json:"service_count"`
	DeploymentCount  int            `json:"deployment_count"`
	StatefulSetCount int            `json:"statefulset_count"`
	DaemonSetCount   int            `json:"daemonset_count"`
	NamespaceCount   int            `json:"namespace_count"`
}

// Overview 集群概览：状态 + 节点/Pod/Service/Deployment 汇总。
func (l *K8sLogic) Overview(ctx context.Context) (*K8sOverview, error) {
	status := l.Detect(ctx)
	if !status.Available {
		return nil, ErrK8sNotAvailable
	}

	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}

	ov := &K8sOverview{
		Status: status,
		Summary: K8sSummary{
			PodsByPhase: map[string]int{},
		},
	}

	// 节点列表。
	if nodeList, err := cli.CoreV1().Nodes().List(ctx, metav1.ListOptions{}); err == nil {
		for i := range nodeList.Items {
			item := toK8sNodeItem(&nodeList.Items[i])
			ov.Nodes = append(ov.Nodes, item)
			ov.Summary.NodeCount++
			if item.Role == "master" {
				ov.Summary.MasterCount++
			} else {
				ov.Summary.WorkerCount++
			}
			if item.Ready {
				ov.Summary.NodesReady++
			}
		}
	}

	// 命名空间数量。
	if nsList, err := cli.CoreV1().Namespaces().List(ctx, metav1.ListOptions{}); err == nil {
		ov.Namespaces = len(nsList.Items)
		ov.Summary.NamespaceCount = len(nsList.Items)
	}

	// Pod 列表（全集群）。
	if podList, err := cli.CoreV1().Pods("").List(ctx, metav1.ListOptions{}); err == nil {
		ov.Pods = len(podList.Items)
		ov.Summary.PodCount = len(podList.Items)
		for _, pod := range podList.Items {
			ov.Summary.PodsByPhase[string(pod.Status.Phase)]++
		}
	}

	// Service 列表（全集群）。
	if svcList, err := cli.CoreV1().Services("").List(ctx, metav1.ListOptions{}); err == nil {
		ov.Services = len(svcList.Items)
		ov.Summary.ServiceCount = len(svcList.Items)
	}

	// Deployment 列表（全集群）。
	if depList, err := cli.AppsV1().Deployments("").List(ctx, metav1.ListOptions{}); err == nil {
		ov.Deployments = len(depList.Items)
		ov.Summary.DeploymentCount = len(depList.Items)
	}

	// StatefulSet 列表（全集群）。
	if stsList, err := cli.AppsV1().StatefulSets("").List(ctx, metav1.ListOptions{}); err == nil {
		ov.StatefulSets = len(stsList.Items)
		ov.Summary.StatefulSetCount = len(stsList.Items)
	}

	// DaemonSet 列表（全集群）。
	if dsList, err := cli.AppsV1().DaemonSets("").List(ctx, metav1.ListOptions{}); err == nil {
		ov.DaemonSets = len(dsList.Items)
		ov.Summary.DaemonSetCount = len(dsList.Items)
	}

	return ov, nil
}

// formatResource 格式化资源数量（内存转为人类可读）。
func formatResource(q resource.Quantity) string {
	return q.String()
}
