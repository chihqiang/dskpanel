package logic

import (
	"context"

	"github.com/chihqiang/infra-go/logger"
	"github.com/moby/moby/client"
)

// VolumeItem 卷列表项。
type VolumeItem struct {
	Name       string            `json:"name"`
	Driver     string            `json:"driver"`
	Mountpoint string            `json:"mountpoint"`
	Scope      string            `json:"scope"`
	Labels     map[string]string `json:"labels,omitempty"`
	CreatedAt  string            `json:"created_at"`
	Size       int64             `json:"size"`
	Used       bool              `json:"used"`
}

// VolumeLogic 卷管理逻辑。
type VolumeLogic struct{}

// NewVolumeLogic 创建卷管理逻辑。
func NewVolumeLogic() *VolumeLogic {
	return &VolumeLogic{}
}

// newClient 创建本机 Docker 客户端。
func (l *VolumeLogic) newClient() (*client.Client, error) {
	return client.New(client.FromEnv)
}

// List 列出卷。
func (l *VolumeLogic) List(ctx context.Context) ([]VolumeItem, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	res, err := cli.VolumeList(ctx, client.VolumeListOptions{})
	if err != nil {
		return nil, err
	}

	items := make([]VolumeItem, 0, len(res.Items))
	for _, v := range res.Items {
		item := VolumeItem{
			Name:       v.Name,
			Driver:     v.Driver,
			Mountpoint: v.Mountpoint,
			Scope:      v.Scope,
			Labels:     v.Labels,
			CreatedAt:  v.CreatedAt,
		}
		// UsageData 可能为 nil（如未统计用量时）。
		if v.UsageData != nil {
			item.Size = v.UsageData.Size
			item.Used = v.UsageData.RefCount > 0
		}
		items = append(items, item)
	}
	return items, nil
}

// Create 创建卷。
func (l *VolumeLogic) Create(ctx context.Context, name, driver string) error {
	logger.InfoCtx(ctx, "volume create", logger.String("name", name), logger.String("driver", driver))
	cli, err := l.newClient()
	if err != nil {
		logger.ErrorCtx(ctx, "volume create failed", logger.String("name", name), logger.Err(err))
		return err
	}
	defer cli.Close()
	_, err = cli.VolumeCreate(ctx, client.VolumeCreateOptions{
		Name:   name,
		Driver: driver,
	})
	if err != nil {
		logger.ErrorCtx(ctx, "volume create failed", logger.String("name", name), logger.Err(err))
	}
	return err
}

// VolumeDetail 卷详情。
type VolumeDetail struct {
	Name       string            `json:"name"`
	Driver     string            `json:"driver"`
	Mountpoint string            `json:"mountpoint"`
	Scope      string            `json:"scope"`
	CreatedAt  string            `json:"created_at"`
	Labels     map[string]string `json:"labels,omitempty"`
	Options    map[string]string `json:"options,omitempty"`
	Size       int64             `json:"size"`
	RefCount   int64             `json:"ref_count"`
	// 使用该卷的容器（挂载点）。
	Containers []VolumeContainer `json:"containers,omitempty"`
}

// VolumeContainer 使用卷的容器。
type VolumeContainer struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"`
	Dest  string `json:"dest"` // 容器内挂载路径
}

// Inspect 查看卷详情。
func (l *VolumeLogic) Inspect(ctx context.Context, name string) (*VolumeDetail, error) {
	logger.InfoCtx(ctx, "volume inspect", logger.String("name", name))
	cli, err := l.newClient()
	if err != nil {
		logger.ErrorCtx(ctx, "volume inspect failed", logger.String("name", name), logger.Err(err))
		return nil, err
	}
	defer cli.Close()

	res, err := cli.VolumeInspect(ctx, name, client.VolumeInspectOptions{})
	if err != nil {
		logger.ErrorCtx(ctx, "volume inspect failed", logger.String("name", name), logger.Err(err))
		return nil, err
	}

	v := res.Volume
	detail := &VolumeDetail{
		Name:       v.Name,
		Driver:     v.Driver,
		Mountpoint: v.Mountpoint,
		Scope:      v.Scope,
		CreatedAt:  v.CreatedAt,
		Labels:     v.Labels,
		Options:    v.Options,
	}
	if v.UsageData != nil {
		detail.Size = v.UsageData.Size
		detail.RefCount = v.UsageData.RefCount
	}

	// 查询挂载该卷的容器（docker ps --filter volume=<name>）。
	if res, err := cli.ContainerList(ctx, client.ContainerListOptions{
		All:     true,
		Filters: client.Filters{}.Add("volume", name),
	}); err == nil {
		if len(res.Items) > 0 {
			detail.Containers = make([]VolumeContainer, 0, len(res.Items))
			for _, c := range res.Items {
				dest := ""
				for _, m := range c.Mounts {
					if m.Name == name || m.Source == name {
						dest = m.Destination
						break
					}
				}
				detail.Containers = append(detail.Containers, VolumeContainer{
					ID:    c.ID,
					Name:  c.Names[0],
					State: string(c.State),
					Dest:  dest,
				})
			}
		}
	}
	return detail, nil
}

// Remove 删除卷。
func (l *VolumeLogic) Remove(ctx context.Context, name string, force bool) error {
	logger.InfoCtx(ctx, "volume remove", logger.String("name", name), logger.Bool("force", force))
	cli, err := l.newClient()
	if err != nil {
		logger.ErrorCtx(ctx, "volume remove failed", logger.String("name", name), logger.Err(err))
		return err
	}
	defer cli.Close()
	_, err = cli.VolumeRemove(ctx, name, client.VolumeRemoveOptions{Force: force})
	if err != nil {
		logger.ErrorCtx(ctx, "volume remove failed", logger.String("name", name), logger.Err(err))
	}
	return err
}

// Prune 清理未使用卷（未被任何容器引用的匿名卷），返回被清理的卷名与回收字节数。
func (l *VolumeLogic) Prune(ctx context.Context) ([]string, int64, error) {
	logger.InfoCtx(ctx, "volume prune")
	cli, err := l.newClient()
	if err != nil {
		logger.ErrorCtx(ctx, "volume prune failed", logger.Err(err))
		return nil, 0, err
	}
	defer cli.Close()

	res, err := cli.VolumePrune(ctx, client.VolumePruneOptions{})
	if err != nil {
		logger.ErrorCtx(ctx, "volume prune failed", logger.Err(err))
		return nil, 0, err
	}
	return res.Report.VolumesDeleted, int64(res.Report.SpaceReclaimed), nil
}
