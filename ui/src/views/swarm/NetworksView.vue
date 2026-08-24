<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { Plus, Eye, Trash2, RefreshCw } from '@lucide/vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import Modal from '@/components/ui/Modal.vue'
import DataTable, { type DataTableColumn } from '@/components/ui/DataTable.vue'
import RowActions, { type RowAction } from '@/components/ui/RowActions.vue'
import { useToast } from '@/composables/useToast'
import { useConfirm } from '@/composables/useConfirm'
import {
  swarmNetworks,
  swarmCreateNetwork,
  swarmNetworkInspect,
  swarmRemoveNetwork,
  type SwarmNetworkItem,
} from '@/api/swarm'

const toast = useToast()
const confirm = useConfirm()

const loading = ref(false)
const errorMsg = ref('')
const items = ref<SwarmNetworkItem[]>([])

// 过滤：只显示 swarm 范围的网络。
const filteredItems = computed(() => items.value.filter((n) => n.scope === 'swarm'))

const createOpen = ref(false)
const createKind = ref<'overlay' | 'bridge'>('overlay')
const form = reactive({
  name: '',
  subnet: '',
  gateway: '',
  attachable: true,
  internal: false,
})
const saving = ref(false)

const detailOpen = ref(false)
const detail = ref('')
const detailLoading = ref(false)

async function load(): Promise<void> {
  loading.value = true
  errorMsg.value = ''
  try {
    items.value = await swarmNetworks()
  } catch (err) {
    errorMsg.value = (err as Error).message
    items.value = []
  } finally {
    loading.value = false
  }
}
onMounted(load)

function openCreate(kind: 'overlay' | 'bridge'): void {
  createKind.value = kind
  form.name = ''
  form.subnet = ''
  form.gateway = ''
  form.attachable = true
  form.internal = false
  createOpen.value = true
}

async function submit(): Promise<void> {
  if (!form.name) {
    toast.error('请输入网络名称')
    return
  }
  saving.value = true
  try {
    await swarmCreateNetwork({
      name: form.name,
      driver: createKind.value,
      subnet: form.subnet || undefined,
      gateway: form.gateway || undefined,
      attachable: form.attachable,
      internal: form.internal,
    })
    toast.success(`网络「${form.name}」已创建`)
    createOpen.value = false
    await load()
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    saving.value = false
  }
}

async function openDetail(row: SwarmNetworkItem): Promise<void> {
  detailOpen.value = true
  detail.value = ''
  detailLoading.value = true
  try {
    const raw = await swarmNetworkInspect(row.id)
    detail.value = JSON.stringify(raw, null, 2)
  } catch (err) {
    detail.value = `加载失败: ${(err as Error).message}`
  } finally {
    detailLoading.value = false
  }
}

function removeNetwork(row: SwarmNetworkItem): void {
  void confirm(
    '删除网络',
    `确认删除网络「${row.name}」？若被服务引用将导致任务失败。`,
    async () => {
      await swarmRemoveNetwork(row.id)
      toast.success(`已删除网络「${row.name}」`)
      await load()
    },
    { danger: true },
  )
}

function buildActions(row: SwarmNetworkItem): RowAction[] {
  return [
    { key: 'detail', label: '详情', icon: Eye, onClick: () => openDetail(row) },
    { key: 'remove', label: '删除', icon: Trash2, danger: true, onClick: () => removeNetwork(row) },
  ]
}

const columns: DataTableColumn[] = [
  { label: '名称', key: 'name' },
  { label: '驱动', key: 'driver', width: '100px' },
  { label: '作用域', key: 'scope', width: '100px' },
  { label: '可连接', key: 'attachable', width: '90px' },
  { label: '操作', key: 'actions', width: '160px', align: 'right' },
]
</script>

<template>
  <div>
    <DataTable
      title="Swarm 网络"
      :columns="columns"
      :data="filteredItems"
      :loading="loading"
      :error="errorMsg"
      row-key="id"
      empty-text="暂无 Swarm 网络"
    >
      <template #toolbar>
        <Button variant="secondary" size="sm" :loading="loading" @click="load">
          <RefreshCw class="h-3.5 w-3.5" />
          刷新
        </Button>
        <Button size="sm" @click="openCreate('overlay')">
          <Plus class="h-3.5 w-3.5" />
          新建 Overlay
        </Button>
      </template>
      <template #cell-driver="{ row }">
        <Badge variant="blue">{{ (row as SwarmNetworkItem).driver }}</Badge>
      </template>
      <template #cell-scope="{ row }">
        <Badge variant="purple" dot>{{ (row as SwarmNetworkItem).scope }}</Badge>
      </template>
      <template #cell-attachable="{ row }">
        <Badge :variant="(row as SwarmNetworkItem).attachable ? 'green' : 'gray'" dot>
          {{ (row as SwarmNetworkItem).attachable ? '是' : '否' }}
        </Badge>
      </template>
      <template #cell-actions="{ row }">
        <RowActions :actions="buildActions(row as SwarmNetworkItem)" :visible="2" />
      </template>
    </DataTable>

    <!-- 创建网络 -->
    <Modal :open="createOpen" @update:open="createOpen = $event" title="新建 Swarm 网络" width="max-w-lg">
      <div class="space-y-4">
        <div>
          <label class="mb-1.5 block text-sm text-slate-500">类型</label>
          <select v-model="createKind" class="input">
            <option value="overlay">overlay（Swarm 专用）</option>
            <option value="bridge">bridge（本地）</option>
          </select>
        </div>
        <div>
          <label class="mb-1.5 block text-sm text-slate-500">名称 *</label>
          <input v-model="form.name" class="input font-mono" placeholder="例如 app-net" />
        </div>
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="mb-1.5 block text-sm text-slate-500">子网（可选）</label>
            <input v-model="form.subnet" class="input font-mono" placeholder="例如 10.0.1.0/24" />
          </div>
          <div>
            <label class="mb-1.5 block text-sm text-slate-500">网关（可选）</label>
            <input v-model="form.gateway" class="input font-mono" placeholder="例如 10.0.1.1" />
          </div>
        </div>
        <div class="flex gap-6">
          <label class="flex cursor-pointer items-center gap-2 text-sm text-slate-600 dark:text-slate-300">
            <input v-model="form.attachable" type="checkbox" class="h-4 w-4 rounded border-slate-300 text-blue-600" />
            允许独立容器连接
          </label>
          <label class="flex cursor-pointer items-center gap-2 text-sm text-slate-600 dark:text-slate-300">
            <input v-model="form.internal" type="checkbox" class="h-4 w-4 rounded border-slate-300 text-blue-600" />
            内部网络（禁止外部访问）
          </label>
        </div>
      </div>
      <template #footer>
        <Button variant="secondary" @click="createOpen = false">取消</Button>
        <Button :loading="saving" @click="submit">创建</Button>
      </template>
    </Modal>

    <!-- 网络详情 -->
    <Modal :open="detailOpen" @update:open="detailOpen = $event" title="网络详情" width="max-w-3xl">
      <pre
        class="max-h-[70vh] overflow-auto rounded-lg bg-slate-50 p-4 text-xs leading-relaxed text-slate-700 dark:bg-slate-900 dark:text-slate-300"
      >{{ detailLoading ? '加载中…' : detail }}</pre>
    </Modal>
  </div>
</template>
