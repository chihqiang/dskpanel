/**
 * Kubernetes 状态相关常量与映射。
 *
 * 与 utils/docker.ts 保持一致风格：Badge 颜色变体集中维护。
 */
import type { BadgeVariant } from './docker'

// ──────────────────────────────────────────────
// Pod
// ──────────────────────────────────────────────

/** Pod 阶段 → Badge 颜色变体。 */
export function podPhaseVariant(phase: string): BadgeVariant {
  switch (phase) {
    case 'Running':
      return 'green'
    case 'Succeeded':
      return 'blue'
    case 'Pending':
      return 'yellow'
    case 'Failed':
      return 'red'
    case 'Unknown':
      return 'gray'
    default:
      return 'gray'
  }
}

/** Pod 内容器状态 → Badge 颜色变体。 */
export function k8sContainerStateVariant(state: string): BadgeVariant {
  if (state === 'running') return 'green'
  if (state === 'waiting') return 'yellow'
  if (state === 'terminated') return 'gray'
  return 'gray'
}

// ──────────────────────────────────────────────
// 节点
// ──────────────────────────────────────────────

/** 节点角色 → Badge 颜色变体。 */
export function nodeRoleVariantK8s(role: string): BadgeVariant {
  if (role === 'master') return 'blue'
  return 'gray'
}

/** 节点是否 Ready → Badge 颜色变体。 */
export function nodeReadyVariant(ready: boolean): BadgeVariant {
  return ready ? 'green' : 'red'
}

// ──────────────────────────────────────────────
// 工作负载
// ──────────────────────────────────────────────

/** 工作负载就绪数（如 "2/3"）→ Badge 颜色变体。 */
export function workloadReadyVariant(ready: string): BadgeVariant {
  const parts = ready.split('/')
  const a = Number(parts[0]) || 0
  const b = Number(parts[1]) || 0
  if (b === 0) return 'gray'
  if (a >= b) return 'green'
  if (a > 0) return 'yellow'
  return 'red'
}

// ──────────────────────────────────────────────
// 网络 / 配置
// ──────────────────────────────────────────────

/** Service 类型 → Badge 颜色变体。 */
export function serviceTypeVariant(type: string): BadgeVariant {
  switch (type) {
    case 'LoadBalancer':
      return 'blue'
    case 'NodePort':
      return 'purple'
    case 'ExternalName':
      return 'yellow'
    default:
      return 'gray'
  }
}

/** 命名空间状态 → Badge 颜色变体。 */
export function nsStatusVariant(status: string): BadgeVariant {
  return status === 'Active' ? 'green' : 'yellow'
}

// ──────────────────────────────────────────────
// 事件
// ──────────────────────────────────────────────

/** 事件类型（Normal / Warning）→ Badge 颜色变体。 */
export function eventTypeVariant(type: string): BadgeVariant {
  return type === 'Warning' ? 'red' : 'green'
}
