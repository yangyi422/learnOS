import { afterEach, describe, expect, it, vi } from 'vitest'
import { createProject, createTask, listProjects, listTasks, moveTask, updateTask } from './projects'

describe('project API', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('uses board task endpoints and moves by task ID', async () => {
    const fetchMock = vi.fn(async (_path: string, _init?: RequestInit) => new Response(JSON.stringify({ data: { id: 3, title: '家庭照片', tasks: [] } }), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)

    await createProject({ title: '家庭照片' })
    await createTask({ project_id: 3, title: '筛选照片', status: 'inbox' })
    await updateTask(7, { priority: 'high' })
    await moveTask(7, { status: 'next', before_id: 8, expected_updated_at: '2026-09-24T00:00:00Z' })

    expect(fetchMock.mock.calls.map(call => call[0])).toEqual([
      '/api/v1/projects', '/api/v1/tasks', '/api/v1/tasks/7', '/api/v1/tasks/7/move',
    ])
    expect(fetchMock.mock.calls[3]?.[1]).toMatchObject({ method: 'PATCH', body: '{"status":"next","before_id":8,"expected_updated_at":"2026-09-24T00:00:00Z"}' })
  })

  it('returns an empty list from an empty workspace', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => new Response(JSON.stringify({ data: [] }), { status: 200 })))
    await expect(listProjects()).resolves.toEqual([])
    await expect(listTasks({ projectId: 3, status: 'doing', due: '2026-09-24' })).resolves.toEqual([])
  })
})
