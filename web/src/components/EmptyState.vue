<template>
  <div class="empty-state">
    <div class="empty-state__mark" aria-hidden="true">◇</div>
    <h3>{{ title }}</h3>
    <p v-if="description">{{ description }}</p>
    <el-button v-if="actionLabel" type="primary" @click="$emit('action')">{{ actionLabel }}</el-button>
    <slot />
  </div>
</template>

<script setup lang="ts">
defineEmits<{ action: [] }>()
defineProps<{ title: string; description?: string; actionLabel?: string }>()
</script>
