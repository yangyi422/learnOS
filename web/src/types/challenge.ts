export interface AssessmentChallenge {
  id: number
  course_id: number
  lesson_id: number
  challenge_type: 'transfer' | 'misconception_recheck'
  target_misconception_id: number | null
  prompt: string
  scenario_context: string
  evaluation_criteria: string[]
  why_this_is_transfer: string
  source_concepts: string[]
  target_level: string
  status: string
  provider: string
  model: string
  prompt_version: string
  created_at: string
}

export interface ChallengeAnswerResult {
  challenge_id: number
  attempt_id: number
  learning_turn_id: number
  result: string
  demonstrated_level: string
  passed: boolean
  feedback: string
  explanation: string
  mastery_score_before: number
  mastery_score_after: number
  mastery_impact: string
  cognitive_evidence: Array<{
    evidence_type: string
    cognitive_level: string
    polarity: 'support' | 'contradict'
    description: string
  }>
  cognitive_state: {
    current_level: string
    status: string
  } | null
  misconception_validation: {
    target_misconception_id: number
    status: 'corrected' | 'persists' | 'unclear'
    evidence: string
  } | null
}
