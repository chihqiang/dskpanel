<script setup lang="ts">
import { ref, watch } from 'vue'
import Button from '@/components/ui/Button.vue'
import Modal from '@/components/ui/Modal.vue'
import { useToast } from '@/composables/useToast'
import { k8sScaleDeployment, k8sScaleStatefulSet } from '@/api/k8s'

const props = withDefaults(
  defineProps<{
    open: boolean
    kind?: 'deployment' | 'statefulset'
    name: string
    namespace: string
    initial?: number
  }>(),
  { kind: 'deployment', initial: 1 },
)

const emit = defineEmits<{ 'update:open': [value: boolean]; scaled: [] }>()

const toast = useToast()

const scaleValue = ref(1)
const saving = ref(false)

watch(
  () => [props.open, props.initial] as const,
  ([open, initial]) => {
    if (open) {
      scaleValue.value = initial || 1
    }
  },
  { immediate: true },
)

async function doScale(): Promise<void> {
  saving.value = true
  try {
    if (props.kind === 'statefulset') {
      await k8sScaleStatefulSet(props.name, props.namespace, scaleValue.value)
    } else {
      await k8sScaleDeployment(props.name, props.namespace, scaleValue.value)
    }
    toast.success(`${props.kind === 'statefulset' ? 'StatefulSet' : 'Deployment'}「${props.name}」已伸缩为 ${scaleValue.value} 副本`)
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
  <Modal :open="open" @update:open="emit('update:open', $event)" title="伸缩副本" width="max-w-sm">
    <p class="mb-4 text-sm text-slate-500">
      调整{{ kind === 'statefulset' ? 'StatefulSet' : 'Deployment' }}「{{ name }}」（命名空间 {{ namespace || 'default' }}）的副本数：
    </p>
    <input v-model.number="scaleValue" type="number" min="0" class="input w-full" />
    <template #footer>
      <Button variant="secondary" @click="emit('update:open', false)">取消</Button>
      <Button :loading="saving" @click="doScale">应用</Button>
    </template>
  </Modal>
</template>
