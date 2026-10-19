import { expect, test, type Page, type Route } from '@playwright/test'
import type { InboxItem } from '../src/api/inbox'
import type { Project, ProjectTask } from '../src/api/projects'
import type { WorkspaceHome, WorkspaceLearningItem, WorkspaceRecent } from '../src/api/workspace'

const timestamp = '2026-09-29T08:00:00Z'
const project: Project = { id: 7, title: 'LearnOS', description: '', status: 'active', icon: 'L', accent: '#3b82f6', archived_at: null, tasks: [], open_task_count: 1, next_task_count: 1, doing_task_count: 0, done_task_count: 0, created_at: timestamp, updated_at: timestamp }
const task: ProjectTask = { id: 41, project_id: 7, title: '整理看板交互', description: '', status: 'next', sort_order: 1024, priority: 'high', due_date: null, completed_at: null, created_at: timestamp, updated_at: timestamp }
const emptyToday = { date: '2026-09-29', doing: [], due: [], next: [] }

interface MockOptions {
  recent?: WorkspaceRecent
  recentGate?: Promise<void>
  failRecentTimes?: number
  homeDelayMs?: number
  homeGate?: Promise<void>
  failHomeTimes?: number
  failCreateTimes?: number
  firstHomeOverrides?: Partial<WorkspaceHome>
}

async function installWorkspaceMocks(page: Page, homeOverrides: Partial<WorkspaceHome> = {}, options: MockOptions = {}) {
  const inbox: InboxItem[] = [{ id: 9, user_id: 1, content: '研究 SQLite WAL backup', status: 'inbox', source_type: 'manual', source_url: '', processed_to_type: '', processed_to_id: null, created_at: timestamp, updated_at: timestamp, processed_at: null, archived_at: null }]
  let recentCalls = 0
  let homeCalls = 0
  let createFailures = 0
  await page.route('**/api/v1/**', async (route: Route) => {
    const { pathname } = new URL(route.request().url())
    const method = route.request().method()
    const reply = (data: unknown, status = 200) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify({ data }) })
    if (pathname === '/api/v1/auth/me') return reply({ id: 1, username: 'isolated-test', role: 'admin', status: 'active' })
    if (pathname === '/api/v1/workspace/recent') {
      recentCalls += 1
      if (options.recentGate) await options.recentGate
      if (recentCalls <= (options.failRecentTimes ?? 0)) return route.fulfill({ status: 503, contentType: 'application/json', body: JSON.stringify({ error: { code: 'RECENT_UNAVAILABLE', message: '最近内容暂时无法读取。', retryable: true } }) })
      return reply(options.recent ?? { library: [], events: [], goals: [], asset_date: '', errors: {} })
    }
    if (pathname === '/api/v1/workspace/home') {
      homeCalls += 1
      if (homeCalls === 1 && options.homeGate) await options.homeGate
      if (options.homeDelayMs) await new Promise(resolve => setTimeout(resolve, options.homeDelayMs))
      if (homeCalls <= (options.failHomeTimes ?? 0)) {
        return route.fulfill({
          status: 503,
          contentType: 'application/json',
          body: JSON.stringify({ error: { code: 'WORKSPACE_UNAVAILABLE', message: '工作区暂时无法加载，请重试。', retryable: true } }),
        })
      }
      const overrides = homeCalls === 1 && options.firstHomeOverrides ? options.firstHomeOverrides : homeOverrides
      return reply({
        date: '2026-09-29',
        focus: { kind: 'task', title: task.title, task_id: task.id, project_id: project.id, project_title: project.title, status: 'next', priority: 'high' },
        today: { date: '2026-09-29', doing: [], due: [], next: [task] },
        projects: [project],
        learning: [],
        inbox: { count: inbox.filter(item => item.status === 'inbox').length, items: inbox.filter(item => item.status === 'inbox').slice(0, 3) },
        errors: {},
        ...overrides,
      })
    }
    if (pathname === '/api/v1/inbox' && method === 'GET') {
      return reply({
        items: inbox,
        counts: {
          inbox: inbox.filter(item => item.status === 'inbox').length,
          processed: inbox.filter(item => item.status === 'processed').length,
          archived: inbox.filter(item => item.status === 'archived').length,
        },
      })
    }
    if (pathname === '/api/v1/inbox' && method === 'POST') {
      if (createFailures < (options.failCreateTimes ?? 0)) {
        createFailures += 1
        return route.fulfill({
          status: 400,
          contentType: 'application/json',
          body: JSON.stringify({ error: { code: 'INBOX_INPUT_INVALID', message: '请检查收集内容。', retryable: false } }),
        })
      }
      const body = route.request().postDataJSON() as { content: string }
      const item: InboxItem = {
        id: Date.now(), user_id: 1, content: body.content, status: 'inbox',
        source_type: /^https?:\/\//.test(body.content) ? 'url' : 'manual',
        source_url: /^https?:\/\//.test(body.content) ? body.content : '',
        processed_to_type: '', processed_to_id: null,
        created_at: new Date().toISOString(), updated_at: new Date().toISOString(),
        processed_at: null, archived_at: null,
      }
      inbox.unshift(item)
      return reply(item, 201)
    }
    if (/^\/api\/v1\/inbox\/\d+\/convert-to-task$/.test(pathname) && method === 'POST') {
      const id = Number(pathname.split('/')[4])
      const index = inbox.findIndex(item => item.id === id)
      const item: InboxItem = { ...inbox[index], status: 'processed', processed_to_type: 'task', processed_to_id: task.id, processed_at: timestamp }
      inbox[index] = item
      return reply({ item, task }, 201)
    }
    if (pathname === '/api/v1/projects') return reply([project])
    if (pathname === '/api/v1/tasks' && method === 'GET') return reply([task])
    if (pathname === '/api/v1/tasks/41' && method === 'GET') return reply(task)
    if (pathname === '/api/v1/projects/today') return reply({ date: '2026-09-29', doing: [], due: [], next: [task] })
    return reply({})
  })
  return inbox
}

test('Workspace Home 保留核心区域并能从焦点打开任务', async ({ page }, testInfo) => {
  await page.setViewportSize({ width: 1360, height: 900 })
  await page.addInitScript(() => localStorage.setItem('learnos-theme', 'system'))
  await page.emulateMedia({ colorScheme: 'light' })
  await installWorkspaceMocks(page)
  await page.goto('/')
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'light')
  await expect(page.locator('#focus-title')).toHaveText(task.title)
  await expect(page.locator('.focus-card__reason')).toContainText('当前优先级最高的下一步')
  await expect(page.getByRole('heading', { name: '今天的安排', exact: true })).toBeVisible()
  await expect(page.getByRole('heading', { name: '正在推进' })).toBeVisible()
  await expect(page.getByRole('heading', { name: '继续学习' })).toBeVisible()
  await expect(page.getByRole('link', { name: '收集箱 · 1 条待整理' })).toBeVisible()
  await expect(page.locator('.inbox-section')).toHaveCount(0)
  await expect(page.getByRole('textbox', { name: '快速记录到收集箱' })).toHaveCount(0)
  if (process.env.CAPTURE_VISUALS) await page.screenshot({ path: testInfo.outputPath('workspace-home-light.png'), fullPage: true })
  await page.emulateMedia({ colorScheme: 'dark' })
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark')
  await page.waitForTimeout(200)
  if (process.env.CAPTURE_VISUALS) await page.screenshot({ path: testInfo.outputPath('workspace-home-dark.png'), fullPage: true })
  await page.locator('.focus-card-entry').click({ position: { x: 12, y: 12 } })
  await expect(page).toHaveURL(/\/projects\?project=7&view=board$/)
  await expect(page.getByRole('dialog', { name: '任务详情' })).toBeVisible()
  if (process.env.CAPTURE_VISUALS) await page.screenshot({ path: testInfo.outputPath('workspace-task-drawer-dark.png'), fullPage: true })
})

const taskFocusScenarios: { name: string; status: ProjectTask['status']; dueDate: string | null; reason: string }[] = [
  { name: 'doing', status: 'doing', dueDate: null, reason: '当前正在推进' },
  { name: 'overdue', status: 'next', dueDate: '2026-09-27', reason: '已到期，需要处理' },
  { name: 'due today', status: 'next', dueDate: '2026-09-29', reason: '今天到期' },
  { name: 'high priority next', status: 'next', dueDate: null, reason: '当前优先级最高的下一步' },
]

for (const scenario of taskFocusScenarios) {
  test('Current Focus 说明任务原因：' + scenario.name, async ({ page }) => {
    const focusTask: ProjectTask = { ...task, status: scenario.status, due_date: scenario.dueDate }
    const today = {
      date: '2026-09-29',
      doing: scenario.status === 'doing' ? [focusTask] : [],
      due: scenario.status !== 'doing' && scenario.dueDate && scenario.dueDate <= '2026-09-29' ? [focusTask] : [],
      next: scenario.status === 'next' && (!scenario.dueDate || scenario.dueDate > '2026-09-29') ? [focusTask] : [],
    }
    await installWorkspaceMocks(page, {
      focus: { kind: 'task', title: focusTask.title, task_id: focusTask.id, project_id: focusTask.project_id, project_title: project.title, status: focusTask.status, priority: focusTask.priority, due_date: focusTask.due_date },
      today,
    })
    await page.goto('/')
    await expect(page.locator('.focus-card__reason')).toContainText(scenario.reason)
  })
}

test('Current Focus 使用最近学习位置作为回退', async ({ page }) => {
  const learning: WorkspaceLearningItem = {
    course_id: 21, course_name: '营养学', unit_title: '能量与能量平衡', lesson_id: 22,
    lesson_title: '能量摄入与消耗的三大出口', core_question: '能量平衡如何形成？',
    current_level: 'understand', cognitive_status: 'developing', last_learning_at: timestamp,
    has_current_lesson: true, updated_at: timestamp,
  }
  await installWorkspaceMocks(page, {
    focus: { kind: 'learning', title: learning.lesson_title, course_id: learning.course_id, course_name: learning.course_name, unit_title: learning.unit_title, lesson_id: learning.lesson_id, current_level: learning.current_level, cognitive_status: learning.cognitive_status },
    today: emptyToday,
    learning: [learning],
  })
  await page.goto('/')
  await expect(page.locator('.focus-card__reason')).toContainText('最近的学习位置')
  await expect(page.locator('.focus-card__question')).toHaveText(learning.core_question)
  await expect(page.locator('.focus-card-entry')).toBeVisible()
})

test('Today 先展示逾期、今天到期，再展示进行中，并限制三条', async ({ page }) => {
  const overdue: ProjectTask = { ...task, title: '逾期两天', status: 'next', due_date: '2026-09-27' }
  const dueToday: ProjectTask = { ...task, id: 62, title: '今天到期', status: 'next', due_date: '2026-09-29' }
  const doing: ProjectTask = { ...task, id: 63, title: '正在推进', status: 'doing', due_date: null }
  const next: ProjectTask = { ...task, id: 64, title: '下一步候选', status: 'next', due_date: null }
  await installWorkspaceMocks(page, {
    today: { date: '2026-09-29', doing: [doing], due: [dueToday, overdue], next: [next] },
  })
  await page.goto('/')
  await expect(page.locator('.home-task__title')).toHaveText(['逾期两天', '今天到期', '正在推进'])
  await expect(page.locator('.home-task').first()).toContainText('逾期 2 天')
  await expect(page.locator('.home-task').nth(1)).toContainText('今天到期')
  await page.locator('.home-task').first().click()
  await expect(page.getByRole('dialog', { name: '任务详情' })).toBeVisible()
})

test('项目列表最多显示五项，点击整行进入对应看板', async ({ page }) => {
  const projects = Array.from({ length: 6 }, (_, index) => ({ ...project, id: index + 7, title: '项目 ' + (index + 1) }))
  await installWorkspaceMocks(page, { projects })
  await page.goto('/')
  await expect(page.locator('.project-row')).toHaveCount(5)
  await page.locator('.project-row').first().click()
  await expect(page).toHaveURL(/\/projects\?project=7&view=board$/)
})

test('Continue Learning 展示最近课程并提供继续入口', async ({ page }, testInfo) => {
  const learning: WorkspaceLearningItem = {
    course_id: 21, course_name: '营养学', unit_title: '能量与能量平衡', lesson_id: 22,
    lesson_title: '能量摄入与消耗的三大出口', core_question: '身体如何分配能量？',
    current_level: 'understand', cognitive_status: 'stable', last_learning_at: timestamp,
    has_current_lesson: true, updated_at: timestamp,
  }
  await page.setViewportSize({ width: 1360, height: 900 })
  await page.addInitScript(() => localStorage.setItem('learnos-theme', 'system'))
  await page.emulateMedia({ colorScheme: 'light' })
  await installWorkspaceMocks(page, { learning: [learning] })
  await page.goto('/')
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'light')
  await expect(page.locator('.learning-row')).toContainText('营养学')
  await expect(page.locator('.learning-row')).toContainText(learning.core_question)
  if (process.env.CAPTURE_VISUALS) await page.screenshot({ path: testInfo.outputPath('workspace-home-learning-light.png'), fullPage: true })
  await page.emulateMedia({ colorScheme: 'dark' })
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark')
  await page.waitForTimeout(200)
  if (process.env.CAPTURE_VISUALS) await page.screenshot({ path: testInfo.outputPath('workspace-home-learning-dark.png'), fullPage: true })
  await page.locator('.learning-card-entry').focus()
  await page.keyboard.press('Enter')
  await expect(page).toHaveURL('/courses/21/learn')
})

test('没有当前 Lesson 的学习领域仍提供打开知识地图入口', async ({ page }) => {
  const learning: WorkspaceLearningItem = {
    course_id: 21, course_name: '心理学', unit_title: '', lesson_id: 0,
    lesson_title: '', core_question: '', current_level: 'unseen', cognitive_status: 'unknown',
    last_learning_at: null, has_current_lesson: false, updated_at: timestamp,
  }
  await installWorkspaceMocks(page, { learning: [learning] })
  await page.goto('/')
  await expect(page.locator('.learning-row')).toContainText('选择下一段学习')
  await expect(page.locator('.learning-card-entry')).toHaveAccessibleName(/打开知识地图/)
  await page.locator('.learning-card-entry').focus()
  await page.keyboard.press('Enter')
  await expect(page).toHaveURL('/courses/21/map')
})

test('首页顶部快速记录仍打开原有对话框', async ({ page }) => {
  await installWorkspaceMocks(page)
  await page.goto('/')
  await page.getByRole('button', { name: '快速记录' }).click()
  await expect(page.getByRole('dialog', { name: '快速记录' })).toBeVisible()
})

test('快速记录统一在弹层保存文字和链接，并刷新真实待整理数量', async ({ page }) => {
  const inbox = await installWorkspaceMocks(page)
  await page.goto('/')
  const capture = page.getByRole('dialog', { name: '快速记录', exact: true })
  await page.getByRole('button', { name: '快速记录', exact: true }).click()
  const input = capture.getByRole('textbox', { name: '快速记录到收集箱' })
  await expect(input).toBeFocused()
  await input.fill('先记录文字\n第二行')
  await capture.getByRole('button', { name: '记录到收集箱' }).click()
  await expect(capture).not.toBeVisible()
  await expect(page.getByRole('link', { name: '收集箱 · 2 条待整理' })).toBeVisible()
  expect(inbox[0]?.content).toBe('先记录文字\n第二行')
  await page.getByRole('button', { name: '快速记录', exact: true }).click()
  await input.fill('https://example.com/guide')
  await input.press('Enter')
  await expect(capture).not.toBeVisible()
  await expect(page.getByRole('link', { name: '收集箱 · 3 条待整理' })).toBeVisible()
  expect(inbox[0]?.source_type).toBe('url')
  await expect(page.locator('.inbox-row')).toHaveCount(0)
})

test('快速记录失败保留输入，取消关闭和放弃操作不会保存记录', async ({ page }) => {
  const inbox = await installWorkspaceMocks(page, {}, { failCreateTimes: 1 })
  await page.goto('/')
  await page.getByRole('button', { name: '快速记录', exact: true }).click()
  const capture = page.getByRole('dialog', { name: '快速记录', exact: true })
  const input = capture.getByRole('textbox')
  await input.fill('不能丢失的记录')
  await input.press('Enter')
  await expect(capture.getByRole('alert')).toContainText('请检查收集内容')
  await expect(input).toHaveValue('不能丢失的记录')
  await capture.getByRole('button', { name: '取消', exact: true }).click()
  await page.getByRole('button', { name: '继续编辑', exact: true }).click()
  await expect(input).toHaveValue('不能丢失的记录')
  await capture.getByRole('button', { name: '取消', exact: true }).click()
  await page.getByRole('button', { name: '放弃记录', exact: true }).click()
  await expect(capture).not.toBeVisible()
  expect(inbox).toHaveLength(1)
  await page.getByRole('button', { name: '快速记录', exact: true }).click()
  await expect(input).toHaveValue('')
})

test('首页显示各区骨架，首屏请求失败后可重试', async ({ page }) => {
  let release!: () => void
  const homeGate = new Promise<void>(resolve => { release = resolve })
  await installWorkspaceMocks(page, {}, { homeGate, failHomeTimes: 1 })
  await page.goto('/')
  await expect(page.locator('.focus-card--skeleton')).toBeVisible()
  await expect(page.locator('.home-section--skeleton')).toHaveCount(3)
  release()
  await expect(page.locator('.workspace-home__fatal-error')).toBeVisible()
  await page.getByRole('button', { name: '重试' }).click()
  await expect(page.locator('#focus-title')).toHaveText(task.title)
})

test('首页分区错误可重试且不会遮盖其它区域', async ({ page }) => {
  await installWorkspaceMocks(page, {}, { firstHomeOverrides: { errors: { projects: '项目数据暂时无法加载。' } } })
  await page.goto('/')
  await expect(page.locator('.project-row')).toHaveCount(0)
  await expect(page.locator('.module-error')).toContainText('项目数据暂时无法加载')
  await expect(page.locator('.today-section .today-summary')).toBeVisible()
  await page.locator('.projects-section .module-error .inline-action').click()
  await expect(page.locator('.project-row')).toHaveCount(1)
})

test('窄屏核心区域顺序保持，快捷入口和弹层记录可用', async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 812 })
  await installWorkspaceMocks(page)
  await page.goto('/')
  const layout = await page.evaluate(() => ({
    width: document.documentElement.scrollWidth,
    viewport: window.innerWidth,
    focus: document.querySelector('.focus-card')?.getBoundingClientRect().top ?? 0,
    today: document.querySelector('.today-section')?.getBoundingClientRect().top ?? 0,
    learning: document.querySelector('.learning-section')?.getBoundingClientRect().top ?? 0,
    projects: document.querySelector('.projects-section')?.getBoundingClientRect().top ?? 0,
    inbox: document.querySelector('.home-shortcuts')?.getBoundingClientRect().top ?? 0,
  }))
  expect(layout.width).toBeLessThanOrEqual(layout.viewport)
  expect(layout.focus).toBeLessThan(layout.today)
  expect(layout.today).toBeLessThan(layout.learning)
  expect(layout.learning).toBeLessThan(layout.projects)
  expect(layout.projects).toBeLessThan(layout.inbox)
  await page.getByRole('button', { name: '快速记录', exact: true }).click()
  await expect(page.getByRole('textbox', { name: '快速记录到收集箱' })).toBeVisible()
  await expect(page.getByRole('button', { name: '记录到收集箱' })).toBeVisible()
})

test('空工作区不虚构数据，收集箱与最近内容没有大型空块', async ({ page }) => {
  await installWorkspaceMocks(page, {
    focus: { kind: 'empty', title: '' },
    today: emptyToday,
    projects: [],
    learning: [],
    inbox: { count: 0, items: [] },
  })
  await page.goto('/')
  await expect(page.getByRole('heading', { name: '暂时没有需要继续的事项' })).toBeVisible()
  await expect(page.getByRole('heading', { name: '今天没有优先事项' })).toBeVisible()
  await expect(page.getByRole('button', { name: '查看项目' })).toBeVisible()
  await expect(page.getByRole('button', { name: '开始一个学习领域' })).toBeVisible()
  await expect(page.getByRole('button', { name: '快速记录', exact: true })).toBeVisible()
  await expect(page.getByRole('heading', { name: '收集箱是空的' })).toHaveCount(0)
  await expect(page.locator('.recent-section')).toHaveCount(0)
  await expect(page.getByRole('link', { name: '收集箱 · 0 条待整理' })).toBeVisible()
  await expect(page.locator('.home-task')).toHaveCount(0)
  await expect(page.locator('.project-row')).toHaveCount(0)
  await expect(page.locator('.learning-row')).toHaveCount(0)
  await expect(page.locator('.inbox-row')).toHaveCount(0)
})

test('Inbox 管理页快速记录和转换为任务仍可用', async ({ page }) => {
  await installWorkspaceMocks(page)
  await page.goto('/inbox')
  await page.getByRole('button', { name: '快速记录' }).click()
  const capture = page.getByRole('dialog', { name: '快速记录' })
  await capture.getByRole('textbox').fill('记下一个新的想法')
  await page.keyboard.press('Control+Enter')
  await expect(page.getByText('已保存到收集箱')).toBeVisible()
  await expect(page.getByText('记下一个新的想法')).toBeVisible()

  await page.getByRole('button', { name: '转为任务' }).first().click()
  const convert = page.getByRole('dialog', { name: '整理为项目任务' })
  await convert.locator('.el-form-item').filter({ hasText: '所属项目' }).locator('.el-select__wrapper').click()
  await page.getByRole('option', { name: project.title }).click()
  await convert.getByRole('button', { name: '转换为任务' }).click()
  await expect(page).toHaveURL(/\/projects\?project=7&view=board$/)
  await expect(page.getByRole('dialog', { name: '任务详情' })).toBeVisible()
})

const imageID = 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'
const recentFacts: WorkspaceRecent = {
  library: [{ id: 6, collection_id: 5, title: '合成菜谱页面', collection_name: '自己的菜谱', display_mode: 'list', cover_url: `/api/v1/library/images/${imageID}`, updated_at: '2026-10-11T08:00:00Z' }],
  events: [{ id: 12, title: '最近的生活经历', occurred_on: '2026-10-12', primary_domain: 'experiences' }, { id: 13, title: '补录九月经历', occurred_on: '2026-09-01', primary_domain: '' }],
  goals: [{ id: 21, goal_id: 8, goal_title: '长期愿望', content: '重新评估下一步', occurred_on: '2026-10-10' }],
  asset_date: '2026-10-01', errors: {},
}
async function installRecentSourceMocks(page: Page) {
  await page.route('**/api/v1/library/**', async route => {
    const path = new URL(route.request().url()).pathname
    const reply = (data: unknown) => route.fulfill({ contentType: 'application/json', body: JSON.stringify({ data }) })
    if (path.includes('/images/')) return route.fulfill({ contentType: 'image/png', body: Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jpV8AAAAASUVORK5CYII=', 'base64') })
    if (path === '/api/v1/library/collections') return reply([])
    if (path === '/api/v1/library/collections/5') return reply({ collection: { id: 5, name: '自己的菜谱', display_mode: 'list', description: '', category: '', cover_url: '' }, sections: [] })
    if (path.endsWith('/pages/6')) return reply({ ...recentFacts.library[0], body: '原始菜谱正文', section_id: null, sources: [], tags: [], version: 1 })
    return reply({ items: [], total: 0 })
  })
  await page.route('**/api/v1/life/**', route => {
    const path = new URL(route.request().url()).pathname
    let data: unknown = []
    if (path === '/api/v1/life/events/12') data = { ...recentFacts.events[0], description: '原始生活事件', milestone: false, source_type: '', created_at: timestamp, updated_at: timestamp }
    if (path === '/api/v1/life/goals/8') data = { goal: { id: 8, title: '长期愿望', why: '', current_note: '', status: 'active', domain: '', created_at: timestamp, updated_at: timestamp }, entries: [{ ...recentFacts.goals[0], content: '重新评估下一步' }], events: [] }
    return route.fulfill({ contentType: 'application/json', body: JSON.stringify({ data }) })
  })
}

for (const width of [1920, 1440, 390]) {
  test(`首页轻量聚合真实时间与私有缩略图，${width}px 布局可用`, async ({ page }, info) => {
    await page.setViewportSize({ width, height: 960 })
    await installWorkspaceMocks(page, {}, { recent: recentFacts })
    await installRecentSourceMocks(page)
    const images: string[] = [], homeRequests: string[] = []
    page.on('request', request => {
      if (request.url().includes('/library/images/')) images.push(request.url())
      if (request.url().includes('/workspace/')) homeRequests.push(new URL(request.url()).pathname)
    })
    await page.goto('/')
    const rows = page.locator('.home-recent-row')
    await expect(rows).toHaveCount(4)
    await expect(rows.first()).toContainText('最近的生活经历')
    await expect(rows.nth(1)).toContainText('合成菜谱页面')
    await expect(rows.nth(2)).toContainText('重新评估下一步')
    await expect(rows.last()).toContainText('发生 2026年9月1日')
    await expect(rows.nth(1)).toHaveAttribute('href', '/explore/library/5/pages/6?view=list')
    await expect(rows.first()).toHaveAttribute('href', '/life?event=12')
    await expect(rows.nth(2)).toHaveAttribute('href', '/life?view=goals&goal=8')
    await expect(page.getByRole('link', { name: /财务档案/ })).toContainText('最近快照 2026年10月1日')
    await expect(page.locator('.home-shortcuts')).not.toContainText(/净资产|银行余额|收入|支出|负债/)
    await page.locator('.home-shortcuts').scrollIntoViewIfNeeded()
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true)
    await expect(page.locator('.home-recent-cover img')).toHaveAttribute('src', `/api/v1/library/images/${imageID}?variant=card-v1`)
    expect(homeRequests.filter(path => path.endsWith('/home'))).toHaveLength(1)
    expect(homeRequests.filter(path => path.endsWith('/recent'))).toHaveLength(1)
    await expect.poll(() => images.length).toBe(1)
    expect(images.every(url => url.includes('variant=card-v1'))).toBe(true)
    if (process.env.CAPTURE_VISUALS) {
      await page.evaluate(() => { window.scrollTo(0, 0); document.querySelector('.main-content')?.scrollTo(0, 0) })
      await expect(page.getByRole('button', { name: '快速记录', exact: true })).toBeInViewport()
      await page.screenshot({ path: info.outputPath(`home-v02-${width}.png`), fullPage: true })
    }
  })
}

test('最近内容直达原始菜谱与生活详情', async ({ page }) => {
  await installWorkspaceMocks(page, {}, { recent: recentFacts })
  await installRecentSourceMocks(page)
  await page.goto('/')
  await page.getByRole('link', { name: /合成菜谱页面/ }).click()
  await expect(page).toHaveURL(/\/explore\/library\/5\/pages\/6\?view=list$/)
  await expect(page.locator('.library-markdown')).toContainText('原始菜谱正文')
  await page.goBack()
  await page.getByRole('link', { name: /最近的生活经历/ }).click()
  await expect(page).toHaveURL(/\/life\?event=12/)
  await expect(page.getByRole('dialog')).toContainText('原始生活事件')
})

test('次要模块慢或失败不阻塞焦点，重试与部分错误保留其他来源', async ({ page }) => {
  let release!: () => void
  const recentGate = new Promise<void>(resolve => { release = resolve })
  await installWorkspaceMocks(page, {}, { recent: { ...recentFacts, library: [], errors: { library: '最近资料暂时无法读取。' } }, recentGate, failRecentTimes: 1 })
  await page.goto('/')
  await expect(page.locator('#focus-title')).toHaveText(task.title)
  await expect(page.locator('.recent-section')).toContainText('正在读取最近记录')
  release()
  await expect(page.locator('.recent-section')).toContainText('最近内容暂时无法读取')
  await page.getByRole('button', { name: '重试最近内容' }).click()
  await expect(page.locator('.home-recent-row')).toHaveCount(3)
  await expect(page.locator('.recent-section')).toContainText('最近资料暂时无法读取')
  await expect(page.locator('#focus-title')).toHaveText(task.title)
})

test('浏览器后退时确认未保存记录，继续编辑保留输入，放弃不创建条目', async ({ page }) => {
  const inbox = await installWorkspaceMocks(page)
  await page.goto('/inbox')
  await page.getByRole('link', { name: 'LearnOS 工作区', exact: true }).click()
  await page.getByRole('button', { name: '快速记录', exact: true }).click()
  const capture = page.getByRole('dialog', { name: '快速记录', exact: true })
  await capture.getByRole('textbox').fill('导航时不要丢失')
  await page.evaluate(() => history.back())
  await page.getByRole('button', { name: '继续编辑', exact: true }).click()
  await expect(page).toHaveURL(/\/$/)
  await expect(capture.getByRole('textbox')).toHaveValue('导航时不要丢失')
  await expect(page.getByRole('dialog', { name: '放弃未保存的记录？', exact: true })).not.toBeVisible()
  await page.evaluate(() => history.back())
  await page.getByRole('button', { name: '放弃记录', exact: true }).click()
  await expect(page).toHaveURL(/\/inbox$/)
  expect(inbox).toHaveLength(1)
})

for (const theme of ['light', 'dark'] as const) for (const width of [1920, 1440, 1280, 1180, 1024, 390]) {
  test(`Prismatic B ${theme} at ${width}px`, async ({ page }, info) => {
    await page.setViewportSize({ width, height: 1080 })
    await page.addInitScript(theme => localStorage.setItem('learnos-theme', theme), theme)
    const focusTask = { ...task, title: '完成个人知识库首页', description: '整理导航与内容层级，让每条记录都有去处。' }
    const learning: WorkspaceLearningItem = { course_id: 21, course_name: '认知科学', unit_title: '认知偏差', lesson_id: 22, lesson_title: '为什么我们会高估自己的判断？', core_question: '从真实判断中识别直觉与证据，重新审视自己的结论。', current_level: 'understand', cognitive_status: 'stable', last_learning_at: timestamp, has_current_lesson: true, updated_at: timestamp }
    await installWorkspaceMocks(page, {
      focus: { kind: 'task', title: focusTask.title, task_id: task.id, project_id: project.id, project_title: '个人知识库', status: 'next', priority: 'high' },
      projects: [{ ...project, title: '个人知识库', open_task_count: 4, done_task_count: 6 }, { ...project, id: 8, title: '阅读与写作', icon: '写', open_task_count: 5, done_task_count: 3 }, { ...project, id: 9, title: '城市漫游', icon: '城', open_task_count: 3, done_task_count: 2 }],
      learning: [learning],
      today: { ...emptyToday, doing: [{ ...task, id: 42, title: '阅读《思考，快与慢》', status: 'doing' }], due: [{ ...task, id: 43, title: '整理周末出行安排', due_date: '2026-09-29' }], next: [focusTask] },
    }, { recent: recentFacts })
    await installRecentSourceMocks(page)
    const errors: string[] = []
    page.on('pageerror', error => errors.push(error.message))
    await page.goto('/')
    await expect(page.locator('#focus-title')).toHaveText(focusTask.title)
    await expect(page.locator('.focus-card button')).toHaveCount(0)
    await expect(page.locator('.learning-section button')).toHaveCount(0)
    await expect(page.locator('.learning-card-entry')).toHaveAttribute('href', '/courses/21/learn')
    await expect(page.locator('.project-row__counts')).toHaveText(['6 / 10', '3 / 8', '2 / 5'])
    await expect(page.locator('.home-recent-row')).toHaveCount(4)
    await expect(page.locator('.sidebar')).toHaveCount(0)
    await expect(page.locator('.home-light-field')).toHaveCount(1)
    await expect(page.locator('.home-light-field')).toHaveAttribute('aria-hidden', 'true')
    const rect = async (selector: string) => (await page.locator(selector).boundingBox())!
    const focus = await rect('.focus-card'), projects = await rect('.projects-section'), today = await rect('.today-section'), lesson = await rect('.learning-section'), recent = await rect('.recent-section')
    if (width >= 1100) {
      await expect(page.getByRole('navigation', { name: '主要导航' })).toBeVisible()
      const nav = await rect('.home-topnav')
      expect(Math.abs(nav.x + nav.width / 2 - focus.x - focus.width / 2)).toBeLessThan(1)
      const actions = await rect('.home-header-actions')
      expect(nav.x + nav.width).toBeLessThanOrEqual(actions.x)
      expect(projects.x + projects.width).toBeLessThan(focus.x)
      expect(focus.x + focus.width).toBeLessThan(recent.x)
      expect(Math.abs(projects.y - focus.y)).toBeLessThan(1)
      expect(Math.abs(recent.y - focus.y)).toBeLessThan(1)
      expect(today.y).toBeGreaterThan(projects.y + projects.height)
      expect(lesson.y).toBeGreaterThan(focus.y + focus.height)
      expect(focus.width).toBeGreaterThan(projects.width)
      expect(recent.height).toBeGreaterThan(focus.height)
    } else if (width > 700) {
      await expect(page.getByLabel('打开导航')).toBeVisible()
      expect(focus.x + focus.width).toBeLessThan(projects.x)
      expect(lesson.y).toBeGreaterThan(focus.y + focus.height)
    } else {
      expect(focus.y).toBeLessThan(today.y)
      expect(today.y).toBeLessThan(lesson.y)
      expect(lesson.y).toBeLessThan(projects.y)
      expect(projects.y).toBeLessThan(recent.y)
    }
    const material = await page.evaluate(() => {
      const cards = ['.focus-card', '.projects-section', '.today-section', '.learning-section', '.recent-section'].map(s => getComputedStyle(document.querySelector(s)!))
      return { opacity: cards.map(s => s.opacity), fills: cards.map(s => s.backgroundColor), blur: cards.map(s => s.backdropFilter), light: getComputedStyle(document.querySelector('.home-light-field')!, '::before').backgroundImage, pointer: getComputedStyle(document.querySelector('.home-light-field')!).pointerEvents, overflow: document.documentElement.scrollWidth > innerWidth, ambient: getComputedStyle(document.querySelector('.home-ambient')!).backgroundImage }
    })
    expect(material.opacity.every(value => value === '1')).toBe(true)
    expect(material.fills.every(value => /^rgba/.test(value))).toBe(true)
    expect(material.blur.every(value => value === (theme === 'light' ? 'blur(18px) saturate(1.2)' : 'blur(14px)'))).toBe(true)
    expect(material.light.match(/radial-gradient/g)).toHaveLength(3)
    expect(material.pointer).toBe('none')
    if (theme === 'dark') expect(material.ambient).toBe('none')
    else expect(material.ambient).toContain('linear-gradient')
    expect(material.overflow).toBe(false)
    await page.emulateMedia({ reducedMotion: 'reduce' })
    await page.mouse.move(0, 0)
    await page.evaluate(() => window.scrollTo(0, 0))
    await page.screenshot({ path: info.outputPath(`selected-b-${theme}-${width}.png`), fullPage: true })
    await page.screenshot({ path: info.outputPath(`selected-b-${theme}-${width}-viewport.png`) })
    // Compare hierarchy with the decoration removed; content and geometry must survive.
    const noBacklight = await page.addStyleTag({ content: '.home-light-field { display: none !important; }' })
    await expect(page.locator('#focus-title')).toBeVisible()
    expect(await rect('.focus-card')).toEqual(focus)
    await page.screenshot({ path: info.outputPath(`quiet-b-${theme}-${width}-no-backlight.png`) })
    await noBacklight.evaluate(element => element.parentNode?.removeChild(element))
    // The art layer must never block the real task/capture controls.
    await page.getByRole('button', { name: '快速记录', exact: true }).click()
    await expect(page.getByRole('dialog', { name: '快速记录', exact: true })).toBeVisible()
    await page.getByRole('button', { name: '取消', exact: true }).click()
    await expect(page.getByRole('dialog', { name: '快速记录', exact: true })).toBeHidden()
    await page.getByRole('link', { name: /我的资料库/ }).click()
    await expect(page).toHaveURL(/\/explore\/library$/)
    await expect(page.locator('.home-light-field')).toHaveCount(1)
    await expect(page.locator('.home-topbar')).toHaveCount(1)
    await expect(page.locator('.sidebar')).toHaveCount(0)
    expect(errors).toEqual([])
  })
}

test('工作区搜索只检索当前真实内容，支持 Escape 和键盘直达任务', async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 })
  await installWorkspaceMocks(page)
  await page.goto('/')
  const search = page.getByRole('searchbox', { name: '搜索当前工作区' })
  await search.fill('看板')
  await expect(page.getByRole('list', { name: '工作区搜索结果' })).toContainText(task.title)
  await page.keyboard.press('Escape')
  await expect(page.locator('#home-search-results')).toHaveCount(0)
  await search.fill('不存在的内容')
  await expect(page.getByRole('status')).toContainText('当前工作区没有匹配内容')
  await search.fill('看板')
  await page.keyboard.press('Enter')
  await expect(page).toHaveURL(/\/projects\?project=7&view=board$/)
  await expect(page.getByRole('dialog', { name: '任务详情' })).toBeVisible()
})

for (const theme of ['light', 'dark'] as const) for (const width of [1920, 1440, 1024, 390]) for (const count of [0, 1]) {
  test(`Home content density ${theme} ${width}px ${count} projects`, async ({ page }, info) => {
    await page.setViewportSize({ width, height: 900 })
    await page.addInitScript(theme => localStorage.setItem('learnos-theme', theme), theme)
    await installWorkspaceMocks(page, { projects: count ? [{ ...project, title: '把分散的个人攻略和实际生活经验整理成能够长期查阅的知识专题', description: '整理下一篇攻略，保留自己的实测结论。' }] : [], learning: [] })
    await page.goto('/')
    const projects = page.locator('.projects-section')
    await expect(projects).toBeVisible()
    await expect(page.locator('.project-row')).toHaveCount(count)
    const box = (await projects.boundingBox())!
    // No reserved project rows; desktop stretches only to the natural height of its sibling.
    expect(box.height).toBeLessThan(300)
    if (count) {
      await expect(projects).toContainText('整理下一篇攻略')
      const title = await page.locator('.project-row__text strong').boundingBox()
      const arrow = await page.locator('.project-row__arrow').boundingBox()
      expect(arrow!.x - (title!.x + title!.width)).toBeLessThanOrEqual(9)
      await page.screenshot({ path: info.outputPath(`density-${theme}-${width}-${count}.png`), fullPage: true })
      await page.locator('.project-row').click()
      await expect(page).toHaveURL(/\/projects\?project=7/)
    } else {
      await expect(projects).toContainText('还没有进行中的项目')
      await expect(page.locator('.learning-section')).toContainText('知识世界还是空的')
      expect(await page.locator('.recent-section').count()).toBe(0)
      await page.screenshot({ path: info.outputPath(`density-${theme}-${width}-${count}.png`), fullPage: true })
      await projects.getByRole('button', { name: '查看项目', exact: true }).click()
      await expect(page).toHaveURL(/\/projects\?view=list/)
    }
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true)
  })
}

for (const theme of ['light', 'dark'] as const) {
  test(`Home final details ${theme}: icon fallback and grouped counts`, async ({ page }, info) => {
    await page.setViewportSize({ width: 1440, height: 900 })
    await page.addInitScript(theme => localStorage.setItem('learnos-theme', theme), theme)
    // Real API boundaries can contain null/missing icons despite the legacy TS string type.
    const projects = [
      { title: '空白图标', icon: '  ' }, { title: '未设置图标', icon: null },
      { title: '圆点占位', icon: '·' }, { title: '保留原图', icon: '🎮' }, { title: '🍳 我的菜谱', icon: undefined },
    ].map((item, index) => ({ ...project, ...item, id: index + 10 } as unknown as Project))
    await installWorkspaceMocks(page, {
      projects,
      today: { ...emptyToday, doing: [{ ...task, id: 44, status: 'doing' }], due: [{ ...task, id: 42, due_date: '2026-09-28' }], next: [task, { ...task, id: 43 }] },
    })
    await page.goto('/')
    await expect(page.locator('.project-row__accent')).toHaveText(['空', '未', '圆', '🎮', '🍳'])
    const counts = page.locator('.today-summary')
    await expect(counts.locator('dd')).toHaveText(['1', '1', '2'])
    await expect(counts.locator('dt')).toHaveText(['进行中', '到期 / 逾期', '下一步'])
    await expect(counts.locator('.has-overdue')).toHaveCount(1)
    expect(await counts.getByRole('button').count()).toBe(0)
    const normal = await counts.locator('dd').first().evaluate(el => getComputedStyle(el).color)
    expect(await counts.locator('.has-overdue dd').evaluate(el => getComputedStyle(el).color)).not.toBe(normal)
    await page.screenshot({ path: info.outputPath(`details-${theme}.png`), fullPage: true })
  })
}
for (const width of [1440, 390]) {
  test(`Home long focus grows safely at ${width}px`, async ({ page }, info) => {
    await page.setViewportSize({ width, height: 900 })
    await installWorkspaceMocks(page, { focus: { kind: 'task', title: '将多年学习与生活经验整理为一套可以持续完善的个人资料库，同时检查每份资料的来源与自己的实践记录，保留未来回顾需要的真实背景', task_id: task.id, project_id: project.id, status: 'next' } })
    await page.goto('/')
    const focus = page.locator('.focus-card')
    await expect(focus.getByRole('link', { name: /^打开：/ })).toBeVisible()
    const bounds = (await focus.boundingBox())!
    const entry = (await focus.getByRole('link').boundingBox())!
    expect(Math.abs(entry.height - bounds.height)).toBeLessThanOrEqual(2)
    // The intentionally overflowing decorative halo contributes to scrollHeight.
    expect(await focus.evaluate(el => {
      const parent = el.getBoundingClientRect()
      return [...el.children].filter(child => !child.matches('.home-card-entry')).every(child => {
        const rect = child.getBoundingClientRect()
        return rect.top >= parent.top && rect.bottom <= parent.bottom && rect.left >= parent.left && rect.right <= parent.right
      })
    })).toBe(true)
    await page.screenshot({ path: info.outputPath(`long-focus-${width}.png`), fullPage: true })
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true)
  })
}

test.describe('Home border continuity', () => {
  test.use({ deviceScaleFactor: 2 })
  for (const theme of ['light', 'dark'] as const) {
    test(`complete base outlines in ${theme}`, async ({ page }, info) => {
      await page.setViewportSize({ width: 1440, height: 1000 })
      await page.addInitScript(theme => localStorage.setItem('learnos-theme', theme), theme)
      await installWorkspaceMocks(page, {}, { recent: recentFacts })
      await installRecentSourceMocks(page)
      await page.goto('/')
      await expect(page.locator('.home-recent-row')).toHaveCount(4)
      await page.emulateMedia({ reducedMotion: 'reduce' })
      let sharedHover: string[] | undefined
      for (const selector of ['.focus-card', '.today-section', '.projects-section', '.learning-section', '.recent-section']) {
        const card = page.locator(selector)
        await card.scrollIntoViewIfNeeded()
        await page.mouse.move(0, 0)
        const base = await card.evaluate(el => {
          const s = getComputedStyle(el)
          return {
            colors: [s.borderTopColor, s.borderRightColor, s.borderBottomColor, s.borderLeftColor],
            widths: [s.borderTopWidth, s.borderRightWidth, s.borderBottomWidth, s.borderLeftWidth],
            radii: [s.borderTopLeftRadius, s.borderTopRightRadius, s.borderBottomRightRadius, s.borderBottomLeftRadius],
            opacity: s.opacity, outline: s.outlineStyle,
          }
        })
        expect(new Set(base.colors).size).toBe(1)
        expect(base.colors[0]).not.toBe('rgba(0, 0, 0, 0)')
        expect(base.widths).toEqual(['1px', '1px', '1px', '1px'])
        expect(base.radii).toEqual(['16px', '16px', '16px', '16px'])
        expect(base.opacity).toBe('1')
        expect(base.outline).toBe('none')
        if (selector !== '.focus-card') {
          expect(await card.evaluate(el => getComputedStyle(el).backgroundClip.split(',').every(value => value.trim() === 'padding-box'))).toBe(true)
        }
        const bounds = (await card.boundingBox())!
        const name = selector.slice(1)
        await card.screenshot({ path: info.outputPath(`border-${theme}-${name}-default-2x.png`) })
        await card.hover()
        await expect.poll(() => card.evaluate(el => getComputedStyle(el).borderTopColor)).not.toBe(base.colors[0])
        const hover = await card.evaluate(el => { const s = getComputedStyle(el); return [s.borderTopColor, s.backgroundColor, s.boxShadow] })
        if (sharedHover) expect(hover).toEqual(sharedHover)
        else sharedHover = hover
        expect(await card.boundingBox()).toEqual(bounds)
        expect(await card.evaluate(el => getComputedStyle(el).borderBottomColor)).toBe(await card.evaluate(el => getComputedStyle(el).borderTopColor))
        await card.screenshot({ path: info.outputPath(`border-${theme}-${name}-hover-2x.png`) })
      }
      await expect(page.locator('.home-light-field')).toHaveCount(1)
      expect(await page.locator('.home-light-field').evaluate(el => getComputedStyle(el).pointerEvents)).toBe('none')
      await page.mouse.move(0, 0)
      await page.evaluate(() => window.scrollTo(0, 0))
      await page.screenshot({ path: info.outputPath(`border-${theme}-page.png`) })
      await page.getByRole('button', { name: '快速记录', exact: true }).click()
      await expect(page.getByRole('dialog', { name: '快速记录', exact: true })).toBeVisible()
    })
  }
})


test('首页色系支持自定义、刷新保留、主题切换与恢复默认', async ({ page }, info) => {
  await page.setViewportSize({ width: 1440, height: 1080 })
  await installWorkspaceMocks(page)
  await page.goto('/')
  async function openAppearance() {
    await page.getByRole('button', { name: '个人空间菜单' }).click()
    await page.getByRole('menuitem', { name: '外观设置', exact: true }).click()
    await expect(page.getByRole('dialog', { name: '外观设置' })).toBeVisible()
  }
  await openAppearance()
  const dialog = page.getByRole('dialog', { name: '外观设置' })
  await dialog.getByRole('button', { name: '浅色', exact: true }).click()
  await dialog.getByRole('button', { name: '玫瑰色系', exact: true }).click()
  await expect(page.locator('html')).toHaveAttribute('data-home-accent', 'custom')
  await expect(dialog.getByRole('button', { name: '玫瑰色系' })).toHaveAttribute('aria-pressed', 'true')
  await expect(page.locator('.focus-card')).toHaveCSS('--home-primary', '#be477d')
  await page.screenshot({ path: info.outputPath('appearance-desktop.png') })
  await dialog.getByRole('button', { name: '完成', exact: true }).click()
  await page.reload()
  await expect(page.locator('.focus-card')).toHaveCSS('--home-primary', '#be477d')
  await openAppearance()
  await dialog.getByLabel('自定义界面颜色').fill('#ffff00')
  await expect(page.locator('html')).toHaveCSS('--home-user-foreground', '#000000')
  await dialog.getByRole('button', { name: '深色', exact: true }).click()
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark')
  await expect(page.locator('.focus-card')).toHaveCSS('--home-primary', '#ffff00')
  await page.setViewportSize({ width: 390, height: 844 })
  await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
  await page.screenshot({ path: info.outputPath('appearance-mobile.png') })
  await dialog.getByRole('button', { name: '恢复默认色系' }).click()
  await expect(page.locator('html')).not.toHaveAttribute('data-home-accent')
  await expect(page.locator('.focus-card')).toHaveCSS('--home-primary', '#713dff')
  await dialog.getByRole('button', { name: '完成', exact: true }).click()
  await page.reload()
  await expect(page.locator('html')).not.toHaveAttribute('data-home-accent')
  await openAppearance()
  await expect(dialog.getByRole('button', { name: '默认色系', exact: true })).toHaveAttribute('aria-pressed', 'true')
})


test('损坏的色系偏好回退默认', async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem('learnos-home-accent', '#bad;display:none'))
  await installWorkspaceMocks(page)
  await page.goto('/')
  await expect(page.locator('#focus-title')).toBeVisible()
  await expect(page.locator('html')).not.toHaveAttribute('data-home-accent')
})

test('普通用户可调色，存储不可用时仍即时生效并提示', async ({ page }) => {
  await page.addInitScript(() => {
    const original = Storage.prototype.setItem
    Storage.prototype.setItem = function(key, value) {
      if (key === 'learnos-home-accent') throw new DOMException('Unavailable', 'QuotaExceededError')
      original.call(this, key, value)
    }
  })
  await installWorkspaceMocks(page)
  await page.route('**/api/v1/auth/me', route => route.fulfill({ json: { data: { id: 1, username: 'reader', role: 'user', status: 'active' } } }))
  await page.goto('/')
  await page.getByRole('button', { name: '个人空间菜单' }).click()
  await page.getByRole('menuitem', { name: '外观设置', exact: true }).click()
  const dialog = page.getByRole('dialog', { name: '外观设置' })
  await dialog.getByRole('button', { name: '冰川蓝色系', exact: true }).click()
  await expect(page.locator('.focus-card')).toHaveCSS('--home-primary', '#2864d7')
  await expect(dialog.getByRole('status')).toContainText('未允许保存')
})
