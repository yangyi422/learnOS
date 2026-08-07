import { request } from './http'
import type { AssessmentChallenge, ChallengeAnswerResult } from '@/types/challenge'

interface ChallengeResponse { data: AssessmentChallenge }
interface ChallengeAnswerResponse { data: ChallengeAnswerResult }

export async function createChallenge(courseID: number, lessonID: number, challengeType: AssessmentChallenge['challenge_type'], misconceptionID?: number): Promise<AssessmentChallenge> {
  const response = await request<ChallengeResponse>(`/api/v1/courses/${courseID}/lessons/${lessonID}/challenges`, {
    method: 'POST',
    body: JSON.stringify({ challenge_type: challengeType, misconception_id: misconceptionID ?? null }),
  })
  return response.data
}

export async function answerChallenge(courseID: number, challengeID: number, answer: string): Promise<ChallengeAnswerResult> {
  const response = await request<ChallengeAnswerResponse>(`/api/v1/courses/${courseID}/challenges/${challengeID}/answers`, {
    method: 'POST',
    body: JSON.stringify({ answer }),
  })
  return response.data
}
