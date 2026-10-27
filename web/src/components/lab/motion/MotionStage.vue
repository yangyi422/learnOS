<template>
  <div
    class="ms-stage"
    :data-motion-reduced="reduced ? '' : undefined"
    :style="vars"
  >
    <template v-if="motion.id === 'button-feedback'">
      <div class="ms-row">
        <button type="button" class="ms-button ms-button--primary">保存更改</button>
        <button type="button" class="ms-button">取消</button>
        <button type="button" class="ms-button ms-button--ghost">仅文字</button>
      </div>
      <p class="ms-hint">悬停看边缘与底色；按下只有 1px 位移。</p>
    </template>

    <template v-else-if="motion.id === 'card-hover'">
      <article class="ms-card">
        <p class="ms-eyebrow">今日焦点 · 工作</p>
        <h3>完成首页信息架构评审并给出修改清单</h3>
        <p>悬停时只提升边缘与阴影，不移动、不缩放卡片。</p>
      </article>
    </template>

    <template v-else-if="motion.id === 'tab-transition'">
      <el-tabs v-model="tab" class="ms-tabs">
        <el-tab-pane label="今日" name="today">
          <MotionReveal :trigger="tab" :duration="params.duration" :distance="params.distance" :ease="ease">
            <div class="ms-list">
              <div class="ms-list__row"><span>完成首页信息架构评审</span><span class="ms-muted">工作</span></div>
              <div class="ms-list__row"><span>整理本周项目风险</span><span class="ms-muted">工作</span></div>
            </div>
          </MotionReveal>
        </el-tab-pane>
        <el-tab-pane label="本周" name="week">
          <MotionReveal :trigger="tab" :duration="params.duration" :distance="params.distance" :ease="ease">
            <div class="ms-list">
              <div class="ms-list__row"><span>评审浅色主题对比度</span><span class="ms-muted">设计</span></div>
              <div class="ms-list__row"><span>记录一次实验结论</span><span class="ms-muted">学习</span></div>
            </div>
          </MotionReveal>
        </el-tab-pane>
      </el-tabs>
    </template>

    <template v-else-if="motion.id === 'content-reveal'">
      <MotionReveal
        :trigger="replay"
        :duration="params.duration"
        :distance="params.distance"
        :ease="ease"
        :play-on-mount="true"
      >
        <div class="ms-list">
          <div class="ms-list__row"><span>交付一个阶段成果</span><span class="ms-muted">工作</span></div>
          <div class="ms-list__row"><span>重新布置书桌</span><span class="ms-muted">居住</span></div>
          <div class="ms-list__row"><span>记录一段没有分类的经历</span><span class="ms-muted">其他</span></div>
        </div>
      </MotionReveal>
    </template>

    <template v-else-if="motion.id === 'panel-presence'">
      <button type="button" class="ms-button" data-testid="mg-panel-toggle" @click="panelOpen = !panelOpen">
        {{ panelOpen ? '关闭面板' : '打开面板' }}
      </button>
      <MotionFade :show="panelOpen" :duration="params.duration">
        <section class="ms-panel" data-testid="mg-panel">
          <p class="ms-eyebrow">面板</p>
          <h3>这是一次显隐过渡</h3>
          <p>离开动画结束后元素才从 DOM 移除；关闭动画不会让内容永久隐藏。</p>
        </section>
      </MotionFade>
    </template>

    <template v-else>
      <div :key="replay" class="ms-timing">
        <div v-for="item in timing" :key="item.label" class="ms-timing__row">
          <span class="ms-timing__label">{{ item.label }}</span>
          <span class="ms-timing__track"><span class="ms-timing__bar" :style="{ animationDuration: item.duration }" /></span>
          <span class="ms-muted">{{ item.duration }}</span>
        </div>
      </div>
      <p class="ms-hint">三档 token 同时播放，便于比较节奏。</p>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import MotionFade from '@/motion/MotionFade.vue'
import MotionReveal from '@/motion/MotionReveal.vue'
import { easeVar, type MotionDefinition, type MotionParams } from '@/motion/catalog'

const props = defineProps<{
  motion: MotionDefinition
  params: MotionParams
  reduced: boolean
  replay: number
}>()

const tab = ref('today')
const panelOpen = ref(false)

const ease = computed(() => easeVar(props.params.easing))
const vars = computed(() => ({
  '--motion-duration': `${props.params.duration}ms`,
  '--motion-distance': `${props.params.distance}px`,
  '--motion-ease': ease.value,
}))

const timing = [
  { label: 'Instant', duration: '100ms' },
  { label: 'Fast', duration: '160ms' },
  { label: 'Slow', duration: '300ms' },
]
</script>

<style scoped>
.ms-stage {
  display: grid;
  gap: 14px;
  min-height: 300px;
  padding: 24px;
  align-content: start;
  border: 1px solid var(--home-edge);
  border-radius: var(--radius-card);
  background: var(--home-canvas);
}
.ms-row { display: flex; flex-wrap: wrap; gap: 10px; align-items: center; }
.ms-stage > .ms-button { justify-self: start; }
.ms-button {
  min-height: 36px;
  padding: 0 16px;
  border: 1px solid var(--home-edge);
  border-radius: var(--radius-pill);
  background: var(--home-panel);
  color: var(--home-ink);
  font: inherit;
  font-size: var(--font-meta);
  cursor: pointer;
  transition:
    border-color var(--motion-duration) var(--motion-ease),
    background-color var(--motion-duration) var(--motion-ease),
    color var(--motion-duration) var(--motion-ease);
}
.ms-button:hover { border-color: var(--home-reflection); background: var(--home-panel-hover); }
.ms-button:active { transform: translateY(1px); }
.ms-button:focus-visible { outline: 2px solid var(--home-link); outline-offset: 2px; }
.ms-button--primary { background: var(--home-primary); border-color: var(--home-primary); color: #fff; }
.ms-button--ghost { border-color: transparent; background: transparent; color: var(--home-link); }
.ms-card {
  display: grid;
  gap: 8px;
  padding: 20px;
  border: 1px solid var(--home-edge);
  border-radius: var(--radius-card);
  background: var(--home-panel);
  transition:
    border-color var(--motion-duration) var(--motion-ease),
    box-shadow var(--motion-duration) var(--motion-ease);
}
.ms-card:hover { border-color: var(--home-reflection); box-shadow: 0 16px 34px -22px rgba(0, 0, 0, .4); }
.ms-card h3 { margin: 0; font-size: var(--font-card-title); }
.ms-card p { margin: 0; color: var(--home-secondary); font-size: var(--font-meta); line-height: 1.7; }
.ms-eyebrow { margin: 0; color: var(--home-muted); font-size: var(--font-caption); }
.ms-tabs { max-width: 520px; }
.ms-list { display: grid; gap: 0; }
.ms-list__row {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 2px;
  border-bottom: 1px solid var(--home-edge);
  color: var(--home-ink);
  font-size: var(--font-meta);
}
.ms-list__row:last-child { border-bottom: 0; }
.ms-muted { color: var(--home-muted); font-size: var(--font-caption); }
.ms-panel {
  display: grid;
  gap: 6px;
  margin-top: 12px;
  padding: 18px;
  border: 1px solid var(--home-edge);
  border-radius: var(--radius-card);
  background: var(--home-panel);
}
.ms-panel h3 { margin: 0; font-size: var(--font-card-title); }
.ms-panel p { margin: 0; color: var(--home-secondary); font-size: var(--font-meta); line-height: 1.7; }
.ms-hint { margin: 0; color: var(--home-muted); font-size: var(--font-caption); }
.ms-timing { display: grid; gap: 12px; }
.ms-timing__row { display: grid; grid-template-columns: 60px minmax(0, 1fr) 48px; gap: 12px; align-items: center; }
.ms-timing__label { color: var(--home-secondary); font-size: var(--font-caption); }
.ms-timing__track { height: 6px; border-radius: 999px; background: var(--home-edge); overflow: hidden; }
.ms-timing__bar {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--home-link);
  transform-origin: left center;
  animation: ms-grow 300ms var(--motion-ease) both;
}
@keyframes ms-grow { from { transform: scaleX(0); } to { transform: scaleX(1); } }
[data-motion-reduced] .ms-button,
[data-motion-reduced] .ms-card { transition: none; }
[data-motion-reduced] .ms-button:active { transform: none; }
[data-motion-reduced] .ms-timing__bar { animation: none; transform: none; }
</style>
