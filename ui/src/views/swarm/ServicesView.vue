<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Plus, Eye, FileText, ListChecks, Scaling, Settings2, Trash2, RefreshCw, RotateCcw } from '@lucide/vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import Modal from '@/components/ui/Modal.vue'
import DataTable, { type DataTableColumn } from '@/components/ui/DataTable.vue'
import RowActions, { type RowAction } from '@/components/ui/RowActions.vue'
import { useToast } from '@/composables/useToast'
import { useConfirm } from '@/composables/useConfirm'
import { serviceStateVariant } from '@/utils/docker'
import { fmtSize } from '@/utils/format'
import Tooltip from '@/components/ui/Tooltip.vue'
import ServiceFormModal from '@/components/swarm/ServiceFormModal.vue'
import ServiceLogsModal from '@/components/swarm/ServiceLogsModal.vue'
import TaskLogsModal from '@/components/swarm/TaskLogsModal.vue'
import {
  swarmServices,
  swarmServiceInspect,
  swarmServiceResources,
  swarmScaleService,
  swarmRemoveService,
  swarmRollbackService,
  swarmForceUpdateService,
  swarmNetworks,
  swarmTasks,
  type SwarmServiceItem,
  type SwarmTaskItem,
  type SwarmContainerResource,
} from '@/api/swarm'

const toast = useToast()
const confirm = useConfirm()
const router = useRouter()

const loading = ref(false)
const errorMsg = ref('')
const items = ref<SwarmServiceItem[]>([])

// 弹窗状态。
const createOpen = ref(false)
const editTarget = ref<SwarmServiceItem | null>(null)
const editOpen = ref(false)
const logsTarget = ref<SwarmServiceItem | null>(null)
const logsOpen = ref(false)
const scaleTarget = ref<SwarmServiceItem | null>(null)
const scaleOpen = ref(false)
const scaleValue = ref(1)
const scaleSaving = ref(false)
const tasksTarget = ref<SwarmServiceItem | null>(null)
const tasksOpen = ref(false)
const tasks = ref<SwarmTaskItem[]>([])
const tasksLoading = ref(false)
const taskLogsTarget = ref<SwarmTaskItem | null>(null)
const taskLogsOpen = ref(false)
const detailOpen = ref(false)
const detail = ref('')
const detailLoading = ref(false)
// 服务资源监控（任务容器 CPU/内存）。
const serviceResources = ref<SwarmContainerResource[]>([])
const resourcesLoading = ref(false)
// 服务详情结构化展示。
const serviceDetail = ref<{
  id: string
  name: string
  mode: string
  image: string
  replicas: string
  ports: string[]
  env: string[]
  command: string[]
  restart: string
  mounts: string[]
  secrets: string[]
  configs: string[]
  networks: string[]
  labels: Record<string, string>
  constraints: string[]
  limit_cpu: string
  limit_memory: string
  update_status: string
  raw: string
} | null>(null)

async function load(): Promise<void> {
  loading.value = true
  errorMsg.value = ''
  try {
    items.value = await swarmServices()
  } catch (err) {
    errorMsg.value = (err as Error).message
    items.value = []
  } finally {
    loading.value = false
  }
}
onMounted(load)

/**
 * 创建/更新服务后轮询等待就绪。
 * 轮询直到指定服务 state === 'running' 或超时（默认 60s），期间持续刷新列表。
 */
async function waitForReady(name: string, timeoutMs = 60000): Promise<void> {
  const start = Date.now()
  while (Date.now() - start < timeoutMs) {
    await load()
    const svc = items.value.find((s) => s.name === name)
    if (svc && svc.state === 'running' && !svc.has_update) {
      toast.success(`服务「${name}」已就绪`)
      return
    }
    // 短暂间隔后继续。
    await new Promise((r) => setTimeout(r, 2000))
  }
  toast.error(`服务「${name}」等待就绪超时，请检查任务状态`)
}

/** 创建/更新成功回调：刷新 + 自动等待就绪（name 为空时仅刷新）。 */
function onServiceSaved(name: string): void {
  void load()
  if (name) {
    void waitForReady(name)
  }
}

function openCreate(): void {
  editTarget.value = null
  createOpen.value = true
}

function openEdit(row: SwarmServiceItem): void {
  editTarget.value = row
  editOpen.value = true
}

function openLogs(row: SwarmServiceItem): void {
  logsTarget.value = row
  logsOpen.value = true
}

function openScale(row: SwarmServiceItem): void {
  scaleTarget.value = row
  scaleValue.value = row.mode === 'global' ? 1 : Number(row.replicas.split('/')[0]) || 1
  scaleOpen.value = true
}

async function doScale(): Promise<void> {
  if (!scaleTarget.value) return
  scaleSaving.value = true
  try {
    await swarmScaleService(scaleTarget.value.id, scaleValue.value)
    toast.success(`服务「${scaleTarget.value.name}」已伸缩为 ${scaleValue.value} 副本`)
    scaleOpen.value = false
    await load()
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    scaleSaving.value = false
  }
}

function removeService(row: SwarmServiceItem): void {
  void confirm(
    '删除服务',
    `确认删除服务「${row.name}」？该服务的全部任务将被终止，此操作不可恢复。`,
    async () => {
      await swarmRemoveService(row.id)
      toast.success(`已删除服务「${row.name}」`)
      await load()
    },
    { danger: true },
  )
}

async function openTasks(row: SwarmServiceItem): Promise<void> {
  tasksTarget.value = row
  tasksOpen.value = true
  tasks.value = []
  tasksLoading.value = true
  try {
    tasks.value = await swarmTasks(row.id)
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    tasksLoading.value = false
  }
}

async function openDetail(row: SwarmServiceItem): Promise<void> {
  detailOpen.value = true
  serviceDetail.value = null
  detailLoading.value = true
  try {
    const raw = (await swarmServiceInspect(row.id)) as Record<string, unknown>
    const spec = (raw?.Spec ?? {}) as Record<string, unknown>
    const task = (spec?.TaskTemplate ?? {}) as Record<string, unknown>
    const cs = (task?.ContainerSpec ?? {}) as Record<string, unknown>
    const mode = (spec?.Mode ?? {}) as Record<string, unknown>
    const limits = (task?.Resources as Record<string, unknown> | undefined)?.Limits as Record<string, unknown> | undefined
    const placement = (task?.Placement ?? {}) as Record<string, unknown>
    const nws = (task?.Networks ?? []) as { Target?: string }[]
    const eps = (spec?.EndpointSpec ?? {}) as { Ports?: { PublishedPort?: number; TargetPort?: number; Protocol?: string }[] }
    const uc = (spec?.UpdateConfig ?? {}) as Record<string, unknown>
    const updateStatus = (raw?.UpdateStatus ?? {}) as Record<string, unknown>

    // 网络 ID → 名称映射。
    const netNameMap = new Map<string, string>()
    try {
      const nets = await swarmNetworks()
      for (const n of nets) netNameMap.set(n.id, n.name)
    } catch {
      // 忽略
    }

    let replicas = '—'
    if (mode.Global) {
      replicas = 'global'
    } else if (mode.Replicated) {
      replicas = String((mode.Replicated as { Replicas?: number }).Replicas ?? '—')
    }
    serviceDetail.value = {
      id: String(raw?.ID ?? ''),
      name: String(spec?.Name ?? ''),
      mode: mode.Global ? 'global' : 'replicated',
      image: String((cs?.Image as string) ?? ''),
      replicas,
      ports: (eps.Ports ?? []).map((p) => `${p.PublishedPort ?? '?'}→${p.TargetPort ?? '?'}/${p.Protocol ?? 'tcp'}`),
      env: (cs?.Env as string[] | undefined) ?? [],
      command: (cs?.Args as string[] | undefined) ?? [],
      restart: String((task?.RestartPolicy as Record<string, unknown> | undefined)?.Condition ?? ''),
      mounts: ((cs?.Mounts as { Type?: string; Source?: string; Target?: string }[] | undefined) ?? []).map(
        (m) => `${m.Type ?? 'volume'} ${m.Source ?? ''} → ${m.Target ?? ''}`,
      ),
      secrets: ((cs?.Secrets as { SecretName?: string }[] | undefined) ?? []).map((s) => s.SecretName ?? ''),
      configs: ((cs?.Configs as { ConfigName?: string }[] | undefined) ?? []).map((c) => c.ConfigName ?? ''),
      networks: (nws ?? []).map((n) => netNameMap.get(n.Target ?? '') ?? n.Target ?? '').filter(Boolean),
      labels: (spec?.Labels as Record<string, string> | undefined) ?? {},
      constraints: (placement?.Constraints as string[] | undefined) ?? [],
      limit_cpu: limits ? String((limits?.NanoCPUs as number | undefined) ?? 0) : '',
      limit_memory: limits ? String(Math.round(((limits?.MemoryBytes as number | undefined) ?? 0) / 1024 / 1024)) : '',
      update_status: String(updateStatus?.State ?? uc?.State ?? ''),
      raw: JSON.stringify(raw, null, 2),
    }
    detail.value = serviceDetail.value?.raw ?? ''
  } catch (err) {
    detail.value = `加载失败: ${(err as Error).message}`
  } finally {
    detailLoading.value = false
  }

  // 并行加载服务资源监控（任务容器 CPU/内存）。
  resourcesLoading.value = true
  serviceResources.value = []
  try {
    serviceResources.value = await swarmServiceResources(row.id)
  } catch {
    serviceResources.value = []
  } finally {
    resourcesLoading.value = false
  }
}

function rollbackService(row: SwarmServiceItem): void {
  void confirm(
    '回滚服务',
    `确认将服务「${row.name}」回滚到上一版本？`,
    async () => {
      await swarmRollbackService(row.id)
      toast.success(`服务「${row.name}」已开始回滚`)
      await load()
    },
    { danger: true },
  )
}

function forceUpdateService(row: SwarmServiceItem): void {
  void confirm(
    '强制更新',
    `确认强制更新服务「${row.name}」？将重建其全部任务（可用于恢复暂停的更新或滚动重启）。`,
    async () => {
      await swarmForceUpdateService(row.id)
      toast.success(`服务「${row.name}」已触发强制更新`)
      await load()
    },
    { danger: false },
  )
}

function buildActions(row: SwarmServiceItem): RowAction[] {
  return [
    { key: 'detail', label: '详情', icon: Eye, onClick: () => openDetail(row) },
    { key: 'logs', label: '日志', icon: FileText, onClick: () => openLogs(row) },
    { key: 'tasks', label: '任务', icon: ListChecks, onClick: () => openTasks(row) },
    {
      key: 'scale',
      label: '伸缩',
      icon: Scaling,
      disabled: row.mode === 'global',
      onClick: () => openScale(row),
    },
    { key: 'force', label: '强制更新', icon: RefreshCw, onClick: () => forceUpdateService(row) },
    { key: 'rollback', label: '回滚', icon: RotateCcw, onClick: () => rollbackService(row) },
    { key: 'update', label: '更新', icon: Settings2, onClick: () => openEdit(row) },
    { key: 'remove', label: '删除', icon: Trash2, danger: true, onClick: () => removeService(row) },
  ]
}

const columns: DataTableColumn[] = [
  { label: '名称', key: 'name' },
  { label: '模式', key: 'mode', width: '100px' },
  { label: '副本', key: 'replicas', width: '90px' },
  { label: '镜像', key: 'image' },
  { label: '端口', key: 'ports', width: '180px' },
  { label: '状态', key: 'state', width: '110px' },
  { label: '操作', key: 'actions', width: '200px', align: 'right' },
]

function taskStateVariant(s: string): 'green' | 'red' | 'yellow' | 'gray' {
  if (s === 'running') return 'green'
  if (s === 'failed' || s === 'rejected') return 'red'
  if (s === 'preparing' || s === 'pending' || s === 'starting') return 'yellow'
  return 'gray'
}
</script>

<template>
  <div>
    <DataTable
      title="服务列表"
      :columns="columns"
      :data="items"
      :loading="loading"
      :error="errorMsg"
      row-key="id"
      empty-text="暂无服务"
    >
      <template #toolbar>
        <Button variant="secondary" size="sm" :loading="loading" @click="load">
          <RefreshCw class="h-3.5 w-3.5" />
          刷新
        </Button>
        <Button size="sm" @click="openCreate">
          <Plus class="h-3.5 w-3.5" />
          创建服务
        </Button>
      </template>
      <template #cell-mode="{ row }">
        <Badge variant="blue">{{ (row as SwarmServiceItem).mode }}</Badge>
      </template>
      <template #cell-state="{ row }">
        <Badge :variant="serviceStateVariant((row as SwarmServiceItem).state)" dot>
          {{ (row as SwarmServiceItem).state }}
        </Badge>
      </template>
      <template #cell-ports="{ row }">
        <span class="font-mono text-xs">{{ (row as SwarmServiceItem).ports?.join(', ') || '—' }}</span>
      </template>
      <template #cell-actions="{ row }">
        <RowActions :actions="buildActions(row as SwarmServiceItem)" :visible="2" />
      </template>
    </DataTable>

    <!-- 创建 / 更新服务 -->
    <ServiceFormModal v-model:open="createOpen" @saved="onServiceSaved" />
    <ServiceFormModal
      v-if="editTarget"
      v-model:open="editOpen"
      :service-id="editTarget.id"
      @saved="onServiceSaved"
    />

    <!-- 日志 -->
    <ServiceLogsModal
      v-if="logsTarget"
      v-model:open="logsOpen"
      :service-id="logsTarget.id"
      :service-name="logsTarget.name"
    />

    <!-- 伸缩 -->
    <Modal
      :open="scaleOpen"
      @update:open="scaleOpen = $event"
      title="服务伸缩"
      width="max-w-sm"
    >
      <p v-if="scaleTarget" class="mb-4 text-sm text-slate-500">
        调整服务「{{ scaleTarget.name }}」的副本数：
      </p>
      <input v-model.number="scaleValue" type="number" min="0" class="input w-full" />
      <template #footer>
        <Button variant="secondary" @click="scaleOpen = false">取消</Button>
        <Button :loading="scaleSaving" @click="doScale">应用</Button>
      </template>
    </Modal>

    <!-- 任务 -->
    <Modal
      :open="tasksOpen"
      @update:open="tasksOpen = $event"
      :title="`任务 - ${tasksTarget?.name ?? ''}`"
      width="max-w-3xl"
    >
      <div class="max-h-[60vh] overflow-auto">
        <table class="w-full text-sm">
          <thead class="border-b border-slate-200 text-left text-xs font-medium text-slate-500 dark:border-slate-700 dark:text-slate-400">
            <tr>
              <th class="px-3 py-2">槽位</th>
              <th class="px-3 py-2">节点</th>
              <th class="px-3 py-2">状态</th>
              <th class="px-3 py-2">期望</th>
              <th class="px-3 py-2">容器 ID</th>
              <th class="px-3 py-2">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="t in tasks" :key="t.id" class="border-b border-slate-100 last:border-0 dark:border-slate-800">
              <td class="px-3 py-2 text-slate-700 dark:text-slate-200">{{ t.slot }}</td>
              <td class="px-3 py-2 text-slate-700 dark:text-slate-200">{{ t.node_name || t.node_id.slice(0, 12) }}</td>
              <td class="px-3 py-2">
                <Badge :variant="taskStateVariant(t.state)" dot>{{ t.state }}</Badge>
                <Tooltip v-if="t.error" :text="t.error" placement="top">
                  <p class="mt-0.5 max-w-[200px] truncate text-xs text-red-500">{{ t.error }}</p>
                </Tooltip>
              </td>
              <td class="px-3 py-2 text-slate-500 dark:text-slate-400">{{ t.desired_state }}</td>
              <td class="px-3 py-2 font-mono text-xs text-slate-500 dark:text-slate-400">{{ t.container_id?.slice(0, 12) || '—' }}</td>
              <td class="px-3 py-2">
                <button
                  v-if="t.container_id"
                  class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs text-slate-600 transition-colors hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-700"
                  @click="taskLogsTarget = t; taskLogsOpen = true"
                >
                  <FileText class="h-3.5 w-3.5" />
                  日志
                </button>
                <span v-else class="text-xs text-slate-300 dark:text-slate-600">—</span>
              </td>
            </tr>
            <tr v-if="tasksLoading">
              <td colspan="6" class="px-3 py-8 text-center text-sm text-slate-400">加载中…</td>
            </tr>
            <tr v-else-if="tasks.length === 0">
              <td colspan="6" class="px-3 py-8 text-center text-sm text-slate-400">暂无任务</td>
            </tr>
          </tbody>
        </table>
      </div>
      <template #footer>
        <Button variant="secondary" @click="tasksOpen = false">关闭</Button>
      </template>
    </Modal>

    <!-- 任务日志 -->
    <TaskLogsModal
      v-if="taskLogsTarget"
      v-model:open="taskLogsOpen"
      :task-id="taskLogsTarget.id"
      :task-label="`${tasksTarget?.name ?? ''} #${taskLogsTarget.slot}`"
    />

    <!-- 详情 -->
    <Modal :open="detailOpen" @update:open="detailOpen = $event" title="服务详情" width="max-w-3xl">
      <div v-if="detailLoading" class="py-8 text-center text-sm text-slate-400">加载中…</div>
      <div v-else-if="serviceDetail" class="max-h-[70vh] space-y-4 overflow-y-auto pr-1">
        <div class="grid grid-cols-2 gap-x-6 gap-y-3 sm:grid-cols-3">
          <div>
            <p class="text-xs text-slate-400">名称</p>
            <p class="text-sm font-medium text-slate-700 dark:text-slate-200">{{ serviceDetail.name }}</p>
          </div>
          <div>
            <p class="text-xs text-slate-400">模式</p>
            <p class="text-sm"><Badge variant="blue">{{ serviceDetail.mode }}</Badge></p>
          </div>
          <div>
            <p class="text-xs text-slate-400">副本</p>
            <p class="text-sm text-slate-700 dark:text-slate-200">{{ serviceDetail.replicas }}</p>
          </div>
          <div>
            <p class="text-xs text-slate-400">镜像</p>
            <p class="text-sm font-mono text-slate-700 dark:text-slate-200">{{ serviceDetail.image }}</p>
          </div>
          <div>
            <p class="text-xs text-slate-400">重启策略</p>
            <p class="text-sm text-slate-700 dark:text-slate-200">{{ serviceDetail.restart || '—' }}</p>
          </div>
          <div>
            <p class="text-xs text-slate-400">更新状态</p>
            <p class="text-sm text-slate-700 dark:text-slate-200">{{ serviceDetail.update_status || '—' }}</p>
          </div>
          <div>
            <p class="text-xs text-slate-400">CPU 限制</p>
            <p class="text-sm text-slate-700 dark:text-slate-200">{{ serviceDetail.limit_cpu ? `${serviceDetail.limit_cpu} 核` : '不限' }}</p>
          </div>
          <div>
            <p class="text-xs text-slate-400">内存限制</p>
            <p class="text-sm text-slate-700 dark:text-slate-200">{{ serviceDetail.limit_memory ? `${serviceDetail.limit_memory} MB` : '不限' }}</p>
          </div>
          <div>
            <p class="text-xs text-slate-400">服务 ID</p>
            <p class="text-sm font-mono text-slate-500 dark:text-slate-400">{{ serviceDetail.id.slice(0, 12) }}</p>
          </div>
        </div>

        <div v-if="serviceDetail.ports.length">
          <p class="mb-1.5 text-xs text-slate-400">端口</p>
          <div class="flex flex-wrap gap-1.5">
            <span v-for="(p, i) in serviceDetail.ports" :key="i" class="rounded-md bg-slate-100 px-2 py-0.5 font-mono text-xs text-slate-600 dark:bg-slate-700 dark:text-slate-300">{{ p }}</span>
          </div>
        </div>
        <div v-if="serviceDetail.networks.length">
          <p class="mb-1.5 text-xs text-slate-400">网络</p>
          <div class="flex flex-wrap gap-1.5">
            <span
              v-for="(n, i) in serviceDetail.networks"
              :key="i"
              class="cursor-pointer rounded-md bg-blue-50 px-2 py-0.5 text-xs text-blue-700 transition-colors hover:bg-blue-100 dark:bg-blue-900/40 dark:text-blue-300 dark:hover:bg-blue-900/60"
              title="查看网络"
              @click="router.push('/swarm/networks')"
            >{{ n }}</span>
          </div>
        </div>
        <div v-if="serviceDetail.mounts.length">
          <p class="mb-1.5 text-xs text-slate-400">挂载</p>
          <div class="flex flex-wrap gap-1.5">
            <span v-for="(m, i) in serviceDetail.mounts" :key="i" class="rounded-md bg-slate-100 px-2 py-0.5 font-mono text-xs text-slate-600 dark:bg-slate-700 dark:text-slate-300">{{ m }}</span>
          </div>
        </div>
        <div v-if="serviceDetail.env.length">
          <p class="mb-1.5 text-xs text-slate-400">环境变量</p>
          <div class="space-y-0.5">
            <p v-for="(e, i) in serviceDetail.env" :key="i" class="font-mono text-xs text-slate-600 dark:text-slate-300">{{ e }}</p>
          </div>
        </div>
        <div v-if="serviceDetail.command?.length">
          <p class="mb-1.5 text-xs text-slate-400">命令</p>
          <p class="font-mono text-xs text-slate-600 dark:text-slate-300">{{ serviceDetail.command?.join(' ') }}</p>
        </div>
        <div v-if="serviceDetail.secrets.length || serviceDetail.configs.length" class="grid grid-cols-2 gap-4">
          <div v-if="serviceDetail.secrets.length">
            <p class="mb-1.5 text-xs text-slate-400">Secret</p>
            <div class="flex flex-wrap gap-1.5">
              <span
                v-for="(s, i) in serviceDetail.secrets"
                :key="i"
                class="cursor-pointer rounded-md bg-rose-50 px-2 py-0.5 text-xs text-rose-700 transition-colors hover:bg-rose-100 dark:bg-rose-900/40 dark:text-rose-300 dark:hover:bg-rose-900/60"
                title="查看 Secret"
                @click="router.push('/swarm/secrets')"
              >{{ s }}</span>
            </div>
          </div>
          <div v-if="serviceDetail.configs.length">
            <p class="mb-1.5 text-xs text-slate-400">Config</p>
            <div class="flex flex-wrap gap-1.5">
              <span
                v-for="(c, i) in serviceDetail.configs"
                :key="i"
                class="cursor-pointer rounded-md bg-indigo-50 px-2 py-0.5 text-xs text-indigo-700 transition-colors hover:bg-indigo-100 dark:bg-indigo-900/40 dark:text-indigo-300 dark:hover:bg-indigo-900/60"
                title="查看 Config"
                @click="router.push('/swarm/secrets')"
              >{{ c }}</span>
            </div>
          </div>
        </div>
        <div v-if="serviceDetail.constraints.length">
          <p class="mb-1.5 text-xs text-slate-400">节点约束</p>
          <div class="space-y-0.5">
            <p v-for="(c, i) in serviceDetail.constraints" :key="i" class="font-mono text-xs text-slate-600 dark:text-slate-300">{{ c }}</p>
          </div>
        </div>
        <div v-if="Object.keys(serviceDetail.labels).length">
          <p class="mb-1.5 text-xs text-slate-400">标签</p>
          <div class="flex flex-wrap gap-1.5">
            <span v-for="(v, k) in serviceDetail.labels" :key="k" class="rounded-md bg-slate-100 px-2 py-0.5 text-xs text-slate-600 dark:bg-slate-700 dark:text-slate-300">{{ k }}={{ v }}</span>
          </div>
        </div>

        <!-- 服务资源监控（任务容器 CPU/内存） -->
        <div>
          <p class="mb-1.5 text-xs text-slate-400">任务容器资源</p>
          <div v-if="resourcesLoading" class="py-3 text-center text-xs text-slate-400">加载中…</div>
          <div v-else-if="serviceResources.length === 0" class="rounded-md border border-dashed border-slate-300 px-3 py-3 text-center text-xs text-slate-400 dark:border-slate-700">
            暂无运行中的任务容器
          </div>
          <div v-else class="space-y-2">
            <div
              v-for="r in serviceResources"
              :key="r.task_id"
              class="rounded-md border border-slate-200 bg-slate-50 p-3 dark:border-slate-700 dark:bg-slate-900"
            >
              <div class="flex items-center justify-between text-xs">
                <span class="font-mono text-slate-600 dark:text-slate-300">#{{ r.slot }} · {{ r.node_name || '—' }}</span>
                <Badge :variant="r.state === 'running' ? 'green' : 'gray'" dot>{{ r.state }}</Badge>
              </div>
              <div class="mt-2 grid grid-cols-2 gap-3">
                <div>
                  <p class="text-[10px] text-slate-400">CPU</p>
                  <p class="font-mono text-sm font-semibold text-slate-800 dark:text-slate-100">
                    {{ r.cpu_percent.toFixed(1) }}%
                  </p>
                </div>
                <div>
                  <p class="text-[10px] text-slate-400">内存</p>
                  <p class="font-mono text-sm font-semibold text-slate-800 dark:text-slate-100">
                    {{ fmtSize(r.mem_usage) }}
                    <span class="text-xs font-normal text-slate-400">/ {{ fmtSize(r.mem_limit) }} ({{ r.mem_percent.toFixed(1) }}%)</span>
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>

        <details>
          <summary class="cursor-pointer text-sm text-slate-500 hover:text-slate-700 dark:hover:text-slate-300">原始 inspect</summary>
          <pre class="mt-2 overflow-auto rounded-lg bg-slate-50 p-3 text-xs leading-relaxed text-slate-700 dark:bg-slate-900 dark:text-slate-300">{{ serviceDetail.raw }}</pre>
        </details>
      </div>
    </Modal>
  </div>
</template>
