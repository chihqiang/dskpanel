<script setup lang="ts">
import { ref, watch } from 'vue'
import Button from '@/components/ui/Button.vue'
import Modal from '@/components/ui/Modal.vue'
import { useToast } from '@/composables/useToast'
import { useNProgress } from '@/composables/useNProgress'
import { exportImage } from '@/api/image'

const props = defineProps<{ open: boolean; target: { ids: string[]; tag: string } | null }>()

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()

const exporting = ref(false)
const done = ref(false)
const toast = useToast()
const nprogress = useNProgress()

watch(
  () => props.open,
  (open) => {
    if (open && props.target) {
      done.value = false
      start()
    }
  },
  { immediate: true },
)

async function start(): Promise<void> {
  if (!props.target) return
  exporting.value = true
  done.value = false
  nprogress.start()
  try {
    await exportImage(props.target.ids, props.target.tag, (loaded, total) => {
      nprogress.set(total > 0 ? loaded / total : 0)
    })
    done.value = true
    nprogress.done()
    toast.success('已导出镜像，开始下载')
  } catch (err) {
    nprogress.done()
    toast.error((err as Error).message)
  } finally {
    exporting.value = false
  }
}

function close(): void {
  emit('update:open', false)
}
</script>

<template>
  <Modal :open="open" @update:open="(v) => !v && close()" title="导出镜像" width="max-w-md">
    <div class="space-y-4">
      <div class="truncate text-sm text-slate-700 dark:text-slate-200">
        <span class="text-slate-500">正在导出：</span>{{ target?.tag || target?.ids[0] }}
      </div>

      <div v-if="done" class="flex items-center gap-2 text-sm text-green-600">
        <span class="h-4 w-4 rounded-full bg-green-100 text-center leading-4">✓</span>已生成 .tar 文件，开始下载
      </div>
    </div>
    <template #footer>
      <Button variant="secondary" :disabled="exporting" @click="close">关闭</Button>
    </template>
  </Modal>
</template>
