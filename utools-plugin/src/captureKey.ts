// 8-80 characters of [A-Za-z0-9_-], frozen by the backend contract.
const KEY_PATTERN = /^[A-Za-z0-9_-]{8,80}$/

/**
 * Creates one idempotency key per user submission. Retries must reuse it, so it
 * is only replaced when the draft content changes or a new submission starts.
 */
export function createCaptureKey(): string {
  const uuid = globalThis.crypto?.randomUUID?.()
  if (uuid) return `cap-${uuid.replace(/-/g, '')}`
  const stamp = Date.now().toString(36)
  const noise = `${Math.random().toString(36).slice(2, 10)}${Math.random().toString(36).slice(2, 6)}`
  return `cap-${stamp}${noise}`.slice(0, 48)
}

export function isValidCaptureKey(key: string): boolean {
  return KEY_PATTERN.test(key)
}
