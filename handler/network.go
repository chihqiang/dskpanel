package handler

import (
	"net/http"

	"github.com/chihqiang/ten-go/rest"

	"chihqiang/dskpanel/logic"
	"chihqiang/dskpanel/svc"
)

// NetworkHandler 网络管理处理器。
type NetworkHandler struct {
	ctx *svc.AppContext
}

// NewNetworkHandler 创建网络管理处理器。
func NewNetworkHandler(ctx *svc.AppContext) *NetworkHandler {
	return &NetworkHandler{ctx: ctx}
}

// List 网络列表。
func (h *NetworkHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.ctx.NetworkLogic.List(r.Context())
	if err != nil {
		rest.ErrorJSON(w, rest.NewCodeError(http.StatusInternalServerError, err.Error()))
		return
	}
	rest.OkJSON(w, items)
}

// Create 创建网络（含高级参数：子网/网关/IP范围/内部/IPv6/标签/驱动选项）。
func (h *NetworkHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req logic.CreateNetworkRequest
	if err := rest.Bind(r, &req); err != nil {
		_ = rest.ErrorJSONCtx(r.Context(), w, rest.NewCodeError(rest.StatusOf(err), err.Error()))
		return
	}
	id, err := h.ctx.NetworkLogic.Create(r.Context(), &req)
	if err != nil {
		rest.ErrorJSON(w, rest.NewCodeError(http.StatusInternalServerError, err.Error()))
		return
	}
	rest.OkJSON(w, map[string]string{"id": id})
}

// Inspect 网络详情。
func (h *NetworkHandler) Inspect(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		rest.ErrorJSON(w, rest.NewCodeError(http.StatusBadRequest, "missing network id"))
		return
	}
	detail, err := h.ctx.NetworkLogic.Inspect(r.Context(), id)
	if err != nil {
		rest.ErrorJSON(w, rest.NewCodeError(http.StatusInternalServerError, err.Error()))
		return
	}
	rest.OkJSON(w, detail)
}

// Remove 删除网络。
func (h *NetworkHandler) Remove(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.ctx.NetworkLogic.Remove(r.Context(), id); err != nil {
		rest.ErrorJSON(w, rest.NewCodeError(http.StatusInternalServerError, err.Error()))
		return
	}
	rest.OkJSON(w, "ok")
}

// Prune 清理未使用网络。
func (h *NetworkHandler) Prune(w http.ResponseWriter, r *http.Request) {
	deleted, err := h.ctx.NetworkLogic.Prune(r.Context())
	if err != nil {
		rest.ErrorJSON(w, rest.NewCodeError(http.StatusInternalServerError, err.Error()))
		return
	}
	rest.OkJSON(w, map[string]any{"deleted": deleted})
}

// ConnectContainer 将容器连接到网络。
func (h *NetworkHandler) ConnectContainer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ContainerID string `json:"container_id" binding:"required"`
		IPv4        string `json:"ipv4"`
	}
	if err := rest.Bind(r, &req); err != nil {
		_ = rest.ErrorJSONCtx(r.Context(), w, rest.NewCodeError(rest.StatusOf(err), err.Error()))
		return
	}
	id := r.PathValue("id")
	if err := h.ctx.NetworkLogic.ConnectContainer(r.Context(), id, req.ContainerID, req.IPv4); err != nil {
		rest.ErrorJSON(w, rest.NewCodeError(http.StatusInternalServerError, err.Error()))
		return
	}
	rest.OkJSON(w, "ok")
}

// DisconnectContainer 将容器从网络断开。
func (h *NetworkHandler) DisconnectContainer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ContainerID string `json:"container_id" binding:"required"`
		Force       bool   `json:"force"`
	}
	if err := rest.Bind(r, &req); err != nil {
		_ = rest.ErrorJSONCtx(r.Context(), w, rest.NewCodeError(rest.StatusOf(err), err.Error()))
		return
	}
	id := r.PathValue("id")
	if err := h.ctx.NetworkLogic.DisconnectContainer(r.Context(), id, req.ContainerID, req.Force); err != nil {
		rest.ErrorJSON(w, rest.NewCodeError(http.StatusInternalServerError, err.Error()))
		return
	}
	rest.OkJSON(w, "ok")
}
