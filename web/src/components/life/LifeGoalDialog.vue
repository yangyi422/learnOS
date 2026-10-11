<template>
  <el-dialog :model-value="modelValue" :title="detail && !editing ? '长期目标' : detail ? '编辑长期目标' : '写下长期目标'" width="min(680px, calc(100vw - 28px))" class="life-dialog" align-center :before-close="close" :close-on-click-modal="false" @opened="focusTitle" @closed="restoreFocus">
    <el-alert v-if="error" :title="error" type="error" :closable="false" show-icon />
    <div v-if="detail && !editing" class="life-reading">
      <div class="life-metadata"><el-tag size="small" effect="plain">{{ goalStatusLabel(detail.goal.status) }}</el-tag><span v-if="detail.goal.domain">{{ domainLabel(detail.goal.domain) }}</span><span>创建于 {{ new Date(detail.goal.created_at).toLocaleDateString('zh-CN') }}</span></div>
      <h2>{{ detail.goal.title }}</h2>
      <div class="life-goal-overview">
        <section><h3>为什么重要</h3><LinkedText v-if="detail.goal.why" :content="detail.goal.why" /><p v-else class="life-muted">还没有写下这个目标对你的意义。</p></section>
        <section><h3>当前判断</h3><LinkedText v-if="detail.goal.current_note" :content="detail.goal.current_note" /><p v-else class="life-muted">可以记录此刻的看法，之后继续更新。</p></section>
      </div>
      <p v-if="lifeUpdatedLabel(detail.goal.updated_at)" class="life-muted">最近更新 {{ lifeUpdatedLabel(detail.goal.updated_at) }}</p>
      <p v-if="detail.goal.project_id"><RouterLink v-if="detail.goal.project_available" :to="`/projects?project=${detail.goal.project_id}&view=board`" @click="emit('update:modelValue', false)">关联项目：{{ detail.goal.project_title }}</RouterLink><span v-else class="life-muted">关联项目已删除或不可访问，目标仍保留。</span></p>
      <ExternalLink v-if="detail.goal.external_url" :url="detail.goal.external_url" :name="detail.goal.link_name" />
      <section class="life-history"><div class="life-metadata"><h3 class="life-spacer">变化历程</h3><el-button v-if="!addingEntry" text :disabled="financialBusy || financialDirty" @click="startEntry()">+ 记录进展或决定</el-button></div>
        <el-form v-if="addingEntry" label-position="top" @submit.prevent="saveEntry">
          <el-form-item label="记录日期"><el-date-picker v-model="entry.occurred_on" type="date" value-format="YYYY-MM-DD" :clearable="false" /></el-form-item>
          <el-form-item label="进展或决定" required><el-input ref="entryInput" v-model="entry.content" type="textarea" :rows="3" maxlength="10000" /></el-form-item>
          <el-form-item label="原因或心得（可选）"><el-input v-model="entry.reason" type="textarea" :rows="2" maxlength="4000" /></el-form-item>
          <el-button :disabled="saving" @click="cancelEntry">取消记录</el-button><el-button type="primary" :loading="saving" :disabled="(!entry.content.trim() && !editingEntry?.to_status) || !entry.occurred_on" @click="saveEntry">保存阶段记录</el-button>
        </el-form>
        <p v-if="!detail.entries.length && !addingEntry" class="life-muted">目标会随生活变化。在重要节点留下一条当时的想法。</p>
        <article v-for="item in sortedEntries" :key="item.id" class="life-history-entry">
          <div class="life-metadata"><time :datetime="item.occurred_on">{{ item.occurred_on }}</time><span v-if="item.to_status">{{ goalStatusLabel(item.from_status) }} → {{ goalStatusLabel(item.to_status) }}</span><span class="life-spacer" /><el-button text size="small" :disabled="addingEntry || saving || financialBusy || financialDirty" :aria-label="`编辑阶段记录：${item.occurred_on}`" @click="startEntry(item)">编辑</el-button><el-button text size="small" :disabled="addingEntry || saving || financialBusy || financialDirty" :aria-label="`删除阶段记录：${item.occurred_on}`" @click="removeEntry(item)">删除</el-button></div>
          <LinkedText v-if="item.content" :content="item.content" /><p v-if="item.reason" class="life-muted">{{ item.reason }}</p>
        </article>
      </section>
      <GoalFinancialReferences ref="financial" :goal-id="detail.goal.id" :disabled="addingEntry || saving" @state="financialState" @asset="openAsset" />
      <section class="life-history"><h3>相关经历与里程碑</h3><p v-if="!detail.events.length" class="life-muted">记录生活事件时，可以选择关联这个目标。</p><button v-for="event in detail.events" :key="event.id" type="button" class="life-event-card" @click="openRelatedEvent(event.id)"><div class="life-metadata">{{ event.occurred_on }} <span v-if="event.milestone">里程碑</span></div><h3>{{ event.title }}</h3></button></section>
    </div>
    <el-form v-else label-position="top" @submit.prevent="save">
      <el-form-item label="目标名称" required><el-input ref="titleInput" v-model="form.title" maxlength="200" placeholder="一件希望长期实现的事情" /></el-form-item>
      <el-form-item label="为什么重要（可选）"><el-input v-model="form.why" type="textarea" :rows="3" maxlength="10000" /></el-form-item>
      <el-form-item label="当前判断或进展（可选）"><el-input v-model="form.current_note" type="textarea" :rows="3" maxlength="10000" /></el-form-item>
      <el-checkbox v-if="detail && form.current_note !== detail.goal.current_note" v-model="form.record_change">将当前判断保存为阶段记录</el-checkbox>
      <el-form-item label="当前状态"><el-select v-model="form.status"><el-option v-for="s in goalStatuses" :key="s[0]" :label="s[1]" :value="s[0]" /></el-select></el-form-item>
      <template v-if="detail && (form.status !== detail.goal.status || form.record_change)"><div class="life-form-row"><el-form-item label="状态变化日期"><el-date-picker v-model="form.status_date" type="date" value-format="YYYY-MM-DD" :clearable="false" /></el-form-item><el-form-item label="变化原因（可选）"><el-input v-model="form.status_reason" maxlength="4000" /></el-form-item></div><p class="life-muted">状态变化或主动保存的判断会留在历史中，普通标题编辑不会产生记录。</p></template>
      <details class="life-options" :open="Boolean(form.domain || form.project_id || form.external_url)"><summary>领域、项目与外部资料（可选）</summary>
        <el-form-item label="生活领域"><el-select v-model="form.domain" clearable placeholder="不分类"><el-option v-for="d in lifeDomains" :key="d[0]" :value="d[0]" :label="d[1]" /></el-select></el-form-item>
        <el-form-item label="关联项目"><el-select v-model="form.project_id" clearable placeholder="不关联"><el-option v-for="p in projects" :key="p.id" :value="p.id" :label="`${p.title}${p.status === 'archived' ? '（已归档）' : ''}`" /></el-select></el-form-item>
        <el-form-item label="外部资料链接"><el-input v-model="form.external_url" maxlength="2048" placeholder="https:// 或 obsidian://open?..." /></el-form-item><el-form-item v-if="form.external_url" label="链接显示名称"><el-input v-model="form.link_name" maxlength="160" /></el-form-item>
      </details>
    </el-form>
    <template #footer><div class="life-footer"><span class="life-spacer" /><template v-if="detail && !editing"><el-button :disabled="saving" @click="close()">关闭</el-button><el-button text :disabled="addingEntry || saving || financialBusy || financialDirty" @click="beginEdit">编辑目标</el-button></template><template v-else><el-button :disabled="saving" @click="cancel">取消</el-button><el-button type="primary" :loading="saving" :disabled="!form.title.trim()" @click="save">保存目标</el-button></template></div></template>
  </el-dialog>
</template>
<script setup lang="ts">
import { computed, nextTick, reactive, ref, watch } from 'vue'
import { onBeforeRouteLeave, onBeforeRouteUpdate } from 'vue-router'
import { ElMessageBox, ElMessage } from 'element-plus'
import GoalFinancialReferences from './GoalFinancialReferences.vue'
import LinkedText from '@/components/LinkedText.vue'
import ExternalLink from '@/components/ExternalLink.vue'
import { listProjects, type Project } from '@/api/projects'
import { addLifeEntry, editLifeEntry, deleteLifeEntry, lifeUpdatedLabel, domainLabel, goalStatuses, goalStatusLabel, lifeDomains, lifeRequestKey, saveLifeGoal, getLifeGoal, type GoalDetail, type LifeGoalInput, type LifeGoal, type LifeEntry } from '@/api/life'
import { safeExternalLink } from '@/utils/externalLinks'
import { localCalendarDate } from '@/utils/projectTasks'
const props = defineProps<{ modelValue: boolean; detail?: GoalDetail | null }>()
const emit = defineEmits<{ 'update:modelValue': [boolean]; saved: [LifeGoal]; detail: [GoalDetail]; event: [number]; asset: [number] }>()
const financial = ref<InstanceType<typeof GoalFinancialReferences>>()
const financialDirty = ref(false), financialBusy = ref(false)
function financialState(dirty: boolean, busy: boolean) { financialDirty.value = dirty; financialBusy.value = busy }
const editing = ref(false), addingEntry = ref(false), saving = ref(false), error = ref(''), projects = ref<Project[]>([]), titleInput = ref<{ focus: () => void }>()
const form = reactive<LifeGoalInput>({ title: '', why: '', current_note: '', status: 'considering', domain: '', project_id: null, external_url: '', link_name: '', status_date: '', status_reason: '' })
const entry = reactive({ creation_key: '', occurred_on: '', content: '', reason: '' })
const editingEntry = ref<LifeEntry | null>(null), entryInput = ref<{ focus: () => void }>()
const sortedEntries = computed(() => [...(props.detail?.entries ?? [])].sort((a, b) => b.occurred_on.localeCompare(a.occurred_on) || b.id - a.id))
let snapshot = '', entrySnapshot = '', sequence = 0
let returnFocus: HTMLElement | null = null
function restoreFocus() { if (returnFocus?.isConnected) returnFocus.focus(); else document.querySelector<HTMLElement>('.life-tabs button[aria-selected=true]')?.focus() }
const message = (reason: unknown) => reason instanceof Error ? reason.message : '操作失败，请重试。'
function reset() { Object.assign(form, { record_change: false, creation_key: lifeRequestKey(), title: '', why: '', current_note: '', status: 'considering', domain: '', project_id: null, external_url: '', link_name: '', status_date: localCalendarDate(new Date()), status_reason: '' }); if (props.detail) Object.assign(form, props.detail.goal, { status_date: localCalendarDate(new Date()), status_reason: '' }); snapshot = JSON.stringify(form) }
watch(() => props.modelValue, async open => { const token = ++sequence; if (!open) return; financialDirty.value = false; financialBusy.value = false; returnFocus = document.activeElement instanceof HTMLElement && document.activeElement !== document.body ? document.activeElement : null; editingEntry.value = null; editing.value = !props.detail; addingEntry.value = false; error.value = ''; reset(); try { const result = await listProjects(); if (token === sequence) projects.value = result } catch (reason) { if (token === sequence) error.value = message(reason) } })
function focusTitle() { if (editing.value) titleInput.value?.focus() }
async function beginEdit() { if (!(await allowDiscard())) return; financialDirty.value = false; editing.value = true; error.value = ''; reset(); void nextTick(focusTitle) }
async function confirmDiscard() { try { await ElMessageBox.confirm('放弃尚未保存的内容？', '未保存的修改', { confirmButtonText: '放弃修改', cancelButtonText: '继续编辑' }); return true } catch { return false } }
async function allowDiscard() { if (saving.value || financialBusy.value) return false; if (financialDirty.value && !(await financial.value?.allowDiscard())) return false; if ((editing.value && JSON.stringify(form) !== snapshot) || (addingEntry.value && JSON.stringify(entry) !== entrySnapshot)) return confirmDiscard(); return true }
async function close(done?: () => void) { if (!(await allowDiscard())) return; done?.(); emit('update:modelValue', false) }
async function cancel() { if (!(await allowDiscard())) return; if (props.detail) { editing.value = false; reset(); error.value = '' } else emit('update:modelValue', false) }
onBeforeRouteLeave(async () => !props.modelValue || await allowDiscard())
onBeforeRouteUpdate(async () => !props.modelValue || await allowDiscard())
async function save() { if (saving.value) return; if (form.external_url.trim() && !safeExternalLink(form.external_url.trim())) { error.value = '链接仅支持 http、https 和有效的 Obsidian open URI。'; return }; saving.value = true; error.value = ''; try { const saved = await saveLifeGoal({ ...form }, props.detail?.goal.id); snapshot = JSON.stringify(form); editing.value = false; emit('saved', saved); if (!props.detail) emit('update:modelValue', false); ElMessage.success('已保存目标') } catch (reason) { error.value = message(reason) } finally { saving.value = false } }
function startEntry(item?: LifeEntry) { editingEntry.value = item ?? null; Object.assign(entry, { creation_key: lifeRequestKey(), occurred_on: localCalendarDate(new Date()), content: '', reason: '' }, item ? { occurred_on: item.occurred_on, content: item.content, reason: item.reason } : {}); entrySnapshot = JSON.stringify(entry); addingEntry.value = true; error.value = ''; void nextTick(() => entryInput.value?.focus()) }
async function cancelEntry() { if (saving.value) return; if (JSON.stringify(entry) !== entrySnapshot && !(await confirmDiscard())) return; addingEntry.value = false; error.value = '' }
async function saveEntry() { if (!props.detail || saving.value) return; saving.value = true; error.value = ''; try { const saved = editingEntry.value ? await editLifeEntry(props.detail.goal.id, editingEntry.value.id, { ...entry }) : await addLifeEntry(props.detail.goal.id, { ...entry }); emit('detail', { ...props.detail, entries: [...props.detail.entries.filter(e => e.id !== saved.id), saved] }); entrySnapshot = JSON.stringify(entry); addingEntry.value = false; ElMessage.success('已保存阶段记录'); await refreshDetail() } catch (reason) { error.value = message(reason) } finally { saving.value = false } }
async function refreshDetail() { if (!props.detail) return; try { emit('detail', await getLifeGoal(props.detail.goal.id)) } catch (reason) { error.value = `操作已保存，但详情刷新失败：${message(reason)}` } }
async function removeEntry(item: LifeEntry) {
  if (!props.detail || saving.value || addingEntry.value) return
  try {
    await ElMessageBox.confirm(item.to_status ? '删除这条变化记录？目标当前状态不会回退。' : '删除这条阶段记录？删除后无法恢复。', '确认删除阶段记录', { confirmButtonText: '删除记录', cancelButtonText: '取消' })
    saving.value = true; error.value = ''
    await deleteLifeEntry(props.detail.goal.id, item.id)
    emit('detail', { ...props.detail, entries: props.detail.entries.filter(e => e.id !== item.id) })
    await refreshDetail()
  } catch (reason) { if (reason !== 'cancel' && reason !== 'close') error.value = message(reason) } finally { saving.value = false }
}
async function openAsset(id: number) { if (await allowDiscard()) { emit('update:modelValue', false); emit('asset', id) } }
async function openRelatedEvent(id: number) { if (await allowDiscard()) emit('event', id) }
</script>
