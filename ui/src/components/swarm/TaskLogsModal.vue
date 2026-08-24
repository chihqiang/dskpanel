<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import Button from '@/components/ui/Button.vue'
import Modal from '@/components/ui/Modal.vue'
import Log from '@/components/ui/Log.vue'
import { streamSwarmTaskLogs } from '@/api/swarm'

const props = defineProps<{
  open: boolean
  taskId: string
  taskLabel?: string
}>()

const emit = defineEmits<{ 'update:open': [value: boolean] }>()

const logLines = ref<string[]>([])
const connecting = ref(false)
const streaming = ref(false)
const errorMsg = ref('')
let stopStream: (() => void) | null = null

function start(): void {
  stopStream?.()
  logLines.value = []
  errorMsg.value = ''
  connecting.value = true
  streaming.value = true

  stopStream = streamSwarmTaskLogs(
    props.taskId,
    300,
    (line) => {
      connecting.value = false
      logLines.value.push(line)
      if (logLines.value.length > 3000) {
        logLines.value.splice(0, logLines.value.length - 3000)
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

watch(
  () => props.open,
  (open) => {
    if (open && props.taskId) {
      start()
    } else {
      stopStream?.()
      stopStream = null
    }
  },
  { immediate: true },
)
onBeforeUnmount(() => stopStream?.())
</script>

<template>
  <Modal
    :open="open"
    @update:open="(v) => (v ? null : close())"
    :title="`任务日志 - ${taskLabel || taskId.slice(0, 12)}`"
    width="max-w-4xl"
  >
    <Log
      :lines="logLines"
      :loading="connecting"
      :streaming="streaming"
      :error="errorMsg"
      height="h-[60vh]"
      empty-text="暂无日志"
    />
    <template #footer>
      <Button variant="secondary" @click="close">关闭</Button>
    </template>
  </Modal>
</template>
