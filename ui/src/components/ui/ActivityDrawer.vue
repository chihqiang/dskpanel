<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import {
  Bell, X, CheckCheck, Trash2, CheckCircle2, XCircle, Info, AlertTriangle,
  Container, Image, Network, HardDrive, Server, Boxes, Activity,
} from '@lucide/vue'
import { storeToRefs } from 'pinia'
import { useActivityStore, type ActivityType } from '@/stores/activity'
import { useDockerEventsStore, type EventKind } from '@/stores/dockerEvents'
import { dockerEventsStream } from '@/api/docker'
import { useFocusTrap } from '@/composables/useFocusTrap'
import { fmtRelativeTime } from '@/utils/format'

const store = useActivityStore()
const { sorted, unread } = storeToRefs(store)

const events = useDockerEventsStore()
const { sorted: eventSorted, unread: eventUnread } = storeToRefs(events)

const open = ref(false)
const panel = ref<HTMLElement | null>(null)
useFocusTrap(panel, () => open.value)

/** 当前 Tab：my = 我的操作，system = 系统事件。 */
const tab = ref<'my' | 'system'>('my')

/** 系统事件 SSE 取消函数。 */
let stopEvents: (() => void) | null = null

onMounted(() => {
  // 全局订阅 Docker 系统事件。
  stopEvents = dockerEventsStream(
    (ev) => events.push(ev),
    () => {
      // 忽略（连接断开后组件重挂载会重连）
    },
  )
})
onBeforeUnmount(() => {
  stopEvents?.()
  stopEvents = null
})

/** 顶部铃铛总未读数 = 我的操作 + 系统事件。 */
const totalUnread = computed(() => unread.value + eventUnread.value)

const activityIcons: Record<ActivityType, typeof Bell> = {
  success: CheckCircle2,
  error: XCircle,
  info: Info,
  warning: AlertTriangle,
}
const activityColors: Record<ActivityType, string> = {
  success: 'text-emerald-500',
  error: 'text-red-500',
  info: 'text-blue-500',
  warning: 'text-amber-500',
}

/** 系统事件分类 → 图标。 */
const eventIcons: Record<EventKind, typeof Bell> = {
  container: Container,
  image: Image,
  network: Network,
  volume: HardDrive,
  daemon: Server,
  swarm: Boxes,
  other: Activity,
}
/** 系统事件分类 → 颜色。 */
const eventColors: Record<EventKind, string> = {
  container: 'text-blue-500',
  image: 'text-purple-500',
  network: 'text-cyan-500',
  volume: 'text-amber-500',
  daemon: 'text-slate-500',
  swarm: 'text-green-500',
  other: 'text-slate-400',
}

function toggle(): void {
  open.value = !open.value
  if (open.value) {
    store.markAllRead()
    events.markAllRead()
  }
}

function switchTab(t: 'my' | 'system'): void {
  tab.value = t
  // 切到系统事件时清其未读。
  if (t === 'system') events.markAllRead()
  else store.markAllRead()
}

/** 事件相对时间：Docker 事件 time 为秒。 */
function evTime(sec: number): string {
  return fmtRelativeTime(sec * 1000)
}
</script>

<template>
  <!-- 铃铛按钮 -->
  <button
    class="relative flex h-9 w-9 items-center justify-center rounded-lg text-slate-500 transition-colors hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-700"
    :aria-label="`通知${totalUnread > 0 ? `，${totalUnread} 条未读` : ''}`"
    @click="toggle"
  >
    <Bell class="h-5 w-5" />
    <span
      v-if="totalUnread > 0"
      class="absolute -right-0.5 -top-0.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-red-500 px-1 text-xs font-medium text-white"
    >
      {{ totalUnread > 99 ? '99+' : totalUnread }}
    </span>
  </button>

  <!-- 抽屉 -->
  <Teleport to="body">
    <Transition name="drawer-fade">
      <div v-if="open" class="fixed inset-0 z-50">
        <!-- 遮罩 -->
        <div class="absolute inset-0 bg-black/40" @click="open = false" />
        <!-- 面板 -->
        <div
          ref="panel"
          tabindex="-1"
          class="absolute right-0 top-0 flex h-full w-full max-w-md flex-col bg-white shadow-2xl outline-none dark:bg-slate-800"
          role="dialog"
          aria-modal="true"
          aria-label="通知"
        >
          <!-- 头部 -->
          <div class="flex shrink-0 items-center justify-between border-b border-slate-200 px-5 py-4 dark:border-slate-700">
            <div class="flex items-center gap-2">
              <Bell class="h-5 w-5 text-blue-600 dark:text-blue-400" />
              <h3 class="text-base font-semibold text-slate-800 dark:text-slate-100">通知</h3>
            </div>
            <button
              class="text-slate-400 transition-colors hover:text-slate-600 dark:hover:text-slate-200"
              aria-label="关闭"
              @click="open = false"
            >
              <X class="h-5 w-5" />
            </button>
          </div>

          <!-- Tab 切换 -->
          <div class="flex shrink-0 gap-1 border-b border-slate-200 px-4 pt-3 dark:border-slate-700">
            <button
              class="border-b-2 px-3 py-2 text-sm font-medium transition-colors"
              :class="tab === 'my' ? 'border-blue-500 text-blue-600 dark:text-blue-400' : 'border-transparent text-slate-500 hover:text-slate-700 dark:hover:text-slate-300'"
              @click="switchTab('my')"
            >
              我的操作
              <span v-if="unread" class="ml-1 rounded-full bg-red-100 px-1.5 text-xs text-red-600 dark:bg-red-900 dark:text-red-300">{{ unread }}</span>
            </button>
            <button
              class="border-b-2 px-3 py-2 text-sm font-medium transition-colors"
              :class="tab === 'system' ? 'border-blue-500 text-blue-600 dark:text-blue-400' : 'border-transparent text-slate-500 hover:text-slate-700 dark:hover:text-slate-300'"
              @click="switchTab('system')"
            >
              系统事件
              <span v-if="eventUnread" class="ml-1 rounded-full bg-red-100 px-1.5 text-xs text-red-600 dark:bg-red-900 dark:text-red-300">{{ eventUnread }}</span>
            </button>
          </div>

          <!-- 我的操作 -->
          <template v-if="tab === 'my'">
            <!-- 操作栏 -->
            <div v-if="sorted.length > 0" class="flex shrink-0 items-center gap-2 border-b border-slate-100 px-5 py-2 dark:border-slate-700">
              <button
                class="flex items-center gap-1.5 text-xs text-slate-500 transition-colors hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200"
                @click="store.markAllRead()"
              >
                <CheckCheck class="h-3.5 w-3.5" />
                全部已读
              </button>
              <button
                class="ml-auto flex items-center gap-1.5 text-xs text-slate-500 transition-colors hover:text-red-600 dark:text-slate-400"
                @click="store.clear()"
              >
                <Trash2 class="h-3.5 w-3.5" />
                清空
              </button>
            </div>

            <!-- 列表 -->
            <div class="flex-1 overflow-y-auto">
              <div v-if="sorted.length === 0" class="flex flex-col items-center justify-center gap-3 py-20 text-slate-400">
                <Bell class="h-10 w-10" />
                <p class="text-sm">暂无活动记录</p>
              </div>
              <ul v-else class="divide-y divide-slate-100 dark:divide-slate-700">
                <li
                  v-for="item in sorted"
                  :key="'a' + item.id"
                  class="flex items-start gap-3 px-5 py-3.5 transition-colors hover:bg-slate-50 dark:hover:bg-slate-700/40"
                >
                  <component :is="activityIcons[item.type]" class="mt-0.5 h-5 w-5 shrink-0" :class="activityColors[item.type]" />
                  <div class="min-w-0 flex-1">
                    <p class="text-sm text-slate-700 dark:text-slate-200">{{ item.message }}</p>
                    <div class="mt-0.5 flex items-center gap-2 text-xs text-slate-400">
                      <span v-if="item.target" class="truncate font-mono">{{ item.target }}</span>
                      <span>{{ fmtRelativeTime(item.time) }}</span>
                    </div>
                  </div>
                </li>
              </ul>
            </div>
          </template>

          <!-- 系统事件 -->
          <template v-else>
            <!-- 操作栏 -->
            <div v-if="eventSorted.length > 0" class="flex shrink-0 items-center gap-2 border-b border-slate-100 px-5 py-2 dark:border-slate-700">
              <button
                class="flex items-center gap-1.5 text-xs text-slate-500 transition-colors hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200"
                @click="events.markAllRead()"
              >
                <CheckCheck class="h-3.5 w-3.5" />
                全部已读
              </button>
              <button
                class="ml-auto flex items-center gap-1.5 text-xs text-slate-500 transition-colors hover:text-red-600 dark:text-slate-400"
                @click="events.clear()"
              >
                <Trash2 class="h-3.5 w-3.5" />
                清空
              </button>
            </div>

            <!-- 列表 -->
            <div class="flex-1 overflow-y-auto">
              <div v-if="eventSorted.length === 0" class="flex flex-col items-center justify-center gap-3 py-20 text-slate-400">
                <Activity class="h-10 w-10" />
                <p class="text-sm">等待 Docker 事件…</p>
                <p class="text-xs">容器启停、镜像拉取、网络/卷变化等实时推送</p>
              </div>
              <ul v-else class="divide-y divide-slate-100 dark:divide-slate-700">
                <li
                  v-for="item in eventSorted"
                  :key="'e' + item.id"
                  class="flex items-start gap-3 px-5 py-3.5 transition-colors hover:bg-slate-50 dark:hover:bg-slate-700/40"
                >
                  <component :is="eventIcons[item.kind]" class="mt-0.5 h-5 w-5 shrink-0" :class="eventColors[item.kind]" />
                  <div class="min-w-0 flex-1">
                    <p class="text-sm text-slate-700 dark:text-slate-200">
                      <span class="font-medium">{{ item.type }}</span>
                      <span class="text-slate-400"> {{ item.actionText }}</span>
                    </p>
                    <div class="mt-0.5 flex items-center gap-2 text-xs text-slate-400">
                      <span class="truncate font-mono">{{ item.actor }}</span>
                      <span>{{ evTime(item.time) }}</span>
                    </div>
                  </div>
                </li>
              </ul>
            </div>
          </template>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.drawer-fade-enter-active,
.drawer-fade-leave-active {
  transition: opacity 0.2s ease;
}
.drawer-fade-enter-from,
.drawer-fade-leave-to {
  opacity: 0;
}
</style>
