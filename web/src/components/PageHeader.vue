<template>
  <header class="page-header">
    <div class="page-header__copy">
      <p v-if="eyebrow" class="eyebrow">{{ eyebrow }}</p>
      <h1 tabindex="-1">{{ title }}</h1>
      <p v-if="description" class="page-header__description">{{ description }}</p>
    </div>
    <div v-if="$slots.actions" class="page-header__actions">
      <slot name="actions" />
    </div>
  </header>
</template>

<script setup lang="ts">
defineProps<{
  title: string
  description?: string
  eyebrow?: string
}>()
</script>
