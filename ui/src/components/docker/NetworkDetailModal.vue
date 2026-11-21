<script setup lang="ts">
import { ref, watch } from 'vue'
import Button from '@/components/ui/Button.vue'
import Modal from '@/components/ui/Modal.vue'
import { RefreshCw } from '@lucide/vue'
import { useConfirm } from '@/composables/useConfirm'
import { useToast } from '@/composables/useToast'
import {
  inspectNetwork,
  connectContainerToNetwork,
  disconnectContainerFromNetwork,
  type NetworkDetail,
} from '@/api/network'
import { listContainers, getContainerStats, type ContainerItem } from '@/api/container'
import { fmtSize, fmtISOTime, kvEntries } from '@/utils/format'
import Skeleton from '@/components/ui/Skeleton.vue'

const props = defineProps<{ open: boolean; networkId: string; networkName?: string }>()

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()

const loading = ref(false)
const errorMsg = ref('')
const detail = ref<NetworkDetail | null>(null)

// 各容器实时网络流量（收/发字节）。
const netStats = ref<Record<string, { rx: number; tx: number }>>({})
const trafficLoading = ref(false)

async function loadTraffic(): Promise<void> {
  const cs = detail.value?.containers
  if (!cs || cs.length === 0) return
  trafficLoading.value = true
  try {
    const entries = await Promise.all(
      cs.map(async (c) => {
        try {
          const s = await getContainerStats(c.id)
          return [c.id, { rx: s.net_rx_bytes || 0, tx: s.net_tx_bytes || 0 }] as const
        } catch {
          return [c.id, { rx: 0, tx: 0 }] as const
        }
      }),
    )
    netStats.value = Object.fromEntries(entries)
  } finally {
    trafficLoading.value = false
  }
}

// 连接容器。
const connectOpen = ref(false)
const containers = ref<ContainerItem[]>([])
const connectLoading = ref(false)
const selectedContainer = ref('')
const fixedIPv4 = ref('')
const connectSubmitting = ref(false)
const disconnectingId = ref('')

const confirm = useConfirm()
const toast = useToast()

watch(
  () => props.open,
  (open) => {
    if (open && props.networkId) {
      load()
    }
  },
  { immediate: true },
)

async function load(): Promise<void> {
  loading.value = true
  errorMsg.value = ''
  detail.value = null
  try {
    detail.value = await inspectNetwork(props.networkId)
    await loadTraffic()
  } catch (err) {
    errorMsg.value = (err as Error).message
    toast.error((err as Error).message)
  } finally {
    loading.value = false
  }
}

async function openConnect(): Promise<void> {
  connectOpen.value = true
  selectedContainer.value = ''
  fixedIPv4.value = ''
  connectLoading.value = true
  try {
    containers.value = await listContainers(true)
    // 过滤掉已连接该网络的容器。
    const connected = new Set(detail.value?.containers?.map((c) => c.id))
    containers.value = containers.value.filter((c) => !connected.has(c.id))
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    connectLoading.value = false
  }
}

async function doConnect(): Promise<void> {
  if (!selectedContainer.value) {
    toast.error('请选择容器')
    return
  }
  connectSubmitting.value = true
  try {
    await connectContainerToNetwork(props.networkId, selectedContainer.value, fixedIPv4.value.trim() || undefined)
    connectOpen.value = false
    toast.success('已连接到网络')
    await load()
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    connectSubmitting.value = false
  }
}

function openDisconnect(name: string, id: string): void {
  void confirm(
    '断开容器',
    `确认将容器「${name}」从网络断开？`,
    async () => {
      await disconnectContainerFromNetwork(props.networkId, id, false)
      toast.success(`已断开「${name}」`)
    },
    { danger: true, onSuccess: load },
  )
}

</script>

<template>
  <Modal :open="open" @update:open="(v) => emit('update:open', v)" title="网络详情" width="max-w-3xl">
    <div v-if="loading" class="space-y-4 py-6">
      <div class="grid grid-cols-2 gap-4">
        <Skeleton height="h-8" />
        <Skeleton height="h-8" />
      </div>
      <Skeleton :count="3" />
    </div>
    <div v-else-if="errorMsg" class="py-10 text-center">
      <p class="text-sm text-slate-400">加载失败，请关闭后重试</p>
    </div>
    <div v-else-if="detail" class="space-y-5">
      <!-- 基本信息 -->
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div>
          <label class="mb-1 block text-xs text-slate-500">名称</label>
          <div class="text-sm font-medium text-slate-800 dark:text-slate-100">{{ detail.name }}</div>
        </div>
        <div>
          <label class="mb-1 block text-xs text-slate-500">驱动</label>
          <div class="text-sm text-slate-800 dark:text-slate-100">{{ detail.driver }}</div>
        </div>
        <div>
          <label class="mb-1 block text-xs text-slate-500">网络 ID</label>
          <div class="truncate font-mono text-sm text-slate-800 dark:text-slate-100">{{ detail.id }}</div>
        </div>
        <div>
          <label class="mb-1 block text-xs text-slate-500">作用域</label>
          <div class="text-sm text-slate-800 dark:text-slate-100">{{ detail.scope }}</div>
        </div>
        <div>
          <label class="mb-1 block text-xs text-slate-500">属性</label>
          <div class="text-sm text-slate-800 dark:text-slate-100">
            {{ detail.internal ? '内置 ' : '' }}{{ detail.attachable ? '可挂载 ' : '' }}{{ detail.enable_ipv6 ? 'IPv6 ' : '' }}
          </div>
        </div>
        <div>
          <label class="mb-1 block text-xs text-slate-500">创建时间</label>
          <div class="text-sm text-slate-800 dark:text-slate-100">{{ fmtISOTime(detail.created) }}</div>
        </div>
      </div>

      <!-- IPAM -->
      <div v-if="detail.ipam?.length">
        <label class="mb-1 block text-xs text-slate-500">IPAM 配置</label>
        <div class="overflow-x-auto rounded-md border border-slate-200 bg-slate-50 dark:border-slate-700 dark:bg-slate-900">
          <table class="w-full font-mono text-xs text-slate-700 dark:text-slate-300">
            <thead>
              <tr class="border-b border-slate-200 text-left text-slate-400 dark:border-slate-700">
                <th class="px-3 py-1.5 font-normal">子网</th>
                <th class="px-3 py-1.5 font-normal">网关</th>
                <th class="px-3 py-1.5 font-normal">IP 范围</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(c, idx) in detail.ipam" :key="idx" class="border-b border-slate-200 last:border-b-0 dark:border-slate-700">
                <td class="px-3 py-1.5">{{ c.subnet || '-' }}</td>
                <td class="px-3 py-1.5">{{ c.gateway || '-' }}</td>
                <td class="px-3 py-1.5">{{ c.ip_range || '-' }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- 已连接容器 -->
      <div v-if="detail.containers?.length">
        <div class="mb-1 flex items-center justify-between">
          <label class="block text-xs text-slate-500">已连接容器 ({{ detail.containers.length }})</label>
          <Button variant="ghost" size="sm" :loading="trafficLoading" @click="loadTraffic">
            <RefreshCw class="mr-1 h-3 w-3" />刷新流量
          </Button>
        </div>
        <div class="overflow-x-auto rounded-md border border-slate-200 bg-slate-50 dark:border-slate-700 dark:bg-slate-900">
          <table class="w-full font-mono text-xs text-slate-700 dark:text-slate-300">
            <thead>
              <tr class="border-b border-slate-200 text-left text-slate-400 dark:border-slate-700">
                <th class="px-3 py-1.5 font-normal">名称</th>
                <th class="px-3 py-1.5 font-normal">IPv4</th>
                <th class="px-3 py-1.5 font-normal">MAC</th>
                <th class="px-3 py-1.5 font-normal">接收</th>
                <th class="px-3 py-1.5 font-normal">发送</th>
                <th class="px-3 py-1.5 font-normal">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="c in detail.containers"
                :key="c.name"
                class="border-b border-slate-200 last:border-b-0 dark:border-slate-700"
              >
                <td class="px-3 py-1.5">{{ c.name }}</td>
                <td class="px-3 py-1.5">{{ c.ipv4_address || '-' }}</td>
                <td class="px-3 py-1.5">{{ c.mac_address || '-' }}</td>
                <td class="px-3 py-1.5 text-green-600 dark:text-green-400">↓ {{ fmtSize(netStats[c.id]?.rx ?? 0) }}</td>
                <td class="px-3 py-1.5 text-blue-600 dark:text-blue-400">↑ {{ fmtSize(netStats[c.id]?.tx ?? 0) }}</td>
                <td class="px-3 py-1.5">
                  <Button
                    variant="ghost"
                    size="sm"
                    class="text-red-600!"
                    :loading="disconnectingId === c.id"
                    @click="openDisconnect(c.name, c.id)"
                  >
                    断开
                  </Button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
      <div v-else class="flex items-center justify-between rounded-md border border-dashed border-slate-300 px-3 py-2 dark:border-slate-700">
        <span class="text-xs text-slate-400">暂无容器连接</span>
        <Button variant="secondary" size="sm" @click="openConnect">连接容器</Button>
      </div>
      <div v-if="detail.containers?.length" class="flex justify-end">
        <Button variant="secondary" size="sm" @click="openConnect">连接容器</Button>
      </div>

      <!-- 连接容器 modal -->
      <Modal
        :open="connectOpen"
        @update:open="(v) => { connectOpen = v }"
        title="连接容器到网络"
        width="max-w-md"
      >
        <div class="space-y-3">
          <div>
            <label class="mb-1.5 block text-sm text-slate-500">选择容器</label>
            <select v-model="selectedContainer" class="input">
              <option value="" disabled>{{ connectLoading ? '加载中...' : '请选择容器' }}</option>
              <option v-for="c in containers" :key="c.id" :value="c.id">
                {{ c.names[0] || c.id.slice(0, 12) }}（{{ c.state }}）
              </option>
            </select>
          </div>
          <div>
            <label class="mb-1.5 block text-sm text-slate-500">固定 IPv4（可选）</label>
            <input v-model="fixedIPv4" class="input font-mono" placeholder="如 172.20.0.10" />
          </div>
        </div>
        <template #footer>
          <Button variant="secondary" @click="connectOpen = false">取消</Button>
          <Button :loading="connectSubmitting" @click="doConnect">连接</Button>
        </template>
      </Modal>

      <!-- 标签 -->
      <div v-if="kvEntries(detail.labels).length">
        <label class="mb-1 block text-xs text-slate-500">标签 ({{ kvEntries(detail.labels).length }})</label>
        <div class="max-h-32 overflow-y-auto rounded-md bg-slate-100 px-3 py-2 font-mono text-xs text-slate-700 dark:bg-slate-700 dark:text-slate-200">
          <div v-for="[k, v] in kvEntries(detail.labels)" :key="k" class="break-all">
            <span class="text-slate-400">{{ k }}</span>: {{ v }}
          </div>
        </div>
      </div>
    </div>
  </Modal>
</template>
