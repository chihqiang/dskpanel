package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/chihqiang/infra-go/httpx"

	"chihqiang/dskpanel/logic"
	"chihqiang/dskpanel/svc"
)

// SwarmHandler Swarm 集群管理处理器。
type SwarmHandler struct {
	ctx *svc.AppContext
}

// NewSwarmHandler 创建 Swarm 处理器。
func NewSwarmHandler(ctx *svc.AppContext) *SwarmHandler {
	return &SwarmHandler{ctx: ctx}
}

// writeSwarmError 统一错误响应。
func writeSwarmError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, logic.ErrSwarmNotActive):
		httpx.WriteHTTPError(w, http.StatusConflict, err.Error())
	default:
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
	}
}

// Detect 检测 Swarm 集群状态（连接目标由 config.Swarm 决定：本机或远程）。
func (h *SwarmHandler) Detect(w http.ResponseWriter, r *http.Request) {
	st, err := h.ctx.SwarmLogic.Info(r.Context())
	if err != nil {
		httpx.OkJSON(w, logic.SwarmStatus{Available: false, Error: err.Error()})
		return
	}
	httpx.OkJSON(w, st)
}

// Overview 集群概览。
func (h *SwarmHandler) Overview(w http.ResponseWriter, r *http.Request) {
	ov, err := h.ctx.SwarmLogic.Overview(r.Context())
	if err != nil {
		writeSwarmError(w, err)
		return
	}
	httpx.OkJSON(w, ov)
}

// ListNodes 节点列表。
func (h *SwarmHandler) ListNodes(w http.ResponseWriter, r *http.Request) {
	items, err := h.ctx.SwarmLogic.ListNodes(r.Context())
	if err != nil {
		writeSwarmError(w, err)
		return
	}
	httpx.OkJSON(w, items)
}

// InspectNode 节点详情（原始 inspect）。
func (h *SwarmHandler) InspectNode(w http.ResponseWriter, r *http.Request) {
	raw, err := h.ctx.SwarmLogic.InspectNode(r.Context(), r.PathValue("id"))
	if err != nil {
		writeSwarmError(w, err)
		return
	}
	httpx.OkJSON(w, json.RawMessage(raw))
}

// SetNodeAvailability 切换节点可用性。
func (h *SwarmHandler) SetNodeAvailability(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Availability string `json:"availability" binding:"required"`
	}
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}
	if err := h.ctx.SwarmLogic.SetNodeAvailability(r.Context(), r.PathValue("id"), req.Availability); err != nil {
		writeSwarmError(w, err)
		return
	}
	httpx.OkJSON(w, "ok")
}

// RemoveNode 删除节点。
func (h *SwarmHandler) RemoveNode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Force bool `json:"force"`
	}
	_ = httpx.MustBindJSON(w, r, &req)
	if err := h.ctx.SwarmLogic.RemoveNode(r.Context(), r.PathValue("id"), req.Force); err != nil {
		writeSwarmError(w, err)
		return
	}
	httpx.OkJSON(w, "ok")
}

// ListServices 服务列表。
func (h *SwarmHandler) ListServices(w http.ResponseWriter, r *http.Request) {
	items, err := h.ctx.SwarmLogic.ListServices(r.Context())
	if err != nil {
		writeSwarmError(w, err)
		return
	}
	httpx.OkJSON(w, items)
}

// InspectService 服务详情（原始 inspect）。
func (h *SwarmHandler) InspectService(w http.ResponseWriter, r *http.Request) {
	raw, err := h.ctx.SwarmLogic.InspectService(r.Context(), r.PathValue("id"))
	if err != nil {
		writeSwarmError(w, err)
		return
	}
	httpx.OkJSON(w, json.RawMessage(raw))
}

// ServiceResources 服务级资源监控（任务容器 CPU/内存聚合）。
func (h *SwarmHandler) ServiceResources(w http.ResponseWriter, r *http.Request) {
	items, err := h.ctx.SwarmLogic.ServiceResources(r.Context(), r.PathValue("id"))
	if err != nil {
		writeSwarmError(w, err)
		return
	}
	httpx.OkJSON(w, items)
}

// CreateService 创建服务。
func (h *SwarmHandler) CreateService(w http.ResponseWriter, r *http.Request) {
	var req logic.ServiceRequest
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}
	if err := h.ctx.SwarmLogic.CreateService(r.Context(), req); err != nil {
		writeSwarmError(w, err)
		return
	}
	httpx.OkJSON(w, "ok")
}

// UpdateService 更新服务。
func (h *SwarmHandler) UpdateService(w http.ResponseWriter, r *http.Request) {
	var req logic.ServiceRequest
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}
	if err := h.ctx.SwarmLogic.UpdateService(r.Context(), r.PathValue("id"), req); err != nil {
		writeSwarmError(w, err)
		return
	}
	httpx.OkJSON(w, "ok")
}

// ScaleService 服务伸缩。
func (h *SwarmHandler) ScaleService(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Replicas uint64 `json:"replicas" binding:"required"`
	}
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}
	if err := h.ctx.SwarmLogic.ScaleService(r.Context(), r.PathValue("id"), req.Replicas); err != nil {
		writeSwarmError(w, err)
		return
	}
	httpx.OkJSON(w, "ok")
}

// RemoveService 删除服务。
func (h *SwarmHandler) RemoveService(w http.ResponseWriter, r *http.Request) {
	if err := h.ctx.SwarmLogic.RemoveService(r.Context(), r.PathValue("id")); err != nil {
		writeSwarmError(w, err)
		return
	}
	httpx.OkJSON(w, "ok")
}

// RollbackService 回滚服务。
func (h *SwarmHandler) RollbackService(w http.ResponseWriter, r *http.Request) {
	if err := h.ctx.SwarmLogic.RollbackService(r.Context(), r.PathValue("id")); err != nil {
		writeSwarmError(w, err)
		return
	}
	httpx.OkJSON(w, "ok")
}

// ForceUpdateService 强制更新（恢复暂停的更新 / 滚动重启）。
func (h *SwarmHandler) ForceUpdateService(w http.ResponseWriter, r *http.Request) {
	if err := h.ctx.SwarmLogic.ForceUpdateService(r.Context(), r.PathValue("id")); err != nil {
		writeSwarmError(w, err)
		return
	}
	httpx.OkJSON(w, "ok")
}

// ListTasks 任务列表（可按服务过滤）。
func (h *SwarmHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	serviceID := r.URL.Query().Get("service")
	items, err := h.ctx.SwarmLogic.ListTasks(r.Context(), serviceID)
	if err != nil {
		writeSwarmError(w, err)
		return
	}
	httpx.OkJSON(w, items)
}

// TaskLogs 任务日志（SSE 流式推送，复用服务日志同样的推流逻辑）。
func (h *SwarmHandler) TaskLogs(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Tail   string `form:"tail,default=200"`
		Follow bool   `form:"follow,default=true"`
	}
	if err := httpx.MustBindQuery(w, r, &req); err != nil {
		return
	}

	reader, err := h.ctx.SwarmLogic.StreamTaskLogs(r.Context(), r.PathValue("id"), req.Tail, req.Follow)
	if err != nil {
		writeSwarmError(w, err)
		return
	}
	defer reader.Close()

	sse := httpx.NewSSEWriter(w)
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
				flushPending()
				_ = sse.Event("error", err.Error())
			}
			return
		}
	}
}

// ServiceLogs 服务日志（SSE 流式推送）。
func (h *SwarmHandler) ServiceLogs(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Tail   string `form:"tail,default=200"`
		Follow bool   `form:"follow,default=true"`
	}
	if err := httpx.MustBindQuery(w, r, &req); err != nil {
		return
	}

	reader, err := h.ctx.SwarmLogic.StreamServiceLogs(r.Context(), r.PathValue("id"), req.Tail, req.Follow)
	if err != nil {
		writeSwarmError(w, err)
		return
	}
	defer reader.Close()

	sse := httpx.NewSSEWriter(w)
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
				flushPending()
				_ = sse.Event("error", err.Error())
			}
			return
		}
	}
}

// ListNetworks 集群网络列表。
func (h *SwarmHandler) ListNetworks(w http.ResponseWriter, r *http.Request) {
	items, err := h.ctx.SwarmLogic.ListNetworks(r.Context())
	if err != nil {
		writeSwarmError(w, err)
		return
	}
	httpx.OkJSON(w, items)
}

// CreateNetwork 创建集群网络。
func (h *SwarmHandler) CreateNetwork(w http.ResponseWriter, r *http.Request) {
	var req logic.SwarmNetworkCreateRequest
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}
	if err := h.ctx.SwarmLogic.CreateNetwork(r.Context(), req); err != nil {
		writeSwarmError(w, err)
		return
	}
	httpx.OkJSON(w, "ok")
}

// InspectNetwork 网络详情。
func (h *SwarmHandler) InspectNetwork(w http.ResponseWriter, r *http.Request) {
	raw, err := h.ctx.SwarmLogic.InspectNetwork(r.Context(), r.PathValue("id"))
	if err != nil {
		writeSwarmError(w, err)
		return
	}
	httpx.OkJSON(w, json.RawMessage(raw))
}

// RemoveNetwork 删除网络。
func (h *SwarmHandler) RemoveNetwork(w http.ResponseWriter, r *http.Request) {
	if err := h.ctx.SwarmLogic.RemoveNetwork(r.Context(), r.PathValue("id")); err != nil {
		writeSwarmError(w, err)
		return
	}
	httpx.OkJSON(w, "ok")
}

// GetJoinTokens 获取集群加入令牌。
func (h *SwarmHandler) GetJoinTokens(w http.ResponseWriter, r *http.Request) {
	tokens, err := h.ctx.SwarmLogic.GetJoinTokens(r.Context())
	if err != nil {
		writeSwarmError(w, err)
		return
	}
	httpx.OkJSON(w, tokens)
}

// ListImages 集群镜像列表（供服务创建表单选择）。
func (h *SwarmHandler) ListImages(w http.ResponseWriter, r *http.Request) {
	items, err := h.ctx.SwarmLogic.ListImages(r.Context())
	if err != nil {
		writeSwarmError(w, err)
		return
	}
	httpx.OkJSON(w, items)
}

// ListSecrets Secret 列表。
func (h *SwarmHandler) ListSecrets(w http.ResponseWriter, r *http.Request) {
	items, err := h.ctx.SwarmLogic.ListSecrets(r.Context())
	if err != nil {
		writeSwarmError(w, err)
		return
	}
	httpx.OkJSON(w, items)
}

// InspectSecret Secret 详情。
func (h *SwarmHandler) InspectSecret(w http.ResponseWriter, r *http.Request) {
	detail, err := h.ctx.SwarmLogic.InspectSecret(r.Context(), r.PathValue("id"))
	if err != nil {
		writeSwarmError(w, err)
		return
	}
	httpx.OkJSON(w, detail)
}

// CreateSecret 创建 Secret。
func (h *SwarmHandler) CreateSecret(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name" binding:"required"`
		Data string `json:"data"`
	}
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}
	if err := h.ctx.SwarmLogic.CreateSecret(r.Context(), req.Name, req.Data); err != nil {
		writeSwarmError(w, err)
		return
	}
	httpx.OkJSON(w, "ok")
}

// RemoveSecret 删除 Secret。
func (h *SwarmHandler) RemoveSecret(w http.ResponseWriter, r *http.Request) {
	if err := h.ctx.SwarmLogic.RemoveSecret(r.Context(), r.PathValue("id")); err != nil {
		writeSwarmError(w, err)
		return
	}
	httpx.OkJSON(w, "ok")
}

// ListConfigs Config 列表。
func (h *SwarmHandler) ListConfigs(w http.ResponseWriter, r *http.Request) {
	items, err := h.ctx.SwarmLogic.ListConfigs(r.Context())
	if err != nil {
		writeSwarmError(w, err)
		return
	}
	httpx.OkJSON(w, items)
}

// InspectConfig Config 详情。
func (h *SwarmHandler) InspectConfig(w http.ResponseWriter, r *http.Request) {
	detail, err := h.ctx.SwarmLogic.InspectConfig(r.Context(), r.PathValue("id"))
	if err != nil {
		writeSwarmError(w, err)
		return
	}
	httpx.OkJSON(w, detail)
}

// CreateConfig 创建 Config。
func (h *SwarmHandler) CreateConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name" binding:"required"`
		Data string `json:"data"`
	}
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}
	if err := h.ctx.SwarmLogic.CreateConfig(r.Context(), req.Name, req.Data); err != nil {
		writeSwarmError(w, err)
		return
	}
	httpx.OkJSON(w, "ok")
}

// RemoveConfig 删除 Config。
func (h *SwarmHandler) RemoveConfig(w http.ResponseWriter, r *http.Request) {
	if err := h.ctx.SwarmLogic.RemoveConfig(r.Context(), r.PathValue("id")); err != nil {
		writeSwarmError(w, err)
		return
	}
	httpx.OkJSON(w, "ok")
}
