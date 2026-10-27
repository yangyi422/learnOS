<template>
  <div v-if="authorized" class="fx-pilot" data-testid="fx-pilot-bar">
    <button
      type="button"
      class="fx-pilot__toggle"
      data-testid="fx-pilot-toggle"
      :aria-expanded="open"
      aria-label="实验效果"
      @click="open = !open"
    >实验</button>

    <section v-if="open" class="fx-pilot__panel" aria-label="实验效果设置">
      <header class="fx-pilot__head">
        <strong>{{ title }}</strong>
        <span>仅当前会话</span>
      </header>

      <template v-if="page === 'projects'">
        <div class="fx-pilot__row">
          <span>看板卡片反馈</span>
          <el-switch :model-value="settings.projects.cards" data-testid="fx-pilot-projects-cards" aria-label="看板卡片反馈" @update:model-value="(value: string | number | boolean) => setPilot('projects', { cards: Boolean(value) })" />
        </div>
        <div class="fx-pilot__row">
          <span>视图切换过渡</span>
          <el-switch :model-value="settings.projects.tabs" data-testid="fx-pilot-projects-tabs" aria-label="视图切换过渡" @update:model-value="(value: string | number | boolean) => setPilot('projects', { tabs: Boolean(value) })" />
        </div>
        <div class="fx-pilot__row">
          <span>任务抽屉节奏</span>
          <el-switch :model-value="settings.projects.drawer" data-testid="fx-pilot-projects-drawer" aria-label="任务抽屉节奏" @update:model-value="(value: string | number | boolean) => setPilot('projects', { drawer: Boolean(value) })" />
        </div>
      </template>

      <template v-else>
      <div class="fx-pilot__row">
        <span>{{ page === 'home' ? '焦点卡反光' : '卡片反光' }}</span>
        <el-switch
          :model-value="enabled"
          :data-testid="`fx-pilot-${page}`"
          :aria-label="page === 'home' ? '焦点卡反光' : '卡片反光'"
          @update:model-value="(value: string | number | boolean) => toggle(Boolean(value))"
        />
      </div>

      <div class="fx-pilot__row fx-pilot__row--slider">
        <span>强度</span>
        <el-slider
          :model-value="intensity"
          :min="0.1"
          :max="1"
          :step="0.05"
          :disabled="!enabled"
          :data-testid="`fx-pilot-${page}-intensity`"
          aria-label="实验强度"
          @update:model-value="(value: number | number[]) => setIntensity(value)"
        />
      </div>

      <div v-if="page === 'home'" class="fx-pilot__row">
        <span>玻璃折射</span>
        <el-switch
          :model-value="settings.home.refraction"
          data-testid="fx-pilot-refraction"
          aria-label="玻璃折射"
          @update:model-value="(value: string | number | boolean) => setPilot('home', { refraction: Boolean(value) })"
        />
      </div>

      <div v-if="page === 'explore'" class="fx-pilot__row">
        <span>Tab 内容淡入</span>
        <el-switch
          :model-value="settings.explore.reveal"
          data-testid="fx-pilot-reveal"
          aria-label="Tab 内容淡入"
          @update:model-value="(value: string | number | boolean) => setReveal(Boolean(value))"
        />
      </div>

      </template>

      <div class="fx-pilot__actions">
        <el-button size="small" data-testid="fx-pilot-reset" @click="reset">恢复默认</el-button>
      </div>
      <p class="fx-pilot__note">仅管理员可见；不影响普通用户，也不修改业务数据。</p>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import '@/effects/effects.css'
import {
  ensurePilotAuthorization,
  resetPilot,
  setPilot,
  usePilot,
  type PilotPage,
} from '@/effects/pilot'

const props = defineProps<{ page: PilotPage }>()

const { authorized, settings } = usePilot()
const open = ref(false)

const title = computed(() => (props.page === 'home' ? '首页焦点卡' : props.page === 'projects' ? '项目看板' : '探索入口'))
const enabled = computed(() => (props.page === 'home' ? settings.home.enabled : props.page === 'explore' ? settings.explore.enabled : settings.projects.cards))
const intensity = computed(() => (props.page === 'home' ? settings.home.intensity : settings.explore.intensity))


function toggle(value: boolean): void {
  setPilot(props.page, { enabled: value })
}

function setIntensity(value: number | number[]): void {
  const next = Array.isArray(value) ? value[0] : value
  if (typeof next === 'number') setPilot(props.page, { intensity: next })
}

function setReveal(value: boolean): void {
  setPilot('explore', { reveal: value })
}

function reset(): void {
  resetPilot()
}

onMounted(() => { void ensurePilotAuthorization() })
</script>

<style scoped>
.fx-pilot {
  position: fixed;
  right: 20px;
  bottom: 20px;
  z-index: 40;
  display: grid;
  justify-items: end;
  gap: 8px;
}
.fx-pilot__toggle {
  min-height: 34px;
  padding: 0 14px;
  border: 1px solid var(--home-edge);
  border-radius: var(--radius-pill);
  background: color-mix(in srgb, var(--home-canvas) 78%, transparent);
  backdrop-filter: var(--home-glass-filter);
  color: var(--home-secondary);
  font: inherit;
  font-size: 12px;
  cursor: pointer;
}
.fx-pilot__toggle:hover { color: var(--home-ink); }
.fx-pilot__toggle:focus-visible { outline: 2px solid var(--home-link); outline-offset: 2px; }
.fx-pilot__toggle[aria-expanded='true'] { border-color: var(--home-link); color: var(--home-ink); }
.fx-pilot__panel {
  width: 268px;
  display: grid;
  gap: 10px;
  padding: 14px 16px;
  border: 1px solid var(--home-edge);
  border-radius: var(--radius-card);
  background: color-mix(in srgb, var(--home-canvas) 88%, transparent);
  backdrop-filter: var(--home-glass-filter);
  box-shadow: 0 18px 40px -24px rgba(0, 0, 0, .45);
}
.fx-pilot__head { display: flex; align-items: baseline; justify-content: space-between; gap: 8px; }
.fx-pilot__head strong { font-size: var(--font-meta); font-weight: 600; color: var(--home-ink); }
.fx-pilot__head span { color: var(--home-muted); font-size: var(--font-caption); }
.fx-pilot__row { display: flex; align-items: center; justify-content: space-between; gap: 10px; color: var(--home-secondary); font-size: var(--font-caption); }
.fx-pilot__row--slider { align-items: center; }
.fx-pilot__row--slider :deep(.el-slider) { flex: 1; min-width: 80px; }
.fx-pilot__actions { display: flex; justify-content: flex-end; }
.fx-pilot__note { margin: 0; color: var(--home-muted); font-size: 11px; line-height: 1.5; }
@media (max-width: 640px) {
  .fx-pilot { right: 12px; bottom: 12px; }
  .fx-pilot__panel { width: min(268px, calc(100vw - 24px)); }
}
</style>
