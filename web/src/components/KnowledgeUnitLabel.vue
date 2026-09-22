<template>
  <div class="knowledge-unit-label">{{ data.title }}</div>
</template>

<script setup lang="ts">
defineProps<{ data: { title: string } }>()
</script>
