package handler

import (
	"net/http"

	"github.com/chihqiang/infra-go/httpx"

	"chihqiang/dskpanel/logic"
)

// ApplyYAML YAML 透传接口（kubectl apply 语义，支持多文档 YAML）。
func (h *K8sHandler) ApplyYAML(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}
	result, err := h.ctx.K8sLogic.ApplyYAML(r.Context(), req.Content)
	if err != nil {
		writeK8sError(w, err)
		return
	}
	if !result.OK {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, result.Message)
		return
	}
	httpx.OkJSON(w, result)
}

// DeleteYAML 按 YAML 删除资源接口（kubectl delete -f 语义，支持多文档 YAML）。
func (h *K8sHandler) DeleteYAML(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}
	result, err := h.ctx.K8sLogic.DeleteYAML(r.Context(), req.Content)
	if err != nil {
		writeK8sError(w, err)
		return
	}
	if !result.OK {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, result.Message)
		return
	}
	httpx.OkJSON(w, result)
}

// DryRunYAML 验证 YAML 语法和资源合法性（kubectl apply --dry-run=server 语义）。
func (h *K8sHandler) DryRunYAML(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}
	result, err := h.ctx.K8sLogic.DryRunYAML(r.Context(), req.Content)
	if err != nil {
		writeK8sError(w, err)
		return
	}
	httpx.OkJSON(w, result)
}

// DeleteResources 批量删除资源接口（支持跨类型、跨命名空间）。
// 请求体：{"items": [{"kind":"Pod","name":"my-pod","namespace":"default"}, ...]}
// NotFound 的资源视为已删除（幂等），返回 skipped。
func (h *K8sHandler) DeleteResources(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Items []logic.K8sDeleteResourceRequest `json:"items" binding:"required"`
	}
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}
	result, err := h.ctx.K8sLogic.DeleteResources(r.Context(), req.Items)
	if err != nil {
		writeK8sError(w, err)
		return
	}
	httpx.OkJSON(w, result)
}
