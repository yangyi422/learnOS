<template>
  <section class="inbox-page page-stack">
    <PageHeader title="收集箱" description="先记下来，稍后再整理。">
      <template #actions><el-button type="primary" @click="captureOpen = true">快速记录</el-button></template>
    </PageHeader>
    <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />
    <nav class="inbox-tabs" aria-label="收集箱筛选">
      <button v-for="tab in tabs" :key="tab.key" type="button" :class="{ active: status === tab.key }" :aria-current="status === tab.key ? 'page' : undefined" @click="select(tab.key)">{{ tab.label }}<span>{{ counts[tab.key] ?? 0 }}</span></button>
    </nav>
    <div v-loading="loading" class="inbox-list">
      <section v-for="group in groupedItems" :key="group.key" class="inbox-day-group">
      <h2>{{ group.label }}</h2>
      <article v-for="item in group.items" :key="item.id" class="inbox-item">
        <div class="inbox-item__main">
          <p>{{ item.content }}</p>
          <div class="inbox-item__meta"><time>{{ formatDate(item.created_at) }}</time><span v-if="item.source_type === 'url'">链接</span><a v-if="item.source_url" :href="item.source_url" target="_blank" rel="noopener noreferrer">打开来源 ↗</a><span v-if="item.status === 'processed' && item.processed_to_type === 'task'">已转为任务</span></div>
        </div>
        <div class="inbox-item__actions">
          <el-button v-if="item.status === 'inbox'" text @click="openConvert(item)">转为任务</el-button>
          <el-button v-if="item.status === 'inbox'" text @click="openEdit(item)">编辑</el-button>
          <el-button v-if="item.status !== 'archived'" text @click="archive(item)">归档</el-button>
          <el-button v-if="item.status === 'processed' && item.processed_to_type === 'task' && item.processed_to_id" text @click="openTask(item)">查看任务</el-button>
          <el-button text class="delete-action" @click="remove(item)">删除</el-button>
        </div>
      </article>
      </section>
      <el-empty v-if="!loading && items.length === 0" :description="emptyLabel" />
    </div>
    <QuickCaptureDialog v-model="captureOpen" @created="load" />
    <ConvertInboxDialog v-model="convertOpen" :item="selectedItem" @converted="handleConverted" />
    <el-dialog v-model="editOpen" title="编辑收集内容" width="min(520px, calc(100vw - 28px))">
      <el-input v-model="editContent" type="textarea" :rows="5" maxlength="10000" show-word-limit />
      <template #footer><el-button @click="editOpen = false">取消</el-button><el-button type="primary" :loading="saving" :disabled="!editContent.trim()" @click="saveEdit">保存</el-button></template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import PageHeader from '@/components/PageHeader.vue'
import QuickCaptureDialog from '@/components/QuickCaptureDialog.vue'
import ConvertInboxDialog from '@/components/ConvertInboxDialog.vue'
import { archiveInboxItem, deleteInboxItem, listInbox, updateInboxItem, type InboxItem, type InboxStatus } from '@/api/inbox'

const tabs: { key: InboxStatus; label: string }[] = [{ key: 'inbox', label: '待整理' }, { key: 'processed', label: '已处理' }, { key: 'archived', label: '已归档' }]
const router = useRouter()
const status = ref<InboxStatus>('inbox')
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
  loading.value = true; error.value = ''
  try { const view = await listInbox(status.value); items.value = view.items; counts.value = view.counts }
  catch (reason) { error.value = reason instanceof Error ? reason.message : '收集箱读取失败' }
  finally { loading.value = false }
}
function select(next: InboxStatus) { if (status.value !== next) status.value = next }
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
  try { await archiveInboxItem(item.id); await load(); ElMessage.success('已归档') }
  catch (reason) { ElMessage.error(reason instanceof Error ? reason.message : '归档失败') }
}
async function remove(item: InboxItem) {
  try { await ElMessageBox.confirm('删除这条收集内容？删除后无法恢复。', '确认删除', { type: 'warning' }); await deleteInboxItem(item.id); await load(); ElMessage.success('已删除') }
  catch (reason) { if (reason !== 'cancel' && reason !== 'close') ElMessage.error(reason instanceof Error ? reason.message : '删除失败') }
}
function handleConverted(taskId: number, projectId: number) { void load(); router.push({ path: '/projects', query: { project: String(projectId), view: 'board', task: String(taskId) } }) }
function openTask(item: InboxItem) { if (!item.processed_to_id) return; router.push({ path: '/projects', query: { view: 'board', task: String(item.processed_to_id) } }) }
watch(status, () => { void load() })
onMounted(load)
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
.inbox-day-group { margin-top:20px; }
.inbox-day-group h2 { margin:0; padding-bottom:8px; color:var(--text-tertiary); font-size:13px; font-weight:600; }
.inbox-item { display:flex; align-items:flex-start; justify-content:space-between; gap:20px; padding:18px 4px; border-bottom:1px solid var(--border-subtle); }
.inbox-item__main { min-width:0; flex:1; }
.inbox-item__main p { margin:0; color:var(--text-primary); font-size:15px; line-height:1.65; white-space:pre-wrap; overflow-wrap:anywhere; }
.inbox-item__meta { display:flex; align-items:center; flex-wrap:wrap; gap:10px; margin-top:9px; color:var(--text-tertiary); font-size:12px; }
.inbox-item__meta a { color:var(--text-secondary); }
.inbox-item__actions { display:flex; flex:none; align-items:center; flex-wrap:wrap; justify-content:flex-end; }
.inbox-item__actions :deep(.el-button) { color:var(--text-secondary); }
.inbox-item__actions :deep(.delete-action) { color:var(--color-danger); }
@media(max-width:760px){.inbox-item{flex-direction:column;gap:8px}.inbox-item__actions{justify-content:flex-start;margin-left:-10px}}
</style>
