<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { FileText, Trash2, RefreshCw, Terminal } from '@lucide/vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import DataTable, { type DataTableColumn } from '@/components/ui/DataTable.vue'
import RowActions, { type RowAction } from '@/components/ui/RowActions.vue'
import LogsModal from '@/components/ui/LogsModal.vue'
import TerminalModal from '@/components/ui/TerminalModal.vue'
import { PodDetailModal, ResourceToolbar, YamlCreateModal } from '@/components/k8s'
import { useToast } from '@/composables/useToast'
import { useConfirm } from '@/composables/useConfirm'
import { useNamespaces, ALL_NS } from '@/composables/useNamespaces'
import { k8sPodTemplates } from '@/templates'
import { podPhaseVariant } from '@/utils/k8s'
import { k8sPods, k8sDeletePod, streamK8sPodLogs, type K8sPodItem } from '@/api/k8s'

const toast = useToast()
const confirm = useConfirm()

const { current: namespace, loadNamespaces } = useNamespaces()

const loading = ref(false)
const errorMsg = ref('')
const items = ref<K8sPodItem[]>([])

// 名称搜索。
const keyword = ref('')

// YAML 创建弹窗。
const createOpen = ref(false)

// 详情 / 日志 / 终端弹窗。
const detailOpen = ref(false)
const detailPod = ref<K8sPodItem | null>(null)
const logsOpen = ref(false)
const logsPod = ref<K8sPodItem | null>(null)
const terminalOpen = ref(false)
const terminalPod = ref<K8sPodItem | null>(null)

/** 按名称过滤后的列表。 */
const filtered = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  if (!kw) return items.value
  return items.value.filter((p) => p.name.toLowerCase().includes(kw))
})

async function load(): Promise<void> {
  loading.value = true
  errorMsg.value = ''
  try {
    items.value = await k8sPods({ namespace: namespace.value })
  } catch (err) {
    errorMsg.value = (err as Error).message
    items.value = []
  } finally {
    loading.value = false
  }
}

watch(namespace, () => void load())

onMounted(async () => {
  await loadNamespaces()
  await load()
})

function removePod(row: K8sPodItem): void {
  void confirm(
    '删除 Pod',
    `确认删除 Pod「${row.name}」？受 Deployment 等控制器管理的 Pod 会被自动重建。`,
    async () => {
      await k8sDeletePod(row.name, row.namespace)
      toast.success(`已删除 Pod「${row.name}」`)
      await load()
    },
    { danger: true },
  )
}

function openDetail(row: K8sPodItem): void {
  detailPod.value = row
  detailOpen.value = true
}

function openLogs(row: K8sPodItem): void {
  logsPod.value = row
  logsOpen.value = true
}

function openTerminal(row: K8sPodItem): void {
  terminalPod.value = row
  terminalOpen.value = true
}

/** 构建 Pod 终端 ws 地址。 */
const terminalWsUrl = computed(() => {
  if (!terminalPod.value) return ''
  const params = new URLSearchParams({ namespace: terminalPod.value.namespace })
  const c = terminalPod.value.containers?.[0]?.name
  if (c) params.set('container', c)
  return `/api/v1/k8s/pods/${terminalPod.value.name}/terminal?${params}`
})

/** LogsModal stream 包装。 */
function streamLogs(tail: string, container: string, onLine: (line: string) => void, onError: (msg: string) => void, onClose: () => void): () => void {
  const p = logsPod.value!
  return streamK8sPodLogs(p.name, p.namespace, Number(tail), container, onLine, onError, onClose)
}

function buildActions(row: K8sPodItem): RowAction[] {
  return [
    { key: 'logs', label: '日志', icon: FileText, onClick: () => openLogs(row) },
    { key: 'terminal', label: '终端', icon: Terminal, disabled: row.status !== 'Running', onClick: () => openTerminal(row) },
    { key: 'delete', label: '删除', icon: Trash2, danger: true, onClick: () => removePod(row) },
  ]
}

const columns = computed<DataTableColumn[]>(() => {
  const cols: DataTableColumn[] = [
    { label: '名称', key: 'name' },
    { label: '状态', key: 'status', width: '100px' },
    { label: '就绪', key: 'ready', width: '80px', align: 'center' },
    { label: '重启', key: 'restarts', width: '70px', align: 'center' },
    { label: 'IP', key: 'ip', width: '130px' },
    { label: '镜像', key: 'image' },
    { label: '创建时间', key: 'created_at', width: '150px' },
    { label: '操作', key: 'actions', width: '160px', align: 'right' },
  ]
  // 所有命名空间模式下展示命名空间列。
  if (namespace.value === ALL_NS) {
    cols.splice(1, 0, { label: '命名空间', key: 'namespace', width: '140px' })
  }
  return cols
})
</script>

<template>
  <div class="space-y-4">
    <!-- 通用工具栏：YAML 创建 + 命名空间 + 名称搜索 + 刷新 -->
    <ResourceToolbar v-model:keyword="keyword" create-label="创建 Pod" @create="createOpen = true">
      <template #extra>
        <Button variant="secondary" size="sm" :loading="loading" @click="load">
          <RefreshCw class="h-3.5 w-3.5" />
          刷新
        </Button>
      </template>
    </ResourceToolbar>

    <DataTable
      title="Pod 列表"
      :columns="columns"
      :data="filtered"
      :loading="loading"
      :error="errorMsg"
      row-key="name"
      :empty-text="keyword ? '无匹配的 Pod' : '当前命名空间下暂无 Pod'"
      @retry="load"
    >
      <!-- 名称列：点击打开详情 -->
      <template #cell-name="{ row }">
        <span
          class="cursor-pointer font-medium text-blue-600 hover:underline dark:text-blue-400"
          @click="openDetail(row as K8sPodItem)"
        >
          {{ (row as K8sPodItem).name }}
        </span>
      </template>
      <template #cell-status="{ row }">
        <Badge :variant="podPhaseVariant((row as K8sPodItem).status)" dot>
          {{ (row as K8sPodItem).status }}
        </Badge>
      </template>
      <template #cell-ready="{ row }">
        <span :class="(row as K8sPodItem).ready.startsWith('0/') ? 'text-red-500' : 'text-slate-700 dark:text-slate-200'">
          {{ (row as K8sPodItem).ready }}
        </span>
      </template>
      <template #cell-actions="{ row }">
        <RowActions :actions="buildActions(row as K8sPodItem)" :visible="3" />
      </template>
    </DataTable>

    <!-- YAML 创建 -->
    <YamlCreateModal v-model:open="createOpen" title="创建 Pod" :templates="k8sPodTemplates" @created="load" />

    <!-- Pod 详情 -->
    <PodDetailModal v-model:open="detailOpen" :pod="detailPod" />

    <!-- Pod 日志 -->
    <LogsModal
      v-if="logsPod"
      v-model:open="logsOpen"
      :title="`Pod 日志 - ${logsPod.name}`"
      :stream="streamLogs"
      :containers="logsPod.containers ?? []"
    />

    <!-- Pod 终端 -->
    <TerminalModal
      v-if="terminalPod"
      v-model:open="terminalOpen"
      :url="terminalWsUrl"
      :title="`终端 - ${terminalPod.name}`"
    />
  </div>
</template>
