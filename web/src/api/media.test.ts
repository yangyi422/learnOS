import { afterEach, describe, expect, it, vi } from 'vitest'
import { deleteMovies } from './media'

afterEach(() => vi.unstubAllGlobals())

describe('movie batch deletion', () => {
  it('sends only explicit IDs and requires the complete count', async () => {
    const fetch = vi.fn().mockResolvedValue(new Response(JSON.stringify({ data: { deleted: 2 } })))
    vi.stubGlobal('fetch', fetch)
    await expect(deleteMovies([4, 9])).resolves.toBe(2)
    expect(fetch).toHaveBeenCalledWith('/api/v1/library/media/batch-delete', {
      method: 'POST', headers: { 'Content-Type': 'application/json' }, body: '{"ids":[4,9]}',
    })
    fetch.mockResolvedValue(new Response(JSON.stringify({ data: { deleted: 1 } })))
    await expect(deleteMovies([4, 9])).rejects.toThrow('删除响应异常')
  })

  it.each([[401, '登录已失效'], [404, '刷新后重选'], [500, '媒体删除失败']])('reports HTTP %s without leaking internal errors', async (status, message) => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('private database stack', { status: Number(status) })))
    await expect(deleteMovies([4])).rejects.toThrow(String(message))
  })
})
