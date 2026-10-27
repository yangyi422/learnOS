<template>
  <div ref="host" class="fx-ambient" :data-state="state">
    <canvas v-if="canvasReady" :key="canvasKey" ref="canvas" class="fx-ambient__canvas" aria-hidden="true" />
    <HomeLightField v-show="!webglActive" />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import HomeLightField from '@/components/HomeLightField.vue'
import {
  createRenderer,
  detectWebGL2,
  prefersReducedMotion,
  readSurfaceLight,
  subscribeThemeChanges,
  type QualityTier,
  type RenderMetrics,
  type Renderer,
} from '@/visual-engine'

/**
 * Page-level WebGL ambient host. One renderer per host, never per card.
 * Reuses the existing engine; pauses offscreen; never throws at the caller.
 */
const props = withDefaults(defineProps<{
  enabled?: boolean
  scene?: 'ambient' | 'material'
  strength?: number
  quality?: QualityTier
  respectReducedMotion?: boolean
}>(), {
  enabled: false,
  scene: 'ambient',
  strength: 0.8,
  quality: 'high',
  respectReducedMotion: true,
})

const emit = defineEmits<{ metrics: [metrics: RenderMetrics]; state: [state: 'css' | 'webgl' | 'degraded'] }>()

const host = ref<HTMLElement | null>(null)
const canvas = ref<HTMLCanvasElement | null>(null)
const canvasReady = ref(false)
const canvasKey = ref(0)
const webglActive = ref(false)
const degraded = ref(false)
const state = computed(() => (webglActive.value ? 'webgl' : degraded.value ? 'degraded' : 'css'))

const supported = ref(detectWebGL2().supported)
const reduced = ref(prefersReducedMotion())
let renderer: Renderer | null = null
let stopThemeWatch: (() => void) | null = null
let observer: IntersectionObserver | null = null
let visible = true
let disposed = false

function canRun(): boolean {
  return props.enabled && supported.value && !(props.respectReducedMotion && reduced.value)
}

function teardown(): void {
  renderer?.dispose()
  renderer = null
  webglActive.value = false
  canvasReady.value = false
}

function degrade(): void {
  if (disposed) return
  teardown()
  degraded.value = true
  emit('state', 'degraded')
}

async function sync(): Promise<void> {
  if (disposed) return
  if (!canRun()) {
    teardown()
    degraded.value = false
    emit('state', 'css')
    return
  }
  if (webglActive.value) return

  canvasKey.value += 1
  canvasReady.value = true
  await nextTick()
  const element = canvas.value
  if (!element) return
  try {
    renderer = createRenderer(element, {
      scene: props.scene,
      light: readSurfaceLight(),
      strength: props.strength,
      quality: props.quality,
      onMetrics: (metrics) => emit('metrics', metrics),
      onContext: (next) => { if (next === 'unavailable') degrade() },
    })
    if (visible) renderer.start()
    webglActive.value = true
    degraded.value = false
    emit('state', 'webgl')
  } catch {
    degrade()
  }
}

watch(() => props.enabled, () => { void sync() })
watch(reduced, () => { void sync() })
watch(() => props.strength, (value) => renderer?.setStrength(value))
watch(() => props.quality, (value) => renderer?.setQuality(value))

onMounted(() => {
  stopThemeWatch = subscribeThemeChanges(() => renderer?.setLight(readSurfaceLight()))
  observer = new IntersectionObserver((entries) => {
    const entry = entries[0]
    if (!entry) return
    visible = entry.isIntersecting
    if (!renderer) return
    if (visible) renderer.start()
    else renderer.stop()
  }, { threshold: 0.05 })
  if (host.value) observer.observe(host.value)
  void sync()
})

onBeforeUnmount(() => {
  disposed = true
  observer?.disconnect()
  observer = null
  stopThemeWatch?.()
  stopThemeWatch = null
  teardown()
})
</script>

<style scoped>
.fx-ambient { position: absolute; inset: 0; overflow: hidden; pointer-events: none; }
.fx-ambient__canvas { position: absolute; inset: 0; width: 100%; height: 100%; opacity: .9; }
</style>
