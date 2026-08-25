<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { Eye, RefreshCw, RotateCw, SlidersHorizontal, Trash2 } from '@lucide/vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import DataTable, { type DataTableColumn } from '@/components/ui/DataTable.vue'
import RowActions, { type RowAction } from '@/components/ui/RowActions.vue'
import { ScaleModal, ResourceDetailModal, ResourceToolbar, YamlCreateModal } from '@/components/k8s'
import { useToast } from '@/composables/useToast'
import { useConfirm } from '@/composables/useConfirm'
import { useNamespaces, ALL_NS } from '@/composables/useNamespaces'
import { k8sWorkloadTemplates } from '@/templates'
import { workloadReadyVariant } from '@/utils/k8s'
import {
  k8sDeployments,
  k8sDeleteDeployment,
  k8sRestartDeployment,
  k8sStatefulSets,
  k8sDeleteStatefulSet,
  k8sRestartStatefulSet,
  k8sDaemonSets,
  k8sDeleteDaemonSet,
  k8sRestartDaemonSet,
  k8sRawYaml,
  type K8sDeploymentItem,
  type K8sStatefulSetItem,
  type K8sDaemonSetItem,
} from '@/api/k8s'

type Tab = 'deployment' | 'statefulset' | 'daemonset'

const toast = useToast()
const confirm = useConfirm()

const { current: namespace, loadNamespaces } = useNamespaces()

const activeTab = ref<Tab>('deployment')
const tabs: { key: Tab; label: string }[] = [
  { key: 'deployment', label: 'Deployment' },
  { key: 'statefulset', label: 'StatefulSet' },
  { key: 'daemonset', label: 'DaemonSet' },
]

/** 当前 Tab 的显示名（用于表格标题）。 */
const activeTabLabel = computed(() => tabs.find((t) => t.key === activeTab.value)?.label ?? '')

// 名称搜索。
const keyword = ref('')

// YAML 创建弹窗。
const createOpen = ref(false)

// 各类型数据。
const deployments = ref<K8sDeploymentItem[]>([])
const statefulSets = ref<K8sStatefulSetItem[]>([])
const daemonSets = ref<K8sDaemonSetItem[]>([])
const loading = ref(false)
const errorMsg = ref('')

/** 当前 Tab 展示的数据。 */
const currentItems = computed<unknown[]>(() => {
  let list: unknown[]
  if (activeTab.value === 'deployment') list = deployments.value
  else if (activeTab.value === 'statefulset') list = statefulSets.value
  else list = daemonSets.value
  const kw = keyword.value.trim().toLowerCase()
  if (!kw) return list
  return list.filter((x) => (x as { name: string }).name.toLowerCase().includes(kw))
})

// 伸缩弹窗。
const scaleOpen = ref(false)
const scaleTarget = ref<{ kind: 'deployment' | 'statefulset'; name: string; namespace: string; current: number } | null>(null)

// 详情（YAML）弹窗。
const detailOpen = ref(false)
const detailTitle = ref('')
const detailFetch = ref<(() => Promise<string>) | null>(null)

async function load(): Promise<void> {
  loading.value = true
  errorMsg.value = ''
  try {
    const ns = namespace.value
    const [deps, sts, ds] = await Promise.all([
      k8sDeployments(ns),
      k8sStatefulSets(ns),
      k8sDaemonSets(ns),
    ])
    deployments.value = deps
    statefulSets.value = sts
    daemonSets.value = ds
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

// ──────────────────────────────────────────────
// 操作
// ──────────────────────────────────────────────

function openScale(kind: 'deployment' | 'statefulset', name: string, ns: string, current: number): void {
  scaleTarget.value = { kind, name, namespace: ns, current }
  scaleOpen.value = true
}

function openYaml(title: string, path: string): void {
  detailTitle.value = title
  detailFetch.value = () => k8sRawYaml(path)
  detailOpen.value = true
}

function restart(kind: string, name: string, fn: () => Promise<unknown>): void {
  void confirm(
    '重启工作负载',
    `确认重启 ${kind}「${name}」？将通过注入 restart 注解触发滚动更新。`,
    async () => {
      await fn()
      toast.success(`${kind}「${name}」已触发重启`)
      await load()
    },
  )
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

// ──────────────────────────────────────────────
// 列定义
// ──────────────────────────────────────────────

const depColumns: DataTableColumn[] = [
  { label: '名称', key: 'name' },
  { label: '就绪', key: 'ready', width: '90px', align: 'center' },
  { label: '期望', key: 'desired', width: '70px', align: 'center' },
  { label: '当前', key: 'replicas', width: '70px', align: 'center' },
  { label: '镜像', key: 'image' },
  { label: '创建时间', key: 'created_at', width: '150px' },
  { label: '操作', key: 'actions', width: '200px', align: 'right' },
]

const stsColumns: DataTableColumn[] = [
  { label: '名称', key: 'name' },
  { label: '就绪', key: 'ready', width: '90px', align: 'center' },
  { label: '副本', key: 'replicas', width: '80px', align: 'center' },
  { label: '镜像', key: 'image' },
  { label: '创建时间', key: 'created_at', width: '150px' },
  { label: '操作', key: 'actions', width: '200px', align: 'right' },
]

const dsColumns: DataTableColumn[] = [
  { label: '名称', key: 'name' },
  { label: '就绪', key: 'ready', width: '80px', align: 'center' },
  { label: '期望', key: 'desired', width: '70px', align: 'center' },
  { label: '当前', key: 'current', width: '70px', align: 'center' },
  { label: '镜像', key: 'image' },
  { label: '创建时间', key: 'created_at', width: '150px' },
  { label: '操作', key: 'actions', width: '200px', align: 'right' },
]

const currentColumns = computed<DataTableColumn[]>(() => {
  let cols: DataTableColumn[]
  if (activeTab.value === 'deployment') cols = [...depColumns]
  else if (activeTab.value === 'statefulset') cols = [...stsColumns]
  else cols = [...dsColumns]
  // 所有命名空间模式下展示命名空间列。
  if (namespace.value === ALL_NS) {
    cols.splice(1, 0, { label: '命名空间', key: 'namespace', width: '140px' })
  }
  return cols
})

function buildActions(row: Record<string, unknown>): RowAction[] {
  const name = row.name as string
  const ns = row.namespace as string
  if (activeTab.value === 'deployment') {
    const d = row as unknown as K8sDeploymentItem
    return [
      { key: 'detail', label: 'YAML', icon: Eye, onClick: () => openYaml(`Deployment/${name}`, `deployments/${name}?namespace=${encodeURIComponent(ns)}`) },
      { key: 'scale', label: '伸缩', icon: SlidersHorizontal, onClick: () => openScale('deployment', name, ns, d.desired) },
      { key: 'restart', label: '重启', icon: RotateCw, onClick: () => restart('Deployment', name, () => k8sRestartDeployment(name, ns)) },
      { key: 'delete', label: '删除', icon: Trash2, danger: true, onClick: () => remove('Deployment', name, () => k8sDeleteDeployment(name, ns)) },
    ]
  }
  if (activeTab.value === 'statefulset') {
    const s = row as unknown as K8sStatefulSetItem
    return [
      { key: 'detail', label: 'YAML', icon: Eye, onClick: () => openYaml(`StatefulSet/${name}`, `statefulsets/${name}?namespace=${encodeURIComponent(ns)}`) },
      { key: 'scale', label: '伸缩', icon: SlidersHorizontal, onClick: () => openScale('statefulset', name, ns, s.replicas) },
      { key: 'restart', label: '重启', icon: RotateCw, onClick: () => restart('StatefulSet', name, () => k8sRestartStatefulSet(name, ns)) },
      { key: 'delete', label: '删除', icon: Trash2, danger: true, onClick: () => remove('StatefulSet', name, () => k8sDeleteStatefulSet(name, ns)) },
    ]
  }
  return [
    { key: 'detail', label: 'YAML', icon: Eye, onClick: () => openYaml(`DaemonSet/${name}`, `daemonsets/${name}?namespace=${encodeURIComponent(ns)}`) },
    { key: 'restart', label: '重启', icon: RotateCw, onClick: () => restart('DaemonSet', name, () => k8sRestartDaemonSet(name, ns)) },
    { key: 'delete', label: '删除', icon: Trash2, danger: true, onClick: () => remove('DaemonSet', name, () => k8sDeleteDaemonSet(name, ns)) },
  ]
}
</script>

<template>
  <div class="space-y-4">
    <!-- 通用工具栏：Tab + YAML 创建 + 命名空间 + 搜索 + 刷新 -->
    <ResourceToolbar v-model:keyword="keyword" create-label="创建工作负载" @create="createOpen = true">
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
      :empty-text="keyword ? '无匹配的工作负载' : `当前命名空间下暂无 ${activeTabLabel}`"
      @retry="load"
    >
      <template #cell-ready="{ row }">
        <template v-if="activeTab === 'daemonset'">
          <Badge :variant="(row as K8sDaemonSetItem).ready > 0 ? 'green' : 'yellow'">
            {{ (row as K8sDaemonSetItem).ready }}
          </Badge>
        </template>
        <template v-else>
          <Badge :variant="workloadReadyVariant((row as { ready: string }).ready)">
            {{ (row as { ready: string }).ready }}
          </Badge>
        </template>
      </template>
      <template #cell-actions="{ row }">
        <RowActions :actions="buildActions(row as Record<string, unknown>)" :visible="3" />
      </template>
    </DataTable>

    <!-- YAML 创建 -->
    <YamlCreateModal v-model:open="createOpen" title="创建工作负载" :templates="k8sWorkloadTemplates" @created="load" />

    <!-- 伸缩 -->
    <ScaleModal
      v-model:open="scaleOpen"
      :kind="scaleTarget?.kind ?? 'deployment'"
      :name="scaleTarget?.name ?? ''"
      :namespace="scaleTarget?.namespace ?? ''"
      :initial="scaleTarget?.current ?? 1"
      @scaled="load"
    />

    <!-- 详情（YAML） -->
    <ResourceDetailModal v-model:open="detailOpen" :title="detailTitle" :fetch-yaml="detailFetch" />
  </div>
</template>
