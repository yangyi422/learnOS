export interface MediaItem {
  id: number
  Type: string
  Title: string
  Source: string
  Description?: string
  Authors?: string[]
  Genres?: string[]
  Tags?: string[]
  Year?: number
  cover_id?: string
  favorite?: boolean
  notes?: string
  updated_at?: string
}

export interface MediaPage {
  items: MediaItem[]
  total: number
}

export const MEDIA_PREVIEW_LIMIT = 6
export const mediaSectionKinds = ['book', 'movie', 'series'] as const
export type MediaSectionKind = (typeof mediaSectionKinds)[number]

export const mediaCoverURL = (coverId: string) => `/api/v1/library/images/${encodeURIComponent(coverId)}?variant=card-v1`

export async function listMedia(query: { q?: string; type?: string; source?: string; page?: number; showR18?: boolean } = {}): Promise<MediaPage> {
  const params = new URLSearchParams({
    q: query.q ?? '',
    type: query.type ?? '',
    source: query.source ?? '',
    page: String(query.page ?? 1),
    show_r18: String(query.showR18 ?? false),
  })
  const response = await fetch('/api/v1/library/media?' + params)
  if (!response.ok) throw new Error('媒体读取失败，请重试。')
  const payload = await response.json()
  const data = payload?.data ?? {}
  return { items: Array.isArray(data.items) ? data.items : [], total: Number(data.total ?? 0) }
}

export interface MediaOverview {
  total: number
  counts: Partial<Record<MediaSectionKind, number>>
  recent: MediaItem[]
  previews: Partial<Record<MediaSectionKind, MediaItem[]>>
  // 任一请求失败时置位：调用方降级展示，不把统计当 0。
  partial: boolean
}

// 复用现有 /library/media 接口聚合概览：每种类型一次请求就同时拿到数量与预览，不新增后端能力。
export async function loadMediaOverview(): Promise<MediaOverview> {
  const [all, ...perKind] = await Promise.allSettled([
    listMedia({ page: 1 }),
    ...mediaSectionKinds.map(type => listMedia({ type, page: 1 })),
  ])
  const counts: Partial<Record<MediaSectionKind, number>> = {}
  const previews: Partial<Record<MediaSectionKind, MediaItem[]>> = {}
  let partial = false
  perKind.forEach((result, index) => {
    const kind = mediaSectionKinds[index]!
    if (result.status === 'fulfilled') {
      counts[kind] = result.value.total
      previews[kind] = result.value.items.slice(0, MEDIA_PREVIEW_LIMIT)
      return
    }
    partial = true
  })
  let recent: MediaItem[] = []
  let total = 0
  if (all.status === 'fulfilled') {
    recent = all.value.items.slice(0, MEDIA_PREVIEW_LIMIT)
    total = all.value.total
  } else {
    partial = true
    total = Object.values(counts).reduce<number>((sum, value) => sum + (value ?? 0), 0)
  }
  return { total, counts, recent, previews, partial }
}