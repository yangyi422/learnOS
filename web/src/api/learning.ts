import { request } from './http'
import type { CognitiveEvidence } from '@/types/cognitive'

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
}

export interface CurrentLessonData {
  course: CurrentCourse
  unit: CurrentUnit
  lesson: CurrentLesson
}

export interface LearningTurn {
  id: number
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
  mastery_score: number
  needs_review: boolean
  evaluation_source: string
  provider: string
  model: string
  prompt_version: string
  demonstrated_level: string
  user_understanding_summary: string
  cognitive_evidence: CognitiveEvidence[]
  created_at: string
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
  mastery_score: number
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

interface LearningTurnsResponse {
  data: LearningTurnPayload[]
}

interface AnswerResponse {
  data: AnswerResultPayload
}

interface LearningTurnPayload {
  id: number
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
  mastery_score?: number
  needs_review?: boolean
  evaluation_source?: string
  provider?: string
  model?: string
  prompt_version?: string
  demonstrated_level?: string
  user_understanding_summary?: string
  cognitive_evidence?: CognitiveEvidence[]
  cognitive_state?: {
    current_level: string
    status: string
  } | null
  created_at: string
}

type AnswerResultPayload = Omit<LearningTurnPayload, 'id' | 'question' | 'user_answer' | 'created_at'> & {
  turn_id: number
}

export async function getCurrentLesson(courseID: number): Promise<CurrentLessonData> {
  const response = await request<CurrentLessonResponse>(`/api/v1/courses/${courseID}/current-lesson`)
  return response.data
}

export async function submitAnswer(courseID: number, lessonID: number, answer: string): Promise<AnswerResult> {
  const response = await request<AnswerResponse>(`/api/v1/courses/${courseID}/answers`, {
    method: 'POST',
    body: JSON.stringify({ lesson_id: lessonID, answer }),
  })
  return normalizeAnswerResult(response.data)
}

export async function listLearningTurns(courseID: number, limit = 10): Promise<LearningTurn[]> {
  const response = await request<LearningTurnsResponse>(`/api/v1/courses/${courseID}/learning-turns?limit=${limit}`)
  return response.data.map((turn) => ({
    id: turn.id,
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
    mastery_score: turn.mastery_score ?? 0,
    needs_review: turn.needs_review ?? false,
    evaluation_source: turn.evaluation_source ?? 'mock',
    provider: turn.provider ?? '',
    model: turn.model ?? '',
    prompt_version: turn.prompt_version ?? '',
    demonstrated_level: turn.demonstrated_level ?? '',
    user_understanding_summary: turn.user_understanding_summary ?? '',
    cognitive_evidence: turn.cognitive_evidence ?? [],
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
    mastery_score: result.mastery_score ?? 0,
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
