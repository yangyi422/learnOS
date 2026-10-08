<template>
  <el-dialog :model-value="modelValue" :title="record ? '记录详情' : inboxItem ? '保存为记录' : '新增记录'" width="min(620px, calc(100vw - 28px))" class="record-dialog" :before-close="close" @update:model-value="$emit('update:modelValue', $event)">
    <div class="record-dialog__body">
      <template v-if="record && !editing">
        <p class="record-meta">{{ projectName }} · {{ formatTime(record.created_at) }}{{ record.archived_at ? ' · 已归档' : '' }}</p>
        <LinkedText :content="record.content" />
        <p v-if="record.external_url"><ExternalLink :url="record.external_url" :name="record.link_name" /></p>
        <details class="record-times"><summary>记录信息</summary><p>最近修改：{{ formatTime(record.updated_at) }}</p><p v-if="record.source_inbox_id">来自 Inbox #{{ record.source_inbox_id }}</p></details>
      </template>
      <el-form v-else label-position="top" :disabled="saving" @submit.prevent="save">
        <el-form-item label="内容"><el-input v-model="form.content" type="textarea" :rows="6" maxlength="10000" aria-label="记录内容" /></el-form-item>
        <details class="record-options" :open="!!form.project_id || !!form.external_url">
          <summary>项目与外部链接（可选）</summary>
          <el-form-item label="关联项目"><el-select v-model="form.project_id" clearable placeholder="不关联项目" style="width:100%"><el-option v-for="project in projects" :key="project.id" :label="`${project.title}${project.status === 'archived' ? '（已归档）' : ''}`" :value="project.id" /></el-select></el-form-item>
          <el-form-item label="外部链接"><el-input v-model="form.external_url" aria-label="外部链接" placeholder="https://… 或 obsidian://open?vault=…&file=…" maxlength="4096" /></el-form-item>
          <el-form-item v-if="form.external_url" label="链接显示名称"><el-input v-model="form.link_name" aria-label="链接显示名称" placeholder="可选" maxlength="160" /></el-form-item>
        </details>
      </el-form>
      <p v-if="error" class="record-error" role="alert">{{ error }}</p>
    </div>
    <template #footer>
      <div v-if="record && !editing" class="record-actions">
        <el-button text type="danger" :disabled="saving" @click="remove">删除</el-button>
        <div><el-button :disabled="saving" @click="archive">{{ record.archived_at ? '恢复记录' : '归档记录' }}</el-button><el-button type="primary" :disabled="saving" @click="editing = true">编辑</el-button></div>
      </div>
      <template v-else><el-button :disabled="saving" @click="cancel">取消</el-button><el-button type="primary" :loading="saving" :disabled="!form.content.trim()" @click="save">{{ inboxItem ? '保存为记录' : '保存' }}</el-button></template>
    </template>
  </el-dialog>
</template>
<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listProjects, type Project } from '@/api/projects'
import type { InboxItem } from '@/api/inbox'
import { archiveRecord, convertToRecord, deleteRecord, saveRecord, type LightweightRecord, type RecordInput } from '@/api/records'
import { safeExternalLink } from '@/utils/externalLinks'
import { rememberProject } from '@/utils/recentProject'
import LinkedText from '@/components/LinkedText.vue'
import ExternalLink from '@/components/ExternalLink.vue'
const props = defineProps<{ modelValue: boolean; record?: LightweightRecord | null; inboxItem?: InboxItem | null; projectId?: number | null }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; saved: [record: LightweightRecord]; changed: [] }>()
const projects = ref<Project[]>([]); const editing = ref(false); const saving = ref(false); const error = ref('')
const form = reactive<RecordInput>({ content: '', project_id: null, external_url: '', link_name: '' })
const creationKey = ref('')
const initial = ref(''); let sequence = 0
const projectName = computed(() => projects.value.find(project => project.id === props.record?.project_id)?.title || '未关联项目')
function reset() {
  Object.assign(form, { content: props.record?.content ?? props.inboxItem?.content ?? '', project_id: props.record ? props.record.project_id : props.projectId ?? null, external_url: props.record?.external_url ?? props.inboxItem?.source_url ?? '', link_name: props.record?.link_name ?? '' })
  initial.value = JSON.stringify(form)
}
watch(() => [props.modelValue, props.record, props.inboxItem], async () => {
  const current = ++sequence
  if (!props.modelValue) return
  error.value = ''; editing.value = !props.record; creationKey.value = ''; reset()
  try { const result = await listProjects(); if (current === sequence) projects.value = result }
  catch { if (current === sequence) error.value = '项目列表读取失败，可重开弹层重试。' }
}, { immediate: true })
async function discard() {
  if (saving.value) return false
  if (!editing.value || JSON.stringify(form) === initial.value) return true
  try { await ElMessageBox.confirm('当前记录修改尚未保存，确定放弃吗？', '放弃未保存的修改', { confirmButtonText: '放弃修改', cancelButtonText: '继续编辑' }); return true } catch { return false }
}
async function close(done: () => void) { if (await discard()) done() }
async function cancel() { if (!await discard()) return; reset(); error.value = ''; if (props.record) editing.value = false; else emit('update:modelValue', false) }
async function save() {
  if (saving.value || !form.content.trim()) return
  if (form.external_url.trim() && !safeExternalLink(form.external_url.trim())) { error.value = '链接须为 http、https，或包含 vault 和 file 的 Obsidian open URI。'; return }
  saving.value = true; error.value = ''
  try {
    if (!props.record && !props.inboxItem) creationKey.value ||= crypto.randomUUID?.() || `record-${Date.now()}-${Math.random().toString(36).slice(2)}`
    const input = { ...form, creation_key: creationKey.value || undefined, content: form.content.trim(), external_url: form.external_url.trim(), project_id: form.project_id || null }
    const saved = props.inboxItem ? (await convertToRecord(props.inboxItem.id, input)).record : await saveRecord(input, props.record?.id)
    initial.value = JSON.stringify(form)
    if (saved.project_id) rememberProject(saved.project_id)
    emit('saved', saved); editing.value = false
    if (!props.record) emit('update:modelValue', false)
    ElMessage.success('记录已保存')
  } catch (reason) { error.value = reason instanceof Error ? reason.message : '保存失败，请重试。' }
  finally { saving.value = false }
}
async function archive() {
  if (!props.record || saving.value) return
  saving.value = true; error.value = ''
  try { await archiveRecord(props.record.id, !props.record.archived_at); emit('changed'); emit('update:modelValue', false) }
  catch (reason) { error.value = reason instanceof Error ? reason.message : '归档失败，请重试。' }
  finally { saving.value = false }
}
async function remove() {
  if (!props.record || saving.value) return
  try { await ElMessageBox.confirm('删除这条记录？原始 Inbox 内容和项目不会被删除。', '删除记录', { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' }) } catch { return }
  saving.value = true; error.value = ''
  try { await deleteRecord(props.record.id); emit('changed'); emit('update:modelValue', false) }
  catch (reason) { error.value = reason instanceof Error ? reason.message : '删除失败，请重试。' }
  finally { saving.value = false }
}
function formatTime(value: string) { return new Date(value).toLocaleString('zh-CN') }
</script>
<style scoped>
.record-dialog__body { max-height: 60dvh; overflow-y: auto; padding-right: 4px; }
.record-meta, .record-times { color: var(--text-secondary); font-size: 12px; }
.record-meta { margin: 0 0 18px; }
.record-times { margin-top: 24px; }
.record-options summary, .record-times summary { cursor: pointer; margin-bottom: 14px; }
.record-actions { display: flex; justify-content: space-between; gap: 8px; flex-wrap: wrap; }
.record-error { color: var(--color-danger); font-size: 13px; }
</style>
