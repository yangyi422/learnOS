<template>
  <el-drawer
    class="task-drawer"
    modal-class="task-drawer-overlay"
    :model-value="modelValue"
    :title="task ? '任务详情' : '新建任务'"
    direction="rtl"
    :size="drawerSize"
    :with-header="false"
    :before-close="confirmClose"
    destroy-on-close
    @open-auto-focus="focusTitle"
    @closed="restoreFocus"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div ref="workspace" class="task-drawer__workspace" :aria-busy="busy">
      <header class="task-drawer__header">
        <h2>{{ task ? '任务详情' : '新建任务' }}</h2>
        <button type="button" class="task-drawer__close" :aria-label="task ? '关闭任务详情' : '关闭新建任务'" @click="requestClose">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" aria-hidden="true"><path d="M5 5l14 14M19 5L5 19" /></svg>
        </button>
      </header>
      <div class="task-drawer__content">
        <article v-if="task && !editing" class="task-drawer__reading" aria-label="任务内容">
          <h3 ref="readingTitle" tabindex="-1">{{ task.title }}</h3>
          <dl class="task-drawer__metadata">
            <div><dt>项目</dt><dd>{{ projects.find(project => project.id === task?.project_id)?.title || '项目' }}</dd></div>
            <div><dt>状态</dt><dd>{{ statusLabel(task.status) }}</dd></div>
            <div><dt>优先级</dt><dd>{{ priorityLabel(task.priority) }}</dd></div>
            <div><dt>截止日期</dt><dd>{{ task.due_date || '无截止日期' }}</dd></div>
          </dl>
          <p v-if="task.description" class="task-drawer__read-description">{{ task.description }}</p>
          <p v-else class="task-drawer__empty-description">暂无任务说明</p>
          <el-alert v-if="saveError" :title="saveError" type="error" :closable="false" show-icon />
        </article>
      <el-form v-if="editing || !task" class="task-drawer__form" label-position="top" :disabled="busy" @submit.prevent="submit">
        <div class="task-drawer__title-field">
          <span class="task-drawer__field-label">任务标题</span>
          <el-input ref="titleInput" v-model="form.title" aria-label="任务名称" maxlength="200" placeholder="要完成什么？" />
        </div>

        <div class="task-drawer__attributes">
          <el-form-item label="所属项目" class="task-drawer__attribute--wide">
            <el-select v-model="form.project_id" placeholder="选择项目" popper-class="task-select-popper">
              <el-option v-for="project in availableProjects" :key="project.id" :label="project.title" :value="project.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="状态">
            <el-select v-model="form.status" popper-class="task-select-popper">
              <el-option label="待整理" value="inbox" />
              <el-option label="下一步" value="next" />
              <el-option label="进行中" value="doing" />
              <el-option label="已完成" value="done" />
            </el-select>
          </el-form-item>
          <el-form-item label="优先级">
            <el-select v-model="form.priority" popper-class="task-select-popper">
              <el-option label="高" value="high" />
              <el-option label="普通" value="normal" />
              <el-option label="低" value="low" />
            </el-select>
          </el-form-item>
          <el-form-item label="到期日" class="task-drawer__attribute--wide">
            <el-config-provider :locale="zhCn">
              <el-date-picker v-model="form.due_date" type="date" value-format="YYYY-MM-DD" placeholder="可选" popper-class="task-date-popper" clearable />
            </el-config-provider>
          </el-form-item>
        </div>

        <el-form-item label="说明" class="task-drawer__description" :class="{ 'is-near-limit': form.description.length >= 9000 }">
          <el-input v-model="form.description" type="textarea" :rows="6" maxlength="10000" show-word-limit placeholder="记录完成这项任务所需的上下文" @input="resizeDescription" />
        </el-form-item>

        <el-alert v-if="saveError" :title="saveError" type="error" :closable="false" show-icon />
      </el-form>
      <details v-if="task" class="task-drawer__activity">
        <summary>记录</summary>
        <div><p>创建：{{ formatTime(task.created_at) }}</p><p>最近修改：{{ formatTime(task.updated_at) }}</p><p v-if="task.completed_at">完成：{{ formatTime(task.completed_at) }}</p></div>
      </details>
      </div>
      <footer class="task-drawer__actions">
        <el-button v-if="task && !editing" text type="danger" :disabled="busy" @click="$emit('delete', task)">删除任务</el-button>
        <div class="task-drawer__actions-main">
          <template v-if="task && !editing">
            <el-button :disabled="busy" @click="startEditing">编辑</el-button>
            <el-button type="primary" :loading="busy" @click="$emit('status', task, task.status === 'done' ? 'next' : 'done')">{{ task.status === 'done' ? '重新打开' : '标记完成' }}</el-button>
          </template>
          <template v-else>
            <el-button :disabled="busy" @click="cancelEditing">取消</el-button>
            <el-button type="primary" :loading="busy" :disabled="!form.title.trim() || !form.project_id" @click="submit">{{ task ? '保存' : '创建任务' }}</el-button>
          </template>
        </div>
      </footer>
    </div>
  </el-drawer>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { ElMessageBox } from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import type { Project, ProjectTask, TaskInput, TaskPriority, TaskStatus } from '@/api/projects'

const props = defineProps<{
  modelValue: boolean
  task: ProjectTask | null
  projects: Project[]
  defaultProjectId: number | null
  defaultStatus: TaskStatus
  busy: boolean
  saveError?: string
}>()
const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'save', input: TaskInput): void
  (e: 'delete', task: ProjectTask): void
  (e: 'status', task: ProjectTask, status: TaskStatus): void
}>()
const width = ref(window.innerWidth)
const editing = ref(false)
const readingTitle = ref<HTMLElement | null>(null)
const titleInput = ref<{ focus: () => void } | null>(null)
const workspace = ref<HTMLElement | null>(null)
let returnFocus: HTMLElement | null = null
const drawerSize = computed(() => width.value <= 760 ? '100%' : '520px')
const form = reactive<{ title: string; project_id: number | null; status: TaskStatus; priority: TaskPriority; due_date: string; description: string }>({
  title: '', project_id: null, status: 'inbox', priority: 'normal', due_date: '', description: '',
})
const availableProjects = computed(() => props.projects.filter(project => project.status !== 'archived' || project.id === props.task?.project_id))
const initialSnapshot = ref('')
function snapshot() { return JSON.stringify([form.title, form.project_id, form.status, form.priority, form.due_date, form.description]) }
const dirty = computed(() => props.modelValue && (editing.value || !props.task) && initialSnapshot.value !== snapshot())
watch(() => props.modelValue, (open, wasOpen) => {
  if (open && !wasOpen && document.activeElement instanceof HTMLElement) returnFocus = document.activeElement
}, { flush: 'sync' })
function resetForm() {
  form.title = props.task?.title ?? ''
  form.project_id = props.task?.project_id ?? props.defaultProjectId
  form.status = props.task?.status ?? props.defaultStatus
  form.priority = props.task?.priority || 'normal'
  form.due_date = props.task?.due_date ?? ''
  form.description = props.task?.description ?? ''
  initialSnapshot.value = snapshot()
  resizeDescription()
}
watch(() => [props.modelValue, props.task], () => {
  if (!props.modelValue) return
  editing.value = !props.task
  resetForm()
  void nextTick(focusTitle)
}, { immediate: true })
let discardDecision: Promise<boolean> | null = null
async function canReplace(): Promise<boolean> {
  if (props.busy) return false
  if (!dirty.value) return true
  if (!discardDecision) {
    discardDecision = ElMessageBox.confirm('当前修改尚未保存。确定放弃吗？', '放弃未保存的修改', {
      confirmButtonText: '放弃修改', cancelButtonText: '继续编辑', type: 'warning',
    }).then(() => true, () => false).finally(() => { discardDecision = null })
  }
  return discardDecision
}
async function confirmClose(done: () => void) { if (await canReplace()) done() }
function requestClose() { void confirmClose(() => emit('update:modelValue', false)) }
function focusTitle() {
  resizeDescription()
  requestAnimationFrame(() => { if (editing.value || !props.task) titleInput.value?.focus(); else readingTitle.value?.focus({ preventScroll: true }) })
}
function startEditing() { resetForm(); editing.value = true; void nextTick(focusTitle) }
async function cancelEditing() {
  if (!await canReplace()) return
  if (!props.task) { emit('update:modelValue', false); return }
  resetForm()
  editing.value = false
  void nextTick(focusTitle)
}
defineExpose({ canReplace })
function statusLabel(status: TaskStatus) { return { inbox: '待整理', next: '下一步', doing: '进行中', done: '已完成' }[status] }
function priorityLabel(priority: TaskPriority) { return { high: '高', normal: '普通', low: '低' }[priority] || '普通' }
function resizeDescription() {
  requestAnimationFrame(() => {
    const textarea = workspace.value?.querySelector<HTMLTextAreaElement>('.task-drawer__description textarea')
    if (!textarea) return
    textarea.style.height = 'auto'
    const style = getComputedStyle(textarea)
    const minHeight = Number.parseFloat(style.minHeight)
    const maxHeight = Number.parseFloat(style.maxHeight)
    const contentHeight = textarea.scrollHeight
    textarea.style.height = `${Math.min(Math.max(contentHeight, minHeight), maxHeight)}px`
    textarea.style.overflowY = contentHeight > maxHeight ? 'auto' : 'hidden'
  })
}
function restoreFocus() {
  if (returnFocus?.isConnected) returnFocus.focus({ preventScroll: true })
  returnFocus = null
}
function submit() {
  if (!form.title.trim() || !form.project_id || props.busy) return
  emit('save', {
    title: form.title.trim(),
    project_id: form.project_id,
    status: form.status,
    priority: form.priority,
    due_date: form.due_date,
    description: form.description.trim(),
  })
}
function formatTime(value: string) { return new Date(value).toLocaleString('zh-CN') }
function onResize() { width.value = window.innerWidth; resizeDescription() }
onMounted(() => window.addEventListener('resize', onResize))
onUnmounted(() => window.removeEventListener('resize', onResize))
</script>

<style>
.task-drawer.el-drawer.rtl { top: 12px; right: 12px; height: calc(100dvh - 24px); border: 1px solid var(--drawer-edge); border-radius: 12px; background: var(--drawer-bg); color: var(--text-primary); box-shadow: var(--drawer-shadow); overflow: hidden; transition: transform 170ms ease-out; }
.task-drawer .el-drawer__body { display: flex; min-height: 0; padding: 0; overflow: hidden; }
.task-drawer__workspace { display: flex; width: 100%; min-height: 0; flex-direction: column; }
.task-drawer__header { display: flex; flex: none; align-items: center; justify-content: space-between; gap: 16px; padding: 16px 22px 14px; border-bottom: 1px solid var(--border-subtle); }
.task-drawer__header h2 { margin: 0; color: var(--text-primary); font-size: 17px; font-weight: 680; line-height: 1.4; }
.task-drawer__close { display: grid; width: 32px; height: 32px; flex: none; place-items: center; padding: 0; border: 1px solid transparent; border-radius: 8px; background: transparent; color: var(--text-secondary); cursor: pointer; transition: border-color 140ms ease, background-color 140ms ease, color 140ms ease; }
.task-drawer__close:hover { border-color: var(--border-default); background: var(--drawer-control); color: var(--text-primary); }
.task-drawer__close svg { width: 16px; height: 16px; }
.task-drawer__content { min-height: 0; flex: 1; overflow-y: auto; overscroll-behavior: contain; padding: 24px 26px 30px; scrollbar-color: var(--border-default) transparent; }
.task-drawer__form { min-width: 0; }
.task-drawer__reading h3 { margin: 0 0 20px; overflow-wrap: anywhere; color: var(--text-primary); font-size: 24px; line-height: 1.5; }
.task-drawer__metadata { display: flex; flex-wrap: wrap; gap: 12px 18px; margin: 0 0 24px; color: var(--text-secondary); font-size: 12px; }
.task-drawer__metadata div { min-width: 0; }
.task-drawer__metadata dt { color: var(--text-tertiary); margin-bottom: 4px; }
.task-drawer__metadata dd { margin: 0; overflow-wrap: anywhere; }
.task-drawer__read-description { margin: 0; white-space: pre-wrap; overflow-wrap: anywhere; color: var(--text-primary); font-size: 14px; line-height: 1.85; }
.task-drawer__empty-description { color: var(--text-tertiary); font-size: 13px; }
.task-drawer__field-label { display: block; margin-bottom: 4px; color: var(--text-tertiary); font-size: 12px; }
.task-drawer__title-field { margin-bottom: 24px; }
.task-drawer__title-field .el-input__wrapper { margin: 0 -7px; padding: 7px 7px 10px; border-bottom: 1px solid var(--border-subtle); border-radius: 6px 6px 0 0; background: transparent; box-shadow: none !important; transition: border-color 140ms ease, background-color 140ms ease, box-shadow 140ms ease; }
.task-drawer__title-field .el-input__wrapper:hover { border-color: var(--drawer-control-border-hover); background: var(--drawer-attribute-bg); }
.task-drawer__title-field .el-input__wrapper:focus-within { border-color: var(--color-primary); box-shadow: 0 1px 0 var(--color-primary) !important; }
.task-drawer__title-field .el-input__inner { height: auto; color: var(--text-primary); font-size: 23px; font-weight: 650; line-height: 1.45; }
.task-drawer__title-field .el-input__inner::placeholder { color: var(--text-tertiary); opacity: .8; }
.task-drawer__attributes { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 0 12px; padding: 16px 18px 2px; border-radius: 9px; background: var(--drawer-attribute-bg); }
.task-drawer__attributes .el-form-item { min-width: 0; margin-bottom: 14px; }
.task-drawer__attributes .el-form-item__label, .task-drawer__description .el-form-item__label { margin-bottom: 6px; color: var(--text-tertiary); font-size: 12px; font-weight: 500; line-height: 1.4; }
.task-drawer__attribute--wide { grid-column: 1 / -1; }
.task-drawer__attributes .el-select, .task-drawer__attributes .el-date-editor, .task-drawer__attributes .el-config-provider { width: 100%; }
.task-drawer__attributes .el-select__wrapper, .task-drawer__attributes .el-input__wrapper { min-height: 37px; border-radius: 7px; background: var(--drawer-control); box-shadow: 0 0 0 1px var(--drawer-control-border) inset; transition: box-shadow 140ms ease, background-color 140ms ease; }
.task-drawer__attributes .el-select__wrapper:hover, .task-drawer__attributes .el-input__wrapper:hover { box-shadow: 0 0 0 1px var(--drawer-control-border-hover) inset; }
.task-drawer__attributes .el-select__selected-item, .task-drawer__attributes .el-input__inner { color: var(--text-primary); font-size: 13px; }
.task-drawer__description { margin: 22px 0 0; }
.task-drawer__description .el-textarea__inner { min-height: 154px; max-height: 326px; padding: 13px 14px 27px; overflow-y: hidden; border: 1px solid var(--drawer-control-border); border-radius: 8px; background: var(--drawer-control); box-shadow: none; color: var(--text-primary); font-size: 14px; line-height: 1.65; resize: none; transition: border-color 140ms ease, background-color 140ms ease, box-shadow 140ms ease; }
.task-drawer__description .el-textarea__inner:hover { border-color: var(--drawer-control-border-hover); }
.task-drawer__description .el-textarea__inner:focus { border-color: var(--color-primary); box-shadow: 0 0 0 3px var(--focus-ring); }
.task-drawer__description .el-input__count { right: 11px; bottom: 7px; background: var(--drawer-control); color: var(--text-tertiary); opacity: 0; transition: opacity 140ms ease; }
.task-drawer__description:focus-within .el-input__count, .task-drawer__description.is-near-limit .el-input__count { opacity: 1; }
.task-drawer__activity { margin-top: 26px; padding-top: 14px; border-top: 1px solid var(--border-subtle); color: var(--text-tertiary); font-size: 12px; }
.task-drawer__activity summary { color: var(--text-secondary); cursor: pointer; transition: color 140ms ease; }
.task-drawer__activity summary:hover { color: var(--text-primary); }
.task-drawer__activity div { padding: 11px 0 0 18px; }
.task-drawer__activity p { margin: 0 0 6px; }
.task-drawer__actions { display: flex; flex: none; align-items: center; justify-content: space-between; gap: 10px; padding: 14px 22px; border-top: 1px solid var(--border-subtle); background: var(--drawer-bg); box-shadow: var(--drawer-footer-shadow); }
.task-drawer__actions-main { display: flex; align-items: center; justify-content: flex-end; gap: 7px; margin-left: auto; }
.task-drawer__actions .el-button { min-height: 36px; margin-left: 0; transition: border-color 140ms ease, background-color 140ms ease, color 140ms ease, box-shadow 140ms ease; }
.task-drawer-overlay.el-overlay { background: var(--task-overlay); }
.task-drawer-overlay.el-drawer-fade-enter-active, .task-drawer-overlay.el-drawer-fade-leave-active { transition: background-color 170ms ease-out, opacity 170ms ease-out; }
.task-select-popper.el-popper, .task-date-popper.el-popper { border-color: var(--drawer-control-border); background: var(--bg-raised); color: var(--text-primary); box-shadow: var(--shadow-popover); }
.task-select-popper .el-select-dropdown__item { color: var(--text-secondary); }
.task-select-popper .el-select-dropdown__item.hover, .task-select-popper .el-select-dropdown__item:hover { background: var(--bg-subtle); color: var(--text-primary); }
.task-select-popper .el-select-dropdown__item.is-selected { color: var(--color-primary); }
.task-date-popper .el-picker-panel { border-color: var(--border-default); background: var(--bg-raised); color: var(--text-primary); }
.task-date-popper .el-date-table td.available:hover, .task-date-popper .el-picker-panel__icon-btn:hover { color: var(--color-primary); }
.el-message-box { border-color: var(--border-default); background: var(--bg-raised); color: var(--text-primary); }
.el-message-box__title { color: var(--text-primary); }
.el-message-box__message { color: var(--text-secondary); }
@media (max-width: 760px) {
  .task-drawer.el-drawer.rtl { top: 0; right: 0; height: 100dvh; border: 0; border-radius: 0; }
  .task-drawer__header { padding: 14px 18px; }
  .task-drawer__content { padding: 20px 18px 28px; }
  .task-drawer__actions { flex-wrap: wrap; padding: 12px 16px max(12px, env(safe-area-inset-bottom)); }
}
</style>
