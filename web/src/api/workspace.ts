import { request } from './http'
import type { InboxSummary } from './inbox'
import type { Project, TodayView, TaskPriority } from './projects'

export interface WorkspaceLearningItem {
  course_id: number; course_name: string; unit_title: string; lesson_id: number; lesson_title: string; core_question: string
  current_level: string; cognitive_status: string; last_learning_at: string | null; has_current_lesson: boolean; updated_at: string
}
export interface WorkspaceFocus {
  kind: 'empty' | 'task' | 'learning'; title: string; project_id?: number; project_title?: string; task_id?: number
  status?: string; priority?: TaskPriority; due_date?: string | null; course_id?: number; course_name?: string
  unit_title?: string; lesson_id?: number; current_level?: string; cognitive_status?: string
}
export interface WorkspaceHome {
  date: string; focus: WorkspaceFocus; today: TodayView | null; projects: Project[]; learning: WorkspaceLearningItem[]
  inbox: InboxSummary | null; errors: Record<string, string>
}
export async function getWorkspaceHome(date: string, signal?: AbortSignal): Promise<WorkspaceHome> {
  return (await request<{ data: WorkspaceHome }>(`/api/v1/workspace/home?date=${encodeURIComponent(date)}`, { signal })).data
}


export interface WorkspaceRecent {
  library: { id: number; collection_id: number; title: string; collection_name: string; display_mode: 'wiki' | 'list'; cover_url: string; updated_at: string }[]
  events: { id: number; title: string; occurred_on: string; primary_domain: string }[]
  goals: { id: number; goal_id: number; goal_title: string; content: string; occurred_on: string }[]
  asset_date: string
  errors: Record<string, string>
}
export async function getWorkspaceRecent(signal?: AbortSignal): Promise<WorkspaceRecent> {
  return (await request<{ data: WorkspaceRecent }>('/api/v1/workspace/recent', { signal })).data
}
