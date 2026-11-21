<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { Eye, RefreshCw, Trash2 } from '@lucide/vue'
import Button from '@/components/ui/Button.vue'
import DataTable, { type DataTableColumn } from '@/components/ui/DataTable.vue'
import RowActions, { type RowAction } from '@/components/ui/RowActions.vue'
import { ResourceDetailModal, ResourceToolbar, YamlCreateModal } from '@/components/k8s'
import { useToast } from '@/composables/useToast'
import { useConfirm } from '@/composables/useConfirm'
import { useNamespaces, ALL_NS } from '@/composables/useNamespaces'
import { k8sHpaTemplates } from '@/templates'
import {
  k8sHPAs,
  k8sDeleteHPA,
  k8sRawYaml,
  type K8sHPAItem,
} from '@/api/k8s'

const toast = useToast()
const confirm = useConfirm()

const { current: namespace, loadNamespaces } = useNamespaces()

const loading = ref(false)
const errorMsg = ref('')
const items = ref<K8sHPAItem[]>([])

// 名称搜索。
const keyword = ref('')

// YAML 创建弹窗。
const createOpen = ref(false)

const filtered = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  if (!kw) return items.value
  return items.value.filter((h) =>
    h.name.toLowerCase().includes(kw) ||
    h.target_ref.toLowerCase().includes(kw),
  )
})

// 详情（YAML）弹窗。
const detailOpen = ref(false)
const detailTitle = ref('')
const detailFetch = ref<(() => Promise<string>) | null>(null)
const detailKind = ref('')
const detailName = ref('')
const detailNamespace = ref('')

async function load(): Promise<void> {
  loading.value = true
  errorMsg.value = ''
  try {
    items.value = await k8sHPAs(namespace.value)
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

function openYaml(name: string, ns: string): void {
  detailTitle.value = `HPA/${name}`
  detailFetch.value = () => k8sRawYaml(`hpas/${name}?namespace=${encodeURIComponent(ns)}`)
  detailKind.value = 'HorizontalPodAutoscaler'
  detailName.value = name
  detailNamespace.value = ns
  detailOpen.value = true
}

function removeHPA(name: string, ns: string): void {
  void confirm(
    '删除 HPA',
    `确认删除 HPA「${name}」？此操作不可恢复。`,
    async () => {
      await k8sDeleteHPA(name, ns)
      toast.success(`已删除 HPA「${name}」`)
      await load()
    },
    { danger: true },
  )
}

function buildActions(row: K8sHPAItem): RowAction[] {
  return [
    { key: 'detail', label: 'YAML', icon: Eye, onClick: () => openYaml(row.name, row.namespace) },
    { key: 'delete', label: '删除', icon: Trash2, danger: true, onClick: () => removeHPA(row.name, row.namespace) },
  ]
}

const columns = computed<DataTableColumn[]>(() => {
  const cols: DataTableColumn[] = [
    { label: '名称', key: 'name' },
    { label: '目标', key: 'target_ref' },
    { label: '最小', key: 'min_replicas', width: '60px', align: 'center' },
    { label: '最大', key: 'max_replicas', width: '60px', align: 'center' },
    { label: '当前', key: 'current_replicas', width: '60px', align: 'center' },
    { label: '期望', key: 'desired_replicas', width: '60px', align: 'center' },
    { label: '指标', key: 'metrics' },
    { label: '创建时间', key: 'created_at', width: '150px' },
    { label: '操作', key: 'actions', width: '120px', align: 'right' },
  ]
  if (namespace.value === ALL_NS) {
    cols.splice(1, 0, { label: '命名空间', key: 'namespace', width: '140px' })
  }
  return cols
})
</script>

<template>
  <div class="space-y-4">
    <!-- 通用工具栏：YAML 创建 + 命名空间 + 搜索 + 刷新 -->
    <ResourceToolbar v-model:keyword="keyword" create-label="创建 HPA" @create="createOpen = true">
      <template #extra>
        <Button variant="secondary" size="sm" :loading="loading" @click="load">
          <RefreshCw class="h-3.5 w-3.5" />
          刷新
        </Button>
      </template>
    </ResourceToolbar>

    <DataTable
      title="HPA 列表"
      :columns="columns"
      :data="filtered"
      :loading="loading"
      :error="errorMsg"
      row-key="name"
      :empty-text="keyword ? '无匹配的 HPA' : '当前命名空间下暂无 HPA'"
      @retry="load"
    >
      <template #cell-target_ref="{ row }">
        <span class="font-mono text-xs text-slate-600 dark:text-slate-300">{{ (row as K8sHPAItem).target_ref }}</span>
      </template>
      <template #cell-min_replicas="{ row }">
        <span class="text-slate-600 dark:text-slate-300">{{ (row as K8sHPAItem).min_replicas }}</span>
      </template>
      <template #cell-max_replicas="{ row }">
        <span class="font-medium text-slate-700 dark:text-slate-200">{{ (row as K8sHPAItem).max_replicas }}</span>
      </template>
      <template #cell-current_replicas="{ row }">
        <span class="text-blue-600 dark:text-blue-400">{{ (row as K8sHPAItem).current_replicas }}</span>
      </template>
      <template #cell-desired_replicas="{ row }">
        <span class="text-green-600 dark:text-green-400">{{ (row as K8sHPAItem).desired_replicas }}</span>
      </template>
      <template #cell-metrics="{ row }">
        <span class="truncate text-xs text-slate-500 dark:text-slate-400">{{ (row as K8sHPAItem).metrics }}</span>
      </template>
      <template #cell-actions="{ row }">
        <RowActions :actions="buildActions(row as K8sHPAItem)" :visible="2" />
      </template>
    </DataTable>

    <!-- YAML 创建 -->
    <YamlCreateModal v-model:open="createOpen" title="创建 HPA" :templates="k8sHpaTemplates" @created="load" />

    <!-- 详情（YAML） -->
    <ResourceDetailModal v-model:open="detailOpen" :title="detailTitle" :fetch-yaml="detailFetch" :resource-kind="detailKind" :resource-name="detailName" :resource-namespace="detailNamespace" @saved="load" />
  </div>
</template>
