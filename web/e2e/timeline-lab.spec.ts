import { expect, test, type Page, type Route } from '@playwright/test'

/**
 * Phase 11.4A — timeline visual lab (DEV-only route).
 *
 * Read-only: every API call is intercepted, and the spec also asserts that the
 * page never issues a write request while browsing.
 */

const STAMP = '2026-10-21T08:00:00Z'

const lifeEvent = (id: number, occurredOn: string, patch: Record<string, unknown> = {}) => ({
  id,
  title: `经历 ${id}`,
  occurred_on: occurredOn,
  description: `第 ${id} 条记录的说明。`,
  primary_domain: 'work',
  secondary_domain: '',
  milestone: false,
  goal_id: null,
  external_url: '',
  link_name: '',
  creation_key: '',
  source_type: '',
  source_id: null,
  source_title: '',
  source_available: true,
  source_status: '',
  created_at: STAMP,
  updated_at: STAMP,
  ...patch,
})

const EVENTS = [
  lifeEvent(1, '2026-10-09', { title: '在旧书店找到一本书', primary_domain: 'experiences', goal_id: 5 }),
  lifeEvent(2, '2026-08-12', { title: '完成首页信息架构评审', primary_domain: 'work', milestone: true }),
]

const GOALS = [{ id: 5, title: '长期阅读计划', why: '', current_note: '', status: 'active', domain: 'learning', project_id: null, project_title: '', project_available: false, external_url: '', link_name: '', created_at: STAMP, updated_at: STAMP }]

async function installMocks(page: Page) {
  await page.route('**/api/v1/**', async (route: Route) => {
    const path = new URL(route.request().url()).pathname
    const json = (data: unknown) => route.fulfill({ json: { data } })
    if (path === '/api/v1/auth/me') {
      return json({ id: 1, username: 'timeline', display_name: '时间线', role: 'admin', status: 'active', last_login_at: null, created_at: STAMP })
    }
    if (path === '/api/v1/life/events') return json(EVENTS)
    if (path === '/api/v1/life/goals') return json(GOALS)
    return json({})
  })
}

async function openLab(page: Page, path = '/lab/timeline') {
  await installMocks(page)
  await page.goto(path, { waitUntil: 'networkidle' })
  await expect(page.locator('.tvl-page')).toBeVisible()
}

async function visibleRows(page: Page): Promise<number> {
  return page.locator('.tvl-row').count()
}

test('默认读取真实记录，并提示记录较少', async ({ page }) => {
  const writes: string[] = []
  page.on('request', (request) => {
    if (request.method() !== 'GET' && request.url().includes('/api/v1/')) writes.push(`${request.method()} ${request.url()}`)
  })
  await openLab(page)

  await expect(page.locator('.page-header h1')).toHaveText('Timeline Visual Lab')
  await expect(page.locator('.tvl-row')).toHaveCount(2)
  await expect(page.locator('.tvl-status')).toContainText('真实记录还不多（2 条）')
  await expect(page.getByText('当前显示 2 / 2 条')).toBeVisible()
  expect(writes).toEqual([])
})

test('示例数据 20 条并可按月份分组', async ({ page }) => {
  await openLab(page, '/lab/timeline?source=sample&size=20')
  await page.getByText('全部', { exact: true }).click()
  await expect(page.locator('.tvl-row')).toHaveCount(20)
  await expect(page.getByText(/当前显示 20 \/ 20 条/)).toBeVisible()
  await expect(page.locator('.tvl-month__label').first()).toBeVisible()
})

test('500 条示例数据可渲染（容量实验）', async ({ page }) => {
  await openLab(page, '/lab/timeline?source=sample&size=500')
  await page.getByText('全部', { exact: true }).click()
  await expect(page.locator('.tvl-row')).toHaveCount(500)
})

test('时间范围与里程碑筛选真实生效', async ({ page }) => {
  await openLab(page, '/lab/timeline?source=sample&size=100')
  await page.getByText('全部', { exact: true }).click()
  const all = await visibleRows(page)
  expect(all).toBe(100)

  await page.getByText('近 1 月', { exact: true }).click()
  const recent = await visibleRows(page)
  expect(recent).toBeLessThan(all)

  await page.getByText('全部', { exact: true }).click()
  await page.getByText('只看里程碑', { exact: true }).click()
  const milestones = await visibleRows(page)
  expect(milestones).toBeGreaterThan(0)
  expect(milestones).toBeLessThan(all)
  const flags = await page.locator('.tvl-row__flag').count()
  expect(flags).toBe(milestones)
})

test('领域筛选作用于主领域与次领域', async ({ page }) => {
  await openLab(page, '/lab/timeline?source=sample&size=100')
  await page.getByText('全部', { exact: true }).click()
  await page.locator('.tvl-controls__select').click()
  await page.getByRole('option', { name: '工作与创造' }).click()
  await page.keyboard.press('Escape')

  const rows = await visibleRows(page)
  expect(rows).toBeGreaterThan(0)
  expect(rows).toBeLessThan(100)
  const categories = await page.locator('.tvl-row__meta').allInnerTexts()
  expect(categories.every((text) => text.includes('工作与创造'))).toBe(true)
})

test('点击事件查看详情并跳回原记录', async ({ page }) => {
  await openLab(page)
  await page.locator('.tvl-row').first().click()

  const detail = page.locator('.tvl-detail')
  await expect(detail).toBeVisible()
  await expect(detail).toContainText('在旧书店找到一本书')
  await expect(detail.locator('a.tvl-detail__link')).toHaveAttribute('href', '/life?event=1')

  await page.keyboard.press('Escape')
  await expect(page.locator('.tvl-detail')).toHaveCount(0)
})

test('移动端单列且不横向溢出', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  await openLab(page, '/lab/timeline?source=sample&size=20')
  await expect(page.locator('.tvl-row').first()).toBeVisible()
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth + 1)).toBe(true)
})

test('键盘可完成选择与查看详情', async ({ page }) => {
  await openLab(page)
  await page.locator('.tvl-row').first().focus()
  await page.keyboard.press('Enter')
  await expect(page.locator('.tvl-detail')).toBeVisible()
})

test('读取失败时显示错误与重试', async ({ page }) => {
  await installMocks(page)
  // Registered after the broad mock so this handler wins.
  // The client sends ?domain=&milestones=false, so match the path plus query.
  await page.route(/\/api\/v1\/life\/events(\?|$)/, (route) => route.fulfill({ status: 500, json: { error: { code: 'FAIL', message: '读取失败', retryable: true } } }))
  await page.goto('/lab/timeline', { waitUntil: 'networkidle' })
  await expect(page.locator('.tvl-page')).toBeVisible()
  await expect(page.locator('.tvl-status--error')).toContainText('无法读取生活时间线')
  await expect(page.getByRole('button', { name: '重试' })).toBeVisible()
})
