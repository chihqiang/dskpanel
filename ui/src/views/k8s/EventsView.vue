<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { RefreshCw, AlertTriangle, Info } from '@lucide/vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import DataTable, { type DataTableColumn } from '@/components/ui/DataTable.vue'
import { ResourceToolbar } from '@/components/k8s'
import { useNamespaces } from '@/composables/useNamespaces'
import { eventTypeVariant } from '@/utils/k8s'
import { k8sEvents, type K8sEventItem } from '@/api/k8s'

const { current: namespace, loadNamespaces } = useNamespaces()

const loading = ref(false)
const errorMsg = ref('')
const items = ref<K8sEventItem[]>([])

// 关键词搜索（匹配对象 / 原因 / 消息）。
const keyword = ref('')

/** 按关键词过滤。 */
const filtered = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  if (!kw) return items.value
  return items.value.filter(
    (e) =>
      e.object.toLowerCase().includes(kw) ||
      e.reason.toLowerCase().includes(kw) ||
      e.message.toLowerCase().includes(kw),
  )
})

async function load(): Promise<void> {
  loading.value = true
  errorMsg.value = ''
  try {
    items.value = await k8sEvents(namespace.value)
  } catch (err) {
    errorMsg.value = (err as Error).message
    items.value = []
  } finally {
    loading.value = false
  }
}

watch(namespace, () => void load())

onMounted(async () => {
  await loadNamespaces()
  await load()
})

const columns: DataTableColumn[] = [
  { label: '类型', key: 'type', width: '90px' },
  { label: '原因', key: 'reason', width: '150px' },
  { label: '对象', key: 'object', width: '200px' },
  { label: '次数', key: 'count', width: '70px', align: 'center' },
  { label: '最近时间', key: 'last_time', width: '150px' },
  { label: '消息', key: 'message' },
]
</script>

<template>
  <div class="space-y-4">
    <!-- 通用工具栏：命名空间 + 搜索 + 刷新 -->
    <ResourceToolbar
      v-model:keyword="keyword"
      :show-create="false"
      placeholder="搜索对象 / 原因 / 消息…"
    >
      <template #extra>
        <Button variant="secondary" size="sm" :loading="loading" @click="load">
          <RefreshCw class="h-3.5 w-3.5" />
          刷新
        </Button>
      </template>
    </ResourceToolbar>

    <DataTable
      title="集群事件"
      :columns="columns"
      :data="filtered"
      :loading="loading"
      :error="errorMsg"
      row-key="__k8s_event__"
      :empty-text="keyword ? '无匹配的事件' : '当前命名空间下暂无事件'"
      @retry="load"
    >
      <template #cell-type="{ row }">
        <Badge :variant="eventTypeVariant((row as K8sEventItem).type)" dot>
          {{ (row as K8sEventItem).type }}
        </Badge>
      </template>
      <template #cell-reason="{ row }">
        <span class="inline-flex items-center gap-1 text-slate-700 dark:text-slate-200">
          <Info v-if="(row as K8sEventItem).type !== 'Warning'" class="h-3.5 w-3.5 text-blue-500" />
          <AlertTriangle v-else class="h-3.5 w-3.5 text-red-500" />
          {{ (row as K8sEventItem).reason }}
        </span>
      </template>
      <template #cell-message="{ row }">
        <span class="text-xs text-slate-500 dark:text-slate-400">{{ (row as K8sEventItem).message }}</span>
      </template>
    </DataTable>
  </div>
</template>
