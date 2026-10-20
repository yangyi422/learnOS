export type WorkspaceContext = 'all' | 'work' | 'life'
export function normalizeWorkspaceContext(value: unknown): WorkspaceContext {
  return value === 'work' || value === 'life' ? value : 'all'
}
export function workspaceContextKey(userID: number) { return `learnos:workspace-context:${userID}` }
export function readWorkspaceContext(userID: number): WorkspaceContext {
  try { return normalizeWorkspaceContext(localStorage.getItem(workspaceContextKey(userID))) } catch { return 'all' }
}
export function saveWorkspaceContext(userID: number, context: WorkspaceContext): boolean {
  try { localStorage.setItem(workspaceContextKey(userID), context); return true } catch { return false }
}
