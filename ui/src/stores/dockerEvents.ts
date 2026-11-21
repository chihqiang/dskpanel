import { defineStore } from 'pinia'

export type EventKind = 'container' | 'image' | 'network' | 'volume' | 'daemon' | 'swarm' | 'other'

export interface DockerEventItem {
  id: number
  kind: EventKind
  type: string
  action: string
  /** 可读动作文案（如「已启动」）。 */
  actionText: string
  /** 关联资源名（actor 名称）。 */
  actor: string
  /** 时间戳（秒）。 */
  time: number
}

const MAX_ITEMS = 300
let seq = 0

/** 事件类型 → 分类（决定图标/颜色）。 */
function kindOf(type: string): EventKind {
  switch (type) {
    case 'container':
      return 'container'
    case 'image':
      return 'image'
    case 'network':
      return 'network'
    case 'volume':
      return 'volume'
    case 'daemon':
      return 'daemon'
    case 'service':
    case 'secret':
    case 'config':
    case 'node':
    case 'task':
      return 'swarm'
    default:
      return 'other'
  }
}

/** 动作 → 可读文案。 */
const ACTION_TEXT: Record<string, string> = {
  start: '已启动',
  die: '已停止',
  stop: '已停止',
  kill: '已被终止',
  create: '已创建',
  destroy: '已删除',
  delete: '已删除',
  remove: '已移除',
  rename: '已重命名',
  pause: '已暂停',
  unpause: '已恢复',
  restart: '已重启',
  pull: '已拉取',
  push: '已推送',
  tag: '已打标签',
  untag: '已移除标签',
  commit: '已提交',
  mount: '已挂载',
  unmount: '已卸载',
  connect: '已连接',
  disconnect: '已断开',
  reload: '已重载',
  exec_create: '创建 Exec',
  exec_start: '启动 Exec',
  attach: '已附加',
  export: '已导出',
  import: '已导入',
  load: '已加载',
  save: '已保存',
  prune: '已清理',
  update: '已更新',
  resize: '已调整',
  copy: '已复制',
  oom: 'OOM 终止',
  health_status: '健康检查',
}

function actionText(action: string): string {
  return ACTION_TEXT[action] ?? action
}

/** 取 actor 可读名称：优先 name 属性，其次 id 缩写。 */
function actorName(id: string, attrs?: Record<string, string>): string {
  const name = attrs?.name
  if (name) return name
  const short = id.replace('sha256:', '')
  return short.length > 16 ? `${short.slice(0, 16)}…` : short
}

/** 全局 Docker 系统事件 store（SSE 实时订阅，独立于用户操作日志）。 */
export const useDockerEventsStore = defineStore('dockerEvents', {
  state: () => ({
    items: [] as DockerEventItem[],
    /** 未读数。 */
    unread: 0,
  }),
  getters: {
    sorted(): DockerEventItem[] {
      return [...this.items].reverse()
    },
  },
  actions: {
    push(raw: { type: string; action: string; actor_id: string; actor_attr?: Record<string, string>; time: number }) {
      const id = ++seq
      this.items.push({
        id,
        kind: kindOf(raw.type),
        type: raw.type,
        action: raw.action,
        actionText: actionText(raw.action),
        actor: actorName(raw.actor_id, raw.actor_attr),
        time: raw.time,
      })
      if (this.items.length > MAX_ITEMS) {
        this.items = this.items.slice(-MAX_ITEMS)
      }
      this.unread++
    },
    markAllRead() {
      this.unread = 0
    },
    clear() {
      this.items = []
      this.unread = 0
    },
  },
})
