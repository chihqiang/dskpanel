import { getToken, http } from './http'
import { createSSEStreamJSON, createUploadTask, type UploadTask } from './stream'

export interface ImageItem {
  id: string
  repo_tags: string[]
  repo_digests: string[]
  size: number
  created: number
  containers: number
}

export interface ImageDetail {
  id: string
  repo_tags: string[]
  repo_digests: string[]
  architecture: string
  variant?: string
  os: string
  os_version?: string
  author?: string
  created: string
  size: number
  rootfs_type: string
  layers: string[]
  manifests?: {
    id: string
    platform: string
    available: boolean
    content_size: number
  }[]
  history?: {
    id: string
    created_by: string
    size: number
    created: number
    comment?: string
  }[]
  config?: {
    user?: string
    working_dir?: string
    env?: string[]
    cmd?: string[]
    entrypoint?: string[]
    volumes?: Record<string, unknown>
    exposed_ports?: Record<string, unknown>
    labels?: Record<string, string>
    shell?: string[]
  }
}

/** 镜像列表。 */
export function listImages(dangling?: boolean) {
  const q = dangling === undefined ? '' : `?dangling=${dangling}`
  return http.get<ImageItem[]>(`/api/v1/images${q}`)
}

/** 批量删除镜像。 */
export function removeImages(ids: string[], force = false) {
  return http.post<{ deleted: number }>('/api/v1/images/remove', { ids, force })
}

/** 拉取镜像（SSE 流式，逐条回调进度消息）。
 * @returns 取消函数 */
export function pullImageStream(
  ref: string,
  onMessage: (msg: Record<string, unknown>) => void,
  onDone: () => void,
  onError: (msg: string) => void,
): () => void {
  return createSSEStreamJSON('/api/v1/images/pull', {
    method: 'POST',
    body: { ref },
    onMessage: (msg) => onMessage(msg),
    onDone: () => onDone(),
    onError: (msg) => onError(msg),
  })
}

/** 镜像详情。 */
export function inspectImage(id: string) {
  return http.get<ImageDetail>(`/api/v1/images/${id}`)
}

/** 清理未使用镜像；dangling=true 时仅清理悬空镜像。 */
export function pruneImages(dangling = false) {
  return http.post<{ deleted: number; reclaimed_bytes: number }>(
    `/api/v1/images/prune?dangling=${dangling}`,
  )
}

/** 导出镜像为 tar 文件（流式下载，onProgress 回调已下载字节数）。 */
export async function exportImage(
  ids: string[],
  nameHint: string,
  onProgress?: (loaded: number, total: number) => void,
): Promise<void> {
  const token = getToken() ?? ''
  const resp = await fetch(`/api/v1/images/export?ids=${encodeURIComponent(ids.join(','))}`, {
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
  // 完成 → 触发浏览器下载。
  const blob = new Blob(chunks as unknown as BlobPart[], { type: 'application/x-tar' })
  const name = (nameHint || ids[0]).replace(/[:\/]/g, '_') + '.tar'
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = name
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}

/** 导入镜像（上传 tar，XHR 上报上传进度）。
 * @returns promise + abort 取消函数 */
export function importImage(
  file: File,
  onProgress?: (loaded: number, total: number) => void,
): UploadTask<void> {
  return createUploadTask({
    url: '/api/v1/images/import',
    file,
    onProgress,
  })
}

/** 磁盘占用汇总。 */
export function getDiskUsage() {
  return http.get<{
    containers: { active_count: number; total_count: number; total_size: number; reclaimable: number }
    images: { active_count: number; total_count: number; total_size: number; reclaimable: number }
    build_cache: { active_count: number; total_count: number; total_size: number; reclaimable: number }
    volumes: { active_count: number; total_count: number; total_size: number; reclaimable: number }
  }>('/api/v1/images/disk-usage')
}

/** 推送镜像（SSE 流式，逐条回调进度消息）。
 * @returns 取消函数 */
export function pushImageStream(
  ref: string,
  onMessage: (msg: Record<string, unknown>) => void,
  onDone: () => void,
  onError: (msg: string) => void,
): () => void {
  return createSSEStreamJSON('/api/v1/images/push', {
    method: 'POST',
    body: { ref },
    onMessage: (msg) => onMessage(msg),
    onDone: () => onDone(),
    onError: (msg) => onError(msg),
  })
}

/** 删除镜像。 */
export function removeImage(id: string, force = false) {
  return http.delete<string>(`/api/v1/images/${id}?force=${force}`)
}

/** 镜像打标签。 */
export function tagImage(source: string, target: string) {
  return http.post<string>(`/api/v1/images/tag`, { source, target })
}
