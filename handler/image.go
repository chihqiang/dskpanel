package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/chihqiang/infra-go/httpx"

	"chihqiang/dskpanel/svc"
)

// ImageHandler 镜像管理处理器。
type ImageHandler struct {
	ctx *svc.AppContext
}

// NewImageHandler 创建镜像管理处理器。
func NewImageHandler(ctx *svc.AppContext) *ImageHandler {
	return &ImageHandler{ctx: ctx}
}

// List 镜像列表。
// query: dangling=true|false（可选，过滤悬空镜像）。
func (h *ImageHandler) List(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Dangling string `form:"dangling"`
	}
	if err := httpx.MustBindQuery(w, r, &req); err != nil {
		return
	}
	var dangling *bool
	switch req.Dangling {
	case "true":
		t := true
		dangling = &t
	case "false":
		f := false
		dangling = &f
	}
	items, err := h.ctx.ImageLogic.List(r.Context(), dangling)
	if err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, items)
}

// Pull 拉取镜像（SSE 实时推送进度）。
// Content-Type: text/event-stream，每条 Docker pull JSON 消息作为 data 事件推送。
func (h *ImageHandler) Pull(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Ref string `json:"ref" binding:"required"`
	}
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}

	body, err := h.ctx.ImageLogic.Pull(r.Context(), req.Ref)
	if err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer body.Close()

	sse := httpx.NewSSEWriter(w)
	// 先发握手事件。
	_ = sse.Event("open", "connected")

	dec := json.NewDecoder(body)
	for {
		var msg json.RawMessage
		if err := dec.Decode(&msg); err != nil {
			break
		}
		// 每个 JSON 消息作为一行 data 推送。
		_ = sse.Data(string(msg))
	}
	// 结束事件。
	_ = sse.Event("done", "success")
}

// RemoveBatch 批量删除镜像。
func (h *ImageHandler) RemoveBatch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs   []string `json:"ids" binding:"required"`
		Force bool     `json:"force,default=false"`
	}
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}
	if len(req.IDs) == 0 {
		httpx.WriteHTTPError(w, http.StatusBadRequest, "missing image ids")
		return
	}
	deleted, err := h.ctx.ImageLogic.RemoveBatch(r.Context(), req.IDs, req.Force)
	if err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, map[string]int{"deleted": deleted})
}

// Remove 删除单个镜像。
func (h *ImageHandler) Remove(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Force bool `form:"force,default=false"`
	}
	if err := httpx.MustBindQuery(w, r, &req); err != nil {
		return
	}
	id := r.PathValue("id")
	if err := h.ctx.ImageLogic.Remove(r.Context(), id, req.Force); err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, "ok")
}

// Push 推送镜像（SSE 实时推送进度）。
func (h *ImageHandler) Push(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Ref string `json:"ref" binding:"required"`
	}
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}

	body, err := h.ctx.ImageLogic.Push(r.Context(), req.Ref)
	if err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer body.Close()

	sse := httpx.NewSSEWriter(w)
	// 先发握手事件。
	_ = sse.Event("open", "connected")

	// 读取并转发 JSON 流消息。
	dec := json.NewDecoder(body)
	for {
		var msg json.RawMessage
		if err := dec.Decode(&msg); err != nil {
			break
		}
		_ = sse.Data(string(msg))
	}
	_ = sse.Event("done", "success")
}

// Inspect 镜像详情（含 Layers）。
func (h *ImageHandler) Inspect(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httpx.WriteHTTPError(w, http.StatusBadRequest, "missing image id")
		return
	}
	detail, err := h.ctx.ImageLogic.Inspect(r.Context(), id)
	if err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, detail)
}

// Prune 清理未使用镜像。
// query: dangling=true 仅清理悬空镜像。
func (h *ImageHandler) Prune(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Dangling bool `form:"dangling,default=false"`
	}
	if err := httpx.MustBindQuery(w, r, &req); err != nil {
		return
	}
	reclaimed, deleted, err := h.ctx.ImageLogic.Prune(r.Context(), req.Dangling)
	if err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, map[string]any{
		"reclaimed_bytes": reclaimed,
		"deleted":         deleted,
	})
}

// Export 导出镜像（docker save），返回 tar 流。
// query: ids=repo:tag,repo2:tag2（逗号分隔）或 id
func (h *ImageHandler) Export(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs string `form:"ids" binding:"required"`
	}
	if err := httpx.MustBindQuery(w, r, &req); err != nil {
		return
	}
	ids := strings.Split(req.IDs, ",")
	ids = trimStrings(ids)
	if len(ids) == 0 {
		httpx.WriteHTTPError(w, http.StatusBadRequest, "missing image ids")
		return
	}

	body, err := h.ctx.ImageLogic.Export(r.Context(), ids)
	if err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer body.Close()

	// 文件名取第一个镜像 tag 安全化。
	name := strings.ReplaceAll(ids[0], ":", "_")
	name = strings.ReplaceAll(name, "/", "_")
	w.Header().Set("Content-Type", "application/x-tar")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+name+".tar\"")
	_, _ = io.Copy(w, body)
}

// Import 导入镜像（docker load），body 为 tar 流。
func (h *ImageHandler) Import(w http.ResponseWriter, r *http.Request) {
	if err := h.ctx.ImageLogic.Import(r.Context(), r.Body); err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, "ok")
}

// DiskUsage 磁盘占用汇总。
func (h *ImageHandler) DiskUsage(w http.ResponseWriter, r *http.Request) {
	summary, err := h.ctx.ImageLogic.DiskUsage(r.Context())
	if err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, summary)
}

// trimStrings 去除字符串切片中的空白项。
func trimStrings(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// Tag 为镜像打标签。
func (h *ImageHandler) Tag(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Source string `json:"source" binding:"required"`
		Target string `json:"target" binding:"required"`
	}
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}
	if err := h.ctx.ImageLogic.Tag(r.Context(), req.Source, req.Target); err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, "ok")
}
