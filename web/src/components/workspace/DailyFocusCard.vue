<template>
  <section
    class="focus-card daily-focus-card"
    :class="{ 'is-switching': switching }"
    :inert="switching"
    aria-labelledby="focus-card-title"
  >
    <div class="daily-focus-card__heading">
      <span class="section-kicker"><WorkspaceIcon name="focus" /><span id="focus-card-title">今日焦点</span></span>
      <span class="daily-focus-card__hint">每天每个场景最多一件</span>
    </div>

    <div v-if="loadError" class="daily-focus-error daily-focus-load-error" role="alert">
      <span>{{ loadError }}</span>
      <button type="button" class="focus-action focus-action--reload" @click="$emit('reload')">重试</button>
    </div>

    <template v-else-if="hasAnyFocus">
      <div
        v-for="slot in slots"
        :key="slot.context"
        class="daily-focus-slot"
        :class="{ 'daily-focus-slot--compact': compact }"
        :data-context="slot.context"
        :data-state="slot.focus ? slot.focus.state : 'empty'"
        :aria-busy="busy[slot.context]"
      >
        <template v-if="slot.focus">
          <div class="daily-focus-slot__head">
            <span class="daily-focus-context">{{ contextLabel(slot.context) }}</span>
            <span class="daily-focus-state" :data-state="slot.focus.state" :class="'is-' + stateInfo(slot.focus).tone">{{ stateLabel(slot.focus) }}</span>
          </div>
          <h2 class="daily-focus-heading">
            <RouterLink
              v-if="slot.route"
              :id="compact ? undefined : 'focus-title'"
              :to="slot.route"
              class="daily-focus-title"
              :aria-label="'打开：' + (slot.focus.title || '原任务')"
            >{{ slot.focus.title || '原任务不可用' }}</RouterLink>
            <span v-else :id="compact ? undefined : 'focus-title'" class="daily-focus-title" :class="{ 'is-muted': stateInfo(slot.focus).tone !== 'active' }">{{ slot.focus.title || '原任务不可用' }}</span>
          </h2>
          <p v-if="metaItems(slot.focus).length" class="daily-focus-meta">
            <span v-for="item in metaItems(slot.focus)" :key="item">{{ item }}</span>
          </p>
          <p v-if="stateInfo(slot.focus).tone !== 'active'" class="daily-focus-reason" :class="'is-' + stateInfo(slot.focus).tone">{{ stateReason(slot.focus) }}</p>
          <div class="daily-focus-actions">
            <button v-if="stateInfo(slot.focus).canComplete" type="button" class="focus-action focus-action--complete" :disabled="busy[slot.context]" :aria-label="actionLabel('完成', slot.context)" @click="$emit('complete', slot.focus)">完成</button>
            <button type="button" class="focus-action focus-action--replace" :disabled="busy[slot.context]" :aria-label="actionLabel('更换', slot.context)" @click="$emit('pick', slot.context)">更换</button>
            <button v-if="stateInfo(slot.focus).canClear" type="button" class="focus-action focus-action--clear" :disabled="busy[slot.context]" :aria-label="actionLabel('清空', slot.context)" @click="$emit('clear', slot.context)">清空</button>
            <span v-if="busy[slot.context]" class="daily-focus-saving" role="status">正在保存…</span>
          </div>
        </template>
        <template v-else>
          <div class="daily-focus-slot__head">
            <span class="daily-focus-context">{{ contextLabel(slot.context) }}</span>
            <span class="daily-focus-state is-empty" data-state="empty">未设置</span>
          </div>
          <p class="daily-focus-slot__empty-copy">{{ emptySlotCopy(slot.context) }}</p>
          <div class="daily-focus-actions">
            <button type="button" class="focus-action focus-action--pick" :disabled="busy[slot.context]" :aria-label="actionLabel('设置', slot.context)" @click="$emit('pick', slot.context)">{{ slot.context === 'life' ? '选择生活任务' : '选择工作任务' }}</button>
          </div>
        </template>
        <p v-if="errors[slot.context]" class="daily-focus-error" role="alert">
          <span>{{ errors[slot.context] }}</span>
          <button v-if="conflicts[slot.context]" type="button" class="focus-action focus-action--reload" @click="$emit('reload')">刷新</button>
          <button v-else type="button" class="focus-action focus-action--retry" @click="$emit('retry', slot.context)">重试</button>
        </p>
      </div>
    </template>

    <div v-else class="daily-focus-empty">
      <template v-if="scene === 'all'">
        <h2 id="focus-title">今天想先做哪件事？</h2>
        <p>工作和生活各选一件就好，也可以先只选一件，或者都先空着。</p>
        <div class="daily-focus-empty__actions">
          <el-button class="focus-action focus-action--pick" :aria-label="actionLabel('设置', 'work')" @click="$emit('pick', 'work')">选择工作任务</el-button>
          <el-button class="focus-action focus-action--pick" :aria-label="actionLabel('设置', 'life')" @click="$emit('pick', 'life')">选择生活任务</el-button>
        </div>
      </template>
      <template v-else-if="scene === 'life'">
        <h2 id="focus-title">今天可以留白</h2>
        <p>不设焦点也完全没问题。如果有一件想为自己做的事，可以选一个。</p>
        <div class="daily-focus-empty__actions">
          <el-button class="focus-action focus-action--pick" :aria-label="actionLabel('设置', 'life')" @click="$emit('pick', 'life')">选择生活任务</el-button>
        </div>
      </template>
      <template v-else>
        <h2 id="focus-title">设置今日焦点</h2>
        <p>从今天计划的任务里挑一件最想推进的事，其他安排先放一边。</p>
        <div class="daily-focus-empty__actions">
          <el-button class="focus-action focus-action--pick" :aria-label="actionLabel('设置', 'work')" @click="$emit('pick', 'work')">设置今日焦点</el-button>
        </div>
      </template>
    </div>

  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import WorkspaceIcon from '@/components/WorkspaceIcon.vue'
import type { WorkspaceContext } from '@/workspace-context'
import {
  dailyFocusStateReason,
  dailyFocusTaskRoute,
  describeDailyFocusState,
  taskStatusLabel,
  type DailyFocusContext,
  type DailyFocusView,
  type WorkspaceDailyFocus,
} from '@/api/dailyFocus'

const props = withDefaults(defineProps<{
  scene: WorkspaceContext
  date: string
  focuses: WorkspaceDailyFocus
  busy: Record<DailyFocusContext, boolean>
  errors: Record<DailyFocusContext, string>
  conflicts: Record<DailyFocusContext, boolean>
  loadError?: string
  switching?: boolean
}>(), { switching: false, loadError: '' })

defineEmits<{
  pick: [context: DailyFocusContext]
  clear: [context: DailyFocusContext]
  complete: [focus: DailyFocusView]
  retry: [context: DailyFocusContext]
  reload: []
}>()

const contexts = computed<DailyFocusContext[]>(() => (props.scene === 'all' ? ['work', 'life'] : [props.scene as DailyFocusContext]))
const compact = computed(() => props.scene === 'all')
const slots = computed(() => contexts.value.map(context => {
  const focus = props.focuses[context]
  return { context, focus, route: focus ? dailyFocusTaskRoute(focus) : null }
}))
const hasAnyFocus = computed(() => contexts.value.some(context => props.focuses[context] !== null))

function contextLabel(context: DailyFocusContext) { return context === 'work' ? '工作' : '生活' }
function actionLabel(action: string, context: DailyFocusContext) { return `${action}${contextLabel(context)}焦点` }
function stateInfo(focus: DailyFocusView) { return describeDailyFocusState(focus.state) }
function stateLabel(focus: DailyFocusView) {
  const info = stateInfo(focus)
  return info.tone === 'active' ? taskStatusLabel(focus.task_status) : info.label
}
function stateReason(focus: DailyFocusView) { return dailyFocusStateReason(focus) }

function formatDateLabel(value: string) {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value)
  if (!match) return value
  return `${Number(match[2])}月${Number(match[3])}日`
}
function dueLabel(dueDate: string) {
  if (dueDate < props.date) return `已逾期 · 截止 ${formatDateLabel(dueDate)}`
  if (dueDate === props.date) return '今天截止'
  return `截止 ${formatDateLabel(dueDate)}`
}
function metaItems(focus: DailyFocusView) {
  const items: string[] = []
  if (focus.project_title) items.push(focus.project_title)
  if (focus.state === 'ok' || focus.state === 'completed') items.push(taskStatusLabel(focus.task_status))
  if (focus.state === 'ok' && focus.priority === 'high') items.push('高优先级')
  if (focus.due_date) items.push(dueLabel(focus.due_date))
  if (focus.plan_date && focus.plan_date !== props.date) items.push(`计划 ${formatDateLabel(focus.plan_date)}`)
  return items
}
function emptySlotCopy(context: DailyFocusContext) {
  return context === 'life' ? '生活可以留白，也可以选一件想做的事。' : '工作焦点还没有设置。'
}
</script>

<style scoped>
.daily-focus-card { gap: 20px; }
.daily-focus-card.is-switching > * { visibility: hidden; }
.daily-focus-card__heading { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 8px 12px; }
.daily-focus-card__heading .section-kicker { display: flex; align-items: center; gap: 10px; font-size: 18px; font-weight: 600; }
.daily-focus-card__heading .section-kicker svg { width: 22px; height: 22px; color: var(--home-secondary); }
.daily-focus-card__hint { color: var(--home-muted); font-size: 12px; }
.daily-focus-slot { display: flex; flex-direction: column; gap: 12px; min-width: 0; flex: 1; }
.daily-focus-slot + .daily-focus-slot { padding-top: 18px; border-top: 1px solid var(--home-edge); }
.daily-focus-slot__head { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.daily-focus-context { color: var(--home-secondary); font-size: 13px; font-weight: 600; letter-spacing: .04em; }
.daily-focus-state { border: 1px solid var(--home-edge); border-radius: 999px; padding: 2px 10px; background: var(--home-tint); color: var(--home-secondary); font-size: 12px; line-height: 1.6; }
.daily-focus-state.is-done { color: var(--color-success, var(--home-link)); }
.daily-focus-state.is-invalid { color: var(--color-warning); }
.daily-focus-state.is-empty { color: var(--home-muted); background: transparent; }
.daily-focus-heading { margin: 0; min-width: 0; }
.daily-focus-title { display: inline-block; color: var(--home-ink); font-size: 36px; font-weight: 600; line-height: 1.45; text-decoration: none; overflow-wrap: anywhere; border-radius: 6px; }
a.daily-focus-title { text-underline-offset: 4px; }
a.daily-focus-title:hover { color: var(--home-link); text-decoration: underline; }
a.daily-focus-title:focus-visible { outline: 2px solid var(--home-link); outline-offset: 4px; }
.daily-focus-title.is-muted { color: var(--home-secondary); font-size: 28px; }
.daily-focus-meta { display: flex; flex-wrap: wrap; gap: 6px 10px; margin: 0; color: var(--home-muted); font-size: 13px; line-height: 1.7; }
.daily-focus-meta span { border: 1px solid var(--home-edge); border-radius: 999px; padding: 2px 10px; background: var(--home-tint); }
.daily-focus-reason { margin: 0; font-size: 14px; line-height: 1.75; color: var(--home-secondary); overflow-wrap: anywhere; }
.daily-focus-reason.is-invalid { color: var(--color-warning); }
.daily-focus-reason.is-done { color: var(--home-secondary); }
.daily-focus-actions { display: flex; flex-wrap: wrap; align-items: center; gap: 8px 10px; }
.focus-action { min-height: 36px; padding: 6px 16px; border: 1px solid var(--home-edge); border-radius: 999px; background: var(--home-tint); color: var(--home-ink); font: inherit; font-size: 13px; line-height: 1.5; cursor: pointer; transition: background-color 180ms ease, border-color 180ms ease, color 180ms ease; }
.focus-action:hover:not(:disabled) { border-color: var(--home-reflection); color: var(--home-link); }
.focus-action:focus-visible { outline: 2px solid var(--home-link); outline-offset: 3px; }
.focus-action:disabled { opacity: .55; cursor: wait; }
.focus-action--complete { border-color: transparent; background: var(--home-primary); color: var(--button-primary-text, #fff); }
.focus-action--complete:hover:not(:disabled) { color: #fff; filter: brightness(1.06); }
.focus-action--clear { background: transparent; color: var(--home-secondary); }
.focus-action--pick { border-color: var(--home-edge); background: var(--home-tint); color: var(--home-link); }
.daily-focus-saving { color: var(--home-muted); font-size: 12px; }
.daily-focus-error { display: flex; flex-wrap: wrap; align-items: center; gap: 4px 10px; margin: 0; color: var(--color-danger); font-size: 13px; line-height: 1.7; overflow-wrap: anywhere; }
.daily-focus-error .focus-action { min-height: 32px; padding: 2px 12px; }
.daily-focus-load-error { padding: 6px 0; }
.daily-focus-empty { flex: 1; display: flex; flex-direction: column; justify-content: center; gap: 12px; min-width: 0; }
.daily-focus-empty h2 { margin: 0; color: var(--home-ink); font-size: 30px; font-weight: 600; line-height: 1.45; overflow-wrap: anywhere; }
.daily-focus-empty p { margin: 0; max-width: 48ch; color: var(--home-secondary); font-size: 14px; line-height: 1.8; }
.daily-focus-empty__actions { display: flex; flex-wrap: wrap; gap: 10px; margin-top: 6px; }
.daily-focus-empty__actions :deep(.el-button + .el-button) { margin-left: 0; }
.daily-focus-empty__actions :deep(.el-button) { min-height: 36px; padding-inline: 18px; border-radius: 999px; border-color: var(--home-edge); background: var(--home-tint); color: var(--home-link); font-size: 13px; }
.daily-focus-empty__actions :deep(.el-button:focus-visible) { outline: 2px solid var(--home-link); outline-offset: 3px; }
/* All-context view keeps both scenes as compact summaries instead of two large cards. */
.daily-focus-slot--compact .daily-focus-title { font-size: 20px; line-height: 1.5; }
.daily-focus-slot--compact .daily-focus-title.is-muted { font-size: 18px; }
.daily-focus-slot--compact .daily-focus-actions { gap: 6px 8px; }
.daily-focus-slot--compact .focus-action { min-height: 32px; padding: 4px 14px; font-size: 12px; }
.daily-focus-slot--compact .daily-focus-slot__empty-copy { margin: 0; color: var(--home-muted); font-size: 13px; line-height: 1.7; }
.daily-focus-slot__empty-copy { margin: 0; color: var(--home-secondary); font-size: 14px; line-height: 1.75; }
@media (max-width: 700px) {
  .daily-focus-title { font-size: 26px; }
  .daily-focus-slot--compact .daily-focus-title { font-size: 18px; }
  .daily-focus-empty h2 { font-size: 24px; }
  .focus-action, .daily-focus-empty__actions :deep(.el-button) { min-height: 44px; }
  .daily-focus-slot--compact .focus-action { min-height: 44px; }
}
@media (prefers-reduced-motion: reduce) { .focus-action { transition: none; } }
</style>
