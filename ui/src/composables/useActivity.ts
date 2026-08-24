import { useActivityStore } from '@/stores/activity'

/**
 * 全局活动日志 composable（封装 pinia store）。
 * 用法：
 *   const activity = useActivity()
 *   activity.success('已删除容器', containerName)
 *   activity.error('拉取镜像失败', imageTag)
 */
export function useActivity() {
  return useActivityStore()
}
