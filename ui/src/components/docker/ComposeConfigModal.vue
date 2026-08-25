<script setup lang="ts">
/**
 * Compose 配置文件查看/编辑弹窗：
 * - 查看：只读展示 YAML
 * - 编辑：可修改 YAML 后重新部署（调用 deployComposeStream）
 */
import { ref, watch } from 'vue'
import { Copy, Check, Rocket, Pencil, Eye, Download } from '@lucide/vue'
import Button from '@/components/ui/Button.vue'
import Modal from '@/components/ui/Modal.vue'
import Badge from '@/components/ui/Badge.vue'
import { useClipboard } from '@/utils/clipboard'
import { useToast } from '@/composables/useToast'
import { composeProjectConfig, deployComposeStream } from '@/api/compose'

const props = defineProps<{
  open: boolean
  project: { name: string } | null
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  'deployed': []
}>()

const { copy } = useClipboard()
const toast = useToast()

const content = ref('')
const loading = ref(false)
const errorMsg = ref('')
const copied = ref(false)
const editing = ref(false)
const deploying = ref(false)
const deployLines = ref<string[]>([])
const deployDone = ref(false)
const deployOk = ref(false)
let stopDeploy: (() => void) | null = null

watch(
  () => [props.open, props.project] as const,
  ([open, project]) => {
    if (open && project) {
      content.value = ''
      errorMsg.value = ''
      editing.value = false
      deployLines.value = []
      deployDone.value = false
      loading.value = true
      composeProjectConfig(project.name)
        .then((text) => {
          content.value = text
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

async function copyYaml(): Promise<void> {
  await copy(content.value, '已复制到剪贴板', '复制失败，请手动复制')
  copied.value = true
  setTimeout(() => (copied.value = false), 1500)
}

/** 下载 YAML 文件。 */
function downloadYaml(): void {
  if (!content.value) return
  const fileName = `compose-${props.project?.name ?? 'project'}.yaml`
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
  deployLines.value = []
  deployDone.value = false
}

function onDeploy(): void {
  if (!content.value.trim()) {
    toast.error('Compose 内容为空')
    return
  }
  stopDeploy?.()
  deploying.value = true
  deployLines.value = []
  deployDone.value = false

  stopDeploy = deployComposeStream(
    content.value,
    (line) => {
      deployLines.value.push(line)
    },
    (success) => {
      deployOk.value = success
      deployDone.value = true
      deploying.value = false
      if (success) {
        toast.success('Compose 重新部署成功')
        emit('deployed')
      } else {
        toast.error('Compose 部署失败，请查看输出日志')
      }
    },
    (msg) => {
      toast.error(msg)
      deploying.value = false
      deployDone.value = true
      deployOk.value = false
    },
  )
}

function closeHandler(v: boolean): void {
  if (!v) {
    stopDeploy?.()
  }
  emit('update:open', v)
}
</script>

<template>
  <Modal :open="open" @update:open="closeHandler" :title="`Compose 配置 - ${project?.name ?? ''}`" width="max-w-3xl">
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
        <!-- 查看模式 -->
        <pre
          v-if="!editing"
          class="max-h-[50vh] overflow-auto rounded-lg bg-slate-900 px-4 py-3 font-mono text-xs leading-relaxed text-green-300"
        >{{ content || '（空）' }}</pre>
        <!-- 编辑模式 -->
        <div v-else class="overflow-hidden rounded-lg border border-slate-200 dark:border-slate-700">
          <div class="flex items-center gap-2 border-b border-slate-200 bg-slate-50 px-3 py-2 dark:border-slate-700 dark:bg-slate-800">
            <span class="h-2.5 w-2.5 rounded-full bg-red-400" />
            <span class="h-2.5 w-2.5 rounded-full bg-yellow-400" />
            <span class="h-2.5 w-2.5 rounded-full bg-green-400" />
            <span class="ml-2 font-mono text-xs text-slate-400">docker-compose.yaml</span>
            <span v-if="deploying" class="ml-auto flex items-center gap-1.5 text-xs text-slate-400">
              <span class="h-3 w-3 animate-spin rounded-full border-2 border-slate-300 border-t-blue-500" />
              部署中…
            </span>
          </div>
          <textarea
            v-model="content"
            spellcheck="false"
            class="block h-72 w-full resize-y bg-slate-900 px-3 py-2 font-mono text-xs leading-relaxed text-green-300 outline-none placeholder:text-slate-500"
            placeholder="# 编辑 Compose YAML"
          />
        </div>

        <!-- 部署实时回显 -->
        <div v-if="deployDone || deployLines.length > 0" class="mt-3 space-y-2">
          <div class="flex items-center gap-2">
            <span class="text-sm font-medium text-slate-700 dark:text-slate-200">部署输出</span>
            <Badge v-if="deploying" variant="blue">部署中...</Badge>
            <Badge v-else-if="deployDone" :variant="deployOk ? 'green' : 'red'">
              {{ deployOk ? '部署成功' : '部署失败' }}
            </Badge>
          </div>
          <div class="max-h-48 overflow-y-auto rounded-md bg-slate-900 p-3 font-mono text-xs text-slate-100">
            <div v-for="(line, idx) in deployLines" :key="idx" class="whitespace-pre-wrap break-all">{{ line }}</div>
            <div v-if="deploying && deployLines.length === 0" class="text-slate-500">部署中...</div>
          </div>
        </div>
      </template>
    </div>
    <template #footer>
      <div class="flex items-center gap-2">
        <Button variant="secondary" size="sm" :disabled="!content" @click="copyYaml">
          <Check v-if="copied" class="h-3.5 w-3.5 text-green-500" />
          <Copy v-else class="h-3.5 w-3.5" />
          复制
        </Button>
        <Button variant="secondary" size="sm" :disabled="!content" @click="downloadYaml">
          <Download class="h-3.5 w-3.5" />
          下载
        </Button>
      </div>
      <div class="ml-auto flex items-center gap-2">
        <Button v-if="!editing" variant="secondary" size="sm" @click="toggleEdit">
          <Pencil class="h-3.5 w-3.5" />
          编辑并重新部署
        </Button>
        <Button v-else variant="secondary" size="sm" @click="toggleEdit">
          <Eye class="h-3.5 w-3.5" />
          取消编辑
        </Button>
        <Button v-if="editing" size="sm" :loading="deploying" @click="onDeploy">
          <Rocket class="h-3.5 w-3.5" />
          部署
        </Button>
        <Button v-else variant="secondary" @click="emit('update:open', false)">关闭</Button>
      </div>
    </template>
  </Modal>
</template>
