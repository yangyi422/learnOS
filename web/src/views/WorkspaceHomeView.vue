<template>
  <PageShell class="workspace-home" variant="wide">
    <PageHeader title="今天，专注一件事" :description="dateLabel">
      <template #actions>
        <div class="workspace-context-bar">
          <el-radio-group v-model="scene" :disabled="!contextReady" aria-label="工作台场景" class="workspace-context-switch">
            <el-radio-button value="all">全部</el-radio-button>
            <el-radio-button value="work">工作</el-radio-button>
            <el-radio-button value="life">生活</el-radio-button>
          </el-radio-group>
          <RouterLink v-if="scene === 'all' && !home?.errors.projects && (home?.unclassified_count ?? 0) > 0" class="workspace-context-organize" to="/projects?view=list&classify=unclassified">整理未分类项目（{{ home?.unclassified_count }}）</RouterLink>
          <span v-if="!contextCanPersist" class="home-muted workspace-context-notice" role="status">当前场景可用，但无法保存到浏览器。</span>
        </div>
      </template>
    </PageHeader>
    <Teleport to="#home-header-actions">
      <form class="home-search" role="search" @submit.prevent="openSearchResult(searchResults[0])" @focusout="leaveSearch" @keydown.esc.prevent="searchFocused = false">
        <WorkspaceIcon name="search" />
        <input v-model="homeSearch" type="search" aria-label="搜索当前工作区" placeholder="搜索工作区" aria-controls="home-search-results" :aria-expanded="searchFocused && !!homeSearch.trim()" autocomplete="off" @focus="searchFocused = true" @input="searchFocused = true">
        <div v-if="searchFocused && homeSearch.trim()" id="home-search-results" class="home-search__results">
          <ul v-if="searchResults.length" aria-label="工作区搜索结果">
            <li v-for="item in searchResults" :key="item.key"><button type="button" @click="openSearchResult(item)"><span>{{ item.title }}</span><small>{{ item.group }}</small></button></li>
          </ul>
          <p v-else role="status">当前工作区没有匹配内容</p>
        </div>
      </form>
      <el-button class="home-capture-button" @click="captureOpen = true"><WorkspaceIcon name="inbox" />快速记录</el-button>
    </Teleport>
    <Teleport to="#home-inbox-count"><span v-if="home?.inbox && !home.errors.inbox" :aria-label="home.inbox.count + ' 条待整理'">{{ home.inbox.count }}</span></Teleport>

    <p v-if="loadError && home" class="workspace-home__refresh-error" role="status">
      {{ isSwitching ? '该场景暂时无法加载。' : '工作区更新失败，仍显示上次内容。' }}{{ loadError }}
      <button class="inline-action" type="button" @click="load">重试</button>
    </p>

    <div v-if="loading && !home" class="workspace-home__grid workspace-home__grid--loading" aria-label="正在整理工作区" aria-busy="true">
      <div ref="focusFrame" class="focus-card-frame">
        <section class="focus-card focus-card--skeleton daily-focus-card--skeleton" aria-label="今日焦点正在加载">
          <el-skeleton animated :rows="3" />
        </section>
      </div>
      <section v-for="section in loadingSections" :key="section" class="home-section home-section--skeleton" :class="sectionClass(section)">
        <SectionHeader :title="sectionTitle(section)" />
        <el-skeleton animated :rows="3" />
      </section>
    </div>

    <div v-else-if="!home && loadError" class="workspace-home__fatal-error" role="alert">
      <EmptyState title="工作区暂时无法加载" :description="loadError" action-label="重试" @action="load" />
    </div>

    <div v-else-if="home" class="workspace-home__grid" :class="{ 'workspace-home__grid--switching': isSwitching }" :aria-busy="loading">
      <span v-if="isSwitching" class="sr-only" role="status">{{ loadError || '正在切换场景…' }}</span>
      <div ref="focusLane" class="home-focus-lane" :style="focusLaneStyle">
      <div ref="focusCluster" class="focus-cluster" :style="focusClusterStyle">
      <div ref="focusFrame" class="focus-card-frame">
        <DailyFocusCard
          class="daily-focus-card"
          :style="focusCardStyle"
          :scene="scene"
          :date="currentDate"
          :focuses="focusViews"
          :busy="focusBusy"
          :errors="focusErrors"
          :conflicts="focusConflicts"
          :load-error="home.errors.daily_focus ?? ''"
          :switching="isSwitching"
          @pick="openFocusPicker"
          @clear="clearFocus"
          @complete="completeFocus"
          @retry="retryFocus"
          @reload="load"
        />
      </div>
      <div v-if="home.focus.kind !== 'empty'" class="continue-focus">
        <span class="continue-focus__label">继续处理建议</span>
        <RouterLink v-if="suggestionTarget" :to="suggestionTarget" class="continue-focus__link" :aria-label="'打开建议：' + home.focus.title">{{ home.focus.title }}</RouterLink>
        <strong v-else class="continue-focus__link">{{ home.focus.title }}</strong>
        <small v-if="suggestionNote" class="continue-focus__note">{{ suggestionNote }}</small>
      </div>
      </div>

      <section class="home-section learning-section" aria-labelledby="learning-title">
        <WorkspaceIcon name="courses" class="home-section__icon" />
        <SectionHeader title="继续学习 · 全局" id="learning-title" />
        <div v-if="home.errors.learning" class="module-error" role="alert">
          <span>{{ home.errors.learning }}</span>
          <button class="inline-action" type="button" @click="load">重试</button>
        </div>
        <div v-else-if="home.learning[0]" class="learning-row">
          <span class="learning-row__mark" aria-hidden="true"><WorkspaceIcon name="courses" /></span>
          <span class="learning-row__text">
            <small class="learning-row__course">{{ home.learning[0].course_name }}</small>
            <strong>{{ home.learning[0].lesson_title || '选择下一段学习' }}</strong>
            <small class="learning-row__summary" :title="home.learning[0].core_question || home.learning[0].unit_title">{{ home.learning[0].core_question || home.learning[0].unit_title || (home.learning[0].has_current_lesson ? '当前学习单元' : '从知识地图选择一个主题') }}</small>
            <span class="learning-row__meta">
              <small v-if="home.learning[0].core_question && home.learning[0].unit_title">{{ home.learning[0].unit_title }}</small>
              <small class="learning-row__level">{{ levelLabel(home.learning[0].current_level) }} · {{ cognitiveStatusLabel(home.learning[0].cognitive_status) }}</small>
            </span>
            <svg class="learning-path" viewBox="0 0 220 200" fill="none" aria-hidden="true" focusable="false">
              <circle cx="110" cy="18" r="10" /><path d="M110 28v12c0 18-62 12-62 38M110 40c0 18 62 12 62 38" />
              <rect x="18" y="78" width="60" height="38" rx="4" /><rect x="142" y="78" width="60" height="38" rx="4" />
              <path d="m37 96 7 7 15-16M164 91l16 14m0-14-16 14M48 116c0 34 62 24 62 52M172 116c0 34-62 24-62 52" /><circle cx="110" cy="179" r="10" />
            </svg>
          </span>
        </div>
        <EmptyState
          v-else
          title="知识世界还是空的"
          description="创建一个学习领域后，这里会保存你最近的学习位置。"
          action-label="开始一个学习领域"
          @action="router.push('/domains/new')"
        />
        <RouterLink v-if="!home.errors.learning && home.learning[0]" :to="learningTarget(home.learning[0])" class="home-card-entry learning-card-entry" :aria-label="(home.learning[0].has_current_lesson ? '继续学习：' : '打开知识地图：') + (home.learning[0].lesson_title || home.learning[0].course_name)"><span aria-hidden="true">↗</span></RouterLink>
      </section>
      </div>
      <div class="home-project-lane">
      <section ref="projectsCard" class="home-section projects-section" :inert="isSwitching" :style="{ minHeight: sceneGeometry.projects ? sceneGeometry.projects + 'px' : undefined }" aria-labelledby="projects-title">
        <WorkspaceIcon name="projects" class="home-section__icon" />
        <SectionHeader title="正在推进" id="projects-title">
          <template #actions>
            <el-button text @click="router.push('/projects?view=list')">项目列表 →</el-button>
          </template>
        </SectionHeader>
        <div v-if="home.errors.projects" class="module-error" role="alert">
          <span>{{ home.errors.projects }}</span>
          <button class="inline-action" type="button" @click="load">重试</button>
        </div>
        <div v-else-if="home.projects.length" class="home-project-grid">
          <button
            v-for="project in home.projects.slice(0, 5)"
            :key="project.id"
            class="project-row"
            type="button"
            :aria-label="project.title + '，打开项目看板'"
            @click="openProject(project.id)"
          >
            <span class="project-row__accent" :style="{ color: project.accent || undefined, borderColor: project.accent || undefined }">{{ projectIcon(project) }}</span>
            <span class="project-row__text">
              <span class="project-row__heading"><strong>{{ project.title }}</strong><span class="project-row__arrow" aria-hidden="true">→</span></span>
              <small v-if="project.description">{{ project.description }}</small>
              <span v-if="projectTaskTotal(project)" class="project-row__progress-line">
                <span class="project-row__progress" aria-hidden="true"><i :style="{ width: Math.min(100, project.done_task_count / projectTaskTotal(project) * 100) + '%' }" /></span>
                <small class="project-row__counts" :title="'已完成 ' + project.done_task_count + ' 项，共 ' + projectTaskTotal(project) + ' 项任务'">{{ project.done_task_count }} / {{ projectTaskTotal(project) }}</small>
              </span>
            </span>
          </button>
        </div>
        <EmptyState
          v-else
          title="当前场景还没有进行中的项目"
          description="创建项目后，正在推进的任务会集中显示在这里。"
          action-label="查看项目"
          @action="router.push('/projects?view=list')"
        />
      </section>

      <section ref="todayCard" class="home-section today-section" :inert="isSwitching" :style="{ minHeight: sceneGeometry.today ? sceneGeometry.today + 'px' : undefined }" aria-labelledby="today-title">
        <WorkspaceIcon name="calendar" class="home-section__icon" />
        <DailyPlanPanel :planning="home.planning" :date="currentDate" :context="scene" :loading="loading || isSwitching" :error="home.errors.planning" @open="openTask" @refresh="load" />
      </section>

      </div>
      <section v-if="recentItems.length || recentLoading || recentError || recentProblems.length" class="home-section recent-section" aria-labelledby="recent-title">
        <WorkspaceIcon name="inbox" class="home-section__icon" />
        <SectionHeader title="最近留下的 · 全局" id="recent-title" />
        <p v-if="recentLoading && !recent" class="home-muted" role="status">正在读取最近记录…</p>
        <div v-if="recentItems.length" class="home-recent-grid">
        <RouterLink v-for="item in recentItems" :key="item.key" :to="item.to" class="home-recent-row" :title="item.title">
          <span v-if="item.cover" class="home-recent-cover"><img :src="item.cover" alt="" loading="lazy" decoding="async" referrerpolicy="no-referrer" @error="hideFailedImage" /></span>
          <span v-else class="home-recent-icon" aria-hidden="true"><WorkspaceIcon :name="item.key.startsWith('library-') ? 'courses' : 'life'" /></span>
          <span class="home-recent-copy"><strong>{{ item.title }}</strong><small><span>{{ item.source }}</span><span class="home-recent-date">{{ item.dateLabel }}</span></small></span>
          <span class="home-muted" aria-hidden="true">→</span>
        </RouterLink>
        </div>
        <p v-if="recentError || recentProblems.length" class="module-error" role="status">
          <span>{{ recentError || recentProblems.join(' ') }}</span>
          <button class="inline-action" type="button" @click="loadRecent">重试最近内容</button>
        </p>
      </section>

      <nav class="home-shortcuts" aria-label="个人档案快捷入口">
        <span class="home-shortcuts__label">个人档案</span>
        <RouterLink to="/inbox">收集箱 · {{ home.errors.inbox ? '数量暂不可用' : (home.inbox?.count ?? 0) + ' 条待整理' }}</RouterLink>
        <button v-if="home.errors.inbox" class="inline-action" type="button" @click="load">重试收集箱</button>
        <RouterLink to="/explore/library">我的资料库</RouterLink>
        <RouterLink to="/life">生活档案</RouterLink>
        <RouterLink to="/life/finance">财务档案<small v-if="recent?.asset_date"> · 最近快照 {{ formatShortDate(recent.asset_date) }}</small></RouterLink>
      </nav>
    </div>

    <FocusPickerDialog
      v-model="pickerOpen"
      :date="currentDate"
      :context="pickerContext ?? 'work'"
      :current="pickerFocus"
      @saved="applyPickedFocus"
      @conflict="load"
    />
    <QuickCaptureDialog v-model="captureOpen" @created="captured" />
  </PageShell>
</template>

<script setup lang="ts">
import DailyPlanPanel from '@/components/workspace/DailyPlanPanel.vue'
import DailyFocusCard from '@/components/workspace/DailyFocusCard.vue'
import FocusPickerDialog from '@/components/workspace/FocusPickerDialog.vue'
import { subscribeTaskChanges } from '@/task-changes'
import { currentUser } from '@/api/auth'
import { readWorkspaceContext, saveWorkspaceContext, type WorkspaceContext } from '@/workspace-context'
import PageShell from '@/components/PageShell.vue'
import WorkspaceIcon from '@/components/WorkspaceIcon.vue'
import { computed, onMounted, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { APIRequestError } from '@/api/http'
import { type InboxItem } from '@/api/inbox'
import { emptyDailyFocus, setDailyFocus, dailyFocusTaskRoute, type DailyFocusContext, type DailyFocusView, type WorkspaceDailyFocus } from '@/api/dailyFocus'
import { getTask, updateTask, type Project, type ProjectTask } from '@/api/projects'
import { getWorkspaceHome, getWorkspaceRecent, type WorkspaceRecent, type WorkspaceHome, type WorkspaceLearningItem } from '@/api/workspace'
import EmptyState from '@/components/EmptyState.vue'
import PageHeader from '@/components/PageHeader.vue'
import SectionHeader from '@/components/SectionHeader.vue'
import QuickCaptureDialog from '@/components/QuickCaptureDialog.vue'
import { libraryCardImageURL, localLibraryImage } from '@/utils/libraryMarkdown'
import { safeExternalLink } from '@/utils/externalLinks'
import { domainLabel } from '@/api/life'

type HomeSection = 'today' | 'projects' | 'learning'

const router = useRouter()
const scene = ref<WorkspaceContext>('all')
const currentDate = ref(localDate())
const loadedRequestDate = ref('')
const contextCanPersist = ref(true)
const contextReady = ref(false)
let contextUserID: number | undefined
const loadedScene = ref<WorkspaceContext>()
const focusLane = ref<HTMLElement | null>(null)
const focusCluster = ref<HTMLElement | null>(null)
const focusFrame = ref<HTMLElement | null>(null)
const projectsCard = ref<HTMLElement | null>(null)
const todayCard = ref<HTMLElement | null>(null)
const sceneGeometry = ref({ focus: 0, focusCluster: 0, focusLane: 0, projects: 0, today: 0 })
function preserveSceneGeometry() {
  // Keep the space already used in this viewport; longer content can still grow.
  // The focus lane and its card+suggestion cluster are recorded separately:
  // a scene switch can remove the secondary suggestion row, and the reserved
  // heights keep the learning/today/projects positions stable without shrinking.
  sceneGeometry.value = {
    focus: Math.max(sceneGeometry.value.focus, focusFrame.value?.getBoundingClientRect().height ?? 0),
    focusCluster: Math.max(sceneGeometry.value.focusCluster, focusCluster.value?.getBoundingClientRect().height ?? 0),
    focusLane: Math.max(sceneGeometry.value.focusLane, focusLane.value?.getBoundingClientRect().height ?? 0),
    projects: Math.max(sceneGeometry.value.projects, projectsCard.value?.getBoundingClientRect().height ?? 0),
    today: Math.max(sceneGeometry.value.today, todayCard.value?.getBoundingClientRect().height ?? 0),
  }
}
let sceneViewportWidth = 0
function resetSceneGeometry() {
  if (sceneViewportWidth === window.innerWidth) return
  sceneViewportWidth = window.innerWidth
  sceneGeometry.value = { focus: 0, focusCluster: 0, focusLane: 0, projects: 0, today: 0 }
}
watch([scene, currentDate], () => {
 if (contextUserID !== undefined) contextCanPersist.value = saveWorkspaceContext(contextUserID, scene.value)
 resetFocusWriteState()
 preserveSceneGeometry()
 void load()
}, { flush: 'sync' })
const home = ref<WorkspaceHome | null>(null)
const isSwitching = computed(() => home.value !== null && (loadedScene.value !== scene.value || loadedRequestDate.value !== currentDate.value))
const loading = ref(true)
const loadError = ref('')
const captureOpen = ref(false)
const homeSearch = ref(''), searchFocused = ref(false)
interface HomeSearchResult { key: string; title: string; group: string; open: () => void }
const loadingSections: HomeSection[] = ['today', 'projects', 'learning']
const recent = ref<WorkspaceRecent | null>(null)
const recentLoading = ref(true), recentError = ref('')
const recentProblems = computed(() => Object.values(recent.value?.errors ?? {}))
const recentItems = computed(() => {
  const facts = recent.value
  if (!facts) return []
  const library = (facts.library ?? []).map(item => ({
    key: 'library-' + item.id, title: item.title, source: '资料库 · ' + item.collection_name,
    dateLabel: '更新 ' + formatDate(item.updated_at), sort: new Date(item.updated_at).getTime(),
    to: { path: `/explore/library/${item.collection_id}/pages/${item.id}`, query: { view: item.display_mode } },
    cover: localLibraryImage(item.cover_url) || (/^https?:\/\//i.test(item.cover_url) && safeExternalLink(item.cover_url)) ? libraryCardImageURL(item.cover_url) : '',
  }))
  // Calendar dates are interpreted at local midnight, never as UTC timestamps.
  const occurrence = (date: string) => new Date(date + 'T00:00:00').getTime()
  const events = (facts.events ?? []).map(item => ({
    key: 'event-' + item.id, title: item.title, source: '生活' + (domainLabel(item.primary_domain) ? ' · ' + domainLabel(item.primary_domain) : ''),
    dateLabel: '发生 ' + formatShortDate(item.occurred_on), sort: occurrence(item.occurred_on),
    to: { path: '/life', query: { event: String(item.id) } }, cover: '',
  }))
  const goals = (facts.goals ?? []).map(item => ({
    key: 'goal-' + item.id, title: item.content || '阶段变化', source: '长期目标 · ' + item.goal_title,
    dateLabel: '变化 ' + formatShortDate(item.occurred_on), sort: occurrence(item.occurred_on),
    to: { path: '/life', query: { view: 'goals', goal: String(item.goal_id) } }, cover: '',
  }))
  return [...library, ...events, ...goals].sort((a, b) => b.sort - a.sort || a.key.localeCompare(b.key)).slice(0, 6)
})
let homeRequest: AbortController | undefined, recentRequest: AbortController | undefined
let disposed = false
const dateLabel = computed(() => new Intl.DateTimeFormat('zh-CN', { year: 'numeric', month: 'long', day: 'numeric', weekday: 'long' }).format(new Date(currentDate.value + 'T12:00:00')))
function projectIcon(project: Pick<Project, 'title'> & { icon?: string | null }) {
  const icon = project.icon?.trim()
  return icon && !/^[·•.。∙⋅…]+$/u.test(icon) ? icon : Array.from(project.title.trim())[0] || '项'
}
// Daily focus is the user's explicit choice for each scene. It only changes
// after the server confirms the write; failed writes keep the last confirmed
// view (no optimistic UI) and expose a retry.
const focusViews = ref<WorkspaceDailyFocus>(emptyDailyFocus())
const focusBusy = reactive<Record<DailyFocusContext, boolean>>({ work: false, life: false })
const focusErrors = reactive<Record<DailyFocusContext, string>>({ work: '', life: '' })
const focusConflicts = reactive<Record<DailyFocusContext, boolean>>({ work: false, life: false })
type FocusAttempt = { kind: 'clear' } | { kind: 'set'; taskID: number } | { kind: 'complete' }
const focusAttempts = reactive<Record<DailyFocusContext, FocusAttempt | null>>({ work: null, life: null })
const pickerContext = ref<DailyFocusContext | null>(null)
const pickerOpen = computed({
  get: () => pickerContext.value !== null,
  set: value => { if (!value) pickerContext.value = null },
})
const pickerFocus = computed(() => (pickerContext.value ? focusViews.value[pickerContext.value] : null))
const focusCardStyle = computed(() => (sceneGeometry.value.focus ? { minHeight: sceneGeometry.value.focus + 'px' } : {}))
const focusClusterStyle = computed(() => (sceneGeometry.value.focusCluster ? { minHeight: sceneGeometry.value.focusCluster + 'px' } : {}))
const focusLaneStyle = computed(() => (sceneGeometry.value.focusLane ? { minHeight: sceneGeometry.value.focusLane + 'px' } : {}))
function resetFocusWriteState() {
  for (const context of ['work', 'life'] as const) {
    focusBusy[context] = false
    focusErrors[context] = ''
    focusConflicts[context] = false
    focusAttempts[context] = null
  }
  pickerContext.value = null
}
function applyDailyFocus(source?: WorkspaceDailyFocus | null) {
  focusViews.value = { work: source?.work ?? null, life: source?.life ?? null }
}
function focusScopeKey() { return `${scene.value}:${currentDate.value}` }
function openFocusPicker(context: DailyFocusContext) {
  focusErrors[context] = ''
  focusConflicts[context] = false
  pickerContext.value = context
}
function applyPickedFocus(focus: DailyFocusView) {
  focusViews.value = { ...focusViews.value, [focus.context]: focus }
  focusErrors[focus.context] = ''
  focusConflicts[focus.context] = false
  focusAttempts[focus.context] = null
  pickerContext.value = null
}
function findHomeTask(taskID: number): ProjectTask | undefined {
  const data = home.value
  if (!data) return undefined
  const pools = [data.planning?.items, data.planning?.due_today, data.planning?.overdue, data.planning?.previous_plan, data.today?.doing, data.today?.due, data.today?.next, ...data.projects.map(project => project.tasks)]
  for (const pool of pools) {
    const match = pool?.find(task => task.id === taskID)
    if (match) return match
  }
  return undefined
}
async function resolveFocusTask(focus: DailyFocusView): Promise<ProjectTask> {
  if (!focus.task_id) throw new APIRequestError({ code: 'DAILY_FOCUS_TASK_INVALID', message: '焦点任务已不存在，请刷新后重试。', retryable: true })
  return findHomeTask(focus.task_id) ?? await getTask(focus.task_id)
}
async function runFocusAttempt(context: DailyFocusContext, attempt: FocusAttempt) {
  if (focusBusy[context]) return
  const scope = focusScopeKey()
  focusBusy[context] = true
  focusErrors[context] = ''
  focusConflicts[context] = false
  focusAttempts[context] = attempt
  try {
    if (attempt.kind === 'complete') {
      const focus = focusViews.value[context]
      if (!focus?.task_id) return
      // The focus view carries the task's updated_at; only older payloads need
      // the extra lookup, so the normal path never issues an extra GET.
      const expectedUpdatedAt = focus.task_updated_at ?? (await resolveFocusTask(focus)).updated_at
      const saved = await updateTask(focus.task_id, { status: 'done', expected_updated_at: expectedUpdatedAt })
      if (disposed || scope !== focusScopeKey()) return
      focusViews.value = {
        ...focusViews.value,
        [context]: { ...focus, state: 'completed', task_status: 'done', completed_at: saved.completed_at ?? new Date().toISOString(), task_updated_at: saved.updated_at, updated_at: saved.updated_at },
      }
      focusAttempts[context] = null
      return
    }
    const current = focusViews.value[context]
    const result = await setDailyFocus(currentDate.value, context, attempt.kind === 'set' ? attempt.taskID : null, current?.revision ?? 0)
    if (disposed || scope !== focusScopeKey()) return
    focusViews.value = { ...focusViews.value, [context]: result.focus }
    focusAttempts[context] = null
  } catch (reason) {
    if (disposed || scope !== focusScopeKey()) return
    if (reason instanceof APIRequestError && reason.status === 409) {
      focusConflicts[context] = true
      focusErrors[context] = '内容刚刚在别处更新，已重新加载最新状态，请重试。'
      void load()
    } else {
      focusErrors[context] = safeMessage(reason, attempt.kind === 'complete' ? '任务没有完成，焦点保持原状态。' : '焦点没有保存，仍保持之前确认的状态。')
    }
  } finally {
    focusBusy[context] = false
  }
}
function clearFocus(context: DailyFocusContext) { void runFocusAttempt(context, { kind: 'clear' }) }
function completeFocus(focus: DailyFocusView) { void runFocusAttempt(focus.context, { kind: 'complete' }) }
function retryFocus(context: DailyFocusContext) {
  const attempt = focusAttempts[context]
  if (attempt) void runFocusAttempt(context, attempt)
}

function projectTaskTotal(project: Project) { return Math.max(0, project.open_task_count + project.done_task_count) }
const searchResults = computed<HomeSearchResult[]>(() => {
  const query = homeSearch.value.trim().toLocaleLowerCase()
  if (!query || !home.value || isSwitching.value) return []
  const entries: HomeSearchResult[] = []
  for (const context of ['work', 'life'] as const) {
    const focus = focusViews.value[context]
    if (!focus?.title) continue
    entries.push({
      key: 'daily-focus-' + context,
      title: focus.title,
      group: '今日焦点',
      open: () => {
        const target = dailyFocusTaskRoute(focus)
        if (target) void router.push(target)
        else openFocusPicker(context)
      },
    })
  }
  const suggestion = home.value.focus
  if (suggestion.kind !== 'empty') entries.push({ key: 'suggestion', title: suggestion.title, group: '继续处理建议', open: openSuggestion })
  const focusTasks = new Set(Object.values(focusViews.value).filter((focus): focus is DailyFocusView => focus !== null).map(focus => focus.task_id))
  const tasks = [...(home.value.planning?.items ?? []), ...(home.value.planning?.due_today ?? []), ...(home.value.planning?.overdue ?? []), ...(home.value.planning?.previous_plan ?? []), ...(home.value.today?.doing ?? []), ...(home.value.today?.due ?? []), ...(home.value.today?.next ?? [])]
  for (const task of new Map(tasks.map(task => [task.id, task])).values()) {
    if (focusTasks.has(task.id)) continue
    entries.push({ key: 'task-' + task.id, title: task.title, group: '任务', open: () => openTask(task) })
  }
  for (const project of home.value.projects) entries.push({ key: 'project-' + project.id, title: project.title, group: '项目', open: () => openProject(project.id) })
  for (const item of home.value.learning) entries.push({ key: 'learning-' + item.course_id, title: item.course_name + '：' + (item.lesson_title || item.unit_title), group: '学习', open: () => openLearning(item) })
  for (const item of recentItems.value) entries.push({ key: item.key, title: item.title, group: '最近记录', open: () => { void router.push(item.to) } })
  return entries.filter(item => item.title.toLocaleLowerCase().includes(query)).slice(0, 8)
})
function openSearchResult(item?: HomeSearchResult) { if (item) { searchFocused.value = false; item.open() } }
function leaveSearch(event: FocusEvent) {
  if (!(event.relatedTarget instanceof Node) || !(event.currentTarget as HTMLElement).contains(event.relatedTarget)) searchFocused.value = false
}
function localDate() {
  const date = new Date()
  return date.getFullYear() + '-' + String(date.getMonth() + 1).padStart(2, '0') + '-' + String(date.getDate()).padStart(2, '0')
}
function sectionClass(section: HomeSection) { return section + '-section' }
function sectionTitle(section: HomeSection) {
  return ({ today: '今日计划', projects: '正在推进', learning: '继续学习' })[section]
}
function taskStatusLabel(status?: string) {
  return ({ doing: '进行中', next: '下一步', inbox: '待整理' } as Record<string, string>)[status ?? ''] ?? '项目任务'
}
function levelLabel(level: string) {
  return ({ unseen: '未接触', exposed: '初见', recognize: '识别', understand: '理解', apply: '应用', transfer: '迁移' } as Record<string, string>)[level] ?? '未接触'
}
function cognitiveStatusLabel(status?: string) {
  return ({ unknown: '尚无证据', developing: '发展中', stable: '稳定', needs_review: '待复核' } as Record<string, string>)[status ?? ''] ?? '尚无证据'
}
function relativeTime(value: string) {
  const timestamp = new Date(value).getTime()
  if (!Number.isFinite(timestamp)) return '最近'
  const elapsed = Math.max(0, Date.now() - timestamp)
  if (elapsed < 60_000) return '刚刚'
  if (elapsed < 3_600_000) return Math.floor(elapsed / 60_000) + ' 分钟前'
  if (elapsed < 86_400_000) return Math.floor(elapsed / 3_600_000) + ' 小时前'
  return Math.floor(elapsed / 86_400_000) + ' 天前'
}
function formatDate(value: string) {
  return new Intl.DateTimeFormat('zh-CN', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' }).format(new Date(value))
}
function formatShortDate(value: string) {
  const [year, month, day] = value.split('-')
  return year + '年' + Number(month) + '月' + Number(day) + '日'
}
function safeMessage(reason: unknown, fallback: string) {
  return reason instanceof APIRequestError ? reason.message : fallback
}
async function load() {
  homeRequest?.abort()
  const controller = homeRequest = new AbortController()
  loading.value = true
  loadError.value = ''
  try {
    const requestedScene = scene.value
    const requestedDate = currentDate.value
    const result = await getWorkspaceHome(requestedDate, controller.signal, requestedScene)
    if (!disposed && controller === homeRequest && requestedScene === scene.value && requestedDate === currentDate.value) {
      home.value = result
      applyDailyFocus(result.daily_focus)
      loadedScene.value = requestedScene
      loadedRequestDate.value = requestedDate
    }
  } catch (reason) {
    if (!disposed && !controller.signal.aborted) loadError.value = safeMessage(reason, '工作区暂时无法加载，请检查连接后重试。')
  } finally {
    if (!disposed && controller === homeRequest) loading.value = false
  }
}
async function loadRecent() {
  recentRequest?.abort()
  const controller = recentRequest = new AbortController()
  recentLoading.value = true
  recentError.value = ''
  // Clear transient recent links on refresh so removed sources are not retained.
  recent.value = null
  try {
    const result = await getWorkspaceRecent(controller.signal)
    if (!disposed && controller === recentRequest) recent.value = result
  } catch (reason) {
    if (!disposed && !controller.signal.aborted) recentError.value = safeMessage(reason, '最近内容暂时无法读取，其他功能仍可使用。')
  } finally {
    if (!disposed && controller === recentRequest) recentLoading.value = false
  }
}
function hideFailedImage(event: Event) {
  if (event.target instanceof HTMLImageElement) event.target.hidden = true
}

function captured(item: InboxItem) {
  if (home.value?.inbox && item.status === 'inbox' && !home.value.inbox.items.some(existing => existing.id === item.id)) {
    home.value.inbox.items = [item, ...home.value.inbox.items.filter(existing => existing.id !== item.id)].slice(0, 3)
    home.value.inbox.count += 1
  }
  void load()
}
function openTask(task: ProjectTask) {
  router.push({ path: '/projects', query: { project: String(task.project_id), view: 'board', task: String(task.id) } })
}
function openProject(projectID: number) {
  router.push({ path: '/projects', query: { project: String(projectID), view: 'board' } })
}
// The former auto-suggestion is secondary now: it never writes daily focus.
const suggestionTarget = computed(() => {
  const focus = home.value?.focus
  if (focus?.kind === 'task' && focus.task_id) return { path: '/projects', query: { project: String(focus.project_id ?? 'all'), view: 'board', task: String(focus.task_id) } }
  if (focus?.kind === 'learning' && focus.course_id) return focus.lesson_id ? '/courses/' + focus.course_id + '/learn' : '/courses/' + focus.course_id + '/map'
  return null
})
const suggestionNote = computed(() => {
  const focus = home.value?.focus
  if (!focus || focus.kind === 'empty') return ''
  if (focus.kind === 'learning') return ['学习位置', focus.course_name, focus.unit_title].filter(Boolean).join(' · ')
  return [focus.project_title || '项目任务', taskStatusLabel(focus.status)].filter(Boolean).join(' · ')
})
function openSuggestion() { if (suggestionTarget.value) void router.push(suggestionTarget.value) }
function learningTarget(item: WorkspaceLearningItem) { return '/courses/' + item.course_id + (item.has_current_lesson ? '/learn' : '/map') }
function openLearning(item: WorkspaceLearningItem) { void router.push(learningTarget(item)) }
// Share the actual card axis with the header, including constrained reading widths.
let axisObserver: ResizeObserver | undefined
let axisShell: HTMLElement | null = null
function updateNavigationAxis() {
  const frame = focusFrame.value
  if (!frame) return
  const header = frame.closest('.app-shell--home')?.querySelector('.home-topbar')
  axisShell = frame.closest('.app-shell--home') as HTMLElement | null
  if (!header || !axisShell) return
  const card = frame.getBoundingClientRect()
  axisShell.style.setProperty('--home-focus-axis', `${card.left + card.width / 2 - header.getBoundingClientRect().left}px`)
}
watch(focusFrame, frame => {
  axisObserver?.disconnect()
  if (!frame) return
  axisObserver = new ResizeObserver(updateNavigationAxis)
  axisObserver.observe(frame)
  const main = frame.closest('.main-content')
  if (main) axisObserver.observe(main)
  updateNavigationAxis()
}, { flush: 'post' })
onMounted(() => {
  sceneViewportWidth = window.innerWidth
  window.addEventListener('resize', updateNavigationAxis)
  window.addEventListener('resize', resetSceneGeometry)
})
onBeforeUnmount(() => {
  axisObserver?.disconnect()
  window.removeEventListener('resize', updateNavigationAxis)
  window.removeEventListener('resize', resetSceneGeometry)
  axisShell?.style.removeProperty('--home-focus-axis')
})

let stopTaskChanges: (() => void) | undefined
let calendarTimer: ReturnType<typeof setTimeout> | undefined
function refreshCalendarDate() {
  const nextDate = localDate()
  if (nextDate !== currentDate.value) currentDate.value = nextDate
}
function scheduleCalendarRefresh() {
  clearTimeout(calendarTimer)
  refreshCalendarDate()
  const now = new Date()
  const midnight = new Date(now.getFullYear(), now.getMonth(), now.getDate() + 1)
  calendarTimer = setTimeout(scheduleCalendarRefresh, Math.min(60_000, midnight.getTime() - now.getTime() + 25))
}
function resumeWorkspace() {
  if (document.visibilityState === 'hidden') return
  const previous = currentDate.value
  refreshCalendarDate()
  if (previous === currentDate.value && contextReady.value) void load()
  scheduleCalendarRefresh()
}
onMounted(() => {
  stopTaskChanges = subscribeTaskChanges(() => { if (contextReady.value) void load() })
  document.addEventListener('visibilitychange', resumeWorkspace)
  window.addEventListener('focus', resumeWorkspace)
  scheduleCalendarRefresh()
  void loadRecent()
  void initializeContext()
})
async function initializeContext() {
  try {
    const user = await currentUser()
    if (disposed) return
    contextUserID = user.id
    contextReady.value = true
    const saved = readWorkspaceContext(user.id)
    if (saved !== scene.value) { scene.value = saved; return }
  } catch {
    if (disposed) return
    contextCanPersist.value = false
    contextReady.value = true
  }
  if (!disposed) void load()
}
onBeforeUnmount(() => {
  disposed = true; homeRequest?.abort(); recentRequest?.abort(); stopTaskChanges?.(); clearTimeout(calendarTimer)
  document.removeEventListener('visibilitychange', resumeWorkspace)
  window.removeEventListener('focus', resumeWorkspace)
})
</script>

<style scoped>
:global(html:has(.workspace-home)) { scrollbar-gutter:stable; }
.workspace-home__grid--switching :is(.focus-card,.projects-section,.today-section) > * { visibility:hidden; }
.workspace-home__grid--switching :is(.focus-card,.projects-section,.today-section)::after { content:'正在加载…'; position:absolute; inset:0; display:flex; align-items:center; justify-content:center; color:var(--home-muted); font-size:14px; pointer-events:none; }

.workspace-context-bar { position:relative; display:flex; flex-direction:column; align-items:flex-end; gap:8px; min-width:0; }
.workspace-context-switch { display:inline-flex; flex-wrap:nowrap; gap:4px; padding:4px; border:1px solid var(--home-edge); border-radius:999px; background:var(--home-panel); box-shadow:var(--home-panel-inset); backdrop-filter:var(--home-glass-filter); -webkit-backdrop-filter:var(--home-glass-filter); }
.workspace-context-switch :deep(.el-radio-button__inner) { min-height:36px; display:flex; align-items:center; padding:0 18px; border:0; outline:0; border-radius:999px; background:transparent; color:var(--home-secondary); font-family:var(--font-ui); font-size:14px; font-weight:500; line-height:1.4; box-shadow:none; transition:background-color 180ms ease,color 180ms ease; }
.workspace-context-switch :deep(.el-radio-button.is-active .el-radio-button__original-radio:not(:disabled) + .el-radio-button__inner) { background:var(--home-tint); color:var(--home-link); box-shadow:none; }
.workspace-context-switch :deep(.el-radio-button__inner:hover) { color:var(--home-link); }
.workspace-context-switch :deep(.el-radio-button__original-radio:focus-visible + .el-radio-button__inner) { outline:2px solid var(--home-link); outline-offset:3px; }
.workspace-context-organize { position:absolute; top:calc(100% + 8px); right:0; white-space:nowrap; color:var(--home-muted); font-size:13px; line-height:1.6; text-underline-offset:3px; }
.workspace-context-organize:hover { color:var(--home-link); }
.workspace-context-notice { max-width:280px; text-align:right; }
.workspace-home :deep(.page-header__row) { align-items:center; }
.workspace-home :deep(.page-header__actions) { justify-content:flex-end; margin-left:auto; }
@media (max-width:700px) { .workspace-context-switch :deep(.el-radio-button__inner) { min-height:44px; } }
@media (prefers-reduced-motion:reduce) { .workspace-context-switch :deep(.el-radio-button__inner) { transition:none; } }
.workspace-home {
  --text-primary: var(--home-ink);
  --text-secondary: var(--home-secondary);
  --text-tertiary: var(--home-muted);
  --color-primary: var(--home-link);
  --color-focus: var(--home-link);
  --bg-surface: var(--home-panel);
  --bg-subtle: var(--home-tint);
  --border-default: var(--home-edge);
  --el-color-primary: #713dff;
  --el-color-primary-dark-2: #6030dc;
  --button-primary-text: #fff;
  color: var(--home-ink);
  max-width: 1680px;
}
.workspace-home :deep(.page-header) { margin-bottom: 40px; }
.workspace-home :deep(.page-header h1) { font-family: var(--font-ui); font-size: 44px; font-weight: 600; line-height: 1.4; color: var(--home-ink); }
.workspace-home :deep(.page-header__description) { margin-top: 12px; font-size: 15px; color: var(--home-muted); }
.workspace-home__grid { position: relative; isolation: isolate; display: grid; grid-template-columns: minmax(0,1fr) minmax(0,1.3fr) minmax(0,1fr); align-items: stretch; gap: 24px; }
.home-focus-lane, .home-project-lane { display: flex; flex-direction: column; gap: 22px; min-width: 0; grid-row: 1; }
.home-focus-lane { grid-column: 2; }
.home-project-lane { grid-column: 1; }
/* Card + secondary suggestion row form one block so a scene switch that hides
   the suggestion cannot shrink the lane or move later sections upward. */
.focus-cluster { display: flex; flex-direction: column; gap: 22px; min-width: 0; }
.focus-card-frame { position: relative; min-width: 0; }
.focus-card, .home-section { position: relative; min-width: 0; border: 1px solid var(--home-edge); border-radius: 16px; background: linear-gradient(120deg,var(--home-glass-glint),transparent 26%,transparent 76%,var(--home-glass-shade)),linear-gradient(145deg,var(--home-sheen),transparent 42%),var(--home-panel); background-clip: padding-box; box-shadow: var(--home-panel-inset); backdrop-filter: var(--home-glass-filter); -webkit-backdrop-filter: var(--home-glass-filter); transition: background-color 180ms ease, border-color 180ms ease, box-shadow 180ms ease; }
.focus-card { min-height: 360px; padding: 28px; display: flex; flex-direction: column; gap: 24px; }
.focus-card { background: linear-gradient(120deg,var(--home-glass-glint),transparent 26%,transparent 76%,var(--home-glass-shade)),linear-gradient(180deg,var(--home-focus-top) 0%,var(--home-focus-top) 52%,var(--home-focus-mid) 82%,var(--home-focus-bottom) 100%); }
.home-section { padding: 26px; }
.projects-section, .today-section { min-height:300px; }
.home-section__icon { position: absolute; top: 29px; left: 26px; width: 21px; height: 21px; color: var(--home-secondary); }
.home-section :deep(.section-header) { padding-left: 32px; margin-bottom: 22px; gap: 10px; flex-direction: row; flex-wrap: wrap; align-items: center; text-align: left; }
.home-section :deep(.section-header__actions) { justify-content: flex-end; }
.home-section :deep(.section-header h2) { font-size: 18px; font-weight: 600; line-height: 1.5; color: var(--home-ink); }
.home-section :deep(.section-header__actions .el-button) { padding: 0; height: auto; font-size: 12px; color: var(--home-muted); }
.home-section :deep(.empty-state) { padding: 18px 0; text-align: left; border: 0; background: transparent; }
.home-section :deep(.empty-state h3) { margin: 0 0 10px; font-size: 15px; font-weight: 500; color: var(--home-secondary); }
.home-section :deep(.empty-state p) { margin: 0 0 16px; font-size: 13px; color: var(--home-muted); line-height: 1.8; }
.home-section :deep(.empty-state .el-button) { background: var(--home-tint); border-color: var(--home-edge); color: var(--home-link); border-radius: 999px; }
@media (hover: hover) {
  .focus-card:hover, .home-section:hover { border-color: var(--home-reflection); background-color: var(--home-panel-hover); box-shadow: var(--home-panel-hover-shadow); }
  .home-section:hover :deep(.section-header h2) { color: var(--home-link); }
}
.projects-section, .today-section { flex: none; }
.home-project-grid { display: grid; }
.project-row { display: grid; grid-template-columns: 24px minmax(0,1fr); gap: 12px; align-items: start; width: 100%; min-width: 0; padding: 17px 0; border: 0; border-bottom: 1px solid var(--home-edge); border-radius: 0; background: transparent; color: var(--home-ink); text-align: left; cursor: pointer; }
.project-row:last-child { border-bottom: 0; padding-bottom: 3px; }
.project-row__accent { display: grid; width: 24px; height: 26px; place-items: center; border: 0; background: transparent; font-size: 14px; }
.project-row__text { display: grid; min-width: 0; gap: 7px; }
.project-row__heading { display: flex; align-items: baseline; gap: 8px; min-width: 0; }
.project-row__heading strong { min-width: 0; font-size: 16px; line-height: 1.6; font-weight: 500; overflow-wrap: anywhere; }
.project-row__arrow { flex: none; color: var(--home-muted); font-size: 12px; }
.project-row__text > small { display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; font-size: 13px; color: var(--home-muted); line-height: 1.7; white-space: pre-line; }
.project-row__progress-line { display: flex; align-items: center; gap: 14px; margin-top: 3px; }
.project-row__progress { height: 5px; border-radius: 99px; background: var(--home-tint); flex: 1; overflow: hidden; }
.project-row__progress i { display: block; height: 100%; border-radius: inherit; background: var(--home-link); opacity: .8; }
.project-row__counts { flex: none; font-size: 13px; color: var(--home-secondary); font-variant-numeric: tabular-nums; }
.project-row:hover strong { color: var(--home-link); }
.today-section :deep(.section-header) { margin-bottom: 14px; }
.recent-section { grid-column: 3; grid-row: 1; align-self: stretch; }
.learning-row { display: block; width: 100%; min-width: 0; padding: 5px 0 0; color: var(--home-ink); }
.learning-row__mark { display: none; }
.home-card-entry { position: absolute; inset: -1px; z-index: 1; border-radius: inherit; color: var(--home-link); text-decoration: none; }
.home-card-entry > span { position: absolute; top: 26px; right: 26px; font-size: 20px; line-height: 24px; }
.home-card-entry:focus-visible { outline: 2px solid var(--home-link); outline-offset: 4px; }
.focus-card:focus-within, .home-section:focus-within { border-color: var(--home-reflection); background-color: var(--home-panel-hover); box-shadow: var(--home-panel-hover-shadow); }
.daily-focus-card--skeleton { min-height: 320px; }
/* Secondary suggestion row: the old automatic focus, demoted below the card. */
.continue-focus { display: flex; align-items: baseline; flex-wrap: wrap; gap: 6px 12px; margin-top: -6px; padding: 12px 4px 0; border-top: 1px solid var(--home-edge); color: var(--home-muted); font-size: 13px; line-height: 1.7; }
.continue-focus__label { flex: none; font-size: 12px; color: var(--home-muted); }
.continue-focus__link { min-width: 0; color: var(--home-secondary); font-size: 14px; font-weight: 500; text-decoration: none; overflow-wrap: anywhere; }
a.continue-focus__link:hover { color: var(--home-link); text-decoration: underline; text-underline-offset: 3px; }
a.continue-focus__link:focus-visible { outline: 2px solid var(--home-link); outline-offset: 3px; border-radius: 4px; }
.continue-focus__note { color: var(--home-muted); font-size: 12px; }


.learning-row__text { display: flex; flex-direction: column; align-items: stretch; gap: 14px; min-width: 0; }
.learning-row__course { color: var(--home-secondary); font-size: 14px; line-height: 1.8; }
.learning-row__text strong { font-size: 28px; font-weight: 600; line-height: 1.6; overflow-wrap: anywhere; }
.learning-row__summary { display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; color: var(--home-secondary); font-size: 14px; line-height: 1.8; }
.learning-row__meta { display: flex; flex-wrap: wrap; gap: 5px 12px; color: var(--home-muted); font-size: 13px; line-height: 1.7; }
.learning-row__meta small { font-size: inherit; }
.learning-path { display: block; width: min(200px,100%); height: 200px; margin: 18px auto; stroke: var(--home-link); stroke-width: 1.2; opacity: .65; }
.home-recent-grid { display: grid; }
.home-recent-row { display: grid; grid-template-columns: 26px minmax(0,1fr) 12px; align-items: start; gap: 12px; min-width: 0; padding: 14px 0; border-bottom: 1px solid var(--home-edge); text-decoration: none; color: var(--home-ink); }
.home-recent-row:last-child { border-bottom: 0; padding-bottom: 4px; }
.home-recent-cover, .home-recent-icon { display: grid; place-items: center; width: 26px; height: 32px; border-radius: 5px; overflow: hidden; color: var(--home-secondary); }
.home-recent-cover img { width: 100%; height: 100%; object-fit: cover; }
.home-recent-icon svg { width: 20px; height: 20px; }
.home-recent-copy { display: grid; gap: 7px; min-width: 0; }
.home-recent-copy strong { font-size: 15px; line-height: 1.7; font-weight: 500; overflow-wrap: anywhere; }
.home-recent-copy small { display: flex; flex-wrap: wrap; gap: 4px 12px; font-size: 13px; line-height: 1.65; color: var(--home-muted); }
.home-recent-date { font-variant-numeric: tabular-nums; }
.home-muted { color: var(--home-muted); font-size: 13px; }
.home-recent-row:hover strong { color: var(--home-link); }
.module-error { display: flex; align-items: center; flex-wrap: wrap; gap: 8px 12px; font-size: 14px; line-height: 1.8; color: var(--home-secondary); padding: 12px 0; }
.inline-action { padding: 0; border: 0; background: transparent; color: var(--home-link); font: inherit; cursor: pointer; }
.workspace-home__refresh-error { display: flex; flex-wrap: wrap; gap: 8px; color: var(--color-warning); font-size: 14px; margin: -16px 0 24px; }
.workspace-home__fatal-error { max-width: 560px; margin: 8vh auto; }
.workspace-home__grid--loading .focus-card-frame { grid-column: 2; grid-row: 1; }
.workspace-home__grid--loading .projects-section { grid-column: 1; grid-row: 1; }
.workspace-home__grid--loading .today-section { grid-column: 1; grid-row: 2; }
.workspace-home__grid--loading .learning-section { grid-column: 2; grid-row: 2; }
.home-section--skeleton { min-height: 230px; }
.home-shortcuts { grid-column: 1 / -1; grid-row: 2; display: flex; align-items: center; flex-wrap: wrap; gap: 12px 24px; padding: 10px 0; font-size: 12px; }
.home-shortcuts__label { color: var(--home-muted); }
.home-shortcuts a { color: var(--home-secondary); text-decoration: none; line-height: 1.8; }
.home-shortcuts small { color: var(--home-muted); font-size: 12px; }
.home-shortcuts a:hover { color: var(--home-link); text-decoration: underline; text-underline-offset: 4px; }
.home-search { position: relative; display: flex; align-items: center; gap: 8px; width: 158px; height: 38px; padding: 0 12px; border: 1px solid var(--home-edge); border-radius: 999px; background: var(--home-bar); color: var(--home-muted); }
.home-search > svg { flex: none; width: 16px; height: 16px; }
.home-search input { min-width: 0; width: 100%; padding: 0; border: 0; outline: none; background: transparent; font: inherit; font-size: 13px; color: var(--home-ink); }
.home-search input::placeholder { color: var(--home-muted); }
.home-search:focus-within { outline: 2px solid var(--home-link); outline-offset: 2px; }
.home-search__results { position: absolute; top: 46px; right: 0; width: 360px; max-width: calc(100vw - 32px); max-height: 420px; overflow-y: auto; padding: 8px; background: var(--home-canvas); border: 1px solid var(--home-edge); border-radius: 12px; box-shadow: 0 12px 30px rgba(0,0,0,.18); color: var(--home-ink); }
.home-search__results ul { list-style: none; padding: 0; margin: 0; }
.home-search__results button { display: flex; gap: 12px; justify-content: space-between; align-items: start; width: 100%; min-height: 44px; padding: 10px; border: 0; border-radius: 6px; background: transparent; text-align: left; color: var(--home-ink); font: inherit; font-size: 14px; cursor: pointer; }
.home-search__results button:hover, .home-search__results button:focus-visible { background: var(--home-tint); }
.home-search__results button > span { min-width: 0; overflow-wrap: anywhere; }
.home-search__results small { flex: none; font-size: 11px; color: var(--home-muted); }
.home-search__results p { margin: 8px; font-size: 13px; }
.home-capture-button.el-button { height: 38px; padding-inline: 16px; border: 1px solid var(--home-edge); border-radius: 999px; background: var(--home-tint); color: var(--home-ink); font-size: 13px; }
.home-capture-button :deep(svg) { width: 16px; height: 16px; margin-right: 7px; }
@media (min-width: 701px) {
  .learning-row__text { display: grid; grid-template-columns: minmax(0,1fr) 130px; gap: 12px; }
  .learning-row__text > * { grid-column: 1 / -1; }
  .learning-row__text > .learning-row__summary, .learning-row__text > .learning-row__meta { grid-column: 1; }
  .learning-row__text > .learning-path { grid-column: 2; grid-row: 3 / span 2; width: 130px; height: 110px; margin: 0; align-self: center; }
}
@media (max-width: 1099px) and (min-width: 701px) {
  .workspace-home__grid { grid-template-columns: repeat(2,minmax(0,1fr)); }
  .home-focus-lane { grid-column: 1; }
  .home-project-lane { grid-column: 2; }
  .recent-section { grid-column: 1 / -1; grid-row: 2; }
  .home-shortcuts { grid-row: 3; }
  .workspace-home__grid--loading .focus-card-frame { grid-column: 1; }
  .workspace-home__grid--loading .projects-section, .workspace-home__grid--loading .today-section { grid-column: 2; }
  .workspace-home__grid--loading .learning-section { grid-column: 1; grid-row: 2; }
}
@media (max-width: 700px) {
  .workspace-home :deep(.page-header) { margin-bottom: 30px; }
  .workspace-home :deep(.page-header h1) { font-size: 28px; line-height: 1.5; }
  .workspace-home :deep(.page-header__description) { font-size: 13px; margin-top: 10px; }
  .workspace-home__grid { display: flex; flex-direction: column; gap: 22px; }
  .home-focus-lane, .home-project-lane { display: contents; }
  .focus-cluster { order: 1; }
  .focus-card-frame { order: 1; }
  .today-section { order: 2; }
  .learning-section { order: 3; }
  .projects-section { order: 4; }
  .recent-section { order: 5; }
  .home-shortcuts { order: 6; gap: 10px 18px; }
  .continue-focus { order: 2; margin-top: 0; padding-top: 10px; }
  .continue-focus__link { display: inline-flex; align-items: center; min-height: 44px; }
  .focus-card { min-height: 0; padding: 24px; gap: 24px; }
  .focus-card h2 { font-size: 26px; }
  .home-section { padding: 24px; }
  .home-section__icon { left: 24px; top: 27px; }
  .home-section :deep(.section-header) { margin-bottom: 22px; }
  .home-section :deep(.section-header h2) { font-size: 17px; }
  .learning-row__text strong { font-size: 21px; }
  .learning-path { width: 160px; height: 150px; margin-block: 10px; }
  .home-shortcuts__label { width: 100%; }
  .home-search { display: none; }
  .home-capture-button.el-button { min-height: 44px; padding-inline: 12px; font-size: 12px; }
  .home-capture-button :deep(svg) { width: 14px; height: 14px; margin-right: 5px; }
}
@media (prefers-reduced-motion: reduce) { .focus-card, .home-section { transition: none; } }
</style>
