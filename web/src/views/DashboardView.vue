<template>
  <section class="dashboard-page">
    <PageHeader v-if="loading || error || courses.length > 0" :title="pageTitle" />

    <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />
    <div v-if="loading" v-loading="loading" class="dashboard-loading" aria-label="正在读取学习空间" />
    <template v-else-if="!error && courses.length === 0">
      <section class="dashboard-onboarding" aria-labelledby="dashboard-onboarding-title">
        <div class="dashboard-onboarding__mark" aria-hidden="true"><WorkspaceIcon name="explore" /></div>
        <h1 id="dashboard-onboarding-title" tabindex="-1">从一个好奇的问题开始</h1>
        <p>从一个想弄明白的问题出发，逐步建立自己的学习路径。</p>
        <el-button type="primary" @click="router.push('/domains/new')">创建第一个学习领域</el-button>
      </section>
      <p class="dashboard-onboarding__hint">开始学习后，这里会逐步呈现新的探索方向。</p>
    </template>

    <template v-else-if="courses.length > 0">
      <CurrentFocusPanel v-if="focusCourse" :course="focusCourse" :lesson="focusLesson" :cognitive="focusCognitive" :loading="focusLoading" @continue="continueFocus" />

      <section class="dashboard-section">
        <SectionHeader title="知识世界">
          <template #actions>
            <el-button text @click="loadCourses">刷新</el-button>
            <el-button @click="router.push('/domains/new')">+ 新建学习领域</el-button>
          </template>
        </SectionHeader>
        <div class="domain-grid dashboard-domain-grid">
          <DomainTile v-for="course in courses" :key="course.id" :course="course" @open="openCourse(course)" @map="openMap(course)" @delete="loadCourses" />
        </div>
      </section>

      <section class="dashboard-section dashboard-section--exploration">
        <SectionHeader title="探索">
          <template #actions><el-button text @click="router.push('/exploration/questions')">打开探索空间</el-button></template>
        </SectionHeader>
        <div v-if="directions.length" class="discovery-grid">
          <DiscoveryCard v-for="direction in directions.slice(0, 3)" :key="direction.id" :direction="direction" @open="openDirection(direction)" />
        </div>
        <p v-else-if="!radarLoading" class="dashboard-exploration-empty">完成一些学习回答后，这里会逐步出现值得追问的方向。</p>
      </section>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import CurrentFocusPanel from '@/components/CurrentFocusPanel.vue'
import DiscoveryCard from '@/components/DiscoveryCard.vue'
import DomainTile from '@/components/DomainTile.vue'
import PageHeader from '@/components/PageHeader.vue'
import SectionHeader from '@/components/SectionHeader.vue'
import WorkspaceIcon from '@/components/WorkspaceIcon.vue'
import { listCourses } from '@/api/courses'
import { getExplorationRadar, updateDirection } from '@/api/exploration'
import { getCurrentLesson } from '@/api/learning'
import { getLessonCognitiveState } from '@/api/cognitive'
import type { Course } from '@/types/course'
import type { ExplorationDirection } from '@/types/exploration'
import type { CurrentLessonData } from '@/api/learning'
import type { CognitiveLevel, CognitiveStatus } from '@/types/cognitive'

const courses = ref<Course[]>([])
const loading = ref(true)
const error = ref('')
const directions = ref<ExplorationDirection[]>([])
const radarLoading = ref(false)
const focusCourse = ref<Course | null>(null)
const focusLesson = ref<CurrentLessonData | null>(null)
const focusCognitive = ref<{ current_level: CognitiveLevel; status: CognitiveStatus } | undefined>()
const focusLoading = ref(false)
const router = useRouter()
const pageTitle = computed(() => {
  if (loading.value || error.value) return '学习空间'
  return courses.value.some(course => course.current_lesson_id) ? '继续你的学习' : '让学习从这里展开'
})

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
    const lessonDifference = Number(Boolean(right.current_lesson_id)) - Number(Boolean(left.current_lesson_id))
    if (lessonDifference) return lessonDifference
    const leftTime = left.last_studied_at ? new Date(left.last_studied_at).getTime() : 0
    const rightTime = right.last_studied_at ? new Date(right.last_studied_at).getTime() : 0
    return rightTime - leftTime || right.coverage_progress - left.coverage_progress || right.mastery_progress - left.mastery_progress
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

<style scoped>
.dashboard-page { max-width: 1440px; }
.dashboard-page :deep(.page-header) { margin-bottom: 20px; }
.dashboard-page :deep(.page-header h1) { font-size: clamp(29px, 2.5vw, 36px); line-height: 1.15; }
.dashboard-loading { min-height: 160px; }
.dashboard-onboarding { width: min(100%, 680px); margin: clamp(58px, 11vh, 108px) auto 0; padding: 38px 42px; border: 1px solid var(--border-default); border-radius: 10px; background: linear-gradient(128deg, var(--color-primary-soft), var(--bg-surface) 58%); box-shadow: var(--shadow-card); }
.dashboard-onboarding__mark { display: grid; width: 43px; height: 43px; place-items: center; margin-bottom: 20px; border: 1px solid var(--border-default); border-radius: 9px; background: var(--bg-surface); color: var(--color-primary); }
.dashboard-onboarding__mark svg { width: 22px; height: 22px; }
.dashboard-onboarding h1 { margin: 0; font-size: clamp(26px, 2.6vw, 32px); font-weight: 680; letter-spacing: -.035em; line-height: 1.3; }
.dashboard-onboarding p { margin: 13px 0 24px; color: var(--text-secondary); font-size: 15px; line-height: 1.7; }
.dashboard-onboarding .el-button { min-height: 40px; padding: 0 18px; }
.dashboard-onboarding__hint { margin: 17px auto 0; color: var(--text-tertiary); font-size: 13px; text-align: center; }
.dashboard-page .dashboard-section { margin-top: 24px; }
.dashboard-page .dashboard-section .section-header { margin-bottom: 13px; }
.dashboard-page .dashboard-section .section-header h2 { font-size: 19px; }
.dashboard-domain-grid { grid-template-columns: repeat(auto-fill, minmax(220px, 270px)); gap: 12px; }
.dashboard-domain-grid :deep(.domain-tile) { min-height: 0; }
.dashboard-domain-grid :deep(.domain-tile h3) { margin: 14px 0 12px; }
.dashboard-domain-grid :deep(.domain-tile p) { margin: 10px 0 8px; }
.dashboard-exploration-empty { margin: 0; padding: 4px 0 0; color: var(--text-tertiary); font-size: 13px; line-height: 1.6; }
@media (max-width: 760px) { .dashboard-onboarding { margin-top: 26px; padding: 28px 24px; } .dashboard-domain-grid { grid-template-columns: repeat(auto-fit, minmax(min(100%, 220px), 1fr)); } }
</style>
