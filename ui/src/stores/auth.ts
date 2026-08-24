import { defineStore } from 'pinia'
import { login as apiLogin, type LoginRequest } from '@/api/auth'
import { clearToken, getToken, setToken } from '@/api/http'

/** 从 token 的 base64url payload 解码（JWT 载荷，非加密，仅展示用）。 */
function decodePayload(token: string): { username?: string; expire_at?: number } | null {
  try {
    const part = token.split('.')[0]
    if (!part) return null
    const normalized = part.replace(/-/g, '+').replace(/_/g, '/')
    const json = decodeURIComponent(
      atob(normalized)
        .split('')
        .map((c) => `%${c.charCodeAt(0).toString(16).padStart(2, '0')}`)
        .join(''),
    )
    return JSON.parse(json)
  } catch {
    return null
  }
}

/**
 * 用户状态 store（pinia）。
 * - token 存取仍走 http.ts 的 localStorage（与 401 拦截、API 请求共用一份）。
 * - 本 store 额外维护用户名 + 过期时间，登录/登出统一入口。
 */
export const useAuthStore = defineStore('auth', {
  state: () => ({
    username: '',
    expireAt: 0, // unix 秒
  }),
  getters: {
    /** 当前 token（无则 null）。 */
    token(): string | null {
      return getToken()
    },
    /** 是否已登录（有 token 且未过期）。 */
    isAuthenticated(): boolean {
      const token = getToken()
      if (!token) return false
      if (this.expireAt && Date.now() / 1000 > this.expireAt) return false
      return true
    },
  },
  actions: {
    /** 登录：调用后端 + 持久化 token 与用户信息。 */
    async login(req: LoginRequest) {
      const res = await apiLogin(req)
      setToken(res.token)
      this.username = res.username
      this.expireAt = res.expire_at
      return res
    },
    /** 登出：清 token 与用户状态。 */
    logout() {
      clearToken()
      this.username = ''
      this.expireAt = 0
    },
    /** 从已有 token 恢复用户状态（兼容旧 localStorage / 刷新页面）。 */
    initFromToken() {
      const token = getToken()
      if (!token) return
      const payload = decodePayload(token)
      this.username = payload?.username ?? ''
      this.expireAt = payload?.expire_at ?? 0
    },
  },
})
