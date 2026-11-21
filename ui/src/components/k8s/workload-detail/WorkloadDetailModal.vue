<script setup lang="ts">
/**
 * 工作负载详情弹窗：统一展示 Deployment / StatefulSet / DaemonSet / Job / CronJob 的详情。
 *
 * 上下布局：
 * - 上方：基本信息（BasicInfo 组件）
 * - 下方：Tab 选项（容器组 / 访问方式 / 历史版本 / YAML / 事件）
 *
 * 每个 Tab 都是独立子组件，主组件只负责数据获取和 Tab 切换。
 * rawData 加载完成后再渲染 Tab 内容，确保子组件首次挂载时就能拿到正确的 labelSelector。
 */
import { ref, watch, computed } from 'vue'
import Modal from '@/components/ui/Modal.vue'
import ResourceEventsTab from '../ResourceEventsTab.vue'
import BasicInfo from './BasicInfo.vue'
import PodsTab from './PodsTab.vue'
import ServicesTab from './ServicesTab.vue'
import RevisionsTab from './RevisionsTab.vue'
import TerminalModal from '@/components/ui/TerminalModal.vue'
import LogsModal from '@/components/ui/LogsModal.vue'
import { k8sRawYaml, k8sRawJSON, streamK8sPodLogs, type K8sPodItem } from '@/api/k8s'
import {
  extractBasicInfo,
  extractLabelSelector,
  type WorkloadKind,
  type WorkloadBasicInfo,
} from './types'

type DetailTab = 'pods' | 'services' | 'revisions' | 'yaml' | 'events'

const props = defineProps<{
  /** 弹窗 open 状态。 */
  open: boolean
  /** 工作负载类型。 */
  kind: WorkloadKind
  /** 名称。 */
  name: string
  /** 命名空间。 */
  namespace: string
  /** 详情 inspect 路径（如 deployments/nginx?namespace=default）。 */
  inspectPath: string
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()

const loading = ref(false)
const errorMsg = ref('')
const rawData = ref<Record<string, unknown> | null>(null)
const yamlContent = ref('')
const activeTab = ref<DetailTab>('pods')

// Pod 日志弹窗。
const logsOpen = ref(false)
const logsPod = ref<K8sPodItem | null>(null)

// Pod 终端弹窗。
const terminalOpen = ref(false)
const terminalPod = ref<K8sPodItem | null>(null)

/** 从原始数据提取的基本信息。 */
const basicInfo = computed<WorkloadBasicInfo | null>(() => {
  if (!rawData.value) return null
  return extractBasicInfo(rawData.value, props.kind)
})

/** label selector 字符串（用于查询关联 Pod / Service）。 */
const labelSelector = computed(() => {
  if (!rawData.value) return ''
  return extractLabelSelector(rawData.value)
})

/** 工作负载 selector（键值对，用于 Service 匹配）。 */
const selectorMap = computed<Record<string, string>>(() => {
  if (!rawData.value) return {}
  const spec = (rawData.value.spec ?? {}) as Record<string, unknown>
  const selectorObj = (spec.selector ?? {}) as Record<string, unknown>
  return (selectorObj.matchLabels ?? {}) as Record<string, string>
})

/** 是否显示历史版本 Tab（仅 Deployment / StatefulSet / DaemonSet）。 */
const showRevisions = computed(() => ['Deployment', 'StatefulSet', 'DaemonSet'].includes(props.kind))

/** Tab 列表。 */
const tabs = computed<{ key: DetailTab; label: string }[]>(() => {
  const list: { key: DetailTab; label: string }[] = [
    { key: 'pods', label: '容器组' },
    { key: 'services', label: '访问方式' },
  ]
  if (showRevisions.value) {
    list.push({ key: 'revisions', label: '历史版本' })
  }
  list.push({ key: 'yaml', label: 'YAML' })
  list.push({ key: 'events', label: '事件' })
  return list
})

/** 构建 Pod 终端 ws 地址。 */
const terminalWsUrl = computed(() => {
  if (!terminalPod.value) return ''
  const params = new URLSearchParams({ namespace: terminalPod.value.namespace })
  const c = terminalPod.value.containers?.[0]?.name
  if (c) params.set('container', c)
  return `/api/v1/k8s/pods/${terminalPod.value.name}/terminal?${params}`
})

/** LogsModal stream 包装。 */
function streamLogs(tail: string, container: string, onLine: (line: string) => void, onError: (msg: string) => void, onClose: () => void): () => void {
  const p = logsPod.value!
  return streamK8sPodLogs(p.name, p.namespace, Number(tail), container, onLine, onError, onClose)
}

async function loadDetail(): Promise<void> {
  loading.value = true
  errorMsg.value = ''
  rawData.value = null
  yamlContent.value = ''
  activeTab.value = 'pods'
  try {
    // 获取 JSON 格式的原始对象，http 封装会自动解包 ApiResponse.data。
    rawData.value = await k8sRawJSON(props.inspectPath)
    // 同时获取 YAML（YAML 接口直接返回文本，不包裹）。
    yamlContent.value = await k8sRawYaml(props.inspectPath)
  } catch (err) {
    errorMsg.value = (err as Error).message
  } finally {
    loading.value = false
  }
}

function openPodLogs(pod: K8sPodItem): void {
  logsPod.value = pod
  logsOpen.value = true
}

function openPodTerminal(pod: K8sPodItem): void {
  terminalPod.value = pod
  terminalOpen.value = true
}

watch(
  () => [props.open, props.inspectPath] as const,
  ([open, path]) => {
    if (open && path) {
      void loadDetail()
    }
  },
  { immediate: true },
)
</script>

<template>
  <Modal
    :open="open"
    @update:open="(v) => emit('update:open', v)"
    :title="`${kind} - ${name}`"
    width="max-w-5xl"
  >
    <div class="min-h-[50vh]">
      <!-- 加载中 -->
      <div v-if="loading" class="flex items-center justify-center py-16 text-sm text-slate-400">
        <svg class="mr-2 h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none" aria-hidden="true">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
        </svg>
        加载中…
      </div>

      <!-- 错误 -->
      <div v-else-if="errorMsg" class="rounded-lg bg-red-50 px-4 py-3 text-sm text-red-600 dark:bg-red-900/20 dark:text-red-300">
        {{ errorMsg }}
      </div>

      <!-- 内容：rawData 加载完成后再渲染，确保子组件首次挂载时 labelSelector 已有值 -->
      <div v-else-if="basicInfo" class="space-y-5">
        <!-- 上方：基本信息 -->
        <BasicInfo :info="basicInfo" :kind="kind" :raw-object="rawData" />

        <!-- 分隔线 -->
        <div class="border-t border-slate-200 dark:border-slate-700" />

        <!-- Tab 切换 -->
        <div class="flex items-center gap-1 overflow-x-auto border-b border-slate-200 dark:border-slate-700">
          <button
            v-for="t in tabs"
            :key="t.key"
            class="relative whitespace-nowrap px-4 py-2.5 text-sm font-medium transition-colors"
            :class="activeTab === t.key
              ? 'border-b-2 border-blue-500 text-blue-600 dark:text-blue-400'
              : 'text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200'"
            @click="activeTab = t.key"
          >
            {{ t.label }}
          </button>
        </div>

        <!-- 容器组 Tab -->
        <div v-show="activeTab === 'pods'">
          <PodsTab
            :namespace="namespace"
            :label-selector="labelSelector"
            :active="activeTab === 'pods'"
            @logs="openPodLogs"
            @terminal="openPodTerminal"
          />
        </div>

        <!-- 访问方式 Tab -->
        <div v-show="activeTab === 'services'">
          <ServicesTab
            :namespace="namespace"
            :selector="selectorMap"
            :active="activeTab === 'services'"
          />
        </div>

        <!-- 历史版本 Tab -->
        <div v-if="showRevisions" v-show="activeTab === 'revisions'">
          <RevisionsTab
            :kind="kind"
            :name="name"
            :namespace="namespace"
            :raw-object="rawData"
            :active="activeTab === 'revisions'"
          />
        </div>

        <!-- YAML Tab -->
        <div v-show="activeTab === 'yaml'">
          <pre class="max-h-[55vh] overflow-auto rounded-lg bg-slate-900 px-4 py-3 font-mono text-xs leading-relaxed text-green-300">{{ yamlContent || '（空）' }}</pre>
        </div>

        <!-- 事件 Tab -->
        <div v-show="activeTab === 'events'">
          <ResourceEventsTab
            :kind="kind"
            :name="name"
            :namespace="namespace"
            :active="activeTab === 'events'"
          />
        </div>
      </div>
    </div>

  </Modal>

  <!-- Pod 日志弹窗（从容器组 Tab 中触发） -->
  <LogsModal
    v-if="logsPod"
    v-model:open="logsOpen"
    :title="`Pod 日志 - ${logsPod.name}`"
    :stream="streamLogs"
    :containers="logsPod.containers ?? []"
  />

  <!-- Pod 终端弹窗（从容器组 Tab 中触发） -->
  <TerminalModal
    v-if="terminalPod"
    v-model:open="terminalOpen"
    :url="terminalWsUrl"
    :title="`终端 - ${terminalPod.name}`"
  />
</template>
