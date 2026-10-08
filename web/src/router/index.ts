import { createRouter, createWebHistory } from 'vue-router'
import DashboardView from '@/views/DashboardView.vue'
import CoursesView from '@/views/CoursesView.vue'
import SettingsView from '@/views/SettingsView.vue'
import LearningView from '@/views/LearningView.vue'
import CourseArchiveView from '@/views/CourseArchiveView.vue'
import KnowledgeGraphView from '@/views/KnowledgeGraphView.vue'
import MisconceptionNetworkView from '@/views/MisconceptionNetworkView.vue'
import ExplorationView from '@/views/ExplorationView.vue'
import DomainInitializationView from '@/views/DomainInitializationView.vue'
import LoginView from '@/views/LoginView.vue'
import UsersView from '@/views/UsersView.vue'
import ProjectsView from '@/views/ProjectsView.vue'
import WorkspaceHomeView from '@/views/WorkspaceHomeView.vue'
import LifeView from '@/views/LifeView.vue'
import InboxView from '@/views/InboxView.vue'

export default createRouter({
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
      meta: { title: '系统设置' },
    },
    { path: '/users', component: UsersView, meta: { title: '用户管理' } },
    { path: '/projects', component: ProjectsView, meta: { title: '项目' } },
    {
      path: '/exploration/questions',
      redirect: '/explore',
    },
    { path: '/explore', component: ExplorationView, meta: { title: '探索' } },
  ],
})
