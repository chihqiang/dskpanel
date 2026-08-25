<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Eye, Trash2, RefreshCw } from '@lucide/vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import DataTable, { type DataTableColumn } from '@/components/ui/DataTable.vue'
import RowActions, { type RowAction } from '@/components/ui/RowActions.vue'
import { ResourceDetailModal, ResourceToolbar, YamlCreateModal } from '@/components/k8s'
import { useToast } from '@/composables/useToast'
import { useConfirm } from '@/composables/useConfirm'
import { useNamespaces } from '@/composables/useNamespaces'
import { k8sNamespaceTemplates } from '@/templates'
import {
  k8sNamespaces,
  k8sDeleteNamespace,
  k8sRawYaml,
  type K8sNamespaceItem,
} from '@/api/k8s'

const toast = useToast()
const confirm = useConfirm()

const loading = ref(false)
const errorMsg = ref('')
const items = ref<K8sNamespaceItem[]>([])

// 名称搜索。
const keyword = ref('')

// YAML 创建弹窗。
const createOpen = ref(false)

// 详情（YAML）弹窗。
const detailOpen = ref(false)
const detailTitle = ref('')
const detailFetch = ref<(() => Promise<string>) | null>(null)
const detailKind = ref('')
const detailName = ref('')
const detailNamespace = ref('')

/** 按名称过滤后的命名空间。 */
const filtered = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  if (!kw) return items.value
  return items.value.filter((n) => n.name.toLowerCase().includes(kw))
})

async function load(): Promise<void> {
  loading.value = true
  errorMsg.value = ''
  try {
    items.value = await k8sNamespaces()
  } catch (err) {
    errorMsg.value = (err as Error).message
    items.value = []
  } finally {
    loading.value = false
  }
}
onMounted(load)

function openYaml(name: string): void {
  detailTitle.value = `Namespace/${name}`
  detailFetch.value = () => k8sRawYaml(`namespaces/${name}`)
  detailKind.value = 'Namespace'
  detailName.value = name
  detailNamespace.value = ''
  detailOpen.value = true
}

function removeNamespace(name: string): void {
  void confirm(
    '删除命名空间',
    `确认删除命名空间「${name}」？该命名空间下的所有资源都将被删除，此操作不可恢复。`,
    async () => {
      await k8sDeleteNamespace(name)
      toast.success(`已删除命名空间「${name}」`)
      // 刷新命名空间列表（useNamespaces 全局缓存也需要刷新）。
      const { loadNamespaces } = useNamespaces()
      await loadNamespaces(true)
      await load()
    },
    { danger: true },
  )
}

function buildActions(row: K8sNamespaceItem): RowAction[] {
  const isSystem = ['default', 'kube-system', 'kube-public', 'kube-node-lease'].includes(row.name)
  return [
    { key: 'detail', label: 'YAML', icon: Eye, onClick: () => openYaml(row.name) },
    { key: 'delete', label: '删除', icon: Trash2, danger: true, disabled: isSystem, onClick: () => removeNamespace(row.name) },
  ]
}

const columns: DataTableColumn[] = [
  { label: '名称', key: 'name' },
  { label: '状态', key: 'status', width: '120px' },
  { label: '创建时间', key: 'created_at', width: '170px' },
  { label: '操作', key: 'actions', width: '120px', align: 'right' },
]
</script>

<template>
  <div class="space-y-4">
    <!-- 通用工具栏：YAML 创建 + 名称搜索 + 刷新 -->
    <ResourceToolbar v-model:keyword="keyword" :show-namespace="false" create-label="创建命名空间" @create="createOpen = true">
      <template #extra>
        <Button variant="secondary" size="sm" :loading="loading" @click="load">
          <RefreshCw class="h-3.5 w-3.5" />
          刷新
        </Button>
      </template>
    </ResourceToolbar>

    <DataTable
      title="命名空间列表"
      :columns="columns"
      :data="filtered"
      :loading="loading"
      :error="errorMsg"
      row-key="name"
      :empty-text="keyword ? '无匹配的命名空间' : '暂无命名空间'"
      @retry="load"
    >
      <template #cell-status="{ row }">
        <Badge :variant="(row as K8sNamespaceItem).status === 'Active' ? 'green' : 'yellow'" dot>
          {{ (row as K8sNamespaceItem).status }}
        </Badge>
      </template>
      <template #cell-actions="{ row }">
        <RowActions :actions="buildActions(row as K8sNamespaceItem)" :visible="2" />
      </template>
    </DataTable>

    <!-- YAML 创建 -->
    <YamlCreateModal v-model:open="createOpen" title="创建命名空间" :templates="k8sNamespaceTemplates" @created="load" />

    <!-- 详情（YAML） -->
    <ResourceDetailModal v-model:open="detailOpen" :title="detailTitle" :fetch-yaml="detailFetch" :resource-kind="detailKind" :resource-name="detailName" :resource-namespace="detailNamespace" @saved="load" />
  </div>
</template>
