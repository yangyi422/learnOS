export interface ExplorationCourse {
  id: number
  name: string
}

export interface ExplorationLesson {
  id: number
  title: string
  course_id: number
}

export interface ExplorationDirection {
  id: number
  course_id: number
  context_course_id: number
  context_lesson_id: number | null
  source_domain_id: number
  source_course_id: number
  source_lesson_id: number
  target_course_id: number
  target_lesson_id: number
  direction_type: 'adjacent' | 'cross_domain' | 'unknown' | string
  title: string
  summary: string
  why_worth_exploring: string
  score: number
  reason_code: string
  reason_data: Record<string, unknown>
  status: 'active' | 'saved' | 'dismissed' | 'opened' | string
  generated_by: 'rule' | 'ai' | string
  provider: string
  model: string
  prompt_version: string
  source_domain: ExplorationCourse
  source_course: ExplorationCourse
  source_lesson: ExplorationLesson
  target_course: ExplorationCourse
  target_lesson: ExplorationLesson
  created_at: string
  updated_at: string
}

export interface ExplorationQuestion {
  id: number
  course_id: number
  source_direction_id: number | null
  source_lesson_id: number | null
  target_course_id: number
  target_lesson_id: number
  question: string
  context: string
  why_this_question: string
  question_type: string
  status: 'open' | 'exploring' | 'later' | 'resolved' | 'archived' | string
  priority: 'low' | 'normal' | 'high' | string
  origin: string
  source_course: ExplorationCourse
  source_lesson: ExplorationLesson | null
  target_course: ExplorationCourse
  target_lesson: ExplorationLesson
  created_at: string
  updated_at: string
}
