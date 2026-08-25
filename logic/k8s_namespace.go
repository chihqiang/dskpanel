package logic

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/chihqiang/infra-go/logger"
)

// K8sNamespaceItem 命名空间列表项。
type K8sNamespaceItem struct {
	Name      string            `json:"name"`
	Status    string            `json:"status"` // Active / Terminating
	CreatedAt string            `json:"created_at"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// ListNamespaces 命名空间列表。
func (l *K8sLogic) ListNamespaces(ctx context.Context) ([]K8sNamespaceItem, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}

	nsList, err := cli.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	items := make([]K8sNamespaceItem, 0, len(nsList.Items))
	for _, ns := range nsList.Items {
		items = append(items, K8sNamespaceItem{
			Name:      ns.Name,
			Status:    string(ns.Status.Phase),
			CreatedAt: ns.CreationTimestamp.Format("2006-01-02 15:04:05"),
			Labels:    ns.Labels,
		})
	}
	return items, nil
}

// InspectNamespace 命名空间详情（原始对象）。
func (l *K8sLogic) InspectNamespace(ctx context.Context, name string) (*corev1.Namespace, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	return cli.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
}

// DeleteNamespace 删除命名空间。
func (l *K8sLogic) DeleteNamespace(ctx context.Context, name string) error {
	cli, err := l.newClient()
	if err != nil {
		return err
	}
	err = cli.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		logger.ErrorCtx(ctx, "k8s delete namespace failed", logger.String("name", name), logger.Err(err))
	}
	return err
}
