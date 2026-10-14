<template>
  <PageShell class="workspace-home" variant="standard">
    <PageHeader title="今天继续什么？" :description="dateLabel">
      <template #actions>
        <el-button type="primary" @click="captureOpen = true">快速记录</el-button>
      </template>
    </PageHeader>

    <p v-if="loadError && home" class="workspace-home__refresh-error" role="status">
      工作区更新失败，仍显示上次内容。{{ loadError }}
      <button class="inline-action" type="button" @click="load">重试</button>
    </p>

    <div v-if="loading && !home" class="workspace-home__grid workspace-home__grid--loading" aria-label="正在整理工作区" aria-busy="true">
      <section class="focus-card focus-card--skeleton" aria-label="Current Focus 正在加载">
        <el-skeleton animated :rows="3" />
      </section>
      <section v-for="section in loadingSections" :key="section" class="home-section home-section--skeleton" :class="sectionClass(section)">
        <SectionHeader :title="sectionTitle(section)" />
        <el-skeleton animated :rows="3" />
      </section>
    </div>

    <div v-else-if="!home && loadError" class="workspace-home__fatal-error" role="alert">
      <EmptyState title="工作区暂时无法加载" :description="loadError" action-label="重试" @action="load" />
    </div>

    <div v-else-if="home" class="workspace-home__grid" :aria-busy="loading">
      <section class="focus-card" aria-labelledby="focus-title">
        <div class="focus-card__copy">
          <span class="section-kicker">CURRENT FOCUS</span>
          <template v-if="home.focus.kind === 'task'">
            <p class="focus-card__context">
              <span>{{ home.focus.project_title || '项目任务' }}</span>
              <span>{{ taskStatusLabel(home.focus.status) }}</span>
              <span v-if="home.focus.priority === 'high'">高优先级</span>
              <span v-if="focusDueLabel">{{ focusDueLabel }}</span>
            </p>
            <h2 id="focus-title">{{ home.focus.title }}</h2>
            <p class="focus-card__reason">
              <span>{{ focusReason }}</span>
              <span v-if="focusUpdatedAt">最近更新 {{ relativeTime(focusUpdatedAt) }}</span>
            </p>
          </template>
          <template v-else-if="home.focus.kind === 'learning'">
            <p class="focus-card__context">
              <span>{{ home.focus.course_name }}</span>
              <span v-if="home.focus.unit_title">{{ home.focus.unit_title }}</span>
            </p>
            <h2 id="focus-title">{{ home.focus.title }}</h2>
            <p v-if="focusLearning?.core_question" class="focus-card__question">{{ focusLearning.core_question }}</p>
            <p class="focus-card__reason">
              <span>最近的学习位置</span>
              <span v-if="focusLearning?.last_learning_at">最近学习 {{ relativeTime(focusLearning.last_learning_at) }}</span>
              <span v-else-if="focusLearning?.updated_at">领域更新 {{ relativeTime(focusLearning.updated_at) }}</span>
              <span v-if="home.focus.current_level">{{ levelLabel(home.focus.current_level) }} · {{ cognitiveStatusLabel(home.focus.cognitive_status) }}</span>
            </p>
          </template>
          <template v-else>
            <h2 id="focus-title">暂时没有需要继续的事项</h2>
            <p class="focus-card__reason">开始任务或学习后，这里会根据真实记录显示最近的下一步。</p>
          </template>
        </div>
        <el-button v-if="home.focus.kind !== 'empty'" type="primary" @click="continueFocus">
          {{ home.focus.kind === 'task' ? '继续处理' : home.focus.lesson_id ? '继续学习' : '查看知识地图' }}
          <span aria-hidden="true"> →</span>
        </el-button>
      </section>

      <section class="home-section today-section" aria-labelledby="today-title">
        <SectionHeader title="Today" id="today-title">
          <template #actions>
            <el-button text @click="router.push({ path: '/projects', query: { view: 'today' } })">查看全部 →</el-button>
          </template>
        </SectionHeader>
        <div v-if="home.errors.today" class="module-error" role="alert">
          <span>{{ home.errors.today }}</span>
          <button class="inline-action" type="button" @click="load">重试</button>
        </div>
        <template v-else>
          <div class="today-summary" aria-label="今日任务数量">
            <div><strong>{{ home.today?.doing.length ?? 0 }}</strong><span>进行中</span></div>
            <div><strong>{{ home.today?.due.length ?? 0 }}</strong><span>到期 / 逾期</span></div>
            <div><strong>{{ home.today?.next.length ?? 0 }}</strong><span>下一步</span></div>
          </div>
          <div v-if="todayPreview.length" class="today-task-list">
            <button v-for="task in todayPreview" :key="task.id" class="home-task" type="button" @click="openTask(task)">
              <span class="home-task__dot" :class="todayStateClass(task)" aria-hidden="true" />
              <span class="home-task__title">{{ task.title }}</span>
              <small :class="todayStateClass(task)">{{ todayStateLabel(task) }}</small>
            </button>
          </div>
          <EmptyState v-else title="今天没有优先事项" description="可以从项目里挑一个下一步，或继续最近的学习。" />
        </template>
      </section>

      <section class="home-section projects-section" aria-labelledby="projects-title">
        <SectionHeader title="Active Projects" id="projects-title">
          <template #actions>
            <el-button text @click="router.push('/projects?view=list')">项目列表 →</el-button>
          </template>
        </SectionHeader>
        <div v-if="home.errors.projects" class="module-error" role="alert">
          <span>{{ home.errors.projects }}</span>
          <button class="inline-action" type="button" @click="load">重试</button>
        </div>
        <template v-else-if="home.projects.length">
          <button
            v-for="project in home.projects.slice(0, 5)"
            :key="project.id"
            class="project-row"
            type="button"
            :aria-label="project.title + '，打开项目看板'"
            @click="openProject(project.id)"
          >
            <span class="project-row__accent" :style="{ color: project.accent || undefined, borderColor: project.accent || undefined }">{{ project.icon || '·' }}</span>
            <span class="project-row__text">
              <strong>{{ project.title }}</strong>
              <span class="project-row__counts">
                <span>{{ project.doing_task_count }} 进行中</span>
                <span>{{ project.next_task_count }} 下一步</span>
                <span>{{ project.open_task_count }} 未完成</span>
              </span>
            </span>
            <span class="project-row__arrow" aria-hidden="true">→</span>
          </button>
        </template>
        <EmptyState
          v-else
          title="还没有进行中的项目"
          description="创建项目后，正在推进的任务会集中显示在这里。"
          action-label="查看项目"
          @action="router.push('/projects?view=list')"
        />
      </section>

      <section class="home-section learning-section" aria-labelledby="learning-title">
        <SectionHeader title="Continue Learning" id="learning-title">
          <template #actions>
            <el-button text @click="router.push('/learn')">学习空间 →</el-button>
          </template>
        </SectionHeader>
        <div v-if="home.errors.learning" class="module-error" role="alert">
          <span>{{ home.errors.learning }}</span>
          <button class="inline-action" type="button" @click="load">重试</button>
        </div>
        <button v-else-if="home.learning[0]" class="learning-row" type="button" @click="openLearning(home.learning[0])">
          <span class="learning-row__mark" aria-hidden="true">{{ home.learning[0].has_current_lesson ? '→' : '＋' }}</span>
          <span class="learning-row__text">
            <small class="learning-row__course">{{ home.learning[0].course_name }}</small>
            <strong>{{ home.learning[0].lesson_title || '选择下一段学习' }}</strong>
            <small>{{ home.learning[0].core_question || home.learning[0].unit_title || (home.learning[0].has_current_lesson ? '当前学习单元' : '从知识地图选择一个主题') }}</small>
            <small v-if="home.learning[0].core_question && home.learning[0].unit_title">{{ home.learning[0].unit_title }}</small>
            <small class="learning-row__level">{{ levelLabel(home.learning[0].current_level) }} · {{ cognitiveStatusLabel(home.learning[0].cognitive_status) }}</small>
          </span>
          <span class="learning-row__action">{{ home.learning[0].has_current_lesson ? '继续学习' : '打开地图' }} <span aria-hidden="true">→</span></span>
        </button>
        <EmptyState
          v-else
          title="知识世界还是空的"
          description="创建一个学习领域后，这里会保存你最近的学习位置。"
          action-label="开始一个学习领域"
          @action="router.push('/domains/new')"
        />
      </section>

      <section v-if="recentItems.length || recentLoading || recentError || recentProblems.length" class="home-section recent-section" aria-labelledby="recent-title">
        <SectionHeader title="最近留下的内容" id="recent-title" />
        <p v-if="recentLoading && !recent" class="home-muted" role="status">正在读取最近记录…</p>
        <RouterLink v-for="item in recentItems" :key="item.key" :to="item.to" class="home-recent-row">
          <span v-if="item.cover" class="home-recent-cover"><img :src="item.cover" alt="" loading="lazy" decoding="async" referrerpolicy="no-referrer" @error="hideFailedImage" /></span>
          <span class="home-recent-copy"><strong>{{ item.title }}</strong><small>{{ item.source }} · {{ item.dateLabel }}</small></span>
          <span class="home-muted" aria-hidden="true">→</span>
        </RouterLink>
        <p v-if="recentError || recentProblems.length" class="module-error" role="status">
          <span>{{ recentError || recentProblems.join(' ') }}</span>
          <button class="inline-action" type="button" @click="loadRecent">重试最近内容</button>
        </p>
      </section>
      <nav class="home-shortcuts" aria-label="个人档案快捷入口">
        <RouterLink to="/inbox">收集箱 · {{ home.errors.inbox ? '数量暂不可用' : (home.inbox?.count ?? 0) + ' 条待整理' }}</RouterLink>
        <button v-if="home.errors.inbox" class="inline-action" type="button" @click="load">重试收集箱</button>
        <RouterLink to="/explore/library">我的资料库</RouterLink>
        <RouterLink to="/life">生活档案</RouterLink>
        <RouterLink to="/life/finance">财务档案<small v-if="recent?.asset_date"> · 最近快照 {{ formatShortDate(recent.asset_date) }}</small></RouterLink>
      </nav>
    </div>

    <QuickCaptureDialog v-model="captureOpen" @created="captured" />
  </PageShell>
</template>

<script setup lang="ts">
import PageShell from '@/components/PageShell.vue'
import { computed, onMounted, onBeforeUnmount, ref } from 'vue'
import { useRouter } from 'vue-router'
import { APIRequestError } from '@/api/http'
import { type InboxItem } from '@/api/inbox'
import type { ProjectTask } from '@/api/projects'
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
const home = ref<WorkspaceHome | null>(null)
const loading = ref(true)
const loadError = ref('')
const captureOpen = ref(false)
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
const dateLabel = new Intl.DateTimeFormat('zh-CN', { year: 'numeric', month: 'long', day: 'numeric', weekday: 'long' }).format(new Date())
const todayPreview = computed(() => {
  const view = home.value?.today
  if (!view) return []
  return [...view.doing, ...view.due, ...view.next]
    .sort((left, right) => {
      const leftUrgency = todayUrgency(left)
      const rightUrgency = todayUrgency(right)
      if (leftUrgency !== rightUrgency) return leftUrgency - rightUrgency
      if (leftUrgency === 0) return dueDays(right) - dueDays(left)
      const priorityDifference = priorityRank(left.priority) - priorityRank(right.priority)
      if (priorityDifference !== 0) return priorityDifference
      const leftDue = left.due_date ?? '9999-12-31'
      const rightDue = right.due_date ?? '9999-12-31'
      if (leftDue !== rightDue) return leftDue.localeCompare(rightDue)
      return right.updated_at.localeCompare(left.updated_at)
    })
    .slice(0, 3)
})
const focusTask = computed(() => {
  const focus = home.value?.focus
  if (focus?.kind !== 'task' || !focus.task_id) return undefined
  const today = home.value?.today
  return [...(today?.doing ?? []), ...(today?.due ?? []), ...(today?.next ?? [])].find(task => task.id === focus.task_id)
})
const focusLearning = computed(() => {
  const focus = home.value?.focus
  return focus?.kind === 'learning' ? home.value?.learning.find(item => item.course_id === focus.course_id) : undefined
})
const focusUpdatedAt = computed(() => focusTask.value?.updated_at ?? focusLearning.value?.last_learning_at ?? focusLearning.value?.updated_at ?? '')
const focusDueLabel = computed(() => {
  const dueDate = home.value?.focus.due_date
  if (!dueDate) return ''
  const days = dateDifference(dueDate, home.value?.date ?? '')
  if (days > 0) return '逾期 ' + days + ' 天'
  if (days === 0) return '今天到期'
  return '到期 ' + formatShortDate(dueDate)
})
const focusReason = computed(() => {
  const focus = home.value?.focus
  if (!focus) return ''
  if (focus.kind === 'learning') return '最近的学习位置'
  if (focus.kind !== 'task') return ''
  if (focus.status === 'doing') return '当前正在推进'
  if (focus.due_date) {
    const days = dateDifference(focus.due_date, home.value?.date ?? '')
    if (days > 0) return '已到期，需要处理'
    if (days === 0) return '今天到期'
  }
  if (focus.status === 'next' && focus.priority === 'high') return '当前优先级最高的下一步'
  return '当前可继续处理的任务'
})

function localDate() {
  const date = new Date()
  return date.getFullYear() + '-' + String(date.getMonth() + 1).padStart(2, '0') + '-' + String(date.getDate()).padStart(2, '0')
}
function sectionClass(section: HomeSection) { return section + '-section' }
function sectionTitle(section: HomeSection) {
  return ({ today: 'Today', projects: 'Active Projects', learning: 'Continue Learning' })[section]
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
function dateOrdinal(value: string) {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value)
  if (!match) return Number.NaN
  return Date.UTC(Number(match[1]), Number(match[2]) - 1, Number(match[3]))
}
function dateDifference(from: string, to: string) {
  const fromOrdinal = dateOrdinal(from)
  const toOrdinal = dateOrdinal(to)
  return Number.isFinite(fromOrdinal) && Number.isFinite(toOrdinal) ? Math.round((toOrdinal - fromOrdinal) / 86_400_000) : 0
}
function dueDays(task: ProjectTask) {
  return task.due_date ? Math.max(0, dateDifference(task.due_date, home.value?.date ?? '')) : 0
}
function todayUrgency(task: ProjectTask) {
  const days = dueDays(task)
  if (days > 0) return 0
  if (task.due_date && days === 0 && task.due_date <= (home.value?.date ?? '')) return 1
  if (task.status === 'doing') return 2
  return 3
}
function priorityRank(priority?: string) {
  return ({ high: 0, normal: 1, low: 2 } as Record<string, number>)[priority ?? ''] ?? 3
}
function todayStateLabel(task: ProjectTask) {
  const days = dueDays(task)
  if (days > 0) return '逾期 ' + days + ' 天'
  if (task.due_date && task.due_date === home.value?.date) return '今天到期'
  if (task.status === 'doing') return '进行中'
  return taskStatusLabel(task.status)
}
function todayStateClass(task: ProjectTask) {
  const days = dueDays(task)
  if (days > 0) return 'is-overdue'
  if (task.due_date && task.due_date === home.value?.date) return 'is-due-today'
  if (task.status === 'doing') return 'is-doing'
  return 'is-next'
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
    const result = await getWorkspaceHome(localDate(), controller.signal)
    if (!disposed && controller === homeRequest) home.value = result
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
function continueFocus() {
  const focus = home.value?.focus
  if (!focus) return
  if (focus.kind === 'task' && focus.task_id) {
    router.push({ path: '/projects', query: { project: String(focus.project_id ?? 'all'), view: 'board', task: String(focus.task_id) } })
  } else if (focus.kind === 'learning' && focus.course_id) {
    router.push(focus.lesson_id ? '/courses/' + focus.course_id + '/learn' : '/courses/' + focus.course_id + '/map')
  }
}
function openLearning(item: WorkspaceLearningItem) {
  router.push(item.has_current_lesson ? '/courses/' + item.course_id + '/learn' : '/courses/' + item.course_id + '/map')
}
onMounted(() => { void load(); void loadRecent() })
onBeforeUnmount(() => { disposed = true; homeRequest?.abort(); recentRequest?.abort() })
</script>

<style scoped>

.workspace-home__refresh-error { display: flex; align-items: center; gap: 8px; margin: -8px 0 14px; color: var(--color-warning); font-size: 13px; }
.workspace-home__fatal-error { max-width: 560px; margin: 8vh auto 0; }
.workspace-home__grid { display: grid; grid-template-columns: repeat(10, minmax(0, 1fr)); gap: 18px 24px; margin-top: 4px; }
.focus-card { grid-column: span 7; display: flex; align-items: center; justify-content: space-between; gap: 24px; min-width: 0; padding: 23px 26px; border: 1px solid var(--border-default); border-radius: var(--radius-card); background: linear-gradient(115deg, var(--color-primary-soft), var(--bg-surface) 54%); box-shadow: var(--shadow-card); }
.focus-card__copy { min-width: 0; }
.section-kicker { color: var(--text-tertiary); font-size: 11px; font-weight: 700; letter-spacing: .11em; }
.focus-card__context { display: flex; flex-wrap: wrap; gap: 5px 10px; margin: 8px 0 0; color: var(--text-secondary); font-size: 13px; }
.focus-card__context span + span::before { margin-right: 9px; color: var(--text-tertiary); content: '·'; }
.focus-card h2 { margin: 8px 0 6px; font-size: clamp(21px, 2vw, 27px); line-height: 1.32; overflow-wrap: anywhere; }
.focus-card__question { max-width: 56em; margin: 0 0 8px; color: var(--text-secondary); font-size: 14px; line-height: 1.5; }
.focus-card__reason { display: flex; flex-wrap: wrap; gap: 6px 14px; margin: 0; color: var(--text-secondary); font-size: 13px; }
.focus-card__reason span + span { color: var(--text-tertiary); }
.focus-card > :deep(.el-button) { flex: none; }
.focus-card--skeleton { min-height: 142px; align-items: center; }
.focus-card--skeleton :deep(.el-skeleton) { width: 72%; }
.today-section { grid-column: span 3; }
.projects-section, .learning-section { grid-column: span 5; }
.recent-section { grid-column: span 10; }
.home-section { min-width: 0; padding: 2px 0 8px; }
.home-section :deep(.section-header) { margin-bottom: 10px; }
.home-section :deep(.section-header h2) { font-size: 16px; }
.home-section :deep(.empty-state) { padding: 15px 10px 10px; }
.home-section :deep(.empty-state__mark) { display: none; }
.home-section :deep(.empty-state h3) { margin: 0 0 5px; font-size: 14px; }
.home-section :deep(.empty-state p) { max-width: 34em; margin: 0 auto 10px; font-size: 13px; }
.home-section--skeleton { min-height: 146px; }
.today-summary { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); padding: 8px 0 10px; }
.today-summary div { display: grid; gap: 2px; }
.today-summary strong { font-size: 21px; font-weight: 650; font-variant-numeric: tabular-nums; }
.today-summary span, .home-muted { color: var(--text-tertiary); font-size: 13px; }
.today-task-list { border-top: 1px solid var(--border-subtle); }
.home-task, .project-row, .learning-row { display: flex; align-items: center; width: 100%; min-width: 0; text-align: left; border: 0; border-bottom: 1px solid var(--border-subtle); background: transparent; color: var(--text-primary); transition: background-color 140ms ease, color 140ms ease; }
.home-task { gap: 9px; padding: 10px 2px; cursor: pointer; }
.home-task:hover, .project-row:hover, .learning-row:hover { background: var(--bg-subtle); }
.home-task__dot { width: 7px; height: 7px; flex: none; border-radius: 50%; background: var(--color-primary); }
.home-task__dot.is-overdue { background: var(--color-danger); }
.home-task__dot.is-due-today { background: var(--color-warning); }
.home-task__dot.is-doing { background: var(--color-primary); }
.home-task__dot.is-next { background: var(--text-tertiary); }
.home-task__title { min-width: 0; flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 14px; }
.home-task small { flex: none; color: var(--text-tertiary); font-size: 12px; }
.home-task small.is-overdue { color: var(--color-danger); }
.home-task small.is-due-today { color: var(--color-warning); }
.home-task small.is-doing { color: var(--text-secondary); }
.module-error { display: flex; align-items: center; flex-wrap: wrap; gap: 6px 10px; padding: 12px 2px; color: var(--text-secondary); font-size: 13px; }
.inline-action { padding: 0; border: 0; background: transparent; color: var(--color-primary); font: inherit; cursor: pointer; }
.project-row { gap: 11px; padding: 11px 3px; cursor: pointer; }
.project-row__accent { display: grid; width: 32px; height: 32px; flex: none; place-items: center; border: 1px solid var(--border-subtle); border-radius: 9px; background: var(--bg-subtle); font-size: 15px; }
.project-row__text, .learning-row__text { display: grid; min-width: 0; flex: 1; gap: 4px; }
.project-row__text strong, .learning-row__text strong { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 14px; font-weight: 600; }
.project-row__counts { display: flex; flex-wrap: wrap; gap: 5px 12px; color: var(--text-tertiary); font-size: 12px; }
.project-row__arrow { color: var(--text-tertiary); transition: transform 140ms ease, color 140ms ease; }
.project-row:hover .project-row__arrow { transform: translateX(2px); color: var(--color-primary); }
.learning-row { gap: 12px; padding: 13px 3px; cursor: pointer; }
.learning-row__mark { display: grid; width: 34px; height: 34px; flex: none; place-items: center; border: 1px solid var(--border-subtle); border-radius: 50%; color: var(--color-primary); }
.learning-row__text { gap: 4px; }
.learning-row__text .learning-row__course { color: var(--text-secondary); font-size: 13px; }
.learning-row__text > small { color: var(--text-tertiary); font-size: 12px; }
.learning-row__text .learning-row__level { color: var(--text-secondary); }
.learning-row__action { display: inline-flex; flex: none; align-items: center; gap: 4px; color: var(--color-primary); font-size: 13px; }
.workspace-home__grid--loading { grid-template-rows: auto auto auto; }
.workspace-home__grid--loading .focus-card { grid-row: span 1; }
.workspace-home__grid--loading .today-section { grid-column: span 3; }
.workspace-home__grid--loading .projects-section, .workspace-home__grid--loading .learning-section { grid-column: span 5; }
.workspace-home__grid--loading .recent-section { grid-column: span 10; }
@media (max-width: 1020px) {
  .workspace-home__grid { grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 14px 20px; }
  .focus-card, .today-section { grid-column: span 2; }
  .projects-section, .learning-section { grid-column: span 1; }
  .recent-section { grid-column: span 2; }
  .workspace-home__grid--loading .focus-card, .workspace-home__grid--loading .today-section { grid-column: span 2; }
  .workspace-home__grid--loading .projects-section, .workspace-home__grid--loading .learning-section { grid-column: span 1; }
  .workspace-home__grid--loading .recent-section { grid-column: span 2; }
}
@media (max-width: 760px) {
  .workspace-home :deep(.page-header) { align-items: flex-start; }
  .workspace-home__grid { grid-template-columns: 1fr; gap: 12px; margin-top: 2px; }
  .focus-card { align-items: flex-start; flex-direction: column; padding: 20px; }
  .focus-card > :deep(.el-button) { align-self: flex-start; }
  .focus-card, .today-section, .projects-section, .learning-section, .recent-section,
  .workspace-home__grid--loading .focus-card, .workspace-home__grid--loading .today-section,
  .workspace-home__grid--loading .projects-section, .workspace-home__grid--loading .learning-section,
  .workspace-home__grid--loading .recent-section { grid-column: span 1; }
  .today-section { order: 2; }
  .learning-section { order: 3; }
  .projects-section { order: 4; }
  .recent-section { order: 5; }
  .home-task { align-items: flex-start; }
  .home-task small { max-width: 84px; text-align: right; }
  .project-row__counts { gap: 4px 8px; }
  .learning-row { align-items: flex-start; }
  .learning-row__action { margin-left: auto; padding-top: 7px; }
}
@media (prefers-reduced-motion: reduce) {
  .home-task, .project-row, .learning-row, .project-row__arrow { transition: none; }
}
.home-shortcuts { grid-column: 1 / -1; display: flex; flex-wrap: wrap; gap: 12px 24px; padding: 12px 0; border-top: 1px solid var(--border-subtle); order: 6; }
.home-shortcuts a { color: var(--text-secondary); text-decoration: none; font-size: 13px; line-height: 1.7; }
.home-shortcuts a:hover { color: var(--color-primary); }
.home-shortcuts small { font-size: 12px; color: var(--text-tertiary); }
.home-recent-row { display: flex; align-items: center; gap: 12px; padding: 12px 2px; min-width: 0; text-decoration: none; border-bottom: 1px solid var(--border-subtle); color: var(--text-primary); }
.home-recent-row:hover { background: var(--bg-subtle); }
.home-recent-cover { width: 48px; height: 48px; flex: none; overflow: hidden; border-radius: var(--radius-control); background: var(--bg-subtle); }
.home-recent-cover img { width: 100%; height: 100%; object-fit: cover; }
.home-recent-copy { display: grid; flex: 1; min-width: 0; gap: 4px; }
.home-recent-copy strong { font-size: 14px; font-weight: 550; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.home-recent-copy small { font-size: 12px; color: var(--text-tertiary); line-height: 1.6; overflow-wrap: anywhere; }
.home-shortcuts a:focus-visible, .home-recent-row:focus-visible { outline: 2px solid var(--color-primary); outline-offset: 3px; }
</style>
