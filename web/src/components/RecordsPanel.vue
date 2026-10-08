<template>
  <section class="records-panel" aria-label="轻量记录">
    <header><div><h3 v-if="!projectId">轻量记录</h3><p>保留工作上下文，长篇笔记通过链接打开。</p></div><el-button @click="create">新增记录</el-button></header>
    <nav aria-label="记录筛选"><button type="button" :class="{ active: !archived }" @click="archived = false">记录</button><button type="button" :class="{ active: archived }" @click="archived = true">已归档记录</button></nav>
    <el-alert v-if="error" :title="error" type="error" :closable="false" /><el-button v-if="error" text @click="load">重试</el-button>
    <div v-loading="loading">
      <article v-for="item in visibleRecords" :key="item.id" class="record-row">
        <button type="button" class="record-row__open" @click="openRecord(item)"><span>{{ item.content }}</span><time>{{ new Date(item.created_at).toLocaleString('zh-CN') }}</time></button>
        <ExternalLink v-if="item.external_url" :url="item.external_url" :name="item.link_name" />
      </article>
      <p v-if="!loading && !records.length && !error" class="records-empty">{{ archived ? '暂无归档记录。' : '暂无记录，可以保存一条想法或资料。' }}</p>
      <el-button v-if="records.length > initialLimit" text @click="expanded = !expanded">{{ expanded ? '收起记录' : `查看全部记录（${records.length}）` }}</el-button>
    </div>
    <RecordDialog v-model="dialogOpen" :record="selected" :project-id="projectId" @saved="saved" @changed="load" />
  </section>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { getRecord, listRecords, type LightweightRecord } from '@/api/records'
import RecordDialog from '@/components/RecordDialog.vue'
import ExternalLink from '@/components/ExternalLink.vue'
const props = defineProps<{ projectId?: number | null }>()
const records = ref<LightweightRecord[]>([]); const archived = ref(false); const expanded = ref(false)
const dialogOpen = ref(false); const selected = ref<LightweightRecord | null>(null)
const loading = ref(false); const error = ref(''); const initialLimit = 8; let sequence = 0
const visibleRecords = computed(() => expanded.value ? records.value : records.value.slice(0, initialLimit))
async function load() {
  const current = ++sequence; loading.value = true; error.value = ''
  try { const result = await listRecords(props.projectId || undefined, archived.value); if (sequence === current) records.value = result }
  catch (reason) { if (sequence === current) error.value = reason instanceof Error ? reason.message : '记录读取失败，请重试。' }
  finally { if (sequence === current) loading.value = false }
}
function create() { selected.value = null; dialogOpen.value = true }
function openRecord(item: LightweightRecord) { selected.value = item; dialogOpen.value = true }
async function openById(id: number) {
  try { openRecord(await getRecord(id)) } catch (reason) { error.value = reason instanceof Error ? reason.message : '关联记录已删除或不可用。' }
}
function saved(record: LightweightRecord) {
  selected.value = record
  records.value = records.value.filter(item => item.id !== record.id)
  if ((!props.projectId || record.project_id === props.projectId) && !!record.archived_at === archived.value) records.value = [record, ...records.value]
  void load()
}
watch(() => [props.projectId, archived.value], () => { records.value = []; expanded.value = false; void load() }, { immediate: true })
onBeforeUnmount(() => { ++sequence })
defineExpose({ load, openById })
</script>
<style scoped>
.records-panel { max-width: 860px; padding: 16px 0; }
header { display: flex; justify-content: space-between; align-items: center; gap: 16px; }
h3 { margin: 0; font-size: 17px; } header p { margin: 6px 0 12px; color: var(--text-secondary); font-size: 13px; }
nav { display: flex; gap: 20px; border-bottom: 1px solid var(--border-subtle); }
nav button { border: 0; padding: 10px 0; color: var(--text-secondary); background: none; cursor: pointer; } nav button.active { color: var(--color-primary); border-bottom: 2px solid var(--color-primary); }
.record-row { padding: 16px 0; border-bottom: 1px solid var(--border-subtle); }
.record-row__open { display: flex; width: 100%; flex-direction: column; gap: 8px; padding: 0; background: none; border: 0; text-align: left; cursor: pointer; color: var(--text-primary); }
.record-row__open span { display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; overflow-wrap: anywhere; white-space: pre-wrap; line-height: 1.7; }
time, .records-empty { color: var(--text-tertiary); font-size: 12px; }
.record-row :deep(.external-link) { margin-top: 8px; }
</style>
