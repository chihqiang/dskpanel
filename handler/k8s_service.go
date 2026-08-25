package handler

import (
	"net/http"

	"github.com/chihqiang/infra-go/httpx"

	"chihqiang/dskpanel/logic"
)

// ListServices Service 列表（支持 labelSelector / fieldSelector 过滤）。
func (h *K8sHandler) ListServices(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Namespace     string `form:"namespace"`
		LabelSelector string `form:"labelSelector"`
		FieldSelector string `form:"fieldSelector"`
		Limit         int64  `form:"limit"`
	}
	if err := httpx.MustBindQuery(w, r, &req); err != nil {
		return
	}
	items, err := h.ctx.K8sLogic.ListServicesWithOptions(r.Context(), logic.K8sListOptions{
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

// InspectService Service 详情（支持 ?format=yaml）。
func (h *K8sHandler) InspectService(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")
	svc, err := h.ctx.K8sLogic.InspectService(r.Context(), namespace, r.PathValue("name"))
	if err != nil {
		writeK8sError(w, err)
		return
	}
	writeK8sObject(w, r, svc)
}

// DeleteService 删除 Service（NotFound 幂等）。
func (h *K8sHandler) DeleteService(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")
	err := h.ctx.K8sLogic.DeleteService(r.Context(), namespace, r.PathValue("name"))
	writeK8sDeleteResult(w, err)
}

// ListIngresses Ingress 列表（支持 labelSelector / fieldSelector 过滤）。
func (h *K8sHandler) ListIngresses(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Namespace     string `form:"namespace"`
		LabelSelector string `form:"labelSelector"`
		FieldSelector string `form:"fieldSelector"`
		Limit         int64  `form:"limit"`
	}
	if err := httpx.MustBindQuery(w, r, &req); err != nil {
		return
	}
	items, err := h.ctx.K8sLogic.ListIngressesWithOptions(r.Context(), logic.K8sListOptions{
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

// InspectIngress Ingress 详情（支持 ?format=yaml）。
func (h *K8sHandler) InspectIngress(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")
	ing, err := h.ctx.K8sLogic.InspectIngress(r.Context(), namespace, r.PathValue("name"))
	if err != nil {
		writeK8sError(w, err)
		return
	}
	writeK8sObject(w, r, ing)
}

// DeleteIngress 删除 Ingress（NotFound 幂等）。
func (h *K8sHandler) DeleteIngress(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")
	err := h.ctx.K8sLogic.DeleteIngress(r.Context(), namespace, r.PathValue("name"))
	writeK8sDeleteResult(w, err)
}
