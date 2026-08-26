<script setup lang="ts">
/**
 * 通用重命名弹窗：Docker 容器重命名等场景共用。
 *
 * 调用方通过 props 传入标题、当前名称和提交函数，
 * 组件负责输入框 + 验证 + 提交 + Toast 反馈。
 */
import { ref, watch } from 'vue'
import Button from '@/components/ui/Button.vue'
import Modal from '@/components/ui/Modal.vue'
import { useToast } from '@/composables/useToast'

const props = withDefaults(
  defineProps<{
    /** 弹窗 open 状态（v-model:open）。 */
    open: boolean
    /** 弹窗标题。 */
    title?: string
    /** 输入框 label。 */
    label?: string
    /** 当前名称（打开时预填）。 */
    currentName: string
    /** 输入框 placeholder。 */
    placeholder?: string
    /**
     * 提交函数：接收新名称，成功后弹窗关闭并 emit renamed。
     */
    submit: (newName: string) => Promise<void>
  }>(),
  {
    title: '重命名',
    label: '名称',
    placeholder: '',
  },
)

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

async function submit_(): Promise<void> {
  const newName = name.value.trim().replace(/^\//, '')
  if (!newName) {
    toast.error('请输入名称')
    return
  }
  if (newName === props.currentName.replace(/^\//, '')) {
    emit('update:open', false)
    return
  }
  submitting.value = true
  try {
    await props.submit(newName)
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
  <Modal :open="open" @update:open="(v) => emit('update:open', v)" :title="title" width="max-w-md">
    <div class="space-y-3">
      <div>
        <label class="mb-1.5 block text-sm text-slate-500">{{ label }}</label>
        <input v-model="name" class="input font-mono" :placeholder="placeholder" @keyup.enter="submit_" />
      </div>
    </div>
    <template #footer>
      <Button variant="secondary" @click="emit('update:open', false)">取消</Button>
      <Button :loading="submitting" @click="submit_">保存</Button>
    </template>
  </Modal>
</template>
