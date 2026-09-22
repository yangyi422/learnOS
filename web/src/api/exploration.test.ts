import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { getExplorationRadar, invalidateExplorationRadar } from './exploration'

function responseFor(courseID: number) {
  return new Response(JSON.stringify({ data: { directions: [{
    id: courseID,
    context_course_id: courseID,
    source_domain_id: courseID,
    source_course_id: courseID,
    source_lesson_id: courseID * 10,
    target_course_id: courseID,
    target_lesson_id: courseID * 10,
    source_domain: { id: courseID, name: `domain-${courseID}` },
    source_course: { id: courseID, name: `course-${courseID}` },
    source_lesson: { id: courseID * 10, course_id: courseID, title: `lesson-${courseID}` },
    target_course: { id: courseID, name: `course-${courseID}` },
    target_lesson: { id: courseID * 10, course_id: courseID, title: `lesson-${courseID}` },
    reason_data: {},
  }] } }), { status: 200, headers: { 'Content-Type': 'application/json' } })
}

describe('exploration radar cache', () => {
  beforeEach(() => invalidateExplorationRadar())
  afterEach(() => vi.unstubAllGlobals())

  it('isolates cached recommendations by source domain', async () => {
    const fetchMock = vi.fn(async (input: string | URL | Request) => {
      const courseID = Number(new URL(String(input), 'http://learnos.test').searchParams.get('course_id'))
      return responseFor(courseID)
    })
    vi.stubGlobal('fetch', fetchMock)

    const first = await getExplorationRadar(101)
    const second = await getExplorationRadar(202)
    const firstAgain = await getExplorationRadar(101)

    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(first[0]?.source_domain_id).toBe(101)
    expect(second[0]?.source_domain_id).toBe(202)
    expect(firstAgain[0]?.source_course.name).toBe('course-101')
  })

  it('invalidates one domain without evicting the others', async () => {
    const fetchMock = vi.fn(async (input: string | URL | Request) => {
      const courseID = Number(new URL(String(input), 'http://learnos.test').searchParams.get('course_id'))
      return responseFor(courseID)
    })
    vi.stubGlobal('fetch', fetchMock)

    await getExplorationRadar(301)
    await getExplorationRadar(302)
    invalidateExplorationRadar(301)
    await getExplorationRadar(301)
    await getExplorationRadar(302)

    expect(fetchMock).toHaveBeenCalledTimes(3)
  })
})
