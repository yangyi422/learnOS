<template>
  <PageShell class="tvl-page" variant="standard">
    <PageHeader title="Timeline Visual Lab" description="用真实生活记录检验时间线视图，只读，不写入任何业务数据。">
      <template #actions><span class="tvl-badge">仅本地开发 · 未注册正式路由</span></template>
    </PageHeader>

    <TimelineControls
      :range-months="rangeMonths"
      :categories="categories"
      :milestones-only="milestonesOnly"
      :source="source"
      :sample-size="sampleSize"
      :visible="visibleEvents.length"
      :total="events.length"
      @update:range-months="changeRange"
      @update:categories="categories = $event"
      @update:milestones-only="milestonesOnly = $event"
      @update:source="changeSource"
      @update:sample-size="changeSampleSize"
    />

    <p v-if="loading" class="tvl-status" role="status">正在读取时间线…</p>
    <p v-else-if="error" class="tvl-status tvl-status--error" role="alert">
      {{ error }}
      <button type="button" class="tvl-inline" @click="reload">重试</button>
    </p>
    <p v-else-if="source === 'real' && realCount > 0 && realCount < 5" class="tvl-status">
      真实记录还不多（{{ realCount }} 条）。可以先用示例数据观察空间视图。
      <button type="button" class="tvl-inline" @click="changeSource('sample')">载入示例数据</button>
    </p>

    <div class="tvl-grid" :class="{ 'has-detail': Boolean(selected) }">
      <section class="tvl-main" aria-label="时间线视图">
        <TimelineRuler
          :events="visibleEvents"
          :from="range.from"
          :to="range.to"
          :selected-id="selected?.id"
          @pan="panBy"
          @zoom="zoomBy"
        />
        <TimelineEventList :events="visibleEvents" :selected-id="selected?.id" @select="selectEvent" />
      </section>
      <TimelineDetailPanel v-if="selected" :event="selected" @close="selected = null" />
    </div>

    <p class="tvl-foot">本页只读：不写入生活记录，也不改变现有时间线的可见性规则；示例数据不对应任何真实个人内容。</p>
  </PageShell>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import PageHeader from '@/components/PageHeader.vue'
import PageShell from '@/components/PageShell.vue'
import TimelineControls from '@/components/timeline-visual/TimelineControls.vue'
import TimelineDetailPanel from '@/components/timeline-visual/TimelineDetailPanel.vue'
import TimelineEventList from '@/components/timeline-visual/TimelineEventList.vue'
import TimelineRuler from '@/components/timeline-visual/TimelineRuler.vue'
import { listLifeEvents, listLifeGoals } from '@/api/life'
import { sampleTimeline } from '@/utils/timelineSample'
import {
  addMonths,
  dateToDay,
  dayToDate,
  filterTimelineEvents,
  sortTimelineEvents,
  toTimelineEvent,
  type TimelineCategoryKey,
  type TimelineEvent,
} from '@/utils/timelineVisual'

const RANGE_PRESETS = [1, 3, 12, 24, 0]
const route = useRoute()
const router = useRouter()

const today = (() => {
  const now = new Date()
  return new Date(now.getTime() - now.getTimezoneOffset() * 60_000).toISOString().slice(0, 10)
})()

const source = ref<'real' | 'sample'>(route.query.source === 'sample' ? 'sample' : 'real')
const sampleSize = ref([20, 100, 500].includes(Number(route.query.size)) ? Number(route.query.size) : 100)
const rangeMonths = ref(12)
const categories = ref<TimelineCategoryKey[]>([])
const milestonesOnly = ref(false)
const events = ref<TimelineEvent[]>([])
const realCount = ref(0)
const loading = ref(false)
const error = ref('')
const selected = ref<TimelineEvent | null>(null)
const panDays = ref(0)
let sequence = 0

const earliest = computed(() => events.value.reduce((min, event) => (event.occurredAt < min ? event.occurredAt : min), today))

const range = computed(() => {
  const base = rangeMonths.value === 0
    ? { from: earliest.value, to: today }
    : { from: addMonths(today, -rangeMonths.value), to: today }
  if (!panDays.value) return base
  return {
    from: dayToDate(dateToDay(base.from) + panDays.value),
    to: dayToDate(dateToDay(base.to) + panDays.value),
  }
})

const visibleEvents = computed(() => filterTimelineEvents(events.value, {
  categories: categories.value,
  milestonesOnly: milestonesOnly.value,
  from: range.value.from,
  to: range.value.to,
}))

function selectEvent(event: TimelineEvent): void {
  selected.value = event.id === selected.value?.id ? null : event
}

function changeRange(value: number): void {
  rangeMonths.value = value
  panDays.value = 0
}

function changeSource(value: 'real' | 'sample'): void {
  if (source.value === value) return
  source.value = value
  selected.value = null
}

function changeSampleSize(value: number): void {
  sampleSize.value = value
}

function panBy(days: number): void {
  panDays.value = Math.max(-730, Math.min(730, panDays.value + days))
}

function zoomBy(factor: number): void {
  const current = RANGE_PRESETS.indexOf(rangeMonths.value)
  const index = current === -1 ? 2 : current
  const next = factor > 1 ? Math.max(0, index - 1) : Math.min(RANGE_PRESETS.length - 1, index + 1)
  changeRange(RANGE_PRESETS[next]!)
}

async function load(): Promise<void> {
  const current = ++sequence
  selected.value = null
  if (source.value === 'sample') {
    loading.value = false
    error.value = ''
    events.value = sortTimelineEvents(sampleTimeline(sampleSize.value))
    return
  }
  loading.value = true
  error.value = ''
  try {
    const [records, goals] = await Promise.all([listLifeEvents(), listLifeGoals()])
    if (current !== sequence) return
    const goalTitles = new Map(goals.map((goal) => [goal.id, goal.title]))
    const mapped = sortTimelineEvents(records.map((record) => toTimelineEvent(record, goalTitles)))
    events.value = mapped
    realCount.value = mapped.length
  } catch {
    if (current !== sequence) return
    error.value = '无法读取生活时间线，请稍后重试。'
    events.value = []
  } finally {
    if (current === sequence) loading.value = false
  }
}

function reload(): void {
  void load()
}

function syncQuery(): void {
  void router.replace({
    query: {
      ...route.query,
      source: source.value === 'sample' ? 'sample' : undefined,
      size: source.value === 'sample' ? String(sampleSize.value) : undefined,
      view: undefined,
    },
  })
}

function handleKeydown(event: KeyboardEvent): void {
  if (event.key === 'Escape' && selected.value) selected.value = null
}

watch([source, sampleSize], () => { void load() })
watch([source, sampleSize], syncQuery)

onMounted(() => {
  window.addEventListener('keydown', handleKeydown)
  void load()
})

onBeforeUnmount(() => {
  sequence += 1
  window.removeEventListener('keydown', handleKeydown)
})
</script>

<style scoped>
.tvl-page { display: grid; gap: var(--space-4); }
.tvl-badge {
  display: inline-flex;
  align-items: center;
  height: 28px;
  padding: 0 12px;
  border: 1px solid var(--home-edge);
  border-radius: var(--radius-pill);
  color: var(--home-secondary);
  font-size: var(--font-caption);
  white-space: nowrap;
}
.tvl-status { margin: 0; color: var(--home-muted); font-size: var(--font-meta); }
.tvl-status--error { color: var(--home-secondary); }
.tvl-inline { padding: 0; border: 0; background: none; color: var(--home-link); font: inherit; font-size: var(--font-meta); text-decoration: underline; text-underline-offset: 3px; cursor: pointer; }
.tvl-inline:focus-visible { outline: 2px solid var(--home-link); outline-offset: 2px; }
.tvl-grid { display: grid; grid-template-columns: minmax(0, 1fr); gap: var(--space-4); align-items: start; }
.tvl-grid.has-detail { grid-template-columns: minmax(0, 1fr) 340px; }
.tvl-main { display: grid; gap: var(--space-3); min-width: 0; }
.tvl-foot { margin: 0; color: var(--home-muted); font-size: var(--font-caption); line-height: 1.7; }
@media (max-width: 899px) {
  .tvl-grid.has-detail { grid-template-columns: minmax(0, 1fr); }
}
</style>
