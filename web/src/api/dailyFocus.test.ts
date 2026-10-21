import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  dailyFocusStateReason,
  dailyFocusTaskRoute,
  describeDailyFocusState,
  emptyDailyFocus,
  getDailyFocus,
  isFocusCandidate,
  setDailyFocus,
  taskStatusLabel,
  type DailyFocusView,
} from './dailyFocus'

const timestamp = '2026-10-21T02:10:00Z'

function focusView(overrides: Partial<DailyFocusView> = {}): DailyFocusView {
  return {
    id: 7,
    date: '2026-10-21',
    context: 'work',
    task_id: 42,
    revision: 3,
    state: 'ok',
    title: '完成提案初稿',
    task_status: 'doing',
    priority: 'high',
    plan_date: '2026-10-21',
    due_date: '2026-10-23',
    completed_at: null,
    project_id: 5,
    project_title: '正职项目',
    project_context: 'work',
    project_status: 'active',
    state_reason: '',
    task_updated_at: timestamp,
    updated_at: timestamp,
    ...overrides,
  }
}

function jsonResponse(data: unknown, status = 200) {
  return new Response(JSON.stringify(data), { status })
}

describe('daily focus API', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('reads the focus map for one day and scene', async () => {
    const fetchMock = vi.fn(async (_path: string, _init?: RequestInit) => jsonResponse({
      data: { date: '2026-10-21', context: 'all', focuses: { work: focusView(), life: null } },
    }))
    vi.stubGlobal('fetch', fetchMock)

    const result = await getDailyFocus('2026-10-21', 'all')

    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/workspace/daily-focus?date=2026-10-21&context=all')
    expect(result.focuses.work?.title).toBe('完成提案初稿')
    expect(result.focuses.life).toBeNull()
  })

  it('PUTs a task id with the expected revision and returns the confirmed view', async () => {
    const fetchMock = vi.fn(async (_path: string, _init?: RequestInit) => jsonResponse({
      data: { focus: focusView({ revision: 4 }), revision: 4 },
    }))
    vi.stubGlobal('fetch', fetchMock)

    const result = await setDailyFocus('2026-10-21', 'work', 42, 3)

    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/workspace/daily-focus/2026-10-21/work')
    expect(fetchMock.mock.calls[0]?.[1]).toMatchObject({ method: 'PUT', body: '{"task_id":42,"expected_revision":3}' })
    expect(result.focus?.revision).toBe(4)
  })

  it('clears a focus with task_id null and no revision guard when unset', async () => {
    const fetchMock = vi.fn(async (_path: string, _init?: RequestInit) => jsonResponse({ data: { focus: null, revision: 0 } }))
    vi.stubGlobal('fetch', fetchMock)

    const result = await setDailyFocus('2026-10-21', 'life', null)

    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/workspace/daily-focus/2026-10-21/life')
    expect(fetchMock.mock.calls[0]?.[1]).toMatchObject({ method: 'PUT', body: '{"task_id":null}' })
    expect(result).toEqual({ focus: null, revision: 0 })
  })

  it('surfaces a stale revision as an APIRequestError with HTTP 409', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => jsonResponse({
      error: { code: 'DAILY_FOCUS_CONFLICT', message: '焦点已在其他位置更新。', retryable: true },
    }, 409)))

    await expect(setDailyFocus('2026-10-21', 'work', 42, 2)).rejects.toMatchObject({
      name: 'APIRequestError',
      code: 'DAILY_FOCUS_CONFLICT',
      status: 409,
    })
  })
})

describe('daily focus helpers', () => {
  it('starts from an empty focus map', () => {
    expect(emptyDailyFocus()).toEqual({ work: null, life: null })
  })

  it('links only actionable focuses to the task board', () => {
    expect(dailyFocusTaskRoute(focusView())).toEqual({
      path: '/projects',
      query: { project: '5', view: 'board', task: '42' },
    })
    expect(dailyFocusTaskRoute(focusView({ state: 'completed' }))).toBeNull()
    expect(dailyFocusTaskRoute(focusView({ state: 'missing', task_id: null }))).toBeNull()
    expect(dailyFocusTaskRoute(null)).toBeNull()
  })

  it('describes every server state with consistent actionability', () => {
    const states = {
      ok: { canComplete: true, canClear: true, canLink: true, tone: 'active' },
      completed: { canComplete: false, canClear: false, canLink: false, tone: 'done' },
      missing: { canComplete: false, canClear: true, canLink: false, tone: 'invalid' },
      context_changed: { canComplete: false, canClear: true, canLink: false, tone: 'invalid' },
      project_inactive: { canComplete: false, canClear: true, canLink: false, tone: 'invalid' },
      archived: { canComplete: false, canClear: true, canLink: false, tone: 'invalid' },
    } as const
    for (const [state, expected] of Object.entries(states)) {
      expect(describeDailyFocusState(state)).toMatchObject(expected)
    }
    expect(describeDailyFocusState('unexpected').tone).toBe('invalid')
    expect(describeDailyFocusState(null).canComplete).toBe(false)
  })

  it('falls back to a readable Chinese reason when the server sends none', () => {
    expect(dailyFocusStateReason(focusView({ state: 'missing', state_reason: '原任务已被删除。' }))).toBe('原任务已被删除。')
    expect(dailyFocusStateReason(focusView({ state: 'missing' }))).toContain('删除')
    expect(dailyFocusStateReason(focusView({ state: 'context_changed' }))).toContain('场景')
    expect(dailyFocusStateReason(focusView({ state: 'archived' }))).toContain('归档')
  })

  it('never offers completed or archived tasks as candidates', () => {
    expect(isFocusCandidate({ status: 'doing' })).toBe(true)
    expect(isFocusCandidate({ status: 'done' })).toBe(false)
    expect(isFocusCandidate({ status: 'next', archived_at: timestamp })).toBe(false)
    expect(taskStatusLabel('doing')).toBe('进行中')
    expect(taskStatusLabel('')).toBe('项目任务')
  })
})
