package logic

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

// K8sExecStream 交互式 exec 的双向流接口。
// Reader 读取容器 stdout/stderr，Writer 写入容器 stdin，
// Resize 更新 TTY 尺寸，Close 关闭流。
type K8sExecStream interface {
	io.ReadWriteCloser
	Resize(cols, rows uint16) error
}

// k8sExecStream 交互式 exec 流实现。
type k8sExecStream struct {
	stdin  *io.PipeWriter
	stdout *io.PipeReader
	sizeQ  chan remotecommand.TerminalSize
	closed chan struct{}
	cancel context.CancelFunc
	done   chan struct{}
	once   sync.Once
}

// Read 读取容器输出。
func (s *k8sExecStream) Read(p []byte) (int, error) {
	return s.stdout.Read(p)
}

// Write 写入容器 stdin。
func (s *k8sExecStream) Write(p []byte) (int, error) {
	return s.stdin.Write(p)
}

// Close 关闭流。
// 通过 closed 信号 + 超时等待流 goroutine 结束，避免连接卡死时永久阻塞。
func (s *k8sExecStream) Close() error {
	s.once.Do(func() {
		close(s.closed)
		s.cancel()
		_ = s.stdin.Close()
		_ = s.stdout.Close()
		select {
		case <-s.done:
		case <-time.After(2 * time.Second):
		}
	})
	return nil
}

// Resize 更新 TTY 尺寸。
// 流已关闭时立即返回，避免在无接收方时永久阻塞。
func (s *k8sExecStream) Resize(cols, rows uint16) error {
	select {
	case s.sizeQ <- remotecommand.TerminalSize{Width: cols, Height: rows}:
		return nil
	case <-s.closed:
		return io.ErrClosedPipe
	}
}

// ExecPodInteractive 交互式 exec（TTY 模式），返回双向流。
// 调用方负责关闭返回的流。
func (l *K8sLogic) ExecPodInteractive(ctx context.Context, opts PodExecOptions) (K8sExecStream, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	ns := l.namespace()
	if opts.Namespace != "" {
		ns = opts.Namespace
	}

	req := cli.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(ns).
		Name(opts.Name).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: opts.Container,
			Command:   opts.Command,
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, scheme.ParameterCodec)

	restCfg, err := l.restConfig()
	if err != nil {
		return nil, err
	}
	executor, err := remotecommand.NewSPDYExecutor(restCfg, "POST", req.URL())
	if err != nil {
		return nil, err
	}

	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()
	sizeQ := make(chan remotecommand.TerminalSize, 1)
	closed := make(chan struct{})

	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})

	stream := &k8sExecStream{
		stdin:  stdinW,
		stdout: stdoutR,
		sizeQ:  sizeQ,
		closed: closed,
		cancel: cancel,
		done:   done,
	}

	go func() {
		defer close(done)
		defer stdinR.Close()
		defer stdoutW.Close()

		err := executor.StreamWithContext(ctx, remotecommand.StreamOptions{
			Stdin:             stdinR,
			Stdout:            stdoutW,
			Stderr:            stdoutW,
			Tty:               true,
			TerminalSizeQueue: &terminalSizeQueue{ch: sizeQ, closed: closed},
		})
		// 将流错误回显到终端，使客户端能看到失败原因而非静默断开。
		if err != nil && !errors.Is(err, context.Canceled) {
			_, _ = stdoutW.Write([]byte("\r\n[exec stream error] " + err.Error() + "\r\n"))
		}
	}()

	return stream, nil
}

// terminalSizeQueue 实现 remotecommand.TerminalSizeQueue。
type terminalSizeQueue struct {
	ch     chan remotecommand.TerminalSize
	closed chan struct{}
}

func (q *terminalSizeQueue) Next() *remotecommand.TerminalSize {
	select {
	case size, ok := <-q.ch:
		if !ok {
			return nil
		}
		return &size
	case <-q.closed:
		return nil
	}
}
