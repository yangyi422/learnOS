<template>
  <div class="relation-group">
    <h4>{{ title }}</h4>
    <ul v-if="items.length > 0">
      <li v-for="item in items" :key="item.id">
        <button v-if="clickable" type="button" @click="$emit('select', item)">{{ item.title }} <span aria-hidden="true">→</span></button>
        <span v-else>{{ item.title }}</span>
      </li>
    </ul>
    <p v-else class="muted-text">暂无</p>
  </div>
</template>

<script setup lang="ts">
import type { LessonRelationLesson } from '@/types/knowledgeGraph'

withDefaults(defineProps<{
  title: string
  items: LessonRelationLesson[]
  clickable?: boolean
}>(), { clickable: false })

defineEmits<{ select: [item: LessonRelationLesson] }>()
</script>
