package handler

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/chihqiang/infra-go/httpx"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"chihqiang/dskpanel/logic"
)

// ListPods Pod 列表（支持 labelSelector / fieldSelector 过滤）。
func (h *K8sHandler) ListPods(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Namespace     string `form:"namespace"`
		LabelSelector string `form:"labelSelector"`
		FieldSelector string `form:"fieldSelector"`
		Limit         int64  `form:"limit"`
	}
	if err := httpx.MustBindQuery(w, r, &req); err != nil {
		return
	}
	items, err := h.ctx.K8sLogic.ListPodsWithOptions(r.Context(), logic.K8sListOptions{
		Namespace:     req.Namespace,
		LabelSelector: req.LabelSelector,
		FieldSelector: req.FieldSelector,
		Limit:         req.Limit,
	})
	if err != nil {
		writeK8sError(w, err)
		return
	}
	writeK8sList(w, r, items)
}

// InspectPod Pod 详情（支持 ?format=yaml）。
func (h *K8sHandler) InspectPod(w http.ResponseWriter, r *http.Request) {
	namespace := httpx.QueryValue[string](r, "namespace")
	pod, err := h.ctx.K8sLogic.InspectPod(r.Context(), namespace, r.PathValue("name"))
	if err != nil {
		writeK8sError(w, err)
		return
	}
	writeK8sObject(w, r, pod)
}

// DeletePod 删除 Pod（NotFound 幂等）。
func (h *K8sHandler) DeletePod(w http.ResponseWriter, r *http.Request) {
	namespace := httpx.QueryValue[string](r, "namespace")
	err := h.ctx.K8sLogic.DeletePod(r.Context(), namespace, r.PathValue("name"))
	writeK8sDeleteResult(w, err)
}

// PodLogs Pod 日志（SSE 流式推送，支持 since/sinceTime/timestamps 过滤）。
func (h *K8sHandler) PodLogs(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Namespace  string `form:"namespace"`
		Container  string `form:"container"`
		Tail       string `form:"tail,default=200"`
		Follow     bool   `form:"follow,default=true"`
		Previous   bool   `form:"previous,default=false"`
		Since      string `form:"since"`     // 相对时间，如 "5m"、"1h"、"3600"
		SinceTime  string `form:"sinceTime"` // 绝对时间，RFC3339 格式
		Timestamps bool   `form:"timestamps,default=false"`
	}
	if err := httpx.MustBindQuery(w, r, &req); err != nil {
		return
	}

	var tailLines *int64
	if req.Tail != "" && req.Tail != "all" {
		if n, err := strconv.ParseInt(req.Tail, 10, 64); err == nil {
			tailLines = &n
		}
	}

	var sinceSeconds *int64
	if req.Since != "" {
		if d, err := parseDuration(req.Since); err == nil {
			s := int64(d.Seconds())
			sinceSeconds = &s
		}
	}

	var sinceTime *metav1.Time
	if req.SinceTime != "" {
		if t, err := time.Parse(time.RFC3339, req.SinceTime); err == nil {
			mt := metav1.NewTime(t)
			sinceTime = &mt
		}
	}

	reader, err := h.ctx.K8sLogic.StreamPodLogs(r.Context(), logic.PodLogsOptions{
		Namespace:    req.Namespace,
		Name:         r.PathValue("name"),
		Container:    req.Container,
		Follow:       req.Follow,
		TailLines:    tailLines,
		SinceSeconds: sinceSeconds,
		SinceTime:    sinceTime,
		Previous:     req.Previous,
		Timestamps:   req.Timestamps,
	})
	if err != nil {
		writeK8sError(w, err)
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

// ExecPod 在 Pod 中执行命令。
func (h *K8sHandler) ExecPod(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Namespace string   `json:"namespace"`
		Container string   `json:"container"`
		Command   []string `json:"command" binding:"required"`
		TTY       bool     `json:"tty"`
	}
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return
	}

	result, err := h.ctx.K8sLogic.ExecPod(r.Context(), logic.PodExecOptions{
		Namespace: req.Namespace,
		Name:      r.PathValue("name"),
		Container: req.Container,
		Command:   req.Command,
		TTY:       req.TTY,
	})
	if err != nil {
		httpx.WriteHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OkJSON(w, result)
}

// PodTerminal Pod 交互式终端（WebSocket）。
// GET /api/v1/k8s/pods/{name}/terminal?namespace=xxx&container=xxx
// 升级为 WebSocket 后，将 exec 的 stdin/stdout 与 ws 双向桥接。
func (h *K8sHandler) PodTerminal(w http.ResponseWriter, r *http.Request) {
	namespace := httpx.QueryValue[string](r, "namespace")
	container := r.URL.Query().Get("container")
	podName := r.PathValue("name")
	if podName == "" {
		http.Error(w, "missing pod name", http.StatusBadRequest)
		return
	}

	HandleWS(w, r, func() (WSStream, error) {
		return h.ctx.K8sLogic.ExecPodInteractive(r.Context(), logic.PodExecOptions{
			Namespace: namespace,
			Name:      podName,
			Container: container,
			Command:   []string{"/bin/sh"},
			TTY:       true,
		})
	})
}

// parseDuration 解析时间字符串，支持 "5m"、"1h"、"3600"（纯数字视为秒）。
func parseDuration(s string) (time.Duration, error) {
	// 纯数字 → 视为秒。
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.Duration(n) * time.Second, nil
	}
	return time.ParseDuration(s)
}
