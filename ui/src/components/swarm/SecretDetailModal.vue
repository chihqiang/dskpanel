<script setup lang="ts">
import { ref, watch } from 'vue'
import Button from '@/components/ui/Button.vue'
import Modal from '@/components/ui/Modal.vue'
import { useToast } from '@/composables/useToast'
import {
  swarmSecretInspect,
  swarmConfigInspect,
  type SwarmSecretItem,
  type SwarmSecretDetail,
} from '@/api/swarm'

const props = defineProps<{
  open: boolean
  row: SwarmSecretItem | null
  kind: 'secret' | 'config'
}>()

const emit = defineEmits<{ 'update:open': [value: boolean] }>()

const toast = useToast()

const detailData = ref<SwarmSecretDetail | null>(null)
const detailLoading = ref(false)

async function fetchDetail(row: SwarmSecretItem, kind: 'secret' | 'config'): Promise<void> {
  detailData.value = null
  detailLoading.value = true
  try {
    detailData.value = kind === 'secret'
      ? await swarmSecretInspect(row.id)
      : await swarmConfigInspect(row.id)
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    detailLoading.value = false
  }
}

watch(
  () => [props.open, props.row, props.kind] as const,
  ([open, row, kind]) => {
    if (open && row) {
      void fetchDetail(row, kind)
    }
  },
  { immediate: true },
)
</script>

<template>
  <Modal
    :open="open"
    @update:open="emit('update:open', $event)"
    :title="`${kind === 'secret' ? 'Secret' : 'Config'} 详情 - ${detailData?.name ?? ''}`"
    width="max-w-2xl"
  >
    <div v-if="detailLoading" class="py-8 text-center text-sm text-slate-400">加载中…</div>
    <div v-else-if="detailData" class="space-y-4">
      <div class="grid grid-cols-2 gap-4">
        <div>
          <p class="text-xs text-slate-400">名称</p>
          <p class="font-mono text-sm text-slate-700 dark:text-slate-200">{{ detailData.name }}</p>
        </div>
        <div>
          <p class="text-xs text-slate-400">创建时间</p>
          <p class="text-sm text-slate-700 dark:text-slate-200">{{ detailData.created_at }}</p>
        </div>
      </div>
      <div>
        <p class="mb-1.5 text-sm text-slate-500">内容</p>
        <pre
          v-if="kind === 'config'"
          class="max-h-72 overflow-auto rounded-lg bg-slate-50 p-3 font-mono text-xs leading-relaxed text-slate-700 dark:bg-slate-900 dark:text-slate-300"
        >{{ detailData.data }}</pre>
        <div v-else class="rounded-lg bg-slate-50 p-4 text-center text-sm text-slate-400 dark:bg-slate-900">
          Secret 内容由 Docker 安全存储，API 不提供明文查看
        </div>
      </div>
      <div v-if="detailData.labels && Object.keys(detailData.labels).length">
        <p class="mb-1.5 text-xs text-slate-400">标签</p>
        <div class="flex flex-wrap gap-1.5">
          <span v-for="(v, k) in detailData.labels" :key="k" class="rounded-md bg-slate-100 px-2 py-0.5 text-xs text-slate-600 dark:bg-slate-700 dark:text-slate-300">{{ k }}={{ v }}</span>
        </div>
      </div>
    </div>
    <template #footer>
      <Button variant="secondary" @click="emit('update:open', false)">关闭</Button>
    </template>
  </Modal>
</template>
