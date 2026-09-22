<template>
  <article class="curriculum-draft-row">
    <div class="curriculum-draft-row__header"><div><span class="eyebrow">课程草案</span><h2>{{ item.draft.title }}</h2><p>{{ item.draft.summary }}</p></div><StatusTag :status="item.draft.status" /></div>
    <div class="curriculum-draft-row__meta"><span>{{ item.change_set.new_lessons.length }} 个课程节点</span><span>{{ item.draft.generated_by === 'ai' ? 'AI 草案' : '规则草案' }}</span><span>{{ formatDate(item.draft.created_at) }}</span></div>
    <ul class="curriculum-draft-row__lessons"><li v-for="lesson in item.change_set.new_lessons" :key="lesson.temp_key"><strong>{{ lesson.title }}</strong><span>{{ lesson.summary }}</span></li></ul>
    <div v-if="item.draft.status === 'draft'" class="curriculum-draft-row__actions"><el-button size="small" @click="$emit('reject', item.draft.id)">拒绝草案</el-button><el-button size="small" type="primary" @click="$emit('apply', item.draft.id)">审核通过并应用 →</el-button></div>
  </article>
</template>

<script setup lang="ts">
import StatusTag from '@/components/StatusTag.vue'
import type { CurriculumDraftView } from '@/types/curriculum'

defineProps<{ item: CurriculumDraftView }>()
defineEmits<{ reject: [id: number]; apply: [id: number] }>()
function formatDate(value: string) { return new Date(value).toLocaleString('zh-CN') }
</script>
