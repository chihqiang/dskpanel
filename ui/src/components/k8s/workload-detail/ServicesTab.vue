<script setup lang="ts">
/**
 * 访问方式 Tab：展示工作负载关联的 Service 列表。
 * 通过 namespace 查询所有 Service，再按 selector 匹配。
 */
import { ref, watch, computed, onMounted } from 'vue'
import { RefreshCw } from '@lucide/vue'
import Badge from '@/components/ui/Badge.vue'
import Button from '@/components/ui/Button.vue'
import { serviceTypeVariant } from '@/utils/k8s'
import { k8sServices, type K8sServiceItem } from '@/api/k8s'

const props = defineProps<{
  /** 命名空间。 */
  namespace: string
  /** 工作负载的 label selector（如 { app: nginx }）。 */
  selector: Record<string, string>
  /** 是否激活。 */
  active: boolean
}>()

const loading = ref(false)
const errorMsg = ref('')
const services = ref<K8sServiceItem[]>([])
let loaded = false

/** selector 是否匹配。 */
function selectorMatches(svcSelector: Record<string, string> | undefined): boolean {
  if (!svcSelector || Object.keys(svcSelector).length === 0) return false
  const entries = Object.entries(props.selector)
  if (entries.length === 0) return false
  return entries.every(([k, v]) => svcSelector[k] === v)
}

/** 过滤出匹配当前工作负载 selector 的 Service。 */
const matchedServices = computed(() => {
  return services.value.filter((s) => selectorMatches(s.selector))
})

async function load(): Promise<void> {
  if (!props.namespace) return
  loading.value = true
  errorMsg.value = ''
  try {
    services.value = await k8sServices(props.namespace)
    loaded = true
  } catch (err) {
    errorMsg.value = (err as Error).message
    services.value = []
  } finally {
    loading.value = false
  }
}

function refresh(): void {
  loaded = false
  void load()
}

watch(
  () => props.active,
  (active) => {
    if (active && props.namespace && !loaded) {
      void load()
    }
  },
)

onMounted(() => {
  if (props.active && props.namespace) {
    void load()
  }
})

/** 端口列表格式化。 */
function fmtPorts(p: { port: number; target_port: string; protocol: string; node_port?: number }[]): string {
  return (p ?? [])
    .map((x) => `${x.port}:${x.target_port}/${x.protocol}${x.node_port ? `(NodePort ${x.node_port})` : ''}`)
    .join('，')
}
</script>

<template>
  <div class="space-y-3">
    <!-- 头部 -->
    <div class="flex items-center justify-between">
      <p class="text-sm text-slate-500 dark:text-slate-400">
        关联 Service（{{ matchedServices.length }} 个）
      </p>
      <Button variant="ghost" size="sm" :loading="loading" @click="refresh">
        <RefreshCw class="h-3.5 w-3.5" />
        刷新
      </Button>
    </div>

    <!-- 加载中 -->
    <div v-if="loading" class="flex items-center justify-center py-8 text-sm text-slate-400">
      <svg class="mr-2 h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none" aria-hidden="true">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
      </svg>
      加载中…
    </div>

    <!-- 错误 -->
    <div v-else-if="errorMsg" class="rounded-lg bg-red-50 px-4 py-3 text-sm text-red-600 dark:bg-red-900/20 dark:text-red-300">
      {{ errorMsg }}
    </div>

    <!-- Service 列表 -->
    <div v-else-if="matchedServices.length > 0" class="space-y-2">
      <div
        v-for="svc in matchedServices"
        :key="svc.name"
        class="rounded-lg border border-slate-200 px-3 py-2.5 dark:border-slate-700"
      >
        <div class="flex flex-wrap items-center gap-2">
          <span class="text-sm font-medium text-slate-700 dark:text-slate-200">{{ svc.name }}</span>
          <Badge :variant="serviceTypeVariant(svc.type)">{{ svc.type }}</Badge>
          <span class="text-xs text-slate-400">ClusterIP {{ svc.cluster_ip || '—' }}</span>
          <span v-if="svc.external_ip" class="text-xs text-slate-400">External {{ svc.external_ip }}</span>
          <span class="text-xs text-slate-400">{{ svc.created_at }}</span>
        </div>
        <p class="mt-1 font-mono text-xs text-slate-500 dark:text-slate-400">{{ fmtPorts(svc.ports ?? []) }}</p>
      </div>
    </div>

    <!-- 空 -->
    <div v-else class="py-8 text-center text-sm text-slate-400">
      暂无关联 Service
    </div>
  </div>
</template>
