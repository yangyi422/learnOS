export interface WebGL2Capability {
  supported: boolean
  reason?: 'no-canvas' | 'no-context' | 'blocked'
}

export interface RenderCapabilities extends WebGL2Capability {
  /** EXT_disjoint_timer_query_webgl2, probed on a scratch context. */
  timerQuery: boolean
  reducedMotion: boolean
}

const CONTEXT_ATTRIBUTES: WebGLContextAttributes = {
  alpha: false,
  antialias: false,
  depth: false,
  stencil: false,
  powerPreference: 'low-power',
}

const REDUCED_MOTION_QUERY = '(prefers-reduced-motion: reduce)'

/**
 * Probes WebGL2 without keeping a context alive: the scratch canvas is dropped
 * and its context released before returning, so probing costs no GPU memory.
 */
export function detectWebGL2(): WebGL2Capability {
  if (typeof document === 'undefined') return { supported: false, reason: 'no-canvas' }
  try {
    const canvas = document.createElement('canvas')
    const gl = canvas.getContext('webgl2', CONTEXT_ATTRIBUTES)
    if (!gl) return { supported: false, reason: 'no-context' }
    gl.getExtension('WEBGL_lose_context')?.loseContext()
    return { supported: true }
  } catch {
    return { supported: false, reason: 'blocked' }
  }
}

/** GPU timing is optional; a missing extension is a normal outcome, not an error. */
export function detectTimerQuery(canvas?: HTMLCanvasElement): boolean {
  if (typeof document === 'undefined') return false
  try {
    const target = canvas ?? document.createElement('canvas')
    const gl = target.getContext('webgl2', CONTEXT_ATTRIBUTES)
    if (!gl) return false
    const available = Boolean(gl.getExtension('EXT_disjoint_timer_query_webgl2'))
    if (!canvas) gl.getExtension('WEBGL_lose_context')?.loseContext()
    return available
  } catch {
    return false
  }
}

export function detectCapabilities(): RenderCapabilities {
  const webgl2 = detectWebGL2()
  return {
    ...webgl2,
    timerQuery: webgl2.supported && detectTimerQuery(),
    reducedMotion: prefersReducedMotion(),
  }
}

export function prefersReducedMotion(): boolean {
  return typeof window !== 'undefined'
    && typeof window.matchMedia === 'function'
    && window.matchMedia(REDUCED_MOTION_QUERY).matches
}

export function subscribeReducedMotion(listener: (reduced: boolean) => void): () => void {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') return () => {}
  const media = window.matchMedia(REDUCED_MOTION_QUERY)
  const handler = () => listener(media.matches)
  media.addEventListener('change', handler)
  return () => media.removeEventListener('change', handler)
}
