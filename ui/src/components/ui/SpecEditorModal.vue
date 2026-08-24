<script setup lang="ts">
import { ref, watch } from 'vue'
import Button from './Button.vue'
import Modal from './Modal.vue'
import SpecEditorBody from './SpecEditorBody.vue'
import { useToast } from '@/composables/useToast'

/** 下拉选项。 */
export interface SpecFieldOption {
  value: string
  label: string
}

/** 动态列表项的子字段。 */
export interface SpecListField {
  key: string
  label: string
  type: 'text' | 'number' | 'select' | 'checkbox' | 'textarea'
  placeholder?: string
  widthClass?: string
  rows?: number
  options?: SpecFieldOption[]
  /** text 输入的可选值提示（datalist）。 */
  datalist?: string[]
  /** 新增列表项时该字段的默认值（number 默认 0、text 默认空串、checkbox 默认 false）。 */
  default?: unknown
}

/** 通用表单字段定义（schema 驱动）。 */
export type SpecField =
  | { key: string; label: string; type: 'section' }
  | {
      key: string
      label: string
      type: 'text'
      placeholder?: string
      datalist?: string[]
      help?: string
      span?: number
      required?: boolean
    }
  | {
      key: string
      label: string
      type: 'number'
      placeholder?: string
      min?: number
      step?: number
      help?: string
      span?: number
    }
  | {
      key: string
      label: string
      type: 'textarea'
      placeholder?: string
      rows?: number
      help?: string
      span?: number
    }
  | {
      key: string
      label: string
      type: 'select'
      options: SpecFieldOption[]
      help?: string
      span?: number
    }
  | {
      key: string
      label: string
      type: 'multiselect'
      options: SpecFieldOption[]
      help?: string
      span?: number
    }
  | {
      key: string
      label: string
      type: 'list'
      addLabel: string
      fields: SpecListField[]
      span?: number
      /** 布局：cards（默认，卡片/单行）或 tabs（Tab 切换，如 Compose 多服务）。 */
      layout?: 'cards' | 'tabs'
    }
  | {
      key: string
      label: string
      type: 'checkbox'
      help?: string
      span?: number
    }

const props = withDefaults(
  defineProps<{
    open?: boolean
    title?: string
    submitLabel?: string
    width?: string
    fields: SpecField[]
    yamlHint?: string
    yamlPlaceholder?: string
    loading?: boolean
    /** 内嵌模式：不套 Modal 外壳（用于页面内联，如 Compose）。 */
    embedded?: boolean
    /** 表单 → YAML 文本（可抛错做校验）。 */
    formToYaml: (form: Record<string, any>) => string
    /** YAML 文本 → 表单对象。 */
    yamlToForm: (yamlText: string) => Record<string, any>
    /** 提交回调（收到 YAML 文本）。 */
    onSubmit: (yamlText: string) => Promise<void> | void
  }>(),
  {
    open: false,
    title: '创建资源',
    submitLabel: '提交',
    width: 'max-w-3xl',
    yamlHint: '',
    yamlPlaceholder: '# 在此粘贴或编辑资源定义 YAML\nName: my-resource',
    loading: false,
    embedded: false,
  },
)

const emit = defineEmits<{ 'update:open': [value: boolean]; saved: [name: string] }>()
const form = defineModel<Record<string, any>>('form', { required: true })
const bodyRef = ref<InstanceType<typeof SpecEditorBody> | null>(null)

const toast = useToast()
const saving = ref(false)

/** 编辑时预填 YAML / 获取当前 YAML（转发给 body）。 */
function setYaml(text: string): void {
  bodyRef.value?.setYaml(text)
}
function getYamlText(): string | null {
  return bodyRef.value?.getYamlText() ?? null
}
defineExpose({ setYaml, getYamlText })

watch(
  () => props.open,
  (open) => {
    if (open) {
      saving.value = false
      bodyRef.value?.setYaml('')
    }
  },
)

async function submit(): Promise<void> {
  const yaml = bodyRef.value?.getYamlText() ?? null
  if (!yaml) {
    toast.error('请输入资源定义')
    return
  }
  saving.value = true
  try {
    await props.onSubmit(yaml)
    emit('update:open', false)
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <!-- 内嵌模式：无 Modal 外壳，footer 用默认插槽自定义 -->
  <div v-if="embedded" class="space-y-4">
    <SpecEditorBody
      ref="bodyRef"
      v-model:form="form"
      :fields="fields"
      :form-to-yaml="formToYaml"
      :yaml-to-form="yamlToForm"
      :yaml-hint="yamlHint"
      :yaml-placeholder="yamlPlaceholder"
    />
    <div v-if="$slots.footer" class="flex flex-wrap items-center justify-end gap-2">
      <slot name="footer" />
    </div>
  </div>

  <!-- 弹窗模式 -->
  <Modal v-else :open="open" @update:open="(v) => emit('update:open', v)" :title="title" :width="width">
    <div v-if="loading" class="py-8 text-center text-sm text-slate-400">加载配置…</div>
    <div v-else class="max-h-[70vh] overflow-y-auto pr-1">
      <SpecEditorBody
        ref="bodyRef"
        v-model:form="form"
        :fields="fields"
        :form-to-yaml="formToYaml"
        :yaml-to-form="yamlToForm"
        :yaml-hint="yamlHint"
        :yaml-placeholder="yamlPlaceholder"
      />
    </div>

    <template #footer>
      <Button variant="secondary" @click="emit('update:open', false)">取消</Button>
      <Button :loading="saving" @click="submit">{{ submitLabel }}</Button>
    </template>
  </Modal>
</template>
