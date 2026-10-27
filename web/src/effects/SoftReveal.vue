<template>
  <MotionReveal
    class="fx-reveal-group"
    :trigger="trigger"
    :duration="duration"
    :delay="delay"
    :distance="distance"
    :ease="easingValue(easing)"
    :enabled="enabled"
    :play-on-mount="playOnMount"
  >
    <slot />
  </MotionReveal>
</template>

<script setup lang="ts">
import MotionReveal from '@/motion/MotionReveal.vue'
import { easingValue } from '@/effects/catalog'

/**
 * Fade + small translate for content that changes (filter results, tab bodies).
 * Animation is opt-in per change and always ends visible, so a failed animation
 * can never leave the content hidden.
 */
const props = withDefaults(defineProps<{
  trigger?: string | number
  enabled?: boolean
  duration?: number
  delay?: number
  distance?: number
  easing?: 'standard' | 'gentle' | 'snappy'
  playOnMount?: boolean
}>(), {
  trigger: 0,
  enabled: true,
  duration: 220,
  delay: 24,
  distance: 6,
  easing: 'standard',
  playOnMount: false,
})

</script>
