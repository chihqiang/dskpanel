<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { Eye, RefreshCw, Trash2 } from '@lucide/vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import DataTable, { type DataTableColumn } from '@/components/ui/DataTable.vue'
import RowActions, { type RowAction } from '@/components/ui/RowActions.vue'
import { ResourceDetailModal, ResourceToolbar, YamlCreateModal } from '@/components/k8s'
import { useToast } from '@/composables/useToast'
import { useConfirm } from '@/composables/useConfirm'
import { useNamespaces, ALL_NS } from '@/composables/useNamespaces'
import { k8sConfigTemplates } from '@/templates'
import {
  k8sConfigMaps,
  k8sDeleteConfigMap,
  k8sSecrets,
  k8sDeleteSecret,
  k8sRawYaml,
  type K8sConfigMapItem,
  type K8sSecretItem,
} from '@/api/k8s'

type Tab = 'configmap' | 'secret'

const toast = useToast()
const confirm = useConfirm()

const { current: namespace, loadNamespaces } = useNamespaces()

const activeTab = ref<Tab>('configmap')
const tabs: { key: Tab; label: string }[] = [
  { key: 'configmap', label: 'ConfigMap' },
  { key: 'secret', label: 'Secret' },
]
const activeTabLabel = computed(() => tabs.find((t) => t.key === activeTab.value)?.label ?? '')

const configMaps = ref<K8sConfigMapItem[]>([])
const secrets = ref<K8sSecretItem[]>([])
const loading = ref(false)
const errorMsg = ref('')

// 名称搜索。
const keyword = ref('')

// YAML 创建弹窗。
const createOpen = ref(false)

const currentItems = computed<unknown[]>(() => {
  const list = activeTab.value === 'configmap' ? configMaps.value : secrets.value
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
    const ns = namespace.value
    const [cm, sec] = await Promise.all([k8sConfigMaps(ns), k8sSecrets(ns)])
    configMaps.value = cm
    secrets.value = sec
  } catch (err) {
    errorMsg.value = (err as Error).message
  } finally {
    loading.value = false
  }
}

watch(namespace, () => void load())

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

function remove(kind: string, name: string, fn: () => Promise<unknown>): void {
  void confirm(
    `删除 ${kind}`,
    `确认删除 ${kind}「${name}」？此操作不可恢复。`,
    async () => {
      await fn()
      toast.success(`已删除 ${kind}「${name}」`)
      await load()
    },
    { danger: true },
  )
}

function buildActions(row: Record<string, unknown>): RowAction[] {
  const name = row.name as string
  const ns = row.namespace as string
  if (activeTab.value === 'configmap') {
    return [
      { key: 'detail', label: 'YAML', icon: Eye, onClick: () => openYaml(`ConfigMap/${name}`, `configmaps/${name}?namespace=${encodeURIComponent(ns)}`, 'ConfigMap', name, ns) },
      { key: 'delete', label: '删除', icon: Trash2, danger: true, onClick: () => remove('ConfigMap', name, () => k8sDeleteConfigMap(name, ns)) },
    ]
  }
  return [
    { key: 'detail', label: 'YAML', icon: Eye, onClick: () => openYaml(`Secret/${name}`, `secrets/${name}?namespace=${encodeURIComponent(ns)}`, 'Secret', name, ns) },
    { key: 'delete', label: '删除', icon: Trash2, danger: true, onClick: () => remove('Secret', name, () => k8sDeleteSecret(name, ns)) },
  ]
}

const cmColumns: DataTableColumn[] = [
  { label: '名称', key: 'name' },
  { label: '数据键', key: 'data_keys', width: '90px', align: 'center' },
  { label: '创建时间', key: 'created_at', width: '150px' },
  { label: '操作', key: 'actions', width: '120px', align: 'right' },
]

const secColumns: DataTableColumn[] = [
  { label: '名称', key: 'name' },
  { label: '类型', key: 'type', width: '160px' },
  { label: '数据键', key: 'data_keys', width: '90px', align: 'center' },
  { label: '创建时间', key: 'created_at', width: '150px' },
  { label: '操作', key: 'actions', width: '120px', align: 'right' },
]

const currentColumns = computed<DataTableColumn[]>(() => {
  const cols = [...(activeTab.value === 'configmap' ? cmColumns : secColumns)]
  // 所有命名空间模式下展示命名空间列。
  if (namespace.value === ALL_NS) {
    cols.splice(1, 0, { label: '命名空间', key: 'namespace', width: '140px' })
  }
  return cols
})
</script>

<template>
  <div class="space-y-4">
    <!-- 通用工具栏：Tab + YAML 创建 + 命名空间 + 搜索 + 刷新 -->
    <ResourceToolbar v-model:keyword="keyword" create-label="创建配置" @create="createOpen = true">
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

    <DataTable
      :title="`${activeTabLabel} 列表`"
      :columns="currentColumns"
      :data="currentItems"
      :loading="loading"
      :error="errorMsg"
      row-key="name"
      :empty-text="keyword ? '无匹配的资源' : `当前命名空间下暂无 ${activeTabLabel}`"
      @retry="load"
    >
      <template #cell-type="{ row }">
        <Badge variant="gray">{{ (row as K8sSecretItem).type }}</Badge>
      </template>
      <template #cell-actions="{ row }">
        <RowActions :actions="buildActions(row as Record<string, unknown>)" :visible="2" />
      </template>
    </DataTable>

    <!-- YAML 创建 -->
    <YamlCreateModal v-model:open="createOpen" title="创建 ConfigMap / Secret" :templates="k8sConfigTemplates" @created="load" />

    <!-- 详情（YAML，Secret 为脱敏 YAML） -->
    <ResourceDetailModal v-model:open="detailOpen" :title="detailTitle" :fetch-yaml="detailFetch" :resource-kind="detailKind" :resource-name="detailName" :resource-namespace="detailNamespace" @saved="load" />
  </div>
</template>
