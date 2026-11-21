import { ref } from 'vue'
import { useToast } from '@/composables/useToast'

/**
 * 通用剪贴板复制 composable。
 *
 * 封装 navigator.clipboard.writeText + toast 提示 + 复制状态反馈。
 * 用法：
 *   const { copy, copied } = useClipboard()
 *   await copy('hello', '已复制 ID')
 *   // copied.value 在 1.5s 内为 true
 */

export function useClipboard() {
  const toast = useToast()
  const copied = ref(false)
  let timer: ReturnType<typeof setTimeout> | null = null

  /**
   * 复制文本到剪贴板。
   * @param text    要复制的文本
   * @param successMsg 成功提示文案，默认"已复制到剪贴板"
   * @param errorMsg   失败提示文案，默认"复制失败"
   * @returns 是否复制成功
   */
  async function copy(
    text: string,
    successMsg = '已复制到剪贴板',
    errorMsg = '复制失败',
  ): Promise<boolean> {
    try {
      await navigator.clipboard.writeText(text)
      copied.value = true
      toast.success(successMsg)
      if (timer) clearTimeout(timer)
      timer = setTimeout(() => (copied.value = false), 1500)
      return true
    } catch {
      toast.error(errorMsg)
      return false
    }
  }

  return { copy, copied }
}
