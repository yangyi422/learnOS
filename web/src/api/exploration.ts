import { request } from './http'
import type { ExplorationDirection, ExplorationQuestion } from '@/types/exploration'

interface RadarResponse { data: { directions: ExplorationDirection[] } }
interface DirectionResponse { data: ExplorationDirection }
interface QuestionResponse { data: ExplorationQuestion }
interface QuestionsResponse { data: ExplorationQuestion[] }
interface HistoryResponse { data: ExplorationDirection[] }

const radarCache = new Map<string, { expiresAt: number; directions: ExplorationDirection[] }>()
const RADAR_CACHE_TTL_MS = 60_000

function radarCacheKey(courseID: number, lessonID: number, limit: number): string {
  return `${courseID}:${lessonID}:${limit}`
}

function cloneDirections(directions: ExplorationDirection[]): ExplorationDirection[] {
  return directions.map((direction) => ({
    ...direction,
    source_domain: { ...direction.source_domain },
    source_course: { ...direction.source_course },
    source_lesson: { ...direction.source_lesson },
    target_course: { ...direction.target_course },
    target_lesson: { ...direction.target_lesson },
    reason_data: { ...direction.reason_data },
  }))
}

export function invalidateExplorationRadar(courseID?: number): void {
  if (courseID === undefined) {
    radarCache.clear()
    return
  }
  const prefix = `${courseID}:`
  for (const key of radarCache.keys()) {
    if (key.startsWith(prefix)) radarCache.delete(key)
  }
}

export async function getExplorationRadar(courseID: number, lessonID = 0, limit = 6, options: { signal?: AbortSignal; force?: boolean } = {}): Promise<ExplorationDirection[]> {
  const key = radarCacheKey(courseID, lessonID, limit)
  const cached = radarCache.get(key)
  if (!options.force && cached && cached.expiresAt > Date.now()) return cloneDirections(cached.directions)

  const response = await request<RadarResponse>(`/api/v1/exploration/radar?course_id=${courseID}&lesson_id=${lessonID}&limit=${limit}`, {
    cache: 'no-store',
    signal: options.signal,
  })
  const directions = cloneDirections(response.data.directions)
  radarCache.set(key, { expiresAt: Date.now() + RADAR_CACHE_TTL_MS, directions })
  return cloneDirections(directions)
}

export async function findUnfamiliarKnowledge(courseID: number): Promise<ExplorationDirection> {
  const response = await request<DirectionResponse>('/api/v1/exploration/unfamiliar', {
    method: 'POST',
    body: JSON.stringify({ course_id: courseID }),
  })
  return response.data
}

export async function updateDirection(directionID: number, status: 'save' | 'dismiss' | 'open', courseID: number): Promise<ExplorationDirection> {
  const response = await request<DirectionResponse>(`/api/v1/exploration/directions/${directionID}/${status}`, {
    method: 'POST',
    body: JSON.stringify({ course_id: courseID }),
  })
  invalidateExplorationRadar(courseID)
  return response.data
}

export async function addExplorationQuestion(courseID: number, directionID: number): Promise<ExplorationQuestion> {
  const response = await request<QuestionResponse>(`/api/v1/exploration/directions/${directionID}/questions`, {
    method: 'POST',
    body: JSON.stringify({ course_id: courseID }),
  })
  invalidateExplorationRadar(courseID)
  return response.data
}

export async function undoExplorationQuestion(courseID: number, directionID: number): Promise<ExplorationDirection> {
  const response = await request<DirectionResponse>(`/api/v1/exploration/directions/${directionID}/questions/undo`, {
    method: 'POST', body: JSON.stringify({ course_id: courseID }),
  })
  invalidateExplorationRadar(courseID)
  return response.data
}

export async function listExplorationQuestions(courseID: number, filters: { status?: string; targetCourseID?: number; priority?: string } = {}, signal?: AbortSignal): Promise<ExplorationQuestion[]> {
  const params = new URLSearchParams({ course_id: String(courseID) })
  if (filters.status) params.set('status', filters.status)
  if (filters.targetCourseID) params.set('target_course_id', String(filters.targetCourseID))
  if (filters.priority) params.set('priority', filters.priority)
  const response = await request<QuestionsResponse>(`/api/v1/exploration/questions?${params}`, { signal })
  return response.data
}

export async function updateQuestion(questionID: number, status: 'start' | 'later' | 'resolve' | 'reopen' | 'archive', courseID: number): Promise<ExplorationQuestion> {
  const response = await request<QuestionResponse>(`/api/v1/exploration/questions/${questionID}/${status}`, {
    method: 'POST',
    body: JSON.stringify({ course_id: courseID }),
  })
  return response.data
}

export async function updateQuestionPriority(questionID: number, priority: 'low' | 'normal' | 'high', courseID: number): Promise<ExplorationQuestion> {
  const response = await request<QuestionResponse>(`/api/v1/exploration/questions/${questionID}/priority`, {
    method: 'POST', body: JSON.stringify({ course_id: courseID, priority }),
  })
  return response.data
}

export async function listExplorationHistory(courseID: number, signal?: AbortSignal): Promise<ExplorationDirection[]> {
  const response = await request<HistoryResponse>(`/api/v1/exploration/history?course_id=${courseID}`, { signal })
  return response.data
}
