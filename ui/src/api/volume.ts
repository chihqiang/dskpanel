import { http } from './http'

export interface VolumeItem {
  name: string
  driver: string
  mountpoint: string
  scope: string
  labels?: Record<string, string>
  created_at: string
  size: number
  used: boolean
}

export interface VolumeDetail {
  name: string
  driver: string
  mountpoint: string
  scope: string
  created_at: string
  labels?: Record<string, string>
  options?: Record<string, string>
  size: number
  ref_count: number
  containers?: {
    id: string
    name: string
    state: string
    dest: string
  }[]
}

/** 卷列表。 */
export function listVolumes() {
  return http.get<VolumeItem[]>('/api/v1/volumes')
}

/** 卷详情。 */
export function inspectVolume(name: string) {
  return http.get<VolumeDetail>(`/api/v1/volumes/${name}`)
}

/** 创建卷。 */
export function createVolume(name: string, driver = 'local') {
  return http.post<string>('/api/v1/volumes', { name, driver })
}

/** 删除卷。 */
export function removeVolume(name: string, force = false) {
  return http.delete<string>(`/api/v1/volumes/${name}?force=${force}`)
}

/** 清理未使用卷。 */
export function pruneVolumes() {
  return http.post<{ deleted: string[]; reclaimed_bytes: number }>('/api/v1/volumes/prune')
}
