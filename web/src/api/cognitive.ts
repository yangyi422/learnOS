import { request } from './http'
import type { CognitiveStateDetail, CourseCognitiveStates } from '@/types/cognitive'

interface CourseCognitiveStatesResponse {
  data: CourseCognitiveStates
}

interface CognitiveStateDetailResponse {
  data: CognitiveStateDetail
}

export async function getCourseCognitiveStates(courseID: number): Promise<CourseCognitiveStates> {
  const response = await request<CourseCognitiveStatesResponse>(`/api/v1/courses/${courseID}/cognitive-states`)
  return response.data
}

export async function getLessonCognitiveState(courseID: number, lessonID: number): Promise<CognitiveStateDetail> {
  const response = await request<CognitiveStateDetailResponse>(`/api/v1/courses/${courseID}/lessons/${lessonID}/cognitive-state`)
  return response.data
}
