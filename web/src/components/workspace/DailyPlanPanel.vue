<template>
  <div class="daily-plan">
    <SectionHeader title="今日计划" id="today-title"><template #actions><el-button text :disabled="loading || busy" @click="openPicker">加入今天</el-button></template></SectionHeader>
    <p v-if="error" class="plan-error" role="alert">{{ error }} <button class="inline-action" @click="$emit('refresh')">重试</button></p>
    <template v-else>
      <p class="plan-summary">已安排 {{ planning?.summary.planned ?? 0 }} 项<span v-if="reminderCount">，另有 {{ reminderCount }} 项待处理提醒</span></p>
      <template v-if="planning?.items.length">
        <PlanningTaskRow v-for="task in visibleItems" :key="task.id" :task="task" :date="date" :context="context" planned :busy="busy || loading" @complete="complete" @open="$emit('open', $event)" @remove="remove" @schedule="reschedule" />
        <button v-if="planning.items.length > 5" class="inline-action plan-expand" @click="expanded = !expanded">{{ expanded ? '收起计划' : `查看全部计划（${planning.items.length}）` }}</button>
      </template>
      <EmptyState v-else :title="context === 'life' ? '今天可以留白' : '今天还没有安排'" description="可以从已有任务中选择，也可以随时记录想法。" />
      <details v-if="reminderCount" class="plan-reminders">
        <summary>待处理提醒（{{ reminderCount }}）</summary>
        <section v-for="group in reminderGroups.filter(item => item.tasks.length)" :key="group.key" :aria-label="group.title">
          <h3>{{ group.title }} <span>{{ group.tasks.length }}</span></h3>
          <PlanningTaskRow v-for="task in group.tasks.slice(0, reminderExpanded ? undefined : 4)" :key="task.id" :task="task" :date="date" :context="context" :busy="busy || loading" @complete="complete" @open="$emit('open', $event)" @today="addToday" @schedule="reschedule" />
        </section>
        <button v-if="reminderGroups.some(group => group.tasks.length > 4)" class="inline-action plan-expand" @click="reminderExpanded = !reminderExpanded">{{ reminderExpanded ? '收起提醒' : '展开全部提醒' }}</button>
      </details>
    </template>
    <p v-if="message" class="plan-feedback" role="status">{{ message }}</p>
    <p v-if="writeError" class="plan-error" role="alert">{{ writeError }} <button v-if="failed" class="inline-action" :disabled="busy" @click="retryWrite">重试操作</button><button v-else class="inline-action" @click="$emit('refresh')">刷新任务</button></p>

    <el-dialog v-model="pickerOpen" title="选择已有任务加入今天" width="min(560px, 94vw)" append-to-body :close-on-click-modal="!busy" :close-on-press-escape="!busy">
      <p class="plan-dialog-copy">{{ context === 'work' ? '工作项目中的未完成任务' : context === 'life' ? '生活项目中的未完成任务' : '所有可行动项目中的未完成任务' }}。只修改计划日期，不改变截止日期。</p>
      <el-input v-model="query" aria-label="搜索可安排任务" placeholder="搜索任务" clearable />
      <p v-if="candidateError" class="plan-error" role="alert">{{ candidateError }} <el-button text @click="loadCandidates">重试读取</el-button></p>
      <div class="plan-candidates" :aria-busy="candidatesLoading">
        <el-skeleton v-if="candidatesLoading" :rows="4" />
        <template v-else>
          <button v-for="task in candidates" :key="task.id" class="plan-candidate" :disabled="busy || task.plan_date === date" @click="addToday(task, true)"><strong>{{ task.title }}</strong><small>{{ candidateProjects.find(p => p.id === task.project_id)?.title }}<span v-if="context === 'all'"> · {{ projectContextLabel(task.project_id) }}</span><span v-if="task.plan_date"> · 计划 {{ task.plan_date }}</span><span v-if="task.due_date"> · 截止 {{ task.due_date }}</span></small><span>{{ task.plan_date === date ? '已在今天' : '加入今天' }}</span></button>
          <EmptyState v-if="!candidateError && !candidates.length" title="没有可选择的任务" description="可在项目中创建任务，或调整搜索词。" />
        </template>
      </div>
      <p v-if="writeError" class="plan-error" role="alert">{{ writeError }}</p>
      <template #footer><div class="plan-picker-footer"><el-button :disabled="offset === 0 || candidatesLoading || busy" @click="changePage(-1)">上一页</el-button><el-button :disabled="!hasMore || candidatesLoading || busy" @click="changePage(1)">下一页</el-button><el-button :disabled="busy" @click="pickerOpen = false">关闭</el-button></div></template>
    </el-dialog>
    <el-dialog :model-value="rescheduleTask !== null" title="重新安排计划日期" width="min(420px, 94vw)" append-to-body :close-on-click-modal="!busy" :close-on-press-escape="!busy" @update:model-value="(value: boolean) => { if (!value && !busy) rescheduleTask = null }">
      <p class="plan-dialog-copy">{{ rescheduleTask?.title }}</p><p class="plan-dialog-copy">仅调整计划日期，截止日期保持 {{ rescheduleTask?.due_date || '未设置' }}。</p>
      <el-config-provider :locale="zhCn"><el-date-picker v-model="newDate" type="date" value-format="YYYY-MM-DD" aria-label="新的计划日期" placeholder="选择执行日期" :clearable="false" /></el-config-provider>
      <p v-if="writeError" class="plan-error" role="alert">{{ writeError }}</p>
      <template #footer><el-button :disabled="busy" @click="rescheduleTask = null">取消</el-button><el-button type="primary" :disabled="!newDate" :loading="busy" @click="saveDate">保存计划日期</el-button></template>
    </el-dialog>
  </div>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import { APIRequestError } from '@/api/http'
import { listProjects, listTasks, updateTask, type Project, type ProjectTask, type TaskInput } from '@/api/projects'
import type { PlanningTask, WorkspacePlanning } from '@/api/workspace'
import type { WorkspaceContext } from '@/workspace-context'
import SectionHeader from '@/components/SectionHeader.vue'
import EmptyState from '@/components/EmptyState.vue'
import PlanningTaskRow from './PlanningTaskRow.vue'
const props = defineProps<{ planning?: WorkspacePlanning; date: string; context: WorkspaceContext; loading: boolean; error?: string }>()
defineEmits<{ open: [task: ProjectTask]; refresh: [] }>()
const expanded = ref(false), reminderExpanded = ref(false)
const busy = ref(false), message = ref(''), writeError = ref('')
const pickerOpen = ref(false), query = ref(''), candidatesLoading = ref(false), candidateError = ref('')
const candidates = ref<ProjectTask[]>([]), candidateProjects = ref<Project[]>([])
const offset = ref(0), hasMore = ref(false)
const rescheduleTask = ref<ProjectTask | null>(null), newDate = ref('')
type Attempt = { task: ProjectTask; patch: Partial<TaskInput>; closePicker: boolean }
const failed = ref<Attempt | null>(null)
let candidateRequest: AbortController | undefined, searchTimer: ReturnType<typeof setTimeout> | undefined
const visibleItems = computed(() => props.planning?.items.slice(0, expanded.value ? undefined : 5) ?? [])
const reminderGroups = computed(() => [
  { key: 'due', title: '今天到期', tasks: props.planning?.due_today ?? [] },
  { key: 'overdue', title: '已逾期', tasks: props.planning?.overdue ?? [] },
  { key: 'previous', title: '此前计划未完成', tasks: props.planning?.previous_plan ?? [] },
])
const reminderCount = computed(() => reminderGroups.value.reduce((sum, group) => sum + group.tasks.length, 0))
function projectContextLabel(id: number) { const value = candidateProjects.value.find(p => p.id === id)?.context; return value === 'work' ? '工作' : value === 'life' ? '生活' : '未分类' }
async function perform(task: ProjectTask, patch: Partial<TaskInput>, closePicker = false) {
  if (busy.value || props.loading) return
  busy.value = true; writeError.value = ''; message.value = ''; failed.value = null
  const scope = `${props.date}:${props.context}`
  try {
    await updateTask(task.id, { ...patch, expected_updated_at: task.updated_at })
    if (scope !== `${props.date}:${props.context}`) return
    message.value = patch.status === 'done' ? '任务已完成' : patch.plan_date === null ? '已移出今天，任务仍保留在项目中' : '计划日期已保存'
    if (closePicker) pickerOpen.value = false
    rescheduleTask.value = null
  } catch (reason) {
    if (scope !== `${props.date}:${props.context}`) return
    writeError.value = reason instanceof Error ? reason.message : '操作失败，请重试。'
    if (reason instanceof APIRequestError && reason.status === 409) { pickerOpen.value = false; rescheduleTask.value = null }
    else failed.value = { task, patch, closePicker }
  } finally { busy.value = false }
}
function complete(task: ProjectTask) { void perform(task, { status: 'done' }) }
function remove(task: ProjectTask) { void perform(task, { plan_date: null }) }
function addToday(task: ProjectTask, closePicker = false) { void perform(task, { plan_date: props.date }, closePicker) }
function retryWrite() { const attempt = failed.value; if (attempt) void perform(attempt.task, attempt.patch, attempt.closePicker) }
function reschedule(task: ProjectTask) { writeError.value = ''; rescheduleTask.value = task; newDate.value = task.plan_date || props.date }
function saveDate() { if (rescheduleTask.value && newDate.value) void perform(rescheduleTask.value, { plan_date: newDate.value }) }
function openPicker() { query.value = ''; offset.value = 0; writeError.value = ''; pickerOpen.value = true; void loadCandidates() }
async function loadCandidates() {
  candidateRequest?.abort()
  const controller = candidateRequest = new AbortController()
  candidatesLoading.value = true; candidateError.value = ''; candidates.value = []
  try {
    const [tasks, projects] = await Promise.all([listTasks({ context: props.context, actionable: true, q: query.value.trim(), limit: 21, offset: offset.value, signal: controller.signal }), listProjects()])
    if (controller !== candidateRequest || controller.signal.aborted || !pickerOpen.value) return
    candidates.value = tasks.slice(0, 20); hasMore.value = tasks.length > 20; candidateProjects.value = projects
  } catch (reason) { if (!controller.signal.aborted && controller === candidateRequest) candidateError.value = reason instanceof Error ? reason.message : '任务暂时无法读取。' }
  finally { if (controller === candidateRequest) candidatesLoading.value = false }
}
function changePage(delta: number) { offset.value += delta * 20; void loadCandidates() }
watch(query, () => { offset.value = 0; candidateRequest?.abort(); clearTimeout(searchTimer); searchTimer = setTimeout(() => { if (pickerOpen.value) void loadCandidates() }, 250) })
watch(pickerOpen, open => { if (!open) { candidateRequest?.abort(); clearTimeout(searchTimer) } })
watch(() => [props.context, props.date], () => { pickerOpen.value = false; rescheduleTask.value = null; message.value = ''; writeError.value = ''; failed.value = null; expanded.value = false; reminderExpanded.value = false })
onBeforeUnmount(() => { candidateRequest?.abort(); clearTimeout(searchTimer) })
</script>
<style scoped>
.inline-action { min-height:36px; padding:0; border:0; background:transparent; color:var(--home-link); font:inherit; cursor:pointer; }
.inline-action:focus-visible,.plan-reminders summary:focus-visible { outline:2px solid var(--home-link); outline-offset:3px; }
.plan-summary { margin:0 0 6px; font-size:13px; line-height:1.7; color:var(--home-muted); }
.plan-reminders { margin-top:18px; padding-top:16px; border-top:1px solid var(--home-edge); }
.plan-reminders summary { padding:6px 0; color:var(--home-secondary); cursor:pointer; font-size:14px; }
.plan-reminders h3 { margin:18px 0 0; font-size:13px; color:var(--home-secondary); font-weight:600; }
.plan-reminders h3 span { margin-left:8px; color:var(--home-muted); font-weight:400; }
.plan-feedback,.plan-error { margin:12px 0 0; font-size:13px; line-height:1.7; overflow-wrap:anywhere; }
.plan-feedback { color:var(--home-secondary); }
.plan-error { color:var(--color-danger); }
.plan-expand { min-height:36px; margin-top:8px; }
.plan-dialog-copy { color:var(--text-secondary); font-size:14px; line-height:1.7; overflow-wrap:anywhere; }
.plan-candidates { min-height:200px; max-height:48dvh; overflow:auto; margin-top:16px; }
.plan-candidate { width:100%; display:grid; grid-template-columns:minmax(0,1fr) auto; gap:6px 12px; padding:14px 4px; border:0; border-bottom:1px solid var(--border-subtle); background:transparent; color:var(--text-primary); text-align:left; cursor:pointer; font:inherit; }
.plan-candidate strong { font-size:14px; font-weight:500; overflow-wrap:anywhere; }
.plan-candidate small { grid-column:1; font-size:12px; line-height:1.7; color:var(--text-secondary); overflow-wrap:anywhere; }
.plan-candidate > span { grid-column:2; grid-row:1 / span 2; align-self:center; font-size:12px; color:var(--color-link); }
.plan-candidate:hover { background:var(--home-tint); }
.plan-candidate:focus-visible { outline:2px solid var(--color-focus); outline-offset:-2px; }
.plan-candidate:disabled { cursor:default; opacity:.6; }
.plan-picker-footer { display:flex; justify-content:flex-end; flex-wrap:wrap; gap:8px; }
@media(max-width:700px) { .plan-expand,.plan-reminders summary,.inline-action { min-height:44px; } .daily-plan :deep(.section-header .el-button),.plan-picker-footer :deep(.el-button) { min-height:44px; } }
</style>
