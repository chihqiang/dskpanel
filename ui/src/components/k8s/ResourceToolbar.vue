<script setup lang="ts">
import { FileCode2, Search, X } from '@lucide/vue'
import Button from '@/components/ui/Button.vue'
import NamespaceSelect from '@/components/k8s/NamespaceSelect.vue'

/**
 * 通用资源表格工具栏。
 *
 * 统一封装列表页顶部的常用操作：
 * - 左侧：默认插槽（Tab 切换等）
 * - 右侧：YAML 创建按钮 + 命名空间选择 + 名称搜索 + extra 插槽（刷新等）
 *
 * 用法：
 *   <ResourceToolbar v-model:keyword="keyword" @create="createOpen = true">
 *     <template #extra><Button>刷新</Button></template>
 *   </ResourceToolbar>
 */
withDefaults(
  defineProps<{
    /** 搜索关键词（v-model）。 */
    keyword?: string
    /** 是否显示 YAML 创建按钮。 */
    showCreate?: boolean
    /** 是否显示命名空间选择。 */
    showNamespace?: boolean
    /** 创建按钮文案。 */
    createLabel?: string
    /** 搜索框占位。 */
    placeholder?: string
  }>(),
  {
    keyword: '',
    showCreate: true,
    showNamespace: true,
    createLabel: 'YAML 创建',
    placeholder: '搜索名称…',
  },
)

const emit = defineEmits<{
  'update:keyword': [value: string]
  /** 点击 YAML 创建按钮。 */
  create: []
}>()

defineSlots<{
  /** 左侧区域（如 Tab 切换）。 */
  default?: () => unknown
  /** 右侧额外按钮（如刷新）。 */
  extra?: () => unknown
}>()
</script>

<template>
  <div
    class="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-slate-200 bg-white px-4 py-3 shadow-sm dark:border-slate-700 dark:bg-slate-800"
  >
    <!-- 左侧：Tab 等 -->
    <div class="flex flex-wrap items-center gap-2">
      <slot />
    </div>

    <!-- 右侧：创建 + 命名空间 + 搜索 + extra -->
    <div class="flex flex-wrap items-center gap-2">
      <Button v-if="showCreate" size="sm" @click="emit('create')">
        <FileCode2 class="h-3.5 w-3.5" />
        {{ createLabel }}
      </Button>
      <NamespaceSelect v-if="showNamespace" />
      <div class="relative">
        <Search class="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-slate-400" />
        <input
          :value="keyword"
          type="text"
          class="h-8 w-44 rounded-md border border-slate-200 bg-white pl-8 pr-7 text-sm text-slate-700 outline-none transition-colors placeholder:text-slate-400 focus:border-blue-400 focus:ring-1 focus:ring-blue-400 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:border-blue-500"
          :placeholder="placeholder"
          @input="emit('update:keyword', ($event.target as HTMLInputElement).value)"
        />
        <button
          v-if="keyword"
          class="absolute right-2 top-1/2 -translate-y-1/2 text-slate-400 transition-colors hover:text-slate-600"
          aria-label="清空搜索"
          @click="emit('update:keyword', '')"
        >
          <X class="h-3.5 w-3.5" />
        </button>
      </div>
      <slot name="extra" />
    </div>
  </div>
</template>
