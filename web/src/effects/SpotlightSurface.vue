<template>
  <div ref="root" class="fx-spotlight fx-spotlight-surface" :style="style" @pointermove="handleMove" @pointerleave="handleLeave">
    <slot />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'

/**
 * Convenience wrapper around the `fx-spotlight` CSS. For an existing card, the
 * lighter option is the `v-fx-spotlight` directive (no extra DOM).
 */
const props = withDefaults(defineProps<{
  intensity?: number
  radius?: number
  speed?: number
  tone?: 'neutral' | 'accent'
}>(), { intensity: 0.6, radius: 180, speed: 160, tone: 'neutral' })

const root = ref<HTMLElement | null>(null)
const style = computed(() => ({
  '--fx-intensity': String(props.intensity),
  '--fx-spotlight-radius': `${props.radius}px`,
  '--fx-spotlight-speed': `${props.speed}ms`,
  ...(props.tone === 'accent' ? { '--fx-spotlight-color': 'var(--fx-spotlight-accent)' } : {}),
}))

let frame = 0
let last: PointerEvent | null = null
let active = false

function apply(event: PointerEvent): void {
  const element = root.value
  if (!element) return
  const rect = element.getBoundingClientRect()
  element.style.setProperty('--fx-x', `${Math.round(event.clientX - rect.left)}px`)
  element.style.setProperty('--fx-y', `${Math.round(event.clientY - rect.top)}px`)
}

function handleMove(event: PointerEvent): void {
  if (!active || frame) return
  last = event
  frame = requestAnimationFrame(() => {
    frame = 0
    if (last) apply(last)
  })
}

function handleLeave(): void {
  if (frame) cancelAnimationFrame(frame)
  frame = 0
}

onMounted(() => {
  active = window.matchMedia('(hover: hover) and (pointer: fine)').matches
    && !window.matchMedia('(prefers-reduced-motion: reduce)').matches
})

onBeforeUnmount(() => {
  if (frame) cancelAnimationFrame(frame)
  frame = 0
  last = null
})
</script>

<style scoped>
.fx-spotlight-surface { border-radius: var(--radius-card); }
</style>
