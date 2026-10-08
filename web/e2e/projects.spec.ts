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
  let failSave = false
  let saveDelay = 0
  let saveCount = 0
  let moveCount = 0
  let moveDelay = 0
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
        next_task_count: tasks.filter(task => task.project_id === project.id && task.status === 'next').length,
        doing_task_count: tasks.filter(task => task.project_id === project.id && task.status === 'doing').length,
        done_task_count: tasks.filter(task => task.project_id === project.id && task.status === 'done').length,
      })) })
    }
    if (path === '/api/v1/projects/today' && method === 'GET') {
      const date = url.searchParams.get('date') ?? today
      const projectID = Number(url.searchParams.get('project_id'))
      const active = tasks.filter(task => (!projectID || task.project_id === projectID) && projects.some(project => project.id === task.project_id && project.status === 'active') && task.status !== 'done')
      return reply({ data: {
        date,
        doing: active.filter(task => task.status === 'doing'),
        due: active.filter(task => task.status !== 'doing' && task.due_date && task.due_date <= date),
        next: active.filter(task => task.status === 'next' && (!task.due_date || task.due_date > date)),
      } })
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
      saveCount++
      if (saveDelay) await new Promise(resolve => setTimeout(resolve, saveDelay))
      if (failSave) return reply({ error: '任务保存失败' }, 500)
      const input = request.postDataJSON() as Partial<MockTask>
      const task: MockTask = { id: tasks.length + 1, project_id: input.project_id!, title: input.title!, description: input.description ?? '', status: input.status ?? 'inbox', sort_order: 2048, priority: input.priority ?? 'normal', due_date: input.due_date ?? null, completed_at: null, created_at: timestamp, updated_at: timestamp }
      tasks.push(task)
      return reply({ data: task }, 201)
    }
    const taskPath = path.match(/^\/api\/v1\/tasks\/(\d+)$/)
    if (taskPath && method === 'PATCH') {
      saveCount++
      if (saveDelay) await new Promise(resolve => setTimeout(resolve, saveDelay))
      if (failSave) return reply({ error: '任务保存失败' }, 500)
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
      if (moveDelay) await new Promise(resolve => setTimeout(resolve, moveDelay))
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
  return {
    tasks,
    projects,
    setMoveDelay(value: number) { moveDelay = value },
    setFailMove(value: boolean) { failMove = value },
    setFailSave(value: boolean) { failSave = value },
    setSaveDelay(value: number) { saveDelay = value },
    get moveCount() { return moveCount },
    get saveCount() { return saveCount },
  }
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
  await expect(page.locator('.today-task')).toHaveCount(2)
  await expect(page.locator('.today-view')).toContainText('云部署')
  await expect(page.locator('.projects-switcher')).toHaveCount(0)
  await page.getByRole('button', { name: '看板', exact: true }).click()
  await expect(page.locator('.task-card')).toHaveCount(2)
})

test('keeps manual theme choice and follows system when requested', async ({ page }, testInfo) => {
  await page.emulateMedia({ colorScheme: 'dark' })
  await mockWorkspace(page)
  await page.goto('/settings')
  await page.getByRole('button', { name: '深色主题' }).click()
  await page.goto('/projects')
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark')
  await expect(page.locator('.task-card')).toHaveCount(4)
  await expect(page.getByRole('navigation', { name: '主要导航' }).getByRole('link', { name: '项目' })).toHaveAttribute('aria-current', 'page')
  const boardTop = (await page.locator('.kanban-column__header').first().boundingBox())?.y ?? 0
  expect(boardTop).toBeGreaterThanOrEqual(180)
  expect(boardTop).toBeLessThanOrEqual(220)
  expect(await page.locator('.sidebar').evaluate(element => element.scrollWidth <= element.clientWidth + 1)).toBe(true)
  if (process.env.CAPTURE_VISUALS) await page.screenshot({ path: testInfo.outputPath('workspace-dark.png'), fullPage: true })
  await page.locator('.task-card').filter({ hasText: '云部署' }).locator('.task-card__body').click()
  const drawer = page.locator('.task-drawer.el-drawer')
  await expect(drawer).toBeVisible()
  await expect.poll(async () => (await drawer.boundingBox())?.x ?? 2000).toBeLessThan(800)
  if (process.env.CAPTURE_VISUALS) await page.screenshot({ path: testInfo.outputPath('task-drawer-dark.png'), fullPage: true })
  await drawer.getByRole('button', { name: '编辑', exact: true }).click()
  await drawer.locator('.el-select').last().click()
  const selectPopper = page.locator('.el-popper.task-select-popper:visible')
  await expect(selectPopper).toBeVisible()
  await expect(selectPopper).toHaveCSS('opacity', '1')
  if (process.env.CAPTURE_VISUALS) await page.screenshot({ path: testInfo.outputPath('task-select-dark.png'), fullPage: true })
  await drawer.locator('.task-drawer__header h2').click()
  await drawer.locator('.el-date-editor').click()
  const datePopper = page.locator('.el-popper.task-date-popper:visible')
  await expect(datePopper).toBeVisible()
  await expect(datePopper).toHaveCSS('opacity', '1')
  if (process.env.CAPTURE_VISUALS) await page.screenshot({ path: testInfo.outputPath('task-date-dark.png'), fullPage: true })
  await drawer.getByRole('textbox', { name: '任务名称' }).fill('暂存修改')
  await drawer.getByRole('button', { name: '关闭任务详情' }).click()
  await expect(page.locator('.el-message-box')).toBeVisible()
  await expect.poll(async () => page.locator('.el-overlay-message-box').evaluate(element => getComputedStyle(element).opacity)).toBe('1')
  if (process.env.CAPTURE_VISUALS) await page.screenshot({ path: testInfo.outputPath('task-confirm-dark.png'), fullPage: true })
  await page.getByRole('button', { name: '放弃修改' }).click()
  await expect(drawer).not.toBeVisible()
  await page.setViewportSize({ width: 375, height: 812 })
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth + 1)).toBe(true)
  await expect(page.locator('.kanban-scroll')).toBeVisible()
  if (process.env.CAPTURE_VISUALS) await page.screenshot({ path: testInfo.outputPath('workspace-mobile.png'), fullPage: true })
  await page.setViewportSize({ width: 1280, height: 720 })

  await page.goto('/settings')
  await page.getByRole('button', { name: '浅色主题' }).click()
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'light')
  await expect.poll(() => page.evaluate(() => localStorage.getItem('learnos-theme'))).toBe('light')
  await page.goto('/projects')
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'light')
  if (process.env.CAPTURE_VISUALS) await page.screenshot({ path: testInfo.outputPath('workspace-light.png'), fullPage: true })
  await page.locator('.task-card').filter({ hasText: '云部署' }).locator('.task-card__body').click()
  await expect(drawer).toBeVisible()
  await expect.poll(async () => (await drawer.boundingBox())?.x ?? 2000).toBeLessThan(800)
  if (process.env.CAPTURE_VISUALS) await page.screenshot({ path: testInfo.outputPath('task-drawer-light.png'), fullPage: true })
  await drawer.getByRole('button', { name: '编辑', exact: true }).click()
  await drawer.locator('.el-select').last().click()
  await expect(selectPopper).toBeVisible()
  await expect(selectPopper).toHaveCSS('opacity', '1')
  if (process.env.CAPTURE_VISUALS) await page.screenshot({ path: testInfo.outputPath('task-select-light.png'), fullPage: true })
  await drawer.locator('.task-drawer__header h2').click()
  await drawer.locator('.el-date-editor').click()
  await expect(datePopper).toBeVisible()
  await expect(datePopper).toHaveCSS('opacity', '1')
  if (process.env.CAPTURE_VISUALS) await page.screenshot({ path: testInfo.outputPath('task-date-light.png'), fullPage: true })
  await drawer.getByRole('textbox', { name: '任务名称' }).fill('暂存修改')
  await drawer.getByRole('button', { name: '关闭任务详情' }).click()
  await expect(page.locator('.el-message-box')).toBeVisible()
  await expect.poll(async () => page.locator('.el-overlay-message-box').evaluate(element => getComputedStyle(element).opacity)).toBe('1')
  if (process.env.CAPTURE_VISUALS) await page.screenshot({ path: testInfo.outputPath('task-confirm-light.png'), fullPage: true })
  await page.getByRole('button', { name: '放弃修改' }).click()
  await expect(drawer).not.toBeVisible()

  await page.goto('/settings')
  await page.getByRole('button', { name: '深色主题' }).click()
  await page.emulateMedia({ colorScheme: 'light' })
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark')
  await page.getByRole('button', { name: '跟随系统' }).click()
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'light')
  await page.emulateMedia({ colorScheme: 'dark' })
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark')
})

test('moves a card by ID and restores it when the move API fails', async ({ page }) => {
  const workspace = await mockWorkspace(page)
  await page.goto('/projects')
  const inbox = page.locator('.kanban-column').nth(0)
  const next = page.locator('.kanban-column').nth(1)
  await inbox.locator('.task-card').first().dragTo(next.locator('.kanban-column__cards'))
  await expect(next.locator('.task-card').filter({ hasText: 'UI 重构' })).toBeVisible()
  await expect.poll(() => workspace.moveCount).toBe(1)
  await page.reload()
  await expect(next.locator('.task-card').filter({ hasText: 'UI 重构' })).toBeVisible()
  workspace.setFailMove(true)
  await next.locator('.task-card').filter({ hasText: 'UI 重构' }).dragTo(inbox.locator('.kanban-column__cards'))
  await expect(page.getByText('任务状态更新失败，已恢复原位置。')).toBeVisible()
  await expect(next.locator('.task-card').filter({ hasText: 'UI 重构' })).toBeVisible()
})

test('creates and completes a task in the drawer and archives a project', async ({ page }) => {
  await mockWorkspace(page)
  await page.goto('/projects')
  await page.getByRole('button', { name: '+ 新建任务' }).click()
  const drawer = page.locator('.el-drawer.rtl')
  await drawer.locator('.el-date-editor').click()
  await expect(page.locator('.el-picker-panel')).toContainText('年')
  await expect(page.locator('.el-picker-panel')).toContainText('月')
  await drawer.getByText('到期日').click()
  await drawer.locator('input').first().fill('修复地图节点')
  await drawer.locator('.el-select').first().click()
  await page.getByRole('option', { name: 'LearnOS' }).click()
  await drawer.getByRole('button', { name: '创建任务' }).click()
  await expect(page.locator('.task-card').filter({ hasText: '修复地图节点' })).toBeVisible()
  await page.locator('.task-card').filter({ hasText: '修复地图节点' }).locator('.task-card__body').click()
  await drawer.getByRole('button', { name: '标记完成', exact: true }).click()
  await expect(page.locator('.kanban-column').nth(3).locator('.task-card').filter({ hasText: '修复地图节点' })).toBeVisible()
  await drawer.getByRole('button', { name: '关闭任务详情' }).click()
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

test('task card handles long content and drawer protects edits while preserving save and delete', async ({ page }) => {
  const workspace = await mockWorkspace(page)
  workspace.tasks[0].title = '整理一个足够长的任务标题，确认它能在窄看板列中自然换行并保持操作入口可用'
  workspace.tasks[0].description = '这是一段任务背景，卡片只展示简短摘要，完整内容仍在详情抽屉中编辑。'
  await page.goto('/projects')
  const card = page.locator('.task-card').filter({ hasText: workspace.tasks[0].title })
  await expect(card.locator('.task-card__summary')).toBeVisible()
  expect(await card.evaluate(element => element.scrollWidth <= element.clientWidth + 1)).toBe(true)
  await expect(card.locator('.task-card__due')).toHaveCount(0)
  await card.locator('.task-card__body').click()
  const drawer = page.locator('.task-drawer.el-drawer')
  await expect(card).toHaveClass(/task-card--selected/)
  await expect(drawer.locator('input, textarea')).toHaveCount(0)
  await drawer.getByRole('button', { name: '编辑', exact: true }).click()
  await drawer.getByRole('textbox', { name: '任务名称' }).fill('修改后的任务')
  await drawer.getByRole('button', { name: '取消' }).click()
  await expect(page.locator('.el-message-box')).toContainText('当前修改尚未保存')
  await page.getByRole('button', { name: '继续编辑' }).click()
  await expect(drawer).toBeVisible()
  await drawer.getByRole('button', { name: '关闭任务详情' }).click()
  await page.getByRole('button', { name: '放弃修改' }).click()
  await expect(drawer).not.toBeVisible()
  await expect(card).toContainText(workspace.tasks[0].title)

  await page.locator('.task-card').filter({ hasText: '云部署' }).locator('.task-card__body').click()
  await drawer.getByRole('button', { name: '编辑', exact: true }).click()
  await drawer.locator('textarea').fill('保留原有到期日并补充说明')
  await drawer.getByRole('button', { name: '保存', exact: true }).click()
  const saved = page.locator('.task-card').filter({ hasText: '云部署' })
  await expect(saved.locator('.task-card__summary')).toHaveText('保留原有到期日并补充说明')
  await expect(saved.locator('.task-card__due')).toBeVisible()
  await expect(drawer.locator('.task-drawer__read-description')).toHaveText('保留原有到期日并补充说明')
  await expect(drawer.locator('input, textarea')).toHaveCount(0)
  await drawer.getByRole('button', { name: '删除任务' }).click()
  await expect(page.locator('.el-message-box')).toContainText('删除任务')
  await page.locator('.el-message-box__btns .el-button--primary').click()
  await expect(saved).toHaveCount(0)
})

test('task drawer keeps focus, long descriptions and failed saves stable', async ({ page }, testInfo) => {
  await page.emulateMedia({ reducedMotion: 'reduce' })
  const workspace = await mockWorkspace(page)
  await page.goto('/projects')
  const trigger = page.locator('.task-card').filter({ hasText: '云部署' }).locator('.task-card__body')
  await trigger.hover()
  await expect(trigger.locator('..').locator('..')).toHaveCSS('transform', 'none')
  await trigger.click()

  const drawer = page.locator('.task-drawer.el-drawer')
  await expect(drawer.locator('.task-drawer__reading h3')).toBeFocused()
  await drawer.getByRole('button', { name: '编辑', exact: true }).click()
  const title = drawer.getByRole('textbox', { name: '任务名称' })
  await expect(title).toBeFocused()
  expect(await drawer.evaluate(element => Number.parseFloat(getComputedStyle(element).transitionDuration))).toBeLessThan(.02)
  await title.fill('需要确认的修改')
  await page.keyboard.press('Escape')
  await expect(page.locator('.el-message-box')).toContainText('当前修改尚未保存')
  await page.getByRole('button', { name: '继续编辑' }).click()
  await expect(drawer).toBeVisible()
  await title.fill('云部署')
  await page.keyboard.press('Escape')
  await expect(drawer).not.toBeVisible()
  await expect(trigger).toBeFocused()

  await page.setViewportSize({ width: 375, height: 667 })
  await trigger.click()
  await drawer.getByRole('button', { name: '编辑', exact: true }).click()
  const description = drawer.locator('textarea')
  const emptyDescriptionHeight = await description.evaluate(element => element.clientHeight)
  const longDescription = Array.from({ length: 80 }, (_, index) => `第 ${index + 1} 行任务说明，用于验证长文本滚动保持稳定。`).join('\n')
  await description.fill(longDescription)
  await expect.poll(() => description.evaluate(element => element.clientHeight)).toBeGreaterThan(emptyDescriptionHeight)
  expect(await description.evaluate(element => element.scrollHeight > element.clientHeight)).toBe(true)
  const footer = drawer.locator('.task-drawer__actions')
  await expect(footer).toBeVisible()
  const footerBox = await footer.boundingBox()
  expect((footerBox?.y ?? 1000) + (footerBox?.height ?? 1000)).toBeLessThanOrEqual(667)
  if (process.env.CAPTURE_VISUALS) await page.screenshot({ path: testInfo.outputPath('task-drawer-mobile-long.png'), fullPage: true })

  await description.fill('长'.repeat(9000))
  await drawer.locator('.task-drawer__header h2').click()
  await expect(drawer.locator('.el-input__count')).toHaveCSS('opacity', '1')
  await description.fill('保存失败后应保留的说明')
  workspace.setFailSave(true)
  workspace.setSaveDelay(250)
  await drawer.getByRole('button', { name: '保存', exact: true }).click()
  await expect(drawer.getByRole('button', { name: '保存', exact: true })).toBeDisabled()
  await expect(drawer.getByRole('button', { name: '保存', exact: true })).toHaveClass(/is-loading/)
  await expect(page.locator('.el-message--error')).toBeVisible()
  expect(workspace.saveCount).toBe(1)
  await expect(description).toHaveValue('保存失败后应保留的说明')
  await expect(drawer).toBeVisible()

  workspace.setFailSave(false)
  workspace.setSaveDelay(0)
  await drawer.getByRole('button', { name: '保存', exact: true }).click()
  await expect(drawer.locator('.task-drawer__read-description')).toHaveText('保存失败后应保留的说明')
  await expect(drawer.locator('textarea')).toHaveCount(0)
  await expect(page.locator('.task-card').filter({ hasText: '云部署' }).locator('.task-card__summary')).toHaveText('保存失败后应保留的说明')
})

test('read first, cancel edits restores persisted task and completing keeps details readable', async ({ page }) => {
  const workspace = await mockWorkspace(page)
  workspace.tasks[0].description = '第一段完整说明。\n第二段说明仍然可阅读。'
  await page.goto('/projects')
  await page.locator('[data-task-id="1"]').click()
  const drawer = page.locator('.task-drawer')
  await expect(drawer.locator('.task-drawer__read-description')).toHaveText(workspace.tasks[0].description)
  await expect(drawer.locator('input, textarea')).toHaveCount(0)
  await drawer.getByRole('button', { name: '编辑', exact: true }).click()
  await drawer.getByRole('textbox', { name: '任务名称' }).fill('不保存的标题')
  await drawer.getByRole('button', { name: '取消', exact: true }).click()
  await page.getByRole('button', { name: '放弃修改' }).click()
  await expect(drawer.locator('.task-drawer__reading h3')).toHaveText('UI 重构')
  expect(workspace.saveCount).toBe(0)
  await drawer.getByRole('button', { name: '标记完成' }).click()
  await expect(drawer.locator('.task-drawer__metadata')).toContainText('已完成')
  await expect(drawer.getByRole('button', { name: '重新打开' })).toBeVisible()
})

test('Today uses all active projects, gives due dates precedence, expands next and completes without duplicates', async ({ page }) => {
  const workspace = await mockWorkspace(page)
  workspace.tasks[0].status = 'doing'
  workspace.tasks[0].due_date = '2000-01-01'
  workspace.tasks[2].due_date = today
  for (let id = 10; id < 20; id++) workspace.tasks.push({ ...initialTasks[1]!, id, title: `候选 ${id}`, due_date: null })
  await page.goto('/projects?project=1&view=today')
  const overdue = page.locator('.today-section').filter({ has: page.getByRole('heading', { name: '已逾期', exact: true }) })
  const due = page.locator('.today-section').filter({ has: page.getByRole('heading', { name: '今天到期', exact: true }) })
  const doing = page.locator('.today-section').filter({ has: page.getByRole('heading', { name: '正在进行', exact: true }) })
  const next = page.locator('.today-section').filter({ has: page.getByRole('heading', { name: '下一步', exact: true }) })
  await expect(overdue.locator('.today-task')).toHaveCount(1)
  await expect(due.locator('.today-task')).toHaveCount(2)
  await expect(doing.locator('.today-task')).toHaveCount(0)
  await expect(next.locator('.today-task')).toHaveCount(6)
  await page.getByRole('button', { name: '查看全部下一步（10）' }).click()
  await expect(next.locator('.today-task')).toHaveCount(10)
  await expect(page.locator('.today-task').filter({ hasText: '项目初始化' })).toHaveCount(0)
  await page.getByRole('button', { name: '完成任务：角色动画', exact: true }).click()
  await expect(due.locator('.today-task')).toHaveCount(1)
  await page.getByRole('button', { name: '查看项目：LearnOS', exact: true }).first().click()
  await expect(page).toHaveURL(/view=board/)
  await expect(page).toHaveURL(/project=1/)
})

test('invalid and unchanged drops do not update, drag does not open drawer, and pending moves stay in original project', async ({ page }) => {
  const workspace = await mockWorkspace(page)
  await page.goto('/projects')
  const card = page.locator('[data-task-id="1"]')
  await card.dragTo(page.locator('.projects-tabs'))
  await expect(card).toBeVisible()
  expect(workspace.moveCount).toBe(0)
  await expect(page.locator('.task-drawer')).not.toBeVisible()
  workspace.setMoveDelay(1500)
  await card.dragTo(page.locator('[data-status="doing"]'))
  await expect(page.locator('[data-status="doing"] [data-task-id="1"]')).toBeVisible()
  await expect(page.locator('.task-drawer')).not.toBeVisible()
  await expect.poll(() => workspace.moveCount).toBe(1)
  await expect(page.locator('.projects-switcher .el-select__wrapper')).toHaveClass(/is-disabled/)
  await expect.poll(() => workspace.tasks[0]!.status).toBe('doing')
  expect(workspace.tasks[0]!.project_id).toBe(1)
})

for (const width of [375, 768, 1280, 1600]) {
  test(`${width}px board scrolls locally and reading/editing drawer actions stay visible`, async ({ page }, testInfo) => {
    const workspace = await mockWorkspace(page)
    workspace.tasks[0].title = '长标题'.repeat(30)
    workspace.tasks[0].description = '完整说明\n'.repeat(150)
    await page.setViewportSize({ width, height: 800 })
    await page.goto('/projects')
    expect(await page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1)
    const card = page.locator('[data-task-id="1"]')
    expect(await card.locator('strong').evaluate(el => el.clientHeight <= parseFloat(getComputedStyle(el).lineHeight) * 2 + 1)).toBe(true)
    await card.click()
    const drawer = page.locator('.task-drawer')
    await expect(drawer.getByRole('button', { name: '编辑', exact: true })).toBeInViewport()
    await expect.poll(async () => {
      const box = await drawer.boundingBox()
      return box ? Math.abs(box.x + box.width - (width > 760 ? width - 12 : width)) : 999
    }).toBeLessThanOrEqual(1)
    await expect(drawer.locator('input, textarea')).toHaveCount(0)
    await drawer.evaluate(async el => { await Promise.all(el.getAnimations().map(animation => animation.finished.catch(() => {}))) })
    expect(Math.abs((await drawer.boundingBox())!.x + (await drawer.boundingBox())!.width - (width > 760 ? width - 12 : width))).toBeLessThanOrEqual(1)
    if (process.env.CAPTURE_VISUALS) await page.screenshot({ path: testInfo.outputPath('task-reading.png') })
    await drawer.getByRole('button', { name: '编辑', exact: true }).click()
    await expect(drawer.getByRole('button', { name: '保存', exact: true })).toBeInViewport()
    expect(await drawer.evaluate(el => el.scrollWidth - el.clientWidth)).toBeLessThanOrEqual(1)
  })
}

for (const [timezoneId, expectedDate] of [['America/Los_Angeles', '2026-10-07'], ['Asia/Tokyo', '2026-10-08']]) {
  test(`Today uses browser calendar date in ${timezoneId}`, async ({ browser }) => {
    const context = await browser.newContext({ timezoneId })
    try {
      const page = await context.newPage()
      await page.clock.install({ time: new Date('2026-10-08T00:30:00Z') })
      const workspace = await mockWorkspace(page)
      workspace.tasks[1]!.due_date = expectedDate!
      const request = page.waitForRequest(req => new URL(req.url()).pathname === '/api/v1/projects/today')
      await page.goto('/projects?view=today&project=1')
      expect(new URL((await request).url()).searchParams.get('date')).toBe(expectedDate)
      const due = page.locator('.today-section').filter({ has: page.getByRole('heading', { name: '今天到期', exact: true }) })
      await expect(due.getByRole('button', { name: /^云部署/ })).toBeVisible()
    } finally { await context.close() }
  })
}
