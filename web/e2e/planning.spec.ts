import { expect, test, type BrowserContext } from '@playwright/test'
import type { ProjectTask, Project } from '../src/api/projects'
import type { PlanningTask, WorkspacePlanning } from '../src/api/workspace'

const day = '2026-10-20', stamp = '2026-10-20T08:00:00Z'
function seedTask(id: number, title: string, patch: Partial<ProjectTask> = {}): ProjectTask {
  return { id, project_id: 1, title, description: '', status: 'inbox', sort_order: id * 1024, priority: 'normal', due_date: null, plan_date: null, completed_at: null, created_at: stamp, updated_at: stamp, ...patch }
}
async function installPlanning(context: BrowserContext) {
  const tasks = [seedTask(1, '准备提案', { due_date: '2026-10-30' }), seedTask(2, '今天到期但未安排', { due_date: day }), seedTask(3, '生活事项', { project_id: 2 }), seedTask(4, '长期逾期', { due_date: '2020-01-01' }), seedTask(5, '昨天未完成', { plan_date: '2026-10-19' }), seedTask(6, '主动计划且到期', { plan_date: day, due_date: day })]
  const projects: Project[] = [{ id: 1, title: '正职项目', context: 'work', status: 'active', description: '', icon: '', accent: '', archived_at: null, tasks: [], open_task_count: 5, doing_task_count: 0, next_task_count: 0, done_task_count: 0, created_at: stamp, updated_at: stamp }, { id: 2, title: '个人项目', context: 'life', status: 'active', description: '', icon: '', accent: '', archived_at: null, tasks: [], open_task_count: 1, doing_task_count: 0, next_task_count: 0, done_task_count: 0, created_at: stamp, updated_at: stamp }]
  const state = { failWrite: false, failCandidates: false, writes: [] as Record<string, unknown>[] }
  await context.route('**/api/v1/**', async route => {
    const req = route.request(), url = new URL(req.url()), path = url.pathname
    const reply = (data: unknown) => route.fulfill({ json: { data } })
    const error = () => route.fulfill({ status: 503, json: { error: { code: 'TEMPORARY', message: '暂时无法保存，请重试', retryable: true } } })
    if (path === '/api/v1/auth/me') return reply({ id: 1, username: 'planning', role: 'user', status: 'active' })
    if (path === '/api/v1/workspace/recent') return reply({ library: [], events: [], goals: [], asset_date: '', errors: {} })
    if (path === '/api/v1/projects' && req.method() === 'GET') return reply(projects.map(p => ({ ...p, open_task_count: tasks.filter(t => t.project_id === p.id && t.status !== 'done').length, done_task_count: tasks.filter(t => t.project_id === p.id && t.status === 'done').length })))
    if (path === '/api/v1/workspace/home') {
      const date = url.searchParams.get('date')!, scene = url.searchParams.get('context') ?? 'all'
      const rows: PlanningTask[] = tasks.filter(t => t.status !== 'done' && (scene === 'all' || projects.find(p => p.id === t.project_id)?.context === scene)).map(t => ({ ...t, project_title: projects.find(p => p.id === t.project_id)!.title, effective_context: projects.find(p => p.id === t.project_id)!.context! }))
      const planning: WorkspacePlanning = { items: [], due_today: [], overdue: [], previous_plan: [], summary: { planned: 0, due_today: 0, overdue: 0, previous_plan: 0, total: 0 } }
      for (const t of rows) {
        if (t.plan_date === date) planning.items.push(t)
        else if (t.due_date && t.due_date < date) planning.overdue.push(t)
        else if (t.due_date === date) planning.due_today.push(t)
        else if (t.plan_date && t.plan_date < date) planning.previous_plan.push(t)
      }
      planning.summary = { planned: planning.items.length, due_today: planning.due_today.length, overdue: planning.overdue.length, previous_plan: planning.previous_plan.length, total: planning.items.length + planning.due_today.length + planning.overdue.length + planning.previous_plan.length }
      const focus = planning.items[0]
      return reply({ date, context: scene, planning, today: { date, doing: [], due: [], next: [] }, focus: focus ? { kind: 'task', title: focus.title, task_id: focus.id, project_id: focus.project_id, project_title: focus.project_title, status: focus.status } : { kind: 'empty', title: '' }, daily_focus: focus ? { work: { id: 1, date, context: 'work', task_id: focus.id, revision: 1, state: 'ok', title: focus.title, task_status: focus.status, priority: focus.priority, plan_date: focus.plan_date ?? null, due_date: focus.due_date ?? null, completed_at: null, project_id: focus.project_id, project_title: focus.project_title, project_context: 'work', project_status: 'active', state_reason: '', updated_at: stamp, task_updated_at: stamp }, life: null } : { work: null, life: null }, projects: projects.filter(p => scene === 'all' || p.context === scene).map(p => ({ ...p, open_task_count: tasks.filter(t => t.project_id === p.id && t.status !== 'done').length, done_task_count: tasks.filter(t => t.project_id === p.id && t.status === 'done').length })), learning: [], inbox: { count: 0, items: [] }, errors: {} })
    }
    if (path === '/api/v1/tasks' && req.method() === 'GET') {
      if (state.failCandidates && url.searchParams.has('actionable')) return error()
      let rows = tasks.filter(t => !url.searchParams.get('project_id') || t.project_id === Number(url.searchParams.get('project_id')))
      const scene = url.searchParams.get('context'), status = url.searchParams.get('status'), q = url.searchParams.get('q') ?? ''
      if (scene && scene !== 'all') rows = rows.filter(t => projects.find(p => p.id === t.project_id)?.context === scene)
      if (url.searchParams.get('actionable')) rows = rows.filter(t => t.status !== 'done')
      if (status) rows = rows.filter(t => t.status === status)
      rows = rows.filter(t => t.title.includes(q))
      const offset = Number(url.searchParams.get('offset') ?? 0), limit = Number(url.searchParams.get('limit') ?? rows.length)
      return reply(rows.slice(offset, offset + limit))
    }
    const match = path.match(/^\/api\/v1\/tasks\/(\d+)$/)
    if (match) {
      const task = tasks.find(t => t.id === Number(match[1]))!
      if (req.method() === 'GET') return reply(task)
      if (req.method() === 'PATCH') {
        const patch = req.postDataJSON() as Record<string, unknown>; state.writes.push(patch)
        if (state.failWrite) return error()
        if (patch.status === 'done' && task.status !== 'done') task.completed_at = stamp
        Object.assign(task, patch, { updated_at: new Date().toISOString() })
        return reply(task)
      }
    }
    if (path === '/api/v1/projects/today') return reply({ date: day, doing: [], due: [], next: [] })
    return reply({})
  })
  return { tasks, projects, state }
}
async function setup(page: import('@playwright/test').Page, context: BrowserContext) {
  await page.clock.install({ time: new Date('2026-10-20T12:00:00') })
  await page.addInitScript(() => localStorage.setItem('learnos:workspace-context:1', 'work'))
  const mock = await installPlanning(context)
  await page.goto('/')
  await expect(page.locator('.daily-plan > .plan-task .plan-task__title')).toHaveText(['主动计划且到期'])
  return mock
}

test('已有任务加入、移出、改期、完成并同步原看板', async ({ page, context }) => {
  const mock = await setup(page, context)
  await expect(page.locator('.plan-summary')).toHaveText('已安排 1 项，另有 3 项待处理提醒')
  await expect(page.locator('.daily-plan > .plan-task')).toHaveCount(1)
  await page.getByRole('button', { name: '加入今天', exact: true }).click()
  const picker = page.getByRole('dialog', { name: '选择已有任务加入今天' })
  await expect(picker.locator('.plan-candidate')).not.toContainText(['生活事项'])
  await picker.locator('.plan-candidate').filter({ hasText: '准备提案' }).click()
  await expect(picker).toBeHidden()
  await expect(page.locator('.daily-plan > .plan-task')).toHaveCount(2)
  expect(mock.tasks[0].due_date).toBe('2026-10-30')
  await page.getByRole('button', { name: '任务安排操作：准备提案' }).click()
  await page.getByRole('menuitem', { name: '移出今天' }).click()
  await expect(page.locator('.daily-plan > .plan-task')).toHaveCount(1)
  expect(mock.tasks[0].plan_date).toBeNull()
  await page.getByRole('button', { name: '任务安排操作：主动计划且到期' }).click()
  await page.getByRole('menuitem', { name: '重新安排日期' }).click()
  const dialog = page.getByRole('dialog', { name: '重新安排计划日期' })
  await dialog.getByRole('combobox', { name: '新的计划日期' }).fill('2026-10-21')
  await dialog.getByRole('combobox', { name: '新的计划日期' }).press('Enter')
  await dialog.getByRole('button', { name: '保存计划日期' }).click()
  await expect(dialog).toBeHidden()
  await expect(page.locator('.daily-plan > .plan-task')).toHaveCount(0)
  expect(mock.tasks[5].due_date).toBe(day)
  await page.locator('.plan-reminders summary').click()
  const overdue = page.locator('.plan-reminders .plan-task').filter({ hasText: '长期逾期' })
  await overdue.getByRole('button', { name: '加入今天', exact: true }).click()
  await expect(page.locator('.daily-plan > .plan-task .plan-task__title')).toHaveText(['长期逾期'])
  await page.getByRole('button', { name: '完成任务：长期逾期' }).click()
  await expect(page.locator('.daily-plan > .plan-task')).toHaveCount(0)
  expect(mock.tasks[3].status).toBe('done')
  await page.getByRole('link', { name: '项目', exact: true }).click()
  await expect(page.locator('[data-status="done"] .task-card').filter({ hasText: '长期逾期' })).toBeVisible()
  expect(mock.tasks).toHaveLength(6)
})

test('读取与写入失败保留状态，可重试且不复制任务', async ({ page, context }) => {
  const mock = await setup(page, context)
  mock.state.failCandidates = true
  await page.getByRole('button', { name: '加入今天', exact: true }).click()
  const picker = page.getByRole('dialog', { name: '选择已有任务加入今天' })
  await expect(picker.getByRole('alert')).toBeVisible()
  mock.state.failCandidates = false
  await picker.getByRole('button', { name: '重试读取' }).click()
  await expect(picker.locator('.plan-candidate')).toHaveCount(5)
  mock.state.failWrite = true
  await picker.locator('.plan-candidate').filter({ hasText: '准备提案' }).click()
  await expect(picker.getByRole('alert')).toContainText('暂时无法保存')
  expect(mock.tasks[0].plan_date).toBeNull()
  mock.state.failWrite = false
  await picker.locator('.plan-candidate').filter({ hasText: '准备提案' }).click()
  await expect(picker).toBeHidden()
  await expect(page.locator('.daily-plan > .plan-task')).toHaveCount(2)
  await page.reload()
  await expect(page.locator('.daily-plan > .plan-task')).toHaveCount(2)
  mock.state.failWrite = true
  await page.getByRole('button', { name: '完成任务：准备提案' }).click()
  await expect(page.locator('.daily-plan > .plan-task')).toHaveCount(2)
  await expect(page.locator('.daily-plan > .plan-error')).toContainText('暂时无法保存')
  mock.state.failWrite = false
  await page.getByRole('button', { name: '重试操作', exact: true }).click()
  await expect(page.locator('.daily-plan > .plan-task')).toHaveCount(1)
  expect(mock.tasks).toHaveLength(6)
})

for (const theme of ['light', 'dark']) {
  test(`今日计划 ${theme} 手机与详情入口`, async ({ page, context }, info) => {
    await page.setViewportSize({ width: 390, height: 844 })
    await page.addInitScript(theme => localStorage.setItem('learnos-theme', theme), theme)
    await setup(page, context)
    expect((await page.getByRole('button', { name: '加入今天', exact: true }).boundingBox())!.height).toBeGreaterThanOrEqual(44)
    await page.locator('.plan-reminders summary').click()
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
    await page.screenshot({ path: info.outputPath(`planning-${theme}-mobile.png`), fullPage: true })
    await page.locator('.daily-plan > .plan-task .plan-task__title').click()
    const drawer = page.getByRole('dialog', { name: '任务详情' })
    await expect(drawer).toContainText('计划执行日期')
    await expect(drawer).toContainText(day)
  })
}

for (const timezone of ['Asia/Shanghai', 'America/Los_Angeles']) {
  test(`跨日重读计划且场景保持 ${timezone}`, async ({ browser }) => {
    const context = await browser.newContext({ timezoneId: timezone })
    const page = await context.newPage()
    await page.clock.install({ time: new Date(timezone === 'Asia/Shanghai' ? '2026-10-20T15:59:40Z' : '2026-10-21T06:59:40Z') })
    await page.addInitScript(() => localStorage.setItem('learnos:workspace-context:1', 'work'))
    await installPlanning(context)
    await page.goto('/')
    await expect(page.locator('.daily-plan > .plan-task')).toHaveCount(1)
    await page.clock.runFor(21_000)
    await expect(page.locator('.daily-plan > .plan-task')).toHaveCount(0)
    await expect(page.getByRole('radio', { name: '工作', exact: true })).toBeChecked()
    await expect(page.locator('.page-header__description')).toContainText('10月21日')
    await context.close()
  })
}

test('首页与已打开的项目看板跨标签同步完成', async ({ page, context }) => {
  test.setTimeout(60_000)
  await setup(page, context)
  const board = await context.newPage()
  await board.clock.install({ time: new Date('2026-10-20T12:00:00') })
  await board.goto('/projects?project=1&view=board')
  await expect(board.locator('[data-status="inbox"] [data-task-id="6"]')).toBeVisible()
  await page.getByRole('button', { name: '完成任务：主动计划且到期' }).click()
  await expect(board.locator('[data-status="done"] [data-task-id="6"]')).toBeVisible()
  await page.getByRole('button', { name: '加入今天', exact: true }).click()
  await page.getByRole('dialog', { name: '选择已有任务加入今天' }).locator('.plan-candidate').filter({ hasText: '准备提案' }).click()
  await expect(page.locator('.daily-plan > .plan-task')).toHaveCount(1)
  await board.locator('[data-status="inbox"] [data-task-id="1"] .task-card__body').click()
  await board.getByRole('dialog', { name: '任务详情' }).getByRole('button', { name: '标记完成', exact: true }).click()
  await expect(page.locator('.daily-plan > .plan-task')).toHaveCount(0)
  await board.close()
})

test('原任务抽屉编辑并清空计划日期，截止日期不变', async ({ page, context }) => {
  const mock = await setup(page, context)
  await page.goto('/projects?project=1&view=board&task=6')
  const drawer = page.getByRole('dialog', { name: '任务详情' })
  await drawer.getByRole('button', { name: '编辑', exact: true }).click()
  await drawer.getByRole('combobox', { name: '计划执行日期', exact: true }).fill('2026-10-22')
  await drawer.getByRole('combobox', { name: '计划执行日期', exact: true }).press('Enter')
  await drawer.getByRole('button', { name: '保存', exact: true }).click()
  await expect(drawer.getByRole('button', { name: '编辑', exact: true })).toBeVisible()
  expect(mock.tasks[5].plan_date).toBe('2026-10-22')
  await drawer.getByRole('button', { name: '编辑', exact: true }).click()
  await drawer.getByRole('combobox', { name: '计划执行日期', exact: true }).fill('')
  await drawer.getByRole('combobox', { name: '计划执行日期', exact: true }).press('Tab')
  await drawer.getByRole('button', { name: '保存', exact: true }).click()
  await expect(drawer.getByRole('button', { name: '编辑', exact: true })).toBeVisible()
  expect(mock.tasks[5].plan_date).toBeNull()
  expect(mock.tasks[5].due_date).toBe(day)
})
