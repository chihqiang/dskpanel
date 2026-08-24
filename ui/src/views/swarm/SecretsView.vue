<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { Trash2, KeyRound, FileCode2, RefreshCw, Eye } from '@lucide/vue'
import Button from '@/components/ui/Button.vue'
import Modal from '@/components/ui/Modal.vue'
import DataTable, { type DataTableColumn } from '@/components/ui/DataTable.vue'
import RowActions, { type RowAction } from '@/components/ui/RowActions.vue'
import { useToast } from '@/composables/useToast'
import { useConfirm } from '@/composables/useConfirm'
import {
  swarmSecrets,
  swarmCreateSecret,
  swarmRemoveSecret,
  swarmSecretInspect,
  swarmConfigs,
  swarmCreateConfig,
  swarmRemoveConfig,
  swarmConfigInspect,
  type SwarmSecretItem,
  type SwarmSecretDetail,
} from '@/api/swarm'

const toast = useToast()
const confirm = useConfirm()

// ---- Secret ----
const secrets = ref<SwarmSecretItem[]>([])
const secretsLoading = ref(false)
const secretsError = ref('')

// ---- Config ----
const configs = ref<SwarmSecretItem[]>([])
const configsLoading = ref(false)
const configsError = ref('')

// ---- 创建弹窗 ----
const createOpen = ref(false)
const createKind = ref<'secret' | 'config'>('secret')
const form = reactive({ name: '', data: '' })
const saving = ref(false)

// ---- 详情弹窗 ----
const detailOpen = ref(false)
const detailKind = ref<'secret' | 'config'>('secret')
const detailData = ref<SwarmSecretDetail | null>(null)
const detailLoading = ref(false)

async function openDetail(row: SwarmSecretItem, kind: 'secret' | 'config'): Promise<void> {
  detailKind.value = kind
  detailData.value = null
  detailOpen.value = true
  detailLoading.value = true
  try {
    detailData.value = kind === 'secret'
      ? await swarmSecretInspect(row.id)
      : await swarmConfigInspect(row.id)
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    detailLoading.value = false
  }
}

async function loadSecrets(): Promise<void> {
  secretsLoading.value = true
  secretsError.value = ''
  try {
    secrets.value = await swarmSecrets()
  } catch (err) {
    secretsError.value = (err as Error).message
    secrets.value = []
  } finally {
    secretsLoading.value = false
  }
}

async function loadConfigs(): Promise<void> {
  configsLoading.value = true
  configsError.value = ''
  try {
    configs.value = await swarmConfigs()
  } catch (err) {
    configsError.value = (err as Error).message
    configs.value = []
  } finally {
    configsLoading.value = false
  }
}

function loadAll(): void {
  void loadSecrets()
  void loadConfigs()
}
onMounted(loadAll)

function openCreate(kind: 'secret' | 'config'): void {
  createKind.value = kind
  form.name = ''
  form.data = ''
  createOpen.value = true
}

async function submit(): Promise<void> {
  if (!form.name) {
    toast.error('请输入名称')
    return
  }
  saving.value = true
  try {
    if (createKind.value === 'secret') {
      await swarmCreateSecret(form.name, form.data)
      toast.success(`Secret「${form.name}」已创建`)
    } else {
      await swarmCreateConfig(form.name, form.data)
      toast.success(`Config「${form.name}」已创建`)
    }
    createOpen.value = false
    await loadAll()
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    saving.value = false
  }
}

function removeSecret(row: SwarmSecretItem): void {
  void confirm(
    '删除 Secret',
    `确认删除 Secret「${row.name}」？此操作不可恢复。`,
    async () => {
      await swarmRemoveSecret(row.id)
      toast.success(`已删除 Secret「${row.name}」`)
      await loadSecrets()
    },
    { danger: true },
  )
}

function removeConfig(row: SwarmSecretItem): void {
  void confirm(
    '删除 Config',
    `确认删除 Config「${row.name}」？此操作不可恢复。`,
    async () => {
      await swarmRemoveConfig(row.id)
      toast.success(`已删除 Config「${row.name}」`)
      await loadConfigs()
    },
    { danger: true },
  )
}

function buildSecretActions(row: SwarmSecretItem): RowAction[] {
  return [
    { key: 'detail', label: '详情', icon: Eye, onClick: () => openDetail(row, 'secret') },
    { key: 'remove', label: '删除', icon: Trash2, danger: true, onClick: () => removeSecret(row) },
  ]
}

function buildConfigActions(row: SwarmSecretItem): RowAction[] {
  return [
    { key: 'detail', label: '详情', icon: Eye, onClick: () => openDetail(row, 'config') },
    { key: 'remove', label: '删除', icon: Trash2, danger: true, onClick: () => removeConfig(row) },
  ]
}

const columns: DataTableColumn[] = [
  { label: '名称', key: 'name' },
  { label: '创建时间', key: 'created_at', width: '160px' },
  { label: '操作', key: 'actions', width: '160px', align: 'right' },
]
</script>

<template>
  <div class="space-y-5">
    <!-- Secret -->
    <DataTable
      title="Secret"
      :columns="columns"
      :data="secrets"
      :loading="secretsLoading"
      :error="secretsError"
      row-key="id"
      empty-text="暂无 Secret"
    >
      <template #toolbar>
        <Button variant="secondary" size="sm" :loading="secretsLoading" @click="loadSecrets">
          <RefreshCw class="h-3.5 w-3.5" />
          刷新
        </Button>
        <Button size="sm" @click="openCreate('secret')">
          <KeyRound class="h-3.5 w-3.5" />
          新建 Secret
        </Button>
      </template>
      <template #cell-name="{ row }">
        <span class="font-mono text-sm">{{ (row as SwarmSecretItem).name }}</span>
      </template>
      <template #cell-actions="{ row }">
        <RowActions :actions="buildSecretActions(row as SwarmSecretItem)" :visible="1" />
      </template>
    </DataTable>

    <!-- Config -->
    <DataTable
      title="Config"
      :columns="columns"
      :data="configs"
      :loading="configsLoading"
      :error="configsError"
      row-key="id"
      empty-text="暂无 Config"
    >
      <template #toolbar>
        <Button variant="secondary" size="sm" :loading="configsLoading" @click="loadConfigs">
          <RefreshCw class="h-3.5 w-3.5" />
          刷新
        </Button>
        <Button size="sm" @click="openCreate('config')">
          <FileCode2 class="h-3.5 w-3.5" />
          新建 Config
        </Button>
      </template>
      <template #cell-name="{ row }">
        <span class="font-mono text-sm">{{ (row as SwarmSecretItem).name }}</span>
      </template>
      <template #cell-actions="{ row }">
        <RowActions :actions="buildConfigActions(row as SwarmSecretItem)" :visible="1" />
      </template>
    </DataTable>

    <!-- 创建弹窗 -->
    <Modal
      :open="createOpen"
      @update:open="createOpen = $event"
      :title="createKind === 'secret' ? '新建 Secret' : '新建 Config'"
      width="max-w-lg"
    >
      <div class="space-y-4">
        <div>
          <label class="mb-1.5 block text-sm text-slate-500">名称 *</label>
          <input v-model="form.name" class="input font-mono" placeholder="例如 db-pass" />
        </div>
        <div>
          <label class="mb-1.5 block text-sm text-slate-500">
            {{ createKind === 'secret' ? '数据（敏感内容）' : '数据（配置文件内容）' }}
          </label>
          <textarea v-model="form.data" class="input h-32 font-mono" placeholder="输入内容…" />
        </div>
      </div>
      <template #footer>
        <Button variant="secondary" @click="createOpen = false">取消</Button>
        <Button :loading="saving" @click="submit">创建</Button>
      </template>
    </Modal>

    <!-- 详情弹窗（解密展示） -->
    <Modal
      :open="detailOpen"
      @update:open="detailOpen = $event"
      :title="`${detailKind === 'secret' ? 'Secret' : 'Config'} 详情 - ${detailData?.name ?? ''}`"
      width="max-w-2xl"
    >
      <div v-if="detailLoading" class="py-8 text-center text-sm text-slate-400">加载中…</div>
      <div v-else-if="detailData" class="space-y-4">
        <div class="grid grid-cols-2 gap-4">
          <div>
            <p class="text-xs text-slate-400">名称</p>
            <p class="font-mono text-sm text-slate-700 dark:text-slate-200">{{ detailData.name }}</p>
          </div>
          <div>
            <p class="text-xs text-slate-400">创建时间</p>
            <p class="text-sm text-slate-700 dark:text-slate-200">{{ detailData.created_at }}</p>
          </div>
        </div>
        <div>
          <p class="mb-1.5 text-sm text-slate-500">内容</p>
          <pre
            v-if="detailKind === 'config'"
            class="max-h-72 overflow-auto rounded-lg bg-slate-50 p-3 font-mono text-xs leading-relaxed text-slate-700 dark:bg-slate-900 dark:text-slate-300"
          >{{ detailData.data }}</pre>
          <div v-else class="rounded-lg bg-slate-50 p-4 text-center text-sm text-slate-400 dark:bg-slate-900">
            Secret 内容由 Docker 安全存储，API 不提供明文查看
          </div>
        </div>
        <div v-if="detailData.labels && Object.keys(detailData.labels).length">
          <p class="mb-1.5 text-xs text-slate-400">标签</p>
          <div class="flex flex-wrap gap-1.5">
            <span v-for="(v, k) in detailData.labels" :key="k" class="rounded-md bg-slate-100 px-2 py-0.5 text-xs text-slate-600 dark:bg-slate-700 dark:text-slate-300">{{ k }}={{ v }}</span>
          </div>
        </div>
      </div>
      <template #footer>
        <Button variant="secondary" @click="detailOpen = false">关闭</Button>
      </template>
    </Modal>
  </div>
</template>
