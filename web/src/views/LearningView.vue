<template>
  <section class="learning-page editorial-learning-page">
    <header class="learning-context-header">
      <div class="learning-context-header__copy">
        <nav class="editorial-breadcrumb"><button type="button" @click="router.push('/learn')">学习</button><span>›</span><span>{{ current?.course.name || '正在学习' }}</span><span v-if="current">›</span><strong v-if="current">{{ current.unit.title }}</strong></nav>
        <h1 v-if="current">{{ current.lesson.title }} <small v-if="conversation?.total" class="muted-text">{{ conversation.position }} / {{ conversation.total }}</small></h1>
      </div>
      <div class="learning-context-header__actions"><el-button text @click="router.push(`/courses/${courseID}/map`)">知识结构</el-button><el-button text @click="router.push(`/courses/${courseID}/misconceptions`)">误区网络</el-button></div>
    </header>

    <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />
    <AIRequestError v-if="requestError" :message="requestError.message" :retryable="requestError.retryable" :retry="retryAction" />
    <el-alert v-if="isExploration" title="这是探索支线，不会改变课程主线的当前学习位置。" type="info" show-icon :closable="false" />

    <div v-loading="loading" class="learning-layout editorial-layout">
      <main class="learning-main editorial-flow">
        <section v-if="current" class="lesson-notes">
          <button type="button" class="notes-toggle" :aria-expanded="!notesCollapsed" @click="notesCollapsed = !notesCollapsed">{{ notesCollapsed ? '展开课件' : '收起课件' }}</button>
          <div v-if="!notesCollapsed">
            <p v-if="current.lesson.content" class="lesson-copy">{{ current.lesson.content }}</p>
            <template v-else><p class="muted-text">本课保留原有学习要点。</p><p class="lesson-copy">{{ current.lesson.expected_understanding || current.unit.objective }}</p></template>
            <p class="guiding-question"><strong>想一想</strong> {{ current.lesson.core_question }}</p>
          </div>
        </section>
        <section ref="conversationRef" :class="{ 'is-empty': !conversation?.turns.length }" class="conversation-stream" aria-label="当前课程对话" aria-live="polite">
          <p v-if="!conversation?.turns.length" class="muted-text">可以回答上面的思考问题，也可以直接提问，或告诉我哪里不理解。</p>
          <article v-for="turn in conversation?.turns" :key="turn.id" class="conversation-turn">
            <p class="conversation-user"><strong>你</strong><span>{{ turn.user_answer }}</span></p>
            <p v-if="turn.feedback" class="conversation-assistant"><strong>{{ turn.evaluation_source === 'mock' ? 'AI · Mock' : 'AI' }}</strong><span>{{ turn.feedback }}</span></p>
            <p v-else class="muted-text">{{ submitting ? '正在回复…' : '这条消息已保存，回复尚未完成。' }}<el-button v-if="!submitting" text @click="retrySavedTurn(turn)">重试回复</el-button></p>
          </article>
          <p v-if="submitting" role="status" class="muted-text">正在思考，请稍候…</p>
        </section>
        <details class="learning-details"><summary>可选挑战、掌握证据与详细评价</summary>
          <section v-if="current" class="editorial-section">
            <p class="muted-text">需要完整评价时，可主动提交解释；这不会阻止继续学习。</p>
            <el-input v-model="evaluationAnswer" type="textarea" :rows="3" maxlength="5000" :disabled="submitting || nextActionLoading" placeholder="需要详细评价的解释" />
            <el-button :loading="submitting" :disabled="!evaluationAnswer.trim() || submitting" @click="handleEvaluate">查看详细评价</el-button>
            <EditorialFeedback v-if="feedback" :feedback="feedback" :result-text="resultText" :feedback-source="feedbackSource" />
          </section>
        <section v-if="feedbackMisconceptions.length || activeMisconceptions.length" class="editorial-section misconception-followup"><div class="section-heading-inline"><div><span class="eyebrow">后续澄清</span><h2>需要继续澄清的误区</h2></div><StatusTag status="needs_review" label="优先处理" /></div><div v-for="misconception in feedbackMisconceptions" :key="`evaluation-${misconception.original_understanding}`" class="misconception-item"><p><strong>原理解：</strong>{{ misconception.original_understanding }}</p><p><strong>目标理解：</strong>{{ misconception.correct_understanding }}</p><p class="muted-text">边界：{{ misconception.boundary_notes }}</p></div><div v-for="item in activeMisconceptions" :key="item.id" class="misconception-item misconception-item--active"><p><strong>{{ item.original_understanding }}</strong></p><p class="muted-text">目标理解：{{ item.correct_understanding }}</p><el-button size="small" plain :loading="challengeLoading" @click="startRecheck(item.id)">针对性重新测试</el-button></div></section>

        <section v-if="current && cognitiveDetail" class="editorial-section transfer-section"><div class="section-heading-inline"><div><span class="eyebrow">迁移挑战</span><h2>把理解带到新场景</h2></div><StatusTag :status="canTransfer ? 'stable' : 'unseen'" :label="canTransfer ? '可以验证' : '尚未解锁'" /></div><p class="muted-text">原问题检查你能否解释当前知识；迁移挑战会把同一概念放进陌生场景，必须重新判断，原题答对不能代替迁移证据。</p><p v-if="!canTransfer" class="challenge-unlock">解锁条件：{{ transferUnlockText }}</p><p class="muted-text">迁移成功会新增独立迁移证据，并使累计掌握度增加 10 个百分点（最高 100%）；失败不会扣分。</p><el-button type="primary" plain :disabled="!canTransfer || challengeLoading" :loading="challengeLoading" @click="startTransfer">生成迁移挑战</el-button><div v-if="challenge" class="challenge-box"><StatusTag :status="challenge.challenge_type === 'transfer' ? 'transfer' : 'needs_review'" :label="challenge.challenge_type === 'transfer' ? '迁移挑战（独立记录）' : '误区重新测试（独立记录）'" /><h3>{{ challenge.prompt }}</h3><p>{{ challenge.scenario_context }}</p><p v-if="challenge.why_this_is_transfer" class="muted-text">{{ challenge.why_this_is_transfer }}</p><ul><li v-for="criterion in challenge.evaluation_criteria" :key="criterion">评价标准：{{ criterion }}</li></ul><el-input v-model="challengeAnswer" type="textarea" :rows="5" maxlength="5000" show-word-limit :disabled="challengeSubmitting" placeholder="回答这个新挑战……" /><el-button type="primary" :loading="challengeSubmitting" :disabled="challengeSubmitting || !challengeAnswer.trim()" @click="submitChallenge">提交验证</el-button></div><div v-if="challengeResult" class="challenge-result"><el-alert :title="challengeResult.passed ? '验证通过' : (lastChallengeType === 'transfer' ? '迁移尚未验证' : '误区尚未修正')" :type="challengeResult.passed ? 'success' : 'warning'" :closable="false" /><p>{{ challengeResult.feedback }}</p><p>{{ challengeResult.explanation }}</p><p><strong>对掌握度的影响：</strong>{{ challengeResult.mastery_impact }}</p><p class="muted-text">累计掌握度 {{ Math.round(challengeResult.mastery_score_before * 100) }}% → {{ Math.round(challengeResult.mastery_score_after * 100) }}%</p></div></section>

        <section v-if="cognitiveDetail" class="editorial-section cognitive-explanation">
          <div class="section-heading-inline"><div><span class="eyebrow">掌握证据</span><h2>当前学习状态为什么是“{{ learningStageText(cognitiveDetail.state.current_level) }}”</h2></div><StatusTag :status="cognitiveDetail.state.status" :label="learningStageText(cognitiveDetail.state.current_level)" /></div>
          <p>{{ learningStageDescription(cognitiveDetail.state.current_level, cognitiveDetail.state.status) }}</p>
          <p class="muted-text">系统保留每次证据；新评价只追加证据和状态事件，不会删除或覆盖历史证据。</p>
          <div class="evidence-list"><article v-for="item in cognitiveDetail.evidence" :key="item.id"><div><StatusTag :status="item.polarity === 'support' ? 'stable' : 'needs_review'" :label="item.polarity === 'support' ? '支持证据' : '矛盾证据'" /><time>{{ formatDate(item.created_at) }}</time></div><p>{{ item.description }}</p><small>{{ evidenceTypeText(item.evidence_type) }} · 支持层级 {{ levelText(item.cognitive_level) }}</small></article><p v-if="!cognitiveDetail.evidence.length" class="muted-text">尚无掌握证据。提交回答后，只有通过结构校验的评价才会生成证据。</p></div>
          <details v-if="cognitiveDetail.timeline.length"><summary>查看状态判断时间线（{{ cognitiveDetail.timeline.length }} 条）</summary><ol><li v-for="event in cognitiveDetail.timeline" :key="event.id"><time>{{ formatDate(event.created_at) }}</time><span>{{ levelText(event.from_level) }} → {{ levelText(event.to_level) }}；{{ statusText(event.from_status) }} → {{ statusText(event.to_status) }}</span><p>{{ event.reason }}</p></li></ol></details>
        </section>
        </details>
      </main>

      <details class="supplemental-rail"><summary>知识来源与课程信息</summary><KnowledgeRail :current="current" :cognitive="cognitiveDetail" :relations="lessonRelations"><template #default><div class="knowledge-rail__section knowledge-rail__actions"><el-button type="primary" plain @click="router.push(`/courses/${courseID}/map`)">查看知识结构 →</el-button><el-button text @click="router.push(`/courses/${courseID}/archive`)">课程档案</el-button></div></template></KnowledgeRail></details>
    </div>

    <details class="legacy-history"><summary>历史回答与评价记录</summary><section class="learning-history editorial-section"><div class="section-heading-inline"><div><span class="eyebrow">回答版本</span><h2>学习记录与回答版本</h2></div><div class="history-actions"><el-button text :loading="historyLoading" @click="loadHistory">刷新学习记录</el-button><el-button text @click="historyExpanded = !historyExpanded">{{ historyExpanded ? '收起' : `展开全部${history.length > 1 ? `（${history.length} 条）` : ''}` }}</el-button></div></div><el-empty v-if="!historyLoading && history.length === 0" description="还没有回答记录" /><div v-loading="historyLoading" class="history-list"><article v-for="turn in displayedHistory" :key="turn.id" class="history-item"><div class="history-item__meta"><StatusTag :status="turn.result === 'mostly_correct' || turn.result === 'correct' ? 'stable' : 'needs_review'" :label="historyKindText(turn)" /><strong v-if="turn.turn_kind === 'lesson_answer'">回答版本 {{ answerVersion(turn) }}</strong><time>{{ formatDate(turn.created_at) }}</time></div><p class="history-item__question">{{ turn.question }}</p><template v-if="historyExpanded"><p class="history-item__answer">{{ turn.user_answer }}</p><p v-if="turn.user_understanding_summary"><strong>本次理解：</strong>{{ turn.user_understanding_summary }}</p><p class="history-item__feedback"><strong>评价：</strong>{{ turn.feedback }}</p><p v-if="turn.explanation" class="history-item__explanation">{{ turn.explanation }}</p><p v-if="turn.correct_parts.length"><strong>已理解：</strong>{{ turn.correct_parts.join('；') }}</p><p v-if="turn.missing_parts.length"><strong>关键缺口：</strong>{{ turn.missing_parts.join('；') }}</p><p v-if="turn.confidence"><strong>评价可信度：</strong>{{ Math.round(turn.confidence * 100) }}%<span v-if="turn.uncertainty">；{{ turn.uncertainty }}</span></p><p v-if="turn.recommended_next_action"><strong>当时建议：</strong>{{ turn.recommended_next_action }}</p><p v-if="turn.state_change"><strong>状态变化：</strong>{{ levelText(turn.state_change.from_level) }} → {{ levelText(turn.state_change.to_level) }}；{{ statusText(turn.state_change.from_status) }} → {{ statusText(turn.state_change.to_status) }}</p><ul v-if="turn.cognitive_evidence.length"><li v-for="evidence in turn.cognitive_evidence" :key="`${turn.id}-${evidence.evidence_index}-${evidence.description}`">{{ evidence.polarity === 'support' ? '支持' : '矛盾' }}：{{ evidence.description }}</li></ul></template><p v-else class="history-item__summary">{{ turn.feedback }}</p><div class="history-item__details"><span>本次评价 {{ Math.round(turn.mastery_score * 100) }}%</span><span>累计掌握度 {{ Math.round(turn.mastery_score_before * 100) }}% → {{ Math.round(turn.mastery_score_after * 100) }}%</span><span>证据 {{ turn.cognitive_evidence.length }} 条</span><span v-if="turn.provider">{{ turn.provider }} · {{ turn.model }}</span></div></article></div></section></details>
    <footer v-if="current" class="learning-composer">
      <el-form @submit.prevent="handleSubmit">
        <label class="sr-only" for="lesson-answer">与 AI 交流</label>
        <el-input ref="answerInput" id="lesson-answer" v-model="answer" type="textarea" :rows="2" maxlength="5000" placeholder="提问、说说你的理解，或继续探索…" :disabled="submitting || nextActionLoading" @keydown="handleAnswerShortcut" />
        <div class="composer-actions">
          <span class="muted-text" role="status">{{ lessonStateText }}<small v-if="draftStatus"> · {{ draftStatus }}</small></span>
          <el-button native-type="submit" :loading="submitting" :disabled="!answer.trim() || submitting || nextActionLoading">发送</el-button>
          <el-button v-if="!isExploration && (!terminalLesson || conversation?.has_next)" type="primary" :loading="nextActionLoading" :disabled="submitting || loading" @click="advanceLesson">{{ terminalLesson ? '继续下一课' : conversation?.completion_suggested ? '完成并进入下一课' : '跳过并进入下一课' }}</el-button>
          <el-button v-else-if="!isExploration" type="primary" @click="router.push(`/courses/${courseID}/map`)">返回知识结构</el-button>
        </div>
      </el-form>
    </footer>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router'
import { retryConversation, getConversation, sendConversation, advanceConversation, type ConversationData, type ConversationTurn, getCurrentLesson, getLessonForLearning, listLearningTurns, submitAnswer, submitLessonAnswer } from '@/api/learning'
import { getLessonRelations } from '@/api/knowledgeGraph'
import { getLessonCognitiveState } from '@/api/cognitive'
import { answerChallenge, createChallenge } from '@/api/challenges'
import { listLessonMisconceptions } from '@/api/misconceptions'
import { invalidateExplorationRadar } from '@/api/exploration'
import { APIRequestError, isRequestAborted } from '@/api/http'
import AIRequestError from '@/components/AIRequestError.vue'
import EditorialFeedback from '@/components/EditorialFeedback.vue'
import KnowledgeRail from '@/components/KnowledgeRail.vue'
import StatusTag from '@/components/StatusTag.vue'
import type { AnswerResult, CurrentLessonData, LearningTurn } from '@/api/learning'
import type { AssessmentChallenge, ChallengeAnswerResult } from '@/types/challenge'
import type { MisconceptionView } from '@/types/misconception'
import type { LessonRelations } from '@/types/knowledgeGraph'
import type { CognitiveLevel, CognitiveStateDetail, CognitiveStatus } from '@/types/cognitive'

const route = useRoute()
const router = useRouter()
const courseID = Number(String(route.params.id))
const branchLessonID = Number(String(route.query.lesson_id ?? 0))
const isExploration = branchLessonID > 0
const current = ref<CurrentLessonData | null>(null)
const history = ref<LearningTurn[]>([])
const feedback = ref<AnswerResult | null>(null)
const lessonRelations = ref<LessonRelations | null>(null)
const cognitiveDetail = ref<CognitiveStateDetail | null>(null)
const answer = ref('')
const evaluationAnswer = ref('')
const evaluationKey = ref('')
const conversation = ref<ConversationData | null>(null)
const conversationRef = ref<HTMLElement | null>(null)
const notesCollapsed = ref(false)
const terminalLesson = computed(() => ['completed', 'skipped'].includes(current.value?.lesson.status ?? ''))
const lessonStateText = computed(() => current.value?.lesson.status === 'completed' ? '已完成 · 掌握度独立记录' : current.value?.lesson.status === 'skipped' ? '已跳过 · 理解未验证' : conversation.value?.completion_suggested ? '本课目标基本达成，可确认完成' : '学习中 · 理解尚未验证')
const answerInput = ref<{ focus: () => void } | null>(null)
const draftStatus = ref('')
const answerSubmissionKey = ref('')
const answerSubmissionMessage = ref('')
const loading = ref(false)
const submitting = ref(false)
const historyLoading = ref(false)
const error = ref('')
const requestError = ref<APIRequestError | null>(null)
const retryAction = ref<(() => void) | undefined>()
const misconceptions = ref<MisconceptionView[]>([])
const challenge = ref<AssessmentChallenge | null>(null)
const challengeResult = ref<ChallengeAnswerResult | null>(null)
const challengeAnswer = ref('')
const challengeLoading = ref(false)
const challengeSubmitting = ref(false)
const challengeGenerationKey = ref('')
const challengeAnswerKey = ref('')
const lastChallengeType = ref<'transfer' | 'misconception_recheck'>('transfer')
const historyExpanded = ref(false)
const nextActionLoading = ref(false)
let draftTimer: number | undefined
let readController: AbortController | null = null

const canTransfer = computed(() => cognitiveRank(cognitiveDetail.value?.state.current_level ?? 'unseen') >= cognitiveRank('understand'))
const transferUnlockText = computed(() => `核心问题达到“初步理解”。当前状态为“${learningStageText(cognitiveDetail.value?.state.current_level ?? 'unseen')}”。`)
const draftKey = computed(() => current.value ? `learnos:answer-draft:${courseID}:${current.value.lesson.id}` : '')
const hasUnsubmittedContent = computed(() => Boolean(answer.value.trim() || evaluationAnswer.value.trim() || challengeAnswer.value.trim()))
const activeMisconceptions = computed(() => misconceptions.value.filter((item) => item.status === 'active'))
const feedbackMisconceptions = computed(() => feedback.value?.misconceptions ?? [])
const displayedHistory = computed(() => historyExpanded.value ? history.value : history.value.slice(0, 1))

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
  const signal = readController?.signal
  current.value = isExploration ? await getLessonForLearning(courseID, branchLessonID, signal) : await getCurrentLesson(courseID, signal)
  const lessonID = current.value.lesson.id
  conversation.value = await getConversation(courseID, lessonID, signal)
  const [relationResult, cognitiveResult, misconceptionResult] = await Promise.allSettled([
    getLessonRelations(courseID, lessonID, signal),
    getLessonCognitiveState(courseID, lessonID, signal),
    listLessonMisconceptions(courseID, lessonID, signal),
  ])
  if (signal?.aborted) throw new DOMException('aborted', 'AbortError')
  // These panels are supplemental and must not block the core answer loop.
  lessonRelations.value = relationResult.status === 'fulfilled' ? relationResult.value : null
  cognitiveDetail.value = cognitiveResult.status === 'fulfilled' ? cognitiveResult.value : null
  misconceptions.value = misconceptionResult.status === 'fulfilled' ? misconceptionResult.value : []
}

async function loadHistory() {
  historyLoading.value = true
  try {
    history.value = await listLearningTurns(courseID, 10, readController?.signal)
  } catch (reason) {
    if (isRequestAborted(reason)) return
    error.value = reason instanceof Error ? reason.message : '学习记录读取失败'
  } finally {
    historyLoading.value = false
  }
}

async function loadPage() {
  readController?.abort()
  const controller = new AbortController()
  readController = controller
  loading.value = true
  error.value = ''
  try {
    await Promise.all([loadCurrent(), loadHistory()])
  } catch (reason) {
    if (isRequestAborted(reason)) return
    error.value = reason instanceof Error ? reason.message : '课程读取失败'
  } finally {
    if (readController === controller) loading.value = false
  }
}

async function scrollConversation() {
  await nextTick()
  conversationRef.value?.scrollTo({ top: conversationRef.value.scrollHeight, behavior: 'smooth' })
}
async function handleSubmit() {
  if (submitting.value || nextActionLoading.value || !current.value || !answer.value.trim()) return
  submitting.value = true
  requestError.value = null
  error.value = ''
  const lessonID = current.value.lesson.id
  const message = answer.value.trim()
  if (!answerSubmissionKey.value || answerSubmissionMessage.value !== message) answerSubmissionKey.value = newIdempotencyKey('chat')
  answerSubmissionMessage.value = message
  saveAnswerDraft()
  try {
    conversation.value = await sendConversation(courseID, lessonID, message, answerSubmissionKey.value)
    answer.value = ''
    answerSubmissionKey.value = ''
    clearAnswerDraft()
    notesCollapsed.value = true
    void loadCognitiveState()
    await scrollConversation()
  } catch (reason) {
    if (reason instanceof APIRequestError) { requestError.value = new APIRequestError({ code: reason.code, message: reason.code === 'AI_NOT_CONFIGURED' ? 'AI 尚未配置，请先在设置中配置。你的输入已保留。' : 'AI 回复未完成，你的输入已保留，可重试。', retryable: reason.retryable }, reason.status); retryAction.value = handleSubmit }
    else error.value = reason instanceof Error ? reason.message : '回复失败，可重试'
    try { conversation.value = await getConversation(courseID, lessonID) } catch { /* retain loaded history */ }
  } finally { submitting.value = false }
}
async function retrySavedTurn(turn: ConversationTurn) {
  if (submitting.value || !current.value) return
  // The server exposes a safe per-turn retry endpoint; the original key stays private.
  submitting.value = true
  error.value = ''
  requestError.value = null
  try {
    conversation.value = await retryConversation(courseID, current.value.lesson.id, turn.id)
    if (answer.value.trim() === turn.user_answer) { answer.value = ''; answerSubmissionKey.value = ''; clearAnswerDraft() }
    await scrollConversation()
  } catch (reason) { error.value = reason instanceof Error ? reason.message : '重试失败' }
  finally { submitting.value = false }
}
async function advanceLesson() {
  if (!current.value || nextActionLoading.value || submitting.value) return
  nextActionLoading.value = true
  error.value = ''
  saveAnswerDraft()
  try {
    await advanceConversation(courseID, current.value.lesson.id, terminalLesson.value ? 'continue' : conversation.value?.completion_suggested ? 'complete' : 'skip')
    feedback.value = null
    challenge.value = null
    challengeResult.value = null
    notesCollapsed.value = false
    await loadPage()
    invalidateExplorationRadar(courseID)
  } catch (reason) { error.value = reason instanceof Error ? reason.message : '推进失败，可重试' }
  finally { nextActionLoading.value = false }
}

async function handleEvaluate() {
  if (submitting.value) return
  const trimmedAnswer = evaluationAnswer.value.trim()
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
  requestError.value = null
  retryAction.value = undefined
  feedback.value = null
  try {
    if (!evaluationKey.value) evaluationKey.value = newIdempotencyKey('evaluation')
    feedback.value = isExploration
      ? await submitLessonAnswer(courseID, current.value.lesson.id, trimmedAnswer, evaluationKey.value)
      : await submitAnswer(courseID, current.value.lesson.id, trimmedAnswer, evaluationKey.value)
    invalidateExplorationRadar(courseID)
    evaluationKey.value = ''
    evaluationAnswer.value = ''
    await nextTick()
    await Promise.all([loadHistory(), loadCognitiveState(), loadMisconceptions()])
  } catch (reason) {
    if (reason instanceof APIRequestError) {
      requestError.value = reason
      retryAction.value = handleEvaluate
    } else {
      error.value = reason instanceof Error ? reason.message : '回答提交失败'
    }
  } finally {
    submitting.value = false
  }
}

async function loadCognitiveState() {
  if (!current.value) return
  try {
    cognitiveDetail.value = await getLessonCognitiveState(courseID, current.value.lesson.id, readController?.signal)
  } catch (reason) {
    if (isRequestAborted(reason)) return
    error.value = reason instanceof Error ? reason.message : '认知状态读取失败'
  }
}

async function loadMisconceptions() {
  if (!current.value) return
  try {
    misconceptions.value = await listLessonMisconceptions(courseID, current.value.lesson.id, readController?.signal)
  } catch (reason) {
    if (isRequestAborted(reason)) return
    error.value = reason instanceof Error ? reason.message : '误区读取失败'
  }
}

async function startTransfer() {
  if (challengeLoading.value || !current.value || !canTransfer.value) return
  challengeLoading.value = true
  error.value = ''
  requestError.value = null
  challengeResult.value = null
  try {
    if (!challengeGenerationKey.value) challengeGenerationKey.value = newIdempotencyKey('challenge')
    challenge.value = await createChallenge(courseID, current.value.lesson.id, 'transfer', challengeGenerationKey.value)
    challengeGenerationKey.value = ''
    challengeAnswerKey.value = ''
    lastChallengeType.value = 'transfer'
    challengeAnswer.value = ''
  } catch (reason) {
    if (reason instanceof APIRequestError) { requestError.value = reason; retryAction.value = startTransfer }
    else error.value = reason instanceof Error ? reason.message : '迁移挑战生成失败'
  } finally {
    challengeLoading.value = false
  }
}

async function startRecheck(misconceptionID: number) {
  if (challengeLoading.value || !current.value) return
  challengeLoading.value = true
  error.value = ''
  requestError.value = null
  challengeResult.value = null
  try {
    if (!challengeGenerationKey.value) challengeGenerationKey.value = newIdempotencyKey('challenge')
    challenge.value = await createChallenge(courseID, current.value.lesson.id, 'misconception_recheck', challengeGenerationKey.value, misconceptionID)
    challengeGenerationKey.value = ''
    challengeAnswerKey.value = ''
    lastChallengeType.value = 'misconception_recheck'
    challengeAnswer.value = ''
  } catch (reason) {
    if (reason instanceof APIRequestError) { requestError.value = reason; retryAction.value = () => startRecheck(misconceptionID) }
    else error.value = reason instanceof Error ? reason.message : '误区测试生成失败'
  } finally {
    challengeLoading.value = false
  }
}

async function submitChallenge() {
  if (challengeSubmitting.value) return
  if (!challenge.value || !challengeAnswer.value.trim()) {
    error.value = '请先输入挑战回答'
    return
  }
  challengeSubmitting.value = true
  error.value = ''
  requestError.value = null
  try {
    if (!challengeAnswerKey.value) challengeAnswerKey.value = newIdempotencyKey('challenge-answer')
    challengeResult.value = await answerChallenge(courseID, challenge.value.id, challengeAnswer.value.trim(), challengeAnswerKey.value)
    invalidateExplorationRadar(courseID)
    challengeAnswerKey.value = ''
    challenge.value = null
    challengeAnswer.value = ''
    await Promise.all([loadCognitiveState(), loadMisconceptions(), loadHistory()])
  } catch (reason) {
    if (reason instanceof APIRequestError) { requestError.value = reason; retryAction.value = submitChallenge }
    else error.value = reason instanceof Error ? reason.message : '挑战提交失败'
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

function learningStageText(level: CognitiveLevel | string): string {
  if (level === 'unseen') return '未接触'
  if (level === 'exposed' || level === 'recognize') return '学习中'
  if (level === 'understand') return '初步理解'
  return '掌握'
}

function learningStageDescription(level: CognitiveLevel | string, status: CognitiveStatus | string): string {
  if (status === 'needs_review') return '历史证据仍然保留，但最新回答出现缺口或矛盾，需要再次修正和验证。'
  if (level === 'unseen') return '尚未形成可验证的回答证据。'
  if (level === 'exposed') return '已经接触问题，但目前证据不足以证明能识别核心判断。'
  if (level === 'recognize') return '能够识别核心概念或判断，但还缺少完整解释。'
  if (level === 'understand') return '能够用自己的话解释核心机制或重要边界，接下来可以通过新场景验证迁移。'
  if (level === 'apply') return '已经能在具体案例中使用知识，但仍需跨场景证据确认迁移。'
  return '已经在陌生或跨场景任务中成功使用知识，形成了迁移证据。'
}

function evidenceTypeText(type: string): string {
  return { recognition: '概念识别', concept_explanation: '概念解释', boundary_awareness: '边界意识', application: '场景应用', transfer: '迁移验证', contradiction: '矛盾记录' }[type] ?? type
}

function historyKindText(turn: LearningTurn): string {
  if (turn.turn_kind === 'transfer_challenge') return turn.result === 'correct' || turn.result === 'mostly_correct' ? '迁移挑战通过' : '迁移挑战待完善'
  if (turn.turn_kind === 'misconception_recheck') return '误区重新测试'
  return turn.result === 'mostly_correct' ? '基本正确' : turn.result === 'correct' ? '正确' : '需要补充'
}

function answerVersion(turn: LearningTurn): number {
  const index = history.value.findIndex((item) => item.id === turn.id)
  return history.value.slice(index).filter((item) => item.turn_kind === 'lesson_answer' && item.lesson_id === turn.lesson_id).length
}

function focusAnswerEditor() {
  answerInput.value?.focus()
  document.querySelector('.lesson-editor')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

function handleAnswerShortcut(event: KeyboardEvent) {
  if (event.key !== 'Enter' || (!event.ctrlKey && !event.metaKey)) return
  event.preventDefault()
  void handleSubmit()
}

function newIdempotencyKey(prefix: string): string {
  const suffix = typeof crypto !== 'undefined' && 'randomUUID' in crypto ? crypto.randomUUID() : `${Date.now()}-${Math.random().toString(16).slice(2)}`
  return `${prefix}-${suffix}`
}

function restoreAnswerDraft() {
  if (!draftKey.value) return
  const raw = window.localStorage.getItem(draftKey.value)
  if (!raw) return
  try {
    const draft = JSON.parse(raw) as { answer?: string; updated_at?: string; key?: string }
    if (draft.answer?.trim()) {
      answer.value = draft.answer
      answerSubmissionKey.value = draft.key ?? ''
      answerSubmissionMessage.value = draft.answer.trim()
      draftStatus.value = `已恢复 ${draft.updated_at ? formatDate(draft.updated_at) : ''} 的本机草稿`
    }
  } catch {
    window.localStorage.removeItem(draftKey.value)
  }
}

function saveAnswerDraft() {
  if (!draftKey.value) return
  if (!answer.value.trim()) {
    clearAnswerDraft()
    return
  }
  const updatedAt = new Date().toISOString()
  window.localStorage.setItem(draftKey.value, JSON.stringify({ answer: answer.value, key: answer.value.trim() === answerSubmissionMessage.value ? answerSubmissionKey.value : '', updated_at: updatedAt }))
  draftStatus.value = `草稿已自动保存于 ${formatDate(updatedAt)}`
}

function clearAnswerDraft() {
  if (draftKey.value) window.localStorage.removeItem(draftKey.value)
  draftStatus.value = ''
}

function handleBeforeUnload(event: BeforeUnloadEvent) {
  if (!hasUnsubmittedContent.value || submitting.value || challengeSubmitting.value) return
  event.preventDefault()
  event.returnValue = ''
}

function formatDate(value: string): string {
  return new Date(value).toLocaleString('zh-CN', { dateStyle: 'short', timeStyle: 'short' })
}

watch(evaluationAnswer, () => { if (!submitting.value) evaluationKey.value = '' })
watch(answer, () => {
  window.clearTimeout(draftTimer)
  draftTimer = window.setTimeout(saveAnswerDraft, 300)
})

watch(() => current.value?.lesson.id, (lessonID, previousLessonID) => {
  if (!lessonID || lessonID === previousLessonID) return
  answer.value = ''
  draftStatus.value = ''
  answerSubmissionKey.value = ''
  restoreAnswerDraft()
})

onBeforeRouteLeave(() => {
  if (!hasUnsubmittedContent.value || submitting.value || challengeSubmitting.value) return true
  return window.confirm('尚未提交的回答已保存在本机草稿中，仍要离开当前学习页吗？')
})

onMounted(() => {
  window.addEventListener('beforeunload', handleBeforeUnload)
  void loadPage()
})

onBeforeUnmount(() => {
  readController?.abort()
  window.clearTimeout(draftTimer)
  saveAnswerDraft()
  window.removeEventListener('beforeunload', handleBeforeUnload)
})
</script>

<style scoped>
.answer-guidance {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 14px;
  margin-top: 10px;
  color: var(--text-secondary);
  font-size: 13px;
}

.answer-guidance span::before {
  content: '·';
  margin-right: 6px;
}

.revise-answer-button {
  margin-top: 14px;
}

.challenge-unlock {
  padding: 10px 12px;
  border-left: 3px solid var(--border-strong);
  background: var(--bg-subtle);
}

.evidence-list {
  display: grid;
  gap: 10px;
  margin-top: 14px;
}

.evidence-list article {
  padding: 12px;
  border: 1px solid var(--border-default);
  border-radius: var(--radius-control);
}

.evidence-list article > div {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.evidence-list time,
.evidence-list small,
.cognitive-explanation details time {
  color: var(--text-secondary);
  font-size: 12px;
}

.cognitive-explanation details {
  margin-top: 16px;
}

.cognitive-explanation details li {
  margin-top: 10px;
}

.cognitive-explanation details span {
  margin-left: 10px;
}
</style>

<style scoped>
.learning-page { position: relative; padding-bottom: 180px; }
.learning-page .editorial-layout { display: block; max-width: 980px; margin-inline: auto; }
.learning-context-header h1 { font-size: clamp(22px, 2vw, 30px); }
.learning-context-header h1 small { font-size: 14px; font-weight: 400; margin-left: 12px; }
.lesson-notes { padding: 20px 0; border-bottom: 1px solid var(--border-default); }
.notes-toggle { background: transparent; border: 0; color: var(--text-secondary); cursor: pointer; padding: 4px 0; font: inherit; }
.lesson-copy, .guiding-question { white-space: pre-wrap; line-height: 1.85; max-width: 78ch; }
.guiding-question strong { color: var(--text-secondary); font-size: 13px; margin-right: 8px; }
.conversation-stream { height: clamp(240px, 45dvh, 650px); overflow-y: auto; overscroll-behavior: contain; padding: 16px 8px 20px 0; }
.conversation-stream.is-empty { height: auto; min-height: 100px; }
.conversation-turn { margin-bottom: 24px; }
.conversation-user, .conversation-assistant { display: flex; gap: 14px; white-space: pre-wrap; line-height: 1.8; }
.conversation-turn strong { flex: 0 0 42px; font-size: 12px; color: var(--text-secondary); }
.conversation-user { background: var(--bg-subtle); padding: 12px; border-radius: var(--radius-control); }
.conversation-assistant { padding: 8px 12px; }
.learning-details, .supplemental-rail, .legacy-history { max-width: 980px; margin: 20px auto; color: var(--text-secondary); }
.learning-details summary, .supplemental-rail summary, .legacy-history summary { cursor: pointer; padding: 10px 0; }
.learning-composer { position: fixed; z-index: 20; bottom: 16px; left: 116px; right: 32px; padding: 14px 20px; background: var(--bg-surface); border: 1px solid var(--border-default); border-radius: var(--radius-control); box-shadow: var(--shadow-card); }
.learning-composer form { max-width: 940px; margin: auto; }
.composer-actions { display: flex; align-items: center; gap: 10px; margin-top: 10px; }
.composer-actions > span { flex: 1; font-size: 12px; }
.composer-actions small { font-size: inherit; }
@media (max-width: 820px) {
 .learning-composer { left: 12px; right: 12px; bottom: max(8px, env(safe-area-inset-bottom)); padding: 10px; }
 .composer-actions { flex-wrap: wrap; }
 .composer-actions > span { flex-basis: 100%; }
 .learning-page { padding-bottom: 210px; }
 .conversation-user, .conversation-assistant { gap: 8px; }
}
</style>
