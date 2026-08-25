<script setup lang="ts">
/**
 * 全局命令面板：Cmd/Ctrl+K 打开，可快速搜索并跳转到各功能页面。
 */
import { ref, computed, watch, nextTick, onMounted, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import { Search, CornerDownLeft } from '@lucide/vue'
import { useShortcut } from '@/composables/useShortcut'

const router = useRouter()

const open = ref(false)
const keyword = ref('')
const selectedIndex = ref(0)
const inputRef = ref<HTMLInputElement | null>(null)

/** 外部触发打开（如顶栏搜索按钮 emit 自定义事件）。 */
function onOpenRequest(): void {
  if (!open.value) toggleOpen()
}

onMounted(() => {
  window.addEventListener('open-command-palette', onOpenRequest)
})

onBeforeUnmount(() => {
  window.removeEventListener('open-command-palette', onOpenRequest)
})

/** 可跳转的页面列表。 */
interface CommandItem {
  label: string
  path: string
  group: string
  keywords?: string
}

const commands: CommandItem[] = [
  // Docker
  { label: 'Docker 概览', path: '/docker/overview', group: 'Docker', keywords: 'dashboard overview' },
  { label: '容器列表', path: '/docker/containers', group: 'Docker', keywords: 'container list' },
  { label: '镜像列表', path: '/docker/images', group: 'Docker', keywords: 'image build pull' },
  { label: '网络管理', path: '/docker/networks', group: 'Docker', keywords: 'network' },
  { label: '卷管理', path: '/docker/volumes', group: 'Docker', keywords: 'volume storage' },
  { label: 'Compose 编排', path: '/docker/compose', group: 'Docker', keywords: 'compose deploy' },
  // Swarm
  { label: 'Swarm 概览', path: '/swarm/overview', group: 'Swarm', keywords: 'dashboard' },
  { label: 'Swarm 节点', path: '/swarm/nodes', group: 'Swarm', keywords: 'node' },
  { label: 'Swarm 服务', path: '/swarm/services', group: 'Swarm', keywords: 'service stack' },
  { label: 'Swarm 任务', path: '/swarm/tasks', group: 'Swarm', keywords: 'task container' },
  { label: 'Swarm 网络', path: '/swarm/networks', group: 'Swarm', keywords: 'network' },
  { label: 'Swarm Secret', path: '/swarm/secrets', group: 'Swarm', keywords: 'secret config' },
  // K8s
  { label: 'K8s 概览', path: '/k8s/overview', group: 'Kubernetes', keywords: 'dashboard cluster' },
  { label: 'K8s 节点', path: '/k8s/nodes', group: 'Kubernetes', keywords: 'node' },
  { label: 'K8s 命名空间', path: '/k8s/namespaces', group: 'Kubernetes', keywords: 'namespace ns' },
  { label: 'K8s Pod', path: '/k8s/pods', group: 'Kubernetes', keywords: 'pod' },
  { label: 'K8s 工作负载', path: '/k8s/workloads', group: 'Kubernetes', keywords: 'deployment statefulset daemonset job cronjob' },
  { label: 'K8s 服务', path: '/k8s/services', group: 'Kubernetes', keywords: 'service ingress' },
  { label: 'K8s 配置', path: '/k8s/config', group: 'Kubernetes', keywords: 'configmap secret' },
  { label: 'K8s 存储', path: '/k8s/storage', group: 'Kubernetes', keywords: 'pvc storageclass volume' },
  { label: 'K8s RBAC', path: '/k8s/rbac', group: 'Kubernetes', keywords: 'role clusterrole rolebinding rbac' },
  { label: 'K8s HPA', path: '/k8s/hpa', group: 'Kubernetes', keywords: 'hpa autoscaler autoscaling scale' },
  { label: 'K8s 事件', path: '/k8s/events', group: 'Kubernetes', keywords: 'event log' },
]

/** 按关键词过滤。 */
const filtered = computed<CommandItem[]>(() => {
  const kw = keyword.value.trim().toLowerCase()
  if (!kw) return commands
  return commands.filter(
    (c) =>
      c.label.toLowerCase().includes(kw) ||
      c.group.toLowerCase().includes(kw) ||
      (c.keywords?.toLowerCase().includes(kw) ?? false),
  )
})

/** 选中项随过滤结果变化时重置。 */
watch(filtered, () => {
  selectedIndex.value = 0
})

/** 打开/关闭。 */
function toggleOpen(): void {
  open.value = !open.value
  if (open.value) {
    keyword.value = ''
    selectedIndex.value = 0
    nextTick(() => inputRef.value?.focus())
  }
}

/** 选择并跳转。 */
function select(item: CommandItem): void {
  router.push(item.path)
  open.value = false
}

/** 键盘导航。 */
function onKeydown(e: KeyboardEvent): void {
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    selectedIndex.value = Math.min(selectedIndex.value + 1, filtered.value.length - 1)
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    selectedIndex.value = Math.max(selectedIndex.value - 1, 0)
  } else if (e.key === 'Enter') {
    e.preventDefault()
    const item = filtered.value[selectedIndex.value]
    if (item) select(item)
  }
}

// Cmd/Ctrl+K 打开。
useShortcut('k', () => toggleOpen(), { meta: true, ctrl: true, ignoreInputs: false })
// Escape 关闭。
useShortcut('escape', () => { open.value = false }, { ignoreInputs: false })
</script>

<template>
  <Teleport to="body">
    <Transition name="cmd-fade">
      <div v-if="open" class="fixed inset-0 z-[100] flex items-start justify-center pt-[15vh]" @click.self="open = false">
        <!-- 遮罩 -->
        <div class="absolute inset-0 bg-black/40 backdrop-blur-sm" />

        <!-- 面板 -->
        <div class="relative w-full max-w-xl overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-2xl dark:border-slate-700 dark:bg-slate-800">
          <!-- 搜索框 -->
          <div class="flex items-center gap-3 border-b border-slate-200 px-4 py-3 dark:border-slate-700">
            <Search class="h-5 w-5 shrink-0 text-slate-400" />
            <input
              ref="inputRef"
              v-model="keyword"
              type="text"
              class="flex-1 bg-transparent text-sm text-slate-800 outline-none placeholder:text-slate-400 dark:text-slate-100"
              placeholder="搜索页面或功能…"
              @keydown="onKeydown"
            />
            <kbd class="hidden shrink-0 rounded border border-slate-200 px-1.5 py-0.5 text-xs text-slate-400 dark:border-slate-600 sm:inline">ESC</kbd>
          </div>

          <!-- 结果列表 -->
          <div class="max-h-[50vh] overflow-y-auto p-2">
            <div v-if="filtered.length === 0" class="py-8 text-center text-sm text-slate-400">
              无匹配结果
            </div>
            <button
              v-for="(item, idx) in filtered"
              :key="item.path"
              class="flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-left text-sm transition-colors"
              :class="idx === selectedIndex
                ? 'bg-blue-50 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300'
                : 'text-slate-700 hover:bg-slate-100 dark:text-slate-200 dark:hover:bg-slate-700'"
              @click="select(item)"
              @mouseenter="selectedIndex = idx"
            >
              <span class="shrink-0 text-xs font-medium text-slate-400">{{ item.group }}</span>
              <span class="flex-1 truncate">{{ item.label }}</span>
              <CornerDownLeft v-if="idx === selectedIndex" class="h-3.5 w-3.5 shrink-0 text-slate-400" />
            </button>
          </div>

          <!-- 底部提示 -->
          <div class="flex items-center justify-between border-t border-slate-200 px-4 py-2 text-xs text-slate-400 dark:border-slate-700">
            <span>↑↓ 导航 · Enter 跳转</span>
            <span>共 {{ filtered.length }} 个结果</span>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.cmd-fade-enter-active,
.cmd-fade-leave-active {
  transition: opacity 0.15s ease;
}
.cmd-fade-enter-from,
.cmd-fade-leave-to {
  opacity: 0;
}
</style>
