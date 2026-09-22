<template>
  <el-alert class="ai-request-error" type="error" show-icon :closable="false">
    <template #title>{{ message }}</template>
    <p class="error-detail">本次操作没有修改你的学习数据。</p>
    <el-button v-if="retryable && retry" size="small" type="danger" plain @click="retry">重试</el-button>
  </el-alert>
</template>

<script setup lang="ts">
defineProps<{ message: string; retryable?: boolean; retry?: () => void }>()
</script>

<style scoped>
.error-detail { margin: 6px 0 10px; }
</style>
