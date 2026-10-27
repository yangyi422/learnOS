<template>
  <div ref="host" class="tvl-ruler" aria-hidden="true">
    <svg :width="width" :height="HEIGHT" @pointerdown="panStart" @pointermove="panMove" @pointerup="panEnd" @pointerleave="panEnd">
      <g>
        <template v-for="tick in ticks" :key="tick.month">
          <line :x1="tickX(tick.month)" :x2="tickX(tick.month)" y1="18" :y2="HEIGHT - 6" class="tvl-ruler__tick" />
          <text v-if="tickIndex(tick.month) % labelStep === 0" :x="tickX(tick.month)" y="13" class="tvl-ruler__label" :class="{ 'is-year': tick.yearStart }">
            {{ tick.label }}
          </text>
        </template>
      </g>
      <g>
        <circle
          v-for="point in points"
          :key="point.id"
          :cx="point.x"
          :cy="point.y"
          :r="point.milestone ? 5 : 3"
          class="tvl-ruler__dot"
          :class="{ 'is-milestone': point.milestone, 'is-selected': point.selected }"
          :style="{ '--h': point.hue }"
        />
      </g>
      <line x1="0" :x2="width" :y1="HEIGHT - 4" :y2="HEIGHT - 4" class="tvl-ruler__axis" />
    </svg>
    <p class="tvl-ruler__hint">拖动平移 · 滚轮缩放</p>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { categoryHue, dateToDay, monthTicks, type TimelineEvent } from '@/utils/timelineVisual'

const props = defineProps<{ events: TimelineEvent[]; from: string; to: string; selectedId?: string | null }>()
const emit = defineEmits<{ pan: [days: number]; zoom: [factor: number] }>()

const HEIGHT = 72
const host = ref<HTMLElement | null>(null)
const width = ref(720)
let observer: ResizeObserver | null = null
let dragStartX = 0
let dragging = false

const ticks = computed(() => monthTicks(props.from, props.to))
const labelStep = computed(() => Math.max(1, Math.ceil(ticks.value.length / 16)))
const startDay = computed(() => dateToDay(props.from))
const daySpan = computed(() => Math.max(1, dateToDay(props.to) - startDay.value))

function tickIndex(month: string): number {
  return ticks.value.findIndex((tick) => tick.month === month)
}

function xForDay(day: number): number {
  return Math.round(((day - startDay.value) / daySpan.value) * (width.value - 8) + 4)
}

function tickX(month: string): number {
  return xForDay(dateToDay(`${month}-01`))
}

const points = computed(() => props.events.slice(0, 600).map((event, index) => ({
  id: event.id,
  x: xForDay(dateToDay(event.occurredAt)),
  y: 30 + (index % 3) * 11,
  milestone: event.isMilestone,
  selected: event.id === props.selectedId,
  hue: categoryHue(event.category),
})))

function panStart(event: PointerEvent): void {
  dragging = true
  dragStartX = event.clientX
  try { (event.currentTarget as Element | null)?.setPointerCapture?.(event.pointerId) } catch { /* synthetic pointer */ }
}

function panMove(event: PointerEvent): void {
  if (!dragging) return
  const deltaX = event.clientX - dragStartX
  if (Math.abs(deltaX) < 3) return
  dragStartX = event.clientX
  emit('pan', Math.round((-deltaX / Math.max(1, width.value)) * daySpan.value))
}

function panEnd(): void {
  dragging = false
}

function handleWheel(event: WheelEvent): void {
  event.preventDefault()
  emit('zoom', event.deltaY > 0 ? 1.25 : 0.8)
}

onMounted(() => {
  observer = new ResizeObserver(() => {
    width.value = Math.max(320, Math.round(host.value?.clientWidth ?? 720))
  })
  if (host.value) {
    width.value = Math.max(320, Math.round(host.value.clientWidth))
    observer.observe(host.value)
    host.value.addEventListener('wheel', handleWheel, { passive: false })
  }
})

onBeforeUnmount(() => {
  observer?.disconnect()
  host.value?.removeEventListener('wheel', handleWheel)
})
</script>

<style scoped>
.tvl-ruler { position: relative; padding: 4px 0 0; }
.tvl-ruler svg { display: block; width: 100%; cursor: grab; touch-action: none; }
.tvl-ruler svg:active { cursor: grabbing; }
.tvl-ruler__tick { stroke: var(--home-reflection); stroke-width: 1; }
.tvl-ruler__axis { stroke: var(--home-edge); stroke-width: 1; }
.tvl-ruler__label { fill: var(--home-secondary); font-size: 12px; text-anchor: middle; }
.tvl-ruler__label.is-year { fill: var(--home-secondary); font-weight: 600; }
.tvl-ruler__dot { fill: hsl(var(--h) 55% 52%); opacity: .75; }
.tvl-ruler__dot.is-milestone { opacity: 1; stroke: var(--home-canvas); stroke-width: 1.5; }
.tvl-ruler__dot.is-selected { opacity: 1; stroke: var(--home-link); stroke-width: 2; }
.tvl-ruler__hint { margin: 2px 0 0; color: var(--home-muted); font-size: 11px; }
@media (max-width: 640px) { .tvl-ruler__hint { display: none; } }
:global(html[data-theme='dark']) .tvl-ruler__dot { fill: hsl(var(--h) 55% 66%); }
</style>
