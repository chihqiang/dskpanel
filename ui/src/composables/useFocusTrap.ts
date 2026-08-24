import { onBeforeUnmount, watch, type Ref } from 'vue'

const FOCUSABLE = [
  'a[href]',
  'button:not([disabled])',
  'textarea',
  'input',
  'select',
  '[tabindex]:not([tabindex="-1"])',
].join(',')

/**
 * 焦点陷阱 composable：当 active 为 true 时，把 Tab 焦点限制在 container 内。
 * - 打开时自动聚焦容器内首个可聚焦元素
 * - 关闭时恢复打开前的焦点
 * - Tab/Shift+Tab 在容器边界时回绕
 *
 * 用法：
 *   const container = ref<HTMLElement | null>(null)
 *   useFocusTrap(container, () => props.open)
 */
export function useFocusTrap(
  container: Ref<HTMLElement | null>,
  active: () => boolean,
): void {
  let lastFocused: HTMLElement | null = null

  function getFocusable(): HTMLElement[] {
    if (!container.value) return []
    return Array.from(container.value.querySelectorAll<HTMLElement>(FOCUSABLE)).filter(
      (el) => el.offsetParent !== null || el === document.activeElement,
    )
  }

  function onKeydown(e: KeyboardEvent): void {
    if (e.key !== 'Tab') return
    const els = getFocusable()
    if (els.length === 0) {
      e.preventDefault()
      container.value?.focus()
      return
    }
    const first = els[0]
    const last = els[els.length - 1]
    const active = document.activeElement as HTMLElement
    if (e.shiftKey) {
      if (active === first || !container.value?.contains(active)) {
        e.preventDefault()
        last.focus()
      }
    } else {
      if (active === last || !container.value?.contains(active)) {
        e.preventDefault()
        first.focus()
      }
    }
  }

  watch(
    active,
    (open) => {
      if (open) {
        lastFocused = document.activeElement as HTMLElement
        document.addEventListener('keydown', onKeydown)
        // 下一帧聚焦（等 DOM 渲染）。
        requestAnimationFrame(() => {
          const els = getFocusable()
          if (els.length > 0) {
            els[0].focus()
          } else {
            container.value?.focus()
          }
        })
      } else {
        document.removeEventListener('keydown', onKeydown)
        if (lastFocused && typeof lastFocused.focus === 'function') {
          lastFocused.focus()
          lastFocused = null
        }
      }
    },
    { immediate: true },
  )

  onBeforeUnmount(() => {
    document.removeEventListener('keydown', onKeydown)
  })
}
