import { expect, test, type Page } from '@playwright/test'
import type { LifeGoal } from '../src/api/life'
import type { AssetSnapshot } from '../src/api/assets'
import type { GoalAssetReference } from '../src/api/goalAssets'
const time = '2026-10-10T02:00:00Z'
async function setup(page: Page, linked = false) {
  const goal = { id: 1, title: '买房', why: '长期居住', current_note: '保持观察', status: 'considering', domain: 'finance', project_id: null, external_url: '', link_name: '', created_at: time, updated_at: time } as LifeGoal
  const asset: AssetSnapshot = { id: 1, snapshot_date: '2026-10-10', currency: 'CNY', bank_minor: '100000', platform_minor: null, investment_minor: null, other_minor: null, debt_minor: '0', asset_total_minor: '100000', net_asset_minor: '100000', complete: false, note: '部分记录', version: 1, created_at: time, updated_at: time }
  const reference: GoalAssetReference = { id: 1, goal_id: 1, asset_snapshot_id: 1, note: '仍在评估', judgment_date: '2026-10-10', version: 1, created_at: time, updated_at: time, snapshot: asset }
  const state = { goal, asset, rows: linked ? [reference] : [] as GoalAssetReference[], writes: 0, failSave: false, failRead: false }
  await page.route('**/api/v1/**', async route => {
    const req = route.request(), path = new URL(req.url()).pathname
    const reply = (data: unknown) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data }) })
    const fail = (message: string, status = 500) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify({ error: { code: 'TEST', message } }) })
    if (path.endsWith('/auth/me')) return reply({ id: 1, role: 'admin', status: 'active', username: 'test' })
    if (path === '/api/v1/life/goals') return reply([goal])
    if (path === '/api/v1/life/goals/1') return reply({ goal, entries: [], events: [] })
    if (path === '/api/v1/life/goals/1/assets') return state.failRead ? fail('关联读取失败') : reply(state.rows.map(r => ({ ...r, snapshot: asset })))
    if (path === '/api/v1/life/goals/1/assets/1') {
      state.writes++
      if (state.failSave) return fail('关联保存失败，请重试')
      if (req.method() === 'DELETE') { state.rows = []; return reply({ deleted: true }) }
      const input = req.postDataJSON()
      const saved = { ...reference, ...input, version: req.method() === 'PUT' ? (state.rows[0]?.version ?? 1) + 1 : 1 }
      state.rows = [saved]; return reply(saved)
    }
    if (path === '/api/v1/finance/assets') return reply([asset])
    if (path === '/api/v1/finance/assets/1/goals') return reply(state.rows.length ? [goal] : [])
    if (path === '/api/v1/finance/assets/1') {
      if (req.method() === 'DELETE') return state.rows.length ? fail('此快照已关联长期目标，请先解除关联', 409) : reply({ deleted: true })
      return reply(asset)
    }
    return reply([])
  })
  return state
}
const financial = (page: Page) => page.locator('.goal-financial-references')
test('目标选择单份快照，失败保留说明、重试持久展示且不产生阶段记录', async ({ page }) => {
  const state = await setup(page)
  await page.goto('/life?view=goals&goal=1')
  await financial(page).getByRole('button', { name: '关联资产快照', exact: true }).click()
  await expect(financial(page).getByText('2026-10-10 · 不完整范围', { exact: true }).last()).toBeVisible()
  await financial(page).getByLabel('关联说明（可选）').fill('暂不急于购入')
  state.failSave = true
  await financial(page).getByRole('button', { name: '确认关联' }).click()
  await expect(financial(page).getByText('关联保存失败，请重试')).toBeVisible()
  await expect(financial(page).getByLabel('关联说明（可选）')).toHaveValue('暂不急于购入')
  state.failSave = false
  await financial(page).getByRole('button', { name: '确认关联' }).click()
  await expect(financial(page).getByText('暂不急于购入', { exact: true })).toBeVisible()
  expect(state.rows).toHaveLength(1)
  expect(state.goal.status).toBe('considering')
  await expect(financial(page).locator('article')).toContainText('已填范围差额')
  await expect(financial(page).locator('article')).not.toContainText('净资产')
  await page.reload()
  await expect(financial(page).getByText('暂不急于购入', { exact: true })).toBeVisible()
  await expect(page.getByText('目标会随生活变化', { exact: false })).toBeVisible()
})
test('修改说明、未保存关闭保护、取消关联不删除快照', async ({ page }) => {
  const state = await setup(page, true)
  await page.goto('/life?view=goals&goal=1')
  await financial(page).getByRole('button', { name: '编辑说明' }).click()
  await financial(page).getByLabel('关联说明（可选）').fill('准备继续等待')
  await page.getByRole('dialog').getByRole('button', { name: '关闭', exact: true }).click()
  await page.getByRole('button', { name: '继续编辑', exact: true }).click()
  await financial(page).getByRole('button', { name: '保存说明' }).click()
  await expect(financial(page).getByText('准备继续等待', { exact: true })).toBeVisible()
  await financial(page).getByRole('button', { name: '取消关联', exact: true }).click()
  await page.getByRole('button', { name: '保留关联', exact: true }).click()
  expect(state.rows).toHaveLength(1)
  await financial(page).getByRole('button', { name: '取消关联', exact: true }).click()
  await page.getByRole('button', { name: '解除关联', exact: true }).click()
  await expect(financial(page).getByText('尚未关联资产快照。')).toBeVisible()
  expect(state.asset.id).toBe(1)
})
test('目标到快照与反向目标入口复用详情，引用阻止删除并读取快照修订', async ({ page }) => {
  const state = await setup(page, true)
  await page.setViewportSize({ width: 390, height: 844 })
  await page.goto('/life?view=goals&goal=1')
  await financial(page).getByRole('button', { name: '查看快照' }).click()
  await expect(page).toHaveURL(/tab=assets.*asset=1/)
  await expect(page.getByRole('dialog')).toHaveCount(1)
  const detail = page.getByRole('dialog')
  await expect(detail.getByText('关联长期目标', { exact: true })).toBeVisible()
  await detail.getByRole('button', { name: '删除快照' }).click()
  await page.getByRole('button', { name: '确认删除', exact: true }).click()
  await expect(detail.getByText('此快照已关联长期目标，请先解除关联')).toBeVisible()
  state.asset.asset_total_minor = '200000'; state.asset.net_asset_minor = '200000'; state.asset.bank_minor = '200000'; state.asset.updated_at = '2026-10-11T02:00:00Z'
  await detail.getByRole('button', { name: '买房', exact: true }).click()
  await expect(page).toHaveURL(/goal=1/)
  await expect(page.getByRole('dialog')).toHaveCount(1)
  await expect(financial(page).locator('article')).toContainText('2,000.00')
  await expect(financial(page).getByText('仍在评估', { exact: true })).toBeVisible()
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
  if (process.env.CAPTURE_VISUALS) { await financial(page).locator('article').scrollIntoViewIfNeeded(); await page.waitForTimeout(300); await page.screenshot({ path: 'test-results/goal-assets-390.png', fullPage: true }); await page.setViewportSize({ width: 1440, height: 1000 }); await page.waitForTimeout(300); await page.screenshot({ path: 'test-results/goal-assets-1440.png', fullPage: true }) }
})
