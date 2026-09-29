<template>
  <el-dialog v-model="open" title="整理为项目任务" width="min(560px, calc(100vw - 28px))" class="convert-inbox-dialog">
    <el-form label-position="top" @submit.prevent="submit">
      <el-form-item label="任务标题"><el-input v-model="form.title" maxlength="240" /></el-form-item>
      <el-form-item label="所属项目"><el-select v-model="form.project_id" placeholder="选择一个进行中的项目" style="width:100%"><el-option v-for="project in projects" :key="project.id" :label="project.title" :value="project.id" /></el-select></el-form-item>
      <div class="convert-row">
        <el-form-item label="状态"><el-select v-model="form.status"><el-option label="收件箱" value="inbox"/><el-option label="下一步" value="next"/><el-option label="进行中" value="doing"/></el-select></el-form-item>
        <el-form-item label="优先级"><el-select v-model="form.priority"><el-option label="普通" value="normal"/><el-option label="高" value="high"/><el-option label="低" value="low"/></el-select></el-form-item>
        <el-form-item label="到期日"><el-date-picker v-model="form.due_date" type="date" value-format="YYYY-MM-DD" placeholder="可选" /></el-form-item>
      </div>
      <details class="convert-details"><summary>添加说明</summary><el-input v-model="form.description" type="textarea" :rows="4" maxlength="4000" show-word-limit /></details>
    </el-form>
    <p v-if="!projects.length" class="convert-empty">需要先创建一个进行中的项目，才能转为任务。</p>
    <template #footer><el-button @click="open = false">取消</el-button><el-button type="primary" :loading="saving" :disabled="!form.title.trim() || !form.project_id" @click="submit">转换为任务</el-button></template>
  </el-dialog>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { listProjects, type Project } from '@/api/projects'
import { convertInboxItem, type InboxItem } from '@/api/inbox'

const props = defineProps<{ modelValue: boolean; item: InboxItem | null }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; converted: [taskId: number, projectId: number] }>()
const open = ref(props.modelValue)
const projects = ref<Project[]>([])
const saving = ref(false)
const form = reactive({ title: '', project_id: 0, status: 'next' as 'inbox' | 'next' | 'doing', priority: 'normal' as 'high' | 'normal' | 'low', due_date: null as string | null, description: '' })
watch(() => props.modelValue, async value => {
  open.value = value
  if (!value || !props.item) return
  const first = props.item.content.split(/\r?\n/, 1)[0]?.trim() ?? ''
  form.title = [...first].slice(0, 200).join('') || props.item.content.slice(0, 200)
  form.project_id = 0; form.status = 'next'; form.priority = 'normal'; form.due_date = null; form.description = ''
  try { projects.value = (await listProjects()).filter(project => project.status === 'active') }
  catch (reason) { ElMessage.error(reason instanceof Error ? reason.message : '项目列表读取失败') }
})
watch(open, value => emit('update:modelValue', value))
async function submit() {
  if (!props.item || saving.value || !form.title.trim() || !form.project_id) return
  saving.value = true
  try {
    const result = await convertInboxItem(props.item.id, { ...form, title: form.title.trim(), description: form.description.trim(), due_date: form.due_date || null })
    open.value = false
    emit('converted', result.task.id, result.task.project_id)
    ElMessage.success('已转换为项目任务')
  } catch (reason) { ElMessage.error(reason instanceof Error ? reason.message : '转换失败，请重试') }
  finally { saving.value = false }
}
</script>

<style scoped>
.convert-row { display:grid; grid-template-columns:repeat(3,minmax(0,1fr)); gap:12px; }
.convert-row :deep(.el-select),.convert-row :deep(.el-date-editor) { width:100%; }
.convert-details { border-top:1px solid var(--border-subtle); padding-top:12px; }
.convert-details summary { margin-bottom:10px; color:var(--text-secondary); cursor:pointer; }
.convert-empty { color:var(--text-tertiary); font-size:13px; }
@media(max-width:560px){.convert-row{grid-template-columns:1fr 1fr}.convert-row :last-child{grid-column:1/-1}}
</style>
