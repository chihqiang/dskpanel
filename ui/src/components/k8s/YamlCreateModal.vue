<script setup lang="ts">
import { ref, watch } from 'vue'
import { Play, FileSearch } from '@lucide/vue'
import Button from '@/components/ui/Button.vue'
import Modal from '@/components/ui/Modal.vue'
import { useToast } from '@/composables/useToast'
import { useActivity } from '@/composables/useActivity'
import { k8sTemplates, type YamlTemplate } from '@/templates'
import { k8sApplyYAML, k8sDryRunYAML } from '@/api/k8s'

/**
 * 通过 YAML 创建资源弹窗（kubectl apply 语义）。
 * - 按入口传入 templates，展示对应的模板（如 Pod 页只展示 Pod 模板）
 * - 支持模板选择 + 多文档 YAML + 服务端 DryRun 校验 + Apply
 * - 校验/创建结果统一发送到顶部「通知」，弹窗内保持简洁
 */
const props = withDefaults(
  defineProps<{
    open: boolean
    /** 弹窗标题（如“创建 Pod”“创建工作负载”）。 */
    title?: string
    /** 模板集（默认通用模板）。 */
    templates?: YamlTemplate[]
  }>(),
  {
    title: '创建资源',
    templates: () => k8sTemplates,
  },
)

const emit = defineEmits<{
  'update:open': [value: boolean]
  /** 应用成功后通知父组件刷新列表。 */
  created: []
}>()

const toast = useToast()
const activity = useActivity()

const yamlText = ref(props.templates[0]?.yaml ?? '')
const selectedTemplate = ref(props.templates[0]?.name ?? '')
const running = ref(false)

watch(selectedTemplate, (name) => {
  const tpl = props.templates.find((t) => t.name === name)
  if (!tpl) return
  yamlText.value = tpl.yaml
})

// 打开时重置（并应用最新的模板集）。
watch(
  () => [props.open, props.templates] as const,
  ([open, templates]) => {
    if (open) {
      selectedTemplate.value = templates[0]?.name ?? ''
      yamlText.value = templates[0]?.yaml ?? ''
    }
  },
)

/** 把结果 items 拼成摘要（如 “Pod/demo-pod valid；Service/web created”）。 */
function summarize(res: { items?: { kind: string; name: string; action: string }[]; message: string }): string {
  if (res.items?.length) {
    return res.items.map((i) => `${i.kind}/${i.name} ${i.action}`).join('；')
  }
  return res.message
}

async function runDryRun(): Promise<void> {
  if (!yamlText.value.trim()) {
    toast.error('请先输入 YAML 内容')
    return
  }
  running.value = true
  try {
    const res = await k8sDryRunYAML(yamlText.value)
    if (res.ok) {
      activity.success('DryRun 校验通过', summarize(res))
      toast.success('校验通过，已记录到通知')
    } else {
      activity.error('DryRun 校验失败', res.message)
      toast.error(`校验失败：${res.message}`)
    }
  } catch (err) {
    activity.error('DryRun 校验失败', (err as Error).message)
    toast.error((err as Error).message)
  } finally {
    running.value = false
  }
}

async function runApply(): Promise<void> {
  if (!yamlText.value.trim()) {
    toast.error('请先输入 YAML 内容')
    return
  }
  running.value = true
  try {
    const res = await k8sApplyYAML(yamlText.value)
    if (res.ok) {
      activity.success('资源应用成功', summarize(res))
      toast.success('资源已应用')
      emit('created')
      emit('update:open', false)
    } else {
      activity.error('资源应用失败', res.message)
      toast.error(res.message)
    }
  } catch (err) {
    activity.error('资源应用失败', (err as Error).message)
    toast.error((err as Error).message)
  } finally {
    running.value = false
  }
}
</script>

<template>
  <Modal
    :open="open"
    @update:open="emit('update:open', $event)"
    :title="title"
    width="max-w-3xl"
  >
    <div class="space-y-3">
      <!-- 模板选择 -->
      <div class="flex flex-wrap items-center gap-2">
        <span class="text-sm text-slate-500">模板</span>
        <select v-model="selectedTemplate" class="input input-sm w-56">
          <option v-for="t in props.templates" :key="t.name" :value="t.name">{{ t.name }}</option>
        </select>
        <span v-if="props.templates[0]?.desc" class="text-xs text-slate-400">{{ props.templates.find((t) => t.name === selectedTemplate)?.desc }}</span>
      </div>

      <!-- YAML 编辑器（窗口样式），结果统一发送到顶部「通知」 -->
      <div class="overflow-hidden rounded-lg border border-slate-200 dark:border-slate-700">
        <div class="flex items-center gap-2 border-b border-slate-200 bg-slate-50 px-3 py-2 dark:border-slate-700 dark:bg-slate-800">
          <span class="h-2.5 w-2.5 rounded-full bg-red-400" />
          <span class="h-2.5 w-2.5 rounded-full bg-yellow-400" />
          <span class="h-2.5 w-2.5 rounded-full bg-green-400" />
          <span class="ml-2 font-mono text-xs text-slate-400">manifest.yaml</span>
          <span v-if="running" class="ml-auto flex items-center gap-1.5 text-xs text-slate-400">
            <span class="h-3 w-3 animate-spin rounded-full border-2 border-slate-300 border-t-blue-500" />
            校验中…
          </span>
        </div>
        <textarea
          v-model="yamlText"
          spellcheck="false"
          class="block h-60 w-full resize-y bg-slate-900 px-3 py-2 font-mono text-xs leading-relaxed text-green-300 outline-none placeholder:text-slate-500"
          placeholder="# 粘贴或编写 Kubernetes 资源 YAML（支持 --- 多文档）"
        />
      </div>
    </div>

    <template #footer>
      <div class="flex items-center gap-2">
        <Button variant="secondary" :loading="running" @click="runDryRun">
          <FileSearch class="h-3.5 w-3.5" />
          校验（DryRun）
        </Button>
      </div>
      <div class="ml-auto flex items-center gap-2">
        <Button variant="secondary" @click="emit('update:open', false)">取消</Button>
        <Button :loading="running" @click="runApply">
          <Play class="h-3.5 w-3.5" />
          创建
        </Button>
      </div>
    </template>
  </Modal>
</template>
