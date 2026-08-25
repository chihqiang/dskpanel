package handler

import (
	"net/http"

	"github.com/chihqiang/infra-go/httpx"

	"chihqiang/dskpanel/logic"
)

// ListNodes 节点列表（支持 labelSelector 过滤）。
func (h *K8sHandler) ListNodes(w http.ResponseWriter, r *http.Request) {
	var req struct {
		LabelSelector string `form:"labelSelector"`
		FieldSelector string `form:"fieldSelector"`
	}
	if err := httpx.MustBindQuery(w, r, &req); err != nil {
		return
	}
	items, err := h.ctx.K8sLogic.ListNodes(r.Context(), logic.K8sListOptions{
		LabelSelector: req.LabelSelector,
		FieldSelector: req.FieldSelector,
	})
	if err != nil {
		writeK8sError(w, err)
		return
	}
	writeK8sList(w, r, items)
}

// InspectNode 节点详情（支持 ?format=yaml）。
func (h *K8sHandler) InspectNode(w http.ResponseWriter, r *http.Request) {
	node, err := h.ctx.K8sLogic.InspectNode(r.Context(), r.PathValue("name"))
	if err != nil {
		writeK8sError(w, err)
		return
	}
	writeK8sObject(w, r, node)
}

// CordonNode 节点设置为不可调度。
func (h *K8sHandler) CordonNode(w http.ResponseWriter, r *http.Request) {
	if err := h.ctx.K8sLogic.CordonNode(r.Context(), r.PathValue("name")); err != nil {
		writeK8sError(w, err)
		return
	}
	httpx.OkJSON(w, "ok")
}

// UncordonNode 节点恢复调度。
func (h *K8sHandler) UncordonNode(w http.ResponseWriter, r *http.Request) {
	if err := h.ctx.K8sLogic.UncordonNode(r.Context(), r.PathValue("name")); err != nil {
		writeK8sError(w, err)
		return
	}
	httpx.OkJSON(w, "ok")
}

// DrainNode 驱逐节点上的 Pod。
func (h *K8sHandler) DrainNode(w http.ResponseWriter, r *http.Request) {
	if err := h.ctx.K8sLogic.DrainNode(r.Context(), r.PathValue("name")); err != nil {
		writeK8sError(w, err)
		return
	}
	httpx.OkJSON(w, "ok")
}

// NodeUsage 节点资源使用率。
func (h *K8sHandler) NodeUsage(w http.ResponseWriter, r *http.Request) {
	usage, err := h.ctx.K8sLogic.NodeUsage(r.Context(), r.PathValue("name"))
	if err != nil {
		writeK8sError(w, err)
		return
	}
	httpx.OkJSON(w, usage)
}
