<template>
  <article class="course-card">
    <div class="course-card__topline">
      <el-tag :type="statusType" effect="light">{{ statusText }}</el-tag>
      <span>{{ course.progress }}%</span>
    </div>

    <h3>{{ course.name }}</h3>
    <p>{{ course.description }}</p>

    <div class="course-card__meta">
      <span>当前模块</span>
      <strong>{{ course.current_unit || '尚未开始' }}</strong>
    </div>

    <el-progress :percentage="course.progress" :stroke-width="8" />

    <div class="course-card__actions">
      <el-button type="primary" @click="goToLearning">继续学习</el-button>
      <el-button @click="goToMap">查看知识结构</el-button>
    </div>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import type { Course } from '@/types/course'

const props = defineProps<{ course: Course }>()
const router = useRouter()

function goToLearning() {
  router.push(`/courses/${props.course.id}/learn`)
}

function goToMap() {
  router.push(`/courses/${props.course.id}/map`)
}

const statusText = computed(() => ({
  initializing: '初始化中',
  learning: '学习中',
  paused: '已暂停',
  completed: '已完成',
})[props.course.status])

const statusType = computed(() => ({
  initializing: 'warning',
  learning: 'primary',
  paused: 'info',
  completed: 'success',
})[props.course.status] as 'primary' | 'success' | 'warning' | 'info')
</script>
