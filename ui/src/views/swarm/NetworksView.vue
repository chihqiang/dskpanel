<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Plus, Eye, Trash2, RefreshCw } from '@lucide/vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import DataTable, { type DataTableColumn } from '@/components/ui/DataTable.vue'
import RowActions, { type RowAction } from '@/components/ui/RowActions.vue'
import { SwarmNetworkCreateModal, SwarmNetworkDetailModal } from '@/components/swarm'
import { useToast } from '@/composables/useToast'
import { useConfirm } from '@/composables/useConfirm'
import {
  swarmNetworks,
  swarmRemoveNetwork,
  type SwarmNetworkItem,
} from '@/api/swarm'

const toast = useToast()
const confirm = useConfirm()

const loading = ref(false)
const errorMsg = ref('')
const items = ref<SwarmNetworkItem[]>([])

// 过滤：只显示 swarm 范围的网络。
const filteredItems = computed(() => items.value.filter((n) => n.scope === 'swarm'))

const createOpen = ref(false)
const createKind = ref<'overlay' | 'bridge'>('overlay')

const detailOpen = ref(false)
const detailNetwork = ref<SwarmNetworkItem | null>(null)

async function load(): Promise<void> {
  loading.value = true
  errorMsg.value = ''
  try {
    items.value = await swarmNetworks()
  } catch (err) {
    errorMsg.value = (err as Error).message
    items.value = []
  } finally {
    loading.value = false
  }
}
onMounted(load)

function openCreate(kind: 'overlay' | 'bridge'): void {
  createKind.value = kind
  createOpen.value = true
}

function openDetail(row: SwarmNetworkItem): void {
  detailNetwork.value = row
  detailOpen.value = true
}

function removeNetwork(row: SwarmNetworkItem): void {
  void confirm(
    '删除网络',
    `确认删除网络「${row.name}」？若被服务引用将导致任务失败。`,
    async () => {
      await swarmRemoveNetwork(row.id)
      toast.success(`已删除网络「${row.name}」`)
      await load()
    },
    { danger: true },
  )
}

function buildActions(row: SwarmNetworkItem): RowAction[] {
  return [
    { key: 'detail', label: '详情', icon: Eye, onClick: () => openDetail(row) },
    { key: 'remove', label: '删除', icon: Trash2, danger: true, onClick: () => removeNetwork(row) },
  ]
}

const columns: DataTableColumn[] = [
  { label: '名称', key: 'name' },
  { label: '驱动', key: 'driver', width: '100px' },
  { label: '作用域', key: 'scope', width: '100px' },
  { label: '可连接', key: 'attachable', width: '90px' },
  { label: '操作', key: 'actions', width: '160px', align: 'right' },
]
</script>

<template>
  <div>
    <DataTable
      title="Swarm 网络"
      :columns="columns"
      :data="filteredItems"
      :loading="loading"
      :error="errorMsg"
      row-key="id"
      empty-text="暂无 Swarm 网络"
    >
      <template #toolbar>
        <Button variant="secondary" size="sm" :loading="loading" @click="load">
          <RefreshCw class="h-3.5 w-3.5" />
          刷新
        </Button>
        <Button size="sm" @click="openCreate('overlay')">
          <Plus class="h-3.5 w-3.5" />
          新建 Overlay
        </Button>
      </template>
      <template #cell-driver="{ row }">
        <Badge variant="blue">{{ (row as SwarmNetworkItem).driver }}</Badge>
      </template>
      <template #cell-scope="{ row }">
        <Badge variant="purple" dot>{{ (row as SwarmNetworkItem).scope }}</Badge>
      </template>
      <template #cell-attachable="{ row }">
        <Badge :variant="(row as SwarmNetworkItem).attachable ? 'green' : 'gray'" dot>
          {{ (row as SwarmNetworkItem).attachable ? '是' : '否' }}
        </Badge>
      </template>
      <template #cell-actions="{ row }">
        <RowActions :actions="buildActions(row as SwarmNetworkItem)" :visible="2" />
      </template>
    </DataTable>

    <!-- 创建网络 -->
    <SwarmNetworkCreateModal v-model:open="createOpen" :kind="createKind" @created="load" />

    <!-- 网络详情 -->
    <SwarmNetworkDetailModal v-model:open="detailOpen" :network="detailNetwork" />
  </div>
</template>
