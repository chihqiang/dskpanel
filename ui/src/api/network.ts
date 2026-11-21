import { http } from './http'

export interface NetworkItem {
  id: string
  name: string
  driver: string
  scope: string
  internal: boolean
  attachable: boolean
  ipam_driver: string
  labels?: Record<string, string>
  created: string
}

export interface NetworkDetail {
  id: string
  name: string
  driver: string
  scope: string
  internal: boolean
  attachable: boolean
  enable_ipv6: boolean
  ipam?: { subnet?: string; gateway?: string; ip_range?: string }[]
  containers?: {
    id: string
    name: string
    mac_address?: string
    ipv4_address?: string
    ipv6_address?: string
  }[]
  labels?: Record<string, string>
  created: string
}

/** 网络列表。 */
export function listNetworks() {
  return http.get<NetworkItem[]>('/api/v1/networks')
}

/** 网络详情。 */
export function inspectNetwork(id: string) {
  return http.get<NetworkDetail>(`/api/v1/networks/${id}`)
}

/** 创建网络（支持高级参数）。 */
export function createNetwork(req: {
  name: string
  driver?: string
  subnet?: string
  gateway?: string
  ip_range?: string
  internal?: boolean
  enable_ipv6?: boolean
  labels?: Record<string, string>
  driver_opts?: Record<string, string>
  ipam_driver?: string
}) {
  return http.post<{ id: string }>('/api/v1/networks', req)
}

/** 删除网络。 */
export function removeNetwork(id: string) {
  return http.delete<string>(`/api/v1/networks/${id}`)
}

/** 清理未使用网络。 */
export function pruneNetworks() {
  return http.post<{ deleted: string[] }>('/api/v1/networks/prune')
}

/** 将容器连接到网络。 */
export function connectContainerToNetwork(networkId: string, containerId: string, ipv4?: string) {
  return http.post<string>(`/api/v1/networks/${networkId}/connect`, {
    container_id: containerId,
    ipv4,
  })
}

/** 将容器从网络断开。 */
export function disconnectContainerFromNetwork(networkId: string, containerId: string, force = false) {
  return http.post<string>(`/api/v1/networks/${networkId}/disconnect`, {
    container_id: containerId,
    force,
  })
}
