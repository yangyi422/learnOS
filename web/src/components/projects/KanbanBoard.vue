<template>
  <div class="kanban-scroll">
    <div class="kanban-grid">
      <section v-for="column in columns" :key="column.status" class="kanban-column" :class="[`kanban-column--${column.status}`, { 'is-drop-target': dropTarget === column.status }]" :aria-label="column.label">
        <header class="kanban-column__header">
          <div><div class="kanban-column__title"><span class="kanban-column__dot" aria-hidden="true" /><h2>{{ column.label }}</h2></div><small v-if="column.status === 'doing' && counts.doing > 3">建议减少同时进行的任务</small></div>
          <span class="kanban-column__count">{{ counts[column.status] }}</span>
        </header>
        <VueDraggable
          v-model="lists[column.status]"
          class="kanban-column__cards"
          :data-status="column.status"
          :group="{ name: 'project-tasks' }"
          :disabled="busy"
          :delay="240"
          :touch-start-threshold="8"
          :fallback-tolerance="8"
          :bubble-scroll="false"
          :on-move="onDragMove"
          :delay-on-touch-only="true"
          :animation="150"
          ghost-class="task-card--ghost"
          chosen-class="task-card--chosen"
          drag-class="task-card--dragging"
          @start="onDragStart"
          @end="onDragEnd"
        >
          <div
            v-for="task in lists[column.status]"
            :key="task.id"
            class="task-card"
            :class="{ 'task-card--selected': activeTaskId === task.id }"
            :data-task-id="task.id"
            role="button"
            tabindex="0"
            :aria-label="`打开任务：${task.title}`"
            @click="openTaskFromCard(task, $event)"
            @keydown.enter.stop.prevent="openTaskFromCard(task)"
            @keydown.space.stop.prevent="openTaskFromCard(task)"
          >
            <div class="task-card__top">
              <div class="task-card__body" tabindex="-1">
                <strong>{{ task.title }}</strong>
                <span v-if="task.description" class="task-card__summary">{{ task.description }}</span>
              </div>
            </div>
            <div v-if="selectedProjectId === null || task.priority !== 'normal' || task.due_date" class="task-card__footer">
              <span v-if="selectedProjectId === null" class="task-card__project" :style="{ '--project-accent': projects[task.project_id]?.accent || '#94a3b8' }">{{ projects[task.project_id]?.title || '项目' }}</span>
              <span v-if="task.priority !== 'normal'" class="task-card__priority" :class="`task-card__priority--${task.priority}`">{{ priorityLabel(task.priority) }}</span>
              <span v-if="task.due_date" class="task-card__due" :class="{ 'task-card__due--overdue': task.due_date < today && task.status !== 'done' }">{{ dueLabel(task.due_date, task.status) }}</span>
            </div>
          </div>
        </VueDraggable>
        <button class="kanban-column__add" type="button" @click="$emit('add', column.status)">+ 添加任务</button>
        <button v-if="column.status === 'done' && counts.done > lists.done.length" class="kanban-column__more" type="button" @click="$emit('moreDone')">加载更早的已完成任务</button>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { VueDraggable } from 'vue-draggable-plus'
import { localCalendarDate, taskDueDate } from '@/utils/projectTasks'
import type { Project, ProjectTask, TaskPriority, TaskStatus } from '@/api/projects'

const props = defineProps<{ tasks: ProjectTask[]; projects: Record<number, Project>; selectedProjectId: number | null; activeTaskId?: number | null; doneCount: number; busy: boolean; today?: string }>()
const emit = defineEmits<{
  (e: 'open', task: ProjectTask): void
  (e: 'add', status: TaskStatus): void
  (e: 'move', payload: { task: ProjectTask; status: TaskStatus; before_id?: number; after_id?: number }): void
  (e: 'moreDone'): void
}>()
const columns: { status: TaskStatus; label: string }[] = [
  { status: 'inbox', label: '待整理' },
  { status: 'next', label: '下一步' },
  { status: 'doing', label: '进行中' },
  { status: 'done', label: '已完成' },
]
const lists = reactive<Record<TaskStatus, ProjectTask[]>>({ inbox: [], next: [], doing: [], done: [] })
const suppressCardClick = ref(false)
const dropTarget = ref<TaskStatus | null>(null)
let dragScope: number | null = null
let clickSuppressionTimer: number | undefined
const today = computed(() => props.today ?? localCalendarDate(new Date()))
const counts = computed<Record<TaskStatus, number>>(() => ({
  inbox: props.tasks.filter(task => task.status === 'inbox').length,
  next: props.tasks.filter(task => task.status === 'next').length,
  doing: props.tasks.filter(task => task.status === 'doing').length,
  done: props.doneCount,
}))

function restoreLists() {
  for (const column of columns) {
    lists[column.status] = props.tasks.filter(task => task.status === column.status).sort((a, b) => a.sort_order - b.sort_order || a.id - b.id)
  }
}
watch(() => props.tasks, restoreLists, { immediate: true })
onBeforeUnmount(() => window.clearTimeout(clickSuppressionTimer))

function openTaskFromCard(task: ProjectTask, event?: MouseEvent) {
  if (suppressCardClick.value) return
  const clickedBody = event?.target instanceof HTMLElement ? event.target.closest<HTMLElement>('.task-card__body') : null
  const focusTarget = clickedBody || (event?.currentTarget instanceof HTMLElement ? event.currentTarget : null)
  focusTarget?.focus({ preventScroll: true })
  emit('open', task)
}
function onDragStart() {
  if (clickSuppressionTimer) window.clearTimeout(clickSuppressionTimer)
  suppressCardClick.value = true
  dragScope = props.selectedProjectId
}
function onDragMove(event: { to: HTMLElement }) {
  const status = event.to.dataset.status as TaskStatus
  const valid = !props.busy && props.selectedProjectId === dragScope && columns.some(column => column.status === status)
  dropTarget.value = valid ? status : null
  return valid
}
function onDragEnd(event: { item: HTMLElement; to: HTMLElement; from?: HTMLElement; newIndex?: number; oldIndex?: number }) {
  try {
    const id = Number(event.item.dataset.taskId)
    const status = event.to.dataset.status as TaskStatus
    const task = props.tasks.find(item => item.id === id)
    if (!task || props.busy || props.selectedProjectId !== dragScope || !columns.some(column => column.status === status)) { restoreLists(); return }
    // Preserve existing reorder operations, but unchanged placement is a no-op.
    if (event.from === event.to && event.oldIndex === event.newIndex) { restoreLists(); return }
    // VueDraggable reconciles the DOM during the end event. The dropped card
    // may already have been removed from the target DOM before Vue renders it.
    const otherIDs = Array.from(event.to.querySelectorAll<HTMLElement>(':scope > .task-card'))
      .map(card => Number(card.dataset.taskId)).filter(cardID => cardID !== id)
    const index = Math.min(event.newIndex ?? otherIDs.length, otherIDs.length)
    emit('move', { task, status, before_id: otherIDs[index - 1], after_id: otherIDs[index] })
  } finally {
    dropTarget.value = null
    // Sortable dispatches click after dragend. Keep the card from opening the
    // drawer when a long press was used to move it.
    if (clickSuppressionTimer) window.clearTimeout(clickSuppressionTimer)
    clickSuppressionTimer = window.setTimeout(() => {
      suppressCardClick.value = false
      clickSuppressionTimer = undefined
    }, 250)
  }
}

function priorityLabel(value: TaskPriority) { return { high: 'P1', normal: 'P2', low: 'P3' }[value] }
function dueLabel(value: string, status: TaskStatus) {
  value = taskDueDate(value) ?? value
  if (value === today.value) return '今天'
  const tomorrow = new Date()
  tomorrow.setDate(tomorrow.getDate() + 1)
  if (value === localCalendarDate(tomorrow)) return '明天'
  if (value < today.value && status !== 'done') {
    const days = Math.max(1, Math.round((Date.parse(today.value) - Date.parse(value)) / 86400000))
    return `逾期 ${days} 天`
  }
  return `${Number(value.slice(5, 7))}月${Number(value.slice(8, 10))}日`
}
</script>

<style scoped>
.kanban-scroll { width: 100%; min-width: 0; max-width: 100%; overflow-x: auto; overscroll-behavior-x: contain; padding: 0 0 14px; scrollbar-color: var(--border-default) transparent; }
.kanban-grid { display: grid; grid-template-columns: repeat(4, minmax(255px, 1fr)); min-width: 1020px; align-items: start; }
.kanban-column { min-width: 0; min-height: 360px; padding: 0 15px 12px; border-left: 1px solid var(--board-divider); }
.kanban-column.is-drop-target { background: var(--color-primary-soft); outline: 1px solid var(--color-primary); outline-offset: -1px; border-radius: 8px; }
.kanban-column:first-child { padding-left: 0; border-left: 0; }
.kanban-column:last-child { padding-right: 0; }
.kanban-column__header { display: flex; justify-content: space-between; align-items: flex-start; gap: 10px; min-height: 42px; margin-bottom: 13px; padding: 4px 2px 10px; }
.kanban-column__title { display: flex; align-items: center; gap: 9px; }
.kanban-column__dot { width: 7px; height: 7px; flex: none; border-radius: 50%; background: var(--text-tertiary); }
.kanban-column--next .kanban-column__dot { background: var(--color-primary); }
.kanban-column--doing .kanban-column__dot { background: var(--color-cyan); }
.kanban-column--done .kanban-column__dot { background: var(--color-success); }
.kanban-column__header h2 { margin: 0; color: var(--text-primary); font-size: 14px; font-weight: 690; letter-spacing: -.01em; }
.kanban-column__header small { display: block; margin: 7px 0 0 16px; color: var(--color-warning); font-size: 11px; line-height: 1.4; }
.kanban-column__count { display: grid; min-width: 23px; height: 22px; place-items: center; padding: 0 6px; border-radius: 6px; background: var(--board-count-bg); color: var(--text-tertiary); font-size: 11px; font-variant-numeric: tabular-nums; }
.kanban-column__cards { display: grid; align-content: start; gap: 9px; min-height: 42px; }
.task-card { position: relative; display: grid; min-width: 0; gap: 13px; padding: 15px 16px 14px; border: 1px solid var(--card-border); border-radius: 9px; background: var(--card-surface); box-shadow: var(--shadow-card); cursor: grab; transition: border-color 140ms ease, background-color 140ms ease, box-shadow 140ms ease; }
.task-card:active { cursor: grabbing; }
.task-card::before { position: absolute; top: 0; right: 8px; left: 8px; height: 1px; background: linear-gradient(90deg, transparent, var(--card-highlight), transparent); content: ''; pointer-events: none; }
.task-card:hover { border-color: var(--card-border-hover); background: var(--card-surface-hover); box-shadow: var(--shadow-card-hover); }
.task-card:focus-within { border-color: var(--color-primary); box-shadow: 0 0 0 2px var(--focus-ring), var(--shadow-card-hover); }
.task-card--selected { border-color: var(--color-primary); box-shadow: 0 0 0 1px var(--focus-ring), var(--shadow-card); }
.task-card--ghost { opacity: .55; border-color: var(--color-cyan); background: var(--color-primary-soft); }
.task-card--chosen { border-color: var(--color-cyan); box-shadow: 0 0 0 2px var(--focus-ring), var(--shadow-card-hover); }
.task-card--dragging { border-color: var(--color-primary); background: var(--card-surface-hover); box-shadow: 0 0 0 2px var(--focus-ring), var(--shadow-card-hover); }
.task-card__top { display: block; min-width: 0; }
.task-card__body { display: block; min-width: 0; padding: 0; border: 0; background: transparent; color: var(--text-primary); text-align: left; cursor: inherit; }
.task-card__body strong { display: -webkit-box; overflow: hidden; -webkit-box-orient: vertical; -webkit-line-clamp: 2; overflow-wrap: anywhere; font-size: 15px; font-weight: 650; line-height: 1.45; }
.task-card__summary { display: -webkit-box; overflow: hidden; margin-top: 8px; color: var(--text-secondary); font-size: 12px; line-height: 1.55; overflow-wrap: anywhere; -webkit-box-orient: vertical; -webkit-line-clamp: 2; }
.task-card__footer { display: flex; min-width: 0; align-items: center; gap: 7px; color: var(--text-tertiary); font-size: 12px; line-height: 1.35; min-height: 16px; }
.task-card__project { display: inline-flex; flex: 1; min-width: 0; align-items: center; gap: 6px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.task-card__project::before { width: 5px; height: 5px; flex: none; border-radius: 50%; background: var(--project-accent); content: ''; }
.task-card__priority { flex: none; }
.task-card__due { flex: none; margin-left: auto; white-space: nowrap; }
.task-card__priority--high, .task-card__due--overdue { color: var(--color-danger); }
.task-card__priority--low { color: var(--text-tertiary); }
.kanban-column__add, .kanban-column__more { display: block; width: 100%; margin-top: 5px; padding: 8px 9px; border: 1px solid transparent; border-radius: 7px; background: transparent; color: var(--text-secondary); font-size: 12px; text-align: left; cursor: pointer; }
.kanban-column__add:hover, .kanban-column__add:focus-visible, .kanban-column__more:hover { border-color: var(--border-subtle); background: var(--color-primary-soft); color: var(--color-primary); }
@media (max-width: 760px) { .kanban-grid { min-width: 1020px; } .kanban-column { padding-right: 12px; padding-left: 12px; } }
</style>
