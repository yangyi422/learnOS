import type { Directive } from 'vue'

/**
 * `v-fx-spotlight="intensity"` — tracks the pointer and exposes `--fx-x/--fx-y`
 * for the CSS highlight. Numeric values activate it; `undefined` removes both the
 * class and the listener, so a disabled experiment costs nothing.
 */
interface SpotlightState {
  move: (event: PointerEvent) => void
  frame: number
  active: boolean
}

const states = new WeakMap<HTMLElement, SpotlightState>()

function pointerAllowsTracking(): boolean {
  return window.matchMedia('(hover: hover) and (pointer: fine)').matches
    && !window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

function activate(element: HTMLElement, state: SpotlightState, intensity: number): void {
  element.style.setProperty('--fx-intensity', String(intensity))
  if (state.active || !pointerAllowsTracking()) return
  element.classList.add('fx-spotlight')
  element.addEventListener('pointermove', state.move, { passive: true })
  state.active = true
}

function deactivate(element: HTMLElement, state: SpotlightState): void {
  if (!state.active) return
  element.removeEventListener('pointermove', state.move)
  if (state.frame) cancelAnimationFrame(state.frame)
  state.frame = 0
  element.classList.remove('fx-spotlight')
  element.style.removeProperty('--fx-x')
  element.style.removeProperty('--fx-y')
  state.active = false
}

export const fxSpotlight: Directive<HTMLElement, number | undefined> = {
  mounted(element, binding) {
    const state: SpotlightState = { frame: 0, active: false, move: () => {} }
    state.move = (event: PointerEvent) => {
      if (state.frame) return
      state.frame = requestAnimationFrame(() => {
        state.frame = 0
        const rect = element.getBoundingClientRect()
        element.style.setProperty('--fx-x', `${Math.round(event.clientX - rect.left)}px`)
        element.style.setProperty('--fx-y', `${Math.round(event.clientY - rect.top)}px`)
      })
    }
    states.set(element, state)
    if (typeof binding.value === 'number') activate(element, state, binding.value)
  },
  updated(element, binding) {
    const state = states.get(element)
    if (!state) return
    if (typeof binding.value === 'number') activate(element, state, binding.value)
    else deactivate(element, state)
  },
  unmounted(element) {
    const state = states.get(element)
    if (state) deactivate(element, state)
    states.delete(element)
  },
}
