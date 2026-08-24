<script setup lang="ts">
import { onMounted, ref, computed, watch } from 'vue'
import { RouterLink, RouterView, useRoute, useRouter } from 'vue-router'
import { Container, Workflow, Boxes, Menu, X } from '@lucide/vue'
import { useDockerDetect } from '@/composables/useDocker'
import { useSwarmDetect } from '@/composables/useSwarmDetect'
import { useAuthStore } from '@/stores/auth'
import ActivityDrawer from '@/components/ui/ActivityDrawer.vue'
import Tooltip from '@/components/ui/Tooltip.vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const { dockerInfo, dockerChecked, detect } = useDockerDetect()
const { swarmStatus, detect: detectSwarm } = useSwarmDetect()

/** 桌面侧边栏折叠（仅 lg 及以上生效）。 */
const sidebarCollapsed = ref(false)
/** 移动端抽屉是否打开（lg 以下使用）。 */
const mobileOpen = ref(false)

onMounted(() => {
  detect()
  detectSwarm()
  auth.initFromToken()
})

/** 路由变化时自动关闭移动端抽屉。 */
watch(route, () => {
  mobileOpen.value = false
  // 进入 Swarm 区域时强制刷新集群检测，启用/关闭后菜单能即时更新。
  if (route.path.startsWith('/swarm')) {
    void detectSwarm(true)
  }
})

const username = computed(() => auth.username || 'admin')

/** Docker 栏目是否可用。 */
const dockerEnabled = computed(() => dockerInfo.value?.available ?? false)

/** Swarm 集群是否可用（决定子菜单显隐）。 */
const swarmEnabled = computed(() => swarmStatus.value?.available ?? false)

/** 路由 meta.icon 字符串 → lucide 图标组件。 */
const iconMap: Record<string, unknown> = {
  container: Container,
  workflow: Workflow,
  boxes: Boxes,
}

interface ChildNavItem {
  label: string
  to: string
  active: boolean
}

interface NavItem {
  label: string
  icon: unknown
  to: string
  disabled?: boolean
  active: boolean
  badge?: string
  children?: ChildNavItem[]
}

/** 侧边栏菜单：完全由 router/index.ts 的路由表驱动。 */
const navItems = computed<NavItem[]>(() => {
  const layoutChildren = router.options.routes.find((r) => r.path === '/')?.children ?? []
  return layoutChildren
    .filter((r) => r.meta?.menu)
    .map((r) => {
      const base = `/${r.path}`
      const children: ChildNavItem[] = (r.children ?? [])
        .filter((c) => c.path && c.meta?.menu && c.meta.title)
        // requiresSwarm 的子菜单仅在集群可用时显示。
        .filter((c) => !c.meta?.requiresSwarm || swarmEnabled.value)
        .map((c) => ({
          label: c.meta!.title as string,
          to: `${base}/${c.path}`,
          active: route.path === `${base}/${c.path}`,
        }))
      return {
        label: (r.meta?.title as string) || r.path,
        icon: iconMap[(r.meta?.icon as string) ?? ''] ?? Container,
        to: base,
        disabled: r.meta?.requiresDocker ? !dockerEnabled.value : false,
        active: route.path.startsWith(base),
        badge: (r.meta?.badge as string) || undefined,
        children,
      }
    })
})

function onNavClick(item: NavItem): void {
  if (item.disabled) return
  router.push(item.to)
}

function logout(): void {
  auth.logout()
  router.push('/login')
}
</script>

<template>
  <div class="flex h-full">
    <!-- 移动端遮罩 -->
    <Transition name="fade">
      <div
        v-if="mobileOpen"
        class="fixed inset-0 z-40 bg-black/50 lg:hidden"
        @click="mobileOpen = false"
      />
    </Transition>

    <!-- 侧边导航 -->
    <aside
      class="fixed inset-y-0 left-0 z-50 flex shrink-0 flex-col border-r border-slate-200 bg-white transition-transform duration-200 dark:border-slate-700 dark:bg-slate-800 lg:static lg:transition-all"
      :class="[
        sidebarCollapsed ? 'lg:w-16' : 'lg:w-64',
        mobileOpen ? 'translate-x-0 w-64' : '-translate-x-full w-64 lg:translate-x-0',
      ]"
    >
      <!-- Logo -->
      <div class="flex h-16 items-center gap-2.5 border-b border-slate-200 px-4 dark:border-slate-700">
        <svg class="h-8 w-8 shrink-0 text-blue-600" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
          <path d="M13 2L3 14h7l-1 8 10-12h-7l1-8z" />
        </svg>
        <span v-if="!sidebarCollapsed || mobileOpen" class="text-lg font-semibold text-slate-800 dark:text-slate-100">dskpanel</span>
        <button
          class="ml-auto text-slate-400 transition-colors hover:text-slate-600 dark:hover:text-slate-200 lg:hidden"
          aria-label="关闭菜单"
          @click="mobileOpen = false"
        >
          <X class="h-5 w-5" />
        </button>
      </div>

      <!-- 导航 -->
      <nav class="flex-1 space-y-1 overflow-y-auto p-3">
        <template v-for="item in navItems" :key="item.label">
          <Tooltip
            :text="item.disabled ? '本机未检测到 Docker' : item.label"
            placement="right"
            as="button"
            :class="{ 'w-full': !sidebarCollapsed || mobileOpen }"
          >
            <button
              :disabled="item.disabled"
              class="flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-sm transition-colors disabled:cursor-not-allowed disabled:opacity-40"
              :class="[
                item.active
                  ? 'bg-blue-50 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300'
                  : 'text-slate-600 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-700',
              ]"
              @click="onNavClick(item)"
            >
            <component :is="item.icon" class="h-5 w-5 shrink-0" />
            <span v-if="!sidebarCollapsed || mobileOpen" class="truncate">{{ item.label }}</span>
            <span
              v-if="(!sidebarCollapsed || mobileOpen) && item.badge"
              class="ml-auto shrink-0 rounded-full bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-900/40 dark:text-amber-400"
            >
              {{ item.badge }}
            </span>
          </button>
          </Tooltip>
          <!-- 子菜单 -->
          <div
            v-if="(!sidebarCollapsed || mobileOpen) && item.children?.length"
            class="mb-1 ml-4 space-y-1 border-l border-slate-200 pl-2.5 dark:border-slate-700"
          >
            <RouterLink
              v-for="child in item.children"
              :key="child.to"
              :to="child.to"
              class="block truncate rounded-md px-2.5 py-1.5 text-sm transition-colors"
              :class="
                child.active
                  ? 'bg-blue-50 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300'
                  : 'text-slate-500 hover:text-slate-800 dark:text-slate-400 dark:hover:text-slate-100'
              "
            >
              {{ child.label }}
            </RouterLink>
          </div>
        </template>
      </nav>

      <!-- 折叠按钮（仅桌面） -->
      <div class="hidden border-t border-slate-200 p-3 dark:border-slate-700 lg:block">
        <button
          class="flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-sm text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-700"
          :aria-label="sidebarCollapsed ? '展开侧边栏' : '收起侧边栏'"
          @click="sidebarCollapsed = !sidebarCollapsed"
        >
          <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path v-if="sidebarCollapsed" d="M13 5l7 7-7 7M5 5l7 7-7 7" stroke-linecap="round" stroke-linejoin="round" />
            <path v-else d="M11 5l-7 7 7 7M19 5l-7 7 7 7" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
          <span v-if="!sidebarCollapsed">收起</span>
        </button>
      </div>
    </aside>

    <!-- 主区域 -->
    <div class="flex min-w-0 flex-1 flex-col">
      <!-- 顶栏 -->
      <header class="flex h-16 shrink-0 items-center justify-between border-b border-slate-200 bg-white px-4 dark:border-slate-700 dark:bg-slate-800 sm:px-6">
        <div class="flex items-center gap-3">
          <!-- 移动端汉堡 -->
          <button
            class="text-slate-500 transition-colors hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200 lg:hidden"
            aria-label="打开菜单"
            @click="mobileOpen = true"
          >
            <Menu class="h-6 w-6" />
          </button>
          <div class="text-base font-semibold text-slate-700 dark:text-slate-200">
            {{ (route.meta.title as string) || '' }}
          </div>
        </div>
        <div class="flex items-center gap-3 sm:gap-4">
          <!-- Docker 状态 -->
          <span
            v-if="dockerChecked"
            class="hidden items-center gap-2 text-sm sm:inline-flex"
            :class="dockerEnabled ? 'text-green-600 dark:text-green-400' : 'text-slate-400'"
          >
            <span class="h-2.5 w-2.5 rounded-full" :class="dockerEnabled ? 'bg-green-500' : 'bg-slate-300'" />
            Docker {{ dockerEnabled ? dockerInfo?.version : '未检测到' }}
          </span>
          <!-- 活动通知 -->
          <ActivityDrawer />
          <span class="hidden text-sm text-slate-600 dark:text-slate-300 sm:inline">{{ username }}</span>
          <button
            class="text-sm text-slate-500 transition-colors hover:text-red-600 dark:text-slate-400"
            @click="logout"
          >
            退出
          </button>
        </div>
      </header>

      <!-- 内容区 -->
      <main class="min-h-0 flex-1 overflow-y-auto bg-slate-50 p-4 dark:bg-slate-900 sm:p-6">
        <RouterView />
      </main>
    </div>
  </div>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
