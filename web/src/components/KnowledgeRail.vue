<template>
  <aside class="knowledge-rail" aria-label="知识侧栏">
    <section v-if="cognitive" class="knowledge-rail__section">
      <span class="knowledge-rail__label">理解状态</span>
      <StatusTag :status="cognitive.state.current_level" />
      <strong>{{ levelText(cognitive.state.current_level) }}</strong>
      <p>{{ cognitive.state.understanding_summary || '尚无认知证据，当前为未接触。' }}</p>
      <span class="knowledge-rail__meta">掌握证据 {{ cognitive.evidence.length }} 条</span>
    </section>
    <section v-if="current" class="knowledge-rail__section">
      <span class="knowledge-rail__label">知识位置</span>
      <div class="knowledge-rail__location">
        <span>{{ current.course.name }}</span><b>›</b><span>{{ current.unit.title }}</span><b>›</b><strong>{{ current.lesson.title }}</strong>
      </div>
      <p class="knowledge-rail__meta">节点类型：{{ knowledgeNodeTypeText(current.lesson.content_role) }} · {{ knowledgeDepthText(current.lesson.depth_level) }}</p>
      <p v-if="relations" class="knowledge-rail__meta">前置 {{ relations.prerequisites.length }} · 深化 {{ relations.extensions.length }} · 应用 {{ relations.applications.length }}</p>
    </section>
    <section v-if="current" class="knowledge-rail__section">
      <span class="knowledge-rail__label">内容来源</span>
      <p>当前课程内容</p>
      <span class="knowledge-rail__meta">{{ current.unit.title }} · {{ current.lesson.title }}</span>
    </section>
    <slot />
  </aside>
</template>

<script setup lang="ts">
import StatusTag from '@/components/StatusTag.vue'
import type { CurrentLessonData } from '@/api/learning'
import type { LessonRelations } from '@/types/knowledgeGraph'
import type { CognitiveStateDetail } from '@/types/cognitive'
import { knowledgeDepthText, knowledgeNodeTypeText } from '@/utils/knowledgeNodeType'

defineProps<{ current: CurrentLessonData | null; cognitive: CognitiveStateDetail | null; relations: LessonRelations | null }>()

function levelText(level: string): string {
  return { unseen: '未接触', exposed: '已接触', recognize: '识别', understand: '理解', apply: '应用', transfer: '迁移' }[level] ?? level
}
</script>
