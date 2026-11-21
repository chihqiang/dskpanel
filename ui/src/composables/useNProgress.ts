import NProgress from 'nprogress'
import 'nprogress/nprogress.css'

let configured = false

function ensureConfigured(): void {
  if (configured) return
  configured = true
  NProgress.configure({
    showSpinner: false,
    trickleSpeed: 150,
    minimum: 0.08,
  })
}

/**
 * 顶部全局进度条（nprogress 封装）。
 * 用于文件上传/下载、长任务等场景；configure 全局只执行一次。
 *
 * 用法：
 *   const nprogress = useNProgress()
 *   nprogress.start()
 *   nprogress.set(0.5)   // 0~1
 *   nprogress.done()     // 完成/失败都要调用，否则进度条残留
 */
export function useNProgress() {
  ensureConfigured()
  return {
    start: () => NProgress.start(),
    set: (p: number) => NProgress.set(p),
    done: () => NProgress.done(),
  }
}
