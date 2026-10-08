import { afterEach, expect, it, vi } from 'vitest'
import { archiveRecord, convertToRecord, listRecords, saveRecord } from './records'
afterEach(() => vi.unstubAllGlobals())
it('filters archived project records and uses existing Inbox conversion identity', async () => {
  const fetch = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ data: [] }) })
  vi.stubGlobal('fetch', fetch)
  await listRecords(12, true)
  expect(fetch.mock.calls[0]?.[0]).toBe('/api/v1/records?status=archived&project_id=12')
  const input = { content: 'context', project_id: null, external_url: '', link_name: '' }
  await convertToRecord(3, input)
  expect(fetch.mock.calls[1]?.[0]).toBe('/api/v1/inbox/3/convert-to-record')
  expect(fetch.mock.calls[1]?.[1].method).toBe('POST')
})
it('saves without inventing a title, carries create key, and clears optional project', async () => {
  const fetch = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ data: {} }) })
  vi.stubGlobal('fetch', fetch)
  const input = { content: 'context', creation_key: 'request-key', project_id: null, external_url: '', link_name: '' }
  await saveRecord(input)
  expect(JSON.parse(fetch.mock.calls[0]?.[1].body)).toEqual(input)
  await saveRecord(input, 9)
  expect(fetch.mock.calls[1]?.[0]).toBe('/api/v1/records/9')
  expect(fetch.mock.calls[1]?.[1].method).toBe('PUT')
  await archiveRecord(9, false)
  expect(JSON.parse(fetch.mock.calls[2]?.[1].body)).toEqual({ archived: false })
})
