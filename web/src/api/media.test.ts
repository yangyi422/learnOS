import { afterEach, describe, expect, it, vi } from 'vitest'
import { deleteMedia } from './media'

afterEach(() => vi.unstubAllGlobals())

describe('media batch deletion', () => {
  it('sends explicit IDs and defaults to movies, requiring the complete count', async () => {
    const fetch = vi.fn().mockResolvedValue(new Response(JSON.stringify({ data: { deleted: 2 } })))
    vi.stubGlobal('fetch', fetch)
    await expect(deleteMedia([4, 9])).resolves.toBe(2)
    expect(fetch).toHaveBeenCalledWith('/api/v1/library/media/batch-delete', {
      method: 'POST', headers: { 'Content-Type': 'application/json' }, body: '{"ids":[4,9],"type":"movie"}',
    })
    fetch.mockResolvedValue(new Response(JSON.stringify({ data: { deleted: 1 } })))
    await expect(deleteMedia([4, 9])).rejects.toThrow('删除响应异常')
  })

  it.each(['book', 'series'] as const)('sends the selected %s category', async type => {
    const fetch = vi.fn().mockResolvedValue(new Response(JSON.stringify({ data: { deleted: 1 } })))
    vi.stubGlobal('fetch', fetch)
    await expect(deleteMedia([3], type)).resolves.toBe(1)
    expect(fetch).toHaveBeenCalledWith('/api/v1/library/media/batch-delete', expect.objectContaining({
      body: JSON.stringify({ ids: [3], type }),
    }))
  })

  it.each([[401, '登录已失效'], [404, '刷新后重选'], [500, '媒体删除失败']])('reports HTTP %s without leaking internal errors', async (status, message) => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('private database stack', { status: Number(status) })))
    await expect(deleteMedia([4])).rejects.toThrow(String(message))
  })
})
