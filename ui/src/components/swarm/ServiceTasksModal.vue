<script setup lang="ts">
import { ref, watch } from 'vue'
import { FileText } from '@lucide/vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import Modal from '@/components/ui/Modal.vue'
import Tooltip from '@/components/ui/Tooltip.vue'
import TaskLogsModal from './TaskLogsModal.vue'
import { useToast } from '@/composables/useToast'
import { swarmTasks, type SwarmServiceItem, type SwarmTaskItem } from '@/api/swarm'

const props = defineProps<{
  open: boolean
  service: SwarmServiceItem | null
}>()

const emit = defineEmits<{ 'update:open': [value: boolean] }>()

const toast = useToast()

const tasks = ref<SwarmTaskItem[]>([])
const tasksLoading = ref(false)
const taskLogsTarget = ref<SwarmTaskItem | null>(null)
const taskLogsOpen = ref(false)

function taskStateVariant(s: string): 'green' | 'red' | 'yellow' | 'gray' {
  if (s === 'running') return 'green'
  if (s === 'failed' || s === 'rejected') return 'red'
  if (s === 'preparing' || s === 'pending' || s === 'starting') return 'yellow'
  return 'gray'
}

async function fetchTasks(serviceId: string): Promise<void> {
  tasks.value = []
  tasksLoading.value = true
  try {
    tasks.value = await swarmTasks(serviceId)
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    tasksLoading.value = false
  }
}

watch(
  () => [props.open, props.service] as const,
  ([open, service]) => {
    if (open && service) {
      void fetchTasks(service.id)
    }
  },
  { immediate: true },
)
</script>

<template>
  <Modal
    :open="open"
    @update:open="emit('update:open', $event)"
    :title="`任务 - ${service?.name ?? ''}`"
    width="max-w-3xl"
  >
    <div class="max-h-[60vh] overflow-auto">
      <table class="w-full text-sm">
        <thead class="border-b border-slate-200 text-left text-xs font-medium text-slate-500 dark:border-slate-700 dark:text-slate-400">
          <tr>
            <th class="px-3 py-2">槽位</th>
            <th class="px-3 py-2">节点</th>
            <th class="px-3 py-2">状态</th>
            <th class="px-3 py-2">期望</th>
            <th class="px-3 py-2">容器 ID</th>
            <th class="px-3 py-2">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="t in tasks" :key="t.id" class="border-b border-slate-100 last:border-0 dark:border-slate-800">
            <td class="px-3 py-2 text-slate-700 dark:text-slate-200">{{ t.slot }}</td>
            <td class="px-3 py-2 text-slate-700 dark:text-slate-200">{{ t.node_name || t.node_id.slice(0, 12) }}</td>
            <td class="px-3 py-2">
              <Badge :variant="taskStateVariant(t.state)" dot>{{ t.state }}</Badge>
              <Tooltip v-if="t.error" :text="t.error" placement="top">
                <p class="mt-0.5 max-w-[200px] truncate text-xs text-red-500">{{ t.error }}</p>
              </Tooltip>
            </td>
            <td class="px-3 py-2 text-slate-500 dark:text-slate-400">{{ t.desired_state }}</td>
            <td class="px-3 py-2 font-mono text-xs text-slate-500 dark:text-slate-400">{{ t.container_id?.slice(0, 12) || '—' }}</td>
            <td class="px-3 py-2">
              <button
                v-if="t.container_id"
                class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs text-slate-600 transition-colors hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-700"
                @click="taskLogsTarget = t; taskLogsOpen = true"
              >
                <FileText class="h-3.5 w-3.5" />
                日志
              </button>
              <span v-else class="text-xs text-slate-300 dark:text-slate-600">—</span>
            </td>
          </tr>
          <tr v-if="tasksLoading">
            <td colspan="6" class="px-3 py-8 text-center text-sm text-slate-400">加载中…</td>
          </tr>
          <tr v-else-if="tasks.length === 0">
            <td colspan="6" class="px-3 py-8 text-center text-sm text-slate-400">暂无任务</td>
          </tr>
        </tbody>
      </table>
    </div>
    <template #footer>
      <Button variant="secondary" @click="emit('update:open', false)">关闭</Button>
    </template>

    <!-- 任务日志 -->
    <TaskLogsModal
      v-if="taskLogsTarget"
      v-model:open="taskLogsOpen"
      :task-id="taskLogsTarget.id"
      :task-label="`${service?.name ?? ''} #${taskLogsTarget.slot}`"
    />
  </Modal>
</template>
