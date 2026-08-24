import { getToken, http } from './http'
import { createSSEStream } from './stream'

/** Swarm 检测结果（连接目标由后端 config 决定）：即集群状态摘要。 */
export type SwarmDetect = SwarmStatus

/** 集群状态摘要。 */
export interface SwarmStatus {
  available: boolean
  error?: string
  id?: string
  name?: string
  managers?: number
  nodes?: number
  version?: string
}

/** 节点列表项。 */
export interface SwarmNodeItem {
  id: string
  name: string
  role: 'manager' | 'worker'
  state: string
  availability: string
  status: string
  addr: string
  version: string
  labels: number
  engine_err?: string
  updated_at: string
}

/** 服务列表项。 */
export interface SwarmServiceItem {
  id: string
  name: string
  mode: 'replicated' | 'global'
  replicas: string
  image: string
  ports: string[]
  state: 'running' | 'partially' | 'down' | ''
  updated_at: string
  has_update: boolean
}

/** 任务列表项。 */
export interface SwarmTaskItem {
  id: string
  service_id: string
  service_name: string
  node_id: string
  node_name: string
  slot: number
  image: string
  state: string
  desired_state: string
  error?: string
  container_id?: string
  updated_at: string
}

/** Secret / Config 列表项。 */
export interface SwarmSecretItem {
  id: string
  name: string
  created_at: string
  updated_at: string
}

/** 概览。 */
export interface SwarmOverview {
  status: SwarmStatus
  nodes: SwarmNodeItem[]
  services: SwarmServiceItem[]
  tasks: SwarmTaskItem[]
  secrets: number
  configs: number
  summary: {
    node_count: number
    manager_count: number
    worker_count: number
    nodes_by_state: Record<string, number>
    service_count: number
    service_running: number
    task_count: number
    tasks_by_state: Record<string, number>
    secrets_count: number
    configs_count: number
  }
}

/** 服务创建/更新请求。 */
export interface ServicePort {
  published: number
  target: number
  protocol?: 'tcp' | 'udp' | 'sctp'
}

export interface ServiceMount {
  type?: 'volume' | 'bind' | 'tmpfs'
  source?: string
  target: string
  read_only?: boolean
}

/** 服务创建/更新请求：透传完整 ServiceSpec（YAML 或 JSON 格式）。 */
export interface ServiceRequest {
  spec?: string
}

/** 网络列表项。 */
export interface SwarmNetworkItem {
  id: string
  name: string
  scope: string
  driver: string
  attachable: boolean
}

/** 集群镜像列表项。 */
export interface SwarmImageItem {
  id: string
  repo_tags: string[]
}

/** 集群镜像列表（服务创建表单选择）。 */
export function swarmImages() {
  return http.get<SwarmImageItem[]>(`/api/v1/swarm/images`)
}

/** 加入令牌。 */
export interface JoinTokens {
  worker: string
  manager: string
  addr: string
}

/** 创建网络请求。 */
export interface SwarmNetworkCreateRequest {
  name: string
  driver?: string
  subnet?: string
  gateway?: string
  attachable?: boolean
  internal?: boolean
  enable_ipv6?: boolean
  labels?: Record<string, string>
}

/** Secret / Config 详情。 */
export interface SwarmSecretDetail {
  id: string
  name: string
  /** 仅 Config 有内容；Secret 为空（Docker 不返回明文）。 */
  data?: string
  labels?: Record<string, string>
  created_at: string
  updated_at: string
}

/** 集群网络列表。 */
export function swarmNetworks() {
  return http.get<SwarmNetworkItem[]>(`/api/v1/swarm/networks`)
}

/** 创建集群网络。 */
export function swarmCreateNetwork(req: SwarmNetworkCreateRequest) {
  return http.post<string>(`/api/v1/swarm/networks`, req)
}

/** 网络详情（原始 JSON）。 */
export function swarmNetworkInspect(id: string) {
  return http.get<unknown>(`/api/v1/swarm/networks/${id}`)
}

/** 删除网络。 */
export function swarmRemoveNetwork(id: string) {
  return http.delete<string>(`/api/v1/swarm/networks/${id}`)
}

/** 获取集群加入令牌。 */
export function swarmJoinTokens() {
  return http.get<JoinTokens>(`/api/v1/swarm/join-tokens`)
}

/** 回滚服务。 */
export function swarmRollbackService(id: string) {
  return http.post<string>(`/api/v1/swarm/services/${id}/rollback`)
}

/** 强制更新服务（恢复暂停更新 / 滚动重启）。 */
export function swarmForceUpdateService(id: string) {
  return http.post<string>(`/api/v1/swarm/services/${id}/force-update`)
}

/** Secret 详情。 */
export function swarmSecretInspect(id: string) {
  return http.get<SwarmSecretDetail>(`/api/v1/swarm/secrets/${id}`)
}

/** Config 详情。 */
export function swarmConfigInspect(id: string) {
  return http.get<SwarmSecretDetail>(`/api/v1/swarm/configs/${id}`)
}

/** 检测可用 Swarm 集群。 */
export function swarmDetect() {
  return http.get<SwarmDetect>('/api/v1/swarm/detect')
}

/** 集群概览。 */
export function swarmOverview() {
  return http.get<SwarmOverview>(`/api/v1/swarm/overview`)
}

/** 节点列表。 */
export function swarmNodes() {
  return http.get<SwarmNodeItem[]>(`/api/v1/swarm/nodes`)
}

/** 节点详情（原始 JSON）。 */
export function swarmNodeInspect(id: string) {
  return http.get<unknown>(`/api/v1/swarm/nodes/${id}`)
}

/** 切换节点可用性。 */
export function swarmSetNodeAvailability(id: string, availability: string) {
  return http.post<string>(`/api/v1/swarm/nodes/${id}/availability`, { availability })
}

/** 删除节点。 */
export function swarmRemoveNode(id: string, force = false) {
  return http.delete<string>(`/api/v1/swarm/nodes/${id}`, { body: { force } })
}

/** 服务列表。 */
export function swarmServices() {
  return http.get<SwarmServiceItem[]>(`/api/v1/swarm/services`)
}

/** 服务详情（原始 JSON）。 */
export function swarmServiceInspect(id: string) {
  return http.get<unknown>(`/api/v1/swarm/services/${id}`)
}

/** 服务任务容器资源统计。 */
export interface SwarmContainerResource {
  task_id: string
  container_id: string
  service: string
  slot: number
  node_name: string
  state: string
  cpu_percent: number
  mem_usage: number
  mem_limit: number
  mem_percent: number
}

/** 服务级资源监控（任务容器 CPU/内存聚合）。 */
export function swarmServiceResources(id: string) {
  return http.get<SwarmContainerResource[]>(`/api/v1/swarm/services/${id}/resources`)
}

/** 创建服务。 */
export function swarmCreateService(req: ServiceRequest) {
  return http.post<string>(`/api/v1/swarm/services`, req)
}

/** 更新服务。 */
export function swarmUpdateService(id: string, req: ServiceRequest) {
  return http.post<string>(`/api/v1/swarm/services/${id}`, req)
}

/** 服务伸缩。 */
export function swarmScaleService(id: string, replicas: number) {
  return http.post<string>(`/api/v1/swarm/services/${id}/scale`, { replicas })
}

/** 删除服务。 */
export function swarmRemoveService(id: string) {
  return http.delete<string>(`/api/v1/swarm/services/${id}`)
}

/** 任务列表（可按服务过滤）。 */
export function swarmTasks(serviceId?: string) {
  const q = new URLSearchParams()
  if (serviceId) q.set('service', serviceId)
  const qs = q.toString()
  return http.get<SwarmTaskItem[]>(`/api/v1/swarm/tasks${qs ? `?${qs}` : ''}`)
}

/** 服务日志 SSE 地址（配合 EventSource 使用）。 */
export function swarmServiceLogsUrl(id: string, opts?: { tail?: number; follow?: boolean }) {
  const params = new URLSearchParams()
  if (opts?.tail != null) params.set('tail', String(opts.tail))
  if (opts?.follow != null) params.set('follow', String(opts.follow))
  const qs = params.toString()
  return `/api/v1/swarm/services/${id}/logs${qs ? `?${qs}` : ''}`
}

/**
 * 服务日志 SSE 流式消费（后端主动推送）。
 * @returns 取消函数（关闭连接）
 */
export function streamSwarmServiceLogs(
  id: string,
  tail: number,
  onLine: (line: string) => void,
  onError: (msg: string) => void,
  onClose: () => void,
): () => void {
  return createSSEStream(swarmServiceLogsUrl(id, { tail, follow: true }), {
    onMessage: (data) => onLine(data),
    onError: (msg) => onError(msg),
    onClose: () => onClose(),
  })
}

/** 拉取服务日志（一次性，供下载）。 */
export async function fetchSwarmServiceLogs(id: string, tail = 10000): Promise<string> {
  const url = swarmServiceLogsUrl(id, { tail, follow: false })
  const resp = await fetch(url, { headers: { Authorization: `Bearer ${getToken() ?? ''}` } })
  if (!resp.ok) throw new Error(`无法获取日志: HTTP ${resp.status}`)
  return resp.text()
}

/** 任务日志 SSE 流式消费。 */
export function streamSwarmTaskLogs(
  taskId: string,
  tail: number,
  onLine: (line: string) => void,
  onError: (msg: string) => void,
  onClose: () => void,
): () => void {
  const params = new URLSearchParams({ tail: String(tail), follow: 'true' })
  return createSSEStream(`/api/v1/swarm/tasks/${taskId}/logs?${params.toString()}`, {
    onMessage: (data) => onLine(data),
    onError: (msg) => onError(msg),
    onClose: () => onClose(),
  })
}

/** Secret 列表。 */
export function swarmSecrets() {
  return http.get<SwarmSecretItem[]>(`/api/v1/swarm/secrets`)
}

/** 创建 Secret。 */
export function swarmCreateSecret(name: string, data: string) {
  return http.post<string>(`/api/v1/swarm/secrets`, { name, data })
}

/** 删除 Secret。 */
export function swarmRemoveSecret(id: string) {
  return http.delete<string>(`/api/v1/swarm/secrets/${id}`)
}

/** Config 列表。 */
export function swarmConfigs() {
  return http.get<SwarmSecretItem[]>(`/api/v1/swarm/configs`)
}

/** 创建 Config。 */
export function swarmCreateConfig(name: string, data: string) {
  return http.post<string>(`/api/v1/swarm/configs`, { name, data })
}

/** 删除 Config。 */
export function swarmRemoveConfig(id: string) {
  return http.delete<string>(`/api/v1/swarm/configs/${id}`)
}
