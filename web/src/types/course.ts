export type CourseStatus = 'initializing' | 'learning' | 'paused' | 'completed'
export type GenerationStatus = 'not_started' | 'in_progress' | 'ready'
export type LearningStatus = 'not_started' | 'in_progress' | 'paused' | 'completed'
export type MasteryStatus = 'unseen' | 'developing' | 'stable' | 'needs_review'

export interface Course {
  id: number
  name: string
  description: string
  goal: string
  status: CourseStatus
  /** @deprecated compatibility only; use the three explicit progress fields. */
  progress: number
  generation_status: GenerationStatus
  generation_progress: number
  learning_status: LearningStatus
  coverage_progress: number
  mastery_status: MasteryStatus
  mastery_progress: number
  current_unit: string
  current_unit_id?: number | null
  current_lesson_id?: number | null
  last_studied_at: string | null
  created_at: string
  updated_at: string
}
