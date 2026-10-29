<template>
  <header class="page-header">
    <div v-if="$slots.breadcrumbs" class="page-header__breadcrumbs"><slot name="breadcrumbs" /></div>
    <p v-if="eyebrow" class="eyebrow">{{ eyebrow }}</p>
    <div class="page-header__row">
    <div class="page-header__copy">
      <h1 tabindex="-1">{{ title }}</h1>
      <p v-if="description" class="page-header__description">{{ description }}</p>
    </div>
    <div v-if="$slots.actions" class="page-header__actions">
      <slot name="actions" />
    </div>
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

<style scoped>
.page-header { display: block; margin-bottom: var(--page-header-gap); }
.page-header__breadcrumbs { margin-bottom: var(--space-4); }
.page-header__row { display: flex; justify-content: space-between; align-items: flex-start; gap: var(--space-6); }
.page-header__copy { min-width: 0; }
.page-header h1 { margin: 0; font-size: var(--font-page-title); font-family: var(--font-display); font-weight: 500; line-height: 1.35; letter-spacing: -.02em; overflow-wrap: anywhere; }
.page-header .eyebrow { margin: 0 0 var(--space-2); }
.page-header__description { max-width: 64ch; margin: var(--space-2) 0 0; font-size: var(--font-body); line-height: 1.6; color: var(--color-text-secondary); }
.page-header__actions :deep(.el-button + .el-button) { margin-left: 0; }
.page-header__actions { min-width: 0; flex-shrink: 0; max-width: 55%; display: flex; flex-wrap: wrap; align-items: center; justify-content: flex-end; gap: var(--space-2); }
@media (max-width: 820px) {
  .page-header__row { flex-wrap: wrap; gap: var(--space-4); }
  .page-header__actions { width: 100%; max-width: 100%; justify-content: flex-start; flex-shrink: 1; }
}
</style>
