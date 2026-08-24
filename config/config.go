package config

import (
	"time"

	"github.com/chihqiang/infra-go/httpx"
	"github.com/chihqiang/infra-go/logger"
	"github.com/chihqiang/infra-go/orm"
)

// Auth 登录鉴权配置（单账号）。
type Auth struct {
	// Secret token 签名密钥（生产环境务必用环境变量覆盖）。
	Secret string `json:",default=dskpanel-change-me"`
	// TokenTTL token 有效期，默认 24h。
	TokenTTL time.Duration `json:",default=24h"`
	// Username 登录用户名。
	Username string `json:"username"`
	// Password 登录密码（支持 bcrypt 哈希或明文）。
	Password string `json:"password"`
}

// Metric 指标采集配置（开启才采集，默认关闭）。
// 指标表（nodes/pods）复用 DB 配置的同一个 SQLite 库，无需单独指定库文件。
type Metric struct {
	Enabled    bool          `json:",default=false"`
	Resolution time.Duration `json:",default=60s"`
	Duration   time.Duration `json:",default=7200s"`
}

// Deploy 编排透传配置。
// 编排文件（compose 等）持久化到该目录作为备份，不自动清理。
type Deploy struct {
	// Dir 编排文件备份目录。
	Dir string `json:",default=/tmp/dskpanel/deploy"`
}

// Swarm Swarm 连接配置（单一目标，无需切换）。
// Endpoint 为空则连本机 Docker（要求已启用 swarm mode 且为 manager）；
// 配置 Endpoint 后连远程 Swarm manager（可选用 TLS 凭据）。
type Swarm struct {
	// Endpoint 远程 Swarm manager 地址（如 tcp://192.168.1.10:2376）；为空连本机。
	Endpoint string `json:"endpoint"`
	// CA CA 证书 PEM（校验服务端，可选）。
	CA string `json:"ca"`
	// Cert 客户端证书 PEM。
	Cert string `json:"cert"`
	// Key 客户端私钥 PEM。
	Key string `json:"key"`
}

// Config 应用配置。
type Config struct {
	App    App                `json:"app"`
	Server httpx.ServerConfig `json:"server"`
	Logger logger.Config      `json:"logger"`
	DB     orm.Config         `json:"db"`
	Auth   Auth               `json:"auth"`
	Metric Metric             `json:"metric"`
	Deploy Deploy             `json:"deploy"`
	Swarm  Swarm              `json:"swarm"`
}

// App 应用基础信息。
type App struct {
	Name string `json:",default=dskpanel"`
}
