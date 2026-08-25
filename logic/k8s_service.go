package logic

import (
	"context"

	"github.com/chihqiang/infra-go/logger"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// K8sServiceItem Service 列表项。
type K8sServiceItem struct {
	Name       string            `json:"name"`
	Namespace  string            `json:"namespace"`
	Type       string            `json:"type"` // ClusterIP / NodePort / LoadBalancer / ExternalName
	ClusterIP  string            `json:"cluster_ip"`
	ExternalIP string            `json:"external_ip,omitempty"`
	Ports      []K8sServicePort  `json:"ports,omitempty"`
	Selector   map[string]string `json:"selector,omitempty"`
	CreatedAt  string            `json:"created_at"`
}

// K8sServicePort Service 端口。
type K8sServicePort struct {
	Name       string `json:"name,omitempty"`
	Port       int32  `json:"port"`
	TargetPort string `json:"target_port"`
	Protocol   string `json:"protocol"`
	NodePort   int32  `json:"node_port,omitempty"`
}

// ListServices Service 列表。
func (l *K8sLogic) ListServices(ctx context.Context, namespace string) ([]K8sServiceItem, error) {
	return l.ListServicesWithOptions(ctx, K8sListOptions{Namespace: namespace})
}

// ListServicesWithOptions Service 列表（支持标签/字段过滤）。
func (l *K8sLogic) ListServicesWithOptions(ctx context.Context, opts K8sListOptions) ([]K8sServiceItem, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	ns := l.namespace()
	if opts.Namespace != "" {
		ns = opts.Namespace
	}

	svcList, err := cli.CoreV1().Services(ns).List(ctx, opts.toListOptions())
	if err != nil {
		return nil, err
	}

	items := make([]K8sServiceItem, 0, len(svcList.Items))
	for i := range svcList.Items {
		items = append(items, toK8sServiceItem(&svcList.Items[i]))
	}
	return items, nil
}

// InspectService Service 详情。
func (l *K8sLogic) InspectService(ctx context.Context, namespace, name string) (*corev1.Service, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	ns := l.namespace()
	if namespace != "" {
		ns = namespace
	}
	return cli.CoreV1().Services(ns).Get(ctx, name, metav1.GetOptions{})
}

// DeleteService 删除 Service。
func (l *K8sLogic) DeleteService(ctx context.Context, namespace, name string) error {
	cli, err := l.newClient()
	if err != nil {
		return err
	}
	ns := l.namespace()
	if namespace != "" {
		ns = namespace
	}
	err = cli.CoreV1().Services(ns).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		logger.ErrorCtx(ctx, "k8s delete service failed", logger.String("ns", ns), logger.String("name", name), logger.Err(err))
	}
	return err
}

// toK8sServiceItem corev1.Service → 列表项。
func toK8sServiceItem(s *corev1.Service) K8sServiceItem {
	item := K8sServiceItem{
		Name:      s.Name,
		Namespace: s.Namespace,
		Type:      string(s.Spec.Type),
		ClusterIP: s.Spec.ClusterIP,
		Selector:  s.Spec.Selector,
		CreatedAt: s.CreationTimestamp.Format("2006-01-02 15:04:05"),
	}

	// 外部 IP。
	if len(s.Status.LoadBalancer.Ingress) > 0 {
		ingress := s.Status.LoadBalancer.Ingress[0]
		if ingress.IP != "" {
			item.ExternalIP = ingress.IP
		} else if ingress.Hostname != "" {
			item.ExternalIP = ingress.Hostname
		}
	}

	// 端口。
	for _, p := range s.Spec.Ports {
		port := K8sServicePort{
			Name:       p.Name,
			Port:       p.Port,
			TargetPort: p.TargetPort.String(),
			Protocol:   string(p.Protocol),
		}
		if p.NodePort > 0 {
			port.NodePort = p.NodePort
		}
		item.Ports = append(item.Ports, port)
	}

	return item
}

// K8sIngressItem Ingress 列表项。
type K8sIngressItem struct {
	Name      string   `json:"name"`
	Namespace string   `json:"namespace"`
	Hosts     []string `json:"hosts,omitempty"`
	Address   string   `json:"address,omitempty"`
	ClassName string   `json:"class_name,omitempty"`
	CreatedAt string   `json:"created_at"`
}

// ListIngresses Ingress 列表。
func (l *K8sLogic) ListIngresses(ctx context.Context, namespace string) ([]K8sIngressItem, error) {
	return l.ListIngressesWithOptions(ctx, K8sListOptions{Namespace: namespace})
}

// ListIngressesWithOptions Ingress 列表（支持标签/字段过滤）。
func (l *K8sLogic) ListIngressesWithOptions(ctx context.Context, opts K8sListOptions) ([]K8sIngressItem, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	ns := l.namespace()
	if opts.Namespace != "" {
		ns = opts.Namespace
	}
	ingList, err := cli.NetworkingV1().Ingresses(ns).List(ctx, opts.toListOptions())
	if err != nil {
		return nil, err
	}

	items := make([]K8sIngressItem, 0, len(ingList.Items))
	for i := range ingList.Items {
		ing := &ingList.Items[i]
		item := K8sIngressItem{
			Name:      ing.Name,
			Namespace: ing.Namespace,
			CreatedAt: ing.CreationTimestamp.Format("2006-01-02 15:04:05"),
		}
		if ing.Spec.IngressClassName != nil {
			item.ClassName = *ing.Spec.IngressClassName
		}
		for _, rule := range ing.Spec.Rules {
			if rule.Host != "" {
				item.Hosts = append(item.Hosts, rule.Host)
			}
		}
		if len(ing.Status.LoadBalancer.Ingress) > 0 {
			ig := ing.Status.LoadBalancer.Ingress[0]
			if ig.IP != "" {
				item.Address = ig.IP
			} else if ig.Hostname != "" {
				item.Address = ig.Hostname
			}
		}
		items = append(items, item)
	}
	return items, nil
}

// InspectIngress Ingress 详情。
func (l *K8sLogic) InspectIngress(ctx context.Context, namespace, name string) (*networkingv1.Ingress, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	ns := l.namespace()
	if namespace != "" {
		ns = namespace
	}
	return cli.NetworkingV1().Ingresses(ns).Get(ctx, name, metav1.GetOptions{})
}

// DeleteIngress 删除 Ingress。
func (l *K8sLogic) DeleteIngress(ctx context.Context, namespace, name string) error {
	cli, err := l.newClient()
	if err != nil {
		return err
	}
	ns := l.namespace()
	if namespace != "" {
		ns = namespace
	}
	return cli.NetworkingV1().Ingresses(ns).Delete(ctx, name, metav1.DeleteOptions{})
}

// K8sConfigMapItem ConfigMap 列表项。
type K8sConfigMapItem struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	DataKeys  int               `json:"data_keys"`
	CreatedAt string            `json:"created_at"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// ListConfigMaps ConfigMap 列表。
func (l *K8sLogic) ListConfigMaps(ctx context.Context, namespace string) ([]K8sConfigMapItem, error) {
	return l.ListConfigMapsWithOptions(ctx, K8sListOptions{Namespace: namespace})
}

// ListConfigMapsWithOptions ConfigMap 列表（支持标签/字段过滤）。
func (l *K8sLogic) ListConfigMapsWithOptions(ctx context.Context, opts K8sListOptions) ([]K8sConfigMapItem, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	ns := l.namespace()
	if opts.Namespace != "" {
		ns = opts.Namespace
	}

	cmList, err := cli.CoreV1().ConfigMaps(ns).List(ctx, opts.toListOptions())
	if err != nil {
		return nil, err
	}

	items := make([]K8sConfigMapItem, 0, len(cmList.Items))
	for i := range cmList.Items {
		cm := &cmList.Items[i]
		items = append(items, K8sConfigMapItem{
			Name:      cm.Name,
			Namespace: cm.Namespace,
			DataKeys:  len(cm.Data),
			CreatedAt: cm.CreationTimestamp.Format("2006-01-02 15:04:05"),
			Labels:    cm.Labels,
		})
	}
	return items, nil
}

// InspectConfigMap ConfigMap 详情。
func (l *K8sLogic) InspectConfigMap(ctx context.Context, namespace, name string) (*corev1.ConfigMap, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	ns := l.namespace()
	if namespace != "" {
		ns = namespace
	}
	return cli.CoreV1().ConfigMaps(ns).Get(ctx, name, metav1.GetOptions{})
}

// DeleteConfigMap 删除 ConfigMap。
func (l *K8sLogic) DeleteConfigMap(ctx context.Context, namespace, name string) error {
	cli, err := l.newClient()
	if err != nil {
		return err
	}
	ns := l.namespace()
	if namespace != "" {
		ns = namespace
	}
	return cli.CoreV1().ConfigMaps(ns).Delete(ctx, name, metav1.DeleteOptions{})
}

// K8sSecretItem Secret 列表项。
type K8sSecretItem struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Type      string            `json:"type"`
	DataKeys  int               `json:"data_keys"`
	CreatedAt string            `json:"created_at"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// ListSecrets Secret 列表。
func (l *K8sLogic) ListSecrets(ctx context.Context, namespace string) ([]K8sSecretItem, error) {
	return l.ListSecretsWithOptions(ctx, K8sListOptions{Namespace: namespace})
}

// ListSecretsWithOptions Secret 列表（支持标签/字段过滤）。
func (l *K8sLogic) ListSecretsWithOptions(ctx context.Context, opts K8sListOptions) ([]K8sSecretItem, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	ns := l.namespace()
	if opts.Namespace != "" {
		ns = opts.Namespace
	}

	secList, err := cli.CoreV1().Secrets(ns).List(ctx, opts.toListOptions())
	if err != nil {
		return nil, err
	}

	items := make([]K8sSecretItem, 0, len(secList.Items))
	for i := range secList.Items {
		sec := &secList.Items[i]
		items = append(items, K8sSecretItem{
			Name:      sec.Name,
			Namespace: sec.Namespace,
			Type:      string(sec.Type),
			DataKeys:  len(sec.Data),
			CreatedAt: sec.CreationTimestamp.Format("2006-01-02 15:04:05"),
			Labels:    sec.Labels,
		})
	}
	return items, nil
}

// InspectSecret Secret 详情（不返回 data 明文，仅返回 key 列表）。
type K8sSecretDetail struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Type      string            `json:"type"`
	DataKeys  []string          `json:"data_keys"`
	Labels    map[string]string `json:"labels,omitempty"`
	CreatedAt string            `json:"created_at"`
}

// InspectSecretDetail Secret 详情（不返回 data 明文）。
func (l *K8sLogic) InspectSecretDetail(ctx context.Context, namespace, name string) (*K8sSecretDetail, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	ns := l.namespace()
	if namespace != "" {
		ns = namespace
	}

	sec, err := cli.CoreV1().Secrets(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	detail := &K8sSecretDetail{
		Name:      sec.Name,
		Namespace: sec.Namespace,
		Type:      string(sec.Type),
		Labels:    sec.Labels,
		CreatedAt: sec.CreationTimestamp.Format("2006-01-02 15:04:05"),
	}
	for k := range sec.Data {
		detail.DataKeys = append(detail.DataKeys, k)
	}
	return detail, nil
}

// InspectSecretRaw Secret 原始对象（data 值清空，仅保留 key，支持 ?format=yaml）。
func (l *K8sLogic) InspectSecretRaw(ctx context.Context, namespace, name string) (*corev1.Secret, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	ns := l.namespace()
	if namespace != "" {
		ns = namespace
	}
	sec, err := cli.CoreV1().Secrets(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	// 脱敏：清空 data 值，仅保留 key（值设为空字节）。
	for k := range sec.Data {
		sec.Data[k] = []byte{}
	}
	sec.StringData = nil
	return sec, nil
}

// DeleteSecret 删除 Secret。
func (l *K8sLogic) DeleteSecret(ctx context.Context, namespace, name string) error {
	cli, err := l.newClient()
	if err != nil {
		return err
	}
	ns := l.namespace()
	if namespace != "" {
		ns = namespace
	}
	return cli.CoreV1().Secrets(ns).Delete(ctx, name, metav1.DeleteOptions{})
}
