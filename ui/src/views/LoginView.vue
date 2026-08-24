<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Button from '@/components/ui/Button.vue'
import { useToast } from '@/composables/useToast'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const toast = useToast()
const auth = useAuthStore()

const form = reactive({ username: 'admin', password: '' })
const loading = ref(false)

async function onSubmit(): Promise<void> {
  if (!form.username || !form.password) {
    toast.error('请输入用户名和密码')
    return
  }
  loading.value = true
  try {
    await auth.login({ username: form.username, password: form.password })
    toast.success('登录成功')
    const redirect = (route.query.redirect as string) || '/'
    router.push(redirect)
  } catch (err) {
    toast.error((err as Error).message || '登录失败')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-slate-100 dark:bg-slate-900">
    <div class="w-full max-w-md">
      <div class="mb-8 flex flex-col items-center gap-3">
        <svg class="h-16 w-16 text-blue-600" viewBox="0 0 24 24" fill="currentColor">
          <path d="M13 2L3 14h7l-1 8 10-12h-7l1-8z" />
        </svg>
        <h1 class="text-3xl font-semibold text-slate-800 dark:text-slate-100">dskpanel</h1>
        <p class="text-base text-slate-500 dark:text-slate-400">容器管理面板</p>
      </div>

      <div class="rounded-xl border border-slate-200 bg-white p-8 shadow-sm dark:border-slate-700 dark:bg-slate-800">
        <form @submit.prevent="onSubmit" class="space-y-5">
          <div>
            <label class="mb-1.5 block text-sm font-medium text-slate-600 dark:text-slate-300" for="username">用户名</label>
            <input
              id="username"
              v-model="form.username"
              type="text"
              class="w-full rounded-lg border border-slate-300 px-4 py-2.5 text-sm outline-none transition-colors focus:border-blue-500 focus:ring-1 focus:ring-blue-500 dark:border-slate-600 dark:bg-slate-900 dark:text-slate-100"
              autocomplete="username"
            />
          </div>
          <div>
            <label class="mb-1.5 block text-sm font-medium text-slate-600 dark:text-slate-300" for="password">密码</label>
            <input
              id="password"
              v-model="form.password"
              type="password"
              class="w-full rounded-lg border border-slate-300 px-4 py-2.5 text-sm outline-none transition-colors focus:border-blue-500 focus:ring-1 focus:ring-blue-500 dark:border-slate-600 dark:bg-slate-900 dark:text-slate-100"
              autocomplete="current-password"
            />
          </div>

          <Button type="submit" :loading="loading" class="w-full" size="lg">登录</Button>
        </form>
      </div>
    </div>
  </div>
</template>
