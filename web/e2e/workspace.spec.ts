import { expect, test, type Page, type Route } from '@playwright/test'
import type { InboxItem } from '../src/api/inbox'
import type { Project, ProjectTask } from '../src/api/projects'
import type { WorkspaceHome, WorkspaceLearningItem } from '../src/api/workspace'

const timestamp = '2026-09-29T08:00:00Z'
const project: Project = { id: 7, title: 'LearnOS', description: '', status: 'active', icon: 'L', accent: '#3b82f6', archived_at: null, tasks: [], open_task_count: 1, next_task_count: 1, doing_task_count: 0, done_task_count: 0, created_at: timestamp, updated_at: timestamp }
const task: ProjectTask = { id: 41, project_id: 7, title: '整理看板交互', description: '', status: 'next', sort_order: 1024, priority: 'high', due_date: null, completed_at: null, created_at: timestamp, updated_at: timestamp }
const emptyToday = { date: '2026-09-29', doing: [], due: [], next: [] }

interface MockOptions {
  homeDelayMs?: number
  failHomeTimes?: number
  failCreateTimes?: number
  firstHomeOverrides?: Partial<WorkspaceHome>
}

async function installWorkspaceMocks(page: Page, homeOverrides: Partial<WorkspaceHome> = {}, options: MockOptions = {}) {
  const inbox: InboxItem[] = [{ id: 9, user_id: 1, content: '研究 SQLite WAL backup', status: 'inbox', source_type: 'manual', source_url: '', processed_to_type: '', processed_to_id: null, created_at: timestamp, updated_at: timestamp, processed_at: null, archived_at: null }]
  let homeCalls = 0
  let createFailures = 0
  await page.route('**/api/v1/**', async (route: Route) => {
    const { pathname } = new URL(route.request().url())
    const method = route.request().method()
    const reply = (data: unknown, status = 200) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify({ data }) })
    if (pathname === '/api/v1/auth/me') return reply({ id: 1, username: 'isolated-test', role: 'admin', status: 'active' })
    if (pathname === '/api/v1/workspace/home') {
      homeCalls += 1
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

test('Workspace Home 展示五个入口并能从焦点打开任务', async ({ page }, testInfo) => {
  await page.setViewportSize({ width: 1360, height: 900 })
  await page.emulateMedia({ colorScheme: 'light' })
  await installWorkspaceMocks(page)
  await page.goto('/')
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'light')
  await expect(page.locator('#focus-title')).toHaveText(task.title)
  await expect(page.locator('.focus-card__reason')).toContainText('当前优先级最高的下一步')
  await expect(page.getByRole('heading', { name: 'Today' })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Active Projects' })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Continue Learning' })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Inbox' })).toBeVisible()
  await expect(page.getByRole('textbox', { name: '快速记录到收集箱' })).toBeVisible()
  if (process.env.CAPTURE_VISUALS) await page.screenshot({ path: testInfo.outputPath('workspace-home-light.png'), fullPage: true })
  await page.emulateMedia({ colorScheme: 'dark' })
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark')
  await page.waitForTimeout(200)
  if (process.env.CAPTURE_VISUALS) await page.screenshot({ path: testInfo.outputPath('workspace-home-dark.png'), fullPage: true })
  await page.getByRole('button', { name: /继续处理/ }).click()
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
  await expect(page.locator('.focus-card').getByRole('button', { name: /继续学习/ })).toBeVisible()
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
  await page.locator('.learning-row').click()
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
  await expect(page.locator('.learning-row')).toContainText('打开地图')
  await page.locator('.learning-row').click()
  await expect(page).toHaveURL('/courses/21/map')
})

test('首页顶部快速记录仍打开原有对话框', async ({ page }) => {
  await installWorkspaceMocks(page)
  await page.goto('/')
  await page.getByRole('button', { name: '快速记录' }).click()
  await expect(page.getByRole('dialog', { name: '快速记录' })).toBeVisible()
})

test('Inbox 首页输入支持 Enter 和按钮，成功后刷新且最多显示三条', async ({ page }) => {
  await installWorkspaceMocks(page)
  await page.goto('/')
  const input = page.getByRole('textbox', { name: '快速记录到收集箱' })
  await input.fill('先通过回车记录')
  await input.press('Enter')
  await expect(page.getByText('已记录')).toBeVisible()
  await expect(input).toHaveValue('')
  await expect(page.locator('.inbox-row').filter({ hasText: '先通过回车记录' })).toBeVisible()
  await input.fill('再通过按钮记录')
  await page.getByRole('button', { name: '记录到收集箱' }).click()
  await expect(page.locator('.inbox-row').filter({ hasText: '再通过按钮记录' })).toBeVisible()
  await input.fill('第三条快速记录')
  await page.getByRole('button', { name: '记录到收集箱' }).click()
  await expect(page.locator('.inbox-row')).toHaveCount(3)
})

test('Inline Quick Capture 失败时保留输入并显示可读错误', async ({ page }) => {
  await installWorkspaceMocks(page, {}, { failCreateTimes: 1 })
  await page.goto('/')
  const input = page.getByRole('textbox', { name: '快速记录到收集箱' })
  await input.fill('不能丢失的记录')
  await input.press('Enter')
  await expect(page.getByRole('alert')).toContainText('请检查收集内容')
  await expect(input).toHaveValue('不能丢失的记录')
})

test('首页显示各区骨架，首屏请求失败后可重试', async ({ page }) => {
  await installWorkspaceMocks(page, {}, { homeDelayMs: 250, failHomeTimes: 1 })
  await page.goto('/')
  await expect(page.locator('.focus-card--skeleton')).toBeVisible()
  await expect(page.locator('.home-section--skeleton')).toHaveCount(4)
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

test('窄屏按学习优先顺序排列并让 Inbox 输入保持可用', async ({ page }) => {
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
    inbox: document.querySelector('.inbox-section')?.getBoundingClientRect().top ?? 0,
  }))
  expect(layout.width).toBeLessThanOrEqual(layout.viewport)
  expect(layout.focus).toBeLessThan(layout.today)
  expect(layout.today).toBeLessThan(layout.learning)
  expect(layout.learning).toBeLessThan(layout.projects)
  expect(layout.projects).toBeLessThan(layout.inbox)
  await expect(page.getByRole('textbox', { name: '快速记录到收集箱' })).toBeVisible()
  await expect(page.getByRole('button', { name: '记录到收集箱' })).toBeVisible()
})

test('空工作区不虚构数据并保留学习创建入口和 Inbox 输入', async ({ page }) => {
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
  await expect(page.getByRole('textbox', { name: '快速记录到收集箱' })).toBeVisible()
  await expect(page.getByRole('heading', { name: '收集箱是空的' })).toBeVisible()
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
