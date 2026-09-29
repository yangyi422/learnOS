import { request } from './http'
import type { ProjectTask, TaskPriority, TaskStatus } from './projects'

export type InboxStatus = 'inbox' | 'processed' | 'archived'
export interface InboxItem {
  id: number; user_id: number; content: string; status: InboxStatus; source_type: 'manual' | 'url' | 'system'; source_url: string
  processed_to_type: string; processed_to_id: number | null; created_at: string; updated_at: string; processed_at: string | null; archived_at: string | null
}
export interface InboxView { items: InboxItem[]; counts: Record<InboxStatus, number> }
export interface InboxSummary { count: number; items: InboxItem[] }

export async function listInbox(status?: InboxStatus): Promise<InboxView> {
  return (await request<{ data: InboxView }>(`/api/v1/inbox${status ? `?status=${status}` : ''}`)).data
}
export async function createInboxItem(content: string): Promise<InboxItem> {
  return (await request<{ data: InboxItem }>('/api/v1/inbox', { method: 'POST', body: JSON.stringify({ content }) })).data
}
export async function updateInboxItem(id: number, content: string): Promise<InboxItem> {
  return (await request<{ data: InboxItem }>(`/api/v1/inbox/${id}`, { method: 'PATCH', body: JSON.stringify({ content }) })).data
}
export async function archiveInboxItem(id: number): Promise<void> {
  await request(`/api/v1/inbox/${id}/archive`, { method: 'POST' })
}
export async function deleteInboxItem(id: number): Promise<void> {
  await request(`/api/v1/inbox/${id}`, { method: 'DELETE' })
}
export async function convertInboxItem(id: number, input: { project_id: number; title: string; description: string; status: TaskStatus; priority: TaskPriority; due_date: string | null }): Promise<{ item: InboxItem; task: ProjectTask }> {
  return (await request<{ data: { item: InboxItem; task: ProjectTask } }>(`/api/v1/inbox/${id}/convert-to-task`, { method: 'POST', body: JSON.stringify(input) })).data
}
