<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import { GridComponent, LegendComponent, TooltipComponent } from 'echarts/components'
import VChart from 'vue-echarts'
import { Server, Boxes, User, Users, Layers, CircleDot, ListChecks, RefreshCw, Activity, Network, FileCode2, Rocket, BookOpen, Cpu } from '@lucide/vue'
import Card from '@/components/ui/Card.vue'
import Badge from '@/components/ui/Badge.vue'
import Button from '@/components/ui/Button.vue'
import DataTable, { type DataTableColumn } from '@/components/ui/DataTable.vue'
import { k8sOverview, type K8sOverview } from '@/api/k8s'
import { listNodeMetrics } from '@/api/docker'
import { nodeReadyVariant } from '@/utils/k8s'

use([CanvasRenderer, LineChart, GridComponent, LegendComponent, TooltipComponent])

const router = useRouter()

const loading = ref(false)
const errorMsg = ref('')
const data = ref<K8sOverview | null>(null)

async function load(): Promise<void> {
  loading.value = true
  errorMsg.value = ''
  try {
    data.value = await k8sOverview()
  } catch (err) {
    errorMsg.value = (err as Error).message
    data.value = null
  } finally {
    loading.value = false
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
    const rows = await listNodeMetrics('k8s', 120)
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

/** K8s 是否可用。 */
const available = computed(() => data.value?.status?.available ?? false)

/** 统计卡配置。 */
const statCards = computed(() => [
  { key: 'node_count', label: '节点', value: summary.value?.node_count ?? 0, icon: Server, color: 'text-blue-600' },
  { key: 'master_count', label: '控制节点', value: summary.value?.master_count ?? 0, icon: User, color: 'text-purple-600' },
  { key: 'worker_count', label: '工作节点', value: summary.value?.worker_count ?? 0, icon: Users, color: 'text-cyan-600' },
  { key: 'pod_count', label: 'Pod', value: summary.value?.pod_count ?? 0, icon: Layers, color: 'text-green-600' },
  { key: 'service_count', label: 'Service', value: summary.value?.service_count ?? 0, icon: Network, color: 'text-emerald-600' },
  { key: 'deployment_count', label: 'Deployment', value: summary.value?.deployment_count ?? 0, icon: Boxes, color: 'text-amber-600' },
  { key: 'statefulset_count', label: 'StatefulSet', value: summary.value?.statefulset_count ?? 0, icon: ListChecks, color: 'text-rose-600' },
  { key: 'daemonset_count', label: 'DaemonSet', value: summary.value?.daemonset_count ?? 0, icon: Activity, color: 'text-indigo-600' },
])

const nodeColumns: DataTableColumn[] = [
  { label: '名称', key: 'name' },
  { label: '角色', key: 'role', width: '90px' },
  { label: '状态', key: 'status', width: '100px' },
  { label: '版本', key: 'version', width: '90px' },
  { label: '地址', key: 'internal_ip', width: '130px' },
]
</script>

<template>
  <div class="space-y-5">
    <!-- 集群信息条（轻量，不复用 Layout 状态栏） -->
    <div v-if="available" class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex flex-wrap items-center gap-2 text-sm text-slate-500 dark:text-slate-400">
        <Badge variant="green" dot>集群可用</Badge>
        <span>{{ data?.status?.platform || '—' }}</span>
        <span class="text-slate-300 dark:text-slate-600">·</span>
        <span>{{ data?.status?.git_version || '—' }}</span>
        <span class="text-slate-300 dark:text-slate-600">·</span>
        <span>节点 {{ summary?.node_count ?? 0 }}（就绪 {{ summary?.nodes_ready ?? 0 }}）</span>
      </div>
      <div class="flex items-center gap-2">
        <Button variant="secondary" size="sm" :loading="loading" @click="load">
          <RefreshCw class="h-3.5 w-3.5" />
          刷新
        </Button>
        <Button size="sm" @click="router.push('/k8s/pods')">
          <Layers class="h-3.5 w-3.5" />
          查看 Pod
        </Button>
      </div>
    </div>

    <!-- 集群不可用：引导 -->
    <div
      v-if="!available"
      class="rounded-xl border border-amber-200 bg-amber-50/60 p-5 dark:border-amber-900/50 dark:bg-amber-900/10"
    >
      <div class="flex items-center gap-2">
        <Rocket class="h-5 w-5 text-amber-600 dark:text-amber-400" />
        <h3 class="text-sm font-semibold text-slate-800 dark:text-slate-100">Kubernetes 集群不可用</h3>
        <Badge variant="yellow">未连接</Badge>
      </div>
      <p class="mt-2 text-sm leading-relaxed text-slate-600 dark:text-slate-300">
        后端未检测到可用的 Kubernetes 集群。请确认本机 <code class="font-mono">kubectl config current-context</code> 指向可用集群，
        或在 <code class="font-mono">config.yaml</code> 的 <code class="font-mono">k8s</code> 段配置 <code class="font-mono">kubeconfig</code> 连接远程集群后重启服务。
      </p>
      <div class="mt-4 flex flex-wrap items-center gap-2">
        <Button variant="secondary" size="sm" @click="load">
          <RefreshCw class="h-3.5 w-3.5" />
          重新检测
        </Button>
        <a
          href="https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/"
          target="_blank"
          rel="noopener noreferrer"
          class="inline-flex h-8 items-center gap-1.5 rounded-md border border-slate-200 px-3 text-sm text-slate-600 transition-colors hover:bg-slate-100 dark:border-slate-600 dark:text-slate-300 dark:hover:bg-slate-700"
        >
          <BookOpen class="h-3.5 w-3.5" />
          kubeconfig 文档
        </a>
      </div>
    </div>

    <!-- 集群可用 -->
    <template v-if="available">
      <!-- 统计卡 -->
      <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <div
          v-for="c in statCards"
          :key="c.key"
          class="flex items-center gap-3 rounded-xl border border-slate-200 bg-white px-4 py-3.5 shadow-sm dark:border-slate-700 dark:bg-slate-800"
        >
          <component :is="c.icon" class="h-7 w-7 shrink-0" :class="c.color" />
          <div class="min-w-0">
            <p class="text-xl font-semibold leading-tight text-slate-800 dark:text-slate-100">{{ c.value }}</p>
            <p class="truncate text-xs text-slate-500 dark:text-slate-400">{{ c.label }}</p>
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

      <!-- 阶段分布 + 资源概览 -->
      <div class="grid gap-4 lg:grid-cols-3">
        <Card title="Pod 阶段分布" class="lg:col-span-2">
          <div v-if="summary" class="flex flex-wrap gap-2">
            <span
              v-for="(cnt, phase) in summary.pods_by_phase"
              :key="phase"
              class="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 bg-slate-50 px-3 py-1.5 text-sm dark:border-slate-700 dark:bg-slate-700/40"
            >
              <CircleDot class="h-3.5 w-3.5" :class="phase === 'Running' ? 'text-green-500' : 'text-yellow-500'" />
              <span class="text-slate-600 dark:text-slate-300">{{ phase }}</span>
              <span class="font-semibold text-slate-800 dark:text-slate-100">{{ cnt }}</span>
            </span>
            <span v-if="Object.keys(summary.pods_by_phase).length === 0" class="text-sm text-slate-400">暂无 Pod</span>
          </div>
        </Card>

        <Card title="资源概览" padded>
          <div class="grid grid-cols-2 gap-3">
            <div class="rounded-lg bg-slate-50 px-3 py-2.5 dark:bg-slate-700/40">
              <p class="flex items-center gap-1.5 text-xs text-slate-400"><FileCode2 class="h-3.5 w-3.5" />命名空间</p>
              <p class="mt-1 text-xl font-semibold text-slate-800 dark:text-slate-100">{{ summary?.namespace_count ?? 0 }}</p>
            </div>
            <div class="rounded-lg bg-slate-50 px-3 py-2.5 dark:bg-slate-700/40">
              <p class="flex items-center gap-1.5 text-xs text-slate-400"><ListChecks class="h-3.5 w-3.5" />StatefulSet</p>
              <p class="mt-1 text-xl font-semibold text-slate-800 dark:text-slate-100">{{ summary?.statefulset_count ?? 0 }}</p>
            </div>
            <div class="rounded-lg bg-slate-50 px-3 py-2.5 dark:bg-slate-700/40">
              <p class="flex items-center gap-1.5 text-xs text-slate-400"><Activity class="h-3.5 w-3.5" />DaemonSet</p>
              <p class="mt-1 text-xl font-semibold text-slate-800 dark:text-slate-100">{{ summary?.daemonset_count ?? 0 }}</p>
            </div>
            <div class="rounded-lg bg-slate-50 px-3 py-2.5 dark:bg-slate-700/40">
              <p class="flex items-center gap-1.5 text-xs text-slate-400"><CircleDot class="h-3.5 w-3.5 text-green-500" />就绪节点</p>
              <p class="mt-1 text-xl font-semibold text-slate-800 dark:text-slate-100">
                {{ summary?.nodes_ready ?? 0 }}<span class="text-sm text-slate-400">/{{ summary?.node_count ?? 0 }}</span>
              </p>
            </div>
          </div>
        </Card>
      </div>

      <!-- 节点简表 -->
      <DataTable
        title="节点"
        :columns="nodeColumns"
        :data="data?.nodes ?? []"
        :loading="loading"
        :error="errorMsg"
        row-key="name"
        empty-text="暂无节点"
        @row-click="() => router.push('/k8s/nodes')"
      >
        <template #cell-status="{ row }">
          <Badge :variant="nodeReadyVariant((row as { ready: boolean }).ready)" dot>
            {{ (row as { status: string }).status }}
          </Badge>
        </template>
        <template #cell-role="{ row }">
          <Badge variant="blue">{{ (row as { role: string }).role }}</Badge>
        </template>
      </DataTable>
    </template>
  </div>
</template>
