<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import Button from '@/components/ui/Button.vue'
import Modal from '@/components/ui/Modal.vue'
import { useToast } from '@/composables/useToast'
import {
  swarmCreateSecret,
  swarmCreateConfig,
} from '@/api/swarm'

const props = defineProps<{
  open: boolean
  kind: 'secret' | 'config'
}>()

const emit = defineEmits<{ 'update:open': [value: boolean]; saved: [] }>()

const toast = useToast()

const form = reactive({ name: '', data: '' })
const saving = ref(false)

/** 弹窗打开时重置表单（通过 watch + key 触发）。 */
watch(
  () => [props.open, props.kind] as const,
  ([open]) => {
    if (open) {
      form.name = ''
      form.data = ''
    }
  },
  { immediate: true },
)

async function submit(): Promise<void> {
  if (!form.name) {
    toast.error('请输入名称')
    return
  }
  saving.value = true
  try {
    if (props.kind === 'secret') {
      await swarmCreateSecret(form.name, form.data)
      toast.success(`Secret「${form.name}」已创建`)
    } else {
      await swarmCreateConfig(form.name, form.data)
      toast.success(`Config「${form.name}」已创建`)
    }
    emit('update:open', false)
    emit('saved')
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <Modal
    :open="open"
    @update:open="emit('update:open', $event)"
    :title="kind === 'secret' ? '新建 Secret' : '新建 Config'"
    width="max-w-lg"
  >
    <div class="space-y-4">
      <div>
        <label class="mb-1.5 block text-sm text-slate-500">名称 *</label>
        <input v-model="form.name" class="input font-mono" placeholder="例如 db-pass" />
      </div>
      <div>
        <label class="mb-1.5 block text-sm text-slate-500">
          {{ kind === 'secret' ? '数据（敏感内容）' : '数据（配置文件内容）' }}
        </label>
        <textarea v-model="form.data" class="input h-32 font-mono" placeholder="输入内容…" />
      </div>
    </div>
    <template #footer>
      <Button variant="secondary" @click="emit('update:open', false)">取消</Button>
      <Button :loading="saving" @click="submit">创建</Button>
    </template>
  </Modal>
</template>
