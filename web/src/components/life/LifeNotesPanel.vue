<template>
  <section class="life-notes" v-loading="loading">
    <el-alert v-if="error" :title="error" type="error" :closable="false" show-icon><el-button text @click="load">重试</el-button></el-alert>
    <div class="life-notes__layout">
      <aside class="life-notes__list">
        <div class="life-notes__head">
          <h2 class="life-month">笔记</h2>
          <el-button size="small" text @click="startNew">+ 新笔记</el-button>
        </div>
        <EmptyState v-if="!loading && !notes.length" title="还没有笔记"
          description="写下一篇长一点的想法、复盘或读后记，它会和这条时间线一起留在这里。"
          action-label="写第一篇笔记" @action="startNew" />
        <button v-for="note in notes" :key="note.id" type="button" class="life-event-card life-notes__item"
          :class="{ 'is-active': note.id === currentId }" @click="open(note)">
          <h3>{{ noteTitle(note.content) }}</h3>
          <p>{{ noteSnippet(note.content) }}</p>
          <div class="life-metadata"><time :datetime="note.updated_at">{{ updatedLabel(note.updated_at) }}</time><span>{{ charCountOf(note.content) }} 字</span></div>
        </button>
      </aside>
      <article v-if="editing" class="life-note-editor">
        <input v-model="title" class="life-note-editor__title" maxlength="120" placeholder="标题（可留空，第一行会作为标题）" aria-label="笔记标题" />
        <textarea ref="bodyEl" v-model="body" class="life-note-editor__body life-reading" spellcheck="false"
          placeholder="写点什么：想法、复盘、读后记、一段还没想清楚的记录……" aria-label="笔记正文"
          @keydown.ctrl.s.prevent="save" @keydown.meta.s.prevent="save" />
        <footer class="life-footer">
          <span class="life-muted">{{ charCount }} / 10000 字 · {{ dirty ? '未保存' : '已保存' }}</span>
          <span class="life-spacer" />
          <el-button v-if="currentId" text type="danger" :disabled="saving" @click="remove">删除</el-button>
          <el-button type="primary" :disabled="!dirty || !canSave || saving" @click="save">{{ saving ? '保存中…' : '保存' }}</el-button>
        </footer>
      </article>
      <article v-else class="life-note-editor life-note-editor--idle">
        <p class="life-muted">从左侧选择一篇笔记，或点「新笔记」开始写。</p>
      </article>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import EmptyState from '@/components/EmptyState.vue'
import { deleteRecord, listRecords, saveRecord, type LightweightRecord } from '@/api/records'

const LIMIT = 10000
const notes = ref<LightweightRecord[]>([])
const loading = ref(false)
const saving = ref(false)
const error = ref('')
const editing = ref(false)
const currentId = ref<number | null>(null)
const title = ref('')
const body = ref('')
const baseline = ref('')
const bodyEl = ref<HTMLTextAreaElement | null>(null)
const resetScroll = () => { void nextTick(() => { if (bodyEl.value) bodyEl.value.scrollTop = 0 }) }

const parse = (content: string) => {
  const lines = content.split('\n')
  const first = (lines[0] ?? '').trim()
  const rest = lines.slice(1).join('\n').replace(/^\s*\n+/, '')
  return first ? { title: first, body: rest } : { title: '', body: content.trim() }
}
const compose = () => {
  const t = title.value.trim()
  const b = body.value.replace(/^\s*\n+/, '').trimEnd()
  return t ? (b ? t + '\n\n' + b : t) : b
}
const charCount = computed(() => compose().replace(/\s/g, '').length)
const canSave = computed(() => compose().trim().length > 0 && compose().length <= LIMIT)
const dirty = computed(() => editing.value && compose() !== baseline.value)

const noteTitle = (content: string) => parse(content).title || '未命名笔记'
const charCountOf = (content: string) => content.replace(/\s/g, '').length
const noteSnippet = (content: string) => {
  const text = parse(content).body.replace(/\s+/g, ' ').trim()
  return text || '（只有标题）'
}
const updatedLabel = (value: string) => {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return '更新于 ' + date.getFullYear() + '-' + String(date.getMonth() + 1).padStart(2, '0') + '-' + String(date.getDate()).padStart(2, '0') + ' ' + String(date.getHours()).padStart(2, '0') + ':' + String(date.getMinutes()).padStart(2, '0')
}

async function load() {
  loading.value = true; error.value = ''
  try { notes.value = (await listRecords()).slice().sort((a, b) => b.updated_at.localeCompare(a.updated_at)) }
  catch (reason) { error.value = reason instanceof Error ? reason.message : '笔记读取失败，请重试。' }
  finally { loading.value = false }
}

async function confirmDiscard() {
  if (!dirty.value) return true
  try {
    await ElMessageBox.confirm('这篇笔记还有未保存的修改，继续会丢掉它们。', '未保存的修改', { confirmButtonText: '丢弃修改', cancelButtonText: '继续编辑', type: 'warning' })
    return true
  } catch (reason) { if (reason === 'cancel' || reason === 'close') return false; throw reason }
}

async function startNew() {
  if (!(await confirmDiscard())) return
  editing.value = true; currentId.value = null; title.value = ''; body.value = ''; baseline.value = ''
  resetScroll()
}

async function open(note: LightweightRecord) {
  if (note.id === currentId.value && editing.value) return
  if (!(await confirmDiscard())) return
  const parsed = parse(note.content)
  editing.value = true; currentId.value = note.id; title.value = parsed.title; body.value = parsed.body; baseline.value = note.content
  resetScroll()
}

async function save() {
  if (!canSave.value || saving.value) return
  saving.value = true
  try {
    const saved = await saveRecord({ content: compose(), project_id: null, external_url: '', link_name: '' }, currentId.value ?? undefined)
    currentId.value = saved.id; baseline.value = saved.content
    await load(); ElMessage.success('笔记已保存')
  } catch (reason) { ElMessage.error(reason instanceof Error ? reason.message : '保存失败') }
  finally { saving.value = false }
}

async function remove() {
  if (!currentId.value) { editing.value = false; return }
  try { await ElMessageBox.confirm('删除这篇笔记？删除后无法恢复。', '确认删除', { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' }) }
  catch (reason) { if (reason === 'cancel' || reason === 'close') return; throw reason }
  try {
    await deleteRecord(currentId.value)
    editing.value = false; currentId.value = null; title.value = ''; body.value = ''; baseline.value = ''
    await load(); ElMessage.success('笔记已删除')
  } catch (reason) { ElMessage.error(reason instanceof Error ? reason.message : '删除失败') }
}

onMounted(load)
defineExpose({ startNew })
</script>

<style scoped>
/* 布局与交互细节只写这里；材质、字号、间距一律复用生活页既有类与全局 token。 */
.life-notes { display: grid; gap: 16px; }
.life-notes__layout { display: grid; grid-template-columns: minmax(240px, 340px) minmax(0, 1fr); gap: 20px; align-items: start; padding-top: 12px; }
.life-notes__list { display: grid; gap: 12px; align-content: start; min-width: 0; }
.life-notes__head { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.life-notes__head .life-month { margin: 0; }
.life-notes__item { display: block; }
.life-notes__item.is-active { border-left: 3px solid var(--color-primary); }
.life-notes__item .life-metadata { margin-top: 2px; }
/* 编辑区铺满面板：标题固定、正文吃掉剩余高度、底部操作贴底。 */
/* 高度按"视口减去页头与页脚留白"计算，保证底部操作在首屏可见。 */
.life-note-editor { display: grid; grid-template-rows: auto 1fr auto; gap: 14px; min-height: clamp(340px, calc(100vh - 430px), 720px); }
.life-note-editor--idle { grid-template-rows: 1fr; align-content: center; justify-items: center; min-height: clamp(320px, 46vh, 520px); }
.life-note-editor__title { width: 100%; padding: 0 0 10px; border: 0; border-bottom: 1px solid var(--border-subtle); background: transparent; color: var(--text-primary); font-size: var(--font-section-title); font-weight: 600; }
.life-note-editor__body { width: 100%; height: 100%; min-height: 280px; padding: 0; border: 0; background: transparent; color: var(--text-primary); font: inherit; font-size: var(--font-body); resize: none; }
.life-note-editor__title:focus-visible, .life-note-editor__body:focus-visible { outline: 2px solid var(--color-focus); outline-offset: 3px; }
@media (max-width: 820px) {
  .life-notes__layout { grid-template-columns: minmax(0, 1fr); }
  .life-note-editor { min-height: 0; }
  .life-note-editor__body { min-height: 240px; }
}
</style>