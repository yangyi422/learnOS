<template>
  <section>
    <div class="hero-panel">
      <div>
        <span class="eyebrow">CURRENT FOCUS</span>
        <h2>先完成一条真实学习闭环</h2>
        <p>
          当前版本负责课程持久化与基础展示。下一阶段将实现“读取当前问题 → 提交回答 → DeepSeek 评估 → 更新课程状态”。
        </p>
      </div>
      <el-button type="primary" size="large" disabled>继续学习</el-button>
    </div>

    <div class="section-heading">
      <div>
        <span class="eyebrow">COURSES</span>
        <h2>正在学习</h2>
      </div>
      <el-button text @click="loadCourses">刷新</el-button>
    </div>

    <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />
    <div v-loading="loading" class="course-grid">
      <CourseCard v-for="course in courses" :key="course.id" :course="course" />
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import CourseCard from '@/components/CourseCard.vue'
import { listCourses } from '@/api/courses'
import type { Course } from '@/types/course'

const courses = ref<Course[]>([])
const loading = ref(false)
const error = ref('')

async function loadCourses() {
  loading.value = true
  error.value = ''
  try {
    courses.value = await listCourses()
  } catch (reason) {
    error.value = reason instanceof Error ? reason.message : '课程读取失败'
  } finally {
    loading.value = false
  }
}

onMounted(loadCourses)
</script>
