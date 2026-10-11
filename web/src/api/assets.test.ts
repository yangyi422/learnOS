import { describe, expect, it } from 'vitest'
import { assetFormFrom, assetTotals, assetChange, comparableAssets, localAssetDate, parseAssetYuan, type AssetSnapshot, type AssetForm } from './assets'
const empty = (): AssetForm => ({ snapshot_date: '2026-01-01', bank: '', platform: '', investment: '', other: '', debt: '', note: '' })
const row = (date: string, net: string, complete = true): AssetSnapshot => ({ id: Number(date.replaceAll('-', '')), snapshot_date: date, currency: 'CNY', bank_minor: '10', platform_minor: '20', investment_minor: '0', other_minor: '0', debt_minor: '100', asset_total_minor: '30', net_asset_minor: net, complete, note: '保留历史', version: 1, created_at: '', updated_at: '' })
describe('asset exact calculations and historical scope', () => {
  it('distinguishes zero and missing amounts and keeps negative net assets', () => {
    const form = empty(); expect(assetTotals(form).hasAmount).toBe(false)
    form.bank = '0'; expect(assetTotals(form)).toMatchObject({ asset: 0n, debt: null, net: null, complete: false })
    Object.assign(form, { bank: '0.10', platform: '0.20', investment: '0', other: '0', debt: '1.00' })
    expect(assetTotals(form)).toMatchObject({ asset: 30n, net: -70n, complete: true })
    form.bank = ''; expect(assetTotals(form).complete).toBe(false)
    expect(parseAssetYuan('9999999999.99')).toBe(999999999999n)
    for (const bad of ['-1', '1e3', '0.001', '10000000000', 'NaN']) expect(() => parseAssetYuan(bad)).toThrow()
  })
  it('copies values without changing the original or moving the calendar date', () => {
    const original = row('2025-12-31', '-70'); original.platform_minor = null
    const copied = assetFormFrom(original, '2026-02-01'); expect(copied).toMatchObject({ snapshot_date: '2026-02-01', bank: '0.10', platform: '', investment: '0.00' })
    copied.bank = '300'; expect(original.bank_minor).toBe('10'); expect(original.snapshot_date).toBe('2025-12-31')
    expect(localAssetDate(new Date(2026, 0, 1, 0, 1))).toBe('2026-01-01')
  })
  it('sorts actual dates, excludes partial scopes and calculates comparable differences', () => {
    const rows = [row('2026-03-10', '120'), row('2026-01-01', '9000', false), row('2025-12-31', '-70')]
    expect(comparableAssets(rows).map(r => r.snapshot_date)).toEqual(['2025-12-31', '2026-03-10'])
    expect(assetChange(rows)?.amount).toBe('190')
    expect(assetChange([])).toBe(null); expect(assetChange([rows[0]!])).toBe(null)
  })
})
