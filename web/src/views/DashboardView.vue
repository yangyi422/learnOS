<template>
  <section>
    <div class="hero-panel">
      <div>
        <span class="eyebrow">CURRENT FOCUS</span>
        <h2>先完成一条真实学习闭环</h2>
        <p>
          从当前知识点开始，提交你的理解，查看结构化学习反馈并留下可追溯的学习记录。
        </p>
      </div>
      <el-button type="primary" size="large" :disabled="loading || courses.length === 0" @click="continueLearning">
        继续学习
      </el-button>
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
import { useRouter } from 'vue-router'
import CourseCard from '@/components/CourseCard.vue'
import { listCourses } from '@/api/courses'
import type { Course } from '@/types/course'

const courses = ref<Course[]>([])
const loading = ref(false)
const error = ref('')
const router = useRouter()

function continueLearning() {
  const course = courses.value[0]
  if (course) {
    router.push(`/courses/${course.id}/learn`)
  }
}

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
