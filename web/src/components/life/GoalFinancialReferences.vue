<template>
  <section class="life-history goal-financial-references" v-loading="loading">
    <div class="life-metadata"><h3 class="life-spacer">财务参考</h3><el-button text :disabled="disabled || busy || choosing" @click="start()">关联资产快照</el-button></div>
    <p class="life-muted">保存目标的财务背景；正式进展与决定仍记录在变化历程，不计算资金进度。</p>
    <el-alert v-if="error" :title="error" type="error" :closable="false"><el-button v-if="!choosing" text @click="load">重试读取</el-button></el-alert>
    <el-form v-if="choosing" label-position="top" @submit.prevent="save">
      <p v-if="!editing && !options.length && !loading" class="life-muted">{{ snapshots.length ? '已有快照均已关联，可编辑下方说明。' : '还没有资产快照。请先在财务档案中记录资产，再回来关联。' }}</p>
      <el-form-item v-if="!editing && options.length" label="已有资产快照" required>
        <el-select v-model="selectedID" aria-label="选择已有资产快照" placeholder="选择一份已有记录" style="width:100%"><el-option v-for="row in options" :key="row.id" :value="row.id" :label="`${row.snapshot_date} · ${row.complete ? '完整范围' : '不完整范围'}`" /></el-select>
      </el-form-item>
      <div v-if="chosen" class="goal-financial-summary"><strong>{{ chosen.snapshot_date }} · {{ chosen.complete ? '完整范围' : '不完整范围' }}</strong><p>{{ summary(chosen) }}</p></div>
      <el-form-item label="判断日期"><el-date-picker v-model="form.judgment_date" type="date" value-format="YYYY-MM-DD" :clearable="false" /></el-form-item>
      <el-form-item label="关联说明（可选）"><el-input v-model="form.note" type="textarea" :rows="3" maxlength="1000" placeholder="为什么这份记录与目标有关？" /></el-form-item>
      <el-button :disabled="busy" @click="cancel">取消关联说明</el-button><el-button type="primary" :loading="busy" :disabled="!chosen || !form.judgment_date || disabled" @click="save">{{ editing ? '保存说明' : '确认关联' }}</el-button>
    </el-form>
    <p v-if="!loading && !error && !rows.length && !choosing" class="life-muted">尚未关联资产快照。</p>
    <article v-for="row in rows" :key="row.id" class="life-history-entry">
      <div class="life-metadata"><strong>{{ row.snapshot.snapshot_date }}</strong><span>{{ row.snapshot.complete ? '完整范围' : '不完整范围' }}</span></div>
      <p class="goal-financial-summary">{{ summary(row.snapshot) }}</p>
      <p v-if="row.note" class="goal-financial-note">{{ row.note }}</p><p v-else class="life-muted">未填写关联说明。</p>
      <div class="life-muted">判断日期 {{ row.judgment_date }} · 快照最近修改 {{ localTime(row.snapshot.updated_at) }}</div>
      <p class="life-muted">金额来自快照当前保存的内容，可能已修订；关联说明不会随金额自动改变。</p>
      <div class="life-metadata"><el-button text :disabled="busy || disabled" @click="view(row)">查看快照</el-button><el-button text :disabled="busy || disabled || choosing" @click="start(row)">编辑说明</el-button><el-button text :disabled="busy || disabled || choosing" @click="unlink(row)">取消关联</el-button></div>
    </article>
  </section>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { ElMessageBox } from 'element-plus'
import { listAssets, localAssetDate, type AssetSnapshot } from '@/api/assets'
import { financeMoney } from '@/api/finance'
import { listGoalAssets, saveGoalAsset, unlinkGoalAsset, type GoalAssetReference } from '@/api/goalAssets'
const props = defineProps<{ goalId: number; disabled?: boolean }>()
const emit = defineEmits<{ asset: [number]; state: [boolean, boolean] }>()
const rows = ref<GoalAssetReference[]>([]), snapshots = ref<AssetSnapshot[]>([]), loading = ref(false), busy = ref(false), choosing = ref(false), error = ref(''), editing = ref<GoalAssetReference>(), selectedID = ref<number>()
const form = reactive({ note: '', judgment_date: localAssetDate() })
let sequence = 0, initial = '', disposed = false
const options = computed(() => snapshots.value.filter(s => !rows.value.some(r => r.asset_snapshot_id === s.id)))
const chosen = computed(() => editing.value?.snapshot ?? snapshots.value.find(r => r.id === selectedID.value))
const dirty = computed(() => choosing.value && JSON.stringify({ ...form, selectedID: selectedID.value }) !== initial)
watch([dirty, busy], () => emit('state', dirty.value, busy.value))
const money = (value: string | null) => value === null ? '未记录' : `${financeMoney(value)} CNY`
const summary = (s: AssetSnapshot) => `${s.complete ? '资产总额' : '已填资产合计'} ${money(s.asset_total_minor)} · 负债 ${money(s.debt_minor)} · ${s.complete ? '净资产' : '已填范围差额'} ${money(s.net_asset_minor)}`
const localTime = (value: string) => new Date(value).toLocaleString('zh-CN')
const message = (e: unknown) => e instanceof Error ? e.message : '财务参考暂时无法保存，请重试。'
async function load() { const token = ++sequence; loading.value = true; error.value = ''; try { const data = await listGoalAssets(props.goalId); if (token === sequence && !disposed) rows.value = data } catch (e) { if (token === sequence) error.value = message(e) } finally { if (token === sequence) loading.value = false } }
watch(() => props.goalId, () => { rows.value = []; choosing.value = false; void load() }, { immediate: true })
async function start(row?: GoalAssetReference) {
  if (busy.value || props.disabled) return
  editing.value = row; selectedID.value = row?.asset_snapshot_id; Object.assign(form, { note: row?.note ?? '', judgment_date: row?.judgment_date ?? localAssetDate() }); choosing.value = true; error.value = ''
  initial = JSON.stringify({ ...form, selectedID: selectedID.value })
  if (!row) { const token = ++sequence; loading.value = true; try { const data = await listAssets(); if (token !== sequence || disposed) return; snapshots.value = data; if (options.value.length === 1) selectedID.value = options.value[0]!.id; initial = JSON.stringify({ ...form, selectedID: selectedID.value }) } catch (e) { if (token === sequence) error.value = message(e) } finally { if (token === sequence) loading.value = false } }
}
async function allowDiscard() { if (busy.value) return false; if (dirty.value) { try { await ElMessageBox.confirm('放弃尚未保存的关联说明？', '未保存的修改', { confirmButtonText: '放弃修改', cancelButtonText: '继续编辑' }) } catch { return false } }; return true }
async function cancel() { if (await allowDiscard()) { choosing.value = false; error.value = '' } }
async function save() {
  if (!chosen.value || busy.value || props.disabled) return
  const goal = props.goalId, snapshot = chosen.value, token = sequence
  busy.value = true; error.value = ''
  try { const saved = await saveGoalAsset(goal, snapshot.id, { ...form, version: editing.value?.version }, Boolean(editing.value)); if (disposed || token !== sequence) return; rows.value = [...rows.value.filter(r => r.asset_snapshot_id !== snapshot.id), { ...saved, snapshot }].sort((a,b) => b.snapshot.snapshot_date.localeCompare(a.snapshot.snapshot_date)); choosing.value = false; await load() } catch (e) { if (!disposed && token === sequence) error.value = message(e) } finally { busy.value = false }
}
async function unlink(row: GoalAssetReference) { if (busy.value || props.disabled) return; try { await ElMessageBox.confirm('解除关联？资产快照和目标阶段记录都会保留，关联说明将删除。', '取消财务关联', { confirmButtonText: '解除关联', cancelButtonText: '保留关联' }) } catch { return }; busy.value = true; error.value = ''; try { await unlinkGoalAsset(row); if (!disposed) rows.value = rows.value.filter(r => r.id !== row.id) } catch (e) { error.value = message(e) } finally { busy.value = false } }
function view(row: GoalAssetReference) { emit('asset', row.asset_snapshot_id) }
defineExpose({ allowDiscard })
onBeforeUnmount(() => { disposed = true; ++sequence })
</script>
