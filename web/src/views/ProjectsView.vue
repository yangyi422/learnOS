<template>
  <section class="page-stack projects-page">
    <PageHeader eyebrow="个人工作台" title="项目" description="项目决定推进什么，看板呈现进展，今天聚焦当下。">
      <template #actions><el-button type="primary" @click="openNewTask('inbox')">+ 新建任务</el-button></template>
    </PageHeader>

    <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />
    <div class="projects-toolbar">
      <el-select :model-value="projectKey" class="projects-switcher" aria-label="切换项目" @change="selectProject(String($event))">
        <el-option label="全部项目" value="all" />
        <el-option v-for="project in projects.filter(item => item.status !== 'archived')" :key="project.id" :label="`${project.title} · ${openCount(project)} 项未完成`" :value="String(project.id)" />
        <el-option label="+ 新建项目" value="new" />
      </el-select>
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
        <KanbanBoard :tasks="tasks" :projects="projectMap" :selected-project-id="selectedProjectID" :done-count="doneCount" :busy="busy" @open="openTask" @add="openNewTask" @move="handleMove" @more-done="loadMoreDone" />
      </template>

      <div v-else-if="view === 'today'" class="today-view">
        <h2>今天</h2>
        <p class="projects-muted">正在做和到期的任务排在前面；“下一步”是可执行候选。</p>
        <section v-for="section in todaySections" :key="section.title" class="today-section">
          <div class="today-section__heading"><h3>{{ section.title }}</h3><span>{{ section.tasks.length }}</span></div>
          <p v-if="!section.tasks.length" class="projects-muted">暂无任务</p>
          <button v-for="task in section.tasks" :key="task.id" type="button" class="today-task" @click="openTask(task)">
            <span class="today-task__marker" />
            <span class="today-task__title">{{ task.title }}</span>
            <small>{{ projectMap[task.project_id]?.title }}<template v-if="task.due_date"> · {{ task.due_date }}</template></small>
          </button>
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

    <TaskDrawer v-model="taskDrawerOpen" :task="editingTask" :projects="projects" :default-project-id="selectedProjectID" :default-status="newTaskStatus" :busy="busy" @save="saveTask" @delete="removeTask" />

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
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import PageHeader from '@/components/PageHeader.vue'
import KanbanBoard from '@/components/projects/KanbanBoard.vue'
import TaskDrawer from '@/components/projects/TaskDrawer.vue'
import { createProject, createTask, deleteTask, listProjects, listTasks, moveTask, updateProject, updateTask, type Project, type ProjectStatus, type ProjectTask, type TaskInput, type TaskStatus } from '@/api/projects'

type View = 'board' | 'today' | 'list'
const tabs: { key: View; label: string }[] = [{ key: 'board', label: '看板' }, { key: 'today', label: '今天' }, { key: 'list', label: '项目列表' }]
const route = useRoute()
const router = useRouter()
const projects = ref<Project[]>([])
const tasks = ref<ProjectTask[]>([])
const loading = ref(false)
const busy = ref(false)
const error = ref('')
const taskDrawerOpen = ref(false)
const editingTask = ref<ProjectTask | null>(null)
const newTaskStatus = ref<TaskStatus>('inbox')
const projectDialogOpen = ref(false)
const editingProject = ref<Project | null>(null)
const projectForm = reactive({ title: '', description: '', icon: '', accent: '' })
let loadSequence = 0
const donePageSize = 20

const projectKey = computed(() => typeof route.query.project === 'string' ? route.query.project : 'all')
const selectedProjectID = computed(() => {
  const id = Number(projectKey.value)
  return projectKey.value !== 'all' && Number.isSafeInteger(id) && projects.value.some(project => project.id === id) ? id : null
})
const selectedProject = computed(() => projects.value.find(project => project.id === selectedProjectID.value) ?? null)
const view = computed<View>(() => tabs.some(tab => tab.key === route.query.view) ? route.query.view as View : 'board')
const projectMap = computed<Record<number, Project>>(() => Object.fromEntries(projects.value.map(project => [project.id, project])))
const doneCount = computed(() => selectedProject.value ? selectedProject.value.done_task_count : projects.value.filter(project => project.status !== 'archived').reduce((sum, project) => sum + project.done_task_count, 0))
const today = localDate(new Date())
const todaySections = computed(() => {
  const active = tasks.value.filter(task => projectMap.value[task.project_id]?.status === 'active' && task.status !== 'done')
  const doing = active.filter(task => task.status === 'doing')
  const due = active.filter(task => task.status !== 'doing' && !!task.due_date && task.due_date <= today)
  const next = active.filter(task => task.status === 'next' && !due.some(item => item.id === task.id))
  return [
    { title: '进行中', tasks: doing },
    { title: '今天到期 / 已逾期', tasks: due },
    { title: '下一步候选', tasks: next },
  ]
})
function localDate(date: Date) { return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}` }
function projectStatusLabel(status: ProjectStatus) { return { active: '进行中', paused: '已暂停', completed: '已完成', archived: '已归档' }[status] }
function openCount(project: Project) { return project.open_task_count }
function message(reason: unknown) { return reason instanceof Error ? reason.message : '操作失败，请重试' }

async function load() {
  const sequence = ++loadSequence
  loading.value = true
  error.value = ''
  try {
    const projectList = await listProjects()
    if (sequence !== loadSequence) return
    projects.value = projectList
    const id = selectedProjectID.value
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
  setQuery({ project: key })
}
function setView(next: View) { setQuery({ view: next }) }
function openProject(project: Project) { setQuery({ project: String(project.id), view: 'board' }) }
function openNewTask(status: TaskStatus) { editingTask.value = null; newTaskStatus.value = status; taskDrawerOpen.value = true }
function openTask(task: ProjectTask) { editingTask.value = task; taskDrawerOpen.value = true }
async function saveTask(input: TaskInput) {
  await run(async () => {
    if (editingTask.value) await updateTask(editingTask.value.id, input)
    else await createTask(input)
    taskDrawerOpen.value = false
    ElMessage.success(editingTask.value ? '任务已保存' : '任务已创建')
  })
}
async function removeTask(task: ProjectTask) {
  try {
    await ElMessageBox.confirm(`删除任务“${task.title}”？删除后无法恢复。`, '确认删除', { type: 'warning' })
    await run(async () => { await deleteTask(task.id); taskDrawerOpen.value = false; ElMessage.success('任务已删除') })
  } catch { /* cancelled */ }
}
async function handleMove(payload: { task: ProjectTask; status: TaskStatus; before_id?: number; after_id?: number }) {
  if (busy.value) { await load(); return }
  busy.value = true
  try {
    await moveTask(payload.task.id, { status: payload.status, before_id: payload.before_id, after_id: payload.after_id, expected_updated_at: payload.task.updated_at })
    await load()
  } catch {
    await load()
    ElMessage.error('任务状态更新失败，已恢复原位置。')
  } finally { busy.value = false }
}
async function loadMoreDone() {
  if (busy.value || loading.value) return
  busy.value = true
  try {
    const older = await listTasks({ projectId: selectedProjectID.value ?? undefined, status: 'done', limit: donePageSize, offset: tasks.value.filter(task => task.status === 'done').length })
    tasks.value = [...tasks.value, ...older]
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
onMounted(load)
watch(projectKey, () => { void load() })
</script>

<style scoped>
.projects-toolbar { display: flex; align-items: center; gap: 22px; flex-wrap: wrap; margin-bottom: 22px; border-bottom: 1px solid var(--border-subtle); }
.projects-switcher { width: 245px; margin-bottom: 12px; }
.projects-tabs { display: flex; gap: 20px; }
.projects-tabs button { position: relative; padding: 0 0 14px; border: 0; background: none; color: var(--text-tertiary); cursor: pointer; }
.projects-tabs button.is-active { color: var(--text-primary); font-weight: 650; }
.projects-tabs button.is-active::after { position: absolute; right: 0; bottom: -1px; left: 0; height: 2px; background: var(--color-primary); content: ''; }
.projects-content { min-height: 360px; }
.projects-context { display: flex; align-items: start; gap: 12px; margin-bottom: 18px; }
.projects-context__accent { width: 4px; min-height: 32px; border-radius: 4px; }
.projects-context strong { font-size: 17px; }
.projects-context p { margin: 5px 0 0; color: var(--text-secondary); font-size: 13px; }
.projects-context .el-tag { margin-left: auto; }
.projects-muted { color: var(--text-tertiary); font-size: 13px; }
.today-view { max-width: 800px; }
.today-view h2, .project-list-view h2 { margin: 0 0 6px; font-size: 23px; }
.today-section { margin-top: 30px; }
.today-section__heading { display: flex; justify-content: space-between; padding-bottom: 10px; border-bottom: 1px solid var(--border-default); }
.today-section__heading h3 { margin: 0; font-size: 15px; }
.today-section__heading span { color: var(--text-tertiary); font-size: 13px; }
.today-task { display: flex; align-items: center; gap: 12px; width: 100%; padding: 14px 4px; border: 0; border-bottom: 1px solid var(--border-subtle); background: none; text-align: left; cursor: pointer; }
.today-task:hover { background: var(--bg-subtle); }
.today-task__marker { width: 16px; height: 16px; flex: 0 0 16px; border: 1px solid #b5c2d2; border-radius: 5px; }
.today-task__title { flex: 1; font-size: 14px; }
.today-task small { color: var(--text-tertiary); }
.project-list-view { max-width: 920px; }
.project-list-view__heading { display: flex; justify-content: space-between; align-items: center; margin-bottom: 18px; }
.project-list-card { display: flex; justify-content: space-between; gap: 18px; padding: 20px; margin-bottom: 12px; border: 1px solid var(--border-default); border-left: 4px solid var(--project-accent); border-radius: var(--radius-card); background: var(--bg-surface); }
.project-list-card__title { display: flex; align-items: center; gap: 10px; }
.project-list-card__title h3 { margin: 0; font-size: 17px; }
.project-list-card__main p { margin: 10px 0; color: var(--text-secondary); font-size: 13px; }
.project-list-card__main small { display: block; margin-top: 10px; color: var(--text-tertiary); }
.project-list-card__actions { display: flex; align-items: start; justify-content: flex-end; flex-wrap: wrap; }
.project-form-row { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
.project-form-row :deep(.el-select) { width: 100%; }
@media (max-width: 760px) { .projects-toolbar { align-items: stretch; gap: 4px; } .projects-switcher { width: 100%; } .projects-tabs { width: 100%; justify-content: space-between; } .project-list-card, .today-task { align-items: flex-start; flex-direction: column; } .today-task__marker { display: none; } .project-list-card__actions { justify-content: flex-start; } }
</style>
