<template>
  <div class="visual-lab-stage">
    <canvas ref="canvas" v-show="hasContext" class="visual-lab-canvas" aria-hidden="true" tabindex="-1" />
    <div v-show="!hasContext" class="visual-lab-static" aria-hidden="true">
      <HomeLightField />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import HomeLightField from '@/components/HomeLightField.vue'
import {
  createRenderer,
  liveProgramCount,
  renderedFrameCount,
  rendererInstanceCount,
  type ContextState,
  type QualityTier,
  type Renderer,
  type RenderMetrics,
  type SceneId,
  type SurfaceLight,
} from '@/visual-engine'

const props = defineProps<{
  scene: SceneId
  light: SurfaceLight
  strength: number
  quality: QualityTier
  running: boolean
  reduced: boolean
  supported: boolean
}>()

const emit = defineEmits<{
  metrics: [metrics: RenderMetrics]
  context: [state: ContextState]
  failure: [message: string]
}>()

const canvas = ref<HTMLCanvasElement | null>(null)
const failed = ref(false)
let renderer: Renderer | null = null

/** Without a usable context the component falls back to the CSS light field. */
const hasContext = computed(() => props.supported && !failed.value)
const loops = computed(() => hasContext.value && props.running && props.quality !== 'static')

function exposeDiagnostics(): void {
  if (!import.meta.env.DEV) return
  const target = window as Window & {
    __visualLab?: {
      frames: () => number
      instances: () => number
      programs: () => number
      supported: boolean
    }
  }
  target.__visualLab = {
    frames: renderedFrameCount,
    instances: rendererInstanceCount,
    programs: liveProgramCount,
    supported: props.supported,
  }
}

function mount(): void {
  const element = canvas.value
  // Without a usable canvas there is nothing to probe: keep the CSS fallback and
  // do not turn a known limitation into an "initialisation failed" state.
  if (!props.supported || !element || renderer) return
  try {
    renderer = createRenderer(element, {
      scene: props.scene,
      light: props.light,
      strength: props.strength,
      quality: props.quality,
      onMetrics: (metrics) => emit('metrics', metrics),
      onContext: (state) => {
        emit('context', state)
        if (state === 'unavailable') failed.value = true
      },
    })
    if (loops.value) renderer.start()
    else renderer.renderOnce()
  } catch (error) {
    failed.value = true
    emit('failure', error instanceof Error ? error.message : '着色器初始化失败')
  }
  exposeDiagnostics()
}

function unmount(): void {
  renderer?.dispose()
  renderer = null
}

function syncLoop(): void {
  if (!renderer) return
  // stop() keeps the last frame on screen; it does not clear or hide the canvas.
  if (loops.value) renderer.start()
  else renderer.stop()
}

onMounted(mount)
onBeforeUnmount(unmount)
watch(hasContext, (next) => {
  if (next) void nextTick(mount)
  else unmount()
})
watch([() => props.running, () => props.reduced], syncLoop)
watch(() => props.quality, (next) => {
  renderer?.setQuality(next)
  syncLoop()
})
watch(() => props.scene, (next) => {
  renderer?.setScene(next)
  syncLoop()
})
watch(() => props.light, (next) => renderer?.setLight(next))
watch(() => props.strength, (next) => renderer?.setStrength(next))
</script>

<style scoped>
.visual-lab-stage {
  position: relative;
  height: clamp(300px, 42vh, 460px);
  overflow: hidden;
  border: 1px solid var(--home-edge);
  border-radius: 12px;
  background: var(--home-environment);
}
.visual-lab-canvas { display: block; width: 100%; height: 100%; }
.visual-lab-static { position: absolute; inset: 0; }
</style>
