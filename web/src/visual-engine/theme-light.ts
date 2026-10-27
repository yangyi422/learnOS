import type { LightSource, RGB, SurfaceLight } from './types'

export type { LightSource, RGB, SurfaceLight } from './types'

const FALLBACK: Record<'light' | 'dark', SurfaceLight> = {
  light: {
    theme: 'light',
    accent: false,
    ground: [[0.906, 0.937, 0.973], [0.831, 0.878, 0.933], [0.878, 0.902, 0.953]],
    gradient: true,
    entry: { color: [0.239, 0.757, 0.851], alpha: 0.25 },
    reflection: { color: [0.545, 0.502, 0.863], alpha: 0.17 },
    floor: { color: [0.416, 0.675, 0.776], alpha: 0.1 },
  },
  dark: {
    theme: 'dark',
    accent: false,
    ground: [[0.035, 0.02, 0.106]],
    gradient: false,
    entry: { color: [0.439, 0.259, 0.729], alpha: 0.3 },
    reflection: { color: [0.361, 0.243, 0.604], alpha: 0.19 },
    floor: { color: [0.4, 0.243, 0.624], alpha: 0.12 },
  },
}

function channel(value: number): number {
  return Math.min(1, Math.max(0, value / 255))
}

/** Accepts #rgb, #rgba, #rrggbb, #rrggbbaa, rgb() and rgba(). */
export function parseCssColor(value: string): LightSource | null {
  const text = value.trim()
  if (!text) return null

  if (text.startsWith('#')) {
    const digits = text.slice(1)
    if (!/^[0-9a-f]+$/i.test(digits)) return null
    if (digits.length === 3 || digits.length === 4) {
      const parts = digits.split('').map((digit) => parseInt(digit + digit, 16))
      return {
        color: [channel(parts[0]!), channel(parts[1]!), channel(parts[2]!)],
        alpha: parts.length === 4 ? parts[3]! / 255 : 1,
      }
    }
    if (digits.length === 6 || digits.length === 8) {
      const parts = digits.match(/.{2}/g)!.map((pair) => parseInt(pair, 16))
      return {
        color: [channel(parts[0]!), channel(parts[1]!), channel(parts[2]!)],
        alpha: parts.length === 4 ? parts[3]! / 255 : 1,
      }
    }
    return null
  }

  const match = /^rgba?\(([^)]+)\)$/i.exec(text)
  if (!match) return null
  const parts = match[1]!.split(/[\s,/]+/).filter(Boolean)
  if (parts.length < 3) return null
  const numeric = parts.slice(0, 3).map((part) => Number(part))
  if (numeric.some((value) => !Number.isFinite(value))) return null
  const alpha = parts.length > 3 ? Number(parts[3]) : 1
  return {
    color: [channel(numeric[0]!), channel(numeric[1]!), channel(numeric[2]!)],
    alpha: Number.isFinite(alpha) ? Math.min(1, Math.max(0, alpha)) : 1,
  }
}

/**
 * `--home-environment` is a gradient string, which a shader cannot consume.
 * Only the colour stops matter, in order.
 */
export function parseGradientStops(value: string): RGB[] | null {
  const colors = value.match(/(#[0-9a-f]{3,8}|rgba?\([^)]*\))/gi) ?? []
  const stops: RGB[] = []
  for (const color of colors) {
    const parsed = parseCssColor(color)
    if (!parsed) continue
    stops.push(parsed.color)
    if (stops.length === 3) break
  }
  return stops.length >= 2 ? stops : null
}

export function surfaceLightFallback(theme: 'light' | 'dark'): SurfaceLight {
  return FALLBACK[theme]
}

/** CSS percentage pairs are also the WebGL field's normalized geometry. */
export function parseFieldGeometry(center: string, size: string): [number, number, number, number] | null {
  const parts = `${center.trim()} ${size.trim()}`.split(/\s+/)
  if (parts.length !== 4 || parts.some((part) => !/^-?\d+(?:\.\d+)?%$/.test(part))) return null
  const values = parts.map((part) => parseFloat(part) / 100)
  if (values[2]! <= 0 || values[3]! <= 0) return null
  return values as [number, number, number, number]
}

/** Reads the live design tokens; never hard-codes a palette in the scene. */
export function readSurfaceLight(element: Element = document.documentElement): SurfaceLight {
  const root = element.ownerDocument?.documentElement ?? element
  const theme = (element.getAttribute('data-theme') ?? root.getAttribute('data-theme')) === 'light' ? 'light' : 'dark'
  const fallback = FALLBACK[theme]
  if (typeof getComputedStyle !== 'function') return fallback

  const style = getComputedStyle(element)
  const read = (name: string) => style.getPropertyValue(name).trim()
  const environment = read('--home-environment')
  const flat = parseCssColor(environment)
  const ground = parseGradientStops(environment) ?? (flat ? [flat.color] : fallback.ground)
  const entryField = parseFieldGeometry(read('--home-entry-center'), read('--home-entry-size'))
  const reflectionField = parseFieldGeometry(read('--home-reflection-center'), read('--home-reflection-size'))
  const floorField = parseFieldGeometry(read('--home-floor-center'), read('--home-floor-size'))

  return {
    theme,
    accent: root.getAttribute('data-home-accent') === 'custom',
    vertical: read('--home-ground-axis') === 'vertical',
    fields: entryField && reflectionField && floorField
      ? { entry: entryField, reflection: reflectionField, floor: floorField }
      : undefined,
    ground,
    gradient: ground.length > 1,
    entry: parseCssColor(read('--home-light-entry')) ?? fallback.entry,
    reflection: parseCssColor(read('--home-light-reflection')) ?? fallback.reflection,
    floor: parseCssColor(read('--home-light-floor')) ?? fallback.floor,
  }
}

/** Theme and palette changes both land as attributes on <html>. */
export function subscribeThemeChanges(listener: () => void): () => void {
  if (typeof MutationObserver === 'undefined' || typeof document === 'undefined') return () => {}
  const observer = new MutationObserver(listener)
  observer.observe(document.documentElement, { attributes: true, attributeFilter: ['data-theme', 'data-home-accent'] })
  return () => observer.disconnect()
}
