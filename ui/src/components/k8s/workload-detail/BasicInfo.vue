<script setup lang="ts">
/**
 * 工作负载基本信息组件。
 * 展示名称、命名空间、创建时间、注解、选择器、策略、状态、镜像等。
 */
import { computed } from 'vue'
import Badge from '@/components/ui/Badge.vue'
import { workloadReadyVariant } from '@/utils/k8s'
import { extractContainers, type WorkloadBasicInfo, type WorkloadKind, type WorkloadContainer } from './types'

const props = defineProps<{
  /** 基本信息数据。 */
  info: WorkloadBasicInfo
  /** 工作负载类型。 */
  kind: WorkloadKind
  /** 原始对象（用于提取容器/镜像信息）。 */
  rawObject?: Record<string, unknown> | null
}>()

/** 从原始对象提取容器列表。 */
const containers = computed<WorkloadContainer[]>(() => {
  if (!props.rawObject) return []
  return extractContainers(props.rawObject)
})

/** 镜像列表（去重）。 */
const images = computed(() => {
  const imgs = containers.value.map((c) => c.image).filter(Boolean)
  return [...new Set(imgs)]
})

function entries(obj: Record<string, string> | undefined): [string, string][] {
  return Object.entries(obj ?? {})
}

/** 格式化时间。 */
function fmtTime(ts: string): string {
  if (!ts) return '—'
  // K8s 时间格式如 2024-01-15T08:30:00Z
  const d = new Date(ts)
  if (isNaN(d.getTime())) return ts
  return d.toLocaleString('zh-CN', { hour12: false })
}
</script>

<template>
  <div class="space-y-4">
    <!-- 状态摘要：横向卡片 -->
    <div class="grid grid-cols-2 gap-2 sm:grid-cols-4 lg:grid-cols-6">
      <div class="rounded-lg border border-slate-200 bg-white px-3 py-2.5 dark:border-slate-700 dark:bg-slate-800/50">
        <p class="text-xs text-slate-400">就绪</p>
        <div class="mt-0.5"><Badge :variant="workloadReadyVariant(info.status.ready)" dot>{{ info.status.ready }}</Badge></div>
      </div>
      <div class="rounded-lg border border-slate-200 bg-white px-3 py-2.5 dark:border-slate-700 dark:bg-slate-800/50">
        <p class="text-xs text-slate-400">已更新</p>
        <p class="mt-0.5 text-sm font-semibold text-slate-700 dark:text-slate-200">{{ info.status.up_to_date }}</p>
      </div>
      <div class="rounded-lg border border-slate-200 bg-white px-3 py-2.5 dark:border-slate-700 dark:bg-slate-800/50">
        <p class="text-xs text-slate-400">可用</p>
        <p class="mt-0.5 text-sm font-semibold text-slate-700 dark:text-slate-200">{{ info.status.available }}</p>
      </div>
      <div class="rounded-lg border border-slate-200 bg-white px-3 py-2.5 dark:border-slate-700 dark:bg-slate-800/50">
        <p class="text-xs text-slate-400">期望副本</p>
        <p class="mt-0.5 text-sm font-semibold text-slate-700 dark:text-slate-200">{{ info.status.desired }}</p>
      </div>
      <div class="rounded-lg border border-slate-200 bg-white px-3 py-2.5 dark:border-slate-700 dark:bg-slate-800/50">
        <p class="text-xs text-slate-400">当前副本</p>
        <p class="mt-0.5 text-sm font-semibold text-slate-700 dark:text-slate-200">{{ info.status.replicas }}</p>
      </div>
      <div class="rounded-lg border border-slate-200 bg-white px-3 py-2.5 dark:border-slate-700 dark:bg-slate-800/50">
        <p class="text-xs text-slate-400">更新策略</p>
        <p class="mt-0.5 text-sm font-semibold text-slate-700 dark:text-slate-200">{{ info.strategy }}</p>
      </div>
    </div>

    <!-- 基本信息：键值对 -->
    <div class="grid grid-cols-1 gap-2 sm:grid-cols-3">
      <div class="flex items-center gap-2 rounded-lg bg-slate-50 px-3 py-2 dark:bg-slate-700/40">
        <span class="text-xs text-slate-400">名称</span>
        <span class="text-sm font-medium text-slate-700 dark:text-slate-200">{{ info.name }}</span>
      </div>
      <div class="flex items-center gap-2 rounded-lg bg-slate-50 px-3 py-2 dark:bg-slate-700/40">
        <span class="text-xs text-slate-400">命名空间</span>
        <span class="text-sm font-medium text-slate-700 dark:text-slate-200">{{ info.namespace }}</span>
      </div>
      <div class="flex items-center gap-2 rounded-lg bg-slate-50 px-3 py-2 dark:bg-slate-700/40">
        <span class="text-xs text-slate-400">创建时间</span>
        <span class="text-sm font-medium text-slate-700 dark:text-slate-200">{{ fmtTime(info.created_at) }}</span>
      </div>
    </div>

    <!-- 镜像 / 选择器 / 标签 / 注解：行式布局，不单独换行占块 -->
    <div class="space-y-1.5">
      <!-- 镜像 -->
      <div v-if="images.length > 0" class="flex items-start gap-2">
        <span class="mt-0.5 w-20 shrink-0 text-xs font-medium text-slate-400">镜像</span>
        <div class="flex flex-wrap gap-1.5">
          <span
            v-for="img in images"
            :key="img"
            class="rounded-md bg-indigo-50 px-2 py-0.5 font-mono text-xs text-indigo-600 dark:bg-indigo-900/40 dark:text-indigo-300"
          >{{ img }}</span>
        </div>
      </div>

      <!-- 选择器 -->
      <div class="flex items-start gap-2">
        <span class="mt-0.5 w-20 shrink-0 text-xs font-medium text-slate-400">选择器</span>
        <div v-if="entries(info.selector).length" class="flex flex-wrap gap-1.5">
          <span
            v-for="([k, v]) in entries(info.selector)"
            :key="k"
            class="rounded-md bg-blue-50 px-2 py-0.5 font-mono text-xs text-blue-600 dark:bg-blue-900/40 dark:text-blue-300"
          >{{ k }}={{ v }}</span>
        </div>
        <span v-else class="text-xs text-slate-400">无</span>
      </div>

      <!-- 标签 -->
      <div class="flex items-start gap-2">
        <span class="mt-0.5 w-20 shrink-0 text-xs font-medium text-slate-400">标签</span>
        <div v-if="entries(info.labels).length" class="flex flex-wrap gap-1.5">
          <span
            v-for="([k, v]) in entries(info.labels)"
            :key="k"
            class="rounded-md bg-slate-100 px-2 py-0.5 font-mono text-xs text-slate-600 dark:bg-slate-700 dark:text-slate-300"
          >{{ k }}={{ v }}</span>
        </div>
        <span v-else class="text-xs text-slate-400">无</span>
      </div>

      <!-- 注解 -->
      <div v-if="entries(info.annotations).length > 0" class="flex items-start gap-2">
        <span class="mt-0.5 w-20 shrink-0 text-xs font-medium text-slate-400">注解</span>
        <div class="flex flex-wrap gap-1.5">
          <span
            v-for="([k, v]) in entries(info.annotations)"
            :key="k"
            class="rounded-md bg-slate-100 px-2 py-0.5 font-mono text-xs text-slate-600 dark:bg-slate-700 dark:text-slate-300"
            :title="v"
          >{{ k }}={{ v.length > 60 ? v.slice(0, 60) + '...' : v }}</span>
        </div>
      </div>
    </div>
  </div>
</template>
