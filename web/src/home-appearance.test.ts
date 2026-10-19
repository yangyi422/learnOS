import { describe, expect, it } from 'vitest'
import { accentPalette, normalizeAccent } from './home-appearance'

describe('homepage color preferences', () => {
  it('accepts only complete hex colors from storage', () => {
    expect(normalizeAccent('#AbCdEf')).toBe('#abcdef')
    for (const value of [null, {}, '#abc', '#12345z', 'red', 'var(--x)', '#ffffff;display:none']) expect(normalizeAccent(value)).toBe('')
  })
  it('keeps button text readable across bright, dark and intermediate colors', () => {
    for (let r = 0; r <= 255; r += 17) for (let g = 0; g <= 255; g += 17) for (let b = 0; b <= 255; b += 17) {
      const hex = '#' + [r, g, b].map(value => value.toString(16).padStart(2, '0')).join('')
      expect(accentPalette(hex)!.contrast).toBeGreaterThanOrEqual(4.5)
    }
    expect(accentPalette('#ffffff')!.foreground).toBe('#000000')
    expect(accentPalette('#000000')!.foreground).toBe('#ffffff')
    expect(accentPalette('#808080')!.saturation).toBe(0)
  })
})
