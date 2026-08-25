<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterView } from 'vue-router'
import { RefreshCw, Boxes, Server } from '@lucide/vue'
import Button from '@/components/ui/Button.vue'
import { useToast } from '@/composables/useToast'
import { k8sDetect, type K8sStatus } from '@/api/k8s'
import { provideK8sConn, k8sLabel } from '@/composables/useK8sConn'

const toast = useToast()

const detect = ref<K8sStatus | null>(null)
const loading = ref(false)

const label = computed(() => k8sLabel(detect.value))

let reloadVersion = 0
function reload(): void {
  reloadVersion++
}

async function refresh(): Promise<void> {
  loading.value = true
  try {
    detect.value = await k8sDetect()
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    loading.value = false
  }
}

provideK8sConn({
  detect: computed(() => detect.value),
  label: computed(() => label.value),
  refresh,
  reload,
})

onMounted(() => {
  void refresh()
})
</script>

<template>
  <div class="space-y-5">
    <!-- 集群状态栏（连接目标由 config.yaml 配置，无需切换） -->
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex items-center gap-2.5 text-slate-600 dark:text-slate-300">
        <span
          class="inline-flex h-8 w-8 items-center justify-center rounded-lg bg-blue-50 dark:bg-blue-900/40"
        >
          <Server class="h-4.5 w-4.5 text-blue-600 dark:text-blue-400" />
        </span>
        <span class="text-sm font-medium">Kubernetes</span>
        <span
          class="inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-xs"
          :class="detect?.available ? 'bg-green-50 text-green-700 dark:bg-green-900/40 dark:text-green-300' : 'bg-red-50 text-red-600 dark:bg-red-900/40 dark:text-red-400'"
        >
          <Boxes class="h-3 w-3" />
          {{ label }}
        </span>
      </div>
      <Button variant="ghost" size="sm" :loading="loading" @click="refresh" aria-label="刷新集群状态">
        <RefreshCw class="h-3.5 w-3.5" />
      </Button>
    </div>

    <!-- 子页面（概览/节点/Pod/工作负载/服务/配置/事件/YAML） -->
    <RouterView />
  </div>
</template>
