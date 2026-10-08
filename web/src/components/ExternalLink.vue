<template>
  <span v-if="safe" class="external-link">
    <a :href="safe" :target="safe.toLowerCase().startsWith('obsidian:') ? undefined : '_blank'" rel="noopener noreferrer">{{ name || (safe.toLowerCase().startsWith('obsidian:') ? '打开 Obsidian ↗' : '打开链接 ↗') }}</a>
    <button type="button" aria-label="复制链接" @click="copy">复制链接</button>
  </span>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { ElMessage } from 'element-plus'
import { safeExternalLink } from '@/utils/externalLinks'
const props = defineProps<{ url: string; name?: string }>()
const safe = computed(() => safeExternalLink(props.url))
async function copy() {
  if (!safe.value) return
  try {
    if (navigator.clipboard?.writeText) await navigator.clipboard.writeText(safe.value)
    else {
      const input = document.createElement('textarea')
      input.value = safe.value; input.style.position = 'fixed'; input.style.opacity = '0'; document.body.append(input)
      try { input.select(); if (!document.execCommand('copy')) throw new Error('copy failed') } finally { input.remove() }
    }
    ElMessage.success('链接已复制')
  } catch { ElMessage.error('复制失败，可从链接菜单复制地址。') }
}
</script>
<style scoped>
.external-link { display: inline-flex; gap: 10px; align-items: center; flex-wrap: wrap; font-size: 13px; }
a { color: var(--color-primary); overflow-wrap: anywhere; }
button { padding: 4px 0; border: 0; background: none; color: var(--text-secondary); cursor: pointer; font-size: 12px; }
</style>
