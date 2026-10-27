<template>
  <aside class="tvl-detail" aria-label="事件详情">
    <header class="tvl-detail__head">
      <h2>{{ event.title }}</h2>
      <button type="button" class="tvl-detail__close" aria-label="关闭详情" @click="$emit('close')">×</button>
    </header>
    <p class="tvl-detail__meta">
      <time :datetime="event.occurredAt">{{ event.occurredAt }}</time>
      <span>· {{ categoryLabel(event.category) }}</span>
      <span v-if="event.secondaryCategory">/ {{ categoryLabel(event.secondaryCategory) }}</span>
      <span v-if="event.isMilestone" class="tvl-detail__flag">· 里程碑</span>
    </p>
    <p v-if="event.goalTitle" class="tvl-detail__relation">目标：{{ event.goalTitle }}</p>
    <p v-if="event.sourceTitle" class="tvl-detail__relation">来源：{{ event.sourceTitle }}</p>
    <p class="tvl-detail__body">{{ event.summary || '没有补充描述。' }}</p>
    <div class="tvl-detail__actions">
      <RouterLink :to="event.href" class="tvl-detail__link">在原时间线中打开 →</RouterLink>
      <ExternalLink v-if="event.externalUrl" :url="event.externalUrl" />
    </div>
  </aside>
</template>

<script setup lang="ts">
import ExternalLink from '@/components/ExternalLink.vue'
import { categoryLabel, type TimelineEvent } from '@/utils/timelineVisual'

defineProps<{ event: TimelineEvent }>()
defineEmits<{ close: [] }>()
</script>

<style scoped>
.tvl-detail {
  position: sticky;
  top: 96px;
  display: grid;
  gap: 10px;
  padding: 20px;
  border: 1px solid var(--home-edge);
  border-radius: var(--radius-card);
  background: var(--home-panel);
}
.tvl-detail__head { display: flex; align-items: start; justify-content: space-between; gap: 12px; }
.tvl-detail__head h2 { margin: 0; color: var(--home-ink); font-size: var(--font-card-title); font-weight: 700; line-height: 1.45; overflow-wrap: anywhere; }
.tvl-detail__close { flex: none; width: 28px; height: 28px; border: 1px solid var(--home-edge); border-radius: 8px; background: transparent; color: var(--home-secondary); font-size: 16px; line-height: 1; cursor: pointer; }
.tvl-detail__close:focus-visible, .tvl-detail__link:focus-visible { outline: 2px solid var(--home-link); outline-offset: 2px; }
.tvl-detail__meta { margin: 0; color: var(--home-muted); font-size: var(--font-meta); font-variant-numeric: tabular-nums; }
.tvl-detail__flag { color: var(--home-link); }
.tvl-detail__relation { margin: 0; color: var(--home-secondary); font-size: var(--font-meta); }
.tvl-detail__body { max-height: 260px; margin: 4px 0 0; overflow: auto; color: var(--home-ink); font-size: var(--font-meta); line-height: 1.8; white-space: pre-wrap; overflow-wrap: anywhere; }
.tvl-detail__actions { display: flex; flex-wrap: wrap; gap: 12px; align-items: center; padding-top: 10px; border-top: 1px solid var(--home-edge); }
.tvl-detail__link { color: var(--home-link); font-size: var(--font-meta); text-decoration: none; }
.tvl-detail__link:hover { text-decoration: underline; text-underline-offset: 3px; }
</style>
