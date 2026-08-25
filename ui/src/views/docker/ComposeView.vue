<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Rocket, RefreshCw, Play, Square, RotateCw, Trash2, Eye, FileText, Layers } from '@lucide/vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import DataTable, { type DataTableColumn } from '@/components/ui/DataTable.vue'
import RowActions, { type RowAction } from '@/components/ui/RowActions.vue'
import { ComposeDeployModal, ComposeProjectDetailModal, ComposeProjectLogsModal } from '@/components/docker'
import { useToast } from '@/composables/useToast'
import { useConfirm } from '@/composables/useConfirm'
import {
  listComposeProjects,
  composeProjectStart,
  composeProjectStop,
  composeProjectRestart,
  composeProjectDown,
  type ComposeProjectItem,
} from '@/api/compose'

const toast = useToast()
const confirm = useConfirm()

// ---- 项目列表 ----
const projects = ref<ComposeProjectItem[]>([])
const loading = ref(false)
const error = ref('')
const actionName = ref('')

// ---- 部署弹窗 ----
const deployOpen = ref(false)

// ---- 详情弹窗 ----
const detailOpen = ref(false)
const detailProject = ref<ComposeProjectItem | null>(null)

// ---- 日志弹窗 ----
const logsOpen = ref(false)
const logsProject = ref<ComposeProjectItem | null>(null)

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

/** 项目状态（形如 "running(2), exited(1)"）→ Badge 变体。 */
function statusVariant(status: string): 'green' | 'red' | 'yellow' | 'gray' {
  if (!status) return 'gray'
  if (status.includes('running') || status.includes('restarting')) return 'green'
  if (status.includes('exited') || status.includes('dead')) return 'red'
  if (status.includes('paused')) return 'yellow'
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

function openDetail(row: ComposeProjectItem): void {
  detailProject.value = row
  detailOpen.value = true
}

function openLogs(row: ComposeProjectItem): void {
  logsProject.value = row
  logsOpen.value = true
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
  // 全部容器已运行 → 禁用启动；全部已停止 → 禁用停止/重启。
  const allRunning = row.running > 0 && row.running >= row.total
  const allStopped = row.running === 0
  return [
    { key: 'detail', label: '详情', icon: Eye, onClick: () => openDetail(row) },
    { key: 'logs', label: '日志', icon: FileText, onClick: () => openLogs(row) },
    { key: 'start', label: '启动', icon: Play, disabled: busy || allRunning, loading: busy, onClick: () => onStart(row) },
    { key: 'stop', label: '停止', icon: Square, disabled: busy || allStopped, loading: busy, onClick: () => onStop(row) },
    { key: 'restart', label: '重启', icon: RotateCw, disabled: busy || allStopped, loading: busy, onClick: () => onRestart(row) },
    { key: 'down', label: '移除', icon: Trash2, danger: true, disabled: busy, onClick: () => onDown(row) },
  ]
}
</script>

<template>
  <div class="space-y-5">
    <!-- 项目列表 -->
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
        <Button size="sm" @click="deployOpen = true">
          <Rocket class="h-3.5 w-3.5" />
          部署编排
        </Button>
        <Button variant="secondary" size="sm" :loading="loading" @click="load">
          <RefreshCw class="h-4 w-4" />
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

    <!-- 部署编排弹窗 -->
    <ComposeDeployModal v-model:open="deployOpen" @deployed="load" />

    <!-- 项目详情 -->
    <ComposeProjectDetailModal v-model:open="detailOpen" :project="detailProject" />

    <!-- 项目日志 -->
    <ComposeProjectLogsModal v-model:open="logsOpen" :project="logsProject" />
  </div>
</template>
