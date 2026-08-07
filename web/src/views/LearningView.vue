<template>
  <section class="learning-page">
    <div class="learning-toolbar">
      <el-button text @click="router.push('/')">← 返回首页</el-button>
      <el-button text @click="router.push(`/courses/${courseID}/archive`)">查看课程档案</el-button>
      <el-button text @click="router.push(`/courses/${courseID}/map`)">查看知识结构</el-button>
      <el-button text @click="router.push(`/courses/${courseID}/misconceptions`)">误区网络</el-button>
    </div>

    <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />

    <div v-loading="loading" class="learning-layout">
      <el-card v-if="current" shadow="never" class="question-card">
        <span class="eyebrow">{{ current.course.name }} · {{ current.unit.title }}</span>
        <h2>{{ current.lesson.title }}</h2>
        <div class="question-box">
          <span class="question-box__label">核心问题</span>
          <p>{{ current.lesson.core_question }}</p>
        </div>

        <el-form @submit.prevent="handleSubmit">
          <el-form-item label="你的回答">
            <el-input
              v-model="answer"
              type="textarea"
              :rows="7"
              maxlength="5000"
              show-word-limit
              placeholder="写下你的理解……"
              :disabled="submitting"
            />
          </el-form-item>
          <el-button type="primary" :loading="submitting" @click="handleSubmit">提交回答</el-button>
        </el-form>
      </el-card>

      <el-card v-if="current && lessonRelations" shadow="never" class="position-card">
        <template #header>知识位置</template>
        <div class="position-section">
          <h3>前置知识</h3>
          <p v-if="lessonRelations.prerequisites.length === 0" class="muted-text">暂无明确前置。</p>
          <ul v-else>
            <li v-for="item in lessonRelations.prerequisites" :key="item.id">{{ item.title }}</li>
          </ul>
        </div>
        <div class="position-section">
          <h3>深化内容</h3>
          <p v-if="lessonRelations.extensions.length === 0" class="muted-text">暂无。</p>
          <ul v-else>
            <li v-for="item in lessonRelations.extensions" :key="item.id">{{ item.title }}</li>
          </ul>
        </div>
        <div class="position-section">
          <h3>应用场景</h3>
          <p v-if="lessonRelations.applications.length === 0" class="muted-text">暂无。</p>
          <ul v-else>
            <li v-for="item in lessonRelations.applications" :key="item.id">{{ item.title }}</li>
          </ul>
        </div>
        <el-button @click="router.push(`/courses/${courseID}/map`)">查看知识结构</el-button>
      </el-card>

      <el-card v-if="current && cognitiveDetail" shadow="never" class="cognitive-card">
        <template #header>
          <div class="card-header-row">
            <span>我的认知状态</span>
            <div class="feedback-tags">
              <el-tag type="success">{{ levelText(cognitiveDetail.state.current_level) }}</el-tag>
              <el-tag effect="plain">{{ statusText(cognitiveDetail.state.status) }}</el-tag>
            </div>
          </div>
        </template>
        <p v-if="cognitiveDetail.state.understanding_summary" class="cognitive-summary">
          {{ cognitiveDetail.state.understanding_summary }}
        </p>
        <p v-else class="muted-text">尚无 Phase 5 认知证据，当前为未接触。</p>
        <div v-if="cognitiveDetail.evidence.length > 0" class="feedback-section">
          <h3>证据</h3>
          <ul>
            <li v-for="item in cognitiveDetail.evidence" :key="item.id">
              {{ levelText(item.cognitive_level) }} · {{ item.description }}
            </li>
          </ul>
        </div>
        <div v-if="cognitiveDetail.timeline.length > 0" class="feedback-section">
          <h3>最近变化</h3>
          <ul>
            <li v-for="event in cognitiveDetail.timeline" :key="event.id">
              {{ levelText(event.from_level) }} → {{ levelText(event.to_level) }} · {{ statusText(event.to_status) }}
            </li>
          </ul>
        </div>
      </el-card>

      <el-card v-if="current && cognitiveDetail" shadow="never" class="challenge-card">
        <template #header>
          <div class="card-header-row">
            <span>认知验证</span>
            <el-tag v-if="canTransfer" type="success" effect="plain">已达到理解，可验证迁移</el-tag>
            <el-tag v-else effect="plain">先完成本题理解</el-tag>
          </div>
        </template>
        <p class="muted-text">迁移挑战会把同一概念放进陌生场景，不能用原题正确来代替迁移证据。</p>
        <el-button type="primary" plain :disabled="!canTransfer || challengeLoading" :loading="challengeLoading" @click="startTransfer">生成迁移挑战</el-button>
        <div v-if="activeMisconceptions.length > 0" class="feedback-section">
          <h3>待验证误区</h3>
          <div v-for="item in activeMisconceptions" :key="item.id" class="misconception-item">
            <p><strong>{{ item.original_understanding }}</strong></p>
            <p class="muted-text">目标理解：{{ item.correct_understanding }}</p>
            <el-button size="small" plain :loading="challengeLoading" @click="startRecheck(item.id)">针对性重新测试</el-button>
          </div>
        </div>
        <div v-if="challenge" class="challenge-box">
          <el-tag size="small" effect="plain">{{ challenge.challenge_type === 'transfer' ? '迁移挑战' : '误区重新测试' }}</el-tag>
          <h3>{{ challenge.prompt }}</h3>
          <p>{{ challenge.scenario_context }}</p>
          <p v-if="challenge.why_this_is_transfer" class="muted-text">{{ challenge.why_this_is_transfer }}</p>
          <ul><li v-for="criterion in challenge.evaluation_criteria" :key="criterion">评价标准：{{ criterion }}</li></ul>
          <el-input v-model="challengeAnswer" type="textarea" :rows="5" maxlength="5000" show-word-limit :disabled="challengeSubmitting" placeholder="回答这个新挑战……" />
          <el-button type="primary" :loading="challengeSubmitting" @click="submitChallenge">提交验证</el-button>
        </div>
        <div v-if="challengeResult" class="feedback-section">
          <el-alert :title="challengeResult.passed ? '验证通过' : (lastChallengeType === 'transfer' ? '迁移尚未验证' : '误区尚未修正')" :type="challengeResult.passed ? 'success' : 'warning'" :closable="false" />
          <p>{{ challengeResult.feedback }}</p>
          <p>{{ challengeResult.explanation }}</p>
          <p v-if="challengeResult.misconception_validation"><strong>误区状态：</strong>{{ validationText(challengeResult.misconception_validation.status) }} · {{ challengeResult.misconception_validation.evidence }}</p>
          <ul v-if="challengeResult.cognitive_evidence.length > 0">
            <li v-for="evidence in challengeResult.cognitive_evidence" :key="`${evidence.evidence_type}-${evidence.description}`">
              {{ evidence.evidence_type }} · {{ evidence.description }}
            </li>
          </ul>
        </div>
      </el-card>

      <el-card v-if="feedback" shadow="never" class="feedback-card">
        <template #header>
          <div class="card-header-row">
            <div>
              <span>本次学习反馈</span>
              <small class="feedback-source">{{ feedbackSource }}</small>
            </div>
            <div class="feedback-tags">
              <el-tag :type="feedback.result === 'correct' || feedback.result === 'mostly_correct' ? 'success' : 'warning'">
                {{ resultText }}
              </el-tag>
              <el-tag v-if="feedback.provider" effect="plain">{{ feedback.provider }} · {{ feedback.model }}</el-tag>
            </div>
          </div>
        </template>
        <div class="feedback-metrics">
          <div>
            <span>掌握度</span>
            <strong>{{ Math.round(feedback.mastery_score * 100) }}%</strong>
          </div>
          <div>
            <span>复习建议</span>
            <strong>{{ feedback.needs_review ? '需要复习' : '暂不需要' }}</strong>
          </div>
        </div>
        <div class="feedback-section">
          <h3>回答正确的部分</h3>
          <el-empty v-if="feedback.correct_parts.length === 0" description="暂无" :image-size="40" />
          <ul v-else>
            <li v-for="part in feedback.correct_parts" :key="part">{{ part }}</li>
          </ul>
        </div>
        <div class="feedback-section">
          <h3>仍然缺少的部分</h3>
          <ul v-if="feedback.missing_parts.length > 0">
            <li v-for="part in feedback.missing_parts" :key="part">{{ part }}</li>
          </ul>
          <p v-else class="muted-text">本次回答未发现明显缺失。</p>
        </div>
        <div v-if="feedback.misconceptions.length > 0" class="feedback-section">
          <h3>明确误区</h3>
          <div v-for="misconception in feedback.misconceptions" :key="misconception.original_understanding" class="misconception-item">
            <p><strong>原理解：</strong>{{ misconception.original_understanding }}</p>
            <p><strong>正确理解：</strong>{{ misconception.correct_understanding }}</p>
            <p><strong>边界说明：</strong>{{ misconception.boundary_notes }}</p>
          </div>
        </div>
        <div v-if="feedback.explanation" class="feedback-section">
          <h3>解释</h3>
          <p>{{ feedback.explanation }}</p>
        </div>
        <div v-if="feedback.boundary_conditions.length > 0" class="feedback-section">
          <h3>边界 / 反例</h3>
          <ul>
            <li v-for="condition in feedback.boundary_conditions" :key="condition">{{ condition }}</li>
          </ul>
        </div>
        <div v-if="feedback.mastery_evidence.length > 0" class="feedback-section">
          <h3>本次掌握证据</h3>
          <ul>
            <li v-for="evidence in feedback.mastery_evidence" :key="evidence">{{ evidence }}</li>
          </ul>
        </div>
        <div class="feedback-section">
          <h3>完整反馈</h3>
          <p>{{ feedback.feedback }}</p>
        </div>
      </el-card>
    </div>

    <el-card shadow="never" class="history-card">
      <template #header>
        <div class="card-header-row">
          <span>最近学习记录</span>
          <el-button text :loading="historyLoading" @click="loadHistory">刷新</el-button>
        </div>
      </template>
      <el-empty v-if="!historyLoading && history.length === 0" description="还没有回答记录" />
      <div v-loading="historyLoading" class="history-list">
        <article v-for="turn in history" :key="turn.id" class="history-item">
          <div class="history-item__meta">
            <el-tag size="small" :type="turn.result === 'mostly_correct' ? 'success' : 'warning'">
              {{ turn.result === 'mostly_correct' ? '基本正确' : '需要补充' }}
            </el-tag>
            <time>{{ formatDate(turn.created_at) }}</time>
          </div>
          <p class="history-item__question">{{ turn.question }}</p>
          <p class="history-item__answer">{{ turn.user_answer }}</p>
          <p class="history-item__feedback">{{ turn.feedback }}</p>
          <p v-if="turn.explanation" class="history-item__explanation">{{ turn.explanation }}</p>
          <div v-if="turn.misconceptions.length > 0" class="history-item__details">
            <strong>误区：</strong>
            <span v-for="misconception in turn.misconceptions" :key="misconception.original_understanding">
              {{ misconception.original_understanding }} → {{ misconception.correct_understanding }}
            </span>
          </div>
          <div class="history-item__details">
            <span>{{ sourceText(turn.evaluation_source) }}</span>
            <span v-if="turn.provider">{{ turn.provider }} · {{ turn.model }}</span>
            <span>掌握度 {{ Math.round(turn.mastery_score * 100) }}%</span>
          </div>
        </article>
      </div>
    </el-card>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getCurrentLesson, listLearningTurns, submitAnswer } from '@/api/learning'
import { getLessonRelations } from '@/api/knowledgeGraph'
import { getLessonCognitiveState } from '@/api/cognitive'
import { answerChallenge, createChallenge } from '@/api/challenges'
import { listLessonMisconceptions } from '@/api/misconceptions'
import type { AnswerResult, CurrentLessonData, LearningTurn } from '@/api/learning'
import type { AssessmentChallenge, ChallengeAnswerResult } from '@/types/challenge'
import type { MisconceptionView } from '@/types/misconception'
import type { LessonRelations } from '@/types/knowledgeGraph'
import type { CognitiveLevel, CognitiveStateDetail, CognitiveStatus } from '@/types/cognitive'

const route = useRoute()
const router = useRouter()
const courseID = Number(String(route.params.id))
const current = ref<CurrentLessonData | null>(null)
const history = ref<LearningTurn[]>([])
const feedback = ref<AnswerResult | null>(null)
const lessonRelations = ref<LessonRelations | null>(null)
const cognitiveDetail = ref<CognitiveStateDetail | null>(null)
const answer = ref('')
const loading = ref(false)
const submitting = ref(false)
const historyLoading = ref(false)
const error = ref('')
const misconceptions = ref<MisconceptionView[]>([])
const challenge = ref<AssessmentChallenge | null>(null)
const challengeResult = ref<ChallengeAnswerResult | null>(null)
const challengeAnswer = ref('')
const challengeLoading = ref(false)
const challengeSubmitting = ref(false)
const lastChallengeType = ref<'transfer' | 'misconception_recheck'>('transfer')

const canTransfer = computed(() => cognitiveRank(cognitiveDetail.value?.state.current_level ?? 'unseen') >= cognitiveRank('understand'))
const activeMisconceptions = computed(() => misconceptions.value.filter((item) => item.status === 'active'))

const resultText = computed(() => {
  switch (feedback.value?.result) {
    case 'correct': return '正确'
    case 'mostly_correct': return '基本正确'
    case 'partially_correct': return '部分正确'
    case 'incorrect': return '不正确'
    default: return '需要补充'
  }
})
const feedbackSource = computed(() => sourceText(feedback.value?.evaluation_source ?? 'mock'))
async function loadCurrent() {
  current.value = await getCurrentLesson(courseID)
  try {
    lessonRelations.value = await getLessonRelations(courseID, current.value.lesson.id)
  } catch {
    // The static position is supplemental and must not block the Phase 3 loop.
    lessonRelations.value = null
  }
  try {
    cognitiveDetail.value = await getLessonCognitiveState(courseID, current.value.lesson.id)
  } catch {
    cognitiveDetail.value = null
  }
  try {
    misconceptions.value = await listLessonMisconceptions(courseID, current.value.lesson.id)
  } catch {
    misconceptions.value = []
  }
}

async function loadHistory() {
  historyLoading.value = true
  try {
    history.value = await listLearningTurns(courseID)
  } catch (reason) {
    error.value = reason instanceof Error ? reason.message : '学习记录读取失败'
  } finally {
    historyLoading.value = false
  }
}

async function loadPage() {
  loading.value = true
  error.value = ''
  try {
    await Promise.all([loadCurrent(), loadHistory()])
  } catch (reason) {
    error.value = reason instanceof Error ? reason.message : '课程读取失败'
  } finally {
    loading.value = false
  }
}

async function handleSubmit() {
  const trimmedAnswer = answer.value.trim()
  if (!trimmedAnswer) {
    error.value = '请先输入回答'
    return
  }
  if (!current.value) {
    error.value = '当前知识点尚未加载'
    return
  }

  submitting.value = true
  error.value = ''
  try {
    feedback.value = await submitAnswer(courseID, current.value.lesson.id, trimmedAnswer)
    answer.value = ''
    await Promise.all([loadHistory(), loadCognitiveState(), loadMisconceptions()])
  } catch (reason) {
    error.value = reason instanceof Error ? reason.message : '回答提交失败'
  } finally {
    submitting.value = false
  }
}

async function loadCognitiveState() {
  if (!current.value) return
  try {
    cognitiveDetail.value = await getLessonCognitiveState(courseID, current.value.lesson.id)
  } catch (reason) {
    error.value = reason instanceof Error ? reason.message : '认知状态读取失败'
  }
}

async function loadMisconceptions() {
  if (!current.value) return
  try {
    misconceptions.value = await listLessonMisconceptions(courseID, current.value.lesson.id)
  } catch (reason) {
    error.value = reason instanceof Error ? reason.message : '误区读取失败'
  }
}

async function startTransfer() {
  if (!current.value || !canTransfer.value) return
  challengeLoading.value = true
  error.value = ''
  challengeResult.value = null
  try {
    challenge.value = await createChallenge(courseID, current.value.lesson.id, 'transfer')
    lastChallengeType.value = 'transfer'
    challengeAnswer.value = ''
  } catch (reason) {
    error.value = reason instanceof Error ? reason.message : '迁移挑战生成失败'
  } finally {
    challengeLoading.value = false
  }
}

async function startRecheck(misconceptionID: number) {
  if (!current.value) return
  challengeLoading.value = true
  error.value = ''
  challengeResult.value = null
  try {
    challenge.value = await createChallenge(courseID, current.value.lesson.id, 'misconception_recheck', misconceptionID)
    lastChallengeType.value = 'misconception_recheck'
    challengeAnswer.value = ''
  } catch (reason) {
    error.value = reason instanceof Error ? reason.message : '误区测试生成失败'
  } finally {
    challengeLoading.value = false
  }
}

async function submitChallenge() {
  if (!challenge.value || !challengeAnswer.value.trim()) {
    error.value = '请先输入挑战回答'
    return
  }
  challengeSubmitting.value = true
  error.value = ''
  try {
    challengeResult.value = await answerChallenge(courseID, challenge.value.id, challengeAnswer.value.trim())
    challenge.value = null
    challengeAnswer.value = ''
    await Promise.all([loadCognitiveState(), loadMisconceptions(), loadHistory()])
  } catch (reason) {
    error.value = reason instanceof Error ? reason.message : '挑战提交失败'
  } finally {
    challengeSubmitting.value = false
  }
}

function cognitiveRank(level: string): number {
  return { unseen: 0, exposed: 1, recognize: 2, understand: 3, apply: 4, transfer: 5 }[level] ?? 0
}

function validationText(status: string): string {
  return { corrected: '已修正', persists: '仍然存在', unclear: '尚不明确' }[status] ?? status
}

function sourceText(source: string): string {
  return { ai: 'AI 结构化评价', local: '规则判定', mock: '本地 Mock 评价' }[source] ?? source
}

function levelText(level: CognitiveLevel | string): string {
  return {
    unseen: '未接触', exposed: '已接触', recognize: '识别', understand: '理解', apply: '应用', transfer: '迁移',
  }[level] ?? '未知'
}

function statusText(status: CognitiveStatus | string): string {
  return { unknown: '未知', developing: '发展中', stable: '稳定', needs_review: '待复习' }[status] ?? '未知'
}

function formatDate(value: string): string {
  return new Date(value).toLocaleString('zh-CN', { dateStyle: 'short', timeStyle: 'short' })
}

onMounted(loadPage)
</script>
