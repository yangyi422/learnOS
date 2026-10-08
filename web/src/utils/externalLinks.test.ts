import { describe, expect, it } from 'vitest'
import { linkedTextParts, safeExternalLink } from './externalLinks'
describe('external link allowlist', () => {
  it('preserves encoded Obsidian URI and web URLs', () => {
    for (const value of ['https://example.com/a?q=hello%20world', 'http://localhost:8080', 'obsidian://open?vault=YY-Wiki&file=Projects%2FLearnOS%2F产品设计.md']) expect(safeExternalLink(value)).toBe(value)
  })
  it('blocks dangerous protocols, unsupported Obsidian actions and malformed arguments', () => {
    for (const value of ['javascript:alert(1)', 'data:text/html,x', 'file:///tmp/x', '//example.com', 'https://u:p@example.com', 'https://x\\y', 'https://x\n', 'obsidian://new?vault=x&file=a', 'obsidian://open?vault=x', 'obsidian://open?vault=x&file=a&file=b', 'obsidian://open?vault=x&file=%ZZ', 'obsidian://open?vault=x&file=%00', 'obsidian://open?vault=x&file=a&append=x']) expect(safeExternalLink(value)).toBeNull()
  })
  it('keeps HTML and unsafe inputs plain text while linking permitted URLs', () => {
    const parts = linkedTextParts('<script>alert(1)</script> javascript:alert(1) https://example.com。')
    expect(parts.filter(part => part.href).map(part => part.href)).toEqual(['https://example.com'])
    expect(parts.map(part => part.text).join('')).toBe('<script>alert(1)</script> javascript:alert(1) https://example.com。')
  })
})
