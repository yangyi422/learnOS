import { expect, test, type Page, type Route } from '@playwright/test'

/** Phase 11.5C — admin-only motion pilot on the projects board. */

const STAMP = '2026-10-21T08:00:00Z'

const tasks = [
  { id: 1, project_id: 1, title: 'UI 重构', description: '', status: 'inbox', sort_order: 1024, priority: 'normal', due_date: null, completed_at: null, created_at: STAMP, updated_at: STAMP, archived_at: null },
  { id: 2, project_id: 1, title: '云部署', description: '', status: 'next', sort_order: 1024, priority: 'high', due_date: null, completed_at: null, created_at: STAMP, updated_at: STAMP, archived_at: null },
  { id: 3, project_id: 2, title: '角色动画', description: '', status: 'doing', sort_order: 1024, priority: 'normal', due_date: null, completed_at: null, created_at: STAMP, updated_at: STAMP, archived_at: null },
]

const projects = [
  { id: 1, title: 'LearnOS', description: '完成 v0.1', status: 'active', icon: '', accent: '#6b9dcc', archived_at: null, tasks: [], created_at: STAMP, updated_at: STAMP },
  { id: 2, title: '像素团团', description: '', status: 'active', icon: '', accent: '#7ca98e', archived_at: null, tasks: [], created_at: STAMP, updated_at: STAMP },
]

async function mock(page: Page, role: 'admin' | 'user' = 'admin') {
  await page.route('**/api/v1/**', (route: Route) => {
    const url = new URL(route.request().url())
    const path = url.pathname
    const reply = (data: unknown) => route.fulfill({ json: { data } })
    if (path === '/api/v1/auth/me') {
      return reply({ id: 1, username: role, display_name: role === 'admin' ? '管理员' : '普通用户', role, status: 'active', last_login_at: null, created_at: STAMP })
    }
    if (path === '/api/v1/projects') {
      return reply(projects.map((project) => ({
        ...project,
        open_task_count: tasks.filter((task) => task.project_id === project.id).length,
        next_task_count: 1,
        doing_task_count: 1,
        done_task_count: 0,
        archived_task_count: 0,
      })))
    }
    if (path === '/api/v1/projects/today') {
      return reply({ date: '2026-10-21', doing: [], due: [], next: [] })
    }
    if (path === '/api/v1/tasks') return reply(tasks)
    return reply({})
  })
}

async function openBoard(page: Page, role: 'admin' | 'user' = 'admin') {
  await mock(page, role)
  await page.goto('/projects', { waitUntil: 'networkidle' })
  await expect(page.locator('.task-card').first()).toBeVisible()
}

async function openPanel(page: Page) {
  await page.getByTestId('fx-pilot-toggle').click()
  await expect(page.locator('.fx-pilot__panel')).toBeVisible()
}

test('默认看板没有任何实验类，管理员可见实验入口', async ({ page }) => {
  await openBoard(page, 'admin')
  await expect(page.getByTestId('fx-pilot-bar')).toBeVisible()
  await expect(page.locator('.task-card--motion')).toHaveCount(0)
  await expect(page.locator('.projects-tabs--motion')).toHaveCount(0)
})

test('普通用户看不到入口，伪造存储也不生效', async ({ page }) => {
  await page.addInitScript(() => {
    sessionStorage.setItem('learnos:fx-pilot', JSON.stringify({ projects: { cards: true, tabs: true, drawer: true } }))
  })
  await openBoard(page, 'user')
  await expect(page.getByTestId('fx-pilot-bar')).toHaveCount(0)
  await expect(page.locator('.task-card--motion')).toHaveCount(0)
  await expect(page.locator('.projects-tabs--motion')).toHaveCount(0)
})

test('卡片反馈开启后使用 Motion Token 并保持拖拽类独立', async ({ page }) => {
  await openBoard(page, 'admin')
  await openPanel(page)
  await page.getByTestId('fx-pilot-projects-cards').click()
  const card = page.locator('.task-card').first()
  await expect(card).toHaveClass(/task-card--motion/)
  await card.hover()
  const restTransform = await card.evaluate((element) => getComputedStyle(element).transform)
  await page.mouse.down()
  const pressedTransform = await card.evaluate((element) => getComputedStyle(element).transform)
  await page.mouse.up()
  expect(pressedTransform).not.toBe(restTransform)
  expect(pressedTransform).toContain('1')
  // The click may open the task drawer; close it before touching the panel again.
  await page.keyboard.press('Escape')
  await expect(page.locator('.task-drawer')).toBeHidden()
  await page.getByTestId('fx-pilot-projects-cards').click()
  await expect(card).not.toHaveClass(/task-card--motion/)
})

test('Reduced Motion 下卡片过渡归零', async ({ page }) => {
  await page.emulateMedia({ reducedMotion: 'reduce' })
  await openBoard(page, 'admin')
  await openPanel(page)
  await page.getByTestId('fx-pilot-projects-cards').click()
  const duration = await page.locator('.task-card--motion').first().evaluate((element) => getComputedStyle(element).transitionDuration)
  const seconds = Math.max(...duration.split(',').map((value) => parseFloat(value)))
  expect(seconds).toBeLessThan(0.01)
})

test('视图切换过渡可开启，Tab 与内容仍可用', async ({ page }) => {
  await openBoard(page, 'admin')
  await openPanel(page)
  await page.getByTestId('fx-pilot-projects-tabs').click()
  await expect(page.locator('.projects-tabs--motion')).toHaveCount(1)
  await page.locator('.projects-tabs').getByText('今天', { exact: true }).click()
  await expect(page).toHaveURL(/view=today/)
  await expect(page.locator('.today-view')).toBeVisible()
  await page.locator('.projects-tabs').getByText('看板', { exact: true }).click()
  await expect(page.locator('.task-card').first()).toBeVisible()
})

test('抽屉节奏开启后任务详情仍可打开与关闭', async ({ page }) => {
  await openBoard(page, 'admin')
  await openPanel(page)
  await page.getByTestId('fx-pilot-projects-drawer').click()
  await page.locator('.task-card').first().click()
  await expect(page.locator('.task-drawer')).toBeVisible()
  await expect(page.locator('.task-drawer')).toHaveClass(/task-drawer--motion/)
  await page.keyboard.press('Escape')
  await expect(page.locator('.task-drawer')).toBeHidden()
})

test('实验开关不产生写请求', async ({ page }) => {
  const writes: string[] = []
  page.on('request', (request) => {
    if (request.method() !== 'GET' && request.url().includes('/api/v1/')) writes.push(`${request.method()} ${request.url()}`)
  })
  await openBoard(page, 'admin')
  await openPanel(page)
  await page.getByTestId('fx-pilot-projects-cards').click()
  await page.getByTestId('fx-pilot-projects-tabs').click()
  await page.getByTestId('fx-pilot-reset').click()
  expect(writes).toEqual([])
})
