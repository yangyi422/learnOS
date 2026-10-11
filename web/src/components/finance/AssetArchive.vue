<template>
  <section class="asset-archive" aria-label="资产档案">
    <div class="asset-heading"><div><h2>资产档案</h2><p class="finance-muted">定期留下资产与负债的真实记录。与钱迹收支独立，不推算实时余额。</p></div><el-button type="primary" @click="create">新增资产快照</el-button></div>
    <el-alert v-if="error" :title="error" type="error" :closable="false"><el-button text @click="load">重试读取资产</el-button></el-alert>
    <div v-loading="loading" class="asset-body">
      <EmptyState v-if="!loading && !error && !rows.length" title="留下第一份资产记录" description="按类别填写实际金额，空白表示未记录；不要求每月填写。" action-label="新增第一份快照" @action="create" />
      <template v-if="latest && !error">
        <div class="asset-heading"><div><h3>最近记录的资产 · {{ latest.snapshot_date }}</h3><p class="finance-muted">{{ latest.complete ? '完整范围 · 所有类别及负债均有记录' : '不完整范围 · 仅反映已填写部分，不代表完整净资产' }} · CNY</p></div><el-button text @click="open(latest)">查看最新快照</el-button></div>
        <div class="finance-totals asset-totals"><div><span>{{ latest.complete ? '资产总额' : '已填资产合计' }}</span><strong>{{ money(latest.asset_total_minor) }}</strong></div><div><span>负债总额</span><strong>{{ money(latest.debt_minor) }}</strong></div><div><span>{{ latest.complete ? '净资产' : '已填范围差额' }}</span><strong>{{ money(latest.net_asset_minor) }}</strong></div></div>
        <div class="asset-composition"><div v-for="[key, label] in assetFields.slice(0, 4)" :key="key"><span>{{ label }}</span><strong>{{ money(latest[`${key}_minor`]) }}</strong></div></div>
        <p v-if="latest.note" class="asset-note-preview">{{ latest.note }}</p>
        <section class="finance-section"><h3>净资产历史</h3><p class="finance-muted">仅比较完整范围快照；{{ partialCount }} 份不完整记录保留在历史列表中。日期间的连线仅连接实际记录，不补齐月份。</p>
          <p v-if="!comparable.length" class="finance-muted">还没有完整范围的快照。请记录所有类别及负债；已核实没有余额的类别可明确填写 0。</p>
          <p v-else-if="comparable.length === 1" class="finance-muted">只有一份完整快照，显示单个记录点，暂不形成趋势。</p>
          <div v-if="comparable.length" class="asset-chart">
            <svg viewBox="0 0 800 240" role="group" aria-label="按实际日期记录的完整净资产历史">
              <g v-for="tick in chart.ticks" :key="tick.label"><line x1="150" x2="775" :y1="tick.y" :y2="tick.y" class="asset-grid" /><text x="140" :y="tick.y + 4" text-anchor="end" class="asset-axis">{{ tick.label }}</text></g>
              <polyline v-if="chart.points.length > 1" :points="chart.points.map(p => `${p.x},${p.y}`).join(' ')" class="asset-line" />
              <g v-for="point in chart.points" :key="point.row.id" role="button" tabindex="0" :aria-label="`${point.row.snapshot_date} 净资产 ${money(point.row.net_asset_minor)}`" class="asset-point" @click="open(point.row)" @keydown.enter.prevent="open(point.row)" @keydown.space.prevent="open(point.row)"><title>{{ point.row.snapshot_date }} · {{ money(point.row.net_asset_minor) }} CNY</title><circle :cx="point.x" :cy="point.y" r="6" /></g>
              <text x="150" y="228" class="asset-axis">{{ comparable[0]?.snapshot_date }}</text><text v-if="comparable.length > 1" x="775" y="228" text-anchor="end" class="asset-axis">{{ comparable.at(-1)?.snapshot_date }}</text>
            </svg>
          </div>
          <p v-if="change" class="finance-muted">{{ change.previous.snapshot_date }} → {{ change.latest.snapshot_date }} · 净资产变化 <strong>{{ money(change.amount) }} CNY</strong>。这是两次记录之差，不等于投资收益或当期收支结余。</p>
        </section>
        <section class="finance-section"><h3>历史快照</h3><div class="finance-table-wrap"><el-table :data="rows" @row-click="open" row-class-name="finance-clickable"><el-table-column prop="snapshot_date" label="记录日期" width="125" /><el-table-column label="资产合计 / CNY" min-width="160" align="right"><template #default="{ row }">{{ money(row.asset_total_minor) }}</template></el-table-column><el-table-column label="负债 / CNY" min-width="155" align="right"><template #default="{ row }">{{ money(row.debt_minor) }}</template></el-table-column><el-table-column label="净资产 / 范围差额" min-width="170" align="right"><template #default="{ row }">{{ money(row.net_asset_minor) }}</template></el-table-column><el-table-column label="统计范围" width="115"><template #default="{ row }">{{ row.complete ? '完整范围' : '不完整范围' }}</template></el-table-column><el-table-column prop="note" label="备注" min-width="200" show-overflow-tooltip /><el-table-column width="80"><template #default="{ row }"><el-button text @click.stop="open(row)">详情</el-button></template></el-table-column></el-table></div><p class="finance-muted">空白金额显示“未记录”；不完整快照的合计和差额仅属于已填范围。</p></section>
      </template>
    </div>
    <el-dialog v-model="dialogOpen" :title="mode === 'read' ? '资产快照' : mode === 'edit' ? '编辑资产快照' : '新增资产快照'" width="680px" align-center class="finance-dialog asset-dialog" :before-close="beforeClose" :close-on-click-modal="false" :close-on-press-escape="!busy">
      <el-alert v-if="formError" :title="formError" type="error" :closable="false" />
      <template v-if="mode === 'read' && selected">
        <h3>{{ selected.snapshot_date }} · {{ selected.complete ? '完整资产范围' : '不完整资产范围' }}</h3><p class="finance-muted">自行记录的历史资产状态，非实时余额 · CNY</p>
        <dl class="finance-raw"><template v-for="[key, label] in assetFields" :key="key"><dt>{{ label }}</dt><dd>{{ money(selected[`${key}_minor`]) }}</dd></template><dt>{{ selected.complete ? '资产总额' : '已填资产合计' }}</dt><dd>{{ money(selected.asset_total_minor) }}</dd><dt>{{ selected.complete ? '净资产' : '已填范围差额' }}</dt><dd>{{ money(selected.net_asset_minor) }}</dd><dt>备注</dt><dd>{{ selected.note || '未填写' }}</dd></dl>
        <p v-if="!selected.complete" class="finance-muted">未填写的类别不视为零，不能作为完整净资产比较。</p><p class="finance-muted">创建 {{ localTime(selected.created_at) }} · 最后修改 {{ localTime(selected.updated_at) }}</p>
      <section v-if="relatedGoals.length || relatedError || relatedLoading" class="asset-related-goals"><h3>关联长期目标</h3><p v-if="relatedLoading" class="finance-muted">正在读取关联…</p><el-alert v-if="relatedError" :title="relatedError" type="error" :closable="false"><el-button text @click="loadRelated">重试读取</el-button></el-alert><el-button v-for="goal in relatedGoals" :key="goal.id" text @click="openGoal(goal.id)">{{ goal.title }}</el-button><p v-if="relatedGoals.length" class="finance-muted">已被目标引用；删除前需先在目标详情解除关联。</p></section>
      </template>
      <template v-else>
        <div v-if="mode === 'create' && latest" class="asset-template"><el-button :disabled="busy" @click="useTemplate">使用上次快照作为填写模板</el-button><span class="finance-muted">来自 {{ latest.snapshot_date }}</span></div>
        <el-alert v-if="templateUsed" title="这些是上次记录的金额，不代表当前实时余额。请逐项核对后保存。" type="info" :closable="false" />
        <el-form label-position="top" @submit.prevent="save">
          <el-form-item label="快照日期"><el-date-picker v-model="form.snapshot_date" type="date" value-format="YYYY-MM-DD" format="YYYY-MM-DD" aria-label="快照日期" :disabled="busy" /></el-form-item>
          <div class="asset-form-grid"><el-form-item v-for="[key, label] in assetFields" :key="key" :label="label + '（元）'"><el-input v-model="form[key]" inputmode="decimal" :aria-label="label + '（元）'" placeholder="未记录" :disabled="busy" maxlength="13" /></el-form-item></div>
          <p class="finance-muted">其他资产请采用合理估值，并避免与银行、平台或投资金额重复统计。明确没有余额的类别填写 0；尚未核对的类别留空。</p>
          <el-form-item label="简短备注"><el-input v-model="form.note" type="textarea" :rows="3" maxlength="1000" show-word-limit aria-label="资产快照备注" :disabled="busy" /></el-form-item>
        </el-form>
        <p v-if="calculationError" class="asset-validation" role="status">{{ calculationError }}</p>
        <div v-else class="asset-preview" aria-live="polite"><span>{{ totals?.complete ? '资产总额' : '已填资产合计' }} <b>{{ money(totals?.asset?.toString() ?? null) }}</b></span><span>负债 <b>{{ money(totals?.debt?.toString() ?? null) }}</b></span><span>{{ totals?.complete ? '净资产' : '已填范围差额' }} <b>{{ money(totals?.net?.toString() ?? null) }}</b></span></div>
        <p class="finance-muted">{{ totals?.complete ? '所有类别及负债均有记录，可用于完整净资产趋势。' : '范围尚不完整；保存后仅作为部分记录，不进入完整净资产趋势。' }}</p>
        <el-button v-if="duplicate" text @click="openDuplicate">查看这一天已有快照</el-button>
      </template>
      <template #footer><div class="asset-dialog-footer"><el-button v-if="mode === 'read'" text :disabled="busy" @click="remove">删除快照</el-button><span class="asset-footer-spacer" /><el-button :disabled="busy" @click="close">{{ mode === 'read' ? '关闭' : '取消' }}</el-button><el-button v-if="mode === 'read'" :disabled="busy" @click="edit">编辑快照</el-button><el-button v-else type="primary" :loading="busy" @click="save">保存快照</el-button></div></template>
    </el-dialog>
  </section>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { listAssetGoals } from '@/api/goalAssets'
import type { LifeGoal } from '@/api/life'
import { ElMessage, ElMessageBox } from 'element-plus'
import EmptyState from '@/components/EmptyState.vue'
import { APIRequestError } from '@/api/http'
import { financeMoney } from '@/api/finance'
import { assetFields, assetFormFrom, assetTotals, assetChange, comparableAssets, deleteAsset, getAsset, listAssets, localAssetDate, saveAsset, type AssetForm, type AssetInput, type AssetSnapshot } from '@/api/assets'
import '@/assets.css'
const route = useRoute(), router = useRouter()
const relatedGoals = ref<LifeGoal[]>([]), relatedError = ref(''), relatedLoading = ref(false)
const rows = ref<AssetSnapshot[]>([]), loading = ref(false), busy = ref(false), error = ref(''), formError = ref(''), dialogOpen = ref(false), selected = ref<AssetSnapshot>(), mode = ref<'read' | 'edit' | 'create'>('read'), templateUsed = ref(false), duplicate = ref(false)
const blank = (): AssetForm => ({ snapshot_date: localAssetDate(), bank: '', platform: '', investment: '', other: '', debt: '', note: '' })
const form = reactive<AssetForm>(blank())
let sequence = 0, detailSequence = 0, initial = '', disposed = false
const latest = computed(() => rows.value[0]), comparable = computed(() => comparableAssets(rows.value)), partialCount = computed(() => rows.value.filter(r => !r.complete).length), change = computed(() => assetChange(rows.value))
const money = (value: string | null) => value === null ? '未记录' : financeMoney(value)
const localTime = (value: string) => new Date(value).toLocaleString('zh-CN')
const message = (e: unknown) => e instanceof Error ? e.message : '操作失败，请重试。'
const calculationError = computed(() => { try { assetTotals(form); return '' } catch (e) { return message(e) } })
const totals = computed(() => { try { return assetTotals(form) } catch { return null } })
const chart = computed(() => {
  const ordered = comparable.value
  const amounts = ordered.map(r => BigInt(r.net_asset_minor!))
  let min = amounts.reduce((a, b) => a < b ? a : b, 0n), max = amounts.reduce((a, b) => a > b ? a : b, 0n)
  if (min === max) { min -= 100n; max += 100n }
  const y = (amount: bigint) => 195 - Number((amount - min) * 15000n / (max - min)) / 100
  const days = (date: string) => Date.parse(date + 'T00:00:00Z')
  const start = days(ordered[0]?.snapshot_date ?? '2000-01-01'), end = days(ordered.at(-1)?.snapshot_date ?? '2000-01-01')
  return { points: ordered.map(row => ({ row, x: end === start ? 462 : 150 + (days(row.snapshot_date) - start) / (end - start) * 625, y: y(BigInt(row.net_asset_minor!)) })), ticks: [max, (min + max) / 2n, min].map(amount => ({ label: financeMoney(amount.toString()), y: y(amount) })) }
})
async function load() { const token = ++sequence; loading.value = true; error.value = ''; try { const data = await listAssets(); if (token === sequence) rows.value = data } catch (e) { if (token === sequence) error.value = message(e) } finally { if (token === sequence) loading.value = false } }
function resetForm(value: AssetForm) { Object.assign(form, value); initial = JSON.stringify(form); formError.value = ''; duplicate.value = false; templateUsed.value = false }
function create() { ++detailSequence; selected.value = undefined; mode.value = 'create'; resetForm(blank()); dialogOpen.value = true }
async function open(row: AssetSnapshot) { const token = ++detailSequence; try { const data = await getAsset(row.id); if (token !== detailSequence || disposed) return; selected.value = data; mode.value = 'read'; formError.value = ''; dialogOpen.value = true; void loadRelated() } catch (e) { error.value = message(e) } }
function edit() { if (!selected.value) return; resetForm(assetFormFrom(selected.value)); mode.value = 'edit' }
function useTemplate() { if (!latest.value) return; const date = form.snapshot_date; Object.assign(form, assetFormFrom(latest.value, date), { note: form.note }); templateUsed.value = true }
async function beforeClose(done: () => void) { if (busy.value) return; if (mode.value !== 'read' && JSON.stringify(form) !== initial) { try { await ElMessageBox.confirm('放弃尚未保存的快照内容？', '未保存的修改', { confirmButtonText: '放弃修改', cancelButtonText: '继续填写' }) } catch { return } }; ++detailSequence; done() }
function close() { void beforeClose(() => { dialogOpen.value = false }) }
async function openDuplicate() { const row = rows.value.find(r => r.snapshot_date === form.snapshot_date); if (!row) return; await beforeClose(() => { void open(row) }) }
async function save() {
  if (busy.value) return
  formError.value = ''; duplicate.value = false
  if (!form.snapshot_date || calculationError.value || !totals.value?.hasAmount) { formError.value = calculationError.value || '请选择快照日期，并至少填写一项金额；0 是有效金额。'; return }
  const input = { snapshot_date: form.snapshot_date, note: form.note, version: mode.value === 'edit' ? selected.value?.version : undefined } as AssetInput
  for (const [key] of assetFields) input[key] = form[key].trim() || null
  busy.value = true
  try { const saved = await saveAsset(mode.value === 'edit' ? selected.value?.id : undefined, input); if (disposed) return; rows.value = [...rows.value.filter(r => r.id !== saved.id), saved].sort((a, b) => b.snapshot_date.localeCompare(a.snapshot_date)); selected.value = saved; mode.value = 'read'; templateUsed.value = false; void loadRelated(); ElMessage.success('资产快照已保存') }
  catch (e) { formError.value = message(e); if (e instanceof APIRequestError && e.code === 'ASSET_DATE_EXISTS') { duplicate.value = true; await load() } }
  finally { busy.value = false }
}
async function remove() { const row = selected.value; if (!row || busy.value) return; try { await ElMessageBox.confirm(`删除 ${row.snapshot_date} 的资产快照？`, '删除快照', { confirmButtonText: '确认删除', cancelButtonText: '取消', type: 'warning' }) } catch { return }; busy.value = true; formError.value = ''; try { await deleteAsset(row); rows.value = rows.value.filter(r => r.id !== row.id); dialogOpen.value = false; ElMessage.success('资产快照已删除') } catch (e) { formError.value = message(e) } finally { busy.value = false } }
async function loadRelated() { const id = selected.value?.id, token = detailSequence; if (!id) return; relatedGoals.value = []; relatedError.value = ''; relatedLoading.value = true; try { const data = await listAssetGoals(id); if (!disposed && token === detailSequence) relatedGoals.value = data } catch (e) { if (token === detailSequence) relatedError.value = message(e) } finally { if (token === detailSequence) relatedLoading.value = false } }
async function openGoal(id: number) { if (busy.value) return; dialogOpen.value = false; await router.push({ path: '/life', query: { view: 'goals', goal: id, domain: typeof route.query.life_domain === 'string' ? route.query.life_domain : undefined } }) }
async function queryAsset() { const id = Number(route.query.asset); if (Number.isSafeInteger(id) && id > 0) await open({ id } as AssetSnapshot) }
watch(() => route.query.asset, queryAsset)
onMounted(async () => { await load(); await queryAsset() })
onBeforeUnmount(() => { disposed = true; ++sequence; ++detailSequence })
</script>
