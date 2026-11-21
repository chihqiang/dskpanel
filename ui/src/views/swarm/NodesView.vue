<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Eye, UserCheck, UserX, PauseCircle, Trash2, RefreshCw, KeyRound } from '@lucide/vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import DataTable, { type DataTableColumn } from '@/components/ui/DataTable.vue'
import RowActions, { type RowAction } from '@/components/ui/RowActions.vue'
import { NodeDetailModal, NodeJoinTokensModal } from '@/components/swarm'
import { useToast } from '@/composables/useToast'
import { useConfirm } from '@/composables/useConfirm'
import { nodeStateVariant, nodeAvailVariant, type BadgeVariant } from '@/utils/docker'
import {
  swarmNodes,
  swarmSetNodeAvailability,
  swarmRemoveNode,
  type SwarmNodeItem,
} from '@/api/swarm'

const toast = useToast()
const confirm = useConfirm()

const loading = ref(false)
const errorMsg = ref('')
const items = ref<SwarmNodeItem[]>([])

// 节点详情弹窗。
const detailOpen = ref(false)
const detailNode = ref<SwarmNodeItem | null>(null)

// join token 弹窗。
const tokenOpen = ref(false)

async function load(): Promise<void> {
  loading.value = true
  errorMsg.value = ''
  try {
    items.value = await swarmNodes()
  } catch (err) {
    errorMsg.value = (err as Error).message
    items.value = []
  } finally {
    loading.value = false
  }
}
onMounted(load)

function availVariant(a: string): BadgeVariant {
  return nodeAvailVariant(a)
}

function setAvailability(row: SwarmNodeItem, availability: string): void {
  void confirm(
    '切换可用性',
    `确认将节点「${row.name}」可用性切换为 ${availability}？`,
    async () => {
      await swarmSetNodeAvailability(row.id, availability)
      toast.success(`节点「${row.name}」已切换为 ${availability}`)
      await load()
    },
    { danger: availability === 'drain' },
  )
}

function removeNode(row: SwarmNodeItem): void {
  void confirm(
    '删除节点',
    `确认从集群移除节点「${row.name}」？此操作不可恢复。`,
    async () => {
      await swarmRemoveNode(row.id, true)
      toast.success(`已移除节点「${row.name}」`)
      await load()
    },
    { danger: true },
  )
}

function openDetail(row: SwarmNodeItem): void {
  detailNode.value = row
  detailOpen.value = true
}

function buildActions(row: SwarmNodeItem): RowAction[] {
  return [
    { key: 'detail', label: '详情', icon: Eye, onClick: () => openDetail(row) },
    {
      key: 'active',
      label: '置为 Active',
      icon: UserCheck,
      disabled: row.availability === 'active',
      onClick: () => setAvailability(row, 'active'),
    },
    {
      key: 'drain',
      label: '置为 Drain',
      icon: UserX,
      disabled: row.availability === 'drain',
      onClick: () => setAvailability(row, 'drain'),
    },
    {
      key: 'pause',
      label: '置为 Pause',
      icon: PauseCircle,
      disabled: row.availability === 'pause',
      onClick: () => setAvailability(row, 'pause'),
    },
    { key: 'remove', label: '删除', icon: Trash2, danger: true, onClick: () => removeNode(row) },
  ]
}

const columns: DataTableColumn[] = [
  { label: '名称', key: 'name' },
  { label: '角色', key: 'role', width: '90px' },
  { label: '状态', key: 'state', width: '100px' },
  { label: '可用性', key: 'availability', width: '110px' },
  { label: '地址', key: 'addr', width: '130px' },
  { label: '版本', key: 'version', width: '90px' },
  { label: '操作', key: 'actions', width: '180px', align: 'right' },
]
</script>

<template>
  <div>
    <DataTable
      title="节点列表"
      :columns="columns"
      :data="items"
      :loading="loading"
      :error="errorMsg"
      row-key="id"
      empty-text="暂无节点"
    >
      <template #toolbar>
        <Button variant="secondary" size="sm" :loading="loading" @click="load">
          <RefreshCw class="h-3.5 w-3.5" />
          刷新
        </Button>
        <Button variant="secondary" size="sm" @click="tokenOpen = true">
          <KeyRound class="h-3.5 w-3.5" />
          加入令牌
        </Button>
      </template>
      <template #cell-state="{ row }">
        <Badge :variant="nodeStateVariant((row as SwarmNodeItem).state)" dot>
          {{ (row as SwarmNodeItem).state }}
        </Badge>
      </template>
      <template #cell-availability="{ row }">
        <Badge :variant="availVariant((row as SwarmNodeItem).availability)" dot>
          {{ (row as SwarmNodeItem).availability }}
        </Badge>
      </template>
      <template #cell-role="{ row }">
        <Badge variant="blue">{{ (row as SwarmNodeItem).role }}</Badge>
      </template>
      <template #cell-actions="{ row }">
        <RowActions :actions="buildActions(row as SwarmNodeItem)" :visible="2" />
      </template>
    </DataTable>

    <!-- 节点详情 -->
    <NodeDetailModal v-model:open="detailOpen" :node="detailNode" />

    <!-- 加入令牌 -->
    <NodeJoinTokensModal v-model:open="tokenOpen" />
  </div>
</template>
