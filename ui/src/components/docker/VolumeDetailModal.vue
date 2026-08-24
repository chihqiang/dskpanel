<script setup lang="ts">
import { ref, watch } from 'vue'
import Modal from '@/components/ui/Modal.vue'
import { useToast } from '@/composables/useToast'
import { inspectVolume, type VolumeDetail } from '@/api/volume'
import { fmtSize, kvEntries } from '@/utils/format'
import Skeleton from '@/components/ui/Skeleton.vue'

const props = defineProps<{ open: boolean; volumeName: string }>()

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()

const loading = ref(false)
const errorMsg = ref('')
const detail = ref<VolumeDetail | null>(null)
const toast = useToast()

watch(
  () => props.open,
  (open) => {
    if (open && props.volumeName) {
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
    detail.value = await inspectVolume(props.volumeName)
  } catch (err) {
    errorMsg.value = (err as Error).message
    toast.error((err as Error).message)
  } finally {
    loading.value = false
  }
}

</script>

<template>
  <Modal :open="open" @update:open="(v) => emit('update:open', v)" title="卷详情" width="max-w-2xl">
    <div v-if="loading" class="space-y-4 py-6">
      <div class="grid grid-cols-2 gap-4">
        <Skeleton height="h-8" />
        <Skeleton height="h-8" />
      </div>
      <Skeleton :count="3" />
    </div>
    <div v-else-if="errorMsg" class="py-10 text-center">
      <p class="text-sm text-slate-400">加载失败，请关闭后重试</p>
    </div>
    <div v-else-if="detail" class="space-y-5">
      <!-- 基本信息 -->
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div>
          <label class="mb-1 block text-xs text-slate-500">名称</label>
          <div class="truncate font-mono text-sm font-medium text-slate-800 dark:text-slate-100">{{ detail.name }}</div>
        </div>
        <div>
          <label class="mb-1 block text-xs text-slate-500">驱动</label>
          <div class="text-sm text-slate-800 dark:text-slate-100">{{ detail.driver }}</div>
        </div>
        <div class="sm:col-span-2">
          <label class="mb-1 block text-xs text-slate-500">挂载点</label>
          <div class="truncate rounded-md bg-slate-100 px-3 py-2 font-mono text-xs text-slate-700 dark:bg-slate-700 dark:text-slate-200">
            {{ detail.mountpoint }}
          </div>
        </div>
        <div>
          <label class="mb-1 block text-xs text-slate-500">作用域</label>
          <div class="text-sm text-slate-800 dark:text-slate-100">{{ detail.scope }}</div>
        </div>
        <div>
          <label class="mb-1 block text-xs text-slate-500">创建时间</label>
          <div class="text-sm text-slate-800 dark:text-slate-100">{{ detail.created_at || '-' }}</div>
        </div>
        <div v-if="detail.size > 0">
          <label class="mb-1 block text-xs text-slate-500">数据大小</label>
          <div class="text-sm text-slate-800 dark:text-slate-100">{{ fmtSize(detail.size) }}</div>
        </div>
        <div v-if="detail.ref_count > 0">
          <label class="mb-1 block text-xs text-slate-500">引用数</label>
          <div class="text-sm text-slate-800 dark:text-slate-100">{{ detail.ref_count }}</div>
        </div>
      </div>

      <!-- 标签 -->
      <div v-if="kvEntries(detail.labels).length">
        <label class="mb-1 block text-xs text-slate-500">标签 ({{ kvEntries(detail.labels).length }})</label>
        <div class="max-h-32 overflow-y-auto rounded-md bg-slate-100 px-3 py-2 font-mono text-xs text-slate-700 dark:bg-slate-700 dark:text-slate-200">
          <div v-for="[k, v] in kvEntries(detail.labels)" :key="k" class="break-all">
            <span class="text-slate-400">{{ k }}</span>: {{ v }}
          </div>
        </div>
      </div>

      <!-- 选项 -->
      <div v-if="kvEntries(detail.options).length">
        <label class="mb-1 block text-xs text-slate-500">选项</label>
        <div class="max-h-32 overflow-y-auto rounded-md bg-slate-100 px-3 py-2 font-mono text-xs text-slate-700 dark:bg-slate-700 dark:text-slate-200">
          <div v-for="[k, v] in kvEntries(detail.options)" :key="k" class="break-all">
            <span class="text-slate-400">{{ k }}</span>: {{ v }}
          </div>
        </div>
      </div>

      <!-- 使用方容器 -->
      <div v-if="detail.containers?.length">
        <label class="mb-1 block text-xs text-slate-500">使用该卷的容器 ({{ detail.containers.length }})</label>
        <div class="overflow-x-auto rounded-md border border-slate-200 bg-slate-50 dark:border-slate-700 dark:bg-slate-900">
          <table class="w-full font-mono text-xs text-slate-700 dark:text-slate-300">
            <thead>
              <tr class="border-b border-slate-200 text-left text-slate-400 dark:border-slate-700">
                <th class="px-3 py-1.5 font-normal">容器</th>
                <th class="px-3 py-1.5 font-normal">状态</th>
                <th class="px-3 py-1.5 font-normal">挂载路径</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="c in detail.containers"
                :key="c.id"
                class="border-b border-slate-200 last:border-b-0 dark:border-slate-700"
              >
                <td class="px-3 py-1.5">{{ c.name }}</td>
                <td class="px-3 py-1.5">{{ c.state }}</td>
                <td class="px-3 py-1.5">{{ c.dest || '-' }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
      <div v-else class="rounded-md border border-dashed border-slate-300 px-3 py-3 text-center text-xs text-slate-400 dark:border-slate-700">
        暂无容器使用该卷
      </div>
    </div>
  </Modal>
</template>
