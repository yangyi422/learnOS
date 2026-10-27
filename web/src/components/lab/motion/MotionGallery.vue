<template>
  <div class="mg">
    <aside class="mg__list" aria-label="动效目录">
      <button
        v-for="item in MOTIONS"
        :key="item.id"
        type="button"
        class="mg__item"
        :class="{ 'is-active': motionId === item.id }"
        :aria-pressed="motionId === item.id"
        :data-testid="`mg-motion-${item.id}`"
        @click="selectMotion(item.id)"
      >
        <span class="mg__item-head">
          <span class="mg__item-name">{{ item.name }}</span>
          <span class="mg__kind">{{ item.tech }}</span>
        </span>
        <span class="mg__item-summary">{{ item.summary }}</span>
        <span class="mg__item-where">适用：{{ item.where }}</span>
      </button>
    </aside>

    <section class="mg__stage-col" aria-label="动效预览">
      <div class="mg__previews" :class="{ 'is-split': compare }">
        <div class="mg__panel">
          <p class="mg__panel-tag">当前参数</p>
          <MotionStage :motion="current" :params="params" :reduced="reduced" :replay="replay" />
        </div>
        <div v-if="compare" class="mg__panel">
          <p class="mg__panel-tag">默认参数</p>
          <MotionStage :motion="current" :params="DEFAULT_MOTION" :reduced="reduced" :replay="replay" />
        </div>
      </div>
      <p class="mg__tagline">真实组件预览 · 参数只作用于本页</p>
    </section>

    <aside class="mg__side" aria-label="动效参数">
      <header class="mg__head">
        <h2>{{ current.name }}</h2>
        <span class="mg__kind">{{ current.tech }}</span>
      </header>
      <p class="mg__summary">{{ current.summary }}</p>
      <p class="mg__meta">入口：{{ current.primitive }}</p>

      <div v-if="uses('duration')" class="mg__row">
        <span class="mg__label">时长</span>
        <el-radio-group v-model="params.duration" size="small" data-testid="mg-duration" aria-label="时长">
          <el-radio-button :value="100">100ms</el-radio-button>
          <el-radio-button :value="160">160ms</el-radio-button>
          <el-radio-button :value="180">180ms</el-radio-button>
          <el-radio-button :value="300">300ms</el-radio-button>
        </el-radio-group>
      </div>

      <div class="mg__row">
        <span class="mg__label">缓动</span>
        <el-radio-group v-model="params.easing" size="small" data-testid="mg-easing" aria-label="缓动">
          <el-radio-button value="standard">标准</el-radio-button>
          <el-radio-button value="enter">进入</el-radio-button>
          <el-radio-button value="exit">退出</el-radio-button>
        </el-radio-group>
      </div>

      <div v-if="uses('distance')" class="mg__row">
        <span class="mg__label">距离</span>
        <el-slider v-model="params.distance" :min="2" :max="12" :step="2" data-testid="mg-distance" aria-label="位移距离" />
      </div>

      <div class="mg__row">
        <span class="mg__label">模拟减少动态效果</span>
        <el-switch v-model="reduced" data-testid="mg-reduced" aria-label="模拟减少动态效果" />
      </div>
      <div class="mg__row">
        <span class="mg__label">对比默认</span>
        <el-switch v-model="compare" data-testid="mg-compare" aria-label="对比默认参数" />
      </div>
      <div class="mg__row">
        <span class="mg__label">主题</span>
        <el-radio-group v-model="theme" size="small" data-testid="mg-theme" aria-label="主题">
          <el-radio-button value="light">浅色</el-radio-button>
          <el-radio-button value="dark">深色</el-radio-button>
        </el-radio-group>
      </div>

      <div class="mg__actions">
        <el-button size="small" data-testid="mg-replay" @click="replay += 1">重播</el-button>
        <el-button size="small" data-testid="mg-reset" @click="reset">恢复默认</el-button>
        <el-button size="small" data-testid="mg-copy" @click="copyUsage">复制示例</el-button>
      </div>

      <pre data-testid="mg-usage">{{ usageText }}</pre>
      <p class="mg__note">所有动效由 Motion Tokens 驱动；Reduced Motion 下自动关闭。</p>
    </aside>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import MotionStage from '@/components/lab/motion/MotionStage.vue'
import {
  DEFAULT_MOTION,
  MOTIONS,
  findMotion,
  type MotionEase,
  type MotionId,
} from '@/motion/catalog'

const motionId = ref<MotionId>('button-feedback')
const current = computed(() => findMotion(motionId.value))
const params = reactive({ duration: DEFAULT_MOTION.duration, easing: DEFAULT_MOTION.easing as MotionEase, distance: DEFAULT_MOTION.distance })
const reduced = ref(false)
const compare = ref(false)
const replay = ref(0)
const theme = ref<'light' | 'dark'>(document.documentElement.dataset.theme === 'light' ? 'light' : 'dark')
let previousTheme: string | null = null

function uses(param: 'duration' | 'easing' | 'distance'): boolean {
  return current.value.params.includes(param)
}

const usageText = computed(() => {
  const usage = current.value.usage({ duration: params.duration, easing: params.easing, distance: params.distance })
  return `/* CSS */\n${usage.css}\n\n<!-- Vue -->\n${usage.vue}`
})

function selectMotion(id: MotionId): void {
  motionId.value = id
  replay.value += 1
}

function reset(): void {
  params.duration = DEFAULT_MOTION.duration
  params.easing = DEFAULT_MOTION.easing
  params.distance = DEFAULT_MOTION.distance
  reduced.value = false
  compare.value = false
}

async function copyUsage(): Promise<void> {
  try {
    await navigator.clipboard.writeText(usageText.value)
    ElMessage.success('已复制')
  } catch {
    ElMessage.warning('复制失败，请手动选择文本')
  }
}

function applyTheme(): void {
  document.documentElement.dataset.theme = theme.value
}

watch(theme, applyTheme)

onMounted(() => {
  previousTheme = document.documentElement.dataset.theme ?? null
  applyTheme()
})

onBeforeUnmount(() => {
  if (previousTheme) document.documentElement.dataset.theme = previousTheme
})
</script>

<style scoped>
.mg { display: grid; grid-template-columns: 220px minmax(0, 1fr) 320px; gap: var(--space-4); align-items: start; }
.mg__list { display: grid; gap: 8px; }
.mg__item {
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
.mg__item:hover { background: var(--home-panel-hover); }
.mg__item:focus-visible { outline: 2px solid var(--home-link); outline-offset: 2px; }
.mg__item.is-active { border-color: var(--home-link); }
.mg__item-head { display: flex; align-items: center; justify-content: space-between; gap: 6px; }
.mg__item-name { font-size: var(--font-meta); font-weight: 600; }
.mg__item-summary { color: var(--home-secondary); font-size: 12px; line-height: 1.55; }
.mg__item-where { color: var(--home-muted); font-size: 11px; line-height: 1.5; }
.mg__kind { padding: 1px 7px; border: 1px solid var(--home-edge); border-radius: var(--radius-pill); color: var(--home-muted); font-size: 11px; white-space: nowrap; }
.mg__stage-col { display: grid; gap: 8px; min-width: 0; }
.mg__previews { display: grid; gap: 12px; }
.mg__previews.is-split { grid-template-columns: repeat(2, minmax(0, 1fr)); }
.mg__panel { display: grid; gap: 6px; min-width: 0; }
.mg__panel-tag { margin: 0; color: var(--home-muted); font-size: var(--font-caption); }
.mg__tagline { margin: 0; color: var(--home-muted); font-size: var(--font-caption); }
.mg__side { display: grid; gap: 10px; padding: 16px; border: 1px solid var(--home-edge); border-radius: var(--radius-card); background: var(--home-panel); }
.mg__head { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
.mg__head h2 { margin: 0; font-size: var(--font-card-title); font-weight: 600; }
.mg__summary { margin: 0; color: var(--home-secondary); font-size: var(--font-meta); line-height: 1.7; }
.mg__meta { margin: 0; color: var(--home-muted); font-size: var(--font-caption); }
.mg__row { display: flex; align-items: center; justify-content: space-between; gap: 10px; }
.mg__label { color: var(--home-secondary); font-size: var(--font-caption); white-space: nowrap; }
.mg__row :deep(.el-slider) { flex: 1; min-width: 80px; }
.mg__actions { display: flex; flex-wrap: wrap; gap: 8px; }
.mg__side pre {
  max-height: 190px;
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
  white-space: pre-wrap;
}
.mg__note { margin: 0; color: var(--home-muted); font-size: 11px; line-height: 1.5; }
@media (max-width: 1100px) {
  .mg { grid-template-columns: minmax(0, 1fr); }
  .mg__previews.is-split { grid-template-columns: minmax(0, 1fr); }
}
</style>
