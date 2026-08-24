import { getToken } from './http'

/** SSE 事件回调接口。 */
export interface SSEHandlers {
  /** 收到一条消息（event 非 error/done/open 时触发）。 */
  onMessage?: (data: string, event: string) => void
  /** 收到 done 事件（流正常结束）。 */
  onDone?: (data: string) => void
  /** 收到 error 事件或请求异常。 */
  onError?: (msg: string) => void
  /** 流关闭（无论正常/异常/取消），适合做 UI 状态清理。 */
  onClose?: () => void
}

/** SSE 请求选项。 */
export interface SSEOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE'
  body?: unknown
  /** 是否带 token（默认 true）。 */
  auth?: boolean
}

/**
 * 通用 SSE 流式请求。
 *
 * 封装 fetch + AbortController + ReadableStream + SSE 事件块解析，
 * 统一处理 event:/data: 行、error/done/open 事件、AbortError 等。
 *
 * @returns 取消函数（调用后 abort fetch）
 *
 * 用法：
 *   const cancel = createSSEStream('/api/v1/images/pull', {
 *     method: 'POST',
 *     body: { ref: 'nginx:latest' },
 *   }, {
 *     onMessage: (data) => console.log(data),
 *     onDone: () => console.log('done'),
 *     onError: (msg) => console.error(msg),
 *   })
 *   // 取消：cancel()
 */
export function createSSEStream(
  url: string,
  options: SSEHandlers & SSEOptions,
): () => void {
  const {
    onMessage,
    onDone,
    onError,
    onClose,
    method = 'GET',
    body,
    auth = true,
  } = options

  const controller = new AbortController()

  const headers: Record<string, string> = {}
  if (method !== 'GET') {
    headers['Content-Type'] = 'application/json'
  }
  if (auth) {
    const token = getToken()
    if (token) headers.Authorization = `Bearer ${token}`
  }

  ;(async () => {
    try {
      const resp = await fetch(url, {
        method,
        headers,
        body: body !== undefined ? JSON.stringify(body) : undefined,
        signal: controller.signal,
      })

      if (!resp.ok || !resp.body) {
        onError?.(`请求失败: HTTP ${resp.status}`)
        onClose?.()
        return
      }

      const reader = resp.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''
      let lastEvent = ''

      for (;;) {
        const { done, value } = await reader.read()
        if (done) break
        buffer += decoder.decode(value, { stream: true })

        // 按 SSE 规范，事件块以空行（\n\n）分隔。
        let sep: number
        while ((sep = buffer.indexOf('\n\n')) >= 0) {
          const chunk = buffer.slice(0, sep)
          buffer = buffer.slice(sep + 2)

          for (const raw of chunk.split('\n')) {
            if (raw.startsWith('event: ')) {
              lastEvent = raw.slice(7).trim()
            } else if (raw.startsWith('data: ')) {
              const data = raw.slice(6)

              if (lastEvent === 'error') {
                onError?.(data)
              } else if (lastEvent === 'done') {
                onDone?.(data)
              } else if (lastEvent !== 'open') {
                // open 为握手事件，不回调；其余均为消息。
                onMessage?.(data, lastEvent)
              }
              lastEvent = ''
            }
          }
        }
      }
      // 流自然结束，如果没有收到显式 done 事件，也触发 onDone。
      onClose?.()
    } catch (err) {
      if ((err as Error).name !== 'AbortError') {
        onError?.((err as Error).message)
      }
      onClose?.()
    }
  })()

  return () => controller.abort()
}

/**
 * 通用 SSE 流式请求（JSON 消息版）。
 *
 * 在 createSSEStream 基础上，自动将 onMessage 的 data 字符串 JSON.parse 为对象。
 * 解析失败的非 JSON 消息会被静默忽略。适用于 Docker pull/push 等 JSON 流式 API。
 *
 * 用法：
 *   const cancel = createSSEStreamJSON('/api/v1/images/pull', {
 *     method: 'POST',
 *     body: { ref: 'nginx:latest' },
 *     onMessage: (msg) => console.log(msg.id, msg.status),
 *     onDone: () => console.log('done'),
 *     onError: (msg) => console.error(msg),
 *   })
 */
export function createSSEStreamJSON<T = Record<string, unknown>>(
  url: string,
  options: Omit<SSEHandlers, 'onMessage'> & {
    onMessage?: (msg: T) => void
  } & SSEOptions,
): () => void {
  const { onMessage, ...rest } = options
  return createSSEStream(url, {
    ...rest,
    onMessage: (data: string) => {
      try {
        onMessage?.(JSON.parse(data) as T)
      } catch {
        // 忽略非 JSON
      }
    },
  })
}

// ──────────────────────────────────────────────
// 通用文件上传（XHR + 进度 + 取消）
// ──────────────────────────────────────────────

/** 上传任务返回值：promise + 取消函数。 */
export interface UploadTask<T = void> {
  promise: Promise<T>
  /** 取消上传。 */
  abort: () => void
}

/** 上传选项。 */
export interface UploadOptions<T = void> {
  /** 目标 URL。 */
  url: string
  /** 上传的文件。 */
  file: File | Blob
  /** 是否带 token（默认 true）。 */
  auth?: boolean
  /** 自定义 headers。 */
  headers?: Record<string, string>
  /** 上传进度回调。 */
  onProgress?: (loaded: number, total: number) => void
  /** 响应解析器（默认按 JSON 解析后返回 data 字段，或返回 void）。 */
  parseResponse?: (xhr: XMLHttpRequest) => T
}

/**
 * 通用文件上传（XHR，带进度 + 可取消）。
 *
 * @returns { promise, abort }
 *
 * 用法：
 *   const task = createUploadTask({
 *     url: '/api/v1/images/import',
 *     file: tarFile,
 *     onProgress: (loaded, total) => updateBar(loaded / total),
 *   })
 *   await task.promise
 *   // 取消：task.abort()
 */
export function createUploadTask<T = void>(options: UploadOptions<T>): UploadTask<T> {
  const {
    url,
    file,
    auth = true,
    headers = {},
    onProgress,
    parseResponse,
  } = options

  const xhr = new XMLHttpRequest()
  xhr.open('POST', url)

  if (auth) {
    const token = getToken()
    if (token) xhr.setRequestHeader('Authorization', `Bearer ${token}`)
  }
  for (const [k, v] of Object.entries(headers)) {
    xhr.setRequestHeader(k, v)
  }

  xhr.upload.onprogress = (e) => {
    if (e.lengthComputable) onProgress?.(e.loaded, e.total)
  }

  const promise = new Promise<T>((resolve, reject) => {
    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        if (parseResponse) {
          resolve(parseResponse(xhr))
        } else {
          resolve(undefined as T)
        }
      } else {
        reject(new Error(`上传失败: HTTP ${xhr.status} ${xhr.responseText.slice(0, 200)}`))
      }
    }
    xhr.onerror = () => reject(new Error('上传请求失败'))
    xhr.onabort = () => reject(new DOMException('aborted', 'AbortError'))
  })

  xhr.send(file)

  return {
    promise,
    abort: () => xhr.abort(),
  }
}
