import { defineStore } from 'pinia'

export type ActivityType = 'success' | 'error' | 'info' | 'warning'

export interface ActivityItem {
  id: number
  type: ActivityType
  message: string
  /** 关联资源（如容器名、镜像名）。 */
  target?: string
  /** 时间戳（毫秒）。 */
  time: number
}

const MAX_ITEMS = 200
let seq = 0

/** 全局活动日志 store（pinia）。 */
export const useActivityStore = defineStore('activity', {
  state: () => ({
    items: [] as ActivityItem[],
    /** 未读数。 */
    unread: 0,
  }),
  getters: {
    /** 倒序排列（最新在前）。 */
    sorted(): ActivityItem[] {
      return [...this.items].reverse()
    },
  },
  actions: {
    log(type: ActivityType, message: string, target?: string) {
      const id = ++seq
      this.items.push({ id, type, message, target, time: Date.now() })
      // 限制最多 MAX_ITEMS 条，超出删除最早的。
      if (this.items.length > MAX_ITEMS) {
        this.items = this.items.slice(-MAX_ITEMS)
      }
      this.unread++
    },
    success(message: string, target?: string) {
      this.log('success', message, target)
    },
    error(message: string, target?: string) {
      this.log('error', message, target)
    },
    info(message: string, target?: string) {
      this.log('info', message, target)
    },
    warning(message: string, target?: string) {
      this.log('warning', message, target)
    },
    /** 标记全部已读。 */
    markAllRead() {
      this.unread = 0
    },
    /** 清空全部日志。 */
    clear() {
      this.items = []
      this.unread = 0
    },
  },
})
