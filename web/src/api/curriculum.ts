import { request } from './http'
import type { CurriculumCoverageView, CurriculumDraftView, CurriculumView } from '@/types/curriculum'

interface CurriculumResponse { data: CurriculumView }
interface CoverageResponse { data: CurriculumCoverageView }
interface DraftResponse { data: CurriculumDraftView }
interface DraftsResponse { data: CurriculumDraftView[] }

export async function getCurriculum(courseID: number): Promise<CurriculumView> {
  const response = await request<CurriculumResponse>(`/api/v1/courses/${courseID}/curriculum`)
  return response.data
}

export async function getCurriculumCoverage(courseID: number): Promise<CurriculumCoverageView> {
  const response = await request<CoverageResponse>(`/api/v1/courses/${courseID}/curriculum/coverage`)
  return response.data
}

export async function createCurriculumDraft(courseID: number, scope = 'missing_core', limit = 5, blueprintUnitID?: number): Promise<CurriculumDraftView> {
  const response = await request<DraftResponse>(`/api/v1/courses/${courseID}/curriculum/drafts`, {
    method: 'POST',
    body: JSON.stringify({ scope, limit, ...(blueprintUnitID ? { blueprint_unit_id: blueprintUnitID } : {}) }),
  })
  return response.data
}

export async function expandBlueprintUnit(courseID: number, blueprintUnitID: number): Promise<CurriculumView> {
  const response = await request<CurriculumResponse>(`/api/v1/courses/${courseID}/curriculum/units/${blueprintUnitID}/expand`, { method: 'POST' })
  return response.data
}

export async function listCurriculumDrafts(courseID: number): Promise<CurriculumDraftView[]> {
  const response = await request<DraftsResponse>(`/api/v1/courses/${courseID}/curriculum/drafts`)
  return response.data
}

export async function applyCurriculumDraft(courseID: number, draftID: number): Promise<CurriculumDraftView> {
  const response = await request<DraftResponse>(`/api/v1/courses/${courseID}/curriculum/drafts/${draftID}/apply`, { method: 'POST' })
  return response.data
}

export async function rejectCurriculumDraft(courseID: number, draftID: number): Promise<CurriculumDraftView> {
  const response = await request<DraftResponse>(`/api/v1/courses/${courseID}/curriculum/drafts/${draftID}/reject`, { method: 'POST' })
  return response.data
}
