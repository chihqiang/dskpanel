<script setup lang="ts">
/**
 * 容器组 Tab：展示工作负载关联的 Pod 列表。
 * 通过 labelSelector 查询关联 Pod，支持点击 Pod 名打开日志和终端。
 */
import { ref, watch, onMounted } from 'vue'
import { FileText, Terminal, RefreshCw } from '@lucide/vue'
import Badge from '@/components/ui/Badge.vue'
import Button from '@/components/ui/Button.vue'
import { podPhaseVariant } from '@/utils/k8s'
import { k8sPods, type K8sPodItem } from '@/api/k8s'

const props = defineProps<{
  /** 命名空间。 */
  namespace: string
  /** label selector 字符串（如 app=nginx）。 */
  labelSelector: string
  /** 是否激活（仅激活时加载数据）。 */
  active: boolean
}>()

const emit = defineEmits<{
  /** 打开 Pod 日志。 */
  logs: [pod: K8sPodItem]
  /** 打开 Pod 终端。 */
  terminal: [pod: K8sPodItem]
}>()

const loading = ref(false)
const errorMsg = ref('')
const pods = ref<K8sPodItem[]>([])
/** 标记是否已加载过，避免重复加载。 */
let loaded = false

async function load(): Promise<void> {
  if (!props.namespace || !props.labelSelector) return
  loading.value = true
  errorMsg.value = ''
  try {
    pods.value = await k8sPods({ namespace: props.namespace, labelSelector: props.labelSelector })
    loaded = true
  } catch (err) {
    errorMsg.value = (err as Error).message
    pods.value = []
  } finally {
    loading.value = false
  }
}

/** 强制刷新。 */
function refresh(): void {
  loaded = false
  void load()
}

// 监听 labelSelector 变化（rawData 加载完成后会从空字符串变为实际值）。
watch(
  () => props.labelSelector,
  (sel) => {
    if (props.active && sel && !loaded) {
      void load()
    }
  },
)

// 监听 active 变化（Tab 切换回来时重新加载）。
watch(
  () => props.active,
  (active) => {
    if (active && props.namespace && props.labelSelector && !loaded) {
      void load()
    }
  },
)

// 组件挂载后尝试加载（父组件确保 rawData 已加载后才渲染子组件）。
onMounted(() => {
  if (props.active && props.namespace && props.labelSelector) {
    void load()
  }
})
</script>

<template>
  <div class="space-y-3">
    <!-- 头部 -->
    <div class="flex items-center justify-between">
      <p class="text-sm text-slate-500 dark:text-slate-400">
        关联 Pod（{{ pods.length }} 个）
      </p>
      <Button variant="ghost" size="sm" :loading="loading" @click="refresh">
        <RefreshCw class="h-3.5 w-3.5" />
        刷新
      </Button>
    </div>

    <!-- 加载中 -->
    <div v-if="loading" class="flex items-center justify-center py-8 text-sm text-slate-400">
      <svg class="mr-2 h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none" aria-hidden="true">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
      </svg>
      加载中…
    </div>

    <!-- 错误 -->
    <div v-else-if="errorMsg" class="rounded-lg bg-red-50 px-4 py-3 text-sm text-red-600 dark:bg-red-900/20 dark:text-red-300">
      {{ errorMsg }}
    </div>

    <!-- Pod 列表 -->
    <div v-else-if="pods.length > 0" class="space-y-2">
      <div
        v-for="pod in pods"
        :key="pod.name"
        class="rounded-lg border border-slate-200 px-3 py-2.5 dark:border-slate-700"
      >
        <div class="flex flex-wrap items-center gap-2">
          <span class="text-sm font-medium text-slate-700 dark:text-slate-200">{{ pod.name }}</span>
          <Badge :variant="podPhaseVariant(pod.status)" dot>{{ pod.status }}</Badge>
          <span class="text-xs text-slate-400">就绪 {{ pod.ready }}</span>
          <span class="text-xs text-slate-400">重启 {{ pod.restarts }}</span>
          <span class="text-xs text-slate-400">IP {{ pod.ip || '—' }}</span>
          <span class="text-xs text-slate-400">节点 {{ pod.node_name || '—' }}</span>
          <span class="text-xs text-slate-400">{{ pod.created_at }}</span>
          <div class="ml-auto flex items-center gap-1">
            <Button variant="ghost" size="sm" @click="emit('logs', pod)">
              <FileText class="h-3.5 w-3.5" />
              日志
            </Button>
            <Button variant="ghost" size="sm" :disabled="pod.status !== 'Running'" @click="emit('terminal', pod)">
              <Terminal class="h-3.5 w-3.5" />
              终端
            </Button>
          </div>
        </div>
        <p v-if="pod.image" class="mt-1 font-mono text-xs text-slate-400">{{ pod.image }}</p>
      </div>
    </div>

    <!-- 空 -->
    <div v-else class="py-8 text-center text-sm text-slate-400">
      暂无关联 Pod
    </div>
  </div>
</template>
