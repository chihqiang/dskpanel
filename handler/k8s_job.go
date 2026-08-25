package handler

import (
	"net/http"

	"github.com/chihqiang/infra-go/httpx"

	"chihqiang/dskpanel/logic"
)

// ListJobs Job 列表（支持 labelSelector / fieldSelector 过滤）。
func (h *K8sHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Namespace     string `form:"namespace"`
		LabelSelector string `form:"labelSelector"`
		FieldSelector string `form:"fieldSelector"`
		Limit         int64  `form:"limit"`
	}
	if err := httpx.MustBindQuery(w, r, &req); err != nil {
		return
	}
	items, err := h.ctx.K8sLogic.ListJobsWithOptions(r.Context(), logic.K8sListOptions{
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

// InspectJob Job 详情（支持 ?format=yaml）。
func (h *K8sHandler) InspectJob(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")
	job, err := h.ctx.K8sLogic.InspectJob(r.Context(), namespace, r.PathValue("name"))
	if err != nil {
		writeK8sError(w, err)
		return
	}
	writeK8sObject(w, r, job)
}

// DeleteJob 删除 Job（NotFound 幂等）。
func (h *K8sHandler) DeleteJob(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")
	err := h.ctx.K8sLogic.DeleteJob(r.Context(), namespace, r.PathValue("name"))
	writeK8sDeleteResult(w, err)
}

// ListCronJobs CronJob 列表（支持 labelSelector / fieldSelector 过滤）。
func (h *K8sHandler) ListCronJobs(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Namespace     string `form:"namespace"`
		LabelSelector string `form:"labelSelector"`
		FieldSelector string `form:"fieldSelector"`
		Limit         int64  `form:"limit"`
	}
	if err := httpx.MustBindQuery(w, r, &req); err != nil {
		return
	}
	items, err := h.ctx.K8sLogic.ListCronJobsWithOptions(r.Context(), logic.K8sListOptions{
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

// InspectCronJob CronJob 详情（支持 ?format=yaml）。
func (h *K8sHandler) InspectCronJob(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")
	cj, err := h.ctx.K8sLogic.InspectCronJob(r.Context(), namespace, r.PathValue("name"))
	if err != nil {
		writeK8sError(w, err)
		return
	}
	writeK8sObject(w, r, cj)
}

// DeleteCronJob 删除 CronJob（NotFound 幂等）。
func (h *K8sHandler) DeleteCronJob(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")
	err := h.ctx.K8sLogic.DeleteCronJob(r.Context(), namespace, r.PathValue("name"))
	writeK8sDeleteResult(w, err)
}
