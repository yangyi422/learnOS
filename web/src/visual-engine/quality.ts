import type { QualityTier } from './types'

export const DEFAULT_PIXEL_RATIO_CAP = 1.5

/**
 * Device pixel ratio is capped per tier. `balanced` trades sharpness for fill
 * rate on integrated GPUs; `static` renders a single frame at 1x.
 */
export function targetPixelRatio(device: number, tier: QualityTier, cap = DEFAULT_PIXEL_RATIO_CAP): number {
  const ratio = Number.isFinite(device) && device > 0 ? device : 1
  const tierCap = tier === 'balanced' ? 1 : tier === 'static' ? 1 : cap
  return Math.min(ratio, Math.max(1, tierCap))
}

/**
 * Counts frames over a one second window and only reports once the window has
 * closed, so the caller can update UI at 1 Hz instead of every frame.
 */
export class FrameSampler {
  private windowStart: number | null = null
  private frames = 0
  private fps: number | null = null

  get value(): number | null {
    return this.fps
  }

  reset(): void {
    this.windowStart = null
    this.frames = 0
    this.fps = null
  }

  sample(now: number): number | null {
    if (this.windowStart === null) this.windowStart = now
    this.frames += 1
    const elapsed = now - this.windowStart
    if (elapsed < 1000) return null
    this.fps = Math.round((this.frames / elapsed) * 1000 * 10) / 10
    this.windowStart = now
    this.frames = 0
    return this.fps
  }
}
