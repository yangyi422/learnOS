import { request } from './http'
import type { CredibilityAssessment, GroundingCoverage, GroundingLinkView, GroundingTargetView, KnowledgeSource, SourceEvidence, SourceView } from '@/types/grounding'

interface Data<T> { data: T }

export async function listSources(query = '') { return (await request<Data<KnowledgeSource[]>>(`/api/v1/sources?q=${encodeURIComponent(query)}`)).data }
export async function createSource(input: Partial<KnowledgeSource>) { return (await request<Data<KnowledgeSource>>('/api/v1/sources', { method: 'POST', body: JSON.stringify(input) })).data }
export async function getSource(id: number) { return (await request<Data<SourceView>>(`/api/v1/sources/${id}`)).data }
export async function listSourceLinks(id: number) { return (await request<Data<GroundingLinkView[]>>(`/api/v1/sources/${id}/grounding-links`)).data }
export async function addEvidence(sourceID: number, input: Partial<SourceEvidence>) { return (await request<Data<SourceEvidence>>(`/api/v1/sources/${sourceID}/evidence`, { method: 'POST', body: JSON.stringify(input) })).data }
export async function createCredibility(sourceID: number, input: Partial<CredibilityAssessment>) { return (await request<Data<CredibilityAssessment>>(`/api/v1/sources/${sourceID}/credibility`, { method: 'POST', body: JSON.stringify(input) })).data }
export async function reviewCredibility(sourceID: number, assessmentID: number) { return (await request<Data<CredibilityAssessment>>(`/api/v1/sources/${sourceID}/credibility/${assessmentID}/review`, { method: 'POST' })).data }
export async function createGroundingLink(input: { evidence_id: number; target_type: string; target_id: number; relation: string; strength: string; rationale: string }) { return (await request<Data<unknown>>('/api/v1/grounding/links', { method: 'POST', body: JSON.stringify(input) })).data }
export async function reviewGroundingLink(id: number) { return (await request<Data<GroundingTargetView>>(`/api/v1/grounding/links/${id}/review`, { method: 'POST' })).data }
export async function getGroundingTarget(type: string, id: number) { return (await request<Data<GroundingTargetView>>(`/api/v1/grounding/targets/${type}/${id}`)).data }
export async function getGroundingCoverage(courseID: number) { return (await request<Data<GroundingCoverage>>(`/api/v1/courses/${courseID}/grounding/coverage`)).data }
