import { ref } from 'vue'
import { k8sDetect, type K8sStatus } from '@/api/k8s'

/**
 * K8s 集群可用性检测状态管理（模块级单例，MainLayout 与各页面共享）。
 * 进入应用时自动检测，决定 K8s 栏目的子菜单（节点/Pod/工作负载等）是否显示。
 */
const k8sStatus = ref<K8sStatus | null>(null)
const k8sChecked = ref(false)
const k8sChecking = ref(false)
let detectPromise: Promise<void> | null = null

/** 检测 K8s 集群可用性（并发安全，只检测一次；force 强制刷新）。 */
export function useK8sDetect() {
  async function detect(force = false): Promise<void> {
    if (detectPromise && !force) {
      return detectPromise
    }
    if (k8sChecked.value && !force) {
      return
    }
    k8sChecking.value = true
    detectPromise = (async () => {
      try {
        k8sStatus.value = await k8sDetect()
      } catch {
        k8sStatus.value = null
      } finally {
        k8sChecking.value = false
        k8sChecked.value = true
      }
    })()
    return detectPromise
  }

  return {
    k8sStatus,
    k8sChecked,
    k8sChecking,
    detect,
  }
}

/** K8s 是否可用。 */
export function k8sAvailable(): boolean {
  return k8sStatus.value?.available ?? false
}
