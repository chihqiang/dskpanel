import { useConfirm } from '@/composables/useConfirm'
import { useToast } from '@/composables/useToast'
import { useActivity } from '@/composables/useActivity'

/**
 * 可撤销操作 composable。
 *
 * 封装"确认 → 延时执行 → 可撤销"的通用模式，
 * 统一处理 confirm 弹窗、toast undo 按钮、活动日志记录。
 *
 * 用法：
 *   const { undoableAction } = useUndoableAction()
 *   undoableAction({
 *     title: '删除容器',
 *     message: '确认删除容器「nginx」？5 秒内可撤销。',
 *     label: 'nginx',
 *     action: () => removeContainer(id, true, true),
 *     onDone: () => load(),
 *   })
 */

export interface UndoableActionOptions {
  /** 确认弹窗标题。 */
  title: string
  /** 确认弹窗消息。 */
  message: string
  /** 操作对象名称（用于 toast/activity 文案）。 */
  label: string
  /** 实际执行的操作（延时后调用）。 */
  action: () => Promise<unknown>
  /** 操作完成后回调（无论成功失败，如刷新列表）。 */
  onDone?: () => void | Promise<void>
  /** 撤销延时（ms），默认 5000。 */
  delay?: number
  /** 活动日志的 detail 字段（可选）。 */
  activityDetail?: string
  /** 操作类型文案，如"删除容器"、"删除镜像"。用于 activity 日志。 */
  actionLabel: string
}

export function useUndoableAction() {
  const confirm = useConfirm()
  const toast = useToast()
  const activity = useActivity()

  function undoableAction(opts: UndoableActionOptions): void {
    const {
      title,
      message,
      label,
      action,
      onDone,
      delay = 5000,
      activityDetail,
      actionLabel,
    } = opts

    void confirm(
      title,
      message,
      () => {
        let cancelled = false
        const timer = window.setTimeout(async () => {
          if (cancelled) return
          try {
            await action()
            toast.success(`已${actionLabel}「${label}」`)
            activity.success(`已${actionLabel}「${label}」`, activityDetail ?? label)
          } catch (err) {
            toast.error((err as Error).message)
            activity.error(`${actionLabel}失败：${(err as Error).message}`, label)
          } finally {
            await onDone?.()
          }
        }, delay)

        toast.pushUndo(
          `将在 ${delay / 1000} 秒后${actionLabel}「${label}」`,
          () => {
            cancelled = true
            window.clearTimeout(timer)
            toast.info('已撤销操作')
            activity.info(`已撤销${actionLabel}「${label}」`, label)
          },
          '撤销',
        )
      },
      { danger: true },
    )
  }

  return { undoableAction }
}
