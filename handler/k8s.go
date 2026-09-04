package handler

import (
	"errors"
	"net/http"

	"github.com/chihqiang/infra-go/httpx"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"

	"chihqiang/dskpanel/logic"
	"chihqiang/dskpanel/svc"
)

// K8sHandler Kubernetes 集群管理处理器。
type K8sHandler struct {
	ctx *svc.AppContext
}

// NewK8sHandler 创建 K8s 处理器。
func NewK8sHandler(ctx *svc.AppContext) *K8sHandler {
	return &K8sHandler{ctx: ctx}
}

// writeK8sError 统一错误响应，将 K8s API 错误映射为合适的 HTTP 状态码。
func writeK8sError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, logic.ErrK8sNotAvailable):
		httpx.WriteHTTPError(w, http.StatusConflict, err.Error())
	case apierrors.IsNotFound(err):
		httpx.WriteHTTPError(w, http.StatusNotFound, err.Error())
	case apierrors.IsAlreadyExists(err):
		httpx.WriteHTTPError(w, http.StatusConflict, err.Error())
	case apierrors.IsForbidden(err):
		httpx.WriteHTTPError(w, http.StatusForbidden, err.Error())
	case apierrors.IsUnauthorized(err):
		httpx.WriteHTTPError(w, http.StatusUnauthorized, err.Error())
	default:
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
	}
}

// writeK8sDeleteResult 统一处理 Delete 接口的响应。
// 如果资源不存在（NotFound），幂等返回 200 "ok" 而非 404。
func writeK8sDeleteResult(w http.ResponseWriter, err error) {
	if err == nil {
		httpx.OkJSON(w, "ok")
		return
	}
	if apierrors.IsNotFound(err) {
		httpx.OkJSON(w, "ok")
		return
	}
	writeK8sError(w, err)
}

// writeK8sObject 统一输出 K8s 资源对象（JSON / YAML）。
// ?format=yaml 时输出 text/yaml，否则默认 JSON。
func writeK8sObject(w http.ResponseWriter, r *http.Request, obj runtime.Object) {
	if r.URL.Query().Get("format") == "yaml" {
		data, err := yaml.Marshal(obj)
		if err != nil {
			httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
		return
	}
	httpx.OkJSON(w, obj)
}

// writeK8sList 统一输出列表数据（JSON / YAML）。
// ?format=yaml 时输出 text/yaml，否则默认 JSON。
func writeK8sList(w http.ResponseWriter, r *http.Request, items any) {
	if r.URL.Query().Get("format") == "yaml" {
		data, err := yaml.Marshal(items)
		if err != nil {
			httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
		return
	}
	httpx.OkJSON(w, items)
}

// Detect 检测 K8s 集群状态。
func (h *K8sHandler) Detect(w http.ResponseWriter, r *http.Request) {
	st := h.ctx.K8sLogic.Detect(r.Context())
	httpx.OkJSON(w, st)
}

// Overview 集群概览。
func (h *K8sHandler) Overview(w http.ResponseWriter, r *http.Request) {
	ov, err := h.ctx.K8sLogic.Overview(r.Context())
	if err != nil {
		writeK8sError(w, err)
		return
	}
	httpx.OkJSON(w, ov)
}

// ListEvents 事件列表。
func (h *K8sHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Namespace     string `form:"namespace"`
		FieldSelector string `form:"fieldSelector"`
	}
	if err := httpx.MustBindQuery(w, r, &req); err != nil {
		return
	}
	items, err := h.ctx.K8sLogic.ListEvents(r.Context(), req.Namespace, req.FieldSelector)
	if err != nil {
		writeK8sError(w, err)
		return
	}
	writeK8sList(w, r, items)
}

// ListEventsForResource 查询特定资源的事件（如 Pod / Deployment / Node）。
func (h *K8sHandler) ListEventsForResource(w http.ResponseWriter, r *http.Request) {
	namespace := httpx.QueryValue[string](r, "namespace")
	kind := r.PathValue("kind")
	name := r.PathValue("name")
	items, err := h.ctx.K8sLogic.ListEventsForObject(r.Context(), namespace, kind, name)
	if err != nil {
		writeK8sError(w, err)
		return
	}
	writeK8sList(w, r, items)
}

// ListNamespaces 命名空间列表。
func (h *K8sHandler) ListNamespaces(w http.ResponseWriter, r *http.Request) {
	items, err := h.ctx.K8sLogic.ListNamespaces(r.Context())
	if err != nil {
		writeK8sError(w, err)
		return
	}
	writeK8sList(w, r, items)
}

// InspectNamespace 命名空间详情（支持 ?format=yaml）。
func (h *K8sHandler) InspectNamespace(w http.ResponseWriter, r *http.Request) {
	ns, err := h.ctx.K8sLogic.InspectNamespace(r.Context(), r.PathValue("name"))
	if err != nil {
		writeK8sError(w, err)
		return
	}
	writeK8sObject(w, r, ns)
}

// DeleteNamespace 删除命名空间（NotFound 幂等）。
func (h *K8sHandler) DeleteNamespace(w http.ResponseWriter, r *http.Request) {
	err := h.ctx.K8sLogic.DeleteNamespace(r.Context(), r.PathValue("name"))
	writeK8sDeleteResult(w, err)
}
