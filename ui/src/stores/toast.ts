import { defineStore } from 'pinia'

export type ToastType = 'success' | 'error' | 'info'

export interface ToastItem {
  id: number
  type: ToastType
  message: string
  /** 撤销按钮文案（如有 undo 回调则显示）。 */
  undoLabel?: string
  /** 撤销回调。 */
  undo?: () => void
  /** 自动消失延时 ms（0 = 不自动消失）。 */
  duration?: number
}

const DEFAULT_DURATION = 3200
const UNDO_DURATION = 5000
let seq = 0

/** 全局消息弹出层 store（pinia）。 */
export const useToastStore = defineStore('toast', {
  state: () => ({
    toasts: [] as ToastItem[],
  }),
  actions: {
    push(type: ToastType, message: string, duration = DEFAULT_DURATION) {
      const id = ++seq
      this.toasts.push({ id, type, message, duration })
      if (duration > 0) {
        window.setTimeout(() => this.remove(id), duration)
      }
    },
    success(message: string, duration?: number) {
      this.push('success', message, duration)
    },
    error(message: string, duration?: number) {
      this.push('error', message, duration ?? DEFAULT_DURATION)
    },
    info(message: string, duration?: number) {
      this.push('info', message, duration)
    },
    /**
     * 推送带撤销按钮的 toast。
     * @param message  提示文案
     * @param undo     撤销回调
     * @param undoLabel 撤销按钮文案，默认"撤销"
     */
    pushUndo(message: string, undo: () => void, undoLabel = '撤销') {
      const id = ++seq
      this.toasts.push({ id, type: 'info', message, undo, undoLabel, duration: UNDO_DURATION })
      window.setTimeout(() => this.remove(id), UNDO_DURATION)
    },
    /** 执行撤销并移除该 toast。 */
    undo(id: number) {
      const t = this.toasts.find((x) => x.id === id)
      if (t?.undo) {
        t.undo()
      }
      this.remove(id)
    },
    remove(id: number) {
      this.toasts = this.toasts.filter((t) => t.id !== id)
    },
    /** 清除全部。 */
    clear() {
      this.toasts = []
    },
  },
})
