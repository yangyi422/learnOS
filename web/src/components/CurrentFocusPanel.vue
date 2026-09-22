<template>
  <section class="current-focus" aria-labelledby="current-focus-title">
    <div class="current-focus__eyebrow">当前学习重点</div>
    <div class="current-focus__body">
      <div class="current-focus__copy">
        <span class="current-focus__domain">{{ course.name }}</span>
        <h2 id="current-focus-title">{{ lesson?.lesson.title || course.current_unit || '准备你的第一个学习节点' }}</h2>
        <p v-if="lesson">{{ lesson.lesson.core_question }}</p>
        <p v-else>{{ course.description || '从这个学习领域开始建立你的知识地图。' }}</p>
        <div class="current-focus__meta">
          <span>{{ lesson?.unit.title || course.current_unit || '尚未开始' }}</span>
          <StatusTag v-if="cognitive" :status="cognitive.current_level" />
          <StatusTag v-if="cognitive" :status="cognitive.status" effect="plain" />
        </div>
        <dl class="current-focus__stats">
          <div><dt>最近学习</dt><dd>{{ course.last_studied_at ? relativeDate(course.last_studied_at) : '尚未开始' }}</dd></div>
          <div><dt>理解掌握度</dt><dd>{{ Math.round(course.mastery_progress) }}%</dd></div>
          <div><dt>建议预留</dt><dd>10–15 分钟</dd></div>
        </dl>
        <p class="current-focus__next"><strong>下一步目标：</strong>{{ lesson?.lesson.core_question || course.goal || '打开知识结构并选择第一个学习节点。' }}</p>
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
