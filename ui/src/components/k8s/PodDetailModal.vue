<script setup lang="ts">
import { ref, watch } from 'vue'
import Modal from '@/components/ui/Modal.vue'
import Badge from '@/components/ui/Badge.vue'
import ResourceEventsTab from './ResourceEventsTab.vue'
import { podPhaseVariant, k8sContainerStateVariant } from '@/utils/k8s'
import { type K8sPodItem } from '@/api/k8s'

type DetailTab = 'info' | 'events'
const detailTab = ref<DetailTab>('info')

const props = defineProps<{
  open: boolean
  pod: K8sPodItem | null
}>()

const emit = defineEmits<{ 'update:open': [value: boolean] }>()

watch(
  () => [props.open, props.pod] as const,
  ([open, pod]) => {
    if (open && pod) {
      detailTab.value = 'info'
    }
  },
  { immediate: true },
)

function entries(obj: Record<string, string> | undefined): [string, string][] {
  return Object.entries(obj ?? {})
}
</script>

<template>
  <Modal :open="open" @update:open="emit('update:open', $event)" :title="`Pod - ${pod?.name ?? ''}`" width="max-w-3xl">
    <div v-if="pod" class="space-y-5">
      <!-- Tab 切换 -->
      <div class="inline-flex rounded-lg bg-slate-100 p-1 dark:bg-slate-700/60">
        <button
          v-for="t in [{ key: 'info', label: '详情' }, { key: 'events', label: '事件' }] as const"
          :key="t.key"
          class="rounded-md px-3.5 py-1.5 text-sm font-medium transition-colors"
          :class="detailTab === t.key ? 'bg-white text-blue-700 shadow-sm dark:bg-slate-800 dark:text-blue-300' : 'text-slate-600 hover:text-slate-800 dark:text-slate-300 dark:hover:text-slate-100'"
          @click="detailTab = t.key"
        >
          {{ t.label }}
        </button>
      </div>

      <!-- 详情 Tab -->
      <div v-show="detailTab === 'info'" class="space-y-5">
        <!-- 基本信息 -->
        <div class="grid grid-cols-2 gap-3 text-sm sm:grid-cols-4">
          <div class="rounded-lg bg-slate-50 px-3 py-2 dark:bg-slate-700/40">
            <p class="text-xs text-slate-400">命名空间</p>
            <p class="font-medium text-slate-700 dark:text-slate-200">{{ pod.namespace }}</p>
          </div>
          <div class="rounded-lg bg-slate-50 px-3 py-2 dark:bg-slate-700/40">
            <p class="text-xs text-slate-400">阶段</p>
            <p><Badge :variant="podPhaseVariant(pod.status)" dot>{{ pod.status }}</Badge></p>
          </div>
          <div class="rounded-lg bg-slate-50 px-3 py-2 dark:bg-slate-700/40">
            <p class="text-xs text-slate-400">就绪</p>
            <p class="font-medium text-slate-700 dark:text-slate-200">{{ pod.ready }}</p>
          </div>
          <div class="rounded-lg bg-slate-50 px-3 py-2 dark:bg-slate-700/40">
            <p class="text-xs text-slate-400">重启次数</p>
            <p class="font-medium text-slate-700 dark:text-slate-200">{{ pod.restarts }}</p>
          </div>
          <div class="rounded-lg bg-slate-50 px-3 py-2 dark:bg-slate-700/40">
            <p class="text-xs text-slate-400">IP</p>
            <p class="font-medium text-slate-700 dark:text-slate-200">{{ pod.ip || '—' }}</p>
          </div>
          <div class="rounded-lg bg-slate-50 px-3 py-2 dark:bg-slate-700/40">
            <p class="text-xs text-slate-400">节点</p>
            <p class="font-medium text-slate-700 dark:text-slate-200">{{ pod.node_name || '—' }}</p>
          </div>
          <div class="rounded-lg bg-slate-50 px-3 py-2 dark:bg-slate-700/40">
            <p class="text-xs text-slate-400">QoS</p>
            <p class="font-medium text-slate-700 dark:text-slate-200">{{ pod.qos_class || '—' }}</p>
          </div>
          <div class="rounded-lg bg-slate-50 px-3 py-2 dark:bg-slate-700/40">
            <p class="text-xs text-slate-400">创建时间</p>
            <p class="font-medium text-slate-700 dark:text-slate-200">{{ pod.created_at }}</p>
          </div>
        </div>

        <!-- 容器列表 -->
        <div>
          <p class="mb-2 text-sm font-medium text-slate-700 dark:text-slate-200">容器</p>
          <div v-if="pod.containers?.length" class="space-y-2">
            <div
              v-for="c in pod.containers"
              :key="c.name"
              class="rounded-lg border border-slate-100 px-3 py-2 dark:border-slate-700"
            >
              <div class="flex flex-wrap items-center gap-2 text-sm">
                <span class="font-medium text-slate-700 dark:text-slate-200">{{ c.name }}</span>
                <Badge :variant="k8sContainerStateVariant(c.state)" dot>{{ c.state }}</Badge>
                <Badge :variant="c.ready ? 'green' : 'yellow'">{{ c.ready ? '就绪' : '未就绪' }}</Badge>
                <span class="text-xs text-slate-400">重启 {{ c.restarts }}</span>
              </div>
              <p class="mt-1 font-mono text-xs text-slate-400">{{ c.image }}</p>
              <p v-if="c.reason" class="mt-0.5 text-xs text-yellow-600 dark:text-yellow-400">原因: {{ c.reason }}</p>
            </div>
          </div>
          <p v-else class="text-sm text-slate-400">无容器信息</p>
        </div>

        <!-- Labels -->
        <div>
          <p class="mb-2 text-sm font-medium text-slate-700 dark:text-slate-200">标签</p>
          <div v-if="entries(pod.labels).length" class="flex flex-wrap gap-1.5">
            <span
              v-for="([k, v]) in entries(pod.labels)"
              :key="k"
              class="rounded-md bg-slate-100 px-2 py-0.5 font-mono text-xs text-slate-600 dark:bg-slate-700 dark:text-slate-300"
            >
              {{ k }}={{ v }}
            </span>
          </div>
          <p v-else class="text-sm text-slate-400">无</p>
        </div>
      </div>

      <!-- 事件 Tab -->
      <div v-show="detailTab === 'events'">
        <ResourceEventsTab
          kind="Pod"
          :name="pod.name"
          :namespace="pod.namespace"
          :active="detailTab === 'events'"
        />
      </div>
    </div>
  </Modal>
</template>
