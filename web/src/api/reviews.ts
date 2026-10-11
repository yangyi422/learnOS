import { request } from './http'
import type { Summary as FinanceSummary } from './finance'
import type { AssetSnapshot } from './assets'
export interface ReviewItem { related_event_id?: number; related_available?: boolean; key: string; kind: string; source_id: number; parent_id: number; title: string; date: string; detail: string; state: string; domain: string; milestone: boolean; url: string; available: boolean }
export interface ReviewFacts { month: string; timezone: string; from: string; to: string; generated_at: string; in_progress: boolean; actions: ReviewItem[]; learning: ReviewItem[]; events: ReviewItem[]; goals: ReviewItem[]; finance: { ledger_id: number; name: string; currency: string; summary: FinanceSummary | null; available: boolean }[]; assets: { snapshot: AssetSnapshot; available: boolean }[]; limits: string[] }
export interface PersonalReview { memorable: string; thoughts: string; next_focus: string; external_url: string }
export interface MonthlyReview extends PersonalReview { id: number; month: string; facts_id: number; version: number; created_at: string; updated_at: string }
export interface FactVersion { id: number; generated_at: string; confirmed_at: string }
export interface ReviewDocument { review: MonthlyReview | null; facts_id: number; facts: ReviewFacts; history?: FactVersion[]; archived_personal?: MonthlyReview }
async function data<T>(path: string, method?: string, input?: unknown) { return (await request<{ data: T }>(`/api/v1/life/reviews${path}`, method ? { method, body: input === undefined ? undefined : JSON.stringify(input) } : undefined)).data }
export const listReviews = () => data<MonthlyReview[]>('')
export const getReview = (month: string) => data<ReviewDocument>(`/${month}`)
export const previewReview = (month: string) => data<ReviewDocument>(`/${month}/preview`, 'POST')
export const reviewHistory = (month: string, id: number) => data<ReviewDocument>(`/${month}/history/${id}`)
export const saveReview = (month: string, input: PersonalReview & { facts_id: number; version: number; confirm_replace: boolean }) => data<ReviewDocument>(`/${month}`, 'PUT', input)
export function reviewCurrentMonth(now = new Date()) { const parts = new Intl.DateTimeFormat('en', { timeZone: 'Asia/Shanghai', year: 'numeric', month: '2-digit' }).formatToParts(now); return `${parts.find(p=>p.type==='year')!.value}-${parts.find(p=>p.type==='month')!.value}` }
