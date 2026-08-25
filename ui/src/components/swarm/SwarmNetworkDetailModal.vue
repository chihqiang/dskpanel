<script setup lang="ts">
import { ref, watch } from 'vue'
import Modal from '@/components/ui/Modal.vue'
import { swarmNetworkInspect, type SwarmNetworkItem } from '@/api/swarm'

const props = defineProps<{
  open: boolean
  network: SwarmNetworkItem | null
}>()

const emit = defineEmits<{ 'update:open': [value: boolean] }>()

const detail = ref('')
const detailLoading = ref(false)

async function fetchDetail(id: string): Promise<void> {
  detail.value = ''
  detailLoading.value = true
  try {
    const raw = await swarmNetworkInspect(id)
    detail.value = JSON.stringify(raw, null, 2)
  } catch (err) {
    detail.value = `加载失败: ${(err as Error).message}`
  } finally {
    detailLoading.value = false
  }
}

watch(
  () => [props.open, props.network] as const,
  ([open, network]) => {
    if (open && network) {
      void fetchDetail(network.id)
    }
  },
  { immediate: true },
)
</script>

<template>
  <Modal :open="open" @update:open="emit('update:open', $event)" title="网络详情" width="max-w-3xl">
    <pre
      class="max-h-[70vh] overflow-auto rounded-lg bg-slate-50 p-4 text-xs leading-relaxed text-slate-700 dark:bg-slate-900 dark:text-slate-300"
    >{{ detailLoading ? '加载中…' : detail }}</pre>
  </Modal>
</template>
