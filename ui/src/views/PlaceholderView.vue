<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Boxes, ArrowRight, BookOpen, ExternalLink, Container } from '@lucide/vue'
import Button from '@/components/ui/Button.vue'

const route = useRoute()
const router = useRouter()
const title = computed(() => (route.meta.title as string) || '')

/** 已上线栏目的快速入口。 */
const shortcuts = [
  { label: 'Docker 容器', desc: '管理单机容器、镜像、网络、卷', to: '/docker/containers' },
  { label: 'Swarm 服务', desc: '管理集群节点、服务、Secret', to: '/swarm/services' },
]
</script>

<template>
  <div class="flex min-h-full items-center justify-center py-8">
    <div class="w-full max-w-2xl text-center">
      <!-- 图标 + 标题 -->
      <div class="mb-6 flex flex-col items-center gap-4">
        <div class="flex h-20 w-20 items-center justify-center rounded-2xl bg-gradient-to-br from-blue-500 to-indigo-600 shadow-lg shadow-blue-500/30">
          <Boxes class="h-10 w-10 text-white" />
        </div>
        <div>
          <h1 class="text-2xl font-bold text-slate-800 dark:text-slate-100">{{ title }} 支持即将上线</h1>
          <p class="mt-2 text-sm text-slate-500 dark:text-slate-400">
            dskpanel 正在积极推进 Kubernetes 集成，届时将支持 Pod、Deployment、Service、ConfigMap 等核心资源的管理。
          </p>
        </div>
      </div>

      <!-- 规划中的能力 -->
      <div class="mb-8 rounded-xl border border-slate-200 bg-white p-6 text-left dark:border-slate-700 dark:bg-slate-800">
        <h2 class="mb-4 flex items-center gap-2 text-sm font-semibold text-slate-700 dark:text-slate-200">
          <BookOpen class="h-4 w-4 text-blue-600 dark:text-blue-400" />
          规划中的能力
        </h2>
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div class="flex items-start gap-2.5 rounded-lg bg-slate-50 p-3 dark:bg-slate-700/40">
            <span class="mt-0.5 text-blue-600 dark:text-blue-400">▸</span>
            <div>
              <p class="text-sm font-medium text-slate-700 dark:text-slate-200">多集群管理</p>
              <p class="text-xs text-slate-500 dark:text-slate-400">通过 kubeconfig 连接多个 K8s 集群</p>
            </div>
          </div>
          <div class="flex items-start gap-2.5 rounded-lg bg-slate-50 p-3 dark:bg-slate-700/40">
            <span class="mt-0.5 text-blue-600 dark:text-blue-400">▸</span>
            <div>
              <p class="text-sm font-medium text-slate-700 dark:text-slate-200">Pod 终端与日志</p>
              <p class="text-xs text-slate-500 dark:text-slate-400">实时 exec / logs / port-forward</p>
            </div>
          </div>
          <div class="flex items-start gap-2.5 rounded-lg bg-slate-50 p-3 dark:bg-slate-700/40">
            <span class="mt-0.5 text-blue-600 dark:text-blue-400">▸</span>
            <div>
              <p class="text-sm font-medium text-slate-700 dark:text-slate-200">YAML 编辑部署</p>
              <p class="text-xs text-slate-500 dark:text-slate-400">类 kubectl apply 的声明式管理</p>
            </div>
          </div>
          <div class="flex items-start gap-2.5 rounded-lg bg-slate-50 p-3 dark:bg-slate-700/40">
            <span class="mt-0.5 text-blue-600 dark:text-blue-400">▸</span>
            <div>
              <p class="text-sm font-medium text-slate-700 dark:text-slate-200">资源监控</p>
              <p class="text-xs text-slate-500 dark:text-slate-400">CPU / 内存 / 存储用量趋势</p>
            </div>
          </div>
        </div>
      </div>

      <!-- 快速入口 -->
      <div class="mb-8">
        <p class="mb-3 text-sm font-medium text-slate-600 dark:text-slate-300">在此期间，你可以使用以下已上线功能：</p>
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-3">
          <button
            v-for="s in shortcuts"
            :key="s.to"
            class="group flex flex-col items-start gap-1 rounded-xl border border-slate-200 bg-white p-4 text-left transition-all hover:border-blue-300 hover:shadow-md dark:border-slate-700 dark:bg-slate-800 dark:hover:border-blue-600"
            @click="router.push(s.to)"
          >
            <span class="flex items-center gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
              {{ s.label }}
              <ArrowRight class="h-3.5 w-3.5 transition-transform group-hover:translate-x-0.5" />
            </span>
            <span class="text-xs text-slate-500 dark:text-slate-400">{{ s.desc }}</span>
          </button>
        </div>
      </div>

      <!-- 文档与反馈 -->
      <div class="flex flex-wrap items-center justify-center gap-3">
        <Button variant="secondary" size="sm" @click="router.push('/docker')">
          <Container class="mr-1.5 h-3.5 w-3.5" />
          前往 Docker 管理
        </Button>
        <a
          href="https://kubernetes.io/docs/home/"
          target="_blank"
          rel="noopener noreferrer"
          class="inline-flex h-8 items-center gap-1.5 rounded-md border border-slate-200 px-3 text-sm text-slate-600 transition-colors hover:bg-slate-100 dark:border-slate-600 dark:text-slate-300 dark:hover:bg-slate-700"
        >
          <BookOpen class="h-3.5 w-3.5" />
          K8s 官方文档
        </a>
        <a
          href="https://github.com/chihqiang/dskpanel"
          target="_blank"
          rel="noopener noreferrer"
          class="inline-flex h-8 items-center gap-1.5 rounded-md border border-slate-200 px-3 text-sm text-slate-600 transition-colors hover:bg-slate-100 dark:border-slate-600 dark:text-slate-300 dark:hover:bg-slate-700"
        >
          <ExternalLink class="h-3.5 w-3.5" />
          提交需求 / 反馈
        </a>
      </div>
    </div>
  </div>
</template>
