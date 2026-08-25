<script setup lang="ts">
import { ref, watch } from 'vue'
import Button from '@/components/ui/Button.vue'
import Modal from '@/components/ui/Modal.vue'
import { useToast } from '@/composables/useToast'
import { swarmScaleService, type SwarmServiceItem } from '@/api/swarm'

const props = defineProps<{
  open: boolean
  service: SwarmServiceItem | null
}>()

const emit = defineEmits<{ 'update:open': [value: boolean]; scaled: [] }>()

const toast = useToast()

const scaleValue = ref(1)
const saving = ref(false)

watch(
  () => [props.open, props.service] as const,
  ([open, service]) => {
    if (open && service) {
      scaleValue.value = service.mode === 'global' ? 1 : Number(service.replicas.split('/')[0]) || 1
    }
  },
  { immediate: true },
)

async function doScale(): Promise<void> {
  if (!props.service) return
  saving.value = true
  try {
    await swarmScaleService(props.service.id, scaleValue.value)
    toast.success(`服务「${props.service.name}」已伸缩为 ${scaleValue.value} 副本`)
    emit('update:open', false)
    emit('scaled')
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
    title="服务伸缩"
    width="max-w-sm"
  >
    <p v-if="service" class="mb-4 text-sm text-slate-500">
      调整服务「{{ service.name }}」的副本数：
    </p>
    <input v-model.number="scaleValue" type="number" min="0" class="input w-full" />
    <template #footer>
      <Button variant="secondary" @click="emit('update:open', false)">取消</Button>
      <Button :loading="saving" @click="doScale">应用</Button>
    </template>
  </Modal>
</template>
