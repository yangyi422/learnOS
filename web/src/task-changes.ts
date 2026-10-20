const eventName = 'learnos:tasks-changed'
// Only an invalidation token crosses tabs; task data and identities stay on the API.
export function notifyTaskChange() {
  if (typeof window === 'undefined') return
  window.dispatchEvent(new Event(eventName))
  try { localStorage.setItem(eventName, `${Date.now()}:${Math.random()}`) } catch { /* same-tab notification still works */ }
}
export function subscribeTaskChanges(refresh: () => void): () => void {
  if (typeof window === 'undefined') return () => {}
  const storage = (event: StorageEvent) => { if (event.key === eventName) refresh() }
  window.addEventListener(eventName, refresh)
  window.addEventListener('storage', storage)
  return () => { window.removeEventListener(eventName, refresh); window.removeEventListener('storage', storage) }
}
