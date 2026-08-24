package handler

import (
	"net/http"

	"github.com/chihqiang/infra-go/httpx"

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
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, items)
}

// Create 创建网络（含高级参数：子网/网关/IP范围/内部/IPv6/标签/驱动选项）。
func (h *NetworkHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req logic.CreateNetworkRequest
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}
	id, err := h.ctx.NetworkLogic.Create(r.Context(), &req)
	if err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, map[string]string{"id": id})
}

// Inspect 网络详情。
func (h *NetworkHandler) Inspect(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httpx.WriteHTTPError(w, http.StatusBadRequest, "missing network id")
		return
	}
	detail, err := h.ctx.NetworkLogic.Inspect(r.Context(), id)
	if err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, detail)
}

// Remove 删除网络。
func (h *NetworkHandler) Remove(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.ctx.NetworkLogic.Remove(r.Context(), id); err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, "ok")
}

// Prune 清理未使用网络。
func (h *NetworkHandler) Prune(w http.ResponseWriter, r *http.Request) {
	deleted, err := h.ctx.NetworkLogic.Prune(r.Context())
	if err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, map[string]any{"deleted": deleted})
}

// ConnectContainer 将容器连接到网络。
func (h *NetworkHandler) ConnectContainer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ContainerID string `json:"container_id" binding:"required"`
		IPv4        string `json:"ipv4"`
	}
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}
	id := r.PathValue("id")
	if err := h.ctx.NetworkLogic.ConnectContainer(r.Context(), id, req.ContainerID, req.IPv4); err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, "ok")
}

// DisconnectContainer 将容器从网络断开。
func (h *NetworkHandler) DisconnectContainer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ContainerID string `json:"container_id" binding:"required"`
		Force       bool   `json:"force"`
	}
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}
	id := r.PathValue("id")
	if err := h.ctx.NetworkLogic.DisconnectContainer(r.Context(), id, req.ContainerID, req.Force); err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, "ok")
}
