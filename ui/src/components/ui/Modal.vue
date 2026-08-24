<script setup lang="ts">
import { ref, watch, onBeforeUnmount } from 'vue'
import { useFocusTrap } from '@/composables/useFocusTrap'

const props = withDefaults(
  defineProps<{
    open: boolean
    title?: string
    width?: string
    closeOnBackdrop?: boolean
  }>(),
  { title: '', width: 'max-w-lg', closeOnBackdrop: true },
)

const emit = defineEmits<{ 'update:open': [value: boolean] }>()

const panel = ref<HTMLElement | null>(null)
useFocusTrap(panel, () => props.open)

function close(): void {
  emit('update:open', false)
}

function onBackdrop(): void {
  if (props.closeOnBackdrop) {
    close()
  }
}

function onKeydown(e: KeyboardEvent): void {
  if (e.key === 'Escape') {
    close()
  }
}

watch(
  () => props.open,
  (open) => {
    if (open) {
      document.addEventListener('keydown', onKeydown)
      document.body.style.overflow = 'hidden'
    } else {
      document.removeEventListener('keydown', onKeydown)
      document.body.style.overflow = ''
    }
  },
)

onBeforeUnmount(() => {
  document.removeEventListener('keydown', onKeydown)
  document.body.style.overflow = ''
})
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center p-4" role="dialog" aria-modal="true">
        <!-- 遮罩 -->
        <div class="absolute inset-0 bg-black/50" @click="onBackdrop" />
        <!-- 面板 -->
        <div
          ref="panel"
          tabindex="-1"
          class="relative flex max-h-[85vh] w-full flex-col rounded-xl bg-white shadow-xl outline-none dark:bg-slate-800"
          :class="width"
        >
          <div class="flex shrink-0 items-center justify-between border-b border-slate-200 px-6 py-5 dark:border-slate-700">
            <h3 class="text-lg font-medium text-slate-900 dark:text-slate-100">{{ title }}</h3>
            <button
              class="text-slate-400 transition-colors hover:text-slate-600 dark:hover:text-slate-200"
              @click="close"
              aria-label="关闭"
            >
              <svg class="h-6 w-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M18 6L6 18M6 6l12 12" stroke-linecap="round" />
              </svg>
            </button>
          </div>
          <div class="flex-1 overflow-y-auto px-6 py-5">
            <slot />
          </div>
          <div v-if="$slots.footer" class="flex shrink-0 justify-end gap-2 border-t border-slate-200 px-6 py-4 dark:border-slate-700">
            <slot name="footer" />
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.15s ease;
}
.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}
</style>
