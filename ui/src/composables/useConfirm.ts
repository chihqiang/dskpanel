import { createVNode, reactive, render } from 'vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import { useToast } from '@/composables/useToast'

export interface ConfirmOptions {
  title: string
  message: string
  confirmText?: string
  cancelText?: string
  /** 危险操作（红色确认按钮）。 */
  danger?: boolean
  /** 确认后执行的回调（执行期间确认按钮转圈），出错时自动 toast 并关闭。 */
  onConfirm?: () => unknown
  /** onConfirm 执行成功后的回调（如刷新列表）。 */
  onSuccess?: () => void
}

export type ConfirmCall = {
  (options: ConfirmOptions): Promise<boolean>
  (title: string, message: string, onConfirm?: ConfirmOptions['onConfirm'], options?: Omit<ConfirmOptions, 'title' | 'message' | 'onConfirm'>): Promise<boolean>
}

/**
 * 命令式二次确认弹窗 hook。
 * resolve(true) = 用户已确认（含 onConfirm 执行完成）；resolve(false) = 用户取消。
 *
 * 用法（二选一）：
 *   const confirm = useConfirm()
 *   // 对象式
 *   const ok = await confirm({ title: '删除', message: '确定？', danger: true, onConfirm, onSuccess })
 *   // 扁平式
 *   const ok = await confirm('删除', '确定？', onConfirm, { danger: true, onSuccess })
 */
export function useConfirm(): ConfirmCall {
  const toast = useToast()

  function call(
    optionsOrTitle: ConfirmOptions | string,
    message?: string,
    onConfirm?: ConfirmOptions['onConfirm'],
    extra?: Omit<ConfirmOptions, 'title' | 'message' | 'onConfirm'>,
  ): Promise<boolean> {
    const options: ConfirmOptions =
      typeof optionsOrTitle === 'string'
        ? { title: optionsOrTitle, message: message ?? '', onConfirm, ...extra }
        : optionsOrTitle

    return new Promise<boolean>((resolve) => {
      const container = document.createElement('div')
      let settled = false

      const props = reactive({
        open: true,
        title: options.title,
        message: options.message,
        confirmText: options.confirmText,
        cancelText: options.cancelText,
        danger: options.danger,
        loading: false,
        onConfirm: async () => {
          if (settled) return
          if (options.onConfirm) {
            props.loading = true
            try {
              await options.onConfirm()
            } catch (err) {
              toast.error((err as Error).message)
              close(false)
              return
            }
          }
          close(true)
          options.onSuccess?.()
        },
        onCancel: () => close(false),
      })

      function close(result: boolean): void {
        if (settled) return
        settled = true
        props.open = false
        // 等退出过渡动画播放完再卸载并 resolve。
        window.setTimeout(() => {
          render(null, container)
          container.remove()
          resolve(result)
        }, 150)
      }

      render(createVNode(ConfirmDialog, props), container)
      document.body.appendChild(container)
    })
  }

  return call as ConfirmCall
}
