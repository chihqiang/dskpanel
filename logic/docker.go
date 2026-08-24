package logic

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/client"
)

// DockerInfo 本机 Docker 检测结果。
type DockerInfo struct {
	Available bool          `json:"available"`
	Version   string        `json:"version"`
	Platform  string        `json:"platform"`
	OS        string        `json:"os"`
	Arch      string        `json:"arch"`
	CPU       int           `json:"cpu"`
	Memory    int64         `json:"memory"` // 字节
	Error     string        `json:"error,omitempty"`
	PingTime  time.Duration `json:"ping_time"` // 连接耗时
}

// DockerLogic 本机 Docker 检测逻辑。
type DockerLogic struct{}

// NewDockerLogic 创建本机 Docker 检测逻辑。
func NewDockerLogic() *DockerLogic {
	return &DockerLogic{}
}

// Detect 检测本机 Docker 环境（本地 socket），返回是否可用及信息。
// 通过本机 socket 连接 Docker Engine，检测到才启用 Docker 栏目。
func (l *DockerLogic) Detect(ctx context.Context) *DockerInfo {
	start := time.Now()
	cli, err := client.New(client.FromEnv)
	if err != nil {
		return &DockerInfo{Available: false, Error: err.Error()}
	}
	defer cli.Close()

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	ping, err := cli.Ping(pingCtx, client.PingOptions{})
	if err != nil {
		return &DockerInfo{Available: false, Error: err.Error()}
	}
	infoCtx, cancel2 := context.WithTimeout(ctx, 3*time.Second)
	defer cancel2()

	res, err := cli.Info(infoCtx, client.InfoOptions{})
	if err != nil {
		// Ping 成功但 Info 失败：仍视为可用。
		return &DockerInfo{
			Available: true,
			Version:   ping.APIVersion,
			PingTime:  time.Since(start),
		}
	}

	info := res.Info
	return &DockerInfo{
		Available: true,
		Version:   info.ServerVersion,
		Platform:  info.OperatingSystem,
		OS:        info.OSType,
		Arch:      info.Architecture,
		CPU:       info.NCPU,
		Memory:    info.MemTotal,
		PingTime:  time.Since(start),
	}
}

// ListContainers 列出容器（供后续 Docker 单机管理扩展）。
func (l *DockerLogic) ListContainers(ctx context.Context) ([]container.Summary, error) {
	cli, err := client.New(client.FromEnv)
	if err != nil {
		return nil, err
	}
	defer cli.Close()
	res, err := cli.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err != nil {
		return nil, err
	}
	return res.Items, nil
}

// ListImages 列出镜像（供后续 Docker 单机管理扩展）。
func (l *DockerLogic) ListImages(ctx context.Context) ([]image.Summary, error) {
	cli, err := client.New(client.FromEnv)
	if err != nil {
		return nil, err
	}
	defer cli.Close()
	res, err := cli.ImageList(ctx, client.ImageListOptions{})
	if err != nil {
		return nil, err
	}
	return res.Items, nil
}

// DockerStats 资源统计。
type DockerStats struct {
	Containers int            `json:"containers"`
	Images     int            `json:"images"`
	Networks   int            `json:"networks"`
	Volumes    int            `json:"volumes"`
	ByState    map[string]int `json:"by_state"` // 容器状态分布
}

// Stats 聚合本机 Docker 资源统计（容器/镜像/网络/卷数量 + 容器状态分布）。
func (l *DockerLogic) Stats(ctx context.Context) (*DockerStats, error) {
	cli, err := client.New(client.FromEnv)
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	s := &DockerStats{ByState: map[string]int{}}

	// 容器（含停止的，按状态统计）。
	if res, err := cli.ContainerList(ctx, client.ContainerListOptions{All: true}); err == nil {
		s.Containers = len(res.Items)
		for _, c := range res.Items {
			s.ByState[string(c.State)]++
		}
	}
	// 镜像。
	if res, err := cli.ImageList(ctx, client.ImageListOptions{}); err == nil {
		s.Images = len(res.Items)
	}
	// 网络。
	if res, err := cli.NetworkList(ctx, client.NetworkListOptions{}); err == nil {
		s.Networks = len(res.Items)
	}
	// 卷。
	if res, err := cli.VolumeList(ctx, client.VolumeListOptions{}); err == nil {
		s.Volumes = len(res.Items)
	}
	return s, nil
}

// ErrDockerNotAvailable Docker 不可用。
var ErrDockerNotAvailable = errors.New("docker not available")

// ensureClient 构建客户端并确认连接。
func (l *DockerLogic) ensureClient(ctx context.Context) (*client.Client, error) {
	cli, err := client.New(client.FromEnv)
	if err != nil {
		return nil, err
	}
	if _, err := cli.Ping(ctx, client.PingOptions{}); err != nil {
		cli.Close()
		return nil, fmt.Errorf("%w: %v", ErrDockerNotAvailable, err)
	}
	return cli, nil
}

// DockerVersion Docker 引擎版本信息。
type DockerVersion struct {
	PlatformName  string `json:"platform_name"`
	Version       string `json:"version"`
	APIVersion    string `json:"api_version"`
	MinAPIVersion string `json:"min_api_version"`
	Os            string `json:"os"`
	Arch          string `json:"arch"`
	Experimental  bool   `json:"experimental"`
	ClientVersion string `json:"client_version"` // 客户端 SDK 版本（本项目）
}

// Version 获取 Docker 引擎版本信息（docker version）。
func (l *DockerLogic) Version(ctx context.Context) (*DockerVersion, error) {
	cli, err := client.New(client.FromEnv)
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	res, err := cli.ServerVersion(ctx, client.ServerVersionOptions{})
	if err != nil {
		return nil, err
	}
	return &DockerVersion{
		PlatformName:  res.Platform.Name,
		Version:       res.Version,
		APIVersion:    res.APIVersion,
		MinAPIVersion: res.MinAPIVersion,
		Os:            res.Os,
		Arch:          res.Arch,
		Experimental:  res.Experimental,
		ClientVersion: client.MaxAPIVersion,
	}, nil
}

// DockerOverview 概览聚合：资源统计 + 引擎版本。
type DockerOverview struct {
	Stats   *DockerStats   `json:"stats"`
	Version *DockerVersion `json:"version"`
}

// Overview 合并获取资源统计与版本信息（单个请求一次返回，供概览页使用）。
func (l *DockerLogic) Overview(ctx context.Context) (*DockerOverview, error) {
	stats, err := l.Stats(ctx)
	if err != nil {
		return nil, err
	}
	version, err := l.Version(ctx)
	if err != nil {
		return nil, err
	}
	return &DockerOverview{Stats: stats, Version: version}, nil
}

// DockerSystemInfo 引擎完整信息（docker info 精选字段）。
type DockerSystemInfo struct {
	ID                string      `json:"id"`
	Name              string      `json:"name"`
	ServerVersion     string      `json:"server_version"`
	KernelVersion     string      `json:"kernel_version"`
	OperatingSystem   string      `json:"operating_system"`
	OSVersion         string      `json:"os_version"`
	OSType            string      `json:"os_type"`
	Architecture      string      `json:"architecture"`
	Driver            string      `json:"driver"`
	DriverStatus      [][2]string `json:"driver_status"`
	LoggingDriver     string      `json:"logging_driver"`
	CgroupDriver      string      `json:"cgroup_driver"`
	CgroupVersion     string      `json:"cgroup_version"`
	SecurityOptions   []string    `json:"security_options"`
	DefaultRuntime    string      `json:"default_runtime"`
	Runtimes          []string    `json:"runtimes"`
	NCPU              int         `json:"ncpu"`
	MemTotal          int64       `json:"mem_total"`
	DockerRootDir     string      `json:"docker_root_dir"`
	IndexServer       string      `json:"index_server"`
	Labels            []string    `json:"labels"`
	LiveRestore       bool        `json:"live_restore"`
	ExperimentalBuild bool        `json:"experimental_build"`
	Containers        int         `json:"containers"`
	ContainersRunning int         `json:"containers_running"`
	ContainersPaused  int         `json:"containers_paused"`
	ContainersStopped int         `json:"containers_stopped"`
	Images            int         `json:"images"`
	Debug             bool        `json:"debug"`
	NGoroutines       int         `json:"n_goroutines"`
	SystemTime        string      `json:"system_time"`
}

// Info 获取 Docker 引擎完整信息（docker info）。
func (l *DockerLogic) Info(ctx context.Context) (*DockerSystemInfo, error) {
	cli, err := client.New(client.FromEnv)
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	res, err := cli.Info(ctx, client.InfoOptions{})
	if err != nil {
		return nil, err
	}
	i := res.Info
	runtimes := make([]string, 0, len(i.Runtimes))
	for name := range i.Runtimes {
		runtimes = append(runtimes, name)
	}
	return &DockerSystemInfo{
		ID:                i.ID,
		Name:              i.Name,
		ServerVersion:     i.ServerVersion,
		KernelVersion:     i.KernelVersion,
		OperatingSystem:   i.OperatingSystem,
		OSVersion:         i.OSVersion,
		OSType:            i.OSType,
		Architecture:      i.Architecture,
		Driver:            i.Driver,
		DriverStatus:      i.DriverStatus,
		LoggingDriver:     i.LoggingDriver,
		CgroupDriver:      i.CgroupDriver,
		CgroupVersion:     i.CgroupVersion,
		SecurityOptions:   i.SecurityOptions,
		DefaultRuntime:    i.DefaultRuntime,
		Runtimes:          runtimes,
		NCPU:              i.NCPU,
		MemTotal:          i.MemTotal,
		DockerRootDir:     i.DockerRootDir,
		IndexServer:       i.IndexServerAddress,
		Labels:            i.Labels,
		LiveRestore:       i.LiveRestoreEnabled,
		ExperimentalBuild: i.ExperimentalBuild,
		Containers:        i.Containers,
		ContainersRunning: i.ContainersRunning,
		ContainersPaused:  i.ContainersPaused,
		ContainersStopped: i.ContainersStopped,
		Images:            i.Images,
		Debug:             i.Debug,
		NGoroutines:       i.NGoroutines,
		SystemTime:        i.SystemTime,
	}, nil
}

// PruneCategory 清理分类结果。
type PruneCategory struct {
	Deleted   int    `json:"deleted"`
	Reclaimed int64  `json:"reclaimed"` // 回收字节数
	Error     string `json:"error,omitempty"`
}

// DockerPruneResult 一键清理汇总。
type DockerPruneResult struct {
	Containers PruneCategory `json:"containers"`
	Images     PruneCategory `json:"images"`
	Networks   PruneCategory `json:"networks"`
	Volumes    PruneCategory `json:"volumes"`
	BuildCache PruneCategory `json:"build_cache"`
	Total      PruneCategory `json:"total"`
}

// PruneAll 一键清理未使用资源（停止的容器/悬空镜像/未用网络/匿名卷/构建缓存）。
// 每个分类独立容错：单个失败不影响其他。
func (l *DockerLogic) PruneAll(ctx context.Context) *DockerPruneResult {
	cli, err := client.New(client.FromEnv)
	if err != nil {
		return nil
	}
	defer cli.Close()

	result := &DockerPruneResult{}

	// 停止的容器。
	if res, err := cli.ContainerPrune(ctx, client.ContainerPruneOptions{}); err == nil {
		result.Containers.Deleted = len(res.Report.ContainersDeleted)
		result.Containers.Reclaimed = int64(res.Report.SpaceReclaimed)
	} else {
		result.Containers.Error = err.Error()
	}

	// 悬空镜像。
	if res, err := cli.ImagePrune(ctx, client.ImagePruneOptions{Filters: client.Filters{}.Add("dangling", "true")}); err == nil {
		result.Images.Deleted = len(res.Report.ImagesDeleted)
		result.Images.Reclaimed = int64(res.Report.SpaceReclaimed)
	} else {
		result.Images.Error = err.Error()
	}

	// 未使用网络。
	if res, err := cli.NetworkPrune(ctx, client.NetworkPruneOptions{}); err == nil {
		result.Networks.Deleted = len(res.Report.NetworksDeleted)
	} else {
		result.Networks.Error = err.Error()
	}

	// 未使用卷（匿名卷）。
	if res, err := cli.VolumePrune(ctx, client.VolumePruneOptions{}); err == nil {
		result.Volumes.Deleted = len(res.Report.VolumesDeleted)
		result.Volumes.Reclaimed = int64(res.Report.SpaceReclaimed)
	} else {
		result.Volumes.Error = err.Error()
	}

	// 构建缓存。
	if res, err := cli.BuildCachePrune(ctx, client.BuildCachePruneOptions{}); err == nil {
		result.BuildCache.Deleted = len(res.Report.CachesDeleted)
		result.BuildCache.Reclaimed = int64(res.Report.SpaceReclaimed)
	} else {
		result.BuildCache.Error = err.Error()
	}

	result.Total.Deleted = result.Containers.Deleted + result.Images.Deleted +
		result.Networks.Deleted + result.Volumes.Deleted + result.BuildCache.Deleted
	result.Total.Reclaimed = result.Containers.Reclaimed + result.Images.Reclaimed +
		result.Volumes.Reclaimed + result.BuildCache.Reclaimed
	return result
}

// DockerEvent Docker 系统事件（daemon events）。
type DockerEvent struct {
	Type      string            `json:"type"`
	Action    string            `json:"action"`
	ActorID   string            `json:"actor_id"`
	ActorAttr map[string]string `json:"actor_attr,omitempty"`
	Scope     string            `json:"scope,omitempty"`
	Time      int64             `json:"time"`
}

// StreamEvents 订阅 Docker 系统事件流（阻塞），逐条回调 onEvent。
// ctx 取消即停止；连接异常返回错误。
func (l *DockerLogic) StreamEvents(ctx context.Context, onEvent func(DockerEvent)) error {
	cli, err := client.New(client.FromEnv)
	if err != nil {
		return err
	}
	defer cli.Close()

	result := cli.Events(ctx, client.EventsListOptions{})
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err, ok := <-result.Err:
			if ok && err != nil {
				return err
			}
			return nil
		case msg, ok := <-result.Messages:
			if !ok {
				return nil
			}
			onEvent(DockerEvent{
				Type:      string(msg.Type),
				Action:    string(msg.Action),
				ActorID:   msg.Actor.ID,
				ActorAttr: msg.Actor.Attributes,
				Scope:     msg.Scope,
				Time:      msg.Time,
			})
		}
	}
}
