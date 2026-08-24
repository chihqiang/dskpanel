package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// SSEWriter SSE 推送辅助：统一封装响应头、握手事件与各类型事件发送。
// 所有 SSE handler 复用本结构，避免重复样板代码。
type SSEWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

// NewSSEWriter 创建 SSE 推送器；响应不支持流式时返回错误。
func NewSSEWriter(w http.ResponseWriter) (*SSEWriter, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming unsupported")
	}
	// SSE 是长连接，清除 server 级 WriteTimeout（默认 10s）避免连接被强制关闭。
	if rc := http.NewResponseController(w); rc != nil {
		_ = rc.SetWriteDeadline(time.Time{})
		_ = rc.SetReadDeadline(time.Time{})
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	return &SSEWriter{w: w, flusher: flusher}, nil
}

// Open 发送握手事件（event: open / data: connected），告知前端流已建立。
func (s *SSEWriter) Open() {
	s.Event("open", "connected")
}

// Data 发送日志/进度行（data 事件），自动转义换行保证单行。
func (s *SSEWriter) Data(line string) {
	s.Event("", line)
}

// Event 发送指定类型事件（event: <type> + data: <data>）。
func (s *SSEWriter) Event(eventType, data string) {
	if eventType != "" {
		fmt.Fprintf(s.w, "event: %s\n", eventType)
	}
	fmt.Fprintf(s.w, "data: %s\n\n", strings.ReplaceAll(data, "\n", "\\n"))
	s.flusher.Flush()
}

// Error 发送错误事件。
func (s *SSEWriter) Error(msg string) {
	s.Event("error", msg)
}

// Done 发送完成事件（data: success / fail）。
func (s *SSEWriter) Done(success bool) {
	if success {
		s.Event("done", "success")
	} else {
		s.Event("done", "fail")
	}
}
