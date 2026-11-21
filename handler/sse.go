// SSE 响应写入器：Server-Sent Events 流式输出。
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// sseWriter 向客户端推送 SSE 事件流，每次写入后立即 flush。
type sseWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

// newSSEWriter 创建 SSE writer 并设置 SSE 必需的响应头。
func newSSEWriter(w http.ResponseWriter) *sseWriter {
	h := w.Header()
	h.Set("Content-Type", "text/event-stream; charset=utf-8")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")

	s := &sseWriter{w: w}
	if f, ok := w.(http.Flusher); ok {
		s.flusher = f
	}
	return s
}

// Event 写入一个事件帧；event 为空时使用默认 message 事件。
func (s *sseWriter) Event(event, data string) error {
	var b strings.Builder
	if event != "" {
		s.writeField(&b, "event", event)
	}
	s.writeField(&b, "data", data)
	b.WriteByte('\n')
	return s.write(b.String())
}

// Data 写入默认事件类型的 data 帧。
func (s *sseWriter) Data(data string) error {
	return s.Event("", data)
}

// JSONEvent 将 v 序列化为 JSON 作为事件数据。
func (s *sseWriter) JSONEvent(event string, v any) error {
	bs, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal sse event data failed: %w", err)
	}
	return s.Event(event, string(bs))
}

// Comment 写入注释帧，可用作心跳。
func (s *sseWriter) Comment(text string) error {
	var b strings.Builder
	for _, line := range strings.Split(text, "\n") {
		b.WriteByte(':')
		if line != "" {
			b.WriteByte(' ')
			b.WriteString(line)
		}
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	return s.write(b.String())
}

// Retry 设置浏览器重连间隔（毫秒）。
func (s *sseWriter) Retry(ms int) error {
	return s.write("retry: " + strconv.Itoa(ms) + "\n\n")
}

func (s *sseWriter) writeField(b *strings.Builder, field, value string) {
	for _, line := range strings.Split(value, "\n") {
		b.WriteString(field)
		b.WriteString(": ")
		b.WriteString(line)
		b.WriteByte('\n')
	}
}

func (s *sseWriter) write(payload string) error {
	if _, err := s.w.Write([]byte(payload)); err != nil {
		return err
	}
	if s.flusher != nil {
		s.flusher.Flush()
	}
	return nil
}
