package model

import "time"

// NodeMetric 节点指标（nodes 表），Docker / Swarm / K8s 通用，type 区分。
type NodeMetric struct {
	ID                  int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Type                string    `gorm:"size:16;not null;index:idx_nodes_type_time;comment:集群类型" json:"type"`
	UID                 string    `gorm:"size:128;comment:节点UID" json:"uid"`
	Name                string    `gorm:"size:128;comment:节点名称" json:"name"`
	CPU                 string    `gorm:"size:32;comment:CPU使用量(毫核)" json:"cpu"`
	Memory              string    `gorm:"size:32;comment:内存使用量(KB)" json:"memory"`
	Storage             string    `gorm:"size:32;comment:临时存储使用量(KB)" json:"storage"`
	HostCoreUtilization string    `gorm:"size:32;comment:主机CPU利用率" json:"hostcoreutilization"`
	HostGPUMemoryUsage  string    `gorm:"size:32;comment:主机GPU显存使用量" json:"hostgpumemoryusage"`
	Time                time.Time `gorm:"index:idx_nodes_type_time;comment:采集时间" json:"time"`
}

// TableName 指定表名。
func (NodeMetric) TableName() string { return "nodes" }

// PodMetric Pod 指标（pods 表），K8s / Swarm 通用，type 区分。
type PodMetric struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Type      string    `gorm:"size:16;not null;index:idx_pods_type_time;comment:集群类型" json:"type"`
	UID       string    `gorm:"size:128;comment:Pod UID" json:"uid"`
	Name      string    `gorm:"size:128;comment:Pod名称" json:"name"`
	Namespace string    `gorm:"size:128;comment:命名空间" json:"namespace"`
	Container string    `gorm:"size:128;comment:容器名称" json:"container"`
	CPU       string    `gorm:"size:32;comment:CPU使用量(毫核)" json:"cpu"`
	Memory    string    `gorm:"size:32;comment:内存使用量(KB)" json:"memory"`
	Storage   string    `gorm:"size:32;comment:临时存储使用量(KB)" json:"storage"`
	Time      time.Time `gorm:"index:idx_pods_type_time;comment:采集时间" json:"time"`
}

// TableName 指定表名。
func (PodMetric) TableName() string { return "pods" }
