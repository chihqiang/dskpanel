/**
 * 命名空间管理 Hook（模块级单例）。
 *
 * 功能：
 * - 统一加载命名空间列表（全局共享，避免每页重复请求）
 * - 当前选中命名空间持久化（localStorage，跨页面/会话保留）
 * - 记录「最近使用」与「搜索历史」（可清除）
 * - 提供搜索过滤（列表太多时可快速定位，无需逐个翻看）
 */
import { computed, ref } from 'vue'
import { k8sNamespaces, type K8sNamespaceItem } from '@/api/k8s'

const SELECTED_KEY = 'dskpanel_k8s_ns_current'
const RECENT_KEY = 'dskpanel_k8s_ns_recent'
const SEARCH_HIST_KEY = 'dskpanel_k8s_ns_search_hist'
const MAX_RECENT = 8
const MAX_SEARCH_HIST = 8

/** 从 localStorage 读取字符串数组（容错解析）。 */
function loadList(key: string): string[] {
  try {
    const raw = localStorage.getItem(key)
    const arr = raw ? JSON.parse(raw) : []
    return Array.isArray(arr) ? arr.filter((x) => typeof x === 'string') : []
  } catch {
    return []
  }
}

function persist(key: string, list: string[]): void {
  localStorage.setItem(key, JSON.stringify(list))
}

// ──────────────────────────────────────────────
// 模块级状态（跨页面共享）
// ──────────────────────────────────────────────

/** 命名空间列表（已加载则不再重复请求）。 */
const namespaces = ref<K8sNamespaceItem[]>([])
const loaded = ref(false)
const loading = ref(false)

/** 当前选中的命名空间（持久化）。 */
const current = ref<string>(localStorage.getItem(SELECTED_KEY) || 'default')

/** 搜索关键词。 */
const search = ref('')

/** 最近使用（切换记录）。 */
const recent = ref<string[]>(loadList(RECENT_KEY))

/** 搜索历史（输入过的关键词）。 */
const searchHistory = ref<string[]>(loadList(SEARCH_HIST_KEY))

// ──────────────────────────────────────────────
// 派生状态
// ──────────────────────────────────────────────

/** 按搜索关键词过滤后的命名空间列表。 */
const filtered = computed(() => {
  const kw = search.value.trim().toLowerCase()
  if (!kw) return namespaces.value
  return namespaces.value.filter((n) => n.name.toLowerCase().includes(kw))
})

/** 最近使用（排除当前选中）。 */
const recentList = computed(() => recent.value.filter((n) => n !== current.value))

// ──────────────────────────────────────────────
// 操作
// ──────────────────────────────────────────────

/** 加载命名空间列表（已加载则跳过；force 强制刷新）。 */
async function loadNamespaces(force = false): Promise<void> {
  if (loaded.value && !force) return
  loading.value = true
  try {
    namespaces.value = await k8sNamespaces()
    loaded.value = true
    // 当前命名空间可能已被删除 → 回退到第一个。
    if (namespaces.value.length && !namespaces.value.some((n) => n.name === current.value)) {
      setCurrent(namespaces.value[0].name)
    }
  } catch {
    namespaces.value = []
  } finally {
    loading.value = false
  }
}

/** 切换当前命名空间，并记录最近使用。 */
function setCurrent(name: string): void {
  current.value = name
  localStorage.setItem(SELECTED_KEY, name)
  recent.value = [name, ...recent.value.filter((n) => n !== name)].slice(0, MAX_RECENT)
  persist(RECENT_KEY, recent.value)
}

/** 更新搜索关键词并记录搜索历史。 */
function setSearch(kw: string): void {
  search.value = kw
  const k = kw.trim()
  if (k) {
    searchHistory.value = [k, ...searchHistory.value.filter((n) => n !== k)].slice(0, MAX_SEARCH_HIST)
    persist(SEARCH_HIST_KEY, searchHistory.value)
  }
}

/** 清空搜索关键词。 */
function clearSearch(): void {
  search.value = ''
}

/** 清空搜索历史。 */
function clearSearchHistory(): void {
  searchHistory.value = []
  persist(SEARCH_HIST_KEY, [])
}

/** 清空最近使用。 */
function clearRecent(): void {
  recent.value = []
  persist(RECENT_KEY, [])
}

/** 命名空间 Hook。 */
export function useNamespaces() {
  return {
    namespaces,
    filtered,
    current,
    search,
    recent: recentList,
    searchHistory,
    loading,
    loadNamespaces,
    setCurrent,
    setSearch,
    clearSearch,
    clearSearchHistory,
    clearRecent,
  }
}
