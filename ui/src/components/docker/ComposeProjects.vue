<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { RefreshCw, Play, Square, RotateCw, Trash2, Eye, FileText, Layers } from '@lucide/vue'
import Button from '@/components/ui/Button.vue'
import DataTable, { type DataTableColumn } from '@/components/ui/DataTable.vue'
import RowActions, { type RowAction } from '@/components/ui/RowActions.vue'
import Badge from '@/components/ui/Badge.vue'
import Modal from '@/components/ui/Modal.vue'
import { useToast } from '@/composables/useToast'
import { useConfirm } from '@/composables/useConfirm'
import {
  listComposeProjects,
  composeProjectPs,
  composeProjectStart,
  composeProjectStop,
  composeProjectRestart,
  composeProjectDown,
  composeProjectLogs,
  type ComposeProjectItem,
  type ComposeProjectDetail,
  type ComposeContainerStatus,
} from '@/api/compose'

const toast = useToast()
const confirm = useConfirm()

const projects = ref<ComposeProjectItem[]>([])
const loading = ref(false)
const error = ref('')

/** 当前执行操作的项目名（用于按钮 loading）。 */
const actionName = ref('')

// 详情弹窗。
const detailOpen = ref(false)
const detailLoading = ref(false)
const detail = ref<ComposeProjectDetail | null>(null)

// 日志弹窗。
const logsOpen = ref(false)
const logsLoading = ref(false)
const logsName = ref('')
const logs = ref<string[]>([])

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  try {
    projects.value = await listComposeProjects()
  } catch (err) {
    error.value = (err as Error).message
    projects.value = []
  } finally {
    loading.value = false
  }
}

onMounted(load)

defineExpose({ load })

/** 项目状态（形如 "running(2), exited(1)"）→ Badge 变体。 */
function statusVariant(status: string): 'green' | 'red' | 'yellow' | 'gray' {
  if (!status) return 'gray'
  if (status.includes('running') || status.includes('restarting')) return 'green'
  if (status.includes('exited') || status.includes('dead')) return 'red'
  if (status.includes('paused')) return 'yellow'
  return 'gray'
}

/** 容器状态 → Badge 变体。 */
function stateVariant(state: string): 'green' | 'red' | 'yellow' | 'gray' {
  if (state === 'running') return 'green'
  if (state === 'exited' || state === 'dead') return 'red'
  if (state === 'paused') return 'yellow'
  return 'gray'
}

async function runAction(name: string, fn: () => Promise<unknown>, msg: string): Promise<void> {
  actionName.value = name
  try {
    await fn()
    toast.success(msg)
    void load()
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    actionName.value = ''
  }
}

function onStart(row: ComposeProjectItem): void {
  void runAction(row.name, () => composeProjectStart(row.name), `项目「${row.name}」已启动`)
}

function onStop(row: ComposeProjectItem): void {
  void runAction(row.name, () => composeProjectStop(row.name), `项目「${row.name}」已停止`)
}

function onRestart(row: ComposeProjectItem): void {
  void runAction(row.name, () => composeProjectRestart(row.name), `项目「${row.name}」已重启`)
}

function onDown(row: ComposeProjectItem): void {
  void confirm({
    title: '移除 Compose 项目',
    message: `确定移除项目「${row.name}」？将停止并删除其容器与网络。`,
    danger: true,
    confirmText: '移除',
    onConfirm: () => composeProjectDown(row.name),
    onSuccess: () => {
      toast.success(`项目「${row.name}」已移除`)
      void load()
    },
  })
}

async function openDetail(row: ComposeProjectItem): Promise<void> {
  detail.value = null
  detailOpen.value = true
  detailLoading.value = true
  try {
    detail.value = await composeProjectPs(row.name)
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    detailLoading.value = false
  }
}

async function openLogs(row: ComposeProjectItem): Promise<void> {
  logsName.value = row.name
  logs.value = []
  logsOpen.value = true
  logsLoading.value = true
  try {
    logs.value = await composeProjectLogs(row.name)
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    logsLoading.value = false
  }
}

const columns: DataTableColumn[] = [
  { key: 'name', label: '项目名' },
  { key: 'status', label: '状态' },
  { key: 'services', label: '服务', align: 'center', width: '90px' },
  { key: 'containers', label: '容器', align: 'center', width: '120px' },
  { key: 'config', label: '配置文件', ellipsis: true },
  { key: 'actions', label: '操作', align: 'right', width: '140px' },
]

function actionsFor(row: ComposeProjectItem): RowAction[] {
  const busy = actionName.value === row.name
  return [
    { key: 'detail', label: '详情', icon: Eye, onClick: () => void openDetail(row) },
    { key: 'logs', label: '日志', icon: FileText, onClick: () => void openLogs(row) },
    { key: 'start', label: '启动', icon: Play, disabled: busy, loading: busy, onClick: () => onStart(row) },
    { key: 'stop', label: '停止', icon: Square, disabled: busy, loading: busy, onClick: () => onStop(row) },
    { key: 'restart', label: '重启', icon: RotateCw, disabled: busy, loading: busy, onClick: () => onRestart(row) },
    { key: 'down', label: '移除', icon: Trash2, danger: true, disabled: busy, onClick: () => onDown(row) },
  ]
}

const detailColumns: DataTableColumn[] = [
  { key: 'service', label: '服务' },
  { key: 'name', label: '容器' },
  { key: 'state', label: '状态' },
  { key: 'image', label: '镜像', ellipsis: true },
  { key: 'ports', label: '端口' },
]
</script>

<template>
  <div class="space-y-4">
    <DataTable
      title="Compose 项目"
      :columns="columns"
      :data="projects"
      :loading="loading"
      :error="error"
      row-key="name"
      empty-text="暂无 Compose 项目，可先点击右上角「部署编排」创建"
      @retry="load"
    >
      <template #toolbar>
        <slot name="deploy" />
        <Button variant="secondary" size="sm" :loading="loading" @click="load">
          <RefreshCw class="mr-1.5 h-4 w-4" />
          刷新
        </Button>
      </template>
      <template #cell-name="{ row }">
        <div class="flex items-center gap-2 font-medium text-slate-800 dark:text-slate-100">
          <Layers class="h-4 w-4 shrink-0 text-slate-400" />
          {{ (row as ComposeProjectItem).name }}
        </div>
      </template>
      <template #cell-status="{ row }">
        <Badge :variant="statusVariant((row as ComposeProjectItem).status)" dot>
          {{ (row as ComposeProjectItem).status || 'unknown' }}
        </Badge>
      </template>
      <template #cell-containers="{ row }">
        <span :class="(row as ComposeProjectItem).running > 0 ? 'text-green-600' : 'text-slate-400'">
          {{ (row as ComposeProjectItem).running }}/{{ (row as ComposeProjectItem).total }}
        </span>
      </template>
      <template #cell-config="{ row }">
        <span
          class="block truncate font-mono text-xs text-slate-500 dark:text-slate-400"
          :title="(row as ComposeProjectItem).config_files"
        >
          {{ (row as ComposeProjectItem).config_files || '-' }}
        </span>
      </template>
      <template #cell-actions="{ row }">
        <RowActions :actions="actionsFor(row as ComposeProjectItem)" :visible="3" />
      </template>
    </DataTable>

    <!-- 项目详情 -->
    <Modal v-model:open="detailOpen" title="项目详情" width="max-w-4xl">
      <div v-if="detail" class="space-y-4">
        <div class="flex flex-wrap items-center gap-3">
          <span class="text-lg font-semibold text-slate-900 dark:text-slate-100">{{ detail.name }}</span>
          <Badge :variant="detail.running > 0 ? 'green' : 'gray'" dot>{{ detail.running }}/{{ detail.total }} 运行中</Badge>
          <Badge variant="blue">{{ detail.services }} 个服务</Badge>
        </div>
        <DataTable
          :columns="detailColumns"
          :data="detail.containers"
          :loading="detailLoading"
          row-key="id"
          empty-text="暂无容器"
        >
          <template #cell-name="{ row }">
            <span class="font-mono text-xs">{{ (row as ComposeContainerStatus).name }}</span>
          </template>
          <template #cell-state="{ row }">
            <Badge :variant="stateVariant((row as ComposeContainerStatus).state)" dot>
              {{ (row as ComposeContainerStatus).state }}
            </Badge>
          </template>
          <template #cell-image="{ row }">
            <span class="block truncate font-mono text-xs">{{ (row as ComposeContainerStatus).image }}</span>
          </template>
          <template #cell-ports="{ row }">
            <template v-if="(row as ComposeContainerStatus).ports.length">
              <span
                v-for="p in (row as ComposeContainerStatus).ports"
                :key="p"
                class="mr-1 inline-block rounded bg-slate-100 px-1.5 py-0.5 font-mono text-xs dark:bg-slate-700"
              >
                {{ p }}
              </span>
            </template>
            <span v-else class="text-slate-400">-</span>
          </template>
        </DataTable>
      </div>
      <div v-else-if="detailLoading" class="py-8 text-center text-sm text-slate-400">加载中...</div>
    </Modal>

    <!-- 项目日志 -->
    <Modal v-model:open="logsOpen" :title="`日志 - ${logsName}`" width="max-w-3xl">
      <div class="rounded-md bg-slate-900 p-3 font-mono text-xs text-slate-100">
        <div v-if="logsLoading" class="text-slate-500">加载中...</div>
        <div v-else-if="logs.length === 0" class="text-slate-500">暂无日志</div>
        <div v-else class="max-h-96 space-y-0.5 overflow-y-auto">
          <div v-for="(line, i) in logs" :key="i" class="whitespace-pre-wrap break-all">{{ line }}</div>
        </div>
      </div>
    </Modal>
  </div>
</template>
