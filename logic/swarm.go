package logic

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"time"

	"github.com/chihqiang/infra-go/logger"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"

	"chihqiang/dskpanel/config"
)

// ErrSwarmNotActive 目标引擎未启用 swarm mode。
var ErrSwarmNotActive = errors.New("this node is not a swarm manager")

// SwarmLogic Swarm 集群管理逻辑。
// 连接目标由 config.Swarm 决定：Endpoint 为空连本机 Docker，否则连远程 manager。
type SwarmLogic struct {
	swarm config.Swarm
}

// NewSwarmLogic 创建 Swarm 管理逻辑。
func NewSwarmLogic(swarmCfg config.Swarm) *SwarmLogic {
	return &SwarmLogic{swarm: swarmCfg}
}

// newClient 创建 Swarm 客户端。
// 返回 close 函数用于关闭底层连接；调用方务必 defer close()。
func (l *SwarmLogic) newClient(ctx context.Context) (*client.Client, func(), error) {
	if l.swarm.Endpoint == "" {
		// 本机 Docker（要求已启用 swarm mode）。
		cli, err := client.New(client.FromEnv)
		if err != nil {
			return nil, nil, err
		}
		return cli, func() { _ = cli.Close() }, nil
	}

	// 远程 Swarm manager。
	var cli *client.Client
	var err error
	if l.swarm.Cert != "" && l.swarm.Key != "" {
		tlsCfg, err := parseSwarmCredential(swarmCredential{
			CA:   l.swarm.CA,
			Cert: l.swarm.Cert,
			Key:  l.swarm.Key,
		})
		if err != nil {
			return nil, nil, err
		}
		tr := &http.Transport{TLSClientConfig: tlsCfg}
		cli, err = client.New(
			client.WithHost(l.swarm.Endpoint),
			client.WithHTTPClient(&http.Client{Transport: tr}),
			client.WithAPIVersionNegotiation(),
		)
	} else {
		cli, err = client.New(
			client.WithHost(l.swarm.Endpoint),
			client.WithAPIVersionNegotiation(),
		)
	}
	if err != nil {
		return nil, nil, err
	}
	return cli, func() { _ = cli.Close() }, nil
}

// swarmCredential TLS 凭据（config.Swarm 的 ca/cert/key PEM）。
type swarmCredential struct {
	CA   string `json:"ca"`
	Cert string `json:"cert"`
	Key  string `json:"key"`
}

// parseSwarmCredential 解析 TLS 凭据（ca/cert/key PEM）为 TLS 配置。
func parseSwarmCredential(c swarmCredential) (*tls.Config, error) {
	if c.Cert == "" || c.Key == "" {
		return nil, errors.New("凭据缺少 cert/key（client 证书与私钥）")
	}
	cert, err := tls.X509KeyPair([]byte(c.Cert), []byte(c.Key))
	if err != nil {
		return nil, fmt.Errorf("解析 client 证书失败: %w", err)
	}
	cfg := &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
	if c.CA != "" {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(c.CA)) {
			return nil, errors.New("解析 CA 证书失败")
		}
		cfg.RootCAs = pool
	}
	return cfg, nil
}

// SwarmStatus 集群状态摘要。
type SwarmStatus struct {
	Available bool   `json:"available"`
	Error     string `json:"error,omitempty"`
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Managers  int    `json:"managers,omitempty"`
	Nodes     int    `json:"nodes,omitempty"`
	Version   string `json:"version,omitempty"`
}

// Info 获取集群 swarm 状态。
func (l *SwarmLogic) Info(ctx context.Context) (*SwarmStatus, error) {
	cli, close, err := l.newClient(ctx)
	if err != nil {
		return nil, err
	}
	defer close()

	infoCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	res, err := cli.Info(infoCtx, client.InfoOptions{})
	if err != nil {
		return nil, err
	}
	info := res.Info
	if info.Swarm.LocalNodeState != swarm.LocalNodeStateActive {
		return nil, ErrSwarmNotActive
	}
	st := &SwarmStatus{
		Available: true,
		ID:        info.Swarm.Cluster.ID,
		Name:      info.Swarm.Cluster.Spec.Annotations.Name,
		Managers:  int(info.Swarm.Managers),
		Nodes:     int(info.Swarm.Nodes),
		Version:   info.ServerVersion,
	}
	return st, nil
}

// SwarmOverview 集群概览。
type SwarmOverview struct {
	Status   *SwarmStatus       `json:"status"`
	Nodes    []SwarmNodeItem    `json:"nodes"`
	Services []SwarmServiceItem `json:"services"`
	Tasks    []SwarmTaskItem    `json:"tasks"`
	Secrets  int                `json:"secrets"`
	Configs  int                `json:"configs"`
	Summary  SwarmSummary       `json:"summary"`
}

// SwarmSummary 概览统计。
type SwarmSummary struct {
	NodeCount      int            `json:"node_count"`
	ManagerCount   int            `json:"manager_count"`
	WorkerCount    int            `json:"worker_count"`
	NodesByState   map[string]int `json:"nodes_by_state"`
	ServiceCount   int            `json:"service_count"`
	ServiceRunning int            `json:"service_running"`
	TaskCount      int            `json:"task_count"`
	TasksByState   map[string]int `json:"tasks_by_state"`
	SecretsCount   int            `json:"secrets_count"`
	ConfigsCount   int            `json:"configs_count"`
}

// Overview 集群概览：状态 + 节点/服务/任务/Secret/Config 汇总。
func (l *SwarmLogic) Overview(ctx context.Context) (*SwarmOverview, error) {
	status, err := l.Info(ctx)
	if err != nil {
		return nil, err
	}
	cli, close, err := l.newClient(ctx)
	if err != nil {
		return nil, err
	}
	defer close()

	ov := &SwarmOverview{
		Status: status,
		Summary: SwarmSummary{
			NodesByState: map[string]int{},
			TasksByState: map[string]int{},
		},
	}

	// 节点。
	if res, err := cli.NodeList(ctx, client.NodeListOptions{}); err == nil {
		for _, n := range res.Items {
			ov.Nodes = append(ov.Nodes, toSwarmNodeItem(&n))
		}
	}
	// 服务。
	if res, err := cli.ServiceList(ctx, client.ServiceListOptions{}); err == nil {
		for _, s := range res.Items {
			ov.Services = append(ov.Services, toSwarmServiceItem(&s))
		}
	}
	// 任务。
	if res, err := cli.TaskList(ctx, client.TaskListOptions{}); err == nil {
		for _, t := range res.Items {
			ov.Tasks = append(ov.Tasks, toSwarmTaskItem(&t))
		}
	}
	// Secret / Config。
	if res, err := cli.SecretList(ctx, client.SecretListOptions{}); err == nil {
		ov.Secrets = len(res.Items)
	}
	if res, err := cli.ConfigList(ctx, client.ConfigListOptions{}); err == nil {
		ov.Configs = len(res.Items)
	}

	s := &ov.Summary
	s.NodeCount = len(ov.Nodes)
	s.ServiceCount = len(ov.Services)
	s.TaskCount = len(ov.Tasks)
	s.SecretsCount = ov.Secrets
	s.ConfigsCount = ov.Configs
	for _, n := range ov.Nodes {
		s.NodesByState[string(n.State)]++
		if n.Role == "manager" {
			s.ManagerCount++
		} else {
			s.WorkerCount++
		}
	}
	for _, svc := range ov.Services {
		if svc.State == "running" {
			s.ServiceRunning++
		}
	}
	for _, t := range ov.Tasks {
		s.TasksByState[t.State]++
	}
	return ov, nil
}

// SwarmNodeItem 节点列表项。
type SwarmNodeItem struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Role         string `json:"role"`         // manager / worker
	State        string `json:"state"`        // ready / down / disconnected / unknown
	Availability string `json:"availability"` // active / pause / drain
	Status       string `json:"status"`       // leader / reachable / unreachable / ""
	Addr         string `json:"addr"`
	Version      string `json:"version"`
	Labels       int    `json:"labels"`
	EngineErr    string `json:"engine_err,omitempty"`
	UpdatedAt    string `json:"updated_at"`
}

// toSwarmNodeItem swarm.Node → 列表项。
func toSwarmNodeItem(n *swarm.Node) SwarmNodeItem {
	item := SwarmNodeItem{
		ID:           n.ID,
		Name:         n.Spec.Annotations.Name,
		Role:         string(n.Spec.Role),
		Availability: string(n.Spec.Availability),
		State:        string(n.Status.State),
		Addr:         n.Status.Addr,
		Labels:       len(n.Spec.Annotations.Labels),
		UpdatedAt:    n.Meta.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
	if item.Name == "" {
		item.Name = n.Description.Hostname
	}
	if n.Description.Engine.EngineVersion != "" {
		item.Version = n.Description.Engine.EngineVersion
	}
	if n.ManagerStatus != nil {
		item.Status = string(n.ManagerStatus.Reachability)
		if n.ManagerStatus.Leader {
			item.Status = "leader"
		}
	}
	return item
}

// ListNodes 节点列表。
func (l *SwarmLogic) ListNodes(ctx context.Context) ([]SwarmNodeItem, error) {
	cli, close, err := l.newClient(ctx)
	if err != nil {
		return nil, err
	}
	defer close()

	res, err := cli.NodeList(ctx, client.NodeListOptions{})
	if err != nil {
		return nil, err
	}
	items := make([]SwarmNodeItem, 0, len(res.Items))
	for i := range res.Items {
		items = append(items, toSwarmNodeItem(&res.Items[i]))
	}
	return items, nil
}

// InspectNode 节点详情（原始 inspect JSON）。
func (l *SwarmLogic) InspectNode(ctx context.Context, id string) (json.RawMessage, error) {
	cli, close, err := l.newClient(ctx)
	if err != nil {
		return nil, err
	}
	defer close()

	res, err := cli.NodeInspect(ctx, id, client.NodeInspectOptions{})
	if err != nil {
		return nil, err
	}
	return res.Raw, nil
}

// SetNodeAvailability 切换节点可用性（active / pause / drain）。
func (l *SwarmLogic) SetNodeAvailability(ctx context.Context, id, availability string) error {
	cli, close, err := l.newClient(ctx)
	if err != nil {
		return err
	}
	defer close()

	res, err := cli.NodeInspect(ctx, id, client.NodeInspectOptions{})
	if err != nil {
		return err
	}
	node := res.Node
	avail := swarm.NodeAvailability(availability)
	switch avail {
	case swarm.NodeAvailabilityActive, swarm.NodeAvailabilityPause, swarm.NodeAvailabilityDrain:
	default:
		return fmt.Errorf("invalid availability: %s", availability)
	}
	spec := node.Spec
	spec.Availability = avail
	_, err = cli.NodeUpdate(ctx, id, client.NodeUpdateOptions{
		Version: node.Meta.Version,
		Spec:    spec,
	})
	if err != nil {
		logger.Errorf("swarm node update availability failed: %v", err)
	}
	return err
}

// RemoveNode 删除（移除）节点。
func (l *SwarmLogic) RemoveNode(ctx context.Context, id string, force bool) error {
	cli, close, err := l.newClient(ctx)
	if err != nil {
		return err
	}
	defer close()
	_, err = cli.NodeRemove(ctx, id, client.NodeRemoveOptions{Force: force})
	return err
}

// SwarmNetworkItem 网络列表项（供服务表单选择）。
type SwarmNetworkItem struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Scope      string `json:"scope"`
	Driver     string `json:"driver"`
	Attachable bool   `json:"attachable"`
}

// ListNetworks 列出集群网络。
func (l *SwarmLogic) ListNetworks(ctx context.Context) ([]SwarmNetworkItem, error) {
	cli, close, err := l.newClient(ctx)
	if err != nil {
		return nil, err
	}
	defer close()

	res, err := cli.NetworkList(ctx, client.NetworkListOptions{})
	if err != nil {
		return nil, err
	}
	items := make([]SwarmNetworkItem, 0, len(res.Items))
	for _, n := range res.Items {
		items = append(items, SwarmNetworkItem{
			ID:         n.ID,
			Name:       n.Name,
			Scope:      n.Scope,
			Driver:     n.Driver,
			Attachable: n.Attachable,
		})
	}
	return items, nil
}

// JoinToken 集群 join token（供添加节点）。
type JoinToken struct {
	Worker  string `json:"worker"`
	Manager string `json:"manager"`
	Addr    string `json:"addr"`
}

// GetJoinTokens 获取 worker / manager 加入令牌。
func (l *SwarmLogic) GetJoinTokens(ctx context.Context) (*JoinToken, error) {
	cli, close, err := l.newClient(ctx)
	if err != nil {
		return nil, err
	}
	defer close()

	res, err := cli.SwarmInspect(ctx, client.SwarmInspectOptions{})
	if err != nil {
		return nil, err
	}
	sw := res.Swarm
	addr := ""
	if len(sw.JoinTokens.Worker) == 0 && len(sw.JoinTokens.Manager) == 0 {
		return nil, fmt.Errorf("集群未启用或无法获取加入令牌")
	}
	return &JoinToken{
		Worker:  sw.JoinTokens.Worker,
		Manager: sw.JoinTokens.Manager,
		Addr:    addr,
	}, nil
}

// SwarmNetworkCreateRequest 创建网络请求。
type SwarmNetworkCreateRequest struct {
	Name       string            `json:"name"`
	Driver     string            `json:"driver,omitempty"` // overlay / bridge 等
	Subnet     string            `json:"subnet,omitempty"` // 如 10.0.1.0/24
	Gateway    string            `json:"gateway,omitempty"`
	Attachable bool              `json:"attachable,omitempty"`
	Internal   bool              `json:"internal,omitempty"`
	EnableIPv6 bool              `json:"enable_ipv6,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
}

// CreateNetwork 创建 Swarm 网络。
func (l *SwarmLogic) CreateNetwork(ctx context.Context, req SwarmNetworkCreateRequest) error {
	if req.Name == "" {
		return fmt.Errorf("网络名称不能为空")
	}
	cli, close, err := l.newClient(ctx)
	if err != nil {
		return err
	}
	defer close()

	driver := req.Driver
	if driver == "" {
		driver = "overlay"
	}
	var ipam *network.IPAM
	if req.Subnet != "" || req.Gateway != "" {
		cfg := network.IPAMConfig{}
		if req.Subnet != "" {
			p, err := netip.ParsePrefix(req.Subnet)
			if err != nil {
				return fmt.Errorf("无效子网 %q: %w", req.Subnet, err)
			}
			cfg.Subnet = p
		}
		if req.Gateway != "" {
			g, err := netip.ParseAddr(req.Gateway)
			if err != nil {
				return fmt.Errorf("无效网关 %q: %w", req.Gateway, err)
			}
			cfg.Gateway = g
		}
		ipam = &network.IPAM{Driver: "default", Config: []network.IPAMConfig{cfg}}
	}

	enableIPv6 := req.EnableIPv6
	_, err = cli.NetworkCreate(ctx, req.Name, client.NetworkCreateOptions{
		Driver:     driver,
		IPAM:       ipam,
		Internal:   req.Internal,
		Attachable: req.Attachable,
		EnableIPv6: &enableIPv6,
		Labels:     req.Labels,
		Options:    map[string]string{},
	})
	return err
}

// InspectNetwork 网络详情（原始 inspect）。
func (l *SwarmLogic) InspectNetwork(ctx context.Context, id string) (json.RawMessage, error) {
	cli, close, err := l.newClient(ctx)
	if err != nil {
		return nil, err
	}
	defer close()

	res, err := cli.NetworkInspect(ctx, id, client.NetworkInspectOptions{})
	if err != nil {
		return nil, err
	}
	return res.Raw, nil
}

// RemoveNetwork 删除网络。
func (l *SwarmLogic) RemoveNetwork(ctx context.Context, id string) error {
	cli, close, err := l.newClient(ctx)
	if err != nil {
		return err
	}
	defer close()
	_, err = cli.NetworkRemove(ctx, id, client.NetworkRemoveOptions{})
	return err
}

// SwarmImageItem 集群镜像列表项（供服务创建表单选择）。
type SwarmImageItem struct {
	ID       string   `json:"id"`
	RepoTags []string `json:"repo_tags"`
}

// ListImages 列出集群镜像（含 tag）。
func (l *SwarmLogic) ListImages(ctx context.Context) ([]SwarmImageItem, error) {
	cli, close, err := l.newClient(ctx)
	if err != nil {
		return nil, err
	}
	defer close()

	res, err := cli.ImageList(ctx, client.ImageListOptions{})
	if err != nil {
		return nil, err
	}
	items := make([]SwarmImageItem, 0, len(res.Items))
	for _, img := range res.Items {
		items = append(items, SwarmImageItem{
			ID:       img.ID,
			RepoTags: img.RepoTags,
		})
	}
	return items, nil
}
