<template>
  <section class="page-stack course-library-page">
    <PageHeader title="你的知识世界" description="每个学习领域都拥有独立的课程结构与学习记录。">
      <template #actions>
        <el-button type="primary" @click="router.push('/domains/new')">+ 新建学习领域</el-button>
        <el-button text @click="loadCourses">刷新课程档案</el-button>
      </template>
    </PageHeader>

    <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />
    <div v-loading="loading" class="domain-grid domain-grid--library">
      <DomainTile v-for="course in courses" :key="course.id" :course="course" @open="openCourse(course)" @map="openMap(course)" @delete="loadCourses" />
      <button class="add-domain-tile" type="button" @click="router.push('/domains/new')"><span>＋</span><strong>新建学习领域</strong><small>建立一个独立的知识世界</small></button>
    </div>
    <EmptyState v-if="!loading && courses.length === 0" title="你的知识世界还是空的" description="从一个你真正想了解的领域开始。" action-label="创建第一个学习领域" @action="router.push('/domains/new')" />
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { listCourses } from '@/api/courses'
import type { Course } from '@/types/course'
import DomainTile from '@/components/DomainTile.vue'
import EmptyState from '@/components/EmptyState.vue'
import PageHeader from '@/components/PageHeader.vue'

const router = useRouter()
const courses = ref<Course[]>([])
const loading = ref(false)
const error = ref('')

async function loadCourses() {
  loading.value = true
  error.value = ''
  try { courses.value = await listCourses() } catch (reason) { error.value = reason instanceof Error ? reason.message : '课程读取失败' } finally { loading.value = false }
}

onMounted(loadCourses)

function openCourse(course: Course) { router.push(course.current_lesson_id ? `/courses/${course.id}/learn` : `/courses/${course.id}/map`) }
function openMap(course: Course) { router.push(`/courses/${course.id}/map`) }
</script>
