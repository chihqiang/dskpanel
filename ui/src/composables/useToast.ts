import { useToastStore } from '@/stores/toast'

/**
 * 全局消息弹出层（pinia store 封装）。
 * 用法：
 *   import { useToast } from '@/composables/useToast'
 *   const toast = useToast()
 *   toast.success('已删除')
 *   toast.error('操作失败')
 *   toast.info('处理中')
 */
export function useToast() {
  return useToastStore()
}
