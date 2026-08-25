package svc

import (
	"gorm.io/gorm"

	"github.com/chihqiang/infra-go/logger"
	"github.com/chihqiang/infra-go/orm"

	"chihqiang/dskpanel/config"
	"chihqiang/dskpanel/db"
	"chihqiang/dskpanel/logic"
)

// AppContext 应用依赖装配。
type AppContext struct {
	Config config.Config
	DB     *gorm.DB

	AuthLogic      *logic.AuthLogic
	DockerLogic    *logic.DockerLogic
	ContainerLogic *logic.ContainerLogic
	ImageLogic     *logic.ImageLogic
	NetworkLogic   *logic.NetworkLogic
	VolumeLogic    *logic.VolumeLogic
	ComposeLogic   *logic.ComposeLogic
	MetricLogic    *logic.MetricLogic
	SwarmLogic     *logic.SwarmLogic
	K8sLogic       *logic.K8sLogic
}

// NewAppContext 按依赖顺序装配组件：orm → migrate → 各 Logic。
func NewAppContext(cfg config.Config) (*AppContext, error) {
	g, err := orm.New(cfg.DB)
	if err != nil {
		logger.Errorf("db connect failed: %v", err)
		return nil, err
	}
	if err := db.Migrate(g); err != nil {
		return nil, err
	}

	ctx := &AppContext{
		Config: cfg,
		DB:     g,

		AuthLogic:      logic.NewAuthLogic(cfg.Auth),
		DockerLogic:    logic.NewDockerLogic(),
		ContainerLogic: logic.NewContainerLogic(),
		ImageLogic:     logic.NewImageLogic(),
		NetworkLogic:   logic.NewNetworkLogic(),
		VolumeLogic:    logic.NewVolumeLogic(),
		ComposeLogic:   logic.NewComposeLogic(cfg.Deploy.Dir),
		MetricLogic:    logic.NewMetricLogic(g, cfg.Metric),
		SwarmLogic:     logic.NewSwarmLogic(cfg.Swarm),
		K8sLogic:       logic.NewK8sLogic(cfg.K8s),
	}
	return ctx, nil
}

// Close 关闭数据库连接。
func (s *AppContext) Close() {
	if s.DB != nil {
		_ = orm.Close(s.DB)
	}
}
