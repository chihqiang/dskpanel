<script setup lang="ts">
import { ref, watch } from 'vue'
import Modal from '@/components/ui/Modal.vue'
import Badge from '@/components/ui/Badge.vue'
import Button from '@/components/ui/Button.vue'
import { useToast } from '@/composables/useToast'
import {
  k8sNodeUsage,
  k8sCordonNode,
  k8sUncordonNode,
  k8sDrainNode,
  type K8sNodeItem,
  type K8sNodeUsage,
} from '@/api/k8s'

const props = defineProps<{
  open: boolean
  node: K8sNodeItem | null
}>()

const emit = defineEmits<{ 'update:open': [value: boolean]; updated: [] }>()

const toast = useToast()

const usage = ref<K8sNodeUsage | null>(null)
const usageError = ref('')
const busy = ref(false)

watch(
  () => [props.open, props.node] as const,
  ([open, node]) => {
    if (open && node) {
      usage.value = null
      usageError.value = ''
      void loadUsage(node.name)
    }
  },
  { immediate: true },
)

async function loadUsage(name: string): Promise<void> {
  try {
    usage.value = await k8sNodeUsage(name)
  } catch (err) {
    usageError.value = (err as Error).message
  }
}

async function runAction(fn: () => Promise<unknown>, msg: string): Promise<void> {
  if (!props.node) return
  busy.value = true
  try {
    await fn()
    toast.success(`节点「${props.node.name}」${msg}`)
    emit('updated')
    // 重新拉取用量。
    void loadUsage(props.node.name)
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    busy.value = false
  }
}

/** 标签/污点对象 → 行渲染。 */
function entries(obj: Record<string, string> | undefined): [string, string][] {
  return Object.entries(obj ?? {})
}
</script>

<template>
  <Modal :open="open" @update:open="emit('update:open', $event)" :title="`节点 - ${node?.name ?? ''}`" width="max-w-2xl">
    <div v-if="node" class="space-y-5">
      <!-- 基本信息 -->
      <div class="grid grid-cols-2 gap-3 text-sm sm:grid-cols-4">
        <div class="rounded-lg bg-slate-50 px-3 py-2 dark:bg-slate-700/40">
          <p class="text-xs text-slate-400">角色</p>
          <p class="font-medium text-slate-700 dark:text-slate-200">{{ node.role }}</p>
        </div>
        <div class="rounded-lg bg-slate-50 px-3 py-2 dark:bg-slate-700/40">
          <p class="text-xs text-slate-400">状态</p>
          <p class="font-medium text-slate-700 dark:text-slate-200">{{ node.status }}</p>
        </div>
        <div class="rounded-lg bg-slate-50 px-3 py-2 dark:bg-slate-700/40">
          <p class="text-xs text-slate-400">版本</p>
          <p class="font-medium text-slate-700 dark:text-slate-200">{{ node.version }}</p>
        </div>
        <div class="rounded-lg bg-slate-50 px-3 py-2 dark:bg-slate-700/40">
          <p class="text-xs text-slate-400">架构</p>
          <p class="font-medium text-slate-700 dark:text-slate-200">{{ node.os }}/{{ node.arch }}</p>
        </div>
        <div class="rounded-lg bg-slate-50 px-3 py-2 dark:bg-slate-700/40">
          <p class="text-xs text-slate-400">运行时</p>
          <p class="font-medium text-slate-700 dark:text-slate-200">{{ node.container_runtime }}</p>
        </div>
        <div class="rounded-lg bg-slate-50 px-3 py-2 dark:bg-slate-700/40">
          <p class="text-xs text-slate-400">内网 IP</p>
          <p class="font-medium text-slate-700 dark:text-slate-200">{{ node.internal_ip || '—' }}</p>
        </div>
        <div class="rounded-lg bg-slate-50 px-3 py-2 dark:bg-slate-700/40">
          <p class="text-xs text-slate-400">CPU / 内存</p>
          <p class="font-medium text-slate-700 dark:text-slate-200">{{ node.cpu }} / {{ node.memory }}</p>
        </div>
        <div class="rounded-lg bg-slate-50 px-3 py-2 dark:bg-slate-700/40">
          <p class="text-xs text-slate-400">Pod 容量</p>
          <p class="font-medium text-slate-700 dark:text-slate-200">{{ node.pods_capacity }}</p>
        </div>
      </div>

      <!-- 资源使用率 -->
      <div>
        <p class="mb-2 text-sm font-medium text-slate-700 dark:text-slate-200">资源使用率（按 Pod requests 估算）</p>
        <div v-if="usageError" class="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600 dark:bg-red-900/20 dark:text-red-300">
          {{ usageError }}
        </div>
        <div v-else-if="usage" class="grid grid-cols-2 gap-3 text-sm sm:grid-cols-4">
          <div class="rounded-lg border border-slate-100 px-3 py-2 dark:border-slate-700">
            <p class="text-xs text-slate-400">CPU 使用</p>
            <p class="font-medium text-slate-700 dark:text-slate-200">{{ usage.cpu_used }} / {{ usage.cpu_total }}</p>
            <Badge :variant="Number.parseFloat(usage.cpu_percent) > 80 ? 'red' : 'green'">{{ usage.cpu_percent }}</Badge>
          </div>
          <div class="rounded-lg border border-slate-100 px-3 py-2 dark:border-slate-700">
            <p class="text-xs text-slate-400">内存使用</p>
            <p class="font-medium text-slate-700 dark:text-slate-200">{{ usage.mem_used }} / {{ usage.mem_total }}</p>
            <Badge :variant="Number.parseFloat(usage.mem_percent) > 80 ? 'red' : 'green'">{{ usage.mem_percent }}</Badge>
          </div>
          <div class="rounded-lg border border-slate-100 px-3 py-2 dark:border-slate-700">
            <p class="text-xs text-slate-400">Pod 使用</p>
            <p class="font-medium text-slate-700 dark:text-slate-200">{{ usage.pods_used }} / {{ usage.pods_total }}</p>
          </div>
        </div>
      </div>

      <!-- Labels -->
      <div>
        <p class="mb-2 text-sm font-medium text-slate-700 dark:text-slate-200">标签</p>
        <div v-if="entries(node.labels).length" class="flex flex-wrap gap-1.5">
          <span
            v-for="([k, v]) in entries(node.labels)"
            :key="k"
            class="rounded-md bg-slate-100 px-2 py-0.5 font-mono text-xs text-slate-600 dark:bg-slate-700 dark:text-slate-300"
          >
            {{ k }}={{ v }}
          </span>
        </div>
        <p v-else class="text-sm text-slate-400">无</p>
      </div>

      <!-- Taints -->
      <div>
        <p class="mb-2 text-sm font-medium text-slate-700 dark:text-slate-200">污点</p>
        <div v-if="node.taints?.length" class="flex flex-wrap gap-1.5">
          <Badge v-for="t in node.taints" :key="t.key" variant="yellow">
            {{ t.key }}{{ t.value ? `=${t.value}` : '' }}:{{ t.effect }}
          </Badge>
        </div>
        <p v-else class="text-sm text-slate-400">无</p>
      </div>
    </div>

    <template #footer>
      <div class="flex items-center gap-2">
        <Button variant="secondary" size="sm" :disabled="busy" @click="runAction(() => k8sCordonNode(props.node!.name), '已 Cordon')">Cordon</Button>
        <Button variant="secondary" size="sm" :disabled="busy" @click="runAction(() => k8sUncordonNode(props.node!.name), '已 Uncordon')">Uncordon</Button>
        <Button variant="danger" size="sm" :disabled="busy" @click="runAction(() => k8sDrainNode(props.node!.name), '已开始驱逐')">驱逐</Button>
      </div>
      <div class="ml-auto">
        <Button variant="secondary" @click="emit('update:open', false)">关闭</Button>
      </div>
    </template>
  </Modal>
</template>
