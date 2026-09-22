import { request } from './http'

export interface BackupInfo { name: string; path: string; size: number; created_at: string }
export interface Diagnostics {
  app_version: string; environment: string; schema_version: number; database_status: { status: string; detail?: string }
  database_path: string; database_size: number; last_backup: BackupInfo | null; ai_provider: string; ai_model: string
  ai_timeouts_seconds: Record<string, number>; course_count: number; lesson_count: number; learning_turn_count: number; cognitive_state_count: number; generated_at: string
}
export interface ConsistencyIssue { severity: 'warning' | 'error'; category: string; code: string; entity: string; message: string; suggestion: string }
export interface ConsistencyReport { healthy: boolean; warnings: ConsistencyIssue[]; errors: ConsistencyIssue[] }
export interface AIConfiguration {
  provider: 'mock' | 'deepseek'; api_key_configured: boolean; base_url: string; model: string
  effective_provider: string; effective_model: string; last_successful_call_at: string | null; updated_at?: string
}
export interface AIConnectionTest { status: 'ok'; provider: string; model: string; checked_at: string; latency_ms: number }
export interface RestoreRequestResult { backup: BackupInfo; requested_at: string; restart_pending: boolean }
interface Data<T> { data: T }
export async function createBackup() { return (await request<Data<BackupInfo>>('/api/v1/system/backups', { method: 'POST' })).data }
export async function listBackups() { return (await request<Data<BackupInfo[]>>('/api/v1/system/backups')).data }
export async function getDiagnostics() { return (await request<Data<Diagnostics>>('/api/v1/system/diagnostics')).data }
export async function getConsistency() { return (await request<Data<ConsistencyReport>>('/api/v1/system/consistency')).data }
export async function getAIConfiguration() { return (await request<Data<AIConfiguration>>('/api/v1/system/ai-config')).data }
export async function updateAIConfiguration(input: { provider: 'mock' | 'deepseek'; api_key?: string; clear_api_key?: boolean; base_url: string; model: string }) {
  return (await request<Data<AIConfiguration>>('/api/v1/system/ai-config', { method: 'PATCH', body: JSON.stringify(input) })).data
}
export async function testAIConnection() { return (await request<Data<AIConnectionTest>>('/api/v1/system/ai-config/test', { method: 'POST' })).data }
export async function requestBackupRestore(backupName: string, confirmation: string) {
  return (await request<Data<RestoreRequestResult>>('/api/v1/system/backups/restore', { method: 'POST', body: JSON.stringify({ backup_name: backupName, confirmation }) })).data
}
export async function exportLearningData(format: 'json' | 'markdown') {
  const response = await fetch(`/api/v1/system/export?format=${format}`, { method: 'POST' })
  if (!response.ok) throw new Error('学习数据导出失败，请稍后重试。')
  const disposition = response.headers.get('Content-Disposition') ?? ''
  const filename = disposition.match(/filename="?([^";]+)"?/i)?.[1] ?? `learnos-export.${format === 'json' ? 'json' : 'md'}`
  return { blob: await response.blob(), filename }
}
