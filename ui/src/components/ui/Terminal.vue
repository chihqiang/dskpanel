<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Terminal as XTerm } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import '@xterm/xterm/css/xterm.css'
import { getToken } from '@/api/http'

interface TerminalProps {
  /** 终端标题（显示在顶部）。 */
  title?: string
  /** WebSocket 地址（不含 token query，由组件自动附加鉴权）。 */
  url: string
  /** 是否自动连接（默认 true）。 */
  autoConnect?: boolean
  /** 自定义初始命令（可选，发送到 stdin）。 */
  initialCommand?: string
  /** 断开后自动重连（默认 false）。 */
  reconnect?: boolean
  /** 高度类（Tailwind，默认 h-96）。 */
  height?: string
}

const props = withDefaults(defineProps<TerminalProps>(), {
  title: '终端',
  autoConnect: true,
  initialCommand: '',
  reconnect: false,
  height: 'h-96',
})

const emit = defineEmits<{
  /** 连接状态变化：connected / disconnected / error */
  status: [status: 'connected' | 'disconnected' | 'error', message?: string]
}>()

const containerRef = ref<HTMLElement | null>(null)
const statusText = ref('未连接')
const statusColor = ref('text-slate-400')
const connected = ref(false)

let term: XTerm | null = null
let fitAddon: FitAddon | null = null
let ws: WebSocket | null = null
let reconnectTimer: ReturnType<typeof setTimeout> | null = null
let mountTimer: ReturnType<typeof setTimeout> | null = null
let disposed = false
let termReady = false

function setStatus(text: string, color: string): void {
  statusText.value = text
  statusColor.value = color
}

/** 构建带鉴权的 ws 地址。 */
function buildUrl(): string {
  const token = getToken()
  // 生产同源（空 base）：/api/... → ws(s)://<host>/api/...
  // 开发直连后端：ws://127.0.0.1:8080 + /api/...
  const base = __WS_BASE__ ? `${__WS_BASE__}${props.url}` : `${location.protocol === 'https:' ? 'wss' : 'ws'}://${location.host}${props.url}`
  const sep = base.includes('?') ? '&' : '?'
  return token ? `${base}${sep}token=${encodeURIComponent(token)}` : base
}

/** 初始化 xterm（首次挂载时）。 */
function initTerminal(): void {
  if (term || !containerRef.value) return
  term = new XTerm({
    cursorBlink: true,
    fontSize: 13,
    fontFamily: 'Menlo, Monaco, Consolas, "Courier New", monospace',
    theme: {
      background: '#0f172a', // slate-900
      foreground: '#e2e8f0',
      cursor: '#94a3b8',
      selectionBackground: '#334155',
      black: '#000000',
      red: '#f87171',
      green: '#4ade80',
      yellow: '#facc15',
      blue: '#60a5fa',
      magenta: '#e879f9',
      cyan: '#22d3ee',
      white: '#e2e8f0',
      brightBlack: '#475569',
      brightRed: '#fca5a5',
      brightGreen: '#86efac',
      brightYellow: '#fde047',
      brightBlue: '#93c5fd',
      brightMagenta: '#f0abfc',
      brightCyan: '#67e8f9',
      brightWhite: '#f8fafc',
    },
    scrollback: 5000,
  })
  fitAddon = new FitAddon()
  term.loadAddon(fitAddon)
  term.loadAddon(new WebLinksAddon())
  term.open(containerRef.value)
  termReady = true
  // 延迟 fit：等待容器尺寸稳定（Modal Transition 动画期间尺寸可能不正确）。
  requestAnimationFrame(() => fit())

  // 用户输入 → WebSocket（作为 input 消息）。
  term.onData((data) => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: 'input', data }))
    }
  })
}

/** 自适应尺寸（容器尺寸变化时）。 */
function fit(): void {
  if (!fitAddon || !containerRef.value) return
  try {
    fitAddon.fit()
    // 同步 TTY 尺寸。
    const cols = term?.cols ?? 0
    const rows = term?.rows ?? 0
    if (ws && ws.readyState === WebSocket.OPEN && cols > 0 && rows > 0) {
      ws.send(JSON.stringify({ type: 'resize', cols, rows }))
    }
  } catch {
    // 容器不可见时忽略。
  }
}

/** 建立 WebSocket 连接。 */
function connect(): void {
  if (disposed) return
  if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
    return
  }
  if (!termReady) return

  setStatus('连接中...', 'text-amber-500')
  emit('status', 'disconnected')

  ws = new WebSocket(buildUrl())
  ws.binaryType = 'arraybuffer'

  ws.onopen = () => {
    connected.value = true
    setStatus('已连接', 'text-green-500')
    emit('status', 'connected')
    // 连接建立后同步一次尺寸 + 发送初始命令。
    // 延迟到下一帧再 fit，确保 Modal Transition 动画完成、容器尺寸已确定。
    requestAnimationFrame(() => fit())
    if (props.initialCommand) {
      ws?.send(JSON.stringify({ type: 'input', data: props.initialCommand }))
    }
    // 清屏，重新开始。
    term?.clear()
  }

  ws.onmessage = (ev) => {
    // 服务端推送容器输出（二进制帧）→ 写入 xterm。
    if (ev.data instanceof ArrayBuffer) {
      const bytes = new Uint8Array(ev.data)
      term?.write(bytes)
    } else if (typeof ev.data === 'string') {
      term?.write(ev.data)
    }
  }

  ws.onclose = () => {
    connected.value = false
    setStatus('已断开', 'text-red-500')
    emit('status', 'disconnected')
    // 需要重连时定时重连。
    if (props.reconnect && !disposed) {
      reconnectTimer = setTimeout(connect, 3000)
    }
  }

  ws.onerror = () => {
    setStatus('连接错误', 'text-red-500')
    emit('status', 'error')
  }
}

/** 关闭连接。 */
function disconnect(): void {
  if (reconnectTimer) {
    clearTimeout(reconnectTimer)
    reconnectTimer = null
  }
  if (ws) {
    ws.onclose = null
    ws.close()
    ws = null
  }
  connected.value = false
  setStatus('已断开', 'text-slate-400')
}

function toggle(): void {
  if (connected.value || (ws && ws.readyState === WebSocket.OPEN)) {
    disconnect()
  } else {
    connect()
  }
}

function focusTerminal(): void {
  term?.focus()
}

watch(
  () => props.url,
  () => {
    disconnect()
    if (props.autoConnect) connect()
  },
)

onMounted(async () => {
  initTerminal()
  // 等待 Modal Transition 动画完成（0.15s）后再连接，确保容器尺寸已确定。
  await nextTick()
  mountTimer = setTimeout(() => {
    // 主动聚焦 xterm，避免被 Modal 的 focus trap 抢到按钮上导致无法输入。
    focusTerminal()
    if (props.autoConnect) connect()
  }, 160)
  // 监听尺寸变化自适应。
  window.addEventListener('resize', fit)
})

onBeforeUnmount(() => {
  disposed = true
  if (mountTimer) {
    clearTimeout(mountTimer)
    mountTimer = null
  }
  disconnect()
  if (reconnectTimer) {
    clearTimeout(reconnectTimer)
  }
  window.removeEventListener('resize', fit)
  term?.dispose()
  term = null
})

defineExpose({ connect, disconnect, fit, focus: focusTerminal })
</script>

<template>
  <div class="overflow-hidden rounded-lg border border-slate-700">
    <!-- 终端状态栏 -->
    <div class="flex items-center justify-between bg-slate-800 px-3 py-1.5">
      <div class="flex items-center gap-2 text-xs">
        <span class="h-2 w-2 rounded-full" :class="connected ? 'bg-green-500' : 'bg-slate-500'" />
        <span class="text-slate-300">{{ title }}</span>
      </div>
      <div class="flex items-center gap-3">
        <span class="text-xs" :class="statusColor">{{ statusText }}</span>
        <button
          class="rounded px-1.5 py-0.5 text-xs text-slate-400 transition-colors hover:bg-slate-700 hover:text-slate-200"
          @click="toggle"
        >
          {{ connected ? '断开' : '连接' }}
        </button>
      </div>
    </div>
    <!-- xterm 容器：tabindex=0 使其可聚焦，点击/打开时自动聚焦以接收键盘输入 -->
    <div
      ref="containerRef"
      :class="height"
      class="w-full cursor-text bg-slate-900 p-2 outline-none"
      tabindex="0"
      @click="focusTerminal"
    />
  </div>
</template>

<style scoped>
:deep(.xterm) {
  height: 100%;
}
</style>
