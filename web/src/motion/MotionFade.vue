<template>
  <div class="motion-fade-host" :style="style">
    <Transition name="motion-fade" @after-leave="emit('after-leave')">
      <div v-if="show" class="motion-fade-content">
        <slot />
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

/**
 * Fade for small show/hide regions. The element is removed only after the leave
 * transition finishes; with reduced motion the token collapses and it is instant.
 */
const props = withDefaults(defineProps<{
  show: boolean
  duration?: number
  delay?: number
}>(), {
  duration: 160,
  delay: 0,
})

const emit = defineEmits<{ 'after-leave': [] }>()

const style = computed(() => ({
  '--motion-duration': `${props.duration}ms`,
  '--motion-delay': `${props.delay}ms`,
}))
</script>
