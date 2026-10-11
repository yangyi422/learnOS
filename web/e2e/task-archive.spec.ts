import { expect, test, type Page } from '@playwright/test'

async function setup(page: Page) {
  const time = '2026-01-15T10:00:00Z'
  const tasks = [1, 2, 3].map(id => ({ id, project_id: id === 3 ? 2 : 1, title: `完成任务 ${id}`, description: '保留的内容', status: 'done', sort_order: id * 1024, priority: 'normal', due_date: null, completed_at: time, archived_at: null as string | null, created_at: time, updated_at: time }))
  const projects = [1, 2].map(id => ({ id, title: `项目 ${id}`, description: '', status: 'active', icon: '', accent: '', tasks: [], archived_at: null, created_at: time, updated_at: time }))
  let fail = false
  await page.route('**/api/v1/**', async route => {
    const req = route.request(); const url = new URL(req.url()); const path = url.pathname
    const reply = (data: unknown, status = 200) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify({ data }) })
    if (path === '/api/v1/auth/me') return reply({ id: 1, username: 'test', role: 'admin', status: 'active' })
    if (path === '/api/v1/projects') return reply(projects.map(p => ({ ...p, done_task_count: tasks.filter(t => t.project_id === p.id).length, archived_task_count: tasks.filter(t => t.project_id === p.id && t.archived_at).length, open_task_count: 0, doing_task_count: 0, next_task_count: 0 })))
    if (path === '/api/v1/tasks/archive') {
      if (fail) return route.fulfill({ status: 500, contentType: 'application/json', body: JSON.stringify({ error: { code: 'TEST', message: '保存失败', retryable: true } }) })
      const body = req.postDataJSON(); for (const task of tasks) if (body.task_ids.includes(task.id)) task.archived_at = body.archived ? '2026-02-01T10:00:00Z' : null
      return reply({ updated: true })
    }
    if (path === '/api/v1/tasks') {
      const archived = url.searchParams.get('archived') === 'true'; const project = Number(url.searchParams.get('project_id')); const status = url.searchParams.get('status')
      return reply(tasks.filter(t => Boolean(t.archived_at) === archived && (!project || t.project_id === project) && (!status || t.status === status)))
    }
    const match = path.match(/^\/api\/v1\/tasks\/(\d+)$/)
    if (match) return reply(tasks.find(t => t.id === Number(match[1])))
    return reply([])
  })
  return { tasks, setFail(value: boolean) { fail = value } }
}

test('single archive persists on refresh, project filter and restore keep completion', async ({ page }) => {
  const mock = await setup(page)
  await page.goto('/projects?view=board&project=1')
  await page.getByRole('button', { name: '打开任务：完成任务 1', exact: true }).click()
  await page.locator('.task-drawer__actions').getByRole('button', { name: '归档', exact: true }).click()
  await page.getByRole('dialog', { name: '确认归档' }).getByRole('button', { name: '归档', exact: true }).click()
  await expect(page.getByRole('button', { name: '打开任务：完成任务 1', exact: true })).toHaveCount(0)
  await expect(page.locator('.kanban-column--done .kanban-column__count')).toHaveText('1')
  await page.reload()
  await expect(page.getByRole('button', { name: '打开任务：完成任务 1', exact: true })).toHaveCount(0)
  await page.getByRole('button', { name: '项目列表', exact: true }).click()
  await page.getByRole('button', { name: '已归档任务', exact: true }).click()
  await expect(page.locator('.today-task')).toHaveCount(1)
  await page.locator('.today-task__open').click()
  await expect(page.locator('.task-drawer__activity')).toContainText('归档：')
  await expect(page.locator('.task-drawer__actions').getByRole('button', { name: '重新打开' })).toHaveCount(0)
  await page.locator('.task-drawer__actions').getByRole('button', { name: '取消归档' }).click()
  await page.getByRole('button', { name: '看板', exact: true }).click()
  await expect(page.getByRole('button', { name: '打开任务：完成任务 1', exact: true })).toBeVisible()
  expect(mock.tasks[0].completed_at).toBe('2026-01-15T10:00:00Z')
  expect(mock.tasks[0].description).toBe('保留的内容')
})

test('batch archive confirms count, cancellation and failed save preserve cards', async ({ page }) => {
  const mock = await setup(page)
  await page.goto('/projects?view=board')
  await page.getByRole('button', { name: '批量归档', exact: true }).click()
  await page.getByRole('checkbox', { name: '选择归档：完成任务 1', exact: true }).check()
  await page.getByRole('checkbox', { name: '选择归档：完成任务 2', exact: true }).check()
  await page.getByRole('button', { name: '归档所选（2）' }).click()
  await expect(page.getByRole('dialog', { name: '确认归档' })).toContainText('归档 2 个已完成任务')
  await page.getByRole('dialog', { name: '确认归档' }).getByRole('button', { name: '取消', exact: true }).click()
  expect(mock.tasks.filter(t => t.archived_at)).toHaveLength(0)
  mock.setFail(true)
  await page.getByRole('button', { name: '归档所选（2）' }).click()
  await page.getByRole('dialog', { name: '确认归档' }).getByRole('button', { name: '归档', exact: true }).click()
  await expect(page.getByText('保存失败', { exact: true })).toBeVisible()
  await expect(page.locator('.kanban-column--done .task-card')).toHaveCount(3)
  mock.setFail(false)
  await page.getByRole('button', { name: '批量归档', exact: true }).click()
  await page.getByRole('checkbox', { name: '选择归档：完成任务 1', exact: true }).check()
  await page.getByRole('checkbox', { name: '选择归档：完成任务 2', exact: true }).check()
  await page.getByRole('button', { name: '归档所选（2）' }).click()
  await page.getByRole('dialog', { name: '确认归档' }).getByRole('button', { name: '归档', exact: true }).click()
  await expect(page.locator('.kanban-column--done .kanban-column__count')).toHaveText('1')
  await page.goto('/projects?view=list&tasks=archived&project=2')
  await expect(page.getByText('当前项目范围内暂无归档任务。')).toBeVisible()
  await page.goto('/projects?view=list&tasks=archived&project=1')
  await expect(page.locator('.today-task')).toHaveCount(2)
})

for (const width of [1440, 390]) {
  test(`archive selection and list remain usable at ${width}px`, async ({ page }, testInfo) => {
    await page.setViewportSize({ width, height: 900 })
    await setup(page)
    await page.goto('/projects?view=board')
    await page.getByRole('button', { name: '批量归档', exact: true }).click()
    await page.getByRole('checkbox', { name: '选择归档：完成任务 1', exact: true }).check()
    await expect(page.getByRole('button', { name: '归档所选（1）' })).toBeEnabled()
    if (process.env.CAPTURE_VISUALS) await page.screenshot({ path: testInfo.outputPath('archive-selection.png'), fullPage: true })
    await page.getByRole('button', { name: '取消选择', exact: true }).click()
    await expect(page.getByRole('checkbox')).toHaveCount(0)
    await page.goto('/projects?view=list&tasks=archived')
    await expect(page.getByText('当前项目范围内暂无归档任务。')).toBeVisible()
    const overflow = await page.evaluate(() => document.documentElement.scrollWidth > innerWidth)
    expect(overflow).toBe(false)
  })
}
