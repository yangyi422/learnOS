<template>
  <section class="page-shell" :class="`page-shell--${variant}`">
    <slot name="header">
      <PageHeader v-if="title" :title="title" :description="description" :eyebrow="eyebrow">
        <template v-if="$slots.breadcrumbs" #breadcrumbs><slot name="breadcrumbs" /></template>
        <template v-if="$slots.actions" #actions><slot name="actions" /></template>
      </PageHeader>
    </slot>
    <slot />
  </section>
</template>

<script setup lang="ts">
import PageHeader from './PageHeader.vue'
withDefaults(defineProps<{
  variant?: 'standard' | 'wide' | 'reading' | 'lesson'
  title?: string
  description?: string
  eyebrow?: string
}>(), { variant: 'standard' })
</script>

<style scoped>
.page-shell {
  width: 100%;
  min-width: 0;
  max-width: var(--page-width-standard);
  margin-inline: auto;
}
.page-shell--wide, .page-shell--reading, .page-shell--lesson { max-width: none; }
/* Reading limits belong to article columns, never the directory/header. */
.page-shell--reading { --page-reading-width: var(--width-reading); }
.page-shell--lesson { --page-reading-width: var(--width-lesson); }
</style>
