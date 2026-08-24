package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/chihqiang/infra-go/logger"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"

	"chihqiang/dskpanel/config"
	"chihqiang/dskpanel/model"
)

// MetricLogic 指标采集器。
// 按配置开关（metric.enabled）定时采集本机 Docker 节点指标写入 nodes 表（type=docker）。
// Docker 单机无 Pod 概念，不写 pods 表。
// 数据按 metric.duration 保留，超出自动清理。
//
// 并发安全：ctx/cancel/stopped 在构造时初始化，避免 Start/Stop 并发时的数据竞争
// （Stop 可能在 Start 完成字段赋值前被调用，导致 cancel 为空而无法解除阻塞、进程无法退出）。
type MetricLogic struct {
	db       *gorm.DB
	cfg      config.Metric
	docker   *DockerLogic
	ctx      context.Context
	cancel   context.CancelFunc
	stopped  chan struct{}
	stopOnce sync.Once
}

// NewMetricLogic 创建指标采集器。
func NewMetricLogic(db *gorm.DB, cfg config.Metric) *MetricLogic {
	ctx, cancel := context.WithCancel(context.Background())
	return &MetricLogic{
		db:      db,
		cfg:     cfg,
		docker:  &DockerLogic{},
		ctx:     ctx,
		cancel:  cancel,
		stopped: make(chan struct{}),
	}
}

// Start 启动采集循环（实现 service.Service 接口，阻塞直到 Stop）。
func (l *MetricLogic) Start() {
	defer close(l.stopped)

	if !l.cfg.Enabled {
		logger.Infof("metric collector disabled, skip")
		<-l.ctx.Done()
		return
	}
	logger.Infof("metric collector started, resolution=%s duration=%s", l.cfg.Resolution, l.cfg.Duration)

	// 启动时立即采集一次。
	l.collectOnce(l.ctx)

	ticker := time.NewTicker(l.cfg.Resolution)
	defer ticker.Stop()
	for {
		select {
		case <-l.ctx.Done():
			logger.Infof("metric collector stopped")
			return
		case <-ticker.C:
			l.collectOnce(l.ctx)
		}
	}
}

// Stop 停止采集器（实现 service.Service 接口，幂等）。
func (l *MetricLogic) Stop() {
	l.stopOnce.Do(func() {
		l.cancel()
		<-l.stopped
	})
}

// collectOnce 采集一次本机 Docker 节点指标并清理过期数据。
func (l *MetricLogic) collectOnce(ctx context.Context) {
	info := l.docker.Detect(ctx)
	if !info.Available {
		logger.Infof("metric collect skipped: docker not available")
		return
	}

	hostname, _ := os.Hostname()
	cpuUsageMilli, cpuUtil, memUsedKB := l.collectContainerStats(ctx)
	diskUsedKB := l.collectDiskUsage(ctx)

	record := model.NodeMetric{
		Type:                "docker",
		UID:                 hostname,
		Name:                hostname,
		CPU:                 itoa(cpuUsageMilli), // 毫核
		Memory:              itoa(memUsedKB),     // KB
		Storage:             itoa(diskUsedKB),    // KB
		HostCoreUtilization: fmtFloat(cpuUtil),   // %
		Time:                time.Now(),
	}
	if err := l.db.WithContext(ctx).Create(&record).Error; err != nil {
		logger.ErrorCtx(ctx, "metric collect write failed", logger.Err(err))
		return
	}
	logger.InfoCtx(ctx, "metric collected",
		logger.String("name", hostname),
		logger.String("cpu", record.CPU),
		logger.String("mem", record.Memory),
		logger.String("storage", record.Storage))

	// 清理超过保留期的旧数据。
	cutoff := time.Now().Add(-l.cfg.Duration)
	if err := l.db.WithContext(ctx).
		Where("type = ? AND time < ?", "docker", cutoff).
		Delete(&model.NodeMetric{}).Error; err != nil {
		logger.ErrorCtx(ctx, "metric cleanup failed", logger.Err(err))
	}
}

// NodeMetricItem 节点指标查询项。
type NodeMetricItem struct {
	CPU                 string    `json:"cpu"`
	Memory              string    `json:"memory"`
	Storage             string    `json:"storage"`
	HostCoreUtilization string    `json:"host_core_utilization"`
	Time                time.Time `json:"time"`
}

// ListNodeMetrics 查询指定类型的节点指标（按时间升序，最多 limit 条）。
func (l *MetricLogic) ListNodeMetrics(ctx context.Context, typ string, limit int) ([]NodeMetricItem, error) {
	if limit <= 0 || limit > 10000 {
		limit = 100
	}
	var rows []model.NodeMetric
	if err := l.db.WithContext(ctx).
		Where("type = ?", typ).
		Order("time ASC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		logger.ErrorCtx(ctx, "metric list failed", logger.Err(err))
		return nil, err
	}
	items := make([]NodeMetricItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, NodeMetricItem{
			CPU:                 r.CPU,
			Memory:              r.Memory,
			Storage:             r.Storage,
			HostCoreUtilization: r.HostCoreUtilization,
			Time:                r.Time,
		})
	}
	return items, nil
}

// itoa 整数转字符串。
func itoa(n int64) string { return strconv.FormatInt(n, 10) }

// fmtFloat 浮点转字符串（保留 1 位小数）。
func fmtFloat(f float64) string { return fmt.Sprintf("%.1f", f) }

// collectContainerStats 聚合所有运行中容器的 CPU 使用率与内存使用量。
func (l *MetricLogic) collectContainerStats(ctx context.Context) (cpuMilli int64, cpuUtil float64, memKB int64) {
	cli, err := client.New(client.FromEnv)
	if err != nil {
		return 0, 0, 0
	}
	defer cli.Close()

	res, err := cli.ContainerList(ctx, client.ContainerListOptions{All: false})
	if err != nil {
		return 0, 0, 0
	}
	if len(res.Items) == 0 {
		return 0, 0, 0
	}

	var totalCPU float64
	var totalMem uint64
	for _, c := range res.Items {
		statsRes, err := cli.ContainerStats(ctx, c.ID, client.ContainerStatsOptions{
			Stream:                false,
			IncludePreviousSample: false,
		})
		if err != nil {
			continue
		}
		func() {
			defer statsRes.Body.Close()
			var s container.StatsResponse
			if err := json.NewDecoder(statsRes.Body).Decode(&s); err != nil {
				return
			}
			// CPU 使用率：容器累计 CPU 时间 / 系统累计时间 * 在线核数。
			if s.CPUStats.SystemUsage > 0 {
				cores := float64(s.CPUStats.OnlineCPUs)
				if cores <= 0 {
					cores = 1
				}
				totalCPU += float64(s.CPUStats.CPUUsage.TotalUsage) / float64(s.CPUStats.SystemUsage) * cores * 100
			}
			totalMem += s.MemoryStats.Usage
		}()
	}

	if totalCPU > 100 {
		totalCPU = 100
	}
	// CPU 使用量（毫核）：以利用率估算（按 1 核 1000 毫核 * 利用率近似）。
	cpuMilli = int64(totalCPU) * 10
	return cpuMilli, totalCPU, int64(totalMem / 1024)
}

// collectDiskUsage 采集磁盘占用（docker system df 汇总：容器+镜像+卷+构建缓存），返回 KB。
func (l *MetricLogic) collectDiskUsage(ctx context.Context) int64 {
	cli, err := client.New(client.FromEnv)
	if err != nil {
		return 0
	}
	defer cli.Close()

	res, err := cli.DiskUsage(ctx, client.DiskUsageOptions{})
	if err != nil {
		return 0
	}
	total := res.Containers.TotalSize + res.Images.TotalSize + res.Volumes.TotalSize + res.BuildCache.TotalSize
	return total / 1024
}
