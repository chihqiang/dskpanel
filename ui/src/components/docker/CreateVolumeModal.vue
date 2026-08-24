<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import Button from '@/components/ui/Button.vue'
import Modal from '@/components/ui/Modal.vue'
import { useToast } from '@/composables/useToast'
import { createVolume } from '@/api/volume'

const props = defineProps<{ open: boolean }>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  created: []
}>()

const submitting = ref(false)
const toast = useToast()
const form = reactive({ name: '', driver: 'local' })

watch(
  () => props.open,
  (open) => {
    if (open) {
      form.name = ''
      form.driver = 'local'
    }
  },
)

async function submit(): Promise<void> {
  if (!form.name) {
    toast.error('请输入卷名称')
    return
  }
  submitting.value = true
  try {
    await createVolume(form.name, form.driver)
    toast.success(`已创建卷「${form.name}」`)
    emit('update:open', false)
    emit('created')
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <Modal :open="open" @update:open="(v) => emit('update:open', v)" title="创建卷" width="max-w-md">
    <div class="space-y-3">
      <div>
        <label class="mb-1.5 block text-sm text-slate-500">名称</label>
        <input v-model="form.name" class="input" placeholder="my-volume" />
      </div>
      <div>
        <label class="mb-1.5 block text-sm text-slate-500">驱动</label>
        <input v-model="form.driver" class="input" placeholder="local" />
      </div>
    </div>
    <template #footer>
      <Button variant="secondary" @click="emit('update:open', false)">取消</Button>
      <Button :loading="submitting" @click="submit">创建</Button>
    </template>
  </Modal>
</template>
