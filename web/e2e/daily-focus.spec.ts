import { expect, test, type BrowserContext, type Locator, type Page, type Route } from '@playwright/test'

/**
 * Daily Focus（每日焦点）E2E。
 *
 * 所有 API 全部通过 context.route 拦截，不接触真实数据库；/workspace/home 的
 * daily_focus 字段与 /workspace/daily-focus* 写入均在内存中模拟。
 * 选择器契约见 docs/DAILY_FOCUS.md 第 4 节与前端实现：
 *   .daily-focus-card / .daily-focus-slot[data-context] / .daily-focus-title /
 *   .daily-focus-state / .focus-picker / .focus-picker-candidate / .continue-focus
 */

const day = '2026-10-21'
const nextDay = '2026-10-22'
const stamp = '2026-10-21T08:00:00Z'
const doneStamp = '2026-10-21T09:30:00Z'

type Scene = 'work' | 'life'
type State = 'ok' | 'completed' | 'missing' | 'context_changed' | 'project_inactive' | 'archived'

interface Project {
  id: number
  title: string
  description: string
  context: 'work' | 'life' | 'unclassified'
  status: string
  icon: string
  accent: string
  archived_at: string | null
  tasks: unknown[]
  open_task_count: number
  doing_task_count: number
  next_task_count: number
  done_task_count: number
  created_at: string
  updated_at: string
}

interface Task {
  id: number
  project_id: number
  title: string
  description: string
  status: string
  sort_order: number
  priority: string
  due_date: string | null
  plan_date: string | null
  completed_at: string | null
  created_at: string
  updated_at: string
  archived_at: string | null
}

interface FocusRecord {
  id: number
  date: string
  context: Scene
  task_id: number
  revision: number
  updated_at: string
}

interface FocusView {
  id: number
  date: string
  context: Scene
  task_id: number | null
  revision: number
  state: State
  title: string
  task_status: string
  priority: string
  plan_date: string | null
  due_date: string | null
  completed_at: string | null
  project_id: number | null
  project_title: string
  project_context: string
  project_status: string
  state_reason: string
  updated_at: string
  task_updated_at: string | null
}

function project(id: number, title: string, context: Project['context'], status = 'active'): Project {
  return {
    id, title, description: '', context, status, icon: title.slice(0, 1), accent: '', archived_at: null,
    tasks: [], open_task_count: 1, doing_task_count: 0, next_task_count: 1, done_task_count: 0,
    created_at: stamp, updated_at: stamp,
  }
}

function task(id: number, projectID: number, title: string, patch: Partial<Task> = {}): Task {
  return {
    id, project_id: projectID, title, description: '', status: 'next', sort_order: id * 1024, priority: 'normal',
    due_date: null, plan_date: null, completed_at: null, created_at: stamp, updated_at: stamp, archived_at: null,
    ...patch,
  }
}

async function installDailyFocusMocks(context: BrowserContext) {
  const projects: Project[] = [
    project(1, '正职项目', 'work'),
    project(2, '个人生活', 'life'),
    project(3, '暂停项目', 'work', 'paused'),
    project(4, '未分类项目', 'unclassified'),
    project(5, '归档项目', 'work', 'archived'),
  ]
  const tasks: Task[] = [
    task(11, 1, '完成提案初稿', { status: 'next', priority: 'high', plan_date: day, due_date: '2026-10-23' }),
    task(12, 1, '整理会议记录', { status: 'inbox', plan_date: day }),
    task(13, 1, '写周报', { status: 'doing', due_date: day }),
    task(21, 2, '预约体检', { status: 'next', plan_date: day }),
    task(22, 2, '买跑鞋', { status: 'inbox' }),
    task(31, 1, '已完成任务', { status: 'done', completed_at: doneStamp }),
    task(32, 1, '已归档任务', { status: 'next', archived_at: stamp }),
    task(41, 4, '未分类任务', { status: 'next', plan_date: day }),
    task(51, 3, '暂停项目任务', { status: 'next', plan_date: day }),
    task(61, 2, '生活项目里的旧焦点', { status: 'next', plan_date: day }),
    task(71, 5, '归档项目任务', { status: 'next', plan_date: day }),
  ]
  const records = new Map<string, FocusRecord>()
  let nextFocusID = 100

  const state = {
    puts: [] as { date: string; context: Scene; task_id: number | null; expected_revision?: number }[],
    patches: [] as Record<string, unknown>[],
    patchCalls: 0,
    patchGate: null as Promise<void> | null,
    failPutTimes: 0,
    homeRequests: [] as { date: string; context: string }[],
    candidateRequests: [] as { context: string | null; plan_date: string | null; actionable: boolean; q: string }[],
    workHomeGate: null as Promise<void> | null,
    pendingWorkHome: false,
  }

  function key(date: string, scene: Scene) {
    return date + ':' + scene
  }

  function seedFocus(scene: Scene, taskID: number, date = day, revision = 1) {
    records.set(key(date, scene), { id: nextFocusID++, date, context: scene, task_id: taskID, revision, updated_at: stamp })
  }

  function bumpRevision(scene: Scene, date = day) {
    const record = records.get(key(date, scene))
    if (record) record.revision += 1
  }

  function focusView(date: string, scene: Scene): FocusView | null {
    const record = records.get(key(date, scene))
    if (!record) return null
    const current = tasks.find(item => item.id === record.task_id)
    const currentProject = current ? projects.find(item => item.id === current.project_id) : undefined
    let focusState: State = 'ok'
    let reason = ''
    if (!current) {
      focusState = 'missing'
      reason = '原任务已被删除'
    } else if (current.status === 'done') {
      focusState = 'completed'
    } else if (currentProject && currentProject.context !== scene) {
      focusState = 'context_changed'
      reason = '所属项目已不在当前场景'
    } else if (currentProject && currentProject.status !== 'active') {
      focusState = 'project_inactive'
      reason = currentProject.status === 'paused' ? '所属项目已暂停' : '所属项目已归档'
    } else if (current.archived_at) {
      focusState = 'archived'
      reason = '任务已归档'
    }
    return {
      id: record.id,
      date: record.date,
      context: record.context,
      task_id: record.task_id,
      revision: record.revision,
      state: focusState,
      title: current?.title ?? '',
      task_status: current?.status ?? '',
      priority: current?.priority ?? 'normal',
      plan_date: current?.plan_date ?? null,
      due_date: current?.due_date ?? null,
      completed_at: current?.completed_at ?? null,
      project_id: currentProject?.id ?? null,
      project_title: currentProject?.title ?? '',
      project_context: currentProject?.context ?? '',
      project_status: currentProject?.status ?? '',
      state_reason: reason,
      updated_at: record.updated_at,
      task_updated_at: current?.updated_at ?? null,
    }
  }

  function focusMap(date: string, scene: string) {
    if (scene === 'work') return { work: focusView(date, 'work'), life: null }
    if (scene === 'life') return { work: null, life: focusView(date, 'life') }
    return { work: focusView(date, 'work'), life: focusView(date, 'life') }
  }

  function projectOf(item: Task) {
    return projects.find(entry => entry.id === item.project_id)
  }

  function usableCandidate(item: Task, scene: Scene) {
    const owner = projectOf(item)
    return Boolean(owner) && owner!.context === scene && owner!.status === 'active'
      && !item.archived_at && item.status !== 'done'
  }

  function withProject(item: Task) {
    return { ...item, project_title: projectOf(item)?.title ?? '', effective_context: projectOf(item)?.context ?? '' }
  }

  await context.route('**/api/v1/**', async (route: Route) => {
    const request = route.request()
    const url = new URL(request.url())
    const path = url.pathname
    const method = request.method()
    const reply = (data: unknown, status = 200) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify({ data }) })
    const fail = (status: number, code: string, message: string, retryable = status >= 500) =>
      route.fulfill({ status, contentType: 'application/json', body: JSON.stringify({ error: { code, message, retryable } }) })

    if (path === '/api/v1/auth/me') return reply({ id: 1, username: 'daily-focus', role: 'admin', status: 'active' })
    if (path === '/api/v1/workspace/recent') return reply({ library: [], events: [], goals: [], asset_date: '', errors: {} })
    if (path === '/api/v1/projects/today') return reply({ date: day, doing: [], due: [], next: [] })
    if (path === '/api/v1/inbox' && method === 'GET') return reply({ items: [], counts: { inbox: 0, processed: 0, archived: 0 } })
    if (path === '/api/v1/projects' && method === 'GET') return reply(projects)

    if (path === '/api/v1/workspace/home' && method === 'GET') {
      const requestedDate = url.searchParams.get('date') ?? day
      const scene = url.searchParams.get('context') ?? 'all'
      state.homeRequests.push({ date: requestedDate, context: scene })
      if (scene === 'work' && state.workHomeGate) {
        state.pendingWorkHome = true
        await state.workHomeGate
        state.pendingWorkHome = false
      }
      const inScene = tasks.filter(item => {
        const owner = projectOf(item)
        return Boolean(owner) && owner!.context !== 'unclassified'
          && (scene === 'all' || owner!.context === scene)
          && item.status !== 'done' && !item.archived_at
      })
      const planning = {
        items: inScene.filter(item => item.plan_date === requestedDate).map(withProject),
        due_today: inScene.filter(item => item.plan_date !== requestedDate && item.due_date === requestedDate).map(withProject),
        overdue: [] as ReturnType<typeof withProject>[],
        previous_plan: [] as ReturnType<typeof withProject>[],
        summary: { planned: 0, due_today: 0, overdue: 0, previous_plan: 0, total: 0 },
      }
      planning.summary = {
        planned: planning.items.length, due_today: planning.due_today.length,
        overdue: 0, previous_plan: 0, total: planning.items.length + planning.due_today.length,
      }
      const suggestion = tasks.find(item => item.id === 13)!
      try {
        return await reply({
          date: requestedDate,
          context: scene,
          unclassified_count: 0,
          focus: { kind: 'task', title: suggestion.title, task_id: suggestion.id, project_id: suggestion.project_id, project_title: projectOf(suggestion)!.title, status: suggestion.status, priority: suggestion.priority, due_date: suggestion.due_date },
          daily_focus: focusMap(requestedDate, scene),
          today: { date: requestedDate, doing: [withProject(suggestion)], due: [], next: [] },
          planning,
          projects: projects.filter(entry => entry.context !== 'unclassified' && (scene === 'all' || entry.context === scene)),
          learning: [],
          inbox: { count: 0, items: [] },
          errors: {},
        })
      } catch {
        return route.fulfill({ status: 499, body: 'aborted' }).catch(() => undefined)
      }
    }

    if (path === '/api/v1/workspace/daily-focus' && method === 'GET') {
      const requestedDate = url.searchParams.get('date') ?? ''
      const scene = url.searchParams.get('context') ?? 'all'
      if (!requestedDate) return fail(400, 'DAILY_FOCUS_DATE_INVALID', '日期无效。', false)
      return reply({ date: requestedDate, context: scene, focuses: focusMap(requestedDate, scene) })
    }

    const writeMatch = path.match(/^\/api\/v1\/workspace\/daily-focus\/(\d{4}-\d{2}-\d{2})\/(work|life)$/)
    if (writeMatch && method === 'PUT') {
      const requestedDate = writeMatch[1]
      const scene = writeMatch[2] as Scene
      const body = request.postDataJSON() as { task_id?: number | null; expected_revision?: number }
      state.puts.push({ date: requestedDate, context: scene, task_id: body.task_id ?? null, expected_revision: body.expected_revision })
      if (state.failPutTimes > 0) {
        state.failPutTimes -= 1
        return fail(503, 'TEMPORARY', '暂时无法保存焦点，请重试。')
      }
      const record = records.get(key(requestedDate, scene))
      const expected = body.expected_revision ?? 0
      if (expected > 0 && (!record || record.revision !== expected)) {
        return fail(409, 'DAILY_FOCUS_CONFLICT', '焦点已被其他页面更新，请刷新后重试。')
      }
      if (body.task_id === null || body.task_id === undefined) {
        records.delete(key(requestedDate, scene))
        return reply({ focus: null, revision: 0 })
      }
      const target = tasks.find(item => item.id === body.task_id)
      if (!target || !usableCandidate(target, scene)) {
        return fail(400, 'DAILY_FOCUS_TASK_INVALID', '该任务不能作为当前场景的焦点。', false)
      }
      if (record && record.task_id === target.id) return reply({ focus: focusView(requestedDate, scene), revision: record.revision })
      const revision = (record?.revision ?? 0) + 1
      records.set(key(requestedDate, scene), {
        id: record?.id ?? nextFocusID++, date: requestedDate, context: scene, task_id: target.id, revision,
        updated_at: new Date().toISOString(),
      })
      return reply({ focus: focusView(requestedDate, scene), revision })
    }

    if (path === '/api/v1/tasks' && method === 'GET') {
      const context = url.searchParams.get('context')
      const planDate = url.searchParams.get('plan_date')
      const query = (url.searchParams.get('q') ?? '').trim()
      const actionable = url.searchParams.get('actionable') === 'true'
      state.candidateRequests.push({ context, plan_date: planDate, actionable, q: query })
      let rows = tasks.filter(item => {
        const owner = projectOf(item)
        if (!owner || owner.context === 'unclassified') return false
        if (context && context !== 'all' && owner.context !== context) return false
        if (actionable) {
          if (owner.status !== 'active' || item.archived_at || item.status === 'done') return false
        }
        if (planDate && item.plan_date !== planDate) return false
        if (query && !item.title.includes(query)) return false
        return true
      })
      const offset = Number(url.searchParams.get('offset') ?? 0)
      const limit = Number(url.searchParams.get('limit') ?? rows.length)
      rows = rows.slice(offset, offset + limit)
      return reply(rows.map(withProject))
    }

    const patchMatch = path.match(/^\/api\/v1\/tasks\/(\d+)$/)
    if (patchMatch && method === 'PATCH') {
      const id = Number(patchMatch[1])
      const target = tasks.find(item => item.id === id)
      const body = request.postDataJSON() as Record<string, unknown>
      state.patchCalls += 1
      state.patches.push(body)
      if (state.patchGate) await state.patchGate
      if (!target) return fail(404, 'TASK_NOT_FOUND', '任务不存在。', false)
      if (body.status === 'done' && target.status !== 'done' && !target.completed_at) target.completed_at = doneStamp
      Object.assign(target, body, { updated_at: new Date().toISOString() })
      return reply(target)
    }

    const taskMatch = path.match(/^\/api\/v1\/tasks\/(\d+)$/)
    if (taskMatch && method === 'GET') {
      const target = tasks.find(item => item.id === Number(taskMatch[1]))
      return target ? reply(target) : fail(404, 'TASK_NOT_FOUND', '任务不存在。', false)
    }

    return reply({})
  })

  return { tasks, projects, state, seedFocus, bumpRevision }
}

type Mock = Awaited<ReturnType<typeof installDailyFocusMocks>>

const focusCard = (page: Page) => page.locator('.daily-focus-card')
const slot = (page: Page, scene: Scene) => page.locator(`.daily-focus-slot[data-context="${scene}"]`)
const slotTitle = (page: Page, scene: Scene) => slot(page, scene).locator('.daily-focus-title')
const focusState = (page: Page, scene: Scene) => slot(page, scene).locator('.daily-focus-state')
const pickerOf = (page: Page) => page.getByRole('dialog', { name: '选择今日焦点' })
const candidatesOf = (picker: Locator) => picker.locator('.focus-picker-candidate')
const cardAlert = (page: Page) => focusCard(page).locator('.daily-focus-error, [role="alert"]')
const pickerAlert = (picker: Locator) => picker.locator('.focus-picker__error, [role="alert"]')
const emptyHeading = (page: Page) => focusCard(page).locator('.daily-focus-empty #focus-title')
const pickerSearch = (picker: Locator) => picker.getByRole('textbox', { name: '搜索可操作任务' })

async function setup(page: Page, context: BrowserContext, options: { scene?: 'all' | Scene; theme?: 'light' | 'dark'; seed?: (mock: Mock) => void } = {}) {
  await page.clock.install({ time: new Date('2026-10-21T12:00:00') })
  const scene = options.scene ?? 'all'
  await page.addInitScript(({ scene, theme }: { scene: string; theme?: string }) => {
    localStorage.setItem('learnos:workspace-context:1', scene)
    if (theme) localStorage.setItem('learnos-theme', theme)
  }, { scene, theme: options.theme })
  const mock = await installDailyFocusMocks(context)
  options.seed?.(mock)
  await page.goto('/')
  return mock
}

async function openPicker(page: Page, entry: Locator) {
  await entry.click()
  const picker = pickerOf(page)
  await expect(picker).toBeVisible()
  return picker
}

async function chooseCandidate(picker: Locator, title: string) {
  await candidatesOf(picker).filter({ hasText: title }).click()
  await expect(picker).toBeHidden()
}

test('工作与生活焦点互相独立，all 视图展示两份摘要并在刷新后保持', async ({ page, context }) => {
  const mock = await setup(page, context)
  const card = focusCard(page)
  const work = slot(page, 'work'), life = slot(page, 'life')
  await expect(card.locator('.daily-focus-empty')).toBeVisible()

  await chooseCandidate(await openPicker(page, card.getByRole('button', { name: '设置工作焦点' })), '完成提案初稿')
  await expect(slotTitle(page, 'work')).toHaveText('完成提案初稿')
  await expect(work).toHaveAttribute('data-state', 'ok')
  await expect(life).toHaveAttribute('data-state', 'empty')

  await chooseCandidate(await openPicker(page, life.getByRole('button', { name: '设置生活焦点' })), '预约体检')
  await expect(slotTitle(page, 'life')).toHaveText('预约体检')
  await expect(work).toHaveAttribute('data-state', 'ok')

  const puts = mock.state.puts.map(put => ({ context: put.context, task_id: put.task_id }))
  expect(puts).toEqual([{ context: 'work', task_id: 11 }, { context: 'life', task_id: 21 }])

  await page.reload()
  await expect(slotTitle(page, 'work')).toHaveText('完成提案初稿')
  await expect(slotTitle(page, 'life')).toHaveText('预约体检')
})

test('单场景只展示本场景焦点，切换场景后互不串用', async ({ page, context }) => {
  const mock = await setup(page, context, { scene: 'work', seed: mock => mock.seedFocus('work', 11) })
  await expect(slotTitle(page, 'work')).toHaveText('完成提案初稿')
  await expect(slot(page, 'life')).toHaveCount(0)

  await page.locator('.workspace-context-switch label').filter({ hasText: '生活' }).click()
  await expect(page.getByRole('radio', { name: '生活', exact: true })).toBeChecked()
  await expect(page.locator('.daily-focus-empty')).toBeVisible()
  await expect(page.getByRole('heading', { name: '今天可以留白', exact: true })).toBeVisible()
  await expect(slot(page, 'life')).toHaveCount(0)
  await expect(slot(page, 'work')).toHaveCount(0)
  expect(mock.state.homeRequests[mock.state.homeRequests.length - 1]).toEqual({ date: day, context: 'life' })
})

test('空状态：单场景一个入口，all 双空只渲染一个轻量空状态与两个入口', async ({ page, context }) => {
  await setup(page, context, { scene: 'work' })
  await expect(page.locator('.daily-focus-empty')).toBeVisible()
  await expect(emptyHeading(page)).toHaveText('设置今日焦点')
  await expect(focusCard(page).getByRole('button', { name: '设置工作焦点' })).toBeVisible()
  await expect(page.getByRole('button', { name: '设置生活焦点' })).toHaveCount(0)
  // 自动建议只能留在次级「继续处理建议」行，绝不写入焦点。
  await expect(page.locator('.continue-focus')).toContainText('写周报')
  await expect(page.locator('.daily-focus-slot')).toHaveCount(0)

  await page.locator('.workspace-context-switch label').filter({ hasText: '全部' }).click()
  const card = focusCard(page)
  await expect(card.locator('.daily-focus-empty')).toHaveCount(1)
  await expect(card.getByRole('button', { name: '设置工作焦点' })).toBeVisible()
  await expect(card.getByRole('button', { name: '设置生活焦点' })).toBeVisible()
  await expect(page.locator('.daily-focus-slot')).toHaveCount(0)
})

test('选择器默认展示今天已计划候选，搜索可找到其他可操作任务', async ({ page, context }) => {
  const mock = await setup(page, context, { scene: 'work' })
  const picker = await openPicker(page, focusCard(page).getByRole('button', { name: '设置工作焦点' }))
  const candidates = candidatesOf(picker)
  await expect(candidates).toHaveCount(2)
  await expect(candidates).toContainText(['完成提案初稿', '整理会议记录'])

  const request = mock.state.candidateRequests[mock.state.candidateRequests.length - 1]
  expect(request.context).toBe('work')
  expect(request.actionable).toBe(true)
  expect(request.plan_date).toBe(day)

  await pickerSearch(picker).fill('周报')
  await expect(candidates).toHaveCount(1)
  await expect(candidates.first()).toContainText('写周报')
  await expect(candidates.first()).toContainText('正职项目')
  await candidates.first().click()
  await expect(picker).toBeHidden()
  await expect(slotTitle(page, 'work')).toHaveText('写周报')
  expect(mock.state.puts[mock.state.puts.length - 1].task_id).toBe(13)
})

test('已完成、已归档与未分类任务不作为焦点候选', async ({ page, context }) => {
  await setup(page, context, { scene: 'work' })
  const picker = await openPicker(page, focusCard(page).getByRole('button', { name: '设置工作焦点' }))
  const candidates = candidatesOf(picker)
  await expect(candidates).toHaveCount(2)
  for (const title of ['已完成任务', '已归档任务', '未分类任务', '暂停项目任务', '归档项目任务']) {
    await expect(picker).not.toContainText(title)
  }

  const search = pickerSearch(picker)
  for (const query of ['已完成', '归档', '未分类']) {
    await search.fill(query)
    await expect(candidates).toHaveCount(0)
  }
})

test('完成焦点同步任务：重复点击只发送一次 PATCH 且不覆盖完成时间', async ({ page, context }) => {
  const mock = await setup(page, context, { seed: mock => mock.seedFocus('work', 11) })
  const work = slot(page, 'work')
  let release!: () => void
  mock.state.patchGate = new Promise<void>(resolve => { release = resolve })
  const complete = work.getByRole('button', { name: /完成/ })
  await complete.click()
  await expect.poll(() => mock.state.patchCalls).toBe(1)
  await complete.evaluate(element => (element as HTMLElement).click()).catch(() => undefined)
  release()

  await expect(work).toHaveAttribute('data-state', 'completed')
  expect(mock.state.patchCalls).toBe(1)
  expect(mock.state.patches[0]).toMatchObject({ status: 'done' })
  const target = mock.tasks.find(item => item.id === 11)!
  expect(target.status).toBe('done')
  expect(target.completed_at).toBe(doneStamp)

  await page.reload()
  await expect(slot(page, 'work')).toHaveAttribute('data-state', 'completed')
  expect(mock.state.patchCalls).toBe(1)
  expect(mock.tasks.find(item => item.id === 11)!.completed_at).toBe(doneStamp)
})

test('更换焦点不改旧任务状态、计划日期与截止日期', async ({ page, context }) => {
  const mock = await setup(page, context, { seed: mock => mock.seedFocus('work', 11) })
  const before = structuredClone(mock.tasks.find(item => item.id === 11)!)
  const picker = await openPicker(page, slot(page, 'work').getByRole('button', { name: /更换/ }))
  await chooseCandidate(picker, '整理会议记录')

  await expect(slotTitle(page, 'work')).toHaveText('整理会议记录')
  expect(mock.state.patchCalls).toBe(0)
  expect(mock.tasks.find(item => item.id === 11)).toEqual(before)
  expect(mock.state.puts[mock.state.puts.length - 1]).toMatchObject({ context: 'work', task_id: 12, expected_revision: 1 })
})

test('清空焦点只影响当前场景，生活焦点保持不变', async ({ page, context }) => {
  const mock = await setup(page, context, {
    seed: mock => { mock.seedFocus('work', 11); mock.seedFocus('life', 21) },
  })
  await slot(page, 'work').getByRole('button', { name: /清空/ }).click()
  await expect(slot(page, 'work')).toHaveAttribute('data-state', 'empty')
  await expect(slotTitle(page, 'work')).toHaveCount(0)
  await expect(slotTitle(page, 'life')).toHaveText('预约体检')
  const put = mock.state.puts[mock.state.puts.length - 1]
  expect(put.context).toBe('work')
  expect(put.task_id).toBeNull()
})

const invalidStates: { name: string; taskID: number; state: State; label: string; reason: string }[] = [
  { name: '任务被删除（missing）', taskID: 999, state: 'missing', label: '任务已删除', reason: '原任务已被删除' },
  { name: '项目暂停（project_inactive）', taskID: 51, state: 'project_inactive', label: '项目不可用', reason: '所属项目已暂停' },
  { name: '项目改场景（context_changed）', taskID: 61, state: 'context_changed', label: '场景已变更', reason: '所属项目已不在当前场景' },
  { name: '任务归档（archived）', taskID: 32, state: 'archived', label: '任务已归档', reason: '任务已归档' },
]

for (const scenario of invalidStates) {
  test('失效状态：' + scenario.name, async ({ page, context }) => {
    await setup(page, context, { scene: 'work', seed: mock => mock.seedFocus('work', scenario.taskID) })
    const work = slot(page, 'work')
    await expect(work).toHaveAttribute('data-state', scenario.state)
    await expect(focusState(page, 'work')).toHaveText(scenario.label)
    await expect(work.locator('.daily-focus-reason')).toContainText(scenario.reason)
    await expect(work.getByRole('button', { name: /完成/ })).toHaveCount(0)
    await expect(work.getByRole('link')).toHaveCount(0)
    await expect(work.getByRole('button', { name: /更换/ })).toBeVisible()
    await expect(work.getByRole('button', { name: /清空/ })).toBeVisible()
  })
}

test('旧 revision 冲突提示刷新，刷新后重试成功且保留最后确认状态', async ({ page, context }) => {
  const mock = await setup(page, context, { seed: mock => mock.seedFocus('work', 11) })
  await expect(slotTitle(page, 'work')).toHaveText('完成提案初稿')
  mock.bumpRevision('work')
  const picker = await openPicker(page, slot(page, 'work').getByRole('button', { name: /更换/ }))
  const reloaded = page.waitForResponse(response => response.url().includes('/api/v1/workspace/home') && response.status() === 200)
  await candidatesOf(picker).filter({ hasText: '整理会议记录' }).click()

  await expect(pickerAlert(picker)).toBeVisible()
  await expect(pickerAlert(picker)).toContainText(/刷新|更新/)
  await expect(slotTitle(page, 'work')).toHaveText('完成提案初稿')
  expect(mock.state.puts[0].expected_revision).toBe(1)

  await reloaded
  await page.waitForTimeout(150)
  await candidatesOf(picker).filter({ hasText: '整理会议记录' }).click()
  await expect(picker).toBeHidden()
  await expect(slotTitle(page, 'work')).toHaveText('整理会议记录')
  expect(mock.state.puts[mock.state.puts.length - 1].expected_revision).toBe(2)
})

test('保存失败保留最后确认焦点，重试成功后立即更新', async ({ page, context }) => {
  const mock = await setup(page, context, { seed: mock => mock.seedFocus('work', 11) })
  mock.state.failPutTimes = 1
  const picker = await openPicker(page, slot(page, 'work').getByRole('button', { name: /更换/ }))
  await candidatesOf(picker).filter({ hasText: '整理会议记录' }).click()

  await expect(pickerAlert(picker)).toBeVisible()
  await expect(pickerAlert(picker)).toContainText(/保存|重试/)
  await expect(slotTitle(page, 'work')).toHaveText('完成提案初稿')
  expect(mock.tasks.find(item => item.id === 11)!.plan_date).toBe(day)

  await picker.getByRole('button', { name: '重试' }).click()
  await expect(picker).toBeHidden()
  await expect(slotTitle(page, 'work')).toHaveText('整理会议记录')
  expect(mock.state.puts).toHaveLength(2)
})

test('跨日不继承昨日焦点', async ({ page, context }) => {
  await page.clock.install({ time: new Date('2026-10-21T23:59:50') })
  await page.addInitScript(() => localStorage.setItem('learnos:workspace-context:1', 'work'))
  const mock = await installDailyFocusMocks(context)
  mock.seedFocus('work', 11)
  await page.goto('/')
  await expect(slotTitle(page, 'work')).toHaveText('完成提案初稿')

  await page.clock.runFor(35_000)
  await expect(page.locator('.daily-focus-empty')).toBeVisible()
  await expect(emptyHeading(page)).toHaveText('设置今日焦点')
  await expect(slotTitle(page, 'work')).toHaveCount(0)
  expect(mock.state.homeRequests.some(item => item.date === nextDay && item.context === 'work')).toBe(true)
})

test('快速切换场景时旧响应不覆盖新场景焦点', async ({ page, context }) => {
  const mock = await setup(page, context, {
    seed: mock => { mock.seedFocus('work', 11); mock.seedFocus('life', 21) },
  })
  let release!: () => void
  mock.state.workHomeGate = new Promise<void>(resolve => { release = resolve })

  await page.locator('.workspace-context-switch label').filter({ hasText: '工作' }).click()
  await expect.poll(() => mock.state.pendingWorkHome).toBe(true)
  await page.locator('.workspace-context-switch label').filter({ hasText: '生活' }).click()
  await expect(slotTitle(page, 'life')).toHaveText('预约体检')

  release()
  await page.waitForTimeout(300)
  await expect(page.getByRole('radio', { name: '生活', exact: true })).toBeChecked()
  await expect(slotTitle(page, 'life')).toHaveText('预约体检')
  await expect(slot(page, 'work')).toHaveCount(0)
  const last = mock.state.homeRequests[mock.state.homeRequests.length - 1]
  expect(last.context).toBe('life')
})

for (const theme of ['light', 'dark'] as const) {
  test(`390px ${theme} 主题下焦点卡可操作且不溢出`, async ({ page, context }, info) => {
    await page.setViewportSize({ width: 390, height: 844 })
    await setup(page, context, { theme, seed: mock => mock.seedFocus('work', 11) })
    await expect(page.locator('html')).toHaveAttribute('data-theme', theme)
    const work = slot(page, 'work')
    await expect(work).toHaveAttribute('data-state', 'ok')
    await expect(slotTitle(page, 'work')).toHaveText('完成提案初稿')
    await expect(slot(page, 'life')).toHaveAttribute('data-state', 'empty')
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
    for (const label of [/更换/, /清空/]) {
      const button = work.getByRole('button', { name: label })
      expect((await button.boundingBox())!.height).toBeGreaterThanOrEqual(44)
    }
    if (process.env.CAPTURE_VISUALS) await page.screenshot({ path: info.outputPath(`daily-focus-${theme}-390.png`), fullPage: true })
  })
}
