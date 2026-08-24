/**
 * Docker / Swarm 状态相关常量与映射。
 *
 * 全项目共享，避免在各页面重复定义 stateVariant 等函数。
 */

/** Badge 组件支持的变体类型（与 Badge.vue 的 variant prop 一致）。 */
export type BadgeVariant = 'green' | 'red' | 'yellow' | 'gray' | 'blue' | 'purple'

// ──────────────────────────────────────────────
// Docker 容器
// ──────────────────────────────────────────────

/** 容器状态 → Badge 颜色变体。 */
export function containerStateVariant(state: string): BadgeVariant {
  if (state === 'running') return 'green'
  if (state === 'exited' || state === 'dead') return 'red'
  if (state === 'restarting' || state === 'paused') return 'yellow'
  return 'gray'
}

// ──────────────────────────────────────────────
// Swarm 节点
// ──────────────────────────────────────────────

/** Swarm 节点角色 → Badge 颜色变体。 */
export function nodeRoleVariant(role: string): BadgeVariant {
  if (role === 'manager') return 'blue'
  return 'gray'
}

/** Swarm 节点状态 → Badge 颜色变体。 */
export function nodeStateVariant(state: string): BadgeVariant {
  switch (state) {
    case 'ready':
      return 'green'
    case 'down':
    case 'disconnected':
      return 'red'
    case 'unknown':
      return 'gray'
    default:
      return 'yellow'
  }
}

/** Swarm 节点可用性 → Badge 颜色变体。 */
export function nodeAvailVariant(a: string): BadgeVariant {
  if (a === 'active') return 'green'
  if (a === 'drain') return 'yellow'
  return 'gray'
}

// ──────────────────────────────────────────────
// Swarm 服务
// ──────────────────────────────────────────────

/** Swarm 服务模式 → Badge 颜色变体。 */
export function serviceModeVariant(mode: string): BadgeVariant {
  if (mode === 'replicated') return 'blue'
  if (mode === 'global') return 'purple'
  return 'gray'
}

/** Swarm 服务状态 → Badge 颜色变体。 */
export function serviceStateVariant(s: string): BadgeVariant {
  if (s === 'running') return 'green'
  if (s === 'down') return 'red'
  if (s === 'partially') return 'yellow'
  return 'gray'
}

// ──────────────────────────────────────────────
// Swarm 任务 / 通用状态
// ──────────────────────────────────────────────

/** 任务状态 → Badge 颜色变体。 */
export function taskStateVariant(state: string): BadgeVariant {
  if (state === 'running') return 'green'
  if (state === 'failed') return 'red'
  if (state === 'pending' || state === 'assigned') return 'yellow'
  if (state === 'complete') return 'gray'
  return 'gray'
}

/**
 * 通用 Swarm 状态 → Badge 颜色变体（覆盖节点/服务/任务的混合状态）。
 * 用于概览页等需要统一处理多种状态的场景。
 */
export function swarmStateVariant(state: string): BadgeVariant {
  switch (state) {
    case 'ready':
    case 'running':
      return 'green'
    case 'down':
    case 'failed':
    case 'rejected':
      return 'red'
    case 'preparing':
    case 'pending':
    case 'drain':
    case 'partially':
      return 'yellow'
    case 'unknown':
      return 'gray'
    default:
      return 'gray'
  }
}
