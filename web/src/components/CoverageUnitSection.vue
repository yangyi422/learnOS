<template>
  <section class="coverage-unit-section">
    <header class="coverage-unit-section__header">
      <div><span class="eyebrow">知识区域 {{ String(unit.unit.sort_order + 1).padStart(2, '0') }}</span><h2>{{ unit.unit.title }}</h2><p>{{ unit.unit.description }}</p></div>
      <span class="coverage-unit-section__count">{{ unit.core_covered }} / {{ unit.core_total }} 核心</span>
    </header>
    <div class="coverage-unit-section__progress"><el-progress :percentage="unit.core_total ? Math.round(unit.core_covered / unit.core_total * 100) : 100" :show-text="false" :stroke-width="4" /></div>
    <div class="coverage-lesson-list">
      <div v-for="lesson in unit.lessons" :key="lesson.blueprint_lesson.id" class="coverage-lesson-row">
        <span class="coverage-lesson-row__marker" :class="lesson.state === 'covered' ? 'is-covered' : lesson.blueprint_lesson.importance === 'core' ? 'is-missing' : 'is-pending'">{{ lesson.state === 'covered' ? '●' : lesson.blueprint_lesson.importance === 'core' ? '!' : '○' }}</span>
        <div class="coverage-lesson-row__copy"><strong>{{ lesson.blueprint_lesson.title }}</strong><small>{{ lesson.blueprint_lesson.summary }}</small></div>
        <StatusTag :status="lesson.state === 'covered' ? 'stable' : lesson.blueprint_lesson.importance === 'core' ? 'error' : 'unseen'" :label="lesson.state === 'covered' ? '已映射' : lesson.blueprint_lesson.importance === 'core' ? '缺失核心' : '待补充'" />
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import StatusTag from '@/components/StatusTag.vue'
import type { CurriculumCoverageUnit } from '@/types/curriculum'

defineProps<{ unit: CurriculumCoverageUnit }>()
</script>
