<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, type Component } from 'vue'
import { ChevronDown } from '@lucide/vue'
import Button from '@/components/ui/Button.vue'

/** 单个行操作项。 */
export interface RowAction {
  /** 唯一标识。 */
  key: string
  /** 按钮文案。 */
  label: string
  /** 图标组件（可选）。 */
  icon?: Component
  /** 危险操作（红色）。 */
  danger?: boolean
  /** 禁用。 */
  disabled?: boolean
  /** 加载中（显示 spinner）。 */
  loading?: boolean
  /** 点击回调。 */
  onClick?: () => void
}

const props = withDefaults(
  defineProps<{
    /** 全部操作列表。 */
    actions: RowAction[]
    /** 直接显示的操作数（其余折叠进「更多」），默认 3。 */
    visible?: number
    /** 更多按钮文案。 */
    moreLabel?: string
    /** 对齐方式。 */
    align?: 'left' | 'right'
  }>(),
  { visible: 3, moreLabel: '更多', align: 'right' },
)

const open = ref(false)
const triggerEl = ref<HTMLElement | null>(null)
const menuPos = ref<{ top: number; left: number }>({ top: 0, left: 0 })

/** Button 是组件，函数 ref 取其根元素 $el。 */
function setTrigger(instance: unknown): void {
  triggerEl.value = (instance as { $el?: HTMLElement } | null)?.$el ?? null
}

/** 直接显示的操作。 */
const visibleActions = computed(() => props.actions.slice(0, props.visible))
/** 折叠进「更多」的操作。 */
const moreActions = computed(() => props.actions.slice(props.visible))

function positionMenu(): void {
  const el = triggerEl.value
  if (!el) return
  const r = el.getBoundingClientRect()
  const menuWidth = 168
  const estH = Math.min(moreActions.value.length, 10) * 34 + 8
  const spaceBelow = window.innerHeight - r.bottom
  const openDown = spaceBelow >= estH + 8 || spaceBelow >= r.top
  const left = Math.max(8, Math.min(r.right - menuWidth, window.innerWidth - menuWidth - 8))
  menuPos.value = openDown
    ? { top: r.bottom + 4, left }
    : { top: Math.max(8, r.top - estH - 4), left }
}

function toggleMenu(): void {
  if (open.value) {
    open.value = false
    return
  }
  positionMenu()
  open.value = true
}

function onWindowClick(e: MouseEvent): void {
  if (!triggerEl.value?.contains(e.target as Node)) {
    open.value = false
  }
}

function onWindowScroll(): void {
  open.value = false
}

function onWindowKeydown(e: KeyboardEvent): void {
  if (e.key === 'Escape') open.value = false
}

/** 点击「更多」中的操作：先执行回调，再关闭下拉。 */
function handleMoreClick(a: RowAction): void {
  open.value = false
  a.onClick?.()
}

onMounted(() => {
  document.addEventListener('click', onWindowClick)
  window.addEventListener('scroll', onWindowScroll, true)
  window.addEventListener('resize', onWindowScroll)
  document.addEventListener('keydown', onWindowKeydown)
})
onBeforeUnmount(() => {
  document.removeEventListener('click', onWindowClick)
  window.removeEventListener('scroll', onWindowScroll, true)
  window.removeEventListener('resize', onWindowScroll)
  document.removeEventListener('keydown', onWindowKeydown)
})
</script>

<template>
  <div class="flex items-center gap-1" :class="align === 'right' ? 'justify-end' : 'justify-start'">
    <!-- 直接显示的操作 -->
    <Button
      v-for="a in visibleActions"
      :key="a.key"
      variant="ghost"
      size="sm"
      :disabled="a.disabled"
      :loading="a.loading"
      :class="a.danger ? '!text-red-600' : ''"
      @click="a.onClick"
    >
      <component :is="a.icon" v-if="a.icon" class="h-3.5 w-3.5" />
      {{ a.label }}
    </Button>

    <!-- 更多按钮（存在折叠操作时） -->
    <div v-if="moreActions.length > 0" class="relative">
      <Button
        :ref="setTrigger"
        variant="ghost"
        size="sm"
        :aria-expanded="open"
        aria-haspopup="menu"
        :class="open ? 'bg-slate-100 dark:bg-slate-700' : ''"
        @click="toggleMenu"
      >
        <ChevronDown class="h-3.5 w-3.5" :class="open ? 'rotate-180' : ''" />
        {{ moreLabel }}
      </Button>
    </div>
  </div>

  <!-- 折叠菜单：fixed 定位（脱离文档流，不受父级 overflow 裁剪）+ v-show 保留 Transition -->
  <Transition
    enter-active-class="transition duration-100 ease-out"
    enter-from-class="opacity-0 scale-95"
    enter-to-class="opacity-100 scale-100"
    leave-active-class="transition duration-75 ease-in"
    leave-from-class="opacity-100 scale-100"
    leave-to-class="opacity-0 scale-95"
  >
    <div
      v-show="open"
      role="menu"
      :style="{ top: menuPos.top + 'px', left: menuPos.left + 'px' }"
      class="fixed z-50 w-42 max-h-80 overflow-y-auto rounded-lg border border-slate-200 bg-white py-1 shadow-lg dark:border-slate-700 dark:bg-slate-800"
    >
        <button
          v-for="a in moreActions"
          :key="a.key"
          role="menuitem"
          class="flex w-full items-center gap-2 px-3 py-2 text-left text-sm transition-colors hover:bg-slate-50 dark:hover:bg-slate-700"
          :class="a.danger ? 'text-red-600 dark:text-red-400' : 'text-slate-700 dark:text-slate-200'"
          :disabled="a.disabled"
          @click="handleMoreClick(a)"
        >
          <component :is="a.icon" v-if="a.icon" class="h-4 w-4 shrink-0" />
          {{ a.label }}
        </button>
      </div>
    </Transition>
</template>
