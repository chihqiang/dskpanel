import { http } from './http'

export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResult {
  token: string
  username: string
  expire_at: number
}

/** 登录。 */
export function login(data: LoginRequest) {
  return http.post<LoginResult>('/api/v1/auth/login', data, { auth: false })
}

/** 健康检查。 */
export function ping() {
  return http.get<string>('/api/v1/ping', { auth: false })
}
