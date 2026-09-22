<template>
  <section class="editorial-feedback" aria-live="polite">
    <div class="editorial-feedback__header">
      <div><span class="eyebrow">AI 评价</span><h2>本次学习反馈</h2></div>
      <StatusTag :status="feedback.result === 'correct' || feedback.result === 'mostly_correct' ? 'stable' : 'needs_review'" :label="resultText" />
    </div>
    <p v-if="feedback.user_understanding_summary" class="editorial-feedback__lead">{{ feedback.user_understanding_summary }}</p>
    <div class="editorial-feedback__section editorial-feedback__section--correct">
      <h3><span>✓</span> 已理解的部分</h3>
      <ul v-if="feedback.correct_parts.length"><li v-for="part in feedback.correct_parts" :key="part">{{ part }}</li></ul>
      <p v-else>这次回答还没有形成明确的正确理解。</p>
    </div>
    <div class="editorial-feedback__section editorial-feedback__section--missing">
      <h3><span>△</span> 关键知识缺口</h3>
      <ul v-if="feedback.missing_parts.length"><li v-for="part in feedback.missing_parts" :key="part">{{ part }}</li></ul>
      <p v-else>本次回答未发现明显缺失。</p>
    </div>
    <div class="editorial-feedback__section" :class="{ 'editorial-feedback__section--warning': feedback.misconceptions.length }">
      <h3><span>!</span> 可能存在的误区</h3>
      <div v-for="item in feedback.misconceptions" :key="item.original_understanding" class="editorial-feedback__misconception">
        <strong>{{ item.original_understanding }}</strong><p>{{ item.correct_understanding }}</p><small v-if="item.boundary_notes">边界：{{ item.boundary_notes }}</small>
      </div>
      <p v-if="!feedback.misconceptions.length">本次回答未识别到明确误区；这不代表已经完成迁移验证。</p>
    </div>
    <div v-if="feedback.explanation" class="editorial-feedback__section">
      <h3>为什么</h3><p>{{ feedback.explanation }}</p>
    </div>
    <div v-if="feedback.boundary_conditions.length" class="editorial-feedback__section">
      <h3>边界与反例</h3><ul><li v-for="condition in feedback.boundary_conditions" :key="condition">{{ condition }}</li></ul>
    </div>
    <div v-if="feedback.mastery_evidence.length" class="editorial-feedback__section">
      <h3>本次掌握证据</h3><ul><li v-for="evidence in feedback.mastery_evidence" :key="evidence">{{ evidence }}</li></ul>
    </div>
    <div class="editorial-feedback__section">
      <h3>评价使用的证据</h3>
      <ul><li v-for="evidence in feedback.evidence_used" :key="evidence">{{ evidence }}</li></ul>
    </div>
    <div class="editorial-feedback__section">
      <h3>可信度与不确定性</h3>
      <p>本次结论可信度：{{ Math.round(feedback.confidence * 100) }}%</p>
      <p class="muted-text">{{ feedback.uncertainty }}</p>
    </div>
    <div class="editorial-feedback__section">
      <h3>推荐下一步</h3>
      <p>{{ feedback.recommended_next_action }}</p>
      <p class="muted-text">{{ feedback.transfer_challenge_eligible ? '本次评价已满足迁移挑战的内容条件。' : '本次评价尚未满足迁移挑战条件，请先修正关键缺口。' }}</p>
    </div>
    <div class="editorial-feedback__section"><h3>完整反馈</h3><p>{{ feedback.feedback }}</p></div>
    <div class="editorial-feedback__footer"><span>{{ feedbackSource }}</span><span>本次评价 {{ Math.round(feedback.mastery_score * 100) }}% · 累计掌握度 {{ Math.round(feedback.mastery_score_before * 100) }}% → {{ Math.round(feedback.mastery_score_after * 100) }}%</span></div>
  </section>
</template>

<script setup lang="ts">
import StatusTag from '@/components/StatusTag.vue'
import type { AnswerResult } from '@/api/learning'

defineProps<{ feedback: AnswerResult; resultText: string; feedbackSource: string }>()
</script>
