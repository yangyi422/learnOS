<template>
  <div class="relation-group">
    <h4>{{ title }}</h4>
    <ul v-if="items.length > 0">
      <li v-for="item in items" :key="item.id">{{ item.title }}</li>
    </ul>
    <p v-else class="muted-text">暂无</p>
  </div>
</template>

<script setup lang="ts">
import type { LessonRelationLesson } from '@/types/knowledgeGraph'

defineProps<{
  title: string
  items: LessonRelationLesson[]
}>()
</script>
