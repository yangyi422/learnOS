<template>
  <section class="inbox-page page-stack">
    <PageHeader title="收集箱" description="先记下来，稍后再整理。">
      <template #actions><el-button type="primary" @click="captureOpen = true">快速记录</el-button></template>
    </PageHeader>
    <InboxCaptureForm @created="captured" />
    <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />
    <nav class="inbox-tabs" aria-label="收集箱筛选">
      <button v-for="tab in tabs" :key="tab.key" type="button" :class="{ active: status === tab.key }" :aria-current="status === tab.key ? 'page' : undefined" @click="select(tab.key)">{{ tab.label }}<span>{{ tab.key === 'records' ? '' : counts[tab.key] ?? 0 }}</span></button>
    </nav>
    <RecordsPanel v-if="status === 'records'" ref="recordsPanel" />
    <div v-else v-loading="loading" class="inbox-list">
      <section v-for="group in groupedItems" :key="group.key" class="inbox-day-group">
      <h2>{{ group.label }}</h2>
      <article v-for="item in group.items" :key="item.id" class="inbox-item">
        <div class="inbox-item__main">
          <button type="button" class="inbox-item__content" @click="viewItem(item)">{{ item.content }}</button>
          <div class="inbox-item__meta"><time>{{ formatDate(item.created_at) }}</time><span v-if="item.source_type === 'url'">链接</span><ExternalLink v-if="item.source_url" :url="item.source_url" /><span v-if="item.status === 'processed' && item.processed_to_type === 'task'">已转为任务</span><span v-if="item.processed_to_type === 'record'">已保存为记录</span></div>
        </div>
        <div class="inbox-item__actions">
          <el-button v-if="item.status === 'inbox'" text @click="openConvert(item)">转为任务</el-button>
          <el-button v-if="item.status === 'inbox'" text @click="openSaveRecord(item)">保存为记录</el-button>

          <el-button v-if="item.status === 'processed' && item.processed_to_type === 'task' && item.processed_to_id" text @click="openTask(item)">查看任务</el-button>
          <el-button v-if="item.processed_to_type === 'record' && item.processed_to_id" text @click="openRecord(item)">查看记录</el-button>
        </div>
      </article>
      </section>
      <el-empty v-if="!loading && items.length === 0" :description="emptyLabel" />
    </div>
    <QuickCaptureDialog v-model="captureOpen" @created="captureCompleted" />
    <RecordDialog v-model="recordOpen" :inbox-item="selectedItem" @saved="recordSaved" />
    <el-dialog v-model="detailOpen" title="收集内容" width="min(620px, calc(100vw - 28px))">
      <div class="inbox-detail"><LinkedText v-if="selectedItem" :content="selectedItem.content" /></div>
      <template #footer><div class="inbox-detail-actions"><el-button v-if="selectedItem?.status === 'inbox'" @click="detailOpen = false; openConvert(selectedItem)">转为任务</el-button><el-button v-if="selectedItem?.status === 'inbox'" @click="detailOpen = false; openSaveRecord(selectedItem)">保存为记录</el-button><el-button v-if="selectedItem?.status === 'inbox'" @click="detailOpen = false; openEdit(selectedItem)">编辑</el-button><el-button v-if="selectedItem && selectedItem.status !== 'archived'" @click="archive(selectedItem)">归档</el-button><el-button v-if="selectedItem" text type="danger" @click="remove(selectedItem)">删除</el-button></div></template>
    </el-dialog>
    <ConvertInboxDialog v-model="convertOpen" :item="selectedItem" @converted="handleConverted" />
    <el-dialog v-model="editOpen" title="编辑收集内容" width="min(520px, calc(100vw - 28px))">
      <el-input v-model="editContent" type="textarea" :rows="5" maxlength="10000" show-word-limit />
      <template #footer><el-button @click="editOpen = false">取消</el-button><el-button type="primary" :loading="saving" :disabled="!editContent.trim()" @click="saveEdit">保存</el-button></template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import PageHeader from '@/components/PageHeader.vue'
import QuickCaptureDialog from '@/components/QuickCaptureDialog.vue'
import InboxCaptureForm from '@/components/InboxCaptureForm.vue'
import RecordsPanel from '@/components/RecordsPanel.vue'
import RecordDialog from '@/components/RecordDialog.vue'
import ExternalLink from '@/components/ExternalLink.vue'
import LinkedText from '@/components/LinkedText.vue'
import ConvertInboxDialog from '@/components/ConvertInboxDialog.vue'
import { archiveInboxItem, deleteInboxItem, listInbox, updateInboxItem, type InboxItem, type InboxStatus } from '@/api/inbox'

type Tab = InboxStatus | 'records'
const tabs: { key: Tab; label: string }[] = [{ key: 'inbox', label: '待整理' }, { key: 'processed', label: '已处理' }, { key: 'archived', label: '已归档' }, { key: 'records', label: '记录' }]
const router = useRouter()
const status = ref<Tab>('inbox')
const recordOpen = ref(false); const detailOpen = ref(false)
const recordsPanel = ref<InstanceType<typeof RecordsPanel> | null>(null)
let loadSequence = 0
const pending = new Set<number>()
const items = ref<InboxItem[]>([])
const counts = ref<Record<InboxStatus, number>>({ inbox: 0, processed: 0, archived: 0 })
const loading = ref(false); const saving = ref(false); const error = ref('')
const captureOpen = ref(false); const convertOpen = ref(false); const editOpen = ref(false)
const selectedItem = ref<InboxItem | null>(null); const editContent = ref('')
const emptyLabel = computed(() => status.value === 'inbox' ? '收集箱是空的。想到什么，可以先记在这里。' : '这里还没有内容。')
const groupedItems = computed(() => {
  const now = new Date()
  const today = dateKey(now)
  const yesterdayDate = new Date(now.getFullYear(), now.getMonth(), now.getDate() - 1)
  const yesterday = dateKey(yesterdayDate)
  const groups = new Map<string, InboxItem[]>()
  for (const item of items.value) {
    const key = dateKey(new Date(item.created_at))
    const group = key === today ? 'today' : key === yesterday ? 'yesterday' : 'older'
    groups.set(group, [...(groups.get(group) ?? []), item])
  }
  return (['today', 'yesterday', 'older'] as const).filter(key => groups.has(key)).map(key => ({ key, label: ({ today: '今天', yesterday: '昨天', older: '更早' })[key], items: groups.get(key) ?? [] }))
})
function dateKey(date: Date) { return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}` }
function formatDate(value: string) { return new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value)) }
async function load() {
  const sequence = ++loadSequence
  if (status.value === 'records') { loading.value = false; return }
  loading.value = true; error.value = ''
  try { const view = await listInbox(status.value); if (sequence === loadSequence) { items.value = view.items; counts.value = view.counts } }
  catch (reason) { if (sequence === loadSequence) error.value = reason instanceof Error ? reason.message : '收集箱读取失败' }
  finally { if (sequence === loadSequence) loading.value = false }
}
function captureCompleted() { status.value = 'inbox'; void load() }
function captured(item: InboxItem) {
  if (status.value !== 'inbox') status.value = 'inbox'
  if (item.status === 'inbox') items.value = [item, ...items.value.filter(existing => existing.id !== item.id)]
  void load()
}
function viewItem(item: InboxItem) { selectedItem.value = item; detailOpen.value = true }
function openSaveRecord(item: InboxItem) { selectedItem.value = item; recordOpen.value = true }
function recordSaved() { if (selectedItem.value) items.value = items.value.filter(item => item.id !== selectedItem.value!.id); void load() }
async function openRecord(item: InboxItem) { if (!item.processed_to_id) return; status.value = 'records'; await nextTick(); await recordsPanel.value?.openById(item.processed_to_id) }

function select(next: Tab) { if (status.value !== next) status.value = next }
function openConvert(item: InboxItem) { selectedItem.value = item; convertOpen.value = true }
function openEdit(item: InboxItem) { selectedItem.value = item; editContent.value = item.content; editOpen.value = true }
async function saveEdit() {
  if (!selectedItem.value || saving.value || !editContent.value.trim()) return
  saving.value = true
  try { await updateInboxItem(selectedItem.value.id, editContent.value); editOpen.value = false; await load(); ElMessage.success('已保存') }
  catch (reason) { ElMessage.error(reason instanceof Error ? reason.message : '保存失败') }
  finally { saving.value = false }
}
async function archive(item: InboxItem) {
  if (pending.has(item.id)) return; pending.add(item.id)
  try { await archiveInboxItem(item.id); detailOpen.value = false; items.value = items.value.filter(existing => existing.id !== item.id); await load(); ElMessage.success('已归档') }
  catch (reason) { ElMessage.error(reason instanceof Error ? reason.message : '归档失败') } finally { pending.delete(item.id) }
}
async function remove(item: InboxItem) {
  try { await ElMessageBox.confirm('删除这条收集内容？删除后无法恢复。', '确认删除', { type: 'warning' }); await deleteInboxItem(item.id); detailOpen.value = false; items.value = items.value.filter(existing => existing.id !== item.id); await load(); ElMessage.success('已删除') }
  catch (reason) { if (reason !== 'cancel' && reason !== 'close') ElMessage.error(reason instanceof Error ? reason.message : '删除失败') }
}
function handleConverted(taskId: number, projectId: number) { void load(); router.push({ path: '/projects', query: { project: String(projectId), view: 'board', task: String(taskId) } }) }
function openTask(item: InboxItem) { if (!item.processed_to_id) return; router.push({ path: '/projects', query: { view: 'board', task: String(item.processed_to_id) } }) }
watch(status, () => { void load() })
onMounted(load)
onBeforeUnmount(() => { ++loadSequence })
</script>

<style scoped>
.inbox-page { max-width: 1080px; }
.inbox-page :deep(.page-header) { margin-bottom:20px; }
.inbox-tabs { display:flex; gap:18px; border-bottom:1px solid var(--border-subtle); }
.inbox-tabs button { display:flex; align-items:center; gap:8px; position:relative; min-height:42px; padding:0 3px; border:0; background:transparent; color:var(--text-secondary); font-size:14px; cursor:pointer; }
.inbox-tabs button.active { color:var(--color-primary); font-weight:650; }
.inbox-tabs button.active::after { position:absolute; right:0; bottom:-1px; left:0; height:2px; background:var(--color-primary); content:''; }
.inbox-tabs span { color:var(--text-tertiary); font-size:12px; }
.inbox-list { min-height:180px; }
.inbox-detail { max-height:60dvh; overflow-y:auto; }
.inbox-detail-actions { display:flex; justify-content:flex-end; gap:8px; flex-wrap:wrap; }
.inbox-detail-actions :deep(.el-button) { margin-left:0; }
.inbox-day-group { margin-top:20px; }
.inbox-day-group h2 { margin:0; padding-bottom:8px; color:var(--text-tertiary); font-size:13px; font-weight:600; }
.inbox-item { display:flex; align-items:flex-start; justify-content:space-between; gap:20px; padding:18px 4px; border-bottom:1px solid var(--border-subtle); }
.inbox-item__main { min-width:0; flex:1; }
.inbox-item__content { width:100%; padding:0; border:0; background:none; text-align:left; cursor:pointer; display:-webkit-box; -webkit-line-clamp:2; -webkit-box-orient:vertical; overflow:hidden; margin:0; color:var(--text-primary); font-size:15px; line-height:1.65; white-space:pre-wrap; overflow-wrap:anywhere; }
.inbox-item__meta { display:flex; align-items:center; flex-wrap:wrap; gap:10px; margin-top:9px; color:var(--text-tertiary); font-size:12px; }
.inbox-item__meta a { color:var(--text-secondary); }
.inbox-item__actions { display:flex; flex:none; align-items:center; flex-wrap:wrap; justify-content:flex-end; }
.inbox-item__actions :deep(.el-button) { color:var(--text-secondary); }
.inbox-item__actions :deep(.delete-action) { color:var(--color-danger); }
@media(max-width:760px){.inbox-item{flex-direction:column;gap:8px}.inbox-item__actions{justify-content:flex-start;margin-left:-10px}}
</style>
