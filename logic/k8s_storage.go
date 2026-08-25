package logic

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/chihqiang/infra-go/logger"
)

// K8sPVCItem PVC 列表项。
type K8sPVCItem struct {
	Name         string `json:"name"`
	Namespace    string `json:"namespace"`
	Status       string `json:"status"` // Pending / Bound / Lost
	VolumeName   string `json:"volume_name"`
	StorageClass string `json:"storage_class"`
	AccessModes  string `json:"access_modes"`
	Capacity     string `json:"capacity"`
	Requested    string `json:"requested"`
	CreatedAt    string `json:"created_at"`
	Labels       map[string]string `json:"labels,omitempty"`
}

// K8sStorageClassItem StorageClass 列表项。
type K8sStorageClassItem struct {
	Name         string `json:"name"`
	Provisioner  string `json:"provisioner"`
	ReclaimPolicy string `json:"reclaim_policy"`
	BindingMode   string `json:"binding_mode"`
	Default       bool   `json:"default"`
	VolumeBinding string `json:"volume_binding"`
	CreatedAt     string `json:"created_at"`
}

// ListPVCs PVC 列表。
func (l *K8sLogic) ListPVCs(ctx context.Context, namespace string) ([]K8sPVCItem, error) {
	return l.ListPVCsWithOptions(ctx, K8sListOptions{Namespace: namespace})
}

// ListPVCsWithOptions PVC 列表（支持标签/字段过滤）。
func (l *K8sLogic) ListPVCsWithOptions(ctx context.Context, opts K8sListOptions) ([]K8sPVCItem, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	ns := l.resolveNamespace(opts.Namespace)

	pvcList, err := cli.CoreV1().PersistentVolumeClaims(ns).List(ctx, opts.toListOptions())
	if err != nil {
		return nil, err
	}

	items := make([]K8sPVCItem, 0, len(pvcList.Items))
	for i := range pvcList.Items {
		items = append(items, toK8sPVCItem(&pvcList.Items[i]))
	}
	return items, nil
}

// InspectPVC PVC 详情（原始对象）。
func (l *K8sLogic) InspectPVC(ctx context.Context, namespace, name string) (*corev1.PersistentVolumeClaim, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	ns := l.namespace()
	if namespace != "" {
		ns = namespace
	}
	return cli.CoreV1().PersistentVolumeClaims(ns).Get(ctx, name, metav1.GetOptions{})
}

// DeletePVC 删除 PVC。
func (l *K8sLogic) DeletePVC(ctx context.Context, namespace, name string) error {
	cli, err := l.newClient()
	if err != nil {
		return err
	}
	ns := l.namespace()
	if namespace != "" {
		ns = namespace
	}
	err = cli.CoreV1().PersistentVolumeClaims(ns).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		logger.ErrorCtx(ctx, "k8s delete pvc failed", logger.String("ns", ns), logger.String("name", name), logger.Err(err))
	}
	return err
}

// ListStorageClasses StorageClass 列表。
func (l *K8sLogic) ListStorageClasses(ctx context.Context) ([]K8sStorageClassItem, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}

	scList, err := cli.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// 查找默认 StorageClass（通过 annotation）。
	defaultSC := ""
	for _, sc := range scList.Items {
		if sc.Annotations["storageclass.kubernetes.io/is-default-class"] == "true" {
			defaultSC = sc.Name
			break
		}
	}

	items := make([]K8sStorageClassItem, 0, len(scList.Items))
	for i := range scList.Items {
		sc := &scList.Items[i]
		bindingMode := "Immediate"
		if sc.VolumeBindingMode != nil {
			bindingMode = string(*sc.VolumeBindingMode)
		}
		reclaimPolicy := "Delete"
		if sc.ReclaimPolicy != nil {
			reclaimPolicy = string(*sc.ReclaimPolicy)
		}
		items = append(items, K8sStorageClassItem{
			Name:          sc.Name,
			Provisioner:   sc.Provisioner,
			ReclaimPolicy: reclaimPolicy,
			BindingMode:   bindingMode,
			Default:        sc.Name == defaultSC,
			VolumeBinding: bindingMode,
			CreatedAt:     sc.CreationTimestamp.Format("2006-01-02 15:04:05"),
		})
	}
	return items, nil
}

// InspectStorageClass StorageClass 详情（原始对象）。
func (l *K8sLogic) InspectStorageClass(ctx context.Context, name string) (*storagev1.StorageClass, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	return cli.StorageV1().StorageClasses().Get(ctx, name, metav1.GetOptions{})
}

// toK8sPVCItem corev1.PersistentVolumeClaim → 列表项。
func toK8sPVCItem(pvc *corev1.PersistentVolumeClaim) K8sPVCItem {
	item := K8sPVCItem{
		Name:         pvc.Name,
		Namespace:    pvc.Namespace,
		Status:       string(pvc.Status.Phase),
		VolumeName:   pvc.Spec.VolumeName,
		StorageClass: "",
		AccessModes:  formatAccessModes(pvc.Status.AccessModes),
		Capacity:     "",
		Requested:    "",
		CreatedAt:    pvc.CreationTimestamp.Format("2006-01-02 15:04:05"),
		Labels:       pvc.Labels,
	}

	if pvc.Spec.StorageClassName != nil {
		item.StorageClass = *pvc.Spec.StorageClassName
	}

	if requested, ok := pvc.Spec.Resources.Requests[corev1.ResourceStorage]; ok {
		item.Requested = requested.String()
	}

	if capacity, ok := pvc.Status.Capacity[corev1.ResourceStorage]; ok {
		item.Capacity = capacity.String()
	}

	if item.AccessModes == "" {
		item.AccessModes = formatAccessModes(pvc.Spec.AccessModes)
	}

	return item
}

// formatAccessModes 格式化访问模式。
func formatAccessModes(modes []corev1.PersistentVolumeAccessMode) string {
	names := make([]string, 0, len(modes))
	for _, m := range modes {
		switch m {
		case corev1.ReadWriteOnce:
			names = append(names, "RWO")
		case corev1.ReadOnlyMany:
			names = append(names, "ROX")
		case corev1.ReadWriteMany:
			names = append(names, "RWX")
		case corev1.ReadWriteOncePod:
			names = append(names, "RWOP")
		default:
			names = append(names, string(m))
		}
	}
	result := ""
	for i, n := range names {
		if i > 0 {
			result += ","
		}
		result += n
	}
	return result
}

// Ensure resource import is used.
var _ = resource.Quantity{}
var _ = fmt.Sprintf
