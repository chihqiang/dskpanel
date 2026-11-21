<script setup lang="ts">
import { ref, watch } from 'vue'
import { Plus, Trash2, X } from '@lucide/vue'
import Button from './Button.vue'
import { useToast } from '@/composables/useToast'
import type { SpecField } from './SpecEditorModal.vue'

const props = withDefaults(
  defineProps<{
    fields: SpecField[]
    yamlHint?: string
    yamlPlaceholder?: string
    /** 表单 → YAML 文本（可抛错做校验）。 */
    formToYaml: (form: Record<string, any>) => string
    /** YAML 文本 → 表单对象。 */
    yamlToForm: (yamlText: string) => Record<string, any>
  }>(),
  { yamlHint: '', yamlPlaceholder: '# 在此粘贴或编辑资源定义 YAML\nName: my-resource' },
)

const form = defineModel<Record<string, any>>('form', { required: true })
const toast = useToast()

const editMode = ref<'form' | 'yaml'>('form')
const yamlText = ref('')

/** 栅格占位 → 静态 Tailwind 类（必须字面量，动态拼接不会被扫描）。 */
function spanClass(span?: number): string {
  switch (span) {
    case 2:
      return 'col-span-2'
    case 4:
      return 'col-span-4'
    case 6:
      return 'col-span-6'
    default:
      return 'col-span-3'
  }
}

/** 新增一个列表项（生成默认值）。 */
function emptyListItem(field: Extract<SpecField, { type: 'list' }>): Record<string, unknown> {
  const item: Record<string, unknown> = {}
  for (const f of field.fields) {
    if (f.default !== undefined) {
      item[f.key] = f.default
    } else {
      item[f.key] = f.type === 'checkbox' ? false : f.type === 'number' ? 0 : ''
    }
  }
  return item
}

function addListItem(field: Extract<SpecField, { type: 'list' }>): void {
  form.value[field.key].push(emptyListItem(field))
  // tabs 布局：新增后激活最后一项。
  if (field.layout === 'tabs') {
    activeTab.value[field.key] = form.value[field.key].length - 1
  }
}

function removeListItem(key: string, idx: number | string): void {
  form.value[key].splice(Number(idx), 1)
  // tabs 布局：修正激活项索引。
  const i = Number(idx)
  const len = form.value[key].length
  if (activeTab.value[key] === i || activeTab.value[key] > len - 1) {
    activeTab.value[key] = Math.max(0, Math.min(len - 1, i - 1))
  }
}

/** 每个 list 字段的激活 tab 索引（tabs 布局用）。 */
const activeTab = ref<Record<string, number>>({})

// 表单数据变化时：若 tabs 列表有值但尚未设置激活项，自动激活第一个。
watch(
  () => form.value,
  (val) => {
    for (const field of props.fields) {
      if (field.type === 'list' && field.layout === 'tabs') {
        const items = val[field.key] as unknown[] | undefined
        if (items?.length && activeTab.value[field.key] === undefined) {
          activeTab.value[field.key] = 0
        }
      }
    }
  },
  { deep: true, immediate: true },
)

/** tab 标题：优先取首个 text 子字段值（如服务名），否则「服务 N」。 */
function tabLabel(field: Extract<SpecField, { type: 'list' }>, item: Record<string, any>, i: number): string {
  const nameField = field.fields.find((f) => f.type === 'text')
  const val = nameField ? String(item[nameField.key] ?? '').trim() : ''
  return val || `${field.fields[0]?.label || '项'} ${i + 1}`
}

/** 列表项是否含 textarea（是则垂直堆叠布局）。 */
function listHasTextarea(field: Extract<SpecField, { type: 'list' }>): boolean {
  return field.fields.some((f) => f.type === 'textarea')
}

/** 当前表单 → YAML 文本（失败返回 null）。 */
function currentYaml(): string | null {
  try {
    return props.formToYaml(form.value)
  } catch (err) {
    toast.error((err as Error).message)
    return null
  }
}

function switchMode(mode: 'form' | 'yaml'): void {
  if (mode === editMode.value) return
  if (mode === 'yaml') {
    // 切换宽松处理：即使表单校验未通过（如空表单）也允许切换到 YAML，
    // 内容留空供直接手写/粘贴；校验在提交时（currentYaml）才严格拦截。
    try {
      yamlText.value = props.formToYaml(form.value)
    } catch {
      yamlText.value = ''
    }
  } else {
    try {
      const parsed = props.yamlToForm(yamlText.value)
      form.value = parsed ?? {}
    } catch (err) {
      toast.error(`解析失败，无法切换：${(err as Error).message}`)
      return
    }
  }
  editMode.value = mode
}

/** 供父组件编辑时预填 YAML。 */
function setYaml(text: string): void {
  yamlText.value = text
}
/** 获取当前 YAML 文本（表单模式自动转 YAML；YAML 模式取原文）。 */
function getYamlText(): string | null {
  if (editMode.value === 'yaml') {
    const t = yamlText.value.trim()
    return t || null
  }
  return currentYaml()
}
defineExpose({ setYaml, getYamlText })
</script>

<template>
  <div class="space-y-4">
    <!-- 模式切换：表单 / YAML -->
    <div class="flex items-center justify-between">
      <div class="inline-flex rounded-lg bg-slate-100 p-0.5 dark:bg-slate-700">
        <button
          class="rounded-md px-3 py-1.5 text-sm transition-colors"
          :class="editMode === 'form' ? 'bg-white text-slate-900 shadow-sm dark:bg-slate-900 dark:text-slate-100' : 'text-slate-500 hover:text-slate-700 dark:text-slate-400'"
          @click="switchMode('form')"
        >表单</button>
        <button
          class="rounded-md px-3 py-1.5 text-sm transition-colors"
          :class="editMode === 'yaml' ? 'bg-white text-slate-900 shadow-sm dark:bg-slate-900 dark:text-slate-100' : 'text-slate-500 hover:text-slate-700 dark:text-slate-400'"
          @click="switchMode('yaml')"
        >YAML</button>
      </div>
      <span class="text-xs text-slate-400">
        {{ editMode === 'form' ? '表单配置，提交时自动转换为 YAML' : '直接编辑资源定义 YAML' }}
      </span>
    </div>

    <!-- 表单模式 -->
    <template v-if="editMode === 'form'">
      <div class="grid grid-cols-6 gap-4">
        <template v-for="field in fields" :key="field.key">
          <!-- 分区标题 -->
          <div v-if="field.type === 'section'" class="col-span-6">
            <p class="border-b border-slate-200 pb-1 text-sm font-medium text-slate-700 dark:border-slate-700 dark:text-slate-200">
              {{ field.label }}
            </p>
          </div>

          <!-- 文本 -->
          <div v-else-if="field.type === 'text'" :class="spanClass(field.span)">
            <label class="mb-1.5 block text-sm text-slate-500">
              {{ field.label }}<span v-if="field.required" class="text-red-500"> *</span>
            </label>
            <input v-model="form[field.key]" class="input" :list="`datalist-${field.key}`" :placeholder="field.placeholder" />
            <datalist v-if="field.datalist?.length" :id="`datalist-${field.key}`">
              <option v-for="opt in field.datalist" :key="opt" :value="opt" />
            </datalist>
            <p v-if="field.help" class="mt-1 text-xs text-slate-400">{{ field.help }}</p>
          </div>

          <!-- 数字 -->
          <div v-else-if="field.type === 'number'" :class="spanClass(field.span)">
            <label class="mb-1.5 block text-sm text-slate-500">{{ field.label }}</label>
            <input v-model.number="form[field.key]" type="number" :min="field.min" :step="field.step ?? 1" class="input" :placeholder="field.placeholder" />
            <p v-if="field.help" class="mt-1 text-xs text-slate-400">{{ field.help }}</p>
          </div>

          <!-- 文本域 -->
          <div v-else-if="field.type === 'textarea'" :class="spanClass(field.span)">
            <label class="mb-1.5 block text-sm text-slate-500">{{ field.label }}</label>
            <textarea v-model="form[field.key]" class="input font-mono" :rows="field.rows ?? 5" :placeholder="field.placeholder" />
            <p v-if="field.help" class="mt-1 text-xs text-slate-400">{{ field.help }}</p>
          </div>

          <!-- 单选 -->
          <div v-else-if="field.type === 'select'" :class="spanClass(field.span)">
            <label class="mb-1.5 block text-sm text-slate-500">{{ field.label }}</label>
            <select v-model="form[field.key]" class="input">
              <option v-for="opt in field.options" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
            </select>
            <p v-if="field.help" class="mt-1 text-xs text-slate-400">{{ field.help }}</p>
          </div>

          <!-- 多选 -->
          <div v-else-if="field.type === 'multiselect'" :class="spanClass(field.span)">
            <label class="mb-1.5 block text-sm text-slate-500">{{ field.label }}</label>
            <select v-model="form[field.key]" multiple class="input h-24">
              <option v-for="opt in field.options" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
            </select>
            <p v-if="field.help" class="mt-1 text-xs text-slate-400">{{ field.help }}</p>
          </div>

          <!-- 动态列表 -->
          <div v-else-if="field.type === 'list'" class="col-span-6">
            <label class="mb-1.5 block text-sm text-slate-500">{{ field.label }}</label>

            <!-- Tabs 布局（如 Compose 多服务） -->
            <template v-if="field.layout === 'tabs'">
              <!-- Tab 栏 -->
              <div class="mb-3 flex flex-wrap items-center gap-1.5">
                <template v-for="(item, i) in form[field.key]" :key="i">
                  <span
                    class="group inline-flex max-w-[180px] items-center gap-1 rounded-md border px-2.5 py-1 text-sm transition-colors"
                    :class="
                      activeTab[field.key] === i
                        ? 'border-blue-500 bg-blue-50 text-blue-700 dark:bg-blue-900/40 dark:text-blue-200'
                        : 'cursor-pointer border-slate-200 text-slate-600 hover:border-slate-300 hover:bg-slate-50 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-700'
                    "
                    @click="activeTab[field.key] = Number(i)"
                  >
                    <span class="truncate">{{ tabLabel(field, item, Number(i)) }}</span>
                    <button
                      class="shrink-0 rounded p-0.5 text-slate-400 opacity-0 transition-opacity hover:text-red-500 group-hover:opacity-100"
                      aria-label="删除服务"
                      @click.stop="removeListItem(field.key, Number(i))"
                    >
                      <X class="h-3.5 w-3.5" />
                    </button>
                  </span>
                </template>
                <Button variant="secondary" size="sm" @click="addListItem(field)">
                  <Plus class="h-3.5 w-3.5" />
                  {{ field.addLabel }}
                </Button>
              </div>

              <!-- 当前 tab 内容（垂直布局：顶部非 textarea 字段 + textarea 字段） -->
              <div
                v-if="form[field.key][activeTab[field.key]]"
                class="space-y-3 rounded-lg border border-slate-200 p-3 dark:border-slate-700"
              >
                <div class="flex flex-wrap items-center gap-2">
                  <template
                    v-for="sub in field.fields.filter((f) => f.type !== 'textarea')"
                    :key="sub.key"
                  >
                    <div class="flex items-center gap-1.5">
                      <span class="shrink-0 text-xs text-slate-400">{{ sub.label }}</span>
                      <input
                        v-if="sub.type === 'text'"
                        v-model="form[field.key][activeTab[field.key]][sub.key]"
                        class="input font-mono"
                        :list="sub.datalist?.length ? `datalist-${field.key}-${sub.key}` : undefined"
                        :class="sub.widthClass"
                        :placeholder="sub.placeholder"
                      />
                      <datalist
                        v-if="sub.type === 'text' && sub.datalist?.length"
                        :id="`datalist-${field.key}-${sub.key}`"
                      >
                        <option v-for="opt in sub.datalist" :key="opt" :value="opt" />
                      </datalist>
                      <input
                        v-else-if="sub.type === 'number'"
                        v-model.number="form[field.key][activeTab[field.key]][sub.key]"
                        type="number"
                        class="input"
                        :class="sub.widthClass"
                        :placeholder="sub.placeholder"
                      />
                      <select
                        v-else-if="sub.type === 'select'"
                        v-model="form[field.key][activeTab[field.key]][sub.key]"
                        class="input"
                        :class="sub.widthClass"
                      >
                        <option v-for="opt in sub.options" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
                      </select>
                      <label
                        v-else-if="sub.type === 'checkbox'"
                        class="flex cursor-pointer items-center gap-1 text-sm text-slate-500"
                      >
                        <input
                          v-model="form[field.key][activeTab[field.key]][sub.key]"
                          type="checkbox"
                          class="h-4 w-4 rounded border-slate-300 text-blue-600"
                        />
                        {{ sub.label }}
                      </label>
                    </div>
                  </template>
                </div>
                <template
                  v-for="sub in field.fields.filter((f) => f.type === 'textarea')"
                  :key="sub.key"
                >
                  <div>
                    <label class="mb-1 block text-xs text-slate-400">{{ sub.label }}</label>
                    <textarea
                      v-model="form[field.key][activeTab[field.key]][sub.key]"
                      class="input w-full font-mono"
                      :rows="sub.rows ?? 3"
                      :placeholder="sub.placeholder"
                    />
                  </div>
                </template>
              </div>
              <p v-else class="py-4 text-center text-sm text-slate-400">暂无{{ field.label }}，点击「{{ field.addLabel }}」添加</p>
            </template>

            <!-- 垂直堆叠（含 textarea 子字段，如 Compose 服务，cards 布局） -->
            <template v-else-if="listHasTextarea(field)">
              <div
                v-for="(item, i) in form[field.key]"
                :key="i"
                class="mb-2 space-y-2 rounded-lg border border-slate-200 p-3 dark:border-slate-700"
              >
                <div class="flex flex-wrap items-center gap-2">
                  <template v-for="sub in field.fields.filter((f) => f.type !== 'textarea')" :key="sub.key">
                    <input
                      v-if="sub.type === 'text'"
                      v-model="item[sub.key]"
                      class="input font-mono"
                      :class="sub.widthClass"
                      :placeholder="sub.placeholder"
                    />
                    <select v-else-if="sub.type === 'select'" v-model="item[sub.key]" class="input" :class="sub.widthClass">
                      <option v-for="opt in sub.options" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
                    </select>
                    <label v-else-if="sub.type === 'checkbox'" class="flex shrink-0 cursor-pointer items-center gap-1 text-sm text-slate-500">
                      <input v-model="item[sub.key]" type="checkbox" class="h-4 w-4 rounded border-slate-300 text-blue-600" />
                      {{ sub.label }}
                    </label>
                  </template>
                  <button
                    class="ml-auto rounded-md p-1.5 text-slate-400 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/30"
                    aria-label="删除服务"
                    @click="removeListItem(field.key, i)"
                  >
                    <Trash2 class="h-4 w-4" />
                  </button>
                </div>
                <template v-for="sub in field.fields.filter((f) => f.type === 'textarea')" :key="sub.key">
                  <div>
                    <label class="mb-1 block text-xs text-slate-400">{{ sub.label }}</label>
                    <textarea v-model="item[sub.key]" class="input w-full font-mono" :rows="sub.rows ?? 3" :placeholder="sub.placeholder" />
                  </div>
                </template>
              </div>
            </template>

            <!-- 横向单行 -->
            <template v-else>
              <div v-for="(item, i) in form[field.key]" :key="i" class="mb-2 flex items-center gap-2">
                <template v-for="sub in field.fields" :key="sub.key">
                  <input
                    v-if="sub.type === 'text'"
                    v-model="item[sub.key]"
                    class="input font-mono"
                    :class="sub.widthClass"
                    :placeholder="sub.placeholder"
                  />
                  <input
                    v-else-if="sub.type === 'number'"
                    v-model.number="item[sub.key]"
                    type="number"
                    class="input"
                    :class="sub.widthClass"
                    :placeholder="sub.placeholder"
                  />
                  <select v-else-if="sub.type === 'select'" v-model="item[sub.key]" class="input" :class="sub.widthClass">
                    <option v-for="opt in sub.options" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
                  </select>
                  <label v-else-if="sub.type === 'checkbox'" class="flex shrink-0 cursor-pointer items-center gap-1 text-sm text-slate-500">
                    <input v-model="item[sub.key]" type="checkbox" class="h-4 w-4 rounded border-slate-300 text-blue-600" />
                    {{ sub.label }}
                  </label>
                </template>
                <button
                  class="rounded-md p-1.5 text-slate-400 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/30"
                  aria-label="删除"
                  @click="removeListItem(field.key, i)"
                >
                  <Trash2 class="h-4 w-4" />
                </button>
              </div>
            </template>

            <Button
              v-if="field.layout !== 'tabs'"
              variant="secondary"
              size="sm"
              @click="addListItem(field)"
            >
              <Plus class="h-3.5 w-3.5" />
              {{ field.addLabel }}
            </Button>
          </div>

          <!-- 复选框 -->
          <div v-else-if="field.type === 'checkbox'" :class="spanClass(field.span)">
            <label class="flex cursor-pointer items-center gap-1.5 text-sm text-slate-500">
              <input v-model="form[field.key]" type="checkbox" class="h-4 w-4 rounded border-slate-300 text-blue-600" />
              {{ field.label }}
            </label>
            <p v-if="field.help" class="mt-1 text-xs text-slate-400">{{ field.help }}</p>
          </div>
        </template>
      </div>
    </template>

    <!-- YAML 模式 -->
    <template v-else>
      <div class="space-y-2">
        <textarea
          v-model="yamlText"
          class="input h-[52vh] w-full resize-none font-mono text-xs leading-relaxed"
          :placeholder="yamlPlaceholder"
          spellcheck="false"
        />
        <p v-if="yamlHint" class="text-xs text-slate-400">{{ yamlHint }}</p>
      </div>
    </template>
  </div>
</template>
