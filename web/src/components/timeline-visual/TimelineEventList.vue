<template>
  <div v-if="!months.length" class="tvl-empty">
    <h3>这个范围里还没有记录</h3>
    <p>换一个时间范围或清除筛选，也可以先去生活时间线补录一条经历。</p>
  </div>
  <div v-else class="tvl-list">
    <section v-for="month in months" :key="month.month" class="tvl-month">
      <h3 class="tvl-month__label">{{ month.label }}</h3>
      <button
        v-for="event in month.events"
        :key="event.id"
        type="button"
        class="tvl-row"
        :class="{ 'is-selected': event.id === selectedId, 'is-milestone': event.isMilestone }"
        :aria-current="event.id === selectedId ? 'true' : undefined"
        @click="$emit('select', event)"
      >
        <span class="tvl-row__date">{{ event.occurredAt.slice(5).replace('-', '/') }}</span>
        <span class="tvl-row__main">
          <span class="tvl-row__title">
            <span class="tvl-row__mark" :class="{ 'is-milestone': event.isMilestone }" aria-hidden="true">{{ event.isMilestone ? '◇' : '·' }}</span>
            {{ event.title }}
          </span>
          <span class="tvl-row__meta">
            <span class="tvl-dot" :style="{ '--h': categoryHue(event.category) }" aria-hidden="true" />
            <span>{{ categoryLabel(event.category) }}</span>
            <span v-if="event.secondaryCategory">/ {{ categoryLabel(event.secondaryCategory) }}</span>
            <span v-if="event.goalTitle">· {{ event.goalTitle }}</span>
            <span v-if="event.sourceTitle">· 来源：{{ event.sourceTitle }}</span>
            <span v-if="event.isMilestone" class="tvl-row__flag">· 里程碑</span>
          </span>
        </span>
        <span class="tvl-row__arrow" aria-hidden="true">→</span>
      </button>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { categoryHue, categoryLabel, groupTimelineByMonth, type TimelineEvent } from '@/utils/timelineVisual'

const props = defineProps<{ events: TimelineEvent[]; selectedId?: string | null }>()
defineEmits<{ select: [event: TimelineEvent] }>()

const months = computed(() => groupTimelineByMonth(props.events))
</script>

<style scoped>
.tvl-list { display: grid; gap: 8px; }
.tvl-month { display: grid; gap: 8px; }
.tvl-month__label { margin: 18px 0 2px; color: var(--home-secondary); font-size: 13px; font-weight: 600; }
.tvl-month:first-child .tvl-month__label { margin-top: 6px; }
.tvl-row {
  display: grid;
  grid-template-columns: 56px minmax(0, 1fr) 16px;
  gap: 14px;
  align-items: start;
  width: 100%;
  padding: 14px 16px;
  border: 1px solid var(--home-edge);
  border-radius: var(--radius-card);
  background: var(--home-panel);
  color: var(--home-ink);
  font: inherit;
  text-align: left;
  cursor: pointer;
  transition: border-color 180ms ease, background-color 180ms ease;
}
.tvl-row:hover { border-color: var(--home-reflection); background-color: var(--home-panel-hover); }
.tvl-row:focus-visible { outline: 2px solid var(--home-link); outline-offset: 2px; }
.tvl-row.is-selected { border-color: var(--home-link); }
.tvl-row__date { padding-top: 2px; color: var(--home-muted); font-size: 12px; font-variant-numeric: tabular-nums; text-align: right; }
.tvl-row__main { display: grid; gap: 5px; min-width: 0; }
.tvl-row__title { display: flex; gap: 7px; align-items: baseline; font-size: var(--font-card-title); font-weight: 600; overflow-wrap: anywhere; }
.tvl-row__mark { color: var(--home-muted); }
.tvl-row__mark.is-milestone { color: var(--home-link); }
.tvl-row__meta { display: flex; flex-wrap: wrap; gap: 6px; align-items: center; color: var(--home-muted); font-size: var(--font-meta); }
.tvl-dot { width: 7px; height: 7px; border-radius: 50%; background: hsl(var(--h) 55% 52%); }
.tvl-row__flag { color: var(--home-link); }
.tvl-row__arrow { color: var(--home-muted); font-size: 13px; }
.tvl-empty { padding: 40px 24px; border: 1px dashed var(--home-edge); border-radius: var(--radius-card); text-align: center; }
.tvl-empty h3 { margin: 0 0 6px; font-size: var(--font-card-title); }
.tvl-empty p { margin: 0; color: var(--home-secondary); font-size: var(--font-meta); line-height: 1.7; }
:global(html[data-theme='dark']) .tvl-dot { background: hsl(var(--h) 55% 66%); }
@media (max-width: 640px) {
  .tvl-row { grid-template-columns: 1fr; gap: 6px; }
  .tvl-row__date { text-align: left; padding-top: 0; }
  .tvl-row__arrow { display: none; }
}
@media (prefers-reduced-motion: reduce) { .tvl-row { transition: none; } }
</style>
