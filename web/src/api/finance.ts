import { request } from './http'
export interface Ledger { id: number; name: string; revision: number }
export interface Transaction { id: number; external_id: string; occurred_at: string; kind: string; amount_minor: string; currency: string; category: string; subcategory: string; note: string; tags: string[]; raw?: Record<string, string>; source: string; batch_id: number; created_at: string; updated_at: string }
export interface Batch { id: number; ledger_id: number; filename: string; status: string; added: number; updated: number; unchanged: number; missing: number; removed: number; row_count: number; complete_scope: boolean; scope_from: string; scope_to: string; created_at: string; imported_at: string | null; undone_at: string | null }
export interface Preview { batch: Batch | null; parse: { total: number; invalid: number; income: number; expense: number; transfer: number; from: string; to: string; errors: { line: number; message: string }[] }; sample: Transaction[]; warnings: string[] }
export interface Amount { name: string; amount_minor: string; count: number; children?: Amount[] }
export interface Summary { count: number; income_minor: string; expense_minor: string; difference_minor: string; currency: string; coverage_from: string; coverage_to: string; last_import: string | null; years: string[]; currencies: string[]; all_tags: string[]; all_categories: string[]; months: { month: string; income_minor: string; expense_minor: string; count: number }[]; categories: Amount[]; tags: Amount[] }
export interface TransactionPage { items: Transaction[]; total: number; page: number; page_size: number }
export const financeHeaders = ['ID', '时间', '分类', '二级分类', '类型', '金额', '币种', '账户1', '账户2', '备注', '已报销', '手续费', '优惠券', '记账者', '账单标记', '标签', '账单图片', '关联账单']
async function data<T>(path: string, init?: RequestInit) { return (await request<{ data: T }>(`/api/v1/finance${path}`, init)).data }
export const listLedgers = () => data<Ledger[]>('/ledgers')
export const previewFinance = (body: FormData) => data<Preview>('/preview', { method: 'POST', body })
export const importFinance = (id: number, remove: boolean) => data<Batch>(`/batches/${id}/import`, { method: 'POST', body: JSON.stringify({ remove_missing: remove, confirm_missing: remove, confirm_stable_ids: remove }) })
export const undoFinance = (id: number) => data<Batch>(`/batches/${id}/undo`, { method: 'POST', body: '{}' })
export const listBatches = (ledger: number) => data<Batch[]>(`/ledgers/${ledger}/batches`)
export const getSummary = (ledger: number, params: URLSearchParams) => data<Summary>(`/ledgers/${ledger}/summary?${params}`)
export const listTransactions = (ledger: number, params: URLSearchParams) => data<TransactionPage>(`/ledgers/${ledger}/transactions?${params}`)
export const getTransaction = (ledger: number, id: number) => data<Transaction>(`/ledgers/${ledger}/transactions/${id}`)
export function financeMoney(value: string) { const n = BigInt(value); const a = n < 0n ? -n : n; return `${n < 0n ? '-' : ''}${(a / 100n).toLocaleString('zh-CN')}.${String(a % 100n).padStart(2, '0')}` }
export function financeShare(value: string, total: string) { const v = BigInt(value), t = BigInt(total); return t > 0n ? `${Number(v * 1000n / t) / 10}%` : '—' }
export const kindLabel = (kind: string) => ({ income: '收入', expense: '支出', transfer: '转账' })[kind as 'income'] ?? kind
export function monthIncomplete(month: string, now = new Date()) { return month >= `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}` }
