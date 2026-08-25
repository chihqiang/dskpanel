<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { Eye, FileText, Trash2, RefreshCw } from '@lucide/vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import DataTable, { type DataTableColumn } from '@/components/ui/DataTable.vue'
import RowActions, { type RowAction } from '@/components/ui/RowActions.vue'
import { PodDetailModal, PodLogsModal, ResourceToolbar, YamlCreateModal } from '@/components/k8s'
import { useToast } from '@/composables/useToast'
import { useConfirm } from '@/composables/useConfirm'
import { useNamespaces, ALL_NS } from '@/composables/useNamespaces'
import { k8sPodTemplates } from '@/templates'
import { podPhaseVariant } from '@/utils/k8s'
import { k8sPods, k8sDeletePod, type K8sPodItem } from '@/api/k8s'

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

// 详情 / 日志弹窗。
const detailOpen = ref(false)
const detailPod = ref<K8sPodItem | null>(null)
const logsOpen = ref(false)
const logsPod = ref<K8sPodItem | null>(null)

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

function buildActions(row: K8sPodItem): RowAction[] {
  return [
    { key: 'detail', label: '详情', icon: Eye, onClick: () => openDetail(row) },
    { key: 'logs', label: '日志', icon: FileText, onClick: () => openLogs(row) },
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
        <RowActions :actions="buildActions(row as K8sPodItem)" :visible="2" />
      </template>
    </DataTable>

    <!-- YAML 创建 -->
    <YamlCreateModal v-model:open="createOpen" title="创建 Pod" :templates="k8sPodTemplates" @created="load" />

    <!-- Pod 详情 -->
    <PodDetailModal v-model:open="detailOpen" :pod="detailPod" />

    <!-- Pod 日志 -->
    <PodLogsModal v-model:open="logsOpen" :pod="logsPod" />
  </div>
</template>
