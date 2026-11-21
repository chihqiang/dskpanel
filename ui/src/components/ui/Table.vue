<script lang="ts">
/** 表格列定义。 */
export interface TableColumn {
  /** 列标题。 */
  title: string
  /** 数据字段路径（若用 slot 渲染则可不填）。 */
  key?: string
  /** 宽度。 */
  width?: string
  /** 是否允许省略（过长截断）。 */
  ellipsis?: boolean
  /** 对齐方式。 */
  align?: 'left' | 'center' | 'right'
}
</script>

<script setup lang="ts">
import { computed } from 'vue'
import Skeleton from './Skeleton.vue'

const props = withDefaults(
  defineProps<{
    columns: TableColumn[]
    data: unknown[]
    /** 加载中。 */
    loading?: boolean
    /** 空数据提示文案。 */
    emptyText?: string
    /** 行 key 字段（默认 id）。 */
    rowKey?: string
  }>(),
  {
    loading: false,
    emptyText: '暂无数据',
    rowKey: 'id',
  },
)

defineSlots<{
  default(props: { row: unknown; rowIndex: number }): unknown
  [key: string]: (props: { row: unknown; rowIndex: number }) => unknown
}>()

const alignClass = computed(() => ({
  left: 'text-left',
  center: 'text-center',
  right: 'text-right',
}))
</script>

<template>
  <div class="overflow-x-auto">
    <table class="w-full text-sm">
      <thead>
        <tr class="border-b border-slate-200 text-left text-sm text-slate-500 dark:border-slate-700 dark:text-slate-400">
          <th
            v-for="col in columns"
            :key="col.key ?? col.title"
            :class="alignClass[col.align ?? 'left']"
            :style="{ width: col.width }"
            class="px-5 py-3 font-medium"
          >
            {{ col.title }}
          </th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="(row, idx) in data"
          :key="String((row as Record<string, unknown>)[rowKey] ?? idx)"
          class="border-b border-slate-100 transition-colors hover:bg-slate-50 dark:border-slate-800 dark:hover:bg-slate-700/40"
        >
          <td
            v-for="col in columns"
            :key="col.key ?? col.title"
            :class="[
              alignClass[col.align ?? 'left'],
              col.ellipsis ? 'max-w-0 truncate' : '',
            ]"
            class="px-5 py-3 text-slate-700 dark:text-slate-200"
          >
            <!-- 优先用插槽渲染，否则取字段值 -->
            <slot :name="col.key" :row="row" :row-index="idx">
              <span :class="col.ellipsis ? 'block truncate' : ''">
                {{ (row as Record<string, unknown>)[col.key ?? ''] ?? '-' }}
              </span>
            </slot>
          </td>
        </tr>
        <tr v-if="loading">
          <td :colspan="columns.length" class="px-5 py-6">
            <div class="space-y-3">
              <Skeleton v-for="i in 4" :key="i" :count="1" height="h-4" />
            </div>
          </td>
        </tr>
        <tr v-else-if="data.length === 0">
          <td :colspan="columns.length" class="px-5 py-12 text-center text-slate-400">{{ emptyText }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
