import { ref } from 'vue'
import { detectDocker, type DockerInfo } from '@/api/docker'

/**
 * Docker 环境检测状态管理。
 * 进入应用时自动检测本机 Docker，决定 Docker 栏目是否可用。
 */
const dockerInfo = ref<DockerInfo | null>(null)
const dockerChecked = ref(false)
const dockerChecking = ref(false)
let detectPromise: Promise<void> | null = null

/** 检测本机 Docker（并发安全，只检测一次）。 */
export function useDockerDetect() {
  async function detect(force = false): Promise<void> {
    if (detectPromise && !force) {
      return detectPromise
    }
    if (dockerChecked.value && !force) {
      return
    }
    dockerChecking.value = true
    detectPromise = (async () => {
      try {
        dockerInfo.value = await detectDocker()
      } catch {
        dockerInfo.value = null
      } finally {
        dockerChecking.value = false
        dockerChecked.value = true
      }
    })()
    return detectPromise
  }

  return {
    dockerInfo,
    dockerChecked,
    dockerChecking,
    detect,
  }
}

/** Docker 是否可用。 */
export function dockerAvailable(): boolean {
  return dockerInfo.value?.available ?? false
}
