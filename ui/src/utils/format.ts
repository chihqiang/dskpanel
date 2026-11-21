/**
 * 通用格式化工具函数。
 *
 * 全项目共享，避免在各组件/页面中重复定义 fmtSize、shortId、fmtTime 等。
 */

// ──────────────────────────────────────────────
// 字节大小
// ──────────────────────────────────────────────

/** 将字节数格式化为人类可读的文件大小（GB / MB / KB / B）。 */
export function fmtSize(bytes: number): string {
  if (bytes >= 1024 * 1024 * 1024) return (bytes / 1024 / 1024 / 1024).toFixed(1) + ' GB'
  if (bytes >= 1024 * 1024) return (bytes / 1024 / 1024).toFixed(1) + ' MB'
  if (bytes >= 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return bytes + ' B'
}

// ──────────────────────────────────────────────
// ID 截断
// ──────────────────────────────────────────────

/** 将 Docker 镜像/容器 ID 截断为前 12 位（自动移除 sha256: 前缀）。 */
export function shortId(id: string): string {
  return id.replace('sha256:', '').slice(0, 12)
}

// ──────────────────────────────────────────────
// 时间格式化
// ──────────────────────────────────────────────

/**
 * 将 Unix 秒级时间戳格式化为本地时间字符串。
 * 用于 Docker API 返回的 created 字段（秒级 timestamp）。
 */
export function fmtUnixTime(ts: number): string {
  if (!ts) return '-'
  return new Date(ts * 1000).toLocaleString()
}

/**
 * 将 ISO 8601 字符串或已有时间戳格式化为本地时间。
 * 无效日期时返回原始字符串或 '-'。
 * 用于 Docker inspect 返回的 RFC3339 时间字段。
 */
export function fmtISOTime(ts: string | number): string {
  const d = new Date(ts)
  return isNaN(d.getTime()) ? (String(ts) || '-') : d.toLocaleString()
}

/**
 * 将时间戳格式化为相对时间文案（刚刚 / N 分钟前 / N 小时前 / 完整日期）。
 * 用于活动日志等场景。
 */
export function fmtRelativeTime(ts: number): string {
  const d = new Date(ts)
  const diff = Date.now() - ts
  if (diff < 60000) return '刚刚'
  if (diff < 3600000) return `${Math.floor(diff / 60000)} 分钟前`
  if (diff < 86400000) return `${Math.floor(diff / 3600000)} 小时前`
  return d.toLocaleString('zh-CN', { hour12: false })
}

// ──────────────────────────────────────────────
// 键值对
// ──────────────────────────────────────────────

/**
 * 将 Record<string, string> 转为 [key, value][] 数组（空值返回空数组）。
 * 用于详情面板中 labels/annotations 等字段的渲染。
 */
export function kvEntries(map?: Record<string, string> | null): [string, string][] {
  return map ? Object.entries(map) : []
}

/**
 * 将 Record<string, unknown> 转为 [key, string][] 数组。
 * 对象/数组类型的值会 JSON.stringify，其余 toString。
 * 用于详情面板中 config.env 等混合类型字段的渲染。
 */
export function kvEntriesDeep(map?: Record<string, unknown> | null): [string, string][] {
  if (!map) return []
  return Object.entries(map).map(([k, v]) => [
    k,
    typeof v === 'object' ? JSON.stringify(v) : String(v),
  ])
}
