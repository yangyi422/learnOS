import { request } from './http'
import type { CognitiveEvidence } from '@/types/cognitive'
import type { KnowledgeNodeType } from '@/utils/knowledgeNodeType'

export interface CurrentCourse {
  id: number
  name: string
}

export interface CurrentUnit {
  id: number
  title: string
  objective: string
}

export interface CurrentLesson {
  id: number
  title: string
  core_question: string
  status: 'pending' | 'learning' | 'completed'
  content_role: KnowledgeNodeType
  depth_level: number
}

export interface CurrentLessonData {
  course: CurrentCourse
  unit: CurrentUnit
  lesson: CurrentLesson
}

export interface LearningTurn {
  id: number
  lesson_id: number
  turn_kind: 'lesson_answer' | 'transfer_challenge' | 'misconception_recheck'
  question: string
  user_answer: string
  result: string
  feedback: string
  explanation: string
  correct_parts: string[]
  missing_parts: string[]
  misconceptions: EvaluationMisconception[]
  boundary_conditions: string[]
  mastery_evidence: string[]
  evidence_used: string[]
  confidence: number
  uncertainty: string
  recommended_next_action: string
  transfer_challenge_eligible: boolean
  mastery_score: number
  mastery_score_before: number
  mastery_score_after: number
  needs_review: boolean
  evaluation_source: string
  provider: string
  model: string
  prompt_version: string
  demonstrated_level: string
  user_understanding_summary: string
  cognitive_evidence: CognitiveEvidence[]
  state_change: CognitiveStateChange | null
  created_at: string
}

export interface CognitiveStateChange {
  from_level: string
  to_level: string
  from_status: string
  to_status: string
  reason: string
}

export interface EvaluationMisconception {
  original_understanding: string
  correct_understanding: string
  boundary_notes: string
}

export interface AnswerResult {
  turn_id: number
  result: string
  feedback: string
  explanation: string
  correct_parts: string[]
  missing_parts: string[]
  misconceptions: EvaluationMisconception[]
  boundary_conditions: string[]
  mastery_evidence: string[]
  evidence_used: string[]
  confidence: number
  uncertainty: string
  recommended_next_action: string
  transfer_challenge_eligible: boolean
  mastery_score: number
  mastery_score_before: number
  mastery_score_after: number
  needs_review: boolean
  evaluation_source: string
  provider: string
  model: string
  prompt_version: string
  demonstrated_level: string
  user_understanding_summary: string
  cognitive_evidence: CognitiveEvidence[]
  cognitive_state: {
    current_level: string
    status: string
  } | null
}

interface CurrentLessonResponse {
  data: CurrentLessonData
}

export interface NextLesson {
  id: number
  unit_id: number
  unit_title: string
  title: string
  core_question: string
  status: string
  is_core: boolean
  content_role: string
  depth_level: number
  prerequisites: string[]
  prerequisites_satisfied: boolean
  unseen: boolean
}

export interface NextLessonView {
  recommended: NextLesson | null
  alternatives: NextLesson[]
  reason: string
  needs_expansion: boolean
  needs_generation: boolean
  recommended_unit: {
    id: number
    blueprint_id: number
    key: string
    title: string
    description: string
    sort_order: number
    importance: string
    expansion_status: string
  } | null
}

interface NextLessonResponse { data: NextLessonView }

interface LearningTurnsResponse {
  data: LearningTurnPayload[]
}

interface AnswerResponse {
  data: AnswerResultPayload
}

interface LearningTurnPayload {
  id: number
  lesson_id?: number
  turn_kind?: LearningTurn['turn_kind']
  question: string
  user_answer: string
  result: string
  feedback: string
  explanation?: string
  correct_parts?: string[]
  missing_parts?: string[]
  misconceptions?: EvaluationMisconception[]
  boundary_conditions?: string[]
  mastery_evidence?: string[]
  evidence_used?: string[]
  confidence?: number
  uncertainty?: string
  recommended_next_action?: string
  transfer_challenge_eligible?: boolean
  mastery_score?: number
  mastery_score_before?: number
  mastery_score_after?: number
  needs_review?: boolean
  evaluation_source?: string
  provider?: string
  model?: string
  prompt_version?: string
  demonstrated_level?: string
  user_understanding_summary?: string
  cognitive_evidence?: CognitiveEvidence[]
  state_change?: CognitiveStateChange | null
  cognitive_state?: {
    current_level: string
    status: string
  } | null
  created_at: string
}

type AnswerResultPayload = Omit<LearningTurnPayload, 'id' | 'question' | 'user_answer' | 'created_at'> & {
  turn_id: number
}

export async function getCurrentLesson(courseID: number, signal?: AbortSignal): Promise<CurrentLessonData> {
  const response = await request<CurrentLessonResponse>(`/api/v1/courses/${courseID}/current-lesson`, { signal })
  return response.data
}

export async function getLessonForLearning(courseID: number, lessonID: number, signal?: AbortSignal): Promise<CurrentLessonData> {
  const response = await request<CurrentLessonResponse>(`/api/v1/courses/${courseID}/lessons/${lessonID}/learning`, { signal })
  return response.data
}

export async function setCurrentLesson(courseID: number, lessonID: number): Promise<CurrentLessonData> {
  const response = await request<CurrentLessonResponse>(`/api/v1/courses/${courseID}/current-lesson`, {
    method: 'POST',
    body: JSON.stringify({ lesson_id: lessonID }),
  })
  return response.data
}

export async function getNextLesson(courseID: number, signal?: AbortSignal): Promise<NextLessonView> {
  const response = await request<NextLessonResponse>(`/api/v1/courses/${courseID}/next-lesson`, { signal })
  return response.data
}

export async function submitAnswer(courseID: number, lessonID: number, answer: string, idempotencyKey: string): Promise<AnswerResult> {
  const response = await request<AnswerResponse>(`/api/v1/courses/${courseID}/answers`, {
    method: 'POST',
    body: JSON.stringify({ lesson_id: lessonID, answer, idempotency_key: idempotencyKey }),
  })
  return normalizeAnswerResult(response.data)
}

export async function submitLessonAnswer(courseID: number, lessonID: number, answer: string, idempotencyKey: string): Promise<AnswerResult> {
  const response = await request<AnswerResponse>(`/api/v1/courses/${courseID}/lessons/${lessonID}/answers`, {
    method: 'POST',
    body: JSON.stringify({ answer, idempotency_key: idempotencyKey }),
  })
  return normalizeAnswerResult(response.data)
}

export async function listLearningTurns(courseID: number, limit = 10, signal?: AbortSignal): Promise<LearningTurn[]> {
  const response = await request<LearningTurnsResponse>(`/api/v1/courses/${courseID}/learning-turns?limit=${limit}`, { signal })
  return response.data.map((turn) => ({
    id: turn.id,
    lesson_id: turn.lesson_id ?? 0,
    turn_kind: turn.turn_kind ?? 'lesson_answer',
    question: turn.question,
    user_answer: turn.user_answer,
    result: turn.result,
    feedback: turn.feedback,
    explanation: turn.explanation ?? '',
    correct_parts: turn.correct_parts ?? [],
    missing_parts: turn.missing_parts ?? [],
    misconceptions: turn.misconceptions ?? [],
    boundary_conditions: turn.boundary_conditions ?? [],
    mastery_evidence: turn.mastery_evidence ?? [],
    evidence_used: turn.evidence_used ?? [],
    confidence: turn.confidence ?? 0,
    uncertainty: turn.uncertainty ?? '',
    recommended_next_action: turn.recommended_next_action ?? '',
    transfer_challenge_eligible: turn.transfer_challenge_eligible ?? false,
    mastery_score: turn.mastery_score ?? 0,
    mastery_score_before: turn.mastery_score_before ?? 0,
    mastery_score_after: turn.mastery_score_after ?? turn.mastery_score ?? 0,
    needs_review: turn.needs_review ?? false,
    evaluation_source: turn.evaluation_source ?? 'mock',
    provider: turn.provider ?? '',
    model: turn.model ?? '',
    prompt_version: turn.prompt_version ?? '',
    demonstrated_level: turn.demonstrated_level ?? '',
    user_understanding_summary: turn.user_understanding_summary ?? '',
    cognitive_evidence: turn.cognitive_evidence ?? [],
    state_change: turn.state_change ?? null,
    created_at: turn.created_at,
  }))
}

function normalizeAnswerResult(result: AnswerResultPayload): AnswerResult {
  return {
    turn_id: result.turn_id,
    result: result.result,
    feedback: result.feedback,
    explanation: result.explanation ?? '',
    correct_parts: result.correct_parts ?? [],
    missing_parts: result.missing_parts ?? [],
    misconceptions: result.misconceptions ?? [],
    boundary_conditions: result.boundary_conditions ?? [],
    mastery_evidence: result.mastery_evidence ?? [],
    evidence_used: result.evidence_used ?? [],
    confidence: result.confidence ?? 0,
    uncertainty: result.uncertainty ?? '',
    recommended_next_action: result.recommended_next_action ?? '',
    transfer_challenge_eligible: result.transfer_challenge_eligible ?? false,
    mastery_score: result.mastery_score ?? 0,
    mastery_score_before: result.mastery_score_before ?? 0,
    mastery_score_after: result.mastery_score_after ?? result.mastery_score ?? 0,
    needs_review: result.needs_review ?? false,
    evaluation_source: result.evaluation_source ?? 'mock',
    provider: result.provider ?? '',
    model: result.model ?? '',
    prompt_version: result.prompt_version ?? '',
    demonstrated_level: result.demonstrated_level ?? '',
    user_understanding_summary: result.user_understanding_summary ?? '',
    cognitive_evidence: result.cognitive_evidence ?? [],
    cognitive_state: result.cognitive_state ?? null,
  }
}
