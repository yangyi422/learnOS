import { request } from './http'
import type { DomainInitializationView } from '@/types/domainInitialization'

interface Response { data: DomainInitializationView }
export interface CreateDomainInput { domain_name: string; learning_goal: string; target_depth: 'overview' | 'foundation' | 'systematic' }

export async function createDomainDraft(input: CreateDomainInput) { return (await request<Response>('/api/v1/domains/drafts', { method: 'POST', body: JSON.stringify(input) })).data }
export async function getDomainDraft(id: number) { return (await request<Response>(`/api/v1/domains/drafts/${id}`)).data }
export async function regenerateSkeleton(id: number) { return (await request<Response>(`/api/v1/domains/drafts/${id}/regenerate-skeleton`, { method: 'POST' })).data }
export async function expandStarter(id: number, unitKeys: string[]) { return (await request<Response>(`/api/v1/domains/drafts/${id}/expand-starter`, { method: 'POST', body: JSON.stringify({ unit_keys: unitKeys }) })).data }
export async function generateInitialWorld(id: number) { return (await request<Response>(`/api/v1/domains/drafts/${id}/generate-initial-world`, { method: 'POST' })).data }
export async function applyDomainDraft(id: number) { return (await request<Response>(`/api/v1/domains/drafts/${id}/apply`, { method: 'POST' })).data }
