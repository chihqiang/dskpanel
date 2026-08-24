<script setup lang="ts">
/**
 * 骨架屏组件。
 *
 * 在数据加载期间用灰色占位块替代"加载中..."文案，
 * 提供更接近真实内容结构的视觉反馈，减少布局跳动。
 *
 * 用法：
 *   <Skeleton class="h-4 w-32" />           // 单行
 *   <Skeleton :count="5" />                  // 多行列表
 *   <Skeleton circle class="h-10 w-10" />    // 圆形（头像）
 *   <Skeleton lines />                       // 文本段落（3 行递减宽度）
 *
 * 无障碍：添加 aria-hidden + role="status" + aria-label。
 */

withDefaults(
  defineProps<{
    /** 渲染几条占位条（默认 1）。 */
    count?: number
    /** 是否圆形（头像/图标占位）。 */
    circle?: boolean
    /** 文本段落模式：3 行递减宽度（100% / 90% / 60%）。 */
    lines?: boolean
    /** 高度 class（如 h-4），默认 h-4。 */
    height?: string
  }>(),
  {
    count: 1,
    circle: false,
    lines: false,
    height: 'h-4',
  },
)
</script>

<template>
  <!-- 外层 role=status 供屏幕阅读器播报加载状态 -->
  <div role="status" aria-label="内容加载中" class="animate-pulse">
    <!-- 文本段落模式 -->
    <template v-if="lines">
      <div class="space-y-2.5">
        <div class="h-4 w-full rounded bg-slate-200 dark:bg-slate-700" />
        <div class="h-4 w-full rounded bg-slate-200 dark:bg-slate-700" />
        <div class="h-4 w-3/5 rounded bg-slate-200 dark:bg-slate-700" />
      </div>
    </template>

    <!-- 多行列表模式 -->
    <template v-else-if="count > 1">
      <div class="space-y-2.5">
        <div
          v-for="i in count"
          :key="i"
          :class="[height, circle ? 'rounded-full' : 'rounded']"
          class="w-full bg-slate-200 dark:bg-slate-700"
        />
      </div>
    </template>

    <!-- 单条占位 -->
    <template v-else>
      <div
        :class="[height, circle ? 'rounded-full' : 'rounded']"
        class="w-full bg-slate-200 dark:bg-slate-700"
      />
    </template>

    <!-- 屏幕阅读器隐藏文案 -->
    <span class="sr-only">加载中...</span>
  </div>
</template>
