<script setup lang="ts">
import type { Component } from 'vue'
import Button from '@/components/ui/Button.vue'

withDefaults(
  defineProps<{
    /** 主图标。 */
    icon: Component
    /** 标题。 */
    title: string
    /** 描述文案。 */
    description?: string
    /** 主行动按钮文案。 */
    actionLabel?: string
    /** 文档链接（可选）。 */
    docUrl?: string
    /** 文档链接文案。 */
    docLabel?: string
  }>(),
  {
    docLabel: '查看文档',
  },
)

const emit = defineEmits<{ action: [] }>()
</script>

<template>
  <div class="flex flex-col items-center justify-center gap-4 px-6 py-16 text-center">
    <div class="flex h-16 w-16 items-center justify-center rounded-2xl bg-slate-100 dark:bg-slate-700">
      <component :is="icon" class="h-8 w-8 text-slate-400 dark:text-slate-500" />
    </div>
    <div class="max-w-sm">
      <p class="text-base font-semibold text-slate-700 dark:text-slate-200">{{ title }}</p>
      <p v-if="description" class="mt-1.5 text-sm leading-relaxed text-slate-500 dark:text-slate-400">{{ description }}</p>
    </div>
    <div class="flex flex-wrap items-center justify-center gap-3">
      <Button v-if="actionLabel" size="sm" @click="emit('action')">
        {{ actionLabel }}
      </Button>
      <a
        v-if="docUrl"
        :href="docUrl"
        target="_blank"
        rel="noopener noreferrer"
        class="inline-flex h-8 items-center gap-1.5 rounded-md border border-slate-200 px-3 text-sm text-slate-600 transition-colors hover:bg-slate-100 dark:border-slate-600 dark:text-slate-300 dark:hover:bg-slate-700"
      >
        {{ docLabel }}
      </a>
    </div>
  </div>
</template>
