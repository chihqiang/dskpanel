<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { Eye, RefreshCw, Trash2 } from '@lucide/vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import DataTable, { type DataTableColumn } from '@/components/ui/DataTable.vue'
import RowActions, { type RowAction } from '@/components/ui/RowActions.vue'
import { ResourceDetailModal, ResourceToolbar } from '@/components/k8s'
import { useToast } from '@/composables/useToast'
import { useConfirm } from '@/composables/useConfirm'
import { useNamespaces, ALL_NS } from '@/composables/useNamespaces'
import {
  k8sPVCs,
  k8sDeletePVC,
  k8sStorageClasses,
  k8sRawYaml,
  type K8sPVCItem,
  type K8sStorageClassItem,
} from '@/api/k8s'

type Tab = 'pvc' | 'storageclass'

const toast = useToast()
const confirm = useConfirm()

const { current: namespace, loadNamespaces } = useNamespaces()

const activeTab = ref<Tab>('pvc')
const tabs: { key: Tab; label: string }[] = [
  { key: 'pvc', label: 'PVC' },
  { key: 'storageclass', label: 'StorageClass' },
]
const activeTabLabel = computed(() => tabs.find((t) => t.key === activeTab.value)?.label ?? '')

const pvcs = ref<K8sPVCItem[]>([])
const storageClasses = ref<K8sStorageClassItem[]>([])
const loading = ref(false)
const errorMsg = ref('')

// 名称搜索。
const keyword = ref('')

const currentItems = computed<unknown[]>(() => {
  const list = activeTab.value === 'pvc' ? pvcs.value : storageClasses.value
  const kw = keyword.value.trim().toLowerCase()
  if (!kw) return list
  return list.filter((x) => (x as { name: string }).name.toLowerCase().includes(kw))
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
    if (activeTab.value === 'pvc') {
      pvcs.value = await k8sPVCs(namespace.value)
    } else {
      storageClasses.value = await k8sStorageClasses()
    }
  } catch (err) {
    errorMsg.value = (err as Error).message
  } finally {
    loading.value = false
  }
}

watch(namespace, () => {
  if (activeTab.value === 'pvc') void load()
})

watch(activeTab, () => void load())

onMounted(async () => {
  await loadNamespaces()
  await load()
})

function openYaml(title: string, path: string, kind = '', name = '', ns = ''): void {
  detailTitle.value = title
  detailFetch.value = () => k8sRawYaml(path)
  detailKind.value = kind
  detailName.value = name
  detailNamespace.value = ns
  detailOpen.value = true
}

function removePVC(name: string, ns: string): void {
  void confirm(
    '删除 PVC',
    `确认删除 PVC「${name}」？此操作不可恢复。`,
    async () => {
      await k8sDeletePVC(name, ns)
      toast.success(`已删除 PVC「${name}」`)
      await load()
    },
    { danger: true },
  )
}

function buildPVCActions(row: Record<string, unknown>): RowAction[] {
  const name = row.name as string
  const ns = row.namespace as string
  return [
    { key: 'detail', label: 'YAML', icon: Eye, onClick: () => openYaml(`PVC/${name}`, `pvcs/${name}?namespace=${encodeURIComponent(ns)}`, 'PersistentVolumeClaim', name, ns) },
    { key: 'delete', label: '删除', icon: Trash2, danger: true, onClick: () => removePVC(name, ns) },
  ]
}

function buildSCActions(row: Record<string, unknown>): RowAction[] {
  const name = row.name as string
  return [
    { key: 'detail', label: 'YAML', icon: Eye, onClick: () => openYaml(`StorageClass/${name}`, `storageclasses/${name}`, 'StorageClass', name, '') },
  ]
}

const pvcColumns: DataTableColumn[] = [
  { label: '名称', key: 'name' },
  { label: '状态', key: 'status', width: '90px' },
  { label: 'StorageClass', key: 'storage_class', width: '140px' },
  { label: '容量', key: 'capacity', width: '100px' },
  { label: '请求', key: 'requested', width: '100px' },
  { label: '访问模式', key: 'access_modes', width: '100px' },
  { label: '创建时间', key: 'created_at', width: '150px' },
  { label: '操作', key: 'actions', width: '120px', align: 'right' },
]

const scColumns: DataTableColumn[] = [
  { label: '名称', key: 'name' },
  { label: 'Provisioner', key: 'provisioner', width: '200px' },
  { label: '回收策略', key: 'reclaim_policy', width: '100px' },
  { label: '绑定模式', key: 'binding_mode', width: '120px' },
  { label: '默认', key: 'default', width: '70px', align: 'center' },
  { label: '创建时间', key: 'created_at', width: '150px' },
  { label: '操作', key: 'actions', width: '100px', align: 'right' },
]

const currentColumns = computed<DataTableColumn[]>(() => {
  const cols = [...(activeTab.value === 'pvc' ? pvcColumns : scColumns)]
  // PVC 在所有命名空间模式下展示命名空间列。
  if (activeTab.value === 'pvc' && namespace.value === ALL_NS) {
    cols.splice(1, 0, { label: '命名空间', key: 'namespace', width: '140px' })
  }
  return cols
})
</script>

<template>
  <div class="space-y-4">
    <!-- 通用工具栏：Tab + 命名空间 + 搜索 + 刷新 -->
    <ResourceToolbar v-model:keyword="keyword">
      <template #default>
        <div class="inline-flex rounded-lg bg-slate-100 p-1 dark:bg-slate-700/60">
          <button
            v-for="t in tabs"
            :key="t.key"
            class="rounded-md px-3.5 py-1.5 text-sm font-medium transition-colors"
            :class="activeTab === t.key ? 'bg-white text-blue-700 shadow-sm dark:bg-slate-800 dark:text-blue-300' : 'text-slate-600 hover:text-slate-800 dark:text-slate-300 dark:hover:text-slate-100'"
            @click="activeTab = t.key"
          >
            {{ t.label }}
          </button>
        </div>
      </template>
      <template #extra>
        <Button variant="secondary" size="sm" :loading="loading" @click="load">
          <RefreshCw class="h-3.5 w-3.5" />
          刷新
        </Button>
      </template>
    </ResourceToolbar>

    <!-- PVC 列表 -->
    <DataTable
      v-if="activeTab === 'pvc'"
      :title="`${activeTabLabel} 列表`"
      :columns="currentColumns"
      :data="currentItems"
      :loading="loading"
      :error="errorMsg"
      row-key="name"
      :empty-text="keyword ? '无匹配的资源' : `当前命名空间下暂无 ${activeTabLabel}`"
      @retry="load"
    >
      <template #cell-status="{ row }">
        <Badge :variant="(row as K8sPVCItem).status === 'Bound' ? 'green' : 'yellow'">{{ (row as K8sPVCItem).status }}</Badge>
      </template>
      <template #cell-actions="{ row }">
        <RowActions :actions="buildPVCActions(row as Record<string, unknown>)" :visible="2" />
      </template>
    </DataTable>

    <!-- StorageClass 列表 -->
    <DataTable
      v-else
      :title="`${activeTabLabel} 列表`"
      :columns="currentColumns"
      :data="currentItems"
      :loading="loading"
      :error="errorMsg"
      row-key="name"
      :empty-text="keyword ? '无匹配的资源' : '暂无 StorageClass'"
      @retry="load"
    >
      <template #cell-default="{ row }">
        <Badge v-if="(row as K8sStorageClassItem).default" variant="green">默认</Badge>
        <span v-else class="text-slate-400">—</span>
      </template>
      <template #cell-actions="{ row }">
        <RowActions :actions="buildSCActions(row as Record<string, unknown>)" :visible="2" />
      </template>
    </DataTable>

    <!-- 详情（YAML） -->
    <ResourceDetailModal v-model:open="detailOpen" :title="detailTitle" :fetch-yaml="detailFetch" :resource-kind="detailKind" :resource-name="detailName" :resource-namespace="detailNamespace" @saved="load" />
  </div>
</template>
