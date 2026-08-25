package logic

import (
	"context"
	"fmt"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/chihqiang/infra-go/logger"
)

// K8sHPAItem HPA 列表项。
type K8sHPAItem struct {
	Name             string `json:"name"`
	Namespace        string `json:"namespace"`
	TargetRef        string `json:"target_ref"`  // Kind/Name
	MinReplicas      int32  `json:"min_replicas"`
	MaxReplicas      int32  `json:"max_replicas"`
	CurrentReplicas  int32  `json:"current_replicas"`
	DesiredReplicas  int32  `json:"desired_replicas"`
	Metrics          string `json:"metrics"`   // 简要描述指标
	CreatedAt        string `json:"created_at"`
	Labels           map[string]string `json:"labels,omitempty"`
}

// ListHPAs HPA 列表。
func (l *K8sLogic) ListHPAs(ctx context.Context, namespace string) ([]K8sHPAItem, error) {
	return l.ListHPAsWithOptions(ctx, K8sListOptions{Namespace: namespace})
}

// ListHPAsWithOptions HPA 列表（支持标签/字段过滤）。
func (l *K8sLogic) ListHPAsWithOptions(ctx context.Context, opts K8sListOptions) ([]K8sHPAItem, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	ns := l.resolveNamespace(opts.Namespace)

	hpaList, err := cli.AutoscalingV2().HorizontalPodAutoscalers(ns).List(ctx, opts.toListOptions())
	if err != nil {
		return nil, err
	}

	items := make([]K8sHPAItem, 0, len(hpaList.Items))
	for i := range hpaList.Items {
		items = append(items, toK8sHPAItem(&hpaList.Items[i]))
	}
	return items, nil
}

// InspectHPA HPA 详情（原始对象）。
func (l *K8sLogic) InspectHPA(ctx context.Context, namespace, name string) (*autoscalingv2.HorizontalPodAutoscaler, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	ns := l.namespace()
	if namespace != "" {
		ns = namespace
	}
	return cli.AutoscalingV2().HorizontalPodAutoscalers(ns).Get(ctx, name, metav1.GetOptions{})
}

// DeleteHPA 删除 HPA。
func (l *K8sLogic) DeleteHPA(ctx context.Context, namespace, name string) error {
	cli, err := l.newClient()
	if err != nil {
		return err
	}
	ns := l.namespace()
	if namespace != "" {
		ns = namespace
	}
	err = cli.AutoscalingV2().HorizontalPodAutoscalers(ns).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		logger.ErrorCtx(ctx, "k8s delete hpa failed", logger.String("ns", ns), logger.String("name", name), logger.Err(err))
	}
	return err
}

// toK8sHPAItem autoscalingv2.HorizontalPodAutoscaler → 列表项。
func toK8sHPAItem(hpa *autoscalingv2.HorizontalPodAutoscaler) K8sHPAItem {
	item := K8sHPAItem{
		Name:            hpa.Name,
		Namespace:       hpa.Namespace,
		MaxReplicas:     hpa.Spec.MaxReplicas,
		CurrentReplicas: hpa.Status.CurrentReplicas,
		DesiredReplicas: hpa.Status.DesiredReplicas,
		CreatedAt:       hpa.CreationTimestamp.Format("2006-01-02 15:04:05"),
		Labels:          hpa.Labels,
	}

	if hpa.Spec.MinReplicas != nil {
		item.MinReplicas = *hpa.Spec.MinReplicas
	}

	// 目标引用。
	ref := hpa.Spec.ScaleTargetRef
	item.TargetRef = fmt.Sprintf("%s/%s", ref.Kind, ref.Name)

	// 指标简要描述。
	item.Metrics = formatHPAMetrics(hpa.Spec.Metrics)

	return item
}

// formatHPAMetrics 将 HPA metrics 简要格式化为可读字符串。
func formatHPAMetrics(metrics []autoscalingv2.MetricSpec) string {
	if len(metrics) == 0 {
		return "80% CPU (default)"
	}
	parts := make([]string, 0, len(metrics))
	for _, m := range metrics {
		switch m.Type {
		case autoscalingv2.ResourceMetricSourceType:
			if m.Resource != nil {
				target := ""
				if m.Resource.Target.AverageUtilization != nil {
					target = fmt.Sprintf("%d%%", *m.Resource.Target.AverageUtilization)
				} else if m.Resource.Target.AverageValue != nil {
					target = m.Resource.Target.AverageValue.String()
				}
				parts = append(parts, fmt.Sprintf("%s %s", string(m.Resource.Name), target))
			}
		case autoscalingv2.PodsMetricSourceType:
			if m.Pods != nil {
				target := ""
				if m.Pods.Target.AverageValue != nil {
					target = m.Pods.Target.AverageValue.String()
				}
				parts = append(parts, fmt.Sprintf("Pods/%s %s", m.Pods.Metric.Name, target))
			}
		case autoscalingv2.ExternalMetricSourceType:
			if m.External != nil {
				target := ""
				if m.External.Target.AverageValue != nil {
					target = m.External.Target.AverageValue.String()
				} else if m.External.Target.Value != nil {
					target = m.External.Target.Value.String()
				}
				parts = append(parts, fmt.Sprintf("External/%s %s", m.External.Metric.Name, target))
			}
		case autoscalingv2.ObjectMetricSourceType:
			if m.Object != nil {
				target := ""
				if m.Object.Target.Value != nil {
					target = m.Object.Target.Value.String()
				}
				parts = append(parts, fmt.Sprintf("Object/%s/%s %s", m.Object.DescribedObject.Kind, m.Object.DescribedObject.Name, target))
			}
		case autoscalingv2.ContainerResourceMetricSourceType:
			if m.ContainerResource != nil {
				target := ""
				if m.ContainerResource.Target.AverageUtilization != nil {
					target = fmt.Sprintf("%d%%", *m.ContainerResource.Target.AverageUtilization)
				}
				parts = append(parts, fmt.Sprintf("Container/%s %s", string(m.ContainerResource.Name), target))
			}
		}
	}
	if len(parts) == 0 {
		return "—"
	}
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += ", "
		}
		result += p
	}
	return result
}
