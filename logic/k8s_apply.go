package logic

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/chihqiang/infra-go/logger"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

// K8sApplyResult YAML 透传结果。
type K8sApplyResult struct {
	OK      bool           `json:"ok"`
	Message string         `json:"message"`
	Items   []K8sApplyItem `json:"items,omitempty"`
}

// K8sApplyItem 单个资源透传结果。
type K8sApplyItem struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
	Action    string `json:"action"` // created / updated / failed
	Message   string `json:"message,omitempty"`
}

// ApplyYAML 将多文档 YAML 直接透传到 K8s 集群（kubectl apply 语义）。
// 支持任意 K8s 资源类型（Deployment / Service / ConfigMap / Pod / CRD / ...）。
// 每个文档先尝试 Create，已存在则 Get → SetResourceVersion → Update。
func (l *K8sLogic) ApplyYAML(ctx context.Context, content string) (*K8sApplyResult, error) {
	return l.applyYAML(ctx, content, false)
}

// DryRunYAML 验证多文档 YAML 语法和资源合法性（kubectl apply --dry-run=server 语义）。
// 不会实际创建/更新任何资源，仅做服务端验证。
func (l *K8sLogic) DryRunYAML(ctx context.Context, content string) (*K8sApplyResult, error) {
	return l.applyYAML(ctx, content, true)
}

// applyYAML 内部统一实现 apply / dry-run。
func (l *K8sLogic) applyYAML(ctx context.Context, content string, dryRun bool) (*K8sApplyResult, error) {
	result := &K8sApplyResult{OK: true}

	restCfg, err := l.restConfig()
	if err != nil {
		return nil, err
	}

	dynCli, err := dynamic.NewForConfig(restCfg)
	if err != nil {
		return nil, err
	}

	mapper, err := l.restMapper(restCfg)
	if err != nil {
		return nil, err
	}

	// 将 YAML 按多文档拆分（--- 分隔）。
	docReader := yaml.NewDocumentDecoder(io.NopCloser(bytes.NewReader([]byte(content))))
	defer docReader.Close()

	buf := make([]byte, 0, 1024*1024)
	for {
		chunk, err := readChunk(docReader, &buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			result.OK = false
			result.Message = fmt.Sprintf("read YAML document failed: %v", err)
			return result, nil
		}
		if len(bytes.TrimSpace(chunk)) == 0 {
			continue
		}

		var item K8sApplyItem
		if dryRun {
			item = l.dryRunOneDoc(ctx, dynCli, mapper, chunk)
		} else {
			item = l.applyOneDoc(ctx, dynCli, mapper, chunk)
		}
		result.Items = append(result.Items, item)
		if item.Action == "failed" {
			result.OK = false
			result.Message = item.Message
		}
	}

	verb := "applied"
	if dryRun {
		verb = "validated"
	}
	if len(result.Items) == 0 {
		result.OK = false
		result.Message = "no valid YAML documents found"
	} else {
		result.Message = fmt.Sprintf("%s %d resource(s)", verb, len(result.Items))
	}
	return result, nil
}

// readChunk 从多文档解码器读取一个 YAML 文档。
//
// DocumentDecoder 每次 Read 返回一个完整文档（err=nil）；当文档大于内部
// 缓冲（4096）时返回 io.ErrShortBuffer 表示需继续读取剩余部分。
// 因此 err==nil 时应立即返回当前文档；ErrShortBuffer 时继续累积直到读完。
func readChunk(r io.Reader, buf *[]byte) ([]byte, error) {
	*buf = (*buf)[:0]
	tmp := make([]byte, 4096)
	for {
		n, err := r.Read(tmp)
		if n > 0 {
			*buf = append(*buf, tmp[:n]...)
		}
		if err == io.EOF {
			if len(*buf) == 0 {
				return nil, io.EOF
			}
			return *buf, nil
		}
		if err == io.ErrShortBuffer {
			// 文档尚未读完（超过 4096），继续读取剩余部分。
			continue
		}
		if err != nil {
			return nil, err
		}
		// err == nil：已读到完整的一个文档。
		return *buf, nil
	}
}

// parseYAMLDoc 将单个 YAML 文档解析为 unstructured 对象。
func parseYAMLDoc(raw []byte) (*unstructured.Unstructured, K8sApplyItem, error) {
	jsonBytes, err := yaml.ToJSON(raw)
	if err != nil {
		return nil, K8sApplyItem{Action: "failed", Message: fmt.Sprintf("convert YAML to JSON: %v", err)}, err
	}

	obj := &unstructured.Unstructured{}
	if err := obj.UnmarshalJSON(jsonBytes); err != nil {
		return nil, K8sApplyItem{Action: "failed", Message: fmt.Sprintf("unmarshal: %v", err)}, err
	}
	return obj, K8sApplyItem{}, nil
}

// resolveResource 根据 unstructured 对象解析 RESTMapping 和 ResourceInterface。
func (l *K8sLogic) resolveResource(dynCli dynamic.Interface, mapper meta.RESTMapper, obj *unstructured.Unstructured) (dynamic.ResourceInterface, K8sApplyItem, error) {
	gvk := obj.GroupVersionKind()
	item := K8sApplyItem{
		Kind:      gvk.Kind,
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}

	gk := schema.GroupKind{Group: gvk.Group, Kind: gvk.Kind}
	mapping, err := mapper.RESTMapping(gk, gvk.Version)
	if err != nil {
		item.Action = "failed"
		item.Message = fmt.Sprintf("no REST mapping for %s: %v", gvk.String(), err)
		return nil, item, err
	}

	var r dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		namespace := obj.GetNamespace()
		if namespace == "" {
			namespace = l.namespace()
		}
		r = dynCli.Resource(mapping.Resource).Namespace(namespace)
		item.Namespace = namespace
	} else {
		r = dynCli.Resource(mapping.Resource)
	}
	return r, item, nil
}

// dryRunOneDoc 验证单个 YAML 文档（服务端 dry-run）。
func (l *K8sLogic) dryRunOneDoc(ctx context.Context, dynCli dynamic.Interface, mapper meta.RESTMapper, raw []byte) K8sApplyItem {
	obj, item, err := parseYAMLDoc(raw)
	if err != nil {
		return item
	}

	r, item, err := l.resolveResource(dynCli, mapper, obj)
	if err != nil {
		return item
	}

	// 先尝试 Create（dry-run），验证资源合法性。
	createOpts := metav1.CreateOptions{DryRun: []string{"All"}}
	_, createErr := r.Create(ctx, obj, createOpts)

	if createErr == nil {
		item.Action = "valid"
		item.Message = "resource is valid (dry-run create)"
		return item
	}
	if !apierrors.IsAlreadyExists(createErr) {
		item.Action = "failed"
		item.Message = createErr.Error()
		return item
	}

	// 已存在 → 尝试 Update（dry-run），验证更新合法性。
	existing, err := r.Get(ctx, obj.GetName(), metav1.GetOptions{})
	if err != nil {
		item.Action = "failed"
		item.Message = fmt.Sprintf("get existing: %v", err)
		return item
	}
	obj.SetResourceVersion(existing.GetResourceVersion())
	updateOpts := metav1.UpdateOptions{DryRun: []string{"All"}}
	if _, err = r.Update(ctx, obj, updateOpts); err != nil {
		item.Action = "failed"
		item.Message = err.Error()
		return item
	}
	item.Action = "valid"
	item.Message = "resource is valid (dry-run update)"
	return item
}

// deleteOneDoc 删除单个 YAML 文档对应的资源。
func (l *K8sLogic) deleteOneDoc(ctx context.Context, dynCli dynamic.Interface, mapper meta.RESTMapper, raw []byte) K8sApplyItem {
	obj, item, err := parseYAMLDoc(raw)
	if err != nil {
		return item
	}

	r, item, err := l.resolveResource(dynCli, mapper, obj)
	if err != nil {
		return item
	}

	if err := r.Delete(ctx, obj.GetName(), metav1.DeleteOptions{}); err != nil {
		if apierrors.IsNotFound(err) {
			item.Action = "skipped"
			item.Message = "resource not found"
			return item
		}
		item.Action = "failed"
		item.Message = err.Error()
		logger.ErrorCtx(ctx, "k8s delete yaml failed",
			logger.String("kind", item.Kind), logger.String("name", item.Name), logger.Err(err))
		return item
	}
	item.Action = "deleted"
	return item
}

// DeleteYAML 按多文档 YAML 删除资源（kubectl delete -f 语义）。
func (l *K8sLogic) DeleteYAML(ctx context.Context, content string) (*K8sApplyResult, error) {
	result := &K8sApplyResult{OK: true}

	restCfg, err := l.restConfig()
	if err != nil {
		return nil, err
	}

	dynCli, err := dynamic.NewForConfig(restCfg)
	if err != nil {
		return nil, err
	}

	mapper, err := l.restMapper(restCfg)
	if err != nil {
		return nil, err
	}

	docReader := yaml.NewDocumentDecoder(io.NopCloser(bytes.NewReader([]byte(content))))
	defer docReader.Close()

	buf := make([]byte, 0, 1024*1024)
	for {
		chunk, err := readChunk(docReader, &buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			result.OK = false
			result.Message = fmt.Sprintf("read YAML document failed: %v", err)
			return result, nil
		}
		if len(bytes.TrimSpace(chunk)) == 0 {
			continue
		}

		item := l.deleteOneDoc(ctx, dynCli, mapper, chunk)
		result.Items = append(result.Items, item)
		if item.Action == "failed" {
			result.OK = false
			result.Message = item.Message
		}
	}

	if len(result.Items) == 0 {
		result.OK = false
		result.Message = "no valid YAML documents found"
	} else {
		result.Message = fmt.Sprintf("deleted %d resource(s)", len(result.Items))
	}
	return result, nil
}

// applyOneDoc 将单个 YAML 文档 apply 到集群（统一走 unstructured + dynamic client）。
func (l *K8sLogic) applyOneDoc(ctx context.Context, dynCli dynamic.Interface, mapper meta.RESTMapper, raw []byte) K8sApplyItem {
	obj, item, err := parseYAMLDoc(raw)
	if err != nil {
		return item
	}

	r, item, err := l.resolveResource(dynCli, mapper, obj)
	if err != nil {
		return item
	}

	// 尝试 Create，已存在则 Update。
	_, err = r.Create(ctx, obj, metav1.CreateOptions{})
	if err == nil {
		item.Action = "created"
		return item
	}
	if !apierrors.IsAlreadyExists(err) {
		item.Action = "failed"
		item.Message = err.Error()
		logger.ErrorCtx(ctx, "k8s apply create failed",
			logger.String("kind", item.Kind), logger.String("name", item.Name), logger.Err(err))
		return item
	}

	// 已存在 → Get → SetRV → Update。
	existing, err := r.Get(ctx, obj.GetName(), metav1.GetOptions{})
	if err != nil {
		item.Action = "failed"
		item.Message = fmt.Sprintf("get existing: %v", err)
		return item
	}
	obj.SetResourceVersion(existing.GetResourceVersion())
	if _, err = r.Update(ctx, obj, metav1.UpdateOptions{}); err != nil {
		item.Action = "failed"
		item.Message = err.Error()
		logger.ErrorCtx(ctx, "k8s apply update failed",
			logger.String("kind", item.Kind), logger.String("name", item.Name), logger.Err(err))
		return item
	}
	item.Action = "updated"
	return item
}

// restMapper 创建 RESTMapper（用于 GVK → 资源映射，基于 discovery 缓存）。
func (l *K8sLogic) restMapper(restCfg *rest.Config) (meta.RESTMapper, error) {
	discoveryCli, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, err
	}
	groupResources, err := restmapper.GetAPIGroupResources(discoveryCli.Discovery())
	if err != nil {
		return nil, err
	}
	return restmapper.NewDiscoveryRESTMapper(groupResources), nil
}

// K8sDeleteResourceRequest 单个批量删除请求项。
type K8sDeleteResourceRequest struct {
	Kind      string `json:"kind"`                // 资源类型，如 Pod / Deployment / Service
	Name      string `json:"name"`                // 资源名称
	Namespace string `json:"namespace,omitempty"` // 命名空间（集群级资源可空）
}

// DeleteResources 按资源列表批量删除（支持跨类型、跨命名空间）。
// NotFound 的资源视为已删除（幂等），返回 skipped。
func (l *K8sLogic) DeleteResources(ctx context.Context, reqs []K8sDeleteResourceRequest) (*K8sApplyResult, error) {
	result := &K8sApplyResult{OK: true}

	restCfg, err := l.restConfig()
	if err != nil {
		return nil, err
	}

	dynCli, err := dynamic.NewForConfig(restCfg)
	if err != nil {
		return nil, err
	}

	mapper, err := l.restMapper(restCfg)
	if err != nil {
		return nil, err
	}

	for _, req := range reqs {
		item := K8sApplyItem{
			Kind:      req.Kind,
			Name:      req.Name,
			Namespace: req.Namespace,
		}

		gk := schema.GroupKind{Kind: req.Kind}
		mapping, err := mapper.RESTMapping(gk)
		if err != nil {
			item.Action = "failed"
			item.Message = fmt.Sprintf("no REST mapping for %s: %v", req.Kind, err)
			result.Items = append(result.Items, item)
			result.OK = false
			result.Message = item.Message
			continue
		}

		var r dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			ns := req.Namespace
			if ns == "" {
				ns = l.namespace()
			}
			r = dynCli.Resource(mapping.Resource).Namespace(ns)
			item.Namespace = ns
		} else {
			r = dynCli.Resource(mapping.Resource)
		}

		if err := r.Delete(ctx, req.Name, metav1.DeleteOptions{}); err != nil {
			if apierrors.IsNotFound(err) {
				item.Action = "skipped"
				item.Message = "resource not found"
			} else {
				item.Action = "failed"
				item.Message = err.Error()
				result.OK = false
				result.Message = item.Message
				logger.ErrorCtx(ctx, "k8s batch delete failed",
					logger.String("kind", req.Kind), logger.String("name", req.Name), logger.Err(err))
			}
		} else {
			item.Action = "deleted"
		}

		result.Items = append(result.Items, item)
	}

	if len(result.Items) == 0 {
		result.OK = false
		result.Message = "no resources to delete"
	} else {
		result.Message = fmt.Sprintf("processed %d resource(s)", len(result.Items))
	}
	return result, nil
}
