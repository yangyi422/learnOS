<template>
  <div ref="host" class="home-ambient-host" :class="{ 'is-home': route.path === '/' }">
    <canvas
      v-if="canvasReady"
      :key="canvasKey"
      ref="canvas"
      class="home-ambient-canvas"
      :class="{ 'is-visible': state === 'webgl' }"
      aria-hidden="true"
      tabindex="-1"
    />
    <HomeLightField v-show="state !== 'webgl'" />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import HomeLightField from '@/components/HomeLightField.vue'
import { useAmbientPreference } from '@/ambient-preference'
import { usePilot } from '@/effects/pilot'
import {
  createRenderer,
  prefersReducedMotion,
  readSurfaceLight,
  renderedFrameCount,
  rendererInstanceCount,
  subscribeReducedMotion,
  subscribeThemeChanges,
  type RenderMetrics,
  type Renderer,
  type SurfaceLight,
} from '@/visual-engine'

/**
 * Home ambient host (Phase 11.3A grayscale).
 *
 * Default is the existing CSS light field. WebGL2 is only attempted on the home
 * route with `?ambient=webgl`, never animates under reduced motion, and falls
 * back to CSS on any failure or context-restore timeout. It owns no business
 * state and never becomes a global singleton.
 */
type AmbientState = 'css' | 'initializing' | 'webgl' | 'degraded' | 'static'
type AmbientMode = 'css' | 'webgl' | 'static'

const route = useRoute()
const host = ref<HTMLElement | null>(null)
const canvas = ref<HTMLCanvasElement | null>(null)
const { preference } = useAmbientPreference()
const { homeRefraction } = usePilot()
/** A disposed renderer loses its context permanently, so each attempt gets a new canvas. */
const canvasReady = ref(false)
const canvasKey = ref(0)
const state = ref<AmbientState>('css')
const reduced = ref(prefersReducedMotion())
const light = ref<SurfaceLight>(readSurfaceLight())

let renderer: Renderer | null = null
let stopThemeWatch: (() => void) | null = null
let stopReducedWatch: (() => void) | null = null
let stopSizeWatch: ResizeObserver | null = null
function refreshLight(): void {
  light.value = readSurfaceLight(host.value ?? document.documentElement)
}

let lastGpuMs: number | null = null
let disposed = false

const urlMode = computed<'webgl' | 'css' | null>(() => {
  const value = route.query.ambient
  return value === 'webgl' || value === 'css' ? value : null
})

/**
 * Priority, highest first:
 *   1. `?ambient=css` — emergency kill switch, always wins.
 *   2. reduced motion — never animate, whatever the preference says.
 *   3. `?ambient=webgl` — one-shot preview without changing the saved choice.
 *   4. saved preference: css (default) or webgl/auto.
 * Other routes are always CSS.
 */
const mode = computed<AmbientMode>(() => {
  if (route.path !== '/') return 'css'
  if (urlMode.value === 'css') return 'css'
  if (reduced.value) return 'static'
  if (urlMode.value === 'webgl') return 'webgl'
  if (homeRefraction.value) return 'webgl'
  return preference.value === 'css' ? 'css' : 'webgl'
})

/**
 * Dev-only injection so the restore-timeout path can be tested without waiting
 * eight seconds. Production always uses the renderer default.
 */
function restoreTimeout(): number | undefined {
  if (!import.meta.env.DEV) return undefined
  const raw = route.query.ambientRestoreTimeout
  const value = Number(Array.isArray(raw) ? raw[0] : raw)
  return Number.isFinite(value) && value > 0 ? value : undefined
}

function teardown(): void {
  renderer?.dispose()
  renderer = null
  lastGpuMs = null
  canvasReady.value = false
}

function degrade(): void {
  if (disposed) return
  teardown()
  state.value = 'degraded'
}

async function sync(): Promise<void> {
  if (disposed) return
  if (mode.value !== 'webgl') {
    teardown()
    state.value = mode.value === 'static' ? 'static' : 'css'
    return
  }
  if (state.value === 'webgl' || state.value === 'initializing') return

  canvasKey.value += 1
  canvasReady.value = true
  await nextTick()
  const element = canvas.value
  if (!element) return
  state.value = 'initializing'
  try {
    renderer = createRenderer(element, {
      scene: 'ambient',
      light: light.value,
      strength: 1,
      quality: 'high',
      restoreTimeoutMs: restoreTimeout(),
      onMetrics: (metrics: RenderMetrics) => { lastGpuMs = metrics.gpuMs },
      onContext: (next) => {
        if (next === 'unavailable') degrade()
      },
      glass: {
        // Soft Mist light keeps the ambient, but has no specular/refraction pass.
        enabled: () => homeRefraction.value && light.value.theme === 'dark',
        getRect: () => {
          const card = document.querySelector('.focus-card') as HTMLElement | null
          if (!card) return null
          const rect = card.getBoundingClientRect()
          if (rect.width < 1 || rect.height < 1) return null
          return { left: rect.left, top: rect.top, width: rect.width, height: rect.height }
        },
        getTheme: () => (document.documentElement.dataset.theme === 'dark' ? 'dark' : 'light'),
        radius: 16,
        strength: 26,
      },
    })
    renderer.start()
    // Only now is the WebGL frame on screen; the CSS field was visible until here.
    state.value = 'webgl'
  } catch {
    degrade()
  }
}

/** Dev-only diagnostics: acceptance tests assert real state, not call counts. */
function exposeDiagnostics(): void {
  if (!import.meta.env.DEV) return
  const target = window as Window & {
    __homeAmbient?: {
      state: () => AmbientState
      mode: () => AmbientMode
      preference: () => string
      instances: () => number
      frames: () => number
      gpuMs: () => number | null
    }
  }
  target.__homeAmbient = {
    state: () => state.value,
    mode: () => mode.value,
    preference: () => preference.value,
    instances: rendererInstanceCount,
    frames: renderedFrameCount,
    gpuMs: () => lastGpuMs,
  }
}

onMounted(() => {
  exposeDiagnostics()
  refreshLight()
  stopThemeWatch = subscribeThemeChanges(refreshLight)
  stopSizeWatch = new ResizeObserver(refreshLight)
  if (host.value) stopSizeWatch.observe(host.value)
  stopReducedWatch = subscribeReducedMotion((next) => { reduced.value = next })
  void sync()
})

onBeforeUnmount(() => {
  disposed = true
  teardown()
  stopThemeWatch?.()
  stopReducedWatch?.()
  stopSizeWatch?.disconnect()
})

watch(() => route.path, async () => { await nextTick(); refreshLight() })
watch(mode, () => { void sync() })
watch(light, (next) => { renderer?.setLight(next) })
</script>

<style scoped>
.home-ambient-host { position: absolute; inset: 0; }
/* Route-local tokens: other workspaces retain their existing ambient. The same
   centers/radii are read by WebGL; reduced motion uses this static CSS field. */
.home-ambient-host.is-home {
  /* Mid-scale mist: a deeper cool base plus three distinct soft zones. Full-width
     ramps of this size read as a flat wall, so the zones stay under 70% wide. */
  --home-environment: linear-gradient(180deg, #edf2fb, #e7eef9 50%, #edf2fb);
  --home-ground-axis: vertical;
  --home-entry-center: 32% 20%;
  --home-entry-size: 62% 46%;
  --home-reflection-center: 86% 46%;
  --home-reflection-size: 52% 46%;
  --home-floor-center: 34% 112%;
  --home-floor-size: 66% 52%;
  --home-light-entry: rgba(255,255,255,.95);
  --home-light-reflection: rgba(176,198,228,.22);
  --home-light-floor: rgba(178,200,230,.16);
  background: var(--home-environment);
}
:root[data-theme='dark'] .home-ambient-host.is-home {
  --home-environment: #09051b;
  --home-entry-center: 14% 90%;
  --home-entry-size: 70% 60%;
  --home-reflection-center: 96% 100%;
  --home-reflection-size: 62% 58%;
  --home-floor-center: 52% 78%;
  --home-floor-size: 40% 16%;
  --home-light-entry: rgba(112,66,186,.02);
  --home-light-reflection: rgba(92,62,154,.02);
  --home-light-floor: rgba(133,98,255,.28);
}
:root[data-theme='dark'][data-home-accent='custom'] .home-ambient-host.is-home {
  --home-light-entry: inherit;
  --home-light-reflection: inherit;
  --home-light-floor: inherit;
}
@media (max-width: 700px) {
  .home-ambient-host.is-home { --home-floor-center: 46% 112%; --home-floor-size: 130% 58%; }
  :root[data-theme='light'] .home-ambient-host.is-home {
    --home-entry-center: 50% 10%;
    --home-entry-size: 128% 42%;
    --home-reflection-center: 104% 54%;
    --home-reflection-size: 84% 46%;
  }
  :root[data-theme='dark'] .home-ambient-host.is-home {
    --home-floor-center: 50% 78%;
    --home-floor-size: 72% 18%;
  }
}
.home-ambient-canvas {
  position: absolute;
  inset: 0;
  display: block;
  width: 100%;
  height: 100%;
  opacity: 0;
  pointer-events: none;
}
.home-ambient-canvas.is-visible { opacity: 1; }
</style>
