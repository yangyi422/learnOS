export type CognitiveLevel = 'unseen' | 'exposed' | 'recognize' | 'understand' | 'apply' | 'transfer'
export type CognitiveStatus = 'unknown' | 'developing' | 'stable' | 'needs_review'

export interface CognitiveStateSummary {
  lesson_id: number
  current_level: CognitiveLevel
  status: CognitiveStatus
  understanding_summary: string
  evidence_count: number
}

export interface CourseCognitiveStates {
  course_id: number
  states: CognitiveStateSummary[]
}

export interface CognitiveEvidence {
  id: number
  course_id: number
  lesson_id: number
  learning_turn_id: number
  evidence_index: number
  evidence_type: string
  cognitive_level: CognitiveLevel
  polarity: 'support' | 'contradict'
  description: string
  source: string
  created_at: string
}

export interface CognitiveStateEvent {
  id: number
  course_id: number
  lesson_id: number
  learning_turn_id: number
  from_level: CognitiveLevel
  to_level: CognitiveLevel
  from_status: CognitiveStatus
  to_status: CognitiveStatus
  understanding_summary: string
  reason: string
  created_at: string
}

export interface CognitiveStateDetail {
  lesson: {
    id: number
    title: string
  }
  state: {
    current_level: CognitiveLevel
    status: CognitiveStatus
    understanding_summary: string
    last_evaluated_at: string | null
  }
  evidence: CognitiveEvidence[]
  timeline: CognitiveStateEvent[]
}
