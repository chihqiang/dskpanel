package logic

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/chihqiang/infra-go/logger"
	"github.com/moby/moby/client"
)

// ImageItem 镜像列表项。
type ImageItem struct {
	ID          string   `json:"id"`
	RepoTags    []string `json:"repo_tags"`
	RepoDigests []string `json:"repo_digests"`
	Size        int64    `json:"size"`
	Created     int64    `json:"created"`
	Containers  int64    `json:"containers"`
}

// ImageLogic 镜像管理逻辑。
type ImageLogic struct{}

// NewImageLogic 创建镜像管理逻辑。
func NewImageLogic() *ImageLogic {
	return &ImageLogic{}
}

// newClient 创建本机 Docker 客户端。
func (l *ImageLogic) newClient() (*client.Client, error) {
	return client.New(client.FromEnv)
}

// List 列出镜像。
// dangling：nil=全部，true=仅悬空（<none>），false=仅非悬空。
func (l *ImageLogic) List(ctx context.Context, dangling *bool) ([]ImageItem, error) {
	cli, err := l.newClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	opts := client.ImageListOptions{}
	if dangling != nil {
		opts.Filters = client.Filters{}.Add("dangling", boolToStr(*dangling))
	}
	res, err := cli.ImageList(ctx, opts)
	if err != nil {
		return nil, err
	}

	items := make([]ImageItem, 0, len(res.Items))
	for _, img := range res.Items {
		items = append(items, ImageItem{
			ID:          img.ID,
			RepoTags:    img.RepoTags,
			RepoDigests: img.RepoDigests,
			Size:        img.Size,
			Created:     img.Created,
			Containers:  img.Containers,
		})
	}
	return items, nil
}

// boolToStr 布尔转 "true"/"false"。
func boolToStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// Pull 拉取镜像。
// 返回一个响应体供调用方读取进度；调用方负责关闭。
func (l *ImageLogic) Pull(ctx context.Context, ref string) (io.ReadCloser, error) {
	logger.InfoCtx(ctx, "image pull", logger.String("ref", ref))
	cli, err := l.newClient()
	if err != nil {
		logger.ErrorCtx(ctx, "image pull failed", logger.String("ref", ref), logger.Err(err))
		return nil, err
	}
	// 注意：此处不关闭 cli，返回的读取器依赖连接；由调用方在读取完成后关闭。
	resp, err := cli.ImagePull(ctx, ref, client.ImagePullOptions{})
	if err != nil {
		cli.Close()
		logger.ErrorCtx(ctx, "image pull failed", logger.String("ref", ref), logger.Err(err))
		return nil, err
	}
	return resp, nil
}

// Remove 删除镜像。
func (l *ImageLogic) Remove(ctx context.Context, id string, force bool) error {
	logger.InfoCtx(ctx, "image remove", logger.String("id", id), logger.Bool("force", force))
	cli, err := l.newClient()
	if err != nil {
		logger.ErrorCtx(ctx, "image remove failed", logger.String("id", id), logger.Err(err))
		return err
	}
	defer cli.Close()
	_, err = cli.ImageRemove(ctx, id, client.ImageRemoveOptions{Force: force})
	if err != nil {
		logger.ErrorCtx(ctx, "image remove failed", logger.String("id", id), logger.Err(err))
	}
	return err
}

// RemoveBatch 批量删除镜像，返回成功删除数与第一个错误。
func (l *ImageLogic) RemoveBatch(ctx context.Context, ids []string, force bool) (int, error) {
	logger.InfoCtx(ctx, "image remove batch", logger.Int("count", len(ids)), logger.Bool("force", force))
	cli, err := l.newClient()
	if err != nil {
		logger.ErrorCtx(ctx, "image remove batch failed", logger.Err(err))
		return 0, err
	}
	defer cli.Close()

	deleted := 0
	for _, id := range ids {
		_, err := cli.ImageRemove(ctx, id, client.ImageRemoveOptions{Force: force})
		if err != nil {
			logger.WarnCtx(ctx, "image remove failed in batch", logger.String("id", id), logger.Err(err))
			continue
		}
		deleted++
	}
	return deleted, nil
}

// Export 导出镜像为 tar 流（docker save），调用方负责关闭。
// ids 可传镜像 tag 或 ID。
func (l *ImageLogic) Export(ctx context.Context, ids []string) (io.ReadCloser, error) {
	logger.InfoCtx(ctx, "image export", logger.Any("ids", ids))
	cli, err := l.newClient()
	if err != nil {
		logger.ErrorCtx(ctx, "image export failed", logger.Any("ids", ids), logger.Err(err))
		return nil, err
	}
	// 注意：此处不关闭 cli，返回的读取器依赖连接；由调用方在读取完成后关闭。
	res, err := cli.ImageSave(ctx, ids)
	if err != nil {
		cli.Close()
		logger.ErrorCtx(ctx, "image export failed", logger.Any("ids", ids), logger.Err(err))
		return nil, err
	}
	return res, nil
}

// Import 导入镜像（docker load）。
// reader 为 tar 流（docker save 产物），调用方负责关闭。
func (l *ImageLogic) Import(ctx context.Context, reader io.Reader) error {
	logger.InfoCtx(ctx, "image import")
	cli, err := l.newClient()
	if err != nil {
		logger.ErrorCtx(ctx, "image import failed", logger.Err(err))
		return err
	}
	defer cli.Close()

	res, err := cli.ImageLoad(ctx, reader)
	if err != nil {
		logger.ErrorCtx(ctx, "image import failed", logger.Err(err))
		return err
	}
	defer res.Close()
	// 读取（消费）结果体，确保 load 完成并捕获可能的错误。
	_, _ = io.Copy(io.Discard, res)
	return nil
}

// DiskUsage 汇总各资源磁盘占用（docker system df 语义）。
func (l *ImageLogic) DiskUsage(ctx context.Context) (*DiskUsageSummary, error) {
	logger.InfoCtx(ctx, "image disk usage")
	cli, err := l.newClient()
	if err != nil {
		logger.ErrorCtx(ctx, "image disk usage failed", logger.Err(err))
		return nil, err
	}
	defer cli.Close()

	res, err := cli.DiskUsage(ctx, client.DiskUsageOptions{
		Containers: true,
		Images:     true,
		BuildCache: true,
		Volumes:    true,
	})
	if err != nil {
		logger.ErrorCtx(ctx, "image disk usage failed", logger.Err(err))
		return nil, err
	}
	return &DiskUsageSummary{
		Containers: DiskUsageItem{
			ActiveCount: res.Containers.ActiveCount,
			TotalCount:  res.Containers.TotalCount,
			TotalSize:   res.Containers.TotalSize,
			Reclaimable: res.Containers.Reclaimable,
		},
		Images: DiskUsageItem{
			ActiveCount: res.Images.ActiveCount,
			TotalCount:  res.Images.TotalCount,
			TotalSize:   res.Images.TotalSize,
			Reclaimable: res.Images.Reclaimable,
		},
		BuildCache: DiskUsageItem{
			ActiveCount: res.BuildCache.ActiveCount,
			TotalCount:  res.BuildCache.TotalCount,
			TotalSize:   res.BuildCache.TotalSize,
			Reclaimable: res.BuildCache.Reclaimable,
		},
		Volumes: DiskUsageItem{
			ActiveCount: res.Volumes.ActiveCount,
			TotalCount:  res.Volumes.TotalCount,
			TotalSize:   res.Volumes.TotalSize,
			Reclaimable: res.Volumes.Reclaimable,
		},
	}, nil
}

// DiskUsageSummary 磁盘占用汇总。
type DiskUsageSummary struct {
	Containers DiskUsageItem `json:"containers"`
	Images     DiskUsageItem `json:"images"`
	BuildCache DiskUsageItem `json:"build_cache"`
	Volumes    DiskUsageItem `json:"volumes"`
}

// DiskUsageItem 单类资源占用。
type DiskUsageItem struct {
	ActiveCount int64 `json:"active_count"`
	TotalCount  int64 `json:"total_count"`
	TotalSize   int64 `json:"total_size"`
	Reclaimable int64 `json:"reclaimable"`
}

// Tag 为镜像打标签。
func (l *ImageLogic) Tag(ctx context.Context, source, target string) error {
	logger.InfoCtx(ctx, "image tag", logger.String("source", source), logger.String("target", target))
	cli, err := l.newClient()
	if err != nil {
		logger.ErrorCtx(ctx, "image tag failed", logger.String("source", source), logger.String("target", target), logger.Err(err))
		return err
	}
	defer cli.Close()
	_, err = cli.ImageTag(ctx, client.ImageTagOptions{Source: source, Target: target})
	if err != nil {
		logger.ErrorCtx(ctx, "image tag failed", logger.String("source", source), logger.String("target", target), logger.Err(err))
	}
	return err
}

// ImageDetail 镜像详情。
type ImageDetail struct {
	ID           string             `json:"id"`
	RepoTags     []string           `json:"repo_tags"`
	RepoDigests  []string           `json:"repo_digests"`
	Architecture string             `json:"architecture"`
	Variant      string             `json:"variant,omitempty"`
	Os           string             `json:"os"`
	OsVersion    string             `json:"os_version,omitempty"`
	Author       string             `json:"author,omitempty"`
	Created      string             `json:"created"`
	Size         int64              `json:"size"`
	RootFSType   string             `json:"rootfs_type"`
	Layers       []string           `json:"layers"`
	History      []ImageHistoryItem `json:"history,omitempty"`
	Manifests    []ImageManifest    `json:"manifests,omitempty"`
	Config       *ImageConfig       `json:"config,omitempty"`
}

// ImageManifest 镜像多平台清单摘要（多架构信息）。
type ImageManifest struct {
	ID          string `json:"id"`
	Platform    string `json:"platform"` // os/arch[/variant]
	Available   bool   `json:"available"`
	ContentSize int64  `json:"content_size"`
}

// ImageHistoryItem 镜像分层历史（docker history 语义：每个构建步骤）。
type ImageHistoryItem struct {
	ID        string `json:"id"`
	CreatedBy string `json:"created_by"`
	Size      int64  `json:"size"`
	Created   int64  `json:"created"`
	Comment   string `json:"comment,omitempty"`
}

// ImageConfig 镜像配置（OCI 标准字段子集）。
type ImageConfig struct {
	User         string              `json:"user,omitempty"`
	WorkingDir   string              `json:"working_dir,omitempty"`
	Env          []string            `json:"env,omitempty"`
	Cmd          []string            `json:"cmd,omitempty"`
	Entrypoint   []string            `json:"entrypoint,omitempty"`
	Volumes      map[string]struct{} `json:"volumes,omitempty"`
	ExposedPorts map[string]struct{} `json:"exposed_ports,omitempty"`
	Labels       map[string]string   `json:"labels,omitempty"`
	Shell        []string            `json:"shell,omitempty"`
}

// Inspect 查看镜像详情（含 Layers 分层）。
func (l *ImageLogic) Inspect(ctx context.Context, id string) (*ImageDetail, error) {
	logger.InfoCtx(ctx, "image inspect", logger.String("id", id))
	cli, err := l.newClient()
	if err != nil {
		logger.ErrorCtx(ctx, "image inspect failed", logger.String("id", id), logger.Err(err))
		return nil, err
	}
	defer cli.Close()

	res, err := cli.ImageInspect(ctx, id, client.ImageInspectWithManifests(true))
	if err != nil {
		logger.ErrorCtx(ctx, "image inspect failed", logger.String("id", id), logger.Err(err))
		return nil, err
	}

	detail := &ImageDetail{
		ID:           res.ID,
		RepoTags:     res.RepoTags,
		RepoDigests:  res.RepoDigests,
		Architecture: res.Architecture,
		Variant:      res.Variant,
		Os:           res.Os,
		OsVersion:    res.OsVersion,
		Author:       res.Author,
		Created:      res.Created,
		Size:         res.Size,
		RootFSType:   res.RootFS.Type,
		Layers:       res.RootFS.Layers,
	}
	if len(res.Manifests) > 0 {
		detail.Manifests = make([]ImageManifest, 0, len(res.Manifests))
		for _, m := range res.Manifests {
			platform := "unknown"
			contentSize := int64(0)
			if m.ImageData != nil {
				p := m.ImageData.Platform
				platform = p.OS + "/" + p.Architecture
				if p.Variant != "" {
					platform += "/" + p.Variant
				}
				contentSize = m.ImageData.Size.Unpacked
			}
			detail.Manifests = append(detail.Manifests, ImageManifest{
				ID:          m.ID,
				Platform:    platform,
				Available:   m.Available,
				ContentSize: contentSize,
			})
		}
	}
	if res.Config != nil {
		detail.Config = &ImageConfig{
			User:         res.Config.User,
			WorkingDir:   res.Config.WorkingDir,
			Env:          res.Config.Env,
			Cmd:          res.Config.Cmd,
			Entrypoint:   res.Config.Entrypoint,
			Volumes:      res.Config.Volumes,
			ExposedPorts: res.Config.ExposedPorts,
			Labels:       res.Config.Labels,
			Shell:        res.Config.Shell,
		}
	}

	// 解析构建历史（docker history），提供每层对应的命令/大小。
	hist, err := cli.ImageHistory(ctx, id)
	if err != nil {
		logger.WarnCtx(ctx, "image history failed", logger.String("id", id), logger.Err(err))
	} else {
		detail.History = make([]ImageHistoryItem, 0, len(hist.Items))
		for _, h := range hist.Items {
			detail.History = append(detail.History, ImageHistoryItem{
				ID:        h.ID,
				CreatedBy: h.CreatedBy,
				Size:      h.Size,
				Created:   h.Created,
				Comment:   h.Comment,
			})
		}
	}
	return detail, nil
}

// Prune 清理未使用镜像（dangling 与未被容器引用的），返回回收字节数与删除数量。
// Prune 清理未使用镜像。
// danglingOnly=true 时仅清理悬空镜像（<none>，无 tag 引用）；false 清理所有未被容器引用的镜像。
func (l *ImageLogic) Prune(ctx context.Context, danglingOnly bool) (int64, int, error) {
	logger.InfoCtx(ctx, "image prune", logger.Bool("dangling_only", danglingOnly))
	cli, err := l.newClient()
	if err != nil {
		logger.ErrorCtx(ctx, "image prune failed", logger.Err(err))
		return 0, 0, err
	}
	defer cli.Close()

	opts := client.ImagePruneOptions{}
	if danglingOnly {
		opts.Filters = client.Filters{}.Add("dangling", "true")
	}
	report, err := cli.ImagePrune(ctx, opts)
	if err != nil {
		logger.ErrorCtx(ctx, "image prune failed", logger.Err(err))
		return 0, 0, err
	}
	return int64(report.Report.SpaceReclaimed), len(report.Report.ImagesDeleted), nil
}

// Push 推送镜像到 registry。
// ref 需为完整引用，如 registry.example.com/ns/app:v1；返回流式进度，调用方负责关闭。
func (l *ImageLogic) Push(ctx context.Context, ref string) (io.ReadCloser, error) {
	logger.InfoCtx(ctx, "image push", logger.String("ref", ref))
	cli, err := l.newClient()
	if err != nil {
		logger.ErrorCtx(ctx, "image push failed", logger.String("ref", ref), logger.Err(err))
		return nil, err
	}
	// 注意：此处不关闭 cli，返回的读取器依赖连接；由调用方在读取完成后关闭。
	resp, err := cli.ImagePush(ctx, ref, client.ImagePushOptions{RegistryAuth: registryAuth(ref)})
	if err != nil {
		cli.Close()
		logger.ErrorCtx(ctx, "image push failed", logger.String("ref", ref), logger.Err(err))
		return nil, err
	}
	return resp, nil
}

// registryAuth 从本机 Docker 登录凭据（~/.docker/config.json）中提取 registry 的认证信息（base64）。
func registryAuth(ref string) string {
	reg := normalizeRegistry(registryHost(ref))
	cfgDir := os.Getenv("DOCKER_CONFIG")
	if cfgDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		cfgDir = filepath.Join(home, ".docker")
	}
	data, err := os.ReadFile(filepath.Join(cfgDir, "config.json"))
	if err != nil {
		return ""
	}
	var cfg struct {
		Auths map[string]struct {
			Auth     string `json:"auth"`
			Username string `json:"username"`
			Password string `json:"password"`
		} `json:"auths"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return ""
	}
	for key, entry := range cfg.Auths {
		if normalizeRegistry(key) != reg {
			continue
		}
		if entry.Auth != "" {
			// auth 字段本身就是 base64(user:pass)，可直接复用。
			return entry.Auth
		}
		payload, _ := json.Marshal(map[string]string{
			"username":      entry.Username,
			"password":      entry.Password,
			"serveraddress": key,
		})
		return base64.StdEncoding.EncodeToString(payload)
	}
	return ""
}

// registryHost 从镜像引用中解析出 registry 主机。
func registryHost(ref string) string {
	i := strings.IndexByte(ref, '/')
	if i > 0 {
		host := ref[:i]
		if strings.ContainsAny(host, ".:") || host == "localhost" {
			return host
		}
	}
	return "docker.io"
}

// normalizeRegistry 归一化 registry 标识，Docker Hub 统一为 https://index.docker.io/v1/。
func normalizeRegistry(host string) string {
	h := strings.TrimSuffix(host, "/")
	h = strings.TrimPrefix(h, "https://")
	h = strings.TrimPrefix(h, "http://")
	if h == "" || h == "index.docker.io" || h == "docker.io" || h == "registry-1.docker.io" {
		return "https://index.docker.io/v1/"
	}
	return h
}
