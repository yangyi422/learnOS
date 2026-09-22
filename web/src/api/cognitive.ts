import { request } from './http'
import type { CognitiveStateDetail, CourseCognitiveStates } from '@/types/cognitive'

interface CourseCognitiveStatesResponse {
  data: CourseCognitiveStates
}

interface CognitiveStateDetailResponse {
  data: CognitiveStateDetail
}

export async function getCourseCognitiveStates(courseID: number, signal?: AbortSignal): Promise<CourseCognitiveStates> {
  const response = await request<CourseCognitiveStatesResponse>(`/api/v1/courses/${courseID}/cognitive-states`, { signal })
  return response.data
}

export async function getLessonCognitiveState(courseID: number, lessonID: number, signal?: AbortSignal): Promise<CognitiveStateDetail> {
  const response = await request<CognitiveStateDetailResponse>(`/api/v1/courses/${courseID}/lessons/${lessonID}/cognitive-state`, { signal })
  return response.data
}
