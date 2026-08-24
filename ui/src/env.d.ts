/// <reference types="vite/client" />

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<Record<string, unknown>, Record<string, unknown>, unknown>
  export default component
}

/** WebSocket 后端基础地址（vite.config define 注入）：开发为 ws://127.0.0.1:8080，生产为空串=同源。 */
declare const __WS_BASE__: string
