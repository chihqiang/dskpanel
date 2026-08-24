/**
 * 后端统一响应结构（对应 httpx.Response[T]）。
 */
export interface ApiResponse<T = unknown> {
  code: number
  msg: string
  data: T
}

/** 请求方法类型。 */
export type HttpMethod = 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH'

/** 请求选项。 */
export interface RequestOptions {
  method?: HttpMethod
  /** JSON body（自动序列化）。 */
  body?: unknown
  /** 是否携带 token（默认 true）。 */
  auth?: boolean
  /** 自定义 headers。 */
  headers?: Record<string, string>
  /** 超时时间 ms（默认 30000）。 */
  timeout?: number
  /** 是否原样返回 response（用于下载等，默认 false）。 */
  raw?: boolean
}

const TOKEN_KEY = 'dskpanel_token'

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token)
}

export function clearToken(): void {
  localStorage.removeItem(TOKEN_KEY)
}

/**
 * 统一请求封装。
 * - 自动附加 Bearer token
 * - 解析后端 {code, msg, data} 响应
 * - code !== 0 抛出 ApiError
 * - 401 时清理 token 并跳转登录页
 */
export async function request<T = unknown>(
  url: string,
  options: RequestOptions = {},
): Promise<T> {
  const {
    method = 'GET',
    body,
    auth = true,
    headers = {},
    timeout = 30000,
    raw = false,
  } = options

  const controller = new AbortController()
  const timer = setTimeout(() => controller.abort(), timeout)

  const finalHeaders: Record<string, string> = {
    'Content-Type': 'application/json',
    ...headers,
  }
  if (auth) {
    const token = getToken()
    if (token) {
      finalHeaders.Authorization = `Bearer ${token}`
    }
  }

  try {
    const resp = await fetch(url, {
      method,
      headers: finalHeaders,
      body: body !== undefined ? JSON.stringify(body) : undefined,
      signal: controller.signal,
    })

    if (raw) {
      return resp as unknown as T
    }

    // 尝试解析 JSON；失败则按错误处理。
    let payload: ApiResponse<T>
    try {
      payload = (await resp.json()) as ApiResponse<T>
    } catch {
      throw new ApiError(resp.status, `unexpected response: ${resp.statusText}`)
    }

    if (resp.status === 401) {
      handleUnauthorized()
      throw new ApiError(401, 'unauthorized')
    }

    if (payload.code !== 0 || !resp.ok) {
      throw new ApiError(payload.code, payload.msg || `http ${resp.status}`)
    }
    return payload.data
  } catch (err) {
    if (err instanceof ApiError) {
      throw err
    }
    if (err instanceof DOMException && err.name === 'AbortError') {
      throw new ApiError(-1, 'request timeout')
    }
    throw new ApiError(-1, (err as Error).message)
  } finally {
    clearTimeout(timer)
  }
}

/** 业务错误。 */
export class ApiError extends Error {
  code: number
  constructor(code: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.code = code
  }
}

/** 401 处理：清理 token 并跳转登录页。 */
function handleUnauthorized(): void {
  clearToken()
  if (window.location.pathname !== '/login') {
    window.location.href = '/login'
  }
}

/** 快捷方法。 */
export const http = {
  get: <T = unknown>(url: string, options?: RequestOptions) =>
    request<T>(url, { ...options, method: 'GET' }),
  post: <T = unknown>(url: string, body?: unknown, options?: RequestOptions) =>
    request<T>(url, { ...options, method: 'POST', body }),
  put: <T = unknown>(url: string, body?: unknown, options?: RequestOptions) =>
    request<T>(url, { ...options, method: 'PUT', body }),
  delete: <T = unknown>(url: string, options?: RequestOptions) =>
    request<T>(url, { ...options, method: 'DELETE' }),
}
