import { ref } from 'vue'

export type ThemePreference = 'system' | 'light' | 'dark'
type ResolvedTheme = 'light' | 'dark'

const storageKey = 'learnos-theme'
const preference = ref<ThemePreference>('system')
let media: MediaQueryList | null = null

function savedPreference(): ThemePreference {
  try {
    const value = window.localStorage.getItem(storageKey)
    return value === 'light' || value === 'dark' || value === 'system' ? value : 'system'
  } catch {
    return 'system'
  }
}

function resolvedTheme(): ResolvedTheme {
  return preference.value === 'system' ? (media?.matches ? 'dark' : 'light') : preference.value
}

function applyTheme() {
  const theme = resolvedTheme()
  document.documentElement.dataset.theme = theme
  document.documentElement.style.colorScheme = theme
  document.querySelector('meta[name="theme-color"]')?.setAttribute('content', theme === 'dark' ? '#0b121f' : '#f4f7fa')
}

export function initializeTheme() {
  preference.value = savedPreference()
  media = window.matchMedia('(prefers-color-scheme: dark)')
  media.addEventListener('change', applyTheme)
  applyTheme()
}

export function useTheme() {
  return { preference }
}

export function setThemePreference(value: ThemePreference) {
  preference.value = value
  try { window.localStorage.setItem(storageKey, value) } catch { /* storage may be disabled */ }
  applyTheme()
}
