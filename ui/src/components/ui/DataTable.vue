<script lang="ts">
/** 数据表格列定义。 */
export interface DataTableColumn {
  /** 列标识 / 数据字段 key。 */
  key: string
  /** 列标题。 */
  label: string
  /** 宽度。 */
  width?: string
  /** 对齐方式。 */
  align?: 'left' | 'center' | 'right'
  /** 应用到 th/td 的额外 class。 */
  class?: string
  /** 是否允许省略（过长截断）。 */
  ellipsis?: boolean
}
</script>

<script setup lang="ts">
import { computed, useSlots } from 'vue'
import { RefreshCw, AlertCircle } from '@lucide/vue'
import Skeleton from './Skeleton.vue'

const props = withDefaults(
  defineProps<{
    /** 表格标题（卡片头部左侧）。 */
    title?: string
    columns: DataTableColumn[]
    /** 展示数据（由使用方过滤/切片后传入）。 */
    data: unknown[]
    /** 加载中。 */
    loading?: boolean
    /** 错误信息（非空时显示在表格上方）。 */
    error?: string
    /** 空数据提示文案。 */
    emptyText?: string
    /** 行 key 字段（默认 id）。 */
    rowKey?: string
    /** 是否显示分页。 */
    pageable?: boolean
    /** 当前页码（1 起）。 */
    page?: number
    /** 每页条数。 */
    pageSize?: number
    /** 总条数。 */
    total?: number
    /** 是否支持行选择（勾选）。 */
    selectable?: boolean
    /** 已选中的行 key 集合。 */
    selectedKeys?: (string | number)[]
  }>(),
  {
    title: '',
    loading: false,
    error: '',
    emptyText: '暂无数据',
    rowKey: 'id',
    pageable: false,
    page: 1,
    pageSize: 10,
    total: 0,
    selectable: false,
    selectedKeys: () => [],
  },
)

const emit = defineEmits<{
  'update:page': [page: number]
  'update:selectedKeys': [keys: (string | number)[]]
  /** 点击行（点击操作按钮/输入框等不触发）。 */
  'row-click': [row: unknown]
  /** 点击重试按钮（error 非空时显示）。 */
  retry: []
}>()

const slots = useSlots()

defineSlots<{
  /** 工具栏（如搜索框、刷新、新建按钮）及其它列插槽 `#cell-{key}`。 */
  [key: string]: (props: { row: unknown; rowIndex: number; value?: unknown }) => unknown
}>()

const totalPages = computed(() =>
  props.total > 0 ? Math.max(1, Math.ceil(props.total / props.pageSize)) : 1,
)

/** 从未知类型的行数据中取值。 */
function cellValue(row: unknown, key: string): unknown {
  return (row as Record<string, unknown>)[key]
}

function alignClass(align?: string): string {
  if (align === 'right') return 'text-right'
  if (align === 'center') return 'text-center'
  return 'text-left'
}

function goPage(p: number): void {
  if (p < 1 || p > totalPages.value || p === props.page || props.loading) return
  emit('update:page', p)
}

/** 点击行：忽略交互元素（按钮/链接/输入框等）上的点击。 */
function onRowClick(e: MouseEvent, row: unknown): void {
  const target = e.target as HTMLElement
  if (target.closest('button, a, input, select, textarea, label')) return
  emit('row-click', row)
}

/** 当前页所有行的 key。 */
const pageRowKeys = computed<(string | number)[]>(() =>
  props.data.map((row) => String((row as Record<string, unknown>)[props.rowKey] ?? '') as string | number),
)

/** 当前页是否全部选中。 */
const allPageSelected = computed(() => {
  if (!props.selectable || pageRowKeys.value.length === 0) return false
  return pageRowKeys.value.every((k) => props.selectedKeys.includes(k))
})

function toggleRow(key: string | number, checked: boolean): void {
  const set = new Set(props.selectedKeys)
  if (checked) set.add(key)
  else set.delete(key)
  emit('update:selectedKeys', Array.from(set))
}

function togglePage(checked: boolean): void {
  const set = new Set(props.selectedKeys)
  for (const k of pageRowKeys.value) {
    if (checked) set.add(k)
    else set.delete(k)
  }
  emit('update:selectedKeys', Array.from(set))
}
</script>

<template>
  <div class="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm dark:border-slate-700 dark:bg-slate-800">
    <!-- 头部：标题 + 工具栏 -->
    <div
      v-if="title || $slots.toolbar"
      class="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 px-5 py-4 dark:border-slate-700"
    >
      <h2 class="text-lg font-semibold text-slate-800 dark:text-slate-100">{{ title }}</h2>
      <div class="flex shrink-0 flex-wrap items-center gap-2">
        <slot name="toolbar" :row="undefined" :row-index="0" />
      </div>
    </div>

    <!-- 错误提示（带重试按钮） -->
    <div
      v-if="error"
      class="flex flex-wrap items-center gap-3 border-b border-slate-100 px-5 py-3 dark:border-slate-800"
      role="alert"
    >
      <AlertCircle class="h-4 w-4 shrink-0 text-red-500" />
      <span class="text-sm text-red-600 dark:text-red-400">{{ error }}</span>
      <button
        class="ml-auto inline-flex items-center gap-1.5 rounded-md border border-slate-200 px-2.5 py-1 text-xs text-slate-600 transition-colors hover:bg-slate-100 dark:border-slate-600 dark:text-slate-300 dark:hover:bg-slate-700"
        :disabled="loading"
        @click="emit('retry')"
      >
        <RefreshCw class="h-3.5 w-3.5" :class="loading ? 'animate-spin' : ''" />
        重试
      </button>
    </div>

    <!-- 表格主体 -->
    <div class="overflow-x-auto">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b border-slate-200 text-left text-sm font-medium text-slate-500 dark:border-slate-700 dark:text-slate-400">
            <th v-if="selectable" class="w-10 px-4 py-3">
              <input
                type="checkbox"
                class="h-4 w-4 cursor-pointer rounded border-slate-300 text-blue-600 focus:ring-blue-500 dark:border-slate-600 dark:bg-slate-800"
                :checked="allPageSelected"
                :disabled="loading || data.length === 0"
                aria-label="全选当前页"
                @change="togglePage(($event.target as HTMLInputElement).checked)"
              />
            </th>
            <th
              v-for="col in columns"
              :key="col.key"
              :class="[alignClass(col.align), col.class]"
              :style="col.width ? `width: ${col.width}` : undefined"
              class="px-5 py-3"
            >
              {{ col.label }}
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading">
            <td :colspan="columns.length + (selectable ? 1 : 0)" class="px-5 py-6">
              <div class="space-y-3">
                <Skeleton v-for="i in 5" :key="i" :count="1" height="h-4" />
              </div>
            </td>
          </tr>
            <tr v-else-if="data.length === 0">
            <td :colspan="columns.length + (selectable ? 1 : 0)" class="px-5 py-12 text-center text-slate-400">
              <slot name="empty" :row="undefined" :row-index="0">{{ emptyText }}</slot>
            </td>
          </tr>
          <tr
            v-for="(row, idx) in data"
            :key="String((row as Record<string, unknown>)[rowKey] ?? idx)"
            class="cursor-pointer border-b border-slate-100 transition-colors last:border-0 hover:bg-slate-50 dark:border-slate-800 dark:hover:bg-slate-700/40"
            @click="onRowClick($event, row)"
          >            <td v-if="selectable" class="px-4 py-3">
              <input
                type="checkbox"
                class="h-4 w-4 cursor-pointer rounded border-slate-300 text-blue-600 focus:ring-blue-500 dark:border-slate-600 dark:bg-slate-800"
                :checked="selectedKeys.includes(String((row as Record<string, unknown>)[rowKey] ?? ''))"
                :aria-label="`选择第 ${idx + 1} 行`"
                @change="toggleRow(String((row as Record<string, unknown>)[rowKey] ?? ''), ($event.target as HTMLInputElement).checked)"
              />
            </td>            <td
              v-for="col in columns"
              :key="col.key"
              :class="[alignClass(col.align), col.class, col.ellipsis ? 'max-w-0 truncate' : '']"
              class="px-5 py-3 text-slate-700 dark:text-slate-200"
            >
              <!-- 提供 #cell-{key} 插槽时渲染插槽；否则回退到 row[key] 原文 -->
              <slot
                v-if="slots[`cell-${col.key}`]"
                :name="`cell-${col.key}`"
                :row="row"
                :row-index="idx"
                :value="cellValue(row, col.key)"
              />
              <template v-else>
                <span :class="col.ellipsis ? 'block truncate' : ''">{{ cellValue(row, col.key) ?? '—' }}</span>
              </template>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 分页 -->
    <div
      v-if="pageable"
      class="flex items-center justify-between border-t border-slate-200 px-5 py-3 dark:border-slate-700"
    >
      <p class="text-sm text-slate-500 dark:text-slate-400">共 {{ total }} 条 · 第 {{ page }} / {{ totalPages }} 页</p>
      <div class="flex items-center gap-2">
        <button
          class="h-8 rounded-md border border-slate-200 px-3 text-sm text-slate-600 transition-colors hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-40 dark:border-slate-600 dark:text-slate-300 dark:hover:bg-slate-700"
          :disabled="page <= 1 || loading"
          @click="goPage(page - 1)"
        >
          上一页
        </button>
        <button
          class="h-8 rounded-md border border-slate-200 px-3 text-sm text-slate-600 transition-colors hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-40 dark:border-slate-600 dark:text-slate-300 dark:hover:bg-slate-700"
          :disabled="page >= totalPages || loading"
          @click="goPage(page + 1)"
        >
          下一页
        </button>
      </div>
    </div>
  </div>
</template>
