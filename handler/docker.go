package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/chihqiang/infra-go/httpx"
	"github.com/chihqiang/infra-go/logger"

	"chihqiang/dskpanel/logic"
	"chihqiang/dskpanel/svc"
)

// DockerHandler 本机 Docker 处理器。
type DockerHandler struct {
	ctx *svc.AppContext
}

// NewDockerHandler 创建本机 Docker 处理器。
func NewDockerHandler(ctx *svc.AppContext) *DockerHandler {
	return &DockerHandler{ctx: ctx}
}

// Detect 检测本机 Docker 环境。
func (h *DockerHandler) Detect(w http.ResponseWriter, r *http.Request) {
	info := h.ctx.DockerLogic.Detect(r.Context())
	logger.Infof("docker detect: available=%v version=%s", info.Available, info.Version)
	httpx.OkJSON(w, info)
}

// Overview 概览聚合：本机 Docker 资源统计 + 引擎版本信息。
func (h *DockerHandler) Overview(w http.ResponseWriter, r *http.Request) {
	overview, err := h.ctx.DockerLogic.Overview(r.Context())
	if err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, overview)
}

// Info 本机 Docker 引擎完整信息（docker info）。
func (h *DockerHandler) Info(w http.ResponseWriter, r *http.Request) {
	info, err := h.ctx.DockerLogic.Info(r.Context())
	if err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, info)
}

// PruneAll 一键清理未使用资源。
func (h *DockerHandler) PruneAll(w http.ResponseWriter, r *http.Request) {
	result := h.ctx.DockerLogic.PruneAll(r.Context())
	if result == nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, "docker not available")
		return
	}
	httpx.OkJSON(w, result)
}

// Events 订阅 Docker 系统事件（SSE 流式推送 JSON）。
func (h *DockerHandler) Events(w http.ResponseWriter, r *http.Request) {
	sse, err := NewSSEWriter(w)
	if err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	sse.Open()

	err = h.ctx.DockerLogic.StreamEvents(r.Context(), func(ev logic.DockerEvent) {
		data, mErr := json.Marshal(ev)
		if mErr != nil {
			return
		}
		sse.Data(string(data))
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		sse.Error(err.Error())
	}
}
