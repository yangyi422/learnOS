import { request } from './http'
import type { MisconceptionNetwork, MisconceptionView } from '@/types/misconception'

interface NetworkResponse { data: MisconceptionNetwork }
interface LessonResponse { data: MisconceptionView[] }
interface ReviewResponse { data: MisconceptionView }

export async function getMisconceptionNetwork(courseID: number): Promise<MisconceptionNetwork> {
  const response = await request<NetworkResponse>(`/api/v1/courses/${courseID}/misconception-network`)
  return response.data
}

export async function reviewMisconception(courseID: number, misconceptionID: number, action: 'confirm' | 'correct' | 'ignore', note = ''): Promise<MisconceptionView> {
  const response = await request<ReviewResponse>(`/api/v1/courses/${courseID}/misconceptions/${misconceptionID}/review`, {
    method: 'POST', body: JSON.stringify({ action, note }),
  })
  return response.data
}

export async function listLessonMisconceptions(courseID: number, lessonID: number, signal?: AbortSignal): Promise<MisconceptionView[]> {
  const response = await request<LessonResponse>(`/api/v1/courses/${courseID}/lessons/${lessonID}/misconceptions`, { signal })
  return response.data
}
