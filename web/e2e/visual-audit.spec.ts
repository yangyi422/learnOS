import { writeFile } from 'node:fs/promises'
import { expect, test, type Page, type Route, type TestInfo } from '@playwright/test'
import type { InboxSummary } from '../src/api/inbox'
import type { DailyFocusView } from '../src/api/dailyFocus'
import type { Project, ProjectTask } from '../src/api/projects'
import type { PlanningTask, WorkspaceHome, WorkspaceLearningItem, WorkspaceRecent } from '../src/api/workspace'

/**
 * Reusable home visual evidence, not an aesthetic or pixel-golden verdict.
 * APIs are fixtures; no Go server or database is needed. Every capture writes
 * viewport/full-page PNGs plus context JSON. Use separate --output directories
 * for before/after runs. `npm run test:ui` also runs home-material invariants.
 */

const stamp = '2026-10-21T08:00:00Z'
const date = '2026-10-21'

// Fixed Date, not paused timers: async UI and failure timeouts must still work.
// Only this screenshot spec is stabilized; interaction/motion suites stay unchanged.
test.use({ timezoneId: 'Asia/Shanghai', locale: 'zh-CN', deviceScaleFactor: 1, reducedMotion: 'reduce' })
test.beforeEach(async ({ page }) => {
  await page.clock.setFixedTime(new Date(stamp))
})

function makeProject(id: number, title: string, context: 'work' | 'life', open: number, done: number): Project {
  return {
    id, title, context, description: '', status: 'active', icon: title.slice(0, 1), accent: '',
    archived_at: null, tasks: [], open_task_count: open, doing_task_count: 0, next_task_count: open,
    done_task_count: done, created_at: stamp, updated_at: stamp,
  }
}

function makeTask(id: number, projectId: number, title: string, patch: Partial<ProjectTask> = {}): ProjectTask {
  return {
    id, project_id: projectId, title, description: '', status: 'next', sort_order: id * 1024,
    priority: 'normal', due_date: null, plan_date: null, completed_at: null, created_at: stamp,
    updated_at: stamp, archived_at: null, ...patch,
  }
}

function planningTask(task: ProjectTask, projectTitle: string, context: 'work' | 'life'): PlanningTask {
  return { ...task, project_title: projectTitle, effective_context: context }
}

function focusView(focus: DailyFocusView): DailyFocusView {
  return focus
}

const workFocusTask = makeTask(101, 1, '完成首页信息架构评审并给出修改清单', { status: 'doing', priority: 'high', plan_date: date, due_date: '2026-10-23' })
const workSecondTask = makeTask(102, 1, '整理本周项目风险与依赖', { plan_date: date })
const workOverdueTask = makeTask(103, 1, '补充接口错误处理说明', { due_date: '2026-10-19' })
const workDueTask = makeTask(104, 1, '与设计确认暗色主题对比度', { due_date: date })
const lifeFocusTask = makeTask(201, 2, '周末去江边散步并记录一条生活片段', { plan_date: date })
const lifeSecondTask = makeTask(202, 2, '整理秋天拍的照片')

const projects = [makeProject(1, '产品官网重构', 'work', 4, 6), makeProject(2, '城市漫游计划', 'life', 2, 0)]

function focusOf(task: ProjectTask, context: 'work' | 'life', projectTitle: string, patch: Partial<DailyFocusView> = {}): DailyFocusView {
  return focusView({
    id: context === 'work' ? 11 : 12,
    date,
    context,
    task_id: task.id,
    revision: 1,
    state: 'ok',
    title: task.title,
    task_status: task.status,
    priority: task.priority,
    plan_date: task.plan_date ?? null,
    due_date: task.due_date,
    completed_at: null,
    project_id: task.project_id,
    project_title: projectTitle,
    project_context: context,
    project_status: 'active',
    state_reason: '',
    updated_at: stamp,
    task_updated_at: stamp,
    ...patch,
  })
}

const workFocus = focusOf(workFocusTask, 'work', '产品官网重构')
const lifeFocus = focusOf(lifeFocusTask, 'life', '城市漫游计划')

const learning: WorkspaceLearningItem = {
  course_id: 21, course_name: '认知科学', unit_title: '认知偏差', lesson_id: 22,
  lesson_title: '为什么我们会高估自己的判断？',
  core_question: '从真实判断中识别直觉与证据，重新审视自己的结论。',
  current_level: 'understand', cognitive_status: 'stable', last_learning_at: stamp,
  has_current_lesson: true, updated_at: stamp,
}

const recent: WorkspaceRecent = {
  library: [{ id: 6, collection_id: 5, title: '合成菜谱页面', collection_name: '厨房笔记', display_mode: 'list', cover_url: '', updated_at: stamp }],
  events: [{ id: 12, title: '在旧书店找到一本想读很久的书', occurred_on: '2026-10-20', primary_domain: '' }],
  goals: [{ id: 3, goal_id: 8, goal_title: '长期阅读计划', content: '重新评估下一步', occurred_on: '2026-10-18' }],
  asset_date: '2026-10-01',
  errors: {},
}

const inbox: InboxSummary = { count: 2, items: [] }

function sceneFocus(scene: 'all' | 'work' | 'life', completed: boolean) {
  const work = completed
    ? focusOf(workFocusTask, 'work', '产品官网重构', { state: 'completed', task_status: 'done', completed_at: stamp })
    : workFocus
  if (scene === 'all') return { work, life: lifeFocus }
  if (scene === 'work') return { work, life: null }
  return { work: null, life: lifeFocus }
}

function homeFor(scene: 'all' | 'work' | 'life', completed = false): WorkspaceHome {
  const focus = sceneFocus(scene, completed)
  const planningItems = scene === 'life'
    ? [planningTask(lifeFocusTask, '城市漫游计划', 'life')]
    : [planningTask(workFocusTask, '产品官网重构', 'work'), planningTask(lifeFocusTask, '城市漫游计划', 'life')]
  const visiblePlanning = planningItems.filter(item => scene === 'all' || item.effective_context === scene)
  const visibleProjects = projects.filter(project => scene === 'all' || project.context === scene)
  const suggestion = scene === 'life'
    ? { kind: 'task' as const, title: lifeSecondTask.title, task_id: lifeSecondTask.id, project_id: 2, project_title: '城市漫游计划', status: 'inbox', priority: 'normal' as const, due_date: null }
    : { kind: 'task' as const, title: workSecondTask.title, task_id: workSecondTask.id, project_id: 1, project_title: '产品官网重构', status: 'next', priority: 'normal' as const, due_date: null }
  const today = {
    date,
    doing: scene === 'life' ? [] : [workFocusTask],
    due: scene === 'life' ? [] : [workDueTask],
    next: scene === 'life' ? [lifeSecondTask] : [workSecondTask],
  }
  return {
    date,
    context: scene,
    unclassified_count: 1,
    focus: suggestion,
    daily_focus: focus,
    today,
    planning: {
      items: visiblePlanning,
      due_today: [],
      overdue: scene === 'life' ? [] : [planningTask(workOverdueTask, '产品官网重构', 'work')],
      previous_plan: [],
      summary: { planned: visiblePlanning.length, due_today: 0, overdue: scene === 'life' ? 0 : 1, previous_plan: 0, total: visiblePlanning.length + (scene === 'life' ? 0 : 1) },
    },
    projects: visibleProjects,
    learning: scene === 'life' ? [] : [learning],
    inbox,
    errors: {},
  }
}

function emptyHome(): WorkspaceHome {
  return {
    date,
    context: 'all',
    unclassified_count: 0,
    focus: { kind: 'empty', title: '' },
    daily_focus: { work: null, life: null },
    today: { date, doing: [], due: [], next: [] },
    planning: { items: [], due_today: [], overdue: [], previous_plan: [], summary: { planned: 0, due_today: 0, overdue: 0, previous_plan: 0, total: 0 } },
    projects: [],
    learning: [],
    inbox: { count: 0, items: [] },
    errors: {},
  }
}

async function installMocks(page: Page, options: { home?: (scene: 'all' | 'work' | 'life') => WorkspaceHome; completed?: boolean; recent?: WorkspaceRecent } = {}) {
  await page.route('**/api/v1/**', async (route: Route) => {
    const url = new URL(route.request().url())
    const path = url.pathname
    const json = (data: unknown) => route.fulfill({ json: { data } })
    if (path === '/api/v1/auth/me') {
      return json({ id: 1, username: 'visual', display_name: '视觉评审', role: 'admin', status: 'active', last_login_at: null, created_at: stamp })
    }
    if (path === '/api/v1/workspace/home') {
      const scene = (url.searchParams.get('context') ?? 'all') as 'all' | 'work' | 'life'
      return json(options.home ? options.home(scene) : homeFor(scene, options.completed))
    }
    if (path === '/api/v1/workspace/recent') return json(options.recent ?? recent)
    return json({})
  })
}

async function show(page: Page, theme: 'light' | 'dark', scene: 'all' | 'work' | 'life', width: number) {
  await page.setViewportSize({ width, height: 1000 })
  await page.evaluate(([nextTheme, nextScene]) => {
    localStorage.setItem('learnos-theme', nextTheme)
    localStorage.setItem('learnos:workspace-context:1', nextScene)
  }, [theme, scene])
  await page.reload({ waitUntil: 'networkidle' })
  await expect(page.locator('html')).toHaveAttribute('data-theme', theme)
  await expect(page.locator('.daily-focus-card')).toBeVisible()
  await expect(page.locator('.home-left-lane .today-section')).toBeVisible()
}

async function capture(page: Page, info: TestInfo, name: string) {
  await page.evaluate(async () => {
    await document.fonts.ready
    window.scrollTo(0, 0)
  })
  await page.mouse.move(0, 0)
  const fullPage = info.outputPath(`visual-${name}.png`)
  const viewport = info.outputPath(`visual-${name}-viewport.png`)
  const metadata = info.outputPath(`visual-${name}.json`)
  await page.screenshot({ path: fullPage, fullPage: true, animations: 'disabled' })
  await page.screenshot({ path: viewport, animations: 'disabled' })
  const context = await page.evaluate(() => ({
    url: location.href,
    theme: document.documentElement.dataset.theme,
    viewport: { width: innerWidth, height: innerHeight, dpr: devicePixelRatio },
    locale: navigator.language,
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    currentTime: new Date().toISOString(),
    reducedMotion: matchMedia('(prefers-reduced-motion: reduce)').matches,
    font: getComputedStyle(document.body).fontFamily,
    fontsStatus: document.fonts.status,
    css: Array.from(document.styleSheets).map(sheet => sheet.href).filter(Boolean),
    js: Array.from(document.scripts).map(script => script.src).filter(Boolean),
    horizontalOverflow: document.documentElement.scrollWidth > innerWidth + 1,
  }))
  await writeFile(metadata, JSON.stringify({
    ...context,
    browserVersion: page.context().browser()?.version(),
    dataSource: 'intercepted API fixtures; not personal or live API data',
    fixtureTime: stamp,
  }, null, 2) + '\n')
  await info.attach(`visual-${name}`, { path: fullPage, contentType: 'image/png' })
  await info.attach(`visual-${name}-viewport`, { path: viewport, contentType: 'image/png' })
  await info.attach(`visual-${name}-context`, { path: metadata, contentType: 'application/json' })
}

test('视觉评审：桌面浅色三场景', async ({ page }, info) => {
  await installMocks(page)
  await page.goto('/?ambient=css', { waitUntil: 'networkidle' })
  for (const [scene, title] of [['all', '今天，按自己的节奏'], ['work', '今天，专注重要的事'], ['life', '留一点时间给自己']] as const) {
    await show(page, 'light', scene, 1440)
    await expect(page.locator('.page-header h1')).toHaveText(title)
    await expect(page.locator('.home-right-lane .capture-section')).toBeVisible()
    await expect(page.locator('.home-right-lane .recent-section')).toBeVisible()
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true)
    await capture(page, info, `desktop-light-${scene}`)
  }
})

test('视觉评审：桌面深色全部与生活', async ({ page }, info) => {
  await installMocks(page)
  await page.goto('/?ambient=css', { waitUntil: 'networkidle' })
  for (const scene of ['all', 'life'] as const) {
    await show(page, 'dark', scene, 1440)
    await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark')
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true)
    await capture(page, info, `desktop-dark-${scene}`)
  }
})

test('视觉评审：平板 1024 与手机 390', async ({ page }, info) => {
  await installMocks(page)
  await page.goto('/?ambient=css', { waitUntil: 'networkidle' })
  await show(page, 'light', 'all', 1024)
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true)
  await capture(page, info, 'tablet-light-all')
  await show(page, 'light', 'all', 390)
  await expect(page.locator('.home-right-lane .capture-section')).toBeVisible()
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true)
  await capture(page, info, 'mobile-light-all')
  await show(page, 'dark', 'life', 390)
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true)
  await capture(page, info, 'mobile-dark-life')
})

test('视觉评审：空工作区自然留白', async ({ page }, info) => {
  await installMocks(page, { home: () => emptyHome(), recent: { library: [], events: [], goals: [], asset_date: '', errors: {} } })
  await page.goto('/?ambient=css', { waitUntil: 'networkidle' })
  await show(page, 'light', 'all', 1440)
  await expect(page.locator('.daily-focus-empty')).toBeVisible()
  await expect(page.getByRole('heading', { name: '今天还没有安排' })).toBeVisible()
  await expect(page.locator('.recent-section')).toHaveCount(0)
  await expect(page.locator('.learning-section')).toBeVisible()
  await capture(page, info, 'empty-light-1440')
})

test('视觉评审：已完成焦点不展示完成按钮', async ({ page }, info) => {
  await installMocks(page, { completed: true })
  await page.goto('/?ambient=css', { waitUntil: 'networkidle' })
  await show(page, 'light', 'all', 1440)
  const slot = page.locator('.daily-focus-slot[data-context="work"]')
  await expect(slot).toHaveAttribute('data-state', 'completed')
  await expect(slot).toContainText('已完成')
  await expect(slot.getByRole('button', { name: '完成工作焦点' })).toHaveCount(0)
  await capture(page, info, 'completed-light-1440')
})
