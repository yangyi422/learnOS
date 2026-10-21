<template>
  <el-dialog
    class="focus-picker"
    :model-value="modelValue"
    title="选择今日焦点"
    width="min(600px, 94vw)"
    append-to-body
    :close-on-click-modal="!saving"
    :close-on-press-escape="!saving"
    @update:model-value="onDialogValue"
  >
    <p class="focus-picker__intro">{{ intro }}</p>
    <el-input v-model="query" class="focus-picker__search" aria-label="搜索可操作任务" placeholder="搜索该场景其他可操作任务" clearable />
    <p v-if="loadError" class="focus-picker__error daily-focus-error" role="alert">
      <span>{{ loadError }}</span>
      <button type="button" class="focus-picker__inline-action focus-action--retry" @click="loadCandidates">重试</button>
    </p>
    <div class="focus-picker__list" :aria-busy="loading">
      <el-skeleton v-if="loading" animated :rows="4" />
      <template v-else>
        <button
          v-for="task in candidates"
          :key="task.id"
          type="button"
          class="focus-picker-candidate"
          :data-task-id="task.id"
          :disabled="!selectable(task) || saving"
          :aria-disabled="!selectable(task) || saving"
          @click="choose(task)"
        >
          <span class="focus-picker-candidate__title">{{ task.title }}</span>
          <span class="focus-picker-candidate__meta">
            <span>{{ projectTitle(task.project_id) }}</span>
            <span v-if="task.plan_date">计划 {{ task.plan_date }}</span>
            <span v-if="task.due_date" :class="{ 'is-overdue': task.due_date < date }">截止 {{ task.due_date }}</span>
          </span>
          <span class="focus-picker-candidate__action">{{ candidateAction(task) }}</span>
        </button>
        <div v-if="!candidates.length" class="focus-picker__empty" role="status">
          <p>{{ query.trim() ? '没有找到匹配的任务。' : '今天还没有计划中的未完成任务。' }}</p>
          <p class="focus-picker__hint">{{ query.trim() ? '可以换一个搜索词，或清空搜索查看今天计划的任务。' : '可以输入搜索词，从该场景全部可操作任务中选择。' }}</p>
        </div>
      </template>
    </div>
    <p v-if="saveError" class="focus-picker__error daily-focus-error" role="alert">
      <span>{{ saveError }}</span>
      <button v-if="pendingTask" type="button" class="focus-picker__inline-action focus-action--retry" :disabled="saving" @click="retrySave">重试</button>
    </p>
    <template #footer>
      <div class="focus-picker__footer">
        <span v-if="saveError" class="focus-picker__hint">焦点没有改变，仍是之前确认的结果。</span>
        <el-button :disabled="saving" @click="close">取消</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { APIRequestError } from '@/api/http'
import { isFocusCandidate, setDailyFocus, type DailyFocusContext, type DailyFocusView } from '@/api/dailyFocus'
import { listProjects, listTasks, type Project, type ProjectTask } from '@/api/projects'

const props = defineProps<{
  modelValue: boolean
  date: string
  context: DailyFocusContext
  current: DailyFocusView | null
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  saved: [focus: DailyFocusView]
  conflict: []
}>()

const query = ref('')
const candidates = ref<ProjectTask[]>([])
const projects = ref<Project[]>([])
const loading = ref(false)
const loadError = ref('')
const saving = ref(false)
const saveError = ref('')
const pendingTask = ref<ProjectTask | null>(null)

let candidateRequest: AbortController | undefined
let searchTimer: ReturnType<typeof setTimeout> | undefined
let disposed = false

const intro = computed(() => `默认展示${props.context === 'work' ? '工作' : '生活'}场景今天已计划且未完成的任务；输入搜索词可以查找该场景全部可操作任务。`)
const scopeKey = computed(() => `${props.date}:${props.context}`)

function projectTitle(projectID: number) {
  return projects.value.find(project => project.id === projectID)?.title ?? '项目任务'
}
function selectable(task: ProjectTask) { return isFocusCandidate(task) }
function candidateAction(task: ProjectTask) {
  if (!selectable(task)) return '不可选'
  if (props.current?.task_id === task.id) return '当前焦点'
  return '设为焦点'
}
function close() { if (!saving.value) emit('update:modelValue', false) }
function onDialogValue(value: boolean) { if (!value && !saving.value) emit('update:modelValue', false) }

async function loadCandidates() {
  candidateRequest?.abort()
  const controller = candidateRequest = new AbortController()
  const scope = scopeKey.value
  loading.value = true
  loadError.value = ''
  candidates.value = []
  try {
    const searching = query.value.trim() !== ''
    const filters = searching
      ? { context: props.context, actionable: true, q: query.value.trim(), limit: 20, offset: 0, signal: controller.signal }
      : { context: props.context, actionable: true, planDate: props.date, limit: 20, offset: 0, signal: controller.signal }
    const tasks = await listTasks(filters)
    if (disposed || controller !== candidateRequest || controller.signal.aborted || scope !== scopeKey.value) return
    // Server already filters completed/archived tasks; keep a defensive check so a
    // stale or unexpected payload never makes a finished task selectable.
    candidates.value = tasks.filter(isFocusCandidate).slice(0, 20)
    if (!projects.value.length) {
      try {
        const loaded = await listProjects()
        if (!disposed && controller === candidateRequest) projects.value = loaded
      } catch { /* project names are optional context, never block selection */ }
    }
  } catch (reason) {
    if (!disposed && !controller.signal.aborted && controller === candidateRequest && scope === scopeKey.value) {
      loadError.value = reason instanceof Error ? reason.message : '任务暂时无法读取，请重试。'
    }
  } finally {
    if (!disposed && controller === candidateRequest) loading.value = false
  }
}

async function save(task: ProjectTask) {
  if (saving.value || !selectable(task)) return
  saving.value = true
  saveError.value = ''
  pendingTask.value = task
  const scope = scopeKey.value
  try {
    const result = await setDailyFocus(props.date, props.context, task.id, props.current?.revision ?? 0)
    if (disposed || scope !== scopeKey.value) return
    if (result.focus) {
      pendingTask.value = null
      emit('saved', result.focus)
      emit('update:modelValue', false)
    } else {
      saveError.value = '焦点已提交，但未能读取最新状态，请刷新后确认。'
      emit('conflict')
    }
  } catch (reason) {
    if (disposed || scope !== scopeKey.value) return
    if (reason instanceof APIRequestError && reason.status === 409) {
      saveError.value = '焦点刚刚在别处被更新。已刷新最新状态，可以重试。'
      emit('conflict')
    } else {
      saveError.value = reason instanceof Error ? reason.message : '焦点没有保存，请重试。'
    }
  } finally {
    saving.value = false
  }
}
function choose(task: ProjectTask) { void save(task) }
function retrySave() { if (pendingTask.value) void save(pendingTask.value) }

function reset() {
  candidateRequest?.abort()
  clearTimeout(searchTimer)
  query.value = ''
  candidates.value = []
  loadError.value = ''
  saveError.value = ''
  pendingTask.value = null
  loading.value = false
}

watch(() => props.modelValue, open => {
  if (open) {
    reset()
    void loadCandidates()
  } else {
    candidateRequest?.abort()
    clearTimeout(searchTimer)
  }
})
watch([() => props.context, () => props.date], () => { if (props.modelValue) { reset(); void loadCandidates() } })
watch(query, () => {
  candidateRequest?.abort()
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => { if (props.modelValue) void loadCandidates() }, 250)
})
onBeforeUnmount(() => { disposed = true; candidateRequest?.abort(); clearTimeout(searchTimer) })
</script>

<style scoped>
.focus-picker__intro { margin: 0 0 12px; color: var(--text-secondary); font-size: 14px; line-height: 1.7; }
.focus-picker__search { margin-bottom: 16px; }
.focus-picker__list { min-height: 200px; max-height: 48dvh; overflow: auto; }
.focus-picker-candidate { display: grid; grid-template-columns: minmax(0, 1fr) auto; gap: 5px 12px; width: 100%; padding: 13px 4px; border: 0; border-bottom: 1px solid var(--border-subtle); background: transparent; color: var(--text-primary); font: inherit; text-align: left; cursor: pointer; }
.focus-picker-candidate__title { font-size: 14px; font-weight: 500; line-height: 1.6; overflow-wrap: anywhere; }
.focus-picker-candidate__meta { grid-column: 1; display: flex; flex-wrap: wrap; gap: 4px 10px; font-size: 12px; line-height: 1.7; color: var(--text-secondary); }
.focus-picker-candidate__meta .is-overdue { color: var(--color-danger); }
.focus-picker-candidate__action { grid-column: 2; grid-row: 1 / span 2; align-self: center; font-size: 12px; color: var(--color-link); }
.focus-picker-candidate:hover:not(:disabled) { background: var(--home-tint); }
.focus-picker-candidate:focus-visible { outline: 2px solid var(--color-focus); outline-offset: -2px; }
.focus-picker-candidate:disabled { cursor: default; opacity: .55; }
.focus-picker__empty { display: flex; flex-direction: column; gap: 6px; padding: 20px 4px; }
.focus-picker__empty p { margin: 0; color: var(--text-secondary); font-size: 14px; line-height: 1.7; }
.focus-picker__hint { color: var(--text-tertiary); font-size: 12px; line-height: 1.7; }
.focus-picker__error { margin: 8px 0 4px; color: var(--color-danger); font-size: 13px; line-height: 1.7; overflow-wrap: anywhere; }
.focus-picker__inline-action { min-height: 32px; margin-left: 8px; padding: 0 8px; border: 0; background: transparent; color: var(--color-link); font: inherit; font-size: 13px; cursor: pointer; }
.focus-picker__inline-action:focus-visible { outline: 2px solid var(--color-focus); outline-offset: 2px; }
.focus-picker__footer { display: flex; align-items: center; justify-content: flex-end; flex-wrap: wrap; gap: 8px 12px; }
@media (max-width: 700px) {
  .focus-picker-candidate { padding-block: 14px; }
  .focus-picker__inline-action, .focus-picker__footer :deep(.el-button) { min-height: 44px; }
}
</style>
