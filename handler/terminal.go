package handler

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"

	"chihqiang/dskpanel/svc"
)

// upgrader WebSocket 升级器（终端专用）。
var terminalUpgrader = websocket.Upgrader{
	// 终端从同源前端访问，允许任意 Origin（与全局 CORS 一致）。
	CheckOrigin: func(_ *http.Request) bool { return true },
	// 终端是字节流，使用二进制帧传输，避免文本转义问题。
}

// 终端 WebSocket 消息协议：
//
//	客户端 → 服务端（JSON 文本帧）：
//	  {"type":"input","data":"<字符串>"}     输入（写往容器 stdin）
//	  {"type":"resize","cols":80,"rows":24}  调整 TTY 尺寸
//
//	服务端 → 客户端（二进制帧）：
//	  原始字节流（容器 stdout/stderr 内容）
type terminalMessage struct {
	Type string `json:"type"`
	Data string `json:"data,omitempty"`
	Cols int    `json:"cols,omitempty"`
	Rows int    `json:"rows,omitempty"`
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

	ws, err := terminalUpgrader.Upgrade(w, r, nil)
	if err != nil {
		// Upgrade 失败时已由 gorilla 写入响应，无需额外处理。
		return
	}
	defer ws.Close()

	attach, err := h.ctx.ContainerLogic.Attach(r.Context(), id)
	if err != nil {
		_ = ws.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "attach failed: "+err.Error()))
		return
	}
	defer attach.Close()

	var wg sync.WaitGroup

	// 容器输出 → WebSocket（二进制帧）。
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			n, err := attach.Reader.Read(buf)
			if n > 0 {
				if werr := ws.WriteMessage(websocket.BinaryMessage, buf[:n]); werr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	// WebSocket → 容器 stdin（JSON 消息）。
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			_, data, err := ws.ReadMessage()
			if err != nil {
				// 前端关闭或断线：通知容器结束。
				return
			}
			var msg terminalMessage
			if err := json.Unmarshal(data, &msg); err != nil {
				continue
			}
			switch msg.Type {
			case "input":
				if msg.Data != "" {
					_, _ = attach.Writer.Write([]byte(msg.Data))
				}
			case "resize":
				if msg.Cols > 0 && msg.Rows > 0 {
					_ = h.ctx.ContainerLogic.ResizeContainerTTY(r.Context(), id, uint(msg.Rows), uint(msg.Cols))
				}
			}
		}
	}()

	// 等待任一方向结束（容器退出或 ws 断开）。
	wg.Wait()
	_ = attach.Close()

	// 容器结束（非前端主动关闭）时，发送关闭帧通知前端。
	_ = ws.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "container exited"))
}
