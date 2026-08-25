<script setup lang="ts">
import { ref, watch } from 'vue'
import { Copy, Check } from '@lucide/vue'
import Button from '@/components/ui/Button.vue'
import Modal from '@/components/ui/Modal.vue'
import { useToast } from '@/composables/useToast'
import { useClipboard } from '@/utils/clipboard'
import {
  swarmJoinTokens,
  type JoinTokens,
} from '@/api/swarm'

const props = defineProps<{
  open: boolean
}>()

const emit = defineEmits<{ 'update:open': [value: boolean] }>()

const toast = useToast()
const { copy } = useClipboard()

const tokens = ref<JoinTokens | null>(null)
const tokenLoading = ref(false)
const copied = ref('')

async function fetchTokens(): Promise<void> {
  tokens.value = null
  copied.value = ''
  tokenLoading.value = true
  try {
    tokens.value = await swarmJoinTokens()
  } catch (err) {
    toast.error((err as Error).message)
  } finally {
    tokenLoading.value = false
  }
}

async function copyToken(text: string, key: string): Promise<void> {
  const ok = await copy(text, '已复制到剪贴板', '复制失败，请手动复制')
  if (ok) {
    copied.value = key
    setTimeout(() => {
      if (copied.value === key) copied.value = ''
    }, 1500)
  }
}

watch(
  () => props.open,
  (open) => {
    if (open) {
      void fetchTokens()
    }
  },
  { immediate: true },
)
</script>

<template>
  <Modal :open="open" @update:open="emit('update:open', $event)" title="集群加入令牌" width="max-w-2xl">
    <div v-if="tokenLoading" class="py-8 text-center text-sm text-slate-400">加载中…</div>
    <div v-else-if="tokens" class="space-y-4">
      <p class="text-sm text-slate-500">
        在其它主机上执行 docker swarm join 以加入集群：
      </p>
      <div>
        <label class="mb-1.5 block text-sm text-slate-500">Worker 加入命令</label>
        <div class="flex items-center gap-2">
          <code class="flex-1 select-all overflow-x-auto rounded-lg bg-slate-100 p-2.5 text-xs text-slate-700 dark:bg-slate-900 dark:text-slate-300">
            docker swarm join --token {{ tokens.worker }} {{ tokens.addr || '&lt;manager-addr&gt;:2377' }}
          </code>
          <Button variant="secondary" size="sm" @click="copyToken(`docker swarm join --token ${tokens.worker} ${tokens.addr || '<manager-addr>:2377'}`, 'w')">
            <Check v-if="copied === 'w'" class="h-3.5 w-3.5" />
            <Copy v-else class="h-3.5 w-3.5" />
          </Button>
        </div>
      </div>
      <div>
        <label class="mb-1.5 block text-sm text-slate-500">Manager 加入命令</label>
        <div class="flex items-center gap-2">
          <code class="flex-1 select-all overflow-x-auto rounded-lg bg-slate-100 p-2.5 text-xs text-slate-700 dark:bg-slate-900 dark:text-slate-300">
            docker swarm join --token {{ tokens.manager }} {{ tokens.addr || '&lt;manager-addr&gt;:2377' }}
          </code>
          <Button variant="secondary" size="sm" @click="copyToken(`docker swarm join --token ${tokens.manager} ${tokens.addr || '<manager-addr>:2377'}`, 'm')">
            <Check v-if="copied === 'm'" class="h-3.5 w-3.5" />
            <Copy v-else class="h-3.5 w-3.5" />
          </Button>
        </div>
      </div>
    </div>
    <template #footer>
      <Button variant="secondary" @click="emit('update:open', false)">关闭</Button>
    </template>
  </Modal>
</template>
