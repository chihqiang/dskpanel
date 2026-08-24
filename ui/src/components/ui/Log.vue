<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { Search, Download, Pause, Play, Copy, Check } from '@lucide/vue'
import Button from '@/components/ui/Button.vue'

const props = withDefaults(
  defineProps<{
    /** 日志行（外部驱动，通常由流式/一次性拉取填充）。 */
    lines: string[]
    /** 连接中（无行时显示占位）。 */
    loading?: boolean
    /** 是否实时流中（工具栏显示绿点）。 */
    streaming?: boolean
    /** 错误信息（无行时显示占位）。 */
    error?: string
    /** 日志框高度（Tailwind 高度类，如 h-96 / max-h-56）。 */
    height?: string
    /** 是否显示工具条（搜索/过滤/暂停/复制/下载）。 */
    showToolbar?: boolean
    /** 显示行数上限，超出截断只保留最近 N 行。 */
    maxLines?: number
    /** 空状态文案。 */
    emptyText?: string
    /** 可选：下载全量日志（返回文件名与文本）。提供后显示下载按钮。 */
    download?: () => Promise<{ name: string; text: string }>
  }>(),
  {
    loading: false,
    streaming: false,
    error: '',
    height: 'h-96',
    showToolbar: true,
    maxLines: 5000,
    emptyText: '暂无日志',
  },
)

// ---- 工具条状态 ----
const keyword = ref('')
const showOnlyErrors = ref(false)
const paused = ref(false)
const copied = ref(false)
const downloading = ref(false)

// ---- 滚动 ----
const logBox = ref<HTMLElement | null>(null)
let autoScroll = true

/** 显示行（上限截断，保留最近 maxLines 行）。 */
const shownLines = computed(() =>
  props.lines.length > props.maxLines ? props.lines.slice(-props.maxLines) : props.lines,
)

/** 过滤后的行（关键词 + 仅错误/警告）。 */
const filteredLines = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  let lines = shownLines.value
  if (showOnlyErrors.value) {
    lines = lines.filter((l) => /error|exception|fatal|panic|warn|failed|errno/i.test(l))
  }
  if (!kw) return lines
  return lines.filter((l) => l.toLowerCase().includes(kw))
})

/** 高亮关键词的行（拆分为片段）。 */
function highlight(line: string): { text: string; match: boolean }[] {
  const kw = keyword.value.trim()
  if (!kw) return [{ text: line, match: false }]
  const lower = line.toLowerCase()
  const k = kw.toLowerCase()
  const parts: { text: string; match: boolean }[] = []
  let i = 0
  let idx: number
  while ((idx = lower.indexOf(k, i)) >= 0) {
    if (idx > i) parts.push({ text: line.slice(i, idx), match: false })
    parts.push({ text: line.slice(idx, idx + kw.length), match: true })
    i = idx + kw.length
  }
  if (i < line.length) parts.push({ text: line.slice(i), match: false })
  return parts
}

/** 行类型（用于着色）。 */
function lineType(line: string): 'error' | 'warn' | 'info' {
  if (/\berror\b|\bexception\b|\bfatal\b|\bpanic\b|errno/i.test(line)) return 'error'
  if (/\bwarn|warning/i.test(line)) return 'warn'
  return 'info'
}

/** 暂停 / 继续实时滚动。 */
function togglePause(): void {
  paused.value = !paused.value
  if (!paused.value) scrollToBottom()
}

function onScroll(): void {
  if (!logBox.value) return
  const { scrollTop, scrollHeight, clientHeight } = logBox.value
  // 距底部 < 40px 视为跟随滚动。
  autoScroll = scrollHeight - scrollTop - clientHeight < 40
}

function scrollToBottom(): void {
  autoScroll = true
  nextTick(() => {
    if (logBox.value) logBox.value.scrollTop = logBox.value.scrollHeight
  })
}

// 新行到来时跟随滚动（未暂停时）。
watch(
  () => props.lines.length,
  () => {
    if (autoScroll && !paused.value) scrollToBottom()
  },
)

/** 复制全部日志。 */
async function copyAll(): Promise<void> {
  try {
    await navigator.clipboard.writeText(props.lines.join('\n'))
    copied.value = true
    setTimeout(() => (copied.value = false), 1500)
  } catch {
    // 剪贴板被拒绝时静默。
  }
}

/** 下载全量日志（调用外部 download 函数，纯文本保存）。 */
async function downloadLogs(): Promise<void> {
  if (!props.download) return
  downloading.value = true
  try {
    const { name, text } = await props.download()
    const blob = new Blob([text], { type: 'text/plain;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = name
    document.body.appendChild(a)
    a.click()
    a.remove()
    URL.revokeObjectURL(url)
  } finally {
    downloading.value = false
  }
}

defineExpose({ scrollToBottom })
</script>

<template>
  <div class="space-y-2">
    <!-- 工具条 -->
    <div v-if="showToolbar" class="flex flex-wrap items-center justify-between gap-2">
      <span class="text-xs text-slate-400">
        共 {{ filteredLines.length }} / {{ lines.length }} 行
        <span v-if="paused" class="ml-1 text-amber-500">⏸ 已暂停</span>
        <span v-else-if="streaming" class="ml-1 inline-flex items-center gap-1 text-green-500">
          <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-green-500" />实时
        </span>
        <span v-else-if="loading" class="ml-1 text-slate-400">连接中...</span>
        <span v-else-if="!streaming && lines.length" class="ml-1 text-amber-500">已断开</span>
      </span>
      <div class="flex flex-wrap items-center gap-2">
        <div class="relative">
          <Search class="pointer-events-none absolute left-2 top-1/2 h-3 w-3 -translate-y-1/2 text-slate-400" />
          <input
            v-model="keyword"
            type="text"
            class="h-8 w-44 rounded-md border border-slate-300 bg-white pl-6 pr-2 text-xs text-slate-700 outline-none focus:border-blue-500 dark:border-slate-600 dark:bg-slate-900 dark:text-slate-100"
            placeholder="搜索 / 高亮关键词"
          />
        </div>
        <label class="flex cursor-pointer items-center gap-1 text-xs text-slate-500">
          <input v-model="showOnlyErrors" type="checkbox" class="h-3.5 w-3.5 rounded border-slate-300" />
          仅错误/警告
        </label>
        <Button
          variant="secondary"
          size="sm"
          :disabled="!streaming && !loading && lines.length === 0"
          @click="togglePause"
        >
          <Pause v-if="!paused" class="mr-1 h-3.5 w-3.5" />
          <Play v-else class="mr-1 h-3.5 w-3.5" />
          {{ paused ? '继续' : '暂停' }}
        </Button>
        <Button variant="secondary" size="sm" :disabled="lines.length === 0" @click="copyAll">
          <Check v-if="copied" class="mr-1 h-3.5 w-3.5 text-green-500" />
          <Copy v-else class="mr-1 h-3.5 w-3.5" />
          {{ copied ? '已复制' : '复制' }}
        </Button>
        <Button v-if="download" variant="secondary" size="sm" :loading="downloading" @click="downloadLogs">
          <Download class="mr-1 h-3.5 w-3.5" />下载
        </Button>
        <slot name="actions" />
      </div>
    </div>

    <!-- 日志框 -->
    <div
      ref="logBox"
      :class="height"
      class="overflow-y-auto rounded-md bg-slate-900 p-3 font-mono text-xs text-slate-100"
      @scroll="onScroll"
    >
      <template v-if="loading && lines.length === 0">
        <div class="text-slate-500">连接中...</div>
      </template>
      <template v-else-if="error && lines.length === 0">
        <div class="text-red-400">{{ error }}</div>
      </template>
      <template v-else-if="filteredLines.length > 0">
        <div
          v-for="(line, idx) in filteredLines"
          :key="idx"
          class="whitespace-pre-wrap break-all"
          :class="{
            'text-red-400': lineType(line) === 'error',
            'text-amber-400': lineType(line) === 'warn',
          }"
        >
          <template v-for="(part, i) in highlight(line)" :key="i">
            <mark
              v-if="part.match"
              class="rounded bg-yellow-300 px-0.5 text-slate-900"
            >{{ part.text }}</mark>
            <template v-else>{{ part.text }}</template>
          </template>
        </div>
      </template>
      <template v-else>
        <div class="text-slate-500">{{ emptyText }}{{ keyword || showOnlyErrors ? '（无匹配）' : '' }}</div>
      </template>
    </div>
  </div>
</template>
