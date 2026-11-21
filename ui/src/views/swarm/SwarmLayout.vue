<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterView } from 'vue-router'
import { RefreshCw, Server, Boxes } from '@lucide/vue'
import Button from '@/components/ui/Button.vue'
import { useToast } from '@/composables/useToast'
import { swarmDetect, type SwarmStatus } from '@/api/swarm'
import { provideSwarmConn, swarmLabel } from '@/composables/useSwarmConn'

const toast = useToast()

const detect = ref<SwarmStatus | null>(null)
const loading = ref(false)

const label = computed(() => swarmLabel(detect.value))

const clusterName = computed(() => label.value)

let reloadVersion = 0
function reload(): void {
  reloadVersion++
}

async function refresh(): Promise<void> {
  loading.value = true
  try {
    detect.value = await swarmDetect()
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    loading.value = false
  }
}

provideSwarmConn({
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
      <div class="flex items-center gap-3">
        <div class="flex items-center gap-2 text-slate-600 dark:text-slate-300">
          <Server class="h-5 w-5 text-blue-600 dark:text-blue-400" />
          <span class="text-sm font-medium">Swarm 集群</span>
        </div>
        <Button variant="secondary" size="sm" :loading="loading" @click="refresh">
          <RefreshCw class="h-3.5 w-3.5" />
          刷新
        </Button>
        <span
          class="hidden items-center gap-1 rounded-full px-2.5 py-0.5 text-xs sm:inline-flex"
          :class="detect?.available ? 'bg-green-50 text-green-700 dark:bg-green-900/40 dark:text-green-300' : 'bg-red-50 text-red-600 dark:bg-red-900/40 dark:text-red-400'"
        >
          <Boxes class="h-3.5 w-3.5" />
          {{ clusterName }}
        </span>
      </div>
    </div>

    <!-- 子页面（概览/节点/服务/Secret） -->
    <RouterView />
  </div>
</template>
