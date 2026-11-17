export type MediaKind = 'all' | 'book' | 'movie' | 'series' | 'game'

export interface MediaKindOption {
  label: string
  value: MediaKind
}

// 顺序即「统一内容库」的入口顺序：电子书、电影、电视剧，最后是预留的游戏入口。
export const mediaKindOptions: MediaKindOption[] = [
  { label: '全部类型', value: 'all' },
  { label: '电子书', value: 'book' },
  { label: '电影', value: 'movie' },
  { label: '电视剧', value: 'series' },
  { label: '游戏', value: 'game' },
]

// 概览 chips 与分区的展示顺序（不含「全部类型」）。
export const mediaKindBadges: MediaKindOption[] = mediaKindOptions.filter(option => option.value !== 'all')

// 游戏只是 UI 预留：后端没有 game 类型，查询前必须短路，不能把不支持的类型发给后端。
export function isReservedMediaKind(kind: string): boolean {
  return kind === 'game'
}

export function mediaKindName(value: string): string {
  return mediaKindOptions.find(option => option.value === value)?.label ?? value
}

// 统计与 chip 里的数量：未知（尚未加载或请求失败）显示破折号而不是 0。
export function mediaCountLabel(count: number | undefined): string {
  return count === undefined ? '—' : String(count)
}

// 画廊卡片的辅助信息行：类型 · 来源，再拼年份/作者中存在的几项。
export function mediaSubtitle(item: { Type: string; Source: string; Year?: number; Authors?: string[] }): string {
  const kind = mediaKindName(item.Type)
  const meta = [item.Year ? String(item.Year) : '', item.Authors?.length ? item.Authors.slice(0, 2).join('、') : '']
  return [kind, item.Source, ...meta.filter(Boolean)].filter(Boolean).join(' · ')
}

// 首页卡片里的“最近新增”一行只点到为止，最多两条。
export function recentMediaNames(items: { Title: string }[], limit = 2): string {
  return items.slice(0, limit).map(item => item.Title).join('、')
}
export type MediaCategoryKind = 'book' | 'movie' | 'series' | 'game'

// 首页四张入口卡 → 各自分类详情页。
export const mediaCategoryRoutes: Record<MediaCategoryKind, string> = {
  book: '/explore/library/media/books',
  movie: '/explore/library/media/movies',
  series: '/explore/library/media/series',
  game: '/explore/library/media/games',
}

// 路由段 → 内容类型。unknown 返回 null，由分类页兜底到 404 视图。
export const mediaCategoryByRoute: Record<string, MediaCategoryKind | undefined> = {
  books: 'book',
  movies: 'movie',
  series: 'series',
  games: 'game',
}

// 入口卡与分类页共用的来源说明。
export const mediaSources: Record<MediaCategoryKind, string> = {
  book: 'Calibre · OPDS',
  movie: 'Jellyfin',
  series: 'Jellyfin',
  game: 'Steam · Windows',
}

export function mediaStatusLabel(kind: MediaCategoryKind): string {
  return isReservedMediaKind(kind) ? '即将接入' : '已接入'
}

export type MediaSort = 'recent' | 'title' | 'year'

export const mediaSortOptions: { label: string; value: MediaSort }[] = [
  { label: '最近新增', value: 'recent' },
  { label: '标题 A→Z', value: 'title' },
  { label: '年份新→旧', value: 'year' },
]

// 分类页的排序在同一页数据内完成：接口按 id DESC 分页，不新增后端参数。
export function sortMediaItems<T extends { id: number; Title: string; Year?: number }>(items: T[], sort: MediaSort): T[] {
  const rows = [...items]
  if (sort === 'title') return rows.sort((a, b) => a.Title.localeCompare(b.Title, 'zh-Hans-CN'))
  if (sort === 'year') return rows.sort((a, b) => (b.Year || 0) - (a.Year || 0) || b.id - a.id)
  return rows.sort((a, b) => b.id - a.id)
}
// 剧集编号固定两位：S01E01，便于和 Jellyfin / 播放器字幕对齐。
export function formatEpisodeCode(season: number, episode: number): string {
  const pad = (value: number) => String(Math.max(0, Math.trunc(value))).padStart(2, '0')
  return `S${pad(season)}E${pad(episode)}`
}

export function seasonLabel(season: number): string {
  return season === 0 ? '特别篇' : `第 ${season} 季`
}

// 时长优先用“1 小时 36 分”，不足一小时只说分钟。
export function formatRuntime(seconds: number | undefined): string {
  if (!seconds || seconds <= 0) return ''
  const minutes = Math.round(seconds / 60)
  if (minutes < 60) return `${minutes} 分钟`
  const hours = Math.floor(minutes / 60)
  const rest = minutes % 60
  return rest === 0 ? `${hours} 小时` : `${hours} 小时 ${rest} 分`
}