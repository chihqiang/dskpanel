<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import Badge from '@/components/ui/Badge.vue'
import Modal from '@/components/ui/Modal.vue'
import { useToast } from '@/composables/useToast'
import { fmtSize } from '@/utils/format'
import {
  swarmServiceInspect,
  swarmServiceResources,
  swarmNetworks,
  type SwarmServiceItem,
  type SwarmContainerResource,
} from '@/api/swarm'

const props = defineProps<{
  open: boolean
  service: SwarmServiceItem | null
}>()

const emit = defineEmits<{ 'update:open': [value: boolean] }>()

const toast = useToast()
const router = useRouter()

const detailLoading = ref(false)
const serviceDetail = ref<{
  id: string
  name: string
  mode: string
  image: string
  replicas: string
  ports: string[]
  env: string[]
  command: string[]
  restart: string
  mounts: string[]
  secrets: string[]
  configs: string[]
  networks: string[]
  labels: Record<string, string>
  constraints: string[]
  limit_cpu: string
  limit_memory: string
  update_status: string
  raw: string
} | null>(null)

// 服务资源监控（任务容器 CPU/内存）。
const serviceResources = ref<SwarmContainerResource[]>([])
const resourcesLoading = ref(false)

async function fetchDetail(row: SwarmServiceItem): Promise<void> {
  serviceDetail.value = null
  detailLoading.value = true
  try {
    const raw = (await swarmServiceInspect(row.id)) as Record<string, unknown>
    const spec = (raw?.Spec ?? {}) as Record<string, unknown>
    const task = (spec?.TaskTemplate ?? {}) as Record<string, unknown>
    const cs = (task?.ContainerSpec ?? {}) as Record<string, unknown>
    const mode = (spec?.Mode ?? {}) as Record<string, unknown>
    const limits = (task?.Resources as Record<string, unknown> | undefined)?.Limits as Record<string, unknown> | undefined
    const placement = (task?.Placement ?? {}) as Record<string, unknown>
    const nws = (task?.Networks ?? []) as { Target?: string }[]
    const eps = (spec?.EndpointSpec ?? {}) as { Ports?: { PublishedPort?: number; TargetPort?: number; Protocol?: string }[] }
    const uc = (spec?.UpdateConfig ?? {}) as Record<string, unknown>
    const updateStatus = (raw?.UpdateStatus ?? {}) as Record<string, unknown>

    // 网络 ID → 名称映射。
    const netNameMap = new Map<string, string>()
    try {
      const nets = await swarmNetworks()
      for (const n of nets) netNameMap.set(n.id, n.name)
    } catch {
      // 忽略
    }

    let replicas = '—'
    if (mode.Global) {
      replicas = 'global'
    } else if (mode.Replicated) {
      replicas = String((mode.Replicated as { Replicas?: number }).Replicas ?? '—')
    }
    serviceDetail.value = {
      id: String(raw?.ID ?? ''),
      name: String(spec?.Name ?? ''),
      mode: mode.Global ? 'global' : 'replicated',
      image: String((cs?.Image as string) ?? ''),
      replicas,
      ports: (eps.Ports ?? []).map((p) => `${p.PublishedPort ?? '?'}→${p.TargetPort ?? '?'}/${p.Protocol ?? 'tcp'}`),
      env: (cs?.Env as string[] | undefined) ?? [],
      command: (cs?.Args as string[] | undefined) ?? [],
      restart: String((task?.RestartPolicy as Record<string, unknown> | undefined)?.Condition ?? ''),
      mounts: ((cs?.Mounts as { Type?: string; Source?: string; Target?: string }[] | undefined) ?? []).map(
        (m) => `${m.Type ?? 'volume'} ${m.Source ?? ''} → ${m.Target ?? ''}`,
      ),
      secrets: ((cs?.Secrets as { SecretName?: string }[] | undefined) ?? []).map((s) => s.SecretName ?? ''),
      configs: ((cs?.Configs as { ConfigName?: string }[] | undefined) ?? []).map((c) => c.ConfigName ?? ''),
      networks: (nws ?? []).map((n) => netNameMap.get(n.Target ?? '') ?? n.Target ?? '').filter(Boolean),
      labels: (spec?.Labels as Record<string, string> | undefined) ?? {},
      constraints: (placement?.Constraints as string[] | undefined) ?? [],
      limit_cpu: limits ? String((limits?.NanoCPUs as number | undefined) ?? 0) : '',
      limit_memory: limits ? String(Math.round(((limits?.MemoryBytes as number | undefined) ?? 0) / 1024 / 1024)) : '',
      update_status: String(updateStatus?.State ?? uc?.State ?? ''),
      raw: JSON.stringify(raw, null, 2),
    }
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    detailLoading.value = false
  }

  // 并行加载服务资源监控（任务容器 CPU/内存）。
  resourcesLoading.value = true
  serviceResources.value = []
  try {
    serviceResources.value = await swarmServiceResources(row.id)
  } catch {
    serviceResources.value = []
  } finally {
    resourcesLoading.value = false
  }
}

watch(
  () => [props.open, props.service] as const,
  ([open, service]) => {
    if (open && service) {
      void fetchDetail(service)
    }
  },
  { immediate: true },
)
</script>

<template>
  <Modal :open="open" @update:open="emit('update:open', $event)" title="服务详情" width="max-w-3xl">
    <div v-if="detailLoading" class="py-8 text-center text-sm text-slate-400">加载中…</div>
    <div v-else-if="serviceDetail" class="max-h-[70vh] space-y-4 overflow-y-auto pr-1">
      <div class="grid grid-cols-2 gap-x-6 gap-y-3 sm:grid-cols-3">
        <div>
          <p class="text-xs text-slate-400">名称</p>
          <p class="text-sm font-medium text-slate-700 dark:text-slate-200">{{ serviceDetail.name }}</p>
        </div>
        <div>
          <p class="text-xs text-slate-400">模式</p>
          <p class="text-sm"><Badge variant="blue">{{ serviceDetail.mode }}</Badge></p>
        </div>
        <div>
          <p class="text-xs text-slate-400">副本</p>
          <p class="text-sm text-slate-700 dark:text-slate-200">{{ serviceDetail.replicas }}</p>
        </div>
        <div>
          <p class="text-xs text-slate-400">镜像</p>
          <p class="text-sm font-mono text-slate-700 dark:text-slate-200">{{ serviceDetail.image }}</p>
        </div>
        <div>
          <p class="text-xs text-slate-400">重启策略</p>
          <p class="text-sm text-slate-700 dark:text-slate-200">{{ serviceDetail.restart || '—' }}</p>
        </div>
        <div>
          <p class="text-xs text-slate-400">更新状态</p>
          <p class="text-sm text-slate-700 dark:text-slate-200">{{ serviceDetail.update_status || '—' }}</p>
        </div>
        <div>
          <p class="text-xs text-slate-400">CPU 限制</p>
          <p class="text-sm text-slate-700 dark:text-slate-200">{{ serviceDetail.limit_cpu ? `${serviceDetail.limit_cpu} 核` : '不限' }}</p>
        </div>
        <div>
          <p class="text-xs text-slate-400">内存限制</p>
          <p class="text-sm text-slate-700 dark:text-slate-200">{{ serviceDetail.limit_memory ? `${serviceDetail.limit_memory} MB` : '不限' }}</p>
        </div>
        <div>
          <p class="text-xs text-slate-400">服务 ID</p>
          <p class="text-sm font-mono text-slate-500 dark:text-slate-400">{{ serviceDetail.id.slice(0, 12) }}</p>
        </div>
      </div>

      <div v-if="serviceDetail.ports.length">
        <p class="mb-1.5 text-xs text-slate-400">端口</p>
        <div class="flex flex-wrap gap-1.5">
          <span v-for="(p, i) in serviceDetail.ports" :key="i" class="rounded-md bg-slate-100 px-2 py-0.5 font-mono text-xs text-slate-600 dark:bg-slate-700 dark:text-slate-300">{{ p }}</span>
        </div>
      </div>
      <div v-if="serviceDetail.networks.length">
        <p class="mb-1.5 text-xs text-slate-400">网络</p>
        <div class="flex flex-wrap gap-1.5">
          <span
            v-for="(n, i) in serviceDetail.networks"
            :key="i"
            class="cursor-pointer rounded-md bg-blue-50 px-2 py-0.5 text-xs text-blue-700 transition-colors hover:bg-blue-100 dark:bg-blue-900/40 dark:text-blue-300 dark:hover:bg-blue-900/60"
            title="查看网络"
            @click="router.push('/swarm/networks')"
          >{{ n }}</span>
        </div>
      </div>
      <div v-if="serviceDetail.mounts.length">
        <p class="mb-1.5 text-xs text-slate-400">挂载</p>
        <div class="flex flex-wrap gap-1.5">
          <span v-for="(m, i) in serviceDetail.mounts" :key="i" class="rounded-md bg-slate-100 px-2 py-0.5 font-mono text-xs text-slate-600 dark:bg-slate-700 dark:text-slate-300">{{ m }}</span>
        </div>
      </div>
      <div v-if="serviceDetail.env.length">
        <p class="mb-1.5 text-xs text-slate-400">环境变量</p>
        <div class="space-y-0.5">
          <p v-for="(e, i) in serviceDetail.env" :key="i" class="font-mono text-xs text-slate-600 dark:text-slate-300">{{ e }}</p>
        </div>
      </div>
      <div v-if="serviceDetail.command?.length">
        <p class="mb-1.5 text-xs text-slate-400">命令</p>
        <p class="font-mono text-xs text-slate-600 dark:text-slate-300">{{ serviceDetail.command?.join(' ') }}</p>
      </div>
      <div v-if="serviceDetail.secrets.length || serviceDetail.configs.length" class="grid grid-cols-2 gap-4">
        <div v-if="serviceDetail.secrets.length">
          <p class="mb-1.5 text-xs text-slate-400">Secret</p>
          <div class="flex flex-wrap gap-1.5">
            <span
              v-for="(s, i) in serviceDetail.secrets"
              :key="i"
              class="cursor-pointer rounded-md bg-rose-50 px-2 py-0.5 text-xs text-rose-700 transition-colors hover:bg-rose-100 dark:bg-rose-900/40 dark:text-rose-300 dark:hover:bg-rose-900/60"
              title="查看 Secret"
              @click="router.push('/swarm/secrets')"
            >{{ s }}</span>
          </div>
        </div>
        <div v-if="serviceDetail.configs.length">
          <p class="mb-1.5 text-xs text-slate-400">Config</p>
          <div class="flex flex-wrap gap-1.5">
            <span
              v-for="(c, i) in serviceDetail.configs"
              :key="i"
              class="cursor-pointer rounded-md bg-indigo-50 px-2 py-0.5 text-xs text-indigo-700 transition-colors hover:bg-indigo-100 dark:bg-indigo-900/40 dark:text-indigo-300 dark:hover:bg-indigo-900/60"
              title="查看 Config"
              @click="router.push('/swarm/secrets')"
            >{{ c }}</span>
          </div>
        </div>
      </div>
      <div v-if="serviceDetail.constraints.length">
        <p class="mb-1.5 text-xs text-slate-400">节点约束</p>
        <div class="space-y-0.5">
          <p v-for="(c, i) in serviceDetail.constraints" :key="i" class="font-mono text-xs text-slate-600 dark:text-slate-300">{{ c }}</p>
        </div>
      </div>
      <div v-if="Object.keys(serviceDetail.labels).length">
        <p class="mb-1.5 text-xs text-slate-400">标签</p>
        <div class="flex flex-wrap gap-1.5">
          <span v-for="(v, k) in serviceDetail.labels" :key="k" class="rounded-md bg-slate-100 px-2 py-0.5 text-xs text-slate-600 dark:bg-slate-700 dark:text-slate-300">{{ k }}={{ v }}</span>
        </div>
      </div>

      <!-- 服务资源监控（任务容器 CPU/内存） -->
      <div>
        <p class="mb-1.5 text-xs text-slate-400">任务容器资源</p>
        <div v-if="resourcesLoading" class="py-3 text-center text-xs text-slate-400">加载中…</div>
        <div v-else-if="serviceResources.length === 0" class="rounded-md border border-dashed border-slate-300 px-3 py-3 text-center text-xs text-slate-400 dark:border-slate-700">
          暂无运行中的任务容器
        </div>
        <div v-else class="space-y-2">
          <div
            v-for="r in serviceResources"
            :key="r.task_id"
            class="rounded-md border border-slate-200 bg-slate-50 p-3 dark:border-slate-700 dark:bg-slate-900"
          >
            <div class="flex items-center justify-between text-xs">
              <span class="font-mono text-slate-600 dark:text-slate-300">#{{ r.slot }} · {{ r.node_name || '—' }}</span>
              <Badge :variant="r.state === 'running' ? 'green' : 'gray'" dot>{{ r.state }}</Badge>
            </div>
            <div class="mt-2 grid grid-cols-2 gap-3">
              <div>
                <p class="text-[10px] text-slate-400">CPU</p>
                <p class="font-mono text-sm font-semibold text-slate-800 dark:text-slate-100">
                  {{ r.cpu_percent.toFixed(1) }}%
                </p>
              </div>
              <div>
                <p class="text-[10px] text-slate-400">内存</p>
                <p class="font-mono text-sm font-semibold text-slate-800 dark:text-slate-100">
                  {{ fmtSize(r.mem_usage) }}
                  <span class="text-xs font-normal text-slate-400">/ {{ fmtSize(r.mem_limit) }} ({{ r.mem_percent.toFixed(1) }}%)</span>
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>

      <details>
        <summary class="cursor-pointer text-sm text-slate-500 hover:text-slate-700 dark:hover:text-slate-300">原始 inspect</summary>
        <pre class="mt-2 overflow-auto rounded-lg bg-slate-50 p-3 text-xs leading-relaxed text-slate-700 dark:bg-slate-900 dark:text-slate-300">{{ serviceDetail.raw }}</pre>
      </details>
    </div>
  </Modal>
</template>
