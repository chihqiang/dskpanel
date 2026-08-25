<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Trash2, KeyRound, FileCode2, RefreshCw, Eye } from '@lucide/vue'
import Button from '@/components/ui/Button.vue'
import DataTable, { type DataTableColumn } from '@/components/ui/DataTable.vue'
import RowActions, { type RowAction } from '@/components/ui/RowActions.vue'
import { SecretCreateModal, SecretDetailModal } from '@/components/swarm'
import { useToast } from '@/composables/useToast'
import { useConfirm } from '@/composables/useConfirm'
import {
  swarmSecrets,
  swarmRemoveSecret,
  swarmConfigs,
  swarmRemoveConfig,
  type SwarmSecretItem,
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

// ---- 详情弹窗 ----
const detailOpen = ref(false)
const detailKind = ref<'secret' | 'config'>('secret')
const detailRow = ref<SwarmSecretItem | null>(null)

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
  createOpen.value = true
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

function openDetail(row: SwarmSecretItem, kind: 'secret' | 'config'): void {
  detailKind.value = kind
  detailRow.value = row
  detailOpen.value = true
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
    <SecretCreateModal v-model:open="createOpen" :kind="createKind" @saved="loadAll" />

    <!-- 详情弹窗 -->
    <SecretDetailModal v-model:open="detailOpen" :row="detailRow" :kind="detailKind" />
  </div>
</template>
