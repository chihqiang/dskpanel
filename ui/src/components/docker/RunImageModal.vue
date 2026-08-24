<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import Button from '@/components/ui/Button.vue'
import Modal from '@/components/ui/Modal.vue'
import { useToast } from '@/composables/useToast'
import { createContainer, type PortMapping } from '@/api/container'

const props = defineProps<{ open: boolean; image: string }>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  started: []
}>()

const running = ref(false)
const toast = useToast()

const form = reactive({ name: '' })
const portRows = ref<{ container_port: number; host_port: number }[]>([])

watch(
  () => props.open,
  (open) => {
    if (open) {
      form.name = ''
      portRows.value = []
    }
  },
)

function addPortRow(): void {
  portRows.value.push({ container_port: 0, host_port: 0 })
}

function removePortRow(idx: number): void {
  portRows.value.splice(idx, 1)
}

async function submit(): Promise<void> {
  running.value = true
  try {
    const ports: PortMapping[] = portRows.value
      .filter((p) => p.container_port > 0)
      .map((p) => ({ container_port: p.container_port, host_port: p.host_port }))
    // detach=false → 创建后自动启动。
    await createContainer({
      image: props.image,
      name: form.name || undefined,
      ports,
      detach: false,
    })
    toast.success('容器已启动')
    emit('update:open', false)
    emit('started')
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    running.value = false
  }
}
</script>

<template>
  <Modal :open="open" @update:open="(v) => emit('update:open', v)" title="运行镜像" width="max-w-lg">
    <div class="space-y-5">
      <div>
        <label class="mb-1.5 block text-sm text-slate-500">镜像</label>
        <input :value="image" class="input bg-slate-50 dark:bg-slate-800" readonly />
      </div>
      <div>
        <label class="mb-1.5 block text-sm text-slate-500">容器名称（可选）</label>
        <input v-model="form.name" class="input" placeholder="my-container" />
      </div>
      <div>
        <div class="mb-1 flex items-center justify-between">
          <label class="text-sm text-slate-500">端口映射</label>
          <Button variant="ghost" size="sm" @click="addPortRow">+ 添加端口</Button>
        </div>
        <div v-for="(row, idx) in portRows" :key="idx" class="mb-2 flex items-center gap-3">
          <input v-model.number="row.host_port" class="input flex-1" placeholder="宿主机端口" />
          <span class="text-slate-400">→</span>
          <input v-model.number="row.container_port" class="input flex-1" placeholder="容器端口" />
          <Button variant="ghost" size="sm" class="!text-red-500" @click="removePortRow(idx)">移除</Button>
        </div>
      </div>
    </div>
    <template #footer>
      <Button variant="secondary" @click="emit('update:open', false)">取消</Button>
      <Button :loading="running" @click="submit">运行</Button>
    </template>
  </Modal>
</template>
