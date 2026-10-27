<template>
  <PageShell class="visual-lab" variant="standard">
    <PageHeader title="WebGL Visual Lab" description="验证一层着色器环境光能否进入 LifeOS，且不影响其他页面。">
      <template #actions><span class="visual-lab-badge">仅本地开发 · 未注册正式路由</span></template>
    </PageHeader>

    <section class="visual-lab-stage-frame" aria-labelledby="visual-lab-stage-title">
      <header class="visual-lab-stage-head">
        <h2 id="visual-lab-stage-title">渲染舞台</h2>
        <p class="visual-lab-state" role="status">{{ statusLabel }}</p>
      </header>
      <VisualLabStage
        :key="surfaceKey"
        :scene="scene"
        :light="light"
        :strength="strength"
        :quality="quality"
        :running="running"
        :reduced="reduced"
        :supported="capability.supported"
        @metrics="updateMetrics"
        @context="updateContext"
        @failure="markFailure"
      />
      <p class="visual-lab-stage-note">{{ sceneNote }}</p>
    </section>

    <div class="visual-lab-grid">
      <section class="visual-lab-panel" aria-labelledby="visual-lab-controls-title">
        <h2 id="visual-lab-controls-title">控制</h2>
        <div class="visual-lab-row">
          <span class="visual-lab-row__label">渲染</span>
          <el-switch v-model="running" :disabled="!capability.supported" aria-label="渲染" />
        </div>
        <div class="visual-lab-row">
          <span class="visual-lab-row__label">场景</span>
          <el-radio-group :model-value="scene" size="small" :disabled="!capability.supported" aria-label="场景" @update:model-value="changeScene">
            <el-radio-button value="ambient">A · 空间光场</el-radio-button>
            <el-radio-button value="material">B · 材质</el-radio-button>
          </el-radio-group>
        </div>
        <div class="visual-lab-row">
          <span class="visual-lab-row__label">强度</span>
          <el-slider v-model="strength" :min="0" :max="1" :step="0.05" :disabled="!capability.supported" aria-label="强度" />
          <span class="visual-lab-row__value">{{ strength.toFixed(2) }}</span>
        </div>
        <div class="visual-lab-row">
          <span class="visual-lab-row__label">质量</span>
          <el-radio-group :model-value="quality" size="small" :disabled="!capability.supported" aria-label="质量" @update:model-value="changeQuality">
            <el-radio-button value="high">高</el-radio-button>
            <el-radio-button value="balanced">均衡</el-radio-button>
            <el-radio-button value="static">静态</el-radio-button>
          </el-radio-group>
        </div>
        <div class="visual-lab-row">
          <span class="visual-lab-row__label">主题</span>
          <el-radio-group :model-value="preference" size="small" aria-label="主题" @update:model-value="applyTheme">
            <el-radio-button value="light">浅色</el-radio-button>
            <el-radio-button value="dark">深色</el-radio-button>
            <el-radio-button value="system">跟随系统</el-radio-button>
          </el-radio-group>
        </div>
        <p class="visual-lab-hint">「静态」只渲染一帧；系统开启「减少动态效果」时默认暂停，可用渲染开关手动开启。主题写入与「外观设置」相同的浏览器偏好。</p>
        <el-button v-if="failed" class="visual-lab-retry" @click="retry">重新初始化</el-button>
      </section>

      <section class="visual-lab-panel" aria-labelledby="visual-lab-metrics-title">
        <h2 id="visual-lab-metrics-title">运行指标</h2>
        <dl class="visual-lab-metrics">
          <div><dt>帧率</dt><dd>{{ metrics.fps === null ? '—' : `${metrics.fps} fps` }}</dd></div>
          <div><dt>绘制</dt><dd>{{ metrics.drawCalls }} 次/帧</dd></div>
          <div><dt>宽高比</dt><dd>{{ aspectLabel }}</dd></div>
          <div><dt>画布</dt><dd>{{ canvasLabel }}</dd></div>
          <div><dt>像素比</dt><dd>{{ metrics.pixelRatio.toFixed(2) }}</dd></div>
          <div><dt>GPU 计时</dt><dd>{{ gpuLabel }}</dd></div>
          <div><dt>上下文</dt><dd>{{ contextLabel }}</dd></div>
        </dl>
        <p class="visual-lab-hint">帧率按 1 秒窗口采样，只更新这一块，不触发整页重渲染。GPU 计时来自 EXT_disjoint_timer_query_webgl2；扩展缺失或结果 disjoint 时显示「不可用」。</p>
      </section>
    </div>

    <p class="visual-lab-foot">本页只验证渲染架构：不进入导航，生产构建不注册该路由，可随目录整块删除。</p>
  </PageShell>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import PageHeader from '@/components/PageHeader.vue'
import PageShell from '@/components/PageShell.vue'
import VisualLabStage from '@/components/lab/VisualLabStage.vue'
import { setThemePreference, useTheme, type ThemePreference } from '@/theme'
import {
  detectCapabilities,
  readSurfaceLight,
  subscribeReducedMotion,
  subscribeThemeChanges,
  type ContextState,
  type QualityTier,
  type RenderMetrics,
  type SceneId,
  type SurfaceLight,
} from '@/visual-engine'

const capability = detectCapabilities()
const { preference } = useTheme()

const reduced = ref(capability.reducedMotion)
/** Reduced motion defaults to a still frame; the switch is the explicit opt-in. */
const running = ref(!capability.reducedMotion)
const scene = ref<SceneId>('ambient')
const strength = ref(1)
const quality = ref<QualityTier>('high')
const light = ref<SurfaceLight>(readSurfaceLight())
const metrics = ref<RenderMetrics>({
  fps: null,
  drawCalls: 1,
  width: 0,
  height: 0,
  aspect: 0,
  pixelRatio: 1,
  gpuAvailable: false,
  gpuMs: null,
  context: 'ok',
})
const contextState = ref<ContextState>('ok')
const failure = ref('')
const surfaceKey = ref(0)
const failed = computed(() => Boolean(failure.value) || contextState.value === 'unavailable')

let stopThemeWatch: (() => void) | null = null
let stopReducedWatch: (() => void) | null = null

const canvasLabel = computed(() => (metrics.value.width ? `${metrics.value.width}×${metrics.value.height}` : '—'))
const aspectLabel = computed(() => (metrics.value.aspect ? metrics.value.aspect.toFixed(2) : '—'))
const gpuLabel = computed(() => {
  if (!metrics.value.gpuAvailable) return '不可用'
  return metrics.value.gpuMs === null ? '测量中' : `${metrics.value.gpuMs.toFixed(2)} ms`
})
const contextLabel = computed(() => {
  if (!capability.supported) return '不可用'
  if (contextState.value === 'lost') return '已丢失'
  if (contextState.value === 'unavailable') return '不可用'
  return '正常'
})
const sceneNote = computed(() => (scene.value === 'ambient'
  ? '方案 A · Spatial Ambient：三层光场，线性衰减与现有 CSS 光场对齐，只增加极慢的位移与亮度呼吸。'
  : '方案 B · Material Depth：程序化材质，仅存在于这块舞台内，不铺到页面或任何业务界面。'))

const statusLabel = computed(() => {
  if (!capability.supported) return '当前浏览器不支持 WebGL2 · 显示静态画面'
  if (failure.value) return '着色器初始化失败 · 已显示静态画面'
  if (contextState.value === 'unavailable') return '图形上下文未能恢复 · 已显示静态画面'
  if (contextState.value === 'lost') return '图形上下文已丢失 · 等待恢复'
  if (!running.value) return '已暂停 · 停留在最后一帧'
  if (quality.value === 'static') return '静态模式 · 只渲染一帧'
  if (reduced.value) return '运行中 · 已手动开启动画（系统要求减少动态效果）'
  return metrics.value.fps === null ? '运行中 · 采样中' : `运行中 · ${metrics.value.fps} fps`
})

function updateMetrics(next: RenderMetrics): void {
  metrics.value = next
}

function updateContext(state: ContextState): void {
  contextState.value = state
}

function markFailure(message: string): void {
  failure.value = message
}

function changeScene(value: string | number | boolean | undefined): void {
  if (value === 'ambient' || value === 'material') scene.value = value
}

function changeQuality(value: string | number | boolean | undefined): void {
  if (value === 'high' || value === 'balanced' || value === 'static') quality.value = value
}

function applyTheme(value: string | number | boolean | undefined): void {
  if (value === 'light' || value === 'dark' || value === 'system') setThemePreference(value as ThemePreference)
}

function retry(): void {
  failure.value = ''
  contextState.value = 'ok'
  surfaceKey.value += 1
}

onMounted(() => {
  stopThemeWatch = subscribeThemeChanges(() => { light.value = readSurfaceLight() })
  stopReducedWatch = subscribeReducedMotion((next) => {
    reduced.value = next
    // The system preference always wins by pausing; the user can opt back in.
    if (next) running.value = false
  })
})

onBeforeUnmount(() => {
  stopThemeWatch?.()
  stopReducedWatch?.()
})
</script>

<style scoped>
.visual-lab { display: grid; gap: var(--space-6); }
.visual-lab-badge {
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
.visual-lab-stage-frame {
  padding: 16px;
  border: 1px solid var(--home-edge);
  border-radius: var(--radius-card);
  background: var(--home-panel);
  box-shadow: var(--home-panel-inset);
}
.visual-lab-stage-head { display: flex; align-items: baseline; justify-content: space-between; gap: var(--space-4); margin-bottom: var(--space-3); }
.visual-lab-stage-head h2 { margin: 0; color: var(--home-ink); font-size: var(--font-card-title); font-weight: 700; }
.visual-lab-state { margin: 0; color: var(--home-secondary); font-size: var(--font-meta); }
.visual-lab-stage-note { margin: var(--space-3) 0 0; color: var(--home-muted); font-size: var(--font-caption); line-height: 1.6; }
.visual-lab-grid { display: grid; grid-template-columns: minmax(0, 1.35fr) minmax(0, 1fr); gap: var(--space-6); align-items: start; }
.visual-lab-panel {
  padding: 20px 24px;
  border: 1px solid var(--home-edge);
  border-radius: var(--radius-card);
  background: var(--home-panel);
  box-shadow: var(--home-panel-inset);
}
.visual-lab-panel h2 { margin: 0 0 var(--space-2); color: var(--home-ink); font-size: var(--font-card-title); font-weight: 700; }
.visual-lab-row { display: flex; flex-wrap: wrap; align-items: center; gap: var(--space-4); min-height: 48px; border-top: 1px solid var(--home-edge); }
.visual-lab-row__label { flex: none; width: 3.5em; color: var(--home-secondary); font-size: var(--font-meta); }
.visual-lab-row__value { flex: none; width: 3em; color: var(--home-muted); font-size: var(--font-meta); text-align: right; font-variant-numeric: tabular-nums; }
.visual-lab-row :deep(.el-slider) { flex: 1 1 120px; min-width: 120px; }
.visual-lab-row :deep(.el-radio-group) { margin-left: auto; }
.visual-lab-hint { margin: var(--space-3) 0 0; color: var(--home-muted); font-size: var(--font-caption); line-height: 1.6; }
.visual-lab-retry { margin-top: var(--space-3); }
.visual-lab-metrics { display: grid; grid-template-columns: 1fr; margin: 0; }
.visual-lab-metrics div { display: flex; align-items: baseline; justify-content: space-between; gap: var(--space-3); min-height: 30px; border-top: 1px solid var(--home-edge); }
.visual-lab-metrics div:first-child { border-top: 0; }
.visual-lab-metrics dt { margin: 0; color: var(--home-muted); font-size: var(--font-meta); }
.visual-lab-metrics dd { margin: 0; color: var(--home-ink); font-size: var(--font-meta); font-variant-numeric: tabular-nums; }
.visual-lab-foot { margin: 0; color: var(--home-muted); font-size: var(--font-caption); line-height: 1.6; }
@media (max-width: 900px) {
  .visual-lab-grid { grid-template-columns: 1fr; }
}
@media (prefers-reduced-motion: reduce) {
  .visual-lab-row :deep(.el-switch__core) { transition: none; }
}
</style>
