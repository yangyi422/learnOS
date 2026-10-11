<template>
  <section class="page-stack finance-page">
    <nav class="life-breadcrumb" aria-label="当前位置"><RouterLink :to="lifeReturn">生活</RouterLink><span>/</span><span>财务档案</span><span>/</span><span aria-current="page">{{ tabs.find(t => t[0] === tab)?.[1] }}</span></nav>
    <PageHeader title="财务档案" description="保存收支历史与资产快照，回顾变化并追溯原始记录。"><template #actions><RouterLink class="finance-back" :to="lifeReturn">返回生活</RouterLink><el-button type="primary" @click="switchTab('imports')">导入钱迹 CSV</el-button></template></PageHeader>
    <nav class="life-tabs finance-tabs" aria-label="财务视图" role="tablist"><button v-for="item in tabs" :key="item[0]" role="tab" type="button" :aria-selected="tab === item[0]" @click="switchTab(item[0])">{{ item[1] }}</button></nav>
    <el-alert v-if="error && tab !== 'assets'" :title="error" type="error" :closable="false" show-icon><el-button text @click="refresh">重试读取</el-button></el-alert>
    <div class="finance-filters" v-if="ledgers.length && tab !== 'imports' && tab !== 'assets'">
      <el-select v-model="year" clearable placeholder="全部年份" aria-label="统计年份"><el-option v-for="y in summary?.years ?? []" :key="y" :label="y + ' 年'" :value="y" /></el-select>
      <el-select v-model="month" clearable :disabled="!year" placeholder="全年" aria-label="统计月份"><el-option v-for="m in 12" :key="m" :label="m + ' 月'" :value="String(m).padStart(2, '0')" /></el-select>
      <template v-if="tab === 'transactions'">
        <el-select v-model="filters.kind" clearable placeholder="全部收支类型" aria-label="收支类型"><el-option label="收入" value="income" /><el-option label="支出" value="expense" /><el-option label="转账" value="transfer" /></el-select>
        <el-input v-model="keywordInput" clearable placeholder="搜索原始备注" aria-label="搜索原始备注" @clear="search" @keyup.enter="search" /><el-button @click="search">搜索</el-button>
      </template>
      <el-button :aria-expanded="advancedOpen" @click="advancedOpen = !advancedOpen">{{ advancedOpen ? '收起筛选' : '更多筛选' }}</el-button>
    </div>
    <div v-if="tab !== 'assets' && ledgers.length && (advancedOpen || tab === 'imports')" class="finance-filters finance-advanced">
      <el-select v-model="ledgerID" aria-label="来源账本" placeholder="来源账本"><el-option v-for="l in ledgers" :key="l.id" :label="l.name" :value="l.id" /></el-select>
      <template v-if="tab !== 'imports'">
        <el-select v-model="currency" placeholder="选择币种" aria-label="统计币种"><el-option v-for="c in summary?.currencies ?? []" :key="c" :value="c" :label="c" /></el-select>
        <el-date-picker v-model="dateRange" type="daterange" value-format="YYYY-MM-DD" start-placeholder="开始日期" end-placeholder="结束日期" aria-label="自定义时间范围" />
        <template v-if="tab === 'transactions'">
          <el-select v-model="filters.category" clearable filterable placeholder="全部一级分类" aria-label="一级分类"><el-option v-for="c in summary?.all_categories ?? []" :key="c" :label="c || '未分类'" :value="c || noCategory" /></el-select>
          <el-input v-model="filters.subcategory" clearable placeholder="二级分类" aria-label="二级分类" />
          <el-checkbox v-model="emptySubcategory">仅无二级分类</el-checkbox>
          <el-select v-model="filters.tag" clearable filterable placeholder="全部标签" aria-label="标签"><el-option v-for="t in summary?.all_tags ?? []" :key="t" :label="'#' + t" :value="t" /></el-select>
        </template>
      </template>
    </div>
    <p v-if="ledgers.length && tab !== 'imports' && tab !== 'assets'" class="finance-filter-status"><span>{{ selectedLedger }} · {{ periodLabel }} · {{ currency || '选择币种' }}</span><span v-for="label in activeFilters" :key="label" class="finance-filter-chip">{{ label }}</span><el-button v-if="hasFilters" text @click="clearAll">清除全部筛选</el-button></p>
    <p v-if="summary && tab !== 'imports' && tab !== 'assets'" class="finance-muted">{{ completenessLabel }}</p>
    <div v-loading="loading && tab !== 'assets'" class="finance-content">
      <AssetArchive v-if="tab === 'assets'" />
      <EmptyState v-else-if="!loading && !error && !ledgers.length && tab !== 'imports'" title="还没有财务历史" description="从钱迹导出 UTF-8 CSV，先预览，再确认导入。" action-label="导入账本" @action="tab = 'imports'" />
      <template v-else-if="tab === 'overview' && summary && !error">
        <p class="finance-muted">{{ coverageLabel }}<span v-if="summary.last_import"> · 最近导入 {{ localTime(summary.last_import) }}<span v-if="latestBatch"> · {{ latestBatch.filename }}（{{ latestBatch.row_count }} 条）</span></span></p>
        <EmptyState v-if="!summary.count" title="这个期间没有账目" description="可以调整时间和币种，或导入新的钱迹数据。" />
        <template v-else>
          <div class="finance-totals"><button type="button" @click="drill({ kind: 'income' })"><span>收入 · {{ summary.currency }}</span><strong>{{ financeMoney(summary.income_minor) }}</strong></button><button type="button" @click="drill({ kind: 'expense' })"><span>支出 · {{ summary.currency }}</span><strong>{{ financeMoney(summary.expense_minor) }}</strong></button><button type="button" @click="drill({})"><span>收支差额 · {{ summary.currency }}</span><strong>{{ financeMoney(summary.difference_minor) }}</strong></button></div>
          <p class="finance-muted">收支差额不代表资产增长。未提供账户余额，不计算净资产。</p>
          <section class="finance-section"><h2>月度收支</h2><p class="finance-muted">{{ summary.currency }} · 点击月份查看账目；仅显示有记录的月份。双色条共用比例尺，负金额保留数值。</p><div class="finance-trend-legend"><span class="income-key">收入</span><span class="expense-key">支出</span><span>比例尺上限 {{ financeMoney(trendMax.toString()) }}</span></div><div class="finance-trend"><button v-for="m in summary.months" :key="m.month" class="finance-month" type="button" :aria-label="m.month + ' 月账目'" @click="openMonth(m.month)"><span>{{ m.month }}<small v-if="monthIncomplete(m.month)"> 期间未结束</small></span><div class="finance-bars" aria-hidden="true"><i :style="{ width: barWidth(m.income_minor) }" class="income" /><i :style="{ width: barWidth(m.expense_minor) }" class="expense" /></div><span class="finance-month-values"><span>收入 <b>{{ financeMoney(m.income_minor) }}</b></span><span>支出 <b>{{ financeMoney(m.expense_minor) }}</b></span></span><span class="finance-month-difference" :class="{ negative: BigInt(m.income_minor) < BigInt(m.expense_minor) }"><small>收支差额</small><strong>{{ financeMoney(monthDifference(m)) }}</strong><small v-if="BigInt(m.income_minor) < BigInt(m.expense_minor)">支出高于收入</small></span></button></div></section>
        </template>
      </template>
      <template v-else-if="tab === 'analysis' && summary && !error">
        <h2>分类支出</h2><p class="finance-muted">{{ periodLabel }} · {{ summary.currency }} · 各层级占比均以本期间、本币种总支出为分母，原始负金额保留正负。</p>
        <EmptyState v-if="!summary.categories.length" title="这个期间没有支出分类" description="调整日期或币种查看其他记录。" />
        <div class="finance-rank"><section v-for="a in summary.categories" :key="a.name">
          <button type="button" class="finance-rank-row" :aria-expanded="expanded.includes(a.name)" @click="toggleCategory(a.name)"><strong class="finance-rank-name"><span aria-hidden="true">{{ expanded.includes(a.name) ? '⌄' : '›' }}</span> {{ a.name || '未分类' }}<i class="finance-ratio" :style="{ width: shareWidth(a.amount_minor) }" /></strong><span>{{ a.count }} 笔</span><b>{{ financeMoney(a.amount_minor) }}</b><span>{{ financeShare(a.amount_minor, summary.expense_minor) }}</span></button>
          <div v-if="expanded.includes(a.name)" class="finance-children"><el-button text @click="drill({ category: a.name, category_empty: a.name ? '' : 'true', kind: 'expense' })">查看该分类全部账目</el-button><button v-for="child in a.children" :key="child.name" class="finance-rank-row finance-child" type="button" @click="drill({ category: a.name, category_empty: a.name ? '' : 'true', subcategory: child.name, subcategory_empty: child.name ? '' : 'true', kind: 'expense' })"><span class="finance-rank-name">{{ child.name || '无二级分类' }}</span><span>{{ child.count }} 笔</span><b>{{ financeMoney(child.amount_minor) }}</b><span>{{ financeShare(child.amount_minor, summary.expense_minor) }}</span></button></div>
        </section></div>
        <h2>标签支出</h2><p class="finance-muted">标签金额是命中该标签的支出账单金额。同一交易可能命中多个标签，各标签金额相加可能超过实际总支出；无标签账单仍计入总支出。</p><div class="finance-rank"><button v-for="a in summary.tags" :key="a.name" type="button" class="finance-rank-row finance-tag-row" @click="drill({ tag: a.name, kind: 'expense' })"><strong class="finance-rank-name">#{{ a.name }}</strong><span>{{ a.count }} 笔</span><b>{{ financeMoney(a.amount_minor) }}</b><span>查看账目 ›</span></button></div><p v-if="!summary.tags.length" class="finance-muted">这个期间没有带标签的支出。</p>
      </template>
      <template v-else-if="tab === 'transactions' && ledgerID && !error">
        <el-button v-if="drillReturn" text @click="returnToAnalysis">← 返回{{ drillReturn.tab === 'analysis' ? '分类与标签' : '财务概览' }}</el-button>
        <p class="finance-muted">{{ pageData?.total ?? 0 }} 笔 · {{ periodLabel }} · 原始账目只读，修改请到钱迹操作后重新导入。</p>
        <div class="finance-table-wrap"><el-table :data="pageData?.items ?? []" @row-click="openTransaction" row-class-name="finance-clickable" empty-text="没有匹配的账目"><el-table-column label="日期 / 时间" width="155"><template #default="{ row }">{{ row.occurred_at.slice(0, 16) }}</template></el-table-column><el-table-column label="金额" min-width="165" align="right"><template #default="{ row }">{{ financeMoney(row.amount_minor) }} {{ row.currency }}</template></el-table-column><el-table-column label="类型" width="80"><template #default="{ row }">{{ kindLabel(row.kind) }}</template></el-table-column><el-table-column label="分类" width="165" show-overflow-tooltip><template #default="{ row }">{{ row.category }}<span v-if="row.subcategory"> / {{ row.subcategory }}</span></template></el-table-column><el-table-column prop="note" label="原始备注" min-width="240" show-overflow-tooltip /><el-table-column label="标签" width="170" show-overflow-tooltip><template #default="{ row }">{{ row.tags.map((t: string) => '#' + t).join(' ') }}</template></el-table-column><el-table-column width="75"><template #default="{ row }"><el-button text @click.stop="openTransaction(row)">查看</el-button></template></el-table-column></el-table></div>
        <el-pagination v-if="pageData && pageData.total > 30" v-model:current-page="page" layout="prev, pager, next" :page-size="30" :total="pageData.total" aria-label="交易分页" />
      </template>
      <template v-else-if="tab === 'imports'">
        <section class="finance-import"><h2>导入钱迹 CSV</h2><p class="finance-muted">UTF-8 / BOM · 钱迹原始 18 列 · 最大 10 MB、50000 条。数据上传至你的自托管服务器，不发送给 AI。</p>
          <el-form label-position="top" @submit.prevent="preview"><el-form-item label="来源账本"><el-input v-model="ledgerName" :disabled="busy" maxlength="120" placeholder="例如：钱迹个人账本；同一账本每次使用相同名称" /></el-form-item><el-form-item label="CSV 文件"><input ref="fileInput" :disabled="busy" type="file" accept=".csv,text/csv" aria-label="选择钱迹 CSV" @change="chooseFile" /></el-form-item>
          <el-checkbox v-model="complete" :disabled="busy">这是明确日期范围内的完整导出</el-checkbox><p class="finance-muted">默认按增量导入，缺失交易保留。只有你确认该范围无筛选且完整导出，才检查缺失 ID。</p><el-date-picker v-if="complete" v-model="scope" :disabled="busy" type="daterange" value-format="YYYY-MM-DD" start-placeholder="完整范围开始" end-placeholder="完整范围结束" aria-label="完整导出范围" />
          <div class="finance-actions"><el-button type="primary" :loading="busy" :disabled="!file || !ledgerName.trim() || (complete && !scope?.length)" @click="preview">校验并预览</el-button></div></el-form>
        </section>
        <section v-if="previewData" class="finance-section"><h2>导入预览</h2><p>文件 {{ previewData.parse.total }} 条 · 收入 {{ previewData.parse.income }} · 支出 {{ previewData.parse.expense }} · 转账 {{ previewData.parse.transfer }}</p><p v-if="previewData.parse.invalid">异常记录 {{ previewData.parse.invalid }} 条；需修正后重新预览。</p><p v-if="previewData.parse.from">日期 {{ previewData.parse.from }} 至 {{ previewData.parse.to }}</p>
          <el-alert v-if="previewData.parse.errors.length" title="文件校验失败，尚未导入。最多展示 100 条错误。" type="error" :closable="false" /><ul v-if="previewData.parse.errors.length" class="finance-errors"><li v-for="(e, i) in previewData.parse.errors" :key="i">第 {{ e.line }} 行：{{ e.message }}</li></ul>
          <template v-if="previewData.batch"><p>新增 {{ previewData.batch.added }} · 更新 {{ previewData.batch.updated }} · 未变化 {{ previewData.batch.unchanged }} · {{ previewData.batch.complete_scope ? '范围内缺失 ' + previewData.batch.missing : '缺失：未核验（增量导出）' }}</p><p v-for="w in previewData.warnings" :key="w" class="finance-muted">{{ w }}</p><div class="finance-preview-rows"><p v-for="a in previewData.sample" :key="a.external_id">{{ a.occurred_at }} · {{ kindLabel(a.kind) }} {{ financeMoney(a.amount_minor) }} {{ a.currency }} · {{ a.category }} · {{ a.note }}</p></div><el-checkbox v-if="previewData.batch.missing" v-model="removeMissing" :disabled="busy">我已核对导出范围完整、交易 ID 稳定，移除范围内缺失的旧交易</el-checkbox><div class="finance-actions"><el-button type="primary" :loading="busy" @click="commit">确认导入</el-button><span class="finance-muted">可撤销；预览有效期 24 小时，其他导入后需重新预览。</span></div></template>
        </section>
        <section v-if="ledgerID" class="finance-section"><h2>导入批次</h2><p class="finance-muted">按批次倒序撤销，恢复被新增、更新或移除的交易。每个来源账本分别保留批次历史。</p><div v-for="b in batches" :key="b.id" class="finance-batch"><div><strong>{{ b.filename }}</strong><p class="finance-muted">{{ localTime(b.imported_at || b.created_at) }} · {{ batchLabel(b.status) }} · 新增 {{ b.added }} / 更新 {{ b.updated }} / 未变 {{ b.unchanged }} / 缺失 {{ b.complete_scope ? b.missing : '未核验' }} / 移除 {{ b.removed }}</p><small v-if="b.complete_scope">完整范围声明：{{ b.scope_from }} 至 {{ b.scope_to }}</small></div><el-button v-if="b.status === 'imported'" text :disabled="busy || b.id !== latestImported" @click="undo(b)">撤销批次</el-button></div></section>
      </template>
    </div>
    <el-dialog v-model="detailOpen" title="原始账目" width="650px" align-center class="finance-dialog">
      <template v-if="detail"><h2>{{ kindLabel(detail.kind) }} {{ financeMoney(detail.amount_minor) }} {{ detail.currency }}</h2><p class="finance-muted">{{ detail.occurred_at }} · 来源：钱迹 / {{ detail.source }} · 最近写入批次 #{{ detail.batch_id }}</p><p class="finance-muted">只读档案。需要修改时，请到钱迹编辑并重新导入。</p><dl class="finance-raw"><template v-for="key in financeHeaders" :key="key"><template v-if="detail.raw?.[key]"><dt>{{ key }}</dt><dd>{{ detail.raw[key] }}</dd></template></template></dl><p class="finance-muted">首次入库 {{ localTime(detail.created_at) }} · 最近更新 {{ localTime(detail.updated_at) }}</p></template>
      <template #footer><el-button @click="detailOpen = false">关闭</el-button></template>
    </el-dialog>
  </section>
</template>
<script setup lang="ts">
import { useRoute, useRouter, type LocationQueryRaw } from 'vue-router'
import { nextTick, computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useArchiveScroll } from '@/composables/useArchiveScroll'
import PageHeader from '@/components/PageHeader.vue'
import AssetArchive from '@/components/finance/AssetArchive.vue'
import EmptyState from '@/components/EmptyState.vue'
import { financeHeaders, financeMoney, financeShare, getSummary, getTransaction, importFinance, kindLabel, listBatches, listLedgers, listTransactions, monthIncomplete, previewFinance, undoFinance, type Batch, type Ledger, type Preview, type Summary, type Transaction, type TransactionPage } from '@/api/finance'
import '@/life.css'
import '@/finance.css'
const route = useRoute(), router = useRouter()
const restoreScroll = useArchiveScroll()
const lifeReturn = computed(() => ({ path: '/life', query: { goal: typeof route.query.return_goal === 'string' ? route.query.return_goal : undefined, view: route.query.life_view === 'goals' ? 'goals' : undefined, domain: typeof route.query.life_domain === 'string' ? route.query.life_domain : undefined, milestones: route.query.life_milestones === 'true' ? 'true' : undefined } }))
const noCategory = '\u0000'
const tabs = [['overview', '财务概览'], ['analysis', '分类与标签'], ['transactions', '账目明细'], ['assets', '资产档案'], ['imports', '数据导入']] as const
const tab = ref<typeof tabs[number][0]>('overview'), ledgers = ref<Ledger[]>([]), ledgerID = ref<number>(), summary = ref<Summary>(), pageData = ref<TransactionPage>(), batches = ref<Batch[]>([]), error = ref(''), loading = ref(false), busy = ref(false)
const year = ref(''), month = ref(''), currency = ref(''), dateRange = ref<string[] | null>(null), page = ref(1), keywordInput = ref('')
const filters = reactive({ kind: '', category: '', subcategory: '', tag: '', keyword: '', category_empty: '', subcategory_empty: '' })
const file = ref<File>(), fileInput = ref<HTMLInputElement>(), ledgerName = ref('钱迹个人账本'), complete = ref(false), scope = ref<string[] | null>(null), previewData = ref<Preview>(), removeMissing = ref(false)
const detail = ref<Transaction>(), detailOpen = ref(false)
const advancedOpen = ref(false), expanded = ref<string[]>([])
const drillReturn = ref<{ tab: string; query: LocationQueryRaw; scroll: number }>()
let sequence = 0, detailSequence = 0, restoring = false, ready = false
const message = (e: unknown) => e instanceof Error ? e.message : '操作失败，请重试。'
const localTime = (s: string) => new Date(s).toLocaleString('zh-CN')
const batchLabel = (s: string) => ({ preview: '待确认', imported: '已导入', undone: '已撤销' })[s as 'preview'] ?? s
const latestBatch = computed(() => batches.value.find(b => b.status === 'imported'))
const latestImported = computed(() => batches.value.find(b => b.status === 'imported')?.id)
const periodLabel = computed(() => dateRange.value?.length === 2 ? dateRange.value.join(' 至 ') : year.value ? `${year.value} 年${month.value ? ' ' + Number(month.value) + ' 月' : ''}` : '全部时间')
const coverageLabel = computed(() => summary.value?.coverage_from ? `账本实际覆盖 ${summary.value.coverage_from} 至 ${summary.value.coverage_to} · 数据截至 ${summary.value.coverage_to}（最新账目日期）` : '账本尚无记录')
const completenessLabel = computed(() => {
  if (!year.value && !dateRange.value?.length) return '全部时间包含已导入账目；实际覆盖日期不代表连续记录或完整导出。'
  const end = dateRange.value?.[1] ?? (year.value + '-' + (month.value || '12'))
  return monthIncomplete(end.slice(0, 7)) ? '所选期间尚未结束，数据可能继续增加；导出是否完整需自行核对。' : '所选期间已结束；自然月结束不保证无漏账，请核对钱迹导出范围。'
})
const selectedLedger = computed(() => ledgers.value.find(l => l.id === ledgerID.value)?.name ?? '')
const activeFilters = computed(() => [filters.kind && kindLabel(filters.kind), filters.category_empty && '仅未分类账目', filters.category && filters.category !== noCategory && `分类：${filters.category}`, filters.subcategory_empty && '仅无二级分类账目', filters.subcategory && `二级：${filters.subcategory}`, filters.tag && `标签：#${filters.tag}`, filters.keyword && `备注：${filters.keyword}`].filter(Boolean) as string[])
const hasFilters = computed(() => !!(year.value || dateRange.value?.length || activeFilters.value.length))
const emptySubcategory = computed({ get: () => filters.subcategory_empty === 'true', set: value => { filters.subcategory_empty = value ? 'true' : ''; if (value) filters.subcategory = '' } })
const shareWidth = (value: string) => { const t = BigInt(summary.value?.expense_minor ?? '0'), v = BigInt(value); return `${t > 0n && v > 0n ? Number((v > t ? t : v) * 10000n / t) / 100 : 0}%` }
const monthDifference = (m: Summary['months'][number]) => (BigInt(m.income_minor) - BigInt(m.expense_minor)).toString()
function toggleCategory(name: string) { expanded.value = expanded.value.includes(name) ? expanded.value.filter(n => n !== name) : [...expanded.value, name] }
const trendMax = computed(() => (summary.value?.months ?? []).reduce((max, m) => [BigInt(m.income_minor), BigInt(m.expense_minor), max].reduce((a, b) => a > b ? a : b), 1n))
const barWidth = (s: string) => `${Number((BigInt(s) > 0n ? BigInt(s) : 0n) * 10000n / trendMax.value) / 100}%`
function params(details = false) { const p = new URLSearchParams(); if (currency.value) p.set('currency', currency.value); if (dateRange.value?.length === 2) { p.set('from', dateRange.value[0]!); p.set('to', dateRange.value[1]!) } else if (year.value) { p.set('from', `${year.value}-${month.value || '01'}-01`); const m = Number(month.value || 12); const days = new Date(Number(year.value), m, 0).getDate(); p.set('to', `${year.value}-${month.value || '12'}-${days}`) }; if (details) { for (const [k, v] of Object.entries(filters)) if (v && !(k === 'category' && v === noCategory)) p.set(k, v); p.set('page', String(page.value)); p.set('page_size', '30') }; return p }
async function refresh() { if (tab.value === 'assets') { ++sequence; loading.value = false; return }; const id = ledgerID.value; if (!id) { try { await loadLedgers(); if (ledgerID.value) await refresh() } catch (e) { error.value = message(e) }; return; }; const token = ++sequence; loading.value = true; error.value = ''; const summaryParams = params(), listParams = params(true); try { const [s, rows, bs] = await Promise.all([getSummary(id, summaryParams), listTransactions(id, listParams), listBatches(id)]); if (token !== sequence) return; summary.value = s; pageData.value = rows; batches.value = bs; void restoreScroll(); if (!currency.value && s.currencies.length) currency.value = s.currencies.includes('CNY') ? 'CNY' : s.currencies[0]! } catch (e) { if (token === sequence) error.value = message(e) } finally { if (token === sequence) loading.value = false } }
async function loadLedgers() { const ls = await listLedgers(); ledgers.value = ls; if (!ls.some(l => l.id === ledgerID.value)) ledgerID.value = ls[0]?.id }
function chooseFile(e: Event) { file.value = (e.target as HTMLInputElement).files?.[0]; previewData.value = undefined }
watch([ledgerName, complete, scope], () => { previewData.value = undefined; removeMissing.value = false }, { deep: true })
async function preview() { if (busy.value || !file.value) return; busy.value = true; error.value = ''; previewData.value = undefined; removeMissing.value = false; const form = new FormData(); form.set('file', file.value); form.set('ledger_name', ledgerName.value.trim()); form.set('complete_scope', String(complete.value)); if (complete.value && scope.value?.length === 2) { form.set('scope_from', scope.value[0]!); form.set('scope_to', scope.value[1]!) }; try { previewData.value = await previewFinance(form); await loadLedgers(); const l = ledgers.value.find(l => l.name === ledgerName.value.trim()); if (l) ledgerID.value = l.id; await refresh() } catch (e) { error.value = message(e) } finally { busy.value = false } }
async function commit() { const b = previewData.value?.batch; if (!b || busy.value) return; if (removeMissing.value) { try { await ElMessageBox.confirm(`移除 ${b.missing} 条缺失交易？请确认这份文件是 ${b.scope_from} 至 ${b.scope_to} 的完整导出，且钱迹交易 ID 未变化。`, '确认移除缺失账目', { confirmButtonText: '确认完整并移除', cancelButtonText: '取消', type: 'warning' }) } catch { return } }; busy.value = true; error.value = ''; try { await importFinance(b.id, removeMissing.value); previewData.value = undefined; ElMessage.success('账本已保存'); await loadLedgers(); await refresh() } catch (e) { error.value = message(e) } finally { busy.value = false } }
async function undo(b: Batch) { if (busy.value) return; try { await ElMessageBox.confirm(`撤销「${b.filename}」并恢复本批次影响的交易？`, '撤销导入', { confirmButtonText: '撤销批次', cancelButtonText: '取消' }) } catch { return }; busy.value = true; try { await undoFinance(b.id); previewData.value = undefined; ElMessage.success('已撤销'); await refresh() } catch (e) { error.value = message(e) } finally { busy.value = false } }
function clearDetail() { Object.assign(filters, { kind: '', category: '', subcategory: '', tag: '', keyword: '', category_empty: '', subcategory_empty: '' }); keywordInput.value = ''; page.value = 1 }
function drill(f: Partial<typeof filters>) { restoring = true; drillReturn.value = { tab: tab.value, query: currentQuery(), scroll: document.querySelector('.main-content')?.scrollTop ?? 0 }; clearDetail(); Object.assign(filters, f); tab.value = 'transactions'; void navigate(currentQuery()) }
function openMonth(m: string) { restoring = true; drillReturn.value = { tab: tab.value, query: currentQuery(), scroll: document.querySelector('.main-content')?.scrollTop ?? 0 }; dateRange.value = null; year.value = m.slice(0, 4); month.value = m.slice(5); clearDetail(); tab.value = 'transactions'; void navigate(currentQuery()) }
function search() { filters.keyword = keywordInput.value.trim(); page.value = 1 }
async function openTransaction(a: Transaction) { const id = ledgerID.value, token = ++detailSequence; if (!id) return; try { const d = await getTransaction(id, a.id); if (token === detailSequence && ledgerID.value === id) { detail.value = d; detailOpen.value = true } } catch (e) { error.value = message(e) } }
function clearAll() { clearDetail(); year.value = ''; month.value = ''; dateRange.value = null }
function currentQuery(): LocationQueryRaw {
  const q: LocationQueryRaw = { ...route.query, tab: tab.value, ledger: ledgerID.value ? String(ledgerID.value) : undefined, year: year.value || undefined, month: month.value || undefined, currency: currency.value || undefined, from: dateRange.value?.[0], to: dateRange.value?.[1], page: page.value > 1 ? String(page.value) : undefined }
  for (const [key, value] of Object.entries(filters)) q[key] = value && value !== noCategory ? value : undefined
  return q
}
function readQuery() {
  restoring = true
  const str = (key: string) => typeof route.query[key] === 'string' ? route.query[key] as string : ''
  const t = str('tab'); tab.value = tabs.some(a => a[0] === t) ? t as typeof tab.value : 'overview'
  const id = Number(str('ledger')); if (id > 0 && (!ready || ledgers.value.some(l => l.id === id))) ledgerID.value = id
  year.value = /^\d{4}$/.test(str('year')) ? str('year') : ''; month.value = /^(0[1-9]|1[0-2])$/.test(str('month')) && year.value ? str('month') : ''
  currency.value = str('currency'); dateRange.value = str('from') && str('to') ? [str('from'), str('to')] : null
  for (const key of Object.keys(filters) as (keyof typeof filters)[]) filters[key] = str(key)
  if (filters.category_empty === 'true') filters.category = noCategory
  keywordInput.value = filters.keyword; page.value = Math.max(1, Math.floor(Number(str('page')) || 1))
  void nextTick(() => { restoring = false })
}
async function navigate(query: LocationQueryRaw) { restoring = true; try { await router.push({ query }) } finally { await nextTick(); restoring = false } }
function switchTab(value: typeof tab.value) { restoring = true; tab.value = value; void navigate(currentQuery()) }
async function returnToAnalysis() { const target = drillReturn.value; if (!target) return; await navigate(target.query); await nextTick(); document.querySelector('.main-content')?.scrollTo({ top: target.scroll }) }
readQuery()
watch(ledgerID, () => { ++detailSequence; detailOpen.value = false; summary.value = undefined; pageData.value = undefined; batches.value = []; if (!restoring && ready) { currency.value = ''; year.value = ''; month.value = ''; dateRange.value = null; clearDetail() } }, { flush: 'sync' })
watch(() => filters.category, value => { filters.category_empty = value === noCategory ? 'true' : ''; if (!restoring) { filters.subcategory = ''; filters.subcategory_empty = '' } }, { flush: 'sync' })
watch(() => filters.subcategory, value => { if (value) filters.subcategory_empty = '' }, { flush: 'sync' })
watch(year, () => { if (!year.value) month.value = '' })
watch([ledgerID, year, month, currency, dateRange, filters], () => { if (restoring || !ready) return; page.value = 1; void router.replace({ query: currentQuery() }); void refresh() }, { deep: true, flush: 'post' })
watch(page, () => { if (restoring || !ready) return; void router.replace({ query: currentQuery() }); void refresh() }, { flush: 'post' })
watch(() => route.fullPath, () => { readQuery(); if (ready) void refresh() })
onMounted(async () => { try { await loadLedgers(); await refresh(); ready = true; if (ledgerID.value) { await router.replace({ query: currentQuery() }); await refresh() } } catch (e) { ready = true; error.value = message(e) } })
onBeforeUnmount(() => { ++sequence; ++detailSequence })
</script>
