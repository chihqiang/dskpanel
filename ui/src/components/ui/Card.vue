<script setup lang="ts">
withDefaults(
  defineProps<{
    title?: string
    /** 是否带内边距（默认 true）。 */
    padded?: boolean
  }>(),
  { title: '', padded: true },
)
</script>

<template>
  <div class="rounded-xl border border-slate-200 bg-white shadow-sm dark:border-slate-700 dark:bg-slate-800">
    <div v-if="title" class="border-b border-slate-200 px-5 py-4 text-base font-medium text-slate-800 dark:border-slate-700 dark:text-slate-100">
      {{ title }}
    </div>
    <div :class="padded ? 'p-5' : ''">
      <slot />
    </div>
  </div>
</template>
