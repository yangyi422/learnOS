import { describe, expect, it } from 'vitest'
import { isReservedMediaKind, mediaCategoryByRoute, mediaCategoryRoutes, mediaCountLabel, mediaKindBadges, mediaKindName, mediaKindOptions, mediaSortOptions, mediaSources, mediaStatusLabel, mediaSubtitle, recentMediaNames, sortMediaItems } from './mediaHub'

describe('media hub kinds', () => {
  it('lists the three connected libraries and the reserved game entry in order', () => {
    expect(mediaKindOptions.map(option => option.label)).toEqual(['全部类型', '电子书', '电影', '电视剧', '游戏'])
    expect(mediaKindOptions.map(option => option.value)).toEqual(['all', 'book', 'movie', 'series', 'game'])
    expect(mediaKindBadges.map(option => option.value)).toEqual(['book', 'movie', 'series', 'game'])
  })

  it('treats only the game entry as reserved so it never reaches the backend', () => {
    expect(isReservedMediaKind('game')).toBe(true)
    for (const kind of ['all', 'book', 'movie', 'series']) expect(isReservedMediaKind(kind)).toBe(false)
  })

  it('resolves display names and falls back to the raw value', () => {
    expect(mediaKindName('book')).toBe('电子书')
    expect(mediaKindName('game')).toBe('游戏')
    expect(mediaKindName('unexpected')).toBe('unexpected')
  })

  it('shows a dash for unknown counts so a failed request never reads as zero items', () => {
    expect(mediaCountLabel(undefined)).toBe('—')
    expect(mediaCountLabel(0)).toBe('0')
    expect(mediaCountLabel(12)).toBe('12')
  })

  it('builds the gallery subtitle from type, source, year and authors', () => {
    expect(mediaSubtitle({ Type: 'book', Source: 'calibre', Year: 2021, Authors: ['卡尔·纽波特'] })).toBe('电子书 · calibre · 2021 · 卡尔·纽波特')
    expect(mediaSubtitle({ Type: 'movie', Source: 'jellyfin' })).toBe('电影 · jellyfin')
    expect(mediaSubtitle({ Type: 'series', Source: 'jellyfin', Authors: ['甲', '乙', '丙'] })).toBe('电视剧 · jellyfin · 甲、乙')
  })

  it('caps the recent names shown on the library entry card', () => {
    expect(recentMediaNames([{ Title: '深度工作' }, { Title: '沙丘' }, { Title: '第三部' }])).toBe('深度工作、沙丘')
    expect(recentMediaNames([])).toBe('')
  })
})
describe('media categories', () => {
  it('maps each entry card to its category route', () => {
    expect(mediaCategoryRoutes).toEqual({
      book: '/explore/library/media/books',
      movie: '/explore/library/media/movies',
      series: '/explore/library/media/series',
      game: '/explore/library/media/games',
    })
    expect(mediaCategoryByRoute.movies).toBe('movie')
    expect(mediaCategoryByRoute.unknown).toBeUndefined()
  })

  it('labels sources and reservation state per type', () => {
    expect(mediaSources.book).toBe('Calibre · OPDS')
    expect(mediaSources.game).toBe('Steam · Windows')
    expect(mediaStatusLabel('book')).toBe('已接入')
    expect(mediaStatusLabel('game')).toBe('即将接入')
  })

  it('sorts within the loaded page without mutating the input', () => {
    const rows = [
      { id: 1, Title: '沙丘', Year: 2021 },
      { id: 3, Title: '深度工作', Year: 2016 },
      { id: 2, Title: '没有年份', Year: 0 },
    ]
    expect(sortMediaItems(rows, 'recent').map(row => row.id)).toEqual([3, 2, 1])
    expect(sortMediaItems(rows, 'year').map(row => row.id)).toEqual([1, 3, 2])
    expect(sortMediaItems(rows, 'title').map(row => row.Title)).toEqual(['没有年份', '沙丘', '深度工作'].sort((a, b) => a.localeCompare(b, 'zh-Hans-CN')))
    expect(rows.map(row => row.id)).toEqual([1, 3, 2])
  })
})