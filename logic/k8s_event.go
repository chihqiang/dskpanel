package logic

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// K8sEvent 事件列表项。
type K8sEvent struct {
	Type      string `json:"type"` // Normal / Warning
	Reason    string `json:"reason"`
	Message   string `json:"message"`
	Object    string `json:"object"` // 涉及对象（如 pod/nginx-xxx）
	Count     int32  `json:"count"`
	LastTime  string `json:"last_time"`
	FirstTime string `json:"first_time"`
}

// ListEvents 查询事件列表。
// fieldSelector 可用于过滤特定资源（如 involvedObject.kind=Pod&&involvedObject.name=xxx）。
func (l *K8sLogic) ListEvents(ctx context.Context, namespace, fieldSelector string) ([]K8sEvent, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	ns := l.namespace()
	if namespace != "" {
		ns = namespace
	}

	eventList, err := cli.CoreV1().Events(ns).List(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector,
	})
	if err != nil {
		return nil, err
	}

	items := make([]K8sEvent, 0, len(eventList.Items))
	for _, e := range eventList.Items {
		items = append(items, K8sEvent{
			Type:      e.Type,
			Reason:    e.Reason,
			Message:   e.Message,
			Object:    fmt.Sprintf("%s/%s", e.InvolvedObject.Kind, e.InvolvedObject.Name),
			Count:     e.Count,
			LastTime:  e.LastTimestamp.Format("2006-01-02 15:04:05"),
			FirstTime: e.FirstTimestamp.Format("2006-01-02 15:04:05"),
		})
	}
	return items, nil
}

// ListEventsForObject 查询特定资源的事件。
// kind 如 "Pod"/"Deployment"/"Node"，name 为资源名。
func (l *K8sLogic) ListEventsForObject(ctx context.Context, namespace, kind, name string) ([]K8sEvent, error) {
	fieldSelector := fmt.Sprintf("involvedObject.kind=%s,involvedObject.name=%s", kind, name)
	return l.ListEvents(ctx, namespace, fieldSelector)
}
