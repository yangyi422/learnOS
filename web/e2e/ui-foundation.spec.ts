import { expect, test } from '@playwright/test'

const stamp = '2026-09-20T08:00:00Z'
const course = { id: 1, name: '合成学习领域', description: '界面测试', goal: '', status: 'initializing', progress: 0, current_unit: '', current_lesson_id: null, generation_status: 'not_started', generation_progress: 0, learning_status: 'not_started', coverage_progress: 0, mastery_status: 'unseen', mastery_progress: 0, created_at: stamp, updated_at: stamp }
const collection = { id: 1, name: '合成资料专题', description: '用于界面验收', category: '烹饪', template: 'recipe', display_mode: 'wiki', cover_url: '', version: 1, created_at: stamp, updated_at: stamp }
const pageData = { id: 1, collection_id: 1, section_id: null, title: '合成菜谱', body: '# 正文标题\n\n中文阅读内容。\n\n| 食材 | 用量 |\n| --- | --- |\n| 鸡蛋 | 两个 |\n\n- 操作步骤', cover_url: '', tags: [], sources: [], version: 1, created_at: stamp, updated_at: stamp }
const routes = ['/', '/learn', '/courses', '/projects', '/inbox', '/life', '/life?view=goals', '/life/finance', '/life/reviews?month=2026-09', '/explore', '/explore/library', '/explore/library/1?view=wiki', '/explore/library/1/pages/1?view=wiki', '/explore/library/1?view=list', '/explore/library/1/pages/1?view=list']

for (const width of [1920, 1440, 1024, 390]) {
  test(`shared page baselines, reading columns and overflow at ${width}px`, async ({ page }, info) => {
    test.setTimeout(120_000)
    await page.setViewportSize({ width, height: 960 })
    const errors: string[] = []
    page.on('pageerror', error => errors.push(`${page.url()}: ${error.message}`))
    await page.route('**/api/v1/**', async route => {
      const path = new URL(route.request().url()).pathname
      let data: unknown = []
      if (path.endsWith('/auth/me')) data = { id: 1, username: 'visual-fixture', role: 'admin', status: 'active' }
      else if (path === '/api/v1/workspace/home') data = { date: '2026-09-20', focus: { kind: 'empty', title: '从一件事开始' }, today: { date: '2026-09-20', doing: [], due: [], next: [] }, projects: [], learning: [], inbox: { count: 0, items: [] }, errors: {} }
      else if (path === '/api/v1/courses') data = [course]
      else if (path === '/api/v1/exploration/radar') data = { directions: [] }
      else if (path === '/api/v1/inbox') data = { items: [], counts: { inbox: 0, processed: 0, archived: 0 } }
      else if (/\/life\/reviews\/\d{4}-\d{2}(?:\/preview)?$/.test(path)) data = { review: null, facts_id: 1, history: [], facts: { month: '2026-09', timezone: 'Asia/Shanghai', from: '2026-09-01', to: '2026-09-30', generated_at: stamp, in_progress: false, actions: [], learning: [], events: [], goals: [], finance: [], assets: [], limits: [] } }
      else if (path === '/api/v1/library/collections') data = [collection]
      else if (path === '/api/v1/library/collections/1') data = { collection, sections: [{ id: 1, collection_id: 1, name: '步骤' }] }
      else if (path.endsWith('/pages')) data = { items: [{ ...pageData, summary: '中文阅读内容' }], total: 1, page: 1 }
      else if (path.endsWith('/pages/1')) data = pageData
      await route.fulfill({ contentType: 'application/json', body: JSON.stringify({ data }) })
    })
    const baselines = new Map<string, number>()
    for (const [index, url] of routes.entries()) {
      await page.goto(url)
      await expect(page.locator('.page-header h1')).toBeVisible()
      await page.waitForTimeout(150)
      const metrics = await page.evaluate(() => {
        const shell = document.querySelector('.page-shell')!
        const heading = document.querySelector('.page-header h1')!
        const variant = [...shell.classList].find(c => c.startsWith('page-shell--'))!
        const headingBox = heading.getBoundingClientRect()
        const actions = document.querySelector('.page-header__actions')?.getBoundingClientRect()
        const buttons = [...document.querySelectorAll('.page-header__actions .el-button:not(.el-button--small):not(.el-button--large):not(.is-link):not(.is-circle)')].map(button => button.getBoundingClientRect().height)
        const overlap = Boolean(actions && headingBox.right > actions.left + 1 && headingBox.left < actions.right - 1 && headingBox.bottom > actions.top + 1 && headingBox.top < actions.bottom - 1)
        return { variant, left: shell.getBoundingClientRect().left, headingLeft: headingBox.left, width: shell.getBoundingClientRect().width, overflow: document.documentElement.scrollWidth > window.innerWidth + 1, reading: document.querySelector('.library-reader')?.getBoundingClientRect().width, buttons, overlap }
      })
      expect(metrics.overflow, url).toBe(false)
      expect(metrics.overlap, url).toBe(false)
      for (const height of metrics.buttons) expect(height, url).toBe(36)
      expect(Math.abs(metrics.left - metrics.headingLeft), url).toBeLessThan(1)
      expect(metrics.width).toBeLessThanOrEqual(metrics.variant.endsWith('standard') ? 1320 : 1600)
      if (baselines.has(metrics.variant)) expect(Math.abs(metrics.left - baselines.get(metrics.variant)!), url).toBeLessThan(1)
      else baselines.set(metrics.variant, metrics.left)
      if (metrics.reading) expect(metrics.reading).toBeLessThanOrEqual(860)
      if (process.env.CAPTURE_VISUALS === '1') await page.screenshot({ path: info.outputPath(`page-${index}-${width}.png`), fullPage: true })
    }
    expect(errors).toEqual([])
  })
}
