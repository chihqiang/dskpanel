package logic

import (
	"context"
	"fmt"
	"net/netip"
	"time"

	"github.com/chihqiang/infra-go/logger"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

// NetworkItem 网络列表项。
type NetworkItem struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Driver     string            `json:"driver"`
	Scope      string            `json:"scope"`
	Internal   bool              `json:"internal"`
	Attachable bool              `json:"attachable"`
	IPAMDriver string            `json:"ipam_driver"`
	Labels     map[string]string `json:"labels,omitempty"`
	Created    time.Time         `json:"created"`
}

// NetworkLogic 网络管理逻辑。
type NetworkLogic struct{}

// NewNetworkLogic 创建网络管理逻辑。
func NewNetworkLogic() *NetworkLogic {
	return &NetworkLogic{}
}

// newClient 创建本机 Docker 客户端。
func (l *NetworkLogic) newClient() (*client.Client, error) {
	return client.New(client.FromEnv)
}

// List 列出网络。
func (l *NetworkLogic) List(ctx context.Context) ([]NetworkItem, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	res, err := cli.NetworkList(ctx, client.NetworkListOptions{})
	if err != nil {
		return nil, err
	}

	items := make([]NetworkItem, 0, len(res.Items))
	for _, n := range res.Items {
		ipamDriver := ""
		if len(n.IPAM.Driver) > 0 {
			ipamDriver = n.IPAM.Driver
		}
		items = append(items, NetworkItem{
			ID:         n.ID,
			Name:       n.Name,
			Driver:     n.Driver,
			Scope:      n.Scope,
			Internal:   n.Internal,
			Attachable: n.Attachable,
			IPAMDriver: ipamDriver,
			Labels:     n.Labels,
			Created:    n.Created,
		})
	}
	return items, nil
}

// CreateNetworkRequest 创建网络请求（含高级参数）。
type CreateNetworkRequest struct {
	Name       string            `json:"name" binding:"required"`
	Driver     string            `json:"driver,default=bridge"`
	Subnet     string            `json:"subnet"`   // 如 172.20.0.0/16
	Gateway    string            `json:"gateway"`  // 如 172.20.0.1
	IPRange    string            `json:"ip_range"` // 如 172.20.0.0/24
	Internal   bool              `json:"internal"` // 仅内部网络
	EnableIPv6 bool              `json:"enable_ipv6"`
	Labels     map[string]string `json:"labels"`
	DriverOpts map[string]string `json:"driver_opts"` // 驱动选项
	IPAMDriver string            `json:"ipam_driver"` // IPAM 驱动，默认 default
}

// Create 创建网络。
func (l *NetworkLogic) Create(ctx context.Context, req *CreateNetworkRequest) (string, error) {
	logger.InfoCtx(ctx, "network create",
		logger.String("name", req.Name),
		logger.String("driver", req.Driver),
		logger.String("subnet", req.Subnet))
	cli, err := l.newClient()
	if err != nil {
		logger.ErrorCtx(ctx, "network create failed", logger.String("name", req.Name), logger.Err(err))
		return "", err
	}
	defer cli.Close()

	opts := client.NetworkCreateOptions{
		Driver:     req.Driver,
		Internal:   req.Internal,
		Attachable: true,
		Options:    req.DriverOpts,
		Labels:     req.Labels,
		EnableIPv6: &req.EnableIPv6,
	}
	// 配置 IPAM 子网。
	if req.Subnet != "" || req.Gateway != "" || req.IPRange != "" || req.IPAMDriver != "" {
		cfg := network.IPAMConfig{}
		if req.Subnet != "" {
			p, err := netip.ParsePrefix(req.Subnet)
			if err != nil {
				return "", fmt.Errorf("无效的子网地址: %s", req.Subnet)
			}
			cfg.Subnet = p
		}
		if req.Gateway != "" {
			a, err := netip.ParseAddr(req.Gateway)
			if err != nil {
				return "", fmt.Errorf("无效的网关地址: %s", req.Gateway)
			}
			cfg.Gateway = a
		}
		if req.IPRange != "" {
			p, err := netip.ParsePrefix(req.IPRange)
			if err != nil {
				return "", fmt.Errorf("无效的 IP 范围: %s", req.IPRange)
			}
			cfg.IPRange = p
		}
		ipamDriver := req.IPAMDriver
		if ipamDriver == "" {
			ipamDriver = "default"
		}
		opts.IPAM = &network.IPAM{
			Driver: ipamDriver,
			Config: []network.IPAMConfig{cfg},
		}
	}

	res, err := cli.NetworkCreate(ctx, req.Name, opts)
	if err != nil {
		logger.ErrorCtx(ctx, "network create failed", logger.String("name", req.Name), logger.Err(err))
		return "", err
	}
	logger.InfoCtx(ctx, "network created", logger.String("id", res.ID), logger.String("name", req.Name))
	return res.ID, nil
}

// NetworkDetail 网络详情。
type NetworkDetail struct {
	ID         string              `json:"id"`
	Name       string              `json:"name"`
	Driver     string              `json:"driver"`
	Scope      string              `json:"scope"`
	Internal   bool                `json:"internal"`
	Attachable bool                `json:"attachable"`
	EnableIPv6 bool                `json:"enable_ipv6"`
	IPAM       []NetworkIPAMConfig `json:"ipam,omitempty"`
	Containers []NetworkContainer  `json:"containers,omitempty"`
	Labels     map[string]string   `json:"labels,omitempty"`
	Created    time.Time           `json:"created"`
}

// NetworkIPAMConfig IPAM 子网配置。
type NetworkIPAMConfig struct {
	Subnet  string `json:"subnet,omitempty"`
	Gateway string `json:"gateway,omitempty"`
	IPRange string `json:"ip_range,omitempty"`
}

// NetworkContainer 连接到网络的容器。
type NetworkContainer struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	MacAddress  string `json:"mac_address,omitempty"`
	IPv4Address string `json:"ipv4_address,omitempty"`
	IPv6Address string `json:"ipv6_address,omitempty"`
}

// Inspect 查看网络详情。
func (l *NetworkLogic) Inspect(ctx context.Context, id string) (*NetworkDetail, error) {
	logger.InfoCtx(ctx, "network inspect", logger.String("id", id))
	cli, err := l.newClient()
	if err != nil {
		logger.ErrorCtx(ctx, "network inspect failed", logger.String("id", id), logger.Err(err))
		return nil, err
	}
	defer cli.Close()

	res, err := cli.NetworkInspect(ctx, id, client.NetworkInspectOptions{})
	if err != nil {
		logger.ErrorCtx(ctx, "network inspect failed", logger.String("id", id), logger.Err(err))
		return nil, err
	}

	n := res.Network
	detail := &NetworkDetail{
		ID:         n.ID,
		Name:       n.Name,
		Driver:     n.Driver,
		Scope:      n.Scope,
		Internal:   n.Internal,
		Attachable: n.Attachable,
		EnableIPv6: n.EnableIPv6,
		Labels:     n.Labels,
		Created:    n.Created,
	}
	if len(n.IPAM.Config) > 0 {
		detail.IPAM = make([]NetworkIPAMConfig, 0, len(n.IPAM.Config))
		for _, c := range n.IPAM.Config {
			subnet, gateway, ipRange := "", "", ""
			if c.Subnet.IsValid() {
				subnet = c.Subnet.String()
			}
			if c.Gateway.IsValid() {
				gateway = c.Gateway.String()
			}
			if c.IPRange.IsValid() {
				ipRange = c.IPRange.String()
			}
			detail.IPAM = append(detail.IPAM, NetworkIPAMConfig{
				Subnet:  subnet,
				Gateway: gateway,
				IPRange: ipRange,
			})
		}
	}
	if len(n.Containers) > 0 {
		detail.Containers = make([]NetworkContainer, 0, len(n.Containers))
		for cid, ec := range n.Containers {
			ipv4, ipv6 := "", ""
			if ec.IPv4Address.IsValid() {
				ipv4 = ec.IPv4Address.String()
			}
			if ec.IPv6Address.IsValid() {
				ipv6 = ec.IPv6Address.String()
			}
			detail.Containers = append(detail.Containers, NetworkContainer{
				ID:          cid,
				Name:        ec.Name,
				MacAddress:  ec.MacAddress.String(),
				IPv4Address: ipv4,
				IPv6Address: ipv6,
			})
		}
	}
	return detail, nil
}

// Remove 删除网络。
func (l *NetworkLogic) Remove(ctx context.Context, id string) error {
	logger.InfoCtx(ctx, "network remove", logger.String("id", id))
	cli, err := l.newClient()
	if err != nil {
		logger.ErrorCtx(ctx, "network remove failed", logger.String("id", id), logger.Err(err))
		return err
	}
	defer cli.Close()
	_, err = cli.NetworkRemove(ctx, id, client.NetworkRemoveOptions{})
	if err != nil {
		logger.ErrorCtx(ctx, "network remove failed", logger.String("id", id), logger.Err(err))
	}
	return err
}

// ConnectContainer 将容器连接到网络（可指定固定 IPv4）。
func (l *NetworkLogic) ConnectContainer(ctx context.Context, networkID, containerID, ipv4 string) error {
	logger.InfoCtx(ctx, "network connect", logger.String("network", networkID), logger.String("container", containerID))
	cli, err := l.newClient()
	if err != nil {
		logger.ErrorCtx(ctx, "network connect failed", logger.Err(err))
		return err
	}
	defer cli.Close()

	opts := client.NetworkConnectOptions{Container: containerID}
	if ipv4 != "" {
		addr, err := netip.ParseAddr(ipv4)
		if err != nil {
			logger.ErrorCtx(ctx, "network connect invalid ipv4", logger.String("ipv4", ipv4), logger.Err(err))
			return fmt.Errorf("无效的 IPv4 地址: %s", ipv4)
		}
		opts.EndpointConfig = &network.EndpointSettings{IPAMConfig: &network.EndpointIPAMConfig{IPv4Address: addr}}
	}
	if _, err := cli.NetworkConnect(ctx, networkID, opts); err != nil {
		logger.ErrorCtx(ctx, "network connect failed", logger.String("network", networkID), logger.String("container", containerID), logger.Err(err))
		return err
	}
	return nil
}

// DisconnectContainer 将容器从网络断开。
func (l *NetworkLogic) DisconnectContainer(ctx context.Context, networkID, containerID string, force bool) error {
	logger.InfoCtx(ctx, "network disconnect", logger.String("network", networkID), logger.String("container", containerID))
	cli, err := l.newClient()
	if err != nil {
		logger.ErrorCtx(ctx, "network disconnect failed", logger.Err(err))
		return err
	}
	defer cli.Close()
	if _, err := cli.NetworkDisconnect(ctx, networkID, client.NetworkDisconnectOptions{Container: containerID, Force: force}); err != nil {
		logger.ErrorCtx(ctx, "network disconnect failed", logger.String("network", networkID), logger.String("container", containerID), logger.Err(err))
		return err
	}
	return nil
}

// Prune 清理未使用网络，返回被清理的网络名列表。
func (l *NetworkLogic) Prune(ctx context.Context) ([]string, error) {
	logger.InfoCtx(ctx, "network prune")
	cli, err := l.newClient()
	if err != nil {
		logger.ErrorCtx(ctx, "network prune failed", logger.Err(err))
		return nil, err
	}
	defer cli.Close()

	res, err := cli.NetworkPrune(ctx, client.NetworkPruneOptions{})
	if err != nil {
		logger.ErrorCtx(ctx, "network prune failed", logger.Err(err))
		return nil, err
	}
	return res.Report.NetworksDeleted, nil
}
