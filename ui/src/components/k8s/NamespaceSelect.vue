<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { Boxes, ChevronDown, Check, History, Search, X, Clock3, LayoutGrid } from '@lucide/vue'
import { useNamespaces, ALL_NS } from '@/composables/useNamespaces'

const {
  filtered,
  current,
  search,
  recent,
  searchHistory,
  loading,
  loadNamespaces,
  setCurrent,
  setSearch,
  clearSearch,
  clearSearchHistory,
  clearRecent,
} = useNamespaces()

const open = ref(false)
const rootEl = ref<HTMLElement | null>(null)
const searchInput = ref<HTMLInputElement | null>(null)

function toggle(): void {
  open.value = !open.value
  if (open.value) {
    // 打开时清空搜索并聚焦输入框。
    clearSearch()
    void nextTick(() => searchInput.value?.focus())
  }
}

function pick(name: string): void {
  setCurrent(name)
  open.value = false
}

function onDocClick(e: MouseEvent): void {
  if (rootEl.value && !rootEl.value.contains(e.target as Node)) {
    open.value = false
  }
}

function onDocKeydown(e: KeyboardEvent): void {
  if (e.key === 'Escape') open.value = false
}

watch(open, (v) => {
  if (v) {
    document.addEventListener('click', onDocClick)
    document.addEventListener('keydown', onDocKeydown)
  } else {
    document.removeEventListener('click', onDocClick)
    document.removeEventListener('keydown', onDocKeydown)
  }
})

onBeforeUnmount(() => {
  document.removeEventListener('click', onDocClick)
  document.removeEventListener('keydown', onDocKeydown)
})

// 打开面板时确保列表已加载。
watch(open, (v) => {
  if (v) void loadNamespaces()
})
</script>

<template>
  <div ref="rootEl" class="relative inline-block">
    <!-- 触发器 -->
    <button
      class="flex h-8 items-center gap-1.5 rounded-md border border-slate-200 bg-white px-2.5 text-sm text-slate-700 transition-colors hover:border-blue-300 hover:bg-blue-50/40 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:hover:border-blue-500 dark:hover:bg-blue-900/20"
      :title="`当前命名空间：${current}`"
      @click="toggle"
    >
      <Boxes class="h-3.5 w-3.5 shrink-0 text-slate-400" />
      <span class="max-w-40 truncate">{{ current === ALL_NS ? '所有命名空间' : current }}</span>
      <ChevronDown class="h-3.5 w-3.5 shrink-0 text-slate-400" :class="open ? 'rotate-180 transition-transform' : 'transition-transform'" />
    </button>

    <!-- 面板 -->
    <Transition name="ns-dropdown">
      <div
        v-if="open"
        class="absolute right-0 top-full z-30 mt-1 w-72 overflow-hidden rounded-lg border border-slate-200 bg-white shadow-lg dark:border-slate-700 dark:bg-slate-800"
      >
        <!-- 搜索框 -->
        <div class="border-b border-slate-100 p-2 dark:border-slate-700">
          <div class="flex items-center gap-1.5 rounded-md bg-slate-100 px-2 dark:bg-slate-700/60">
            <Search class="h-3.5 w-3.5 shrink-0 text-slate-400" />
            <input
              ref="searchInput"
              v-model="search"
              type="text"
              placeholder="搜索命名空间…"
              class="h-7 w-full bg-transparent text-sm outline-none placeholder:text-slate-400"
              @input="setSearch(search)"
            />
            <button v-if="search" class="shrink-0 text-slate-400 transition-colors hover:text-slate-600" aria-label="清空搜索" @click="clearSearch">
              <X class="h-3.5 w-3.5" />
            </button>
          </div>
        </div>

        <!-- 最近使用 -->
        <div v-if="recent.length && !search" class="border-b border-slate-100 px-2 py-1.5 dark:border-slate-700">
          <div class="flex items-center justify-between px-1">
            <span class="flex items-center gap-1 text-xs text-slate-400"><Clock3 class="h-3 w-3" />最近使用</span>
            <button class="text-xs text-slate-400 transition-colors hover:text-slate-600" @click="clearRecent">清空</button>
          </div>
          <div class="mt-1 flex flex-wrap gap-1">
            <button
              v-for="n in recent"
              :key="n"
              class="inline-flex max-w-full items-center gap-1 rounded-full bg-blue-50 px-2 py-0.5 text-xs text-blue-700 transition-colors hover:bg-blue-100 dark:bg-blue-900/40 dark:text-blue-300 dark:hover:bg-blue-900/60"
              @click="pick(n)"
            >
              <span class="truncate">{{ n }}</span>
            </button>
          </div>
        </div>

        <!-- 搜索历史 -->
        <div v-if="searchHistory.length && !search" class="border-b border-slate-100 px-2 py-1.5 dark:border-slate-700">
          <div class="flex items-center justify-between px-1">
            <span class="flex items-center gap-1 text-xs text-slate-400"><History class="h-3 w-3" />搜索记录</span>
            <button class="text-xs text-slate-400 transition-colors hover:text-slate-600" @click="clearSearchHistory">清空</button>
          </div>
          <div class="mt-1 flex flex-wrap gap-1">
            <button
              v-for="k in searchHistory"
              :key="k"
              class="inline-flex max-w-full items-center gap-1 rounded bg-slate-100 px-2 py-0.5 text-xs text-slate-600 transition-colors hover:bg-slate-200 dark:bg-slate-700 dark:text-slate-300 dark:hover:bg-slate-600"
              @click="setSearch(k)"
            >
              <span class="truncate">{{ k }}</span>
            </button>
          </div>
        </div>

        <!-- 命名空间列表 -->
        <div class="max-h-60 overflow-y-auto p-1">
          <!-- 所有命名空间（不参与搜索过滤，始终显示在顶部） -->
          <button
            class="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm font-medium text-slate-700 transition-colors hover:bg-blue-50 dark:text-slate-200 dark:hover:bg-blue-900/30"
            :class="current === ALL_NS ? 'bg-blue-50 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300' : ''"
            @click="pick(ALL_NS)"
          >
            <LayoutGrid class="h-3.5 w-3.5 shrink-0 text-slate-400" />
            <span class="truncate">所有命名空间</span>
            <Check v-if="current === ALL_NS" class="ml-auto h-3.5 w-3.5 shrink-0 text-blue-600 dark:text-blue-400" />
          </button>

          <div v-if="loading" class="px-2 py-3 text-center text-sm text-slate-400">加载中…</div>
          <template v-else>
            <div v-if="filtered.length" class="mt-0.5 border-t border-slate-100 pt-0.5 dark:border-slate-700">
              <button
                v-for="n in filtered"
                :key="n.name"
                class="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm text-slate-700 transition-colors hover:bg-blue-50 dark:text-slate-200 dark:hover:bg-blue-900/30"
                @click="pick(n.name)"
              >
                <span class="truncate">{{ n.name }}</span>
                <Check v-if="n.name === current" class="ml-auto h-3.5 w-3.5 shrink-0 text-blue-600 dark:text-blue-400" />
              </button>
            </div>
            <p v-if="filtered.length === 0" class="px-2 py-3 text-center text-sm text-slate-400">无匹配命名空间</p>
          </template>
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.ns-dropdown-enter-active,
.ns-dropdown-leave-active {
  transition:
    opacity 0.12s ease,
    transform 0.12s ease;
}
.ns-dropdown-enter-from,
.ns-dropdown-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
</style>
