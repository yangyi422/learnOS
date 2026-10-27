export type MotionId = 'button-feedback' | 'card-hover' | 'tab-transition' | 'content-reveal' | 'panel-presence' | 'timing-comparison'
export type MotionEase = 'standard' | 'enter' | 'exit'
export type MotionTech = 'CSS' | 'Vue'

export interface MotionParams {
  duration: number
  easing: MotionEase
  distance: number
}

export interface MotionDefinition {
  id: MotionId
  name: string
  tech: MotionTech
  summary: string
  where: string
  primitive: string
  /** Which parameter groups actually apply to this preview. */
  params: Array<'duration' | 'easing' | 'distance'>
  usage: (params: MotionParams) => { css: string; vue: string }
}

export const DEFAULT_MOTION: MotionParams = { duration: 160, easing: 'standard', distance: 4 }

export function easeVar(easing: MotionEase): string {
  return `var(--motion-ease-${easing})`
}

export function clampDistance(value: number): number {
  return Math.min(12, Math.max(2, Math.round(value)))
}

export const MOTIONS: readonly MotionDefinition[] = [
  {
    id: 'button-feedback',
    name: 'Button Feedback',
    tech: 'CSS',
    summary: '悬停与按下只改变颜色、边缘与 1px 位移，不做缩放。',
    where: '按钮、图标按钮、操作行',
    primitive: 'CSS tokens',
    params: ['duration', 'easing'],
    usage: (p) => ({
      css: `transition: border-color ${p.duration}ms ${easeVar(p.easing)},
            background-color ${p.duration}ms ${easeVar(p.easing)};`,
      vue: `<button class="button">保存</button>\n\n/* 只改颜色与 1px 位移，不缩放 */\n.button:active { transform: translateY(1px); }`,
    }),
  },
  {
    id: 'card-hover',
    name: 'Card Hover',
    tech: 'CSS',
    summary: '卡片悬停时边缘反光与阴影提升，不位移、不放大。',
    where: '玻璃卡片、项目卡、资料卡',
    primitive: 'CSS tokens',
    params: ['duration', 'easing'],
    usage: (p) => ({
      css: `transition: border-color ${p.duration}ms ${easeVar(p.easing)},
            box-shadow ${p.duration}ms ${easeVar(p.easing)};`,
      vue: `<article class="card">…</article>\n\n/* 不改变布局，只改边缘与阴影 */`,
    }),
  },
  {
    id: 'tab-transition',
    name: 'Tab Transition',
    tech: 'Vue',
    summary: 'Tab 内容切换使用 4px 淡入，不移动外层容器。',
    where: '探索、生活、设置内的局部切换',
    primitive: 'MotionFade / MotionReveal',
    params: ['duration', 'easing', 'distance'],
    usage: (p) => ({
      css: `/* 由 MotionReveal 提供，无需手写关键帧 */`,
      vue: `<MotionReveal :trigger="tab" :duration="${p.duration}" :distance="${p.distance}">\n  <div v-if="tab === 'today'">…</div>\n</MotionReveal>`,
    }),
  },
  {
    id: 'content-reveal',
    name: 'Content Reveal',
    tech: 'Vue',
    summary: '内容变化时 4–8px 淡入，最多 6 项阶梯；失败也不会隐藏内容。',
    where: '筛选结果、列表更新、局部加载完成',
    primitive: 'MotionReveal',
    params: ['duration', 'easing', 'distance'],
    usage: (p) => ({
      css: `/* 由 MotionReveal 提供 */`,
      vue: `<MotionReveal :trigger="filterKey" :duration="${p.duration}" :distance="${p.distance}">\n  <article v-for="item in items" :key="item.id">…</article>\n</MotionReveal>`,
    }),
  },
  {
    id: 'panel-presence',
    name: 'Panel Presence',
    tech: 'Vue',
    summary: '面板显隐使用透明度过渡，DOM 在离开动画结束后移除。',
    where: '侧栏说明、折叠详情、内联面板',
    primitive: 'MotionFade',
    params: ['duration', 'easing'],
    usage: (p) => ({
      css: `/* 由 MotionFade 提供 */`,
      vue: `<MotionFade :show="open" :duration="${p.duration}">\n  <section class="panel">…</section>\n</MotionFade>`,
    }),
  },
  {
    id: 'timing-comparison',
    name: 'Motion Timing',
    tech: 'CSS',
    summary: '同一动作在 instant / fast / slow 三档下的节奏差异。',
    where: '选择 token 时的对照参考',
    primitive: 'Tokens',
    params: ['easing'],
    usage: (p) => ({
      css: `--motion-instant: 100ms;\n--motion-fast: 160ms;\n--motion-base: 180ms;\n--motion-slow: 300ms;\n\n/* 当前对照使用 ${easeVar(p.easing)} */`,
      vue: `/* 只改变 token，不改变组件结构 */`,
    }),
  },
]

export function findMotion(id: MotionId): MotionDefinition {
  return MOTIONS.find((motion) => motion.id === id) ?? MOTIONS[0]!
}
