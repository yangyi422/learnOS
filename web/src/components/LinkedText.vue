<template><p class="linked-text"><template v-for="(part, index) in parts" :key="index"><ExternalLink v-if="part.href?.toLowerCase().startsWith('obsidian:')" :url="part.href" :name="part.text" /><a v-else-if="part.href" :href="part.href" :target="part.href.toLowerCase().startsWith('obsidian:') ? undefined : '_blank'" rel="noopener noreferrer">{{ part.text }}</a><template v-else>{{ part.text }}</template></template></p></template>
<script setup lang="ts">
import { computed } from 'vue'
import ExternalLink from '@/components/ExternalLink.vue'
import { linkedTextParts } from '@/utils/externalLinks'
const props = defineProps<{ content: string }>()
const parts = computed(() => linkedTextParts(props.content))
</script>
<style scoped>
.linked-text { margin: 0; white-space: pre-wrap; overflow-wrap: anywhere; line-height: 1.8; color: var(--text-primary); }
a { color: var(--color-primary); }
</style>
