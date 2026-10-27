import { computed, reactive, ref } from 'vue'
import { currentUser } from '@/api/auth'

/**
 * Admin-only, session-scoped experiment state for the 11.4D pilots.
 *
 * Authorization is resolved from the real `/auth/me` role; the stored settings
 * are only read after authorization succeeds. A non-admin who writes the storage
 * key gets nothing: `authorized` stays false and every effect stays off.
 */
export type PilotPage = 'home' | 'explore' | 'projects'

export interface PilotSettings {
  home: { enabled: boolean; intensity: number; refraction: boolean }
  explore: { enabled: boolean; intensity: number; reveal: boolean }
  projects: { cards: boolean; tabs: boolean; drawer: boolean }
}

const STORAGE_KEY = 'learnos:fx-pilot'

const DEFAULTS: PilotSettings = {
  home: { enabled: false, intensity: 0.35, refraction: false },
  explore: { enabled: false, intensity: 0.3, reveal: true },
  projects: { cards: false, tabs: false, drawer: false },
}

const authorized = ref(false)
const settings = reactive<PilotSettings>(clone(DEFAULTS))

function clone(source: PilotSettings): PilotSettings {
  return {
    home: { ...source.home },
    explore: { ...source.explore },
    projects: { ...source.projects },
  }
}

function clamp(value: unknown, fallback: number): number {
  const number = typeof value === 'number' && Number.isFinite(value) ? value : fallback
  return Math.min(1, Math.max(0.1, number))
}

function load(): void {
  if (!authorized.value) return
  try {
    const raw = window.sessionStorage.getItem(STORAGE_KEY)
    if (!raw) return
    const parsed = JSON.parse(raw) as Partial<PilotSettings>
    if (parsed.home) {
      settings.home.enabled = parsed.home.enabled === true
      settings.home.intensity = clamp(parsed.home.intensity, DEFAULTS.home.intensity)
      settings.home.refraction = parsed.home.refraction === true
    }
    if (parsed.explore) {
      settings.explore.enabled = parsed.explore.enabled === true
      settings.explore.intensity = clamp(parsed.explore.intensity, DEFAULTS.explore.intensity)
      settings.explore.reveal = parsed.explore.reveal !== false
    }
    if (parsed.projects) {
      settings.projects.cards = parsed.projects.cards === true
      settings.projects.tabs = parsed.projects.tabs === true
      settings.projects.drawer = parsed.projects.drawer === true
    }
  } catch {
    // ignore corrupt storage; defaults stay in place
  }
}

function save(): void {
  if (!authorized.value) return
  try {
    window.sessionStorage.setItem(STORAGE_KEY, JSON.stringify(settings))
  } catch {
    // session-only fallback: keep the in-memory values
  }
}

export function authorizePilot(): void {
  if (authorized.value) return
  authorized.value = true
  load()
}

/** Resolves the current role once; non-admins never become authorized. */
export async function ensurePilotAuthorization(): Promise<boolean> {
  if (authorized.value) return true
  try {
    const user = await currentUser()
    if (user.role !== 'admin') return false
    authorizePilot()
    return true
  } catch {
    return false
  }
}

export function setPilot(page: PilotPage, patch: Partial<PilotSettings[PilotPage]>): void {
  if (!authorized.value) return
  Object.assign(settings[page], patch)
  save()
}

export function resetPilot(): void {
  if (!authorized.value) return
  Object.assign(settings.home, DEFAULTS.home)
  Object.assign(settings.explore, DEFAULTS.explore)
  save()
}

export function usePilot() {
  return {
    authorized,
    settings,
    homeActive: computed(() => authorized.value && settings.home.enabled),
    homeRefraction: computed(() => authorized.value && settings.home.refraction),
    exploreActive: computed(() => authorized.value && settings.explore.enabled),
    revealActive: computed(() => authorized.value && settings.explore.enabled && settings.explore.reveal),
    projectsCards: computed(() => authorized.value && settings.projects.cards),
    projectsTabs: computed(() => authorized.value && settings.projects.tabs),
    projectsDrawer: computed(() => authorized.value && settings.projects.drawer),
  }
}
