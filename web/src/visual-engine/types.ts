/** Rendering configuration, quality, lifecycle and metric types. */

export type SceneId = 'ambient' | 'material'

export type QualityTier = 'high' | 'balanced' | 'static'

/** `lost` keeps the canvas so a restore can reuse it; `unavailable` means give up. */
export type ContextState = 'ok' | 'lost' | 'unavailable'

export type RGB = [number, number, number]

export interface LightSource {
  color: RGB
  alpha: number
}

/** A snapshot of the ambient tokens, read from the live CSS custom properties. */
export interface SurfaceLight {
  theme: 'light' | 'dark'
  accent: boolean
  /** One colour means flat ground; two or three mean a linear gradient. */
  ground: RGB[]
  gradient: boolean
  entry: LightSource
  reflection: LightSource
  floor: LightSource
  /** Optional route-local CSS field geometry: center x/y, radius x/y. */
  fields?: Record<'entry' | 'reflection' | 'floor', [number, number, number, number]>
  vertical?: boolean
}

export interface FrameContext {
  width: number
  height: number
  /** Seconds of visible time; paused while the tab is hidden. */
  time: number
  strength: number
  light: SurfaceLight
}

export type UniformMap = Record<string, WebGLUniformLocation | null>

/** A scene owns shader sources and uniform writes; never context or lifecycle. */
export interface ShaderScene {
  id: SceneId
  label: string
  vertexShader: string
  fragmentShader: string
  uniformNames: readonly string[]
  write(gl: WebGL2RenderingContext, uniforms: UniformMap, frame: FrameContext): void
}

export interface RenderMetrics {
  fps: number | null
  drawCalls: number
  /** Backing-store size of the canvas in device pixels. */
  width: number
  height: number
  /** CSS box aspect ratio (width / height). */
  aspect: number
  pixelRatio: number
  gpuAvailable: boolean
  gpuMs: number | null
  context: ContextState
}

/** Viewport-space rectangle (CSS pixels) the glass pass should refract. */
export interface GlassRect {
  left: number
  top: number
  width: number
  height: number
}

export interface GlassOptions {
  /** Read every frame; false falls back to the plain ambient draw. */
  enabled: () => boolean
  getRect: () => GlassRect | null
  getTheme?: () => 'light' | 'dark'
  radius?: number
  /** Peak UV displacement at the edge, in CSS pixels. */
  strength?: number
}

export interface RendererOptions {
  scene: SceneId
  light: SurfaceLight
  strength?: number
  quality?: QualityTier
  /** Context-restore deadline; production default is 8s, tests may inject less. */
  restoreTimeoutMs?: number
  onMetrics?: (metrics: RenderMetrics) => void
  onContext?: (state: ContextState) => void
  /** Optional admin-only local refraction on top of the ambient scene. */
  glass?: GlassOptions
}

export interface Renderer {
  setScene(scene: SceneId): void
  setLight(light: SurfaceLight): void
  setStrength(strength: number): void
  setQuality(quality: QualityTier): void
  start(): void
  stop(): void
  renderOnce(): void
  dispose(): void
}
