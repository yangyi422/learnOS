import { expect, test, type Page, type Route } from '@playwright/test'

/** Phase 11.4D — admin-only pilot effects on the real home and explore pages. */

const STAMP = '2026-10-21T08:00:00Z'
const DATE = '2026-10-21'

const task = (id: number, title: string, patch: Record<string, unknown> = {}) => ({
  id, project_id: 1, title, description: '', status: 'next', sort_order: id * 1024,
  priority: 'normal', due_date: null, plan_date: DATE, completed_at: null,
  created_at: STAMP, updated_at: STAMP, archived_at: null, ...patch,
})

const focusTask = task(101, '完成首页信息架构评审并给出修改清单', { status: 'doing', priority: 'high', due_date: '2026-10-23' })

const project = (id: number, title: string, context: string, open: number, done: number) => ({
  id, title, context, description: '', status: 'active', icon: title.slice(0, 1), accent: '',
  archived_at: null, tasks: [], open_task_count: open, doing_task_count: 0,
  next_task_count: open, done_task_count: done, created_at: STAMP, updated_at: STAMP,
})

const focusView = {
  id: 11, date: DATE, context: 'work', task_id: focusTask.id, revision: 1, state: 'ok',
  title: focusTask.title, task_status: focusTask.status, priority: focusTask.priority,
  plan_date: focusTask.plan_date, due_date: focusTask.due_date, completed_at: null,
  project_id: 1, project_title: '产品官网重构', project_context: 'work', project_status: 'active',
  state_reason: '', updated_at: STAMP, task_updated_at: STAMP,
}

async function mock(page: Page, role: 'admin' | 'user') {
  await page.route('**/api/v1/**', (route: Route) => {
    const path = new URL(route.request().url()).pathname
    const json = (data: unknown) => route.fulfill({ json: { data } })
    if (path === '/api/v1/auth/me') {
      return json({ id: 1, username: role, display_name: role === 'admin' ? '管理员' : '普通用户', role, status: 'active', last_login_at: null, created_at: STAMP })
    }
    if (path === '/api/v1/workspace/home') {
      return json({
        date: DATE, context: 'all', unclassified_count: 0,
        focus: { kind: 'task', title: '整理本周项目风险与依赖', task_id: 102, project_id: 1, project_title: '产品官网重构', status: 'next', priority: 'normal', due_date: null },
        daily_focus: { work: focusView, life: null },
        today: { date: DATE, doing: [focusTask], due: [], next: [] },
        planning: { items: [focusTask], due_today: [], overdue: [], previous_plan: [], summary: { planned: 1, due_today: 0, overdue: 0, previous_plan: 0, total: 1 } },
        projects: [project(1, '产品官网重构', 'work', 4, 6)],
        learning: [],
        inbox: { count: 0, items: [] },
        errors: {},
      })
    }
    if (path === '/api/v1/workspace/recent') return json({ library: [], events: [], goals: [], asset_date: '', errors: {} })
    if (path === '/api/v1/courses') return json([{ id: 1, name: '认知科学', status: 'active', learning_status: 'active', mastery_progress: 0, coverage_progress: 0, created_at: STAMP, updated_at: STAMP }])
    if (path === '/api/v1/exploration/radar') return json({ directions: [] })
    if (path === '/api/v1/exploration/questions' || path === '/api/v1/exploration/history') return json([])
    return json({})
  })
}

async function openHome(page: Page, role: 'admin' | 'user' = 'admin') {
  await mock(page, role)
  await page.goto('/', { waitUntil: 'networkidle' })
  await expect(page.locator('.daily-focus-card')).toBeVisible()
}

async function openExplore(page: Page, role: 'admin' | 'user' = 'admin') {
  await mock(page, role)
  await page.goto('/explore', { waitUntil: 'networkidle' })
  await expect(page.locator('.exploration-card').first()).toBeVisible()
}

async function openPanel(page: Page) {
  await page.getByTestId('fx-pilot-toggle').click()
  await expect(page.locator('.fx-pilot__panel')).toBeVisible()
}

test('默认页面没有任何实验效果，管理员才看到实验入口', async ({ page }) => {
  await openHome(page, 'admin')
  await expect(page.locator('.fx-spotlight')).toHaveCount(0)
  await expect(page.getByTestId('fx-pilot-bar')).toBeVisible()
})

test('普通用户看不到实验入口，伪造存储也无法启用', async ({ page }) => {
  await page.addInitScript(() => {
    sessionStorage.setItem('learnos:fx-pilot', JSON.stringify({ home: { enabled: true, intensity: 1 }, explore: { enabled: true, intensity: 1, reveal: true } }))
  })
  await openHome(page, 'user')
  await expect(page.getByTestId('fx-pilot-bar')).toHaveCount(0)
  await expect(page.locator('.fx-spotlight')).toHaveCount(0)

  await page.goto('/explore', { waitUntil: 'networkidle' })
  await expect(page.getByTestId('fx-pilot-bar')).toHaveCount(0)
  await expect(page.locator('.fx-spotlight')).toHaveCount(0)
})

test('管理员可以开启与关闭首页焦点卡反光', async ({ page }) => {
  await openHome(page, 'admin')
  await openPanel(page)
  const card = page.locator('.daily-focus-card')
  await expect(card).not.toHaveClass(/fx-spotlight/)

  await page.getByTestId('fx-pilot-home').click()
  await expect(card).toHaveClass(/fx-spotlight/)
  expect(await card.getAttribute('style')).toContain('--fx-intensity: 0.35')

  await page.getByTestId('fx-pilot-home').click()
  await expect(card).not.toHaveClass(/fx-spotlight/)
})

test('强度参数真实作用到效果', async ({ page }) => {
  await openHome(page, 'admin')
  await openPanel(page)
  await page.getByTestId('fx-pilot-home').click()
  const slider = page.getByTestId('fx-pilot-home-intensity').getByRole('slider')
  await slider.focus()
  await slider.press('ArrowRight')
  await expect.poll(async () => await page.locator('.daily-focus-card').getAttribute('style')).toContain('--fx-intensity: 0.4')
})

test('探索入口两张卡片可开启反光，Tab 内容淡入可开关', async ({ page }) => {
  await openExplore(page, 'admin')
  await openPanel(page)
  await page.getByTestId('fx-pilot-explore').click()
  await expect(page.locator('.exploration-card.fx-spotlight')).toHaveCount(2)

  await page.getByRole('tab', { name: '问题池' }).click()
  await expect(page.locator('.fx-reveal-group')).toHaveCount(2)
  await expect(page.locator('.fx-reveal-group.is-entering:visible').first()).toBeVisible({ timeout: 2000 })

  await page.getByTestId('fx-pilot-reveal').click()
  await page.getByRole('tab', { name: '探索雷达' }).click()
  await page.waitForTimeout(500)
  await expect(page.locator('.fx-reveal-group.is-entering')).toHaveCount(0)
})

test('Reduced Motion 下不启用反光', async ({ page }) => {
  await page.emulateMedia({ reducedMotion: 'reduce' })
  await openHome(page, 'admin')
  await openPanel(page)
  await page.getByTestId('fx-pilot-home').click()
  await expect(page.locator('.daily-focus-card')).not.toHaveClass(/fx-spotlight/)
})

test('离开试点页面不残留效果与监听', async ({ page }) => {
  await openHome(page, 'admin')
  await openPanel(page)
  await page.getByTestId('fx-pilot-home').click()
  await expect(page.locator('.daily-focus-card')).toHaveClass(/fx-spotlight/)
  await page.goto('/projects', { waitUntil: 'networkidle' })
  await expect(page.locator('.fx-spotlight')).toHaveCount(0)
})

test('实验开关不产生任何写请求', async ({ page }) => {
  const writes: string[] = []
  page.on('request', (request) => {
    if (request.method() !== 'GET' && request.url().includes('/api/v1/')) writes.push(`${request.method()} ${request.url()}`)
  })
  await openExplore(page, 'admin')
  await openPanel(page)
  await page.getByTestId('fx-pilot-explore').click()
  await page.getByTestId('fx-pilot-reset').click()
  expect(writes).toEqual([])
})
