import { getToken, http } from './http'
import { createSSEStream } from './stream'

export interface ContainerPort {
  ip?: string
  private_port: number
  public_port?: number
  type: string
}

export interface ContainerItem {
  id: string
  names: string[]
  image: string
  image_id: string
  command: string
  state: string
  status: string
  ports: ContainerPort[]
  created: number
}

export interface ContainerDetail {
  id: string
  name: string
  image: string
  state: string
  status: string
  created: string
  restart_count: number
  config?: {
    env?: string[]
    cmd?: string[]
    entrypoint?: string[]
    labels?: Record<string, string>
  }
  host_config?: {
    binds?: string[]
    restart_policy?: string
    cpu_shares?: number
    memory?: number
    nano_cpus?: number
    cpuset_cpus?: string
    restart_max?: number
  }
  network_settings?: {
    ports?: Record<string, ContainerPort[]>
    networks?: Record<string, { ip_address?: string }>
  }
}

export interface PortMapping {
  container_port: number
  host_port: number
  protocol?: string
}

export interface CreateContainerRequest {
  name?: string
  image: string
  command?: string[]
  entrypoint?: string[]
  env?: string[]
  labels?: Record<string, string>
  binds?: string[]
  ports?: PortMapping[]
  network?: string
  restart_policy?: string
  auto_remove?: boolean
  detach?: boolean
  tty?: boolean
  open_stdin?: boolean
  // 高级字段。
  hostname?: string
  user?: string
  working_dir?: string
  cap_add?: string[]
  cap_drop?: string[]
  memory?: number
  nano_cpus?: number
  cpuset_cpus?: string
  env_file?: string[]
  extra_hosts?: string[]
}

export interface CreateContainerResult {
  id: string
  name: string
  warns?: string[]
}

/** 容器列表。 */
export function listContainers(all = true) {
  return http.get<ContainerItem[]>(`/api/v1/containers?all=${all}`)
}

/** 容器详情。 */
export function inspectContainer(id: string) {
  return http.get<ContainerDetail>(`/api/v1/containers/${id}`)
}

/** 容器完整 inspect 原始 JSON（排障用）。 */
export function inspectContainerRaw(id: string) {
  return http.get<unknown>(`/api/v1/containers/${id}/inspect-raw`)
}

/** 创建容器。 */
export function createContainer(req: CreateContainerRequest) {
  return http.post<CreateContainerResult>('/api/v1/containers', req)
}

/** 启动容器。 */
export function startContainer(id: string) {
  return http.post<string>(`/api/v1/containers/${id}/start`)
}

/** 停止容器。 */
export function stopContainer(id: string) {
  return http.post<string>(`/api/v1/containers/${id}/stop`)
}

/** 重启容器。 */
export function restartContainer(id: string) {
  return http.post<string>(`/api/v1/containers/${id}/restart`)
}

/** 删除容器。 */
export function removeContainer(id: string, force = false, removeVolumes = false) {
  return http.delete<string>(
    `/api/v1/containers/${id}?force=${force}&remove_volumes=${removeVolumes}`,
  )
}

/** 容器日志（HTTP 一次性拉取，返回纯文本）。 */
export function getContainerLogs(id: string, tail = '100'): Promise<string> {
  return http.get<Response>(`/api/v1/containers/${id}/logs?tail=${tail}`, { raw: true }).then((r) => r.text())
}

/**
 * 容器日志 SSE 流式消费（后端主动推送）。
 * @returns 取消函数（关闭连接）
 */
export function streamContainerLogs(
  id: string,
  tail: string,
  onLine: (line: string) => void,
  onError: (msg: string) => void,
  onClose: () => void,
): () => void {
  return createSSEStream(
    `/api/v1/containers/${id}/logs/stream?tail=${tail}&follow=true`,
    {
      onMessage: (data) => onLine(data),
      onError: (msg) => onError(msg),
      onClose: () => onClose(),
    },
  )
}

/** 容器资源统计。 */
export interface ContainerStats {
  cpu_percent: number
  mem_usage: number
  mem_limit: number
  mem_percent: number
  net_rx_bytes: number
  net_tx_bytes: number
  block_read: number
  block_write: number
  pids: number
  running: boolean
}

/** 获取容器实时资源统计。 */
export function getContainerStats(id: string) {
  return http.get<ContainerStats>(`/api/v1/containers/${id}/stats`)
}

/** 重命名容器。 */
export function renameContainer(id: string, name: string) {
  return http.post<string>(`/api/v1/containers/${id}/rename`, { name })
}

export type ContainerBatchAction = 'start' | 'stop' | 'restart' | 'remove'

/** 批量操作容器。 */
export function batchContainers(action: ContainerBatchAction, ids: string[]) {
  return http.post<{ done: number; failed?: string[] }>('/api/v1/containers/batch', { action, ids })
}

/** 容器内进程列表。 */
export interface ContainerTop {
  titles: string[]
  procs: string[][]
}

/** 查看容器内进程。 */
export function getContainerTop(id: string) {
  return http.get<ContainerTop>(`/api/v1/containers/${id}/top`)
}

/** 提交容器为镜像（docker commit）。 */
export function commitContainer(id: string, req: { reference?: string; comment?: string; author?: string }) {
  return http.post<{ id: string }>(`/api/v1/containers/${id}/commit`, req)
}

/** 更新容器资源/重启策略（docker update）。 */
export function updateContainer(
  id: string,
  req: {
    cpu_shares?: number
    memory?: number
    nano_cpus?: number
    cpuset_cpus?: string
    restart_policy?: string
    restart_max?: number
  },
) {
  return http.post<string>(`/api/v1/containers/${id}/update`, req)
}

/** 暂停容器。 */
export function pauseContainer(id: string) {
  return http.post<string>(`/api/v1/containers/${id}/pause`)
}

/** 恢复暂停的容器。 */
export function unpauseContainer(id: string) {
  return http.post<string>(`/api/v1/containers/${id}/unpause`)
}

/** 导出容器文件系统为 tar 文件（流式下载，onProgress 回调已下载字节数）。 */
export async function exportContainer(
  id: string,
  nameHint: string,
  onProgress?: (loaded: number, total: number) => void,
): Promise<void> {
  const token = getToken() ?? ''
  const resp = await fetch(`/api/v1/containers/${id}/export`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!resp.ok || !resp.body) {
    throw new Error(`导出失败: HTTP ${resp.status}`)
  }
  const total = Number(resp.headers.get('content-length') || 0)
  const reader = resp.body.getReader()
  const chunks: Uint8Array[] = []
  let loaded = 0
  for (;;) {
    const { done, value } = await reader.read()
    if (done) break
    chunks.push(value)
    loaded += value.byteLength
    onProgress?.(loaded, total || loaded)
  }
  const blob = new Blob(chunks as unknown as BlobPart[], { type: 'application/x-tar' })
  const name = (nameHint || id).replace(/[:\/]/g, '_') + '-export.tar'
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = name
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}
