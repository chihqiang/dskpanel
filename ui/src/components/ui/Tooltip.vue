<script setup lang="ts">
/**
 * 轻量级 Tooltip 组件。
 *
 * 基于 CSS hover/focus 实现（无 JS 定位计算），默认显示在上方。
 * 支持 top / bottom / left / right 四个方向。
 *
 * 用法：
 *   <Tooltip text="这是提示">鼠标悬停看看</Tooltip>
 *   <Tooltip text="删除" placement="top">
 *     <Trash2 class="h-4 w-4" />
 *   </Tooltip>
 *
 * 无障碍：触发元素自动添加 aria-describedby，tooltip 添加 role="tooltip"。
 * 支持键盘 focus 显示（Tab 导航时可看到提示）。
 */

import { computed, ref } from 'vue'

const props = withDefaults(
  defineProps<{
    /** 提示文案。 */
    text: string
    /** 显示位置，默认 top。 */
    placement?: 'top' | 'bottom' | 'left' | 'right'
    /** 触发元素 tag，默认 span。 */
    as?: string
  }>(),
  {
    placement: 'top',
    as: 'span',
  },
)

const visible = ref(false)
const tooltipId = `tt-${Math.random().toString(36).slice(2, 9)}`

/** 方向 → 定位 class。 */
const positionClass = computed(() => {
  const map = {
    top: 'bottom-full left-1/2 -translate-x-1/2 mb-1.5',
    bottom: 'top-full left-1/2 -translate-x-1/2 mt-1.5',
    left: 'right-full top-1/2 -translate-y-1/2 mr-1.5',
    right: 'left-full top-1/2 -translate-y-1/2 ml-1.5',
  }
  return map[props.placement]
})

/** 方向 → 小箭头 class。 */
const arrowClass = computed(() => {
  const map = {
    top: 'top-full left-1/2 -translate-x-1/2 border-t-slate-700 dark:border-t-slate-200',
    bottom: 'bottom-full left-1/2 -translate-x-1/2 border-b-slate-700 dark:border-b-slate-200',
    left: 'left-full top-1/2 -translate-y-1/2 border-l-slate-700 dark:border-l-slate-200',
    right: 'right-full top-1/2 -translate-y-1/2 border-r-slate-700 dark:border-r-slate-200',
  }
  return map[props.placement]
})
</script>

<template>
  <component
    :is="as"
    class="relative inline-flex"
    :aria-describedby="visible ? tooltipId : undefined"
    @mouseenter="visible = true"
    @mouseleave="visible = false"
    @focus="visible = true"
    @blur="visible = false"
  >
    <slot />

    <Transition name="tooltip">
      <span
        v-if="visible && text"
        :id="tooltipId"
        role="tooltip"
        class="pointer-events-none absolute z-50 whitespace-nowrap rounded-md bg-slate-700 px-2 py-1 text-xs font-medium text-white shadow-lg dark:bg-slate-200 dark:text-slate-800"
        :class="positionClass"
      >
        {{ text }}
        <!-- 小箭头 -->
        <span
          class="absolute h-0 w-0 border-4 border-transparent"
          :class="arrowClass"
        />
      </span>
    </Transition>
  </component>
</template>

<style scoped>
.tooltip-enter-active,
.tooltip-leave-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}

.tooltip-enter-from,
.tooltip-leave-to {
  opacity: 0;
  transform: scale(0.96);
}
</style>
