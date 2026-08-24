<template>
  <router-view />
  <ToastViewer />
</template>

<script setup lang="ts">
import ToastViewer from '@/components/ui/ToastViewer.vue'
// 根组件：仅承载路由视图 + 全局消息弹出层。
</script>
