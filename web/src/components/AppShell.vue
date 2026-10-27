<template>
  <el-container class="app-shell" :class="{ 'app-shell--home': route.path === '/' }">
    <el-container class="app-content" direction="vertical">
      <div class="home-ambient" aria-hidden="true"><HomeAmbient /></div>
      <header class="home-topbar">
        <button ref="mobileMenuButton" type="button" class="mobile-menu-button" aria-label="打开导航" aria-controls="mobile-navigation" :aria-expanded="drawerOpen" @click="openDrawer"><WorkspaceIcon name="menu" /></button>
        <RouterLink to="/" class="home-brand" aria-label="LearnOS 工作区"><strong>LifeOS</strong><span>LearnOS</span></RouterLink>
        <nav class="home-topnav" aria-label="主要导航">
          <RouterLink v-for="item in homeNavigation" :key="item.to" :to="item.to" :aria-current="activePath === item.to ? 'page' : undefined"><WorkspaceIcon :name="item.icon" /><span>{{ item.label }}</span><span v-if="item.to === '/inbox'" id="home-inbox-count" class="home-topnav__badge" /></RouterLink>
        </nav>
        <div id="home-header-actions" class="home-header-actions" />
        <el-dropdown trigger="click" class="home-account">
          <button type="button" class="home-account__button" aria-label="个人空间菜单"><span class="home-account__avatar">{{ Array.from(current?.display_name || current?.username || '我')[0] }}</span><span class="home-account__name">{{ current?.display_name || current?.username || '个人空间' }}</span><svg viewBox="0 0 16 16" aria-hidden="true"><path d="m4 6 4 4 4-4" /></svg></button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item @click="appearanceOpen = true">外观设置</el-dropdown-item>
              <el-dropdown-item @click="router.push('/explore')">探索空间</el-dropdown-item>
              <el-dropdown-item v-if="current?.role === 'admin'" @click="router.push('/admin/visual-effects')">视觉实验室</el-dropdown-item>
              <el-dropdown-item v-if="current?.role === 'admin'" @click="router.push('/settings')">系统设置</el-dropdown-item>
              <el-dropdown-item v-if="current?.role === 'admin'" @click="router.push('/users')">用户管理</el-dropdown-item>
              <el-dropdown-item divided @click="signOut">退出登录</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </header>
      <el-main class="main-content">
        <router-view />
      </el-main>
    </el-container>

    <HomeAppearanceDialog v-model="appearanceOpen" />
    <el-drawer append-to-body
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
      <el-button text @click="appearanceOpen = true; drawerOpen = false">外观设置</el-button>
      <el-button class="mobile-logout" text @click="signOut">退出登录</el-button>
    </el-drawer>
  </el-container>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { currentUser, logout, type User } from '@/api/auth'
import WorkspaceIcon from '@/components/WorkspaceIcon.vue'
import HomeAppearanceDialog from '@/components/HomeAppearanceDialog.vue'
import HomeAmbient from '@/components/ambient/HomeAmbient.vue'

const route = useRoute()
const router = useRouter()
const homeNavigation = [
  { to: '/', label: '工作台', icon: 'home' as const },
  { to: '/learn', label: '学习', icon: 'courses' as const },
  { to: '/projects', label: '项目', icon: 'projects' as const },
  { to: '/life', label: '生活', icon: 'life' as const },
  { to: '/inbox', label: '收集箱', icon: 'inbox' as const },
  { to: '/explore/library', label: '资料库', icon: 'courses' as const },
]
const current = ref<User | null>(null)
const baseNavigation: { to: string; label: string; icon: 'life' | 'home' | 'inbox' | 'courses' | 'projects' | 'explore' | 'settings' | 'users' }[] = [
  { to: '/', label: '工作台', icon: 'home' },
  { to: '/learn', label: '学习', icon: 'courses' },
  { to: '/life', label: '生活', icon: 'life' },
  { to: '/projects', label: '项目', icon: 'projects' },
  { to: '/inbox', label: '收集箱', icon: 'inbox' },
  { to: '/explore/library', label: '资料库', icon: 'courses' },
  { to: '/explore', label: '探索', icon: 'explore' },
  { to: '/settings', label: '系统设置', icon: 'settings' },
  { to: '/users', label: '用户管理', icon: 'users' },
]
const navigation = computed(() => current.value?.role === 'admin'
  ? baseNavigation
  : baseNavigation.filter(item => item.to !== '/users' && item.to !== '/settings'))
const drawerOpen = ref(false)
const appearanceOpen = ref(false)
const mobileMenuButton = ref<HTMLElement | { $el: HTMLElement } | null>(null)
let focusAfterClose: 'button' | 'heading' = 'button'
const activePath = computed(() => {
  if (route.path === '/learn') return '/learn'
  if (route.path.startsWith('/courses')) return '/learn'
  if (route.path.startsWith('/life')) return '/life'
  if (route.path.startsWith('/inbox')) return '/inbox'
  if (route.path.startsWith('/projects')) return '/projects'
  if (route.path.startsWith('/explore/library')) return '/explore/library'
  if (route.path.startsWith('/domains')) return '/learn'
  if (route.path.startsWith('/exploration') || route.path.startsWith('/explore')) return '/explore'
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


<style scoped>
.app-shell { --sidebar-width: 0px; }
/* The document scrolls; this separate, non-interactive layer stays in the content viewport. */
.app-shell .app-content { isolation: isolate; background: var(--home-canvas); }
.home-ambient {
  position: fixed; inset: 76px 0 0; z-index: -1;
  pointer-events: none; overflow: hidden;
  background: var(--home-environment);
}
.app-shell .main-content { position: relative; background: transparent; }
/* One horizontal frame across all authenticated routes. */
.app-shell--home { --sidebar-width: 0px; }
.app-shell--home .app-content { background: var(--home-canvas); }
.app-shell--home .home-ambient { inset: 76px 0 0; background: var(--home-environment); }
.app-shell--home .main-content { padding: 48px 36px 28px; scrollbar-gutter: auto; }
.home-topbar { position: sticky; top: 0; z-index: 30; display: flex; align-items: center; gap: 24px; height: 76px; flex: none; padding: 0 36px; background: var(--home-bar); border-bottom: 1px solid var(--home-edge); backdrop-filter: blur(16px); color: var(--home-ink); }
.home-brand { display: flex; align-items: baseline; gap: 12px; text-decoration: none; color: var(--home-ink); flex: none; }
.home-brand strong { font-size: 26px; font-weight: 650; letter-spacing: -.6px; }
.home-brand > span { font-size: 12px; color: var(--home-muted); }
.home-topnav { position: absolute; left: var(--home-focus-axis,50%); transform: translateX(-50%); display: flex; gap: 4px; width: max-content; justify-content: center; }
.home-topnav > a { display: flex; align-items: center; gap: 6px; position: relative; flex: none; height: 42px; padding: 0 10px; border-radius: 12px; color: var(--home-secondary); text-decoration: none; font-size: 14px; transition: background-color 150ms ease; }
.home-topnav > a:hover, .home-topnav > a[aria-current='page'] { background: var(--home-tint); color: var(--home-ink); }
.home-topnav svg { width: 18px; height: 18px; }
.home-topnav__badge:empty { display: none; }
.home-topnav__badge { display: grid; place-items: center; min-width: 18px; height: 18px; padding: 0 4px; border-radius: 99px; font-size: 10px; color: var(--home-link); background: var(--home-tint); }
.home-header-actions { display: flex; align-items: center; gap: 12px; margin-left: auto; min-width: 0; }
.home-account { color: var(--home-ink); flex: none; }
.home-account__button { display: flex; align-items: center; gap: 8px; padding: 0; border: 0; background: transparent; color: var(--home-ink); cursor: pointer; font: inherit; }
.home-account__button > svg { width: 12px; height: 12px; fill: none; stroke: currentColor; stroke-width: 1.5; }
.home-account__avatar { display: grid; place-items: center; width: 32px; height: 32px; border-radius: 50%; border: 1px solid var(--home-edge); background: var(--home-tint); color: var(--home-link); font-size: 13px; }
.home-account__name { max-width: 88px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 13px; }
.home-topbar .mobile-menu-button { background: transparent; color: var(--home-ink); border-color: var(--home-edge); }
@media (max-width: 1599px) { .home-account__name { display: none; } }
@media (max-width: 1399px) { .home-header-actions :deep(.home-search) { width: 100px; } .home-topbar { gap: 16px; padding-inline: 28px; } .home-topnav > a { padding-inline: 10px; gap: 6px; } .home-account__name { display: none; } }
@media (max-width: 1179px) { .home-topnav { display: none; } .home-topbar .mobile-menu-button { display: grid; } .home-brand { margin-right: auto; } }
@media (max-width: 700px) {
  .home-topbar { height: 68px; padding-inline: 16px; gap: 12px; }
  .home-brand strong { font-size: 20px; }
  .home-brand > span { display: none; }
  .home-header-actions { margin-left: 0; }
  .home-account__button { gap: 4px; }
  .home-account__button > svg { display: none; }
  .app-shell .home-ambient { inset-block-start: 68px; }
  .app-shell--home .main-content { padding: 28px 16px; }
}
</style>
