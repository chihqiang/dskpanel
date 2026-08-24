<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import YAML from 'yaml'
import { Rocket } from '@lucide/vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import Modal from '@/components/ui/Modal.vue'
import SpecEditorBody from '@/components/ui/SpecEditorBody.vue'
import type { SpecField } from '@/components/ui/SpecEditorModal.vue'
import ComposeProjects from '@/components/docker/ComposeProjects.vue'
import {
  validateCompose,
  deployComposeStream,
} from '@/api/compose'
import { useToast } from '@/composables/useToast'
import { composeTemplates } from '@/templates'

/** 部署弹窗内的编辑器实例。 */
const bodyRef = ref<InstanceType<typeof SpecEditorBody> | null>(null)

/** 项目列表组件实例（部署成功后刷新）。 */
const projectsRef = ref<InstanceType<typeof ComposeProjects> | null>(null)

/** 部署弹窗开关。 */
const deployOpen = ref(false)

/** 表单模型：项目名 + 服务列表 + 自定义网络/卷。 */
const form = ref<Record<string, any>>({
  name: '',
  services: [],
  networks: '',
  volumes: '',
})
const result = ref<{ ok: boolean; message: string } | null>(null)

/** 模板选择：选择即自动回填表单（immediate 首次加载默认模板）。 */
const selectedTemplate = ref(composeTemplates[0]?.name ?? '')
watch(selectedTemplate, (name) => {
  const tpl = composeTemplates.find((t) => t.name === name)
  if (!tpl) return
  form.value = yamlToForm(tpl.yaml)
  result.value = null
}, { immediate: true })

/** Compose 表单字段 schema。 */
const fields: SpecField[] = [
  { key: 'name', label: '项目名', type: 'text', placeholder: '例如 myapp（可选）', span: 6 },
  {
    key: 'services',
    label: '服务',
    type: 'list',
    addLabel: '添加服务',
    span: 6,
    layout: 'tabs',
    fields: [
      { key: 'name', label: '服务名', type: 'text', placeholder: '例如 web', widthClass: 'w-44' },
      { key: 'image', label: '镜像', type: 'text', placeholder: '例如 nginx:latest', widthClass: 'flex-1' },
      {
        key: 'restart',
        label: '重启策略',
        type: 'select',
        widthClass: 'w-36',
        options: [
          { value: 'no', label: 'no' },
          { value: 'always', label: 'always' },
          { value: 'on-failure', label: 'on-failure' },
          { value: 'unless-stopped', label: 'unless-stopped' },
        ],
      },
      { key: 'ports', label: '端口映射（每行一个，如 8080:80）', type: 'textarea', rows: 3, placeholder: '8080:80' },
      { key: 'environment', label: '环境变量（每行 KEY=VALUE）', type: 'textarea', rows: 3, placeholder: 'NGINX_HOST=example.com' },
      { key: 'volumes', label: '卷挂载（每行一个，如 ./data:/data）', type: 'textarea', rows: 2, placeholder: './data:/data' },
    ],
  },
  { key: 'networks', label: '自定义网络（每行一个，可选）', type: 'textarea', span: 3, rows: 3, placeholder: 'frontend\nbackend' },
  { key: 'volumes', label: '命名卷（每行一个，可选）', type: 'textarea', span: 3, rows: 3, placeholder: 'dbdata' },
]

/** 表单 → Compose YAML。 */
function formToYaml(f: Record<string, any>): string {
  if (!f.services?.length) {
    throw new Error('至少需要一个服务')
  }
  const project: Record<string, any> = {}
  if (f.name) project.name = f.name
  project.services = {}
  for (const s of f.services) {
    if (!s.name || !s.image) {
      throw new Error(`服务「${s.name || '?'}」需填写名称与镜像`)
    }
    const svc: Record<string, any> = { image: s.image }
    if (s.ports?.trim()) {
      svc.ports = s.ports
        .split('\n')
        .map((t: string) => t.trim())
        .filter(Boolean)
    }
    if (s.environment?.trim()) {
      svc.environment = s.environment
        .split('\n')
        .map((t: string) => t.trim())
        .filter(Boolean)
    }
    if (s.volumes?.trim()) {
      svc.volumes = s.volumes
        .split('\n')
        .map((t: string) => t.trim())
        .filter(Boolean)
    }
    if (s.restart) svc.restart = s.restart
    project.services[s.name] = svc
  }
  if (f.networks?.trim()) {
    project.networks = {}
    for (const n of f.networks.split('\n').map((t: string) => t.trim()).filter(Boolean)) {
      project.networks[n] = null
    }
  }
  if (f.volumes?.trim()) {
    project.volumes = {}
    for (const v of f.volumes.split('\n').map((t: string) => t.trim()).filter(Boolean)) {
      project.volumes[v] = null
    }
  }
  return YAML.stringify(project)
}

/** Compose YAML → 表单。 */
function yamlToForm(yamlText: string): Record<string, any> {
  const parsed = YAML.parse(yamlText) || {}
  const services: Record<string, any>[] = []
  for (const [name, s] of Object.entries<any>(parsed.services ?? {})) {
    services.push({
      name,
      image: s.image ?? '',
      restart: s.restart ?? 'no',
      ports: (s.ports ?? []).join('\n'),
      environment: (s.environment ?? []).join('\n'),
      volumes: (s.volumes ?? []).join('\n'),
    })
  }
  return {
    name: parsed.name ?? '',
    services,
    networks: parsed.networks ? Object.keys(parsed.networks).join('\n') : '',
    volumes: parsed.volumes ? Object.keys(parsed.volumes).join('\n') : '',
  }
}

const validating = ref(false)
const deploying = ref(false)
const deployLines = ref<string[]>([])
const deployDone = ref(false)
const deployOk = ref(false)
let stopDeploy: (() => void) | null = null

const toast = useToast()

/** 取当前编辑器内容（表单模式自动转 YAML）。 */
function currentContent(): string | null {
  return bodyRef.value?.getYamlText() ?? null
}

async function onValidate(): Promise<void> {
  const content = currentContent()
  if (!content) {
    toast.error('请先填写 Compose 定义')
    return
  }
  validating.value = true
  result.value = null
  try {
    result.value = await validateCompose(content)
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    validating.value = false
  }
}

function onDeploy(): void {
  const content = currentContent()
  if (!content) {
    toast.error('请先填写 Compose 定义')
    return
  }
  deploying.value = true
  deployLines.value = []
  deployDone.value = false
  result.value = null

  stopDeploy = deployComposeStream(
    content,
    (line) => {
      deployLines.value.push(line)
      scrollDeployToBottom()
    },
    async (success) => {
      deployOk.value = success
      deployDone.value = true
      deploying.value = false
      if (success) {
        toast.success('Compose 应用部署成功')
        // 部署成功后刷新下方项目列表。
        void projectsRef.value?.load()
      }
    },
    (msg) => {
      toast.error(msg)
      deploying.value = false
    },
  )
}

function reset(): void {
  stopDeploy?.()
  stopDeploy = null
  // 预填示例表单（模板库首个案例，通过触发 selectedTemplate watch 加载）。
  selectedTemplate.value = composeTemplates[0]?.name ?? ''
  bodyRef.value?.setYaml('')
  deployLines.value = []
  deployDone.value = false
  result.value = null
}

watch(
  () => form.value,
  () => {
    // 表单变化时清空旧的部署/校验结果，避免与内容不一致。
    result.value = null
  },
)

/** 部署输出自动滚动到底部。 */
const deployBox = ref<HTMLElement | null>(null)
function scrollDeployToBottom(): void {
  requestAnimationFrame(() => {
    if (deployBox.value) deployBox.value.scrollTop = deployBox.value.scrollHeight
  })
}

onBeforeUnmount(() => {
  stopDeploy?.()
  stopDeploy = null
})
</script>

<template>
  <div class="space-y-5">
    <!-- 项目列表（参考 /swarm/services：DataTable 列表 + 工具栏按钮） -->
    <ComposeProjects ref="projectsRef">
      <template #deploy>
        <Button size="sm" @click="deployOpen = true">
          <Rocket class="h-3.5 w-3.5" />
          部署编排
        </Button>
      </template>
    </ComposeProjects>

    <!-- 部署编排弹窗（参考 /swarm/services：创建类操作走 Modal） -->
    <Modal :open="deployOpen" @update:open="deployOpen = $event" title="部署 Compose" width="max-w-3xl">
      <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
        <span class="text-sm font-medium text-slate-700 dark:text-slate-200">Compose 定义</span>
        <div class="flex items-center gap-2">
          <span class="text-xs text-slate-400">模板：</span>
          <select v-model="selectedTemplate" class="input input-sm w-44">
            <option v-for="t in composeTemplates" :key="t.name" :value="t.name">{{ t.name }}</option>
          </select>
        </div>
      </div>

      <SpecEditorBody
        ref="bodyRef"
        v-model:form="form"
        :fields="fields"
        :form-to-yaml="formToYaml"
        :yaml-to-form="yamlToForm"
        yaml-placeholder="# 在此粘贴或编辑 Compose 文件 YAML&#10;services:&#10;  web:&#10;    image: nginx:latest"
      />

      <!-- 校验结果 -->
      <div v-if="result" class="mt-3 flex items-start gap-2">
        <Badge :variant="result.ok ? 'green' : 'red'">{{ result.ok ? '校验成功' : '校验失败' }}</Badge>
        <pre class="min-w-0 flex-1 whitespace-pre-wrap break-all rounded-md bg-slate-900 p-3 text-xs text-slate-100">{{ result.message }}</pre>
      </div>

      <!-- 部署实时回显 -->
      <div v-if="deploying || deployDone || deployLines.length > 0" class="mt-3 space-y-2">
        <div class="flex items-center gap-2">
          <span class="text-sm font-medium text-slate-700 dark:text-slate-200">部署输出</span>
          <Badge v-if="deploying" variant="blue">部署中...</Badge>
          <Badge v-else-if="deployDone" :variant="deployOk ? 'green' : 'red'">
            {{ deployOk ? '部署成功' : '部署失败' }}
          </Badge>
        </div>
        <div
          ref="deployBox"
          class="max-h-64 overflow-y-auto rounded-md bg-slate-900 p-3 font-mono text-xs text-slate-100"
        >
          <div v-for="(line, idx) in deployLines" :key="idx" class="whitespace-pre-wrap break-all">{{ line }}</div>
          <div v-if="deploying && deployLines.length === 0" class="text-slate-500">部署中...</div>
        </div>
      </div>

      <template #footer>
        <Button variant="secondary" @click="reset">重置</Button>
        <Button variant="secondary" :loading="validating" @click="onValidate">校验</Button>
        <Button :loading="deploying" @click="onDeploy">部署</Button>
      </template>
    </Modal>
  </div>
</template>
