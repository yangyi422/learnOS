import { request } from './http'
import { notifyTaskChange } from '../task-changes'

export type ProjectContext = 'work' | 'life' | 'unclassified'
export type ProjectStatus = 'active' | 'paused' | 'completed' | 'archived'
export type TaskStatus = 'inbox' | 'next' | 'doing' | 'done'
export type TaskPriority = 'high' | 'normal' | 'low'

export interface ProjectTask {
  id: number
  project_id: number
  title: string
  description: string
  status: TaskStatus
  sort_order: number
  priority: TaskPriority
  plan_date?: string | null
  due_date: string | null
  archived_at?: string | null
  completed_at: string | null
  created_at: string
  updated_at: string
}

export interface Project {
  context?: ProjectContext
  id: number
  title: string
  description: string
  status: ProjectStatus
  icon: string
  accent: string
  archived_at: string | null
  tasks: ProjectTask[]
  open_task_count: number
  next_task_count: number
  doing_task_count: number
  archived_task_count?: number
  done_task_count: number
  created_at: string
  updated_at: string
}

export interface TodayView {
  date: string
  doing: ProjectTask[]
  due: ProjectTask[]
  next: ProjectTask[]
}

export interface ProjectInput {
  context?: ProjectContext
  title: string
  description?: string
  icon?: string
  accent?: string
}

export interface TaskInput {
  project_id: number
  title: string
  description?: string
  status?: TaskStatus
  priority?: TaskPriority
  plan_date?: string | null
  due_date?: string | null
}

export async function listProjects(): Promise<Project[]> {
  return (await request<{ data: Project[] }>('/api/v1/projects')).data
}

export async function createProject(input: ProjectInput): Promise<Project> {
  const saved = (await request<{ data: Project }>('/api/v1/projects', { method: 'POST', body: JSON.stringify(input) })).data
  notifyTaskChange()
  return saved
}

export async function updateProject(id: number, patch: Partial<ProjectInput> & { status?: ProjectStatus }): Promise<void> {
  await request(`/api/v1/projects/${id}`, { method: 'PATCH', body: JSON.stringify(patch) })
  notifyTaskChange()
}

export async function listTasks(filters: { context?: 'all' | 'work' | 'life'; planDate?: string; actionable?: boolean; q?: string; signal?: AbortSignal; projectId?: number; archived?: boolean; status?: TaskStatus; due?: string; limit?: number; offset?: number } = {}): Promise<ProjectTask[]> {
  const params = new URLSearchParams()
  if (filters.context) params.set('context', filters.context)
  if (filters.planDate) params.set('plan_date', filters.planDate)
  if (filters.actionable) params.set('actionable', 'true')
  if (filters.q) params.set('q', filters.q)
  if (filters.projectId) params.set('project_id', String(filters.projectId))
  if (filters.archived) params.set('archived', 'true')
  if (filters.status) params.set('status', filters.status)
  if (filters.due) params.set('due', filters.due)
  if (filters.limit) params.set('limit', String(filters.limit))
  if (filters.offset) params.set('offset', String(filters.offset))
  const query = params.size ? `?${params}` : ''
  return (await request<{ data: ProjectTask[] }>(`/api/v1/tasks${query}`, { signal: filters.signal })).data
}

export async function getTask(id: number): Promise<ProjectTask> {
  return (await request<{ data: ProjectTask }>(`/api/v1/tasks/${id}`)).data
}

export async function getTodayView(date: string, projectId?: number): Promise<TodayView> {
  const params = new URLSearchParams({ date })
  if (projectId) params.set('project_id', String(projectId))
  return (await request<{ data: TodayView }>(`/api/v1/projects/today?${params}`)).data
}

export async function createTask(input: TaskInput): Promise<ProjectTask> {
  const saved = (await request<{ data: ProjectTask }>('/api/v1/tasks', { method: 'POST', body: JSON.stringify(input) })).data
  notifyTaskChange()
  return saved
}

export async function updateTask(id: number, patch: Partial<TaskInput> & { expected_updated_at?: string }): Promise<ProjectTask> {
  const saved = (await request<{ data: ProjectTask }>(`/api/v1/tasks/${id}`, { method: 'PATCH', body: JSON.stringify(patch) })).data
  notifyTaskChange()
  return saved
}

export async function moveTask(id: number, move: { status: TaskStatus; before_id?: number; after_id?: number; expected_updated_at: string }): Promise<ProjectTask> {
  const saved = (await request<{ data: ProjectTask }>(`/api/v1/tasks/${id}/move`, { method: 'PATCH', body: JSON.stringify(move) })).data
  notifyTaskChange()
  return saved
}

export async function deleteTask(id: number): Promise<void> {
  await request(`/api/v1/tasks/${id}`, { method: 'DELETE' })
  notifyTaskChange()
}

export async function archiveTasks(taskIDs: number[], archived = true): Promise<void> {
  await request('/api/v1/tasks/archive', { method: 'POST', body: JSON.stringify({ task_ids: taskIDs, archived }) })
  notifyTaskChange()
}
