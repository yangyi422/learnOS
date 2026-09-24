<template>
  <el-drawer
    :model-value="modelValue"
    :title="task ? '任务详情' : '新建任务'"
    direction="rtl"
    :size="drawerSize"
    destroy-on-close
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <el-form label-position="top" @submit.prevent="submit">
      <el-form-item label="任务名称">
        <el-input v-model="form.title" maxlength="200" show-word-limit placeholder="要完成什么？" />
      </el-form-item>
      <el-form-item label="所属项目">
        <el-select v-model="form.project_id" placeholder="选择项目" style="width: 100%">
          <el-option v-for="project in availableProjects" :key="project.id" :label="project.title" :value="project.id" />
        </el-select>
      </el-form-item>
      <div class="task-drawer__row">
        <el-form-item label="状态">
          <el-select v-model="form.status">
            <el-option label="待整理" value="inbox" />
            <el-option label="下一步" value="next" />
            <el-option label="进行中" value="doing" />
            <el-option label="已完成" value="done" />
          </el-select>
        </el-form-item>
        <el-form-item label="优先级">
          <el-select v-model="form.priority">
            <el-option label="高" value="high" />
            <el-option label="普通" value="normal" />
            <el-option label="低" value="low" />
          </el-select>
        </el-form-item>
      </div>
      <el-form-item label="到期日">
        <el-date-picker v-model="form.due_date" type="date" value-format="YYYY-MM-DD" placeholder="可选" style="width: 100%" clearable />
      </el-form-item>
      <el-form-item label="说明">
        <el-input v-model="form.description" type="textarea" :rows="6" maxlength="10000" show-word-limit placeholder="记录完成这项任务所需的上下文" />
      </el-form-item>
      <div v-if="task" class="task-drawer__activity">
        <strong>记录</strong>
        <p>创建：{{ formatTime(task.created_at) }}</p>
        <p>最近修改：{{ formatTime(task.updated_at) }}</p>
        <p v-if="task.completed_at">完成：{{ formatTime(task.completed_at) }}</p>
      </div>
      <div class="task-drawer__actions">
        <el-button v-if="task" text type="danger" :disabled="busy" @click="$emit('delete', task)">删除任务</el-button>
        <span />
        <el-button v-if="task && task.status !== 'done'" :disabled="busy" @click="form.status = 'done'; submit()">完成</el-button>
        <el-button v-else-if="task" :disabled="busy" @click="form.status = 'next'; submit()">重新打开</el-button>
        <el-button @click="$emit('update:modelValue', false)">取消</el-button>
        <el-button type="primary" :loading="busy" :disabled="!form.title.trim() || !form.project_id" @click="submit">{{ task ? '保存' : '创建任务' }}</el-button>
      </div>
    </el-form>
  </el-drawer>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import type { Project, ProjectTask, TaskInput, TaskPriority, TaskStatus } from '@/api/projects'

const props = defineProps<{
  modelValue: boolean
  task: ProjectTask | null
  projects: Project[]
  defaultProjectId: number | null
  defaultStatus: TaskStatus
  busy: boolean
}>()
const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'save', input: TaskInput): void
  (e: 'delete', task: ProjectTask): void
}>()
const width = ref(window.innerWidth)
const drawerSize = computed(() => width.value <= 760 ? '100%' : '460px')
const form = reactive<{ title: string; project_id: number | null; status: TaskStatus; priority: TaskPriority; due_date: string; description: string }>({
  title: '', project_id: null, status: 'inbox', priority: 'normal', due_date: '', description: '',
})
const availableProjects = computed(() => props.projects.filter(project => project.status !== 'archived' || project.id === props.task?.project_id))
watch(() => [props.modelValue, props.task, props.defaultProjectId, props.defaultStatus], () => {
  if (!props.modelValue) return
  form.title = props.task?.title ?? ''
  form.project_id = props.task?.project_id ?? props.defaultProjectId
  form.status = props.task?.status ?? props.defaultStatus
  form.priority = props.task?.priority ?? 'normal'
  form.due_date = props.task?.due_date ?? ''
  form.description = props.task?.description ?? ''
}, { immediate: true })
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
function onResize() { width.value = window.innerWidth }
onMounted(() => window.addEventListener('resize', onResize))
onUnmounted(() => window.removeEventListener('resize', onResize))
</script>

<style scoped>
.task-drawer__row { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
.task-drawer__row :deep(.el-select) { width: 100%; }
.task-drawer__activity { margin: 18px 0 26px; padding-top: 20px; border-top: 1px solid var(--border-subtle); color: var(--text-tertiary); font-size: 12px; }
.task-drawer__activity strong { display: block; margin-bottom: 12px; color: var(--text-secondary); }
.task-drawer__activity p { margin: 0 0 6px; }
.task-drawer__actions { display: flex; align-items: center; gap: 8px; padding-top: 18px; border-top: 1px solid var(--border-subtle); }
.task-drawer__actions span { flex: 1; }
</style>
