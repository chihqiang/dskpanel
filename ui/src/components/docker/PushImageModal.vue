<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import Button from '@/components/ui/Button.vue'
import Modal from '@/components/ui/Modal.vue'
import ProgressBar from '@/components/ui/ProgressBar.vue'
import { useToast } from '@/composables/useToast'
import { pushImageStream } from '@/api/image'

const props = defineProps<{ open: boolean; sourceTag?: string }>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  pushed: []
}>()

const pushRef = ref('')
const pushing = ref(false)
const toast = useToast()
const done = ref(false)
/** 按层 ID 聚合的最新进度文本。 */
const layerProgress = ref<Record<string, string>>({})
let stop: (() => void) | null = null

/** 整体进度估算：已完成层数 / 已知总层数。 */
const overallProgress = computed(() => {
  const entries = Object.values(layerProgress.value)
  if (entries.length === 0) return -1
  const completed = entries.filter(
    (p) => p.includes('Pushed') || p.includes('already exists') || p.includes('Mounted from'),
  ).length
  return completed / entries.length
})

watch(
  () => props.open,
  (open) => {
    if (open) {
      done.value = false
      layerProgress.value = {}
      // 预填源标签，用户可改成目标地址。
      pushRef.value = props.sourceTag || ''
    }
  },
)

function submit(): void {
  if (!pushRef.value) return
  pushing.value = true
  done.value = false
  layerProgress.value = {}

  stop = pushImageStream(
    pushRef.value,
    (msg) => {
      // 按层 ID 聚合最新状态（对应 docker push CLI 输出）。
      const status = (msg.status as string) ?? ''
      const progress = (msg.progress as string) ?? ''
      const id = (msg.id as string) ?? ''
      if (id) {
        layerProgress.value[id] = progress || status
      }
    },
    () => {
      done.value = true
      pushing.value = false
      emit('pushed')
    },
    (msg) => {
      toast.error(msg)
      pushing.value = false
    },
  )
}

function close(): void {
  stop?.()
  stop = null
  emit('update:open', false)
}

onBeforeUnmount(() => {
  stop?.()
  stop = null
})
</script>

<template>
  <Modal :open="open" @update:open="(v) => !v && close()" title="推送镜像" width="max-w-xl">
    <div class="space-y-4">
      <div>
        <label class="mb-1.5 block text-sm text-slate-500">推送目标（registry/仓库:标签）</label>
        <input
          v-model="pushRef"
          class="input font-mono"
          placeholder="myrepo/myapp:v1"
          :disabled="pushing"
          @keyup.enter="submit"
        />
      </div>

      <!-- 进度条 -->
      <div v-if="pushing || done" class="space-y-2">
        <div class="flex items-center justify-between text-xs text-slate-500">
          <span>{{ done ? '完成' : '推送中...' }}</span>
          <span v-if="overallProgress >= 0 && !done">{{ Math.round(overallProgress * 100) }}%</span>
        </div>
        <ProgressBar
          :value="done ? 1 : overallProgress"
          :indeterminate="overallProgress < 0 && !done"
        />
      </div>

      <!-- 层级进度列表（按层聚合，对应 docker push 输出） -->
      <div
        v-if="Object.keys(layerProgress).length > 0"
        class="max-h-40 overflow-y-auto rounded-md border border-slate-100 p-2 dark:border-slate-700"
      >
        <div
          v-for="(prog, id) in layerProgress"
          :key="id"
          class="flex items-center gap-2 truncate py-0.5 font-mono text-xs text-slate-500"
        >
          <span
          >{{ id }}:</span>
          <span class="truncate">{{ prog }}</span>
        </div>
      </div>

      <div v-if="done" class="text-sm text-green-600">✅ 推送完成</div>
    </div>
    <template #footer>
      <Button variant="secondary" :disabled="pushing" @click="close">取消</Button>
      <Button :loading="pushing" :disabled="done" @click="submit">{{ done ? '完成' : '推送' }}</Button>
    </template>
  </Modal>
</template>
