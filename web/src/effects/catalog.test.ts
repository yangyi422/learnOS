import { describe, expect, it } from 'vitest'
import { EFFECTS, defaultsOf, findEffect, normalizeParams, resolveEffectState } from './catalog'

const env = { supported: true, reducedMotion: false }

describe('effects catalog', () => {
  it('defines four effects with unique ids and typed params', () => {
    expect(EFFECTS).toHaveLength(4)
    expect(new Set(EFFECTS.map((effect) => effect.id)).size).toBe(4)
    for (const effect of EFFECTS) {
      expect(effect.params.length).toBeGreaterThan(0)
      for (const param of effect.params) {
        if (param.control === 'range') {
          expect(typeof param.default).toBe('number')
          expect(param.min).toBeLessThan(param.max!)
        }
        if (param.control === 'select') expect(param.options?.some((option) => option.value === param.default)).toBe(true)
      }
    }
  })

  it('computes defaults from the parameter definitions', () => {
    const spotlight = findEffect('spotlight-surface')
    expect(defaultsOf(spotlight)).toEqual({ intensity: 0.6, radius: 180, speed: 160, tone: 'neutral' })
  })

  it('clamps ranges and rejects unknown select values', () => {
    const spotlight = findEffect('spotlight-surface')
    expect(normalizeParams(spotlight, { intensity: 5, radius: 10, speed: 999, tone: 'rainbow' })).toEqual({
      intensity: 1, radius: 80, speed: 300, tone: 'neutral',
    })
  })

  it('keeps valid values and coerces booleans', () => {
    const ambient = findEffect('ambient-gradient')
    expect(normalizeParams(ambient, { strength: 0.5, quality: 'balanced', animate: false })).toEqual({ strength: 0.5, quality: 'balanced', animate: false })
    expect(normalizeParams(ambient, { animate: 'yes' as never }).animate).toBe(true)
  })

  it('stops ambient when animation is off', () => {
    const ambient = findEffect('ambient-gradient')
    expect(resolveEffectState(ambient, { ...defaultsOf(ambient), animate: false }, env)).toMatchObject({ shouldRun: false, cssFallback: true, reason: 'disabled' })
  })

  it('lets reduced motion win over CSS and WebGL effects', () => {
    const ambient = findEffect('ambient-gradient')
    const spotlight = findEffect('spotlight-surface')
    expect(resolveEffectState(ambient, defaultsOf(ambient), { supported: true, reducedMotion: true })).toMatchObject({ shouldRun: false, cssFallback: true, reason: 'reduced-motion' })
    expect(resolveEffectState(spotlight, defaultsOf(spotlight), { supported: true, reducedMotion: true })).toMatchObject({ shouldRun: false, cssFallback: false, reason: 'reduced-motion' })
  })

  it('falls back to CSS when WebGL is unavailable', () => {
    const ambient = findEffect('ambient-gradient')
    expect(resolveEffectState(ambient, defaultsOf(ambient), { supported: false, reducedMotion: false })).toMatchObject({ shouldRun: false, cssFallback: true, reason: 'unsupported' })
  })

  it('returns interface-accurate usage snippets', () => {
    const spotlight = findEffect('spotlight-surface')
    const usage = spotlight.usage({ intensity: 0.8, radius: 220, speed: 140, tone: 'accent' })
    expect(usage).toContain(':intensity="0.8"')
    expect(usage).toContain(':radius="220"')
    expect(usage).toContain('tone="accent"')
    const reveal = findEffect('soft-reveal')
    expect(reveal.usage(defaultsOf(reveal))).toContain(':duration="220"')
    const depth = findEffect('surface-depth')
    expect(depth.usage(defaultsOf(depth))).toContain(':background="0.7"')
    const ambient = findEffect('ambient-gradient')
    expect(ambient.usage(defaultsOf(ambient))).toContain('quality="high"')
  })
})
