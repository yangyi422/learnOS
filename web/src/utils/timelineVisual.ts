import type { LifeEvent } from '@/api/life'

/**
 * Read-only adapter from life records to a visual timeline model.
 *
 * One source of truth: `life_events` (the curated timeline). Nothing here writes
 * business data, invents events, or infers relations.
 */

export const TIMELINE_CATEGORIES = [
  { key: 'work', label: '工作与创造', hue: 228 },
  { key: 'learning', label: '学习与认知', hue: 265 },
  { key: 'experiences', label: '兴趣与体验', hue: 338 },
  { key: 'health', label: '身心健康', hue: 162 },
  { key: 'relationships', label: '人际与关系', hue: 22 },
  { key: 'finance', label: '财务与资产', hue: 196 },
  { key: 'home', label: '居住与生活环境', hue: 104 },
  { key: 'rhythm', label: '时间与生活节奏', hue: 44 },
  { key: 'uncategorized', label: '未分类', hue: 222 },
] as const

export type TimelineCategoryKey = (typeof TIMELINE_CATEGORIES)[number]['key']

const CATEGORY_KEYS = new Set<string>(TIMELINE_CATEGORIES.map((category) => category.key))

export interface TimelineEvent {
  /** `life:<id>` — keeps the business identity visible in the DOM. */
  id: string
  sourceId: number
  title: string
  /** `YYYY-MM-DD`, the date the event actually happened. */
  occurredAt: string
  category: TimelineCategoryKey
  secondaryCategory?: TimelineCategoryKey
  summary: string
  isMilestone: boolean
  goalId?: number
  goalTitle?: string
  sourceType: string
  sourceTitle: string
  href: string
  externalUrl: string
}

export interface TimelineMonth {
  month: string
  label: string
  events: TimelineEvent[]
}

export interface TimelineFilter {
  categories?: TimelineCategoryKey[]
  milestonesOnly?: boolean
  from?: string
  to?: string
}

export function normalizeCategory(value: string): TimelineCategoryKey {
  return CATEGORY_KEYS.has(value) ? (value as TimelineCategoryKey) : 'uncategorized'
}

export function categoryLabel(key: TimelineCategoryKey): string {
  return TIMELINE_CATEGORIES.find((category) => category.key === key)?.label ?? '未分类'
}

export function categoryHue(key: TimelineCategoryKey): number {
  return TIMELINE_CATEGORIES.find((category) => category.key === key)?.hue ?? 222
}

export function toTimelineEvent(event: LifeEvent, goals: Map<number, string>): TimelineEvent {
  const goalId = event.goal_id ?? undefined
  return {
    id: `life:${event.id}`,
    sourceId: event.id,
    title: event.title,
    occurredAt: event.occurred_on,
    category: normalizeCategory(event.primary_domain),
    secondaryCategory: event.secondary_domain ? normalizeCategory(event.secondary_domain) : undefined,
    summary: event.description,
    isMilestone: event.milestone,
    goalId,
    goalTitle: goalId ? goals.get(goalId) : undefined,
    sourceType: event.source_type,
    sourceTitle: event.source_title,
    href: `/life?event=${event.id}`,
    externalUrl: event.external_url,
  }
}

export function sortTimelineEvents(events: TimelineEvent[]): TimelineEvent[] {
  return [...events].sort((left, right) => {
    if (left.occurredAt !== right.occurredAt) return left.occurredAt < right.occurredAt ? 1 : -1
    return right.sourceId - left.sourceId
  })
}

export function filterTimelineEvents(events: TimelineEvent[], filter: TimelineFilter = {}): TimelineEvent[] {
  const categories = filter.categories?.length ? new Set(filter.categories) : null
  return events.filter((event) => {
    if (filter.milestonesOnly && !event.isMilestone) return false
    if (filter.from && event.occurredAt < filter.from) return false
    if (filter.to && event.occurredAt > filter.to) return false
    if (!categories) return true
    return categories.has(event.category) || (event.secondaryCategory ? categories.has(event.secondaryCategory) : false)
  })
}

export function groupTimelineByMonth(events: TimelineEvent[]): TimelineMonth[] {
  const months = new Map<string, TimelineEvent[]>()
  for (const event of sortTimelineEvents(events)) {
    const month = event.occurredAt.slice(0, 7)
    const bucket = months.get(month)
    if (bucket) bucket.push(event)
    else months.set(month, [event])
  }
  return [...months.entries()]
    .sort((left, right) => (left[0] < right[0] ? 1 : -1))
    .map(([month, bucket]) => ({
      month,
      label: `${month.slice(0, 4)} 年 ${Number(month.slice(5, 7))} 月`,
      events: bucket,
    }))
}

export function dateToDay(value: string): number {
  const [year, month, day] = value.split('-').map(Number)
  return Date.UTC(year ?? 1970, (month ?? 1) - 1, day ?? 1) / 86_400_000
}

export function dayToDate(day: number): string {
  return new Date(day * 86_400_000).toISOString().slice(0, 10)
}

export function addMonths(date: string, months: number): string {
  const [year, month, day] = date.split('-').map(Number)
  const target = new Date(Date.UTC(year ?? 1970, (month ?? 1) - 1 + months, day ?? 1))
  return target.toISOString().slice(0, 10)
}

export function monthTicks(from: string, to: string): Array<{ month: string; label: string; yearStart: boolean }> {
  const ticks: Array<{ month: string; label: string; yearStart: boolean }> = []
  let current = `${from.slice(0, 7)}-01`
  const end = to.slice(0, 7)
  while (current.slice(0, 7) <= end && ticks.length < 240) {
    const month = current.slice(0, 7)
    ticks.push({
      month,
      label: month.slice(5, 7) === '01' ? month.slice(0, 4) : `${Number(month.slice(5, 7))}月`,
      yearStart: month.slice(5, 7) === '01',
    })
    current = addMonths(current, 1)
  }
  return ticks
}
