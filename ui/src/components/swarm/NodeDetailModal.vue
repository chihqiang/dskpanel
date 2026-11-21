<script setup lang="ts">
import { ref, watch } from 'vue'
import Badge from '@/components/ui/Badge.vue'
import Modal from '@/components/ui/Modal.vue'
import { useToast } from '@/composables/useToast'
import { nodeStateVariant, nodeAvailVariant, type BadgeVariant } from '@/utils/docker'
import {
  swarmNodeInspect,
  type SwarmNodeItem,
} from '@/api/swarm'

const props = defineProps<{
  open: boolean
  node: SwarmNodeItem | null
}>()

const emit = defineEmits<{ 'update:open': [value: boolean] }>()

const toast = useToast()

const detailLoading = ref(false)
const nodeDetail = ref<{
  id: string
  name: string
  hostname: string
  role: string
  state: string
  availability: string
  leader: boolean
  addr: string
  version: string
  os: string
  arch: string
  cpu: number
  memory: number
  labels: Record<string, string>
  raw: string
} | null>(null)

function availVariant(a: string): BadgeVariant {
  return nodeAvailVariant(a)
}

async function fetchDetail(row: SwarmNodeItem): Promise<void> {
  detailLoading.value = true
  try {
    const raw = (await swarmNodeInspect(row.id)) as Record<string, unknown>
    const spec = (raw?.Spec ?? {}) as Record<string, unknown>
    const desc = (raw?.Description ?? {}) as Record<string, unknown>
    const status = (raw?.Status ?? {}) as Record<string, unknown>
    const mgr = (raw?.ManagerStatus ?? null) as Record<string, unknown> | null
    const engine = (desc?.Engine ?? {}) as Record<string, unknown>
    const resources = (desc?.Resources ?? {}) as Record<string, unknown>
    const platform = (desc?.Platform ?? {}) as Record<string, unknown>
    const labels = (spec?.Labels ?? {}) as Record<string, string>
    nodeDetail.value = {
      id: String(raw?.ID ?? ''),
      name: String((spec?.Name as string) ?? '') || String((desc?.Hostname as string) ?? ''),
      hostname: String((desc?.Hostname as string) ?? ''),
      role: String((spec?.Role as string) ?? ''),
      state: String((status?.State as string) ?? ''),
      availability: String((spec?.Availability as string) ?? ''),
      leader: Boolean((mgr?.Leader as boolean) ?? false),
      addr: String((status?.Addr as string) ?? ''),
      version: String((engine?.EngineVersion as string) ?? ''),
      os: String((platform?.OS as string) ?? ''),
      arch: String((platform?.Architecture as string) ?? ''),
      cpu: Number((resources?.NanoCPUs as number) ?? 0) / 1e9,
      memory: Math.round((Number((resources?.MemoryBytes as number) ?? 0) / 1024 / 1024 / 1024) * 10) / 10,
      labels,
      raw: JSON.stringify(raw, null, 2),
    }
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    detailLoading.value = false
  }
}

watch(
  () => [props.open, props.node] as const,
  ([open, node]) => {
    if (open && node) {
      nodeDetail.value = null
      void fetchDetail(node)
    }
  },
  { immediate: true },
)
</script>

<template>
  <Modal :open="open" @update:open="emit('update:open', $event)" title="节点详情" width="max-w-3xl">
    <div v-if="detailLoading" class="py-8 text-center text-sm text-slate-400">加载中…</div>
    <div v-else-if="nodeDetail" class="max-h-[70vh] space-y-4 overflow-y-auto pr-1">
      <div class="grid grid-cols-2 gap-x-6 gap-y-3 sm:grid-cols-3">
        <div>
          <p class="text-xs text-slate-400">名称</p>
          <p class="text-sm font-medium text-slate-700 dark:text-slate-200">{{ nodeDetail.name }}</p>
        </div>
        <div>
          <p class="text-xs text-slate-400">主机名</p>
          <p class="text-sm font-medium text-slate-700 dark:text-slate-200">{{ nodeDetail.hostname }}</p>
        </div>
        <div>
          <p class="text-xs text-slate-400">角色</p>
          <p class="text-sm"><Badge variant="blue">{{ nodeDetail.role }}</Badge></p>
        </div>
        <div>
          <p class="text-xs text-slate-400">状态</p>
          <p class="text-sm"><Badge :variant="nodeStateVariant(nodeDetail.state)">{{ nodeDetail.state }}</Badge></p>
        </div>
        <div>
          <p class="text-xs text-slate-400">可用性</p>
          <p class="text-sm"><Badge :variant="availVariant(nodeDetail.availability)">{{ nodeDetail.availability }}</Badge></p>
        </div>
        <div>
          <p class="text-xs text-slate-400">管理器</p>
          <p class="text-sm text-slate-700 dark:text-slate-200">{{ nodeDetail.leader ? 'Leader' : nodeDetail.role === 'manager' ? 'Manager' : '—' }}</p>
        </div>
        <div>
          <p class="text-xs text-slate-400">地址</p>
          <p class="text-sm font-mono text-slate-700 dark:text-slate-200">{{ nodeDetail.addr }}</p>
        </div>
        <div>
          <p class="text-xs text-slate-400">引擎版本</p>
          <p class="text-sm font-mono text-slate-700 dark:text-slate-200">{{ nodeDetail.version || '—' }}</p>
        </div>
        <div>
          <p class="text-xs text-slate-400">平台</p>
          <p class="text-sm text-slate-700 dark:text-slate-200">{{ nodeDetail.os }} / {{ nodeDetail.arch }}</p>
        </div>
        <div>
          <p class="text-xs text-slate-400">CPU</p>
          <p class="text-sm text-slate-700 dark:text-slate-200">{{ nodeDetail.cpu }} 核</p>
        </div>
        <div>
          <p class="text-xs text-slate-400">内存</p>
          <p class="text-sm text-slate-700 dark:text-slate-200">{{ nodeDetail.memory }} GB</p>
        </div>
        <div>
          <p class="text-xs text-slate-400">节点 ID</p>
          <p class="text-sm font-mono text-slate-500 dark:text-slate-400">{{ nodeDetail.id.slice(0, 12) }}</p>
        </div>
      </div>
      <div v-if="Object.keys(nodeDetail.labels).length">
        <p class="mb-1.5 text-xs text-slate-400">标签</p>
        <div class="flex flex-wrap gap-1.5">
          <span v-for="(v, k) in nodeDetail.labels" :key="k" class="rounded-md bg-slate-100 px-2 py-0.5 text-xs text-slate-600 dark:bg-slate-700 dark:text-slate-300">
            {{ k }}={{ v }}
          </span>
        </div>
      </div>
      <details>
        <summary class="cursor-pointer text-sm text-slate-500 hover:text-slate-700 dark:hover:text-slate-300">原始 inspect</summary>
        <pre class="mt-2 overflow-auto rounded-lg bg-slate-50 p-3 text-xs leading-relaxed text-slate-700 dark:bg-slate-900 dark:text-slate-300">{{ nodeDetail.raw }}</pre>
      </details>
    </div>
  </Modal>
</template>
