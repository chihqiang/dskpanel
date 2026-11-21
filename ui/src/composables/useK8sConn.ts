import { inject, provide, type ComputedRef } from 'vue'
import type { K8sStatus } from '@/api/k8s'

/** 当前 K8s 连接上下文（连接目标由 config.yaml 的 k8s 段决定，无需切换）。 */
export interface K8sConnContext {
  /** 集群状态（检测结果）。 */
  detect: ComputedRef<K8sStatus | null>
  /** 集群显示名。 */
  label: ComputedRef<string>
  /** 刷新检测结果。 */
  refresh: () => Promise<void>
  /** 触发子页面重新加载（刷新后调用）。 */
  reload: () => void
}

const KEY = 'k8s-conn-context'

/** 在 K8sLayout 中提供连接上下文。 */
export function provideK8sConn(ctx: K8sConnContext): void {
  provide(KEY, ctx)
}

/** 在子页面中获取连接上下文。 */
export function useK8sConn(): K8sConnContext {
  const ctx = inject<K8sConnContext>(KEY)
  if (!ctx) {
    throw new Error('useK8sConn must be used within K8sLayout')
  }
  return ctx
}

/** 根据集群状态生成显示名。 */
export function k8sLabel(st: K8sStatus | null): string {
  if (!st) return '未连接'
  if (!st.available) return '不可用'
  if (st.version) return `Kubernetes ${st.version}`
  return 'Kubernetes'
}
