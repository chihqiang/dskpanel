<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import { GridComponent, LegendComponent, TooltipComponent } from 'echarts/components'
import VChart from 'vue-echarts'
import { Server, Cpu, MemoryStick, Box, Container, Image, Network, HardDrive, CircleDot, Trash2, Hash, RefreshCw, TerminalSquare, Layers, Tag, CircuitBoard, Folder, Rocket, Power } from '@lucide/vue'
import Card from '@/components/ui/Card.vue'
import Badge from '@/components/ui/Badge.vue'
import Button from '@/components/ui/Button.vue'
import { useDockerDetect } from '@/composables/useDocker'
import { useConfirm } from '@/composables/useConfirm'
import { useToast } from '@/composables/useToast'
import { fmtSize } from '@/utils/format'
import { containerStateVariant } from '@/utils/docker'
import Tooltip from '@/components/ui/Tooltip.vue'
import { dockerOverview, dockerSystemInfo, pruneAllDocker, listNodeMetrics, type DockerStats, type DockerVersion, type DockerSystemInfo } from '@/api/docker'
import { getDiskUsage } from '@/api/image'

use([CanvasRenderer, LineChart, GridComponent, LegendComponent, TooltipComponent])

const router = useRouter()
const { dockerInfo, dockerChecked, dockerChecking, detect } = useDockerDetect()
const confirm = useConfirm()
const toast = useToast()

const stats = ref<DockerStats | null>(null)
const diskUsage = ref<Awaited<ReturnType<typeof getDiskUsage>> | null>(null)

// 引擎版本信息。
const version = ref<DockerVersion | null>(null)
const versionError = ref('')
const versionLoading = ref(false)

// 引擎完整信息（docker info，用于概览指标补充）。
const sysInfo = ref<DockerSystemInfo | null>(null)

// 磁盘占用历史趋势（metric 采集）。
const diskTrend = ref<{ time: string; storage: number }[]>([])
const diskTrendLoading = ref(false)
// CPU / 内存历史趋势（metric 采集）。
const cpuMemTrend = ref<{ time: string; cpu: number; memory: number }[]>([])
const cpuMemTrendLoading = ref(false)
const pruning = ref(false)

/** 一键清理未使用资源。 */
function openPruneAll(): void {
  void confirm(
    '一键清理',
    '将清理所有未使用资源：已停止的容器、悬空镜像、未使用网络、匿名卷、构建缓存。此操作不可撤销，确定继续？',
    async () => {
      pruning.value = true
      try {
        const res = await pruneAllDocker()
        toast.success(
          `清理完成：${res.total.deleted} 项，释放 ${fmtSize(res.total.reclaimed)}`,
        )
        await loadOverview()
        await loadDiskTrend()
        await loadCpuMemTrend()
      } catch (err) {
        toast.error((err as Error).message)
      } finally {
        pruning.value = false
      }
    },
    { danger: true },
  )
}

const cpuMemTrendOption = computed(() => ({
  tooltip: {
    trigger: 'axis',
  },
  legend: {
    data: ['CPU (m核)', '内存 (MB)'],
    textStyle: { fontSize: 11, color: '#94a3b8' },
    top: 0,
  },
  grid: { left: 8, right: 8, top: 32, bottom: 8, containLabel: true },
  xAxis: {
    type: 'category',
    data: cpuMemTrend.value.map((d) => d.time),
    axisLine: { lineStyle: { color: '#e2e8f0' } },
    axisLabel: { fontSize: 10, color: '#94a3b8' },
  },
  yAxis: [
    {
      type: 'value',
      name: 'CPU (m核)',
      nameTextStyle: { fontSize: 10, color: '#94a3b8' },
      axisLabel: { fontSize: 10, color: '#94a3b8' },
      splitLine: { lineStyle: { color: '#f1f5f9' } },
    },
    {
      type: 'value',
      name: '内存 (MB)',
      nameTextStyle: { fontSize: 10, color: '#94a3b8' },
      axisLabel: { fontSize: 10, color: '#94a3b8', formatter: (v: number) => `${(v / 1024).toFixed(0)}M` },
      splitLine: { show: false },
    },
  ],
  series: [
    {
      name: 'CPU (m核)',
      type: 'line',
      data: cpuMemTrend.value.map((d) => d.cpu),
      smooth: true,
      symbol: 'none',
      lineStyle: { width: 2, color: '#3b82f6' },
      areaStyle: { opacity: 0.08 },
    },
    {
      name: '内存 (MB)',
      type: 'line',
      yAxisIndex: 1,
      data: cpuMemTrend.value.map((d) => d.memory),
      smooth: true,
      symbol: 'none',
      lineStyle: { width: 2, color: '#10b981' },
      areaStyle: { opacity: 0.08 },
    },
  ],
}))

const diskTrendOption = computed(() => ({
  tooltip: {
    trigger: 'axis',
    valueFormatter: (v: number) => `${(v / 1024 / 1024).toFixed(2)} GB`,
  },
  legend: { show: false },
  grid: { left: 8, right: 8, top: 16, bottom: 8, containLabel: true },
  xAxis: {
    type: 'category',
    data: diskTrend.value.map((d) => d.time),
    axisLine: { lineStyle: { color: '#e2e8f0' } },
    axisLabel: { fontSize: 10, color: '#94a3b8' },
  },
  yAxis: {
    type: 'value',
    axisLabel: { fontSize: 10, color: '#94a3b8', formatter: (v: number) => `${(v / 1024 / 1024).toFixed(1)}G` },
    splitLine: { lineStyle: { color: '#f1f5f9' } },
  },
  series: [
    {
      type: 'line',
      data: diskTrend.value.map((d) => d.storage),
      smooth: true,
      symbol: 'none',
      lineStyle: { width: 2, color: '#8b5cf6' },
      areaStyle: { opacity: 0.08 },
    },
  ],
}))

async function loadDiskTrend(): Promise<void> {
  diskTrendLoading.value = true
  try {
    const rows = await listNodeMetrics('docker', 120)
    diskTrend.value = rows.map((r) => ({
      time: new Date(r.time).toLocaleTimeString('zh-CN', { hour12: false }),
      storage: Number(r.storage) || 0,
    }))
  } catch {
    diskTrend.value = []
  } finally {
    diskTrendLoading.value = false
  }
}

async function loadCpuMemTrend(): Promise<void> {
  cpuMemTrendLoading.value = true
  try {
    const rows = await listNodeMetrics('docker', 120)
    cpuMemTrend.value = rows.map((r) => ({
      time: new Date(r.time).toLocaleTimeString('zh-CN', { hour12: false }),
      cpu: Number(r.cpu) || 0,
      memory: Number(r.memory) || 0,
    }))
  } catch {
    cpuMemTrend.value = []
  } finally {
    cpuMemTrendLoading.value = false
  }
}

onMounted(async () => {
  await detect()
  if (dockerInfo.value?.available) {
    loadOverview()
    loadDiskTrend()
    loadCpuMemTrend()
  }
})

async function loadOverview(): Promise<void> {
  versionLoading.value = true
  versionError.value = ''
  try {
    // 一次请求：资源统计 + 引擎版本。
    const data = await dockerOverview()
    stats.value = data.stats
    version.value = data.version
    diskUsage.value = await getDiskUsage().catch(() => null)
    // 引擎完整信息（容错：失败不影响概览主体）。
    sysInfo.value = await dockerSystemInfo().catch(() => null)
  } catch (err) {
    versionError.value = (err as Error).message
    stats.value = null
    version.value = null
  } finally {
    versionLoading.value = false
  }
}

const memoryGB = computed(() => {
  if (!dockerInfo.value) return '-'
  return (dockerInfo.value.memory / 1024 / 1024 / 1024).toFixed(1) + ' GB'
})

// 顶部概览指标（2 行 × 4 卡）：核心信息 + 引擎环境关键信息。
const statCards = computed(() => [
  { label: '版本', value: dockerInfo.value?.version || '-', icon: Server },
  { label: 'CPU', value: dockerInfo.value ? `${dockerInfo.value.cpu} 核` : '-', icon: Cpu },
  { label: '内存', value: memoryGB.value, icon: MemoryStick },
  { label: '架构', value: dockerInfo.value?.arch || '-', icon: Box },
  { label: '存储驱动', value: sysInfo.value?.driver || '-', icon: Layers, mono: true },
  { label: '内核', value: sysInfo.value?.kernel_version || '-', icon: CircuitBoard, mono: true, small: true },
  { label: '主机名', value: sysInfo.value?.name || '-', icon: Tag, mono: true, small: true },
  { label: '数据目录', value: sysInfo.value?.docker_root_dir || '-', icon: Folder, mono: true, small: true },
])

/** 资源统计卡片。 */
const resourceCards = computed(() => [
  { label: '容器', value: stats.value?.containers ?? '-', icon: Container, color: 'bg-blue-50 text-blue-600 dark:bg-blue-900/40 dark:text-blue-400' },
  { label: '镜像', value: stats.value?.images ?? '-', icon: Image, color: 'bg-green-50 text-green-600 dark:bg-green-900/40 dark:text-green-400' },
  { label: '网络', value: stats.value?.networks ?? '-', icon: Network, color: 'bg-purple-50 text-purple-600 dark:bg-purple-900/40 dark:text-purple-400' },
  { label: '卷', value: stats.value?.volumes ?? '-', icon: HardDrive, color: 'bg-amber-50 text-amber-600 dark:bg-amber-900/40 dark:text-amber-400' },
])

/** 容器状态分布（按数量降序）。 */
const stateItems = computed(() => {
  const by = stats.value?.by_state ?? {}
  return Object.entries(by)
    .sort((a, b) => b[1] - a[1])
    .map(([state, count]) => ({ state, count }))
})

/** 点击某个状态 → 跳到容器列表并按状态过滤。 */
function goState(state: string): void {
  router.push({ path: '/docker/containers', query: { state } })
}

interface DiskUsageRow {
  label: string
  icon: typeof Container
  active: number
  total: number
  totalSize: number
  reclaimable: number
}

const diskRows = computed<DiskUsageRow[]>(() => {
  const d = diskUsage.value
  if (!d) return []
  return [
    {
      label: '容器',
      icon: Container,
      active: d.containers.active_count,
      total: d.containers.total_count,
      totalSize: d.containers.total_size,
      reclaimable: d.containers.reclaimable,
    },
    {
      label: '镜像',
      icon: Image,
      active: d.images.active_count,
      total: d.images.total_count,
      totalSize: d.images.total_size,
      reclaimable: d.images.reclaimable,
    },
    {
      label: '构建缓存',
      icon: Box,
      active: d.build_cache.active_count,
      total: d.build_cache.total_count,
      totalSize: d.build_cache.total_size,
      reclaimable: d.build_cache.reclaimable,
    },
    {
      label: '卷',
      icon: HardDrive,
      active: d.volumes.active_count,
      total: d.volumes.total_count,
      totalSize: d.volumes.total_size,
      reclaimable: d.volumes.reclaimable,
    },
  ]
})
</script>

<template>
  <div class="space-y-5">
    <!-- 引擎状态与版本信息 -->
    <Card>
      <div class="flex flex-wrap items-center gap-x-6 gap-y-2">
        <span
          class="h-3 w-3 shrink-0 rounded-full"
          :class="dockerInfo?.available ? 'bg-green-500' : dockerChecking ? 'bg-yellow-400 animate-pulse' : 'bg-red-400'"
        />
        <div class="min-w-0">
          <div class="flex flex-wrap items-center gap-2">
            <p class="text-base font-medium text-slate-800 dark:text-slate-100">
              {{
                dockerChecking
                  ? '正在检测本机 Docker...'
                  : dockerInfo?.available
                    ? `本机 Docker 可用（${dockerInfo.platform}）`
                    : '未检测到本机 Docker 环境'
              }}
            </p>
            <Badge v-if="dockerInfo?.available" variant="green">可用</Badge>
            <Badge v-else-if="dockerChecked" variant="red">不可用</Badge>
          </div>
          <!-- 版本元信息（平台 / API / SDK / 实验特性） -->
          <div
            v-if="version"
            class="mt-1.5 flex flex-wrap items-center gap-x-6 gap-y-1 text-sm text-slate-500 dark:text-slate-400"
          >
            <span class="flex items-center gap-1.5"><Box class="h-4 w-4" />{{ version.platform_name }}</span>
            <span class="flex items-center gap-1.5"><Hash class="h-4 w-4" />API {{ version.api_version }}</span>
            <span class="flex items-center gap-1.5"><TerminalSquare class="h-4 w-4" />SDK {{ version.client_version }}</span>
            <span v-if="version.experimental" class="flex items-center gap-1.5 text-amber-600 dark:text-amber-400">
              <CircleDot class="h-4 w-4" />实验特性开启
            </span>
          </div>
          <p v-if="dockerInfo?.error" class="mt-1 text-sm text-red-500">{{ dockerInfo.error }}</p>
          <p v-else-if="versionError" class="mt-1 text-sm text-red-600">{{ versionError }}</p>
        </div>
        <Button variant="ghost" size="sm" class="ml-auto" :loading="versionLoading" @click="loadOverview">
          <RefreshCw class="mr-1 h-3.5 w-3.5" />刷新
        </Button>
        <Button
          v-if="dockerInfo?.available"
          variant="ghost"
          size="sm"
          class="text-red-600!"
          :loading="pruning"
          @click="openPruneAll"
        >
          <Trash2 class="mr-1 h-3.5 w-3.5" />一键清理
        </Button>
      </div>
    </Card>

    <!-- 概览指标（Docker 可用时显示） -->
    <template v-if="dockerInfo?.available">
      <div class="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-4">
        <Card v-for="s in statCards" :key="s.label">
          <div class="flex items-center gap-4">
            <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-lg bg-blue-50 text-blue-600 dark:bg-blue-900/40 dark:text-blue-400">
              <component :is="s.icon" class="h-6 w-6" />
            </div>
            <div class="min-w-0">
              <p class="text-sm text-slate-500 dark:text-slate-400">{{ s.label }}</p>
              <Tooltip :text="s.value" placement="top">
                <p
                  class="truncate font-semibold text-slate-800 dark:text-slate-100"
                  :class="s.small ? 'text-base' : 'text-2xl'"
                ><span v-if="s.mono" class="font-mono">{{ s.value }}</span><template v-else>{{ s.value }}</template></p>
              </Tooltip>
            </div>
          </div>
        </Card>
      </div>
    </template>

    <!-- Docker 未检测到：引导说明（参考 Swarm 概览的未启用引导） -->
    <div
      v-else-if="dockerChecked"
      class="rounded-xl border border-amber-200 bg-amber-50/60 p-5 dark:border-amber-900/50 dark:bg-amber-900/10"
    >
      <div class="flex items-center gap-2">
        <Rocket class="h-5 w-5 text-amber-600 dark:text-amber-400" />
        <h3 class="text-sm font-semibold text-slate-800 dark:text-slate-100">未检测到本机 Docker 环境</h3>
        <Badge variant="yellow">未连接</Badge>
      </div>
      <p class="mt-2 text-sm leading-relaxed text-slate-600 dark:text-slate-300">
        当前无法连接本机 Docker 守护进程，因此容器、镜像、网络、卷、编排等功能已在侧边栏隐藏。请确认 Docker 已安装并正在运行，然后点击「刷新」重试。
      </p>

      <div class="mt-4">
        <p class="mb-1.5 flex items-center gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
          <Power class="h-4 w-4 text-green-600 dark:text-green-400" />
          检查 Docker 是否正常运行
        </p>
        <code class="block w-fit overflow-x-auto rounded-md bg-slate-900 px-3 py-2 font-mono text-xs text-green-300">
          docker version
        </code>
        <p class="mt-1.5 text-xs text-slate-500 dark:text-slate-400">
          macOS 请确认已安装并启动 Docker Desktop；Linux 可执行 <code class="font-mono">sudo systemctl start docker</code>。
        </p>
      </div>
    </div>

    <!-- 资源统计（Docker 可用时显示） -->
    <template v-if="dockerInfo?.available">
      <div class="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-4">
        <Card v-for="r in resourceCards" :key="r.label">
          <div class="flex items-center gap-4">
            <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-lg" :class="r.color">
              <component :is="r.icon" class="h-6 w-6" />
            </div>
            <div class="min-w-0">
              <p class="text-sm text-slate-500 dark:text-slate-400">{{ r.label }}</p>
              <p class="truncate text-2xl font-semibold text-slate-800 dark:text-slate-100">{{ r.value }}</p>
            </div>
          </div>
        </Card>
      </div>

      <!-- 磁盘占用汇总（docker system df） -->
      <Card v-if="diskRows.length > 0">
        <div class="mb-3 flex items-center gap-2">
          <Trash2 class="h-4 w-4 text-slate-400" />
          <span class="text-sm font-medium text-slate-700 dark:text-slate-200">磁盘占用汇总</span>
        </div>
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <div
            v-for="row in diskRows"
            :key="row.label"
            class="rounded-lg border border-slate-200 p-3 dark:border-slate-700"
          >
            <div class="flex items-center gap-2 text-sm text-slate-500 dark:text-slate-400">
              <component :is="row.icon" class="h-4 w-4" />
              <span>{{ row.label }}</span>
              <span class="ml-auto text-xs text-slate-400">{{ row.active }} 使用中 / {{ row.total }} 总计</span>
            </div>
            <p class="mt-2 text-xl font-semibold text-slate-800 dark:text-slate-100">{{ fmtSize(row.totalSize) }}</p>
            <p class="mt-1 text-xs">
              <span class="text-slate-400">可回收</span>
              <span class="ml-1 font-mono font-medium text-red-500">{{ fmtSize(row.reclaimable) }}</span>
            </p>
          </div>
        </div>
      </Card>

      <!-- CPU / 内存趋势（metric 采集历史） -->
      <Card v-if="cpuMemTrend.length > 0">
        <div class="mb-3 flex items-center justify-between gap-x-4 gap-y-2">
          <div class="flex items-center gap-2">
            <Cpu class="h-4 w-4 text-slate-400" />
            <span class="text-sm font-medium text-slate-700 dark:text-slate-200">CPU / 内存趋势</span>
            <span class="text-xs text-slate-400">（metric 每 {{ '60' }}s 采集，{{ cpuMemTrend.length }} 个点）</span>
          </div>
          <Button variant="ghost" size="sm" :loading="cpuMemTrendLoading" @click="loadCpuMemTrend">
            <RefreshCw class="mr-1 h-3.5 w-3.5" />刷新
          </Button>
        </div>
        <VChart class="w-full" style="height: 180px" :option="cpuMemTrendOption" autoresize />
      </Card>

      <!-- 磁盘占用趋势（metric 采集历史） -->
      <Card v-if="diskTrend.length > 0">
        <div class="mb-3 flex items-center justify-between gap-x-4 gap-y-2">
          <div class="flex items-center gap-2">
            <HardDrive class="h-4 w-4 text-slate-400" />
            <span class="text-sm font-medium text-slate-700 dark:text-slate-200">磁盘占用趋势</span>
            <span class="text-xs text-slate-400">（metric 每 {{ '60' }}s 采集，{{ diskTrend.length }} 个点）</span>
          </div>
          <Button variant="ghost" size="sm" :loading="diskTrendLoading" @click="loadDiskTrend">
            <RefreshCw class="mr-1 h-3.5 w-3.5" />刷新
          </Button>
        </div>
        <VChart class="w-full" style="height: 180px" :option="diskTrendOption" autoresize />
      </Card>

      <!-- 容器状态分布 -->
      <Card v-if="stateItems.length > 0">
        <div class="mb-3 flex items-center gap-2">
          <CircleDot class="h-4 w-4 text-slate-400" />
          <span class="text-sm font-medium text-slate-700 dark:text-slate-200">容器状态分布</span>
        </div>
        <div class="flex flex-wrap gap-2">
          <Tooltip
            v-for="item in stateItems"
            :key="item.state"
            :text="`查看所有「${item.state}」状态的容器`"
            placement="top"
            as="button"
          >
            <span
              class="inline-flex cursor-pointer items-center gap-1.5 rounded-md bg-slate-100 px-2.5 py-1 text-sm text-slate-700 transition-colors hover:bg-blue-50 hover:text-blue-700 dark:bg-slate-700 dark:text-slate-200 dark:hover:bg-blue-900/40 dark:hover:text-blue-300"
              @click="goState(item.state)"
            >
              <Badge :variant="containerStateVariant(item.state)">{{ item.state }}</Badge>
              <span class="font-semibold">{{ item.count }}</span>
            </span>
          </Tooltip>
        </div>
      </Card>
    </template>
  </div>
</template>
