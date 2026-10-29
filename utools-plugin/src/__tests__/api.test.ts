import { describe, expect, it, vi } from 'vitest'
import {
  CAPTURE_PATH,
  CaptureFailure,
  captureInbox,
  describeCaptureFailure,
  isHttpUrl,
  normalizeBaseUrl,
} from '../api'

function fakeResponse(status: number, body: unknown): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: async () => (body === undefined ? '' : JSON.stringify(body)),
  } as unknown as Response
}

async function failureOf(request: Promise<unknown>): Promise<CaptureFailure> {
  try {
    await request
  } catch (error) {
    if (error instanceof CaptureFailure) return error
    throw error
  }
  throw new Error('expected the capture request to fail')
}

const ITEM = {
  id: 7,
  content: '想法',
  source_type: 'manual',
  source_url: '',
  status: 'inbox',
  created_at: '2026-10-29T14:00:00+08:00',
}

const BASE_REQUEST = {
  baseUrl: 'http://127.0.0.1:8080',
  token: 'los_pat_secret',
  content: '想法',
  captureKey: 'cap-abc12345',
}

describe('captureInbox', () => {
  it('posts content and capture_key to the frozen endpoint', async () => {
    const fetchImpl = vi.fn().mockResolvedValue(fakeResponse(201, { data: ITEM }))
    const item = await captureInbox({
      ...BASE_REQUEST,
      baseUrl: 'http://127.0.0.1:8080/',
      fetchImpl: fetchImpl as unknown as typeof fetch,
    })

    expect(item).toEqual(ITEM)
    expect(fetchImpl).toHaveBeenCalledTimes(1)
    const [url, init] = fetchImpl.mock.calls[0]
    expect(url).toBe(`http://127.0.0.1:8080${CAPTURE_PATH}`)
    expect(init.method).toBe('POST')
    expect(init.headers.Authorization).toBe('Bearer los_pat_secret')
    expect(init.headers['Content-Type']).toBe('application/json')
    expect(JSON.parse(init.body)).toEqual({ content: '想法', capture_key: 'cap-abc12345' })
  })

  it('maps 401 to a non-retryable credential failure', async () => {
    const fetchImpl = vi.fn().mockResolvedValue(
      fakeResponse(401, { error: { code: 'UNAUTHORIZED', message: '认证失败，请检查凭证是否有效。' } }),
    )
    const failure = await failureOf(
      captureInbox({ ...BASE_REQUEST, fetchImpl: fetchImpl as unknown as typeof fetch }),
    )

    expect(failure.kind).toBe('unauthorized')
    expect(failure.retryable).toBe(false)
    const guidance = describeCaptureFailure(failure)
    expect(guidance.action).toBe('config')
    expect(guidance.message).toContain('凭证')
  })

  it('keeps the server message as detail without leaking it into UI copy', async () => {
    const fetchImpl = vi
      .fn()
      .mockResolvedValue(fakeResponse(500, { error: { code: 'CAPTURE_FAILED', message: 'database is locked' } }))
    const failure = await failureOf(
      captureInbox({ ...BASE_REQUEST, fetchImpl: fetchImpl as unknown as typeof fetch }),
    )

    expect(failure.kind).toBe('server')
    expect(failure.retryable).toBe(true)
    expect(failure.detail).toBe('database is locked')
    const guidance = describeCaptureFailure(failure)
    expect(guidance.action).toBe('retry')
    expect(guidance.message).not.toContain('database')
  })

  it('maps 5xx to a retryable failure and network errors to network', async () => {
    const server = await failureOf(
      captureInbox({
        ...BASE_REQUEST,
        fetchImpl: vi.fn().mockResolvedValue(fakeResponse(503, undefined)) as unknown as typeof fetch,
      }),
    )
    expect(server.kind).toBe('server')
    expect(server.retryable).toBe(true)

    const network = await failureOf(
      captureInbox({
        ...BASE_REQUEST,
        fetchImpl: vi.fn().mockRejectedValue(new TypeError('Failed to fetch')) as unknown as typeof fetch,
      }),
    )
    expect(network.kind).toBe('network')
    expect(network.retryable).toBe(true)
    expect(describeCaptureFailure(network).action).toBe('retry')
  })

  it('maps 409 to a conflict failure', async () => {
    const failure = await failureOf(
      captureInbox({
        ...BASE_REQUEST,
        fetchImpl: vi
          .fn()
          .mockResolvedValue(fakeResponse(409, { error: { code: 'CAPTURE_DUPLICATE', message: 'duplicate' } })) as unknown as typeof fetch,
      }),
    )
    expect(failure.kind).toBe('conflict')
    expect(failure.retryable).toBe(false)
  })

  it('refuses to send without a config or with a malformed success body', async () => {
    const fetchImpl = vi.fn()
    const missingToken = await failureOf(
      captureInbox({ ...BASE_REQUEST, token: '', fetchImpl: fetchImpl as unknown as typeof fetch }),
    )
    expect(missingToken.kind).toBe('misconfigured')
    expect(fetchImpl).not.toHaveBeenCalled()

    const malformed = await failureOf(
      captureInbox({
        ...BASE_REQUEST,
        fetchImpl: vi.fn().mockResolvedValue(fakeResponse(201, { ok: true })) as unknown as typeof fetch,
      }),
    )
    expect(malformed.kind).toBe('server')
  })
})

describe('config helpers', () => {
  it('normalizes the base url and validates http(s)', () => {
    expect(normalizeBaseUrl('  http://127.0.0.1:8080///  ')).toBe('http://127.0.0.1:8080')
    expect(isHttpUrl('http://127.0.0.1:8080')).toBe(true)
    expect(isHttpUrl('https://lifeos.example.com')).toBe(true)
    expect(isHttpUrl('127.0.0.1:8080')).toBe(false)
    expect(isHttpUrl('file:///tmp/x')).toBe(false)
  })
})

describe('captureInbox transport selection', () => {
  it('uses the preload Node transport when uTools provides it (no page fetch, no CORS)', async () => {
    const request = vi.fn().mockResolvedValue({ status: 201, body: JSON.stringify({ data: ITEM }) })
    ;(globalThis as any).lifeos = { request }
    const pageFetch = vi.fn()
    vi.stubGlobal('fetch', pageFetch)

    const item = await captureInbox({ ...BASE_REQUEST })

    expect(item).toEqual(ITEM)
    expect(request).toHaveBeenCalledTimes(1)
    expect(request.mock.calls[0][0]).toMatchObject({
      url: `http://127.0.0.1:8080${CAPTURE_PATH}`,
      method: 'POST',
    })
    expect(request.mock.calls[0][0].headers.Authorization).toBe('Bearer los_pat_secret')
    expect(pageFetch).not.toHaveBeenCalled()

    delete (globalThis as any).lifeos
    vi.unstubAllGlobals()
  })

  it('maps a 401 from the Node transport to an unauthorized failure kept for retry-copy', async () => {
    const request = vi.fn().mockResolvedValue({
      status: 401,
      body: JSON.stringify({ error: { code: 'UNAUTHORIZED', message: '认证失败，请检查凭证是否有效。' } }),
    })
    ;(globalThis as any).lifeos = { request }

    const failure = await failureOf(captureInbox({ ...BASE_REQUEST }))
    expect(failure.kind).toBe('unauthorized')
    expect(failure.status).toBe(401)

    delete (globalThis as any).lifeos
  })
})
