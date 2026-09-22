import { request } from './http'

export interface User {
  id: number
  username: string
  display_name: string
  role: 'admin' | 'user'
  status: 'active' | 'blocked'
  last_login_at?: string
  created_at: string
}

export interface CourseSummary {
  id: number
  name: string
  status: string
  learning_status: string
  mastery_progress: number
  coverage_progress: number
}

export async function login(username: string, password: string): Promise<User> {
  const response = await request<{ data: User }>('/api/v1/auth/login', { method: 'POST', body: JSON.stringify({ username, password }) })
  return response.data
}

export async function currentUser(): Promise<User> {
  const response = await request<{ data: User }>('/api/v1/auth/me')
  return response.data
}

export async function logout(): Promise<void> {
  await request('/api/v1/auth/logout', { method: 'POST' })
}

export async function listUsers(): Promise<User[]> {
  const response = await request<{ data: User[] }>('/api/v1/users')
  return response.data
}

export async function createUser(payload: { username: string; display_name: string; password: string }): Promise<User> {
  const response = await request<{ data: User }>('/api/v1/users', { method: 'POST', body: JSON.stringify(payload) })
  return response.data
}

export async function setUserStatus(id: number, status: User['status']): Promise<void> {
  await request(`/api/v1/users/${id}/status`, { method: 'PATCH', body: JSON.stringify({ status }) })
}

export async function listUserCourses(id: number): Promise<CourseSummary[]> {
  const response = await request<{ data: CourseSummary[] }>(`/api/v1/users/${id}/courses`)
  return response.data
}
