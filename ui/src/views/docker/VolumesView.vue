<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Search, X } from '@lucide/vue'
import Button from '@/components/ui/Button.vue'
import DataTable, { type DataTableColumn } from '@/components/ui/DataTable.vue'
import CreateVolumeModal from '@/components/docker/CreateVolumeModal.vue'
import VolumeDetailModal from '@/components/docker/VolumeDetailModal.vue'
import { useConfirm } from '@/composables/useConfirm'
import { useToast } from '@/composables/useToast'
import { fmtSize } from '@/utils/format'
import { listVolumes, removeVolume, pruneVolumes, type VolumeItem } from '@/api/volume'

const loading = ref(false)
const items = ref<VolumeItem[]>([])
const errorMsg = ref('')

/** 搜索关键词（直接写在页面）。 */
const keyword = ref('')
const filteredItems = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  if (!kw) return items.value
  return items.value.filter((row) =>
    ['name', 'driver', 'mountpoint'].some((k) => {
      const v = (row as unknown as Record<string, unknown>)[k]
      return v != null && String(v).toLowerCase().includes(kw)
    }),
  )
})

const createOpen = ref(false)
const detailOpen = ref(false)
const detailVolume = ref<VolumeItem | null>(null)

async function load(): Promise<void> {
  loading.value = true
  errorMsg.value = ''
  try {
    items.value = await listVolumes()
  } catch (err) {
    errorMsg.value = (err as Error).message
  } finally {
    loading.value = false
  }
}

onMounted(load)

function openDetail(volume: VolumeItem): void {
  detailVolume.value = volume
  detailOpen.value = true
}

// 二次确认（命令式 hook）。
const confirm = useConfirm()
const toast = useToast()

function openRemoveConfirm(volume: VolumeItem): void {
  void confirm(
    '删除卷',
    `确认删除卷「${volume.name}」？卷中的数据将被删除，不可恢复。`,
    async () => {
      await removeVolume(volume.name, true)
      toast.success(`已删除卷「${volume.name}」`)
    },
    { danger: true, onSuccess: load },
  )
}

function openPruneConfirm(): void {
  void confirm(
    '清理未使用卷',
    '将删除所有未被容器引用的卷（数据不可恢复）。确定继续？',
    async () => {
      const res = await pruneVolumes()
      if (res.deleted.length > 0)
        toast.success(`已清理 ${res.deleted.length} 个卷，释放 ${fmtSize(res.reclaimed_bytes)}`)
      else toast.info('没有可清理的未使用卷')
    },
    { danger: true, onSuccess: load },
  )
}

const columns: DataTableColumn[] = [
  { label: '名称', key: 'name' },
  { label: '驱动', key: 'driver', width: '100px' },
  { label: '挂载点', key: 'mountpoint', ellipsis: true },
  { label: '作用域', key: 'scope', width: '90px' },
  { label: '使用中', key: 'used', width: '80px' },
  { label: '操作', key: 'actions', width: '120px', align: 'right' },
]
</script>

<template>
  <div class="space-y-5">
    <DataTable
      title="卷列表"
      :columns="columns"
      :data="filteredItems"
      :loading="loading"
      :error="errorMsg"
      row-key="name"
    >
      <template #toolbar>
        <div class="relative">
          <Search class="pointer-events-none absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
          <input
            v-model="keyword"
            type="text"
            class="h-9 w-56 rounded-lg border border-slate-300 bg-white pl-9 pr-8 text-sm text-slate-800 outline-none transition-colors placeholder:text-slate-400 focus:border-blue-500 focus:ring-1 focus:ring-blue-500 dark:border-slate-600 dark:bg-slate-900 dark:text-slate-100"
            placeholder="搜索名称 / 挂载点…"
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
        <Button size="sm" @click="createOpen = true">创建卷</Button>
      </template>
      <template #cell-mountpoint="{ row }">
          <span class="font-mono text-xs">{{ (row as VolumeItem).mountpoint }}</span>
        </template>
        <template #cell-used="{ row }">
          {{ (row as VolumeItem).used ? '是' : '否' }}
        </template>
        <template #cell-actions="{ row }">
          <div class="flex justify-end gap-1">
            <Button variant="ghost" size="sm" @click="openDetail(row as VolumeItem)">详情</Button>
            <Button
              variant="ghost"
              size="sm"
              class="!text-red-600"
              @click="openRemoveConfirm(row as VolumeItem)"
            >
              删除
            </Button>
          </div>
        </template>
    </DataTable>

    <!-- 卷详情 -->
    <VolumeDetailModal
      v-if="detailVolume"
      v-model:open="detailOpen"
      :volume-name="detailVolume.name"
    />

    <!-- 创建卷 -->
    <CreateVolumeModal v-model:open="createOpen" @created="load" />
  </div>
</template>
