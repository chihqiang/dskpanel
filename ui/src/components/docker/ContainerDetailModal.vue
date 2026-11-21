<script setup lang="ts">
import { ref, watch } from 'vue'
import { Braces, Pencil, X, Check, Unplug } from '@lucide/vue'
import Button from '@/components/ui/Button.vue'
import Modal from '@/components/ui/Modal.vue'
import Badge from '@/components/ui/Badge.vue'
import { useToast } from '@/composables/useToast'
import { useConfirm } from '@/composables/useConfirm'
import {
  inspectContainer,
  inspectContainerRaw,
  updateContainer,
  renameContainer,
  type ContainerDetail,
} from '@/api/container'
import { disconnectContainerFromNetwork } from '@/api/network'
import Skeleton from '@/components/ui/Skeleton.vue'

const props = defineProps<{ open: boolean; containerId: string; containerName?: string }>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  updated: []
}>()

const loading = ref(false)
const errorMsg = ref('')
const detail = ref<ContainerDetail | null>(null)
const toast = useToast()
const confirm = useConfirm()

// 断开网络。
const disconnectingNet = ref('')

// 完整 inspect 原始 JSON。
const rawOpen = ref(false)
const rawJson = ref('')
const rawLoading = ref(false)
const rawError = ref('')

// ---- 编辑模式（内嵌整合：重命名 + 资源限制 + 重启策略）----
const editing = ref(false)
const saving = ref(false)
const newName = ref('')
const memory = ref('') // MB
const cpuShares = ref('')
const cpus = ref('') // 小数 CPU 核数
const cpuset = ref('')
const restartPolicy = ref('')
const restartMax = ref('')

watch(
  () => props.open,
  (open) => {
    if (open && props.containerId) {
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
    detail.value = await inspectContainer(props.containerId)
  } catch (err) {
    errorMsg.value = (err as Error).message
    toast.error((err as Error).message)
  } finally {
    loading.value = false
  }
}

/** 进入编辑模式：预填当前值。 */
function startEdit(): void {
  const d = detail.value
  if (!d) return
  newName.value = d.name.replace(/^\//, '')
  const hc = d.host_config
  memory.value = hc?.memory ? String(Math.round(hc.memory / 1024 / 1024)) : ''
  cpuShares.value = hc?.cpu_shares ? String(hc.cpu_shares) : ''
  cpus.value = hc?.nano_cpus ? String(hc.nano_cpus / 1e9) : ''
  cpuset.value = hc?.cpuset_cpus ?? ''
  restartPolicy.value = hc?.restart_policy ?? ''
  restartMax.value = hc?.restart_max ? String(hc.restart_max) : ''
  editing.value = true
}

function cancelEdit(): void {
  editing.value = false
}

/** number input 可能返回 number；统一转字符串去空白。 */
function toTrim(v: string | number): string {
  return String(v ?? '').trim()
}

/** 内存 MB → 字节。 */
function mbToBytes(mb: string): number {
  const v = Number(mb)
  return Number.isFinite(v) && v > 0 ? Math.round(v * 1024 * 1024) : 0
}

/** 小数 CPU 核数 → NanoCPUs。 */
function coresToNano(cores: string): number {
  const v = Number(cores)
  return Number.isFinite(v) && v > 0 ? Math.round(v * 1e9) : 0
}

/** 保存编辑：重命名 + 更新资源/重启策略。 */
async function saveEdit(): Promise<void> {
  if (!props.containerId) return
  saving.value = true
  try {
    // 1. 重命名（名称变化才调用）。
    const targetName = newName.value.trim().replace(/^\//, '')
    const currentName = (detail.value?.name ?? '').replace(/^\//, '')
    if (targetName && targetName !== currentName) {
      await renameContainer(props.containerId, targetName)
      toast.success(`已重命名为「${targetName}」`)
    }
    // 2. 更新资源/重启策略（任意字段有值才调用；留空表示不改）。
    const hasUpdates =
      toTrim(memory.value) ||
      toTrim(cpuShares.value) ||
      toTrim(cpus.value) ||
      toTrim(cpuset.value) ||
      restartPolicy.value
    if (hasUpdates) {
      await updateContainer(props.containerId, {
        memory: toTrim(memory.value) ? mbToBytes(toTrim(memory.value)) : undefined,
        cpu_shares: toTrim(cpuShares.value) ? Number(toTrim(cpuShares.value)) : undefined,
        nano_cpus: toTrim(cpus.value) ? coresToNano(toTrim(cpus.value)) : undefined,
        cpuset_cpus: toTrim(cpuset.value) || undefined,
        restart_policy: restartPolicy.value || undefined,
        restart_max: toTrim(restartMax.value) ? Number(toTrim(restartMax.value)) : undefined,
      })
      toast.success('已更新容器配置')
    }
    editing.value = false
    emit('updated')
    // 刷新详情显示最新值。
    await load()
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    saving.value = false
  }
}

async function toggleRaw(): Promise<void> {
  if (rawOpen.value) {
    rawOpen.value = false
    return
  }
  rawOpen.value = true
  rawLoading.value = true
  rawError.value = ''
  try {
    const data = await inspectContainerRaw(props.containerId)
    rawJson.value = JSON.stringify(data, null, 2)
  } catch (err) {
    rawError.value = (err as Error).message
    toast.error((err as Error).message)
    rawJson.value = ''
  } finally {
    rawLoading.value = false
  }
}

function fmtPorts(ports?: Record<string, { private_port: number; public_port?: number; type: string }[]>): string {
  if (!ports || Object.keys(ports).length === 0) return '-'
  return Object.entries(ports)
    .map(([k, v]) => {
      const first = v[0]
      return first?.public_port ? `${first.public_port}->${k}` : k
    })
    .join(', ')
}

function openDisconnectNetwork(netName: string): void {
  void confirm(
    '断开网络',
    `确认将容器从网络「${netName}」断开？`,
    async () => {
      disconnectingNet.value = netName
      await disconnectContainerFromNetwork(netName, props.containerId, false)
      toast.success(`已从网络「${netName}」断开`)
      await load()
    },
    { danger: true, onSuccess: () => { disconnectingNet.value = '' } },
  )
}
</script>

<template>
  <Modal :open="open" @update:open="(v) => emit('update:open', v)" title="容器详情" width="max-w-3xl">
    <div v-if="loading" class="space-y-4 py-6">
      <div class="grid grid-cols-2 gap-4">
        <Skeleton height="h-8" />
        <Skeleton height="h-8" />
      </div>
      <Skeleton :count="4" />
    </div>
    <div v-else-if="errorMsg" class="py-10 text-center">
      <p class="text-sm text-slate-400">加载失败，请关闭后重试</p>
    </div>
    <div v-else-if="detail" class="space-y-4">
      <!-- ===== 编辑模式：重命名 + 资源限制 + 重启策略 ===== -->
      <div v-if="editing" class="space-y-4">
        <div class="flex items-center gap-2 rounded-lg bg-blue-50 px-3 py-2 text-sm text-blue-700 dark:bg-blue-900/30 dark:text-blue-300">
          <Pencil class="h-4 w-4" />编辑容器配置
          <span class="text-xs text-blue-500 dark:text-blue-400">（资源限制与重启策略随时可修改，重命名亦可）</span>
        </div>
        <div>
          <label class="mb-1 block text-xs text-slate-500">容器名称</label>
          <input v-model="newName" class="input font-mono" placeholder="my-container" />
        </div>
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="mb-1 block text-xs text-slate-500">内存限制 (MB)</label>
            <input v-model="memory" type="number" min="0" class="input" placeholder="如 512" />
          </div>
          <div>
            <label class="mb-1 block text-xs text-slate-500">CPU 核数</label>
            <input v-model="cpus" type="number" min="0" step="0.1" class="input" placeholder="如 0.5 / 1.5" />
          </div>
          <div>
            <label class="mb-1 block text-xs text-slate-500">CPU 份额</label>
            <input v-model="cpuShares" type="number" min="0" class="input" placeholder="如 512（相对权重）" />
          </div>
          <div>
            <label class="mb-1 block text-xs text-slate-500">CPU 亲和 (cpuset)</label>
            <input v-model="cpuset" class="input font-mono" placeholder="如 0-1 / 0,2" />
          </div>
          <div>
            <label class="mb-1 block text-xs text-slate-500">重启策略</label>
            <select v-model="restartPolicy" class="input">
              <option value="">不修改</option>
              <option value="no">no</option>
              <option value="always">always</option>
              <option value="on-failure">on-failure</option>
              <option value="unless-stopped">unless-stopped</option>
            </select>
          </div>
          <div>
            <label class="mb-1 block text-xs text-slate-500">最大重试次数 (on-failure)</label>
            <input v-model="restartMax" type="number" min="0" class="input" placeholder="如 5" />
          </div>
        </div>
      </div>

      <!-- ===== 只读详情 ===== -->
      <template v-else>
        <!-- 基本信息 -->
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label class="mb-1 block text-xs text-slate-500">名称</label>
            <div class="text-sm font-medium text-slate-800 dark:text-slate-100">
              {{ detail.name }}
              <Badge :variant="detail.state === 'running' ? 'green' : 'gray'" class="ml-1">{{ detail.state }}</Badge>
            </div>
          </div>
          <div>
            <label class="mb-1 block text-xs text-slate-500">镜像</label>
            <div class="truncate text-sm text-slate-800 dark:text-slate-100">{{ detail.image }}</div>
          </div>
          <div>
            <label class="mb-1 block text-xs text-slate-500">容器 ID</label>
            <div class="truncate font-mono text-sm text-slate-800 dark:text-slate-100">{{ detail.id }}</div>
          </div>
          <div>
            <label class="mb-1 block text-xs text-slate-500">状态 / 重启次数</label>
            <div class="text-sm text-slate-800 dark:text-slate-100">
              {{ detail.status }}<span v-if="detail.restart_count"> · 重启 {{ detail.restart_count }} 次</span>
            </div>
          </div>
          <div>
            <label class="mb-1 block text-xs text-slate-500">创建时间</label>
            <div class="text-sm text-slate-800 dark:text-slate-100">{{ detail.created }}</div>
          </div>
          <div>
            <label class="mb-1 block text-xs text-slate-500">端口映射</label>
            <div class="text-sm text-slate-800 dark:text-slate-100">{{ fmtPorts(detail.network_settings?.ports) }}</div>
          </div>
        </div>

        <!-- 网络 -->
        <div v-if="detail.network_settings?.networks">
          <label class="mb-1 block text-xs text-slate-500">网络</label>
          <div class="space-y-1.5">
            <div
              v-for="(v, name) in detail.network_settings.networks"
              :key="name"
              class="flex items-center gap-2"
            >
              <span class="rounded-md bg-slate-100 px-2 py-1 font-mono text-xs text-slate-700 dark:bg-slate-700 dark:text-slate-200">
                {{ name }}{{ v.ip_address ? ` (${v.ip_address})` : '' }}
              </span>
              <Button
                variant="ghost"
                size="sm"
                class="!text-red-600"
                :loading="disconnectingNet === name"
                @click="openDisconnectNetwork(name as string)"
              >
                <Unplug class="h-3 w-3" />
                断开
              </Button>
            </div>
          </div>
        </div>

        <!-- 命令 -->
        <div v-if="detail.config?.cmd?.length">
          <label class="mb-1 block text-xs text-slate-500">命令 (CMD)</label>
          <div class="rounded-md bg-slate-100 px-3 py-2 font-mono text-xs text-slate-700 dark:bg-slate-700 dark:text-slate-200">
            {{ detail.config.cmd.join(' ') }}
          </div>
        </div>

        <!-- 环境变量 -->
        <div v-if="detail.config?.env?.length">
          <label class="mb-1 block text-xs text-slate-500">环境变量 ({{ detail.config.env.length }})</label>
          <div class="max-h-40 overflow-y-auto rounded-md bg-slate-100 px-3 py-2 font-mono text-xs text-slate-700 dark:bg-slate-700 dark:text-slate-200">
            <div v-for="e in detail.config.env" :key="e">{{ e }}</div>
          </div>
        </div>

        <!-- 卷挂载 -->
        <div v-if="detail.host_config?.binds?.length">
          <label class="mb-1 block text-xs text-slate-500">卷挂载 ({{ detail.host_config.binds.length }})</label>
          <div class="rounded-md bg-slate-100 px-3 py-2 font-mono text-xs text-slate-700 dark:bg-slate-700 dark:text-slate-200">
            <div v-for="b in detail.host_config.binds" :key="b">{{ b }}</div>
          </div>
        </div>

        <!-- 标签 -->
        <div v-if="detail.config?.labels && Object.keys(detail.config.labels).length">
          <label class="mb-1 block text-xs text-slate-500">标签 ({{ Object.keys(detail.config.labels).length }})</label>
          <div class="max-h-32 overflow-y-auto rounded-md bg-slate-100 px-3 py-2 font-mono text-xs text-slate-700 dark:bg-slate-700 dark:text-slate-200">
            <div v-for="(v, k) in detail.config.labels" :key="k">{{ k }}={{ v }}</div>
          </div>
        </div>

        <!-- 完整 inspect 原始 JSON -->
        <div v-if="rawOpen" class="rounded-md border border-slate-200 dark:border-slate-700">
          <div class="flex items-center justify-between border-b border-slate-200 px-3 py-2 dark:border-slate-700">
            <span class="text-xs font-medium text-slate-500">完整 inspect (docker inspect)</span>
            <button class="text-xs text-slate-400 hover:text-slate-600 dark:hover:text-slate-200" @click="rawOpen = false">
              收起
            </button>
          </div>
          <p v-if="rawError" class="px-3 py-2 text-sm text-slate-400">加载失败</p>
          <pre
            v-else-if="rawJson"
            class="max-h-80 overflow-auto whitespace-pre-wrap break-all px-3 py-2 font-mono text-[11px] leading-relaxed text-slate-700 dark:text-slate-200"
          >{{ rawJson }}</pre>
          <Skeleton v-else :count="3" height="h-3" />
        </div>
      </template>
    </div>

    <template #footer>
      <template v-if="editing">
        <Button variant="secondary" :disabled="saving" @click="cancelEdit">
          <X class="mr-1 h-3.5 w-3.5" />取消
        </Button>
        <Button :loading="saving" @click="saveEdit">
          <Check class="mr-1 h-3.5 w-3.5" />保存
        </Button>
      </template>
      <template v-else>
        <Button variant="secondary" :loading="rawLoading" @click="toggleRaw">
          <Braces class="mr-1 h-3.5 w-3.5" />{{ rawOpen ? '收起 JSON' : '完整 JSON' }}
        </Button>
        <Button variant="secondary" @click="startEdit">
          <Pencil class="mr-1 h-3.5 w-3.5" />编辑
        </Button>
        <Button variant="secondary" @click="emit('update:open', false)">关闭</Button>
      </template>
    </template>
  </Modal>
</template>
