import { http } from './http'
import { createSSEStream } from './stream'

export interface ComposeResult {
  ok: boolean
  message: string
}

/** 校验 Compose 文件。 */
export function validateCompose(content: string) {
  return http.post<ComposeResult>('/api/v1/compose/validate', { content })
}

/** 部署 Compose 应用。 */
export function deployCompose(content: string) {
  return http.post<ComposeResult>('/api/v1/compose/deploy', { content })
}

/**
 * 部署 Compose 应用（SSE 流式实时回显）。
 * @returns 取消函数
 */
export function deployComposeStream(
  content: string,
  onLine: (line: string) => void,
  onDone: (success: boolean) => void,
  onError: (msg: string) => void,
): () => void {
  return createSSEStream('/api/v1/compose/deploy/stream', {
    method: 'POST',
    body: { content },
    onMessage: (data) => onLine(data),
    onDone: (data) => onDone(data === 'success'),
    onError: (msg) => onError(msg),
  })
}

// ---- Compose 项目管理 ----

/** 项目列表项。 */
export interface ComposeProjectItem {
  name: string
  status: string
  config_files: string
  services: number
  running: number
  total: number
}

/** 项目内容器状态。 */
export interface ComposeContainerStatus {
  id: string
  name: string
  service: string
  image: string
  state: string
  status: string
  health: string
  ports: string[]
  created: number
}

/** 项目详情。 */
export interface ComposeProjectDetail {
  name: string
  status: string
  services: number
  running: number
  total: number
  containers: ComposeContainerStatus[]
}

/** 列出所有 Compose 项目。 */
export function listComposeProjects() {
  return http.get<ComposeProjectItem[]>('/api/v1/compose/projects')
}

/** 查询项目内容器状态。 */
export function composeProjectPs(name: string) {
  return http.get<ComposeProjectDetail>(`/api/v1/compose/projects/${encodeURIComponent(name)}/ps`)
}

/** 启动项目。 */
export function composeProjectStart(name: string) {
  return http.post<string>(`/api/v1/compose/projects/${encodeURIComponent(name)}/start`)
}

/** 停止项目。 */
export function composeProjectStop(name: string) {
  return http.post<string>(`/api/v1/compose/projects/${encodeURIComponent(name)}/stop`)
}

/** 重启项目。 */
export function composeProjectRestart(name: string) {
  return http.post<string>(`/api/v1/compose/projects/${encodeURIComponent(name)}/restart`)
}

/** 停止并移除项目；volumes 为 true 时同时删除命名卷。 */
export function composeProjectDown(name: string, volumes = false) {
  const query = volumes ? '?volumes=1' : ''
  return http.post<string>(`/api/v1/compose/projects/${encodeURIComponent(name)}/down${query}`)
}

/** 拉取项目日志。 */
export function composeProjectLogs(name: string, tail = 200) {
  return http.get<string[]>(`/api/v1/compose/projects/${encodeURIComponent(name)}/logs?tail=${tail}`)
}
