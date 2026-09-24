import { request } from './http'

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
  due_date: string | null
  completed_at: string | null
  created_at: string
  updated_at: string
}

export interface Project {
  id: number
  title: string
  description: string
  status: ProjectStatus
  icon: string
  accent: string
  archived_at: string | null
  tasks: ProjectTask[]
  open_task_count: number
  doing_task_count: number
  done_task_count: number
  created_at: string
  updated_at: string
}

export interface ProjectInput {
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
  due_date?: string | null
}

export async function listProjects(): Promise<Project[]> {
  return (await request<{ data: Project[] }>('/api/v1/projects')).data
}

export async function createProject(input: ProjectInput): Promise<Project> {
  return (await request<{ data: Project }>('/api/v1/projects', { method: 'POST', body: JSON.stringify(input) })).data
}

export async function updateProject(id: number, patch: Partial<ProjectInput> & { status?: ProjectStatus }): Promise<void> {
  await request(`/api/v1/projects/${id}`, { method: 'PATCH', body: JSON.stringify(patch) })
}

export async function listTasks(filters: { projectId?: number; status?: TaskStatus; due?: string; limit?: number; offset?: number } = {}): Promise<ProjectTask[]> {
  const params = new URLSearchParams()
  if (filters.projectId) params.set('project_id', String(filters.projectId))
  if (filters.status) params.set('status', filters.status)
  if (filters.due) params.set('due', filters.due)
  if (filters.limit) params.set('limit', String(filters.limit))
  if (filters.offset) params.set('offset', String(filters.offset))
  const query = params.size ? `?${params}` : ''
  return (await request<{ data: ProjectTask[] }>(`/api/v1/tasks${query}`)).data
}

export async function createTask(input: TaskInput): Promise<ProjectTask> {
  return (await request<{ data: ProjectTask }>('/api/v1/tasks', { method: 'POST', body: JSON.stringify(input) })).data
}

export async function updateTask(id: number, patch: Partial<TaskInput>): Promise<ProjectTask> {
  return (await request<{ data: ProjectTask }>(`/api/v1/tasks/${id}`, { method: 'PATCH', body: JSON.stringify(patch) })).data
}

export async function moveTask(id: number, move: { status: TaskStatus; before_id?: number; after_id?: number; expected_updated_at: string }): Promise<ProjectTask> {
  return (await request<{ data: ProjectTask }>(`/api/v1/tasks/${id}/move`, { method: 'PATCH', body: JSON.stringify(move) })).data
}

export async function deleteTask(id: number): Promise<void> {
  await request(`/api/v1/tasks/${id}`, { method: 'DELETE' })
}
