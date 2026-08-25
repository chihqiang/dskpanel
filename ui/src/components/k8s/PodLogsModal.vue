<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import Button from '@/components/ui/Button.vue'
import Modal from '@/components/ui/Modal.vue'
import Log from '@/components/ui/Log.vue'
import { streamK8sPodLogs, type K8sPodItem } from '@/api/k8s'

const props = defineProps<{
  open: boolean
  pod: K8sPodItem | null
}>()

const emit = defineEmits<{ 'update:open': [value: boolean] }>()

const logLines = ref<string[]>([])
const connecting = ref(false)
const streaming = ref(false)
const errorMsg = ref('')
const tailCount = ref(500)
const container = ref('')
let stopStream: (() => void) | null = null

function start(): void {
  if (!props.pod) return
  stopStream?.()
  logLines.value = []
  errorMsg.value = ''
  connecting.value = true
  streaming.value = true

  stopStream = streamK8sPodLogs(
    props.pod.name,
    props.pod.namespace,
    tailCount.value,
    container.value,
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

watch(
  () => [props.open, props.pod] as const,
  ([open, pod]) => {
    if (open && pod) {
      container.value = pod.containers?.[0]?.name ?? ''
      start()
    } else {
      stopStream?.()
      stopStream = null
    }
  },
  { immediate: true },
)

watch(container, () => {
  if (props.open && props.pod) start()
})

onBeforeUnmount(() => stopStream?.())
</script>

<template>
  <Modal
    :open="open"
    @update:open="(v) => (v ? null : close())"
    :title="`Pod 日志 - ${pod?.name ?? ''}`"
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
      <div class="flex flex-wrap items-center gap-2">
        <select v-model="container" class="input input-sm w-40" :disabled="(pod?.containers?.length ?? 0) <= 1">
          <option v-for="c in pod?.containers ?? []" :key="c.name" :value="c.name">{{ c.name }}</option>
        </select>
        <select v-model="tailCount" class="input input-sm w-28" @change="start">
          <option :value="100">100 行</option>
          <option :value="500">500 行</option>
          <option :value="1000">1000 行</option>
          <option :value="5000">5000 行</option>
        </select>
        <span class="text-xs text-slate-400">实时跟随</span>
      </div>
      <div class="ml-auto">
        <Button variant="secondary" @click="close">关闭</Button>
      </div>
    </template>
  </Modal>
</template>
