/**
 * Visual Effects Kit catalog — the single source of truth for every effect's
 * identity, parameters, defaults and copy-paste usage. Definitions are plain
 * typed data: no eval, no HTML injection, no dynamic CSS strings.
 */

export type EffectId = 'ambient-gradient' | 'spotlight-surface' | 'soft-reveal' | 'surface-depth'
export type EffectTech = 'css' | 'vue' | 'webgl2'
export type ParamValue = number | string | boolean
export type ParamRecord = Record<string, ParamValue>

export interface ParamOption {
  value: string
  label: string
}

export interface ParamDefinition {
  key: string
  label: string
  control: 'range' | 'select' | 'toggle'
  min?: number
  max?: number
  step?: number
  unit?: string
  options?: ParamOption[]
  default: ParamValue
}

export interface EffectDefinition {
  id: EffectId
  name: string
  tech: EffectTech
  summary: string
  where: string
  /** Component or directive that implements it. */
  entry: string
  params: ParamDefinition[]
  reducedMotion: string
  fallback: string
  /** Returns a minimal, interface-accurate Vue snippet for the given params. */
  usage: (params: ParamRecord) => string
}

const EASING: Record<string, string> = {
  standard: 'cubic-bezier(.2,.7,.3,1)',
  gentle: 'cubic-bezier(.4,0,.2,1)',
  snappy: 'cubic-bezier(.2,.8,.2,1)',
}

export function easingValue(name: string): string {
  return EASING[name] ?? EASING.standard!
}

export const EFFECTS: readonly EffectDefinition[] = [
  {
    id: 'ambient-gradient',
    name: 'Ambient Gradient',
    tech: 'webgl2',
    summary: '页面级环境光：复用现有 WebGL2 引擎，强度、质量与动画可调。',
    where: '页面背景、首页环境光',
    entry: 'AmbientSurface.vue',
    params: [
      { key: 'strength', label: '光效强度', control: 'range', min: 0.2, max: 1, step: 0.05, default: 0.8 },
      { key: 'quality', label: '渲染质量', control: 'select', options: [{ value: 'static', label: '静态' }, { value: 'balanced', label: '均衡' }, { value: 'high', label: '高' }], default: 'high' },
      { key: 'animate', label: '动画', control: 'toggle', default: true },
    ],
    reducedMotion: '不启动渲染循环，回退 CSS 光场。',
    fallback: 'WebGL2 不可用或恢复超时 → CSS 光场。',
    usage: (p) => `<AmbientSurface\n  :enabled="true"\n  :strength="${p.strength}"\n  quality="${p.quality}"\n/>`,
  },
  {
    id: 'spotlight-surface',
    name: 'Spotlight Surface',
    tech: 'css',
    summary: '指针经过卡片时出现跟随的局部反光，不位移、不缩放。',
    where: '玻璃卡片、项目卡、资料卡',
    entry: 'v-fx-spotlight / SpotlightSurface.vue',
    params: [
      { key: 'intensity', label: '强度', control: 'range', min: 0, max: 1, step: 0.05, default: 0.6 },
      { key: 'radius', label: '光照范围', control: 'range', min: 80, max: 320, step: 10, unit: 'px', default: 180 },
      { key: 'speed', label: '过渡速度', control: 'range', min: 80, max: 300, step: 20, unit: 'ms', default: 160 },
      { key: 'tone', label: '光照颜色', control: 'select', options: [{ value: 'neutral', label: '中性' }, { value: 'accent', label: '主题色' }], default: 'neutral' },
    ],
    reducedMotion: '不跟踪指针，不显示反光。',
    fallback: '触屏或粗指针：保持静态卡片。',
    usage: (p) => `<SpotlightSurface\n  :intensity="${p.intensity}"\n  :radius="${p.radius}"\n  :speed="${p.speed}"\n  tone="${p.tone}"\n>\n  <article class="card">…</article>\n</SpotlightSurface>`,
  },
  {
    id: 'soft-reveal',
    name: 'Soft Reveal',
    tech: 'vue',
    summary: '内容变化时 6px 位移淡入，最多 6 项阶梯；只动 opacity 与 transform。',
    where: '筛选结果、Tab 内容、列表更新',
    entry: 'SoftReveal.vue',
    params: [
      { key: 'duration', label: '时长', control: 'range', min: 120, max: 400, step: 20, unit: 'ms', default: 220 },
      { key: 'delay', label: '逐项延迟', control: 'range', min: 0, max: 80, step: 4, unit: 'ms', default: 24 },
      { key: 'distance', label: '移动距离', control: 'range', min: 2, max: 16, step: 1, unit: 'px', default: 6 },
      { key: 'easing', label: '缓动', control: 'select', options: [{ value: 'standard', label: '标准' }, { value: 'gentle', label: '柔和' }, { value: 'snappy', label: '干脆' }], default: 'standard' },
    ],
    reducedMotion: '不播放动画，内容直接可见。',
    fallback: '动画失败时内容保持可见（无 opacity: 0 的终态）。',
    usage: (p) => `<SoftReveal\n  :trigger="filters.key"\n  :duration="${p.duration}"\n  :delay="${p.delay}"\n  :distance="${p.distance}"\n>\n  <div v-for="item in items" :key="item.id">…</div>\n</SoftReveal>`,
  },
  {
    id: 'surface-depth',
    name: 'Surface Depth',
    tech: 'css',
    summary: '可选的卡片层深：背景透明度、边缘强度与阴影深度，不改文字颜色。',
    where: '卡片、面板、浮层容器',
    entry: 'SurfaceDepth.vue',
    params: [
      { key: 'background', label: '背景透明度', control: 'range', min: 0.2, max: 1, step: 0.05, default: 0.7 },
      { key: 'edge', label: '边缘强度', control: 'range', min: 0, max: 1, step: 0.05, default: 0.5 },
      { key: 'shadow', label: '阴影深度', control: 'range', min: 0, max: 1, step: 0.05, default: 0.6 },
    ],
    reducedMotion: '无动画，不受系统偏好影响。',
    fallback: '不支持 color-mix 时回退到现有 --home-panel 材质。',
    usage: (p) => `<SurfaceDepth\n  :background="${p.background}"\n  :edge="${p.edge}"\n  :shadow="${p.shadow}"\n>\n  <article class="card-content">…</article>\n</SurfaceDepth>`,
  },
]

export function findEffect(id: EffectId): EffectDefinition {
  return EFFECTS.find((effect) => effect.id === id) ?? EFFECTS[0]!
}

export function defaultsOf(effect: EffectDefinition): ParamRecord {
  const values: ParamRecord = {}
  for (const param of effect.params) values[param.key] = param.default
  return values
}

/** Clamp ranges, validate select values, coerce toggles; never runs input. */
export function normalizeParams(effect: EffectDefinition, input: Partial<ParamRecord> = {}): ParamRecord {
  const values: ParamRecord = {}
  for (const param of effect.params) {
    const raw = input[param.key]
    if (param.control === 'range') {
      const value = typeof raw === 'number' && Number.isFinite(raw) ? raw : Number(param.default)
      values[param.key] = Math.min(param.max ?? value, Math.max(param.min ?? value, value))
    } else if (param.control === 'select') {
      const allowed = param.options?.some((option) => option.value === raw)
      values[param.key] = allowed ? raw! : param.default
    } else {
      values[param.key] = typeof raw === 'boolean' ? raw : param.default
    }
  }
  return values
}

export interface EffectEnvironment {
  supported: boolean
  reducedMotion: boolean
}

export interface EffectState {
  shouldRun: boolean
  useWebgl: boolean
  cssFallback: boolean
  reason: 'ok' | 'disabled' | 'reduced-motion' | 'unsupported'
}

/** Shared decision point: reduced motion wins, then WebGL capability. */
export function resolveEffectState(effect: EffectDefinition, params: ParamRecord, environment: EffectEnvironment): EffectState {
  if (effect.id === 'ambient-gradient' && params.animate === false) {
    return { shouldRun: false, useWebgl: false, cssFallback: true, reason: 'disabled' }
  }
  if (environment.reducedMotion) {
    return { shouldRun: false, useWebgl: false, cssFallback: effect.id === 'ambient-gradient', reason: 'reduced-motion' }
  }
  if (effect.id === 'ambient-gradient' && !environment.supported) {
    return { shouldRun: false, useWebgl: false, cssFallback: true, reason: 'unsupported' }
  }
  return { shouldRun: true, useWebgl: effect.id === 'ambient-gradient', cssFallback: false, reason: 'ok' }
}
