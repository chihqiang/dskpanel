package logic

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chihqiang/infra-go/logger"
	"github.com/compose-spec/compose-go/v2/cli"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/compose/v5/pkg/api"
	"github.com/docker/compose/v5/pkg/compose"
	"github.com/moby/moby/api/types/container"
)

// ComposeValidateResult 校验结果。
type ComposeValidateResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

// ComposeDeployResult 部署结果。
type ComposeDeployResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

// ComposeLogic Compose 编排逻辑（基于 docker/compose Go SDK，不再 exec docker CLI）。
// 连接从 DOCKER_HOST 环境变量初始化，天然兼容 Docker / Podman 等运行时。
// 编排文件持久化到配置目录（backupDir）作为备份，不自动清理。
type ComposeLogic struct {
	backupDir string
}

// NewComposeLogic 创建 Compose 编排逻辑，dir 为编排文件备份目录。
func NewComposeLogic(dir string) *ComposeLogic {
	return &ComposeLogic{backupDir: dir}
}

// newComposeService 创建 compose SDK 服务。
// 返回 service 与 close 函数（关闭底层 Docker API 客户端连接）。
func newComposeService(opts ...compose.Option) (api.Compose, func(), error) {
	dockerCli, err := command.NewDockerCli()
	if err != nil {
		return nil, nil, err
	}
	if err := dockerCli.Initialize(flags.NewClientOptions()); err != nil {
		return nil, nil, err
	}
	svc, err := compose.NewComposeService(dockerCli, opts...)
	if err != nil {
		_ = dockerCli.Client().Close()
		return nil, nil, err
	}
	return svc, func() { _ = dockerCli.Client().Close() }, nil
}

// validateProject 解析并校验 Compose 文件（compose-go 轻量加载，不注入 label，仅校验）。
func validateProject(ctx context.Context, path string) (*types.Project, error) {
	opts, err := cli.NewProjectOptions([]string{path}, cli.WithWorkingDirectory(filepath.Dir(path)))
	if err != nil {
		return nil, err
	}
	return opts.LoadProject(ctx)
}

// loadProject 用 compose SDK 加载项目模型（注入 project/service label，供 Up 使用）。
func loadProject(ctx context.Context, svc api.Compose, path string) (*types.Project, error) {
	return svc.LoadProject(ctx, api.ProjectLoadOptions{
		ConfigPaths: []string{path},
		WorkingDir:  filepath.Dir(path),
	})
}

// Validate 校验 Compose 文件（解析 + 加载模型）。
func (l *ComposeLogic) Validate(ctx context.Context, content string) *ComposeValidateResult {
	path, err := l.writeBackup(content)
	if err != nil {
		logger.ErrorCtx(ctx, "compose validate failed", logger.Err(err))
		return &ComposeValidateResult{OK: false, Message: err.Error()}
	}

	if _, err := validateProject(ctx, path); err != nil {
		logger.WarnCtx(ctx, "compose validate rejected", logger.String("file", path), logger.Err(err))
		return &ComposeValidateResult{OK: false, Message: err.Error()}
	}
	logger.InfoCtx(ctx, "compose validate ok", logger.String("file", path))
	return &ComposeValidateResult{OK: true, Message: "valid"}
}

// Deploy 部署 Compose 应用（compose up -d）。
func (l *ComposeLogic) Deploy(ctx context.Context, content string) *ComposeDeployResult {
	path, err := l.writeBackup(content)
	if err != nil {
		logger.ErrorCtx(ctx, "compose deploy failed", logger.Err(err))
		return &ComposeDeployResult{OK: false, Message: err.Error()}
	}

	svc, closeFn, err := newComposeService()
	if err != nil {
		logger.ErrorCtx(ctx, "compose service init failed", logger.Err(err))
		return &ComposeDeployResult{OK: false, Message: err.Error()}
	}
	defer closeFn()

	project, err := loadProject(ctx, svc, path)
	if err != nil {
		msg := "invalid compose: " + err.Error()
		logger.WarnCtx(ctx, "compose deploy rejected", logger.String("file", path), logger.Err(err))
		return &ComposeDeployResult{OK: false, Message: msg}
	}

	logger.InfoCtx(ctx, "compose deploy start", logger.String("file", path))
	if err := svc.Up(ctx, project, api.UpOptions{
		Create: api.CreateOptions{},
		Start:  api.StartOptions{Project: project},
	}); err != nil {
		logger.ErrorCtx(ctx, "compose deploy failed", logger.String("file", path), logger.Err(err))
		return &ComposeDeployResult{OK: false, Message: err.Error()}
	}
	logger.InfoCtx(ctx, "compose deploy ok", logger.String("file", path))
	return &ComposeDeployResult{OK: true, Message: "deployed"}
}

// DeployStream 流式部署 Compose（compose up -d），逐行回调进度事件（含校验错误/进度）。
// 返回是否成功；callback 每收到一条 compose 进度事件调用一次。
func (l *ComposeLogic) DeployStream(ctx context.Context, content string, onLine func(string)) (bool, error) {
	path, err := l.writeBackup(content)
	if err != nil {
		logger.ErrorCtx(ctx, "compose deploy failed", logger.Err(err))
		return false, err
	}

	svc, closeFn, err := newComposeService(compose.WithEventProcessor(&composeEventForwarder{onLine: onLine}))
	if err != nil {
		logger.ErrorCtx(ctx, "compose service init failed", logger.Err(err))
		return false, err
	}
	defer closeFn()

	project, err := loadProject(ctx, svc, path)
	if err != nil {
		msg := "invalid compose: " + err.Error()
		logger.WarnCtx(ctx, "compose deploy rejected", logger.String("file", path), logger.Err(err))
		onLine(msg)
		return false, nil
	}

	logger.InfoCtx(ctx, "compose deploy start", logger.String("file", path))
	if err := svc.Up(ctx, project, api.UpOptions{
		Create: api.CreateOptions{},
		Start:  api.StartOptions{Project: project},
	}); err != nil {
		logger.ErrorCtx(ctx, "compose deploy failed", logger.String("file", path), logger.Err(err))
		return false, err
	}
	logger.InfoCtx(ctx, "compose deploy ok", logger.String("file", path))
	return true, nil
}

// composeEventForwarder 将 compose 操作进度事件转发为逐行回调（供 SSE 回显）。
type composeEventForwarder struct {
	onLine func(string)
}

// Start 操作开始。
func (e *composeEventForwarder) Start(_ context.Context, operation string) {
	e.onLine("Starting " + operation)
}

// On 进度事件（资源状态变化）。
func (e *composeEventForwarder) On(events ...api.Resource) {
	for _, ev := range events {
		line := ev.Text
		if line == "" {
			line = ev.ID
		}
		if ev.Details != "" {
			line += ": " + ev.Details
		}
		if line != "" {
			e.onLine(line)
		}
	}
}

// Done 操作完成。失败不在此输出——整体成败由 DeployStream 返回值决定（handler 发 done: success/fail），
// 避免单个子操作的事件误报导致"整体成功却显示 failed"。
func (e *composeEventForwarder) Done(operation string, success bool) {
	if success {
		e.onLine(operation + " done")
	}
}

// writeBackup 将编排文件内容持久化到备份目录（保留备份，不清理），返回文件路径。
// 每次部署写入独立子目录（时间戳唯一），使无顶层 name 时默认项目名（工作目录 basename）唯一，
// 避免 Compose 同名项目互相覆盖（同名=更新，唯一名=新建）。
func (l *ComposeLogic) writeBackup(content string) (string, error) {
	if err := os.MkdirAll(l.backupDir, 0o755); err != nil {
		return "", err
	}
	dir := filepath.Join(l.backupDir, fmt.Sprintf("compose_%s", time.Now().Format("20060102_150405.000")))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "compose.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// ComposeProjectItem 项目列表项。
type ComposeProjectItem struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	ConfigFiles string `json:"config_files"`
	Services    int    `json:"services"`
	Running     int    `json:"running"`
	Total       int    `json:"total"`
}

// ComposeContainerStatus 项目内容器状态。
type ComposeContainerStatus struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Service string   `json:"service"`
	Image   string   `json:"image"`
	State   string   `json:"state"`
	Status  string   `json:"status"`
	Health  string   `json:"health"`
	Ports   []string `json:"ports"`
	Created int64    `json:"created"`
}

// ComposeProjectDetail 项目详情（容器列表 + 聚合状态）。
type ComposeProjectDetail struct {
	Name       string                   `json:"name"`
	Status     string                   `json:"status"`
	Services   int                      `json:"services"`
	Running    int                      `json:"running"`
	Total      int                      `json:"total"`
	Containers []ComposeContainerStatus `json:"containers"`
}

// ListProjects 列出所有 Compose 项目（含服务/容器聚合）。
func (l *ComposeLogic) ListProjects(ctx context.Context) ([]ComposeProjectItem, error) {
	svc, closeFn, err := newComposeService()
	if err != nil {
		logger.ErrorCtx(ctx, "compose service init failed", logger.Err(err))
		return nil, err
	}
	defer closeFn()

	stacks, err := svc.List(ctx, api.ListOptions{All: true})
	if err != nil {
		logger.ErrorCtx(ctx, "compose list failed", logger.Err(err))
		return nil, err
	}

	items := make([]ComposeProjectItem, 0, len(stacks))
	for _, st := range stacks {
		item := ComposeProjectItem{
			Name:        st.Name,
			Status:      st.Status,
			ConfigFiles: st.ConfigFiles,
		}
		// 每个项目聚合服务/容器状态（项目通常较少，N+1 可接受）。
		if summaries, err := svc.Ps(ctx, st.Name, api.PsOptions{All: true}); err == nil {
			services := map[string]struct{}{}
			for _, c := range summaries {
				item.Total++
				if c.Service != "" {
					services[c.Service] = struct{}{}
				}
				if isContainerRunning(c.State) {
					item.Running++
				}
			}
			item.Services = len(services)
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items, nil
}

// ProjectPs 查询单个 Compose 项目内容器状态。
func (l *ComposeLogic) ProjectPs(ctx context.Context, name string) (*ComposeProjectDetail, error) {
	svc, closeFn, err := newComposeService()
	if err != nil {
		logger.ErrorCtx(ctx, "compose service init failed", logger.Err(err))
		return nil, err
	}
	defer closeFn()

	summaries, err := svc.Ps(ctx, name, api.PsOptions{All: true})
	if err != nil {
		logger.ErrorCtx(ctx, "compose ps failed", logger.String("project", name), logger.Err(err))
		return nil, err
	}

	detail := &ComposeProjectDetail{Name: name, Containers: []ComposeContainerStatus{}}
	services := map[string]struct{}{}
	for _, c := range summaries {
		detail.Total++
		if c.Service != "" {
			services[c.Service] = struct{}{}
		}
		if isContainerRunning(c.State) {
			detail.Running++
		}
		ports := make([]string, 0, len(c.Publishers))
		for _, p := range c.Publishers {
			if p.PublishedPort != 0 {
				host := p.URL
				if host == "" {
					host = "0.0.0.0"
				}
				ports = append(ports, fmt.Sprintf("%s:%d->%d/%s", host, p.PublishedPort, p.TargetPort, p.Protocol))
			}
		}
		detail.Containers = append(detail.Containers, ComposeContainerStatus{
			ID:      c.ID,
			Name:    c.Name,
			Service: c.Service,
			Image:   c.Image,
			State:   string(c.State),
			Status:  c.Status,
			Health:  string(c.Health),
			Ports:   ports,
			Created: c.Created,
		})
	}
	detail.Services = len(services)
	return detail, nil
}

// ProjectStart 启动项目所有服务。
func (l *ComposeLogic) ProjectStart(ctx context.Context, name string) error {
	svc, closeFn, err := newComposeService()
	if err != nil {
		logger.ErrorCtx(ctx, "compose service init failed", logger.Err(err))
		return err
	}
	defer closeFn()
	if err := svc.Start(ctx, name, api.StartOptions{}); err != nil {
		logger.ErrorCtx(ctx, "compose start failed", logger.String("project", name), logger.Err(err))
		return err
	}
	return nil
}

// ProjectStop 停止项目所有服务。
func (l *ComposeLogic) ProjectStop(ctx context.Context, name string) error {
	svc, closeFn, err := newComposeService()
	if err != nil {
		logger.ErrorCtx(ctx, "compose service init failed", logger.Err(err))
		return err
	}
	defer closeFn()
	if err := svc.Stop(ctx, name, api.StopOptions{}); err != nil {
		logger.ErrorCtx(ctx, "compose stop failed", logger.String("project", name), logger.Err(err))
		return err
	}
	return nil
}

// ProjectRestart 重启项目所有服务。
func (l *ComposeLogic) ProjectRestart(ctx context.Context, name string) error {
	svc, closeFn, err := newComposeService()
	if err != nil {
		logger.ErrorCtx(ctx, "compose service init failed", logger.Err(err))
		return err
	}
	defer closeFn()
	if err := svc.Restart(ctx, name, api.RestartOptions{}); err != nil {
		logger.ErrorCtx(ctx, "compose restart failed", logger.String("project", name), logger.Err(err))
		return err
	}
	return nil
}

// ProjectDown 停止并移除项目容器与网络；volumes 为 true 时同时删除命名卷。
func (l *ComposeLogic) ProjectDown(ctx context.Context, name string, volumes bool) error {
	svc, closeFn, err := newComposeService()
	if err != nil {
		logger.ErrorCtx(ctx, "compose service init failed", logger.Err(err))
		return err
	}
	defer closeFn()
	if err := svc.Down(ctx, name, api.DownOptions{Volumes: volumes}); err != nil {
		logger.ErrorCtx(ctx, "compose down failed", logger.String("project", name), logger.Err(err))
		return err
	}
	return nil
}

// ProjectLogs 拉取项目日志（非 follow，tail 行，带时间戳）。
func (l *ComposeLogic) ProjectLogs(ctx context.Context, name string, tail int) ([]string, error) {
	svc, closeFn, err := newComposeService()
	if err != nil {
		logger.ErrorCtx(ctx, "compose service init failed", logger.Err(err))
		return nil, err
	}
	defer closeFn()

	collector := &composeLogCollector{}
	opts := api.LogOptions{Timestamps: true, Tail: strconv.Itoa(tail)}
	if err := svc.Logs(ctx, name, collector, opts); err != nil {
		logger.ErrorCtx(ctx, "compose logs failed", logger.String("project", name), logger.Err(err))
		return nil, err
	}
	return collector.Lines(), nil
}

// composeLogCollector 收集 compose 日志行（按容器名前缀）。
type composeLogCollector struct {
	mu    sync.Mutex
	lines []string
}

// Log 普通输出。
func (c *composeLogCollector) Log(containerName, message string) {
	c.append(containerName, message)
}

// Err 错误输出。
func (c *composeLogCollector) Err(containerName, message string) {
	c.append(containerName, message)
}

// Status 状态消息（忽略）。
func (c *composeLogCollector) Status(_ string, _ string) {}

func (c *composeLogCollector) append(containerName, message string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, line := range strings.Split(strings.TrimRight(message, "\n"), "\n") {
		if line == "" {
			continue
		}
		if containerName != "" {
			line = fmt.Sprintf("[%s] %s", containerName, line)
		}
		c.lines = append(c.lines, line)
	}
}

// Lines 返回收集到的日志行。
func (c *composeLogCollector) Lines() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lines
}

// ProjectConfig 读取 Compose 项目的配置文件内容。
// configFiles 为逗号分隔的多个文件路径，返回第一个可读文件的内容。
func (l *ComposeLogic) ProjectConfig(ctx context.Context, name string) (string, error) {
	svc, closeFn, err := newComposeService()
	if err != nil {
		logger.ErrorCtx(ctx, "compose service init failed", logger.Err(err))
		return "", err
	}
	defer closeFn()

	stacks, err := svc.List(ctx, api.ListOptions{All: true})
	if err != nil {
		logger.ErrorCtx(ctx, "compose list failed", logger.Err(err))
		return "", err
	}

	for _, st := range stacks {
		if st.Name != name {
			continue
		}
		if st.ConfigFiles == "" {
			return "", fmt.Errorf("project %s has no config file", name)
		}
		// ConfigFiles 可能是逗号分隔的多个路径，取第一个。
		paths := strings.Split(st.ConfigFiles, ",")
		for _, p := range paths {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			data, err := os.ReadFile(p)
			if err != nil {
				continue
			}
			return string(data), nil
		}
		return "", fmt.Errorf("config file not readable for project %s", name)
	}
	return "", fmt.Errorf("project %s not found", name)
}

// isContainerRunning 判断容器是否处于运行态。
func isContainerRunning(s container.ContainerState) bool {
	return s == container.StateRunning
}
