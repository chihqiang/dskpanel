<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import YAML from 'yaml'
import SpecEditorModal, { type SpecField } from '@/components/ui/SpecEditorModal.vue'
import { useToast } from '@/composables/useToast'
import {
  swarmCreateService,
  swarmUpdateService,
  swarmServiceInspect,
  swarmNetworks,
  swarmSecrets,
  swarmConfigs,
  swarmImages,
  type ServiceRequest,
  type SwarmNetworkItem,
  type SwarmSecretItem,
} from '@/api/swarm'

const props = defineProps<{
  open: boolean
  /** 编辑的服务 ID（空 = 创建）。 */
  serviceId?: string
}>()

const emit = defineEmits<{ 'update:open': [value: boolean]; saved: [name: string] }>()

const toast = useToast()

/** 通用编辑器实例（用于编辑时预填 YAML）。 */
const editorRef = ref<InstanceType<typeof SpecEditorModal> | null>(null)
const loading = ref(false)

/** 可选网络 / Secret / Config（供下拉选择）。 */
const networks = ref<SwarmNetworkItem[]>([])
const secretOptions = ref<SwarmSecretItem[]>([])
const configOptions = ref<SwarmSecretItem[]>([])
/** 集群镜像（含 tag），供下拉提示。 */
const imageOptions = ref<string[]>([])

/** 单个服务表单项。 */
interface ServiceFormItem {
  name: string
  image: string
  mode: 'replicated' | 'global'
  replicas: number
  portsText: string
  envText: string
  commandText: string
  restart: 'none' | 'on-failure' | 'any'
  mountsText: string
  networks: string[]
  secrets: string[]
  configs: string[]
  limitCpu: string
  limitMemory: string
  constraintsText: string
  labelsText: string
  updateParallelism: string
  updateDelay: string
  updateFailureAction: 'pause' | 'continue' | 'rollback'
}

function emptyService(): ServiceFormItem {
  return {
    name: '',
    image: '',
    mode: 'replicated',
    replicas: 1,
    portsText: '',
    envText: '',
    commandText: '',
    restart: 'any',
    mountsText: '',
    networks: [],
    secrets: [],
    configs: [],
    limitCpu: '',
    limitMemory: '',
    constraintsText: '',
    labelsText: '',
    updateParallelism: '',
    updateDelay: '',
    updateFailureAction: 'pause',
  }
}

/** 多服务表单模型。 */
const form = ref<Record<string, any>>({ services: [] })

/** ServiceSpec 对象（PascalCase 顶层键，与 Docker API / docker service inspect 一致）。 */
interface SpecObject {
  Name?: string
  Labels?: Record<string, string>
  TaskTemplate?: {
    ContainerSpec?: {
      Image?: string
      Env?: string[]
      Args?: string[]
      Mounts?: { Type?: string; Source?: string; Target?: string; ReadOnly?: boolean }[]
      Secrets?: { SecretName?: string }[]
      Configs?: { ConfigName?: string }[]
    }
    RestartPolicy?: { Condition?: string }
    Resources?: { Limits?: { NanoCPUs?: number; MemoryBytes?: number } }
    Placement?: { Constraints?: string[] }
    Networks?: { Target?: string }[]
  }
  Mode?: { Replicated?: { Replicas?: number }; Global?: Record<string, never> }
  EndpointSpec?: {
    Ports?: { TargetPort?: number; PublishedPort?: number; Protocol?: string; PublishMode?: string }[]
  }
  UpdateConfig?: { Parallelism?: number; Delay?: number; FailureAction?: string }
}

/** 加载网络 / Secret / Config / 镜像 选项。 */
async function loadOptions(): Promise<void> {
  try {
    const [n, s, c, imgs] = await Promise.all([
      swarmNetworks(),
      swarmSecrets(),
      swarmConfigs(),
      swarmImages(),
    ])
    networks.value = n.filter((x) => x.scope === 'swarm' || x.driver === 'overlay')
    secretOptions.value = s
    configOptions.value = c
    // 提取带 tag 的镜像名，去重后按名称排序。
    const tags = new Set<string>()
    for (const img of imgs) {
      if (img.repo_tags?.length) {
        for (const t of img.repo_tags) tags.add(t)
      }
    }
    imageOptions.value = [...tags].sort()
  } catch {
    // 选项加载失败不阻塞表单。
  }
}

function reset(): void {
  form.value = { services: [] }
}

/** 编辑时从 inspect 预填（原始 camelCase Spec）。 */
async function loadForEdit(id: string): Promise<void> {
  loading.value = true
  try {
    const raw = (await swarmServiceInspect(id)) as { Spec?: SpecObject }
    const spec = raw?.Spec ?? {}
    form.value = { services: [specToService(spec)] }
    editorRef.value?.setYaml(YAML.stringify(spec))
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    loading.value = false
  }
}

watch(
  () => props.open,
  (open) => {
    if (open) {
      reset()
      void loadOptions()
      if (props.serviceId) {
        void loadForEdit(props.serviceId)
      } else {
        // 创建：默认给一个服务。
        form.value = { services: [emptyService()] }
      }
    }
  },
)

/** 按行拆分文本 → 非空数组。 */
function parseLines(text: string): string[] {
  return text
    .split('\n')
    .map((s) => s.trim())
    .filter(Boolean)
}

/** 解析 key=value 多行文本 → 对象。 */
function parseKV(text: string): Record<string, string> {
  const out: Record<string, string> = {}
  for (const line of text.split('\n')) {
    const t = line.trim()
    if (!t) continue
    const idx = t.indexOf('=')
    if (idx > 0) out[t.slice(0, idx).trim()] = t.slice(idx + 1).trim()
  }
  return out
}

/** 解析端口文本（每行 8080:80/tcp）。 */
function parsePorts(text: string): { TargetPort: number; PublishedPort: number; Protocol: string; PublishMode: string }[] {
  const out: { TargetPort: number; PublishedPort: number; Protocol: string; PublishMode: string }[] = []
  for (const line of text.split('\n')) {
    const t = line.trim()
    if (!t) continue
    // [published]:target[/protocol]
    let protocol = 'tcp'
    let rest = t
    const slashIdx = t.lastIndexOf('/')
    if (slashIdx > 0) {
      protocol = t.slice(slashIdx + 1).trim() || 'tcp'
      rest = t.slice(0, slashIdx).trim()
    }
    const parts = rest.split(':')
    if (parts.length < 2) continue
    const target = Number(parts[parts.length - 1])
    const published = Number(parts[0])
    if (!target) continue
    out.push({ TargetPort: target, PublishedPort: published || undefined as unknown as number, Protocol: protocol, PublishMode: 'ingress' })
  }
  return out.filter((p) => p.PublishedPort)
}

/** 解析挂载文本（每行 type:source:target:ro）。 */
function parseMounts(text: string): { Type: string; Source?: string; Target: string; ReadOnly?: boolean }[] {
  const out: { Type: string; Source?: string; Target: string; ReadOnly?: boolean }[] = []
  for (const line of text.split('\n')) {
    const t = line.trim()
    if (!t) continue
    const parts = t.split(':')
    if (parts.length < 2) continue
    let type = parts[0] || 'volume'
    let idx = 1
    if (!['volume', 'bind', 'tmpfs'].includes(type)) {
      // 无类型前缀：默认为 volume，整体当 source:target 处理。
      type = 'volume'
      idx = 0
    }
    const readOnly = parts[parts.length - 1] === 'ro'
    const target = parts[parts.length - 2] || ''
    if (!target) continue
    const source = parts.slice(idx, parts.length - 2).join(':') || undefined
    out.push({ Type: type, Source: source, Target: target, ReadOnly: readOnly || undefined })
  }
  return out
}

/** 单个服务表单项 → ServiceSpec 对象。 */
function serviceToSpec(s: ServiceFormItem): SpecObject {
  const env = parseLines(s.envText)
  const command = parseLines(s.commandText)
  const constraints = parseLines(s.constraintsText)
  const labels = parseKV(s.labelsText)
  const mounts = parseMounts(s.mountsText)
  const ports = parsePorts(s.portsText)
  const spec: SpecObject = {
    Name: s.name,
    ...(Object.keys(labels).length ? { Labels: labels } : {}),
    TaskTemplate: {
      ContainerSpec: {
        Image: s.image,
        ...(env.length ? { Env: env } : {}),
        ...(command.length ? { Args: command } : {}),
        ...(mounts.length ? { Mounts: mounts } : {}),
        ...(s.secrets?.length ? { Secrets: s.secrets.map((n) => ({ SecretName: n })) } : {}),
        ...(s.configs?.length ? { Configs: s.configs.map((n) => ({ ConfigName: n })) } : {}),
      },
      ...(s.networks?.length ? { Networks: s.networks.map((n) => ({ Target: n })) } : {}),
      ...(s.restart ? { RestartPolicy: { Condition: s.restart } } : {}),
      ...(s.limitCpu || s.limitMemory
        ? {
            Resources: {
              Limits: {
                ...(s.limitCpu ? { NanoCPUs: Math.round(Number(s.limitCpu) * 1e9) } : {}),
                ...(s.limitMemory ? { MemoryBytes: Number(s.limitMemory) * 1024 * 1024 } : {}),
              },
            },
          }
        : {}),
      ...(constraints.length
        ? {
            Placement: { Constraints: constraints },
          }
        : {}),
    },
    Mode: s.mode === 'global' ? { Global: {} } : { Replicated: { Replicas: s.replicas } },
  }
  if (ports.length) {
    spec.EndpointSpec = { Ports: ports }
  }
  if (s.updateParallelism || s.updateDelay || s.updateFailureAction) {
    spec.UpdateConfig = {
      ...(s.updateParallelism ? { Parallelism: Number(s.updateParallelism) } : {}),
      ...(s.updateDelay ? { Delay: Number(s.updateDelay) * 1e9 } : {}),
      FailureAction: s.updateFailureAction,
    }
  }
  return spec
}

/** ServiceSpec 对象 → 单个服务表单项。 */
function specToService(spec: SpecObject): ServiceFormItem {
  const cs = spec.TaskTemplate?.ContainerSpec
  const limits = spec.TaskTemplate?.Resources?.Limits
  const uc = spec.UpdateConfig
  // 网络：由网络 ID 反查名称（找不到则保留原文）。
  const nidToName = new Map(networks.value.map((n) => [n.id, n.name]))
  return {
    name: spec.Name ?? '',
    image: cs?.Image ?? '',
    mode: spec.Mode?.Global ? 'global' : 'replicated',
    replicas: spec.Mode?.Replicated?.Replicas ?? 1,
    portsText: (spec.EndpointSpec?.Ports ?? [])
      .map((p) => `${p.PublishedPort ?? ''}:${p.TargetPort ?? ''}${p.Protocol && p.Protocol !== 'tcp' ? `/${p.Protocol}` : ''}`)
      .join('\n'),
    envText: (cs?.Env ?? []).join('\n'),
    commandText: (cs?.Args ?? []).join('\n'),
    restart: (spec.TaskTemplate?.RestartPolicy?.Condition as 'none' | 'on-failure' | 'any') || 'any',
    mountsText: (cs?.Mounts ?? [])
      .map((m) => [m.Type ?? 'volume', m.Source ?? '', m.Target ?? '', m.ReadOnly ? 'ro' : ''].filter(Boolean).join(':'))
      .join('\n'),
    networks: (spec.TaskTemplate?.Networks ?? []).map((n) => nidToName.get(n.Target ?? '') ?? n.Target ?? ''),
    secrets: (cs?.Secrets ?? []).map((s) => s.SecretName ?? ''),
    configs: (cs?.Configs ?? []).map((c) => c.ConfigName ?? ''),
    limitCpu: limits?.NanoCPUs ? String(limits.NanoCPUs / 1e9) : '',
    limitMemory: limits?.MemoryBytes ? String(Math.round(limits.MemoryBytes / 1024 / 1024)) : '',
    constraintsText: (spec.TaskTemplate?.Placement?.Constraints ?? []).join('\n'),
    labelsText: Object.entries(spec.Labels ?? {})
      .map(([k, v]) => `${k}=${v}`)
      .join('\n'),
    updateParallelism: uc?.Parallelism ? String(uc.Parallelism) : '',
    updateDelay: uc?.Delay ? String(Math.round(uc.Delay / 1e9)) : '',
    updateFailureAction: (uc?.FailureAction as 'pause' | 'continue' | 'rollback') || 'pause',
  }
}

/** 表单字段 schema（供通用 SpecEditorModal 渲染）：服务列表 tabs。 */
const fields = computed<SpecField[]>(() => [
  {
    key: 'services',
    label: '服务',
    type: 'list',
    addLabel: '添加服务',
    span: 6,
    layout: 'tabs',
    fields: [
      { key: 'name', label: '服务名', type: 'text', placeholder: '例如 web', widthClass: 'w-44' },
      {
        key: 'image',
        label: '镜像',
        type: 'text',
        placeholder: '例如 nginx:alpine',
        datalist: imageOptions.value,
        widthClass: 'flex-1',
      },
      {
        key: 'mode',
        label: '模式',
        type: 'select',
        widthClass: 'w-32',
        options: [
          { value: 'replicated', label: 'Replicated' },
          { value: 'global', label: 'Global' },
        ],
      },
      { key: 'replicas', label: '副本数', type: 'number', widthClass: 'w-24', default: 1 },
      {
        key: 'restart',
        label: '重启策略',
        type: 'select',
        widthClass: 'w-32',
        default: 'any',
        options: [
          { value: 'any', label: 'any' },
          { value: 'on-failure', label: 'on-failure' },
          { value: 'none', label: 'none' },
        ],
      },
      { key: 'portsText', label: '端口映射（每行 8080:80 或 8080:80/udp）', type: 'textarea', rows: 2, placeholder: '8080:80' },
      { key: 'mountsText', label: '卷挂载（每行 type:source:target 或 :ro 结尾）', type: 'textarea', rows: 2, placeholder: 'volume:myvol:/data' },
      { key: 'envText', label: '环境变量（每行 KEY=VALUE）', type: 'textarea', rows: 3, placeholder: 'TZ=Asia/Shanghai' },
      { key: 'commandText', label: '命令参数（每行一个）', type: 'textarea', rows: 2, placeholder: '--worker' },
      {
        key: 'networks',
        label: '网络',
        type: 'textarea',
        rows: 2,
        placeholder: 'ingress\nfrontend',
        // 多选用文本行代替，避免 tabs 内不支持 multiselect。
      },
      {
        key: 'secrets',
        label: 'Secret 挂载',
        type: 'textarea',
        rows: 2,
        placeholder: 'db_password',
      },
      {
        key: 'configs',
        label: 'Config 挂载',
        type: 'textarea',
        rows: 2,
        placeholder: 'nginx_conf',
      },
      { key: 'limitCpu', label: 'CPU 限制（核，留空不限）', type: 'text', placeholder: '例如 0.5', widthClass: 'w-28' },
      { key: 'limitMemory', label: '内存限制（MB，留空不限）', type: 'number', placeholder: '例如 512', widthClass: 'w-28' },
      { key: 'constraintsText', label: '节点约束（每行一个）', type: 'textarea', rows: 2, placeholder: 'node.role == manager' },
      { key: 'labelsText', label: '标签（每行 key=value）', type: 'textarea', rows: 2, placeholder: 'env=prod' },
      { key: 'updateParallelism', label: '更新并行度', type: 'number', widthClass: 'w-24' },
      { key: 'updateDelay', label: '更新间隔（秒）', type: 'number', widthClass: 'w-24' },
      {
        key: 'updateFailureAction',
        label: '失败策略',
        type: 'select',
        widthClass: 'w-32',
        default: 'pause',
        options: [
          { value: 'pause', label: 'pause' },
          { value: 'continue', label: 'continue' },
          { value: 'rollback', label: 'rollback' },
        ],
      },
    ],
  },
])

/** 表单 → 多文档 YAML（每个文档一个 ServiceSpec）。 */
function formToYaml(f: Record<string, any>): string {
  const services = (f.services ?? []) as ServiceFormItem[]
  if (!services.length) throw new Error('请至少添加一个服务')
  for (const s of services) {
    if (!s.name) throw new Error('请填写服务名')
    if (!s.image) throw new Error(`服务「${s.name}」请填写镜像`)
  }
  return services.map((s) => YAML.stringify(serviceToSpec(s))).join('---\n')
}

/** 多文档 YAML → 表单。 */
function yamlToForm(yamlText: string): Record<string, any> {
  const docs = YAML.parseAllDocuments(yamlText)
  const services: ServiceFormItem[] = []
  for (const doc of docs) {
    const spec = (doc.toJS() ?? {}) as SpecObject
    if (spec.Name) services.push(specToService(spec))
  }
  return { services: services.length ? services : [emptyService()] }
}

/** 提交：解析多文档 specs，逐个创建；编辑模式更新单个。 */
async function onSubmit(specText: string): Promise<void> {
  if (props.serviceId) {
    // 编辑：更新单个服务（取第一个 spec）。
    const docs = YAML.parseAllDocuments(specText)
    const spec = (docs[0]?.toJS() ?? {}) as SpecObject
    await swarmUpdateService(props.serviceId, { spec: YAML.stringify(spec) } satisfies ServiceRequest)
    toast.success('服务已更新')
    emit('update:open', false)
    emit('saved', spec.Name ?? '')
    return
  }
  // 创建：逐个创建。
  const docs = YAML.parseAllDocuments(specText)
  const specs = docs.map((d) => (d.toJS() ?? {}) as SpecObject).filter((s) => s.Name)
  if (!specs.length) {
    toast.error('请至少填写一个服务')
    return
  }
  let ok = 0
  let failed = 0
  const names: string[] = []
  for (const spec of specs) {
    try {
      await swarmCreateService({ spec: YAML.stringify(spec) } satisfies ServiceRequest)
      ok++
      if (spec.Name) names.push(spec.Name)
    } catch (err) {
      failed++
      toast.error(`服务「${spec.Name ?? '?'}」创建失败：${(err as Error).message}`)
    }
  }
  if (ok > 0) {
    toast.success(`已创建 ${ok} 个服务：${names.join('、')}`)
    emit('update:open', false)
    emit('saved', names[0] ?? '')
  } else {
    toast.error('所有服务均创建失败')
  }
}
</script>

<template>
  <SpecEditorModal
    ref="editorRef"
    :open="open"
    @update:open="(v) => emit('update:open', v)"
    v-model:form="form"
    :title="serviceId ? '更新服务' : '创建服务'"
    :submit-label="serviceId ? '保存' : '创建'"
    :fields="fields"
    :loading="loading"
    :form-to-yaml="formToYaml"
    :yaml-to-form="yamlToForm"
    :on-submit="onSubmit"
    yaml-hint="支持手写 YAML，或从已有服务导出：docker service inspect --format '{{json .Spec}}' 服务名"
    yaml-placeholder="# 在此粘贴或编辑 ServiceSpec YAML&#10;Name: my-service&#10;TaskTemplate:&#10;  ContainerSpec:&#10;    Image: nginx:alpine"
  />
</template>
