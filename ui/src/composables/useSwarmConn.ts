import { inject, provide, type ComputedRef } from 'vue'
import type { SwarmStatus } from '@/api/swarm'

/** 当前 Swarm 连接上下文（连接目标由 config.yaml 的 swarm 段决定，无需切换）。 */
export interface SwarmConnContext {
  /** 集群状态（检测结果）。 */
  detect: ComputedRef<SwarmStatus | null>
  /** 集群显示名。 */
  label: ComputedRef<string>
  /** 刷新检测结果。 */
  refresh: () => Promise<void>
  /** 触发子页面重新加载（刷新后调用）。 */
  reload: () => void
}

const KEY = 'swarm-conn-context'

/** 在 SwarmLayout 中提供连接上下文。 */
export function provideSwarmConn(ctx: SwarmConnContext): void {
  provide(KEY, ctx)
}

/** 在子页面中获取连接上下文。 */
export function useSwarmConn(): SwarmConnContext {
  const ctx = inject<SwarmConnContext>(KEY)
  if (!ctx) {
    throw new Error('useSwarmConn must be used within SwarmLayout')
  }
  return ctx
}

/** 根据集群状态生成显示名。 */
export function swarmLabel(st: SwarmStatus | null): string {
  if (!st) return '未连接'
  if (!st.available) return '不可用'
  if (st.name) return st.name
  return 'Swarm'
}
