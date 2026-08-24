<script setup lang="ts">
import Modal from '@/components/ui/Modal.vue'
import Terminal from '@/components/ui/Terminal.vue'

const props = defineProps<{
  open: boolean
  containerId: string
  containerName?: string
}>()

const emit = defineEmits<{ 'update:open': [value: boolean] }>()

const wsUrl = `/api/v1/containers/${props.containerId}/terminal`
</script>

<template>
  <Modal
    :open="open"
    @update:open="(v) => emit('update:open', v)"
    :title="`终端 - ${containerName || containerId.slice(0, 12)}`"
    width="max-w-4xl"
    :close-on-backdrop="false"
  >
    <Terminal
      :url="wsUrl"
      :title="containerName || containerId.slice(0, 12)"
      height="h-[60vh]"
    />
  </Modal>
</template>
