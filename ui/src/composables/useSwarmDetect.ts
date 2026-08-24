import { ref } from 'vue'
import { swarmDetect, type SwarmStatus } from '@/api/swarm'

/**
 * Swarm 集群可用性检测状态管理（模块级单例，MainLayout 与各页面共享）。
 * 进入应用时自动检测，决定 Swarm 栏目的子菜单（节点/服务/网络/Secret）是否显示。
 */
const swarmStatus = ref<SwarmStatus | null>(null)
const swarmChecked = ref(false)
const swarmChecking = ref(false)
let detectPromise: Promise<void> | null = null

/** 检测 Swarm 集群可用性（并发安全，只检测一次；force 强制刷新）。 */
export function useSwarmDetect() {
  async function detect(force = false): Promise<void> {
    if (detectPromise && !force) {
      return detectPromise
    }
    if (swarmChecked.value && !force) {
      return
    }
    swarmChecking.value = true
    detectPromise = (async () => {
      try {
        swarmStatus.value = await swarmDetect()
      } catch {
        swarmStatus.value = null
      } finally {
        swarmChecking.value = false
        swarmChecked.value = true
      }
    })()
    return detectPromise
  }

  return {
    swarmStatus,
    swarmChecked,
    swarmChecking,
    detect,
  }
}

/** Swarm 是否可用。 */
export function swarmAvailable(): boolean {
  return swarmStatus.value?.available ?? false
}
