package handler

import (
	"net/http"

	"github.com/chihqiang/infra-go/httpx"

	"chihqiang/dskpanel/logic"
)

// ListDeployments Deployment 列表（支持 labelSelector / fieldSelector 过滤）。
func (h *K8sHandler) ListDeployments(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Namespace     string `form:"namespace"`
		LabelSelector string `form:"labelSelector"`
		FieldSelector string `form:"fieldSelector"`
		Limit         int64  `form:"limit"`
	}
	if err := httpx.MustBindQuery(w, r, &req); err != nil {
		return
	}
	items, err := h.ctx.K8sLogic.ListDeploymentsWithOptions(r.Context(), logic.K8sListOptions{
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

// InspectDeployment Deployment 详情（支持 ?format=yaml）。
func (h *K8sHandler) InspectDeployment(w http.ResponseWriter, r *http.Request) {
	namespace := httpx.QueryValue[string](r, "namespace")
	dep, err := h.ctx.K8sLogic.InspectDeployment(r.Context(), namespace, r.PathValue("name"))
	if err != nil {
		writeK8sError(w, err)
		return
	}
	writeK8sObject(w, r, dep)
}

// DeleteDeployment 删除 Deployment（NotFound 幂等）。
func (h *K8sHandler) DeleteDeployment(w http.ResponseWriter, r *http.Request) {
	namespace := httpx.QueryValue[string](r, "namespace")
	err := h.ctx.K8sLogic.DeleteDeployment(r.Context(), namespace, r.PathValue("name"))
	writeK8sDeleteResult(w, err)
}

// ScaleDeployment 伸缩 Deployment。
func (h *K8sHandler) ScaleDeployment(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Replicas int32 `json:"replicas" binding:"required"`
	}
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}
	namespace := httpx.QueryValue[string](r, "namespace")
	if err := h.ctx.K8sLogic.ScaleDeployment(r.Context(), namespace, r.PathValue("name"), req.Replicas); err != nil {
		writeK8sError(w, err)
		return
	}
	httpx.OkJSON(w, "ok")
}

// RestartDeployment 重启 Deployment（滚动重启）。
func (h *K8sHandler) RestartDeployment(w http.ResponseWriter, r *http.Request) {
	namespace := httpx.QueryValue[string](r, "namespace")
	if err := h.ctx.K8sLogic.RestartDeployment(r.Context(), namespace, r.PathValue("name")); err != nil {
		writeK8sError(w, err)
		return
	}
	httpx.OkJSON(w, "ok")
}

// ListStatefulSets StatefulSet 列表（支持 labelSelector / fieldSelector 过滤）。
func (h *K8sHandler) ListStatefulSets(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Namespace     string `form:"namespace"`
		LabelSelector string `form:"labelSelector"`
		FieldSelector string `form:"fieldSelector"`
		Limit         int64  `form:"limit"`
	}
	if err := httpx.MustBindQuery(w, r, &req); err != nil {
		return
	}
	items, err := h.ctx.K8sLogic.ListStatefulSetsWithOptions(r.Context(), logic.K8sListOptions{
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

// InspectStatefulSet StatefulSet 详情（支持 ?format=yaml）。
func (h *K8sHandler) InspectStatefulSet(w http.ResponseWriter, r *http.Request) {
	namespace := httpx.QueryValue[string](r, "namespace")
	sts, err := h.ctx.K8sLogic.InspectStatefulSet(r.Context(), namespace, r.PathValue("name"))
	if err != nil {
		writeK8sError(w, err)
		return
	}
	writeK8sObject(w, r, sts)
}

// DeleteStatefulSet 删除 StatefulSet（NotFound 幂等）。
func (h *K8sHandler) DeleteStatefulSet(w http.ResponseWriter, r *http.Request) {
	namespace := httpx.QueryValue[string](r, "namespace")
	err := h.ctx.K8sLogic.DeleteStatefulSet(r.Context(), namespace, r.PathValue("name"))
	writeK8sDeleteResult(w, err)
}

// ScaleStatefulSet 伸缩 StatefulSet。
func (h *K8sHandler) ScaleStatefulSet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Replicas int32 `json:"replicas" binding:"required"`
	}
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}
	namespace := httpx.QueryValue[string](r, "namespace")
	if err := h.ctx.K8sLogic.ScaleStatefulSet(r.Context(), namespace, r.PathValue("name"), req.Replicas); err != nil {
		writeK8sError(w, err)
		return
	}
	httpx.OkJSON(w, "ok")
}

// RestartStatefulSet 重启 StatefulSet。
func (h *K8sHandler) RestartStatefulSet(w http.ResponseWriter, r *http.Request) {
	namespace := httpx.QueryValue[string](r, "namespace")
	if err := h.ctx.K8sLogic.RestartStatefulSet(r.Context(), namespace, r.PathValue("name")); err != nil {
		writeK8sError(w, err)
		return
	}
	httpx.OkJSON(w, "ok")
}

// ListDaemonSets DaemonSet 列表（支持 labelSelector / fieldSelector 过滤）。
func (h *K8sHandler) ListDaemonSets(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Namespace     string `form:"namespace"`
		LabelSelector string `form:"labelSelector"`
		FieldSelector string `form:"fieldSelector"`
		Limit         int64  `form:"limit"`
	}
	if err := httpx.MustBindQuery(w, r, &req); err != nil {
		return
	}
	items, err := h.ctx.K8sLogic.ListDaemonSetsWithOptions(r.Context(), logic.K8sListOptions{
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

// InspectDaemonSet DaemonSet 详情（支持 ?format=yaml）。
func (h *K8sHandler) InspectDaemonSet(w http.ResponseWriter, r *http.Request) {
	namespace := httpx.QueryValue[string](r, "namespace")
	ds, err := h.ctx.K8sLogic.InspectDaemonSet(r.Context(), namespace, r.PathValue("name"))
	if err != nil {
		writeK8sError(w, err)
		return
	}
	writeK8sObject(w, r, ds)
}

// DeleteDaemonSet 删除 DaemonSet（NotFound 幂等）。
func (h *K8sHandler) DeleteDaemonSet(w http.ResponseWriter, r *http.Request) {
	namespace := httpx.QueryValue[string](r, "namespace")
	err := h.ctx.K8sLogic.DeleteDaemonSet(r.Context(), namespace, r.PathValue("name"))
	writeK8sDeleteResult(w, err)
}

// RestartDaemonSet 重启 DaemonSet。
func (h *K8sHandler) RestartDaemonSet(w http.ResponseWriter, r *http.Request) {
	namespace := httpx.QueryValue[string](r, "namespace")
	if err := h.ctx.K8sLogic.RestartDaemonSet(r.Context(), namespace, r.PathValue("name")); err != nil {
		writeK8sError(w, err)
		return
	}
	httpx.OkJSON(w, "ok")
}
