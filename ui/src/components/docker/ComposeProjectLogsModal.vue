<script setup lang="ts">
import { ref, watch } from 'vue'
import Modal from '@/components/ui/Modal.vue'
import { useToast } from '@/composables/useToast'
import { composeProjectLogs, type ComposeProjectItem } from '@/api/compose'

const props = defineProps<{
  open: boolean
  project: ComposeProjectItem | null
}>()

const emit = defineEmits<{ 'update:open': [value: boolean] }>()

const toast = useToast()

const logsLoading = ref(false)
const logs = ref<string[]>([])

async function fetchLogs(name: string): Promise<void> {
  logs.value = []
  logsLoading.value = true
  try {
    logs.value = await composeProjectLogs(name)
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    logsLoading.value = false
  }
}

watch(
  () => [props.open, props.project] as const,
  ([open, project]) => {
    if (open && project) {
      void fetchLogs(project.name)
    }
  },
  { immediate: true },
)
</script>

<template>
  <Modal :open="open" @update:open="emit('update:open', $event)" :title="`日志 - ${project?.name ?? ''}`" width="max-w-3xl">
    <div class="rounded-md bg-slate-900 p-3 font-mono text-xs text-slate-100">
      <div v-if="logsLoading" class="text-slate-500">加载中...</div>
      <div v-else-if="logs.length === 0" class="text-slate-500">暂无日志</div>
      <div v-else class="max-h-96 space-y-0.5 overflow-y-auto">
        <div v-for="(line, i) in logs" :key="i" class="whitespace-pre-wrap break-all">{{ line }}</div>
      </div>
    </div>
  </Modal>
</template>
