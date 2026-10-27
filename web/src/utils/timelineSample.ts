import type { TimelineCategoryKey, TimelineEvent } from '@/utils/timelineVisual'

/** Deterministic, clearly-labelled sample data for capacity and visual experiments. */
const TITLES: Record<TimelineCategoryKey, string[]> = {
  work: ['完成一次项目复盘', '交付一个阶段成果', '整理工作区与优先级'],
  learning: ['读完一本主题书', '完成一次课程单元', '整理一份知识笔记'],
  experiences: ['在城市里散步一整天', '看一场展览', '去一家想去的书店'],
  health: ['恢复一段稳定的作息', '完成一次长距离步行', '调整饮食与运动节奏'],
  relationships: ['和很久没见的朋友吃饭', '陪家人过了一个周末', '写下一封想说的话'],
  finance: ['整理一次年度预算', '记录一笔重要支出', '完成一次资产盘点'],
  home: ['重新布置书桌', '整理一次家里的储物', '换掉用了很久的家具'],
  rhythm: ['调整每周的时间安排', '留出一段完整的空闲', '重新安排早晨的节奏'],
  uncategorized: ['记录一段没有分类的经历'],
}

const CATEGORY_ORDER: TimelineCategoryKey[] = [
  'work', 'learning', 'experiences', 'health', 'relationships', 'finance', 'home', 'rhythm', 'uncategorized',
]

function seeded(seed: number): () => number {
  let state = seed >>> 0
  return () => {
    state = (state * 1664525 + 1013904223) >>> 0
    return state / 0xffffffff
  }
}

function isoDay(dayOffset: number): string {
  const base = Date.UTC(2026, 9, 21)
  return new Date(base - dayOffset * 86_400_000).toISOString().slice(0, 10)
}

/**
 * `count` synthetic events over the last ~18 months. Deterministic for a given
 * size, so screenshots and performance runs are comparable. No real user data.
 */
export function sampleTimeline(count: number): TimelineEvent[] {
  const random = seeded(20261021 + count)
  const events: TimelineEvent[] = []
  for (let index = 0; index < count; index += 1) {
    const category = CATEGORY_ORDER[Math.floor(random() * CATEGORY_ORDER.length)] ?? 'work'
    const pool = TITLES[category]
    const title = pool[Math.floor(random() * pool.length)] ?? '示例事件'
    const goalId = random() < 0.34 ? (random() < 0.5 ? 9001 : 9002) : undefined
    events.push({
      id: `sample:${index}`,
      sourceId: index,
      title: `${title} · 样本 ${index + 1}`,
      occurredAt: isoDay(Math.floor(random() * 540)),
      category,
      secondaryCategory: random() < 0.18 ? (CATEGORY_ORDER[Math.floor(random() * CATEGORY_ORDER.length)] ?? undefined) : undefined,
      summary: '示例数据，用于容量与交互实验，不对应任何真实经历。',
      isMilestone: random() < 0.14,
      goalId,
      goalTitle: goalId === 9001 ? '长期阅读计划' : goalId === 9002 ? '城市漫游计划' : undefined,
      sourceType: '',
      sourceTitle: '',
      href: '/life',
      externalUrl: '',
    })
  }
  return events
}
