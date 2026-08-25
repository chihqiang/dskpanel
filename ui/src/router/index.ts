import { createRouter, createWebHistory } from 'vue-router'
import NProgress from 'nprogress'
import 'nprogress/nprogress.css'
import { getToken } from '@/api/http'

// 顶部路由切换进度条（全局只配置一次）。
NProgress.configure({ showSpinner: false, trickleSpeed: 150, minimum: 0.08 })

// 扩展路由 meta 类型，侧边栏菜单由路由定义驱动。
declare module 'vue-router' {
  interface RouteMeta {
    /** 是否显示在侧边栏菜单。 */
    menu?: boolean
    /** 一级菜单图标标识（对应 MainLayout 的 iconMap）。 */
    icon?: string
    /** 页面标题 / 菜单标签。 */
    title?: string
    /** 公开路由（免登录）。 */
    public?: boolean
    /** 是否需要检测本机 Docker 才可进入。 */
    requiresDocker?: boolean
    /** 侧边栏菜单标签后的角标文案（如"即将上线"）。 */
    badge?: string
    /** 需要 Swarm 集群可用才显示（子菜单，如节点/服务/网络/Secret）。 */
    requiresSwarm?: boolean
    /** 需要 K8s 集群可用才显示（子菜单，如节点/Pod/工作负载）。 */
    requiresK8s?: boolean
  }
}

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
      meta: { public: true, title: '登录' },
    },
    {
      path: '/',
      component: () => import('@/layouts/MainLayout.vue'),
      children: [
        {
          path: '',
          redirect: '/docker',
        },
        {
          path: 'docker',
          component: () => import('@/views/docker/DockerView.vue'),
          meta: { menu: true, title: 'Docker', icon: 'container' },
          children: [
            { path: '', redirect: '/docker/overview' },
            { path: 'overview', name: 'docker-overview', component: () => import('@/views/docker/OverviewView.vue'), meta: { menu: true, title: '概览' } },
            { path: 'containers', name: 'docker-containers', component: () => import('@/views/docker/ContainersView.vue'), meta: { menu: true, title: '容器', requiresDocker: true } },
            { path: 'images', name: 'docker-images', component: () => import('@/views/docker/ImagesView.vue'), meta: { menu: true, title: '镜像', requiresDocker: true } },
            { path: 'networks', name: 'docker-networks', component: () => import('@/views/docker/NetworksView.vue'), meta: { menu: true, title: '网络', requiresDocker: true } },
            { path: 'volumes', name: 'docker-volumes', component: () => import('@/views/docker/VolumesView.vue'), meta: { menu: true, title: '卷', requiresDocker: true } },
            { path: 'compose', name: 'docker-compose', component: () => import('@/views/docker/ComposeView.vue'), meta: { menu: true, title: '编排', requiresDocker: true } },
          ],
        },
        {
          path: 'swarm',
          component: () => import('@/views/swarm/SwarmLayout.vue'),
          meta: { menu: true, title: 'Swarm', icon: 'workflow' },
          children: [
            { path: '', redirect: '/swarm/overview' },
            { path: 'overview', name: 'swarm-overview', component: () => import('@/views/swarm/OverviewView.vue'), meta: { menu: true, title: '概览' } },
            { path: 'nodes', name: 'swarm-nodes', component: () => import('@/views/swarm/NodesView.vue'), meta: { menu: true, title: '节点', requiresSwarm: true } },
            { path: 'services', name: 'swarm-services', component: () => import('@/views/swarm/ServicesView.vue'), meta: { menu: true, title: '服务', requiresSwarm: true } },
            { path: 'tasks', name: 'swarm-tasks', component: () => import('@/views/swarm/TasksView.vue'), meta: { menu: true, title: '任务', requiresSwarm: true } },
            { path: 'networks', name: 'swarm-networks', component: () => import('@/views/swarm/NetworksView.vue'), meta: { menu: true, title: '网络', requiresSwarm: true } },
            { path: 'secrets', name: 'swarm-secrets', component: () => import('@/views/swarm/SecretsView.vue'), meta: { menu: true, title: 'Secret', requiresSwarm: true } },
          ],
        },
        {
          path: 'k8s',
          component: () => import('@/views/k8s/K8sLayout.vue'),
          meta: { menu: true, title: 'Kubernetes', icon: 'orbit' },
          children: [
            { path: '', redirect: '/k8s/overview' },
            { path: 'overview', name: 'k8s-overview', component: () => import('@/views/k8s/OverviewView.vue'), meta: { menu: true, title: '概览' } },
            { path: 'nodes', name: 'k8s-nodes', component: () => import('@/views/k8s/NodesView.vue'), meta: { menu: true, title: '节点', requiresK8s: true } },
            { path: 'namespaces', name: 'k8s-namespaces', component: () => import('@/views/k8s/NamespacesView.vue'), meta: { menu: true, title: '命名空间', requiresK8s: true } },
            { path: 'pods', name: 'k8s-pods', component: () => import('@/views/k8s/PodsView.vue'), meta: { menu: true, title: 'Pod', requiresK8s: true } },
            { path: 'workloads', name: 'k8s-workloads', component: () => import('@/views/k8s/WorkloadsView.vue'), meta: { menu: true, title: '工作负载', requiresK8s: true } },
            { path: 'services', name: 'k8s-services', component: () => import('@/views/k8s/ServicesView.vue'), meta: { menu: true, title: '服务', requiresK8s: true } },
            { path: 'config', name: 'k8s-config', component: () => import('@/views/k8s/ConfigView.vue'), meta: { menu: true, title: '配置', requiresK8s: true } },
            { path: 'storage', name: 'k8s-storage', component: () => import('@/views/k8s/StorageView.vue'), meta: { menu: true, title: '存储', requiresK8s: true } },
            { path: 'rbac', name: 'k8s-rbac', component: () => import('@/views/k8s/RbacView.vue'), meta: { menu: true, title: 'RBAC', requiresK8s: true } },
            { path: 'hpa', name: 'k8s-hpa', component: () => import('@/views/k8s/HpaView.vue'), meta: { menu: true, title: 'HPA', requiresK8s: true } },
            { path: 'events', name: 'k8s-events', component: () => import('@/views/k8s/EventsView.vue'), meta: { menu: true, title: '事件', requiresK8s: true } },
          ],
        },
      ],
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: '/docker',
    },
  ],
})

// 全局前置守卫：未登录跳转登录页。
router.beforeEach((to) => {
  NProgress.start()
  const token = getToken()
  if (!to.meta.public && !token) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }
  if (to.meta.public && token && to.name === 'login') {
    return { path: '/' }
  }
  return true
})

// 设置页面标题。
router.afterEach((to) => {
  NProgress.done()
  const title = to.meta.title as string | undefined
  document.title = title ? `${title} - dskpanel` : 'dskpanel'
})

export default router
