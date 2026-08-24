<script setup lang="ts">
import { ref, watch } from 'vue'
import Button from '@/components/ui/Button.vue'
import Modal from '@/components/ui/Modal.vue'
import { useToast } from '@/composables/useToast'
import { commitContainer } from '@/api/container'

const props = defineProps<{
  open: boolean
  containerId: string
  containerName: string
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  committed: []
}>()

const reference = ref('')
const comment = ref('')
const author = ref('')
const submitting = ref(false)
const toast = useToast()

watch(
  () => props.open,
  (open) => {
    if (open) {
      reference.value = ''
      comment.value = ''
      author.value = ''
    }
  },
)

async function submit(): Promise<void> {
  submitting.value = true
  try {
    await commitContainer(props.containerId, {
      reference: reference.value.trim() || undefined,
      comment: comment.value.trim() || undefined,
      author: author.value.trim() || undefined,
    })
    toast.success('已提交为镜像')
    emit('update:open', false)
    emit('committed')
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <Modal :open="open" @update:open="(v) => emit('update:open', v)" title="提交容器为镜像" width="max-w-md">
    <div class="space-y-3">
      <p class="text-sm text-slate-500">
        将容器「{{ containerName }}」当前文件系统状态保存为一个新镜像（docker commit）。容器会先暂停以保证数据一致性。
      </p>
      <div>
        <label class="mb-1.5 block text-sm text-slate-500">
          镜像名称（如 <code class="font-mono text-xs">myimage:v1</code>）
        </label>
        <input v-model="reference" class="input font-mono" placeholder="myimage:v1（留空则沿用原镜像名）" />
      </div>
      <div>
        <label class="mb-1.5 block text-sm text-slate-500">提交信息</label>
        <input v-model="comment" class="input" placeholder="本次提交说明（可选）" />
      </div>
      <div>
        <label class="mb-1.5 block text-sm text-slate-500">作者</label>
        <input v-model="author" class="input" placeholder="作者（可选）" />
      </div>
    </div>
    <template #footer>
      <Button variant="secondary" @click="emit('update:open', false)">取消</Button>
      <Button :loading="submitting" @click="submit">提交</Button>
    </template>
  </Modal>
</template>
