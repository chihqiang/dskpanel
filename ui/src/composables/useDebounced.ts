import { ref, watch, type Ref } from 'vue'

/**
 * debounced ref：源 ref 变化后延迟 ms 才同步到返回的 ref。
 * 用于搜索框的即时过滤，避免大量数据时每次按键都重算。
 */
export function useDebounced<T>(source: Ref<T>, ms = 200): Ref<T> {
  const debounced = ref(source.value) as Ref<T>
  let timer: ReturnType<typeof setTimeout> | null = null

  watch(source, (val) => {
    if (timer) clearTimeout(timer)
    timer = setTimeout(() => {
      debounced.value = val
    }, ms)
  })

  return debounced
}
