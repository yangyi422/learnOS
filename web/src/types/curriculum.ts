export interface CurriculumBlueprint {
  id: number
  course_id: number | null
  name: string
  domain: string
  description: string
  learning_goal: string
  audience: string
  target_depth: string
  version: string
  status: string
  created_by: string
  grounding_status: string
}

export interface CurriculumBlueprintUnit {
  id: number
  blueprint_id: number
  key: string
  title: string
  description: string
  sort_order: number
  importance: string
  expansion_status: string
}

export interface CurriculumBlueprintLesson {
  id: number
  blueprint_id: number
  blueprint_unit_id: number
  key: string
  title: string
  summary: string
  importance: string
  content_role: string
  depth_level: number
  assessment_target_level: string
  sort_order: number
  applied_lesson_id: number | null
}

export interface CurriculumBlueprintUnitView {
  unit: CurriculumBlueprintUnit
  lessons: CurriculumBlueprintLesson[]
}

export interface CurriculumView {
  blueprint: CurriculumBlueprint
  units: CurriculumBlueprintUnitView[]
  relations: unknown[]
}

export interface CurriculumAppliedLesson {
  id: number
  title: string
  unit_id: number
  unit_title: string
}

export interface CurriculumCoverageLesson {
  blueprint_lesson: CurriculumBlueprintLesson
  state: 'covered' | 'missing' | string
  applied_lesson: CurriculumAppliedLesson | null
}

export interface CurriculumCoverageUnit {
  unit: CurriculumBlueprintUnit
  lessons: CurriculumCoverageLesson[]
  core_total: number
  core_covered: number
  recommended_total: number
  recommended_covered: number
  optional_total: number
  optional_covered: number
}

export interface CurriculumCoverageMetrics {
  core_total: number
  core_covered: number
  core_missing: number
  recommended_total: number
  recommended_covered: number
  optional_total: number
  optional_covered: number
  core_coverage_percent: number
  overall_coverage_percent: number
}

export interface CurriculumCoverageView {
  blueprint: CurriculumBlueprint
  units: CurriculumCoverageUnit[]
  metrics: CurriculumCoverageMetrics
  missing_core: CurriculumCoverageLesson[]
}

export interface CurriculumChangeSet {
  new_units: Array<{ temp_key: string; title: string; description: string; sort_order: number }>
  new_lessons: Array<{ temp_key: string; blueprint_lesson_key: string; title: string; summary: string; unit_temp_key: string }>
  new_relations: Array<{ from_key: string; to_key: string; relation_type: string }>
  blueprint_mappings: Array<{ blueprint_lesson_key: string; lesson_temp_key?: string; applied_lesson_id?: number }>
}

export interface CurriculumDraft {
  id: number
  course_id: number
  blueprint_id: number
  title: string
  summary: string
  status: 'draft' | 'applied' | 'rejected' | string
  generated_by: string
  provider: string
  model: string
  prompt_version: string
  change_set: string
  created_at: string
  applied_at: string | null
}

export interface CurriculumDraftView {
  draft: CurriculumDraft
  change_set: CurriculumChangeSet
}
