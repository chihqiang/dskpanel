import { fileURLToPath, URL } from 'node:url'
import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig(({ mode }) => {
  // 读取 .env 环境变量（VITE_API_BASE：后端 API 基础地址，HTTP 代理 + WS 直连共用）。
  const env = loadEnv(mode, process.cwd(), '')
  const apiBase = env.VITE_API_BASE || 'http://127.0.0.1:8080'
  // WebSocket 地址由 API 地址派生：http(s):// → ws(s)://。
  const wsBase = apiBase.replace(/^http/, 'ws')

  return {
    plugins: [vue(), tailwindcss()],
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
      },
    },
    // 注入 WebSocket 后端基础地址（开发直连后端，绕开 Vite ws 代理兼容问题；生产为空串 = 同源）。
    define: {
      __WS_BASE__: JSON.stringify(wsBase),
    },
    server: {
      host: '0.0.0.0',
      port: 5173,
      proxy: {
        // 开发环境代理后端 HTTP API（不含 ws，ws 由前端直连）。
        '/api': {
          target: apiBase,
          changeOrigin: true,
        },
      },
    },
  }
})
