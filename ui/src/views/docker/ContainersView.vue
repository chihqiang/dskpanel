<script setup lang="ts">
import { computed, onMounted, ref, useTemplateRef, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  Search, Play, Square, RotateCw, X, Filter, Container,
  FileText, Terminal, Activity, Pause, FileDown, Pencil, Camera, Settings, Trash2, Copy,
} from '@lucide/vue'
import { useShortcut } from '@/composables/useShortcut'
import { useDebounced } from '@/composables/useDebounced'
import { useUndoableAction } from '@/composables/useUndoableAction'
import { useClipboard } from '@/utils/clipboard'
import { fmtUnixTime, shortId } from '@/utils/format'
import { containerStateVariant } from '@/utils/docker'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import DataTable, { type DataTableColumn } from '@/components/ui/DataTable.vue'
import RowActions, { type RowAction } from '@/components/ui/RowActions.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import CreateContainerModal from '@/components/docker/CreateContainerModal.vue'
import ContainerLogsModal from '@/components/docker/ContainerLogsModal.vue'
import ContainerDetailModal from '@/components/docker/ContainerDetailModal.vue'
import ContainerStatsModal from '@/components/docker/ContainerStatsModal.vue'
import ContainerTerminalModal from '@/components/docker/ContainerTerminalModal.vue'
import RenameContainerModal from '@/components/docker/RenameContainerModal.vue'
import CommitContainerModal from '@/components/docker/CommitContainerModal.vue'
import UpdateContainerModal from '@/components/docker/UpdateContainerModal.vue'
import { useConfirm } from '@/composables/useConfirm'
import { useToast } from '@/composables/useToast'
import {
  listContainers,
  startContainer,
  stopContainer,
  restartContainer,
  removeContainer,
  batchContainers,
  pauseContainer,
  unpauseContainer,
  exportContainer,
  type ContainerItem,
} from '@/api/container'

const loading = ref(false)
const items = ref<ContainerItem[]>([])
const errorMsg = ref('')

const route = useRoute()
const router = useRouter()

/** 状态筛选（来自 URL ?state=，由概览页状态分布跳转带入）。 */
const stateFilter = ref((route.query.state as string) || '')

/** 搜索关键词 + 状态筛选（直接写在页面）。 */
const keyword = ref('')
const debouncedKeyword = useDebounced(keyword, 200)
const searchInputRef = useTemplateRef<HTMLInputElement>('searchInput')
const filteredItems = computed(() => {
  const kw = debouncedKeyword.value.trim().toLowerCase()
  let list = items.value
  // 状态筛选优先于关键词（更精确）。
  if (stateFilter.value) {
    list = list.filter((row) => row.state === stateFilter.value)
  }
  if (!kw) return list
  return list.filter((row) =>
    ['names', 'image', 'state', 'status'].some((k) => {
      const v = (row as unknown as Record<string, unknown>)[k]
      return v != null && String(v).toLowerCase().includes(kw)
    }),
  )
})

/** 同步 URL 中的 state 参数（概览页跳转、手动修改地址等）。 */
watch(
  () => route.query.state,
  (s) => {
    stateFilter.value = (s as string) || ''
  },
)

/** 清除状态筛选。 */
function clearStateFilter(): void {
  const q = { ...route.query }
  delete q.state
  router.replace({ path: '/docker/containers', query: q })
}

const createOpen = ref(false)
const logOpen = ref(false)
const logContainer = ref<ContainerItem | null>(null)
const detailOpen = ref(false)
const detailContainer = ref<ContainerItem | null>(null)
const statsOpen = ref(false)
const statsContainer = ref<ContainerItem | null>(null)
const terminalOpen = ref(false)
const terminalContainer = ref<ContainerItem | null>(null)
const renameOpen = ref(false)
const renameTarget = ref<ContainerItem | null>(null)
const commitOpen = ref(false)
const commitTarget = ref<ContainerItem | null>(null)
const updateOpen = ref(false)
const updateTarget = ref<ContainerItem | null>(null)
const exportingId = ref('')

// 多选 + 批量。
const selectedKeys = ref<(string | number)[]>([])
const batchLoading = ref(false)
const selectedContainers = computed(() => {
  const set = new Set(selectedKeys.value.map(String))
  return items.value.filter((c) => set.has(c.id))
})

// 二次确认（命令式 hook）。
const confirm = useConfirm()
const toast = useToast()
const { undoableAction } = useUndoableAction()
const { copy } = useClipboard()

function openRemoveConfirm(row: ContainerItem): void {
  const name = row.names?.[0] || shortId(row.id)
  undoableAction({
    title: '删除容器',
    message: `确认删除容器「${name}」？将同时移除其数据卷（force + remove volumes）。5 秒内可撤销。`,
    label: name,
    actionLabel: '删除容器',
    activityDetail: shortId(row.id),
    action: () => removeContainer(row.id, true, true),
    onDone: load,
  })
}

/** 构建行操作列表（RowActions 组件消费：前 visible 个显示，其余进「更多」）。 */
function buildActions(row: ContainerItem): RowAction[] {
  const name = row.names?.[0] || row.id.slice(0, 12)
  const running = row.state === 'running'
  const paused = row.state === 'paused'
  return [
    { key: 'copyid', label: '复制 ID', icon: Copy, onClick: () => copyId(row) },
    { key: 'logs', label: '日志', icon: FileText, onClick: () => openLogs(row) },
    { key: 'terminal', label: '终端', icon: Terminal, onClick: () => openTerminal(row) },
    { key: 'stats', label: '监控', icon: Activity, onClick: () => openStats(row) },
    {
      key: 'pause',
      label: paused ? '恢复' : '暂停',
      icon: Pause,
      disabled: !running && !paused,
      onClick: () => togglePause(row),
    },
    {
      key: 'stop',
      label: '停止',
      icon: Square,
      disabled: !running,
      onClick: () => doAction(() => stopContainer(row.id), `已停止「${name}」`),
    },
    {
      key: 'start',
      label: '启动',
      icon: Play,
      disabled: running,
      onClick: () => doAction(() => startContainer(row.id), `已启动「${name}」`),
    },
    {
      key: 'restart',
      label: '重启',
      icon: RotateCw,
      onClick: () => doAction(() => restartContainer(row.id), `已重启「${name}」`),
    },
    { key: 'rename', label: '重命名', icon: Pencil, onClick: () => openRename(row) },
    { key: 'commit', label: '提交镜像', icon: Camera, onClick: () => openCommit(row) },
    { key: 'update', label: '更新', icon: Settings, onClick: () => openUpdate(row) },
    { key: 'export', label: '导出', icon: FileDown, onClick: () => doExport(row) },
    { key: 'remove', label: '删除', icon: Trash2, danger: true, onClick: () => openRemoveConfirm(row) },
  ]
}

function openDetail(row: ContainerItem): void {
  detailContainer.value = row
  detailOpen.value = true
}

function openStats(row: ContainerItem): void {
  statsContainer.value = row
  statsOpen.value = true
}

function openTerminal(row: ContainerItem): void {
  terminalContainer.value = row
  terminalOpen.value = true
}

function openRename(row: ContainerItem): void {
  renameTarget.value = row
  renameOpen.value = true
}

function openCommit(row: ContainerItem): void {
  commitTarget.value = row
  commitOpen.value = true
}

function openUpdate(row: ContainerItem): void {
  updateTarget.value = row
  updateOpen.value = true
}

async function togglePause(row: ContainerItem): Promise<void> {
  const name = row.names?.[0] || row.id.slice(0, 12)
  const wasPaused = row.state === 'paused'
  // doAction 成功时统一 toast.success，失败时 toast.error（避免失败仍弹成功）。
  await doAction(
    () => (wasPaused ? unpauseContainer(row.id) : pauseContainer(row.id)),
    wasPaused ? `已恢复「${name}」` : `已暂停「${name}」`,
  )
}

async function doExport(row: ContainerItem): Promise<void> {
  const name = row.names?.[0] || row.id.slice(0, 12)
  exportingId.value = row.id
  try {
    await exportContainer(row.id, name)
    toast.success(`已导出容器「${name}」`)
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    exportingId.value = ''
  }
}

async function doBatch(action: 'start' | 'stop' | 'restart' | 'remove'): Promise<void> {
  const sel = selectedContainers.value
  if (sel.length === 0) return

  const actionLabel = { start: '启动', stop: '停止', restart: '重启', remove: '删除' }[action]

  const exec = async () => {
    batchLoading.value = true
    try {
      const res = await batchContainers(action, sel.map((c) => c.id))
      if (res.failed?.length) {
        toast.error(`成功 ${res.done} 个，失败 ${res.failed.length} 个`)
      } else {
        toast.success(`已${actionLabel} ${res.done} 个容器`)
      }
      selectedKeys.value = []
      await load()
    } catch (err) {
      toast.error((err as Error).message)
    } finally {
      batchLoading.value = false
    }
  }

  if (action === 'remove') {
    await confirm(
      '批量删除容器',
      `确认删除选中的 ${sel.length} 个容器？将同时移除其数据卷。`,
      exec,
      { danger: true },
    )
  } else {
    await exec()
  }
}

function openBatchRemoveConfirm(): void {
  void doBatch('remove')
}

function openBatchStart(): void {
  void doBatch('start')
}

function openBatchStop(): void {
  void doBatch('stop')
}

function openBatchRestart(): void {
  void doBatch('restart')
}

async function load(): Promise<void> {
  loading.value = true
  errorMsg.value = ''
  try {
    items.value = await listContainers(true)
    const ids = new Set(items.value.map((c) => c.id))
    selectedKeys.value = selectedKeys.value.filter((k) => ids.has(String(k)))
  } catch (err) {
    errorMsg.value = (err as Error).message
  } finally {
    loading.value = false
  }
}

onMounted(load)

// 全局快捷键：/ 聚焦搜索、r 刷新、n 新建容器。
useShortcut('/', () => searchInputRef.value?.focus())
useShortcut('r', () => void load())
useShortcut('n', () => { createOpen.value = true })

async function doAction(fn: () => Promise<unknown>, successMsg?: string): Promise<void> {
  try {
    await fn()
    await load()
    if (successMsg) toast.success(successMsg)
  } catch (err) {
    toast.error((err as Error).message)
  }
}

function openLogs(row: ContainerItem): void {
  logContainer.value = row
  logOpen.value = true
}

/** 复制容器 ID 到剪贴板。 */
async function copyId(row: ContainerItem): Promise<void> {
  await copy(row.id, '已复制容器 ID')
}

const columns: DataTableColumn[] = [
  { label: '名称', key: 'names', ellipsis: true },
  { label: '镜像', key: 'image', ellipsis: true },
  { label: '状态', key: 'state', width: '100px' },
  { label: '端口', key: 'ports', width: '180px' },
  { label: '创建时间', key: 'created', width: '160px' },
  { label: '操作', key: 'actions', width: '220px', align: 'right' },
]

function fmtPorts(item: ContainerItem): string {
  if (!item.ports || item.ports.length === 0) return '-'
  return item.ports
    .map((p) => (p.public_port ? `${p.public_port}->${p.private_port}/${p.type}` : `${p.private_port}/${p.type}`))
    .join(', ')
}
</script>

<template>
  <div class="space-y-5">
    <!-- 状态筛选提示条（来自概览页状态分布跳转） -->
    <div
      v-if="stateFilter"
      class="flex flex-wrap items-center gap-2 rounded-lg border border-blue-200 bg-blue-50 px-4 py-2 dark:border-blue-900 dark:bg-blue-900/20"
    >
      <Filter class="h-4 w-4 text-blue-600 dark:text-blue-400" />
      <span class="text-sm text-blue-700 dark:text-blue-300">
        已按状态筛选：<Badge :variant="containerStateVariant(stateFilter)">{{ stateFilter }}</Badge>
      </span>
      <span class="text-xs text-blue-500 dark:text-blue-400">共 {{ filteredItems.length }} 个容器</span>
      <Button variant="ghost" size="sm" class="ml-auto" @click="clearStateFilter">
        <X class="mr-1 h-3.5 w-3.5" />清除筛选
      </Button>
    </div>

    <DataTable
      title="容器列表"
      :columns="columns"
      :data="filteredItems"
      :loading="loading"
      :error="errorMsg"
      row-key="id"
      selectable
      :selected-keys="selectedKeys"
      empty-text=""
      @update:selected-keys="selectedKeys = $event"
      @retry="load"
    >
      <template #empty>
        <EmptyState
          :icon="Container"
          title="没有容器"
          :description="stateFilter ? `没有状态为「${stateFilter}」的容器` : '创建一个容器来运行镜像，或从镜像列表直接运行。'"
          :action-label="stateFilter ? undefined : '创建容器'"
          doc-url="https://docs.docker.com/engine/reference/commandline/run/"
          @action="createOpen = true"
        />
      </template>
      <template #toolbar>
        <div class="relative">
          <Search class="pointer-events-none absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
          <input
            ref="searchInput"
            v-model="keyword"
            type="text"
            class="h-9 w-56 rounded-lg border border-slate-300 bg-white pl-9 pr-8 text-sm text-slate-800 outline-none transition-colors placeholder:text-slate-400 focus:border-blue-500 focus:ring-1 focus:ring-blue-500 dark:border-slate-600 dark:bg-slate-900 dark:text-slate-100"
            placeholder="搜索名称 / 镜像 / 状态…"
          />
          <button
            v-if="keyword"
            class="absolute right-2.5 top-1/2 -translate-y-1/2 text-slate-400 transition-colors hover:text-slate-600 dark:hover:text-slate-200"
            aria-label="清除搜索"
            @click="keyword = ''"
          >
            <X class="h-4 w-4" />
          </button>
        </div>
        <Button variant="secondary" size="sm" @click="load">刷新</Button>
        <Button size="sm" @click="createOpen = true">创建容器</Button>
      </template>
      <template #cell-names="{ row }">
          <span
            class="cursor-pointer font-medium text-blue-600 hover:underline dark:text-blue-400"
            @click="openDetail(row as ContainerItem)"
          >
            {{ (row as ContainerItem).names[0] || shortId((row as ContainerItem).id) }}
          </span>
        </template>
        <template #cell-state="{ row }">
          <Badge :variant="containerStateVariant((row as ContainerItem).state)" dot>{{ (row as ContainerItem).state }}</Badge>
        </template>
        <template #cell-ports="{ row }">{{ fmtPorts(row as ContainerItem) }}</template>
        <template #cell-created="{ row }">{{ fmtUnixTime((row as ContainerItem).created) }}</template>
        <template #cell-actions="{ row }">
          <RowActions :actions="buildActions(row as ContainerItem)" :visible="3" />
        </template>
    </DataTable>

    <!-- 批量操作条 -->
    <div
      v-if="selectedContainers.length > 0"
      class="flex flex-wrap items-center gap-2 rounded-lg border border-blue-200 bg-blue-50 px-4 py-2 dark:border-blue-900 dark:bg-blue-900/20"
    >
      <span class="text-sm font-medium text-blue-700 dark:text-blue-300">已选 {{ selectedContainers.length }}</span>
      <Button variant="secondary" size="sm" :loading="batchLoading" @click="openBatchStart">
        <Play class="mr-1 h-3.5 w-3.5" />启动
      </Button>
      <Button variant="secondary" size="sm" :loading="batchLoading" @click="openBatchStop">
        <Square class="mr-1 h-3.5 w-3.5" />停止
      </Button>
      <Button variant="secondary" size="sm" :loading="batchLoading" @click="openBatchRestart">
        <RotateCw class="mr-1 h-3.5 w-3.5" />重启
      </Button>
      <Button variant="danger" size="sm" :loading="batchLoading" @click="openBatchRemoveConfirm">批量删除</Button>
    </div>

    <!-- 创建容器 -->
    <CreateContainerModal v-model:open="createOpen" @created="load" />

    <!-- 日志 -->
    <ContainerLogsModal
      v-if="logContainer"
      v-model:open="logOpen"
      :container-id="logContainer.id"
      :container-name="logContainer.names[0] ?? ''"
    />

    <!-- 详情 -->
    <ContainerDetailModal
      v-if="detailContainer"
      v-model:open="detailOpen"
      :container-id="detailContainer.id"
      :container-name="detailContainer.names[0] ?? ''"
      @updated="load"
    />

    <!-- 监控 -->
    <ContainerStatsModal
      v-if="statsContainer"
      v-model:open="statsOpen"
      :container-id="statsContainer.id"
      :container-name="statsContainer.names[0] ?? ''"
    />

    <!-- 终端 -->
    <ContainerTerminalModal
      v-if="terminalContainer"
      v-model:open="terminalOpen"
      :container-id="terminalContainer.id"
      :container-name="terminalContainer.names[0] ?? ''"
    />

    <!-- 重命名 -->
    <RenameContainerModal
      v-if="renameTarget"
      v-model:open="renameOpen"
      :container-id="renameTarget.id"
      :current-name="renameTarget.names[0] ?? ''"
      @renamed="load"
    />

    <!-- 提交为镜像 -->
    <CommitContainerModal
      v-if="commitTarget"
      v-model:open="commitOpen"
      :container-id="commitTarget.id"
      :container-name="commitTarget.names[0] ?? ''"
      @committed="load"
    />

    <!-- 更新配置 -->
    <UpdateContainerModal
      v-if="updateTarget"
      v-model:open="updateOpen"
      :container-id="updateTarget.id"
      :container-name="updateTarget.names[0] ?? ''"
      @updated="load"
    />
  </div>
</template>
