import { expect, test, type Page } from '@playwright/test'
import { assetTotals, type AssetForm, type AssetSnapshot } from '../src/api/assets'
const time = '2026-10-10T02:00:00Z'
function snapshot(id: number, date: string, amounts: Partial<AssetForm> = {}): AssetSnapshot {
  const form: AssetForm = { snapshot_date: date, bank: '0.10', platform: '0.20', investment: '0', other: '0', debt: '1', note: '', ...amounts }
  const t = assetTotals(form)
  const minor = (key: 'bank' | 'platform' | 'investment' | 'other' | 'debt') => form[key] === '' ? null : String(BigInt(form[key].split('.')[0]!) * 100n + BigInt((form[key].split('.')[1] || '').padEnd(2, '0')))
  return { id, snapshot_date: date, currency: 'CNY', bank_minor: minor('bank'), platform_minor: minor('platform'), investment_minor: minor('investment'), other_minor: minor('other'), debt_minor: minor('debt'), asset_total_minor: t.asset?.toString() ?? null, net_asset_minor: t.net?.toString() ?? null, complete: t.complete, note: form.note, version: 1, created_at: time, updated_at: time }
}
async function mockAssets(page: Page, initial: AssetSnapshot[] = []) {
  const state = { rows: [...initial], writes: 0, failSave: false, failRead: false }
  await page.route('**/api/v1/**', async route => {
    const req = route.request(), url = new URL(req.url()), path = url.pathname
    const reply = (data: unknown) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data }) })
    const fail = (code: string, message: string, status = 500) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify({ error: { code, message, retryable: status >= 500 } }) })
    if (path.endsWith('/auth/me')) return reply({ id: 1, role: 'admin', status: 'active', username: 'test' })
    if (path === '/api/v1/finance/ledgers') return reply([])
    if (path === '/api/v1/finance/assets' && req.method() === 'GET') { if (state.failRead) return fail('TEST', '资产读取失败'); return reply([...state.rows].sort((a, b) => b.snapshot_date.localeCompare(a.snapshot_date))) }
    if (/^\/api\/v1\/finance\/assets\/\d+\/goals$/.test(path)) return reply([])
    if (path.startsWith('/api/v1/finance/assets')) {
      const id = Number(path.split('/')[5])
      if (req.method() === 'GET') return reply(state.rows.find(r => r.id === id))
      if (req.method() === 'DELETE') { state.rows = state.rows.filter(r => r.id !== id); return reply({ deleted: true }) }
      state.writes++
      if (state.failSave) return fail('TEST_SAVE', '保存失败，输入仍保留')
      const input = req.postDataJSON()
      if (state.rows.some(r => r.snapshot_date === input.snapshot_date && r.id !== id)) return fail('ASSET_DATE_EXISTS', '这一天已有资产快照，请查看并编辑现有记录。', 409)
      const form = Object.fromEntries(Object.entries(input).map(([k, v]) => [k, v ?? '']))
      const saved = snapshot(id || Math.max(0, ...state.rows.map(r => r.id)) + 1, input.snapshot_date, form)
      if (id) saved.version = (state.rows.find(r => r.id === id)?.version ?? 0) + 1
      state.rows = [...state.rows.filter(r => r.id !== saved.id), saved]
      return reply(saved)
    }
    return reply([])
  })
  return state
}
async function fillComplete(page: Page, date: string) {
  const d = page.getByRole('dialog')
  await d.getByRole('combobox', { name: '快照日期' }).fill(date)
  await d.getByRole('combobox', { name: '快照日期' }).press('Enter')
  for (const [label, value] of [['银行存款', '0.10'], ['支付平台余额', '0.20'], ['投资资产', '0'], ['其他资产', '0'], ['负债总额', '1.00']]) await d.getByRole('textbox', { name: label + '（元）', exact: true }).fill(value!)
}
test('空资产档案可创建，精确计算、失败保留、刷新与单条快照正常', async ({ page }) => {
  const state = await mockAssets(page)
  await page.goto('/life/finance?tab=assets')
  await expect(page.getByText('留下第一份资产记录')).toBeVisible()
  await page.getByRole('button', { name: '新增第一份快照' }).click()
  const d = page.getByRole('dialog')
  await page.getByRole('button', { name: '保存快照' }).click()
  await expect(d.getByText('请选择快照日期，并至少填写一项金额；0 是有效金额。')).toBeVisible()
  await fillComplete(page, '2025-12-31')
  await expect(d.locator('.asset-preview')).toContainText('-0.70')
  state.failSave = true
  await d.getByRole('button', { name: '保存快照' }).click()
  await expect(d.getByText('保存失败，输入仍保留')).toBeVisible()
  await expect(d.getByRole('textbox', { name: '银行存款（元）', exact: true })).toHaveValue('0.10')
  state.failSave = false
  await d.getByRole('button', { name: '保存快照' }).click({ clickCount: 2 })
  await expect(d.getByRole('button', { name: '编辑快照' })).toBeVisible()
  expect(state.rows).toHaveLength(1); expect(state.writes).toBe(2)
  await d.getByRole('button', { name: '关闭', exact: true }).click()
  await expect(page.getByText('只有一份完整快照', { exact: false })).toBeVisible()
  await page.reload()
  await expect(page.locator('.asset-totals')).toContainText('-0.70')
  await expect(page.getByRole('tab', { name: '资产档案' })).toHaveAttribute('aria-selected', 'true')
})
test('模板不会自动保存或修改历史，编辑、同日冲突及确认删除刷新', async ({ page }) => {
  const state = await mockAssets(page, [snapshot(1, '2025-12-31')])
  await page.goto('/life/finance?tab=assets')
  await page.getByRole('button', { name: '新增资产快照', exact: true }).click()
  const d = page.getByRole('dialog')
  await d.getByRole('button', { name: '使用上次快照作为填写模板' }).click()
  await expect(d.getByText('这些是上次记录的金额', { exact: false })).toBeVisible()
  expect(state.writes).toBe(0)
  await expect(d.getByRole('textbox', { name: '银行存款（元）', exact: true })).toHaveValue('0.10')
  await fillComplete(page, '2026-02-01')
  await d.getByRole('textbox', { name: '银行存款（元）', exact: true }).fill('2.10')
  await d.getByRole('button', { name: '保存快照' }).click()
  await expect(d.getByRole('button', { name: '编辑快照' })).toBeVisible()
  expect(state.rows.find(r => r.id === 1)?.bank_minor).toBe('10')
  await d.getByRole('button', { name: '编辑快照' }).click()
  await d.getByRole('textbox', { name: '银行存款（元）', exact: true }).fill('3.10')
  await d.getByRole('button', { name: '保存快照' }).click()
  await expect(d.getByRole('button', { name: '编辑快照' })).toBeVisible()
  await d.getByRole('button', { name: '关闭', exact: true }).click()
  await expect(page.getByText('净资产变化', { exact: false })).toContainText('3.00')
  await page.getByRole('button', { name: '新增资产快照', exact: true }).click()
  await fillComplete(page, '2026-02-01')
  await d.getByRole('button', { name: '保存快照' }).click()
  await expect(d.getByText('这一天已有资产快照', { exact: false })).toBeVisible()
  expect(state.rows).toHaveLength(2)
  await d.getByRole('button', { name: '取消', exact: true }).click()
  await page.getByRole('dialog', { name: '未保存的修改' }).getByRole('button', { name: '放弃修改' }).click()
  await page.getByRole('button', { name: '查看最新快照' }).click()
  await d.getByRole('button', { name: '删除快照' }).click()
  await page.getByRole('dialog', { name: '删除快照', exact: true }).getByRole('button', { name: '取消', exact: true }).click()
  expect(state.rows).toHaveLength(2)
  await d.getByRole('button', { name: '删除快照' }).click()
  await page.getByRole('dialog', { name: '删除快照', exact: true }).getByRole('button', { name: '确认删除' }).click()
  await expect(d).toHaveCount(0)
  expect(state.rows).toHaveLength(1)
  await expect(page.getByText('只有一份完整快照', { exact: false })).toBeVisible()
})
test('不完整记录与零金额不混入趋势，实际日期点可查看并保留备注', async ({ page }) => {
  await mockAssets(page, [snapshot(1, '2025-12-31'), snapshot(2, '2026-03-10', { bank: '3.10' }), snapshot(3, '2026-06-02', { platform: '', note: '部分记录备注' })])
  await page.goto('/life/finance?tab=assets')
  await expect(page.getByText('不完整范围 · 仅反映', { exact: false })).toBeVisible()
  await expect(page.locator('.asset-totals')).toContainText('已填范围差额')
  await expect(page.locator('.asset-composition')).toContainText('未记录')
  await expect(page.locator('.asset-chart [role=button]')).toHaveCount(2)
  await expect(page.getByText('净资产变化', { exact: false })).toContainText('3.00')
  await page.locator('.asset-chart [role=button]').first().click()
  const d = page.getByRole('dialog'); await expect(d.getByText('2025-12-31 · 完整资产范围')).toBeVisible(); await expect(d.locator('input, textarea')).toHaveCount(0)
})
for (const width of [1440, 390]) test(`资产档案 ${width}px 长金额、详情与编辑可用`, async ({ page }) => {
  await mockAssets(page, [snapshot(1, '2025-12-31', { bank: '9999999999.99', note: '长期留下的备注。'.repeat(100) }), snapshot(2, '2026-03-10', { bank: '0' })])
  await page.setViewportSize({ width, height: 850 }); await page.goto('/life/finance?tab=assets')
  await expect(page.getByText('净资产历史', { exact: true })).toBeVisible()
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
  if (process.env.CAPTURE_VISUALS) await page.screenshot({ path: `test-results/assets-${width}.png`, fullPage: true, animations: 'disabled' })
  await page.getByRole('button', { name: '查看最新快照' }).click(); const d = page.getByRole('dialog')
  await d.getByRole('button', { name: '编辑快照' }).click()
  await expect(d.getByRole('button', { name: '保存快照' })).toBeVisible()
  expect((await d.boundingBox())!.width).toBeLessThanOrEqual(width)
  if (process.env.CAPTURE_VISUALS) await page.screenshot({ path: `test-results/assets-form-${width}.png`, animations: 'disabled' })
})
