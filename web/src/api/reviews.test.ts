import { describe, expect, it } from 'vitest'
import { reviewCurrentMonth } from './reviews'
describe('review calendar month',()=>{it('uses the documented Shanghai month across UTC and year boundaries',()=>{expect(reviewCurrentMonth(new Date('2025-12-31T15:59:59Z'))).toBe('2025-12');expect(reviewCurrentMonth(new Date('2025-12-31T16:00:00Z'))).toBe('2026-01');expect(reviewCurrentMonth(new Date('2026-01-31T16:00:00Z'))).toBe('2026-02')})})
