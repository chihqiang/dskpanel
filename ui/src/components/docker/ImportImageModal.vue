<script setup lang="ts">
import { onBeforeUnmount, ref } from 'vue'
import Button from '@/components/ui/Button.vue'
import Modal from '@/components/ui/Modal.vue'
import ProgressBar from '@/components/ui/ProgressBar.vue'
import { useToast } from '@/composables/useToast'
import { useActivity } from '@/composables/useActivity'
import { importImage } from '@/api/image'
import { fmtSize } from '@/utils/format'

const props = defineProps<{ open: boolean }>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  imported: []
}>()

const file = ref<File | null>(null)
const importing = ref(false)
const done = ref(false)
const progressValue = ref(0)
const progressText = ref('')
const toast = useToast()
const activity = useActivity()
let abortFn: (() => void) | null = null

function onFile(e: Event): void {
  const input = e.target as HTMLInputElement
  const f = input.files?.[0]
  file.value = f ?? null
}

async function submit(): Promise<void> {
  if (!file.value) {
    toast.error('请选择 .tar 镜像文件')
    return
  }
  importing.value = true
  done.value = false
  progressValue.value = 0
  progressText.value = ''

  const { promise, abort } = importImage(file.value, (loaded, total) => {
    progressValue.value = total > 0 ? loaded / total : 0
    progressText.value = `${fmtSize(loaded)} / ${fmtSize(total)}`
  })
  abortFn = abort

  try {
    await promise
    done.value = true
    progressValue.value = 1
    toast.success('镜像导入完成')
    activity.success(`导入镜像「${file.value.name}」`, file.value.name)
    emit('imported')
  } catch (err) {
    if ((err as Error).name === 'AbortError') {
      toast.info('已取消导入')
      activity.info(`取消导入镜像「${file.value?.name ?? ''}」`, file.value?.name)
    } else {
      toast.error((err as Error).message)
      activity.error(`导入镜像失败：${(err as Error).message}`, file.value?.name)
    }
  } finally {
    importing.value = false
    abortFn = null
  }
}

/** 取消导入。 */
function cancelImport(): void {
  abortFn?.()
  abortFn = null
}

function close(): void {
  if (importing.value) return
  file.value = null
  done.value = false
  progressValue.value = 0
  progressText.value = ''
  emit('update:open', false)
}

onBeforeUnmount(() => {
  abortFn?.()
  abortFn = null
})
</script>

<template>
  <Modal :open="open" @update:open="(v) => !v && close()" title="导入镜像" width="max-w-md">
    <div class="space-y-4">
      <div>
        <label class="mb-1.5 block text-sm text-slate-500">选择镜像文件（.tar，来自镜像导出）</label>
        <input
          type="file"
          accept=".tar"
          class="block w-full text-sm text-slate-600 file:mr-3 file:rounded-md file:border-0 file:bg-blue-600 file:px-4 file:py-2 file:text-sm file:font-medium file:text-white hover:file:bg-blue-700 dark:text-slate-300"
          :disabled="importing"
          @change="onFile"
        />
      </div>

      <!-- 进度条 -->
      <div v-if="importing || done" class="space-y-2">
        <div class="flex items-center justify-between text-xs text-slate-500">
          <span>{{ done ? '完成' : '上传中...' }}</span>
          <span v-if="progressText">{{ progressText }}</span>
        </div>
        <ProgressBar :value="progressValue" />
      </div>

      <div v-if="done" class="flex items-center gap-2 text-sm text-green-600">
        <span class="h-4 w-4 rounded-full bg-green-100 text-center leading-4">✓</span>镜像已导入
      </div>
    </div>
    <template #footer>
      <Button v-if="importing" variant="danger" @click="cancelImport">取消导入</Button>
      <Button v-else variant="secondary" :disabled="importing" @click="close">关闭</Button>
      <Button v-if="!done" :loading="importing" @click="submit">导入</Button>
      <Button v-else variant="secondary" @click="close">完成</Button>
    </template>
  </Modal>
</template>
