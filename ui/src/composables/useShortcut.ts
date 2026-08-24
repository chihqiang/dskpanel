import { onBeforeUnmount } from 'vue'

export interface ShortcutOptions {
  /** 是否阻止默认行为，默认 true。 */
  preventDefault?: boolean
  /** 是否在输入框内忽略（默认 true）。 */
  ignoreInputs?: boolean
  /** 附加的修饰键（如 meta/ctrl），满足时才触发。 */
  meta?: boolean
  ctrl?: boolean
  shift?: boolean
  alt?: boolean
}

const INPUT_TAGS = new Set(['INPUT', 'TEXTAREA', 'SELECT'])

function isTyping(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false
  if (INPUT_TAGS.has(target.tagName)) return true
  return target.isContentEditable
}

/**
 * 注册一个全局快捷键。
 *
 * @param key   单个按键（如 '/'、'n'、'r'、'Escape'），大小写不敏感（字母键）
 * @param handler 触发回调
 * @param options 选项
 */
export function useShortcut(
  key: string,
  handler: (e: KeyboardEvent) => void,
  options: ShortcutOptions = {},
): void {
  const {
    preventDefault = true,
    ignoreInputs = true,
    meta = false,
    ctrl = false,
    shift = false,
    alt = false,
  } = options

  const expected = key.toLowerCase()

  function onKeydown(e: KeyboardEvent): void {
    const k = e.key.toLowerCase()
    if (k !== expected) return

    if (ignoreInputs && key !== 'escape' && isTyping(e.target)) return

    if (meta && !e.metaKey) return
    if (ctrl && !e.ctrlKey) return
    if (shift && !e.shiftKey) return
    if (alt && !e.altKey) return

    if (preventDefault) e.preventDefault()
    handler(e)
  }

  window.addEventListener('keydown', onKeydown)
  onBeforeUnmount(() => {
    window.removeEventListener('keydown', onKeydown)
  })
}
