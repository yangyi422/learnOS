<template>
  <el-card shadow="never">
    <template #header>课程档案</template>
    <el-empty description="课程档案将在后续阶段展开；当前学习记录可在学习页查看。">
      <el-button type="primary" @click="router.push(`/courses/${courseID}/learn`)">返回学习</el-button>
    </el-empty>
  </el-card>
</template>

<script setup lang="ts">
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()
const courseID = Number(String(route.params.id))
</script>
