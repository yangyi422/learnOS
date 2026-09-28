<template>
  <section class="current-focus" aria-labelledby="current-focus-title">
    <div class="current-focus__eyebrow">{{ lesson ? '继续学习' : '下一步' }}</div>
    <div class="current-focus__body">
      <div class="current-focus__copy">
        <span class="current-focus__domain">{{ course.name }}</span>
        <h2 id="current-focus-title">{{ lesson?.lesson.title || course.current_unit || `打开${course.name}的知识结构` }}</h2>
        <p v-if="lesson">{{ lesson.lesson.core_question }}</p>
        <p v-else>{{ course.goal || course.description || '查看领域地图，选择第一个想深入的问题。' }}</p>
        <div class="current-focus__meta">
          <span>{{ lesson?.unit.title || '尚无当前学习节点' }}</span>
          <span v-if="course.last_studied_at">最近学习：{{ relativeDate(course.last_studied_at) }}</span>
          <StatusTag v-if="cognitive" :status="cognitive.current_level" />
          <StatusTag v-if="cognitive" :status="cognitive.status" effect="plain" />
        </div>
      </div>
      <el-button type="primary" class="current-focus__action" :disabled="loading" @click="$emit('continue')">
        {{ lesson ? '继续学习' : '打开知识结构' }} <span aria-hidden="true">→</span>
      </el-button>
    </div>
  </section>
</template>

<script setup lang="ts">
import StatusTag from '@/components/StatusTag.vue'
import type { CurrentLessonData } from '@/api/learning'
import type { Course } from '@/types/course'
import type { CognitiveLevel, CognitiveStatus } from '@/types/cognitive'

defineProps<{
  course: Course
  lesson: CurrentLessonData | null
  cognitive?: { current_level: CognitiveLevel; status: CognitiveStatus }
  loading?: boolean
}>()
defineEmits<{ continue: [] }>()

function relativeDate(value: string): string {
  const date = new Date(value)
  const days = Math.floor((Date.now() - date.getTime()) / 86400000)
  if (days <= 0) return '今天'
  if (days === 1) return '昨天'
  if (days < 7) return `${days} 天前`
  return date.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
}
</script>
