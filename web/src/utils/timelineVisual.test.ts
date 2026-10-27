import { describe, expect, it } from 'vitest'
import type { LifeEvent } from '@/api/life'
import {
  addMonths,
  dateToDay,
  dayToDate,
  filterTimelineEvents,
  groupTimelineByMonth,
  monthTicks,
  normalizeCategory,
  sortTimelineEvents,
  toTimelineEvent,
} from './timelineVisual'

const event = (id: number, occurredOn: string, patch: Partial<LifeEvent> = {}): LifeEvent => ({
  id,
  title: `事件 ${id}`,
  occurred_on: occurredOn,
  description: '',
  primary_domain: 'work',
  secondary_domain: '',
  milestone: false,
  goal_id: null,
  external_url: '',
  link_name: '',
  creation_key: '',
  source_type: '',
  source_id: null,
  source_title: '',
  source_available: true,
  source_status: '',
  created_at: '2026-10-21T08:00:00Z',
  updated_at: '2026-10-21T08:00:00Z',
  ...patch,
})

describe('timeline adapter', () => {
  it('keeps business identity and existing relations only', () => {
    const mapped = toTimelineEvent(event(7, '2026-10-09', { goal_id: 3, source_type: 'project', source_id: 4, source_title: '产品官网重构' }), new Map([[3, '长期目标']]))
    expect(mapped.id).toBe('life:7')
    expect(mapped.href).toBe('/life?event=7')
    expect(mapped.category).toBe('work')
    expect(mapped.goalTitle).toBe('长期目标')
    expect(mapped.sourceTitle).toBe('产品官网重构')
  })

  it('falls back to uncategorized for unknown domains', () => {
    expect(normalizeCategory('nope')).toBe('uncategorized')
    expect(normalizeCategory('')).toBe('uncategorized')
    expect(normalizeCategory('experiences')).toBe('experiences')
  })

  it('sorts by occurred date desc and id desc', () => {
    const sorted = sortTimelineEvents([
      toTimelineEvent(event(1, '2026-01-01'), new Map()),
      toTimelineEvent(event(3, '2026-05-01'), new Map()),
      toTimelineEvent(event(2, '2026-05-01'), new Map()),
    ])
    expect(sorted.map((item) => item.id)).toEqual(['life:3', 'life:2', 'life:1'])
  })

  it('filters by category (including secondary), milestone and range', () => {
    const events = [
      toTimelineEvent(event(1, '2026-01-10', { primary_domain: 'work' }), new Map()),
      toTimelineEvent(event(2, '2026-02-10', { primary_domain: 'health', secondary_domain: 'work', milestone: true }), new Map()),
      toTimelineEvent(event(3, '2026-03-10', { primary_domain: 'learning' }), new Map()),
    ]
    expect(filterTimelineEvents(events, { categories: ['work'] }).map((item) => item.id)).toEqual(['life:1', 'life:2'])
    expect(filterTimelineEvents(events, { milestonesOnly: true }).map((item) => item.id)).toEqual(['life:2'])
    expect(filterTimelineEvents(events, { from: '2026-02-01', to: '2026-02-28' }).map((item) => item.id)).toEqual(['life:2'])
  })

  it('groups by month in descending order', () => {
    const months = groupTimelineByMonth([
      toTimelineEvent(event(1, '2026-01-10'), new Map()),
      toTimelineEvent(event(2, '2026-01-20'), new Map()),
      toTimelineEvent(event(3, '2026-02-02'), new Map()),
    ])
    expect(months.map((month) => month.month)).toEqual(['2026-02', '2026-01'])
    expect(months[1]!.events).toHaveLength(2)
  })

  it('handles date helpers around year boundaries', () => {
    expect(addMonths('2026-01-31', 2)).toBe('2026-03-31')
    expect(dayToDate(dateToDay('2026-03-01'))).toBe('2026-03-01')
    expect(monthTicks('2025-11-01', '2026-02-28').map((tick) => tick.month)).toEqual(['2025-11', '2025-12', '2026-01', '2026-02'])
    expect(monthTicks('2025-11-01', '2026-02-28').filter((tick) => tick.yearStart)).toHaveLength(1)
  })
})
