<template>
  <div class="empty-state">
    <h3>{{ title }}</h3>
    <p v-if="description">{{ description }}</p>
    <el-button v-if="actionLabel" @click="$emit('action')">{{ actionLabel }}</el-button>
    <slot />
  </div>
</template>

<script setup lang="ts">
defineEmits<{ action: [] }>()
defineProps<{ title: string; description?: string; actionLabel?: string }>()
</script>
