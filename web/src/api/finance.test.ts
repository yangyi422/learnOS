import { describe, expect, it } from 'vitest'
import { financeMoney, financeShare, monthIncomplete } from './finance'
describe('financial exact values and local calendar', () => {
  it('formats minor units without floating point loss', () => {
    expect(financeMoney('10')).toBe('0.10')
    expect(financeMoney('-1')).toBe('-0.01')
    expect(financeMoney('9999999999999999')).toBe('99,999,999,999,999.99')
    expect(financeShare('1', '3')).toBe('33.3%')
    expect(financeShare('1', '0')).toBe('—')
  })
  it('marks incomplete months using local date fields', () => {
    const now = new Date(2026, 9, 1, 0, 1)
    expect(monthIncomplete('2026-10', now)).toBe(true)
    expect(monthIncomplete('2026-09', now)).toBe(false)
    expect(monthIncomplete('2026-11', now)).toBe(true)
  })
})
