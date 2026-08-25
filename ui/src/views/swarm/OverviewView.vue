<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import { GridComponent, LegendComponent, TooltipComponent } from 'echarts/components'
import VChart from 'vue-echarts'
import { Server, Boxes, User, Users, Activity, KeyRound, FileCode2, RefreshCw, CircleDot, ListChecks, Copy, Check, Rocket, Power, Cpu } from '@lucide/vue'
import Card from '@/components/ui/Card.vue'
import Badge from '@/components/ui/Badge.vue'
import Button from '@/components/ui/Button.vue'
import DataTable, { type DataTableColumn } from '@/components/ui/DataTable.vue'
import { swarmOverview, swarmJoinTokens, type SwarmOverview, type JoinTokens } from '@/api/swarm'
import { listNodeMetrics } from '@/api/docker'
import { useSwarmConn } from '@/composables/useSwarmConn'
import { useClipboard } from '@/utils/clipboard'
import { swarmStateVariant } from '@/utils/docker'

use([CanvasRenderer, LineChart, GridComponent, LegendComponent, TooltipComponent])

const router = useRouter()
const { label } = useSwarmConn()
const { copy } = useClipboard()

const loading = ref(false)
const errorMsg = ref('')
const data = ref<SwarmOverview | null>(null)

/** 集群加入令牌（已启用时用于展示「加入节点」命令）。 */
const tokens = ref<JoinTokens | null>(null)

async function load(): Promise<void> {
  loading.value = true
  errorMsg.value = ''
  try {
    data.value = await swarmOverview()
  } catch (err) {
    errorMsg.value = (err as Error).message
    data.value = null
  } finally {
    loading.value = false
  }
  // 集群可用时顺带加载加入令牌（供「加入节点」命令使用）。
  if (data.value?.status?.available) {
    tokens.value = await swarmJoinTokens().catch(() => null)
  } else {
    tokens.value = null
  }
}
onMounted(async () => {
  await load()
  if (available.value) loadCpuMemTrend()
})

// CPU / 内存趋势。
const cpuMemTrend = ref<{ time: string; cpu: number; memory: number }[]>([])
const cpuMemTrendLoading = ref(false)

async function loadCpuMemTrend(): Promise<void> {
  cpuMemTrendLoading.value = true
  try {
    const rows = await listNodeMetrics('swarm', 120)
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

const cpuMemTrendOption = computed(() => ({
  tooltip: { trigger: 'axis' },
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

const summary = computed(() => data.value?.summary)

/** Swarm 是否可用（未启用或检测失败时展示启动引导）。 */
const available = computed(() => data.value?.status?.available ?? false)

/** 复制状态（用于命令块的复制按钮反馈）。 */
const copiedKey = ref('')
async function copyWithKey(cmd: string, key: string): Promise<void> {
  copiedKey.value = key
  await copy(cmd, '已复制到剪贴板', '复制失败，请手动复制')
  setTimeout(() => {
    if (copiedKey.value === key) copiedKey.value = ''
  }, 1500)
}

/** 启用 Swarm 命令。 */
const INIT_CMD = 'docker swarm init'
/** 关闭 Swarm 命令（manager 上执行将解散集群）。 */
const LEAVE_CMD = 'docker swarm leave --force'

/** 加入节点命令（worker）。 */
const joinCmd = computed(() => {
  const addr = tokens.value?.addr || '<manager-addr>:2377'
  const token = tokens.value?.worker
  return token ? `docker swarm join --token ${token} ${addr}` : ''
})

/** 统计卡配置。 */
const statCards = computed(() => [
  { key: 'node_count', label: '节点', value: summary.value?.node_count ?? 0, icon: Server, color: 'text-blue-600' },
  { key: 'manager_count', label: '管理节点', value: summary.value?.manager_count ?? 0, icon: User, color: 'text-purple-600' },
  { key: 'worker_count', label: '工作节点', value: summary.value?.worker_count ?? 0, icon: Users, color: 'text-cyan-600' },
  { key: 'service_count', label: '服务', value: summary.value?.service_count ?? 0, icon: Boxes, color: 'text-green-600' },
  { key: 'service_running', label: '运行中服务', value: summary.value?.service_running ?? 0, icon: Activity, color: 'text-emerald-600' },
  { key: 'task_count', label: '任务', value: summary.value?.task_count ?? 0, icon: ListChecks, color: 'text-amber-600' },
  { key: 'secrets_count', label: 'Secret', value: summary.value?.secrets_count ?? 0, icon: KeyRound, color: 'text-rose-600' },
  { key: 'configs_count', label: 'Config', value: summary.value?.configs_count ?? 0, icon: FileCode2, color: 'text-indigo-600' },
])

const nodeColumns: DataTableColumn[] = [
  { label: '名称', key: 'name' },
  { label: '角色', key: 'role', width: '90px' },
  { label: '状态', key: 'state', width: '100px' },
  { label: '可用性', key: 'availability', width: '100px' },
  { label: '地址', key: 'addr', width: '130px' },
]

const serviceColumns: DataTableColumn[] = [
  { label: '名称', key: 'name' },
  { label: '模式', key: 'mode', width: '100px' },
  { label: '副本', key: 'replicas', width: '90px' },
  { label: '镜像', key: 'image' },
  { label: '端口', key: 'ports', width: '180px' },
]

function fmtError(err: string | undefined): string {
  return err ? err.slice(0, 60) : '—'
}
</script>

<template>
  <div class="space-y-5">
    <!-- 集群状态横幅 -->
    <div class="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-slate-200 bg-white px-5 py-4 shadow-sm dark:border-slate-700 dark:bg-slate-800">
      <div class="flex items-center gap-3">
        <Server class="h-6 w-6 text-blue-600 dark:text-blue-400" />
        <div>
          <div class="flex items-center gap-2">
            <span class="text-base font-semibold text-slate-800 dark:text-slate-100">
              {{ data?.status?.name || label }}
            </span>
            <Badge v-if="data?.status?.available" variant="green">可用</Badge>
            <Badge v-else variant="red">不可用</Badge>
          </div>
          <p class="mt-0.5 text-sm text-slate-500 dark:text-slate-400">
            ID: {{ data?.status?.id || '—' }} · 引擎 {{ data?.status?.version || '—' }} ·
            管理 {{ data?.status?.managers ?? 0 }} / 节点 {{ data?.status?.nodes ?? 0 }}
          </p>
        </div>
      </div>
      <div class="flex items-center gap-2">
        <Button variant="secondary" size="sm" :loading="loading" @click="load">
          <RefreshCw class="h-3.5 w-3.5" />
          刷新
        </Button>
        <Button size="sm" @click="router.push('/swarm/services')">管理服务</Button>
      </div>
    </div>

    <!-- Swarm 未启用：启动引导 -->
    <div
      v-if="!available"
      class="rounded-xl border border-amber-200 bg-amber-50/60 p-5 dark:border-amber-900/50 dark:bg-amber-900/10"
    >
      <div class="flex items-center gap-2">
        <Rocket class="h-5 w-5 text-amber-600 dark:text-amber-400" />
        <h3 class="text-sm font-semibold text-slate-800 dark:text-slate-100">Swarm 集群未启用</h3>
        <Badge variant="yellow">未启动</Badge>
      </div>
      <p class="mt-2 text-sm leading-relaxed text-slate-600 dark:text-slate-300">
        当前引擎未启用 Docker Swarm 模式。请在本机（或目标 Docker 主机）执行以下命令初始化集群；初始化后点击「刷新」即可在此管理节点 / 服务 / Secret。
      </p>

      <div class="mt-4">
        <p class="mb-1.5 flex items-center gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
          <Power class="h-4 w-4 text-green-600 dark:text-green-400" />
          启用 Swarm
        </p>
        <div class="flex items-center gap-2">
          <code class="flex-1 overflow-x-auto rounded-md bg-slate-900 px-3 py-2 font-mono text-xs text-green-300">
            {{ INIT_CMD }}
          </code>
          <Button variant="secondary" size="sm" @click="copyWithKey(INIT_CMD, 'init')">
            <Check v-if="copiedKey === 'init'" class="h-3.5 w-3.5 text-green-500" />
            <Copy v-else class="h-3.5 w-3.5" />
          </Button>
        </div>
        <p class="mt-1.5 text-xs text-slate-500 dark:text-slate-400">
          多节点集群：manager 用 <code class="font-mono">docker swarm init --advertise-addr &lt;IP&gt;</code>，worker 节点到「节点」页复制加入令牌（<code class="font-mono">docker swarm join</code>）。
        </p>
      </div>
    </div>

    <!-- Swarm 已启用：关闭引导 -->
    <div
      v-else
      class="rounded-xl border border-slate-200 bg-slate-50/60 p-5 dark:border-slate-700 dark:bg-slate-800/40"
    >
      <div class="flex items-center gap-2">
        <Power class="h-5 w-5 text-red-500 dark:text-red-400" />
        <h3 class="text-sm font-semibold text-slate-800 dark:text-slate-100">Swarm 集群运行中</h3>
        <Badge variant="green">已启用</Badge>
      </div>
      <p class="mt-2 text-sm leading-relaxed text-slate-600 dark:text-slate-300">
        如需关闭 Swarm（解散集群），请在 manager 节点执行以下命令；worker 节点执行 <code class="font-mono">docker swarm leave</code> 脱离集群。
      </p>

      <!-- 加入节点 -->
      <div v-if="joinCmd" class="mt-4">
        <p class="mb-1.5 flex items-center gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
          <Rocket class="h-4 w-4 text-blue-600 dark:text-blue-400" />
          加入节点
        </p>
        <div class="flex items-center gap-2">
          <code class="flex-1 overflow-x-auto rounded-md bg-slate-900 px-3 py-2 font-mono text-xs text-blue-300">
            {{ joinCmd }}
          </code>
          <Button variant="secondary" size="sm" @click="copyWithKey(joinCmd, 'join')">
            <Check v-if="copiedKey === 'join'" class="h-3.5 w-3.5 text-green-500" />
            <Copy v-else class="h-3.5 w-3.5" />
          </Button>
        </div>
        <p class="mt-1.5 text-xs text-slate-500 dark:text-slate-400">
          在其它主机上执行该命令即可作为 worker 加入集群；manager 加入令牌见「节点」页。
        </p>
      </div>

      <!-- 关闭 Swarm -->
      <div class="mt-4">
        <p class="mb-1.5 flex items-center gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
          <Power class="h-4 w-4 text-red-500 dark:text-red-400" />
          关闭 Swarm（解散集群）
        </p>
        <div class="flex items-center gap-2">
          <code class="flex-1 overflow-x-auto rounded-md bg-slate-900 px-3 py-2 font-mono text-xs text-red-300">
            {{ LEAVE_CMD }}
          </code>
          <Button variant="secondary" size="sm" @click="copyWithKey(LEAVE_CMD, 'leave')">
            <Check v-if="copiedKey === 'leave'" class="h-3.5 w-3.5 text-green-500" />
            <Copy v-else class="h-3.5 w-3.5" />
          </Button>
        </div>
        <p class="mt-1.5 text-xs text-slate-500 dark:text-slate-400">
          在最后一个 manager 上执行 <code class="font-mono">docker swarm leave --force</code> 将解散整个集群。
        </p>
      </div>
    </div>

    <!-- 统计数据区（仅集群启用时展示） -->
    <template v-if="available">
    <!-- 统计卡 -->
    <div class="grid grid-cols-2 gap-4 sm:grid-cols-4">
      <div
        v-for="c in statCards"
        :key="c.key"
        class="flex items-center gap-3 rounded-xl border border-slate-200 bg-white p-4 shadow-sm dark:border-slate-700 dark:bg-slate-800"
      >
        <component :is="c.icon" class="h-8 w-8 shrink-0" :class="c.color" />
        <div>
          <p class="text-2xl font-semibold text-slate-800 dark:text-slate-100">{{ c.value }}</p>
          <p class="text-sm text-slate-500 dark:text-slate-400">{{ c.label }}</p>
        </div>
      </div>
    </div>

    <!-- CPU / 内存趋势 -->
    <Card v-if="cpuMemTrend.length > 0">
      <div class="mb-3 flex items-center justify-between gap-x-4 gap-y-2">
        <div class="flex items-center gap-2">
          <Cpu class="h-4 w-4 text-slate-400" />
          <span class="text-sm font-medium text-slate-700 dark:text-slate-200">CPU / 内存趋势</span>
          <span class="text-xs text-slate-400">（metric 采集，{{ cpuMemTrend.length }} 个点）</span>
        </div>
        <Button variant="ghost" size="sm" :loading="cpuMemTrendLoading" @click="loadCpuMemTrend">
          <RefreshCw class="mr-1 h-3.5 w-3.5" />刷新
        </Button>
      </div>
      <VChart class="w-full" style="height: 180px" :option="cpuMemTrendOption" autoresize />
    </Card>

    <!-- 状态分布 -->
    <div class="grid gap-4 lg:grid-cols-2">
      <Card title="节点状态分布">
        <div v-if="summary" class="flex flex-wrap gap-2">
          <button
            v-for="(cnt, state) in summary.nodes_by_state"
            :key="state"
            class="inline-flex cursor-pointer items-center gap-1.5 rounded-md bg-slate-100 px-2.5 py-1 text-sm text-slate-700 transition-colors hover:bg-blue-50 hover:text-blue-700 dark:bg-slate-700 dark:text-slate-200 dark:hover:bg-blue-900/40 dark:hover:text-blue-300"
            :title="`查看节点`"
            @click="router.push('/swarm/nodes')"
          >
            <CircleDot class="h-3.5 w-3.5" :class="state === 'ready' ? 'text-green-500' : 'text-yellow-500'" />
            <span class="text-slate-600 dark:text-slate-300">{{ state }}</span>
            <span class="font-semibold text-slate-800 dark:text-slate-100">{{ cnt }}</span>
          </button>
          <span v-if="Object.keys(summary.nodes_by_state).length === 0" class="text-sm text-slate-400">暂无节点</span>
        </div>
      </Card>
      <Card title="任务状态分布">
        <div v-if="summary" class="flex flex-wrap gap-2">
          <span
            v-for="(cnt, state) in summary.tasks_by_state"
            :key="state"
            class="flex items-center gap-1.5 rounded-full border border-slate-200 px-3 py-1 text-sm dark:border-slate-700"
          >
            <CircleDot class="h-3.5 w-3.5" :class="state === 'running' ? 'text-green-500' : 'text-slate-400'" />
            <span class="text-slate-600 dark:text-slate-300">{{ state }}</span>
            <span class="font-semibold text-slate-800 dark:text-slate-100">{{ cnt }}</span>
          </span>
          <span v-if="Object.keys(summary.tasks_by_state).length === 0" class="text-sm text-slate-400">暂无任务</span>
        </div>
      </Card>
    </div>

    <!-- 节点 / 服务简表 -->
    <div class="grid gap-4 lg:grid-cols-2">
      <DataTable
        title="节点"
        :columns="nodeColumns"
        :data="data?.nodes ?? []"
        :loading="loading"
        :error="errorMsg"
        row-key="id"
        empty-text="暂无节点"
        @row-click="() => router.push('/swarm/nodes')"
      />
      <DataTable
        title="服务"
        :columns="serviceColumns"
        :data="data?.services ?? []"
        :loading="loading"
        row-key="id"
        empty-text="暂无服务"
        @row-click="() => router.push('/swarm/services')"
      />
    </div>

    <!-- 节点状态明细（供调试参考） -->
    <Card v-if="data?.nodes?.length" title="节点详情" padded>
      <div class="space-y-2">
        <div v-for="n in data.nodes" :key="n.id" class="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-slate-100 px-4 py-2.5 dark:border-slate-700">
          <div class="flex items-center gap-2">
            <span class="font-medium text-slate-700 dark:text-slate-200">{{ n.name || n.id.slice(0, 12) }}</span>
            <Badge variant="blue">{{ n.role }}</Badge>
            <Badge :variant="swarmStateVariant(n.state)">{{ n.state }}</Badge>
            <Badge variant="purple">{{ n.availability }}</Badge>
          </div>
          <span class="text-sm text-slate-500 dark:text-slate-400">
            {{ n.addr }} · {{ n.version || '—' }} · {{ fmtError(n.engine_err) }}
          </span>
        </div>
      </div>
    </Card>
    </template>
  </div>
</template>
