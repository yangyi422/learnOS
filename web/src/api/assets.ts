import { request } from './http'
export const assetFields = [
  ['bank', '银行存款'], ['platform', '支付平台余额'], ['investment', '投资资产'], ['other', '其他资产'], ['debt', '负债总额'],
] as const
export type AssetField = typeof assetFields[number][0]
export type AssetForm = Record<AssetField, string> & { snapshot_date: string; note: string }
export interface AssetSnapshot {
  id: number; snapshot_date: string; currency: 'CNY'; bank_minor: string | null; platform_minor: string | null;
  investment_minor: string | null; other_minor: string | null; debt_minor: string | null;
  asset_total_minor: string | null; net_asset_minor: string | null; complete: boolean; note: string;
  version: number; created_at: string; updated_at: string;
}
export type AssetInput = Record<AssetField, string | null> & { snapshot_date: string; note: string; version?: number }
async function data<T>(path: string, init?: RequestInit) { return (await request<{ data: T }>(`/api/v1/finance/assets${path}`, init)).data }
export const listAssets = () => data<AssetSnapshot[]>('')
export const getAsset = (id: number) => data<AssetSnapshot>(`/${id}`)
export const saveAsset = (id: number | undefined, input: AssetInput) => data<AssetSnapshot>(id ? `/${id}` : '', { method: id ? 'PUT' : 'POST', body: JSON.stringify(input) })
export const deleteAsset = (row: AssetSnapshot) => data<{ deleted: boolean }>(`/${row.id}?version=${row.version}`, { method: 'DELETE' })
export function localAssetDate(now = new Date()) { return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}` }
export function parseAssetYuan(value: string): bigint | null {
  const s = value.trim()
  if (!s) return null
  if (!/^\d{1,10}(\.\d{1,2})?$/.test(s)) throw new Error('金额需为非负数，最多两位小数、10 位整数。')
  const [whole, fraction = ''] = s.split('.')
  return BigInt(whole!) * 100n + BigInt(fraction.padEnd(2, '0'))
}
export function assetTotals(form: AssetForm) {
  const values = assetFields.map(([key]) => parseAssetYuan(form[key]))
  const knownAssets = values.slice(0, 4).filter((v): v is bigint => v !== null)
  const asset = knownAssets.length ? knownAssets.reduce((a, b) => a + b, 0n) : null
  const debt = values[4] ?? null
  return { asset, debt, net: asset !== null && debt !== null ? asset - debt : null, complete: values.every(v => v !== null), hasAmount: values.some(v => v !== null) }
}
export function assetFormFrom(row: AssetSnapshot, date = row.snapshot_date): AssetForm {
  const yuan = (value: string | null) => { if (value === null) return ''; const n = BigInt(value); return `${n / 100n}.${String(n % 100n).padStart(2, '0')}` }
  return { snapshot_date: date, bank: yuan(row.bank_minor), platform: yuan(row.platform_minor), investment: yuan(row.investment_minor), other: yuan(row.other_minor), debt: yuan(row.debt_minor), note: row.note }
}
export function comparableAssets(rows: AssetSnapshot[]) { return rows.filter(r => r.complete && r.net_asset_minor !== null).sort((a, b) => a.snapshot_date.localeCompare(b.snapshot_date)) }
export function assetChange(rows: AssetSnapshot[]) {
  const ordered = comparableAssets(rows), latest = ordered.at(-1), previous = ordered.at(-2)
  return latest && previous ? { latest, previous, amount: (BigInt(latest.net_asset_minor!) - BigInt(previous.net_asset_minor!)).toString() } : null
}
