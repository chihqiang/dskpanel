<script setup lang="ts">
/**
 * 通用终端弹窗：Docker 容器终端、K8s Pod 终端等共用。
 *
 * 调用方通过 props 传入 wsUrl 和 title 即可，
 * 组件负责 Modal 外壳 + xterm 终端渲染。
 */
import Modal from '@/components/ui/Modal.vue'
import Terminal from '@/components/ui/Terminal.vue'

const props = withDefaults(
  defineProps<{
    /** 弹窗 open 状态（v-model:open）。 */
    open: boolean
    /** WebSocket 地址（不含 token query，由终端组件自动附加鉴权）。 */
    url: string
    /** 弹窗标题。 */
    title: string
    /** 弹窗宽度。 */
    width?: string
    /** 终端高度。 */
    height?: string
  }>(),
  {
    width: 'max-w-4xl',
    height: 'h-[60vh]',
  },
)

const emit = defineEmits<{ 'update:open': [value: boolean] }>()
</script>

<template>
  <Modal
    :open="open"
    @update:open="(v) => emit('update:open', v)"
    :title="title"
    :width="width"
    :close-on-backdrop="false"
    disable-focus-trap
  >
    <Terminal
      :url="url"
      :title="title"
      :height="height"
    />
  </Modal>
</template>
