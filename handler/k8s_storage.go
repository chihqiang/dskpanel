package handler

import (
	"net/http"

	"github.com/chihqiang/infra-go/httpx"

	"chihqiang/dskpanel/logic"
)

// ListPVCs PVC 列表（支持 labelSelector / fieldSelector 过滤）。
func (h *K8sHandler) ListPVCs(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Namespace     string `form:"namespace"`
		LabelSelector string `form:"labelSelector"`
		FieldSelector string `form:"fieldSelector"`
		Limit         int64  `form:"limit"`
	}
	if err := httpx.MustBindQuery(w, r, &req); err != nil {
		return
	}
	items, err := h.ctx.K8sLogic.ListPVCsWithOptions(r.Context(), logic.K8sListOptions{
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

// InspectPVC PVC 详情（支持 ?format=yaml）。
func (h *K8sHandler) InspectPVC(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")
	pvc, err := h.ctx.K8sLogic.InspectPVC(r.Context(), namespace, r.PathValue("name"))
	if err != nil {
		writeK8sError(w, err)
		return
	}
	writeK8sObject(w, r, pvc)
}

// DeletePVC 删除 PVC（NotFound 幂等）。
func (h *K8sHandler) DeletePVC(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")
	err := h.ctx.K8sLogic.DeletePVC(r.Context(), namespace, r.PathValue("name"))
	writeK8sDeleteResult(w, err)
}

// ListStorageClasses StorageClass 列表。
func (h *K8sHandler) ListStorageClasses(w http.ResponseWriter, r *http.Request) {
	items, err := h.ctx.K8sLogic.ListStorageClasses(r.Context())
	if err != nil {
		writeK8sError(w, err)
		return
	}
	writeK8sList(w, r, items)
}

// InspectStorageClass StorageClass 详情（支持 ?format=yaml）。
func (h *K8sHandler) InspectStorageClass(w http.ResponseWriter, r *http.Request) {
	sc, err := h.ctx.K8sLogic.InspectStorageClass(r.Context(), r.PathValue("name"))
	if err != nil {
		writeK8sError(w, err)
		return
	}
	writeK8sObject(w, r, sc)
}
