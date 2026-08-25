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
import { k8sRbacTemplates } from '@/templates'
import {
  k8sRoles,
  k8sDeleteRole,
  k8sClusterRoles,
  k8sDeleteClusterRole,
  k8sRoleBindings,
  k8sDeleteRoleBinding,
  k8sClusterRoleBindings,
  k8sDeleteClusterRoleBinding,
  k8sRawYaml,
  type K8sRoleItem,
  type K8sRoleBindingItem,
} from '@/api/k8s'

type Tab = 'role' | 'clusterrole' | 'rolebinding' | 'clusterrolebinding'

const toast = useToast()
const confirm = useConfirm()

const { current: namespace, loadNamespaces } = useNamespaces()

const activeTab = ref<Tab>('role')
const tabs: { key: Tab; label: string; cluster: boolean }[] = [
  { key: 'role', label: 'Role', cluster: false },
  { key: 'clusterrole', label: 'ClusterRole', cluster: true },
  { key: 'rolebinding', label: 'RoleBinding', cluster: false },
  { key: 'clusterrolebinding', label: 'ClusterRoleBinding', cluster: true },
]
const activeTabLabel = computed(() => tabs.find((t) => t.key === activeTab.value)?.label ?? '')
const isClusterTab = computed(() => tabs.find((t) => t.key === activeTab.value)?.cluster ?? false)

const roles = ref<K8sRoleItem[]>([])
const clusterRoles = ref<K8sRoleItem[]>([])
const roleBindings = ref<K8sRoleBindingItem[]>([])
const clusterRoleBindings = ref<K8sRoleBindingItem[]>([])
const loading = ref(false)
const errorMsg = ref('')

// 名称搜索。
const keyword = ref('')

// YAML 创建弹窗。
const createOpen = ref(false)

const currentItems = computed<unknown[]>(() => {
  let list: unknown[]
  switch (activeTab.value) {
    case 'role': list = roles.value; break
    case 'clusterrole': list = clusterRoles.value; break
    case 'rolebinding': list = roleBindings.value; break
    case 'clusterrolebinding': list = clusterRoleBindings.value; break
    default: list = []
  }
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
    switch (activeTab.value) {
      case 'role':
        roles.value = await k8sRoles(namespace.value)
        break
      case 'clusterrole':
        clusterRoles.value = await k8sClusterRoles()
        break
      case 'rolebinding':
        roleBindings.value = await k8sRoleBindings(namespace.value)
        break
      case 'clusterrolebinding':
        clusterRoleBindings.value = await k8sClusterRoleBindings()
        break
    }
  } catch (err) {
    errorMsg.value = (err as Error).message
  } finally {
    loading.value = false
  }
}

// 命名空间切换时仅刷新命名空间级别的资源。
watch(namespace, () => {
  if (!isClusterTab.value) void load()
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

function removeRole(name: string, ns: string): void {
  void confirm(
    '删除 Role',
    `确认删除 Role「${name}」？此操作不可恢复。`,
    async () => {
      await k8sDeleteRole(name, ns)
      toast.success(`已删除 Role「${name}」`)
      await load()
    },
    { danger: true },
  )
}

function removeClusterRole(name: string): void {
  void confirm(
    '删除 ClusterRole',
    `确认删除 ClusterRole「${name}」？此操作不可恢复。`,
    async () => {
      await k8sDeleteClusterRole(name)
      toast.success(`已删除 ClusterRole「${name}」`)
      await load()
    },
    { danger: true },
  )
}

function removeRoleBinding(name: string, ns: string): void {
  void confirm(
    '删除 RoleBinding',
    `确认删除 RoleBinding「${name}」？此操作不可恢复。`,
    async () => {
      await k8sDeleteRoleBinding(name, ns)
      toast.success(`已删除 RoleBinding「${name}」`)
      await load()
    },
    { danger: true },
  )
}

function removeClusterRoleBinding(name: string): void {
  void confirm(
    '删除 ClusterRoleBinding',
    `确认删除 ClusterRoleBinding「${name}」？此操作不可恢复。`,
    async () => {
      await k8sDeleteClusterRoleBinding(name)
      toast.success(`已删除 ClusterRoleBinding「${name}」`)
      await load()
    },
    { danger: true },
  )
}

function buildRoleActions(row: Record<string, unknown>): RowAction[] {
  const name = row.name as string
  const ns = row.namespace as string
  return [
    { key: 'detail', label: 'YAML', icon: Eye, onClick: () => openYaml(`Role/${name}`, `roles/${name}?namespace=${encodeURIComponent(ns)}`, 'Role', name, ns) },
    { key: 'delete', label: '删除', icon: Trash2, danger: true, onClick: () => removeRole(name, ns) },
  ]
}

function buildClusterRoleActions(row: Record<string, unknown>): RowAction[] {
  const name = row.name as string
  return [
    { key: 'detail', label: 'YAML', icon: Eye, onClick: () => openYaml(`ClusterRole/${name}`, `clusterroles/${name}`, 'ClusterRole', name, '') },
    { key: 'delete', label: '删除', icon: Trash2, danger: true, onClick: () => removeClusterRole(name) },
  ]
}

function buildRoleBindingActions(row: Record<string, unknown>): RowAction[] {
  const name = row.name as string
  const ns = row.namespace as string
  return [
    { key: 'detail', label: 'YAML', icon: Eye, onClick: () => openYaml(`RoleBinding/${name}`, `rolebindings/${name}?namespace=${encodeURIComponent(ns)}`, 'RoleBinding', name, ns) },
    { key: 'delete', label: '删除', icon: Trash2, danger: true, onClick: () => removeRoleBinding(name, ns) },
  ]
}

function buildClusterRoleBindingActions(row: Record<string, unknown>): RowAction[] {
  const name = row.name as string
  return [
    { key: 'detail', label: 'YAML', icon: Eye, onClick: () => openYaml(`ClusterRoleBinding/${name}`, `clusterrolebindings/${name}`, 'ClusterRoleBinding', name, '') },
    { key: 'delete', label: '删除', icon: Trash2, danger: true, onClick: () => removeClusterRoleBinding(name) },
  ]
}

const roleColumns: DataTableColumn[] = [
  { label: '名称', key: 'name' },
  { label: '规则数', key: 'rules', width: '80px', align: 'center' },
  { label: '创建时间', key: 'created_at', width: '150px' },
  { label: '操作', key: 'actions', width: '120px', align: 'right' },
]

const bindingColumns: DataTableColumn[] = [
  { label: '名称', key: 'name' },
  { label: '角色类型', key: 'role_kind', width: '120px' },
  { label: '角色名称', key: 'role_name' },
  { label: '主体数', key: 'subjects', width: '80px', align: 'center' },
  { label: '创建时间', key: 'created_at', width: '150px' },
  { label: '操作', key: 'actions', width: '120px', align: 'right' },
]

const currentColumns = computed<DataTableColumn[]>(() => {
  let cols: DataTableColumn[]
  if (activeTab.value === 'role' || activeTab.value === 'clusterrole') {
    cols = [...roleColumns]
  } else {
    cols = [...bindingColumns]
  }
  // 命名空间级别资源在所有命名空间模式下展示命名空间列。
  if (!isClusterTab.value && namespace.value === ALL_NS) {
    cols.splice(1, 0, { label: '命名空间', key: 'namespace', width: '140px' })
  }
  return cols
})
</script>

<template>
  <div class="space-y-4">
    <!-- 通用工具栏：Tab + YAML 创建 + 命名空间 + 搜索 + 刷新 -->
    <ResourceToolbar v-model:keyword="keyword" create-label="创建 RBAC" @create="createOpen = true">
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

    <!-- 列表 -->
    <DataTable
      :title="`${activeTabLabel} 列表`"
      :columns="currentColumns"
      :data="currentItems"
      :loading="loading"
      :error="errorMsg"
      row-key="name"
      :empty-text="keyword ? '无匹配的资源' : `暂无 ${activeTabLabel}`"
      @retry="load"
    >
      <template #cell-actions="{ row }">
        <RowActions
          :actions="
            activeTab === 'role' ? buildRoleActions(row as Record<string, unknown>)
            : activeTab === 'clusterrole' ? buildClusterRoleActions(row as Record<string, unknown>)
            : activeTab === 'rolebinding' ? buildRoleBindingActions(row as Record<string, unknown>)
            : buildClusterRoleBindingActions(row as Record<string, unknown>)
          "
          :visible="2"
        />
      </template>
    </DataTable>

    <!-- YAML 创建 -->
    <YamlCreateModal v-model:open="createOpen" title="创建 RBAC 资源" :templates="k8sRbacTemplates" @created="load" />

    <!-- 详情（YAML） -->
    <ResourceDetailModal v-model:open="detailOpen" :title="detailTitle" :fetch-yaml="detailFetch" :resource-kind="detailKind" :resource-name="detailName" :resource-namespace="detailNamespace" @saved="load" />
  </div>
</template>
