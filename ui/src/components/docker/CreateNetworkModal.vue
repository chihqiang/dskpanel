<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import Button from '@/components/ui/Button.vue'
import Modal from '@/components/ui/Modal.vue'
import { useToast } from '@/composables/useToast'
import { createNetwork } from '@/api/network'

const props = defineProps<{ open: boolean }>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  created: []
}>()

const submitting = ref(false)
const toast = useToast()
const showAdvanced = ref(false)
const form = reactive({
  name: '',
  driver: 'bridge',
  subnet: '',
  gateway: '',
  ip_range: '',
  internal: false,
  enable_ipv6: false,
  labels: '',
})

watch(
  () => props.open,
  (open) => {
    if (open) {
      form.name = ''
      form.driver = 'bridge'
      form.subnet = ''
      form.gateway = ''
      form.ip_range = ''
      form.internal = false
      form.enable_ipv6 = false
      form.labels = ''
    }
  },
)

/** 标签字符串（逗号分隔 k=v）→ map。 */
function parseLabels(s: string): Record<string, string> | undefined {
  const out: Record<string, string> = {}
  for (const item of s.split(',') || []) {
    const kv = item.trim()
    if (!kv) continue
    const idx = kv.indexOf('=')
    if (idx > 0) out[kv.slice(0, idx).trim()] = kv.slice(idx + 1).trim()
    else out[kv] = ''
  }
  return Object.keys(out).length ? out : undefined
}

async function submit(): Promise<void> {
  if (!form.name) {
    toast.error('请输入网络名称')
    return
  }
  submitting.value = true
  try {
    await createNetwork({
      name: form.name,
      driver: form.driver,
      subnet: form.subnet.trim() || undefined,
      gateway: form.gateway.trim() || undefined,
      ip_range: form.ip_range.trim() || undefined,
      internal: form.internal,
      enable_ipv6: form.enable_ipv6,
      labels: parseLabels(form.labels),
    })
    toast.success(`已创建网络「${form.name}」`)
    emit('update:open', false)
    emit('created')
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <Modal :open="open" @update:open="(v) => emit('update:open', v)" title="创建网络" width="max-w-lg">
    <div class="space-y-3">
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <div>
          <label class="mb-1.5 block text-sm text-slate-500">名称</label>
          <input v-model="form.name" class="input" placeholder="my-network" />
        </div>
        <div>
          <label class="mb-1.5 block text-sm text-slate-500">驱动</label>
          <select v-model="form.driver" class="input">
            <option value="bridge">bridge</option>
            <option value="macvlan">macvlan</option>
            <option value="host">host</option>
            <option value="none">none</option>
          </select>
        </div>
      </div>

      <!-- 高级配置 -->
      <div class="rounded-lg border border-slate-200 dark:border-slate-700">
        <button
          class="flex w-full items-center justify-between px-3 py-2 text-sm font-medium text-slate-600 hover:text-slate-800 dark:text-slate-300 dark:hover:text-slate-100"
          @click="showAdvanced = !showAdvanced"
        >
          <span>高级配置</span>
          <span class="text-slate-400">{{ showAdvanced ? '收起 ▲' : '展开 ▼' }}</span>
        </button>
        <div v-if="showAdvanced" class="space-y-3 border-t border-slate-200 px-3 py-3 dark:border-slate-700">
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <div>
              <label class="mb-1.5 block text-sm text-slate-500">子网 (subnet)</label>
              <input v-model="form.subnet" class="input font-mono" placeholder="如 172.20.0.0/16" />
            </div>
            <div>
              <label class="mb-1.5 block text-sm text-slate-500">网关 (gateway)</label>
              <input v-model="form.gateway" class="input font-mono" placeholder="如 172.20.0.1" />
            </div>
            <div class="sm:col-span-2">
              <label class="mb-1.5 block text-sm text-slate-500">IP 范围 (ip_range)</label>
              <input v-model="form.ip_range" class="input font-mono" placeholder="如 172.20.0.0/24" />
            </div>
            <div class="sm:col-span-2">
              <label class="mb-1.5 block text-sm text-slate-500">标签（逗号分隔 k=v）</label>
              <input v-model="form.labels" class="input font-mono" placeholder="如 env=prod,team=infra" />
            </div>
            <label class="flex items-center gap-2 text-sm text-slate-600 dark:text-slate-300">
              <input v-model="form.internal" type="checkbox" class="h-4 w-4 rounded border-slate-300" />
              仅内部网络 (internal)
            </label>
            <label class="flex items-center gap-2 text-sm text-slate-600 dark:text-slate-300">
              <input v-model="form.enable_ipv6" type="checkbox" class="h-4 w-4 rounded border-slate-300" />
              启用 IPv6
            </label>
          </div>
        </div>
      </div>
    </div>
    <template #footer>
      <Button variant="secondary" @click="emit('update:open', false)">取消</Button>
      <Button :loading="submitting" @click="submit">创建</Button>
    </template>
  </Modal>
</template>
