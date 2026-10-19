<template>
  <header class="section-header">
    <div>
      <p v-if="eyebrow" class="eyebrow">{{ eyebrow }}</p>
      <h2>{{ title }}</h2>
      <p v-if="description" class="section-header__description">{{ description }}</p>
    </div>
    <div v-if="$slots.actions" class="section-header__actions">
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

<style scoped>
.section-header { display: flex; align-items: center; justify-content: space-between; gap: var(--space-6); min-width: 0; }
.section-header > div { min-width: 0; }
h2 { margin: 0; color: var(--text-primary); font-size: var(--font-section-title); font-weight: 700; line-height: 1.4; }
.eyebrow { margin: 0 0 var(--space-2); }
.section-header__description { margin: var(--space-2) 0 0; color: var(--text-secondary); font-size: var(--font-meta); line-height: 1.5; }
.section-header__actions { display: flex; flex-wrap: wrap; align-items: center; justify-content: flex-end; gap: var(--space-2); }
.section-header__actions :deep(.el-button + .el-button) { margin-left: 0; }
@media (max-width: 768px) { .section-header { flex-wrap: wrap; gap: var(--space-3); } .section-header__actions { justify-content: flex-start; } }
</style>
