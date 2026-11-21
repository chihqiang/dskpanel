<script setup lang="ts">
/**
 * 资源关联事件 Tab：在 Pod / Deployment 等详情弹窗中展示与该资源关联的 K8s Events。
 */
import { ref, watch } from 'vue'
import { AlertTriangle, Info, RefreshCw } from '@lucide/vue'
import Badge from '@/components/ui/Badge.vue'
import Button from '@/components/ui/Button.vue'
import { eventTypeVariant } from '@/utils/k8s'
import { k8sResourceEvents, type K8sEventItem } from '@/api/k8s'

const props = defineProps<{
  /** 资源类型，如 Pod / Deployment / StatefulSet。 */
  kind: string
  /** 资源名称。 */
  name: string
  /** 命名空间。 */
  namespace: string
  /** 是否激活（仅在激活时加载数据，避免隐藏 Tab 预请求）。 */
  active: boolean
}>()

const loading = ref(false)
const errorMsg = ref('')
const items = ref<K8sEventItem[]>([])

async function load(): Promise<void> {
  if (!props.name || !props.namespace) return
  loading.value = true
  errorMsg.value = ''
  try {
    items.value = await k8sResourceEvents(props.kind, props.name, props.namespace)
  } catch (err) {
    errorMsg.value = (err as Error).message
    items.value = []
  } finally {
    loading.value = false
  }
}

// 当 Tab 被激活时自动加载。
watch(
  () => [props.active, props.name, props.namespace] as const,
  ([active, name, ns]) => {
    if (active && name && ns) {
      void load()
    }
  },
  { immediate: true },
)
</script>

<template>
  <div class="space-y-3">
    <!-- 头部 -->
    <div class="flex items-center justify-between">
      <p class="text-sm text-slate-500 dark:text-slate-400">
        关联事件（{{ kind }}/{{ name }}）
      </p>
      <Button variant="ghost" size="sm" :loading="loading" @click="load">
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

    <!-- 事件列表 -->
    <div v-else-if="items.length > 0" class="max-h-80 space-y-2 overflow-y-auto">
      <div
        v-for="(evt, idx) in items"
        :key="idx"
        class="flex items-start gap-3 rounded-lg border border-slate-100 px-3 py-2 dark:border-slate-700"
      >
        <Badge :variant="eventTypeVariant(evt.type)" dot class="mt-0.5 shrink-0">
          {{ evt.type }}
        </Badge>
        <div class="min-w-0 flex-1">
          <div class="flex flex-wrap items-center gap-2">
            <span class="inline-flex items-center gap-1 text-sm font-medium text-slate-700 dark:text-slate-200">
              <Info v-if="evt.type !== 'Warning'" class="h-3.5 w-3.5 text-blue-500" />
              <AlertTriangle v-else class="h-3.5 w-3.5 text-red-500" />
              {{ evt.reason }}
            </span>
            <span class="text-xs text-slate-400">×{{ evt.count }}</span>
            <span class="text-xs text-slate-400">{{ evt.last_time }}</span>
          </div>
          <p class="mt-0.5 text-xs text-slate-500 dark:text-slate-400">{{ evt.message }}</p>
        </div>
      </div>
    </div>

    <!-- 空 -->
    <div v-else class="py-8 text-center text-sm text-slate-400">
      该资源暂无关联事件
    </div>
  </div>
</template>
