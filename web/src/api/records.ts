import { request } from './http'
import type { InboxItem } from './inbox'
export interface LightweightRecord {
  id: number; user_id: number; content: string; project_id: number | null; source_inbox_id: number | null
  external_url: string; link_name: string; created_at: string; updated_at: string; archived_at: string | null
}
export interface RecordInput { creation_key?: string; content: string; project_id: number | null; external_url: string; link_name: string }
export async function listRecords(projectId?: number, archived = false): Promise<LightweightRecord[]> {
  const query = new URLSearchParams({ status: archived ? 'archived' : 'active' })
  if (projectId) query.set('project_id', String(projectId))
  return (await request<{ data: LightweightRecord[] }>(`/api/v1/records?${query}`)).data
}
export async function getRecord(id: number): Promise<LightweightRecord> { return (await request<{ data: LightweightRecord }>(`/api/v1/records/${id}`)).data }
export async function saveRecord(input: RecordInput, id?: number): Promise<LightweightRecord> { return (await request<{ data: LightweightRecord }>(`/api/v1/records${id ? `/${id}` : ''}`, { method: id ? 'PUT' : 'POST', body: JSON.stringify(input) })).data }
export async function convertToRecord(id: number, input: RecordInput): Promise<{ item: InboxItem; record: LightweightRecord }> { return (await request<{ data: { item: InboxItem; record: LightweightRecord } }>(`/api/v1/inbox/${id}/convert-to-record`, { method: 'POST', body: JSON.stringify(input) })).data }
export async function archiveRecord(id: number, archived: boolean): Promise<void> { await request(`/api/v1/records/${id}/archive`, { method: 'POST', body: JSON.stringify({ archived }) }) }
export async function deleteRecord(id: number): Promise<void> { await request(`/api/v1/records/${id}`, { method: 'DELETE' }) }
