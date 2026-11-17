export interface MediaItem {
  external_id?: string
  id: number
  Type: string
  Title: string
  Source: string
  Description?: string
  Authors?: string[]
  Genres?: string[]
  Tags?: string[]
  Year?: number
  Rating?: number
  cover_id?: string
  favorite?: boolean
  notes?: string
  updated_at?: string
  // 剧集字段：电影、电视剧、电子书为空。
  series_external_id?: string
  season_external_id?: string
  season_number?: number
  episode_number?: number
  premiere_date?: string
  runtime_seconds?: number
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
export interface MediaSeason {
  season_number: number
  season_external_id?: string
  episodes: number
}

export interface SeriesDetail {
  series: MediaItem
  seasons: MediaSeason[]
  episode_count: number
}

// 剧集详情与分集列表都只读，沿用现有认证与媒体错误文案。
export async function loadSeriesDetail(externalId: string, showR18 = false): Promise<SeriesDetail> {
  const params = new URLSearchParams({ show_r18: String(showR18) })
  const response = await fetch(`/api/v1/library/media/series/${encodeURIComponent(externalId)}?` + params)
  if (response.status === 404) throw new MediaNotFoundError('这部剧集不在资料库中')
  if (!response.ok) throw new Error('剧集读取失败，请重试。')
  const payload = await response.json()
  const data = payload?.data ?? {}
  if (!data.series) throw new Error('剧集数据不完整。')
  return {
    series: data.series as MediaItem,
    seasons: Array.isArray(data.seasons) ? (data.seasons as MediaSeason[]) : [],
    episode_count: Number(data.episode_count ?? 0),
  }
}

export async function loadSeriesEpisodes(externalId: string, season?: number, showR18 = false): Promise<MediaItem[]> {
  const params = new URLSearchParams({ show_r18: String(showR18) })
  if (typeof season === 'number') params.set('season', String(season))
  const response = await fetch(`/api/v1/library/media/series/${encodeURIComponent(externalId)}/episodes?` + params)
  if (response.status === 404) throw new MediaNotFoundError('这部剧集不在资料库中')
  if (!response.ok) throw new Error('分集读取失败，请重试。')
  const payload = await response.json()
  const items = payload?.data?.items
  return Array.isArray(items) ? (items as MediaItem[]) : []
}

export class MediaNotFoundError extends Error {}