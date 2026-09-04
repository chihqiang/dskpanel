package handler

import (
	"net/http"

	"github.com/chihqiang/infra-go/httpx"

	"chihqiang/dskpanel/logic"
)

// ListConfigMaps ConfigMap 列表（支持 labelSelector / fieldSelector 过滤）。
func (h *K8sHandler) ListConfigMaps(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Namespace     string `form:"namespace"`
		LabelSelector string `form:"labelSelector"`
		FieldSelector string `form:"fieldSelector"`
		Limit         int64  `form:"limit"`
	}
	if err := httpx.MustBindQuery(w, r, &req); err != nil {
		return
	}
	items, err := h.ctx.K8sLogic.ListConfigMapsWithOptions(r.Context(), logic.K8sListOptions{
		Namespace:     req.Namespace,
		LabelSelector: req.LabelSelector,
		FieldSelector: req.FieldSelector,
		Limit:         req.Limit,
	})
	if err != nil {
		writeK8sError(w, err)
		return
	}
	writeK8sList(w, r, items)
}

// InspectConfigMap ConfigMap 详情（支持 ?format=yaml）。
func (h *K8sHandler) InspectConfigMap(w http.ResponseWriter, r *http.Request) {
	namespace := httpx.QueryValue[string](r, "namespace")
	cm, err := h.ctx.K8sLogic.InspectConfigMap(r.Context(), namespace, r.PathValue("name"))
	if err != nil {
		writeK8sError(w, err)
		return
	}
	writeK8sObject(w, r, cm)
}

// DeleteConfigMap 删除 ConfigMap（NotFound 幂等）。
func (h *K8sHandler) DeleteConfigMap(w http.ResponseWriter, r *http.Request) {
	namespace := httpx.QueryValue[string](r, "namespace")
	err := h.ctx.K8sLogic.DeleteConfigMap(r.Context(), namespace, r.PathValue("name"))
	writeK8sDeleteResult(w, err)
}

// ListSecrets Secret 列表（支持 labelSelector / fieldSelector 过滤）。
func (h *K8sHandler) ListSecrets(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Namespace     string `form:"namespace"`
		LabelSelector string `form:"labelSelector"`
		FieldSelector string `form:"fieldSelector"`
		Limit         int64  `form:"limit"`
	}
	if err := httpx.MustBindQuery(w, r, &req); err != nil {
		return
	}
	items, err := h.ctx.K8sLogic.ListSecretsWithOptions(r.Context(), logic.K8sListOptions{
		Namespace:     req.Namespace,
		LabelSelector: req.LabelSelector,
		FieldSelector: req.FieldSelector,
		Limit:         req.Limit,
	})
	if err != nil {
		writeK8sError(w, err)
		return
	}
	writeK8sList(w, r, items)
}

// InspectSecret Secret 详情（JSON 返回脱敏摘要，?format=yaml 返回脱敏 YAML）。
func (h *K8sHandler) InspectSecret(w http.ResponseWriter, r *http.Request) {
	namespace := httpx.QueryValue[string](r, "namespace")
	if r.URL.Query().Get("format") == "yaml" {
		sec, err := h.ctx.K8sLogic.InspectSecretRaw(r.Context(), namespace, r.PathValue("name"))
		if err != nil {
			writeK8sError(w, err)
			return
		}
		writeK8sObject(w, r, sec)
		return
	}
	detail, err := h.ctx.K8sLogic.InspectSecretDetail(r.Context(), namespace, r.PathValue("name"))
	if err != nil {
		writeK8sError(w, err)
		return
	}
	httpx.OkJSON(w, detail)
}

// DeleteSecret 删除 Secret（NotFound 幂等）。
func (h *K8sHandler) DeleteSecret(w http.ResponseWriter, r *http.Request) {
	namespace := httpx.QueryValue[string](r, "namespace")
	err := h.ctx.K8sLogic.DeleteSecret(r.Context(), namespace, r.PathValue("name"))
	writeK8sDeleteResult(w, err)
}
