import { describe, expect, it } from 'vitest'
import { normalizeAmbientPreference } from './ambient-preference'

describe('ambient preference', () => {
  it('keeps the three explicit choices', () => {
    for (const value of ['auto', 'css', 'webgl'] as const) expect(normalizeAmbientPreference(value)).toBe(value)
  })

  it('falls back to CSS for anything else', () => {
    for (const value of [null, undefined, '', 'WEBGL', 'true', 'system', 1, {}]) {
      expect(normalizeAmbientPreference(value)).toBe('css')
    }
  })
})
