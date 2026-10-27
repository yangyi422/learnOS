<template>
  <PageShell class="workspace-home" variant="wide">
    <PageHeader :title="pageTitle" :description="dateLabel">
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
      <div class="home-left-lane">
        <section class="home-section home-section--skeleton today-section" aria-label="今日计划正在加载"><el-skeleton animated :rows="3" /></section>
        <section class="home-section home-section--skeleton projects-section" aria-label="项目正在加载"><el-skeleton animated :rows="3" /></section>
      </div>
      <div ref="focusFrame" class="focus-card-frame">
        <section class="focus-card focus-card--skeleton daily-focus-card--skeleton" aria-label="今日焦点正在加载">
          <el-skeleton animated :rows="3" />
        </section>
      </div>
      <div class="home-right-lane">
        <section class="home-section home-section--skeleton capture-section" aria-label="快速记录正在加载"><el-skeleton animated :rows="1" /></section>
        <section class="home-section home-section--skeleton recent-section" aria-label="最近内容正在加载"><el-skeleton animated :rows="3" /></section>
        <section class="home-section home-section--skeleton learning-section" aria-label="学习位置正在加载"><el-skeleton animated :rows="2" /></section>
      </div>
    </div>

    <div v-else-if="!home && loadError" class="workspace-home__fatal-error" role="alert">
      <EmptyState title="工作区暂时无法加载" :description="loadError" action-label="重试" @action="load" />
    </div>

    <div v-else-if="home" class="workspace-home__grid" :class="{ 'workspace-home__grid--switching': isSwitching }" :aria-busy="loading">
      <span v-if="isSwitching" class="sr-only" role="status">{{ loadError || '正在切换场景…' }}</span>
      <div class="home-left-lane">
        <section ref="todayCard" class="home-section today-section glass-tier glass-tier--secondary" :inert="isSwitching" :style="{ minHeight: sceneGeometry.today ? sceneGeometry.today + 'px' : undefined }" aria-labelledby="today-title">
          <WorkspaceIcon name="calendar" class="home-section__icon" />
          <DailyPlanPanel :planning="home.planning" :date="currentDate" :context="scene" :focus-task-ids="focusTaskIds" :loading="loading || isSwitching" :error="home.errors.planning" @open="openTask" @refresh="load" />
        </section>

        <section ref="projectsCard" class="home-section projects-section glass-tier glass-tier--secondary" :inert="isSwitching" :style="{ minHeight: sceneGeometry.projects ? sceneGeometry.projects + 'px' : undefined }" aria-labelledby="projects-title">
          <WorkspaceIcon name="projects" class="home-section__icon" />
          <SectionHeader :title="projectsSectionTitle" id="projects-title">
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
            :title="projectEmptyTitle"
            :description="projectEmptyDescription"
            action-label="查看项目"
            @action="router.push('/projects?view=list')"
          />
        </section>
      </div>

      <div ref="focusLane" class="home-focus-lane" :style="focusLaneStyle">
      <div ref="focusCluster" class="focus-cluster" :style="focusClusterStyle">
      <div ref="focusFrame" class="focus-card-frame">
        <DailyFocusCard
          v-fx-spotlight="homeActive ? pilotSettings.home.intensity : undefined"
          class="daily-focus-card glass-tier glass-tier--primary glass-hero"
          :class="{ 'is-empty': focusEmpty, 'glass-refract': homeRefraction }"
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
      <div v-if="showSuggestion" class="continue-focus">
        <span class="continue-focus__label">继续上次</span>
        <RouterLink v-if="suggestionTarget" :to="suggestionTarget" class="continue-focus__link" :aria-label="'继续上次：' + suggestionTitle">{{ suggestionTitle }}</RouterLink>
        <strong v-else class="continue-focus__link">{{ suggestionTitle }}</strong>
        <small v-if="suggestionNote" class="continue-focus__note">{{ suggestionNote }}</small>
      </div>
      </div>
        <section v-if="learningVisible" class="home-section learning-section learning-section--compact glass-tier glass-tier--reading" aria-labelledby="learning-title">
          <WorkspaceIcon name="courses" class="home-section__icon" />
          <SectionHeader title="继续学习 · 全局" id="learning-title" />
          <div v-if="home.errors.learning" class="module-error" role="alert">
            <span>{{ home.errors.learning }}</span>
            <button class="inline-action" type="button" @click="load">重试</button>
          </div>
          <div v-else-if="home.learning[0]" class="learning-row">
            <span class="learning-row__text">
              <small class="learning-row__course">{{ home.learning[0].course_name }}</small>
              <strong>{{ home.learning[0].lesson_title || '选择下一段学习' }}</strong>
              <small class="learning-row__summary" :title="home.learning[0].core_question || home.learning[0].unit_title">{{ home.learning[0].core_question || home.learning[0].unit_title || (home.learning[0].has_current_lesson ? '当前学习单元' : '从知识地图选择一个主题') }}</small>
              <span class="learning-row__meta">
                <small class="learning-row__level">{{ levelLabel(home.learning[0].current_level) }} · {{ cognitiveStatusLabel(home.learning[0].cognitive_status) }}</small>
              </span>
            </span>
          </div>
          <EmptyState
            v-else
            title="还没有学习位置"
            description="创建一个学习领域后，这里会记住你最近学到哪里。"
            action-label="开始一个学习领域"
            @action="router.push('/domains/new')"
          />
          <RouterLink v-if="!home.errors.learning && home.learning[0]" :to="learningTarget(home.learning[0])" class="home-card-entry learning-card-entry" :aria-label="(home.learning[0].has_current_lesson ? '继续学习：' : '打开知识地图：') + (home.learning[0].lesson_title || home.learning[0].course_name)"><span aria-hidden="true">↗</span></RouterLink>
        </section>
      </div>

      <div class="home-right-lane">
        <section class="home-section capture-section glass-tier glass-tier--secondary" aria-labelledby="capture-title">
          <WorkspaceIcon name="inbox" class="home-section__icon" />
          <SectionHeader title="快速记录" id="capture-title" />
          <button type="button" class="capture-entry" @click="captureOpen = true">
            <span class="capture-entry__text">记下一个想法，或一条链接</span>
            <span class="capture-entry__meta" aria-hidden="true">Enter 保存</span>
          </button>
        </section>

        <section v-if="recentVisible" class="home-section recent-section glass-tier glass-tier--reading" aria-labelledby="recent-title">
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

      </div>

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
    <EffectsPilotBar page="home" />
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
import EffectsPilotBar from '@/components/effects/EffectsPilotBar.vue'
import { usePilot } from '@/effects/pilot'
import { fxSpotlight } from '@/effects/spotlight'
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

const router = useRouter()
const { homeActive, homeRefraction, settings: pilotSettings } = usePilot()
const vFxSpotlight = fxSpotlight
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
// Scene-specific but restrained: the title describes the working mode, never
// the amount of work. The date stays the same line in every scene.
const pageTitle = computed(() => {
  if (scene.value === 'work') return '今天，专注重要的事'
  if (scene.value === 'life') return '留一点时间给自己'
  return '今天，按自己的节奏'
})
const projectsSectionTitle = computed(() => {
  if (scene.value === 'work') return '正在推进的工作项目'
  if (scene.value === 'life') return '生活里正在做的事'
  return '正在推进'
})
const projectEmptyTitle = computed(() => {
  if (scene.value === 'work') return '还没有进行中的工作项目'
  if (scene.value === 'life') return '还没有个人的项目'
  return '当前场景还没有进行中的项目'
})
const projectEmptyDescription = computed(() => {
  if (scene.value === 'work') return '创建项目并拆出下一步，推进中的工作会集中在这里。'
  if (scene.value === 'life') return '生活不需要被项目填满；想整理时再创建一个。'
  return '创建项目后，正在推进的任务会集中显示在这里。'
})
const suggestion = computed(() => home.value?.focus)
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
const focusTaskIds = computed<Set<number>>(() => {
  const ids = new Set<number>()
  for (const focus of Object.values(focusViews.value)) {
    if (focus?.task_id) ids.add(focus.task_id)
  }
  return ids
})
const focusEmpty = computed(() => {
  if (scene.value === 'all') return !focusViews.value.work && !focusViews.value.life
  return !focusViews.value[scene.value as DailyFocusContext]
})
// The secondary "continue last" row is a resume shortcut, never a disguised
// focus: it disappears when the suggestion is already one of today's focuses.
const showSuggestion = computed(() => {
  const value = suggestion.value
  return value?.kind === 'task' && !!value.task_id && !focusTaskIds.value.has(value.task_id)
})
const suggestionTitle = computed(() => suggestion.value?.title ?? '')
const recentVisible = computed(() => recentItems.value.length > 0 || recentLoading.value || !!recentError.value || recentProblems.value.length > 0)
const learningVisible = computed(() => (home.value?.learning.length ?? 0) > 0 || !!home.value?.errors.learning || (!recentVisible.value && !recentLoading.value && !recentError.value))
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
  if (showSuggestion.value) entries.push({ key: 'suggestion', title: suggestionTitle.value, group: '继续上次', open: openSuggestion })
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
.workspace-context-switch { display:inline-flex; flex-wrap:nowrap; gap:4px; padding:4px; border:1px solid var(--home-edge); border-radius:999px; background:var(--home-panel); box-shadow:var(--home-panel-inset); backdrop-filter:var(--home-glass-filter); }
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
.workspace-home__grid { position: relative; isolation: isolate; display: grid; grid-template-columns: minmax(0,1fr) minmax(0,1.3fr) minmax(0,1fr); align-items: start; gap: 24px; }
.home-left-lane, .home-focus-lane, .home-right-lane { display: flex; flex-direction: column; gap: 22px; min-width: 0; grid-row: 1; }
.home-left-lane { grid-column: 1; }
.home-focus-lane { grid-column: 2; }
.home-right-lane { grid-column: 3; }
.home-right-lane > .home-section { flex: none; }
/* Card + secondary suggestion row form one block so a scene switch that hides
   the suggestion cannot shrink the lane or move later sections upward. */
.focus-cluster { display: flex; flex-direction: column; gap: 22px; min-width: 0; }
.focus-card-frame { position: relative; min-width: 0; }
.focus-card, .home-section { position: relative; min-width: 0; border: 1px solid var(--glass-card-edge); border-radius: 16px; background: linear-gradient(120deg,var(--home-glass-glint),transparent 26%,transparent 76%,var(--home-glass-shade)),linear-gradient(145deg,var(--home-sheen),transparent 42%),var(--glass-card-base); background-clip: padding-box; box-shadow: var(--home-panel-inset); backdrop-filter: var(--home-glass-filter); transition: background-color var(--motion-base) var(--motion-ease-standard), border-color var(--motion-base) var(--motion-ease-standard), box-shadow var(--motion-base) var(--motion-ease-standard); }
.focus-card { min-height: 340px; padding: 26px; display: flex; flex-direction: column; gap: 22px; }
.focus-card.is-empty { min-height: 250px; }
.focus-card { background: linear-gradient(120deg,var(--home-glass-glint),transparent 26%,transparent 76%,var(--home-glass-shade)),linear-gradient(180deg,var(--home-focus-top) 0%,var(--home-focus-top) 52%,var(--home-focus-mid) 82%,var(--home-focus-bottom) 100%),var(--glass-card-base); }
.home-section { padding: 24px; }
.projects-section, .today-section { min-height: 220px; }
.projects-section { min-height: 200px; }
.home-section__icon { position: absolute; top: 26px; left: 24px; width: 21px; height: 21px; color: var(--home-secondary); }
.home-section :deep(.section-header) { padding-left: 30px; margin-bottom: 18px; gap: 10px; flex-direction: row; flex-wrap: wrap; align-items: center; text-align: left; }
.home-section :deep(.section-header__actions) { justify-content: flex-end; }
.home-section :deep(.section-header h2) { font-size: 17px; font-weight: 600; line-height: 1.5; color: var(--home-ink); }
.home-section :deep(.section-header__actions .el-button) { padding: 0; height: auto; font-size: 12px; color: var(--home-muted); }
.home-section :deep(.empty-state) { padding: 14px 0; text-align: left; border: 0; background: transparent; }
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
.today-section :deep(.section-header) { margin-bottom: 12px; }
.capture-entry { display: flex; align-items: center; justify-content: space-between; gap: 12px; width: 100%; min-height: 46px; padding: 10px 14px; border: 1px solid var(--home-edge); border-radius: 12px; background: var(--home-tint); color: var(--home-secondary); font: inherit; font-size: 14px; text-align: left; cursor: pointer; transition: border-color 180ms ease, color 180ms ease; }
.capture-entry:hover { border-color: var(--home-reflection); color: var(--home-link); }
.capture-entry:focus-visible { outline: 2px solid var(--home-link); outline-offset: 3px; }
.capture-entry__text { min-width: 0; overflow-wrap: anywhere; }
.capture-entry__meta { flex: none; font-size: 11px; color: var(--home-muted); }
.learning-row { display: block; width: 100%; min-width: 0; padding: 2px 0 0; color: var(--home-ink); }
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
/* Learning is a global entry, not a second focus: keep it compact in the right rail. */
.learning-section--compact .learning-row { padding-top: 0; }
.learning-section--compact .learning-row__text { display: flex; flex-direction: column; gap: 6px; }
.learning-section--compact .learning-row__text strong { font-size: 18px; line-height: 1.55; font-weight: 600; }
.learning-section--compact .learning-row__course { font-size: 13px; line-height: 1.6; }
.learning-section--compact .learning-row__summary { font-size: 13px; line-height: 1.7; -webkit-line-clamp: 2; }
.learning-section--compact .learning-row__meta { font-size: 12px; }
.home-recent-grid { display: grid; }
.home-recent-row { display: grid; grid-template-columns: 26px minmax(0,1fr) 12px; align-items: start; gap: 12px; min-width: 0; padding: 12px 0; border-bottom: 1px solid var(--home-edge); text-decoration: none; color: var(--home-ink); }
.home-recent-row:last-child { border-bottom: 0; padding-bottom: 2px; }
.home-recent-cover, .home-recent-icon { display: grid; place-items: center; width: 26px; height: 30px; border-radius: 5px; overflow: hidden; color: var(--home-secondary); }
.home-recent-cover img { width: 100%; height: 100%; object-fit: cover; }
.home-recent-icon svg { width: 19px; height: 19px; }
.home-recent-copy { display: grid; gap: 5px; min-width: 0; }
.home-recent-copy strong { font-size: 14px; line-height: 1.65; font-weight: 500; overflow-wrap: anywhere; }
.home-recent-copy small { display: flex; flex-wrap: wrap; gap: 3px 10px; font-size: 12px; line-height: 1.6; color: var(--home-muted); }
.home-recent-date { font-variant-numeric: tabular-nums; }
.home-muted { color: var(--home-muted); font-size: 13px; }
.home-recent-row:hover strong { color: var(--home-link); }
.module-error { display: flex; align-items: center; flex-wrap: wrap; gap: 8px 12px; font-size: 14px; line-height: 1.8; color: var(--home-secondary); padding: 12px 0; }
.inline-action { padding: 0; border: 0; background: transparent; color: var(--home-link); font: inherit; cursor: pointer; }
.workspace-home__refresh-error { display: flex; flex-wrap: wrap; gap: 8px; color: var(--color-warning); font-size: 14px; margin: -16px 0 24px; }
.workspace-home__fatal-error { max-width: 560px; margin: 8vh auto; }
.workspace-home__grid--loading > .focus-card-frame { grid-column: 2; grid-row: 1; }
.home-section--skeleton { min-height: 200px; }
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

/* Home glass tiers: one shared material, three translucency/edge levels. */
.glass-tier--primary {
  --tier-base: rgba(255,255,255,.14);
  --tier-glint: rgba(255,255,255,.16);
  --tier-sheen: rgba(255,255,255,.05);
  --tier-shade: rgba(52,84,120,.05);
  --tier-edge: rgba(74,96,122,.20);
  --tier-highlight: rgba(255,255,255,.12);
  --tier-blur: 12px;
  --tier-sat: 1.1;
  --tier-inset: inset 0 -14px 26px -22px rgba(40,64,90,.08);
  --tier-shadow: 0 2px 4px -1px rgba(40,64,90,.075), 0 12px 26px -20px rgba(40,64,90,.18);
  --tier-shadow-hover: 0 2px 4px rgba(40,64,90,.06), 0 16px 32px -20px rgba(40,64,90,.22);
  --tier-fallback: rgba(247,250,255,.86);
}
.glass-tier--secondary {
  --tier-base: rgba(255,255,255,.10);
  --tier-glint: rgba(255,255,255,.12);
  --tier-sheen: rgba(255,255,255,.04);
  --tier-shade: rgba(52,84,120,.04);
  --tier-edge: rgba(74,96,122,.17);
  --tier-highlight: rgba(255,255,255,.09);
  --tier-blur: 10px;
  --tier-sat: 1.08;
  --tier-inset: inset 0 -10px 22px -20px rgba(40,64,90,.06);
  --tier-shadow: 0 2px 4px -1px rgba(40,64,90,.065), 0 8px 20px -18px rgba(40,64,90,.14);
  --tier-shadow-hover: 0 1px 3px rgba(40,64,90,.05), 0 12px 26px -18px rgba(40,64,90,.18);
  --tier-fallback: rgba(246,250,255,.84);
}
.glass-tier--reading {
  --tier-base: rgba(255,255,255,.26);
  --tier-glint: rgba(255,255,255,.08);
  --tier-sheen: rgba(255,255,255,.03);
  --tier-shade: rgba(52,84,120,.03);
  --tier-edge: rgba(74,96,122,.18);
  --tier-highlight: rgba(255,255,255,.06);
  --tier-blur: 10px;
  --tier-sat: 1.05;
  --tier-inset: inset 0 -10px 20px -18px rgba(40,64,90,.05);
  --tier-shadow: 0 2px 4px -1px rgba(40,64,90,.055), 0 8px 18px -18px rgba(40,64,90,.12);
  --tier-shadow-hover: 0 1px 3px rgba(40,64,90,.04), 0 12px 24px -18px rgba(40,64,90,.16);
  --tier-fallback: rgba(248,251,255,.88);
}
:root[data-theme='dark'] .glass-tier--primary {
  --tier-base: rgba(24,16,54,.30);
  --tier-glint: rgba(198,170,255,.16);
  --tier-sheen: rgba(223,202,255,.10);
  --tier-shade: rgba(0,0,0,.22);
  --tier-edge: rgba(167,142,225,.24);
  --tier-highlight: rgba(224,207,255,.08);
  --tier-blur: 16px;
  --tier-sat: 1.2;
  --tier-inset: inset 0 -18px 30px -24px rgba(0,0,0,.45);
  --tier-shadow: 0 2px 4px rgba(0,0,0,.25), 0 20px 40px -24px rgba(0,0,0,.55);
  --tier-shadow-hover: 0 2px 6px rgba(0,0,0,.28), 0 24px 46px -24px rgba(0,0,0,.60);
  --tier-fallback: rgba(12,8,28,.80);
}
:root[data-theme='dark'] .glass-tier--secondary {
  --tier-base: rgba(24,16,54,.24);
  --tier-glint: rgba(198,170,255,.12);
  --tier-sheen: rgba(223,202,255,.06);
  --tier-shade: rgba(0,0,0,.18);
  --tier-edge: rgba(167,142,225,.20);
  --tier-highlight: rgba(224,207,255,.06);
  --tier-blur: 14px;
  --tier-sat: 1.16;
  --tier-inset: inset 0 -14px 24px -20px rgba(0,0,0,.35);
  --tier-shadow: 0 1px 3px rgba(0,0,0,.16), 0 14px 30px -22px rgba(0,0,0,.45);
  --tier-shadow-hover: 0 1px 4px rgba(0,0,0,.18), 0 18px 34px -22px rgba(0,0,0,.50);
  --tier-fallback: rgba(12,8,28,.78);
}
:root[data-theme='dark'] .glass-tier--reading {
  --tier-base: rgba(24,16,54,.40);
  --tier-glint: rgba(198,170,255,.10);
  --tier-sheen: rgba(223,202,255,.05);
  --tier-shade: rgba(0,0,0,.24);
  --tier-edge: rgba(167,142,225,.22);
  --tier-highlight: rgba(224,207,255,.04);
  --tier-blur: 12px;
  --tier-sat: 1.12;
  --tier-inset: inset 0 -12px 22px -18px rgba(0,0,0,.40);
  --tier-shadow: 0 1px 2px rgba(0,0,0,.14), 0 10px 24px -20px rgba(0,0,0,.40);
  --tier-shadow-hover: 0 1px 3px rgba(0,0,0,.16), 0 14px 28px -20px rgba(0,0,0,.44);
  --tier-fallback: rgba(12,8,28,.84);
}
.glass-tier {
  border: 1px solid var(--tier-edge);
  background:
    linear-gradient(120deg, var(--tier-glint), transparent 26%, transparent 76%, var(--tier-shade)),
    linear-gradient(145deg, var(--tier-sheen), transparent 42%),
    var(--tier-base);
  background-clip: padding-box;
  box-shadow: var(--tier-inset), var(--tier-shadow);
  backdrop-filter: blur(var(--tier-blur)) saturate(var(--tier-sat));
}
.glass-tier::before {
  content: '';
  position: absolute;
  top: 0;
  left: 9%;
  width: 36%;
  height: 1px;
  background: linear-gradient(90deg, transparent, var(--tier-highlight), transparent);
  pointer-events: none;
}
@supports not ((backdrop-filter: blur(1px)) or (-webkit-backdrop-filter: blur(1px))) {
  .glass-tier { background: var(--tier-fallback); }
}

/* Hero Glass: the focus card alone gets a larger, curved specular surface. */
.glass-hero {
  --tier-base: rgba(255,255,255,.12);
  isolation: isolate;
  border-color: rgba(74,96,122,.20);
  background:
    radial-gradient(125% 95% at 84% -20%, rgba(255,255,255,.22), transparent 58%),
    radial-gradient(70% 55% at 6% 4%, rgba(255,255,255,.09), transparent 64%),
    linear-gradient(158deg, rgba(255,255,255,.04), transparent 42%, rgba(52,84,120,.04)),
    var(--tier-base);
  box-shadow:
    0 2px 4px -1px rgba(40,64,90,.075),
    0 18px 38px -26px rgba(40,64,90,.22),
    inset 0 5px 12px -10px rgba(255,255,255,.40),
    inset 0 -20px 32px -28px rgba(40,64,90,.10);
  backdrop-filter: blur(12px) saturate(1.12);
}
.glass-hero::before {
  content: '';
  position: absolute;
  inset: 0;
  z-index: -1;
  width: auto;
  height: auto;
  border-radius: inherit;
  background:
    radial-gradient(62% 38% at 86% 4%, rgba(255,255,255,.17), transparent 70%),
    radial-gradient(42% 28% at 8% 8%, rgba(255,255,255,.08), transparent 72%);
  pointer-events: none;
}
:root[data-theme='dark'] .glass-hero {
  --tier-base: rgba(26,17,58,.34);
  border-color: rgba(167,142,225,.24);
  /* Light comes through the shared horizon behind the card, not a top lamp. */
  background:
    linear-gradient(180deg, rgba(0,0,0,.12), transparent 72%),
    var(--tier-base);
  box-shadow:
    0 2px 6px rgba(0,0,0,.26),
    0 28px 60px -30px rgba(0,0,0,.62),
    inset 0 -26px 38px -28px rgba(0,0,0,.52);
  backdrop-filter: blur(10px) saturate(1.18);
}
:root[data-theme='dark'] .glass-hero::before {
  background: none;
}
.glass-hero.glass-refract {
  --tier-base: rgba(255,255,255,.08);
}
:root[data-theme='dark'] .glass-hero.glass-refract {
  --tier-base: rgba(20,14,44,.18);
}

/* Admin-only refraction pilot: a clearer card so the sampled background shows. */
.glass-refract {
  --tier-base: rgba(255,255,255,.08);
  --tier-glint: rgba(255,255,255,.18);
  --tier-sheen: rgba(255,255,255,.04);
  --tier-highlight: rgba(255,255,255,.20);
  backdrop-filter: blur(8px) saturate(1.15);
}
:root[data-theme='dark'] .glass-refract {
  --tier-base: rgba(20,14,44,.18);
  --tier-glint: rgba(198,170,255,.10);
  --tier-sheen: rgba(223,202,255,.04);
  --tier-highlight: rgba(224,207,255,.07);
}

/* Soft Mist Glass — homepage light only. Dark materials and all layout metrics
   stay intact.
    Why the old cards read as solid boxes: the material was a single opaque-ish
   fill painted on the element itself, so its alpha was identical at the centre
   and at the 1px border — a filled rectangle with a hard edge, no matter how
   low the alpha went. The surface is now a separate feathered layer: the card
   element itself is transparent, and `::before` paints a centre-weighted film
   that reaches transparent before the border box, so the edge dissolves into
   the mist instead of ending on a line. Text, buttons and inputs are never in
   that layer and keep full opacity. */
:root[data-theme='light'] .glass-tier {
  --mist-fill: rgba(250,252,255,.44);
  --mist-light: rgba(255,255,255,.46);
  --mist-edge: rgba(74,96,122,.018);
  isolation: isolate;
  border-color: var(--mist-edge);
  /* No rectangular fill on the element: only the feathered layer carries colour. */
  background: none;
  box-shadow:
    0 1px 2px rgba(44,64,92,.02),
    0 10px 26px -20px rgba(44,64,92,.08);
  backdrop-filter: blur(6px) saturate(1.06);
}
:root[data-theme='light'] .glass-tier--primary {
  --mist-fill: rgba(250,252,255,.56);
  --mist-light: rgba(255,255,255,.58);
  --mist-edge: rgba(74,96,122,.025);
}
:root[data-theme='light'] .glass-tier--reading {
  --mist-fill: rgba(250,252,255,.48);
  --mist-light: rgba(255,255,255,.48);
  --mist-edge: rgba(74,96,122,.02);
}
/* Feathered surface: light gathers at the upper-left of the pane, the film thins
   toward every edge and is already transparent at the border box. */
:root[data-theme='light'] .glass-tier::before {
  content: '';
  position: absolute;
  inset: 0;
  z-index: -1;
  width: auto;
  height: auto;
  border-radius: inherit;
  pointer-events: none;
  background:
    radial-gradient(112% 88% at 28% 2%, var(--mist-light), transparent 60%),
    linear-gradient(180deg, transparent 58%, rgba(226,235,247,.12)),
    radial-gradient(122% 118% at 50% 44%, var(--mist-fill) 36%, rgba(247,251,255,.12) 86%, transparent 100%);
  /* Per-axis feather instead of one radial: a wide card keeps its middle but
     every edge still dissolves by ~8% of its own side, at any aspect ratio. */
  /* Fixed-width feather (~26px) rather than a percentage: the rim dissolves at
     every card size while the surface behind the content keeps full strength. */
  -webkit-mask-image:
    linear-gradient(90deg, transparent, #000 26px, #000 calc(100% - 26px), transparent),
    linear-gradient(180deg, transparent, #000 26px, #000 calc(100% - 26px), transparent);
  mask-image:
    linear-gradient(90deg, transparent, #000 26px, #000 calc(100% - 26px), transparent),
    linear-gradient(180deg, transparent, #000 26px, #000 calc(100% - 26px), transparent);
  -webkit-mask-composite: source-in;
  mask-composite: intersect;
}
:root[data-theme='light'] .glass-tier.fx-spotlight::after { background: none; }
@media (hover: hover) {
  :root[data-theme='light'] .glass-tier:hover {
    --mist-fill: rgba(250,252,255,.50);
    --mist-light: rgba(255,255,255,.52);
    border-color: rgba(74,96,122,.08);
    box-shadow: 0 2px 10px -6px rgba(44,64,92,.10);
  }
}
:root[data-theme='light'] .glass-tier:focus-within {
  border-color: rgba(74,96,122,.22);
  box-shadow: 0 2px 10px -6px rgba(44,64,92,.10);
}
/* Without backdrop blur or masks, keep a readable static pane instead. */
@supports not ((backdrop-filter: blur(1px)) or (-webkit-backdrop-filter: blur(1px))) {
  :root[data-theme='light'] .glass-tier { background: rgba(247,250,254,.78); }
  :root[data-theme='light'] .glass-tier::before { display: none; }
}
@supports not ((mask-image: radial-gradient(#000,#000)) or (-webkit-mask-image: radial-gradient(#000,#000))) {
  :root[data-theme='light'] .glass-tier::before {
    background:
      radial-gradient(112% 88% at 28% 2%, var(--mist-light), transparent 60%),
      linear-gradient(180deg, transparent 58%, rgba(226,235,247,.16)),
      radial-gradient(112% 108% at 50% 44%, var(--mist-fill) 34%, rgba(247,251,255,.12) 80%, transparent 96%);
  }
}

:root[data-theme='light'] .workspace-context-switch {
  border-color: rgba(74,96,122,.14);
  background: rgba(255,255,255,.55);
  box-shadow: 0 1px 2px rgba(40,64,90,.05);
  backdrop-filter: blur(10px) saturate(1.05);
}
:root[data-theme='light'] .workspace-context-switch :deep(.el-radio-button.is-active .el-radio-button__original-radio:not(:disabled) + .el-radio-button__inner) {
  background: rgba(70,83,223,.10);
}
:root[data-theme='light'] .capture-entry {
  border-color: rgba(74,96,122,.14);
  background: rgba(255,255,255,.50);
}
:root[data-theme='light'] .capture-entry:hover {
  border-color: rgba(74,96,122,.22);
  background: rgba(255,255,255,.72);
}

@media (min-width: 701px) {
  .learning-row__text { display: flex; flex-direction: column; gap: 6px; }
}
@media (max-width: 1099px) and (min-width: 701px) {
  .workspace-home__grid { grid-template-columns: repeat(2,minmax(0,1fr)); }
  .home-focus-lane { grid-column: 1; grid-row: 1; }
  .home-left-lane { grid-column: 2; grid-row: 1; }
  .home-right-lane { grid-column: 1 / -1; grid-row: 2; display: grid; grid-template-columns: repeat(auto-fit,minmax(250px,1fr)); gap: 22px; }
  .home-shortcuts { grid-row: 3; }
  .workspace-home__grid--loading > .focus-card-frame { grid-column: 1; }
}
@media (max-width: 700px) {
  .workspace-home :deep(.page-header) { margin-bottom: 28px; }
  .workspace-home :deep(.page-header h1) { font-size: 28px; line-height: 1.5; }
  .workspace-home :deep(.page-header__description) { font-size: 13px; margin-top: 10px; }
  .workspace-home__grid { display: flex; flex-direction: column; gap: 18px; }
  .home-left-lane, .home-focus-lane, .home-right-lane { display: contents; }
  .focus-cluster { order: 1; }
  .today-section { order: 2; }
  .projects-section { order: 3; }
  .capture-section { order: 4; }
  .recent-section { order: 5; }
  .learning-section { order: 6; }
  .home-shortcuts { order: 7; gap: 10px 18px; }
  .continue-focus { order: 2; margin-top: 0; padding-top: 10px; }
  .continue-focus__link { display: inline-flex; align-items: center; min-height: 44px; }
  .focus-card { min-height: 0; padding: 22px; gap: 20px; }
  .focus-card.is-empty { min-height: 0; }
  .home-section { padding: 20px; }
  .home-section__icon { left: 20px; top: 23px; }
  .home-section :deep(.section-header) { margin-bottom: 18px; }
  .home-section :deep(.section-header h2) { font-size: 16px; }
  .capture-entry { min-height: 48px; }
  .capture-entry__meta { display: none; }
  .learning-section--compact .learning-row__text strong { font-size: 18px; }
  .home-shortcuts__label { width: 100%; }
  .home-search { display: none; }
  .home-capture-button.el-button { min-height: 44px; padding-inline: 12px; font-size: 12px; }
  .home-capture-button :deep(svg) { width: 14px; height: 14px; margin-right: 5px; }
}
@media (prefers-reduced-motion: reduce) { .focus-card, .home-section { transition: none; } }
</style>
