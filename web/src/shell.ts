// The desktop shell has no tabs: `target="_blank"` there is handed to the
// operating system and opens the user's browser, which leaves the app. The shell
// marks its entry URL with `?shell=desktop`; remember it for the session so the
// marker survives full-page redirects (for example the 401 → /login hop).
const STORAGE_KEY = 'learnos-shell'
export type Shell = 'web' | 'desktop'

export function initializeShell(): void {
  const requested = new URLSearchParams(window.location.search).get('shell')
  if (requested !== 'desktop' && requested !== 'web') return
  try { window.sessionStorage.setItem(STORAGE_KEY, requested) } catch { /* storage may be disabled */ }
}

export function currentShell(): Shell {
  try {
    const stored = window.sessionStorage.getItem(STORAGE_KEY)
    if (stored === 'desktop' || stored === 'web') return stored
  } catch { /* fall through to the web default */ }
  return 'web'
}

export function isDesktopShell(): boolean {
  return currentShell() === 'desktop'
}

/** External links stay in the shell: same-tab navigation renders them in the app. */
export function linkTarget(): '_blank' | undefined {
  return isDesktopShell() ? undefined : '_blank'
}
