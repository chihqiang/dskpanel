<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import Button from '@/components/ui/Button.vue'
import Modal from '@/components/ui/Modal.vue'
import ProgressBar from '@/components/ui/ProgressBar.vue'
import { useToast } from '@/composables/useToast'
import { useActivity } from '@/composables/useActivity'
import { pullImageStream } from '@/api/image'

const props = defineProps<{ open: boolean }>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  pulled: []
}>()

const pullRef = ref('')
const pulling = ref(false)
const toast = useToast()
const activity = useActivity()
const done = ref(false)
/** 按层 ID 聚合的最新进度文本。 */
const layerProgress = ref<Record<string, string>>({})
let stop: (() => void) | null = null

/** 整体进度估算：已完成层数 / 已知总层数。 */
const overallProgress = computed(() => {
  const entries = Object.values(layerProgress.value)
  if (entries.length === 0) return -1
  const completed = entries.filter((p) => p.includes('Pull complete') || p.includes('Already exists')).length
  return completed / entries.length
})

watch(
  () => props.open,
  (open) => {
    if (open) {
      done.value = false
      layerProgress.value = {}
    }
  },
)

function submit(): void {
  if (!pullRef.value) return
  pulling.value = true
  done.value = false
  layerProgress.value = {}

  stop = pullImageStream(
    pullRef.value,
    (msg) => {
      const status = (msg.status as string) ?? ''
      const progress = (msg.progress as string) ?? ''
      const id = (msg.id as string) ?? ''
      // 按 layer ID 聚合最新状态（仅保留一层，避免重复显示）。
      if (id) {
        layerProgress.value[id] = progress || status
      }
    },
    () => {
      done.value = true
      pulling.value = false
      toast.success(`拉取完成：${pullRef.value}`)
      activity.success(`拉取镜像「${pullRef.value}」`, pullRef.value)
      emit('pulled')
    },
    (msg) => {
      toast.error(msg)
      activity.error(`拉取镜像失败：${msg}`, pullRef.value)
      pulling.value = false
    },
  )
}

/** 取消拉取。 */
function cancelPull(): void {
  stop?.()
  stop = null
  pulling.value = false
  toast.info('已取消拉取')
  activity.info(`取消拉取镜像「${pullRef.value}」`, pullRef.value)
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
  <Modal :open="open" @update:open="(v) => !v && close()" title="拉取镜像" width="max-w-xl">
    <div class="space-y-4">
      <div>
        <label class="mb-1.5 block text-sm text-slate-500">镜像引用</label>
        <input
          v-model="pullRef"
          class="input"
          placeholder="nginx:latest"
          :disabled="pulling"
          @keyup.enter="submit"
        />
      </div>

      <!-- 进度条 -->
      <div v-if="pulling || done" class="space-y-2">
        <div class="flex items-center justify-between text-xs text-slate-500">
          <span>{{ done ? '完成' : '拉取中...' }}</span>
          <span v-if="overallProgress >= 0 && !done">{{ Math.round(overallProgress * 100) }}%</span>
        </div>
        <ProgressBar
          :value="done ? 1 : overallProgress"
          :indeterminate="overallProgress < 0 && !done"
        />
      </div>

      <!-- 层级进度列表（按层聚合，对应 docker pull 输出） -->
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

      <div v-if="done" class="flex items-center gap-2 text-sm text-green-600">
        <span class="h-4 w-4 rounded-full bg-green-100 text-center leading-4">✓</span>镜像已拉取
      </div>
    </div>
    <template #footer>
      <Button v-if="pulling" variant="danger" @click="cancelPull">取消拉取</Button>
      <Button v-else variant="secondary" :disabled="pulling" @click="close">关闭</Button>
      <Button v-if="!done" :loading="pulling" @click="submit">拉取</Button>
      <Button v-else variant="secondary" @click="close">完成</Button>
    </template>
  </Modal>
</template>
