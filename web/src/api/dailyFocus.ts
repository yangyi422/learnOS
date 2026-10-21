import type { ProjectTask, TaskPriority } from './projects'
import { request } from './http'

export type DailyFocusContext = 'work' | 'life'
export type DailyFocusState = 'ok' | 'completed' | 'missing' | 'context_changed' | 'project_inactive' | 'archived'

// One scene's focus for one calendar day. The server resolves the referenced
// task on every read, so the view is safe to render even after the task,
// project or scene changed or was removed.
export interface DailyFocusView {
  id: number
  date: string
  context: DailyFocusContext
  task_id: number | null
  revision: number
  state: DailyFocusState
  title: string
  task_status: string
  priority: TaskPriority | null
  plan_date: string | null
  due_date: string | null
  completed_at: string | null
  project_id: number | null
  project_title: string
  project_context: string
  project_status: string
  state_reason: string
  // The referenced task's real updated_at (null when the task is gone). Used as
  // the optimistic-lock precondition when completing the focus.
  task_updated_at?: string | null
  updated_at: string
}

export interface WorkspaceDailyFocus {
  work: DailyFocusView | null
  life: DailyFocusView | null
}

export interface DailyFocusQueryResult {
  date: string
  context: 'all' | DailyFocusContext
  focuses: WorkspaceDailyFocus
}

export interface DailyFocusPutResult {
  focus: DailyFocusView | null
  revision: number
}

export function emptyDailyFocus(): WorkspaceDailyFocus {
  return { work: null, life: null }
}

export async function getDailyFocus(date: string, context: 'all' | DailyFocusContext = 'all', signal?: AbortSignal): Promise<DailyFocusQueryResult> {
  const params = new URLSearchParams({ date, context })
  return (await request<{ data: DailyFocusQueryResult }>(`/api/v1/workspace/daily-focus?${params}`, { signal })).data
}

// taskID null clears the day's focus for that scene. A revision above zero is
// an optimistic-lock precondition; the server answers 409 when it is stale.
export async function setDailyFocus(date: string, context: DailyFocusContext, taskID: number | null, expectedRevision = 0, signal?: AbortSignal): Promise<DailyFocusPutResult> {
  const body: { task_id: number | null; expected_revision?: number } = { task_id: taskID }
  if (expectedRevision > 0) body.expected_revision = expectedRevision
  const path = `/api/v1/workspace/daily-focus/${encodeURIComponent(date)}/${context}`
  return (await request<{ data: DailyFocusPutResult }>(path, { method: 'PUT', body: JSON.stringify(body), signal })).data
}

export interface DailyFocusStateInfo {
  label: string
  tone: 'active' | 'done' | 'invalid'
  canComplete: boolean
  canClear: boolean
  canLink: boolean
}

const stateInfo: Record<DailyFocusState, DailyFocusStateInfo> = {
  ok: { label: '进行中', tone: 'active', canComplete: true, canClear: true, canLink: true },
  completed: { label: '已完成', tone: 'done', canComplete: false, canClear: false, canLink: false },
  missing: { label: '任务已删除', tone: 'invalid', canComplete: false, canClear: true, canLink: false },
  context_changed: { label: '场景已变更', tone: 'invalid', canComplete: false, canClear: true, canLink: false },
  project_inactive: { label: '项目不可用', tone: 'invalid', canComplete: false, canClear: true, canLink: false },
  archived: { label: '任务已归档', tone: 'invalid', canComplete: false, canClear: true, canLink: false },
}

const unknownStateInfo: DailyFocusStateInfo = { label: '状态未知', tone: 'invalid', canComplete: false, canClear: true, canLink: false }

export function describeDailyFocusState(state: DailyFocusState | string | null | undefined): DailyFocusStateInfo {
  return (state && stateInfo[state as DailyFocusState]) || unknownStateInfo
}

export function dailyFocusStateReason(focus: Pick<DailyFocusView, 'state' | 'state_reason'>): string {
  if (focus.state_reason) return focus.state_reason
  return ({
    missing: '原任务已被删除，可以重新选择一个焦点。',
    context_changed: '项目场景已经改变，这个焦点不再适用。',
    project_inactive: '所属项目已暂停或归档，这个焦点暂时无法继续。',
    archived: '原任务已归档，可以重新选择一个焦点。',
    completed: '今天的这件焦点已经完成，可以换一件更想做的事。',
    ok: '',
  } as Record<string, string>)[focus.state] ?? ''
}

// A titled RouterLink is the only click-through; action buttons stay separate
// buttons so they can never trigger card navigation.
export function dailyFocusTaskRoute(focus: Pick<DailyFocusView, 'state' | 'project_id' | 'task_id'> | null | undefined): { path: string; query: Record<string, string> } | null {
  if (!focus || focus.state !== 'ok' || focus.project_id == null || focus.task_id == null) return null
  return { path: '/projects', query: { project: String(focus.project_id), view: 'board', task: String(focus.task_id) } }
}

// Server-side filters already exclude completed/archived tasks and tasks from
// unavailable projects; the picker still checks locally before offering them.
export function isFocusCandidate(task: Pick<ProjectTask, 'status'> & { archived_at?: string | null }): boolean {
  return task.status !== 'done' && !task.archived_at
}

export function taskStatusLabel(status: string | null | undefined): string {
  return ({ doing: '进行中', next: '下一步', inbox: '待整理', done: '已完成' } as Record<string, string>)[status ?? ''] ?? '项目任务'
}
