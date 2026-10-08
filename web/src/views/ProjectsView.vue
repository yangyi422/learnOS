<template>
  <section class="page-stack projects-page">
    <PageHeader title="项目">
      <template #actions>
        <span v-if="view === 'today'" class="projects-muted">全部活动项目</span><el-select v-else :disabled="busy" :model-value="projectKey" class="projects-switcher" aria-label="切换项目" @change="selectProject(String($event))">
          <el-option label="全部项目" value="all" />
          <el-option v-for="project in projects.filter(item => item.status !== 'archived')" :key="project.id" :label="`${project.title} · ${openCount(project)} 项未完成`" :value="String(project.id)" />
          <el-option label="+ 新建项目" value="new" />
        </el-select>
        <el-button type="primary" @click="openNewTask('inbox')">+ 新建任务</el-button>
      </template>
    </PageHeader>

    <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />
    <div class="projects-toolbar">
      <nav class="projects-tabs" aria-label="项目视图">
        <button v-for="tab in tabs" :key="tab.key" type="button" :class="{ 'is-active': view === tab.key }" :aria-current="view === tab.key ? 'page' : undefined" @click="setView(tab.key)">{{ tab.label }}</button>
      </nav>
    </div>

    <div v-loading="loading" class="projects-content">
      <template v-if="view === 'board'">
        <div v-if="selectedProject" class="projects-context">
          <span class="projects-context__accent" :style="{ background: selectedProject.accent || '#94a3b8' }" />
          <div><strong>{{ selectedProject.icon }} {{ selectedProject.title }}</strong><p v-if="selectedProject.description">{{ selectedProject.description }}</p></div>
          <el-tag size="small" effect="plain">{{ projectStatusLabel(selectedProject.status) }}</el-tag>
        </div>
        <KanbanBoard :tasks="tasks" :projects="projectMap" :selected-project-id="selectedProjectID" :active-task-id="taskDrawerOpen ? editingTask?.id ?? null : null" :done-count="doneCount" :busy="busy || loading" :today="today" @open="openTask" @add="openNewTask" @move="handleMove" @more-done="loadMoreDone" />
        <details v-if="selectedProjectID" class="project-records" @toggle="recordsExpanded = ($event.target as HTMLDetailsElement).open"><summary>项目记录</summary><RecordsPanel v-if="recordsExpanded" :key="selectedProjectID" :project-id="selectedProjectID" /></details>
      </template>

      <div v-else-if="view === 'today'" class="today-view">
        <h2>今天</h2>
        <p class="projects-muted">跨项目查看逾期、今天到期和正在进行的事项；下一步由你选择。</p>
        <p v-if="todayView && !todaySections.some(section => section.tasks.length)" class="projects-muted">今天没有待处理事项，可以从看板选择一个下一步。</p>
        <section v-for="section in todaySections" :key="section.title" class="today-section">
          <div class="today-section__heading"><h3>{{ section.title }}</h3><span>{{ section.tasks.length }}</span></div>
          <p v-if="!section.tasks.length" class="projects-muted">暂无任务</p>
          <article v-for="task in shownTodayTasks(section)" :key="task.id" class="today-task">
            <button type="button" class="today-task__complete" :disabled="busy" :aria-label="`完成任务：${task.title}`" @click="changeTaskStatus(task, 'done')"><span aria-hidden="true">✓</span></button>
            <button type="button" class="today-task__open" @click="openTask(task)"><span class="today-task__title">{{ task.title }}</span><small v-if="task.due_date" :class="{ 'is-overdue': section.key === 'overdue' }">{{ task.due_date }}</small></button>
            <button type="button" class="today-task__project" :aria-label="`查看项目：${projectMap[task.project_id]?.title}`" @click="openProject(projectMap[task.project_id]!)">{{ projectMap[task.project_id]?.title }}</button>
          </article>
          <el-button v-if="section.key === 'next' && section.tasks.length > nextVisibleLimit" text @click="nextExpanded = !nextExpanded">{{ nextExpanded ? '收起下一步' : `查看全部下一步（${section.tasks.length}）` }}</el-button>
        </section>
      </div>

      <div v-else class="project-list-view">
        <div class="project-list-view__heading"><h2>项目列表</h2><el-button @click="openProjectForm()">+ 新建项目</el-button></div>
        <p v-if="!projects.length" class="projects-muted">还没有项目。创建一个项目后，就可以记录任务。</p>
        <article v-for="project in projects" :key="project.id" class="project-list-card" :style="{ '--project-accent': project.accent || '#94a3b8' }">
          <div class="project-list-card__main">
            <div class="project-list-card__title"><span>{{ project.icon || '◻' }}</span><h3>{{ project.title }}</h3><el-tag size="small" effect="plain">{{ projectStatusLabel(project.status) }}</el-tag></div>
            <p v-if="project.description">{{ project.description }}</p>
            <small>{{ openCount(project) }} 项未完成 · {{ project.doing_task_count }} 项进行中 · {{ project.done_task_count }} 项已完成</small>
          </div>
          <div class="project-list-card__actions">
            <el-button text @click="openProject(project)">打开看板</el-button>
            <el-button text @click="openProjectForm(project)">编辑</el-button>
            <el-button v-if="project.status === 'active'" text @click="setProjectStatus(project, 'paused')">暂停</el-button>
            <el-button v-else text @click="setProjectStatus(project, 'active')">恢复</el-button>
            <el-button v-if="project.status !== 'archived'" text @click="setProjectStatus(project, 'archived')">归档</el-button>
          </div>
        </article>
      </div>
    </div>

    <TaskDrawer ref="taskDrawer" v-model="taskDrawerOpen" :task="editingTask" :projects="projects" :default-project-id="selectedProjectID" :default-status="newTaskStatus" :busy="busy" :save-error="drawerError" @save="saveTask" @status="changeTaskStatus" @delete="removeTask" />

    <el-dialog v-model="projectDialogOpen" :title="editingProject ? '编辑项目' : '新建项目'" width="min(460px, 92vw)">
      <el-form label-position="top" @submit.prevent="saveProject">
        <el-form-item label="项目名称"><el-input v-model="projectForm.title" maxlength="160" show-word-limit /></el-form-item>
        <el-form-item label="项目目标 / 说明"><el-input v-model="projectForm.description" type="textarea" :rows="3" maxlength="4000" show-word-limit /></el-form-item>
        <div class="project-form-row">
          <el-form-item label="图标"><el-input v-model="projectForm.icon" maxlength="8" placeholder="可选" /></el-form-item>
          <el-form-item label="强调色">
            <el-select v-model="projectForm.accent" placeholder="默认">
              <el-option label="默认灰" value="" />
              <el-option label="静蓝" value="#6b9dcc" />
              <el-option label="灰绿" value="#7ca98e" />
              <el-option label="暖橙" value="#c79967" />
              <el-option label="柔紫" value="#9a8cba" />
            </el-select>
          </el-form-item>
        </div>
      </el-form>
      <template #footer><el-button @click="projectDialogOpen = false">取消</el-button><el-button type="primary" :loading="busy" :disabled="!projectForm.title.trim()" @click="saveProject">保存</el-button></template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import PageHeader from '@/components/PageHeader.vue'
import KanbanBoard from '@/components/projects/KanbanBoard.vue'
import TaskDrawer from '@/components/projects/TaskDrawer.vue'
import RecordsPanel from '@/components/RecordsPanel.vue'
import { rememberProject } from '@/utils/recentProject'
import { createProject, createTask, deleteTask, getTask, getTodayView, listProjects, listTasks, moveTask, updateProject, updateTask, type Project, type ProjectStatus, type ProjectTask, type TaskInput, type TaskStatus, type TodayView } from '@/api/projects'

import { groupTodayTasks, localCalendarDate } from '@/utils/projectTasks'

type View = 'board' | 'today' | 'list'
const tabs: { key: View; label: string }[] = [{ key: 'board', label: '看板' }, { key: 'today', label: '今天' }, { key: 'list', label: '项目列表' }]
const route = useRoute()
const router = useRouter()
const projects = ref<Project[]>([])
const tasks = ref<ProjectTask[]>([])
const recordsExpanded = ref(false)
const loading = ref(false)
const busy = ref(false)
const error = ref('')
const taskDrawerOpen = ref(false)
const taskDrawer = ref<{ canReplace: () => Promise<boolean> } | null>(null)
const drawerError = ref('')
const optimisticDoneDelta = ref(0)
const editingTask = ref<ProjectTask | null>(null)
const newTaskStatus = ref<TaskStatus>('inbox')
const projectDialogOpen = ref(false)
const editingProject = ref<Project | null>(null)
const projectForm = reactive({ title: '', description: '', icon: '', accent: '' })
let loadSequence = 0
let loadScope = ''
let taskOpenSequence = 0
const donePageSize = 20

const projectKey = computed(() => typeof route.query.project === 'string' ? route.query.project : 'all')
const selectedProjectID = computed(() => {
  const id = Number(projectKey.value)
  return projectKey.value !== 'all' && Number.isSafeInteger(id) && projects.value.some(project => project.id === id) ? id : null
})
const selectedProject = computed(() => projects.value.find(project => project.id === selectedProjectID.value) ?? null)
const view = computed<View>(() => tabs.some(tab => tab.key === route.query.view) ? route.query.view as View : 'board')
const projectMap = computed<Record<number, Project>>(() => Object.fromEntries(projects.value.map(project => [project.id, project])))
const doneCount = computed(() => (selectedProject.value ? selectedProject.value.done_task_count : projects.value.filter(project => project.status !== 'archived').reduce((sum, project) => sum + project.done_task_count, 0)) + optimisticDoneDelta.value)
const today = ref(localCalendarDate(new Date()))
const todayView = ref<TodayView | null>(null)
const todaySections = computed(() => groupTodayTasks(todayView.value, today.value))
const nextVisibleLimit = 6
const nextExpanded = ref(false)
function shownTodayTasks(section: ReturnType<typeof groupTodayTasks>[number]) { return section.key === 'next' && !nextExpanded.value ? section.tasks.slice(0, nextVisibleLimit) : section.tasks }
function projectStatusLabel(status: ProjectStatus) { return { active: '进行中', paused: '已暂停', completed: '已完成', archived: '已归档' }[status] }
function openCount(project: Project) { return project.open_task_count }
function message(reason: unknown) { return reason instanceof Error ? reason.message : '操作失败，请重试' }

async function load() {
  const sequence = ++loadSequence
  const scope = view.value === 'today' ? `today:${today.value}` : `${projectKey.value}:${view.value}`
  if (scope !== loadScope) { tasks.value = []; todayView.value = null; loadScope = scope }
  loading.value = true
  error.value = ''
  try {
    const projectList = await listProjects()
    if (sequence !== loadSequence) return
    projects.value = projectList
    const id = selectedProjectID.value
    if (view.value === 'today') {
      const result = await getTodayView(today.value)
      if (sequence !== loadSequence) return
      todayView.value = result
      tasks.value = todaySections.value.flatMap(section => section.tasks)
      return
    }
    const filter = { projectId: id ?? undefined }
    const columns = await Promise.all([
      listTasks({ ...filter, status: 'inbox' }),
      listTasks({ ...filter, status: 'next' }),
      listTasks({ ...filter, status: 'doing' }),
      listTasks({ ...filter, status: 'done', limit: donePageSize }),
    ])
    if (sequence !== loadSequence) return
    tasks.value = columns.flat()
  } catch (reason) {
    if (sequence === loadSequence) error.value = message(reason)
  } finally {
    if (sequence === loadSequence) loading.value = false
  }
}
async function run(action: () => Promise<void>) {
  if (busy.value) return
  busy.value = true
  try { await action(); await load() } catch (reason) { ElMessage.error(message(reason)); await load() } finally { busy.value = false }
}
function setQuery(patch: Record<string, string>) {
  void router.replace({ path: '/projects', query: { project: projectKey.value, view: view.value, ...patch } })
}
function selectProject(key: string) {
  if (key === 'new') { openProjectForm(); return }
  const id = Number(key); if (projects.value.some(project => project.id === id && project.status !== 'archived')) rememberProject(id)
  setQuery({ project: key })
}
function setView(next: View) { setQuery({ view: next }) }
function openProject(project: Project) { rememberProject(project.id); setQuery({ project: String(project.id), view: 'board' }) }
async function maySwitchTask() { return !taskDrawerOpen.value || !taskDrawer.value || await taskDrawer.value.canReplace() }
async function openNewTask(status: TaskStatus) {
  if (busy.value || !await maySwitchTask()) return
  taskDrawerOpen.value = false
  await nextTick()
  editingTask.value = null
  newTaskStatus.value = status
  drawerError.value = ''
  taskDrawerOpen.value = true
}
async function openTask(task: ProjectTask) {
  if (busy.value || !await maySwitchTask()) return
  editingTask.value = task
  drawerError.value = ''
  taskDrawerOpen.value = true
}
async function persistTask(action: () => Promise<ProjectTask>, creating = false) {
  if (busy.value) return
  busy.value = true
  drawerError.value = ''
  const openID = editingTask.value?.id
  try {
    const saved = await action()
    rememberProject(saved.project_id)
    if (creating) taskDrawerOpen.value = false
    else if (taskDrawerOpen.value && editingTask.value?.id === openID && saved.id === openID) editingTask.value = saved
    await load()
    ElMessage.success(creating ? '任务已创建' : '任务已保存')
  } catch (reason) { drawerError.value = message(reason); ElMessage.error(message(reason)) }
  finally { busy.value = false }
}
async function saveTask(input: TaskInput) {
  const task = editingTask.value
  await persistTask(() => task ? updateTask(task.id, input) : createTask(input), !task)
}
async function changeTaskStatus(task: ProjectTask, status: TaskStatus) {
  if (task.status === status) return
  await persistTask(() => updateTask(task.id, { status }))
}
async function removeTask(task: ProjectTask) {
  try {
    await ElMessageBox.confirm(`删除任务“${task.title}”？删除后无法恢复。`, '确认删除', { type: 'warning' })
    await run(async () => { await deleteTask(task.id); taskDrawerOpen.value = false; ElMessage.success('任务已删除') })
  } catch { /* cancelled */ }
}
async function handleMove(payload: { task: ProjectTask; status: TaskStatus; before_id?: number; after_id?: number }) {
  if (busy.value || loading.value) return
  busy.value = true
  const originalTasks = tasks.value
  const scope = `${projectKey.value}:${view.value}`
  const before = originalTasks.find(task => task.id === payload.before_id)
  const after = originalTasks.find(task => task.id === payload.after_id)
  const order = before && after ? (before.sort_order + after.sort_order) / 2 : before ? before.sort_order + 1024 : after ? after.sort_order - 1024 : 1024
  optimisticDoneDelta.value = Number(payload.status === 'done') - Number(payload.task.status === 'done')
  tasks.value = originalTasks.map(task => task.id === payload.task.id ? { ...task, status: payload.status, sort_order: order } : task)
  try {
    const saved = await moveTask(payload.task.id, { status: payload.status, before_id: payload.before_id, after_id: payload.after_id, expected_updated_at: payload.task.updated_at })
    if (`${projectKey.value}:${view.value}` === scope) tasks.value = tasks.value.map(task => task.id === saved.id ? saved : task)
    optimisticDoneDelta.value = 0
    await load()
  } catch {
    if (`${projectKey.value}:${view.value}` === scope) tasks.value = originalTasks
    optimisticDoneDelta.value = 0
    await load()
    ElMessage.error('任务状态更新失败，已恢复原位置。')
  } finally { busy.value = false }
}
async function loadMoreDone() {
  if (busy.value || loading.value) return
  busy.value = true
  try {
    const sequence = loadSequence
    const older = await listTasks({ projectId: selectedProjectID.value ?? undefined, status: 'done', limit: donePageSize, offset: tasks.value.filter(task => task.status === 'done').length })
    if (sequence === loadSequence) tasks.value = [...tasks.value, ...older]
  } catch (reason) { ElMessage.error(message(reason)) } finally { busy.value = false }
}
function openProjectForm(project?: Project) {
  editingProject.value = project ?? null
  projectForm.title = project?.title ?? ''
  projectForm.description = project?.description ?? ''
  projectForm.icon = project?.icon ?? ''
  projectForm.accent = project?.accent ?? ''
  projectDialogOpen.value = true
}
async function saveProject() {
  if (!projectForm.title.trim()) return
  await run(async () => {
    const input = { title: projectForm.title.trim(), description: projectForm.description.trim(), icon: projectForm.icon.trim(), accent: projectForm.accent }
    if (editingProject.value) await updateProject(editingProject.value.id, input)
    else await createProject(input)
    projectDialogOpen.value = false
    ElMessage.success('项目已保存')
  })
}
async function setProjectStatus(project: Project, status: ProjectStatus) {
  await run(async () => {
    await updateProject(project.id, { status })
    if (status === 'archived' && selectedProjectID.value === project.id) setQuery({ project: 'all', view: 'list' })
  })
}
let dateTimer: number | undefined
function refreshDate() {
  const date = localCalendarDate(new Date())
  if (today.value === date) return
  today.value = date
  if (view.value === 'today') void load()
}
onMounted(() => { void load(); dateTimer = window.setInterval(refreshDate, 60_000); document.addEventListener('visibilitychange', refreshDate) })
onBeforeUnmount(() => { ++loadSequence; window.clearInterval(dateTimer); document.removeEventListener('visibilitychange', refreshDate) })
onBeforeRouteLeave(async () => !taskDrawerOpen.value || !taskDrawer.value || await taskDrawer.value.canReplace())
watch(() => [selectedProjectID.value, view.value], () => { if (view.value === 'board' && selectedProject.value?.status === 'active') rememberProject(selectedProject.value.id) })
watch(() => [projectKey.value, view.value], () => { void load() })
watch(() => route.query.task, async value => {
  const sequence = ++taskOpenSequence
  const id = Number(value)
  if (!value || !Number.isSafeInteger(id) || id <= 0) return
  try {
    const task = await getTask(id)
    if (sequence !== taskOpenSequence || route.query.task !== value) return
    await openTask(task)
    const query = { ...route.query }
    delete query.task
    await router.replace({ path: '/projects', query })
  } catch (reason) {
    ElMessage.error(message(reason))
    const query = { ...route.query }
    delete query.task
    await router.replace({ path: '/projects', query })
  }
}, { immediate: true })
</script>

<style scoped>
.projects-page { max-width: 1440px; }
.project-records { margin-top: 18px; border-top: 1px solid var(--border-subtle); padding-top: 16px; }
.project-records summary { color: var(--text-secondary); font-size: 14px; cursor: pointer; }
.projects-page :deep(.page-header) { align-items: flex-end; gap: 16px; margin-bottom: 17px; }
.projects-page :deep(.page-header__copy) { min-width: 0; }
.projects-page :deep(.page-header .eyebrow) { margin-bottom: 5px; }
.projects-page :deep(.page-header h1) { font-size: clamp(29px, 2.5vw, 36px); line-height: 1.15; }
.projects-page :deep(.page-header__actions) { display: flex; align-items: center; gap: 10px; margin-left: auto; }
.projects-page :deep(.page-header__actions .el-button) { min-height: 38px; padding: 0 17px; box-shadow: 0 3px 12px var(--focus-ring); }
.projects-switcher { width: clamp(190px, 21vw, 270px); }
.projects-page :deep(.projects-switcher .el-select__wrapper) { min-height: 38px; border-radius: var(--radius-control); }
.projects-toolbar { display: flex; align-items: center; min-height: 40px; margin-bottom: 18px; padding-bottom: 8px; border-bottom: 1px solid var(--border-subtle); }
.projects-tabs { display: flex; align-items: center; gap: 22px; }
.projects-tabs button { position: relative; min-height: 31px; padding: 0 2px; border: 0; background: transparent; color: var(--text-secondary); font-size: 13px; cursor: pointer; }
.projects-tabs button:hover { color: var(--text-primary); }
.projects-tabs button.is-active { color: var(--color-primary); font-weight: 670; }
.projects-tabs button.is-active::after { position: absolute; right: 0; bottom: -9px; left: 0; height: 2px; border-radius: 2px; background: var(--color-primary); content: ''; }
.projects-content { min-height: 360px; }
.projects-context { display: flex; align-items: center; gap: 12px; margin-bottom: 15px; padding: 2px 0; }
.projects-context__accent { align-self: stretch; width: 3px; min-height: 30px; border-radius: 4px; }
.projects-context strong { font-size: 14px; font-weight: 670; }
.projects-context p { margin: 4px 0 0; color: var(--text-secondary); font-size: 12px; }
.projects-context .el-tag { margin-left: auto; }
.projects-muted { color: var(--text-tertiary); font-size: 13px; }
.today-view { max-width: 800px; }
.today-view h2, .project-list-view h2 { margin: 0 0 6px; font-size: 23px; }
.today-section { margin-top: 30px; }
.today-section__heading { display: flex; justify-content: space-between; padding-bottom: 10px; border-bottom: 1px solid var(--border-default); }
.today-section__heading h3 { margin: 0; font-size: 15px; }
.today-section__heading span { color: var(--text-tertiary); font-size: 13px; }
.today-task { display: flex; align-items: center; gap: 12px; width: 100%; padding: 14px 4px; border: 0; border-bottom: 1px solid var(--border-subtle); background: transparent; color: var(--text-primary); text-align: left; }
.today-task:hover { background: var(--bg-subtle); }
.today-task__open { display: flex; min-width: 0; flex: 1; flex-direction: column; gap: 4px; border: 0; padding: 0; background: transparent; color: var(--text-primary); text-align: left; cursor: pointer; }
.today-task__title { overflow-wrap: anywhere; }
.today-task__complete { display: grid; flex: none; width: 28px; height: 28px; place-items: center; border: 1px solid var(--border-default); border-radius: 6px; background: transparent; color: var(--text-tertiary); cursor: pointer; }
.today-task__complete:hover { color: var(--color-primary); border-color: var(--color-primary); }
.today-task__project { max-width: 35%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; padding: 6px 0; border: 0; background: transparent; color: var(--text-secondary); font-size: 12px; cursor: pointer; }
.today-task small.is-overdue { color: var(--color-danger); }
.today-task__marker { width: 16px; height: 16px; flex: 0 0 16px; border: 1px solid #b5c2d2; border-radius: 5px; }
.today-task__title { flex: 1; font-size: 14px; }
.today-task small { color: var(--text-tertiary); }
.project-list-view { max-width: 920px; }
.project-list-view__heading { display: flex; justify-content: space-between; align-items: center; margin-bottom: 18px; }
.project-list-card { display: flex; justify-content: space-between; gap: 18px; padding: 20px; margin-bottom: 12px; border: 1px solid var(--border-default); border-left: 3px solid var(--project-accent); border-radius: var(--radius-card); background: var(--bg-surface); }
.project-list-card__title { display: flex; align-items: center; gap: 10px; }
.project-list-card__title h3 { margin: 0; font-size: 17px; }
.project-list-card__main p { margin: 10px 0; color: var(--text-secondary); font-size: 13px; }
.project-list-card__main small { display: block; margin-top: 10px; color: var(--text-tertiary); }
.project-list-card__actions { display: flex; align-items: start; justify-content: flex-end; flex-wrap: wrap; }
.project-form-row { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
.project-form-row :deep(.el-select) { width: 100%; }
@media (max-width: 760px) { .projects-page :deep(.page-header) { align-items: flex-start; } .projects-page :deep(.page-header__actions) { width: 100%; margin-left: 0; } .projects-switcher { width: auto; min-width: 0; flex: 1; } .projects-tabs { width: 100%; justify-content: space-between; } .projects-tabs button { flex: 1; padding: 0 9px; } .project-list-card { align-items: flex-start; flex-direction: column; } .today-task { flex-wrap: wrap; } .today-task__project { margin-left: 40px; max-width: calc(100% - 40px); } .today-task__marker { display: none; } .project-list-card__actions { justify-content: flex-start; } }
</style>
