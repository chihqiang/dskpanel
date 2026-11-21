<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { Copy, Check, Save, FileSearch, Pencil, Eye, Download } from '@lucide/vue'
import Button from '@/components/ui/Button.vue'
import Modal from '@/components/ui/Modal.vue'
import ResourceEventsTab from './ResourceEventsTab.vue'
import { useClipboard } from '@/utils/clipboard'
import { useToast } from '@/composables/useToast'
import { useActivity } from '@/composables/useActivity'
import { k8sApplyYAML, k8sDryRunYAML } from '@/api/k8s'

type DetailTab = 'yaml' | 'events'
const detailTab = ref<DetailTab>('yaml')

const props = defineProps<{
  open: boolean
  title: string
  /** 拉取资源 YAML 文本的函数（由父组件按资源类型构造）。 */
  fetchYaml: (() => Promise<string>) | null
  /** 资源类型（用于关联事件查询，可选）。 */
  resourceKind?: string
  /** 资源名称（用于关联事件查询，可选）。 */
  resourceName?: string
  /** 命名空间（用于关联事件查询，可选）。 */
  resourceNamespace?: string
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  /** 保存成功后通知父组件刷新。 */
  saved: []
}>()

const { copy } = useClipboard()
const toast = useToast()
const activity = useActivity()

const content = ref('')
const loading = ref(false)
const errorMsg = ref('')
const copied = ref(false)
const editing = ref(false)
const saving = ref(false)
/** 原始加载的 YAML（用于检测是否有修改）。 */
const original = ref('')

watch(
  () => [props.open, props.fetchYaml] as const,
  ([open, fn]) => {
    if (open && fn) {
      content.value = ''
      original.value = ''
      errorMsg.value = ''
      editing.value = false
      detailTab.value = 'yaml'
      loading.value = true
      fn()
        .then((text) => {
          content.value = text
          original.value = text
        })
        .catch((err: Error) => {
          errorMsg.value = err.message
        })
        .finally(() => {
          loading.value = false
        })
    }
  },
  { immediate: true },
)

/** 是否有未保存的修改。 */
const dirty = computed(() => content.value !== original.value)

async function copyYaml(): Promise<void> {
  await copy(content.value, '已复制到剪贴板', '复制失败，请手动复制')
  copied.value = true
  setTimeout(() => (copied.value = false), 1500)
}

/** 下载 YAML 文件。 */
function downloadYaml(): void {
  if (!content.value) return
  const fileName = props.resourceName
    ? `${props.resourceKind?.toLowerCase() ?? 'resource'}-${props.resourceName}.yaml`
    : 'resource.yaml'
  const blob = new Blob([content.value], { type: 'text/yaml;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = fileName
  a.click()
  URL.revokeObjectURL(url)
  toast.success(`已下载 ${fileName}`)
}

function toggleEdit(): void {
  editing.value = !editing.value
  if (!editing.value && dirty.value) {
    // 退出编辑模式时如果有修改，恢复原始内容。
    content.value = original.value
  }
}

function summarize(res: { items?: { kind: string; name: string; action: string }[]; message: string }): string {
  if (res.items?.length) {
    return res.items.map((i) => `${i.kind}/${i.name} ${i.action}`).join('；')
  }
  return res.message
}

async function runDryRun(): Promise<void> {
  if (!content.value.trim()) {
    toast.error('YAML 内容为空')
    return
  }
  saving.value = true
  try {
    const res = await k8sDryRunYAML(content.value)
    if (res.ok) {
      activity.success('DryRun 校验通过', summarize(res))
      toast.success('校验通过')
    } else {
      activity.error('DryRun 校验失败', res.message)
      toast.error(`校验失败：${res.message}`)
    }
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    saving.value = false
  }
}

async function save(): Promise<void> {
  if (!content.value.trim()) {
    toast.error('YAML 内容为空')
    return
  }
  saving.value = true
  try {
    const res = await k8sApplyYAML(content.value)
    if (res.ok) {
      activity.success('资源更新成功', summarize(res))
      toast.success('资源已更新')
      original.value = content.value
      editing.value = false
      emit('saved')
    } else {
      activity.error('资源更新失败', res.message)
      toast.error(res.message)
    }
  } catch (err) {
    activity.error('资源更新失败', (err as Error).message)
    toast.error((err as Error).message)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <Modal :open="open" @update:open="emit('update:open', $event)" :title="title" width="max-w-3xl">
    <div class="min-h-[40vh]">
      <div v-if="loading" class="flex items-center justify-center py-16 text-sm text-slate-400">
        <svg class="mr-2 h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none" aria-hidden="true">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
        </svg>
        加载中…
      </div>
      <div v-else-if="errorMsg" class="rounded-lg bg-red-50 px-4 py-3 text-sm text-red-600 dark:bg-red-900/20 dark:text-red-300">
        {{ errorMsg }}
      </div>
      <template v-else>
        <!-- Tab 切换（仅在提供了资源信息时显示事件 Tab） -->
        <div v-if="resourceKind && resourceName && resourceNamespace" class="mb-3 inline-flex rounded-lg bg-slate-100 p-1 dark:bg-slate-700/60">
          <button
            v-for="t in [{ key: 'yaml', label: 'YAML' }, { key: 'events', label: '事件' }] as const"
            :key="t.key"
            class="rounded-md px-3.5 py-1.5 text-sm font-medium transition-colors"
            :class="detailTab === t.key ? 'bg-white text-blue-700 shadow-sm dark:bg-slate-800 dark:text-blue-300' : 'text-slate-600 hover:text-slate-800 dark:text-slate-300 dark:hover:text-slate-100'"
            @click="detailTab = t.key"
          >
            {{ t.label }}
          </button>
        </div>

        <!-- YAML 内容 -->
        <div v-show="detailTab === 'yaml'">
        <!-- 查看模式 -->
        <pre
          v-if="!editing"
          class="max-h-[60vh] overflow-auto rounded-lg bg-slate-900 px-4 py-3 font-mono text-xs leading-relaxed text-green-300"
        >{{ content || '（空）' }}</pre>
        <!-- 编辑模式 -->
        <div v-else class="overflow-hidden rounded-lg border border-slate-200 dark:border-slate-700">
          <div class="flex items-center gap-2 border-b border-slate-200 bg-slate-50 px-3 py-2 dark:border-slate-700 dark:bg-slate-800">
            <span class="h-2.5 w-2.5 rounded-full bg-red-400" />
            <span class="h-2.5 w-2.5 rounded-full bg-yellow-400" />
            <span class="h-2.5 w-2.5 rounded-full bg-green-400" />
            <span class="ml-2 font-mono text-xs text-slate-400">manifest.yaml</span>
            <span v-if="dirty" class="ml-2 text-xs text-amber-500">● 未保存</span>
            <span v-if="saving" class="ml-auto flex items-center gap-1.5 text-xs text-slate-400">
              <span class="h-3 w-3 animate-spin rounded-full border-2 border-slate-300 border-t-blue-500" />
              处理中…
            </span>
          </div>
          <textarea
            v-model="content"
            spellcheck="false"
            class="block h-72 w-full resize-y bg-slate-900 px-3 py-2 font-mono text-xs leading-relaxed text-green-300 outline-none placeholder:text-slate-500"
            placeholder="# 编辑 Kubernetes 资源 YAML"
          />
        </div>
        </div>

        <!-- 事件 Tab -->
        <div v-if="resourceKind && resourceName && resourceNamespace" v-show="detailTab === 'events'">
          <ResourceEventsTab
            :kind="resourceKind"
            :name="resourceName"
            :namespace="resourceNamespace"
            :active="detailTab === 'events'"
          />
        </div>
      </template>
    </div>
    <template #footer>
      <div class="flex items-center gap-2">
        <!-- 复制 -->
        <Button variant="secondary" size="sm" :disabled="!content" @click="copyYaml">
          <Check v-if="copied" class="h-3.5 w-3.5 text-green-500" />
          <Copy v-else class="h-3.5 w-3.5" />
          复制
        </Button>
        <!-- 下载 -->
        <Button variant="secondary" size="sm" :disabled="!content" @click="downloadYaml">
          <Download class="h-3.5 w-3.5" />
          下载
        </Button>
        <!-- 编辑模式：DryRun -->
        <Button v-if="editing" variant="secondary" size="sm" :loading="saving" @click="runDryRun">
          <FileSearch class="h-3.5 w-3.5" />
          校验
        </Button>
      </div>
      <div class="ml-auto flex items-center gap-2">
        <!-- 编辑/取消编辑 -->
        <Button v-if="!editing" variant="secondary" size="sm" @click="toggleEdit">
          <Pencil class="h-3.5 w-3.5" />
          编辑
        </Button>
        <Button v-else variant="secondary" size="sm" @click="toggleEdit">
          <Eye class="h-3.5 w-3.5" />
          取消编辑
        </Button>
        <!-- 保存 -->
        <Button v-if="editing" size="sm" :loading="saving" :disabled="!dirty" @click="save">
          <Save class="h-3.5 w-3.5" />
          保存
        </Button>
        <Button v-else variant="secondary" @click="emit('update:open', false)">关闭</Button>
      </div>
    </template>
  </Modal>
</template>
