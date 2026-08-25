<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Plus, Eye, FileText, ListChecks, Scaling, Settings2, Trash2, RefreshCw, RotateCcw } from '@lucide/vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import DataTable, { type DataTableColumn } from '@/components/ui/DataTable.vue'
import RowActions, { type RowAction } from '@/components/ui/RowActions.vue'
import { ServiceFormModal, ServiceLogsModal, ServiceScaleModal, ServiceTasksModal, ServiceDetailModal } from '@/components/swarm'
import { useToast } from '@/composables/useToast'
import { useConfirm } from '@/composables/useConfirm'
import { serviceStateVariant } from '@/utils/docker'
import {
  swarmServices,
  swarmRemoveService,
  swarmRollbackService,
  swarmForceUpdateService,
  type SwarmServiceItem,
} from '@/api/swarm'

const toast = useToast()
const confirm = useConfirm()

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
const tasksTarget = ref<SwarmServiceItem | null>(null)
const tasksOpen = ref(false)
const detailTarget = ref<SwarmServiceItem | null>(null)
const detailOpen = ref(false)

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
  scaleOpen.value = true
}

function openTasks(row: SwarmServiceItem): void {
  tasksTarget.value = row
  tasksOpen.value = true
}

function openDetail(row: SwarmServiceItem): void {
  detailTarget.value = row
  detailOpen.value = true
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
    { key: 'rollback', label: '回滚', icon: RotateCcw, disabled: !row.has_update, onClick: () => rollbackService(row) },
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
    <ServiceScaleModal
      v-if="scaleTarget"
      v-model:open="scaleOpen"
      :service="scaleTarget"
      @scaled="load"
    />

    <!-- 任务 -->
    <ServiceTasksModal
      v-if="tasksTarget"
      v-model:open="tasksOpen"
      :service="tasksTarget"
    />

    <!-- 详情 -->
    <ServiceDetailModal
      v-if="detailTarget"
      v-model:open="detailOpen"
      :service="detailTarget"
    />
  </div>
</template>
