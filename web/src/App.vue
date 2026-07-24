<template>
  <el-container class="app-shell">
    <el-aside width="240px" class="sidebar">
      <div class="brand">
        <div class="brand-mark">L</div>
        <div>
          <strong>LearnOS</strong>
          <span>个人学习系统</span>
        </div>
      </div>

      <el-menu router :default-active="$route.path" class="nav-menu">
        <el-menu-item index="/">学习首页</el-menu-item>
        <el-menu-item index="/courses">课程档案</el-menu-item>
        <el-menu-item index="/settings">系统设置</el-menu-item>
      </el-menu>
    </el-aside>

    <el-container>
      <el-header class="topbar">
        <div>
          <span class="eyebrow">PERSONAL LEARNING AGENT</span>
          <h1>{{ routeTitle }}</h1>
        </div>
        <el-tag effect="plain" type="success">MVP · Phase 1</el-tag>
      </el-header>

      <el-main class="main-content">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()
const routeTitle = computed(() => String(route.meta.title ?? 'LearnOS'))
</script>
