import { expect, test, type Page, type Route } from '@playwright/test'

const timestamp = '2026-09-24T08:00:00Z'
function localDate(date: Date) { return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}` }
const today = localDate(new Date())

type MockTask = {
  id: number; project_id: number; title: string; description: string; status: string; sort_order: number
  priority: string; due_date: string | null; completed_at: string | null; created_at: string; updated_at: string
}
const initialTasks: MockTask[] = [
  { id: 1, project_id: 1, title: 'UI 重构', description: '', status: 'inbox', sort_order: 1024, priority: 'normal', due_date: null, completed_at: null, created_at: timestamp, updated_at: timestamp },
  { id: 2, project_id: 1, title: '云部署', description: '', status: 'next', sort_order: 1024, priority: 'high', due_date: today, completed_at: null, created_at: timestamp, updated_at: timestamp },
  { id: 3, project_id: 2, title: '角色动画', description: '', status: 'doing', sort_order: 1024, priority: 'normal', due_date: null, completed_at: null, created_at: timestamp, updated_at: timestamp },
  { id: 4, project_id: 2, title: '项目初始化', description: '', status: 'done', sort_order: 1024, priority: 'low', due_date: null, completed_at: timestamp, created_at: timestamp, updated_at: timestamp },
]
const initialProjects = [
  { id: 1, title: 'LearnOS', description: '完成 v0.1', status: 'active', icon: '', accent: '#6b9dcc', archived_at: null, tasks: [], created_at: timestamp, updated_at: timestamp },
  { id: 2, title: '像素团团', description: '', status: 'active', icon: '', accent: '#7ca98e', archived_at: null, tasks: [], created_at: timestamp, updated_at: timestamp },
]

async function mockWorkspace(page: Page) {
  const tasks = structuredClone(initialTasks)
  const projects = structuredClone(initialProjects)
  let failMove = false
  let moveCount = 0
  await page.route('**/api/v1/**', async (route: Route) => {
    const request = route.request()
    const url = new URL(request.url())
    const path = url.pathname
    const method = request.method()
    const reply = (body: unknown, status = 200) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) })
    if (path === '/api/v1/auth/me') return reply({ data: { id: 1, username: 'test', role: 'admin', status: 'active' } })
    if (path === '/api/v1/projects' && method === 'GET') {
      return reply({ data: projects.map(project => ({
        ...project,
        open_task_count: tasks.filter(task => task.project_id === project.id && task.status !== 'done').length,
        doing_task_count: tasks.filter(task => task.project_id === project.id && task.status === 'doing').length,
        done_task_count: tasks.filter(task => task.project_id === project.id && task.status === 'done').length,
      })) })
    }
    if (path === '/api/v1/projects' && method === 'POST') {
      const input = request.postDataJSON() as Partial<(typeof projects)[number]>
      const project = { id: projects.length + 1, title: input.title ?? '', description: input.description ?? '', status: 'active', icon: input.icon ?? '', accent: input.accent ?? '', archived_at: null, tasks: [], created_at: timestamp, updated_at: timestamp }
      projects.push(project)
      return reply({ data: project }, 201)
    }
    const projectPath = path.match(/^\/api\/v1\/projects\/(\d+)$/)
    if (projectPath && method === 'PATCH') {
      const project = projects.find(item => item.id === Number(projectPath[1]))!
      Object.assign(project, request.postDataJSON())
      return reply({ data: { updated: true } })
    }
    if (path === '/api/v1/tasks' && method === 'GET') {
      const projectID = Number(url.searchParams.get('project_id'))
      const status = url.searchParams.get('status')
      const limit = Number(url.searchParams.get('limit') || 0)
      const offset = Number(url.searchParams.get('offset') || 0)
      const filtered = tasks.filter(task => (!projectID || task.project_id === projectID) && (!status || task.status === status))
        .sort((a, b) => status === 'done' && limit ? b.sort_order - a.sort_order : a.sort_order - b.sort_order)
      return reply({ data: limit ? filtered.slice(offset, offset + limit) : filtered.slice(offset) })
    }
    if (path === '/api/v1/tasks' && method === 'POST') {
      const input = request.postDataJSON() as Partial<MockTask>
      const task: MockTask = { id: tasks.length + 1, project_id: input.project_id!, title: input.title!, description: input.description ?? '', status: input.status ?? 'inbox', sort_order: 2048, priority: input.priority ?? 'normal', due_date: input.due_date ?? null, completed_at: null, created_at: timestamp, updated_at: timestamp }
      tasks.push(task)
      return reply({ data: task }, 201)
    }
    const taskPath = path.match(/^\/api\/v1\/tasks\/(\d+)$/)
    if (taskPath && method === 'PATCH') {
      const task = tasks.find(item => item.id === Number(taskPath[1]))!
      const input = request.postDataJSON() as Partial<MockTask>
      const wasDone = task.status === 'done'
      Object.assign(task, input)
      if (task.status === 'done' && !wasDone) task.completed_at = timestamp
      if (task.status !== 'done' && wasDone) task.completed_at = null
      return reply({ data: task })
    }
    if (taskPath && method === 'DELETE') {
      tasks.splice(tasks.findIndex(item => item.id === Number(taskPath[1])), 1)
      return reply({ data: { deleted: true } })
    }
    const move = path.match(/^\/api\/v1\/tasks\/(\d+)\/move$/)
    if (move && method === 'PATCH') {
      moveCount++
      if (failMove) return reply({ error: 'project operation failed' }, 500)
      const task = tasks.find(item => item.id === Number(move[1]))!
      const input = request.postDataJSON() as { status: string }
      task.status = input.status
      task.sort_order += 1024
      task.completed_at = input.status === 'done' ? timestamp : null
      return reply({ data: task })
    }
    return reply({ data: {} })
  })
  return { tasks, projects, setFailMove(value: boolean) { failMove = value }, get moveCount() { return moveCount } }
}

test('defaults to all projects, retains a selected project and groups Today without duplicates', async ({ page }) => {
  await mockWorkspace(page)
  await page.goto('/projects')
  await expect(page.getByRole('heading', { name: '项目', exact: true })).toBeVisible()
  await expect(page.locator('.task-card')).toHaveCount(4)
  await expect(page.locator('.task-card').filter({ hasText: 'UI 重构' })).toContainText('LearnOS')
  await page.locator('.projects-switcher').click()
  await page.getByText('LearnOS · 2 项未完成').click()
  await expect(page).toHaveURL(/project=1/)
  await expect(page.locator('.task-card')).toHaveCount(2)
  await page.reload()
  await expect(page.locator('.task-card')).toHaveCount(2)
  await page.getByRole('button', { name: '今天', exact: true }).click()
  await expect(page.locator('.today-task')).toHaveCount(1)
  await expect(page.locator('.today-task')).toContainText('云部署')
  await page.locator('.projects-switcher').click()
  await page.getByText('全部项目').last().click()
  await expect(page.locator('.today-task')).toHaveCount(2)
})

test('moves a card by ID and restores it when the move API fails', async ({ page }) => {
  const workspace = await mockWorkspace(page)
  await page.goto('/projects')
  const inbox = page.locator('.kanban-column').nth(0)
  const next = page.locator('.kanban-column').nth(1)
  await inbox.locator('.task-card__handle').first().dragTo(next.locator('.kanban-column__cards'))
  await expect(next.locator('.task-card').filter({ hasText: 'UI 重构' })).toBeVisible()
  await expect.poll(() => workspace.moveCount).toBe(1)
  await page.reload()
  await expect(next.locator('.task-card').filter({ hasText: 'UI 重构' })).toBeVisible()
  workspace.setFailMove(true)
  await next.locator('.task-card').filter({ hasText: 'UI 重构' }).locator('.task-card__handle').dragTo(inbox.locator('.kanban-column__cards'))
  await expect(page.getByText('任务状态更新失败，已恢复原位置。')).toBeVisible()
  await expect(next.locator('.task-card').filter({ hasText: 'UI 重构' })).toBeVisible()
})

test('creates and completes a task in the drawer and archives a project', async ({ page }) => {
  await mockWorkspace(page)
  await page.goto('/projects')
  await page.getByRole('button', { name: '+ 新建任务' }).click()
  const drawer = page.locator('.el-drawer.rtl')
  await drawer.locator('input').first().fill('修复地图节点')
  await drawer.locator('.el-select').first().click()
  await page.getByRole('option', { name: 'LearnOS' }).click()
  await drawer.getByRole('button', { name: '创建任务' }).click()
  await expect(page.locator('.task-card').filter({ hasText: '修复地图节点' })).toBeVisible()
  await page.locator('.task-card').filter({ hasText: '修复地图节点' }).locator('.task-card__body').click()
  await drawer.getByRole('button', { name: '完成', exact: true }).click()
  await expect(page.locator('.kanban-column').nth(3).locator('.task-card').filter({ hasText: '修复地图节点' })).toBeVisible()
  await page.getByRole('button', { name: '项目列表', exact: true }).click()
  const card = page.locator('.project-list-card').filter({ hasText: '像素团团' })
  await card.getByRole('button', { name: '归档' }).click()
  await expect(card).toContainText('已归档')
  await page.setViewportSize({ width: 375, height: 812 })
  await page.getByRole('button', { name: '看板', exact: true }).click()
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth + 1)).toBe(true)
  await page.locator('.task-card').filter({ hasText: 'UI 重构' }).locator('.task-card__body').click()
  await expect.poll(async () => (await drawer.boundingBox())?.width ?? 0).toBeGreaterThanOrEqual(370)
})
