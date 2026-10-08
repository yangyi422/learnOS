<template>
  <el-dialog :model-value="modelValue" :title="event && !editing ? '生活事件' : event ? '编辑事件' : '记录生活事件'" width="min(640px, calc(100vw - 28px))" class="life-dialog" :before-close="close" :close-on-click-modal="false" @opened="focusTitle">
    <el-alert v-if="error" :title="error" type="error" :closable="false" show-icon />
    <div v-if="event && !editing" class="life-reading">
      <div class="life-metadata"><span>{{ event.occurred_on }}</span><el-tag v-if="event.milestone" size="small" effect="plain">里程碑</el-tag><span v-if="event.primary_domain">{{ domainLabel(event.primary_domain) }}</span><span v-if="event.secondary_domain">{{ domainLabel(event.secondary_domain) }}</span></div>
      <h2>{{ event.title }}</h2><LinkedText v-if="event.description" :content="event.description" /><p v-else class="life-muted">没有补充说明。</p>
      <p v-if="event.goal_id" class="life-muted">长期目标：<RouterLink :to="{ path: '/life', query: { goal: event.goal_id } }" @click="emit('update:modelValue', false)">{{ goals.find(g => g.id === event?.goal_id)?.title || '查看关联目标' }}</RouterLink></p>
      <ExternalLink v-if="event.external_url" :url="event.external_url" :name="event.link_name" />
      <p v-if="event.source_id" class="life-source">来自{{ sourceLabel }}：{{ event.source_title }} <RouterLink v-if="event.source_available && sourcePath" :to="sourcePath" @click="emit('update:modelValue', false)">查看来源</RouterLink><span v-else>（来源已删除或不可访问，档案仍保留）</span><span v-if="event.source_status === 'archived'"> · 来源已归档</span></p>
      <small class="life-muted">记录于 {{ displayTime(event.created_at) }}</small>
    </div>
    <el-form v-else label-position="top" class="life-editor" @submit.prevent="save">
      <el-form-item label="事件名称" required><el-input ref="titleInput" v-model="form.title" maxlength="200" placeholder="留下值得记住的一件事" /></el-form-item>
      <el-form-item label="发生日期" required><el-date-picker v-model="form.occurred_on" type="date" value-format="YYYY-MM-DD" format="YYYY-MM-DD" :clearable="false" /></el-form-item>
      <p v-if="source" class="life-muted">来源没有独立的完成日期，请确认发生日期。收录不会修改来源。</p>
      <el-form-item label="简短描述（可选）"><el-input v-model="form.description" type="textarea" :autosize="{ minRows: 3, maxRows: 8 }" maxlength="10000" placeholder="发生了什么，当时有什么感受？" /></el-form-item>
      <el-checkbox v-model="form.milestone">这是一个重要里程碑</el-checkbox>
      <details class="life-options" :open="Boolean(form.primary_domain || form.goal_id || form.external_url)"><summary>领域、目标与外部资料（可选）</summary>
        <div class="life-form-row"><el-form-item label="主领域"><el-select v-model="form.primary_domain" clearable placeholder="不分类" @change="form.secondary_domain = ''"><el-option v-for="d in lifeDomains" :key="d[0]" :value="d[0]" :label="d[1]" /></el-select></el-form-item><el-form-item label="次领域"><el-select v-model="form.secondary_domain" clearable :disabled="!form.primary_domain" placeholder="可选"><el-option v-for="d in lifeDomains.filter(d => d[0] !== form.primary_domain)" :key="d[0]" :value="d[0]" :label="d[1]" /></el-select></el-form-item></div>
        <el-form-item label="关联长期目标"><el-select v-model="form.goal_id" clearable placeholder="不关联"><el-option v-for="g in goals" :key="g.id" :value="g.id" :label="g.title" /></el-select></el-form-item>
        <el-form-item label="外部资料链接"><el-input v-model="form.external_url" placeholder="https:// 或 obsidian://open?..." maxlength="2048" /></el-form-item>
        <el-form-item v-if="form.external_url" label="链接显示名称"><el-input v-model="form.link_name" maxlength="160" placeholder="可选" /></el-form-item>
      </details>
    </el-form>
    <template #footer><div class="life-footer">
      <template v-if="event && !editing"><el-button text :loading="saving" @click="remove">删除</el-button><span class="life-spacer" /><el-button @click="close()">关闭</el-button><el-button type="primary" @click="beginEdit">编辑</el-button></template>
      <template v-else><span class="life-spacer" /><el-button :disabled="saving" @click="cancel">取消</el-button><el-button type="primary" :loading="saving" :disabled="!form.title.trim() || !form.occurred_on" @click="save">保存</el-button></template>
    </div></template>
  </el-dialog>
</template>
<script setup lang="ts">
import { computed, nextTick, reactive, ref, watch } from 'vue'
import { onBeforeRouteLeave, onBeforeRouteUpdate } from 'vue-router'
import { ElMessageBox, ElMessage } from 'element-plus'
import LinkedText from '@/components/LinkedText.vue'
import ExternalLink from '@/components/ExternalLink.vue'
import type { InboxItem } from '@/api/inbox'
import { localCalendarDate } from '@/utils/projectTasks'
import { safeExternalLink } from '@/utils/externalLinks'
import { deleteLifeEvent, domainLabel, lifeDomains, lifeRequestKey, listLifeGoals, saveLifeEvent, type LifeEvent, type LifeEventInput, type LifeGoal, type LifeSource } from '@/api/life'
const props = defineProps<{ modelValue: boolean; event?: LifeEvent | null; inboxItem?: InboxItem | null; source?: LifeSource | null; goalId?: number | null }>()
const emit = defineEmits<{ 'update:modelValue': [boolean]; saved: [LifeEvent]; deleted: [number] }>()
const form = reactive<LifeEventInput>({ title: '', occurred_on: '', description: '', primary_domain: '', secondary_domain: '', milestone: false, goal_id: null, external_url: '', link_name: '' })
const editing = ref(false), saving = ref(false), error = ref(''), goals = ref<LifeGoal[]>([]), titleInput = ref<{ focus: () => void }>()
let snapshot = '', loadSequence = 0
const sourceLabel = computed(() => ({ project: '项目', course: '课程', inbox: '收集箱' })[props.event?.source_type as 'project' | 'course' | 'inbox'] ?? '来源')
const sourcePath = computed(() => props.event?.source_type === 'project' ? `/projects?project=${props.event.source_id}&view=board` : props.event?.source_type === 'course' ? `/courses/${props.event.source_id}/archive` : props.event?.source_type === 'inbox' ? '/inbox' : '')
function reset() { Object.assign(form, { creation_key: lifeRequestKey(), title: '', occurred_on: localCalendarDate(new Date()), description: props.inboxItem?.content ?? '', primary_domain: '', secondary_domain: '', milestone: Boolean(props.source), goal_id: props.goalId ?? null, external_url: props.inboxItem?.source_url ?? '', link_name: '', source_type: props.source?.type ?? (props.inboxItem ? 'inbox' : ''), source_id: props.source?.id ?? props.inboxItem?.id ?? null }); if (props.source) form.title = props.source.title; if (props.event) Object.assign(form, props.event); snapshot = JSON.stringify(form) }
watch(() => props.modelValue, async open => { const sequence = ++loadSequence; if (!open) return; editing.value = !props.event; error.value = ''; reset(); try { const result = await listLifeGoals(); if (sequence === loadSequence) goals.value = result } catch (reason) { if (sequence === loadSequence) error.value = message(reason) } })
watch(() => props.event, () => { if (props.modelValue && !editing.value) reset() })
function message(reason: unknown) { return reason instanceof Error ? reason.message : '操作失败，请重试。' }
function displayTime(value: string) { return new Date(value).toLocaleString('zh-CN') }
function focusTitle() { if (editing.value) titleInput.value?.focus() }
function beginEdit() { editing.value = true; error.value = ''; reset(); void nextTick(focusTitle) }
async function allowDiscard() { if (saving.value) return false; if (!editing.value || JSON.stringify(form) === snapshot) return true; try { await ElMessageBox.confirm('放弃尚未保存的事件内容？', '未保存的修改', { confirmButtonText: '放弃修改', cancelButtonText: '继续编辑' }); return true } catch { return false } }
async function close(done?: () => void) { if (!(await allowDiscard())) return; done?.(); emit('update:modelValue', false) }
async function cancel() { if (!(await allowDiscard())) return; if (props.event) { editing.value = false; reset(); error.value = '' } else emit('update:modelValue', false) }
onBeforeRouteLeave(async () => !props.modelValue || await allowDiscard())
onBeforeRouteUpdate(async () => !props.modelValue || await allowDiscard())
async function save() {
  if (saving.value) return
  if (form.external_url.trim() && !safeExternalLink(form.external_url.trim())) { error.value = '链接仅支持 http、https 和有效的 Obsidian open URI。'; return }
  saving.value = true; error.value = ''
  try { const saved = await saveLifeEvent({ ...form }, props.event?.id, props.inboxItem?.id); snapshot = JSON.stringify(form); editing.value = false; emit('saved', saved); if (!props.event) emit('update:modelValue', false); ElMessage.success('已保存生活事件') }
  catch (reason) { error.value = message(reason) } finally { saving.value = false }
}
async function remove() { if (!props.event || saving.value) return; try { await ElMessageBox.confirm('删除这条生活事件？原始项目、课程或 Inbox 内容不会删除。', '确认删除', { confirmButtonText: '删除事件', cancelButtonText: '取消' }); saving.value = true; await deleteLifeEvent(props.event.id); emit('deleted', props.event.id); emit('update:modelValue', false) } catch (reason) { if (reason !== 'cancel' && reason !== 'close') error.value = message(reason) } finally { saving.value = false } }
</script>
