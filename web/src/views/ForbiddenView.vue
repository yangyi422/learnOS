<template>
  <PageShell class="forbidden-page" variant="standard">
    <PageHeader title="没有访问权限" description="这个页面仅对管理员开放。" />
    <div class="forbidden-actions">
      <RouterLink to="/" class="forbidden-link">返回工作台</RouterLink>
    </div>
  </PageShell>
</template>

<script setup lang="ts">
import PageHeader from '@/components/PageHeader.vue'
import PageShell from '@/components/PageShell.vue'
</script>

<style scoped>
.forbidden-page { display: grid; gap: var(--space-4); }
.forbidden-actions { display: flex; gap: 12px; }
.forbidden-link {
  display: inline-flex;
  align-items: center;
  min-height: 36px;
  padding: 0 16px;
  border: 1px solid var(--home-edge);
  border-radius: var(--radius-pill);
  color: var(--home-ink);
  font-size: var(--font-meta);
  text-decoration: none;
}
.forbidden-link:hover { background: var(--home-panel-hover); }
.forbidden-link:focus-visible { outline: 2px solid var(--home-link); outline-offset: 2px; }
</style>
