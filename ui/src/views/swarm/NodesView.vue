<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Eye, UserCheck, UserX, PauseCircle, Trash2, RefreshCw, KeyRound, Copy, Check } from '@lucide/vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import Modal from '@/components/ui/Modal.vue'
import DataTable, { type DataTableColumn } from '@/components/ui/DataTable.vue'
import RowActions, { type RowAction } from '@/components/ui/RowActions.vue'
import { useToast } from '@/composables/useToast'
import { useConfirm } from '@/composables/useConfirm'
import { useClipboard } from '@/utils/clipboard'
import { nodeStateVariant, nodeAvailVariant, type BadgeVariant } from '@/utils/docker'
import {
  swarmNodes,
  swarmNodeInspect,
  swarmSetNodeAvailability,
  swarmRemoveNode,
  swarmJoinTokens,
  type SwarmNodeItem,
  type JoinTokens,
} from '@/api/swarm'

const toast = useToast()
const confirm = useConfirm()
const { copy } = useClipboard()

const loading = ref(false)
const errorMsg = ref('')
const items = ref<SwarmNodeItem[]>([])

const detailOpen = ref(false)
const detail = ref('')
const detailLoading = ref(false)
// 节点详情结构化展示。
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

// join token 弹窗。
const tokenOpen = ref(false)
const tokens = ref<JoinTokens | null>(null)
const tokenLoading = ref(false)
const copied = ref('')

async function openTokens(): Promise<void> {
  tokenOpen.value = true
  tokens.value = null
  copied.value = ''
  tokenLoading.value = true
  try {
    tokens.value = await swarmJoinTokens()
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    tokenLoading.value = false
  }
}

async function copyToken(text: string, key: string): Promise<void> {
  const ok = await copy(text, '已复制到剪贴板', '复制失败，请手动复制')
  if (ok) {
    copied.value = key
    setTimeout(() => {
      if (copied.value === key) copied.value = ''
    }, 1500)
  }
}

async function load(): Promise<void> {
  loading.value = true
  errorMsg.value = ''
  try {
    items.value = await swarmNodes()
  } catch (err) {
    errorMsg.value = (err as Error).message
    items.value = []
  } finally {
    loading.value = false
  }
}
onMounted(load)

function availVariant(a: string): BadgeVariant {
  return nodeAvailVariant(a)
}

function setAvailability(row: SwarmNodeItem, availability: string): void {
  void confirm(
    '切换可用性',
    `确认将节点「${row.name}」可用性切换为 ${availability}？`,
    async () => {
      await swarmSetNodeAvailability(row.id, availability)
      toast.success(`节点「${row.name}」已切换为 ${availability}`)
      await load()
    },
    { danger: availability === 'drain' },
  )
}

function removeNode(row: SwarmNodeItem): void {
  void confirm(
    '删除节点',
    `确认从集群移除节点「${row.name}」？此操作不可恢复。`,
    async () => {
      await swarmRemoveNode(row.id, true)
      toast.success(`已移除节点「${row.name}」`)
      await load()
    },
    { danger: true },
  )
}

async function openDetail(row: SwarmNodeItem): Promise<void> {
  detailOpen.value = true
  detail.value = ''
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
    detail.value = nodeDetail.value?.raw ?? ''
  } catch (err) {
    detail.value = `加载失败: ${(err as Error).message}`
  } finally {
    detailLoading.value = false
  }
}

function buildActions(row: SwarmNodeItem): RowAction[] {
  return [
    { key: 'detail', label: '详情', icon: Eye, onClick: () => openDetail(row) },
    {
      key: 'active',
      label: '置为 Active',
      icon: UserCheck,
      disabled: row.availability === 'active',
      onClick: () => setAvailability(row, 'active'),
    },
    {
      key: 'drain',
      label: '置为 Drain',
      icon: UserX,
      disabled: row.availability === 'drain',
      onClick: () => setAvailability(row, 'drain'),
    },
    {
      key: 'pause',
      label: '置为 Pause',
      icon: PauseCircle,
      disabled: row.availability === 'pause',
      onClick: () => setAvailability(row, 'pause'),
    },
    { key: 'remove', label: '删除', icon: Trash2, danger: true, onClick: () => removeNode(row) },
  ]
}

const columns: DataTableColumn[] = [
  { label: '名称', key: 'name' },
  { label: '角色', key: 'role', width: '90px' },
  { label: '状态', key: 'state', width: '100px' },
  { label: '可用性', key: 'availability', width: '110px' },
  { label: '地址', key: 'addr', width: '130px' },
  { label: '版本', key: 'version', width: '90px' },
  { label: '操作', key: 'actions', width: '180px', align: 'right' },
]
</script>

<template>
  <div>
    <DataTable
      title="节点列表"
      :columns="columns"
      :data="items"
      :loading="loading"
      :error="errorMsg"
      row-key="id"
      empty-text="暂无节点"
    >
      <template #toolbar>
        <Button variant="secondary" size="sm" :loading="loading" @click="load">
          <RefreshCw class="h-3.5 w-3.5" />
          刷新
        </Button>
        <Button variant="secondary" size="sm" @click="openTokens">
          <KeyRound class="h-3.5 w-3.5" />
          加入令牌
        </Button>
      </template>
      <template #cell-state="{ row }">
        <Badge :variant="nodeStateVariant((row as SwarmNodeItem).state)" dot>
          {{ (row as SwarmNodeItem).state }}
        </Badge>
      </template>
      <template #cell-availability="{ row }">
        <Badge :variant="availVariant((row as SwarmNodeItem).availability)" dot>
          {{ (row as SwarmNodeItem).availability }}
        </Badge>
      </template>
      <template #cell-role="{ row }">
        <Badge variant="blue">{{ (row as SwarmNodeItem).role }}</Badge>
      </template>
      <template #cell-actions="{ row }">
        <RowActions :actions="buildActions(row as SwarmNodeItem)" :visible="2" />
      </template>
    </DataTable>

    <!-- 节点详情 -->
    <Modal :open="detailOpen" @update:open="detailOpen = $event" title="节点详情" width="max-w-3xl">
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

    <!-- 加入令牌 -->
    <Modal :open="tokenOpen" @update:open="tokenOpen = $event" title="集群加入令牌" width="max-w-2xl">
      <div v-if="tokenLoading" class="py-8 text-center text-sm text-slate-400">加载中…</div>
      <div v-else-if="tokens" class="space-y-4">
        <p class="text-sm text-slate-500">
          在其它主机上执行 docker swarm join 以加入集群：
        </p>
        <div>
          <label class="mb-1.5 block text-sm text-slate-500">Worker 加入命令</label>
          <div class="flex items-center gap-2">
            <code class="flex-1 select-all overflow-x-auto rounded-lg bg-slate-100 p-2.5 text-xs text-slate-700 dark:bg-slate-900 dark:text-slate-300">
              docker swarm join --token {{ tokens.worker }} {{ tokens.addr || '&lt;manager-addr&gt;:2377' }}
            </code>
            <Button variant="secondary" size="sm" @click="copyToken(`docker swarm join --token ${tokens.worker} ${tokens.addr || '<manager-addr>:2377'}`, 'w')">
              <Check v-if="copied === 'w'" class="h-3.5 w-3.5" />
              <Copy v-else class="h-3.5 w-3.5" />
            </Button>
          </div>
        </div>
        <div>
          <label class="mb-1.5 block text-sm text-slate-500">Manager 加入命令</label>
          <div class="flex items-center gap-2">
            <code class="flex-1 select-all overflow-x-auto rounded-lg bg-slate-100 p-2.5 text-xs text-slate-700 dark:bg-slate-900 dark:text-slate-300">
              docker swarm join --token {{ tokens.manager }} {{ tokens.addr || '&lt;manager-addr&gt;:2377' }}
            </code>
            <Button variant="secondary" size="sm" @click="copyToken(`docker swarm join --token ${tokens.manager} ${tokens.addr || '<manager-addr>:2377'}`, 'm')">
              <Check v-if="copied === 'm'" class="h-3.5 w-3.5" />
              <Copy v-else class="h-3.5 w-3.5" />
            </Button>
          </div>
        </div>
      </div>
      <template #footer>
        <Button variant="secondary" @click="tokenOpen = false">关闭</Button>
      </template>
    </Modal>
  </div>
</template>
