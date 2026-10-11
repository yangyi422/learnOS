import { expect, test, type Page } from '@playwright/test'
import type { Batch, Summary, Transaction } from '../src/api/finance'
const time = '2026-10-09T02:00:00Z'
async function mockFinance(page: Page, empty = false, long = false) {
  const transaction: Transaction = { id: 1, external_id: 'sample-id', occurred_at: '2026-09-02 12:00:00', kind: 'expense', amount_minor: '1234', currency: 'CNY', category: '餐饮', subcategory: '', note: '午餐，原始备注\n第二行', tags: ['日常', '工作'], source: '钱迹个人账本', raw: { ID: 'sample-id', 时间: '2026-09-02 12:00:00', 类型: '支出', 金额: '12.34', 币种: 'CNY', 分类: '餐饮', 备注: '午餐，原始备注\n第二行', 标签: '#日常#工作' }, batch_id: 1, created_at: time, updated_at: time }
  if (long) { transaction.category = '餐饮与家庭共同生活开销'.repeat(5); transaction.note = '保留完整的长备注。'.repeat(100); transaction.tags = ['日常与跨领域的长期分类名称'.repeat(5), '工作'] }
  const batch: Batch = { id: 1, ledger_id: 1, filename: 'sample.csv', status: 'preview', added: 1, updated: 0, unchanged: 0, missing: 0, removed: 0, row_count: 1, complete_scope: false, scope_from: '', scope_to: '', created_at: time, imported_at: null, undone_at: null }
  const state = { imported: !empty, previewed: false, failedImport: false, failedPreview: false, importCalls: 0, undoCalls: 0, uploadedContentType: '', queries: [] as URLSearchParams[] }
  await page.route('**/api/v1/**', async route => {
    const req = route.request(), url = new URL(req.url()), p = url.pathname
    const reply = (data: unknown) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data }) })
    const fail = () => route.fulfill({ status: 500, contentType: 'application/json', body: JSON.stringify({ error: { code: 'TEST_FAIL', message: '保存失败，请重试', retryable: true } }) })
    if (p === '/api/v1/auth/me') return reply({ id: 1, username: 'test', role: 'admin', status: 'active' })
    if (p === '/api/v1/finance/ledgers') return reply(state.imported || state.previewed ? [{ id: 1, name: '钱迹个人账本', revision: 1 }] : [])
    if (p === '/api/v1/finance/preview') { state.uploadedContentType = req.headers()['content-type']!; if (state.failedPreview) return fail(); state.previewed = true; return reply({ batch, parse: { total: 1, income: 0, expense: 1, transfer: 0, from: '2026-09-02', to: '2026-09-02', errors: [] }, sample: [transaction], warnings: ['转账不计入收支'] }) }
    if (p.endsWith('/import')) { state.importCalls++; if (state.failedImport) return fail(); state.imported = true; batch.status = 'imported'; batch.imported_at = time; return reply(batch) }
    if (p.endsWith('/undo')) { state.undoCalls++; state.imported = false; batch.status = 'undone'; return reply(batch) }
    if (p.endsWith('/batches')) return reply(state.imported || state.previewed ? [batch] : [])
    if (p.endsWith('/summary')) { const s: Summary = { count: state.imported ? 1 : 0, income_minor: '0', expense_minor: state.imported ? '1234' : '0', difference_minor: state.imported ? '-1234' : '0', currency: 'CNY', coverage_from: state.imported ? '2026-09-02' : '', coverage_to: state.imported ? '2026-09-02' : '', last_import: state.imported ? time : null, years: ['2026'], currencies: ['CNY'], all_tags: transaction.tags, all_categories: [transaction.category], months: state.imported ? [{ month: '2026-09', income_minor: '0', expense_minor: '1234', count: 1 }] : [], categories: [{ name: transaction.category, amount_minor: '1234', count: 1, children: [{ name: '', amount_minor: '1234', count: 1 }] }], tags: [{ name: transaction.tags[0]!, amount_minor: '1234', count: 1 }, { name: '工作', amount_minor: '1234', count: 1 }] }; return reply(s) }
    if (p.endsWith('/transactions')) { state.queries.push(url.searchParams); return reply({ items: state.imported ? [transaction] : [], total: state.imported ? 1 : 0, page: 1, page_size: 30 }) }
    if (p.endsWith('/transactions/1')) return reply(transaction)
    return reply([])
  })
  return state
}
test('空财务档案引导导入，失败保留文件与预览，重复点击受保护', async ({ page }) => {
  const state = await mockFinance(page, true); await page.goto('/life/finance'); await expect(page.getByText('还没有财务历史')).toBeVisible(); await expect(page.getByText('收支差额', { exact: true })).toHaveCount(0)
  await page.getByRole('button', { name: '导入账本', exact: true }).click(); await page.getByLabel('选择钱迹 CSV').setInputFiles({ name: 'sample.csv', mimeType: 'text/csv', buffer: Buffer.from('synthetic test CSV') })
  state.failedPreview = true; await page.getByRole('button', { name: '校验并预览' }).click(); await expect(page.getByText('保存失败，请重试', { exact: true })).toBeVisible(); expect(await page.getByLabel('选择钱迹 CSV').evaluate((input: HTMLInputElement) => input.files?.length)).toBe(1)
  state.failedPreview = false; await page.getByRole('button', { name: '校验并预览' }).click(); await expect(page.getByText('文件 1 条 · 收入 0 · 支出 1 · 转账 0')).toBeVisible(); expect(state.uploadedContentType).toContain('multipart/form-data; boundary=')
  state.failedImport = true; await page.getByRole('button', { name: '确认导入', exact: true }).click(); await expect(page.getByRole('heading', { name: '导入预览' })).toBeVisible()
  state.failedImport = false; await page.getByRole('button', { name: '确认导入', exact: true }).click({ clickCount: 2 }); await expect(page.getByRole('heading', { name: '导入预览' })).toHaveCount(0); expect(state.importCalls).toBe(2)
  await page.getByRole('tab', { name: '财务概览' }).click(); await expect(page.locator('.finance-totals').getByText('12.34', { exact: true })).toBeVisible(); await page.reload(); await expect(page.locator('.finance-totals').getByText('12.34', { exact: true })).toBeVisible()
})
test('分类标签可追溯原始账目，无二级分类筛选准确且详情只读', async ({ page }) => {
  const state = await mockFinance(page); await page.goto('/life/finance'); await page.getByRole('tab', { name: '分类与标签' }).click(); await page.getByRole('button', { name: /餐饮.*1 笔/ }).click(); await page.getByRole('button', { name: /无二级分类/ }).click(); await expect(page.getByText('仅无二级分类账目')).toBeVisible(); await expect.poll(() => state.queries.some(p => p.get('subcategory_empty') === 'true' && p.get('category') === '餐饮')).toBe(true)
  await page.getByRole('button', { name: '查看', exact: true }).click(); const d = page.getByRole('dialog'); await expect(d.getByText('午餐，原始备注\n第二行', { exact: true })).toBeVisible(); await expect(d.locator('input, textarea')).toHaveCount(0); await expect(d.getByText('钱迹 / 钱迹个人账本', { exact: false })).toBeVisible(); await d.getByRole('button', { name: '关闭', exact: true }).click(); await page.getByRole('tab', { name: '分类与标签' }).click(); await page.getByRole('button', { name: /#日常/ }).click(); await expect.poll(() => state.queries.some(p => p.get('tag') === '日常')).toBe(true)
})
test('导入批次撤销需要确认并即时刷新', async ({ page }) => {
  const state = await mockFinance(page); await page.goto('/life/finance'); await page.getByRole('tab', { name: '数据导入' }).click(); await page.getByLabel('选择钱迹 CSV').setInputFiles({ name: 'sample.csv', mimeType: 'text/csv', buffer: Buffer.from('synthetic') }); await page.getByRole('button', { name: '校验并预览' }).click(); await page.getByRole('button', { name: '确认导入', exact: true }).click(); await page.getByRole('button', { name: '撤销批次', exact: true }).click(); await page.getByRole('dialog', { name: '撤销导入' }).getByRole('button', { name: '取消', exact: true }).click(); await expect(page.getByRole('dialog', { name: '撤销导入' })).toBeHidden(); expect(state.undoCalls).toBe(0); await page.getByRole('button', { name: '撤销批次', exact: true }).click(); await page.getByRole('dialog').getByRole('button', { name: '撤销批次', exact: true }).click(); await expect(page.getByText('已撤销', { exact: false }).first()).toBeVisible(); expect(state.undoCalls).toBe(1)
})
for (const width of [1440, 768, 390]) test(`财务档案 ${width}px 概览与长原始备注详情可读`, async ({ page }) => {
  await mockFinance(page); await page.setViewportSize({ width, height: 850 }); await page.goto('/life/finance'); await expect(page.getByText('月度收支', { exact: true })).toBeVisible(); expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
  if (process.env.CAPTURE_VISUALS) await page.screenshot({ path: `test-results/finance-overview-${width}.png`, fullPage: true, animations: 'disabled' })
  await page.getByRole('tab', { name: '账目明细' }).click(); await page.getByRole('button', { name: '查看', exact: true }).click(); const dialog = page.getByRole('dialog'); await expect(dialog.getByRole('button', { name: '关闭', exact: true })).toBeVisible(); const box = await dialog.boundingBox(); expect(box!.width).toBeLessThanOrEqual(width)
  if (process.env.CAPTURE_VISUALS) await page.screenshot({ path: `test-results/finance-detail-${width}.png`, animations: 'disabled' })
})


test('月度下钻、URL 恢复、清除筛选与返回概览保持期间', async ({ page }) => {
  const state = await mockFinance(page)
  await page.goto('/life/finance')
  await expect(page.getByText('全部时间包含已导入账目', { exact: false })).toBeVisible()
  await page.getByRole('button', { name: '2026-09 月账目' }).click()
  await expect.poll(() => state.queries.at(-1)?.get('from')).toBe('2026-09-01')
  await expect.poll(() => state.queries.at(-1)?.get('to')).toBe('2026-09-30')
  await expect(page).toHaveURL(/tab=transactions/)
  await page.reload()
  await expect(page.getByRole('tab', { name: '账目明细' })).toHaveAttribute('aria-selected', 'true')
  await expect(page.getByText('2026 年 9 月', { exact: false }).first()).toBeVisible()
  await page.getByRole('button', { name: '清除全部筛选' }).click()
  await expect.poll(() => state.queries.at(-1)?.get('from')).toBe(null)
  await expect(page.getByText('全部时间', { exact: false }).first()).toBeVisible()
  await page.getByRole('tab', { name: '财务概览' }).click()
  await page.getByRole('button', { name: '2026-09 月账目' }).click()
  await page.getByRole('button', { name: '← 返回财务概览' }).click()
  await expect(page.getByRole('tab', { name: '财务概览' })).toHaveAttribute('aria-selected', 'true')
  await expect(page.getByText('全部时间包含已导入账目', { exact: false })).toBeVisible()
})

test('未分类和标签 URL 条件、高级筛选、浏览器返回与生活视图恢复', async ({ page }) => {
  const state = await mockFinance(page)
  await page.goto('/life/finance?tab=transactions&ledger=1&category_empty=true&subcategory_empty=true&kind=expense&page=2&life_view=goals&life_domain=career')
  await expect.poll(() => state.queries.at(-1)?.get('page')).toBe('2')
  await expect.poll(() => state.queries.some(p => p.get('category_empty') === 'true' && p.get('subcategory_empty') === 'true' && p.get('page') === '2')).toBe(true)
  await expect(page.getByText('仅未分类账目', { exact: true })).toBeVisible()
  await page.getByRole('button', { name: '更多筛选' }).click()
  await expect(page.getByRole('combobox', { name: '来源账本' })).toBeVisible()
  await page.getByRole('tab', { name: '分类与标签' }).click()
  await page.getByRole('button', { name: /#日常/ }).click()
  await expect.poll(() => state.queries.at(-1)?.get('tag')).toBe('日常')
  await page.goBack()
  await expect(page.getByRole('tab', { name: '分类与标签' })).toHaveAttribute('aria-selected', 'true')
  await page.getByRole('link', { name: '返回生活' }).click()
  await expect(page.getByRole('tab', { name: '长期目标' })).toHaveAttribute('aria-selected', 'true')
  await expect(page).toHaveURL(/view=goals/)
})


test('旧请求不覆盖新筛选，读取失败可重试', async ({ page }) => {
  await mockFinance(page)
  let delay = false, fail = false
  await page.route('**/finance/ledgers/1/transactions?**', async route => {
    const url = new URL(route.request().url())
    if (fail) return route.fulfill({ status: 500, contentType: 'application/json', body: JSON.stringify({ error: { code: 'TEST', message: '读取失败，请重试', retryable: true } }) })
    if (!delay) return route.fallback()
    const income = url.searchParams.get('kind') === 'income'
    if (income) await new Promise(resolve => setTimeout(resolve, 500))
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: { items: [], total: income ? 77 : 88, page: 1, page_size: 30 } }) })
  })
  await page.goto('/life/finance?tab=transactions')
  await expect(page.getByText('1 笔 · 全部时间', { exact: false })).toBeVisible()
  delay = true
  await page.locator('.el-select').filter({ has: page.getByRole('combobox', { name: '收支类型' }) }).click()
  await page.getByRole('option', { name: '收入', exact: true }).click()
  await page.locator('.el-select').filter({ has: page.getByRole('combobox', { name: '收支类型' }) }).click()
  await page.getByRole('option', { name: '支出', exact: true }).click()
  await expect(page.getByText('88 笔 · 全部时间', { exact: false })).toBeVisible()
  await page.waitForTimeout(600)
  await expect(page.getByText('77 笔 · 全部时间', { exact: false })).toHaveCount(0)
  fail = true
  await page.getByRole('button', { name: '清除全部筛选' }).click()
  await expect(page.getByText('读取失败，请重试', { exact: true })).toBeVisible()
  fail = false
  await page.getByRole('button', { name: '重试读取' }).click()
  await expect(page.getByText('88 笔 · 全部时间', { exact: false })).toBeVisible()
})

for (const width of [1440, 390]) test(`分类分析与高级筛选 ${width}px 保持宽度和对齐`, async ({ page }) => {
  await mockFinance(page, false, true)
  await page.setViewportSize({ width, height: 850 })
  await page.goto('/life/finance?tab=analysis')
  await page.getByRole('button', { name: /餐饮.*1 笔/ }).click()
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
  if (process.env.CAPTURE_VISUALS) await page.screenshot({ path: `test-results/finance-analysis-${width}.png`, fullPage: true, animations: 'disabled' })
  await page.getByRole('button', { name: /无二级分类/ }).click()
  await page.getByRole('button', { name: '更多筛选' }).click()
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
  if (process.env.CAPTURE_VISUALS) await page.screenshot({ path: `test-results/finance-filters-${width}.png`, fullPage: true, animations: 'disabled' })
})
