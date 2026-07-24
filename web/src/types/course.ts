export type CourseStatus = 'initializing' | 'learning' | 'paused' | 'completed'

export interface Course {
  id: number
  name: string
  description: string
  goal: string
  status: CourseStatus
  progress: number
  current_unit: string
  last_studied_at: string | null
  created_at: string
  updated_at: string
}
