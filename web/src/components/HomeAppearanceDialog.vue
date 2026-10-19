<template>
  <el-drawer append-to-body v-model="open" title="外观设置" size="min(400px, calc(100vw - 28px))" direction="rtl" :modal="false" :lock-scroll="false" class="home-appearance-dialog">
    <p class="appearance-hint">全站即时生效，只保存在当前浏览器。</p>
    <fieldset class="appearance-section">
      <legend>明暗主题</legend>
      <div class="appearance-modes">
        <button v-for="mode in modes" :key="mode.value" type="button" :aria-pressed="preference === mode.value" @click="setThemePreference(mode.value)">{{ mode.label }}</button>
      </div>
    </fieldset>
    <fieldset class="appearance-section">
      <legend>界面色系</legend>
      <div class="appearance-palettes">
        <button v-for="preset in homeColorPresets" :key="preset.name" type="button" :aria-label="preset.name + '色系'" :aria-pressed="accent === preset.color" @click="setHomeAccent(preset.color)">
          <span class="appearance-swatch" :style="{ '--swatch': preset.color || '#4653df' }" aria-hidden="true" />
          <span>{{ preset.name }}</span>
        </button>
      </div>
    </fieldset>
    <label class="appearance-custom">
      <span><strong>自定义颜色</strong><small>各页面的背景光与主题色同步变化</small></span>
      <code>{{ accent || '默认' }}</code>
      <input type="color" aria-label="自定义界面颜色" :value="accent || '#4653df'" @input="chooseColor">
    </label>
    <p v-if="!canPersist" class="appearance-warning" role="status">浏览器未允许保存偏好，本次调整仍然有效，刷新后可能恢复。</p>
    <template #footer>
      <el-button @click="setHomeAccent('')">恢复默认色系</el-button>
      <el-button type="primary" @click="open = false">完成</el-button>
    </template>
  </el-drawer>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { homeColorPresets, setHomeAccent, useHomeAppearance } from '@/home-appearance'
import { setThemePreference, useTheme, type ThemePreference } from '@/theme'
const props = defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()
const open = computed({ get: () => props.modelValue, set: value => emit('update:modelValue', value) })
const { accent, canPersist } = useHomeAppearance()
const { preference } = useTheme()
const modes: { value: ThemePreference; label: string }[] = [{ value: 'light', label: '浅色' }, { value: 'dark', label: '深色' }, { value: 'system', label: '跟随系统' }]
function chooseColor(event: Event) { setHomeAccent((event.target as HTMLInputElement).value) }
</script>

<style scoped>
.appearance-hint { margin: 0 0 26px; color: var(--text-secondary); font-size: 14px; }
.appearance-section { min-width: 0; padding: 0; margin: 0 0 24px; border: 0; }
.appearance-section legend { padding: 0; margin-bottom: 12px; font-size: 14px; font-weight: 600; color: var(--text-primary); }
.appearance-modes { display: flex; gap: 8px; }
.appearance-modes button, .appearance-palettes button { min-height: 44px; border: 1px solid var(--border-default); border-radius: 10px; background: var(--bg-surface); color: var(--text-primary); font: inherit; font-size: 13px; cursor: pointer; }
.appearance-modes button { flex: 1; padding: 8px; }
.appearance-palettes { display: grid; grid-template-columns: repeat(5,minmax(0,1fr)); gap: 8px; }
.appearance-palettes button { display: grid; justify-items: center; gap: 8px; padding: 12px 2px; }
.appearance-swatch { width: 26px; height: 26px; border-radius: 50%; background: var(--swatch); box-shadow: inset 0 0 0 1px rgba(0,0,0,.12); }
button[aria-pressed='true'] { border-color: var(--el-color-primary); background: var(--el-color-primary-light-9); }
button:focus-visible, input:focus-visible { outline: 2px solid var(--el-color-primary); outline-offset: 3px; }
.appearance-custom { display: flex; align-items: center; gap: 12px; padding-block: 16px; border-top: 1px solid var(--border-default); color: var(--text-primary); }
.appearance-custom > span { flex: 1; display: grid; gap: 5px; }
.appearance-custom strong { font-size: 14px; font-weight: 500; }
.appearance-custom small { color: var(--text-secondary); font-size: 12px; }
.appearance-custom code { font-size: 12px; color: var(--text-secondary); }
.appearance-custom input { width: 48px; height: 44px; border: 1px solid var(--border-default); padding: 3px; border-radius: 8px; background: transparent; cursor: pointer; }
.appearance-warning { color: var(--text-secondary); font-size: 13px; line-height: 1.7; }
@media (max-width: 380px) { .appearance-palettes { grid-template-columns: repeat(3,minmax(0,1fr)); } .appearance-custom code { display: none; } }
</style>
