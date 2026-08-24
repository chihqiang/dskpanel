<script setup lang="ts">
import { computed } from 'vue'

type BadgeVariant = 'gray' | 'green' | 'red' | 'yellow' | 'blue' | 'purple'

const props = withDefaults(
  defineProps<{
    variant?: BadgeVariant
    /** 是否显示状态圆点（颜色随 variant）。 */
    dot?: boolean
  }>(),
  { variant: 'gray', dot: false },
)

const classes = computed(() => {
  const map: Record<BadgeVariant, string> = {
    gray: 'bg-slate-100 text-slate-700 dark:bg-slate-700 dark:text-slate-200',
    green: 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300',
    red: 'bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300',
    yellow: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300',
    blue: 'bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300',
    purple: 'bg-purple-100 text-purple-700 dark:bg-purple-900 dark:text-purple-300',
  }
  return [
    'inline-flex items-center rounded-full px-2.5 py-1 text-sm font-medium whitespace-nowrap',
    map[props.variant],
  ].join(' ')
})

const dotClasses = computed(() => {
  const map: Record<BadgeVariant, string> = {
    gray: 'bg-slate-400 dark:bg-slate-500',
    green: 'bg-green-500 dark:bg-green-400',
    red: 'bg-red-500 dark:bg-red-400',
    yellow: 'bg-yellow-500 dark:bg-yellow-400',
    blue: 'bg-blue-500 dark:bg-blue-400',
    purple: 'bg-purple-500 dark:bg-purple-400',
  }
  return map[props.variant]
})
</script>

<template>
  <span :class="classes">
    <span v-if="dot" class="mr-1.5 inline-block h-1.5 w-1.5 shrink-0 rounded-full" :class="dotClasses" />
    <slot />
  </span>
</template>
