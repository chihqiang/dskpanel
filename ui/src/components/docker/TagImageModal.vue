<script setup lang="ts">
import { ref, watch } from 'vue'
import Button from '@/components/ui/Button.vue'
import Modal from '@/components/ui/Modal.vue'
import { useToast } from '@/composables/useToast'
import { tagImage } from '@/api/image'

const props = defineProps<{
  open: boolean
  /** 源镜像（repo_tag 或 id）。 */
  sourceImage: string
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  tagged: []
}>()

const target = ref('')
const submitting = ref(false)
const toast = useToast()

watch(
  () => props.open,
  (open) => {
    if (open) {
      target.value = ''
    }
  },
)

async function submit(): Promise<void> {
  if (!target.value) {
    toast.error('请输入目标标签')
    return
  }
  submitting.value = true
  try {
    await tagImage(props.sourceImage, target.value)
    toast.success(`已打标签「${target.value}」`)
    emit('update:open', false)
    emit('tagged')
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <Modal :open="open" @update:open="(v) => emit('update:open', v)" title="镜像打标签" width="max-w-md">
    <div class="space-y-3">
      <div>
        <label class="mb-1.5 block text-sm text-slate-500">源镜像</label>
        <input :value="sourceImage" class="input" disabled />
      </div>
      <div>
        <label class="mb-1.5 block text-sm text-slate-500">目标标签</label>
        <input v-model="target" class="input" placeholder="myrepo/myapp:v1" />
      </div>
    </div>
    <template #footer>
      <Button variant="secondary" @click="emit('update:open', false)">取消</Button>
      <Button :loading="submitting" @click="submit">保存</Button>
    </template>
  </Modal>
</template>
