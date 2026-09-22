<template>
  <article class="discovery-card">
    <div class="discovery-card__topline">
      <span class="discovery-card__type">{{ directionTypeText(direction.direction_type) }}</span>
      <span class="discovery-card__arrow" aria-hidden="true">探索建议</span>
    </div>
    <h3>{{ direction.title }}</h3>
    <p><strong>推荐理由：</strong>{{ direction.why_worth_exploring || direction.summary }}</p>
    <p><strong>与当前知识的关系：</strong>{{ relationshipText(direction) }}</p>
    <div class="discovery-card__origin">
      <span>来自</span>
      <strong>{{ direction.source_course.name }}</strong>
      <span>· {{ direction.source_lesson.title }}</span>
    </div>
    <div class="exploration-actions">
      <el-button size="small" type="primary" @click="$emit('open')">开始探索</el-button>
      <template v-if="direction.status === 'saved'">
        <el-button size="small" disabled>已保存</el-button>
        <el-button size="small" text @click="$emit('undo')">撤销</el-button>
      </template>
      <el-button v-else size="small" @click="$emit('save')">保存到问题池</el-button>
    </div>
  </article>
</template>

<script setup lang="ts">
import type { ExplorationDirection } from '@/types/exploration'

defineProps<{ direction: ExplorationDirection }>()
defineEmits<{ open: []; save: []; undo: [] }>()

function directionTypeText(type: string): string {
  return { adjacent: '邻近探索', cross_domain: '跨领域', unknown: '陌生知识' }[type] ?? type
}

function relationshipText(direction: ExplorationDirection): string {
  if (direction.direction_type === 'cross_domain') return `把当前领域的判断方法迁移到「${direction.target_lesson.title}」`
  if (direction.direction_type === 'unknown') return `从当前领域之外建立一个新入口：${direction.target_lesson.title}`
  return `沿当前知识的相邻关系延伸到「${direction.target_lesson.title}」`
}
</script>
