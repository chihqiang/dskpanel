package logic

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/chihqiang/infra-go/logger"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

// ContainerItem 容器列表项。
type ContainerItem struct {
	ID      string   `json:"id"`
	Names   []string `json:"names"`
	Image   string   `json:"image"`
	ImageID string   `json:"image_id"`
	Command string   `json:"command"`
	State   string   `json:"state"`
	Status  string   `json:"status"`
	Ports   []Port   `json:"ports"`
	Created int64    `json:"created"`
}

// Port 端口映射。
type Port struct {
	IP          string `json:"ip,omitempty"`
	PrivatePort uint16 `json:"private_port"`
	PublicPort  uint16 `json:"public_port,omitempty"`
	Type        string `json:"type"`
}

// ContainerDetail 容器详情。
type ContainerDetail struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	Image           string           `json:"image"`
	State           string           `json:"state"`
	Status          string           `json:"status"`
	Created         string           `json:"created"`
	RestartCnt      int              `json:"restart_count"`
	Config          *Config          `json:"config,omitempty"`
	HostConfig      *HostConfig      `json:"host_config,omitempty"`
	NetworkSettings *NetworkSettings `json:"network_settings,omitempty"`
}

// Config 容器配置摘要。
type Config struct {
	Env          []string            `json:"env,omitempty"`
	Cmd          []string            `json:"cmd,omitempty"`
	Entrypoint   []string            `json:"entrypoint,omitempty"`
	Labels       map[string]string   `json:"labels,omitempty"`
	ExposedPorts map[string]struct{} `json:"exposed_ports,omitempty"`
}

// HostConfig 主机配置摘要。
type HostConfig struct {
	Binds         []string `json:"binds,omitempty"`
	RestartPolicy string   `json:"restart_policy,omitempty"`
	// 资源限制（docker update 可改）。
	CPUShares  int64  `json:"cpu_shares,omitempty"`
	Memory     int64  `json:"memory,omitempty"`
	NanoCPUs   int64  `json:"nano_cpus,omitempty"`
	CpusetCpus string `json:"cpuset_cpus,omitempty"`
	RestartMax int    `json:"restart_max,omitempty"`
}

// NetworkSettings 网络配置摘要。
type NetworkSettings struct {
	Ports    map[string][]Port `json:"ports,omitempty"`
	Networks map[string]struct {
		IPAddress string `json:"ip_address,omitempty"`
	} `json:"networks,omitempty"`
}

// ContainerLogic Docker 容器管理逻辑。
type ContainerLogic struct{}

// NewContainerLogic 创建容器管理逻辑。
func NewContainerLogic() *ContainerLogic {
	return &ContainerLogic{}
}

// newClient 创建本机 Docker 客户端。
func (l *ContainerLogic) newClient() (*client.Client, error) {
	return client.New(client.FromEnv)
}

// List 列出容器。
func (l *ContainerLogic) List(ctx context.Context, all bool) ([]ContainerItem, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	res, err := cli.ContainerList(ctx, client.ContainerListOptions{All: all})
	if err != nil {
		return nil, err
	}

	items := make([]ContainerItem, 0, len(res.Items))
	for _, c := range res.Items {
		ports := make([]Port, 0, len(c.Ports))
		for _, p := range c.Ports {
			ip := ""
			if p.IP.IsValid() {
				ip = p.IP.String()
			}
			ports = append(ports, Port{
				IP:          ip,
				PrivatePort: p.PrivatePort,
				PublicPort:  p.PublicPort,
				Type:        p.Type,
			})
		}
		items = append(items, ContainerItem{
			ID:      c.ID,
			Names:   c.Names,
			Image:   c.Image,
			ImageID: c.ImageID,
			Command: c.Command,
			State:   string(c.State),
			Status:  c.Status,
			Ports:   ports,
			Created: c.Created,
		})
	}
	return items, nil
}

// InspectRaw 返回容器完整 inspect 原始 JSON（排障用）。
func (l *ContainerLogic) InspectRaw(ctx context.Context, id string) (json.RawMessage, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	res, err := cli.ContainerInspect(ctx, id, client.ContainerInspectOptions{})
	if err != nil {
		return nil, err
	}
	return res.Raw, nil
}

// Inspect 查看容器详情。
func (l *ContainerLogic) Inspect(ctx context.Context, id string) (*ContainerDetail, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	res, err := cli.ContainerInspect(ctx, id, client.ContainerInspectOptions{})
	if err != nil {
		return nil, err
	}
	c := res.Container

	detail := &ContainerDetail{
		ID:      c.ID,
		Name:    c.Name,
		Image:   c.Config.Image,
		State:   string(c.State.Status),
		Status:  string(c.State.Status),
		Created: c.Created,
	}
	if c.RestartCount > 0 {
		detail.RestartCnt = c.RestartCount
	}
	if c.Config != nil {
		detail.Config = &Config{
			Env:        c.Config.Env,
			Cmd:        c.Config.Cmd,
			Entrypoint: c.Config.Entrypoint,
			Labels:     c.Config.Labels,
		}
	}
	if c.HostConfig != nil {
		hc := &HostConfig{
			Binds:      c.HostConfig.Binds,
			CPUShares:  c.HostConfig.CPUShares,
			Memory:     c.HostConfig.Memory,
			NanoCPUs:   c.HostConfig.NanoCPUs,
			CpusetCpus: c.HostConfig.CpusetCpus,
		}
		hc.RestartPolicy = string(c.HostConfig.RestartPolicy.Name)
		hc.RestartMax = c.HostConfig.RestartPolicy.MaximumRetryCount
		detail.HostConfig = hc
	}
	if c.NetworkSettings != nil {
		ns := &NetworkSettings{}
		if c.NetworkSettings.Ports != nil {
			pm := make(map[string][]Port)
			for k, v := range c.NetworkSettings.Ports {
				binds := make([]Port, 0, len(v))
				for _, b := range v {
					ip := ""
					if b.HostIP.IsValid() {
						ip = b.HostIP.String()
					}
					binds = append(binds, Port{
						IP:          ip,
						PrivatePort: k.Num(),
						PublicPort:  parsePort(b.HostPort),
						Type:        string(k.Proto()),
					})
				}
				pm[k.String()] = binds
			}
			ns.Ports = pm
		}
		if len(c.NetworkSettings.Networks) > 0 {
			nm := make(map[string]struct {
				IPAddress string `json:"ip_address,omitempty"`
			})
			for name, ep := range c.NetworkSettings.Networks {
				ip := ""
				if ep != nil && ep.IPAddress.IsValid() {
					ip = ep.IPAddress.String()
				}
				nm[name] = struct {
					IPAddress string `json:"ip_address,omitempty"`
				}{IPAddress: ip}
			}
			ns.Networks = nm
		}
		detail.NetworkSettings = ns
	}
	return detail, nil
}

// Start 启动容器。
func (l *ContainerLogic) Start(ctx context.Context, id string) error {
	logger.InfoCtx(ctx, "container start", logger.String("id", id))
	cli, err := l.newClient()
	if err != nil {
		logger.ErrorCtx(ctx, "container start failed", logger.String("id", id), logger.Err(err))
		return err
	}
	defer cli.Close()
	_, err = cli.ContainerStart(ctx, id, client.ContainerStartOptions{})
	if err != nil {
		logger.ErrorCtx(ctx, "container start failed", logger.String("id", id), logger.Err(err))
	}
	return err
}

// Stop 停止容器。
func (l *ContainerLogic) Stop(ctx context.Context, id string) error {
	logger.InfoCtx(ctx, "container stop", logger.String("id", id))
	cli, err := l.newClient()
	if err != nil {
		logger.ErrorCtx(ctx, "container stop failed", logger.String("id", id), logger.Err(err))
		return err
	}
	defer cli.Close()
	_, err = cli.ContainerStop(ctx, id, client.ContainerStopOptions{})
	if err != nil {
		logger.ErrorCtx(ctx, "container stop failed", logger.String("id", id), logger.Err(err))
	}
	return err
}

// Restart 重启容器。
func (l *ContainerLogic) Restart(ctx context.Context, id string) error {
	logger.InfoCtx(ctx, "container restart", logger.String("id", id))
	cli, err := l.newClient()
	if err != nil {
		logger.ErrorCtx(ctx, "container restart failed", logger.String("id", id), logger.Err(err))
		return err
	}
	defer cli.Close()
	_, err = cli.ContainerRestart(ctx, id, client.ContainerRestartOptions{})
	if err != nil {
		logger.ErrorCtx(ctx, "container restart failed", logger.String("id", id), logger.Err(err))
	}
	return err
}

// Remove 删除容器。
func (l *ContainerLogic) Remove(ctx context.Context, id string, force, removeVolumes bool) error {
	logger.InfoCtx(ctx, "container remove",
		logger.String("id", id),
		logger.Bool("force", force),
		logger.Bool("remove_volumes", removeVolumes))
	cli, err := l.newClient()
	if err != nil {
		logger.ErrorCtx(ctx, "container remove failed", logger.String("id", id), logger.Err(err))
		return err
	}
	defer cli.Close()
	_, err = cli.ContainerRemove(ctx, id, client.ContainerRemoveOptions{
		Force:         force,
		RemoveVolumes: removeVolumes,
	})
	if err != nil {
		logger.ErrorCtx(ctx, "container remove failed", logger.String("id", id), logger.Err(err))
	}
	return err
}

// PortMapping 端口映射（创建容器用）。
type PortMapping struct {
	// ContainerPort 容器端口。
	ContainerPort int `json:"container_port" binding:"required,gt=0"`
	// HostPort 宿主机端口。
	HostPort int `json:"host_port" binding:"gte=0"`
	// Protocol 协议：tcp / udp，默认 tcp。
	Protocol string `json:"protocol,default=tcp"`
}

// CreateContainerRequest 创建容器请求。
type CreateContainerRequest struct {
	Name          string            `json:"name"`
	Image         string            `json:"image" binding:"required"`
	Command       []string          `json:"command"`
	Entrypoint    []string          `json:"entrypoint"`
	Env           []string          `json:"env"` // 形如 KEY=VALUE
	Labels        map[string]string `json:"labels"`
	Binds         []string          `json:"binds"` // 卷挂载，形如 host:container
	Ports         []PortMapping     `json:"ports"`
	Network       string            `json:"network"`        // 网络名，默认 bridge
	RestartPolicy string            `json:"restart_policy"` // no / always / on-failure / unless-stopped
	AutoRemove    bool              `json:"auto_remove"`
	Detach        bool              `json:"detach,default=true"`
	TTY           bool              `json:"tty"`
	OpenStdin     bool              `json:"open_stdin"`
	// 高级字段。
	Hostname   string   `json:"hostname"`    // 容器主机名
	User       string   `json:"user"`        // 运行用户（如 root / 1000:1000）
	WorkingDir string   `json:"working_dir"` // 工作目录
	CapAdd     []string `json:"cap_add"`     // 附加内核能力（如 SYS_PTRACE）
	CapDrop    []string `json:"cap_drop"`    // 移除内核能力（如 ALL）
	Memory     int64    `json:"memory"`      // 内存限制（字节）
	NanoCPUs   int64    `json:"nano_cpus"`   // CPU 限制（纳核）
	CpusetCpus string   `json:"cpuset_cpus"` // CPU 亲和（如 0-1）
	EnvFile    []string `json:"env_file"`    // 环境变量文件路径（宿主机），等价 --env-file
	ExtraHosts []string `json:"extra_hosts"` // 追加 hosts 条目（形如 host:ip）
}

// CreateContainerResult 创建容器结果。
type CreateContainerResult struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Warns []string `json:"warns,omitempty"`
}

// Create 创建容器。
func (l *ContainerLogic) Create(ctx context.Context, req *CreateContainerRequest) (*CreateContainerResult, error) {
	logger.InfoCtx(ctx, "container create",
		logger.String("name", req.Name),
		logger.String("image", req.Image),
		logger.String("network", req.Network),
		logger.Int("ports", len(req.Ports)))
	cli, err := l.newClient()
	if err != nil {
		logger.ErrorCtx(ctx, "container create failed", logger.String("image", req.Image), logger.Err(err))
		return nil, err
	}
	defer cli.Close()

	cfg := &container.Config{
		Image:      req.Image,
		Cmd:        req.Command,
		Entrypoint: req.Entrypoint,
		Env:        req.Env,
		Labels:     req.Labels,
		Tty:        req.TTY,
		OpenStdin:  req.OpenStdin,
		Hostname:   req.Hostname,
		User:       req.User,
		WorkingDir: req.WorkingDir,
	}

	// 合并环境变量文件（env_file，宿主机路径）到 Env。
	if len(req.EnvFile) > 0 {
		for _, f := range req.EnvFile {
			envs, err := readEnvFile(f)
			if err != nil {
				logger.WarnCtx(ctx, "env_file read failed", logger.String("file", f), logger.Err(err))
				continue
			}
			cfg.Env = append(cfg.Env, envs...)
		}
	}

	hostCfg := &container.HostConfig{
		Binds:        req.Binds,
		AutoRemove:   req.AutoRemove,
		NetworkMode:  container.NetworkMode(req.Network),
		PortBindings: buildPortBindings(req.Ports),
		CapAdd:       req.CapAdd,
		CapDrop:      req.CapDrop,
		ExtraHosts:   req.ExtraHosts,
	}
	if req.Memory > 0 || req.NanoCPUs > 0 || req.CpusetCpus != "" {
		hostCfg.Resources = container.Resources{
			Memory:     req.Memory,
			NanoCPUs:   req.NanoCPUs,
			CpusetCpus: req.CpusetCpus,
		}
	}
	if req.RestartPolicy != "" {
		hostCfg.RestartPolicy = container.RestartPolicy{
			Name: container.RestartPolicyMode(req.RestartPolicy),
		}
	}

	// 暴露端口。
	if len(req.Ports) > 0 {
		exposed := make(network.PortSet)
		for _, p := range req.Ports {
			proto := p.Protocol
			if proto == "" {
				proto = "tcp"
			}
			pt, ok := network.PortFrom(uint16(p.ContainerPort), network.IPProtocol(proto))
			if ok {
				exposed[pt] = struct{}{}
			}
		}
		cfg.ExposedPorts = exposed
	}

	// 默认网络为 bridge。
	netMode := req.Network
	if netMode == "" {
		netMode = "bridge"
	}
	hostCfg.NetworkMode = container.NetworkMode(netMode)

	res, err := cli.ContainerCreate(ctx, client.ContainerCreateOptions{
		Config:     cfg,
		HostConfig: hostCfg,
		Name:       req.Name,
	})
	if err != nil {
		logger.ErrorCtx(ctx, "container create failed", logger.String("image", req.Image), logger.String("name", req.Name), logger.Err(err))
		return nil, err
	}
	logger.InfoCtx(ctx, "container created", logger.String("id", res.ID), logger.String("name", req.Name))

	result := &CreateContainerResult{
		ID:   res.ID,
		Name: req.Name,
	}
	if len(res.Warnings) > 0 {
		result.Warns = res.Warnings
	}

	// 非 detach 模式自动启动。
	if !req.Detach {
		_, err = cli.ContainerStart(ctx, res.ID, client.ContainerStartOptions{})
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// readEnvFile 读取宿主机 env 文件，返回形如 KEY=VALUE 的切片（等价 docker --env-file）。
// 忽略空行与 # 注释；兼容无 = 的行视为 KEY=。
func readEnvFile(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out, nil
}

// buildPortBindings 构建端口绑定映射。
func buildPortBindings(ports []PortMapping) network.PortMap {
	if len(ports) == 0 {
		return nil
	}
	m := make(network.PortMap)
	for _, p := range ports {
		proto := p.Protocol
		if proto == "" {
			proto = "tcp"
		}
		pt, ok := network.PortFrom(uint16(p.ContainerPort), network.IPProtocol(proto))
		if !ok {
			continue
		}
		bind := network.PortBinding{HostPort: strconv.Itoa(p.HostPort)}
		m[pt] = append(m[pt], bind)
	}
	return m
}

// parsePort 解析端口字符串为 uint16，失败返回 0。
func parsePort(s string) uint16 {
	if s == "" {
		return 0
	}
	n, err := strconv.ParseUint(s, 10, 16)
	if err != nil {
		return 0
	}
	return uint16(n)
}

// LogsOptions 日志读取选项。
type LogsOptions struct {
	ContainerID string
	Tail        string // 如 "100" 或 "all"
	Follow      bool
	Timestamps  bool
	Since       string
}

// StreamLogs 获取容器日志流（调用方负责关闭返回的读取器）。
// 已对 Docker 日志帧（8 字节头 + payload）解帧，仅输出日志文本。
func (l *ContainerLogic) StreamLogs(ctx context.Context, opts LogsOptions) (io.ReadCloser, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	res, err := cli.ContainerLogs(ctx, opts.ContainerID, client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Since:      opts.Since,
		Timestamps: opts.Timestamps,
		Follow:     opts.Follow,
		Tail:       opts.Tail,
	})
	if err != nil {
		cli.Close()
		return nil, err
	}
	// 解帧后输出；cli 随 reader 一起延迟关闭。
	return &logReadCloser{rc: newDemuxLogReader(res), closer: cli}, nil
}

// demuxLogReader 解析 Docker 容器日志帧并输出 payload（日志文本）。
// 帧格式：8 字节头（1 字节 stream type + 3 字节填充 + 4 字节大端 payload 长度）+ payload。
type demuxLogReader struct {
	r    *bufio.Reader
	rc   io.ReadCloser
	left int // 当前帧剩余 payload 字节数
}

// newDemuxLogReader 创建解帧读取器。
func newDemuxLogReader(rc io.ReadCloser) *demuxLogReader {
	return &demuxLogReader{r: bufio.NewReader(rc), rc: rc}
}

func (d *demuxLogReader) Read(p []byte) (int, error) {
	// 已存在待读 payload，直接返回。
	if d.left > 0 {
		n := len(p)
		if n > d.left {
			n = d.left
		}
		rn, err := d.r.Read(p[:n])
		d.left -= rn
		return rn, err
	}

	// 读取 8 字节帧头。
	hdr := make([]byte, 8)
	if _, err := io.ReadFull(d.r, hdr); err != nil {
		return 0, err
	}
	// 解析 payload 长度（第 5-8 字节，大端）。
	length := binary.BigEndian.Uint32(hdr[4:8])
	d.left = int(length)

	if d.left == 0 {
		// 空帧，递归读取下一帧。
		return d.Read(p)
	}
	n := len(p)
	if n > d.left {
		n = d.left
	}
	rn, err := d.r.Read(p[:n])
	d.left -= rn
	return rn, err
}

// Close 关闭底层读取器。
func (d *demuxLogReader) Close() error { return d.rc.Close() }

// logReadCloser 包装日志读取器与客户端关闭。
type logReadCloser struct {
	rc     io.ReadCloser
	closer io.Closer
	once   sync.Once
}

func (l *logReadCloser) Read(p []byte) (int, error) { return l.rc.Read(p) }

func (l *logReadCloser) Close() error {
	var err1, err2 error
	l.once.Do(func() {
		err1 = l.rc.Close()
		if l.closer != nil {
			err2 = l.closer.Close()
		}
	})
	if err1 != nil {
		return err1
	}
	return err2
}

// Stats 读取容器实时资源统计（一次快照）。
func (l *ContainerLogic) Stats(ctx context.Context, id string) (*ContainerStats, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()

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

// ContainerStats 容器资源统计快照。
type ContainerStats struct {
	CPUPercent float64 `json:"cpu_percent"`
	MemUsage   uint64  `json:"mem_usage"`
	MemLimit   uint64  `json:"mem_limit"`
	MemPercent float64 `json:"mem_percent"`
	NetRxBytes uint64  `json:"net_rx_bytes"`
	NetTxBytes uint64  `json:"net_tx_bytes"`
	BlockRead  uint64  `json:"block_read"`
	BlockWrite uint64  `json:"block_write"`
	Pids       uint64  `json:"pids"`
	Running    bool    `json:"running"`
}

// calcStats 由两次 CPU 采样计算 CPU 百分比（与 docker stats 一致）。
func calcStats(s *container.StatsResponse) *ContainerStats {
	cs := &ContainerStats{}
	cpuDelta := float64(s.CPUStats.CPUUsage.TotalUsage)
	sysDelta := float64(s.CPUStats.SystemUsage)
	if s.PreCPUStats.SystemUsage > 0 && s.PreCPUStats.CPUUsage.TotalUsage > 0 {
		cpuDelta = float64(s.CPUStats.CPUUsage.TotalUsage - s.PreCPUStats.CPUUsage.TotalUsage)
		sysDelta = float64(s.CPUStats.SystemUsage - s.PreCPUStats.SystemUsage)
	}
	online := s.CPUStats.OnlineCPUs
	if online == 0 {
		online = 1
	}
	if sysDelta > 0 && cpuDelta > 0 {
		cs.CPUPercent = (cpuDelta / sysDelta) * float64(online) * 100
	}

	cs.MemUsage = s.MemoryStats.Usage
	cs.MemLimit = s.MemoryStats.Limit
	if s.MemoryStats.Limit > 0 {
		cs.MemPercent = float64(s.MemoryStats.Usage) / float64(s.MemoryStats.Limit) * 100
	}

	var rx, tx uint64
	for _, n := range s.Networks {
		rx += n.RxBytes
		tx += n.TxBytes
	}
	cs.NetRxBytes = rx
	cs.NetTxBytes = tx

	for _, bio := range s.BlkioStats.IoServiceBytesRecursive {
		switch bio.Op {
		case "Read":
			cs.BlockRead += bio.Value
		case "Write":
			cs.BlockWrite += bio.Value
		}
	}
	cs.Pids = uint64(s.PidsStats.Current)
	return cs
}

// Commit 将容器提交为镜像（docker commit）。
// reference 为目标镜像名（如 repo:tag）；不传则默认保留原镜像名。
func (l *ContainerLogic) Commit(ctx context.Context, id, reference, comment, author string) (string, error) {
	logger.InfoCtx(ctx, "container commit", logger.String("id", id), logger.String("reference", reference))
	cli, err := l.newClient()
	if err != nil {
		logger.ErrorCtx(ctx, "container commit failed", logger.String("id", id), logger.Err(err))
		return "", err
	}
	defer cli.Close()

	res, err := cli.ContainerCommit(ctx, id, client.ContainerCommitOptions{
		Reference: reference,
		Comment:   comment,
		Author:    author,
	})
	if err != nil {
		logger.ErrorCtx(ctx, "container commit failed", logger.String("id", id), logger.String("reference", reference), logger.Err(err))
		return "", err
	}
	return res.ID, nil
}

// Update 更新容器资源限制与重启策略（docker update）。
// 传 0/空 表示不改动对应项。memorySwap 为 -1 表示无限制 swap；为 0 表示自动跟随 memory（memory*2）。
func (l *ContainerLogic) Update(ctx context.Context, id string, cpuShares, memory, nanoCPUs int64, cpuset string, memorySwap int64, restartPolicy string, restartMax int) error {
	logger.InfoCtx(ctx, "container update", logger.String("id", id), logger.Int64("memory", memory), logger.Int64("nano_cpus", nanoCPUs))
	cli, err := l.newClient()
	if err != nil {
		logger.ErrorCtx(ctx, "container update failed", logger.String("id", id), logger.Err(err))
		return err
	}
	defer cli.Close()

	resources := &container.Resources{
		CPUShares:  cpuShares,
		Memory:     memory,
		NanoCPUs:   nanoCPUs,
		CpusetCpus: cpuset,
	}
	// 设置了 memory 但未显式指定 swap：跟随 memory（Docker 默认 memory*2）。
	if memory > 0 && memorySwap == 0 {
		memorySwap = memory * 2
	}
	if memorySwap != 0 {
		resources.MemorySwap = memorySwap
	}

	opts := client.ContainerUpdateOptions{Resources: resources}
	if restartPolicy != "" {
		opts.RestartPolicy = &container.RestartPolicy{
			Name:              container.RestartPolicyMode(restartPolicy),
			MaximumRetryCount: restartMax,
		}
	}

	if _, err := cli.ContainerUpdate(ctx, id, opts); err != nil {
		logger.ErrorCtx(ctx, "container update failed", logger.String("id", id), logger.Err(err))
		return err
	}
	return nil
}

// Pause 暂停容器。
func (l *ContainerLogic) Pause(ctx context.Context, id string) error {
	logger.InfoCtx(ctx, "container pause", logger.String("id", id))
	cli, err := l.newClient()
	if err != nil {
		logger.ErrorCtx(ctx, "container pause failed", logger.String("id", id), logger.Err(err))
		return err
	}
	defer cli.Close()
	if _, err := cli.ContainerPause(ctx, id, client.ContainerPauseOptions{}); err != nil {
		logger.ErrorCtx(ctx, "container pause failed", logger.String("id", id), logger.Err(err))
		return err
	}
	return nil
}

// Unpause 恢复暂停的容器。
func (l *ContainerLogic) Unpause(ctx context.Context, id string) error {
	logger.InfoCtx(ctx, "container unpause", logger.String("id", id))
	cli, err := l.newClient()
	if err != nil {
		logger.ErrorCtx(ctx, "container unpause failed", logger.String("id", id), logger.Err(err))
		return err
	}
	defer cli.Close()
	if _, err := cli.ContainerUnpause(ctx, id, client.ContainerUnpauseOptions{}); err != nil {
		logger.ErrorCtx(ctx, "container unpause failed", logger.String("id", id), logger.Err(err))
		return err
	}
	return nil
}

// Export 导出容器文件系统（docker export），返回 tar 流。
// 注意：返回的读取器依赖连接，不在此处关闭 cli；由调用方在读取完成后关闭。
func (l *ContainerLogic) Export(ctx context.Context, id string) (io.ReadCloser, error) {
	logger.InfoCtx(ctx, "container export", logger.String("id", id))
	cli, err := l.newClient()
	if err != nil {
		logger.ErrorCtx(ctx, "container export failed", logger.String("id", id), logger.Err(err))
		return nil, err
	}
	res, err := cli.ContainerExport(ctx, id, client.ContainerExportOptions{})
	if err != nil {
		cli.Close()
		logger.ErrorCtx(ctx, "container export failed", logger.String("id", id), logger.Err(err))
		return nil, err
	}
	return res, nil
}

// Rename 重命名容器。
func (l *ContainerLogic) Rename(ctx context.Context, id, newName string) error {
	logger.InfoCtx(ctx, "container rename", logger.String("id", id), logger.String("new_name", newName))
	cli, err := l.newClient()
	if err != nil {
		logger.ErrorCtx(ctx, "container rename failed", logger.String("id", id), logger.Err(err))
		return err
	}
	defer cli.Close()
	if _, err := cli.ContainerRename(ctx, id, client.ContainerRenameOptions{NewName: newName}); err != nil {
		logger.ErrorCtx(ctx, "container rename failed", logger.String("id", id), logger.String("new_name", newName), logger.Err(err))
		return err
	}
	return nil
}

// BatchAction 批量操作类型。
type BatchAction string

const (
	BatchStart   BatchAction = "start"
	BatchStop    BatchAction = "stop"
	BatchRestart BatchAction = "restart"
	BatchRemove  BatchAction = "remove"
)

// Batch 批量操作容器，返回成功数与失败列表。
func (l *ContainerLogic) Batch(ctx context.Context, action BatchAction, ids []string) (int, []string, error) {
	logger.InfoCtx(ctx, "container batch", logger.String("action", string(action)), logger.Int("count", len(ids)))
	cli, err := l.newClient()
	if err != nil {
		logger.ErrorCtx(ctx, "container batch failed", logger.Err(err))
		return 0, nil, err
	}
	defer cli.Close()

	done := 0
	var failed []string
	for _, id := range ids {
		var err error
		switch action {
		case BatchStart:
			_, err = cli.ContainerStart(ctx, id, client.ContainerStartOptions{})
		case BatchStop:
			_, err = cli.ContainerStop(ctx, id, client.ContainerStopOptions{})
		case BatchRestart:
			_, err = cli.ContainerRestart(ctx, id, client.ContainerRestartOptions{})
		case BatchRemove:
			_, err = cli.ContainerRemove(ctx, id, client.ContainerRemoveOptions{Force: true, RemoveVolumes: true})
		default:
			return 0, nil, strconv.ErrSyntax
		}
		if err != nil {
			failed = append(failed, id)
			continue
		}
		done++
	}
	return done, failed, nil
}

// ProcessItem 容器内进程项（行数据，与 titles 对齐）。
type ProcessItem struct {
	Titles []string   `json:"titles"`
	Procs  [][]string `json:"procs"`
}

// Top 查看容器内进程列表。
func (l *ContainerLogic) Top(ctx context.Context, id string) (*ProcessItem, error) {
	logger.InfoCtx(ctx, "container top", logger.String("id", id))
	cli, err := l.newClient()
	if err != nil {
		logger.ErrorCtx(ctx, "container top failed", logger.String("id", id), logger.Err(err))
		return nil, err
	}
	defer cli.Close()

	res, err := cli.ContainerTop(ctx, id, client.ContainerTopOptions{})
	if err != nil {
		logger.ErrorCtx(ctx, "container top failed", logger.String("id", id), logger.Err(err))
		return nil, err
	}
	return &ProcessItem{Titles: res.Titles, Procs: res.Processes}, nil
}

// AttachResult 容器终端连接结果。
// 暴露底层 hijacked 连接：服务端（handler）用它做双向字节流桥接（WebSocket ↔ 容器 stdin/stdout）。
type AttachResult struct {
	Conn   net.Conn // hijacked 的 TCP 连接（含输入输出）
	Reader io.Reader
	Writer io.Writer
	Close  func() error
}

// Attach 连接容器终端（docker attach，交互式 TTY）。
// 返回一个可读写的双向流：写 = 发送到容器 stdin，读 = 接收容器 stdout/stderr。
// 由 handler 负责将流桥接到 WebSocket，并在结束时关闭 Conn。
func (l *ContainerLogic) Attach(ctx context.Context, id string) (*AttachResult, error) {
	logger.InfoCtx(ctx, "container attach", logger.String("id", id))
	cli, err := l.newClient()
	if err != nil {
		logger.ErrorCtx(ctx, "container attach failed", logger.String("id", id), logger.Err(err))
		return nil, err
	}

	res, err := cli.ContainerAttach(ctx, id, client.ContainerAttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		cli.Close()
		logger.ErrorCtx(ctx, "container attach failed", logger.String("id", id), logger.Err(err))
		return nil, err
	}

	// 返回值内部持有 cli 引用；关闭连接时同时关闭客户端。
	return &AttachResult{
		Conn:   res.Conn,
		Reader: res.Reader,
		Writer: res.Conn,
		Close: func() error {
			res.Close()
			cli.Close()
			return nil
		},
	}, nil
}

// ResizeContainerTTY 调整容器 TTY 尺寸（docker resize）。
func (l *ContainerLogic) ResizeContainerTTY(ctx context.Context, id string, rows, cols uint) error {
	logger.InfoCtx(ctx, "container resize", logger.String("id", id), logger.Int("rows", int(rows)), logger.Int("cols", int(cols)))
	cli, err := l.newClient()
	if err != nil {
		return err
	}
	defer cli.Close()
	_, err = cli.ContainerResize(ctx, id, client.ContainerResizeOptions{
		Height: rows,
		Width:  cols,
	})
	if err != nil {
		logger.ErrorCtx(ctx, "container resize failed", logger.String("id", id), logger.Err(err))
		return err
	}
	return nil
}
