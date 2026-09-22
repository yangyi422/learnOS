<template>
  <div class="progress-bar">
    <div v-if="label" class="progress-bar__label"><span>{{ label }}</span><strong>{{ Math.round(percentage) }}%</strong></div>
    <el-progress :percentage="percentage" :stroke-width="strokeWidth" :show-text="false" />
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{ percentage: number; label?: string; strokeWidth?: number }>(), { strokeWidth: 6 })
</script>
