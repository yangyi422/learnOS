export interface MisconceptionEvent {
  id: number
  event_type: 'observed' | 'resolved' | 'reopened' | 'user_confirmed' | 'user_corrected' | 'ignored'
  notes: string
  created_at: string
  question?: string
  user_answer?: string
  turn_kind?: string
}

export interface MisconceptionView {
  id: number
  lesson_id: number
  original_understanding: string
  correct_understanding: string
  boundary_notes: string
  status: 'active' | 'resolved'
  lesson_title: string
  pattern_keys: string[]
  occurrence_count: number
  review_status: 'ai_inferred' | 'user_confirmed' | 'user_corrected' | 'ignored'
  user_note: string
  stable_pattern_evidence: boolean
  events: MisconceptionEvent[]
}

export interface MisconceptionNetwork {
  patterns: Array<{
    key: string
    name: string
    active_count: number
    resolved_count: number
  }>
  misconceptions: MisconceptionView[]
  edges: Array<{ pattern_key: string; misconception_id: number }>
}
