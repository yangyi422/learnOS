<template>
  <div
    class="fx-preview"
    data-testid="fx-preview"
    :data-fx-frozen="animate ? undefined : ''"
    :data-fx-reduced="reduced ? '' : undefined"
    :style="stageVars"
  >
    <AmbientSurface
      v-if="isAmbient"
      :enabled="ambientEnabled"
      :strength="Number(params.strength)"
      :quality="ambientQuality"
      @metrics="(value) => emit('metrics', value)"
    />
    <HomeLightField v-else />

    <div class="fx-preview__content">
      <SoftReveal
        v-if="revealOn"
        :trigger="replay"
        :duration="Number(params.duration)"
        :delay="Number(params.delay)"
        :distance="Number(params.distance)"
        :easing="revealEasing"
        :play-on-mount="true"
      >
        <SurfaceDepth
          v-for="card in cards"
          :key="card.title"
          class="fx-card"
          :background="depth.background"
          :edge="depth.edge"
          :shadow="depth.shadow"
          v-fx-spotlight="spotlightOn ? Number(params.intensity) : undefined"
        >
          <header>
            <span class="fx-card__eyebrow">{{ card.eyebrow }}</span>
            <h3>{{ card.title }}</h3>
          </header>
          <p>{{ card.body }}</p>
          <footer>
            <span>{{ card.meta }}</span>
            <button type="button" class="fx-card__action">打开</button>
          </footer>
        </SurfaceDepth>
      </SoftReveal>
      <template v-else>
        <SurfaceDepth
          v-for="card in cards"
          :key="card.title"
          class="fx-card"
          :background="depth.background"
          :edge="depth.edge"
          :shadow="depth.shadow"
          v-fx-spotlight="spotlightOn ? Number(params.intensity) : undefined"
        >
          <header>
            <span class="fx-card__eyebrow">{{ card.eyebrow }}</span>
            <h3>{{ card.title }}</h3>
          </header>
          <p>{{ card.body }}</p>
          <footer>
            <span>{{ card.meta }}</span>
            <button type="button" class="fx-card__action">打开</button>
          </footer>
        </SurfaceDepth>
      </template>

      <ol class="fx-rows">
        <li v-for="row in rows" :key="row.title">
          <span class="fx-rows__time">{{ row.time }}</span>
          <span class="fx-rows__title">{{ row.title }}</span>
          <span class="fx-rows__meta">{{ row.meta }}</span>
        </li>
      </ol>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import HomeLightField from '@/components/HomeLightField.vue'
import AmbientSurface from '@/effects/AmbientSurface.vue'
import SoftReveal from '@/effects/SoftReveal.vue'
import SurfaceDepth from '@/effects/SurfaceDepth.vue'
import { easingValue, type EffectDefinition, type ParamRecord } from '@/effects/catalog'
import { fxSpotlight } from '@/effects/spotlight'
import type { QualityTier, RenderMetrics } from '@/visual-engine'

const vFxSpotlight = fxSpotlight

const props = defineProps<{
  effect: EffectDefinition
  params: ParamRecord
  enabled: boolean
  animate: boolean
  reduced: boolean
  replay: number
}>()

const emit = defineEmits<{ metrics: [metrics: RenderMetrics] }>()

const isAmbient = computed(() => props.effect.id === 'ambient-gradient')
const ambientEnabled = computed(() => props.enabled && props.animate && !props.reduced)
const spotlightOn = computed(() => props.effect.id === 'spotlight-surface' && props.enabled && props.animate && !props.reduced)
const revealOn = computed(() => props.effect.id === 'soft-reveal' && props.enabled && props.animate && !props.reduced)
const ambientQuality = computed(() => String(props.params.quality) as QualityTier)
const revealEasing = computed(() => String(props.params.easing) as 'standard' | 'gentle' | 'snappy')

const depth = computed(() => (props.effect.id === 'surface-depth'
  ? { background: Number(props.params.background), edge: Number(props.params.edge), shadow: Number(props.params.shadow) }
  : { background: 0.7, edge: 0.5, shadow: 0.6 }))

const stageVars = computed((): Record<string, string> => {
  const p = props.params
  if (props.effect.id === 'spotlight-surface') {
    return {
      '--fx-intensity': String(p.intensity),
      '--fx-spotlight-radius': `${p.radius}px`,
      '--fx-spotlight-speed': `${p.speed}ms`,
      '--fx-spotlight-color': p.tone === 'accent' ? 'var(--fx-spotlight-accent)' : 'var(--fx-spotlight-color)',
    }
  }
  if (props.effect.id === 'soft-reveal') {
    return {
      '--fx-reveal-duration': `${p.duration}ms`,
      '--fx-reveal-delay': `${p.delay}ms`,
      '--fx-reveal-distance': `${p.distance}px`,
      '--fx-reveal-ease': easingValue(String(p.easing)),
    }
  }
  if (props.effect.id === 'surface-depth') {
    return {
      '--fx-depth-bg': String(p.background),
      '--fx-depth-edge': String(p.edge),
      '--fx-depth-shadow': String(p.shadow),
    }
  }
  return {}
})

const cards = [
  { eyebrow: '今日焦点', title: '把视觉特效整理成可复用的基础件', body: '先在实验页验证强度与性能，再决定是否进入正式页面。', meta: '工作与创造' },
  { eyebrow: '正在推进', title: '视觉效果组件库的最小实现', body: 'Spotlight、Reveal、Surface Depth 与环境光，只保留真正有用的四个。', meta: '6 / 10 项任务' },
]

const rows = [
  { time: '10/21', title: '整理效果参数与降级策略', meta: '工作' },
  { time: '10/19', title: '评审浅色主题下的反光强度', meta: '设计' },
  { time: '10/17', title: '记录一次实验结论', meta: '学习' },
]
</script>

<style scoped>
.fx-preview {
  position: relative;
  min-height: 360px;
  padding: 24px;
  border: 1px solid var(--home-edge);
  border-radius: var(--radius-card);
  background: var(--home-canvas);
  overflow: hidden;
}
.fx-preview__content { position: relative; z-index: 1; display: grid; gap: 12px; }
.fx-card { display: grid; gap: 8px; padding: 16px 18px; color: var(--home-ink); }
.fx-card header { display: grid; gap: 4px; }
.fx-card__eyebrow { color: var(--home-muted); font-size: var(--font-caption); }
.fx-card h3 { margin: 0; font-size: var(--font-card-title); font-weight: 600; line-height: 1.45; }
.fx-card p { margin: 0; color: var(--home-secondary); font-size: var(--font-meta); line-height: 1.7; }
.fx-card footer { display: flex; align-items: center; justify-content: space-between; color: var(--home-muted); font-size: var(--font-caption); }
.fx-card__action { padding: 2px 0; border: 0; background: none; color: var(--home-link); font: inherit; font-size: var(--font-caption); cursor: pointer; }
.fx-card__action:focus-visible { outline: 2px solid var(--home-link); outline-offset: 2px; }
.fx-rows { display: grid; gap: 0; margin: 4px 0 0; padding: 0; list-style: none; }
.fx-rows li { display: grid; grid-template-columns: 56px minmax(0, 1fr) auto; gap: 12px; padding: 10px 4px; border-bottom: 1px solid var(--home-edge); color: var(--home-secondary); font-size: var(--font-meta); }
.fx-rows li:last-child { border-bottom: 0; }
.fx-rows__time { color: var(--home-muted); font-variant-numeric: tabular-nums; }
.fx-rows__title { color: var(--home-ink); }
.fx-rows__meta { color: var(--home-muted); }
@media (max-width: 640px) {
  .fx-preview { padding: 16px; }
  .fx-rows li { grid-template-columns: 44px minmax(0, 1fr); }
  .fx-rows__meta { display: none; }
}
</style>
