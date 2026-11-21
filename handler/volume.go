package handler

import (
	"net/http"

	"github.com/chihqiang/ten-go/rest"

	"chihqiang/dskpanel/svc"
)

// VolumeHandler 卷管理处理器。
type VolumeHandler struct {
	ctx *svc.AppContext
}

// NewVolumeHandler 创建卷管理处理器。
func NewVolumeHandler(ctx *svc.AppContext) *VolumeHandler {
	return &VolumeHandler{ctx: ctx}
}

// List 卷列表。
func (h *VolumeHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.ctx.VolumeLogic.List(r.Context())
	if err != nil {
		rest.ErrorJSON(w, rest.NewCodeError(http.StatusInternalServerError, err.Error()))
		return
	}
	rest.OkJSON(w, items)
}

// Create 创建卷。
func (h *VolumeHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name   string `json:"name" binding:"required"`
		Driver string `json:"driver,default=local"`
	}
	if err := rest.Bind(r, &req); err != nil {
		_ = rest.ErrorJSONCtx(r.Context(), w, rest.NewCodeError(rest.StatusOf(err), err.Error()))
		return
	}
	if err := h.ctx.VolumeLogic.Create(r.Context(), req.Name, req.Driver); err != nil {
		rest.ErrorJSON(w, rest.NewCodeError(http.StatusInternalServerError, err.Error()))
		return
	}
	rest.OkJSON(w, "ok")
}

// Inspect 卷详情。
func (h *VolumeHandler) Inspect(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		rest.ErrorJSON(w, rest.NewCodeError(http.StatusBadRequest, "missing volume name"))
		return
	}
	detail, err := h.ctx.VolumeLogic.Inspect(r.Context(), name)
	if err != nil {
		rest.ErrorJSON(w, rest.NewCodeError(http.StatusInternalServerError, err.Error()))
		return
	}
	rest.OkJSON(w, detail)
}

// Remove 删除卷。
func (h *VolumeHandler) Remove(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Force bool `form:"force,default=false"`
	}
	if err := rest.BindQuery(r, &req); err != nil {
		_ = rest.ErrorJSONCtx(r.Context(), w, rest.NewCodeError(rest.StatusOf(err), err.Error()))
		return
	}
	name := r.PathValue("name")
	if err := h.ctx.VolumeLogic.Remove(r.Context(), name, req.Force); err != nil {
		rest.ErrorJSON(w, rest.NewCodeError(http.StatusInternalServerError, err.Error()))
		return
	}
	rest.OkJSON(w, "ok")
}

// Prune 清理未使用卷。
func (h *VolumeHandler) Prune(w http.ResponseWriter, r *http.Request) {
	deleted, reclaimed, err := h.ctx.VolumeLogic.Prune(r.Context())
	if err != nil {
		rest.ErrorJSON(w, rest.NewCodeError(http.StatusInternalServerError, err.Error()))
		return
	}
	rest.OkJSON(w, map[string]any{"deleted": deleted, "reclaimed_bytes": reclaimed})
}
