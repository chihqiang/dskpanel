package logic

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/chihqiang/infra-go/logger"
)

// K8sRoleItem Role / ClusterRole 列表项。
type K8sRoleItem struct {
	Name       string `json:"name"`
	Namespace  string `json:"namespace,omitempty"` // ClusterRole 为空
	Kind       string `json:"kind"`                // Role / ClusterRole
	Rules      int    `json:"rules"`
	CreatedAt  string `json:"created_at"`
	Labels     map[string]string `json:"labels,omitempty"`
}

// K8sRoleBindingItem RoleBinding / ClusterRoleBinding 列表项。
type K8sRoleBindingItem struct {
	Name       string `json:"name"`
	Namespace  string `json:"namespace,omitempty"` // ClusterRoleBinding 为空
	Kind       string `json:"kind"`                // RoleBinding / ClusterRoleBinding
	RoleKind   string `json:"role_kind"`           // Role / ClusterRole
	RoleName   string `json:"role_name"`
	Subjects   int    `json:"subjects"`
	CreatedAt  string `json:"created_at"`
	Labels     map[string]string `json:"labels,omitempty"`
}

// ListRoles Role 列表。
func (l *K8sLogic) ListRoles(ctx context.Context, namespace string) ([]K8sRoleItem, error) {
	return l.ListRolesWithOptions(ctx, K8sListOptions{Namespace: namespace})
}

// ListRolesWithOptions Role 列表（支持标签/字段过滤）。
func (l *K8sLogic) ListRolesWithOptions(ctx context.Context, opts K8sListOptions) ([]K8sRoleItem, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	ns := l.resolveNamespace(opts.Namespace)

	roleList, err := cli.RbacV1().Roles(ns).List(ctx, opts.toListOptions())
	if err != nil {
		return nil, err
	}

	items := make([]K8sRoleItem, 0, len(roleList.Items))
	for i := range roleList.Items {
		items = append(items, toK8sRoleItem(&roleList.Items[i], "Role"))
	}
	return items, nil
}

// InspectRole Role 详情（原始对象）。
func (l *K8sLogic) InspectRole(ctx context.Context, namespace, name string) (*rbacv1.Role, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	ns := l.namespace()
	if namespace != "" {
		ns = namespace
	}
	return cli.RbacV1().Roles(ns).Get(ctx, name, metav1.GetOptions{})
}

// DeleteRole 删除 Role。
func (l *K8sLogic) DeleteRole(ctx context.Context, namespace, name string) error {
	cli, err := l.newClient()
	if err != nil {
		return err
	}
	ns := l.namespace()
	if namespace != "" {
		ns = namespace
	}
	err = cli.RbacV1().Roles(ns).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		logger.ErrorCtx(ctx, "k8s delete role failed", logger.String("ns", ns), logger.String("name", name), logger.Err(err))
	}
	return err
}

// ListClusterRoles ClusterRole 列表。
func (l *K8sLogic) ListClusterRoles(ctx context.Context) ([]K8sRoleItem, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}

	list, err := cli.RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	items := make([]K8sRoleItem, 0, len(list.Items))
	for i := range list.Items {
		items = append(items, toK8sRoleItem(&list.Items[i], "ClusterRole"))
	}
	return items, nil
}

// InspectClusterRole ClusterRole 详情（原始对象）。
func (l *K8sLogic) InspectClusterRole(ctx context.Context, name string) (*rbacv1.ClusterRole, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	return cli.RbacV1().ClusterRoles().Get(ctx, name, metav1.GetOptions{})
}

// DeleteClusterRole 删除 ClusterRole。
func (l *K8sLogic) DeleteClusterRole(ctx context.Context, name string) error {
	cli, err := l.newClient()
	if err != nil {
		return err
	}
	err = cli.RbacV1().ClusterRoles().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		logger.ErrorCtx(ctx, "k8s delete clusterrole failed", logger.String("name", name), logger.Err(err))
	}
	return err
}

// ListRoleBindings RoleBinding 列表。
func (l *K8sLogic) ListRoleBindings(ctx context.Context, namespace string) ([]K8sRoleBindingItem, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	ns := l.resolveNamespace(namespace)

	list, err := cli.RbacV1().RoleBindings(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	items := make([]K8sRoleBindingItem, 0, len(list.Items))
	for i := range list.Items {
		items = append(items, toK8sRoleBindingItem(&list.Items[i], "RoleBinding"))
	}
	return items, nil
}

// InspectRoleBinding RoleBinding 详情（原始对象）。
func (l *K8sLogic) InspectRoleBinding(ctx context.Context, namespace, name string) (*rbacv1.RoleBinding, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	ns := l.namespace()
	if namespace != "" {
		ns = namespace
	}
	return cli.RbacV1().RoleBindings(ns).Get(ctx, name, metav1.GetOptions{})
}

// DeleteRoleBinding 删除 RoleBinding。
func (l *K8sLogic) DeleteRoleBinding(ctx context.Context, namespace, name string) error {
	cli, err := l.newClient()
	if err != nil {
		return err
	}
	ns := l.namespace()
	if namespace != "" {
		ns = namespace
	}
	err = cli.RbacV1().RoleBindings(ns).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		logger.ErrorCtx(ctx, "k8s delete rolebinding failed", logger.String("ns", ns), logger.String("name", name), logger.Err(err))
	}
	return err
}

// ListClusterRoleBindings ClusterRoleBinding 列表。
func (l *K8sLogic) ListClusterRoleBindings(ctx context.Context) ([]K8sRoleBindingItem, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}

	list, err := cli.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	items := make([]K8sRoleBindingItem, 0, len(list.Items))
	for i := range list.Items {
		items = append(items, toK8sRoleBindingItem(&list.Items[i], "ClusterRoleBinding"))
	}
	return items, nil
}

// InspectClusterRoleBinding ClusterRoleBinding 详情（原始对象）。
func (l *K8sLogic) InspectClusterRoleBinding(ctx context.Context, name string) (*rbacv1.ClusterRoleBinding, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	return cli.RbacV1().ClusterRoleBindings().Get(ctx, name, metav1.GetOptions{})
}

// DeleteClusterRoleBinding 删除 ClusterRoleBinding。
func (l *K8sLogic) DeleteClusterRoleBinding(ctx context.Context, name string) error {
	cli, err := l.newClient()
	if err != nil {
		return err
	}
	err = cli.RbacV1().ClusterRoleBindings().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		logger.ErrorCtx(ctx, "k8s delete clusterrolebinding failed", logger.String("name", name), logger.Err(err))
	}
	return err
}

// toK8sRoleItem rbacv1.Role 或 ClusterRole → 列表项。
// 通过传入 kind 区分。
func toK8sRoleItem(obj interface{}, kind string) K8sRoleItem {
	item := K8sRoleItem{Kind: kind}
	switch r := obj.(type) {
	case *rbacv1.Role:
		item.Name = r.Name
		item.Namespace = r.Namespace
		item.Rules = len(r.Rules)
		item.CreatedAt = r.CreationTimestamp.Format("2006-01-02 15:04:05")
		item.Labels = r.Labels
	case *rbacv1.ClusterRole:
		item.Name = r.Name
		item.Rules = len(r.Rules)
		item.CreatedAt = r.CreationTimestamp.Format("2006-01-02 15:04:05")
		item.Labels = r.Labels
	}
	return item
}

// toK8sRoleBindingItem rbacv1.RoleBinding 或 ClusterRoleBinding → 列表项。
func toK8sRoleBindingItem(obj interface{}, kind string) K8sRoleBindingItem {
	item := K8sRoleBindingItem{Kind: kind}
	switch rb := obj.(type) {
	case *rbacv1.RoleBinding:
		item.Name = rb.Name
		item.Namespace = rb.Namespace
		item.RoleKind = rb.RoleRef.Kind
		item.RoleName = rb.RoleRef.Name
		item.Subjects = len(rb.Subjects)
		item.CreatedAt = rb.CreationTimestamp.Format("2006-01-02 15:04:05")
		item.Labels = rb.Labels
	case *rbacv1.ClusterRoleBinding:
		item.Name = rb.Name
		item.RoleKind = rb.RoleRef.Kind
		item.RoleName = rb.RoleRef.Name
		item.Subjects = len(rb.Subjects)
		item.CreatedAt = rb.CreationTimestamp.Format("2006-01-02 15:04:05")
		item.Labels = rb.Labels
	}
	return item
}
