<template>
  <PageShell class="fx-lab" variant="standard">
    <PageHeader title="Visual Effects Lab" description="管理员专属：浏览可复用效果与动效、调参、复制配置；设置只影响本页，不写入正式页面。">
      <template #actions>
        <div class="fx-lab__modes">
          <el-radio-group v-model="mode" size="small" aria-label="实验室分类">
            <el-radio-button value="effects">效果</el-radio-button>
            <el-radio-button value="motion">动效</el-radio-button>
          </el-radio-group>
          <span class="fx-lab__badge">管理员专属</span>
        </div>
      </template>
    </PageHeader>

    <div v-if="mode === 'effects'" class="fx-lab__grid">
      <aside class="fx-lab__list" aria-label="效果目录">
        <button
          v-for="item in EFFECTS"
          :key="item.id"
          type="button"
          class="fx-lab__item"
          :class="{ 'is-active': effectId === item.id }"
          :aria-pressed="effectId === item.id"
          :data-testid="`fx-effect-${item.id}`"
          @click="selectEffect(item.id)"
        >
          <span class="fx-lab__item-head">
            <span class="fx-lab__item-name">{{ item.name }}</span>
            <span class="fx-lab__kind" :data-kind="item.tech">{{ techLabel(item.tech) }}</span>
          </span>
          <span class="fx-lab__item-summary">{{ item.summary }}</span>
          <span class="fx-lab__item-where">适用：{{ item.where }}</span>
        </button>
      </aside>

      <section class="fx-lab__stage-col" aria-label="效果预览">
        <div class="fx-lab__toolbar">
          <span class="fx-lab__label">启用</span>
          <el-switch v-model="enabled" data-testid="fx-enabled" aria-label="启用效果" />
          <span class="fx-lab__label">动画</span>
          <el-switch v-model="animate" data-testid="fx-animate" aria-label="动画" />
          <span class="fx-lab__label">对比</span>
          <el-switch v-model="compare" data-testid="fx-compare" aria-label="对比静态与动态" />
          <el-button v-if="current.id === 'soft-reveal'" size="small" @click="replay += 1">重播</el-button>
        </div>

        <div class="fx-lab__previews" :class="{ 'is-split': compare }">
          <div class="fx-lab__panel">
            <p class="fx-lab__panel-tag">效果开</p>
            <EffectPreview
              :effect="current"
              :params="currentParams"
              :enabled="enabled"
              :animate="animate"
              :reduced="reducedPreview"
              :replay="replay"
              @metrics="onMetrics"
            />
          </div>
          <div v-if="compare" class="fx-lab__panel">
            <p class="fx-lab__panel-tag">效果关</p>
            <EffectPreview
              :effect="current"
              :params="currentParams"
              :enabled="false"
              :animate="animate"
              :reduced="reducedPreview"
              :replay="replay"
            />
          </div>
        </div>

        <p class="fx-lab__tagline">真实 LifeOS 卡片与列表材质 · 示例内容，不读取真实记录</p>
        <p class="fx-lab__tech">
          <span>{{ techLine }}</span>
          <span>context 数 {{ contexts }}</span>
        </p>
      </section>

      <aside class="fx-lab__side" aria-label="参数与配置">
        <header class="fx-lab__side-head">
          <h2>{{ current.name }}</h2>
          <span class="fx-lab__kind" :data-kind="current.tech">{{ techLabel(current.tech) }}</span>
        </header>
        <p class="fx-lab__side-summary">{{ current.summary }}</p>
        <p class="fx-lab__meta">入口：{{ current.entry }}</p>

        <div class="fx-lab__params">
          <div v-for="param in current.params" :key="param.key" class="fx-lab__param" :data-testid="`fx-param-${param.key}`">
            <label class="fx-lab__param-label">
              {{ param.label }}<span v-if="param.unit">（{{ param.unit }}）</span>
            </label>
            <div class="fx-lab__param-control">
              <el-slider
                v-if="param.control === 'range'"
                :model-value="Number(currentParams[param.key])"
                :min="param.min"
                :max="param.max"
                :step="param.step"
                :aria-label="param.label"
                @update:model-value="(value: number | number[]) => onRange(param.key, value)"
              />
              <el-radio-group
                v-else-if="param.control === 'select'"
                :model-value="String(currentParams[param.key])"
                size="small"
                :aria-label="param.label"
                @update:model-value="(value: string | number | boolean) => onSelect(param.key, value)"
              >
                <el-radio-button v-for="option in param.options" :key="option.value" :value="option.value">{{ option.label }}</el-radio-button>
              </el-radio-group>
              <el-switch
                v-else
                :model-value="Boolean(currentParams[param.key])"
                :aria-label="param.label"
                @update:model-value="(value: string | number | boolean) => onToggle(param.key, value)"
              />
            </div>
          </div>
        </div>

        <div class="fx-lab__actions">
          <el-button size="small" data-testid="fx-reset" @click="resetParams">恢复默认</el-button>
          <el-button size="small" data-testid="fx-copy-config" @click="copyText(configText)">复制配置</el-button>
          <el-button size="small" data-testid="fx-copy-usage" @click="copyText(usageText)">复制示例</el-button>
        </div>

        <div class="fx-lab__side-row">
          <span class="fx-lab__label">主题</span>
          <el-radio-group v-model="theme" size="small" aria-label="主题">
            <el-radio-button value="light">浅色</el-radio-button>
            <el-radio-button value="dark">深色</el-radio-button>
          </el-radio-group>
        </div>
        <div class="fx-lab__side-row">
          <span class="fx-lab__label">模拟减少动态效果</span>
          <el-switch v-model="reducedPreview" data-testid="fx-reduced" aria-label="模拟减少动态效果" />
        </div>

        <div class="fx-lab__code">
          <p class="fx-lab__code-title">使用示例</p>
          <pre data-testid="fx-usage">{{ usageText }}</pre>
        </div>
        <div class="fx-lab__code">
          <p class="fx-lab__code-title">当前配置</p>
          <pre data-testid="fx-config">{{ configText }}</pre>
        </div>
        <p class="fx-lab__note">减少动态效果：{{ current.reducedMotion }}</p>
        <p class="fx-lab__note">降级：{{ current.fallback }}</p>
      </aside>
    </div>

    <MotionGallery v-else />
    <p class="fx-lab__foot">
      本页仅管理员可见，实验设置不写入正式页面，也不会向普通用户开放。
      其他实验：<RouterLink to="/lab/webgl">场景实验室</RouterLink> · <RouterLink to="/lab/timeline">时间线实验室</RouterLink>。
    </p>
  </PageShell>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import PageHeader from '@/components/PageHeader.vue'
import PageShell from '@/components/PageShell.vue'
import EffectPreview from '@/components/lab/effects/EffectPreview.vue'
import MotionGallery from '@/components/lab/motion/MotionGallery.vue'
import {
  EFFECTS,
  defaultsOf,
  findEffect,
  normalizeParams,
  type EffectId,
  type EffectTech,
  type ParamRecord,
  type ParamValue,
} from '@/effects/catalog'
import { rendererInstanceCount, type RenderMetrics } from '@/visual-engine'
import '@/effects/effects.css'

const route = useRoute()

const mode = ref<'effects' | 'motion'>('effects')
const effectId = ref<EffectId>('spotlight-surface')
const current = computed(() => findEffect(effectId.value))
const params = reactive<Record<EffectId, ParamRecord>>(
  Object.fromEntries(EFFECTS.map((effect) => [effect.id, defaultsOf(effect)])) as Record<EffectId, ParamRecord>,
)
const currentParams = computed(() => params[effectId.value])

const enabled = ref(true)
const animate = ref(true)
const compare = ref(false)
const reducedPreview = ref(false)
const replay = ref(0)
const theme = ref<'light' | 'dark'>(document.documentElement.dataset.theme === 'light' ? 'light' : 'dark')
const metrics = ref<RenderMetrics | null>(null)
const contexts = ref(0)
let previousTheme: string | null = null

const techLabel = (tech: EffectTech): string => (tech === 'webgl2' ? 'WebGL2' : tech === 'vue' ? 'Vue' : 'CSS')

const configText = computed(() => JSON.stringify(currentParams.value, null, 2))
const usageText = computed(() => current.value.usage(currentParams.value))
const techLine = computed(() => {
  if (current.value.id !== 'ambient-gradient') return `${techLabel(current.value.tech)} · 无渲染上下文`
  const fps = metrics.value?.fps
  const gpu = metrics.value?.gpuMs
  const dpr = metrics.value?.pixelRatio
  return `WebGL2 · ${currentParams.value.quality} · DPR ${dpr ?? '—'} · ${fps === null || fps === undefined ? '采样中' : `${fps} fps`} · GPU ${gpu === null || gpu === undefined ? '—' : `${gpu.toFixed(2)} ms`}`
})

function selectEffect(id: EffectId): void {
  effectId.value = id
  metrics.value = null
  replay.value += 1
}

function setParam(key: string, value: ParamValue): void {
  Object.assign(params[effectId.value], normalizeParams(current.value, { ...currentParams.value, [key]: value }))
}

function onRange(key: string, value: number | number[]): void {
  setParam(key, Number(Array.isArray(value) ? value[0] : value))
}

function onSelect(key: string, value: string | number | boolean): void {
  setParam(key, String(value))
}

function onToggle(key: string, value: string | number | boolean): void {
  setParam(key, Boolean(value))
}

function resetParams(): void {
  Object.assign(params[effectId.value], defaultsOf(current.value))
}

async function copyText(text: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制')
  } catch {
    ElMessage.warning('复制失败，请手动选择文本')
  }
}

function onMetrics(next: RenderMetrics): void {
  metrics.value = next
  contexts.value = rendererInstanceCount()
}

function applyTheme(): void {
  document.documentElement.dataset.theme = theme.value
}

watch(theme, applyTheme)

onMounted(() => {
  previousTheme = document.documentElement.dataset.theme ?? null
  applyTheme()
  const requested = route.query.effect
  if (typeof requested === 'string' && EFFECTS.some((effect) => effect.id === requested)) selectEffect(requested as EffectId)
  contexts.value = rendererInstanceCount()
})

onBeforeUnmount(() => {
  if (previousTheme) document.documentElement.dataset.theme = previousTheme
})
</script>

<style scoped>
.fx-lab { display: grid; gap: var(--space-4); }
.fx-lab__modes { display: flex; align-items: center; gap: 10px; }
.fx-lab__badge {
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
.fx-lab__grid {
  display: grid;
  grid-template-columns: 220px minmax(0, 1fr) 320px;
  gap: var(--space-4);
  align-items: start;
}
.fx-lab__list { display: grid; gap: 8px; }
.fx-lab__item {
  display: grid;
  gap: 6px;
  padding: 12px 14px;
  border: 1px solid var(--home-edge);
  border-radius: var(--radius-card);
  background: var(--home-panel);
  color: var(--home-ink);
  text-align: left;
  font: inherit;
  cursor: pointer;
}
.fx-lab__item:hover { background: var(--home-panel-hover); }
.fx-lab__item:focus-visible { outline: 2px solid var(--home-link); outline-offset: 2px; }
.fx-lab__item.is-active { border-color: var(--home-link); }
.fx-lab__item-head { display: flex; align-items: center; justify-content: space-between; gap: 6px; }
.fx-lab__item-name { font-size: var(--font-meta); font-weight: 600; }
.fx-lab__item-summary { color: var(--home-secondary); font-size: 12px; line-height: 1.55; }
.fx-lab__item-where { color: var(--home-muted); font-size: 11px; line-height: 1.5; }
.fx-lab__kind {
  padding: 1px 7px;
  border: 1px solid var(--home-edge);
  border-radius: var(--radius-pill);
  color: var(--home-muted);
  font-size: 11px;
  letter-spacing: .02em;
  white-space: nowrap;
}
.fx-lab__kind[data-kind='webgl2'] { color: var(--home-link); border-color: color-mix(in srgb, var(--home-link) 35%, transparent); }
.fx-lab__stage-col { display: grid; gap: 8px; min-width: 0; }
.fx-lab__toolbar { display: flex; flex-wrap: wrap; align-items: center; gap: 10px; }
.fx-lab__previews { display: grid; gap: 12px; }
.fx-lab__previews.is-split { grid-template-columns: repeat(2, minmax(0, 1fr)); }
.fx-lab__panel { display: grid; gap: 6px; min-width: 0; }
.fx-lab__panel-tag { margin: 0; color: var(--home-muted); font-size: var(--font-caption); }
.fx-lab__tagline { margin: 0; color: var(--home-muted); font-size: var(--font-caption); }
.fx-lab__tech { display: flex; flex-wrap: wrap; gap: 14px; margin: 0; color: var(--home-muted); font-size: var(--font-caption); }
.fx-lab__side {
  display: grid;
  gap: 10px;
  padding: 16px;
  border: 1px solid var(--home-edge);
  border-radius: var(--radius-card);
  background: var(--home-panel);
}
.fx-lab__side-head { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
.fx-lab__side-head h2 { margin: 0; font-size: var(--font-card-title); font-weight: 600; }
.fx-lab__side-summary { margin: 0; color: var(--home-secondary); font-size: var(--font-meta); line-height: 1.7; }
.fx-lab__meta { margin: 0; color: var(--home-muted); font-size: var(--font-caption); }
.fx-lab__params { display: grid; gap: 10px; padding-top: 4px; border-top: 1px solid var(--home-edge); }
.fx-lab__param { display: grid; gap: 6px; }
.fx-lab__param-label { color: var(--home-secondary); font-size: var(--font-caption); }
.fx-lab__param-control { display: flex; align-items: center; gap: 8px; }
.fx-lab__actions { display: flex; flex-wrap: wrap; gap: 8px; }
.fx-lab__side-row { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
.fx-lab__label { color: var(--home-secondary); font-size: var(--font-caption); }
.fx-lab__code { display: grid; gap: 4px; }
.fx-lab__code-title { margin: 0; color: var(--home-secondary); font-size: var(--font-caption); }
.fx-lab__code pre {
  max-height: 168px;
  margin: 0;
  padding: 10px 12px;
  overflow: auto;
  border: 1px solid var(--home-edge);
  border-radius: 10px;
  background: var(--home-canvas);
  color: var(--home-ink);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 11px;
  line-height: 1.6;
  white-space: pre;
}
.fx-lab__note { margin: 0; color: var(--home-muted); font-size: var(--font-caption); line-height: 1.6; }
.fx-lab__foot { margin: 0; color: var(--home-muted); font-size: var(--font-caption); line-height: 1.7; }
.fx-lab__foot a { color: var(--home-link); text-decoration: none; }
.fx-lab__foot a:hover { text-decoration: underline; text-underline-offset: 3px; }
@media (max-width: 1100px) {
  .fx-lab__grid { grid-template-columns: minmax(0, 1fr); }
  .fx-lab__previews.is-split { grid-template-columns: minmax(0, 1fr); }
}
</style>
