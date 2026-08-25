<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Eye, UserCheck, UserX, Download, RefreshCw } from '@lucide/vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import DataTable, { type DataTableColumn } from '@/components/ui/DataTable.vue'
import RowActions, { type RowAction } from '@/components/ui/RowActions.vue'
import { NodeDetailModal, ResourceToolbar, YamlCreateModal } from '@/components/k8s'
import { useToast } from '@/composables/useToast'
import { useConfirm } from '@/composables/useConfirm'
import { k8sNodeTemplates } from '@/templates'
import { nodeReadyVariant, nodeRoleVariantK8s } from '@/utils/k8s'
import {
  k8sNodes,
  k8sCordonNode,
  k8sUncordonNode,
  k8sDrainNode,
  type K8sNodeItem,
} from '@/api/k8s'

const toast = useToast()
const confirm = useConfirm()

const loading = ref(false)
const errorMsg = ref('')
const items = ref<K8sNodeItem[]>([])

// 名称搜索。
const keyword = ref('')

// YAML 创建弹窗（节点等集群级资源也可用 YAML 创建）。
const createOpen = ref(false)

/** 按名称过滤后的节点。 */
const filtered = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  if (!kw) return items.value
  return items.value.filter((n) => n.name.toLowerCase().includes(kw))
})

// 节点详情弹窗。
const detailOpen = ref(false)
const detailNode = ref<K8sNodeItem | null>(null)

async function load(): Promise<void> {
  loading.value = true
  errorMsg.value = ''
  try {
    items.value = await k8sNodes()
  } catch (err) {
    errorMsg.value = (err as Error).message
    items.value = []
  } finally {
    loading.value = false
  }
}
onMounted(load)

function setSchedulable(row: K8sNodeItem, cordon: boolean): void {
  const action = cordon ? '标记为不可调度（Cordon）' : '恢复调度（Uncordon）'
  void confirm(
    action,
    cordon
      ? `确认将节点「${row.name}」标记为不可调度？新 Pod 将不会调度到该节点。`
      : `确认恢复节点「${row.name}」的调度？`,
    async () => {
      if (cordon) await k8sCordonNode(row.name)
      else await k8sUncordonNode(row.name)
      toast.success(`节点「${row.name}」已${cordon ? 'Cordon' : 'Uncordon'}`)
      await load()
    },
    { danger: cordon },
  )
}

function drain(row: K8sNodeItem): void {
  void confirm(
    '驱逐节点',
    `确认驱逐节点「${row.name}」上的所有 Pod？将先 Cordon 并逐个驱逐（排除 DaemonSet），此操作可能中断服务。`,
    async () => {
      await k8sDrainNode(row.name)
      toast.success(`节点「${row.name}」已开始驱逐`)
      await load()
    },
    { danger: true },
  )
}

function openDetail(row: K8sNodeItem): void {
  detailNode.value = row
  detailOpen.value = true
}

function buildActions(row: K8sNodeItem): RowAction[] {
  return [
    { key: 'detail', label: '详情', icon: Eye, onClick: () => openDetail(row) },
    { key: 'cordon', label: 'Cordon', icon: UserX, onClick: () => setSchedulable(row, true) },
    { key: 'uncordon', label: 'Uncordon', icon: UserCheck, onClick: () => setSchedulable(row, false) },
    { key: 'drain', label: '驱逐', icon: Download, danger: true, onClick: () => drain(row) },
  ]
}

const columns: DataTableColumn[] = [
  { label: '名称', key: 'name' },
  { label: '角色', key: 'role', width: '90px' },
  { label: '状态', key: 'status', width: '100px' },
  { label: '地址', key: 'internal_ip', width: '140px' },
  { label: 'CPU', key: 'cpu', width: '80px', align: 'center' },
  { label: '内存', key: 'memory', width: '110px' },
  { label: '操作', key: 'actions', width: '200px', align: 'right' },
]
</script>

<template>
  <div class="space-y-4">
    <!-- 通用工具栏：YAML 创建 + 名称搜索 + 刷新 -->
    <ResourceToolbar v-model:keyword="keyword" :show-namespace="false" create-label="创建节点" @create="createOpen = true">
      <template #extra>
        <Button variant="secondary" size="sm" :loading="loading" @click="load">
          <RefreshCw class="h-3.5 w-3.5" />
          刷新
        </Button>
      </template>
    </ResourceToolbar>

    <DataTable
      title="节点列表"
      :columns="columns"
      :data="filtered"
      :loading="loading"
      :error="errorMsg"
      row-key="name"
      :empty-text="keyword ? '无匹配的节点' : '暂无节点'"
      @retry="load"
    >
      <template #cell-status="{ row }">
        <Badge :variant="nodeReadyVariant((row as K8sNodeItem).ready)" dot>
          {{ (row as K8sNodeItem).status }}
        </Badge>
      </template>
      <template #cell-role="{ row }">
        <Badge :variant="nodeRoleVariantK8s((row as K8sNodeItem).role)">
          {{ (row as K8sNodeItem).role }}
        </Badge>
      </template>
      <template #cell-actions="{ row }">
        <RowActions :actions="buildActions(row as K8sNodeItem)" :visible="2" />
      </template>
    </DataTable>

    <!-- YAML 创建 -->
    <YamlCreateModal v-model:open="createOpen" title="创建节点" :templates="k8sNodeTemplates" @created="load" />

    <!-- 节点详情 -->
    <NodeDetailModal v-model:open="detailOpen" :node="detailNode" @updated="load" />
  </div>
</template>
