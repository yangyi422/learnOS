import { createRouter, createWebHistory } from 'vue-router'
import DashboardView from '@/views/DashboardView.vue'
import CoursesView from '@/views/CoursesView.vue'
import SettingsView from '@/views/SettingsView.vue'

export default createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      component: DashboardView,
      meta: { title: '学习首页' },
    },
    {
      path: '/courses',
      component: CoursesView,
      meta: { title: '课程档案' },
    },
    {
      path: '/settings',
      component: SettingsView,
      meta: { title: '系统设置' },
    },
  ],
})
