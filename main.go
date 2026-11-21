package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chihqiang/ten-go/core/conf"
	"github.com/chihqiang/ten-go/core/logger"
	"github.com/chihqiang/ten-go/core/service"
	"github.com/chihqiang/ten-go/rest"

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
	// ten-go rest 的恢复/请求 ID/访问日志默认值只在零值 Config 时启用；
	// 这里由配置文件构造 Config，显式打开这几项内置中间件。
	defaults := rest.DefaultConfig()
	cfg.Middlewares.Recovery = defaults.Middlewares.Recovery
	cfg.Middlewares.RequestID = defaults.Middlewares.RequestID
	logCfg := defaults.Middlewares.Log
	logCfg.ServiceName = cfg.Name
	cfg.Middlewares.Log = logCfg

	server, err := rest.NewServer(cfg.Config, rest.WithLogger(l), rest.WithCORS("*"))
	if err != nil {
		logger.Fatalf("create http server failed: %v", err)
	}
	route.Register(server, appCtx)

	// 5. 服务组统一启停（支持优雅关闭）。
	sg := service.NewServiceGroup()
	sg.Add(appCtx.MetricLogic) // 指标采集器（未启用时直接待命）。

	// ten-go rest.Server 不内置信号处理，这里自行监听并优雅关闭；
	// server 结束后统一触发 sg.Stop()，否则 MetricLogic 等阻塞服务无人调用
	// Stop（cancel），进程将无法退出。
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(sigCh)
	go func() {
		<-sigCh
		shutdownTimeout := cfg.ShutdownTimeout
		if shutdownTimeout <= 0 {
			shutdownTimeout = 10 * time.Second
		}
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			logger.Errorf("server graceful shutdown failed: %v", err)
		}
	}()

	sg.Add(service.WithStart(func() {
		logger.Infof("%s server starting on %s:%d", cfg.Name, cfg.Host, cfg.Port)
		if err := server.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Errorf("server start failed: %v", err)
		}
		sg.Stop()
	}))
	sg.Start()
}
