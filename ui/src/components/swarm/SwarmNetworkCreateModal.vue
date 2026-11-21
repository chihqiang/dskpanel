<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import Button from '@/components/ui/Button.vue'
import Modal from '@/components/ui/Modal.vue'
import { useToast } from '@/composables/useToast'
import { swarmCreateNetwork } from '@/api/swarm'

const props = defineProps<{
  open: boolean
  kind: 'overlay' | 'bridge'
}>()

const emit = defineEmits<{ 'update:open': [value: boolean]; created: [] }>()

const toast = useToast()

const form = reactive({
  name: '',
  subnet: '',
  gateway: '',
  attachable: true,
  internal: false,
})
const saving = ref(false)

watch(
  () => props.open,
  (open) => {
    if (open) {
      form.name = ''
      form.subnet = ''
      form.gateway = ''
      form.attachable = true
      form.internal = false
    }
  },
  { immediate: true },
)

const createKind = ref<'overlay' | 'bridge'>('overlay')
watch(
  () => props.kind,
  (k) => { createKind.value = k },
  { immediate: true },
)

async function submit(): Promise<void> {
  if (!form.name) {
    toast.error('请输入网络名称')
    return
  }
  saving.value = true
  try {
    await swarmCreateNetwork({
      name: form.name,
      driver: props.kind,
      subnet: form.subnet || undefined,
      gateway: form.gateway || undefined,
      attachable: form.attachable,
      internal: form.internal,
    })
    toast.success(`网络「${form.name}」已创建`)
    emit('update:open', false)
    emit('created')
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <Modal :open="open" @update:open="emit('update:open', $event)" title="新建 Swarm 网络" width="max-w-lg">
    <div class="space-y-4">
      <div>
        <label class="mb-1.5 block text-sm text-slate-500">类型</label>
        <select v-model="createKind" class="input">
          <option value="overlay">overlay（Swarm 专用）</option>
          <option value="bridge">bridge（本地）</option>
        </select>
      </div>
      <div>
        <label class="mb-1.5 block text-sm text-slate-500">名称 *</label>
        <input v-model="form.name" class="input font-mono" placeholder="例如 app-net" />
      </div>
      <div class="grid grid-cols-2 gap-4">
        <div>
          <label class="mb-1.5 block text-sm text-slate-500">子网（可选）</label>
          <input v-model="form.subnet" class="input font-mono" placeholder="例如 10.0.1.0/24" />
        </div>
        <div>
          <label class="mb-1.5 block text-sm text-slate-500">网关（可选）</label>
          <input v-model="form.gateway" class="input font-mono" placeholder="例如 10.0.1.1" />
        </div>
      </div>
      <div class="flex gap-6">
        <label class="flex cursor-pointer items-center gap-2 text-sm text-slate-600 dark:text-slate-300">
          <input v-model="form.attachable" type="checkbox" class="h-4 w-4 rounded border-slate-300 text-blue-600" />
          允许独立容器连接
        </label>
        <label class="flex cursor-pointer items-center gap-2 text-sm text-slate-600 dark:text-slate-300">
          <input v-model="form.internal" type="checkbox" class="h-4 w-4 rounded border-slate-300 text-blue-600" />
          内部网络（禁止外部访问）
        </label>
      </div>
    </div>
    <template #footer>
      <Button variant="secondary" @click="emit('update:open', false)">取消</Button>
      <Button :loading="saving" @click="submit">创建</Button>
    </template>
  </Modal>
</template>
