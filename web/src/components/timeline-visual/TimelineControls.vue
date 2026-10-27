<template>
  <section class="tvl-controls" aria-label="时间线筛选">
    <div class="tvl-controls__row">
      <span class="tvl-controls__label">范围</span>
      <el-radio-group :model-value="rangeMonths" size="small" aria-label="时间范围" @update:model-value="onRange">
        <el-radio-button :value="1">近 1 月</el-radio-button>
        <el-radio-button :value="3">3 月</el-radio-button>
        <el-radio-button :value="12">1 年</el-radio-button>
        <el-radio-button :value="0">全部</el-radio-button>
      </el-radio-group>
    </div>

    <div class="tvl-controls__row">
      <span class="tvl-controls__label">领域</span>
      <el-select
        :model-value="categories"
        multiple
        collapse-tags
        clearable
        size="small"
        placeholder="全部领域"
        aria-label="按领域筛选"
        class="tvl-controls__select"
        @update:model-value="onCategories"
      >
        <el-option v-for="category in categories_all" :key="category.key" :label="category.label" :value="category.key" />
      </el-select>
      <el-checkbox :model-value="milestonesOnly" size="small" @update:model-value="onMilestones">只看里程碑</el-checkbox>
    </div>

    <div class="tvl-controls__row">
      <span class="tvl-controls__label">数据</span>
      <el-radio-group :model-value="source" size="small" aria-label="数据来源" @update:model-value="onSource">
        <el-radio-button value="real">真实记录</el-radio-button>
        <el-radio-button value="sample">示例数据</el-radio-button>
      </el-radio-group>
      <el-radio-group v-if="source === 'sample'" :model-value="sampleSize" size="small" aria-label="示例数据量" @update:model-value="onSampleSize">
        <el-radio-button :value="20">20</el-radio-button>
        <el-radio-button :value="100">100</el-radio-button>
        <el-radio-button :value="500">500</el-radio-button>
      </el-radio-group>
    </div>

    <p class="tvl-controls__meta">
      当前显示 {{ visible }} / {{ total }} 条
      <span v-if="source === 'sample'"> · 示例数据，不对应任何真实经历</span>
    </p>
  </section>
</template>

<script setup lang="ts">
import { TIMELINE_CATEGORIES, type TimelineCategoryKey } from '@/utils/timelineVisual'

defineProps<{
  rangeMonths: number
  categories: TimelineCategoryKey[]
  milestonesOnly: boolean
  source: 'real' | 'sample'
  sampleSize: number
  visible: number
  total: number
}>()

const emit = defineEmits<{
  'update:rangeMonths': [value: number]
  'update:categories': [value: TimelineCategoryKey[]]
  'update:milestonesOnly': [value: boolean]
  'update:source': [value: 'real' | 'sample']
  'update:sampleSize': [value: number]
}>()

const categories_all = TIMELINE_CATEGORIES

function onRange(value: string | number | boolean | undefined): void {
  if (typeof value === 'number') emit('update:rangeMonths', value)
}

function onCategories(value: unknown): void {
  if (Array.isArray(value)) emit('update:categories', value as TimelineCategoryKey[])
  else emit('update:categories', [])
}

function onMilestones(value: string | number | boolean): void {
  emit('update:milestonesOnly', Boolean(value))
}

function onSource(value: string | number | boolean | undefined): void {
  if (value === 'real' || value === 'sample') emit('update:source', value)
}

function onSampleSize(value: string | number | boolean | undefined): void {
  if (typeof value === 'number') emit('update:sampleSize', value)
}

</script>

<style scoped>
.tvl-controls {
  display: grid;
  gap: 10px;
  padding: 16px 18px;
  border: 1px solid var(--home-edge);
  border-radius: var(--radius-card);
  background: var(--home-panel);
}
.tvl-controls__row { display: flex; flex-wrap: wrap; align-items: center; gap: 10px; }
.tvl-controls__label { flex: none; width: 2.6em; color: var(--home-secondary); font-size: var(--font-meta); }
.tvl-controls__select { min-width: 190px; max-width: 280px; }
.tvl-controls__meta { margin: 2px 0 0; color: var(--home-muted); font-size: var(--font-caption); }
</style>
