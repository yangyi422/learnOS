import { expect, test } from '@playwright/test'

const stamp = '2026-09-20T08:00:00Z'
const course = { id: 1, name: '合成学习领域', description: '界面测试', goal: '', status: 'initializing', progress: 0, current_unit: '', current_lesson_id: null, generation_status: 'not_started', generation_progress: 0, learning_status: 'not_started', coverage_progress: 0, mastery_status: 'unseen', mastery_progress: 0, created_at: stamp, updated_at: stamp }
const collection = { id: 1, name: '合成资料专题', description: '用于界面验收', category: '烹饪', template: 'recipe', display_mode: 'wiki', cover_url: '', version: 1, created_at: stamp, updated_at: stamp }
const pageData = { id: 1, collection_id: 1, section_id: null, title: '合成菜谱', body: '# 正文标题\n\n中文阅读内容。\n\n| 食材 | 用量 |\n| --- | --- |\n| 鸡蛋 | 两个 |\n\n- 操作步骤', cover_url: '', tags: [], sources: [], version: 1, created_at: stamp, updated_at: stamp }
const routes = ['/', '/learn', '/courses', '/projects', '/inbox', '/life', '/life?view=goals', '/life/finance', '/life/reviews?month=2026-09', '/explore', '/explore/library', '/explore/library/1?view=wiki', '/explore/library/1/pages/1?view=wiki', '/explore/library/1?view=list', '/explore/library/1/pages/1?view=list']

for (const width of [1920, 1440, 1280, 1024, 390]) {
  test(`shared page baselines, reading columns and overflow at ${width}px`, async ({ page }, info) => {
    test.setTimeout(120_000)
    await page.setViewportSize({ width, height: 960 })
    const errors: string[] = []
    page.on('pageerror', error => errors.push(`${page.url()}: ${error.message}`))
    await page.route('**/api/v1/**', async route => {
      const path = new URL(route.request().url()).pathname
      let data: unknown = []
      if (path.endsWith('/auth/me')) data = { id: 1, username: 'visual-fixture', role: 'admin', status: 'active' }
      else if (path === '/api/v1/workspace/home') data = { date: '2026-09-20', focus: { kind: 'empty', title: '从一件事开始' }, daily_focus: { work: null, life: null }, today: { date: '2026-09-20', doing: [], due: [], next: [] }, projects: [], learning: [], inbox: { count: 0, items: [] }, errors: {} }
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
        const buttons = [...document.querySelectorAll('.page-header__actions .el-button:not(.el-button--small):not(.el-button--large):not(.is-link):not(.is-circle)')].map(button => ({ height: button.getBoundingClientRect().height, text: button.classList.contains('is-text') }))
        const overlap = Boolean(actions && headingBox.right > actions.left + 1 && headingBox.left < actions.right - 1 && headingBox.bottom > actions.top + 1 && headingBox.top < actions.bottom - 1)
        return { variant, left: shell.getBoundingClientRect().left, headingLeft: headingBox.left, width: shell.getBoundingClientRect().width, overflow: document.documentElement.scrollWidth > window.innerWidth + 1, reading: document.querySelector('.library-reader')?.getBoundingClientRect().width, buttons, overlap }
      })
      await expect(page.locator('h1'), url).toHaveCount(1)
      expect(metrics.overflow, url).toBe(false)
      expect(metrics.overlap, url).toBe(false)
      for (const button of metrics.buttons) expect(button.height, url).toBe(width <= 820 && !button.text ? 44 : 36)
      expect(Math.abs(metrics.left - metrics.headingLeft), url).toBeLessThan(1)
      if (metrics.variant.endsWith('standard')) expect(metrics.width).toBeLessThanOrEqual(1248)
      else expect(metrics.width).toBeLessThanOrEqual(width)
      // Home has its own approved full-width composition and horizontal frame.
      if (url !== '/') {
        if (baselines.has(metrics.variant)) expect(Math.abs(metrics.left - baselines.get(metrics.variant)!), url).toBeLessThan(1)
        else baselines.set(metrics.variant, metrics.left)
      }
      if (metrics.reading) expect(metrics.reading).toBeLessThanOrEqual(760)
      if (process.env.CAPTURE_VISUALS === '1') await page.screenshot({ path: info.outputPath(`page-${index}-${width}.png`), fullPage: true })
    }
    expect(errors).toEqual([])
  })
}

for (const theme of ['light', 'dark'] as const) {
  test(`semantic contrast and native control focus in ${theme}`, async ({ page }) => {
    await page.addInitScript(theme => localStorage.setItem('learnos-theme', theme), theme)
    await page.route('**/api/v1/**', route => route.fulfill({ contentType: 'application/json', body: JSON.stringify({ data: new URL(route.request().url()).pathname.endsWith('/auth/me') ? { id: 1, role: 'admin' } : { date: stamp, focus: { kind: 'empty' }, daily_focus: { work: null, life: null }, today: { doing: [], due: [], next: [] }, projects: [], learning: [], inbox: { count: 0, items: [] }, errors: {} } }) }))
    await page.goto('/')
    await expect(page.locator('.page-header h1')).toBeVisible()
    await expect(page.locator('.page-header h1')).toHaveCSS('font-size', '44px')
    await expect(page.getByRole('button', { name: '快速记录', exact: true })).toHaveCSS('border-radius', '999px')
    if (theme === 'dark') await expect(page.locator('body')).toHaveCSS('background-color', 'rgb(10, 1, 24)')
    const colors = await page.evaluate(() => {
      const resolve = (name: string) => {
        const element = document.createElement('span'); element.style.color = `var(${name})`; document.body.append(element)
        const value = getComputedStyle(element).color; element.remove(); return value
      }
      return Object.fromEntries(['--color-text', '--color-text-secondary', '--color-text-muted', '--color-surface', '--color-surface-subtle', '--color-surface-hover', '--color-accent-soft', '--color-border-strong', '--color-accent', '--button-primary-text', '--color-focus'].map(name => [name, resolve(name)]))
    })
    const luminance = (value: string) => {
      const channels = value.match(/[\d.]+/g)!.slice(0, 3).map(Number).map(v => v / 255).map(v => v <= .04045 ? v / 12.92 : ((v + .055) / 1.055) ** 2.4)
      return channels[0]! * .2126 + channels[1]! * .7152 + channels[2]! * .0722
    }
    const ratio = (a: string, b: string) => { const [low, high] = [luminance(a), luminance(b)].sort((a,b) => a-b); return (high! + .05) / (low! + .05) }
    for (const surface of ['--color-surface', '--color-surface-subtle', '--color-surface-hover', '--color-accent-soft']) {
      for (const text of ['--color-text', '--color-text-secondary', '--color-text-muted']) expect(ratio(colors[text]!, colors[surface]!), `${text} on ${surface}`).toBeGreaterThanOrEqual(4.5)
    }
    expect(ratio(colors['--color-border-strong']!, colors['--color-surface']!)).toBeGreaterThanOrEqual(3)
    expect(ratio(colors['--button-primary-text']!, colors['--color-accent']!)).toBeGreaterThanOrEqual(4.5)
    const capture = page.getByRole('button', { name: '快速记录', exact: true })
    await capture.focus()
    await expect(capture).toHaveCSS('outline-style', 'solid')
    await expect(capture).toHaveCSS('outline-width', '2px')
    await expect(capture).toHaveCSS('outline-offset', '2px')
    const actualButton = await capture.evaluate(button => {
      const rgba = (value: string) => value.match(/[\d.]+/g)!.map(Number)
      const foreground = rgba(getComputedStyle(button).backgroundColor)
      const backdrop = rgba(getComputedStyle(document.querySelector('.home-topbar')!).backgroundColor)
      const alpha = foreground[3] ?? 1
      return { text: getComputedStyle(button).color, background: 'rgb(' + foreground.slice(0, 3).map((c, i) => Math.round(c * alpha + backdrop[i]! * (1 - alpha))).join(',') + ')' }
    })
    expect(ratio(actualButton.text, actualButton.background)).toBeGreaterThanOrEqual(4.5)
    const semanticButtons = await page.evaluate(() => {
      const values: { kind: string; plain: boolean; text: string; background: string }[] = []
      for (const kind of ['primary', 'success', 'warning', 'danger']) for (const plain of [false, true]) {
        const button = document.createElement('button')
        button.className = `el-button el-button--${kind}${plain ? ' is-plain' : ''}`
        document.querySelector('.app-shell')!.append(button)
        const style = getComputedStyle(button)
        values.push({ kind, plain, text: style.color, background: style.backgroundColor })
        button.remove()
      }
      return values
    })
    for (const button of semanticButtons) expect(ratio(button.text, button.background), `${button.kind} plain=${button.plain}`).toBeGreaterThanOrEqual(4.5)
  })
}

test('default violet dark does not discard an explicit system preference', async ({ page }) => {
  await page.route('**/api/v1/**', route => route.fulfill({ status: 401, contentType: 'application/json', body: '{}' }))
  await page.emulateMedia({ colorScheme: 'light' })
  await page.goto('/login')
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark')
  await page.evaluate(() => localStorage.setItem('learnos-theme', 'system'))
  await page.reload()
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'light')
})
