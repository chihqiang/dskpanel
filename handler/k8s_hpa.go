package handler

import (
	"net/http"

	"github.com/chihqiang/infra-go/httpx"

	"chihqiang/dskpanel/logic"
)

// ListHPAs HPA 列表（支持 labelSelector / fieldSelector 过滤）。
func (h *K8sHandler) ListHPAs(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Namespace     string `form:"namespace"`
		LabelSelector string `form:"labelSelector"`
		FieldSelector string `form:"fieldSelector"`
		Limit         int64  `form:"limit"`
	}
	if err := httpx.MustBindQuery(w, r, &req); err != nil {
		return
	}
	items, err := h.ctx.K8sLogic.ListHPAsWithOptions(r.Context(), logic.K8sListOptions{
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

// InspectHPA HPA 详情（支持 ?format=yaml）。
func (h *K8sHandler) InspectHPA(w http.ResponseWriter, r *http.Request) {
	namespace := httpx.QueryValue[string](r, "namespace")
	hpa, err := h.ctx.K8sLogic.InspectHPA(r.Context(), namespace, r.PathValue("name"))
	if err != nil {
		writeK8sError(w, err)
		return
	}
	writeK8sObject(w, r, hpa)
}

// DeleteHPA 删除 HPA（NotFound 幂等）。
func (h *K8sHandler) DeleteHPA(w http.ResponseWriter, r *http.Request) {
	namespace := httpx.QueryValue[string](r, "namespace")
	err := h.ctx.K8sLogic.DeleteHPA(r.Context(), namespace, r.PathValue("name"))
	writeK8sDeleteResult(w, err)
}
