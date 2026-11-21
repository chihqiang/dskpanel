<script setup lang="ts">
import { ref, watch } from 'vue'
import Badge from '@/components/ui/Badge.vue'
import Modal from '@/components/ui/Modal.vue'
import DataTable, { type DataTableColumn } from '@/components/ui/DataTable.vue'
import { useToast } from '@/composables/useToast'
import { composeProjectPs, type ComposeProjectItem, type ComposeProjectDetail, type ComposeContainerStatus } from '@/api/compose'

const props = defineProps<{
  open: boolean
  project: ComposeProjectItem | null
}>()

const emit = defineEmits<{ 'update:open': [value: boolean] }>()

const toast = useToast()

const detailLoading = ref(false)
const detail = ref<ComposeProjectDetail | null>(null)

/** 容器状态 → Badge 变体。 */
function stateVariant(state: string): 'green' | 'red' | 'yellow' | 'gray' {
  if (state === 'running') return 'green'
  if (state === 'exited' || state === 'dead') return 'red'
  if (state === 'paused') return 'yellow'
  return 'gray'
}

const detailColumns: DataTableColumn[] = [
  { key: 'service', label: '服务' },
  { key: 'name', label: '容器' },
  { key: 'state', label: '状态' },
  { key: 'image', label: '镜像', ellipsis: true },
  { key: 'ports', label: '端口' },
]

async function fetchDetail(name: string): Promise<void> {
  detail.value = null
  detailLoading.value = true
  try {
    detail.value = await composeProjectPs(name)
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    detailLoading.value = false
  }
}

watch(
  () => [props.open, props.project] as const,
  ([open, project]) => {
    if (open && project) {
      void fetchDetail(project.name)
    }
  },
  { immediate: true },
)
</script>

<template>
  <Modal :open="open" @update:open="emit('update:open', $event)" title="项目详情" width="max-w-4xl">
    <div v-if="detail" class="space-y-4">
      <div class="flex flex-wrap items-center gap-3">
        <span class="text-lg font-semibold text-slate-900 dark:text-slate-100">{{ detail.name }}</span>
        <Badge :variant="detail.running > 0 ? 'green' : 'gray'" dot>{{ detail.running }}/{{ detail.total }} 运行中</Badge>
        <Badge variant="blue">{{ detail.services }} 个服务</Badge>
      </div>
      <DataTable
        :columns="detailColumns"
        :data="detail.containers"
        :loading="detailLoading"
        row-key="id"
        empty-text="暂无容器"
      >
        <template #cell-name="{ row }">
          <span class="font-mono text-xs">{{ (row as ComposeContainerStatus).name }}</span>
        </template>
        <template #cell-state="{ row }">
          <Badge :variant="stateVariant((row as ComposeContainerStatus).state)" dot>
            {{ (row as ComposeContainerStatus).state }}
          </Badge>
        </template>
        <template #cell-image="{ row }">
          <span class="block truncate font-mono text-xs">{{ (row as ComposeContainerStatus).image }}</span>
        </template>
        <template #cell-ports="{ row }">
          <template v-if="(row as ComposeContainerStatus).ports.length">
            <span
              v-for="p in (row as ComposeContainerStatus).ports"
              :key="p"
              class="mr-1 inline-block rounded bg-slate-100 px-1.5 py-0.5 font-mono text-xs dark:bg-slate-700"
            >
              {{ p }}
            </span>
          </template>
          <span v-else class="text-slate-400">-</span>
        </template>
      </DataTable>
    </div>
    <div v-else-if="detailLoading" class="py-8 text-center text-sm text-slate-400">加载中...</div>
  </Modal>
</template>
