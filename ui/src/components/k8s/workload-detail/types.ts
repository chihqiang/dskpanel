/**
 * 工作负载详情统一类型定义。
 *
 * 后端 Inspect 接口返回的是原始 K8s 对象（appsv1.Deployment / StatefulSet / DaemonSet / Job / CronJob），
 * 前端通过该类型提取基本信息（名称、命名空间、标签、注解、选择器等），
 * 使详情弹窗不关心具体工作负载类型。
 */

/** 工作负载类型。 */
export type WorkloadKind = 'Deployment' | 'StatefulSet' | 'DaemonSet' | 'Job' | 'CronJob'

/** 工作负载基本信息（从原始对象中提取）。 */
export interface WorkloadBasicInfo {
  /** 名称。 */
  name: string
  /** 命名空间。 */
  namespace: string
  /** 创建时间。 */
  created_at: string
  /** 标签。 */
  labels: Record<string, string>
  /** 注解。 */
  annotations: Record<string, string>
  /** 选择器（label selector），键值对。 */
  selector: Record<string, string>
  /** 更新策略。 */
  strategy: string
  /** 状态摘要。 */
  status: WorkloadStatus
}

/** 工作负载状态摘要。 */
export interface WorkloadStatus {
  /** 就绪，如 "2/3"。 */
  ready: string
  /** 已更新副本数。 */
  up_to_date: number
  /** 可用副本数。 */
  available: number
  /** 期望副本数。 */
  desired: number
  /** 当前副本数。 */
  replicas: number
}

/** 工作负载中的容器信息（从 PodTemplate 中提取）。 */
export interface WorkloadContainer {
  /** 容器名。 */
  name: string
  /** 镜像。 */
  image: string
  /** 端口列表。 */
  ports: { containerPort: number; name?: string; protocol?: string }[]
  /** 环境变量。 */
  env: { name: string; value?: string }[]
  /** 资源请求/限制。 */
  resources?: {
    requests?: Record<string, string>
    limits?: Record<string, string>
  }
}

/** 从原始 K8s Deployment / StatefulSet / DaemonSet 对象提取基本信息。 */
export function extractBasicInfo(
  obj: Record<string, unknown>,
  kind: WorkloadKind,
): WorkloadBasicInfo {
  const meta = (obj.metadata ?? {}) as Record<string, unknown>
  const spec = (obj.spec ?? {}) as Record<string, unknown>
  const status = (obj.status ?? {}) as Record<string, unknown>

  const labels = (meta.labels ?? {}) as Record<string, string>
  const annotations = (meta.annotations ?? {}) as Record<string, string>
  const selectorObj = (spec.selector ?? {}) as Record<string, unknown>
  const matchLabels = (selectorObj.matchLabels ?? {}) as Record<string, string>

  // 更新策略。
  let strategy = '—'
  if (kind === 'Deployment') {
    const s = (spec.strategy ?? {}) as Record<string, unknown>
    strategy = (s.type as string) ?? 'RollingUpdate'
  } else if (kind === 'StatefulSet') {
    const s = (spec.updateStrategy ?? {}) as Record<string, unknown>
    strategy = (s.type as string) ?? 'RollingUpdate'
  } else if (kind === 'DaemonSet') {
    const s = (spec.updateStrategy ?? {}) as Record<string, unknown>
    strategy = (s.type as string) ?? 'RollingUpdate'
  }

  // 状态。
  const readyReplicas = (status.readyReplicas as number) ?? 0
  const replicas = (status.replicas as number) ?? 0
  const updatedReplicas = (status.updatedReplicas as number) ?? 0
  const availableReplicas = (status.availableReplicas as number) ?? 0
  const desiredReplicas = kind === 'DaemonSet'
    ? ((status.desiredNumberScheduled as number) ?? 0)
    : ((spec.replicas as number) ?? replicas)

  return {
    name: (meta.name as string) ?? '',
    namespace: (meta.namespace as string) ?? '',
    created_at: (meta.creationTimestamp as string) ?? '',
    labels,
    annotations,
    selector: matchLabels,
    strategy,
    status: {
      ready: kind === 'DaemonSet'
        ? `${readyReplicas}/${desiredReplicas}`
        : `${readyReplicas}/${replicas}`,
      up_to_date: updatedReplicas,
      available: availableReplicas,
      desired: desiredReplicas,
      replicas,
    },
  }
}

/** 从原始对象提取容器信息列表。 */
export function extractContainers(obj: Record<string, unknown>): WorkloadContainer[] {
  const spec = (obj.spec ?? {}) as Record<string, unknown>
  const template = (spec.template ?? {}) as Record<string, unknown>
  const podSpec = (template.spec ?? {}) as Record<string, unknown>
  const containers = (podSpec.containers ?? []) as Record<string, unknown>[]

  return containers.map((c) => {
    const ports = (c.ports ?? []) as Record<string, unknown>[]
    const env = (c.env ?? []) as Record<string, unknown>[]
    const resources = c.resources as Record<string, unknown> | undefined
    return {
      name: (c.name as string) ?? '',
      image: (c.image as string) ?? '',
      ports: ports.map((p) => ({
        containerPort: (p.containerPort as number) ?? 0,
        name: p.name as string | undefined,
        protocol: p.protocol as string | undefined,
      })),
      env: env.map((e) => ({
        name: (e.name as string) ?? '',
        value: e.value as string | undefined,
      })),
      resources: resources
        ? {
            requests: (resources.requests ?? {}) as Record<string, string>,
            limits: (resources.limits ?? {}) as Record<string, string>,
          }
        : undefined,
    }
  })
}

/** 从原始对象中提取 labelSelector 字符串（用于查询关联 Pod）。 */
export function extractLabelSelector(obj: Record<string, unknown>): string {
  const spec = (obj.spec ?? {}) as Record<string, unknown>
  const selectorObj = (spec.selector ?? {}) as Record<string, unknown>
  const matchLabels = (selectorObj.matchLabels ?? {}) as Record<string, string>
  return Object.entries(matchLabels)
    .map(([k, v]) => `${k}=${v}`)
    .join(',')
}
