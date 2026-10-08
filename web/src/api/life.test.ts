import { afterEach, describe, expect, it, vi } from 'vitest'
import { domainLabel, goalStatusLabel, lifeDomains, lifeRequestKey, listLifeEvents, saveLifeEvent, addLifeEntry } from './life'
import { localCalendarDate } from '../utils/projectTasks'
afterEach(() => vi.unstubAllGlobals())
describe('生活档案公共逻辑', () => {
  it('提供八个独立领域与五种目标状态', () => { expect(lifeDomains).toHaveLength(8); expect(new Set(lifeDomains.map(d => d[0])).size).toBe(8); expect(domainLabel('home')).toBe('居住与生活环境'); expect(goalStatusLabel('achieved')).toBe('已实现') })
  it('日期沿用本地日历且请求键可独立重试', () => { const date = new Date(2026, 9, 8, 0, 5); expect(localCalendarDate(date)).toBe('2026-10-08'); expect(lifeRequestKey()).not.toBe(lifeRequestKey()) })
  it('筛选和 Inbox 转换复用 API 认证通道', async () => {
    const fetch = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ data: [] }) }); vi.stubGlobal('fetch', fetch)
    await listLifeEvents('home', true); expect(fetch.mock.calls[0]![0]).toBe('/api/v1/life/events?domain=home&milestones=true')
    const input = { creation_key: 'stable-key', title: '重逢', occurred_on: '2020-01-02', description: '原文', primary_domain: '', secondary_domain: '', milestone: false, goal_id: null, external_url: '', link_name: '' }
    await saveLifeEvent(input, undefined, 9); expect(fetch.mock.calls[1]![0]).toBe('/api/v1/inbox/9/convert-to-life-event'); expect(JSON.parse(fetch.mock.calls[1]![1].body).creation_key).toBe('stable-key')
    await addLifeEntry(2, { creation_key: 'entry-key', occurred_on: '2026-10-08', content: '决定', reason: '' }); expect(fetch.mock.calls[2]![0]).toBe('/api/v1/life/goals/2/entries')
  })
})
