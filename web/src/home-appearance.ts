import { ref } from 'vue'

const storageKey = 'learnos-home-accent'
const accent = ref('')
const canPersist = ref(true)
let initialized = false

export const homeColorPresets = [
  { name: '默认', color: '' },
  { name: '冰川蓝', color: '#2864d7' },
  { name: '极光紫', color: '#8054dc' },
  { name: '玫瑰', color: '#be477d' },
  { name: '松石绿', color: '#147e75' },
]

export function normalizeAccent(value: unknown): string {
  return typeof value === 'string' && /^#[0-9a-f]{6}$/i.test(value) ? value.toLowerCase() : ''
}

export function accentPalette(hex: string) {
  const color = normalizeAccent(hex)
  if (!color) return null
  const rgb = [1, 3, 5].map(start => parseInt(color.slice(start, start + 2), 16) / 255)
  const [r, g, b] = rgb as [number, number, number]
  const max = Math.max(...rgb), min = Math.min(...rgb), delta = max - min
  let hue = 0
  if (delta) hue = (max === r ? (g - b) / delta : max === g ? (b - r) / delta + 2 : (r - g) / delta + 4) * 60
  const linear = rgb.map(value => value <= .04045 ? value / 12.92 : ((value + .055) / 1.055) ** 2.4)
  const luminance = linear[0]! * .2126 + linear[1]! * .7152 + linear[2]! * .0722
  const whiteContrast = 1.05 / (luminance + .05), blackContrast = (luminance + .05) / .05
  const lightness = (max + min) / 2
  const saturation = delta === 0 ? 0 : delta / (1 - Math.abs(2 * lightness - 1)) * 100
  return { hue: (hue + 360) % 360, saturation, foreground: whiteContrast >= blackContrast ? '#ffffff' : '#000000', contrast: Math.max(whiteContrast, blackContrast) }
}

function applyAccent() {
  const root = document.documentElement
  const palette = accentPalette(accent.value)
  if (palette) {
    root.dataset.homeAccent = 'custom'
    root.style.setProperty('--home-user-accent', accent.value)
    root.style.setProperty('--home-user-hue', String(palette.hue))
    root.style.setProperty('--home-user-saturation', `${Math.min(palette.saturation, 75)}%`)
    root.style.setProperty('--home-user-foreground', palette.foreground)
  } else {
    delete root.dataset.homeAccent
    for (const key of ['--home-user-accent', '--home-user-hue', '--home-user-saturation', '--home-user-foreground']) root.style.removeProperty(key)
  }
}

export function initializeHomeAppearance() {
  if (initialized) return
  initialized = true
  try { accent.value = normalizeAccent(localStorage.getItem(storageKey)) } catch { canPersist.value = false }
  applyAccent()
  window.addEventListener('storage', event => {
    if (event.key === storageKey || event.key === null) {
      try { accent.value = normalizeAccent(localStorage.getItem(storageKey)) } catch { return }
      applyAccent()
    }
  })
}

export function setHomeAccent(value: string) {
  if (value !== '' && !normalizeAccent(value)) return
  accent.value = normalizeAccent(value)
  try {
    if (accent.value) localStorage.setItem(storageKey, accent.value)
    else localStorage.removeItem(storageKey)
    canPersist.value = true
  } catch { canPersist.value = false }
  applyAccent()
}

export function useHomeAppearance() { return { accent, canPersist } }
