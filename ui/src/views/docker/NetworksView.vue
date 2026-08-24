<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Search, X } from '@lucide/vue'
import Button from '@/components/ui/Button.vue'
import DataTable, { type DataTableColumn } from '@/components/ui/DataTable.vue'
import CreateNetworkModal from '@/components/docker/CreateNetworkModal.vue'
import NetworkDetailModal from '@/components/docker/NetworkDetailModal.vue'
import { useConfirm } from '@/composables/useConfirm'
import { useToast } from '@/composables/useToast'
import { listNetworks, removeNetwork, pruneNetworks, type NetworkItem } from '@/api/network'

const loading = ref(false)
const items = ref<NetworkItem[]>([])
const errorMsg = ref('')

/** 搜索关键词（直接写在页面）。 */
const keyword = ref('')
const filteredItems = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  if (!kw) return items.value
  return items.value.filter((row) =>
    ['name', 'driver', 'scope'].some((k) => {
      const v = (row as unknown as Record<string, unknown>)[k]
      return v != null && String(v).toLowerCase().includes(kw)
    }),
  )
})

const createOpen = ref(false)
const detailOpen = ref(false)
const detailNetwork = ref<NetworkItem | null>(null)

async function load(): Promise<void> {
  loading.value = true
  errorMsg.value = ''
  try {
    items.value = await listNetworks()
  } catch (err) {
    errorMsg.value = (err as Error).message
  } finally {
    loading.value = false
  }
}

onMounted(load)

function openDetail(network: NetworkItem): void {
  detailNetwork.value = network
  detailOpen.value = true
}

// 二次确认（命令式 hook）。
const confirm = useConfirm()
const toast = useToast()

function openRemoveConfirm(network: NetworkItem): void {
  void confirm(
    '删除网络',
    `确认删除网络「${network.name}」？若有容器连接将失败。`,
    async () => {
      await removeNetwork(network.id)
      toast.success(`已删除网络「${network.name}」`)
    },
    { danger: true, onSuccess: load },
  )
}

function openPruneConfirm(): void {
  void confirm(
    '清理未使用网络',
    '将删除所有未被容器引用的网络。确定继续？',
    async () => {
      const res = await pruneNetworks()
      if (res.deleted.length > 0) toast.success(`已清理 ${res.deleted.length} 个网络`)
      else toast.info('没有可清理的未使用网络')
    },
    { danger: true, onSuccess: load },
  )
}

const columns: DataTableColumn[] = [
  { label: '名称', key: 'name' },
  { label: '驱动', key: 'driver', width: '100px' },
  { label: '作用域', key: 'scope', width: '90px' },
  { label: 'IPAM', key: 'ipam_driver', width: '100px' },
  { label: '内置', key: 'internal', width: '70px' },
  { label: '操作', key: 'actions', width: '120px', align: 'right' },
]
</script>

<template>
  <div class="space-y-5">
    <DataTable
      title="网络列表"
      :columns="columns"
      :data="filteredItems"
      :loading="loading"
      :error="errorMsg"
      row-key="id"
    >
      <template #toolbar>
        <div class="relative">
          <Search class="pointer-events-none absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
          <input
            v-model="keyword"
            type="text"
            class="h-9 w-56 rounded-lg border border-slate-300 bg-white pl-9 pr-8 text-sm text-slate-800 outline-none transition-colors placeholder:text-slate-400 focus:border-blue-500 focus:ring-1 focus:ring-blue-500 dark:border-slate-600 dark:bg-slate-900 dark:text-slate-100"
            placeholder="搜索名称 / 驱动…"
          />
          <button
            v-if="keyword"
            class="absolute right-2.5 top-1/2 -translate-y-1/2 text-slate-400 transition-colors hover:text-slate-600 dark:hover:text-slate-200"
            aria-label="清除搜索"
            @click="keyword = ''"
          >
            <X class="h-4 w-4" />
          </button>
        </div>
        <Button variant="secondary" size="sm" @click="load">刷新</Button>
        <Button variant="danger" size="sm" @click="openPruneConfirm">清理未使用</Button>
        <Button size="sm" @click="createOpen = true">创建网络</Button>
      </template>
      <template #cell-internal="{ row }">
          {{ (row as NetworkItem).internal ? '是' : '否' }}
        </template>
        <template #cell-actions="{ row }">
          <div class="flex justify-end gap-1">
            <Button variant="ghost" size="sm" @click="openDetail(row as NetworkItem)">详情</Button>
            <Button
              variant="ghost"
              size="sm"
              class="!text-red-600"
              @click="openRemoveConfirm(row as NetworkItem)"
            >
              删除
            </Button>
          </div>
        </template>
    </DataTable>

    <!-- 网络详情 -->
    <NetworkDetailModal
      v-if="detailNetwork"
      v-model:open="detailOpen"
      :network-id="detailNetwork.id"
      :network-name="detailNetwork.name"
    />

    <!-- 创建网络 -->
    <CreateNetworkModal v-model:open="createOpen" @created="load" />
  </div>
</template>
