import { nextTick, watch } from 'vue'
import { onBeforeRouteLeave, useRoute } from 'vue-router'

// Only in-memory positions; no private query or content is persisted to storage.
const positions = new Map<string, number>()
export function useArchiveScroll() {
  const route = useRoute()
  let pending = true
  const save = (key: string) => { positions.set(key, document.querySelector('.main-content')?.scrollTop ?? 0) }
  watch(() => route.fullPath, (_, previous) => { save(previous); pending = true }, { flush: 'sync' })
  onBeforeRouteLeave(() => save(route.fullPath))
  return async () => {
    if (!pending) return
    const key = route.fullPath
    await nextTick()
    if (key !== route.fullPath) return
    pending = false
    document.querySelector('.main-content')?.scrollTo({ top: positions.get(key) ?? 0 })
  }
}
