<template>
  <main class="login-page">
    <el-card class="login-card" shadow="never">
      <p class="eyebrow">LearnOS</p>
      <h1>登录你的学习空间</h1>
      <p class="muted">登录后只能访问属于自己的课程与学习记录。</p>
      <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />
      <el-form label-position="top" @submit.prevent="submit">
        <el-form-item label="用户名"><el-input v-model="username" autocomplete="username" /></el-form-item>
        <el-form-item label="密码"><el-input v-model="password" type="password" show-password autocomplete="current-password" @keyup.enter="submit" /></el-form-item>
        <el-button type="primary" native-type="submit" :loading="loading" style="width:100%">登录</el-button>
      </el-form>
    </el-card>
  </main>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { login } from '@/api/auth'

const router = useRouter()
const username = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

async function submit() {
  error.value = ''
  loading.value = true
  try { await login(username.value, password.value); await router.replace('/') }
  catch (reason) { error.value = reason instanceof Error ? reason.message : '登录失败' }
  finally { loading.value = false }
}
</script>

<style scoped>
.login-page { min-height: 100vh; display: grid; place-items: center; padding: 32px 20px; background: #f7f8fb; }
.login-card { width: min(100%, 420px); padding: 12px; }
.login-card h1 { margin: 0 0 10px; }
.login-card .muted { margin: 0 0 24px; color: #667085; }
.login-card .el-alert { margin-bottom: 18px; }
</style>
