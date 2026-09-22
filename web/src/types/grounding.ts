export type GroundingStatus = 'ungrounded' | 'partially_grounded' | 'grounded' | 'conflicted' | string

export interface KnowledgeSource {
  id: number
  title: string
  authors: string
  organization: string
  source_type: string
  publisher: string
  publication_year: number | null
  url: string
  doi: string
  isbn: string
  language: string
  description: string
  access_status: string
  verification_status: string
  created_by: string
  created_at: string
}

export interface SourceEvidence {
  id: number
  source_id: number
  evidence_type: string
  locator: string
  quote: string
  summary: string
  language: string
  extraction_method: string
  verification_status: string
  created_at: string
}

export interface CredibilityAssessment {
  id: number
  source_id: number
  authority_score: number
  methodology_score: number
  directness_score: number
  recency_score: number
  independence_score: number
  overall_score: number
  authority_reason: string
  methodology_reason: string
  directness_reason: string
  recency_reason: string
  independence_reason: string
  assessment_method: string
  status: string
  reviewed_at: string | null
}

export interface GroundingLink {
  id: number
  evidence_id: number
  target_type: string
  target_id: number
  relation: string
  strength: string
  rationale: string
  status: string
  reviewed_at: string | null
}

export interface GroundingLinkView {
  link: GroundingLink
  evidence: SourceEvidence
  source: KnowledgeSource
  credibility: CredibilityAssessment | null
}

export interface SourceView {
  source: KnowledgeSource
  credibility: CredibilityAssessment[]
  evidence: SourceEvidence[]
}

export interface GroundingTargetView {
  target_type: string
  target_id: number
  grounding_status: GroundingStatus
  reviewed_source_count: number
  reviewed_evidence_count: number
  conflict_count: number
  links: GroundingLinkView[]
}

export interface GroundingCoverageLesson {
  blueprint_lesson: { id: number; title: string; importance: string; grounding_status: GroundingStatus; applied_lesson_id: number | null }
  grounding_status: GroundingStatus
  reviewed_source_count: number
  reviewed_evidence_count: number
  conflict_count: number
  applied_lesson: { id: number; title: string; grounding_status: GroundingStatus; reviewed_source_count: number } | null
}

export interface GroundingCoverage {
  blueprint: { id: number; name: string; version: string; grounding_status: GroundingStatus }
  metrics: { core_total: number; core_grounded: number; core_partial: number; core_ungrounded: number; core_conflicted: number; recommended_total: number; recommended_grounded: number; grounding_percent: number }
  units: Array<{ unit: { id: number; title: string }; lessons: GroundingCoverageLesson[] }>
}
