<script setup lang="ts">
/**
 * 历史版本 Tab：展示工作负载的 Pod 模板信息（容器镜像、端口、环境变量、资源限制等）。
 *
 * 注意：完整的 ReplicaSet 历史和回滚功能需要后端新增 ReplicaSet 查询和 rollout undo 接口，
 * 当前 Tab 先展示 Pod 模板详情作为版本信息参考。
 */
import { computed } from 'vue'
import Badge from '@/components/ui/Badge.vue'
import { extractContainers, type WorkloadContainer } from './types'

const props = defineProps<{
  /** 工作负载类型。 */
  kind: string
  /** 名称。 */
  name: string
  /** 命名空间。 */
  namespace: string
  /** 原始 K8s 对象（由父组件传入）。 */
  rawObject: Record<string, unknown> | null
  /** 是否激活。 */
  active: boolean
}>()

/** 从原始对象提取的容器列表。 */
const containers = computed<WorkloadContainer[]>(() => {
  if (!props.rawObject) return []
  return extractContainers(props.rawObject)
})

/** 从原始对象提取 restart 注解。 */
const restartAnnotation = computed(() => {
  if (!props.rawObject) return ''
  const spec = (props.rawObject.spec ?? {}) as Record<string, unknown>
  const template = (spec.template ?? {}) as Record<string, unknown>
  const meta = (template.metadata ?? {}) as Record<string, unknown>
  const annotations = (meta.annotations ?? {}) as Record<string, string>
  return annotations['kubectl.kubernetes.io/restartedAt'] ?? ''
})

/** 从原始对象提取 Pod 模板标签。 */
const templateLabels = computed<Record<string, string>>(() => {
  if (!props.rawObject) return {}
  const spec = (props.rawObject.spec ?? {}) as Record<string, unknown>
  const template = (spec.template ?? {}) as Record<string, unknown>
  const meta = (template.metadata ?? {}) as Record<string, unknown>
  return (meta.labels ?? {}) as Record<string, string>
})

function entries(obj: Record<string, string> | undefined): [string, string][] {
  return Object.entries(obj ?? {})
}
</script>

<template>
  <div class="space-y-3">
    <!-- 头部 -->
    <div class="flex items-center justify-between">
      <p class="text-sm text-slate-500 dark:text-slate-400">
        Pod 模板（{{ containers.length }} 个容器）
      </p>
      <div v-if="restartAnnotation" class="text-xs text-slate-400">
        最近重启：{{ restartAnnotation }}
      </div>
    </div>

    <!-- 容器模板列表 -->
    <div v-if="containers.length > 0" class="space-y-2">
      <div
        v-for="c in containers"
        :key="c.name"
        class="rounded-lg border border-slate-200 px-3 py-2.5 dark:border-slate-700"
      >
        <!-- 容器名 + 镜像 -->
        <div class="flex flex-wrap items-center gap-2">
          <span class="text-sm font-medium text-slate-700 dark:text-slate-200">{{ c.name }}</span>
          <Badge variant="blue">{{ c.image }}</Badge>
        </div>

        <!-- 端口 -->
        <div v-if="c.ports.length > 0" class="mt-2 flex flex-wrap items-center gap-2 text-xs">
          <span class="text-slate-400">端口：</span>
          <span
            v-for="p in c.ports"
            :key="`${p.containerPort}-${p.name}`"
            class="rounded bg-slate-100 px-1.5 py-0.5 font-mono text-slate-600 dark:bg-slate-700 dark:text-slate-300"
          >
            {{ p.containerPort }}{{ p.name ? `/${p.name}` : '' }}{{ p.protocol ? `/${p.protocol}` : '' }}
          </span>
        </div>

        <!-- 环境变量 -->
        <div v-if="c.env.length > 0" class="mt-2 flex flex-wrap items-center gap-2 text-xs">
          <span class="text-slate-400">环境变量：</span>
          <span
            v-for="e in c.env"
            :key="e.name"
            class="rounded bg-slate-100 px-1.5 py-0.5 font-mono text-slate-600 dark:bg-slate-700 dark:text-slate-300"
          >
            {{ e.name }}{{ e.value !== undefined ? `=${e.value}` : '' }}
          </span>
        </div>

        <!-- 资源限制 -->
        <div v-if="c.resources && (entries(c.resources.requests).length > 0 || entries(c.resources.limits).length > 0)" class="mt-2 flex flex-wrap items-center gap-4 text-xs">
          <div v-if="entries(c.resources.requests).length > 0" class="flex items-center gap-1.5">
            <span class="text-slate-400">请求：</span>
            <span
              v-for="([k, v]) in entries(c.resources.requests)"
              :key="`req-${k}`"
              class="rounded bg-amber-50 px-1.5 py-0.5 font-mono text-amber-600 dark:bg-amber-900/40 dark:text-amber-300"
            >
              {{ k }}={{ v }}
            </span>
          </div>
          <div v-if="entries(c.resources.limits).length > 0" class="flex items-center gap-1.5">
            <span class="text-slate-400">限制：</span>
            <span
              v-for="([k, v]) in entries(c.resources.limits)"
              :key="`lim-${k}`"
              class="rounded bg-red-50 px-1.5 py-0.5 font-mono text-red-600 dark:bg-red-900/40 dark:text-red-300"
            >
              {{ k }}={{ v }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- 模板标签 -->
    <div v-if="entries(templateLabels).length > 0">
      <p class="mb-1.5 text-xs font-medium text-slate-500 dark:text-slate-400">模板标签</p>
      <div class="flex flex-wrap gap-1.5">
        <span
          v-for="([k, v]) in entries(templateLabels)"
          :key="k"
          class="rounded-md bg-slate-100 px-2 py-0.5 font-mono text-xs text-slate-600 dark:bg-slate-700 dark:text-slate-300"
        >
          {{ k }}={{ v }}
        </span>
      </div>
    </div>

    <!-- 空 -->
    <div v-if="containers.length === 0" class="py-8 text-center text-sm text-slate-400">
      暂无 Pod 模板信息
    </div>
  </div>
</template>
