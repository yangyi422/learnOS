import { request } from './http'
import type { MisconceptionNetwork, MisconceptionView } from '@/types/misconception'

interface NetworkResponse { data: MisconceptionNetwork }
interface LessonResponse { data: MisconceptionView[] }

export async function getMisconceptionNetwork(courseID: number): Promise<MisconceptionNetwork> {
  const response = await request<NetworkResponse>(`/api/v1/courses/${courseID}/misconception-network`)
  return response.data
}

export async function listLessonMisconceptions(courseID: number, lessonID: number): Promise<MisconceptionView[]> {
  const response = await request<LessonResponse>(`/api/v1/courses/${courseID}/lessons/${lessonID}/misconceptions`)
  return response.data
}
