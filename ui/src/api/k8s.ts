/**
 * Kubernetes 后端 API 封装。
 * 命名风格与 api/swarm.ts 一致：k8sXxx 前缀 + 类型定义集中在文件顶部。
 */
import { http, getToken, ApiError } from './http'
import { createSSEStream } from './stream'

// ──────────────────────────────────────────────
// 集群状态
// ──────────────────────────────────────────────

/** 集群状态摘要。 */
export interface K8sStatus {
  available: boolean
  error?: string
  version?: string
  platform?: string
  git_version?: string
  build_date?: string
  go_version?: string
}

/** 集群概览。 */
export interface K8sOverview {
  status: K8sStatus
  nodes: K8sNodeItem[]
  namespaces: number
  pods: number
  services: number
  deployments: number
  statefulsets: number
  daemonsets: number
  summary: {
    node_count: number
    master_count: number
    worker_count: number
    nodes_ready: number
    pod_count: number
    pods_by_phase: Record<string, number>
    service_count: number
    deployment_count: number
    statefulset_count: number
    daemonset_count: number
    namespace_count: number
  }
}

// ──────────────────────────────────────────────
// 命名空间
// ──────────────────────────────────────────────

/** 命名空间列表项。 */
export interface K8sNamespaceItem {
  name: string
  status: string
  created_at: string
  labels?: Record<string, string>
}

// ──────────────────────────────────────────────
// 节点
// ──────────────────────────────────────────────

/** 节点污点。 */
export interface K8sTaint {
  key: string
  value?: string
  effect: string
}

/** 节点列表项。 */
export interface K8sNodeItem {
  name: string
  role: 'master' | 'worker'
  ready: boolean
  status: string
  version: string
  os: string
  arch: string
  kernel_version: string
  container_runtime: string
  internal_ip: string
  external_ip: string
  cpu: string
  memory: string
  pods_capacity: number
  labels?: Record<string, string>
  taints?: K8sTaint[]
  created_at: string
}

/** 节点资源使用率。 */
export interface K8sNodeUsage {
  name: string
  cpu_used: string
  cpu_total: string
  cpu_percent: string
  mem_used: string
  mem_total: string
  mem_percent: string
  pods_used: number
  pods_total: number
}

// ──────────────────────────────────────────────
// Pod
// ──────────────────────────────────────────────

/** Pod 内容器信息。 */
export interface K8sContainerItem {
  name: string
  image: string
  state: string
  ready: boolean
  restarts: number
  reason?: string
}

/** Pod 列表项。 */
export interface K8sPodItem {
  name: string
  namespace: string
  status: string
  ready: string
  restarts: number
  node_name: string
  ip: string
  image: string
  created_at: string
  labels?: Record<string, string>
  qos_class?: string
  containers?: K8sContainerItem[]
}

/** Pod exec 结果。 */
export interface K8sPodExecResult {
  stdout: string
  stderr?: string
}

// ──────────────────────────────────────────────
// 工作负载
// ──────────────────────────────────────────────

/** Deployment 列表项。 */
export interface K8sDeploymentItem {
  name: string
  namespace: string
  ready: string
  up_to_date: number
  available: number
  replicas: number
  desired: number
  image: string
  created_at: string
  labels?: Record<string, string>
}

/** StatefulSet 列表项。 */
export interface K8sStatefulSetItem {
  name: string
  namespace: string
  ready: string
  replicas: number
  image: string
  created_at: string
  labels?: Record<string, string>
}

/** DaemonSet 列表项。 */
export interface K8sDaemonSetItem {
  name: string
  namespace: string
  desired: number
  current: number
  ready: number
  available: number
  image: string
  created_at: string
  labels?: Record<string, string>
}

/** Job 列表项。 */
export interface K8sJobItem {
  name: string
  namespace: string
  completions: string
  duration: string
  status: string
  parallelism: number
  image: string
  created_at: string
  labels?: Record<string, string>
}

/** CronJob 列表项。 */
export interface K8sCronJobItem {
  name: string
  namespace: string
  schedule: string
  suspend: boolean
  active: number
  last_schedule: string
  image: string
  created_at: string
  labels?: Record<string, string>
}

// ──────────────────────────────────────────────
// Service / Ingress
// ──────────────────────────────────────────────

/** Service 端口。 */
export interface K8sServicePort {
  name?: string
  port: number
  target_port: string
  protocol: string
  node_port?: number
}

/** Service 列表项。 */
export interface K8sServiceItem {
  name: string
  namespace: string
  type: string
  cluster_ip: string
  external_ip?: string
  ports?: K8sServicePort[]
  selector?: Record<string, string>
  created_at: string
}

/** Ingress 列表项。 */
export interface K8sIngressItem {
  name: string
  namespace: string
  hosts?: string[]
  address?: string
  class_name?: string
  created_at: string
}

// ──────────────────────────────────────────────
// ConfigMap / Secret
// ──────────────────────────────────────────────

/** ConfigMap 列表项。 */
export interface K8sConfigMapItem {
  name: string
  namespace: string
  data_keys: number
  created_at: string
  labels?: Record<string, string>
}

/** Secret 列表项。 */
export interface K8sSecretItem {
  name: string
  namespace: string
  type: string
  data_keys: number
  created_at: string
  labels?: Record<string, string>
}

/** Secret 详情（不返回明文，仅 key 列表）。 */
export interface K8sSecretDetail {
  name: string
  namespace: string
  type: string
  data_keys: string[]
  labels?: Record<string, string>
  created_at: string
}

// ──────────────────────────────────────────────
// 事件
// ──────────────────────────────────────────────

/** 事件列表项。 */
export interface K8sEventItem {
  type: string
  reason: string
  message: string
  object: string
  count: number
  last_time: string
  first_time: string
}

// ──────────────────────────────────────────────
// PVC / StorageClass
// ──────────────────────────────────────────────
export interface K8sPVCItem {
  name: string
  namespace: string
  status: string
  volume_name: string
  storage_class: string
  access_modes: string
  capacity: string
  requested: string
  created_at: string
  labels?: Record<string, string>
}

/** StorageClass 列表项。 */
export interface K8sStorageClassItem {
  name: string
  provisioner: string
  reclaim_policy: string
  binding_mode: string
  default: boolean
  volume_binding: string
  created_at: string
}

// ──────────────────────────────────────────────
// YAML 透传
// ──────────────────────────────────────────────

/** 单个资源 apply/delete 结果。 */
export interface K8sApplyItem {
  kind: string
  name: string
  namespace?: string
  action: string
  message?: string
}

/** YAML apply/delete/dryrun 结果。 */
export interface K8sApplyResult {
  ok: boolean
  message: string
  items?: K8sApplyItem[]
}

/** 批量删除资源请求项。 */
export interface K8sDeleteResourceRequest {
  kind: string
  name: string
  namespace?: string
}

// ──────────────────────────────────────────────
// API
// ──────────────────────────────────────────────

/** 检测 K8s 集群可用性。 */
export function k8sDetect() {
  return http.get<K8sStatus>('/api/v1/k8s/detect')
}

/** 集群概览。 */
export function k8sOverview() {
  return http.get<K8sOverview>('/api/v1/k8s/overview')
}

/** 命名空间列表。 */
export function k8sNamespaces() {
  return http.get<K8sNamespaceItem[]>('/api/v1/k8s/namespaces')
}

/** 命名空间详情（原始对象，可 ?format=yaml）。 */
export function k8sNamespaceInspect(name: string, format: 'json' | 'yaml' = 'json') {
  return http.get<unknown>(`/api/v1/k8s/namespaces/${name}?format=${format}`)
}

/** 删除命名空间（NotFound 幂等）。 */
export function k8sDeleteNamespace(name: string) {
  return http.delete<string>(`/api/v1/k8s/namespaces/${name}`)
}

/** 节点列表。 */
export function k8sNodes() {
  return http.get<K8sNodeItem[]>('/api/v1/k8s/nodes')
}

/** 节点详情（原始对象，可 ?format=yaml）。 */
export function k8sNodeInspect(name: string, format: 'json' | 'yaml' = 'json') {
  return http.get<unknown>(`/api/v1/k8s/nodes/${name}?format=${format}`)
}

/** 节点资源使用率。 */
export function k8sNodeUsage(name: string) {
  return http.get<K8sNodeUsage>(`/api/v1/k8s/nodes/${name}/usage`)
}

/** 节点标记为不可调度。 */
export function k8sCordonNode(name: string) {
  return http.post<string>(`/api/v1/k8s/nodes/${name}/cordon`)
}

/** 节点恢复调度。 */
export function k8sUncordonNode(name: string) {
  return http.post<string>(`/api/v1/k8s/nodes/${name}/uncordon`)
}

/** 驱逐节点上的 Pod。 */
export function k8sDrainNode(name: string) {
  return http.post<string>(`/api/v1/k8s/nodes/${name}/drain`)
}

/** Pod 列表。 */
export function k8sPods(options: { namespace?: string; labelSelector?: string } = {}) {
  const params = new URLSearchParams()
  if (options.namespace) params.set('namespace', options.namespace)
  if (options.labelSelector) params.set('labelSelector', options.labelSelector)
  const qs = params.toString()
  return http.get<K8sPodItem[]>(`/api/v1/k8s/pods${qs ? `?${qs}` : ''}`)
}

/** Pod 详情（原始对象，可 ?format=yaml）。 */
export function k8sPodInspect(name: string, namespace: string, format: 'json' | 'yaml' = 'json') {
  return http.get<unknown>(`/api/v1/k8s/pods/${name}?namespace=${encodeURIComponent(namespace)}&format=${format}`)
}

/** 删除 Pod（NotFound 幂等）。 */
export function k8sDeletePod(name: string, namespace: string) {
  return http.delete<string>(`/api/v1/k8s/pods/${name}?namespace=${encodeURIComponent(namespace)}`)
}

/** Pod 日志 SSE 流（follow 实时推送）。返回取消函数。 */
export function streamK8sPodLogs(
  name: string,
  namespace: string,
  tail: number,
  container = '',
  onLine?: (line: string) => void,
  onError?: (msg: string) => void,
  onClose?: () => void,
): () => void {
  const params = new URLSearchParams({ namespace, tail: String(tail), follow: 'true' })
  if (container) params.set('container', container)
  return createSSEStream(`/api/v1/k8s/pods/${name}/logs?${params}`, {
    onMessage: onLine,
    onError,
    onClose,
  })
}

/** 在 Pod 中执行一次性命令（非交互）。 */
export function k8sExecPod(
  name: string,
  req: { namespace: string; container?: string; command: string[]; tty?: boolean },
) {
  return http.post<K8sPodExecResult>(`/api/v1/k8s/pods/${name}/exec`, req)
}

/** Deployment 列表。 */
export function k8sDeployments(namespace?: string) {
  const qs = namespace ? `?namespace=${encodeURIComponent(namespace)}` : ''
  return http.get<K8sDeploymentItem[]>(`/api/v1/k8s/deployments${qs}`)
}

/** Deployment 详情（原始对象，可 ?format=yaml）。 */
export function k8sDeploymentInspect(name: string, namespace: string, format: 'json' | 'yaml' = 'json') {
  return http.get<unknown>(`/api/v1/k8s/deployments/${name}?namespace=${encodeURIComponent(namespace)}&format=${format}`)
}

/** 删除 Deployment（NotFound 幂等）。 */
export function k8sDeleteDeployment(name: string, namespace: string) {
  return http.delete<string>(`/api/v1/k8s/deployments/${name}?namespace=${encodeURIComponent(namespace)}`)
}

/** 伸缩 Deployment。 */
export function k8sScaleDeployment(name: string, namespace: string, replicas: number) {
  return http.post<string>(`/api/v1/k8s/deployments/${name}/scale?namespace=${encodeURIComponent(namespace)}`, {
    replicas,
  })
}

/** 重启 Deployment（滚动重启）。 */
export function k8sRestartDeployment(name: string, namespace: string) {
  return http.post<string>(`/api/v1/k8s/deployments/${name}/restart?namespace=${encodeURIComponent(namespace)}`)
}

/** StatefulSet 列表。 */
export function k8sStatefulSets(namespace?: string) {
  const qs = namespace ? `?namespace=${encodeURIComponent(namespace)}` : ''
  return http.get<K8sStatefulSetItem[]>(`/api/v1/k8s/statefulsets${qs}`)
}

/** StatefulSet 详情（原始对象，可 ?format=yaml）。 */
export function k8sStatefulSetInspect(name: string, namespace: string, format: 'json' | 'yaml' = 'json') {
  return http.get<unknown>(`/api/v1/k8s/statefulsets/${name}?namespace=${encodeURIComponent(namespace)}&format=${format}`)
}

/** 删除 StatefulSet（NotFound 幂等）。 */
export function k8sDeleteStatefulSet(name: string, namespace: string) {
  return http.delete<string>(`/api/v1/k8s/statefulsets/${name}?namespace=${encodeURIComponent(namespace)}`)
}

/** 伸缩 StatefulSet。 */
export function k8sScaleStatefulSet(name: string, namespace: string, replicas: number) {
  return http.post<string>(`/api/v1/k8s/statefulsets/${name}/scale?namespace=${encodeURIComponent(namespace)}`, {
    replicas,
  })
}

/** 重启 StatefulSet。 */
export function k8sRestartStatefulSet(name: string, namespace: string) {
  return http.post<string>(`/api/v1/k8s/statefulsets/${name}/restart?namespace=${encodeURIComponent(namespace)}`)
}

/** DaemonSet 列表。 */
export function k8sDaemonSets(namespace?: string) {
  const qs = namespace ? `?namespace=${encodeURIComponent(namespace)}` : ''
  return http.get<K8sDaemonSetItem[]>(`/api/v1/k8s/daemonsets${qs}`)
}

/** DaemonSet 详情（原始对象，可 ?format=yaml）。 */
export function k8sDaemonSetInspect(name: string, namespace: string, format: 'json' | 'yaml' = 'json') {
  return http.get<unknown>(`/api/v1/k8s/daemonsets/${name}?namespace=${encodeURIComponent(namespace)}&format=${format}`)
}

/** 删除 DaemonSet（NotFound 幂等）。 */
export function k8sDeleteDaemonSet(name: string, namespace: string) {
  return http.delete<string>(`/api/v1/k8s/daemonsets/${name}?namespace=${encodeURIComponent(namespace)}`)
}

/** 重启 DaemonSet。 */
export function k8sRestartDaemonSet(name: string, namespace: string) {
  return http.post<string>(`/api/v1/k8s/daemonsets/${name}/restart?namespace=${encodeURIComponent(namespace)}`)
}

/** Service 列表。 */
export function k8sServices(namespace?: string) {
  const qs = namespace ? `?namespace=${encodeURIComponent(namespace)}` : ''
  return http.get<K8sServiceItem[]>(`/api/v1/k8s/services${qs}`)
}

/** Service 详情（原始对象，可 ?format=yaml）。 */
export function k8sServiceInspect(name: string, namespace: string, format: 'json' | 'yaml' = 'json') {
  return http.get<unknown>(`/api/v1/k8s/services/${name}?namespace=${encodeURIComponent(namespace)}&format=${format}`)
}

/** 删除 Service（NotFound 幂等）。 */
export function k8sDeleteService(name: string, namespace: string) {
  return http.delete<string>(`/api/v1/k8s/services/${name}?namespace=${encodeURIComponent(namespace)}`)
}

/** Ingress 列表。 */
export function k8sIngresses(namespace?: string) {
  const qs = namespace ? `?namespace=${encodeURIComponent(namespace)}` : ''
  return http.get<K8sIngressItem[]>(`/api/v1/k8s/ingresses${qs}`)
}

/** Ingress 详情（原始对象，可 ?format=yaml）。 */
export function k8sIngressInspect(name: string, namespace: string, format: 'json' | 'yaml' = 'json') {
  return http.get<unknown>(`/api/v1/k8s/ingresses/${name}?namespace=${encodeURIComponent(namespace)}&format=${format}`)
}

/** 删除 Ingress（NotFound 幂等）。 */
export function k8sDeleteIngress(name: string, namespace: string) {
  return http.delete<string>(`/api/v1/k8s/ingresses/${name}?namespace=${encodeURIComponent(namespace)}`)
}

/** ConfigMap 列表。 */
export function k8sConfigMaps(namespace?: string) {
  const qs = namespace ? `?namespace=${encodeURIComponent(namespace)}` : ''
  return http.get<K8sConfigMapItem[]>(`/api/v1/k8s/configmaps${qs}`)
}

/** ConfigMap 详情（原始对象，可 ?format=yaml）。 */
export function k8sConfigMapInspect(name: string, namespace: string, format: 'json' | 'yaml' = 'json') {
  return http.get<unknown>(`/api/v1/k8s/configmaps/${name}?namespace=${encodeURIComponent(namespace)}&format=${format}`)
}

/** 删除 ConfigMap（NotFound 幂等）。 */
export function k8sDeleteConfigMap(name: string, namespace: string) {
  return http.delete<string>(`/api/v1/k8s/configmaps/${name}?namespace=${encodeURIComponent(namespace)}`)
}

/** Secret 列表。 */
export function k8sSecrets(namespace?: string) {
  const qs = namespace ? `?namespace=${encodeURIComponent(namespace)}` : ''
  return http.get<K8sSecretItem[]>(`/api/v1/k8s/secrets${qs}`)
}

/** Secret 详情（脱敏，不返回明文）。 */
export function k8sSecretInspect(name: string, namespace: string) {
  return http.get<K8sSecretDetail>(`/api/v1/k8s/secrets/${name}?namespace=${encodeURIComponent(namespace)}`)
}

/** 删除 Secret（NotFound 幂等）。 */
export function k8sDeleteSecret(name: string, namespace: string) {
  return http.delete<string>(`/api/v1/k8s/secrets/${name}?namespace=${encodeURIComponent(namespace)}`)
}

/** 查询特定资源的事件（如 Pod / Deployment / Node）。 */
export function k8sResourceEvents(kind: string, name: string, namespace: string) {
  const qs = namespace ? `?namespace=${encodeURIComponent(namespace)}` : ''
  return http.get<K8sEventItem[]>(`/api/v1/k8s/events/${encodeURIComponent(kind)}/${encodeURIComponent(name)}${qs}`)
}

/** YAML 透传：apply（kubectl apply 语义，支持多文档）。 */
export function k8sApplyYAML(content: string) {
  return http.post<K8sApplyResult>('/api/v1/k8s/apply', { content })
}

/** YAML 透传：delete -f 语义。 */
export function k8sDeleteYAML(content: string) {
  return http.post<K8sApplyResult>('/api/v1/k8s/delete', { content })
}

/** YAML 透传：dry-run=server 语义（不实际创建）。 */
export function k8sDryRunYAML(content: string) {
  return http.post<K8sApplyResult>('/api/v1/k8s/dryrun', { content })
}

/** 批量删除资源（跨类型、跨命名空间）。 */
export function k8sDeleteResources(items: K8sDeleteResourceRequest[]) {
  return http.post<K8sApplyResult>('/api/v1/k8s/delete/resources', { items })
}

/**
 * 获取资源的原始 YAML 文本（后端 ?format=yaml 返回 text/yaml）。
 * @param path 资源路径，如 `deployments/nginx?namespace=default`（可能已带 query）。
 */
export async function k8sRawYaml(path: string): Promise<string> {
  const token = getToken()
  const sep = path.includes('?') ? '&' : '?'
  const resp = await fetch(`/api/v1/k8s/${path}${sep}format=yaml`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  })
  if (!resp.ok) {
    const text = await resp.text().catch(() => '')
    throw new ApiError(resp.status, text || `http ${resp.status}`)
  }
  return resp.text()
}

/**
 * 获取资源的原始 JSON 对象（后端 ?format=json，通过 httpx.OkJSON 返回 {code,msg,data}）。
 * @param path 资源路径，如 `deployments/nginx?namespace=default`（可能已带 query）。
 */
export function k8sRawJSON(path: string): Promise<Record<string, unknown>> {
  const sep = path.includes('?') ? '&' : '?'
  return http.get<Record<string, unknown>>(`/api/v1/k8s/${path}${sep}format=json`)
}

/** PVC 列表。 */
export function k8sPVCs(namespace?: string) {
  const qs = namespace ? `?namespace=${encodeURIComponent(namespace)}` : ''
  return http.get<K8sPVCItem[]>(`/api/v1/k8s/pvcs${qs}`)
}

/** PVC 详情（原始对象，可 ?format=yaml）。 */
export function k8sPVCInspect(name: string, namespace: string, format: 'json' | 'yaml' = 'json') {
  return http.get<unknown>(`/api/v1/k8s/pvcs/${name}?namespace=${encodeURIComponent(namespace)}&format=${format}`)
}

/** 删除 PVC（NotFound 幂等）。 */
export function k8sDeletePVC(name: string, namespace: string) {
  return http.delete<string>(`/api/v1/k8s/pvcs/${name}?namespace=${encodeURIComponent(namespace)}`)
}

/** StorageClass 列表。 */
export function k8sStorageClasses() {
  return http.get<K8sStorageClassItem[]>('/api/v1/k8s/storageclasses')
}

/** StorageClass 详情（原始对象，可 ?format=yaml）。 */
export function k8sStorageClassInspect(name: string, format: 'json' | 'yaml' = 'json') {
  return http.get<unknown>(`/api/v1/k8s/storageclasses/${name}?format=${format}`)
}

/** Job 列表。 */
export function k8sJobs(namespace?: string) {
  const qs = namespace ? `?namespace=${encodeURIComponent(namespace)}` : ''
  return http.get<K8sJobItem[]>(`/api/v1/k8s/jobs${qs}`)
}

/** Job 详情（原始对象，可 ?format=yaml）。 */
export function k8sJobInspect(name: string, namespace: string, format: 'json' | 'yaml' = 'json') {
  return http.get<unknown>(`/api/v1/k8s/jobs/${name}?namespace=${encodeURIComponent(namespace)}&format=${format}`)
}

/** 删除 Job（NotFound 幂等）。 */
export function k8sDeleteJob(name: string, namespace: string) {
  return http.delete<string>(`/api/v1/k8s/jobs/${name}?namespace=${encodeURIComponent(namespace)}`)
}

/** CronJob 列表。 */
export function k8sCronJobs(namespace?: string) {
  const qs = namespace ? `?namespace=${encodeURIComponent(namespace)}` : ''
  return http.get<K8sCronJobItem[]>(`/api/v1/k8s/cronjobs${qs}`)
}

/** CronJob 详情（原始对象，可 ?format=yaml）。 */
export function k8sCronJobInspect(name: string, namespace: string, format: 'json' | 'yaml' = 'json') {
  return http.get<unknown>(`/api/v1/k8s/cronjobs/${name}?namespace=${encodeURIComponent(namespace)}&format=${format}`)
}

/** 删除 CronJob（NotFound 幂等）。 */
export function k8sDeleteCronJob(name: string, namespace: string) {
  return http.delete<string>(`/api/v1/k8s/cronjobs/${name}?namespace=${encodeURIComponent(namespace)}`)
}

// ──────────────────────────────────────────────
// RBAC - Role / ClusterRole / RoleBinding / ClusterRoleBinding
// ──────────────────────────────────────────────

/** Role / ClusterRole 列表项。 */
export interface K8sRoleItem {
  name: string
  namespace?: string
  kind: string // Role / ClusterRole
  rules: number
  created_at: string
  labels?: Record<string, string>
}

/** RoleBinding / ClusterRoleBinding 列表项。 */
export interface K8sRoleBindingItem {
  name: string
  namespace?: string
  kind: string // RoleBinding / ClusterRoleBinding
  role_kind: string // Role / ClusterRole
  role_name: string
  subjects: number
  created_at: string
  labels?: Record<string, string>
}

/** Role 列表。 */
export function k8sRoles(namespace?: string) {
  const qs = namespace ? `?namespace=${encodeURIComponent(namespace)}` : ''
  return http.get<K8sRoleItem[]>(`/api/v1/k8s/roles${qs}`)
}

/** Role 详情（原始对象，可 ?format=yaml）。 */
export function k8sRoleInspect(name: string, namespace: string, format: 'json' | 'yaml' = 'json') {
  return http.get<unknown>(`/api/v1/k8s/roles/${name}?namespace=${encodeURIComponent(namespace)}&format=${format}`)
}

/** 删除 Role（NotFound 幂等）。 */
export function k8sDeleteRole(name: string, namespace: string) {
  return http.delete<string>(`/api/v1/k8s/roles/${name}?namespace=${encodeURIComponent(namespace)}`)
}

/** ClusterRole 列表。 */
export function k8sClusterRoles() {
  return http.get<K8sRoleItem[]>('/api/v1/k8s/clusterroles')
}

/** ClusterRole 详情（原始对象，可 ?format=yaml）。 */
export function k8sClusterRoleInspect(name: string, format: 'json' | 'yaml' = 'json') {
  return http.get<unknown>(`/api/v1/k8s/clusterroles/${name}?format=${format}`)
}

/** 删除 ClusterRole（NotFound 幂等）。 */
export function k8sDeleteClusterRole(name: string) {
  return http.delete<string>(`/api/v1/k8s/clusterroles/${name}`)
}

/** RoleBinding 列表。 */
export function k8sRoleBindings(namespace?: string) {
  const qs = namespace ? `?namespace=${encodeURIComponent(namespace)}` : ''
  return http.get<K8sRoleBindingItem[]>(`/api/v1/k8s/rolebindings${qs}`)
}

/** RoleBinding 详情（原始对象，可 ?format=yaml）。 */
export function k8sRoleBindingInspect(name: string, namespace: string, format: 'json' | 'yaml' = 'json') {
  return http.get<unknown>(`/api/v1/k8s/rolebindings/${name}?namespace=${encodeURIComponent(namespace)}&format=${format}`)
}

/** 删除 RoleBinding（NotFound 幂等）。 */
export function k8sDeleteRoleBinding(name: string, namespace: string) {
  return http.delete<string>(`/api/v1/k8s/rolebindings/${name}?namespace=${encodeURIComponent(namespace)}`)
}

/** ClusterRoleBinding 列表。 */
export function k8sClusterRoleBindings() {
  return http.get<K8sRoleBindingItem[]>('/api/v1/k8s/clusterrolebindings')
}

/** ClusterRoleBinding 详情（原始对象，可 ?format=yaml）。 */
export function k8sClusterRoleBindingInspect(name: string, format: 'json' | 'yaml' = 'json') {
  return http.get<unknown>(`/api/v1/k8s/clusterrolebindings/${name}?format=${format}`)
}

/** 删除 ClusterRoleBinding（NotFound 幂等）。 */
export function k8sDeleteClusterRoleBinding(name: string) {
  return http.delete<string>(`/api/v1/k8s/clusterrolebindings/${name}`)
}

// ──────────────────────────────────────────────
// HPA (HorizontalPodAutoscaler)
// ──────────────────────────────────────────────

/** HPA 列表项。 */
export interface K8sHPAItem {
  name: string
  namespace: string
  target_ref: string
  min_replicas: number
  max_replicas: number
  current_replicas: number
  desired_replicas: number
  metrics: string
  created_at: string
  labels?: Record<string, string>
}

/** HPA 列表。 */
export function k8sHPAs(namespace?: string) {
  const qs = namespace ? `?namespace=${encodeURIComponent(namespace)}` : ''
  return http.get<K8sHPAItem[]>(`/api/v1/k8s/hpas${qs}`)
}

/** HPA 详情（原始对象，可 ?format=yaml）。 */
export function k8sHPAInspect(name: string, namespace: string, format: 'json' | 'yaml' = 'json') {
  return http.get<unknown>(`/api/v1/k8s/hpas/${name}?namespace=${encodeURIComponent(namespace)}&format=${format}`)
}

/** 删除 HPA（NotFound 幂等）。 */
export function k8sDeleteHPA(name: string, namespace: string) {
  return http.delete<string>(`/api/v1/k8s/hpas/${name}?namespace=${encodeURIComponent(namespace)}`)
}
