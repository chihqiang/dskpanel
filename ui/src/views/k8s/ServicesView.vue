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
import { k8sServiceTemplates } from '@/templates'
import { serviceTypeVariant } from '@/utils/k8s'
import {
  k8sServices,
  k8sDeleteService,
  k8sIngresses,
  k8sDeleteIngress,
  k8sRawYaml,
  type K8sServiceItem,
  type K8sIngressItem,
} from '@/api/k8s'

type Tab = 'service' | 'ingress'

const toast = useToast()
const confirm = useConfirm()

const { current: namespace, loadNamespaces } = useNamespaces()

const activeTab = ref<Tab>('service')
const tabs: { key: Tab; label: string }[] = [
  { key: 'service', label: 'Service' },
  { key: 'ingress', label: 'Ingress' },
]
const activeTabLabel = computed(() => tabs.find((t) => t.key === activeTab.value)?.label ?? '')

const services = ref<K8sServiceItem[]>([])
const ingresses = ref<K8sIngressItem[]>([])
const loading = ref(false)
const errorMsg = ref('')

// 名称搜索。
const keyword = ref('')

// YAML 创建弹窗。
const createOpen = ref(false)

const currentItems = computed<unknown[]>(() => {
  const list = activeTab.value === 'service' ? services.value : ingresses.value
  const kw = keyword.value.trim().toLowerCase()
  if (!kw) return list
  return list.filter((x) => (x as { name: string }).name.toLowerCase().includes(kw))
})

// 详情（YAML）弹窗。
const detailOpen = ref(false)
const detailTitle = ref('')
const detailFetch = ref<(() => Promise<string>) | null>(null)

async function load(): Promise<void> {
  loading.value = true
  errorMsg.value = ''
  try {
    const ns = namespace.value
    const [svc, ing] = await Promise.all([k8sServices(ns), k8sIngresses(ns)])
    services.value = svc
    ingresses.value = ing
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

function openYaml(title: string, path: string): void {
  detailTitle.value = title
  detailFetch.value = () => k8sRawYaml(path)
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
  if (activeTab.value === 'service') {
    return [
      { key: 'detail', label: 'YAML', icon: Eye, onClick: () => openYaml(`Service/${name}`, `services/${name}?namespace=${encodeURIComponent(ns)}`) },
      { key: 'delete', label: '删除', icon: Trash2, danger: true, onClick: () => remove('Service', name, () => k8sDeleteService(name, ns)) },
    ]
  }
  return [
    { key: 'detail', label: 'YAML', icon: Eye, onClick: () => openYaml(`Ingress/${name}`, `ingresses/${name}?namespace=${encodeURIComponent(ns)}`) },
    { key: 'delete', label: '删除', icon: Trash2, danger: true, onClick: () => remove('Ingress', name, () => k8sDeleteIngress(name, ns)) },
  ]
}

const svcColumns: DataTableColumn[] = [
  { label: '名称', key: 'name' },
  { label: '类型', key: 'type', width: '120px' },
  { label: 'ClusterIP', key: 'cluster_ip', width: '130px' },
  { label: '外部 IP', key: 'external_ip', width: '130px' },
  { label: '端口', key: 'ports' },
  { label: '创建时间', key: 'created_at', width: '150px' },
  { label: '操作', key: 'actions', width: '120px', align: 'right' },
]

const ingColumns: DataTableColumn[] = [
  { label: '名称', key: 'name' },
  { label: '域名', key: 'hosts' },
  { label: '地址', key: 'address', width: '150px' },
  { label: 'Class', key: 'class_name', width: '100px' },
  { label: '创建时间', key: 'created_at', width: '150px' },
  { label: '操作', key: 'actions', width: '120px', align: 'right' },
]

const currentColumns = computed<DataTableColumn[]>(() => {
  const cols = [...(activeTab.value === 'service' ? svcColumns : ingColumns)]
  // 所有命名空间模式下展示命名空间列。
  if (namespace.value === ALL_NS) {
    cols.splice(1, 0, { label: '命名空间', key: 'namespace', width: '140px' })
  }
  return cols
})

/** 端口列表格式化：如 "80:80/tcp"、"30080:80/tcp(NodePort)"。 */
function fmtPorts(p: { port: number; target_port: string; protocol: string; node_port?: number }[]): string {
  return (p ?? [])
    .map((x) => `${x.port}:${x.target_port}/${x.protocol}${x.node_port ? `(NodePort ${x.node_port})` : ''}`)
    .join('，')
}

function fmtHosts(hosts: string[] | undefined): string {
  return hosts?.length ? hosts.join('，') : '—'
}
</script>

<template>
  <div class="space-y-4">
    <!-- 通用工具栏：Tab + YAML 创建 + 命名空间 + 搜索 + 刷新 -->
    <ResourceToolbar v-model:keyword="keyword" create-label="创建 Service" @create="createOpen = true">
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
        <Badge :variant="serviceTypeVariant((row as K8sServiceItem).type)">
          {{ (row as K8sServiceItem).type }}
        </Badge>
      </template>
      <template #cell-ports="{ row }">
        <span class="text-xs text-slate-600 dark:text-slate-300">{{ fmtPorts((row as K8sServiceItem).ports ?? []) }}</span>
      </template>
      <template #cell-hosts="{ row }">
        <span class="text-xs text-slate-600 dark:text-slate-300">{{ fmtHosts((row as K8sIngressItem).hosts) }}</span>
      </template>
      <template #cell-actions="{ row }">
        <RowActions :actions="buildActions(row as Record<string, unknown>)" :visible="2" />
      </template>
    </DataTable>

    <!-- YAML 创建 -->
    <YamlCreateModal v-model:open="createOpen" title="创建 Service / Ingress" :templates="k8sServiceTemplates" @created="load" />

    <!-- 详情（YAML） -->
    <ResourceDetailModal v-model:open="detailOpen" :title="detailTitle" :fetch-yaml="detailFetch" />
  </div>
</template>
