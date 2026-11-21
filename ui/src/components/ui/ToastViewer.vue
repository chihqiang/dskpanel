<script setup lang="ts">
import { CheckCircle2, XCircle, Info, X } from '@lucide/vue'
import { storeToRefs } from 'pinia'
import { useToastStore } from '@/stores/toast'

const store = useToastStore()
const { toasts } = storeToRefs(store)

const icons = {
  success: CheckCircle2,
  error: XCircle,
  info: Info,
}
const labels = {
  success: '成功',
  error: '失败',
  info: '提示',
}
</script>

<template>
  <Teleport to="body">
    <div
      class="pointer-events-none fixed right-5 top-5 z-50 flex w-96 flex-col gap-3"
      role="status"
      aria-live="polite"
    >
      <!-- 顶栏：当有多个 toast 时显示「全部清除」 -->
      <div v-if="toasts.length > 1" class="pointer-events-auto flex justify-end">
        <button
          class="text-xs text-slate-400 transition-colors hover:text-slate-600 dark:hover:text-slate-200"
          @click="store.clear()"
        >
          全部清除
        </button>
      </div>

      <TransitionGroup name="toast">
        <div
          v-for="t in toasts"
          :key="t.id"
          class="pointer-events-auto flex cursor-pointer items-start gap-4 rounded-xl border border-slate-200 bg-white px-5 py-4 shadow-2xl shadow-slate-900/10 ring-1 ring-slate-900/5 transition-transform hover:scale-[1.02] dark:border-slate-700 dark:bg-slate-800 dark:shadow-black/40 dark:ring-white/10"
          @click="store.remove(t.id)"
        >
          <component
            :is="icons[t.type]"
            class="mt-0.5 h-6 w-6 shrink-0"
            :class="{
              'text-emerald-500': t.type === 'success',
              'text-red-500': t.type === 'error',
              'text-blue-500': t.type === 'info',
            }"
          />
          <div class="min-w-0 flex-1">
            <p class="text-sm font-medium text-slate-500 dark:text-slate-400">{{ labels[t.type] }}</p>
            <p class="mt-0.5 break-words text-sm leading-snug text-slate-800 dark:text-slate-100">{{ t.message }}</p>
            <!-- 撤销按钮 -->
            <button
              v-if="t.undo"
              class="pointer-events-auto mt-2 rounded-md bg-slate-100 px-3 py-1 text-xs font-medium text-slate-700 transition-colors hover:bg-slate-200 dark:bg-slate-700 dark:text-slate-200 dark:hover:bg-slate-600"
              @click.stop="store.undo(t.id)"
            >
              {{ t.undoLabel || '撤销' }}
            </button>
          </div>
          <!-- 手动关闭 -->
          <button
            class="shrink-0 text-slate-400 transition-colors hover:text-slate-600 dark:hover:text-slate-200"
            aria-label="关闭提示"
            @click.stop="store.remove(t.id)"
          >
            <X class="h-4 w-4" />
          </button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<style scoped>
.toast-enter-active,
.toast-leave-active {
  transition: all 0.3s cubic-bezier(0.21, 1.02, 0.73, 1);
}
.toast-enter-from {
  opacity: 0;
  transform: translateX(24px);
}
.toast-leave-to {
  opacity: 0;
  transform: translateX(24px);
}
</style>
