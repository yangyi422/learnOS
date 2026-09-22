<template>
  <div class="learning-state-summary">
    <div class="learning-state-summary__topline">
      <span class="eyebrow">学习状态</span>
      <StatusTag :status="state.current_level" />
    </div>
    <div class="learning-state-summary__status">
      <strong>{{ levelText(state.current_level) }}</strong>
      <StatusTag :status="state.status" effect="plain" />
    </div>
    <p v-if="state.understanding_summary">{{ state.understanding_summary }}</p>
    <p v-else class="muted-text">尚无认知证据，当前为未接触。</p>
    <div v-if="evidenceCount !== undefined" class="learning-state-summary__meta">掌握证据 {{ evidenceCount }} 条</div>
  </div>
</template>

<script setup lang="ts">
import StatusTag from '@/components/StatusTag.vue'
import type { CognitiveLevel, CognitiveStatus } from '@/types/cognitive'

defineProps<{
  state: { current_level: CognitiveLevel; status: CognitiveStatus; understanding_summary: string }
  evidenceCount?: number
}>()

function levelText(level: CognitiveLevel): string {
  return { unseen: '未接触', exposed: '已接触', recognize: '识别', understand: '理解', apply: '应用', transfer: '迁移' }[level]
}
</script>
