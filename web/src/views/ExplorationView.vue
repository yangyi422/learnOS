<template>
  <section class="exploration-page">
    <el-card v-if="!courses.length && !loading" shadow="never" class="empty-exploration">
      <el-empty description="探索空间将在你建立第一个学习领域后开始工作"><el-button type="primary" @click="router.push('/domains/new')">创建学习领域</el-button></el-empty>
    </el-card>
    <PageHeader eyebrow="探索引擎" title="探索空间" description="从当前课程旁支看看其他知识连接。探索不会改变课程主线。">
      <template #actions>
        <div v-if="courses.length" class="exploration-domain-selector">
          <div class="exploration-domain-selector__label">
            <label for="exploration-course-select">当前探索领域</label>
            <span v-if="courseSwitching" role="status">切换中…</span>
          </div>
          <el-select
            id="exploration-course-select"
            v-model="courseID"
            class="exploration-course-select"
            aria-label="当前探索领域"
            placeholder="选择课程"
            :loading="courseSwitching"
            @change="switchCourse"
          >
            <el-option v-for="course in courses" :key="course.id" :label="course.name" :value="course.id" />
          </el-select>
        </div>
      </template>
    </PageHeader>

    <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />
    <el-tabs v-model="activeTab" class="exploration-tabs">
      <el-tab-pane label="探索雷达" name="radar">
        <el-card shadow="never" class="exploration-card unfamiliar-card">
          <div class="unfamiliar-card__content">
            <div>
              <span class="eyebrow">陌生知识</span>
              <h3>发现一个陌生知识</h3>
              <p class="muted-text">从其他学习领域尚未接触的正式节点中抽取；不含当前领域和已掌握节点，连续发现时尽量不重复。</p>
            </div>
            <el-button type="warning" plain :loading="unfamiliarLoading" @click="discoverUnfamiliar">发现一个陌生知识</el-button>
          </div>
          <div v-if="unfamiliarDirection" class="unfamiliar-result">
            <div class="history-item__meta"><el-tag size="small" type="warning">陌生知识</el-tag><span>{{ unfamiliarDirection.target_course.name }}</span></div>
            <h3>{{ unfamiliarDirection.title }}</h3>
            <p>{{ unfamiliarDirection.summary }}</p>
            <p><strong>为什么现在值得探索：</strong>{{ unfamiliarDirection.why_worth_exploring }}</p>
            <div class="exploration-actions">
              <el-button size="small" type="primary" @click="open(unfamiliarDirection)">开始探索</el-button>
              <template v-if="unfamiliarDirection.status === 'saved'"><el-button size="small" disabled>已保存</el-button><el-button size="small" text @click="undoQuestion(unfamiliarDirection)">撤销</el-button></template>
              <el-button v-else size="small" @click="makeQuestion(unfamiliarDirection)">保存到问题池</el-button>
            </div>
          </div>
        </el-card>

        <section class="exploration-workspace" v-loading="loading">
          <div class="exploration-workspace__header"><div><span class="eyebrow">探索雷达</span><h2>值得打开的新区域</h2></div><el-button text @click="refreshRadar">刷新探索推荐</el-button></div>
          <div class="discovery-grid discovery-grid--page">
            <DiscoveryCard v-for="direction in directions" :key="direction.id" :direction="direction" @open="open(direction)" @save="makeQuestion(direction)" @undo="undoQuestion(direction)" />
            <EmptyState v-if="!loading && directions.length === 0" title="暂时没有可推荐的探索方向" description="完成更多学习回答后，探索雷达会继续寻找新的连接。" />
          </div>
        </section>
      </el-tab-pane>

      <el-tab-pane label="问题池" name="questions">
        <el-card shadow="never" class="exploration-card" v-loading="questionsLoading">
          <template #header><div class="card-header-row"><span>问题池</span><el-button text @click="refreshQuestions">刷新问题池</el-button></div></template>
          <div class="question-pool-filters" aria-label="问题池筛选">
            <el-select v-model="questionDomainID" placeholder="全部领域" clearable @change="loadQuestions()"><el-option v-for="course in courses" :key="course.id" :label="course.name" :value="course.id" /></el-select>
            <el-select v-model="questionPriority" placeholder="全部优先级" clearable @change="loadQuestions()"><el-option label="高优先级" value="high" /><el-option label="普通" value="normal" /><el-option label="低优先级" value="low" /></el-select>
            <el-select v-model="questionStatus" placeholder="全部状态" clearable @change="loadQuestions()"><el-option label="待探索" value="open" /><el-option label="探索中" value="exploring" /><el-option label="稍后学习" value="later" /><el-option label="已解决" value="resolved" /></el-select>
          </div>
          <el-empty v-if="!questionsLoading && questions.length === 0" description="还没有符合条件的探索问题"><el-button type="primary" plain @click="activeTab = 'radar'">返回探索雷达</el-button></el-empty>
          <div v-for="question in visibleQuestions" :key="question.id" class="exploration-item">
            <div>
              <div class="history-item__meta">
                <el-tag size="small" effect="plain">{{ questionTypeText(question.question_type) }}</el-tag>
                <el-tag size="small" :type="question.status === 'archived' ? 'info' : 'success'" effect="plain">{{ questionStatusText(question.status) }}</el-tag>
                <el-select :model-value="question.priority || 'normal'" size="small" aria-label="问题优先级" @change="setPriority(question, $event)"><el-option label="高优先级" value="high" /><el-option label="普通" value="normal" /><el-option label="低优先级" value="low" /></el-select>
              </div>
              <h3>{{ question.question }}</h3>
              <p v-if="question.context"><strong>Context：</strong>{{ question.context }}</p>
              <p><strong>WhyThisQuestion：</strong>{{ question.why_this_question }}</p>
              <p class="muted-text">来源：{{ question.source_course.name }}<span v-if="question.source_lesson"> · {{ question.source_lesson.title }}</span></p>
            </div>
            <div v-if="question.status !== 'archived'" class="exploration-actions">
              <el-button v-if="question.status !== 'resolved'" size="small" type="primary" @click="startQuestion(question)">开始探索</el-button>
              <el-button v-if="question.status !== 'later' && question.status !== 'resolved'" size="small" @click="changeQuestion(question, 'later')">稍后学习</el-button>
              <el-button v-if="question.status !== 'resolved'" size="small" @click="changeQuestion(question, 'resolve')">标记已解决</el-button>
              <el-button v-else size="small" @click="changeQuestion(question, 'reopen')">重新打开</el-button>
              <el-button size="small" text @click="archiveQuestion(question)">归档</el-button>
            </div>
          </div>
          <div v-if="visibleQuestions.length < questions.length" class="question-pool-more">
            <el-button plain @click="visibleQuestionCount += QUESTION_PAGE_SIZE">再显示 {{ Math.min(QUESTION_PAGE_SIZE, questions.length - visibleQuestions.length) }} 个问题</el-button>
          </div>
        </el-card>
      </el-tab-pane>
    </el-tabs>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { listCourses } from '@/api/courses'
import PageHeader from '@/components/PageHeader.vue'
import DiscoveryCard from '@/components/DiscoveryCard.vue'
import EmptyState from '@/components/EmptyState.vue'
import { addExplorationQuestion, findUnfamiliarKnowledge, getExplorationRadar, listExplorationQuestions, undoExplorationQuestion, updateDirection, updateQuestion, updateQuestionPriority } from '@/api/exploration'
import type { Course } from '@/types/course'
import type { ExplorationDirection, ExplorationQuestion } from '@/types/exploration'
import { isRequestAborted } from '@/api/http'

const router = useRouter()
const courses = ref<Course[]>([])
const courseID = ref(0)
const directions = ref<ExplorationDirection[]>([])
const questions = ref<ExplorationQuestion[]>([])
const loading = ref(false)
const questionsLoading = ref(false)
const courseSwitching = ref(false)
const error = ref('')
const activeTab = ref<'radar' | 'questions'>('radar')
const unfamiliarDirection = ref<ExplorationDirection | null>(null)
const unfamiliarLoading = ref(false)
const questionDomainID = ref<number | undefined>()
const questionPriority = ref('')
const questionStatus = ref('')
let selectionVersion = 0
let pageController: AbortController | null = null
const pendingActions = new Set<string>()
const QUESTION_PAGE_SIZE = 20
const visibleQuestionCount = ref(QUESTION_PAGE_SIZE)
const visibleQuestions = computed(() => questions.value.slice(0, visibleQuestionCount.value))

async function switchCourse() {
  const version = ++selectionVersion
  pageController?.abort()
  pageController = new AbortController()
  directions.value = []
  questions.value = []
  unfamiliarDirection.value = null
  error.value = ''
  courseSwitching.value = true
  try {
    await loadPage(version, courseID.value, pageController.signal)
  } finally {
    if (version === selectionVersion) courseSwitching.value = false
  }
}

async function loadPage(version = selectionVersion, selectedCourseID = courseID.value, signal = pageController?.signal) {
  if (!selectedCourseID) return
  await Promise.all([loadRadar(version, selectedCourseID, signal), loadQuestions(version, selectedCourseID, signal)])
}

async function loadRadar(version = selectionVersion, selectedCourseID = courseID.value, signal = pageController?.signal, force = false) {
  if (!selectedCourseID) return
  loading.value = true
  try {
    const result = await getExplorationRadar(selectedCourseID, 0, 6, { signal, force })
    if (version === selectionVersion && selectedCourseID === courseID.value) directions.value = result
  } catch (reason) {
    if (isRequestAborted(reason)) return
    if (version === selectionVersion) error.value = reason instanceof Error ? reason.message : '探索雷达读取失败'
  } finally {
    if (version === selectionVersion) loading.value = false
  }
}

async function loadQuestions(version = selectionVersion, selectedCourseID = courseID.value, signal = pageController?.signal) {
  if (!selectedCourseID) return
  questionsLoading.value = true
  try {
    const result = await listExplorationQuestions(selectedCourseID, { status: questionStatus.value, targetCourseID: questionDomainID.value, priority: questionPriority.value }, signal)
    if (version === selectionVersion && selectedCourseID === courseID.value) {
      questions.value = result
      visibleQuestionCount.value = QUESTION_PAGE_SIZE
    }
  } catch (reason) {
    if (isRequestAborted(reason)) return
    if (version === selectionVersion) error.value = reason instanceof Error ? reason.message : '问题池读取失败'
  } finally {
    if (version === selectionVersion) questionsLoading.value = false
  }
}

function refreshRadar() {
  void loadRadar(selectionVersion, courseID.value, pageController?.signal, true)
}

function refreshQuestions() {
  void loadQuestions(selectionVersion, courseID.value, pageController?.signal)
}

async function runOnce(key: string, action: () => Promise<void>) {
  if (pendingActions.has(key)) return
  pendingActions.add(key)
  try { await action() } finally { pendingActions.delete(key) }
}

async function discoverUnfamiliar() {
  if (!courseID.value) return
  const selectedCourseID = courseID.value
  const version = selectionVersion
  unfamiliarLoading.value = true
  error.value = ''
  try {
    const result = await findUnfamiliarKnowledge(selectedCourseID)
    if (version === selectionVersion && selectedCourseID === courseID.value) unfamiliarDirection.value = result
  } catch (reason) {
    if (version === selectionVersion) error.value = reason instanceof Error ? reason.message : '陌生知识读取失败'
  } finally {
    if (version === selectionVersion) unfamiliarLoading.value = false
  }
}

async function open(direction: ExplorationDirection) {
  await runOnce(`direction:${direction.id}`, async () => {
    try {
      await updateDirection(direction.id, 'open', direction.context_course_id)
      router.push(`/courses/${direction.target_course_id}/learn?lesson_id=${direction.target_lesson_id}`)
    } catch (reason) { error.value = reason instanceof Error ? reason.message : '探索方向打开失败' }
  })
}

async function makeQuestion(direction: ExplorationDirection) {
  await runOnce(`direction:${direction.id}`, async () => {
    try { await addExplorationQuestion(direction.context_course_id, direction.id); direction.status = 'saved'; await loadQuestions() } catch (reason) { error.value = reason instanceof Error ? reason.message : '探索问题保存失败' }
  })
}

async function undoQuestion(direction: ExplorationDirection) {
  await runOnce(`direction:${direction.id}`, async () => {
    try { direction.status = (await undoExplorationQuestion(direction.context_course_id, direction.id)).status; await loadQuestions() } catch (reason) { error.value = reason instanceof Error ? reason.message : '撤销保存失败' }
  })
}

async function startQuestion(question: ExplorationQuestion) {
  await runOnce(`question:${question.id}`, async () => {
    try {
      await updateQuestion(question.id, 'start', courseID.value)
      router.push(`/courses/${question.target_course_id}/learn?lesson_id=${question.target_lesson_id}`)
    } catch (reason) { error.value = reason instanceof Error ? reason.message : '探索问题启动失败' }
  })
}

async function archiveQuestion(question: ExplorationQuestion) {
  await runOnce(`question:${question.id}`, async () => {
    try { await updateQuestion(question.id, 'archive', courseID.value); questions.value = questions.value.filter((item) => item.id !== question.id) } catch (reason) { error.value = reason instanceof Error ? reason.message : '问题归档失败' }
  })
}

async function changeQuestion(question: ExplorationQuestion, action: 'later' | 'resolve' | 'reopen') {
  await runOnce(`question:${question.id}`, async () => {
    try { Object.assign(question, await updateQuestion(question.id, action, courseID.value)) } catch (reason) { error.value = reason instanceof Error ? reason.message : '问题状态更新失败' }
  })
}

async function setPriority(question: ExplorationQuestion, value: unknown) {
  const priority = String(value) as 'low' | 'normal' | 'high'
  await runOnce(`question:${question.id}`, async () => {
    try { Object.assign(question, await updateQuestionPriority(question.id, priority, courseID.value)) } catch (reason) { error.value = reason instanceof Error ? reason.message : '优先级更新失败' }
  })
}

function directionTypeText(type: string) { return { adjacent: '邻近探索', cross_domain: '跨领域', unknown: '陌生知识' }[type] ?? type }
function questionTypeText(type: string) { return { deepen: '深化', connect: '连接', challenge: '挑战', unfamiliar: '陌生知识' }[type] ?? type }
function questionStatusText(status: string) { return { open: '待探索', exploring: '探索中', later: '稍后学习', resolved: '已解决', archived: '已归档' }[status] ?? status }

onMounted(async () => {
  try {
    courses.value = await listCourses()
    courseID.value = courses.value[0]?.id ?? 0
    await switchCourse()
  } catch (reason) { error.value = reason instanceof Error ? reason.message : '课程读取失败' }
})
onBeforeUnmount(() => pageController?.abort())
</script>

<style scoped>
.exploration-domain-selector {
  display: grid;
  width: 280px;
  max-width: calc(100vw - 40px);
  gap: 6px;
}

.exploration-domain-selector__label {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.exploration-domain-selector label,
.exploration-domain-selector [role='status'] {
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 600;
}

.exploration-course-select {
  width: 100%;
}

.exploration-course-select :deep(.el-select__selected-item) {
  overflow: hidden;
  color: var(--text-primary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.question-pool-more {
  display: flex;
  justify-content: center;
  padding-top: 16px;
}

</style>
