<script setup lang="ts">
import { ref, watch } from 'vue'
import { Copy, Check } from '@lucide/vue'
import Button from '@/components/ui/Button.vue'
import Modal from '@/components/ui/Modal.vue'
import { useClipboard } from '@/utils/clipboard'

const props = defineProps<{
  open: boolean
  title: string
  /** 拉取资源 YAML 文本的函数（由父组件按资源类型构造）。 */
  fetchYaml: (() => Promise<string>) | null
}>()

const emit = defineEmits<{ 'update:open': [value: boolean] }>()

const { copy } = useClipboard()

const content = ref('')
const loading = ref(false)
const errorMsg = ref('')
const copied = ref(false)

watch(
  () => [props.open, props.fetchYaml] as const,
  ([open, fn]) => {
    if (open && fn) {
      content.value = ''
      errorMsg.value = ''
      loading.value = true
      fn()
        .then((text) => {
          content.value = text
        })
        .catch((err: Error) => {
          errorMsg.value = err.message
        })
        .finally(() => {
          loading.value = false
        })
    }
  },
  { immediate: true },
)

async function copyYaml(): Promise<void> {
  await copy(content.value, '已复制到剪贴板', '复制失败，请手动复制')
  copied.value = true
  setTimeout(() => (copied.value = false), 1500)
}
</script>

<template>
  <Modal :open="open" @update:open="emit('update:open', $event)" :title="title" width="max-w-3xl">
    <div class="min-h-[40vh]">
      <div v-if="loading" class="flex items-center justify-center py-16 text-sm text-slate-400">
        <svg class="mr-2 h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none" aria-hidden="true">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
        </svg>
        加载中…
      </div>
      <div v-else-if="errorMsg" class="rounded-lg bg-red-50 px-4 py-3 text-sm text-red-600 dark:bg-red-900/20 dark:text-red-300">
        {{ errorMsg }}
      </div>
      <pre
        v-else
        class="max-h-[60vh] overflow-auto rounded-lg bg-slate-900 px-4 py-3 font-mono text-xs leading-relaxed text-green-300"
      >{{ content || '（空）' }}</pre>
    </div>
    <template #footer>
      <div class="flex items-center gap-2">
        <Button variant="secondary" size="sm" :disabled="!content" @click="copyYaml">
          <Check v-if="copied" class="h-3.5 w-3.5 text-green-500" />
          <Copy v-else class="h-3.5 w-3.5" />
          复制
        </Button>
      </div>
      <div class="ml-auto">
        <Button variant="secondary" @click="emit('update:open', false)">关闭</Button>
      </div>
    </template>
  </Modal>
</template>
