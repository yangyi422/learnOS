<template>
  <el-container class="app-shell">
    <el-aside width="76px" class="sidebar">
      <RouterLink to="/" class="brand" aria-label="LearnOS 工作区" title="LearnOS">
        <span class="brand-mark">L</span>
      </RouterLink>

      <nav class="nav-menu" aria-label="主要导航">
        <RouterLink v-for="item in navigation" :key="item.to" :to="item.to" class="nav-link" :aria-label="item.label" :aria-current="activePath === item.to ? 'page' : undefined" :title="item.label">
          <WorkspaceIcon :name="item.icon" />
          <span class="sr-only">{{ item.label }}</span>
        </RouterLink>
      </nav>

      <div class="sidebar-footer">
        <button type="button" class="nav-link sidebar-logout" aria-label="退出登录" title="退出登录" @click="signOut"><WorkspaceIcon name="logout" /></button>
      </div>
    </el-aside>

    <el-container class="app-content">
      <el-header class="workspace-topbar mobile-topbar">
        <button
          ref="mobileMenuButton"
          type="button"
          class="mobile-menu-button"
          aria-label="打开导航"
          aria-controls="mobile-navigation"
          :aria-expanded="drawerOpen"
          @click="openDrawer"
        ><WorkspaceIcon name="menu" /></button>
        <div class="workspace-topbar__identity"><span class="workspace-topbar__brand">LEARNOS</span><span class="workspace-topbar__separator" aria-hidden="true">/</span><span class="workspace-topbar__location">{{ currentPageLabel }}</span></div>
        <div class="workspace-topbar__spacer" />
      </el-header>
      <el-main class="main-content">
        <router-view />
      </el-main>
    </el-container>

    <el-drawer
      v-model="drawerOpen"
      title="LearnOS"
      direction="ltr"
      size="280px"
      class="mobile-nav-drawer"
      :lock-scroll="false"
      :close-on-click-modal="true"
      :close-on-press-escape="true"
      destroy-on-close
      @opened="focusDrawer"
      @closed="restoreFocus"
    >
      <nav id="mobile-navigation" class="nav-menu mobile-nav-menu" aria-label="移动端主要导航">
        <RouterLink v-for="item in navigation" :key="item.to" :to="item.to" class="nav-link" :aria-current="activePath === item.to ? 'page' : undefined" @click="navigateFromDrawer(item.to)">
          <WorkspaceIcon :name="item.icon" />{{ item.label }}
        </RouterLink>
      </nav>
      <el-button class="mobile-logout" text @click="signOut">退出登录</el-button>
    </el-drawer>
  </el-container>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { currentUser, logout, type User } from '@/api/auth'
import WorkspaceIcon from '@/components/WorkspaceIcon.vue'

const route = useRoute()
const router = useRouter()
const current = ref<User | null>(null)
const baseNavigation: { to: string; label: string; icon: 'life' | 'home' | 'inbox' | 'courses' | 'projects' | 'explore' | 'settings' | 'users' }[] = [
  { to: '/', label: '工作区', icon: 'home' },
  { to: '/learn', label: '学习', icon: 'courses' },
  { to: '/life', label: '生活', icon: 'life' },
  { to: '/projects', label: '项目', icon: 'projects' },
  { to: '/inbox', label: '收集箱', icon: 'inbox' },
  { to: '/explore', label: '探索', icon: 'explore' },
  { to: '/settings', label: '系统设置', icon: 'settings' },
  { to: '/users', label: '用户管理', icon: 'users' },
]
const navigation = computed(() => current.value?.role === 'admin'
  ? baseNavigation
  : baseNavigation.filter(item => item.to !== '/users' && item.to !== '/settings'))
const drawerOpen = ref(false)
const mobileMenuButton = ref<HTMLElement | { $el: HTMLElement } | null>(null)
let focusAfterClose: 'button' | 'heading' = 'button'
const activePath = computed(() => {
  if (route.path === '/learn') return '/learn'
  if (route.path.startsWith('/courses')) return '/learn'
  if (route.path.startsWith('/life')) return '/life'
  if (route.path.startsWith('/inbox')) return '/inbox'
  if (route.path.startsWith('/projects')) return '/projects'
  if (route.path.startsWith('/exploration') || route.path.startsWith('/explore')) return '/explore'
  if (route.path.startsWith('/settings')) return '/settings'
  if (route.path.startsWith('/users')) return '/users'
  return '/'
})
const currentPageLabel = computed(() => baseNavigation.find(item => item.to === activePath.value)?.label ?? '工作区')

function openDrawer() {
  focusAfterClose = 'button'
  drawerOpen.value = true
}

async function signOut() {
  try { await logout() } finally { current.value = null; await router.replace('/login') }
}

onMounted(async () => {
  try { current.value = await currentUser() } catch { current.value = null }
})

function focusDrawer() {
  void nextTick(() => document.querySelector<HTMLElement>('.mobile-nav-drawer .nav-link')?.focus())
}

function navigateFromDrawer(_path: string) {
  focusAfterClose = 'heading'
  drawerOpen.value = false
}

function menuButtonElement() {
  const target = mobileMenuButton.value
  return target instanceof HTMLElement ? target : target?.$el ?? null
}

function restoreFocus() {
  void nextTick(() => {
    if (focusAfterClose === 'heading') {
      const heading = document.querySelector<HTMLElement>('.main-content h1')
      if (heading) {
        heading.focus()
        return
      }
    }
    menuButtonElement()?.focus()
  })
}

watch(() => route.fullPath, () => {
  if (!drawerOpen.value) return
  focusAfterClose = 'heading'
  drawerOpen.value = false
})
</script>
