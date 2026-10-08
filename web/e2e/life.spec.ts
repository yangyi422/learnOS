import { expect, test, type Page, type Locator } from '@playwright/test'
import type { LifeEvent, LifeGoal, LifeEntry } from '../src/api/life'
const time = '2026-10-08T08:00:00Z'
const uri = 'obsidian://open?vault=Wiki&file=Life%2FHome.md'
async function mockLife(page: Page) {
  const events: LifeEvent[] = [], goals: LifeGoal[] = [], entries: LifeEntry[] = []
  const inbox = [{ id: 1, user_id: 1, content: '和老朋友重逢的那一天', status: 'inbox', source_type: 'manual', source_url: '', processed_to_type: '', processed_to_id: null as number | null, created_at: time, updated_at: time, processed_at: null, archived_at: null }]
  const projects = [{ id: 1, title: '独立作品', status: 'completed', description: '', icon: '', accent: '', open_task_count: 0, next_task_count: 0, doing_task_count: 0, done_task_count: 0 }]
  const state = { events, goals, entries, inbox, failSave: false, loseReply: false, saveKeys: [] as string[], graduateCalls: 0, graduated: false }
  await page.route('**/api/v1/**', async route => {
    const req = route.request(), url = new URL(req.url()), path = url.pathname, method = req.method()
    const reply = (data: unknown, status = 200) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify({ data }) })
    const failure = (message: string, status = 500) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify({ error: { code: 'TEST', message, retryable: status >= 500 } }) })
    if (path === '/api/v1/auth/me') return reply({ id: 1, username: 'test', role: 'admin', status: 'active' })
    if (path === '/api/v1/projects') return reply(projects)
    if (path === '/api/v1/projects/tasks' || /\/tasks$/.test(path)) return reply({ tasks: [], done_count: 0 })
    if (path === '/api/v1/inbox') return reply({ items: inbox.filter(i => i.status === (url.searchParams.get('status') || 'inbox')), counts: { inbox: inbox.filter(i => i.status === 'inbox').length, processed: inbox.filter(i => i.status === 'processed').length, archived: 0 } })
    if (path === '/api/v1/life/events' && method === 'GET') return reply(events.filter(e => (!url.searchParams.get('domain') || [e.primary_domain, e.secondary_domain].includes(url.searchParams.get('domain')!)) && (url.searchParams.get('milestones') !== 'true' || e.milestone)).sort((a, b) => b.occurred_on.localeCompare(a.occurred_on) || b.id - a.id))
    const convert = path.match(/^\/api\/v1\/inbox\/(\d+)\/convert-to-life-event$/)
    if ((path === '/api/v1/life/events' && method === 'POST') || convert) {
      const input = req.postDataJSON(); state.saveKeys.push(input.creation_key)
      if (state.failSave) return failure('保存失败，请重试。')
      let event = events.find(e => e.creation_key === input.creation_key || (input.source_id && e.source_type === input.source_type && e.source_id === input.source_id))
      if (!event) { event = { ...input, id: events.length + 1, source_type: input.source_type || '', source_id: input.source_id || null, source_title: input.source_type === 'project' ? '独立作品' : convert ? inbox[0]!.content : '', source_available: !!input.source_id, source_status: 'completed', created_at: time, updated_at: time }; events.push(event!) }
      if (convert) Object.assign(inbox[0]!, { status: 'processed', processed_to_type: 'life_event', processed_to_id: event!.id })
      if (state.loseReply) { state.loseReply = false; return route.abort('failed') }
      return reply(event, 201)
    }
    const eventPath = path.match(/^\/api\/v1\/life\/events\/(\d+)$/)
    if (eventPath) { const index = events.findIndex(e => e.id === Number(eventPath[1])); if (index < 0) return failure('生活事件已不存在。', 404); if (method === 'GET') return reply(events[index]); if (method === 'PUT') { if (state.failSave) return failure('保存失败，请重试。'); Object.assign(events[index]!, req.postDataJSON()); return reply(events[index]) }; events.splice(index, 1); return reply({ deleted: true }) }
    if (path === '/api/v1/life/goals' && method === 'GET') return reply(goals)
    if (path === '/api/v1/life/goals' && method === 'POST') { const input = req.postDataJSON(); if (state.failSave) return failure('保存失败，请重试。'); const goal = { ...input, id: goals.length + 1, created_at: time, updated_at: time }; goals.push(goal); return reply(goal) }
    const goalPath = path.match(/^\/api\/v1\/life\/goals\/(\d+)(\/entries)?$/)
    if (goalPath) {
      const goal = goals.find(g => g.id === Number(goalPath[1]))!; if (!goal) return failure('目标不存在', 404)
      if (method === 'GET') return reply({ goal, entries: entries.filter(e => e.goal_id === goal.id), events: events.filter(e => e.goal_id === goal.id) })
      const input = req.postDataJSON(); if (state.failSave) return failure('保存失败，请重试。')
      if (goalPath[2]) { let entry = entries.find(e => e.creation_key === input.creation_key); if (!entry) { entry = { ...input, id: entries.length + 1, goal_id: goal.id, from_status: '', to_status: '', created_at: time }; entries.push(entry!) }; return reply(entry, 201) }
      if (goal.status !== input.status || (input.record_change && goal.current_note !== input.current_note)) entries.push({ ...input, id: entries.length + 1, goal_id: goal.id, occurred_on: input.status_date, content: input.current_note, reason: input.status_reason, from_status: goal.status !== input.status ? goal.status : '', to_status: goal.status !== input.status ? input.status : '', created_at: time })
      Object.assign(goal, input); return reply(goal)
    }
    const sourcePath = path.match(/^\/api\/v1\/life\/sources\/(project|course)\/(\d+)$/)
    if (sourcePath) return reply({ type: sourcePath[1], id: Number(sourcePath[2]), title: sourcePath[1] === 'project' ? '独立作品' : '系统思考', status: sourcePath[1] === 'project' || state.graduated ? 'completed' : 'learning', eligible: sourcePath[1] === 'project' || state.graduated, can_graduate: sourcePath[1] === 'course' && !state.graduated, event_id: events.find(e => e.source_type === sourcePath[1])?.id ?? null })
    if (path === '/api/v1/courses/2/graduate') { state.graduateCalls++; state.graduated = true; return reply({ status: 'completed' }) }
    if (/\/curriculum\/coverage$/.test(path)) return reply({ metrics: { core_missing: 0, core_coverage_percent: 100, overall_coverage_percent: 100, core_covered: 1, core_total: 1, recommended_covered: 0, optional_covered: 0 }, units: [] })
    if (/\/curriculum\/drafts$/.test(path)) return reply([])
    return reply([])
  })
  return state
}
async function selectOption(page: Page, field: Locator, label: string) { await field.press('Enter'); const id = await field.getAttribute('aria-controls'); await page.locator(`[id="${id}"]`).getByRole('option', { name: label, exact: true }).click() }
const dialog = (page: Page) => page.getByRole('dialog').filter({ has: page.locator('.life-footer') })
async function newEvent(page: Page, title: string, date: string) {
  await page.getByRole('button', { name: '+ 记录生活事件' }).click(); const d = dialog(page)
  await d.getByLabel('事件名称').fill(title); await d.getByLabel('发生日期').fill(date); await d.getByLabel('发生日期').press('Tab'); await d.getByRole('button', { name: '保存', exact: true }).click(); await expect(d).not.toBeVisible()
}
test('生活事件支持补录、阅读、编辑、删除及刷新恢复', async ({ page }) => {
  const state = await mockLife(page); await page.goto('/life'); await expect(page.getByRole('link', { name: '生活', exact: true })).toBeVisible()
  await newEvent(page, '过去的搬家', '2020-03-02'); await newEvent(page, '今天的决定', '2026-10-08')
  await expect(page.locator('.life-event-card h3')).toHaveText(['今天的决定', '过去的搬家']); await page.reload(); await expect(page.locator('.life-event-card h3')).toHaveText(['今天的决定', '过去的搬家'])
  await page.getByRole('button', { name: '过去的搬家', exact: true }).click(); await expect(dialog(page).getByRole('textbox')).toHaveCount(0); await dialog(page).getByRole('button', { name: '编辑', exact: true }).click(); await dialog(page).getByLabel('事件名称').fill('搬家纪念'); await dialog(page).getByRole('button', { name: '保存', exact: true }).click(); await expect(dialog(page).getByRole('heading', { name: '搬家纪念' })).toBeVisible()
  await dialog(page).getByRole('button', { name: '删除', exact: true }).click(); await page.getByRole('button', { name: '删除事件', exact: true }).click(); await expect(page.locator('.life-event-card h3')).toHaveText(['今天的决定']); expect(state.events).toHaveLength(1)
})
test('保存失败保留输入，回复丢失后重试不重复，关闭保护修改', async ({ page }) => {
  const state = await mockLife(page); await page.goto('/life'); await page.getByRole('button', { name: '+ 记录生活事件' }).click(); await dialog(page).getByLabel('事件名称').fill('重要决定'); state.failSave = true; await dialog(page).getByRole('button', { name: '保存', exact: true }).click(); await expect(dialog(page).getByText('保存失败，请重试。')).toBeVisible(); await expect(dialog(page).getByLabel('事件名称')).toHaveValue('重要决定')
  await dialog(page).getByRole('button', { name: '取消', exact: true }).click(); await page.getByRole('button', { name: '继续编辑', exact: true }).click(); await expect(dialog(page)).toBeVisible()
  state.failSave = false; state.loseReply = true; await dialog(page).getByRole('button', { name: '保存', exact: true }).click(); await expect(dialog(page).getByText(/无法连接/)).toBeVisible(); await dialog(page).getByRole('button', { name: '保存', exact: true }).click(); await expect(dialog(page)).not.toBeVisible(); expect(state.events).toHaveLength(1); expect(new Set(state.saveKeys).size).toBe(1)
})
test('领域与里程碑筛选，安全 Obsidian 链接和移动端弹层', async ({ page }) => {
  await mockLife(page); await page.setViewportSize({ width: 390, height: 844 }); await page.goto('/life'); await page.getByRole('button', { name: '+ 记录生活事件' }).click(); const d = dialog(page); await d.getByLabel('事件名称').fill('自己的家'); await d.getByText('这是一个重要里程碑', { exact: true }).click(); await d.locator('summary').click(); await selectOption(page, d.getByLabel('主领域', { exact: true }), '居住与生活环境'); await selectOption(page, d.getByLabel('次领域', { exact: true }), '工作与创造'); await d.getByLabel('外部资料链接').fill('javascript:alert(1)'); await d.getByRole('button', { name: '保存', exact: true }).click(); await expect(d.getByText(/链接仅支持/)).toBeVisible(); await d.getByLabel('外部资料链接').fill(uri); await d.getByRole('button', { name: '保存', exact: true }).click(); await expect(d).not.toBeVisible()
  await selectOption(page, page.getByRole('combobox', { name: '筛选生活领域' }), '工作与创造'); await page.getByText('只看里程碑', { exact: true }).click(); await expect(page.locator('.life-event-card h3')).toHaveText(['自己的家']); await page.locator('.life-event-card').click(); await expect(dialog(page).getByRole('link', { name: /打开 Obsidian/ })).toHaveAttribute('href', uri); await expect(dialog(page).getByRole('button', { name: '复制链接' })).toBeVisible(); expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true); await expect(dialog(page).getByRole('button', { name: '编辑', exact: true })).toBeInViewport()
})
test('目标状态、主动判断历史与阶段记录恢复', async ({ page }) => {
  const state = await mockLife(page); await page.goto('/life'); await page.getByRole('tab', { name: '长期目标' }).click(); await page.getByRole('button', { name: '+ 写下长期目标' }).click(); await dialog(page).getByLabel('目标名称').fill('自己的家'); await dialog(page).getByLabel('为什么重要（可选）').fill('长期居住'); await dialog(page).getByRole('button', { name: '保存目标' }).click(); await expect(dialog(page)).not.toBeVisible(); await page.locator('.life-goal-card').click(); await dialog(page).getByRole('button', { name: '编辑目标' }).click(); await selectOption(page, dialog(page).getByLabel('当前状态'), '进行中'); await dialog(page).getByLabel('变化原因（可选）').fill('开始准备'); await dialog(page).getByRole('button', { name: '保存目标' }).click(); await expect(dialog(page).getByText('考虑中 → 进行中')).toBeVisible(); await dialog(page).getByRole('button', { name: '+ 记录进展或决定' }).click(); await dialog(page).getByLabel('进展或决定').fill('重新评估通勤区域'); await dialog(page).getByRole('button', { name: '保存阶段记录' }).click(); await expect(dialog(page).getByText('重新评估通勤区域')).toBeVisible(); expect(state.entries).toHaveLength(2); await page.reload(); await page.getByRole('tab', { name: '长期目标' }).click(); await page.locator('.life-goal-card').click(); await expect(dialog(page).getByText('重新评估通勤区域')).toBeVisible()
})
test('Inbox 转事件保留原文，已处理入口可重新打开', async ({ page }) => {
  const state = await mockLife(page); await page.goto('/inbox'); await page.getByRole('button', { name: '保存为生活事件', exact: true }).click(); await expect(dialog(page).getByLabel('简短描述（可选）')).toHaveValue('和老朋友重逢的那一天'); await dialog(page).getByLabel('事件名称').fill('重逢'); await dialog(page).getByRole('button', { name: '保存', exact: true }).click(); await expect(page.getByRole('button', { name: '保存为生活事件', exact: true })).toHaveCount(0); expect(state.inbox[0]!.content).toBe('和老朋友重逢的那一天'); expect(state.inbox[0]!.status).toBe('processed'); await page.getByRole('button', { name: /已处理/ }).click(); await page.getByRole('button', { name: '查看生活事件' }).click(); await expect(dialog(page).getByRole('heading', { name: '重逢' })).toBeVisible()
})
test('完成项目手动收录，重复入口打开已有档案', async ({ page }) => {
  const state = await mockLife(page); await page.goto('/projects?view=list'); await page.getByRole('button', { name: '收录到生活档案' }).click(); await expect(dialog(page).getByLabel('事件名称')).toHaveValue('独立作品'); await expect(dialog(page).getByLabel('这是一个重要里程碑')).toBeChecked(); await dialog(page).getByRole('button', { name: '保存', exact: true }).click(); await page.getByRole('button', { name: '收录到生活档案' }).click(); await expect(dialog(page).getByRole('heading', { name: '独立作品' })).toBeVisible(); expect(state.events).toHaveLength(1)
})
for (const width of [1440, 1280, 768, 390]) test(`生活页面 ${width}px 与深色主题保持可读`, async ({ page }) => {
  const state = await mockLife(page); state.events.push({ id: 1, title: '一次值得记住的长途旅行与生活决定', occurred_on: '2026-10-08', description: '整理当时的感受，保留以后可以回看的线索。', primary_domain: 'experiences', secondary_domain: 'rhythm', milestone: true, goal_id: null, external_url: '', link_name: '', source_type: '', source_id: null, source_title: '', source_available: false, source_status: '', created_at: time, updated_at: time }); await page.setViewportSize({ width, height: 900 }); await page.goto('/life'); await expect(page.locator('.life-event-card')).toBeVisible(); expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true); if (process.env.CAPTURE_VISUALS) await page.screenshot({ path: `/app/test-results/life-light-${width}.png`, fullPage: true }); await page.evaluate(() => { document.documentElement.dataset.theme = 'dark'; document.documentElement.style.colorScheme = 'dark' }); await expect.poll(() => page.locator('.life-toolbar .el-select__wrapper').evaluate(e => getComputedStyle(e).backgroundColor)).toBe('rgb(21, 36, 54)'); if (process.env.CAPTURE_VISUALS) await page.screenshot({ path: `/app/test-results/life-dark-${width}.png`, fullPage: true }); await page.locator('.life-event-card').click(); await expect(dialog(page).getByRole('button', { name: '编辑', exact: true })).toBeInViewport()
})

test('课程正式结业后手动收录，结业操作不自动创建事件', async ({ page }) => {
  const state = await mockLife(page); await page.goto('/courses/2/archive'); await expect(page.getByRole('button', { name: '收录到生活档案' })).toHaveCount(0); await page.getByRole('button', { name: '确认课程结业', exact: true }).click(); await page.locator('.el-message-box').getByRole('button', { name: '确认结业', exact: true }).click(); await expect(page.getByRole('button', { name: '收录到生活档案' })).toBeVisible(); expect(state.events).toHaveLength(0); expect(state.graduateCalls).toBe(1); await page.getByRole('button', { name: '收录到生活档案' }).click(); await expect(dialog(page).getByLabel('事件名称')).toHaveValue('系统思考'); await dialog(page).getByRole('button', { name: '保存', exact: true }).click(); expect(state.events[0]!.source_type).toBe('course'); await page.getByRole('button', { name: '收录到生活档案' }).click(); await expect(dialog(page).getByRole('heading', { name: '系统思考' })).toBeVisible(); expect(state.events).toHaveLength(1)
})
