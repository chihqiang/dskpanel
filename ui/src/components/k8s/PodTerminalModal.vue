<script setup lang="ts">
import { computed } from 'vue'
import Modal from '@/components/ui/Modal.vue'
import Terminal from '@/components/ui/Terminal.vue'
import type { K8sPodItem } from '@/api/k8s'

const props = defineProps<{
  open: boolean
  pod: K8sPodItem | null
  container?: string
}>()

const emit = defineEmits<{ 'update:open': [value: boolean] }>()

/** 构建 ws 地址（含 namespace 和 container query）。 */
const wsUrl = computed(() => {
  if (!props.pod) return ''
  const params = new URLSearchParams({
    namespace: props.pod.namespace,
  })
  const c = props.container || props.pod.containers?.[0]?.name
  if (c) params.set('container', c)
  return `/api/v1/k8s/pods/${props.pod.name}/terminal?${params}`
})

const title = computed(() => {
  const c = props.container || props.pod?.containers?.[0]?.name || ''
  return `终端 - ${props.pod?.name ?? ''}${c ? ` / ${c}` : ''}`
})
</script>

<template>
  <Modal
    :open="open"
    @update:open="(v) => emit('update:open', v)"
    :title="title"
    width="max-w-4xl"
    :close-on-backdrop="false"
    disable-focus-trap
  >
    <Terminal
      :url="wsUrl"
      :title="title"
      height="h-[60vh]"
    />
  </Modal>
</template>
