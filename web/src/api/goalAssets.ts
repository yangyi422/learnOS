import { request } from './http'
import type { AssetSnapshot } from './assets'
import type { LifeGoal } from './life'
export interface GoalAssetReference { id: number; goal_id: number; asset_snapshot_id: number; note: string; judgment_date: string; version: number; created_at: string; updated_at: string; snapshot: AssetSnapshot }
export interface GoalAssetInput { note: string; judgment_date: string; version?: number }
async function data<T>(path: string, method?: string, input?: unknown) { return (await request<{ data: T }>(`/api/v1${path}`, method ? { method, body: input === undefined ? undefined : JSON.stringify(input) } : undefined)).data }
export const listGoalAssets = (goal: number) => data<GoalAssetReference[]>(`/life/goals/${goal}/assets`)
export const listAssetGoals = (asset: number) => data<LifeGoal[]>(`/finance/assets/${asset}/goals`)
export const saveGoalAsset = (goal: number, asset: number, input: GoalAssetInput, edit = false) => data<Omit<GoalAssetReference, 'snapshot'>>(`/life/goals/${goal}/assets/${asset}`, edit ? 'PUT' : 'POST', input)
export const unlinkGoalAsset = (row: GoalAssetReference) => data(`/life/goals/${row.goal_id}/assets/${row.asset_snapshot_id}?version=${row.version}`, 'DELETE')
