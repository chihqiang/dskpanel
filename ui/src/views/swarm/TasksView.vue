<script setup lang="ts">
import { computed, onMounted, ref, useTemplateRef } from 'vue'
import { Search, X, RefreshCw, FileText, ListChecks } from '@lucide/vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import DataTable, { type DataTableColumn } from '@/components/ui/DataTable.vue'
import RowActions, { type RowAction } from '@/components/ui/RowActions.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import TaskLogsModal from '@/components/swarm/TaskLogsModal.vue'
import { useDebounced } from '@/composables/useDebounced'
import { useShortcut } from '@/composables/useShortcut'
import { swarmTasks, type SwarmTaskItem } from '@/api/swarm'

const loading = ref(false)
const errorMsg = ref('')
const items = ref<SwarmTaskItem[]>([])

// 搜索。
const keyword = ref('')
const debouncedKeyword = useDebounced(keyword, 200)
const searchInputRef = useTemplateRef<HTMLInputElement>('searchInput')

// 筛选：all=全部 running=运行中 failed=失败 pending=等待中
type FilterMode = 'all' | 'running' | 'failed' | 'pending'
const filterMode = ref<FilterMode>('all')

const filteredItems = computed(() => {
  let list = items.value
  if (filterMode.value === 'running') list = list.filter((t) => t.state === 'running')
  else if (filterMode.value === 'failed') list = list.filter((t) => t.state === 'failed' || t.state === 'rejected')
  else if (filterMode.value === 'pending') list = list.filter((t) => t.state === 'pending' || t.state === 'preparing' || t.state === 'starting')
  const kw = debouncedKeyword.value.trim().toLowerCase()
  if (!kw) return list
  return list.filter((t) =>
    ['service_name', 'node_name', 'image', 'id', 'state', 'container_id'].some((k) => {
      const v = (t as unknown as Record<string, unknown>)[k]
      return v != null && String(v).toLowerCase().includes(kw)
    }),
  )
})

// 前端分页。
const page = ref(1)
const pageSize = 20
const pagedItems = computed(() =>
  filteredItems.value.slice((page.value - 1) * pageSize, page.value * pageSize),
)

// 任务日志弹窗。
const logsTarget = ref<SwarmTaskItem | null>(null)
const logsOpen = ref(false)

async function load(): Promise<void> {
  loading.value = true
  errorMsg.value = ''
  try {
    items.value = await swarmTasks()
  } catch (err) {
    errorMsg.value = (err as Error).message
    items.value = []
  } finally {
    loading.value = false
  }
}
onMounted(load)

// 快捷键。
useShortcut('/', () => searchInputRef.value?.focus())
useShortcut('r', () => void load())

function taskStateVariant(s: string): 'green' | 'red' | 'yellow' | 'gray' {
  if (s === 'running') return 'green'
  if (s === 'failed' || s === 'rejected') return 'red'
  if (s === 'preparing' || s === 'pending' || s === 'starting') return 'yellow'
  return 'gray'
}

/** 统计。 */
const stats = computed(() => {
  const total = items.value.length
  const running = items.value.filter((t) => t.state === 'running').length
  const failed = items.value.filter((t) => t.state === 'failed' || t.state === 'rejected').length
  return { total, running, failed }
})

function openLogs(task: SwarmTaskItem): void {
  logsTarget.value = task
  logsOpen.value = true
}

function buildActions(task: SwarmTaskItem): RowAction[] {
  return [
    {
      key: 'logs',
      label: '日志',
      icon: FileText,
      disabled: !task.container_id,
      onClick: () => openLogs(task),
    },
  ]
}

const columns: DataTableColumn[] = [
  { label: '服务', key: 'service_name', ellipsis: true },
  { label: '槽位', key: 'slot', width: '70px' },
  { label: '镜像', key: 'image', ellipsis: true },
  { label: '节点', key: 'node_name', width: '120px' },
  { label: '状态', key: 'state', width: '110px' },
  { label: '期望', key: 'desired_state', width: '90px' },
  { label: '容器 ID', key: 'container_id', width: '120px' },
  { label: '更新时间', key: 'updated_at', width: '160px' },
  { label: '操作', key: 'actions', width: '80px', align: 'right' },
]
</script>

<template>
  <div class="space-y-4">
    <!-- 统计卡片 -->
    <div class="grid grid-cols-3 gap-3">
      <div class="rounded-lg border border-slate-200 bg-white px-4 py-3 dark:border-slate-700 dark:bg-slate-800">
        <div class="text-xs text-slate-500 dark:text-slate-400">总任务</div>
        <div class="mt-1 text-2xl font-semibold text-slate-800 dark:text-slate-100">{{ stats.total }}</div>
      </div>
      <div class="rounded-lg border border-slate-200 bg-white px-4 py-3 dark:border-slate-700 dark:bg-slate-800">
        <div class="text-xs text-slate-500 dark:text-slate-400">运行中</div>
        <div class="mt-1 text-2xl font-semibold text-green-600 dark:text-green-400">{{ stats.running }}</div>
      </div>
      <div class="rounded-lg border border-slate-200 bg-white px-4 py-3 dark:border-slate-700 dark:bg-slate-800">
        <div class="text-xs text-slate-500 dark:text-slate-400">失败</div>
        <div class="mt-1 text-2xl font-semibold text-red-600 dark:text-red-400">{{ stats.failed }}</div>
      </div>
    </div>

    <DataTable
      title="任务列表"
      :columns="columns"
      :data="pagedItems"
      :loading="loading"
      :error="errorMsg"
      row-key="id"
      pageable
      :page="page"
      :page-size="pageSize"
      :total="filteredItems.length"
      empty-text=""
      @update:page="page = $event"
      @retry="load"
    >
      <template #empty>
        <EmptyState
          :icon="ListChecks"
          title="暂无任务"
          description="集群中暂无 Swarm 任务。创建服务后将自动产生任务。"
        />
      </template>
      <template #toolbar>
        <div class="flex flex-wrap items-center gap-2">
          <select
            v-model="filterMode"
            class="h-9 rounded-lg border border-slate-300 bg-white px-2 text-sm text-slate-700 outline-none transition-colors focus:border-blue-500 focus:ring-1 focus:ring-blue-500 dark:border-slate-600 dark:bg-slate-900 dark:text-slate-200"
            aria-label="任务筛选"
          >
            <option value="all">全部</option>
            <option value="running">运行中</option>
            <option value="failed">失败</option>
            <option value="pending">等待中</option>
          </select>
          <div class="relative">
            <Search class="pointer-events-none absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
            <input
              ref="searchInput"
              v-model="keyword"
              type="text"
              class="h-9 w-56 rounded-lg border border-slate-300 bg-white pl-9 pr-8 text-sm text-slate-800 outline-none transition-colors placeholder:text-slate-400 focus:border-blue-500 focus:ring-1 focus:ring-blue-500 dark:border-slate-600 dark:bg-slate-900 dark:text-slate-100"
              placeholder="搜索服务 / 节点 / 镜像 / ID…"
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
        </div>
        <Button variant="secondary" size="sm" :loading="loading" @click="load">
          <RefreshCw class="h-3.5 w-3.5" />
          刷新
        </Button>
      </template>
      <template #cell-service_name="{ row }">
        <span class="font-medium text-slate-700 dark:text-slate-200">{{ (row as SwarmTaskItem).service_name || '—' }}</span>
      </template>
      <template #cell-image="{ row }">
        <span class="truncate font-mono text-xs text-slate-500 dark:text-slate-400">{{ (row as SwarmTaskItem).image || '—' }}</span>
      </template>
      <template #cell-node_name="{ row }">
        <span class="text-slate-700 dark:text-slate-200">{{ (row as SwarmTaskItem).node_name || (row as SwarmTaskItem).node_id?.slice(0, 12) || '—' }}</span>
      </template>
      <template #cell-state="{ row }">
        <div>
          <Badge :variant="taskStateVariant((row as SwarmTaskItem).state)" dot>{{ (row as SwarmTaskItem).state }}</Badge>
          <p v-if="(row as SwarmTaskItem).error" class="mt-0.5 max-w-[200px] truncate text-xs text-red-500" :title="(row as SwarmTaskItem).error">
            {{ (row as SwarmTaskItem).error }}
          </p>
        </div>
      </template>
      <template #cell-desired_state="{ row }">
        <span class="text-slate-500 dark:text-slate-400">{{ (row as SwarmTaskItem).desired_state }}</span>
      </template>
      <template #cell-container_id="{ row }">
        <span class="font-mono text-xs text-slate-500 dark:text-slate-400">{{ (row as SwarmTaskItem).container_id?.slice(0, 12) || '—' }}</span>
      </template>
      <template #cell-updated_at="{ row }">
        <span class="text-xs text-slate-500 dark:text-slate-400">{{ (row as SwarmTaskItem).updated_at }}</span>
      </template>
      <template #cell-actions="{ row }">
        <RowActions :actions="buildActions(row as SwarmTaskItem)" :visible="1" />
      </template>
    </DataTable>

    <!-- 任务日志 -->
    <TaskLogsModal
      v-if="logsTarget"
      v-model:open="logsOpen"
      :task-id="logsTarget.id"
      :task-label="`${logsTarget.service_name} #${logsTarget.slot}`"
    />
  </div>
</template>
