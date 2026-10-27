import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import DashboardView from '@/views/DashboardView.vue'
import CoursesView from '@/views/CoursesView.vue'
import SettingsView from '@/views/SettingsView.vue'
import LearningView from '@/views/LearningView.vue'
import CourseArchiveView from '@/views/CourseArchiveView.vue'
import KnowledgeGraphView from '@/views/KnowledgeGraphView.vue'
import MisconceptionNetworkView from '@/views/MisconceptionNetworkView.vue'
import LibraryView from '@/views/LibraryView.vue'
import LibraryCollectionView from '@/views/LibraryCollectionView.vue'
import ExplorationView from '@/views/ExplorationView.vue'
import DomainInitializationView from '@/views/DomainInitializationView.vue'
import LoginView from '@/views/LoginView.vue'
import UsersView from '@/views/UsersView.vue'
import ProjectsView from '@/views/ProjectsView.vue'
import WorkspaceHomeView from '@/views/WorkspaceHomeView.vue'
import MonthlyReviewView from '@/views/MonthlyReviewView.vue'
import LifeView from '@/views/LifeView.vue'
import FinanceView from '@/views/FinanceView.vue'
import InboxView from '@/views/InboxView.vue'
import { currentUser } from '@/api/auth'
import { APIRequestError } from '@/api/http'

// Development-only experiments: not registered in production builds.
const labRoutes: RouteRecordRaw[] = import.meta.env.DEV
  ? [
      { path: '/lab/webgl', component: () => import('@/views/VisualLabView.vue'), meta: { title: 'WebGL Visual Lab', requiresAdmin: true } },
      { path: '/lab/timeline', component: () => import('@/views/TimelineVisualLabView.vue'), meta: { title: 'Timeline Visual Lab', requiresAdmin: true } },
    ]
  : []

// Admin tools ship with normal builds. The view stays a lazy chunk so ordinary
// business pages never download the effects gallery or its CSS.
const adminRoutes: RouteRecordRaw[] = [
  { path: '/admin/visual-effects', component: () => import('@/views/EffectsLabView.vue'), meta: { title: 'Visual Effects Lab', requiresAdmin: true } },
]

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', component: LoginView, meta: { title: '登录' } },
    {
      path: '/',
      component: WorkspaceHomeView,
      meta: { title: '工作区' },
    },
    { path: '/learn', component: DashboardView, meta: { title: '学习' } },
    { path: '/life', component: LifeView, meta: { title: '生活' } },
    { path: '/life/reviews', component: MonthlyReviewView, meta: { title: '月度回顾' } },
    { path: '/life/finance', component: FinanceView, meta: { title: '财务档案' } },
    { path: '/inbox', component: InboxView, meta: { title: '收集箱' } },
    {
      path: '/courses',
      component: CoursesView,
      meta: { title: '课程档案' },
    },
    { path: '/domains/new', component: DomainInitializationView, meta: { title: '创建学习领域' } },
    {
      path: '/courses/:id/learn',
      component: LearningView,
      meta: { title: '正在学习' },
    },
    {
      path: '/courses/:id/archive',
      component: CourseArchiveView,
      meta: { title: '课程档案' },
    },
    {
      path: '/courses/:id/map',
      component: KnowledgeGraphView,
      meta: { title: '知识结构' },
    },
    {
      path: '/courses/:id/misconceptions',
      component: MisconceptionNetworkView,
      meta: { title: '误区网络' },
    },
    {
      path: '/settings',
      component: SettingsView,
      meta: { title: '系统设置', requiresAdmin: true },
    },
    { path: '/users', component: UsersView, meta: { title: '用户管理', requiresAdmin: true } },
    { path: '/forbidden', component: () => import('@/views/ForbiddenView.vue'), meta: { title: '没有访问权限' } },
    { path: '/projects', component: ProjectsView, meta: { title: '项目' } },
    {
      path: '/exploration/questions',
      redirect: '/explore',
    },
    { path: '/explore/library', component: LibraryView, meta: { title: '我的资料库' } },
 { path: '/explore/library/:collection', component: LibraryCollectionView, meta: { title: '资料专题' } },
 { path: '/explore/library/:collection/pages/:page', component: LibraryCollectionView, meta: { title: '资料页面' } },
 { path: '/explore', component: ExplorationView, meta: { title: '探索' } },
    ...labRoutes,
    ...adminRoutes,
  ],
})

/**
 * Admin-only routes fail closed: any unknown error (network, 500, malformed
 * response) is treated as "no access". 401 keeps the existing redirect to /login
 * performed by the HTTP client, so the guard only aborts the navigation.
 */
router.beforeEach(async (to) => {
  if (!to.meta.requiresAdmin) return true
  try {
    const user = await currentUser()
    return user.role === 'admin' ? true : { path: '/forbidden' }
  } catch (error) {
    if (error instanceof APIRequestError && error.status === 401) return false
    return { path: '/forbidden' }
  }
})

export default router
