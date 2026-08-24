<script setup lang="ts">
import { computed, onMounted, ref, useTemplateRef } from 'vue'
import { Search, Trash2, X, Eye, Tag, Upload, Play, FileDown, Copy, Image } from '@lucide/vue'
import { useShortcut } from '@/composables/useShortcut'
import { useDebounced } from '@/composables/useDebounced'
import { useUndoableAction } from '@/composables/useUndoableAction'
import { useClipboard } from '@/utils/clipboard'
import { fmtSize, shortId, fmtUnixTime } from '@/utils/format'
import Button from '@/components/ui/Button.vue'
import DataTable, { type DataTableColumn } from '@/components/ui/DataTable.vue'
import RowActions, { type RowAction } from '@/components/ui/RowActions.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import PullImageModal from '@/components/docker/PullImageModal.vue'
import PushImageModal from '@/components/docker/PushImageModal.vue'
import RunImageModal from '@/components/docker/RunImageModal.vue'
import TagImageModal from '@/components/docker/TagImageModal.vue'
import ImageDetailModal from '@/components/docker/ImageDetailModal.vue'
import ImportImageModal from '@/components/docker/ImportImageModal.vue'
import ExportImageModal from '@/components/docker/ExportImageModal.vue'
import { useConfirm } from '@/composables/useConfirm'
import { useToast } from '@/composables/useToast'
import { listImages, pruneImages, removeImage, removeImages, type ImageItem } from '@/api/image'

const loading = ref(false)
const items = ref<ImageItem[]>([])
const errorMsg = ref('')

// 搜索 + 分页（页面内处理）。
const keyword = ref('')
const debouncedKeyword = useDebounced(keyword, 200)
const searchInputRef = useTemplateRef<HTMLInputElement>('searchInput')
// 筛选：all=全部 dangling=仅悬空 used=仅使用中(非悬空)
type FilterMode = 'all' | 'dangling' | 'used'
const filterMode = ref<FilterMode>('all')
const filteredItems = computed(() => {
  let list = items.value
  if (filterMode.value === 'dangling') list = list.filter((i) => i.repo_tags.length === 0)
  else if (filterMode.value === 'used') list = list.filter((i) => i.repo_tags.length > 0)
  const kw = debouncedKeyword.value.trim().toLowerCase()
  if (!kw) return list
  return list.filter((row) =>
    ['repo_tags', 'repo_digests', 'id'].some((k) => {
      const v = (row as unknown as Record<string, unknown>)[k]
      return v != null && String(v).toLowerCase().includes(kw)
    }),
  )
})
const page = ref(1)
const pageSize = 10
const pagedItems = computed(() =>
  filteredItems.value.slice((page.value - 1) * pageSize, page.value * pageSize),
)

const pullOpen = ref(false)
const importOpen = ref(false)
const exportOpen = ref(false)
const exportTarget = ref<{ ids: string[]; tag: string } | null>(null)
const tagOpen = ref(false)
const tagImage = ref<ImageItem | null>(null)
const pushOpen = ref(false)
const runOpen = ref(false)
const actionImage = ref<ImageItem | null>(null)
const detailOpen = ref(false)
const detailImage = ref<ImageItem | null>(null)
const selectedKeys = ref<(string | number)[]>([])

// 二次确认（命令式 hook）。
const confirm = useConfirm()
const toast = useToast()
const { undoableAction } = useUndoableAction()
const { copy } = useClipboard()

async function load(): Promise<void> {
  loading.value = true
  errorMsg.value = ''
  try {
    items.value = await listImages()
    // 数据变化后清理失效的选中。
    const ids = new Set(items.value.map((i) => i.id))
    selectedKeys.value = selectedKeys.value.filter((k) => ids.has(String(k)))
  } catch (err) {
    errorMsg.value = (err as Error).message
  } finally {
    loading.value = false
  }
}

onMounted(load)

// 全局快捷键：/ 聚焦搜索、r 刷新、p 拉取镜像。
useShortcut('/', () => searchInputRef.value?.focus())
useShortcut('r', () => void load())
useShortcut('p', () => { pullOpen.value = true })

function openDetail(image: ImageItem): void {
  detailImage.value = image
  detailOpen.value = true
}

function openPruneConfirm(): void {
  void confirm(
    '清理未使用镜像',
    '将清理所有未被容器使用的镜像（dangling 与未引用镜像），释放磁盘空间。此操作不可撤销，确定继续？',
    async () => {
      const res = await pruneImages(false)
      if (res.deleted > 0) toast.success(`已清理 ${res.deleted} 个镜像，释放 ${fmtSize(res.reclaimed_bytes)}`)
      else toast.info('没有可清理的未使用镜像')
    },
    { danger: true, onSuccess: load },
  )
}

function openPruneDanglingConfirm(): void {
  void confirm(
    '仅清理悬空镜像',
    '将清理所有悬空镜像（<none>，无 tag 引用的镜像），释放磁盘空间。此操作不可撤销，确定继续？',
    async () => {
      const res = await pruneImages(true)
      if (res.deleted > 0) toast.success(`已清理 ${res.deleted} 个悬空镜像，释放 ${fmtSize(res.reclaimed_bytes)}`)
      else toast.info('没有可清理的悬空镜像')
    },
    { danger: true, onSuccess: load },
  )
}

/** 选中项的镜像对象集合。 */
const selectedImages = computed(() => {
  const set = new Set(selectedKeys.value.map(String))
  return items.value.filter((i) => set.has(i.id))
})

function openBatchDeleteConfirm(): void {
  const sel = selectedImages.value
  if (sel.length === 0) return
  void confirm(
    '批量删除镜像',
    `确认删除选中的 ${sel.length} 个镜像？（${sel
      .slice(0, 5)
      .map((i) => i.repo_tags[0] || i.id.slice(7, 19))
      .join('、')}${sel.length > 5 ? ' 等' : ''}）`,
    async () => {
      const res = await removeImages(sel.map((i) => i.id), true)
      toast.success(`已删除 ${res.deleted} 个镜像`)
      selectedKeys.value = []
    },
    { danger: true, onSuccess: load },
  )
}

async function doBatchExport(): Promise<void> {
  const sel = selectedImages.value
  if (sel.length === 0) return
  exportTarget.value = {
    ids: sel.map((i) => i.id),
    tag: sel.length === 1 ? sel[0].repo_tags[0] || sel[0].id : `${sel.length} 个镜像`,
  }
  exportOpen.value = true
}

function openTag(image: ImageItem): void {
  tagImage.value = image
  tagOpen.value = true
}

function openPush(image: ImageItem): void {
  actionImage.value = image
  pushOpen.value = true
}

function openRun(image: ImageItem): void {
  actionImage.value = image
  runOpen.value = true
}

function openExport(image: ImageItem): void {
  exportTarget.value = { ids: [image.id], tag: image.repo_tags[0] || image.id }
  exportOpen.value = true
}

function openRemoveConfirm(image: ImageItem): void {
  const label = image.repo_tags[0] || shortId(image.id)
  undoableAction({
    title: '删除镜像',
    message: `确认删除镜像「${label}」？删除后不可恢复。5 秒内可撤销。`,
    label,
    actionLabel: '删除镜像',
    activityDetail: shortId(image.id),
    action: () => removeImage(image.id, true),
    onDone: load,
  })
}

/** 构建行操作列表（RowActions 组件消费）。 */
function buildImageActions(image: ImageItem): RowAction[] {
  return [
    { key: 'detail', label: '详情', icon: Eye, onClick: () => openDetail(image) },
    { key: 'copyid', label: '复制 ID', icon: Copy, onClick: () => copyId(image) },
    { key: 'tag', label: '打标签', icon: Tag, onClick: () => openTag(image) },
    { key: 'push', label: '推送', icon: Upload, onClick: () => openPush(image) },
    { key: 'run', label: '运行', icon: Play, onClick: () => openRun(image) },
    { key: 'export', label: '导出', icon: FileDown, onClick: () => openExport(image) },
    { key: 'remove', label: '删除', icon: Trash2, danger: true, onClick: () => openRemoveConfirm(image) },
  ]
}
const columns: DataTableColumn[] = [
  { label: '仓库标签', key: 'repo_tags', ellipsis: true },
  { label: '镜像 ID', key: 'id', width: '120px' },
  { label: '大小', key: 'size', width: '100px' },
  { label: '创建时间', key: 'created', width: '160px' },
  { label: '操作', key: 'actions', width: '230px', align: 'right' },
]

/** 复制镜像 ID 到剪贴板。 */
async function copyId(image: ImageItem): Promise<void> {
  await copy(image.id, '已复制镜像 ID')
}
</script>

<template>
  <div class="space-y-5">
    <DataTable
      title="镜像列表"
      :columns="columns"
      :data="pagedItems"
      :loading="loading"
      :error="errorMsg"
      row-key="id"
      pageable
      :page="page"
      :page-size="pageSize"
      :total="filteredItems.length"
      selectable
      :selected-keys="selectedKeys"
      empty-text=""
      @update:page="page = $event"
      @update:selected-keys="selectedKeys = $event"
      @retry="load"
    >
      <template #empty>
        <EmptyState
          :icon="Image"
          title="没有镜像"
          description="拉取一个镜像来开始运行容器，或从 Dockerfile 构建。"
          action-label="拉取镜像"
          doc-url="https://docs.docker.com/engine/reference/commandline/pull/"
          @action="pullOpen = true"
        />
      </template>
      <template #toolbar>
        <div class="flex flex-wrap items-center gap-2">
          <!-- 筛选 -->
          <select
            v-model="filterMode"
            class="h-9 rounded-lg border border-slate-300 bg-white px-2 text-sm text-slate-700 outline-none transition-colors focus:border-blue-500 focus:ring-1 focus:ring-blue-500 dark:border-slate-600 dark:bg-slate-900 dark:text-slate-200"
            aria-label="镜像筛选"
          >
            <option value="all">全部镜像</option>
            <option value="dangling">仅悬空 (&lt;none&gt;)</option>
            <option value="used">仅已标记</option>
          </select>
          <div class="relative">
            <Search class="pointer-events-none absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
          <input
            ref="searchInput"
            v-model="keyword"
              type="text"
              class="h-9 w-56 rounded-lg border border-slate-300 bg-white pl-9 pr-8 text-sm text-slate-800 outline-none transition-colors placeholder:text-slate-400 focus:border-blue-500 focus:ring-1 focus:ring-blue-500 dark:border-slate-600 dark:bg-slate-900 dark:text-slate-100"
              placeholder="搜索仓库 / 标签 / ID / digest…"
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
        </div>
        <!-- 批量操作 -->
        <div v-if="selectedImages.length > 0" class="flex items-center gap-2">
          <span class="text-sm text-slate-500 dark:text-slate-400">已选 {{ selectedImages.length }}</span>
          <Button variant="secondary" size="sm" @click="doBatchExport">批量导出</Button>
          <Button variant="danger" size="sm" @click="openBatchDeleteConfirm">批量删除</Button>
        </div>
        <Button variant="secondary" size="sm" @click="load">刷新</Button>
        <Button variant="danger" size="sm" @click="openPruneConfirm">
          <Trash2 class="mr-1 h-3.5 w-3.5" />清理未使用
        </Button>
        <Button variant="secondary" size="sm" @click="openPruneDanglingConfirm">清理悬空</Button>
        <Button variant="secondary" size="sm" @click="importOpen = true">导入镜像</Button>
        <Button size="sm" @click="pullOpen = true">拉取镜像</Button>
      </template>
      <template #cell-repo_tags="{ row }">
          <button
            class="truncate text-left text-blue-600 hover:underline dark:text-blue-400"
            @click="openDetail(row as ImageItem)"
          >
            {{ (row as ImageItem).repo_tags?.[0] || '(未打标签)' }}
          </button>
        </template>
        <template #cell-id="{ row }">{{ shortId((row as ImageItem).id) }}</template>
        <template #cell-size="{ row }">{{ fmtSize((row as ImageItem).size) }}</template>
        <template #cell-created="{ row }">{{ fmtUnixTime((row as ImageItem).created) }}</template>
        <template #cell-actions="{ row }">
          <RowActions :actions="buildImageActions(row as ImageItem)" :visible="3" />
        </template>
    </DataTable>

    <!-- 镜像详情 -->
    <ImageDetailModal
      v-if="detailImage"
      v-model:open="detailOpen"
      :image-id="detailImage.id"
      :image-tag="detailImage.repo_tags[0]"
    />

    <!-- 拉取镜像 -->
    <PullImageModal v-model:open="pullOpen" @pulled="load" />

    <!-- 导入镜像 -->
    <ImportImageModal v-model:open="importOpen" @imported="load" />

    <!-- 导出镜像 -->
    <ExportImageModal v-if="exportTarget" v-model:open="exportOpen" :target="exportTarget" />

    <!-- 打标签 -->
    <TagImageModal
      v-if="tagImage"
      v-model:open="tagOpen"
      :source-image="tagImage.repo_tags[0] || tagImage.id"
      @tagged="load"
    />

    <!-- 推送镜像 -->
    <PushImageModal
      v-if="actionImage"
      v-model:open="pushOpen"
      :source-tag="actionImage.repo_tags[0] || actionImage.id"
      @pushed="load"
    />

    <!-- 运行镜像 -->
    <RunImageModal
      v-if="actionImage"
      v-model:open="runOpen"
      :image="actionImage.repo_tags[0] || actionImage.id"
      @started="load"
    />
  </div>
</template>
