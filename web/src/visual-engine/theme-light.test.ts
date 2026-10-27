import { describe, expect, it } from 'vitest'
import { parseCssColor, parseFieldGeometry, parseGradientStops, surfaceLightFallback } from './theme-light'

describe('design token parsing', () => {
  it('reads the hex and rgba values the ambient field uses', () => {
    expect(parseCssColor('#3dc1d9')).toEqual({ color: [61 / 255, 193 / 255, 217 / 255], alpha: 1 })
    expect(parseCssColor('rgba(112, 66, 186, 0.3)')).toEqual({ color: [112 / 255, 66 / 255, 186 / 255], alpha: 0.3 })
    expect(parseCssColor('#fff')).toEqual({ color: [1, 1, 1], alpha: 1 })
    expect(parseCssColor('#09051b')).toEqual({ color: [9 / 255, 5 / 255, 27 / 255], alpha: 1 })
  })

  it('rejects values a shader cannot use', () => {
    for (const value of ['', 'transparent', '#12345', 'rgb(1,2)', 'hsl(200 50% 40%)', 'linear-gradient(red, blue)']) {
      expect(parseCssColor(value)).toBeNull()
    }
  })

  it('turns the environment gradient into ordered stops', () => {
    const light = parseGradientStops('linear-gradient(135deg,#e7eff8,#d4e0ee 55%,#e0e6f3)')
    expect(light).toHaveLength(3)
    expect(light![0]).toEqual([231 / 255, 239 / 255, 248 / 255])
    expect(light![1]).toEqual([212 / 255, 224 / 255, 238 / 255])
    expect(parseGradientStops('#09051b')).toBeNull()
  })

  it('shares CSS horizon geometry with WebGL without accepting invalid radii', () => {
    expect(parseFieldGeometry('52% 78%', '46% 42%')).toEqual([.52, .78, .46, .42])
    expect(parseFieldGeometry('-12% 24%', '76% 48%')).toEqual([-.12, .24, .76, .48])
    for (const pair of [['', ''], ['52px 78%', '46% 42%'], ['52% 78%', '0% 42%'], ['NaN% 78%', '46% 42%']]) {
      expect(parseFieldGeometry(pair[0]!, pair[1]!)).toBeNull()
    }
  })

  it('keeps a documented fallback for both themes', () => {
    expect(surfaceLightFallback('light').gradient).toBe(true)
    expect(surfaceLightFallback('dark').gradient).toBe(false)
    expect(surfaceLightFallback('dark').entry.alpha).toBeCloseTo(0.3)
  })
})
