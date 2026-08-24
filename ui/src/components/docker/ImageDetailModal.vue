<script setup lang="ts">
import { ref, watch } from 'vue'
import { Plus, X } from '@lucide/vue'
import Button from '@/components/ui/Button.vue'
import Modal from '@/components/ui/Modal.vue'
import { useToast } from '@/composables/useToast'
import { inspectImage, tagImage, removeImage, type ImageDetail } from '@/api/image'

const props = defineProps<{ open: boolean; imageId: string; imageTag?: string }>()

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()

const loading = ref(false)
const errorMsg = ref('')
const detail = ref<ImageDetail | null>(null)
const toast = useToast()

// 标签管理。
const tagInput = ref('')
const tagSubmitting = ref(false)

watch(
  () => props.open,
  (open) => {
    if (open && props.imageId) {
      load()
    }
  },
  { immediate: true },
)

async function load(): Promise<void> {
  loading.value = true
  errorMsg.value = ''
  detail.value = null
  try {
    detail.value = await inspectImage(props.imageId)
  } catch (err) {
    errorMsg.value = (err as Error).message
    toast.error((err as Error).message)
  } finally {
    loading.value = false
  }
}

async function addTag(): Promise<void> {
  const target = tagInput.value.trim()
  if (!target) return
  const source = props.imageTag || detail.value?.id || props.imageId
  tagSubmitting.value = true
  try {
    await tagImage(source, target)
    tagInput.value = ''
    toast.success(`已添加标签 ${target}`)
    await load()
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    tagSubmitting.value = false
  }
}

async function removeTag(tag: string): Promise<void> {
  // 删除单个 tag 引用（镜像有其他 tag 时仅移除该引用）。
  try {
    await removeImage(tag, false)
    toast.success(`已移除标签 ${tag}`)
    await load()
  } catch (err) {
    toast.error((err as Error).message)
  }
}

function fmtSize(size: number): string {
  if (size >= 1024 * 1024 * 1024) return (size / 1024 / 1024 / 1024).toFixed(1) + ' GB'
  if (size >= 1024 * 1024) return (size / 1024 / 1024).toFixed(1) + ' MB'
  if (size >= 1024) return (size / 1024).toFixed(1) + ' KB'
  return size + ' B'
}

function shortId(id: string): string {
  return id.replace('sha256:', '').slice(0, 12)
}

function fmtTime(ts: string): string {
  const d = new Date(ts)
  return isNaN(d.getTime()) ? (ts || '-') : d.toLocaleString()
}

function fmtTs(ts: number): string {
  if (!ts) return '-'
  return new Date(ts * 1000).toLocaleString()
}

function kvEntries(map?: Record<string, unknown>): [string, string][] {
  if (!map) return []
  return Object.entries(map).map(([k, v]) => [k, typeof v === 'object' ? JSON.stringify(v) : String(v)])
}

interface ParsedCmd {
  instruction: string
  args: string
}

/** 解析 docker history 的 CreatedBy 原始命令 → 可读的 Dockerfile 指令 + 参数。
 * 兼容格式：/bin/sh -c #(nop)  ENV x=y、/bin/sh -c set -x ...、COPY a b # buildkit、JSON 数组等。 */
function parseCmd(createdBy: string): ParsedCmd {
  let raw = (createdBy || '').replace(/\s*# buildkit$/, '')
  let instruction = ''
  let args = ''

  // /bin/sh -c #(nop)  指令 ...
  let m = raw.match(/^\/bin\/sh -c #\(nop\)\s+([A-Z][A-Z0-9]*)\s*(.*)$/)
  if (m) {
    instruction = m[1]
    args = m[2]
  } else if ((m = raw.match(/^\/bin\/sh -c\s+(.*)$/))) {
    // /bin/sh -c ...  → RUN
    instruction = 'RUN'
    args = m[1]
  } else if ((m = raw.match(/^(?:#\(nop\)\s+)?([A-Z][A-Z0-9]*)\s*(.*)$/))) {
    instruction = m[1]
    args = m[2]
  }

  // JSON 数组参数 [..] → 空格拼接，更可读。
  const trimmed = args.trim()
  if (trimmed.startsWith('[')) {
    try {
      const arr = JSON.parse(trimmed) as unknown[]
      if (Array.isArray(arr)) args = arr.map(String).join(' ')
    } catch {
      // 保留原样
    }
  }
  if (!instruction) instruction = 'RUN'
  return { instruction, args: args || raw }
}

/** 指令徽章配色。 */
const instructionColors: Record<string, string> = {
  RUN: 'bg-blue-100 text-blue-700 dark:bg-blue-900/50 dark:text-blue-300',
  ENV: 'bg-amber-100 text-amber-700 dark:bg-amber-900/50 dark:text-amber-300',
  CMD: 'bg-green-100 text-green-700 dark:bg-green-900/50 dark:text-green-300',
  ENTRYPOINT: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/50 dark:text-emerald-300',
  COPY: 'bg-violet-100 text-violet-700 dark:bg-violet-900/50 dark:text-violet-300',
  ADD: 'bg-purple-100 text-purple-700 dark:bg-purple-900/50 dark:text-purple-300',
  EXPOSE: 'bg-pink-100 text-pink-700 dark:bg-pink-900/50 dark:text-pink-300',
  WORKDIR: 'bg-cyan-100 text-cyan-700 dark:bg-cyan-900/50 dark:text-cyan-300',
  USER: 'bg-teal-100 text-teal-700 dark:bg-teal-900/50 dark:text-teal-300',
  LABEL: 'bg-slate-200 text-slate-700 dark:bg-slate-700 dark:text-slate-300',
  STOPSIGNAL: 'bg-slate-200 text-slate-700 dark:bg-slate-700 dark:text-slate-300',
  VOLUME: 'bg-slate-200 text-slate-700 dark:bg-slate-700 dark:text-slate-300',
  SHELL: 'bg-slate-200 text-slate-700 dark:bg-slate-700 dark:text-slate-300',
}

function instructionClass(ins: string): string {
  return instructionColors[ins] ?? 'bg-slate-200 text-slate-700 dark:bg-slate-700 dark:text-slate-300'
}

/** 是否为真实文件系统层（有磁盘占用）。 */
function isRealLayer(size: number): boolean {
  return size > 0
}
</script>

<template>
  <Modal :open="open" @update:open="(v) => emit('update:open', v)" title="镜像详情" width="max-w-3xl">
    <div v-if="loading" class="space-y-4 py-6">
      <Skeleton lines />
      <div class="grid grid-cols-2 gap-4">
        <Skeleton height="h-8" />
        <Skeleton height="h-8" />
      </div>
      <Skeleton :count="4" />
    </div>
    <div v-else-if="errorMsg" class="py-10 text-center">
      <p class="text-sm text-slate-400">加载失败，请关闭后重试</p>
    </div>
    <div v-else-if="detail" class="space-y-5">
      <!-- 基本信息 -->
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div class="sm:col-span-2">
          <label class="mb-1 block text-xs text-slate-500">仓库标签</label>
          <div class="flex flex-wrap gap-1.5">
            <span
              v-for="t in detail.repo_tags"
              :key="t"
              class="group inline-flex items-center gap-1 rounded-md bg-slate-100 px-2 py-0.5 font-mono text-xs text-slate-700 dark:bg-slate-700 dark:text-slate-200"
            >
              {{ t }}
              <button
                v-if="detail.repo_tags.length > 1"
                class="text-slate-400 hover:text-red-500"
                title="移除标签"
                @click="removeTag(t)"
              >
                <X class="h-3 w-3" />
              </button>
            </span>
            <span v-if="!detail.repo_tags.length" class="text-sm text-slate-500">(未打标签)</span>
          </div>
          <div class="mt-2 flex items-center gap-2">
            <input
              v-model="tagInput"
              class="input input-sm flex-1 font-mono"
              placeholder="新标签，如 myrepo/myapp:v1"
              @keyup.enter="addTag"
            />
            <Button variant="secondary" size="sm" :loading="tagSubmitting" @click="addTag">
              <Plus class="mr-1 h-3.5 w-3.5" />添加标签
            </Button>
          </div>
        </div>
        <div>
          <label class="mb-1 block text-xs text-slate-500">镜像 ID</label>
          <div class="truncate font-mono text-sm text-slate-800 dark:text-slate-100">{{ shortId(detail.id) }}</div>
        </div>
        <div>
          <label class="mb-1 block text-xs text-slate-500">架构 / 系统</label>
          <div class="text-sm text-slate-800 dark:text-slate-100">
            {{ detail.architecture }}{{ detail.variant ? `/${detail.variant}` : '' }} · {{ detail.os
            }}{{ detail.os_version ? ` (${detail.os_version})` : '' }}
          </div>
        </div>
        <div>
          <label class="mb-1 block text-xs text-slate-500">大小</label>
          <div class="text-sm text-slate-800 dark:text-slate-100">{{ fmtSize(detail.size) }}</div>
        </div>
        <div>
          <label class="mb-1 block text-xs text-slate-500">创建时间</label>
          <div class="text-sm text-slate-800 dark:text-slate-100">{{ fmtTime(detail.created) }}</div>
        </div>
        <div v-if="detail.author" class="sm:col-span-2">
          <label class="mb-1 block text-xs text-slate-500">作者</label>
          <div class="text-sm text-slate-800 dark:text-slate-100">{{ detail.author }}</div>
        </div>
        <div v-if="detail.repo_digests?.length" class="sm:col-span-2">
          <label class="mb-1 block text-xs text-slate-500">摘要 (Digest)</label>
          <div class="max-h-24 space-y-1 overflow-y-auto rounded-md bg-slate-100 px-3 py-2 font-mono text-xs text-slate-700 dark:bg-slate-700 dark:text-slate-200">
            <div v-for="d in detail.repo_digests" :key="d" class="break-all">{{ d }}</div>
          </div>
        </div>
      </div>

      <!-- 多平台架构 (Manifests) -->
      <div v-if="detail.manifests?.length" class="space-y-1">
        <label class="block text-xs text-slate-500">多平台架构 ({{ detail.manifests.length }})</label>
        <div class="overflow-x-auto rounded-md border border-slate-200 bg-slate-50 dark:border-slate-700 dark:bg-slate-900">
          <table class="w-full text-xs text-slate-700 dark:text-slate-300">
            <thead>
              <tr class="border-b border-slate-200 text-left text-slate-400 dark:border-slate-700">
                <th class="px-3 py-1.5 font-normal">平台</th>
                <th class="px-3 py-1.5 font-normal">本地方量</th>
                <th class="px-3 py-1.5 font-normal">大小</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="m in detail.manifests"
                :key="m.id"
                class="border-b border-slate-200 last:border-b-0 dark:border-slate-700"
              >
                <td class="px-3 py-1.5 font-mono">{{ m.platform }}</td>
                <td class="px-3 py-1.5">
                  <span
                    :class="m.available ? 'text-green-600 dark:text-green-400' : 'text-slate-400'"
                  >
                    {{ m.available ? '已获取' : '未获取' }}
                  </span>
                </td>
                <td class="px-3 py-1.5">{{ fmtSize(m.content_size) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- 配置 -->
      <div v-if="detail.config" class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div v-if="detail.config.cmd?.length">
          <label class="mb-1 block text-xs text-slate-500">命令 (CMD)</label>
          <div class="rounded-md bg-slate-100 px-3 py-2 font-mono text-xs text-slate-700 dark:bg-slate-700 dark:text-slate-200">
            {{ detail.config.cmd.join(' ') }}
          </div>
        </div>
        <div v-if="detail.config.entrypoint?.length">
          <label class="mb-1 block text-xs text-slate-500">入口点 (ENTRYPOINT)</label>
          <div class="rounded-md bg-slate-100 px-3 py-2 font-mono text-xs text-slate-700 dark:bg-slate-700 dark:text-slate-200">
            {{ detail.config.entrypoint.join(' ') }}
          </div>
        </div>
        <div v-if="detail.config.user">
          <label class="mb-1 block text-xs text-slate-500">用户 (USER)</label>
          <div class="rounded-md bg-slate-100 px-3 py-2 font-mono text-xs text-slate-700 dark:bg-slate-700 dark:text-slate-200">
            {{ detail.config.user }}
          </div>
        </div>
        <div v-if="detail.config.working_dir">
          <label class="mb-1 block text-xs text-slate-500">工作目录 (WORKDIR)</label>
          <div class="rounded-md bg-slate-100 px-3 py-2 font-mono text-xs text-slate-700 dark:bg-slate-700 dark:text-slate-200">
            {{ detail.config.working_dir }}
          </div>
        </div>
        <div v-if="detail.config.env?.length" class="sm:col-span-2">
          <label class="mb-1 block text-xs text-slate-500">环境变量 ({{ detail.config.env.length }})</label>
          <div class="max-h-36 overflow-y-auto rounded-md bg-slate-100 px-3 py-2 font-mono text-xs text-slate-700 dark:bg-slate-700 dark:text-slate-200">
            <div v-for="e in detail.config.env" :key="e" class="break-all">{{ e }}</div>
          </div>
        </div>
        <div v-if="kvEntries(detail.config.exposed_ports).length" class="sm:col-span-2">
          <label class="mb-1 block text-xs text-slate-500">暴露端口 (EXPOSE)</label>
          <div class="flex flex-wrap gap-1.5">
            <span
              v-for="[k] in kvEntries(detail.config.exposed_ports)"
              :key="k"
              class="rounded-md bg-slate-100 px-2 py-0.5 font-mono text-xs text-slate-700 dark:bg-slate-700 dark:text-slate-200"
            >
              {{ k }}
            </span>
          </div>
        </div>
        <div v-if="kvEntries(detail.config.volumes).length" class="sm:col-span-2">
          <label class="mb-1 block text-xs text-slate-500">卷 (VOLUME)</label>
          <div class="flex flex-wrap gap-1.5">
            <span
              v-for="[k] in kvEntries(detail.config.volumes)"
              :key="k"
              class="rounded-md bg-slate-100 px-2 py-0.5 font-mono text-xs text-slate-700 dark:bg-slate-700 dark:text-slate-200"
            >
              {{ k }}
            </span>
          </div>
        </div>
        <div v-if="kvEntries(detail.config.labels).length" class="sm:col-span-2">
          <label class="mb-1 block text-xs text-slate-500">标签 ({{ kvEntries(detail.config.labels).length }})</label>
          <div class="max-h-32 overflow-y-auto rounded-md bg-slate-100 px-3 py-2 font-mono text-xs text-slate-700 dark:bg-slate-700 dark:text-slate-200">
            <div v-for="[k, v] in kvEntries(detail.config.labels)" :key="k" class="break-all">
              <span class="text-slate-400">{{ k }}</span>: {{ v }}
            </div>
          </div>
        </div>
      </div>

      <!-- LAYERS 分层（docker history：每层命令 + 大小） -->
      <div>
        <label class="mb-1 flex items-center justify-between text-xs text-slate-500">
          <span>镜像分层 (LAYERS)</span>
          <span class="font-mono text-slate-400">
            {{ detail.rootfs_type || 'rootfs' }} ·
            {{ detail.history?.length || detail.layers.length }} 步
          </span>
        </label>

        <!-- 有 history：展示每步命令 -->
        <div
          v-if="detail.history?.length"
          class="max-h-72 overflow-y-auto rounded-md border border-slate-200 bg-slate-50 dark:border-slate-700 dark:bg-slate-900"
        >
          <div
            v-for="(item, idx) in detail.history"
            :key="item.id + idx"
            class="flex items-start gap-2 border-b border-slate-200 px-3 py-2 last:border-b-0 dark:border-slate-700"
            :class="isRealLayer(item.size) ? 'bg-slate-100/60 dark:bg-slate-800/40' : 'opacity-70'"
          >
            <span class="w-7 shrink-0 pt-0.5 text-right font-mono text-xs text-slate-400">
              #{{ (detail.history?.length ?? 0) - idx }}
            </span>
            <span
              class="mt-px inline-flex shrink-0 items-center rounded px-1.5 py-0.5 text-center text-[10px] font-semibold tracking-wide"
              :class="instructionClass(parseCmd(item.created_by).instruction)"
            >
              {{ parseCmd(item.created_by).instruction }}
            </span>
            <div class="min-w-0 flex-1">
              <div class="whitespace-pre-wrap break-all font-mono text-xs leading-relaxed text-slate-700 dark:text-slate-200">
                {{ parseCmd(item.created_by).args }}
              </div>
              <div class="mt-0.5 font-mono text-[10px] text-slate-400">
                {{ item.id.startsWith('sha256:') ? shortId(item.id) : 'meta' }}
                <span class="mx-1">·</span>{{ fmtTs(item.created) }}
              </div>
            </div>
            <span class="w-14 shrink-0 pt-0.5 text-right font-mono text-xs text-slate-500 dark:text-slate-400">
              {{ item.size > 0 ? fmtSize(item.size) : '—' }}
            </span>
          </div>
        </div>

        <!-- 无 history：回退为纯 sha256 列表 -->
        <div
          v-else
          class="max-h-48 overflow-y-auto rounded-md border border-slate-200 bg-slate-50 font-mono text-xs dark:border-slate-700 dark:bg-slate-900"
        >
          <div
            v-for="(layer, idx) in detail.layers"
            :key="layer"
            class="flex items-center gap-3 border-b border-slate-200 px-3 py-1.5 last:border-b-0 dark:border-slate-700"
          >
            <span class="w-16 shrink-0 text-slate-400">#{{ detail.layers.length - idx }}</span>
            <span class="truncate text-slate-700 dark:text-slate-300">{{ layer }}</span>
          </div>
          <div v-if="!detail.layers.length" class="px-3 py-2 text-slate-500">无分层信息</div>
        </div>
      </div>
    </div>
  </Modal>
</template>
