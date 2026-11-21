<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import Button from '@/components/ui/Button.vue'
import Modal from '@/components/ui/Modal.vue'
import Log from '@/components/ui/Log.vue'

/**
 * 通用日志弹窗：Docker 容器日志、K8s Pod 日志等共用。
 *
 * 调用方通过 props 注入 SSE stream 和可选的一次性下载函数，
 * 组件负责日志显示、行数选择、容器选择、刷新、下载等交互。
 */

const props = withDefaults(
  defineProps<{
    /** 弹窗 open 状态（v-model:open）。 */
    open: boolean
    /** 弹窗标题（如「日志 - my-container」）。 */
    title: string
    /**
     * SSE stream 函数：调用后开始推送日志，返回取消函数。
     * 参数为行数 + 容器名（可选），回调为 onLine / onError / onClose。
     */
    stream: (
      tail: string,
      container: string,
      onLine: (line: string) => void,
      onError: (msg: string) => void,
      onClose: () => void,
    ) => () => void
    /**
     * 可选：一次性下载全量日志（返回文件名和文本）。
     * 提供后显示下载按钮。
     */
    download?: (container: string) => Promise<{ name: string; text: string }>
    /** 可选：容器选择列表（K8s Pod 多容器场景）。 */
    containers?: { name: string }[]
    /** 弹窗宽度。 */
    width?: string
    /** 日志框高度。 */
    height?: string
    /** 空状态文案。 */
    emptyText?: string
  }>(),
  {
    width: 'max-w-4xl',
    height: 'h-[60vh]',
    emptyText: '暂无日志',
    containers: () => [],
  },
)

const emit = defineEmits<{ 'update:open': [value: boolean] }>()

const logLines = ref<string[]>([])
const connecting = ref(false)
const streaming = ref(false)
const errorMsg = ref('')
const tailCount = ref('500')
const container = ref('')
let stopStream: (() => void) | null = null

function start(): void {
  stopStream?.()
  logLines.value = []
  errorMsg.value = ''
  connecting.value = true
  streaming.value = true

  stopStream = props.stream(
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

/** 下载全量日志。 */
async function downloadLogs(): Promise<{ name: string; text: string }> {
  if (!props.download) return { name: 'logs.txt', text: '' }
  return props.download(container.value)
}

watch(
  () => props.open,
  (open) => {
    if (open) {
      // 默认选第一个容器。
      if (props.containers.length > 0 && !container.value) {
        container.value = props.containers[0].name
      }
      start()
    } else {
      stopStream?.()
      stopStream = null
    }
  },
  { immediate: true },
)

// 容器切换时重新加载。
watch(container, () => {
  if (props.open) start()
})

onBeforeUnmount(() => {
  stopStream?.()
  stopStream = null
})
</script>

<template>
  <Modal
    :open="open"
    @update:open="(v) => !v && close()"
    :title="title"
    :width="width"
  >
    <Log
      :lines="logLines"
      :loading="connecting"
      :streaming="streaming"
      :error="errorMsg"
      :height="height"
      :empty-text="emptyText"
      :download="download ? downloadLogs : undefined"
    >
      <template #actions>
        <select
          v-if="containers.length > 1"
          v-model="container"
          class="h-8 rounded-md border border-slate-300 bg-white px-1.5 text-xs text-slate-700 outline-none focus:border-blue-500 dark:border-slate-600 dark:bg-slate-900 dark:text-slate-100"
        >
          <option v-for="c in containers" :key="c.name" :value="c.name">{{ c.name }}</option>
        </select>
        <select
          v-model="tailCount"
          class="h-8 rounded-md border border-slate-300 bg-white px-1.5 text-xs text-slate-700 outline-none focus:border-blue-500 dark:border-slate-600 dark:bg-slate-900 dark:text-slate-100"
          @change="start"
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
