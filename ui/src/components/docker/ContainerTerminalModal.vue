<script setup lang="ts">
import { computed } from 'vue'
import Modal from '@/components/ui/Modal.vue'
import Terminal from '@/components/ui/Terminal.vue'

const props = defineProps<{
  open: boolean
  containerId: string
  containerName?: string
}>()

const emit = defineEmits<{ 'update:open': [value: boolean] }>()

/** 响应式构建 ws 地址，containerId 变化时自动更新。 */
const wsUrl = computed(() => `/api/v1/containers/${props.containerId}/terminal`)
</script>

<template>
  <Modal
    :open="open"
    @update:open="(v) => emit('update:open', v)"
    :title="`终端 - ${containerName || containerId.slice(0, 12)}`"
    width="max-w-4xl"
    :close-on-backdrop="false"
    disable-focus-trap
  >
    <Terminal
      :url="wsUrl"
      :title="containerName || containerId.slice(0, 12)"
      height="h-[60vh]"
    />
  </Modal>
</template>
