import { expect, test, type Page } from '@playwright/test'

const cardSelectors = ['.focus-card', '.today-section', '.projects-section', '.learning-section', '.capture-section', '.recent-section']
const controls = ['.home-search', '.home-capture-button', '.home-account__avatar', '.workspace-context-switch', '.capture-entry']
const stamp = '2026-10-28T08:00:00Z'

async function open(page: Page, theme: 'light' | 'dark', width = 1440) {
  await page.setViewportSize({ width, height: width === 390 ? 844 : 960 })
  await page.addInitScript(t => localStorage.setItem('learnos-theme', t), theme)
  await page.route('**/api/v1/**', route => {
    const url = new URL(route.request().url())
    const json = (data: unknown) => route.fulfill({ json: { data } })
    if (url.pathname.endsWith('/auth/me')) return json({ id: 1, username: 'material', display_name: '材质验收', role: 'admin', status: 'active', created_at: stamp })
    if (url.pathname.endsWith('/workspace/home')) return json({
      date: '2026-10-28', context: url.searchParams.get('context') || 'all', unclassified_count: 0,
      focus: null, daily_focus: { work: null, life: null },
      today: { date: '2026-10-28', doing: [], due: [], next: [] },
      planning: { items: [], due_today: [], overdue: [], previous_plan: [], summary: { planned: 0, due_today: 0, overdue: 0, previous_plan: 0, total: 0 } },
      projects: [], inbox: { count: 0, items: [] }, errors: {},
      learning: [{ course_id: 21, course_name: '认知科学', lesson_id: 22, lesson_title: '从直觉到证据', core_question: '如何检验判断？', current_level: 'understand', cognitive_status: 'stable', last_learning_at: stamp, has_current_lesson: true }],
    })
    if (url.pathname.endsWith('/workspace/recent')) return json({ library: [{ id: 1, collection_id: 1, title: '参考资料', collection_name: '资料库', display_mode: 'list', cover_url: '', updated_at: stamp }], events: [], goals: [], asset_date: '2026-10-28', errors: {} })
    return json([])
  })
  await page.goto('/?ambient=css', { waitUntil: 'networkidle' })
  await expect(page.locator('.glass-tier')).toHaveCount(6)
}

async function materials(page: Page) {
  return page.evaluate(selectors => selectors.map(selector => {
    const el = document.querySelector(selector)!
    const s = getComputedStyle(el), p = getComputedStyle(el, '::before')
    return { background: s.backgroundColor, image: s.backgroundImage, edge: s.borderTopColor, shadow: s.boxShadow, filter: s.backdropFilter, radius: s.borderRadius, opacity: s.opacity, isolation: s.isolation, film: p.backgroundImage, mask: p.maskImage, composite: p.maskComposite, z: p.zIndex, pointer: p.pointerEvents }
  }), cardSelectors)
}

for (const theme of ['light', 'dark'] as const) {
  for (const width of [1440, 390]) {
    test(`六卡同材质、内容不遮罩且不泄漏到其他路由 ${theme} ${width}px`, async ({ page }) => {
      await open(page, theme, width)
      const actual = await materials(page)
      expect(new Set(actual.map(m => JSON.stringify(m))).size).toBe(1)
      expect(actual[0]).toMatchObject({ background: 'rgba(0, 0, 0, 0)', image: 'none', opacity: '1', radius: '16px', isolation: 'isolate', z: '-1', pointer: 'none' })
      expect(actual[0]!.film).not.toContain('radial-gradient')
      expect(actual[0]!.mask.match(/linear-gradient/g)).toHaveLength(2)
      expect(actual[0]!.filter).toBe(theme === 'light' ? 'blur(8px) saturate(1.03)' : 'blur(10px) saturate(1.1)')
      const compact = await page.evaluate(selectors => selectors.map(selector => {
        const s = getComputedStyle(document.querySelector(selector)!)
        return JSON.stringify([s.backgroundColor, s.borderTopColor, s.boxShadow, s.backdropFilter])
      }), controls)
      expect(new Set(compact).size).toBe(1)
      expect(await page.locator('.workspace-home').evaluate(el => getComputedStyle(el).maskImage)).toBe('none')
      expect(await page.locator('.daily-focus-empty p').evaluate(el => getComputedStyle(el).opacity)).toBe('1')
      expect(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth)).toBe(false)
      await page.emulateMedia({ reducedMotion: 'reduce' })
      // The global accessibility floor uses !important 0.01ms, not literal 0s.
      expect(await page.locator('.focus-card').evaluate(el => parseFloat(getComputedStyle(el).transitionDuration))).toBeLessThanOrEqual(0.0001)
      await expect(page.locator('.home-ambient-canvas')).toHaveCount(0)
      await page.locator('.home-topnav a[href="/projects"]').evaluate(el => (el as HTMLAnchorElement).click())
      await expect(page).toHaveURL(/\/projects$/)
      await expect(page.locator('.home-topbar')).toHaveCSS('background-color', theme === 'light' ? 'rgba(245, 250, 255, 0.58)' : 'rgba(10, 6, 27, 0.92)')
      expect(await page.locator('.app-shell').evaluate(el => getComputedStyle(el).getPropertyValue('--home-surface-film').trim())).toBe('')
    })
  }

  test(`紧凑控件保持 Hover / Focus / Selected / Disabled 与快速记录 ${theme}`, async ({ page }) => {
    await open(page, theme)
    const search = page.locator('.home-search')
    await search.locator('input').focus()
    await expect(search).toHaveCSS('outline-width', '2px')
    await search.locator('input').press('Tab')
    const capture = page.locator('.home-capture-button')
    await expect(capture).toBeFocused()
    await expect(capture).toHaveCSS('outline-width', '2px')
    const normal = await capture.evaluate(el => getComputedStyle(el).backgroundColor)
    await capture.hover()
    await expect.poll(() => capture.evaluate(el => getComputedStyle(el).backgroundColor)).not.toBe(normal)
    await capture.click()
    await expect(page.getByRole('dialog')).toBeVisible()
    await page.keyboard.press('Escape')
    await expect(page.getByRole('dialog')).not.toBeVisible()
    await capture.evaluate(el => { (el as HTMLButtonElement).disabled = true })
    await expect(capture).toHaveCSS('opacity', '0.6')
    await expect(capture).toHaveCSS('cursor', 'not-allowed')
    await capture.evaluate(el => { (el as HTMLButtonElement).disabled = false })
    await expect(page.locator('.home-topnav a[aria-current="page"]')).toHaveAttribute('href', '/')
    const selected = page.locator('.workspace-context-switch .el-radio-button.is-active .el-radio-button__inner')
    expect(await selected.evaluate(el => getComputedStyle(el).backgroundColor)).not.toBe('rgba(0, 0, 0, 0)')
    // A saved/custom interaction palette must not split cards into different materials.
    await page.evaluate(() => {
      document.documentElement.dataset.homeAccent = 'custom'
      document.documentElement.style.setProperty('--home-user-hue', '220')
      document.documentElement.style.setProperty('--home-user-saturation', '70%')
    })
    expect(new Set((await materials(page)).map(m => JSON.stringify(m))).size).toBe(1)
  })

  test(`无滤镜/遮罩的静态降级声明仍可读且不改变几何 ${theme}`, async ({ page }) => {
    await open(page, theme)
    const box = await page.locator('.focus-card').boundingBox()
    // Exercise the shipped @supports fallback declarations on this capable browser.
    // This is fallback emulation, not a claim of native Safari/Firefox verification.
    const fallback = await page.evaluate(() => {
      const find = (rules: CSSRuleList): string => {
        for (const rule of Array.from(rules)) {
          if (rule instanceof CSSSupportsRule && rule.conditionText.startsWith('not') && rule.cssText.includes('--home-surface-fallback')) {
            return Array.from(rule.cssRules).map(r => r.cssText).join('\n')
          }
          if ('cssRules' in rule) { const found = find((rule as CSSGroupingRule).cssRules); if (found) return found }
        }
        return ''
      }
      for (const sheet of Array.from(document.styleSheets)) { const found = find(sheet.cssRules); if (found) return found }
      return ''
    })
    expect(fallback).not.toBe('')
    const emulation = await page.addStyleTag({ content: fallback })
    await expect(page.locator('.focus-card')).toHaveCSS('backdrop-filter', 'none')
    await expect(page.locator('.focus-card')).toHaveCSS('background-color', theme === 'light' ? 'rgba(246, 250, 254, 0.88)' : 'rgba(16, 10, 32, 0.92)')
    expect(await page.locator('.focus-card').evaluate(el => getComputedStyle(el, '::before').display)).toBe('none')
    expect(await page.locator('.focus-card').boundingBox()).toEqual(box)
    await expect(page.getByRole('button', { name: '设置工作焦点', exact: true })).toBeVisible()
    await emulation.evaluate(el => el.parentNode?.removeChild(el))
  })
}
