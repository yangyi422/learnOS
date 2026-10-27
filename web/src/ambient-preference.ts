import { ref } from 'vue'

export type AmbientPreference = 'auto' | 'css' | 'webgl'

const storageKey = 'learnos-ambient'
const preference = ref<AmbientPreference>('css')
let initialized = false

/** Anything unexpected falls back to the safe default: the CSS light field. */
export function normalizeAmbientPreference(value: unknown): AmbientPreference {
  return value === 'auto' || value === 'webgl' || value === 'css' ? value : 'css'
}

function load(): void {
  if (initialized) return
  initialized = true
  try {
    preference.value = normalizeAmbientPreference(window.localStorage.getItem(storageKey))
  } catch {
    // Storage may be disabled; the in-memory default still applies.
  }
  window.addEventListener('storage', (event) => {
    if (event.key !== storageKey && event.key !== null) return
    try {
      preference.value = normalizeAmbientPreference(window.localStorage.getItem(storageKey))
    } catch {
      // ignore: keep the current preference
    }
  })
}

export function useAmbientPreference(): { preference: typeof preference } {
  load()
  return { preference }
}

export function setAmbientPreference(value: AmbientPreference): void {
  preference.value = normalizeAmbientPreference(value)
  try {
    window.localStorage.setItem(storageKey, preference.value)
  } catch {
    // Storage disabled: the choice still applies for this session.
  }
}
