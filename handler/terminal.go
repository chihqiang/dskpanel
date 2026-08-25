package handler

import (
	"net/http"

	"chihqiang/dskpanel/logic"
	"chihqiang/dskpanel/svc"
)

// containerTerminalStream 将 *logic.AttachResult 适配为 WSStream。
type containerTerminalStream struct {
	attach *logic.AttachResult
	resize func(execID string, rows, cols uint) error
}

func (s *containerTerminalStream) Read(p []byte) (int, error)  { return s.attach.Reader.Read(p) }
func (s *containerTerminalStream) Write(p []byte) (int, error) { return s.attach.Writer.Write(p) }
func (s *containerTerminalStream) Close() error                { return s.attach.Close() }

// Resize 调整 TTY 尺寸。
func (s *containerTerminalStream) Resize(cols, rows uint16) error {
	return s.resize(s.attach.ExecID, uint(rows), uint(cols))
}

// TerminalHandler 容器终端（WebSocket）。
type TerminalHandler struct {
	ctx *svc.AppContext
}

// NewTerminalHandler 创建容器终端处理器。
func NewTerminalHandler(ctx *svc.AppContext) *TerminalHandler {
	return &TerminalHandler{ctx: ctx}
}

// Attach 容器终端 WebSocket 端点。
// GET /api/v1/containers/{id}/terminal
// 升级为 WebSocket 后，将容器 attach 的 stdin/stdout 与 ws 双向桥接。
func (h *TerminalHandler) Attach(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "missing container id", http.StatusBadRequest)
		return
	}

	HandleWS(w, r, func() (WSStream, error) {
		attach, err := h.ctx.ContainerLogic.Attach(r.Context(), id)
		if err != nil {
			return nil, err
		}
		return &containerTerminalStream{
			attach: attach,
			resize: func(execID string, rows, cols uint) error {
				return h.ctx.ContainerLogic.ResizeContainerTTY(r.Context(), execID, rows, cols)
			},
		}, nil
	})
}
