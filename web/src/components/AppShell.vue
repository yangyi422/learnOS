<template>
  <el-container class="app-shell">
    <el-aside width="208px" class="sidebar">
      <div class="brand">
        <div class="brand-mark">L</div>
        <div>
          <strong>LearnOS</strong>
          <span>个人学习系统</span>
        </div>
      </div>

      <nav class="nav-menu" aria-label="主要导航">
        <RouterLink v-for="item in navigation" :key="item.to" :to="item.to" class="nav-link" :aria-current="activePath === item.to ? 'page' : undefined">
        {{ item.label }}
      </RouterLink>
      </nav>

      <div class="sidebar-footer">
        <span>v0.1</span>
        <el-button text size="small" @click="signOut">退出登录</el-button>
      </div>
    </el-aside>

    <el-container class="app-content">
      <el-header class="mobile-topbar">
        <el-button
          ref="mobileMenuButton"
          text
          class="mobile-menu-button"
          aria-label="打开导航"
          aria-controls="mobile-navigation"
          :aria-expanded="drawerOpen"
          @click="openDrawer"
        >☰</el-button>
        <strong>LearnOS</strong>
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
          {{ item.label }}
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

const route = useRoute()
const router = useRouter()
const current = ref<User | null>(null)
const baseNavigation = [
  { to: '/', label: '学习首页' },
  { to: '/courses', label: '课程档案' },
  { to: '/exploration/questions', label: '探索空间' },
  { to: '/settings', label: '系统设置' },
  { to: '/users', label: '用户管理' },
]
const navigation = computed(() => current.value?.role === 'admin'
  ? baseNavigation
  : baseNavigation.filter(item => item.to !== '/users' && item.to !== '/settings'))
const drawerOpen = ref(false)
const mobileMenuButton = ref<HTMLElement | { $el: HTMLElement } | null>(null)
let focusAfterClose: 'button' | 'heading' = 'button'
const activePath = computed(() => {
  if (route.path.startsWith('/courses')) return '/courses'
  if (route.path.startsWith('/exploration')) return '/exploration/questions'
  if (route.path.startsWith('/settings')) return '/settings'
  if (route.path.startsWith('/users')) return '/users'
  return '/'
})

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
