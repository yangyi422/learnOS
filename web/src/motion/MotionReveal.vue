<template>
  <div class="motion-reveal" :class="{ 'is-entering': entering }" :style="style">
    <slot />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'

/**
 * Entrance for a block or a small list. Children stay visible by default, so a
 * missed animation never hides content. `trigger` replays on content changes;
 * `playOnMount` is for deliberate entrances, not every page load.
 */
const props = withDefaults(defineProps<{
  trigger?: string | number
  duration?: number
  delay?: number
  distance?: number
  ease?: string
  enabled?: boolean
  playOnMount?: boolean
}>(), {
  trigger: 0,
  duration: 180,
  delay: 0,
  distance: 4,
  ease: 'var(--motion-ease-enter)',
  enabled: true,
  playOnMount: false,
})

const entering = ref(false)
let timer: ReturnType<typeof setTimeout> | undefined

const style = computed(() => ({
  '--motion-duration': `${props.duration}ms`,
  '--motion-stagger': `${props.delay}ms`,
  '--motion-distance': `${props.distance}px`,
  '--motion-ease': props.ease,
}))

function play(): void {
  if (!props.enabled) return
  if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return
  entering.value = false
  requestAnimationFrame(() => {
    entering.value = true
    clearTimeout(timer)
    timer = setTimeout(() => { entering.value = false }, props.duration + props.delay * 6 + 80)
  })
}

watch(() => props.trigger, () => play())
watch(() => props.enabled, (value) => { if (!value) entering.value = false })
if (props.playOnMount) play()

onBeforeUnmount(() => clearTimeout(timer))
</script>
