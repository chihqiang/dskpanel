<script setup lang="ts">
import { ref, watch } from 'vue'
import Button from '@/components/ui/Button.vue'
import Modal from '@/components/ui/Modal.vue'
import { useToast } from '@/composables/useToast'
import { renameContainer } from '@/api/container'

const props = defineProps<{
  open: boolean
  containerId: string
  currentName: string
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  renamed: []
}>()

const name = ref('')
const submitting = ref(false)
const toast = useToast()

watch(
  () => props.open,
  (open) => {
    if (open) {
      name.value = props.currentName
    }
  },
  { immediate: true },
)

async function submit(): Promise<void> {
  const newName = name.value.trim().replace(/^\//, '')
  if (!newName) {
    toast.error('请输入容器名称')
    return
  }
  if (newName === props.currentName.replace(/^\//, '')) {
    emit('update:open', false)
    return
  }
  submitting.value = true
  try {
    await renameContainer(props.containerId, newName)
    toast.success(`已重命名为「${newName}」`)
    emit('update:open', false)
    emit('renamed')
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <Modal :open="open" @update:open="(v) => emit('update:open', v)" title="重命名容器" width="max-w-md">
    <div class="space-y-3">
      <div>
        <label class="mb-1.5 block text-sm text-slate-500">容器名称</label>
        <input v-model="name" class="input font-mono" placeholder="my-container" @keyup.enter="submit" />
      </div>
    </div>
    <template #footer>
      <Button variant="secondary" @click="emit('update:open', false)">取消</Button>
      <Button :loading="submitting" @click="submit">保存</Button>
    </template>
  </Modal>
</template>
