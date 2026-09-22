import { afterEach, describe, expect, it, vi } from 'vitest'
import { APIRequestError, request } from './http'

describe('request lifecycle errors', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('keeps lifecycle cancellation distinct from a network failure', async () => {
    const cancellation = new DOMException('aborted', 'AbortError')
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(cancellation))
    await expect(request('/api/v1/test')).rejects.toBe(cancellation)
  })

  it('maps an actual fetch failure to a retryable network error', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('socket closed')))
    await expect(request('/api/v1/test')).rejects.toMatchObject<Partial<APIRequestError>>({
      code: 'NETWORK_ERROR',
      retryable: true,
    })
  })
})
