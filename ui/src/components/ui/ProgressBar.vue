<script setup lang="ts">
withDefaults(
  defineProps<{
    /** 当前进度 0-1（-1 表示不确定）。 */
    value?: number
    /** 是否不确定（滚动动画）。 */
    indeterminate?: boolean
    /** 高度 px。 */
    height?: number
    /** 颜色（tailwind class）。 */
    color?: string
  }>(),
  {
    value: 0,
    indeterminate: false,
    height: 6,
    color: 'bg-blue-600',
  },
)

</script>

<template>
  <div
    class="w-full overflow-hidden rounded-full bg-slate-200 dark:bg-slate-700"
    :style="{ height: height + 'px' }"
    role="progressbar"
    :aria-valuenow="indeterminate ? undefined : Math.round(value * 100)"
    aria-valuemin="0"
    aria-valuemax="100"
  >
    <!-- 确定进度 -->
    <div
      v-if="!indeterminate"
      class="h-full rounded-full transition-all duration-300 ease-out"
      :class="color"
      :style="{ width: `${Math.min(100, Math.max(0, value * 100))}%` }"
    />
    <!-- 不确定进度（滚动动画） -->
    <div
      v-else
      class="indeterminate-bar h-full rounded-full"
      :class="color"
    />
  </div>
</template>

<style scoped>
.indeterminate-bar {
  width: 40%;
  animation: indeterminate 1.5s infinite ease-in-out;
}

@keyframes indeterminate {
  0% {
    transform: translateX(-100%);
  }
  100% {
    transform: translateX(350%);
  }
}
</style>
