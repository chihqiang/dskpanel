<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import Button from '@/components/ui/Button.vue'
import Modal from '@/components/ui/Modal.vue'
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
const progressLines = ref<string[]>([])
let stop: (() => void) | null = null

watch(
  () => props.open,
  (open) => {
    if (open) {
      done.value = false
      progressLines.value = []
      // 预填源标签，用户可改成目标地址。
      pushRef.value = props.sourceTag || ''
    }
  },
)

function submit(): void {
  if (!pushRef.value) return
  pushing.value = true
  done.value = false
  progressLines.value = []

  stop = pushImageStream(
    pushRef.value,
    (msg) => {
      // 解析 Docker push 进度消息。
      const status = (msg.status as string) ?? ''
      const progress = (msg.progress as string) ?? ''
      const id = (msg.id as string) ?? ''
      let line = status
      if (id) line = `${id}: ${line}`
      if (progress) line += ` ${progress}`
      if (line) progressLines.value.push(line)
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

      <!-- 进度区 -->
      <div
        v-if="pushing || done || progressLines.length > 0"
        class="max-h-56 overflow-y-auto rounded-md bg-slate-900 p-3 font-mono text-xs text-slate-100"
      >
        <template v-if="progressLines.length > 0">
          <div v-for="(line, idx) in progressLines" :key="idx" class="whitespace-pre-wrap break-all">{{ line }}</div>
        </template>
        <div v-else-if="pushing" class="text-slate-500">推送中...</div>
      </div>

      <div v-if="done" class="text-sm text-green-600">✅ 推送完成</div>
    </div>
    <template #footer>
      <Button variant="secondary" :disabled="pushing" @click="close">取消</Button>
      <Button :loading="pushing" :disabled="done" @click="submit">{{ done ? '完成' : '推送' }}</Button>
    </template>
  </Modal>
</template>
