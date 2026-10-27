import { describe, expect, it } from 'vitest'
import { DEFAULT_MOTION, MOTIONS, clampDistance, easeVar, findMotion } from './catalog'

describe('motion catalog', () => {
  it('defines six motions with stable ids', () => {
    expect(MOTIONS).toHaveLength(6)
    expect(new Set(MOTIONS.map((motion) => motion.id)).size).toBe(6)
  })

  it('maps easing names to real CSS variables', () => {
    expect(easeVar('standard')).toBe('var(--motion-ease-standard)')
    expect(easeVar('enter')).toBe('var(--motion-ease-enter)')
    expect(easeVar('exit')).toBe('var(--motion-ease-exit)')
  })

  it('clamps distance to the preview range', () => {
    expect(clampDistance(0)).toBe(2)
    expect(clampDistance(99)).toBe(12)
    expect(clampDistance(5.4)).toBe(5)
  })

  it('reflects current parameters in the copyable snippets', () => {
    const reveal = findMotion('content-reveal')
    const usage = reveal.usage({ duration: 300, easing: 'enter', distance: 8 })
    expect(usage.vue).toContain(':duration="300"')
    expect(usage.vue).toContain(':distance="8"')
    const panel = findMotion('panel-presence')
    expect(panel.usage(DEFAULT_MOTION).vue).toContain('<MotionFade')
    expect(panel.usage(DEFAULT_MOTION).vue).toContain(':duration="160"')
  })

  it('only declares parameters that the preview actually uses', () => {
    expect(findMotion('card-hover').params).toEqual(['duration', 'easing'])
    expect(findMotion('button-feedback').params).not.toContain('distance')
    expect(findMotion('content-reveal').params).toContain('distance')
  })
})
