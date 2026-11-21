<script setup lang="ts">
import { ref, watch } from 'vue'
import Button from '@/components/ui/Button.vue'
import Modal from '@/components/ui/Modal.vue'
import { useToast } from '@/composables/useToast'
import { inspectContainer, updateContainer } from '@/api/container'

const props = defineProps<{
  open: boolean
  containerId: string
  containerName: string
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  updated: []
}>()

const memory = ref('') // MB
const cpuShares = ref('')
const cpus = ref('') // 小数 CPU 核数（如 0.5、1.5）
const cpuset = ref('')
const restartPolicy = ref('')
const restartMax = ref('')
const submitting = ref(false)
const toast = useToast()

// 内存 MB → 字节。
function mbToBytes(mb: string): number {
  const v = Number(mb)
  return Number.isFinite(v) && v > 0 ? Math.round(v * 1024 * 1024) : 0
}

// 小数 CPU 核数 → NanoCPUs。
function coresToNano(cores: string): number {
  const v = Number(cores)
  return Number.isFinite(v) && v > 0 ? Math.round(v * 1e9) : 0
}

// number input 的 v-model 可能返回 number；统一转字符串并去空白。
function toTrim(v: string | number): string {
  return String(v ?? '').trim()
}

watch(
  () => props.open,
  async (open) => {
    if (!open) return
    memory.value = ''
    cpuShares.value = ''
    cpus.value = ''
    cpuset.value = ''
    restartPolicy.value = ''
    restartMax.value = ''
    // 拉取当前配置预填。
    try {
      const detail = await inspectContainer(props.containerId)
      const hc = detail.host_config
      if (!hc) return
      if (hc.memory) memory.value = String(Math.round(hc.memory / 1024 / 1024))
      if (hc.cpu_shares) cpuShares.value = String(hc.cpu_shares)
      if (hc.nano_cpus) cpus.value = String(hc.nano_cpus / 1e9)
      if (hc.cpuset_cpus) cpuset.value = hc.cpuset_cpus
      if (hc.restart_policy) restartPolicy.value = hc.restart_policy
      if (hc.restart_max) restartMax.value = String(hc.restart_max)
    } catch (err) {
      toast.error((err as Error).message)
    }
  },
)

async function submit(): Promise<void> {
  submitting.value = true
  try {
    await updateContainer(props.containerId, {
      memory: toTrim(memory.value) ? mbToBytes(toTrim(memory.value)) : undefined,
      cpu_shares: toTrim(cpuShares.value) ? Number(toTrim(cpuShares.value)) : undefined,
      nano_cpus: toTrim(cpus.value) ? coresToNano(toTrim(cpus.value)) : undefined,
      cpuset_cpus: toTrim(cpuset.value) || undefined,
      restart_policy: restartPolicy.value || undefined,
      restart_max: toTrim(restartMax.value) ? Number(toTrim(restartMax.value)) : undefined,
    })
    toast.success('已更新容器配置')
    emit('update:open', false)
    emit('updated')
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <Modal :open="open" @update:open="(v) => emit('update:open', v)" title="更新容器配置" width="max-w-lg">
    <div class="space-y-3">
      <p class="text-sm text-slate-500">
        更新容器「{{ containerName }}」的资源限制与重启策略（docker update）。留空表示不修改该项。
      </p>
      <div class="grid grid-cols-2 gap-3">
        <div>
          <label class="mb-1.5 block text-sm text-slate-500">内存限制 (MB)</label>
          <input v-model="memory" type="number" min="0" class="input" placeholder="如 512" />
        </div>
        <div>
          <label class="mb-1.5 block text-sm text-slate-500">CPU 核数</label>
          <input v-model="cpus" type="number" min="0" step="0.1" class="input" placeholder="如 0.5 / 1.5" />
        </div>
        <div>
          <label class="mb-1.5 block text-sm text-slate-500">CPU 份额</label>
          <input v-model="cpuShares" type="number" min="0" class="input" placeholder="如 512（相对权重）" />
        </div>
        <div>
          <label class="mb-1.5 block text-sm text-slate-500">CPU 亲和 (cpuset)</label>
          <input v-model="cpuset" class="input font-mono" placeholder="如 0-1 / 0,2" />
        </div>
        <div>
          <label class="mb-1.5 block text-sm text-slate-500">重启策略</label>
          <select v-model="restartPolicy" class="input">
            <option value="">不修改</option>
            <option value="no">no</option>
            <option value="always">always</option>
            <option value="on-failure">on-failure</option>
            <option value="unless-stopped">unless-stopped</option>
          </select>
        </div>
        <div>
          <label class="mb-1.5 block text-sm text-slate-500">最大重试次数 (on-failure)</label>
          <input v-model="restartMax" type="number" min="0" class="input" placeholder="如 5" />
        </div>
      </div>
    </div>
    <template #footer>
      <Button variant="secondary" @click="emit('update:open', false)">取消</Button>
      <Button :loading="submitting" @click="submit">保存</Button>
    </template>
  </Modal>
</template>
