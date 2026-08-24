package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
	"gopkg.in/yaml.v3"
)

// SwarmServiceItem 服务列表项。
type SwarmServiceItem struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Mode      string   `json:"mode"`     // replicated / global
	Replicas  string   `json:"replicas"` // 如 "2/3"
	Image     string   `json:"image"`
	Ports     []string `json:"ports"`
	State     string   `json:"state"` // running / partially / down / ""
	UpdatedAt string   `json:"updated_at"`
	HasUpdate bool     `json:"has_update"`
}

// toSwarmServiceItem swarm.Service → 列表项。
func toSwarmServiceItem(s *swarm.Service) SwarmServiceItem {
	item := SwarmServiceItem{
		ID:        s.ID,
		Name:      s.Spec.Annotations.Name,
		Image:     imageName(s.Spec.TaskTemplate.ContainerSpec),
		UpdatedAt: s.Meta.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
	if s.Spec.Mode.Replicated != nil {
		item.Mode = "replicated"
		item.Replicas = strconv.FormatUint(*s.Spec.Mode.Replicated.Replicas, 10)
	} else if s.Spec.Mode.Global != nil {
		item.Mode = "global"
		item.Replicas = "global"
	}
	// 实际运行数（ServiceStatus 由 ServiceList Status=true 返回）。
	if s.ServiceStatus != nil {
		if item.Mode == "global" {
			item.Replicas = fmt.Sprintf("%d", s.ServiceStatus.RunningTasks)
		} else {
			item.Replicas = fmt.Sprintf("%d/%d", s.ServiceStatus.RunningTasks, s.ServiceStatus.DesiredTasks)
		}
		item.HasUpdate = s.ServiceStatus.RunningTasks != s.ServiceStatus.DesiredTasks
		switch {
		case item.Mode == "global":
			item.State = "running"
		case s.ServiceStatus.RunningTasks >= s.ServiceStatus.DesiredTasks:
			item.State = "running"
		case s.ServiceStatus.RunningTasks == 0:
			item.State = "down"
		default:
			item.State = "partially"
		}
	}
	if len(s.Endpoint.Spec.Ports) > 0 {
		for _, p := range s.Endpoint.Spec.Ports {
			if p.PublishedPort > 0 {
				item.Ports = append(item.Ports, fmt.Sprintf("%d→%d/%s", p.PublishedPort, p.TargetPort, p.Protocol))
			}
		}
	}
	return item
}

// imageName 提取容器镜像名。
func imageName(cs *swarm.ContainerSpec) string {
	if cs == nil {
		return ""
	}
	return cs.Image
}

// ListServices 服务列表。
func (l *SwarmLogic) ListServices(ctx context.Context) ([]SwarmServiceItem, error) {
	cli, close, err := l.newClient(ctx)
	if err != nil {
		return nil, err
	}
	defer close()

	res, err := cli.ServiceList(ctx, client.ServiceListOptions{Status: true})
	if err != nil {
		return nil, err
	}
	items := make([]SwarmServiceItem, 0, len(res.Items))
	for i := range res.Items {
		items = append(items, toSwarmServiceItem(&res.Items[i]))
	}
	return items, nil
}

// InspectService 服务详情（原始 inspect JSON）。
func (l *SwarmLogic) InspectService(ctx context.Context, id string) (json.RawMessage, error) {
	cli, close, err := l.newClient(ctx)
	if err != nil {
		return nil, err
	}
	defer close()

	res, err := cli.ServiceInspect(ctx, id, client.ServiceInspectOptions{})
	if err != nil {
		return nil, err
	}
	return res.Raw, nil
}

// ServiceRequest 服务创建/更新请求：透传完整 ServiceSpec（YAML 或 JSON 格式）。
// 由前端表单或用户直接提供，后端只负责解析与提交，不再逐字段映射。
type ServiceRequest struct {
	Spec string `json:"spec,omitempty"`
}

// parseServiceSpec 解析 ServiceSpec 文本，兼容 YAML 与 JSON 两种格式。
// 路径：文本 → 通用结构 → JSON → json.Unmarshal，以完全兼容 docker 的 json tag。
func parseServiceSpec(text string) (*swarm.ServiceSpec, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil, fmt.Errorf("服务定义不能为空")
	}
	var raw any
	if strings.HasPrefix(trimmed, "{") {
		if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
			return nil, fmt.Errorf("JSON 解析失败：%v", err)
		}
	} else {
		if err := yaml.Unmarshal([]byte(trimmed), &raw); err != nil {
			return nil, fmt.Errorf("YAML 解析失败：%v", err)
		}
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("定义转 JSON 失败：%v", err)
	}
	var spec swarm.ServiceSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("服务定义解析失败：%v", err)
	}
	return &spec, nil
}

// resolveSpecRefs 将 spec 中的网络 / Secret / Config 名称解析为 ID。
// Docker API 要求引用使用 ID，这里支持直接填写名称（更友好）。
func (l *SwarmLogic) resolveSpecRefs(ctx context.Context, cli *client.Client, spec *swarm.ServiceSpec) error {
	// 网络：target 名称 → ID。
	if len(spec.TaskTemplate.Networks) > 0 {
		nres, err := cli.NetworkList(ctx, client.NetworkListOptions{})
		if err != nil {
			return err
		}
		byName := map[string]string{}
		byID := map[string]bool{}
		for _, n := range nres.Items {
			byName[n.Name] = n.ID
			byID[n.ID] = true
		}
		for i := range spec.TaskTemplate.Networks {
			target := spec.TaskTemplate.Networks[i].Target
			if target == "" || byID[target] {
				continue
			}
			id, ok := byName[target]
			if !ok {
				return fmt.Errorf("网络 %q 不存在", target)
			}
			spec.TaskTemplate.Networks[i].Target = id
		}
	}

	cs := spec.TaskTemplate.ContainerSpec
	if cs == nil {
		return nil
	}
	// Secret：名称 → ID（文件挂载目标保留名称）。
	if len(cs.Secrets) > 0 {
		sres, err := cli.SecretList(ctx, client.SecretListOptions{})
		if err != nil {
			return err
		}
		byName := map[string]string{}
		for _, s := range sres.Items {
			byName[s.Spec.Annotations.Name] = s.ID
		}
		for _, ref := range cs.Secrets {
			if ref.SecretID == "" && ref.SecretName != "" {
				id, ok := byName[ref.SecretName]
				if !ok {
					return fmt.Errorf("Secret %q 不存在", ref.SecretName)
				}
				ref.SecretID = id
			}
		}
	}
	// Config：名称 → ID。
	if len(cs.Configs) > 0 {
		cres, err := cli.ConfigList(ctx, client.ConfigListOptions{})
		if err != nil {
			return err
		}
		byName := map[string]string{}
		for _, c := range cres.Items {
			byName[c.Spec.Annotations.Name] = c.ID
		}
		for _, ref := range cs.Configs {
			if ref.ConfigID == "" && ref.ConfigName != "" {
				id, ok := byName[ref.ConfigName]
				if !ok {
					return fmt.Errorf("Config %q 不存在", ref.ConfigName)
				}
				ref.ConfigID = id
			}
		}
	}
	return nil
}

// CreateService 创建服务：解析 spec（YAML/JSON）并创建。
func (l *SwarmLogic) CreateService(ctx context.Context, req ServiceRequest) error {
	spec, err := parseServiceSpec(req.Spec)
	if err != nil {
		return err
	}
	if spec.Annotations.Name == "" {
		return fmt.Errorf("服务名称不能为空（annotations.name）")
	}
	if spec.TaskTemplate.ContainerSpec == nil || spec.TaskTemplate.ContainerSpec.Image == "" {
		return fmt.Errorf("镜像不能为空（taskTemplate.containerSpec.image）")
	}
	cli, close, err := l.newClient(ctx)
	if err != nil {
		return err
	}
	defer close()
	if err := l.resolveSpecRefs(ctx, cli, spec); err != nil {
		return err
	}
	if err := validateSwarmSpec(spec); err != nil {
		return err
	}
	_, err = cli.ServiceCreate(ctx, client.ServiceCreateOptions{Spec: *spec})
	return err
}

// UpdateService 更新服务：按请求中的 spec 全量替换。
func (l *SwarmLogic) UpdateService(ctx context.Context, id string, req ServiceRequest) error {
	cli, close, err := l.newClient(ctx)
	if err != nil {
		return err
	}
	defer close()

	res, err := cli.ServiceInspect(ctx, id, client.ServiceInspectOptions{})
	if err != nil {
		return err
	}
	svc := res.Service

	newSpec, err := parseServiceSpec(req.Spec)
	if err != nil {
		return err
	}
	if newSpec.Annotations.Name != "" && newSpec.Annotations.Name != svc.Spec.Annotations.Name {
		return fmt.Errorf("服务名称不允许修改")
	}
	if err := l.resolveSpecRefs(ctx, cli, newSpec); err != nil {
		return err
	}
	if err := validateSwarmSpec(newSpec); err != nil {
		return err
	}
	_, err = cli.ServiceUpdate(ctx, id, client.ServiceUpdateOptions{
		Version: svc.Meta.Version,
		Spec:    *newSpec,
	})
	return err
}

// ScaleService 服务伸缩（仅改副本数）。
func (l *SwarmLogic) ScaleService(ctx context.Context, id string, replicas uint64) error {
	cli, close, err := l.newClient(ctx)
	if err != nil {
		return err
	}
	defer close()

	res, err := cli.ServiceInspect(ctx, id, client.ServiceInspectOptions{})
	if err != nil {
		return err
	}
	spec := res.Service.Spec
	if spec.Mode.Replicated == nil {
		return fmt.Errorf("仅 replicated 模式服务支持伸缩")
	}
	spec.Mode.Replicated.Replicas = &replicas
	_, err = cli.ServiceUpdate(ctx, id, client.ServiceUpdateOptions{
		Version: res.Service.Meta.Version,
		Spec:    spec,
	})
	return err
}

// RollbackService 回滚服务到上一版本（docker service rollback）。
func (l *SwarmLogic) RollbackService(ctx context.Context, id string) error {
	cli, close, err := l.newClient(ctx)
	if err != nil {
		return err
	}
	defer close()

	res, err := cli.ServiceInspect(ctx, id, client.ServiceInspectOptions{})
	if err != nil {
		return err
	}
	_, err = cli.ServiceUpdate(ctx, id, client.ServiceUpdateOptions{
		Version:  res.Service.Meta.Version,
		Spec:     res.Service.Spec,
		Rollback: "previous",
	})
	return err
}

// ForceUpdateService 强制更新（docker service update --force），用于恢复暂停的更新或滚动重启。
func (l *SwarmLogic) ForceUpdateService(ctx context.Context, id string) error {
	cli, close, err := l.newClient(ctx)
	if err != nil {
		return err
	}
	defer close()

	res, err := cli.ServiceInspect(ctx, id, client.ServiceInspectOptions{})
	if err != nil {
		return err
	}
	spec := res.Service.Spec
	spec.TaskTemplate.ForceUpdate++
	_, err = cli.ServiceUpdate(ctx, id, client.ServiceUpdateOptions{
		Version: res.Service.Meta.Version,
		Spec:    spec,
	})
	return err
}

// RemoveService 删除服务。
func (l *SwarmLogic) RemoveService(ctx context.Context, id string) error {
	cli, close, err := l.newClient(ctx)
	if err != nil {
		return err
	}
	defer close()
	_, err = cli.ServiceRemove(ctx, id, client.ServiceRemoveOptions{})
	return err
}

// validateSwarmSpec 基础校验：container spec 必须存在。
func validateSwarmSpec(spec *swarm.ServiceSpec) error {
	if spec.TaskTemplate.ContainerSpec == nil {
		return fmt.Errorf("container spec 缺失")
	}
	return nil
}

// SwarmTaskItem 任务列表项。
type SwarmTaskItem struct {
	ID           string `json:"id"`
	ServiceID    string `json:"service_id"`
	ServiceName  string `json:"service_name"`
	NodeID       string `json:"node_id"`
	NodeName     string `json:"node_name"`
	Slot         int    `json:"slot"`
	Image        string `json:"image"`
	State        string `json:"state"`
	DesiredState string `json:"desired_state"`
	Error        string `json:"error,omitempty"`
	ContainerID  string `json:"container_id,omitempty"`
	UpdatedAt    string `json:"updated_at"`
}

// toSwarmTaskItem swarm.Task → 列表项。
func toSwarmTaskItem(t *swarm.Task) SwarmTaskItem {
	item := SwarmTaskItem{
		ID:           t.ID,
		ServiceID:    t.ServiceID,
		NodeID:       t.NodeID,
		Slot:         t.Slot,
		State:        string(t.Status.State),
		DesiredState: string(t.DesiredState),
		Error:        t.Status.Err,
		UpdatedAt:    t.Status.Timestamp.Format("2006-01-02 15:04:05"),
	}
	if t.Spec.ContainerSpec != nil {
		item.Image = t.Spec.ContainerSpec.Image
	}
	if t.Status.ContainerStatus != nil {
		item.ContainerID = t.Status.ContainerStatus.ContainerID
	}
	return item
}

// ListTasks 任务列表；serviceID 非空时仅返回该服务任务。
func (l *SwarmLogic) ListTasks(ctx context.Context, serviceID string) ([]SwarmTaskItem, error) {
	cli, close, err := l.newClient(ctx)
	if err != nil {
		return nil, err
	}
	defer close()

	opts := client.TaskListOptions{Filters: client.Filters{}}
	if serviceID != "" {
		opts.Filters.Add("service", serviceID)
	}
	res, err := cli.TaskList(ctx, opts)
	if err != nil {
		return nil, err
	}

	// 构建服务名映射。
	nameMap := map[string]string{}
	if svcRes, err := cli.ServiceList(ctx, client.ServiceListOptions{}); err == nil {
		for _, s := range svcRes.Items {
			nameMap[s.ID] = s.Spec.Annotations.Name
		}
	}
	nodeMap := map[string]string{}
	if nodeRes, err := cli.NodeList(ctx, client.NodeListOptions{}); err == nil {
		for _, n := range nodeRes.Items {
			name := n.Spec.Annotations.Name
			if name == "" {
				name = n.Description.Hostname
			}
			nodeMap[n.ID] = name
		}
	}

	items := make([]SwarmTaskItem, 0, len(res.Items))
	for i := range res.Items {
		item := toSwarmTaskItem(&res.Items[i])
		item.ServiceName = nameMap[item.ServiceID]
		item.NodeName = nodeMap[item.NodeID]
		items = append(items, item)
	}
	return items, nil
}

// StreamServiceLogs 获取服务日志流（调用方负责关闭）。
// serviceID 支持服务 ID 或名称。
func (l *SwarmLogic) StreamServiceLogs(ctx context.Context, serviceID, tail string, follow bool) (io.ReadCloser, error) {
	cli, close, err := l.newClient(ctx)
	if err != nil {
		return nil, err
	}
	res, err := cli.ServiceLogs(ctx, serviceID, client.ServiceLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     follow,
		Tail:       tail,
		Timestamps: true,
	})
	if err != nil {
		close()
		return nil, err
	}
	// 与容器日志一样是 Docker 帧格式，解帧输出。
	return &logReadCloser{rc: newDemuxLogReader(res), closer: closeFunc{fn: close}}, nil
}

// closeFunc 适配关闭函数为 io.Closer。
type closeFunc struct {
	fn func()
}

func (c closeFunc) Close() error {
	if c.fn != nil {
		c.fn()
	}
	return nil
}

// StreamTaskLogs 获取单个任务的容器日志（任务 → 容器 → 日志，解帧输出）。
func (l *SwarmLogic) StreamTaskLogs(ctx context.Context, taskID, tail string, follow bool) (io.ReadCloser, error) {
	cli, close, err := l.newClient(ctx)
	if err != nil {
		return nil, err
	}

	// 查任务容器 ID。
	opts := client.TaskListOptions{Filters: client.Filters{}}
	opts.Filters.Add("id", taskID)
	res, err := cli.TaskList(ctx, opts)
	if err != nil {
		close()
		return nil, err
	}
	if len(res.Items) == 0 || res.Items[0].Status.ContainerStatus == nil {
		close()
		return nil, fmt.Errorf("任务 %s 无容器（可能未运行）", taskID)
	}
	containerID := res.Items[0].Status.ContainerStatus.ContainerID

	cres, err := cli.ContainerLogs(ctx, containerID, client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     follow,
		Tail:       tail,
		Timestamps: true,
	})
	if err != nil {
		close()
		return nil, err
	}
	return &logReadCloser{rc: newDemuxLogReader(cres), closer: closeFunc{fn: close}}, nil
}

// SwarmContainerResource 服务任务容器资源统计。
type SwarmContainerResource struct {
	TaskID      string  `json:"task_id"`
	ContainerID string  `json:"container_id"`
	Service     string  `json:"service"`
	Slot        int     `json:"slot"`
	NodeName    string  `json:"node_name"`
	State       string  `json:"state"`
	CPUPercent  float64 `json:"cpu_percent"`
	MemUsage    uint64  `json:"mem_usage"`
	MemLimit    uint64  `json:"mem_limit"`
	MemPercent  float64 `json:"mem_percent"`
}

// ServiceResources 服务级资源监控：聚合该服务所有任务容器的实时 CPU/内存。
func (l *SwarmLogic) ServiceResources(ctx context.Context, serviceID string) ([]SwarmContainerResource, error) {
	cli, close, err := l.newClient(ctx)
	if err != nil {
		return nil, err
	}
	defer close()

	opts := client.TaskListOptions{Filters: client.Filters{}}
	if serviceID != "" {
		opts.Filters.Add("service", serviceID)
	}
	res, err := cli.TaskList(ctx, opts)
	if err != nil {
		return nil, err
	}

	// 服务名映射。
	nameMap := map[string]string{}
	if svcRes, err := cli.ServiceList(ctx, client.ServiceListOptions{}); err == nil {
		for _, s := range svcRes.Items {
			nameMap[s.ID] = s.Spec.Annotations.Name
		}
	}
	// 节点名映射。
	nodeMap := map[string]string{}
	if nodeRes, err := cli.NodeList(ctx, client.NodeListOptions{}); err == nil {
		for _, n := range nodeRes.Items {
			name := n.Spec.Annotations.Name
			if name == "" {
				name = n.Description.Hostname
			}
			nodeMap[n.ID] = name
		}
	}

	items := make([]SwarmContainerResource, 0, len(res.Items))
	for i := range res.Items {
		t := &res.Items[i]
		item := SwarmContainerResource{
			TaskID:   t.ID,
			Service:  nameMap[t.ServiceID],
			Slot:     int(t.Slot),
			NodeName: nodeMap[t.NodeID],
			State:    string(t.Status.State),
		}
		if t.Status.ContainerStatus != nil {
			item.ContainerID = t.Status.ContainerStatus.ContainerID
		}
		// 仅对 running 容器采样。
		if item.State == "running" && item.ContainerID != "" {
			if stats, err := l.containerStats(ctx, cli, item.ContainerID); err == nil {
				item.CPUPercent = stats.CPUPercent
				item.MemUsage = stats.MemUsage
				item.MemLimit = stats.MemLimit
				item.MemPercent = stats.MemPercent
			}
		}
		items = append(items, item)
	}
	return items, nil
}

// containerStats 对单容器取一次资源快照（复用 calcStats 计算逻辑）。
func (l *SwarmLogic) containerStats(ctx context.Context, cli *client.Client, id string) (*ContainerStats, error) {
	res, err := cli.ContainerStats(ctx, id, client.ContainerStatsOptions{Stream: false})
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var s container.StatsResponse
	if err := json.NewDecoder(res.Body).Decode(&s); err != nil {
		return nil, err
	}
	return calcStats(&s), nil
}
