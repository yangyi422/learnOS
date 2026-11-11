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
/* 看板按自己的列宽需求收窄，与全站统一宽度解耦。 */
.page-shell[data-width='board'] { max-width: var(--width-board); }
.page-shell {
  width: 100%;
  min-width: 0;
  max-width: var(--page-width-standard);
  margin-inline: auto;
}
/* standard 与 wide 取同一个上限：切换路由时页面宽度不再变化。 */
.page-shell--wide { max-width: var(--width-wide); }
.page-shell--reading, .page-shell--lesson { max-width: none; }
/* Reading limits belong to article columns, never the directory/header. */
.page-shell--reading { --page-reading-width: var(--width-reading); }
.page-shell--lesson { --page-reading-width: var(--width-lesson); }
</style>
