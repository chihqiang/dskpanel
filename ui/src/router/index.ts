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
          meta: { menu: true, title: 'Docker', icon: 'container', requiresDocker: true },
          children: [
            { path: '', redirect: '/docker/overview' },
            { path: 'overview', name: 'docker-overview', component: () => import('@/views/docker/OverviewView.vue'), meta: { menu: true, title: '概览' } },
            { path: 'containers', name: 'docker-containers', component: () => import('@/views/docker/ContainersView.vue'), meta: { menu: true, title: '容器' } },
            { path: 'images', name: 'docker-images', component: () => import('@/views/docker/ImagesView.vue'), meta: { menu: true, title: '镜像' } },
            { path: 'networks', name: 'docker-networks', component: () => import('@/views/docker/NetworksView.vue'), meta: { menu: true, title: '网络' } },
            { path: 'volumes', name: 'docker-volumes', component: () => import('@/views/docker/VolumesView.vue'), meta: { menu: true, title: '卷' } },
            { path: 'compose', name: 'docker-compose', component: () => import('@/views/docker/ComposeView.vue'), meta: { menu: true, title: '编排' } },
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
            { path: 'networks', name: 'swarm-networks', component: () => import('@/views/swarm/NetworksView.vue'), meta: { menu: true, title: '网络', requiresSwarm: true } },
            { path: 'secrets', name: 'swarm-secrets', component: () => import('@/views/swarm/SecretsView.vue'), meta: { menu: true, title: 'Secret', requiresSwarm: true } },
          ],
        },
        {
          path: 'kubernetes',
          name: 'kubernetes',
          component: () => import('@/views/PlaceholderView.vue'),
          meta: { menu: true, title: 'Kubernetes', icon: 'boxes', badge: '即将上线' },
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
