<template>
  <section class="coverage-summary" aria-label="课程覆盖统计">
    <div class="coverage-summary__lead"><span class="coverage-summary__label">核心内容覆盖</span><strong>{{ metrics.core_coverage_percent.toFixed(0) }}%</strong><span>{{ metrics.core_covered }} / {{ metrics.core_total }} 核心节点</span></div>
    <div><span class="coverage-summary__label">整体内容覆盖</span><strong>{{ metrics.overall_coverage_percent.toFixed(0) }}%</strong><span>{{ metrics.core_covered + metrics.recommended_covered + metrics.optional_covered }} 个已映射节点</span></div>
    <div><span class="coverage-summary__label">待补核心节点</span><strong>{{ metrics.core_missing }}</strong><span>不代表个人学习进度</span></div>
  </section>
</template>

<script setup lang="ts">
import type { CurriculumCoverageMetrics } from '@/types/curriculum'

defineProps<{ metrics: CurriculumCoverageMetrics }>()
</script>
