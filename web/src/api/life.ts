import { request } from './http'
export const lifeDomains = [
  ['finance', '财务与资产'], ['health', '身心健康'], ['relationships', '人际与关系'], ['work', '工作与创造'],
  ['learning', '学习与认知'], ['experiences', '兴趣与体验'], ['home', '居住与生活环境'], ['rhythm', '时间与生活节奏'],
] as const
export const goalStatuses = [['considering', '考虑中'], ['active', '进行中'], ['paused', '暂停'], ['achieved', '已实现'], ['ended', '已结束']] as const
export const domainLabel = (key: string) => lifeDomains.find(d => d[0] === key)?.[1] ?? ''
export const goalStatusLabel = (key: string) => goalStatuses.find(d => d[0] === key)?.[1] ?? key
export interface LifeEventInput {
  creation_key?: string; title: string; occurred_on: string; description: string; primary_domain: string; secondary_domain: string
  milestone: boolean; goal_id: number | null; external_url: string; link_name: string; source_type?: string; source_id?: number | null
}
export interface LifeEvent extends LifeEventInput {
  id: number; source_type: string; source_id: number | null; source_title: string; source_available: boolean; source_status: string; created_at: string; updated_at: string
}
export interface LifeGoalInput {
  record_change?: boolean; creation_key?: string; title: string; why: string; current_note: string; status: string; domain: string; project_id: number | null; external_url: string; link_name: string; status_date: string; status_reason: string
}
export interface LifeGoal extends LifeGoalInput { id: number; created_at: string; updated_at: string; project_available: boolean; project_title: string }
export interface LifeEntryInput { creation_key: string; occurred_on: string; content: string; reason: string }
export interface LifeEntry extends LifeEntryInput { id: number; goal_id: number; from_status: string; to_status: string; created_at: string }
export interface GoalDetail { goal: LifeGoal; entries: LifeEntry[]; events: LifeEvent[] }
export interface LifeSource { type: 'project' | 'course'; id: number; title: string; status: string; eligible: boolean; can_graduate: boolean; event_id: number | null }
const api = '/api/v1'
async function data<T>(path: string, method?: string, input?: unknown): Promise<T> { return (await request<{ data: T }>(`${api}${path}`, method ? { method, body: input === undefined ? undefined : JSON.stringify(input) } : undefined)).data }
export const listLifeEvents = (domain = '', milestones = false) => data<LifeEvent[]>(`/life/events?${new URLSearchParams({ domain, milestones: String(milestones) })}`)
export const getLifeEvent = (id: number) => data<LifeEvent>(`/life/events/${id}`)
export const saveLifeEvent = (input: LifeEventInput, id?: number, inboxID?: number) => data<LifeEvent>(inboxID ? `/inbox/${inboxID}/convert-to-life-event` : `/life/events${id ? `/${id}` : ''}`, id ? 'PUT' : 'POST', input)
export const deleteLifeEvent = (id: number) => data(`/life/events/${id}`, 'DELETE')
export const listLifeGoals = () => data<LifeGoal[]>('/life/goals')
export const getLifeGoal = (id: number) => data<GoalDetail>(`/life/goals/${id}`)
export const saveLifeGoal = (input: LifeGoalInput, id?: number) => data<LifeGoal>(`/life/goals${id ? `/${id}` : ''}`, id ? 'PUT' : 'POST', input)
export const addLifeEntry = (id: number, input: LifeEntryInput) => data<LifeEntry>(`/life/goals/${id}/entries`, 'POST', input)
export const getLifeSource = (type: string, id: number) => data<LifeSource>(`/life/sources/${type}/${id}`)
export const graduateCourse = (id: number) => data(`/courses/${id}/graduate`, 'POST')
export function lifeRequestKey() { return globalThis.crypto?.randomUUID?.() ?? `life-${Date.now()}-${Math.random().toString(36).slice(2)}` }
