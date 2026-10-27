<template>
  <div class="fx-depth" :style="style">
    <slot />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

/** Optional card/panel surface enhancement. Never replaces the global card. */
const props = withDefaults(defineProps<{
  background?: number
  edge?: number
  shadow?: number
}>(), { background: 0.7, edge: 0.5, shadow: 0.6 })

const style = computed(() => ({
  '--fx-depth-bg': String(props.background),
  '--fx-depth-edge': String(props.edge),
  '--fx-depth-shadow': String(props.shadow),
}))
</script>
