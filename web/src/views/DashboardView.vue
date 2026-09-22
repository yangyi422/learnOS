<template>
  <section class="dashboard-page">
    <PageHeader title="今天继续什么？">
      <template #actions><el-button type="primary" @click="router.push('/domains/new')">+ 新建学习领域</el-button></template>
    </PageHeader>

    <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />
    <CurrentFocusPanel v-if="focusCourse" :course="focusCourse" :lesson="focusLesson" :cognitive="focusCognitive" :loading="focusLoading" @continue="continueFocus" />
    <EmptyState v-else-if="!loading" title="你的知识世界还是空的" description="从一个你真正想了解的领域开始。LearnOS 会先建立领域地图，再逐步展开适合你的学习区域。" action-label="创建第一个学习领域" @action="router.push('/domains/new')" />

    <section class="dashboard-section">
      <SectionHeader title="知识世界" description="每个学习领域都有自己的地图、进度和学习位置。">
        <template #actions><el-button text @click="loadCourses">刷新知识世界</el-button></template>
      </SectionHeader>
      <div v-loading="loading" class="domain-grid">
        <DomainTile v-for="course in courses" :key="course.id" :course="course" @open="openCourse(course)" @map="openMap(course)" @delete="loadCourses" />
        <button class="add-domain-tile" type="button" @click="router.push('/domains/new')"><span>＋</span><strong>新建学习领域</strong><small>建立一个独立的知识世界</small></button>
      </div>
    </section>

    <section class="dashboard-section dashboard-section--exploration">
      <SectionHeader title="探索" description="从当前学习轨迹发现值得追问的新区域。">
        <template #actions><el-button text @click="router.push('/exploration/questions')">打开探索空间</el-button></template>
      </SectionHeader>
      <div v-loading="radarLoading" class="discovery-grid">
        <DiscoveryCard v-for="direction in directions.slice(0, 3)" :key="direction.id" :direction="direction" @open="openDirection(direction)" />
        <EmptyState v-if="!radarLoading && directions.length === 0" title="还没有新的探索方向" description="继续完成几次学习回答，探索雷达会在这里留下新的入口。" />
      </div>
    </section>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import CurrentFocusPanel from '@/components/CurrentFocusPanel.vue'
import DiscoveryCard from '@/components/DiscoveryCard.vue'
import DomainTile from '@/components/DomainTile.vue'
import EmptyState from '@/components/EmptyState.vue'
import PageHeader from '@/components/PageHeader.vue'
import SectionHeader from '@/components/SectionHeader.vue'
import { listCourses } from '@/api/courses'
import { getExplorationRadar, updateDirection } from '@/api/exploration'
import { getCurrentLesson } from '@/api/learning'
import { getLessonCognitiveState } from '@/api/cognitive'
import type { Course } from '@/types/course'
import type { ExplorationDirection } from '@/types/exploration'
import type { CurrentLessonData } from '@/api/learning'
import type { CognitiveLevel, CognitiveStatus } from '@/types/cognitive'

const courses = ref<Course[]>([])
const loading = ref(false)
const error = ref('')
const directions = ref<ExplorationDirection[]>([])
const radarLoading = ref(false)
const focusCourse = ref<Course | null>(null)
const focusLesson = ref<CurrentLessonData | null>(null)
const focusCognitive = ref<{ current_level: CognitiveLevel; status: CognitiveStatus } | undefined>()
const focusLoading = ref(false)
const router = useRouter()

async function loadCourses() {
  loading.value = true
  error.value = ''
  try {
    courses.value = await listCourses()
    await loadFocus()
    await loadRadar()
  } catch (reason) {
    error.value = reason instanceof Error ? reason.message : '课程读取失败'
  } finally {
    loading.value = false
  }
}

async function loadFocus() {
  focusCourse.value = [...courses.value].sort((left, right) => {
    const leftTime = left.last_studied_at ? new Date(left.last_studied_at).getTime() : 0
    const rightTime = right.last_studied_at ? new Date(right.last_studied_at).getTime() : 0
    return rightTime - leftTime || Number(Boolean(right.current_lesson_id)) - Number(Boolean(left.current_lesson_id)) || right.coverage_progress - left.coverage_progress || right.mastery_progress - left.mastery_progress
  })[0] ?? null
  focusLesson.value = null
  focusCognitive.value = undefined
  if (!focusCourse.value || !focusCourse.value.current_lesson_id) return
  focusLoading.value = true
  try {
    focusLesson.value = await getCurrentLesson(focusCourse.value.id)
    const cognitive = await getLessonCognitiveState(focusCourse.value.id, focusLesson.value.lesson.id)
    focusCognitive.value = { current_level: cognitive.state.current_level, status: cognitive.state.status }
  } catch {
    focusLesson.value = null
  } finally { focusLoading.value = false }
}

function continueFocus() {
  if (!focusCourse.value) return
  router.push(focusLesson.value ? `/courses/${focusCourse.value.id}/learn` : `/courses/${focusCourse.value.id}/map`)
}

function openCourse(course: Course) { router.push(course.current_lesson_id ? `/courses/${course.id}/learn` : `/courses/${course.id}/map`) }
function openMap(course: Course) { router.push(`/courses/${course.id}/map`) }
async function loadRadar() {
  if (courses.value.length === 0) {
    directions.value = []
    return
  }
  radarLoading.value = true
  try {
    const results = await Promise.allSettled(courses.value.map((course) => getExplorationRadar(course.id, 0, 3)))
    directions.value = results.flatMap((result) => result.status === 'fulfilled' ? result.value : []).sort((left, right) => right.score - left.score).slice(0, 3)
  } catch { directions.value = [] } finally { radarLoading.value = false }
}

async function openDirection(direction: ExplorationDirection) {
  try {
    await updateDirection(direction.id, 'open', direction.context_course_id)
    router.push(`/courses/${direction.target_course_id}/learn?lesson_id=${direction.target_lesson_id}`)
  } catch (reason) { error.value = reason instanceof Error ? reason.message : '探索方向打开失败' }
}

onMounted(loadCourses)
</script>
