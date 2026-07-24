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

    <el-button type="primary" class="course-card__action" disabled>
      学习页将在下一阶段实现
    </el-button>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Course } from '@/types/course'

const props = defineProps<{ course: Course }>()

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
