import { expect, test, type Page } from '@playwright/test'
import type { InboxItem } from '../src/api/inbox'
import type { LightweightRecord } from '../src/api/records'

const timestamp = '2026-10-08T08:00:00Z'
const uri = 'obsidian://open?vault=YY-Wiki&file=Projects%2FLearnOS%2F产品设计.md'
async function mockRecords(page: Page) {
  const projects = [{ id: 1, title: 'LearnOS', status: 'active', description: '', icon: '', accent: '', open_task_count: 0, next_task_count: 0, doing_task_count: 0, done_task_count: 0 }, { id: 2, title: '资料项目', status: 'active', description: '', icon: '', accent: '', open_task_count: 0, next_task_count: 0, doing_task_count: 0, done_task_count: 0 }]
  const inbox: InboxItem[] = [{ id: 1, user_id: 1, content: '技术决策原文', status: 'inbox', source_type: 'manual', source_url: '', processed_to_type: '', processed_to_id: null, created_at: timestamp, updated_at: timestamp, processed_at: null, archived_at: null }]
  const records: LightweightRecord[] = []
  const tasks: Record<string, unknown>[] = []
  const captures = new Map<string, InboxItem>()
  const state = { inbox, records, projects, tasks, failCapture: false, loseCaptureReply: false, failTask: false, failRecordSave: false, loseRecordReply: false, captureKeys: [] as string[] }
  await page.route('**/api/v1/**', async route => {
    const req = route.request(); const url = new URL(req.url()); const path = url.pathname; const method = req.method()
    const reply = (data: unknown, status = 200) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify({ data }) })
    const failure = (message: string, status = 500) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify({ error: { code: 'MOCK_ERROR', message, retryable: status >= 500 } }) })
    if (path === '/api/v1/auth/me') return reply({ id: 1, username: 'test', role: 'admin', status: 'active' })
    if (path === '/api/v1/projects' && method === 'GET') return reply(projects)
    if (/^\/api\/v1\/projects\/\d+$/.test(path) && method === 'PATCH') { Object.assign(projects.find(p => p.id === Number(path.split('/').at(-1)))!, req.postDataJSON()); return reply({ updated: true }) }
    if (path === '/api/v1/inbox' && method === 'GET') return reply({ items: inbox.filter(item => item.status === (url.searchParams.get('status') || 'inbox')), counts: { inbox: inbox.filter(item => item.status === 'inbox').length, processed: inbox.filter(item => item.status === 'processed').length, archived: inbox.filter(item => item.status === 'archived').length } })
    if (path === '/api/v1/inbox' && method === 'POST') {
      const input = req.postDataJSON(); state.captureKeys.push(input.capture_key)
      if (state.failCapture) return failure('保存失败，请重试。')
      let item = captures.get(input.capture_key)
      if (!item) { item = { ...inbox[0]!, id: inbox.length + 1, content: input.content, status: 'inbox', processed_to_type: '', processed_to_id: null, created_at: timestamp }; inbox.unshift(item); captures.set(input.capture_key, item) }
      if (state.loseCaptureReply) { state.loseCaptureReply = false; return route.abort('failed') }
      return reply(item, 201)
    }
    const conversion = path.match(/^\/api\/v1\/inbox\/(\d+)\/convert-to-(task|record)$/)
    if (conversion && method === 'POST') {
      const item = inbox.find(item => item.id === Number(conversion[1]))!
      if (item.status !== 'inbox') return failure('这条内容已经整理过。', 409)
      const input = req.postDataJSON()
      if (conversion[2] === 'task') {
        if (state.failTask) return failure('任务创建失败，请重试。')
        const task = { id: tasks.length + 1, project_id: input.project_id, title: input.title, description: input.description, status: input.status, priority: input.priority, due_date: input.due_date, sort_order: 1024, created_at: timestamp, updated_at: timestamp, completed_at: null }
        tasks.push(task); Object.assign(item, { status: 'processed', processed_to_type: 'task', processed_to_id: task.id, processed_at: timestamp }); return reply({ item, task }, 201)
      }
      if (state.failRecordSave) return failure('记录保存失败，请重试。')
      const record = { id: records.length + 1, user_id: 1, ...input, source_inbox_id: item.id, created_at: timestamp, updated_at: timestamp, archived_at: null }
      records.push(record); Object.assign(item, { status: 'processed', processed_to_type: 'record', processed_to_id: record.id, processed_at: timestamp }); return reply({ item, record }, 201)
    }
    if (path === '/api/v1/records' && method === 'GET') return reply(records.filter(item => (!!item.archived_at === (url.searchParams.get('status') === 'archived')) && (!url.searchParams.get('project_id') || item.project_id === Number(url.searchParams.get('project_id')))))
    if (path === '/api/v1/records' && method === 'POST') {
      const input = req.postDataJSON()
      let record = records.find(item => (item as LightweightRecord & { creation_key?: string }).creation_key === input.creation_key && !!input.creation_key)
      if (!record) { record = { id: records.length + 1, user_id: 1, ...input, source_inbox_id: null, created_at: timestamp, updated_at: timestamp, archived_at: null }; records.push(record!) }
      if (state.loseRecordReply) { state.loseRecordReply = false; return route.abort('failed') }
      return reply(record, 201)
    }
    const recordPath = path.match(/^\/api\/v1\/records\/(\d+)(\/archive)?$/)
    if (recordPath) {
      const index = records.findIndex(item => item.id === Number(recordPath[1])); const record = records[index]
      if (!record) return failure('记录不存在。', 404)
      if (method === 'GET') return reply(record)
      if (method === 'PUT') { if (state.failRecordSave) return failure('记录保存失败，请重试。'); Object.assign(record, req.postDataJSON()); return reply(record) }
      if (method === 'DELETE') { records.splice(index, 1); return reply({ deleted: true }) }
      if (method === 'POST') { record.archived_at = req.postDataJSON().archived ? timestamp : null; return reply({ archived: !!record.archived_at }) }
    }
    if (path === '/api/v1/tasks' && method === 'GET') return reply(tasks.filter(task => task.status === url.searchParams.get('status')))
    if (/^\/api\/v1\/tasks\/\d+$/.test(path) && method === 'GET') return reply(tasks.find(task => task.id === Number(path.split('/').at(-1))))
    return reply([])
  })
  return state
}

test('single capture supports blank rejection, multiline, failed response retry and immediate list recovery', async ({ page }) => {
  const state = await mockRecords(page)
  await page.goto('/inbox')
  const input = page.getByRole('textbox', { name: '快速记录到收集箱' })
  await expect(page.getByRole('button', { name: '记录到收集箱', exact: true })).toBeDisabled()
  await input.fill('临时想法'); await input.press('Shift+Enter'); await input.press('x')
  expect(state.captureKeys).toHaveLength(0)
  state.loseCaptureReply = true
  await input.press('Enter')
  await expect(page.locator('.capture-error')).toBeVisible()
  await expect(input).toHaveValue('临时想法\nx')
  await input.press('Enter')
  await expect(input).toHaveValue('')
  expect(state.captureKeys[0]).toBe(state.captureKeys[1])
  expect(state.inbox.filter(item => item.content === '临时想法\nx')).toHaveLength(1)
  await expect(page.locator('.inbox-item__content').filter({ hasText: '临时想法' })).toBeVisible()
  await page.reload()
  await expect(page.locator('.inbox-item__content').filter({ hasText: '临时想法' })).toHaveCount(1)
})

test('task conversion reuses recent active project and preserves Inbox after failure', async ({ page }) => {
  const state = await mockRecords(page)
  await page.addInitScript(() => localStorage.setItem('learnos:last-project', '2'))
  await page.goto('/inbox'); await page.getByRole('button', { name: '转为任务', exact: true }).click()
  const dialog = page.getByRole('dialog', { name: '整理为项目任务' })
  await expect(dialog.locator('.el-select').first()).toContainText('资料项目')
  state.failTask = true; await dialog.getByRole('button', { name: '转换为任务' }).click()
  await expect(page.getByText('任务创建失败，请重试。')).toBeVisible()
  expect(state.inbox[0]!.status).toBe('inbox')
  state.failTask = false; await dialog.getByRole('button', { name: '转换为任务' }).click()
  await expect(page).toHaveURL(/project=2/)
  await expect(page.locator('.task-drawer__reading')).toContainText('技术决策原文')
  expect(state.tasks).toHaveLength(1); expect(state.inbox[0]!.processed_to_type).toBe('task')
})

test('save as record needs only content, preserves source, and unassigned records can be found after reload', async ({ page }) => {
  const state = await mockRecords(page)
  await page.goto('/inbox'); await page.getByRole('button', { name: '保存为记录', exact: true }).click()
  const dialog = page.getByRole('dialog', { name: '保存为记录', exact: true })
  state.failRecordSave = true; await dialog.getByRole('button', { name: '保存为记录' }).click()
  await expect(dialog.getByRole('alert')).toContainText('记录保存失败')
  expect(state.inbox[0]!.status).toBe('inbox')
  state.failRecordSave = false; await dialog.getByRole('button', { name: '保存为记录' }).click()
  await expect(page.locator('.inbox-item')).toHaveCount(0)
  expect(state.records[0]!.project_id).toBeNull(); expect(state.inbox[0]!.content).toBe('技术决策原文')
  await page.getByRole('button', { name: /^已处理/ }).click()
  await expect(page.getByText('已保存为记录', { exact: true })).toBeVisible()
  await page.getByRole('button', { name: '查看记录', exact: true }).click()
  const detail = page.getByRole('dialog', { name: '记录详情' })
  await expect(detail.locator('textarea')).toHaveCount(0)
  await expect(detail).toContainText('技术决策原文')
  await detail.getByRole('button', { name: 'Close this dialog' }).click()
  await page.reload(); await page.getByRole('button', { name: '记录', exact: true }).click()
  await expect(page.locator('.record-row')).toContainText('技术决策原文')
})

test('record editing, safe links, archive recovery and deletion leave original Inbox intact', async ({ page }) => {
  const state = await mockRecords(page)
  state.records.push({ id: 1, user_id: 1, content: '<script>alert(1)</script> 完整说明 https://example.com', project_id: null, source_inbox_id: 1, external_url: uri, link_name: '产品笔记', created_at: timestamp, updated_at: timestamp, archived_at: null })
  state.inbox[0]!.status = 'processed'; state.inbox[0]!.processed_to_type = 'record'; state.inbox[0]!.processed_to_id = 1
  await page.goto('/inbox'); await page.getByRole('button', { name: '记录', exact: true }).click()
  await page.locator('.record-row__open').click()
  const dialog = page.getByRole('dialog', { name: '记录详情' })
  await expect(dialog.locator('script')).toHaveCount(0)
  await expect(dialog.getByRole('link', { name: 'https://example.com', exact: true })).toHaveAttribute('target', '_blank')
  await expect(dialog.getByRole('link', { name: '产品笔记' })).toHaveAttribute('href', uri)
  await page.evaluate(() => Object.defineProperty(navigator, 'clipboard', { configurable: true, value: { writeText: async (text: string) => { (window as unknown as { copied: string }).copied = text } } }))
  await dialog.getByRole('button', { name: '复制链接', exact: true }).click()
  expect(await page.evaluate(() => (window as unknown as { copied: string }).copied)).toBe(uri)
  await dialog.getByRole('button', { name: '编辑', exact: true }).click()
  await dialog.getByRole('textbox', { name: '记录内容' }).fill('修改内容')
  await dialog.getByRole('textbox', { name: '外部链接', exact: true }).fill('javascript:alert(1)')
  await dialog.getByRole('button', { name: '保存', exact: true }).click()
  await expect(dialog.getByRole('alert')).toContainText('链接须为')
  await dialog.getByRole('textbox', { name: '外部链接', exact: true }).fill('https://example.org')
  state.failRecordSave = true; await dialog.getByRole('button', { name: '保存', exact: true }).click()
  await expect(dialog.getByRole('textbox', { name: '记录内容' })).toHaveValue('修改内容')
  state.failRecordSave = false; await dialog.getByRole('button', { name: '保存', exact: true }).click()
  await expect(dialog.locator('textarea')).toHaveCount(0)
  await expect(dialog).toContainText('修改内容')
  await dialog.getByRole('button', { name: '归档记录' }).click()
  await expect(page.locator('.record-row')).toHaveCount(0)
  await page.getByRole('button', { name: '已归档记录', exact: true }).click()
  await page.locator('.record-row__open').click(); await dialog.getByRole('button', { name: '恢复记录' }).click()
  await page.getByRole('button', { name: '记录', exact: true }).last().click()
  await page.locator('.record-row__open').click(); await dialog.getByRole('button', { name: '删除', exact: true }).click()
  await page.getByRole('dialog', { name: '删除记录', exact: true }).getByRole('button', { name: '删除', exact: true }).click()
  await expect(page.locator('.record-row')).toHaveCount(0)
  expect(state.inbox[0]!.status).toBe('processed'); expect(state.inbox[0]!.content).toBe('技术决策原文')
})

for (const width of [375, 768, 1440]) {
  test(`${width}px project records are separate from tasks and survive project archival`, async ({ page }, testInfo) => {
    const state = await mockRecords(page)
    await page.setViewportSize({ width, height: 850 })
    await page.goto('/projects?project=1'); await page.locator('.project-records > summary').click()
    await page.getByRole('button', { name: '新增记录', exact: true }).click()
    const dialog = page.getByRole('dialog', { name: '新增记录' })
    await dialog.getByRole('textbox', { name: '记录内容' }).fill('项目技术决策\n完整上下文')
    await dialog.getByRole('textbox', { name: '外部链接', exact: true }).fill(uri)
    await expect(dialog.getByRole('button', { name: '保存', exact: true })).toBeInViewport()
    await dialog.getByRole('button', { name: '保存', exact: true }).click()
    await expect(dialog).not.toBeVisible()
    expect(state.records[0]!.project_id).toBe(1)
    expect(state.tasks).toHaveLength(0)
    await expect(page.locator('.record-row')).toContainText('项目技术决策')
    state.projects[0]!.status = 'archived'; await page.reload()
    await page.locator('.project-records > summary').click()
    await expect(page.locator('.record-row')).toContainText('项目技术决策')
    await page.locator('.record-row__open').click()
    const detail = page.getByRole('dialog', { name: '记录详情' })
    await expect(detail.locator('textarea')).toHaveCount(0)
    expect(await page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1)
    if (process.env.CAPTURE_VISUALS) {
      await detail.evaluate(async el => { const animations: Animation[] = []; for (let node: Element | null = el; node; node = node.parentElement) animations.push(...node.getAnimations()); await Promise.all(animations.map(animation => animation.finished.catch(() => {}))) })
      await page.screenshot({ path: testInfo.outputPath('project-record-reading.png') })
    }
  })
}


test('direct project record creation retries reuse the same saved record', async ({ page }) => {
  const state = await mockRecords(page)
  await page.goto('/projects?project=1'); await page.locator('.project-records > summary').click()
  await page.getByRole('button', { name: '新增记录', exact: true }).click()
  const dialog = page.getByRole('dialog', { name: '新增记录' })
  await dialog.getByRole('textbox', { name: '记录内容' }).fill('必须保留的一次记录')
  state.loseRecordReply = true
  await dialog.getByRole('button', { name: '保存', exact: true }).click()
  await expect(dialog.getByRole('alert')).toContainText('无法连接')
  await expect(dialog.getByRole('textbox', { name: '记录内容' })).toHaveValue('必须保留的一次记录')
  await dialog.getByRole('button', { name: '保存', exact: true }).click()
  await expect(dialog).not.toBeVisible()
  expect(state.records).toHaveLength(1)
  await expect(page.locator('.record-row')).toHaveCount(1)
})

test('record project association can change and unsaved edits are protected', async ({ page }) => {
  const state = await mockRecords(page)
  state.records.push({ id: 1, user_id: 1, content: '项目上下文', project_id: 1, source_inbox_id: null, external_url: '', link_name: '', created_at: timestamp, updated_at: timestamp, archived_at: null })
  await page.goto('/inbox'); await page.getByRole('button', { name: '记录', exact: true }).click()
  await page.locator('.record-row__open').click()
  const dialog = page.getByRole('dialog', { name: '记录详情' })
  await dialog.getByRole('button', { name: '编辑', exact: true }).click()
  await dialog.getByRole('textbox', { name: '记录内容' }).fill('尚未保存的内容')
  await dialog.getByRole('button', { name: 'Close this dialog' }).click()
  await page.getByRole('button', { name: '继续编辑', exact: true }).click()
  await expect(dialog.getByRole('textbox', { name: '记录内容' })).toHaveValue('尚未保存的内容')
  expect(state.records[0]!.content).toBe('项目上下文')
  await dialog.locator('.el-select__wrapper').click()
  await page.getByRole('option', { name: '资料项目', exact: true }).click()
  await dialog.getByRole('button', { name: '保存', exact: true }).click()
  await expect(dialog.locator('.record-meta')).toContainText('资料项目')
  expect(state.records[0]!.project_id).toBe(2)
  await dialog.getByRole('button', { name: 'Close this dialog' }).click()
  await page.goto('/projects?project=1'); await page.locator('.project-records > summary').click()
  await expect(page.locator('.record-row')).toHaveCount(0)
  await page.goto('/projects?project=2'); await page.locator('.project-records > summary').click()
  await expect(page.locator('.record-row')).toContainText('尚未保存的内容')
})
