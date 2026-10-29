import { request } from './http'

export interface APITokenView {
  id: number
  name: string
  prefix: string
  created_at: string
  last_used_at: string | null
  expires_at: string
  revoked_at: string | null
  expired: boolean
}

export interface CreatedAPIToken extends APITokenView {
  /** The secret is present exactly once, in the create response. */
  token: string
}

interface Data<T> { data: T }

export async function listAPITokens(): Promise<APITokenView[]> {
  return (await request<Data<APITokenView[]>>('/api/v1/api-tokens')).data
}

export async function createAPIToken(input: { name: string; expires_in_days: number }): Promise<CreatedAPIToken> {
  return (await request<Data<CreatedAPIToken>>('/api/v1/api-tokens', { method: 'POST', body: JSON.stringify(input) })).data
}

export async function revokeAPIToken(id: number): Promise<void> {
  await request<{ data: { revoked: boolean } }>(`/api/v1/api-tokens/${id}`, { method: 'DELETE' })
}
