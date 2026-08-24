<script setup lang="ts">
import { ref, watch } from 'vue'
import YAML from 'yaml'
import SpecEditorModal, { type SpecField } from '@/components/ui/SpecEditorModal.vue'
import { useToast } from '@/composables/useToast'
import { createContainer, type CreateContainerRequest } from '@/api/container'

const props = defineProps<{ open: boolean }>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  created: []
}>()

const toast = useToast()

/** 表单模型（扁平结构，与 CreateContainerRequest 对齐）。 */
const form = ref<Record<string, any>>({
  name: '',
  image: '',
  commandText: '',
  entrypointText: '',
  envText: '',
  labelsText: '',
  bindsText: '',
  portsText: '',
  network: 'bridge',
  restart_policy: '',
  auto_remove: false,
  tty: false,
  open_stdin: false,
  hostname: '',
  user: '',
  working_dir: '',
  capAddText: '',
  capDropText: '',
  memoryMb: '',
  cpuCores: '',
  cpuset_cpus: '',
  envFileText: '',
  extraHostsText: '',
})

/** 逗号/空格/换行分隔 → 数组。 */
function splitList(v: string): string[] | undefined {
  const list = v
    .split(/[\s,]+/)
    .map((s) => s.trim())
    .filter(Boolean)
  return list.length ? list : undefined
}

/** key=value 多行 → 对象。 */
function kvToObject(text: string): Record<string, string> | undefined {
  const out: Record<string, string> = {}
  for (const line of text.split('\n')) {
    const t = line.trim()
    if (!t) continue
    const idx = t.indexOf('=')
    if (idx > 0) out[t.slice(0, idx).trim()] = t.slice(idx + 1).trim()
  }
  return Object.keys(out).length ? out : undefined
}

/** 端口文本（每行 host:container[:protocol]）→ PortMapping[]。 */
function parsePorts(text: string): { container_port: number; host_port: number; protocol?: string }[] {
  const out: { container_port: number; host_port: number; protocol?: string }[] = []
  for (const line of text.split('\n')) {
    const t = line.trim()
    if (!t) continue
    let protocol: string | undefined
    let rest = t
    const slashIdx = t.lastIndexOf('/')
    if (slashIdx > 0) {
      protocol = t.slice(slashIdx + 1).trim() || undefined
      rest = t.slice(0, slashIdx).trim()
    }
    const parts = rest.split(':')
    if (parts.length < 2) continue
    const host = Number(parts[0])
    const container = Number(parts[1])
    if (!container) continue
    out.push({ container_port: container, host_port: host || 0, protocol })
  }
  return out
}

/** 表单字段 schema。 */
const fields: SpecField[] = [
  { key: 'name', label: '名称（可选）', type: 'text', span: 3, placeholder: 'my-container' },
  { key: 'image', label: '镜像', type: 'text', span: 3, required: true, placeholder: '例如 nginx:latest' },
  {
    key: 'network',
    label: '网络',
    type: 'text',
    span: 2,
    placeholder: 'bridge',
    help: '网络名（默认 bridge）',
  },
  {
    key: 'restart_policy',
    label: '重启策略',
    type: 'select',
    span: 2,
    options: [
      { value: '', label: '无' },
      { value: 'no', label: 'no' },
      { value: 'always', label: 'always' },
      { value: 'on-failure', label: 'on-failure' },
      { value: 'unless-stopped', label: 'unless-stopped' },
    ],
  },
  { key: 'commandText', label: '命令（每行一个参数）', type: 'textarea', span: 3, rows: 3, placeholder: '--worker' },
  { key: 'entrypointText', label: '入口点（每行一个参数）', type: 'textarea', span: 3, rows: 3, placeholder: 'docker-entrypoint.sh' },
  { key: 'portsText', label: '端口映射（每行 宿主机:容器）', type: 'textarea', span: 3, rows: 3, placeholder: '8080:80' },
  { key: 'bindsText', label: '卷挂载（每行 宿主机:容器）', type: 'textarea', span: 3, rows: 3, placeholder: '/host/path:/container/path' },
  { key: 'envText', label: '环境变量（每行 KEY=VALUE）', type: 'textarea', span: 3, rows: 4, placeholder: 'TZ=Asia/Shanghai' },
  { key: 'labelsText', label: '标签（每行 key=value）', type: 'textarea', span: 3, rows: 4, placeholder: 'app=web' },
  { key: 'hostname', label: '主机名', type: 'text', span: 2, placeholder: '如 my-host' },
  { key: 'user', label: '运行用户', type: 'text', span: 2, placeholder: '如 root / 1000:1000' },
  { key: 'working_dir', label: '工作目录', type: 'text', span: 2, placeholder: '如 /app' },
  { key: 'memoryMb', label: '内存限制（MB）', type: 'number', span: 2, min: 0, placeholder: '如 512' },
  { key: 'cpuCores', label: 'CPU 核数', type: 'number', span: 2, min: 0, step: 0.1, placeholder: '如 0.5' },
  { key: 'cpuset_cpus', label: 'CPU 亲和（cpuset）', type: 'text', span: 2, placeholder: '如 0-1' },
  { key: 'capAddText', label: '附加内核能力（空格/逗号分隔）', type: 'text', span: 3, placeholder: 'SYS_PTRACE NET_ADMIN' },
  { key: 'capDropText', label: '移除内核能力', type: 'text', span: 3, placeholder: 'ALL 或 NET_RAW' },
  { key: 'envFileText', label: '环境变量文件（宿主机路径，空格分隔）', type: 'text', span: 3, placeholder: '/tmp/app.env' },
  { key: 'extraHostsText', label: '额外 hosts（空格分隔）', type: 'text', span: 3, placeholder: 'db:192.168.1.10' },
  { key: 'auto_remove', label: '退出后自动删除', type: 'checkbox', span: 2 },
  { key: 'tty', label: '分配 TTY', type: 'checkbox', span: 2 },
  { key: 'open_stdin', label: '打开标准输入', type: 'checkbox', span: 2 },
]

/** 表单 → YAML 文本（友好格式，字段与请求对齐）。 */
function formToYaml(f: Record<string, any>): string {
  if (!f.image) throw new Error('请填写镜像')
  const req: Record<string, any> = {
    name: f.name?.trim() || undefined,
    image: f.image.trim(),
    network: f.network?.trim() || undefined,
    restart_policy: f.restart_policy || undefined,
    command: splitList(f.commandText),
    entrypoint: splitList(f.entrypointText),
    env: splitList(f.envText),
    labels: kvToObject(f.labelsText),
    binds: splitList(f.bindsText),
    ports: parsePorts(f.portsText),
    hostname: f.hostname?.trim() || undefined,
    user: f.user?.trim() || undefined,
    working_dir: f.working_dir?.trim() || undefined,
    cap_add: splitList(f.capAddText),
    cap_drop: splitList(f.capDropText),
    memory: f.memoryMb ? Math.round(Number(f.memoryMb) * 1024 * 1024) : undefined,
    nano_cpus: f.cpuCores ? Math.round(Number(f.cpuCores) * 1e9) : undefined,
    cpuset_cpus: f.cpuset_cpus?.trim() || undefined,
    env_file: splitList(f.envFileText),
    extra_hosts: splitList(f.extraHostsText),
    auto_remove: f.auto_remove || undefined,
    tty: f.tty || undefined,
    open_stdin: f.open_stdin || undefined,
  }
  for (const k of Object.keys(req)) {
    if (req[k] === undefined) delete req[k]
  }
  return YAML.stringify(req)
}

/** YAML → 表单。 */
function yamlToForm(yamlText: string): Record<string, any> {
  const r = (YAML.parse(yamlText) ?? {}) as CreateContainerRequest & Record<string, any>
  return {
    name: r.name ?? '',
    image: r.image ?? '',
    commandText: (r.command ?? []).join('\n'),
    entrypointText: (r.entrypoint ?? []).join('\n'),
    envText: (r.env ?? []).join('\n'),
    labelsText: Object.entries(r.labels ?? {})
      .map(([k, v]) => `${k}=${v}`)
      .join('\n'),
    bindsText: (r.binds ?? []).join('\n'),
    portsText: (r.ports ?? []).map((p) => `${p.host_port ?? ''}:${p.container_port ?? ''}`).join('\n'),
    network: r.network ?? 'bridge',
    restart_policy: r.restart_policy ?? '',
    auto_remove: !!r.auto_remove,
    tty: !!r.tty,
    open_stdin: !!r.open_stdin,
    hostname: r.hostname ?? '',
    user: r.user ?? '',
    working_dir: r.working_dir ?? '',
    capAddText: (r.cap_add ?? []).join(' '),
    capDropText: (r.cap_drop ?? []).join(' '),
    memoryMb: r.memory ? String(Math.round(r.memory / 1024 / 1024)) : '',
    cpuCores: r.nano_cpus ? String(r.nano_cpus / 1e9) : '',
    cpuset_cpus: r.cpuset_cpus ?? '',
    envFileText: (r.env_file ?? []).join(' '),
    extraHostsText: (r.extra_hosts ?? []).join(' '),
  }
}

/** 表单 → 请求结构。 */
function buildRequest(f: Record<string, any>): CreateContainerRequest {
  return {
    name: f.name || undefined,
    image: f.image,
    network: f.network || undefined,
    restart_policy: f.restart_policy || undefined,
    command: splitList(f.commandText),
    entrypoint: splitList(f.entrypointText),
    env: splitList(f.envText),
    labels: kvToObject(f.labelsText),
    binds: splitList(f.bindsText),
    ports: parsePorts(f.portsText),
    hostname: f.hostname || undefined,
    user: f.user || undefined,
    working_dir: f.working_dir || undefined,
    cap_add: splitList(f.capAddText),
    cap_drop: splitList(f.capDropText),
    memory: f.memoryMb ? Math.round(Number(f.memoryMb) * 1024 * 1024) : undefined,
    nano_cpus: f.cpuCores ? Math.round(Number(f.cpuCores) * 1e9) : undefined,
    cpuset_cpus: f.cpuset_cpus || undefined,
    env_file: splitList(f.envFileText),
    extra_hosts: splitList(f.extraHostsText),
    auto_remove: f.auto_remove || undefined,
    tty: f.tty || undefined,
    open_stdin: f.open_stdin || undefined,
  }
}

/** 提交回调（SpecEditor 统一传 YAML 文本）。 */
async function submit(yamlText: string): Promise<void> {
  const f = yamlToForm(yamlText)
  if (!f.image) {
    toast.error('请填写镜像')
    return
  }
  await createContainer(buildRequest(f))
  toast.success(`已创建容器「${f.name || f.image}」`)
  emit('update:open', false)
  emit('created')
}

watch(
  () => props.open,
  (open) => {
    if (open) {
      form.value = {
        name: '',
        image: '',
        commandText: '',
        entrypointText: '',
        envText: '',
        labelsText: '',
        bindsText: '',
        portsText: '',
        network: 'bridge',
        restart_policy: '',
        auto_remove: false,
        tty: false,
        open_stdin: false,
        hostname: '',
        user: '',
        working_dir: '',
        capAddText: '',
        capDropText: '',
        memoryMb: '',
        cpuCores: '',
        cpuset_cpus: '',
        envFileText: '',
        extraHostsText: '',
      }
    }
  },
)
</script>

<template>
  <SpecEditorModal
    :open="open"
    @update:open="(v) => emit('update:open', v)"
    v-model:form="form"
    title="创建容器"
    submit-label="创建"
    :fields="fields"
    :form-to-yaml="formToYaml"
    :yaml-to-form="yamlToForm"
    :on-submit="submit"
    yaml-hint="支持手写 YAML，字段与 docker create 参数对应（image/name/command/env/ports/binds/...）"
    yaml-placeholder="# 在此粘贴或编辑容器定义 YAML&#10;image: nginx:latest&#10;name: my-container&#10;ports:&#10;  - container_port: 80&#10;    host_port: 8080"
  />
</template>
