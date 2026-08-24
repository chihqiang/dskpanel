<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import Button from '@/components/ui/Button.vue'
import Modal from '@/components/ui/Modal.vue'
import Log from '@/components/ui/Log.vue'
import { streamSwarmServiceLogs, fetchSwarmServiceLogs } from '@/api/swarm'

const props = defineProps<{
  open: boolean
  serviceId: string
  serviceName?: string
}>()

const emit = defineEmits<{ 'update:open': [value: boolean] }>()

const logLines = ref<string[]>([])
const connecting = ref(false)
const streaming = ref(false)
const errorMsg = ref('')
const tailCount = ref(500)
let stopStream: (() => void) | null = null

function start(): void {
  stopStream?.()
  logLines.value = []
  errorMsg.value = ''
  connecting.value = true
  streaming.value = true

  stopStream = streamSwarmServiceLogs(
    props.serviceId,
    tailCount.value,
    (line) => {
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

async function downloadLogs(): Promise<{ name: string; text: string }> {
  const text = await fetchSwarmServiceLogs(props.serviceId, 10000)
  const name = `${props.serviceName || props.serviceId.slice(0, 12)}-logs.txt`
  return { name, text }
}

watch(
  () => props.open,
  (open) => {
    if (open && props.serviceId) {
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
    :title="`服务日志 - ${serviceName || serviceId.slice(0, 12)}`"
    width="max-w-4xl"
  >
    <Log
      :lines="logLines"
      :loading="connecting"
      :streaming="streaming"
      :error="errorMsg"
      :download="downloadLogs"
      height="h-[60vh]"
      empty-text="暂无日志"
    />
    <template #footer>
      <div class="flex items-center gap-2">
        <select v-model="tailCount" class="input input-sm w-28" @change="start">
          <option :value="100">100 行</option>
          <option :value="500">500 行</option>
          <option :value="1000">1000 行</option>
          <option :value="5000">5000 行</option>
        </select>
        <span class="text-xs text-slate-400">重连</span>
      </div>
      <div class="ml-auto">
        <Button variant="secondary" @click="close">关闭</Button>
      </div>
    </template>
  </Modal>
</template>
