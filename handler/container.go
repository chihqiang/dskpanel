package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/chihqiang/infra-go/httpx"

	"chihqiang/dskpanel/logic"
	"chihqiang/dskpanel/svc"
)

// ContainerHandler 容器管理处理器。
type ContainerHandler struct {
	ctx *svc.AppContext
}

// NewContainerHandler 创建容器管理处理器。
func NewContainerHandler(ctx *svc.AppContext) *ContainerHandler {
	return &ContainerHandler{ctx: ctx}
}

// List 容器列表。
func (h *ContainerHandler) List(w http.ResponseWriter, r *http.Request) {
	var req struct {
		All bool `form:"all,default=true"`
	}
	if err := httpx.MustBindQuery(w, r, &req); err != nil {
		return
	}
	items, err := h.ctx.ContainerLogic.List(r.Context(), req.All)
	if err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, items)
}

// Inspect 容器详情。
func (h *ContainerHandler) Inspect(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httpx.WriteHTTPError(w, http.StatusBadRequest, "missing container id")
		return
	}
	detail, err := h.ctx.ContainerLogic.Inspect(r.Context(), id)
	if err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, detail)
}

// InspectRaw 返回容器完整 inspect 原始 JSON（排障用）。
func (h *ContainerHandler) InspectRaw(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httpx.WriteHTTPError(w, http.StatusBadRequest, "missing container id")
		return
	}
	raw, err := h.ctx.ContainerLogic.InspectRaw(r.Context(), id)
	if err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, json.RawMessage(raw))
}

// Create 创建容器。
func (h *ContainerHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req logic.CreateContainerRequest
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}
	result, err := h.ctx.ContainerLogic.Create(r.Context(), &req)
	if err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, result)
}

// Logs 容器日志（HTTP 一次性读取，已解帧）。
// query: tail=行数(默认100) timestamps=true|false
// 返回纯文本日志内容。
func (h *ContainerHandler) Logs(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Tail       string `form:"tail,default=100"`
		Timestamps bool   `form:"timestamps,default=false"`
	}
	if err := httpx.MustBindQuery(w, r, &req); err != nil {
		return
	}
	id := r.PathValue("id")

	reader, err := h.ctx.ContainerLogic.StreamLogs(r.Context(), logic.LogsOptions{
		ContainerID: id,
		Tail:        req.Tail,
		Follow:      false,
		Timestamps:  req.Timestamps,
	})
	if err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write(data)
}

// LogsStream 容器日志 SSE 流式推送（后端主动推送）。
// query: tail=行数(默认100) follow=true|false(默认true) timestamps=true|false
// Content-Type: text/event-stream，每行日志作为 data: <line> 事件推送。
func (h *ContainerHandler) LogsStream(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Tail       string `form:"tail,default=100"`
		Follow     bool   `form:"follow,default=true"`
		Timestamps bool   `form:"timestamps,default=false"`
	}
	if err := httpx.MustBindQuery(w, r, &req); err != nil {
		return
	}
	id := r.PathValue("id")

	reader, err := h.ctx.ContainerLogic.StreamLogs(r.Context(), logic.LogsOptions{
		ContainerID: id,
		Tail:        req.Tail,
		Follow:      req.Follow,
		Timestamps:  req.Timestamps,
	})
	if err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer reader.Close()

	sse := httpx.NewSSEWriter(w)
	// 先发一个连接事件，告知前端流已建立。
	_ = sse.Event("open", "connected")

	buf := make([]byte, 4096)
	var pending []byte
	flushPending := func() {
		if len(pending) == 0 {
			return
		}
		line := bytes.TrimRight(pending, "\r\n")
		pending = pending[:0]
		if len(line) > 0 {
			_ = sse.Data(string(line))
		}
	}

	for {
		n, err := reader.Read(buf)
		if n > 0 {
			pending = append(pending, buf[:n]...)
			// 按换行切分推送。
			for {
				idx := bytes.IndexByte(pending, '\n')
				if idx < 0 {
					break
				}
				line := pending[:idx]
				pending = pending[idx+1:]
				line = bytes.TrimRight(line, "\r")
				if len(line) > 0 {
					_ = sse.Data(string(line))
				}
			}
		}
		if err != nil {
			if err != io.EOF {
				// 读取错误：先 flush 残余，再发错误事件。
				flushPending()
				_ = sse.Event("error", err.Error())
			}
			return
		}
		if !req.Follow && n == 0 {
			flushPending()
			return
		}
	}
}

// Start 启动容器。
func (h *ContainerHandler) Start(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.ctx.ContainerLogic.Start(r.Context(), id); err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, "ok")
}

// Stop 停止容器。
func (h *ContainerHandler) Stop(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.ctx.ContainerLogic.Stop(r.Context(), id); err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, "ok")
}

// Restart 重启容器。
func (h *ContainerHandler) Restart(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.ctx.ContainerLogic.Restart(r.Context(), id); err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, "ok")
}

// Remove 删除容器。
func (h *ContainerHandler) Remove(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Force         bool `form:"force,default=false"`
		RemoveVolumes bool `form:"remove_volumes,default=false"`
	}
	if err := httpx.MustBindQuery(w, r, &req); err != nil {
		return
	}
	id := r.PathValue("id")
	if err := h.ctx.ContainerLogic.Remove(r.Context(), id, req.Force, req.RemoveVolumes); err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, "ok")
}

// Stats 容器实时资源统计。
func (h *ContainerHandler) Stats(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httpx.WriteHTTPError(w, http.StatusBadRequest, "missing container id")
		return
	}
	stats, err := h.ctx.ContainerLogic.Stats(r.Context(), id)
	if err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, stats)
}

// Rename 重命名容器。
func (h *ContainerHandler) Rename(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}
	id := r.PathValue("id")
	if err := h.ctx.ContainerLogic.Rename(r.Context(), id, req.Name); err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, "ok")
}

// Commit 将容器提交为镜像（docker commit）。
func (h *ContainerHandler) Commit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Reference string `json:"reference"`
		Comment   string `json:"comment"`
		Author    string `json:"author"`
	}
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}
	id := r.PathValue("id")
	imageID, err := h.ctx.ContainerLogic.Commit(r.Context(), id, req.Reference, req.Comment, req.Author)
	if err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, map[string]any{"id": imageID})
}

// Update 更新容器资源限制与重启策略（docker update）。
func (h *ContainerHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CPUShares     int64  `json:"cpu_shares"`
		Memory        int64  `json:"memory"` // 字节
		NanoCPUs      int64  `json:"nano_cpus"`
		CpusetCpus    string `json:"cpuset_cpus"`
		MemorySwap    int64  `json:"memory_swap"` // -1 无限制；0 跟随 memory
		RestartPolicy string `json:"restart_policy"`
		RestartMax    int    `json:"restart_max"`
	}
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}
	id := r.PathValue("id")
	if err := h.ctx.ContainerLogic.Update(r.Context(), id, req.CPUShares, req.Memory, req.NanoCPUs, req.CpusetCpus, req.MemorySwap, req.RestartPolicy, req.RestartMax); err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, "ok")
}

// Pause 暂停容器。
func (h *ContainerHandler) Pause(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.ctx.ContainerLogic.Pause(r.Context(), id); err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, "ok")
}

// Unpause 恢复暂停的容器。
func (h *ContainerHandler) Unpause(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.ctx.ContainerLogic.Unpause(r.Context(), id); err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, "ok")
}

// Export 导出容器文件系统（docker export），返回 tar 流。
func (h *ContainerHandler) Export(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httpx.WriteHTTPError(w, http.StatusBadRequest, "missing container id")
		return
	}
	body, err := h.ctx.ContainerLogic.Export(r.Context(), id)
	if err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer body.Close()

	// 文件名安全化：容器短 ID。
	name := id
	if len(name) > 12 {
		name = name[:12]
	}
	w.Header().Set("Content-Type", "application/x-tar")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+name+"-export.tar\"")
	_, _ = io.Copy(w, body)
}

// Batch 批量操作容器（start/stop/restart/remove）。
func (h *ContainerHandler) Batch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action string   `json:"action" binding:"required"`
		IDs    []string `json:"ids" binding:"required"`
	}
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}
	if len(req.IDs) == 0 {
		httpx.WriteHTTPError(w, http.StatusBadRequest, "missing container ids")
		return
	}
	done, failed, err := h.ctx.ContainerLogic.Batch(r.Context(), logic.BatchAction(req.Action), req.IDs)
	if err != nil {
		httpx.WriteHTTPError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.OkJSON(w, map[string]any{"done": done, "failed": failed})
}

// Top 查看容器内进程列表。
func (h *ContainerHandler) Top(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httpx.WriteHTTPError(w, http.StatusBadRequest, "missing container id")
		return
	}
	procs, err := h.ctx.ContainerLogic.Top(r.Context(), id)
	if err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, procs)
}

// Exec 在容器中执行一次性命令（非交互式）。
func (h *ContainerHandler) Exec(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httpx.WriteHTTPError(w, http.StatusBadRequest, "missing container id")
		return
	}
	var req struct {
		Command []string `json:"command" binding:"required"`
	}
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}
	result, err := h.ctx.ContainerLogic.Exec(r.Context(), id, req.Command)
	if err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, result)
}
