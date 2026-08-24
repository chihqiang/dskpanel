<script setup lang="ts">
import { onBeforeUnmount, reactive, ref, watch } from 'vue'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import { GridComponent, LegendComponent, TooltipComponent } from 'echarts/components'
import VChart from 'vue-echarts'
import Modal from '@/components/ui/Modal.vue'
import Button from '@/components/ui/Button.vue'
import { getContainerStats, getContainerTop, type ContainerStats, type ContainerTop } from '@/api/container'
import { fmtSize } from '@/utils/format'

// 按需注册 ECharts 模块。
use([CanvasRenderer, LineChart, GridComponent, LegendComponent, TooltipComponent])

const props = defineProps<{ open: boolean; containerId: string; containerName?: string }>()

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()

const loading = ref(false)
const errorMsg = ref('')
const latest = ref<ContainerStats | null>(null)

// 进程列表（监控内展示）。
const topLoading = ref(false)
const topError = ref('')
const topData = ref<ContainerTop | null>(null)

async function loadTop(): Promise<void> {
  topLoading.value = true
  topError.value = ''
  try {
    topData.value = await getContainerTop(props.containerId)
  } catch (err) {
    topError.value = (err as Error).message
  } finally {
    topLoading.value = false
  }
}

// 曲线数据（reactive，最多 60 点）——数组需为响应式，push 才能触发 ECharts 更新。
const chart = reactive<{
  times: string[]
  cpu: number[]
  mem: number[]
  rx: number[]
  tx: number[]
  br: number[]
  bw: number[]
}>({ times: [], cpu: [], mem: [], rx: [], tx: [], br: [], bw: [] })

let timer: ReturnType<typeof setInterval> | null = null
let stopped = false

function push<T>(arr: T[], v: T): void {
  arr.push(v)
  if (arr.length > 60) arr.shift()
}

function now(): string {
  return new Date().toLocaleTimeString('zh-CN', { hour12: false })
}

async function poll(): Promise<void> {
  if (stopped) return
  try {
    const s = await getContainerStats(props.containerId)
    if (stopped) return
    latest.value = s
    push(chart.times, now())
    push(chart.cpu, Number(s.cpu_percent.toFixed(2)))
    push(chart.mem, Number(s.mem_percent.toFixed(2)))
    push(chart.rx, s.net_rx_bytes)
    push(chart.tx, s.net_tx_bytes)
    push(chart.br, s.block_read)
    push(chart.bw, s.block_write)
  } catch (err) {
    errorMsg.value = (err as Error).message
  }
}

watch(
  () => props.open,
  (open) => {
    if (open) {
      stopped = false
      loading.value = true
      errorMsg.value = ''
      latest.value = null
      chart.times = []
      chart.cpu = []
      chart.mem = []
      chart.rx = []
      chart.tx = []
      chart.br = []
      chart.bw = []
      void poll()
      void loadTop()
      timer = setInterval(poll, 1000)
      setTimeout(() => {
        loading.value = false
      }, 300)
    } else {
      stop()
    }
  },
  { immediate: true },
)

function stop(): void {
  stopped = true
  if (timer) {
    clearInterval(timer)
    timer = null
  }
}

onBeforeUnmount(stop)

// ECharts option 用 reactive 就地更新（避免每次返回新对象触发 setOption 警告）。
const baseAxis = {
  type: 'category' as const,
  boundaryGap: false as const,
  axisLabel: { fontSize: 10, color: '#94a3b8' },
  axisLine: { lineStyle: { color: '#e2e8f0' } },
}
const baseValueAxis = {
  type: 'value' as const,
  scale: true as const,
  splitLine: { lineStyle: { color: '#f1f5f9' } },
  axisLabel: { fontSize: 10, color: '#94a3b8' },
}

const cpuOption = reactive({
  tooltip: { trigger: 'axis' },
  grid: { left: 42, right: 12, top: 18, bottom: 24 },
  xAxis: { ...baseAxis, data: chart.times },
  yAxis: { ...baseValueAxis, axisLabel: { fontSize: 10, color: '#94a3b8', formatter: '{value}%' } },
  series: [
    {
      name: 'CPU',
      type: 'line',
      smooth: true,
      showSymbol: false,
      lineStyle: { width: 2, color: '#3b82f6' },
      itemStyle: { color: '#3b82f6' },
      data: chart.cpu,
    },
  ],
})

const memOption = reactive({
  tooltip: { trigger: 'axis' },
  grid: { left: 42, right: 12, top: 18, bottom: 24 },
  xAxis: { ...baseAxis, data: chart.times },
  yAxis: { ...baseValueAxis, axisLabel: { fontSize: 10, color: '#94a3b8', formatter: '{value}%' } },
  series: [
    {
      name: '内存',
      type: 'line',
      smooth: true,
      showSymbol: false,
      lineStyle: { width: 2, color: '#10b981' },
      itemStyle: { color: '#10b981' },
      areaStyle: { opacity: 0.08 },
      data: chart.mem,
    },
  ],
})

const netOption = reactive({
  tooltip: { trigger: 'axis', valueFormatter: (v: number) => fmtSize(v) },
  grid: { left: 52, right: 12, top: 26, bottom: 24 },
  xAxis: { ...baseAxis, data: chart.times },
  yAxis: { ...baseValueAxis, axisLabel: { fontSize: 10, color: '#94a3b8', formatter: (v: number) => fmtSize(v) } },
  legend: { data: ['接收', '发送'], top: 0, itemWidth: 12, itemHeight: 8, textStyle: { fontSize: 10, color: '#64748b' } },
  series: [
    {
      name: '接收',
      type: 'line',
      smooth: true,
      showSymbol: false,
      lineStyle: { width: 2, color: '#8b5cf6' },
      itemStyle: { color: '#8b5cf6' },
      data: chart.rx,
    },
    {
      name: '发送',
      type: 'line',
      smooth: true,
      showSymbol: false,
      lineStyle: { width: 2, color: '#f59e0b' },
      itemStyle: { color: '#f59e0b' },
      data: chart.tx,
    },
  ],
})

const diskOption = reactive({
  tooltip: { trigger: 'axis', valueFormatter: (v: number) => fmtSize(v) },
  grid: { left: 52, right: 12, top: 26, bottom: 24 },
  xAxis: { ...baseAxis, data: chart.times },
  yAxis: { ...baseValueAxis, axisLabel: { fontSize: 10, color: '#94a3b8', formatter: (v: number) => fmtSize(v) } },
  legend: { data: ['读', '写'], top: 0, itemWidth: 12, itemHeight: 8, textStyle: { fontSize: 10, color: '#64748b' } },
  series: [
    {
      name: '读',
      type: 'line',
      smooth: true,
      showSymbol: false,
      lineStyle: { width: 2, color: '#06b6d4' },
      itemStyle: { color: '#06b6d4' },
      data: chart.br,
    },
    {
      name: '写',
      type: 'line',
      smooth: true,
      showSymbol: false,
      lineStyle: { width: 2, color: '#f43f5e' },
      itemStyle: { color: '#f43f5e' },
      data: chart.bw,
    },
  ],
})
</script>

<template>
  <Modal :open="open" @update:open="(v) => !v && emit('update:open', v)" title="容器监控" width="max-w-2xl">
    <div class="space-y-5">
      <div class="truncate text-sm text-slate-500">
        容器：<span class="font-medium text-slate-700 dark:text-slate-200">{{ containerName || containerId }}</span>
      </div>

      <p v-if="errorMsg" class="text-sm text-red-600">{{ errorMsg }}</p>

      <!-- 实时数值 -->
      <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <div class="rounded-lg border border-slate-200 p-3 dark:border-slate-700">
          <p class="text-xs text-slate-500">CPU</p>
          <p class="mt-1 font-mono text-lg font-semibold text-slate-800 dark:text-slate-100">
            {{ latest ? latest.cpu_percent.toFixed(1) : '—' }}%
          </p>
        </div>
        <div class="rounded-lg border border-slate-200 p-3 dark:border-slate-700">
          <p class="text-xs text-slate-500">内存</p>
          <p class="mt-1 font-mono text-lg font-semibold text-slate-800 dark:text-slate-100">
            {{ latest ? fmtSize(latest.mem_usage) : '—' }}
          </p>
          <p v-if="latest" class="text-[10px] text-slate-400">{{ latest.mem_percent.toFixed(1) }}%</p>
        </div>
        <div class="rounded-lg border border-slate-200 p-3 dark:border-slate-700">
          <p class="text-xs text-slate-500">网络</p>
          <p class="mt-1 font-mono text-lg font-semibold text-slate-800 dark:text-slate-100">
            {{ latest ? fmtSize(latest.net_rx_bytes + latest.net_tx_bytes) : '—' }}
          </p>
          <p v-if="latest" class="text-[10px] text-slate-400">↓ {{ fmtSize(latest.net_rx_bytes) }} ↑ {{ fmtSize(latest.net_tx_bytes) }}</p>
        </div>
        <div class="rounded-lg border border-slate-200 p-3 dark:border-slate-700">
          <p class="text-xs text-slate-500">进程 / 块IO</p>
          <p class="mt-1 font-mono text-lg font-semibold text-slate-800 dark:text-slate-100">{{ latest?.pids ?? '—' }} 个</p>
          <p v-if="latest" class="text-[10px] text-slate-400">读 {{ fmtSize(latest.block_read) }} · 写 {{ fmtSize(latest.block_write) }}</p>
        </div>
      </div>

      <!-- 曲线图（ECharts） -->
      <div v-if="chart.times.length > 0 || loading" class="space-y-4">
        <div>
          <p class="mb-1 text-xs text-slate-500">CPU 使用率</p>
          <VChart class="w-full" style="height: 120px" :option="cpuOption" autoresize />
        </div>
        <div>
          <p class="mb-1 text-xs text-slate-500">内存使用率</p>
          <VChart class="w-full" style="height: 120px" :option="memOption" autoresize />
        </div>
        <div>
          <p class="mb-1 text-xs text-slate-500">网络（累计字节）</p>
          <VChart class="w-full" style="height: 120px" :option="netOption" autoresize />
        </div>
        <div>
          <p class="mb-1 text-xs text-slate-500">磁盘 IO（累计字节）</p>
          <VChart class="w-full" style="height: 120px" :option="diskOption" autoresize />
        </div>
      </div>

      <!-- 进程列表 -->
      <div class="rounded-lg border border-slate-200 dark:border-slate-700">
        <div class="flex items-center justify-between border-b border-slate-200 px-4 py-2 dark:border-slate-700">
          <p class="text-sm font-medium text-slate-700 dark:text-slate-200">
            进程
            <span v-if="topData" class="ml-1 text-xs font-normal text-slate-400">{{ topData.procs.length }} 个</span>
          </p>
          <Button variant="ghost" size="sm" :loading="topLoading" @click="loadTop">刷新</Button>
        </div>
        <div class="max-h-64 overflow-auto">
          <p v-if="topError" class="px-4 py-3 text-sm text-red-600">{{ topError }}</p>
          <p v-else-if="topLoading && !topData" class="px-4 py-3 text-sm text-slate-400">加载中…</p>
          <table v-else-if="topData && topData.titles.length" class="w-full text-left text-xs">
            <thead class="sticky top-0 bg-slate-50 dark:bg-slate-800">
              <tr>
                <th
                  v-for="(t, i) in topData.titles"
                  :key="i"
                  class="whitespace-nowrap px-3 py-2 font-medium text-slate-500 dark:text-slate-400"
                >
                  {{ t }}
                </th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="(proc, idx) in topData.procs"
                :key="idx"
                class="border-t border-slate-100 dark:border-slate-800"
              >
                <td
                  v-for="(cell, ci) in proc"
                  :key="ci"
                  class="whitespace-nowrap px-3 py-1.5 font-mono text-slate-700 dark:text-slate-300"
                >
                  {{ cell }}
                </td>
              </tr>
            </tbody>
          </table>
          <p v-else class="px-4 py-3 text-sm text-slate-400">无进程数据</p>
        </div>
      </div>
    </div>
  </Modal>
</template>
