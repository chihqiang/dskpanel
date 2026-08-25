package logic

import (
	"context"
	"io"
	"sync"

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
	cancel context.CancelFunc
	done   chan struct{}
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
func (s *k8sExecStream) Close() error {
	s.cancel()
	s.stdin.Close()
	s.stdout.Close()
	<-s.done
	return nil
}

// Resize 更新 TTY 尺寸。
func (s *k8sExecStream) Resize(cols, rows uint16) error {
	s.sizeQ <- remotecommand.TerminalSize{Width: cols, Height: rows}
	return nil
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

	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})

	stream := &k8sExecStream{
		stdin:  stdinW,
		stdout: stdoutR,
		sizeQ:  sizeQ,
		cancel: cancel,
		done:   done,
	}

	go func() {
		defer close(done)
		defer stdinR.Close()
		defer stdoutW.Close()

		err = executor.StreamWithContext(ctx, remotecommand.StreamOptions{
			Stdin:             stdinR,
			Stdout:            stdoutW,
			Stderr:            stdoutW,
			Tty:               true,
			TerminalSizeQueue: &terminalSizeQueue{ch: sizeQ},
		})
		_ = err
	}()

	return stream, nil
}

// terminalSizeQueue 实现 remotecommand.TerminalSizeQueue。
type terminalSizeQueue struct {
	ch chan remotecommand.TerminalSize
}

func (q *terminalSizeQueue) Next() *remotecommand.TerminalSize {
	size, ok := <-q.ch
	if !ok {
		return nil
	}
	return &size
}

// ensure sync package is used.
var _ = sync.WaitGroup{}
