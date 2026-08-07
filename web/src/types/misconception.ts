export interface MisconceptionEvent {
  id: number
  event_type: 'observed' | 'resolved' | 'reopened'
  notes: string
  created_at: string
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
