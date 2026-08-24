package main

import (
	"os"

	"github.com/chihqiang/infra-go/conf"
	"github.com/chihqiang/infra-go/httpx"
	"github.com/chihqiang/infra-go/logger"
	"github.com/chihqiang/infra-go/service"

	"chihqiang/dskpanel/config"
	"chihqiang/dskpanel/route"
	"chihqiang/dskpanel/svc"
)

func main() {
	// 1. 加载配置（默认从 config.yaml，支持 -c 指定；敏感信息用环境变量覆盖）。
	cfgFile := "config.yaml"
	if len(os.Args) > 1 && os.Args[1] == "-c" && len(os.Args) > 2 {
		cfgFile = os.Args[2]
	}

	var cfg config.Config
	conf.MustLoad(cfgFile, &cfg, conf.UseEnv())

	// 2. 初始化日志（最先初始化）。
	l := logger.New(cfg.Logger)
	logger.SetGlobal(l)
	defer logger.Sync()

	// 3. 创建 AppContext（orm → migrate → 各 Logic）。
	appCtx, err := svc.NewAppContext(cfg)
	if err != nil {
		logger.Fatalf("init app context failed: %v", err)
	}
	defer appCtx.Close()

	// 4. 创建 HTTP 服务并注册路由。
	server := httpx.NewServer(cfg.Server)
	route.Register(server, appCtx)

	// 5. 服务组统一启停（支持优雅关闭）。
	sg := service.NewServiceGroup()
	sg.Add(appCtx.MetricLogic) // 指标采集器（未启用时直接待命）。

	// server 启动：httpx.Server 内部捕获 SIGINT/SIGTERM 自行优雅关闭并返回。
	// 这里在 server 结束后统一触发 sg.Stop()，否则 MetricLogic 等阻塞服务无人
	// 调用 Stop（cancel），进程将无法退出。
	sg.Add(service.WithStart(func() {
		logger.Infof("%s server starting on %s:%d", cfg.App.Name, cfg.Server.Host, cfg.Server.Port)
		if err := server.Start(); err != nil {
			logger.Errorf("server start failed: %v", err)
		}
		sg.Stop()
	}))
	sg.Start()
}
