<template>
  <div class="kanban-scroll">
    <div class="kanban-grid">
      <section v-for="column in columns" :key="column.status" class="kanban-column" :aria-label="column.label">
        <header class="kanban-column__header">
          <div><h2>{{ column.label }}</h2><small v-if="column.status === 'doing' && counts.doing > 3">建议减少同时进行的任务</small></div>
          <span>{{ counts[column.status] }}</span>
        </header>
        <VueDraggable
          v-model="lists[column.status]"
          class="kanban-column__cards"
          :data-status="column.status"
          :group="{ name: 'project-tasks' }"
          :disabled="busy"
          handle=".task-card__handle"
          :animation="150"
          ghost-class="task-card--ghost"
          @end="onDragEnd"
        >
          <div v-for="task in lists[column.status]" :key="task.id" class="task-card" :data-task-id="task.id">
            <button class="task-card__handle" type="button" :aria-label="`拖动任务：${task.title}`" title="拖动排序">⋮⋮</button>
            <button class="task-card__body" type="button" @click="$emit('open', task)">
              <strong>{{ task.title }}</strong>
              <span class="task-card__meta">
                <span v-if="selectedProjectId === null" class="task-card__project" :style="{ '--project-accent': projects[task.project_id]?.accent || '#94a3b8' }">{{ projects[task.project_id]?.title || '项目' }}</span>
                <span v-if="task.priority !== 'normal'" class="task-card__priority" :class="`task-card__priority--${task.priority}`">{{ priorityLabel(task.priority) }}</span>
                <span v-if="task.due_date" :class="{ 'task-card__due--overdue': task.due_date < today && task.status !== 'done' }">{{ dueLabel(task.due_date, task.status) }}</span>
              </span>
            </button>
          </div>
        </VueDraggable>
        <button class="kanban-column__add" type="button" @click="$emit('add', column.status)">+ 添加任务</button>
        <button v-if="column.status === 'done' && counts.done > lists.done.length" class="kanban-column__more" type="button" @click="$emit('moreDone')">加载更早的已完成任务</button>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { VueDraggable } from 'vue-draggable-plus'
import type { Project, ProjectTask, TaskPriority, TaskStatus } from '@/api/projects'

const props = defineProps<{ tasks: ProjectTask[]; projects: Record<number, Project>; selectedProjectId: number | null; doneCount: number; busy: boolean }>()
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
const today = localDate(new Date())
const counts = computed<Record<TaskStatus, number>>(() => ({
  inbox: props.tasks.filter(task => task.status === 'inbox').length,
  next: props.tasks.filter(task => task.status === 'next').length,
  doing: props.tasks.filter(task => task.status === 'doing').length,
  done: props.doneCount,
}))

watch(() => props.tasks, () => {
  for (const column of columns) {
    const sorted = props.tasks.filter(task => task.status === column.status).sort((a, b) => a.sort_order - b.sort_order || a.id - b.id)
    lists[column.status] = sorted
  }
}, { immediate: true })

function onDragEnd(event: { item: HTMLElement; to: HTMLElement; newIndex?: number }) {
  const id = Number(event.item.dataset.taskId)
  const status = event.to.dataset.status as TaskStatus
  const task = props.tasks.find(item => item.id === id)
  if (!task || !columns.some(column => column.status === status)) return
  // VueDraggable reconciles the DOM during the end event. The dropped card
  // may already have been removed from the target DOM before Vue renders it.
  const otherIDs = Array.from(event.to.querySelectorAll<HTMLElement>(':scope > .task-card'))
    .map(card => Number(card.dataset.taskId)).filter(cardID => cardID !== id)
  const index = Math.min(event.newIndex ?? otherIDs.length, otherIDs.length)
  emit('move', { task, status, before_id: otherIDs[index - 1], after_id: otherIDs[index] })
}

function priorityLabel(value: TaskPriority) { return { high: 'P1', normal: 'P2', low: 'P3' }[value] }
function dueLabel(value: string, status: TaskStatus) {
  if (value === today) return '今天'
  const tomorrow = new Date()
  tomorrow.setDate(tomorrow.getDate() + 1)
  if (value === localDate(tomorrow)) return '明天'
  if (value < today && status !== 'done') {
    const days = Math.max(1, Math.round((Date.parse(today) - Date.parse(value)) / 86400000))
    return `逾期 ${days} 天`
  }
  return `${Number(value.slice(5, 7))}月${Number(value.slice(8, 10))}日`
}
function localDate(date: Date) { return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}` }
</script>

<style scoped>
.kanban-scroll { overflow-x: auto; padding-bottom: 8px; }
.kanban-grid { display: grid; grid-template-columns: repeat(4, minmax(260px, 1fr)); gap: 14px; min-width: 1100px; align-items: start; }
.kanban-column { min-height: 380px; padding: 14px; border: 1px solid var(--border-default); border-radius: var(--radius-surface); background: var(--bg-surface); }
.kanban-column__header { display: flex; justify-content: space-between; align-items: start; gap: 8px; min-height: 48px; margin-bottom: 10px; }
.kanban-column__header h2 { margin: 0; font-size: 15px; }
.kanban-column__header small { display: block; margin-top: 5px; color: var(--color-warning); font-size: 11px; }
.kanban-column__header > span { color: var(--text-tertiary); font-size: 13px; }
.kanban-column__cards { display: grid; align-content: start; gap: 8px; min-height: 230px; }
.task-card { display: flex; gap: 6px; border: 1px solid var(--border-default); border-radius: var(--radius-card); background: #fff; box-shadow: 0 2px 8px rgba(15, 23, 42, .035); }
.task-card--ghost { opacity: .35; background: var(--color-primary-soft); }
.task-card__handle { flex: 0 0 20px; padding: 11px 0 0 7px; border: 0; background: none; color: #a3afbf; cursor: grab; }
.task-card__handle:active { cursor: grabbing; }
.task-card__body { display: grid; flex: 1; gap: 12px; padding: 13px 12px 13px 0; border: 0; background: none; color: var(--text-primary); text-align: left; cursor: pointer; }
.task-card__body strong { overflow-wrap: anywhere; font-size: 13px; line-height: 1.5; }
.task-card__meta { display: flex; flex-wrap: wrap; gap: 5px 9px; color: var(--text-tertiary); font-size: 11px; }
.task-card__project { padding-left: 7px; border-left: 3px solid var(--project-accent); }
.task-card__priority--high, .task-card__due--overdue { color: var(--color-danger); }
.task-card__priority--low { color: var(--text-tertiary); }
.kanban-column__add, .kanban-column__more { width: 100%; margin-top: 10px; padding: 8px; border: 0; background: none; color: var(--text-tertiary); text-align: left; cursor: pointer; }
.kanban-column__add:hover, .kanban-column__more:hover { color: var(--color-primary); }
@media (max-width: 760px) { .kanban-grid { min-width: 1020px; gap: 10px; } .kanban-column { padding: 10px; } }
</style>
