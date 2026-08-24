<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import Button from '@/components/ui/Button.vue'
import Modal from '@/components/ui/Modal.vue'
import Log from '@/components/ui/Log.vue'
import { streamContainerLogs, getContainerLogs } from '@/api/container'

const props = withDefaults(
  defineProps<{
    open: boolean
    containerId: string
    containerName?: string
    tail?: string
  }>(),
  { containerName: '', tail: '100' },
)

const emit = defineEmits<{ 'update:open': [value: boolean] }>()

const logLines = ref<string[]>([])
const connecting = ref(false)
const streaming = ref(false)
const errorMsg = ref('')
let stopStream: (() => void) | null = null

// 行数选择（容器日志专属，通过 Log 的 actions 插槽注入）。
const tailCount = ref('500')

function changeTail(): void {
  start()
}

/** 启动 SSE 日志流（打开时 + 手动重连）。 */
function start(): void {
  stopStream?.()
  logLines.value = []
  errorMsg.value = ''
  connecting.value = true
  streaming.value = true

  stopStream = streamContainerLogs(
    props.containerId,
    tailCount.value,
    (line) => {
      // 收到数据即视为已连接（避免 connecting 一直为 true）。
      connecting.value = false
      logLines.value.push(line)
      if (logLines.value.length > 5000) {
        logLines.value.splice(0, logLines.value.length - 5000)
      }
    },
    (msg) => {
      errorMsg.value = msg
      streaming.value = false
    },
    () => {
      connecting.value = false
      streaming.value = false
    },
  )
}

function close(): void {
  stopStream?.()
  stopStream = null
  emit('update:open', false)
}

/** 下载全量日志（供 Log 组件下载按钮调用）。 */
async function downloadLogs(): Promise<{ name: string; text: string }> {
  const text = await getContainerLogs(props.containerId, '10000')
  const name = `${props.containerName || props.containerId.slice(0, 12)}-logs.txt`
  return { name, text }
}

watch(
  () => props.open,
  (open) => {
    if (open && props.containerId) {
      start()
    } else {
      stopStream?.()
      stopStream = null
    }
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  stopStream?.()
  stopStream = null
})
</script>

<template>
  <Modal
    :open="open"
    @update:open="(v) => !v && close()"
    :title="`日志 - ${containerName || containerId.slice(0, 12)}`"
    width="max-w-3xl"
  >
    <Log
      :lines="logLines"
      :loading="connecting"
      :streaming="streaming"
      :error="errorMsg"
      height="h-96"
      :download="downloadLogs"
    >
      <template #actions>
        <select
          v-model="tailCount"
          class="h-8 rounded-md border border-slate-300 bg-white px-1.5 text-xs text-slate-700 outline-none focus:border-blue-500 dark:border-slate-600 dark:bg-slate-900 dark:text-slate-100"
          @change="changeTail"
        >
          <option value="100">100 行</option>
          <option value="500">500 行</option>
          <option value="2000">2000 行</option>
          <option value="5000">5000 行</option>
        </select>
        <Button variant="secondary" size="sm" :loading="connecting" @click="start">刷新</Button>
      </template>
    </Log>
    <template #footer>
      <Button variant="secondary" @click="close">关闭</Button>
    </template>
  </Modal>
</template>
