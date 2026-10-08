// Keep this allowlist aligned with internal/links/external.go.
export function safeExternalLink(value: string): string | null {
  if (!/^(?:https?|obsidian):\/\//i.test(value) || value.length > 4096 || /[\s\u0000-\u001f\u007f\\]/u.test(value)) return null
  try {
    const url = new URL(value)
    if (url.username || url.password) return null
    if (url.protocol === 'http:' || url.protocol === 'https:') return url.hostname ? value : null
    if (url.protocol !== 'obsidian:' || url.host !== 'open' || !['', '/'].includes(url.pathname) || url.hash) return null
    if (!url.searchParams.get('vault') || !url.searchParams.get('file')) return null
    for (const [key, entry] of url.searchParams) {
      if (!['vault', 'file'].includes(key) || url.searchParams.getAll(key).length !== 1 || /[\u0000-\u001f\u007f]/u.test(entry)) return null
    }
    // URLSearchParams tolerates malformed percent escapes, which the API rejects.
    decodeURIComponent(url.search)
    return value
  } catch { return null }
}

export function linkedTextParts(text: string): { text: string; href: string | null }[] {
  const result: { text: string; href: string | null }[] = []
  const regex = /(?:https?:\/\/|obsidian:\/\/)[^\s<>"']+/gi
  let start = 0
  for (const match of text.matchAll(regex)) {
    const index = match.index!
    if (index > start) result.push({ text: text.slice(start, index), href: null })
    const value = match[0].replace(/[。，、！？）),.;!?]+$/u, '')
    result.push({ text: value, href: safeExternalLink(value) })
    start = index + value.length
  }
  if (start < text.length) result.push({ text: text.slice(start), href: null })
  return result
}
