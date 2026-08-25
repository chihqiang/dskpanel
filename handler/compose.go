package handler

import (
	"net/http"
	"strconv"

	"github.com/chihqiang/infra-go/httpx"

	"chihqiang/dskpanel/svc"
)

// ComposeHandler Compose 编排处理器。
type ComposeHandler struct {
	ctx *svc.AppContext
}

// NewComposeHandler 创建 Compose 编排处理器。
func NewComposeHandler(ctx *svc.AppContext) *ComposeHandler {
	return &ComposeHandler{ctx: ctx}
}

// Validate 校验 Compose 文件。
func (h *ComposeHandler) Validate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}
	result := h.ctx.ComposeLogic.Validate(r.Context(), req.Content)
	httpx.OkJSON(w, result)
}

// Deploy 部署 Compose 应用（兼容同步调用，保留）。
func (h *ComposeHandler) Deploy(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}
	result := h.ctx.ComposeLogic.Deploy(r.Context(), req.Content)
	if !result.OK {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, result.Message)
		return
	}
	httpx.OkJSON(w, result)
}

// DeployStream 部署 Compose 应用（SSE 流式实时回显）。
// 逐行推送 docker compose up -d 输出，结束推送 event: done（含 success/fail）。
func (h *ComposeHandler) DeployStream(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}

	sse, err := NewSSEWriter(w)
	if err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// 握手事件。
	sse.Open()

	ok, err := h.ctx.ComposeLogic.DeployStream(r.Context(), req.Content, func(line string) {
		sse.Data(line)
	})
	if err != nil {
		sse.Error(err.Error())
	} else {
		sse.Done(ok)
	}
}

// ListProjects 列出所有 Compose 项目。
func (h *ComposeHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	items, err := h.ctx.ComposeLogic.ListProjects(r.Context())
	if err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, items)
}

// ProjectPs 查询 Compose 项目内容器状态。
func (h *ComposeHandler) ProjectPs(w http.ResponseWriter, r *http.Request) {
	detail, err := h.ctx.ComposeLogic.ProjectPs(r.Context(), r.PathValue("name"))
	if err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, detail)
}

// ProjectStart 启动项目。
func (h *ComposeHandler) ProjectStart(w http.ResponseWriter, r *http.Request) {
	if err := h.ctx.ComposeLogic.ProjectStart(r.Context(), r.PathValue("name")); err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, "started")
}

// ProjectStop 停止项目。
func (h *ComposeHandler) ProjectStop(w http.ResponseWriter, r *http.Request) {
	if err := h.ctx.ComposeLogic.ProjectStop(r.Context(), r.PathValue("name")); err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, "stopped")
}

// ProjectRestart 重启项目。
func (h *ComposeHandler) ProjectRestart(w http.ResponseWriter, r *http.Request) {
	if err := h.ctx.ComposeLogic.ProjectRestart(r.Context(), r.PathValue("name")); err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, "restarted")
}

// ProjectDown 停止并移除项目。
func (h *ComposeHandler) ProjectDown(w http.ResponseWriter, r *http.Request) {
	volumes := r.URL.Query().Get("volumes") == "1"
	if err := h.ctx.ComposeLogic.ProjectDown(r.Context(), r.PathValue("name"), volumes); err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, "down")
}

// ProjectLogs 拉取项目日志。
func (h *ComposeHandler) ProjectLogs(w http.ResponseWriter, r *http.Request) {
	tail := 200
	if v := r.URL.Query().Get("tail"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			tail = n
		}
	}
	lines, err := h.ctx.ComposeLogic.ProjectLogs(r.Context(), r.PathValue("name"), tail)
	if err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, lines)
}

// ProjectConfig 读取 Compose 项目的配置文件内容。
func (h *ComposeHandler) ProjectConfig(w http.ResponseWriter, r *http.Request) {
	content, err := h.ctx.ComposeLogic.ProjectConfig(r.Context(), r.PathValue("name"))
	if err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, content)
}
