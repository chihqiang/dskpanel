import { http } from './http'
import { createSSEStream } from './stream'

export interface DockerInfo {
  available: boolean
  version: string
  platform: string
  os: string
  arch: string
  cpu: number
  memory: number
  error?: string
  ping_time: number
}

/** 检测本机 Docker 环境。 */
export function detectDocker() {
  return http.get<DockerInfo>('/api/v1/docker/detect')
}

/** Docker 资源统计。 */
export interface DockerStats {
  containers: number
  images: number
  networks: number
  volumes: number
  by_state: Record<string, number>
}

/** Docker 版本信息。 */
export interface DockerVersion {
  platform_name: string
  version: string
  api_version: string
  min_api_version: string
  os: string
  arch: string
  experimental: boolean
  client_version: string
}

/** 概览聚合：资源统计 + 版本信息。 */
export interface DockerOverview {
  stats: DockerStats
  version: DockerVersion
}

/** 获取概览聚合（资源统计 + 引擎版本，一次请求）。 */
export function dockerOverview() {
  return http.get<DockerOverview>('/api/v1/docker/overview')
}

/** 引擎完整信息（docker info 精选字段）。 */
export interface DockerSystemInfo {
  id: string
  name: string
  server_version: string
  kernel_version: string
  operating_system: string
  os_version: string
  os_type: string
  architecture: string
  driver: string
  driver_status: string[][]
  logging_driver: string
  cgroup_driver: string
  cgroup_version: string
  security_options: string[]
  default_runtime: string
  runtimes: string[]
  ncpu: number
  mem_total: number
  docker_root_dir: string
  index_server: string
  labels: string[]
  live_restore: boolean
  experimental_build: boolean
  containers: number
  containers_running: number
  containers_paused: number
  containers_stopped: number
  images: number
  debug: boolean
  n_goroutines: number
  system_time: string
}

/** 获取引擎完整信息。 */
export function dockerSystemInfo() {
  return http.get<DockerSystemInfo>('/api/v1/docker/info')
}

/** 一键清理结果分类。 */
export interface PruneCategory {
  deleted: number
  reclaimed: number
  error?: string
}

/** 一键清理汇总。 */
export interface DockerPruneResult {
  containers: PruneCategory
  images: PruneCategory
  networks: PruneCategory
  volumes: PruneCategory
  build_cache: PruneCategory
  total: PruneCategory
}

/** 一键清理未使用资源。 */
export function pruneAllDocker() {
  return http.post<DockerPruneResult>('/api/v1/docker/prune')
}

/** 节点指标（metric 采集的历史数据，含磁盘占用 storage）。 */
export interface NodeMetric {
  cpu: string
  memory: string
  storage: string
  host_core_utilization: string
  time: string
}

/** 查询节点指标历史（type=docker|swarm|k8s，limit 条数）。 */
export function listNodeMetrics(type = 'docker', limit = 100) {
  return http.get<NodeMetric[]>(`/api/v1/metrics/nodes?type=${type}&limit=${limit}`)
}

/** Docker 系统事件。 */
export interface DockerEvent {
  type: string
  action: string
  actor_id: string
  actor_attr?: Record<string, string>
  scope?: string
  time: number
}

/**
 * 订阅 Docker 系统事件（SSE 流式，实时推送容器/镜像/网络等 daemon 事件）。
 * @returns 取消函数
 */
export function dockerEventsStream(
  onEvent: (ev: DockerEvent) => void,
  onError: (msg: string) => void,
): () => void {
  return createSSEStream('/api/v1/docker/events', {
    method: 'GET',
    onMessage: (data) => {
      try {
        onEvent(JSON.parse(data) as DockerEvent)
      } catch {
        // 忽略非 JSON
      }
    },
    onError,
  })
}
