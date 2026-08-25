package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// wsUpgrader 统一的 WebSocket 升级器（终端/交互场景）。
var wsUpgrader = websocket.Upgrader{
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

// WSStream 终端双向流：可读写、可调整 TTY 尺寸、可关闭。
// 容器 attach（*AttachResult）与 k8s exec（K8sExecStream）均适配此接口。
type WSStream interface {
	io.Reader
	io.Writer
	Resize(cols, rows uint16) error
	Close() error
}

// UpgradeWS 统一的 WebSocket 升级封装。
// Upgrade 失败时 gorilla 已写入响应，调用方无需额外处理。
func UpgradeWS(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	return wsUpgrader.Upgrade(w, r, nil)
}

// HandleWS 统一的终端 WebSocket 处理流程：
//
//  1. 升级为 WebSocket；
//  2. open 创建终端流（容器 attach / k8s exec），失败时发送错误关闭帧；
//  3. 双向桥接：流输出 → WebSocket 二进制帧，WebSocket JSON 消息 → 流输入；
//  4. 任一端结束或断线后，关闭流并发送正常关闭帧通知前端。
func HandleWS(w http.ResponseWriter, r *http.Request, open func() (WSStream, error)) {
	ws, err := UpgradeWS(w, r)
	if err != nil {
		return
	}
	defer ws.Close()

	stream, err := open()
	if err != nil {
		_ = ws.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "terminal failed: "+err.Error()))
		return
	}
	defer stream.Close()

	var wg sync.WaitGroup

	// 流输出 → WebSocket（二进制帧）。
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			n, err := stream.Read(buf)
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

	// WebSocket → 流输入（JSON 消息）。
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			_, data, err := ws.ReadMessage()
			if err != nil {
				// 前端关闭或断线：通知终端结束。
				return
			}
			var msg terminalMessage
			if err := json.Unmarshal(data, &msg); err != nil {
				continue
			}
			switch msg.Type {
			case "input":
				if msg.Data != "" {
					_, _ = stream.Write([]byte(msg.Data))
				}
			case "resize":
				if msg.Cols > 0 && msg.Rows > 0 {
					_ = stream.Resize(uint16(msg.Cols), uint16(msg.Rows))
				}
			}
		}
	}()

	// 等待任一方向结束（终端退出或 ws 断开）。
	wg.Wait()

	// 终端结束（非前端主动关闭）时，发送关闭帧通知前端。
	_ = ws.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "terminal closed"))
}
