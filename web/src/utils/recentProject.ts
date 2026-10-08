const key = 'learnos:last-project'
export function rememberProject(id: number) { try { localStorage.setItem(key, String(id)) } catch { /* storage unavailable */ } }
export function recentProject(ids: number[]): number | null {
  try { const id = Number(localStorage.getItem(key)); if (ids.includes(id)) return id } catch { /* storage unavailable */ }
  return ids[0] ?? null
}
