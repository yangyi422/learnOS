import { expect, test, type Page, type Route } from '@playwright/test'

type DashboardState = 'empty' | 'domain' | 'continue'
type Theme = 'light' | 'dark'

const timestamp = '2026-09-20T08:00:00Z'
const baseCourse = {
  id: 1, name: '植物观察', description: '认识身边植物的生长规律。', goal: '理解植物如何适应环境',
  status: 'initializing', progress: 0, current_unit: '', current_unit_id: null, current_lesson_id: null,
  generation_status: 'not_started', generation_progress: 0, learning_status: 'not_started', coverage_progress: 0,
  mastery_status: 'unseen', mastery_progress: 0,
  last_studied_at: null, created_at: timestamp, updated_at: timestamp,
}
const continuingCourse = {
  ...baseCourse, id: 2, name: '测试营养学', goal: '理解日常补水判断', status: 'learning',
  current_unit: '饮水判断', current_unit_id: 10, current_lesson_id: 101,
  generation_status: 'ready', generation_progress: 100, learning_status: 'in_progress', coverage_progress: 40,
  mastery_status: 'developing', mastery_progress: 24, last_studied_at: timestamp,
}

async function mockDashboard(page: Page, state: DashboardState) {
  let radarReads = 0
  await page.route('**/api/v1/**', async (route: Route) => {
    const path = new URL(route.request().url()).pathname
    const reply = (data: unknown) => route.fulfill({ contentType: 'application/json', body: JSON.stringify({ data }) })
    if (path === '/api/v1/auth/me') return reply({ id: 1, username: 'visual-test', role: 'admin', status: 'active' })
    if (path === '/api/v1/courses') return reply(state === 'empty' ? [] : state === 'domain' ? [baseCourse] : [baseCourse, continuingCourse])
    if (path === '/api/v1/courses/2/current-lesson') return reply({
      course: { id: 2, name: continuingCourse.name }, unit: { id: 10, title: '饮水判断', objective: '判断补水需求。' },
      lesson: { id: 101, title: '口渴是否可靠', core_question: '只要不口渴，是否说明身体不缺水？', status: 'learning', content_role: 'application', depth_level: 2 },
    })
    if (path.endsWith('/lessons/101/cognitive-state')) return reply({ state: { current_level: 'understand', status: 'stable' } })
    if (path === '/api/v1/exploration/radar') { radarReads++; return reply({ directions: [] }) }
    return reply({})
  })
  return { get radarReads() { return radarReads } }
}

for (const state of ['empty', 'domain', 'continue'] as const) {
  for (const theme of ['light', 'dark'] as const) {
    test(`dashboard ${state} ${theme}`, async ({ page }, testInfo) => {
      await page.setViewportSize({ width: 1280, height: 720 })
      await page.emulateMedia({ colorScheme: theme })
      const mock = await mockDashboard(page, state)
      await page.goto('/learn')
      await expect(page.locator('html')).toHaveAttribute('data-theme', theme)
      const frame = await page.evaluate(() => {
        const rail = document.querySelector('.sidebar')!.getBoundingClientRect()
        const workspace = document.querySelector('.app-content')!.getBoundingClientRect()
        const topbar = document.querySelector('.workspace-topbar')!.getBoundingClientRect()
        return { outer: rail.left, gap: workspace.left - rail.right, right: innerWidth - workspace.right, topbarLeft: topbar.left, workspaceLeft: workspace.left }
      })
      expect(frame.outer).toBeGreaterThanOrEqual(12)
      expect(frame.outer).toBeLessThanOrEqual(16)
      expect(frame.gap).toBeGreaterThanOrEqual(10)
      expect(frame.gap).toBeLessThanOrEqual(12)
      expect(frame.right).toBeGreaterThanOrEqual(12)
      expect(frame.topbarLeft).toBeGreaterThanOrEqual(frame.workspaceLeft)

      if (state === 'empty') {
        await expect(page.getByRole('heading', { name: '从一个好奇的问题开始' })).toBeVisible()
        await expect(page.getByRole('button', { name: '创建第一个学习领域' })).toHaveCount(1)
        await expect(page.locator('.dashboard-section')).toHaveCount(0)
        await expect(page.locator('.current-focus')).toHaveCount(0)
        expect(mock.radarReads).toBe(0)
        expect(await page.evaluate(() => document.documentElement.scrollHeight <= window.innerHeight + 1)).toBe(true)
      } else if (state === 'domain') {
        await expect(page.getByRole('heading', { name: '让学习从这里展开' })).toBeVisible()
        await expect(page.getByRole('button', { name: /打开知识结构/ })).toBeVisible()
        await expect(page.locator('.domain-tile')).toHaveCount(1)
        await expect(page.getByRole('button', { name: '+ 新建学习领域' })).toHaveCount(1)
        await expect(page.locator('.dashboard-exploration-empty')).toBeVisible()
      } else {
        await expect(page.getByRole('heading', { name: '继续你的学习' })).toBeVisible()
        await expect(page.locator('#current-focus-title')).toHaveText('口渴是否可靠')
        await expect(page.getByRole('button', { name: /继续学习/ })).toBeVisible()
        await expect(page.locator('.domain-tile')).toHaveCount(2)
        await expect(page.locator('.dashboard-exploration-empty')).toBeVisible()
      }

      if (process.env.CAPTURE_VISUALS) await page.screenshot({ path: testInfo.outputPath(`dashboard-${state}-${theme}.png`), fullPage: true })
    })
  }
}
