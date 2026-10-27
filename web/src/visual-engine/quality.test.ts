import { describe, expect, it } from 'vitest'
import { FrameSampler, targetPixelRatio } from './quality'

describe('render quality', () => {
  it('caps the device pixel ratio at 1.5 and never goes below 1', () => {
    expect(targetPixelRatio(2, 'high')).toBe(1.5)
    expect(targetPixelRatio(3, 'high')).toBe(1.5)
    expect(targetPixelRatio(1.25, 'high')).toBe(1.25)
    expect(targetPixelRatio(2, 'balanced')).toBe(1)
    expect(targetPixelRatio(2, 'static')).toBe(1)
    expect(targetPixelRatio(0, 'high')).toBe(1)
    expect(targetPixelRatio(Number.NaN, 'high')).toBe(1)
    expect(targetPixelRatio(3, 'high', 2)).toBe(2)
  })
})

describe('frame sampler', () => {
  it('reports once per closed one second window', () => {
    const sampler = new FrameSampler()
    expect(sampler.sample(0)).toBeNull()
    expect(sampler.sample(500)).toBeNull()
    expect(sampler.sample(1000)).toBe(3)
    expect(sampler.value).toBe(3)
    expect(sampler.sample(1500)).toBeNull()
    expect(sampler.sample(2000)).toBe(2)
  })

  it('resets the window when restarted', () => {
    const sampler = new FrameSampler()
    sampler.sample(0)
    sampler.sample(1200)
    expect(sampler.value).not.toBeNull()
    sampler.reset()
    expect(sampler.value).toBeNull()
    expect(sampler.sample(5000)).toBeNull()
  })
})
