<script setup lang="ts">
import { ref, watch, onBeforeUnmount } from 'vue'
import { AlertTriangle } from '@lucide/vue'
import Button from '@/components/ui/Button.vue'
import { useFocusTrap } from '@/composables/useFocusTrap'

const props = withDefaults(
  defineProps<{
    open: boolean
    title: string
    message: string
    confirmText?: string
    cancelText?: string
    /** 确认中（按钮转圈并禁用）。 */
    loading?: boolean
    /** 危险操作（红色确认按钮）。 */
    danger?: boolean
  }>(),
  {
    confirmText: '确认',
    cancelText: '取消',
    loading: false,
    danger: false,
  },
)

const emit = defineEmits<{
  confirm: []
  cancel: []
}>()

const panel = ref<HTMLElement | null>(null)
useFocusTrap(panel, () => props.open)

function onKeydown(e: KeyboardEvent): void {
  if (e.key === 'Escape' && props.open) {
    emit('cancel')
  }
}

watch(
  () => props.open,
  (open) => {
    if (open) document.addEventListener('keydown', onKeydown)
    else document.removeEventListener('keydown', onKeydown)
  },
)

onBeforeUnmount(() => {
  document.removeEventListener('keydown', onKeydown)
})
</script>

<template>
  <Teleport to="body">
    <Transition name="confirm">
      <div
        v-if="open"
        class="fixed inset-0 z-50 flex items-center justify-center p-4"
        role="alertdialog"
        aria-modal="true"
        :aria-label="title"
      >
        <!-- 遮罩 -->
        <div class="absolute inset-0 bg-black/50" @click="emit('cancel')" />
        <!-- 面板 -->
        <div
          ref="panel"
          tabindex="-1"
          class="relative w-full max-w-md rounded-2xl bg-white shadow-2xl outline-none ring-1 ring-slate-900/5 dark:bg-slate-800 dark:ring-white/10"
        >
          <div class="flex gap-5 px-6 py-6">
            <div
              class="flex h-12 w-12 shrink-0 items-center justify-center rounded-full"
              :class="
                danger
                  ? 'bg-red-50 text-red-600 dark:bg-red-900/40 dark:text-red-400'
                  : 'bg-amber-50 text-amber-600 dark:bg-amber-900/40 dark:text-amber-400'
              "
            >
              <AlertTriangle class="h-6 w-6" />
            </div>
            <div class="min-w-0 pt-0.5">
              <h3 class="text-lg font-semibold leading-snug text-slate-900 dark:text-slate-100">{{ title }}</h3>
              <p class="mt-2 text-sm leading-relaxed text-slate-500 dark:text-slate-400">{{ message }}</p>
            </div>
          </div>
          <div class="flex justify-end gap-3 border-t border-slate-200 px-6 py-4 dark:border-slate-700">
            <Button variant="secondary" size="md" :disabled="loading" @click="emit('cancel')">{{ cancelText }}</Button>
            <Button size="md" :variant="danger ? 'danger' : 'primary'" :loading="loading" @click="emit('confirm')">
              {{ confirmText }}
            </Button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.confirm-enter-active,
.confirm-leave-active {
  transition: opacity 0.18s ease;
}
.confirm-enter-from,
.confirm-leave-to {
  opacity: 0;
}
.confirm-enter-from .relative,
.confirm-leave-to .relative {
  opacity: 0;
  transform: translateY(12px) scale(0.97);
}
</style>
