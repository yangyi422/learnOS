import { request } from './http'
import type { KnowledgeGraph, LessonRelations } from '@/types/knowledgeGraph'

interface KnowledgeGraphResponse {
  data: KnowledgeGraph
}

interface LessonRelationsResponse {
  data: LessonRelations
}

export async function getKnowledgeGraph(courseID: number, signal?: AbortSignal): Promise<KnowledgeGraph> {
  const response = await request<KnowledgeGraphResponse>(`/api/v1/courses/${courseID}/knowledge-graph`, { signal })
  return response.data
}

export async function getLessonRelations(courseID: number, lessonID: number, signal?: AbortSignal): Promise<LessonRelations> {
  const response = await request<LessonRelationsResponse>(`/api/v1/courses/${courseID}/lessons/${lessonID}/relations`, { signal })
  return response.data
}
