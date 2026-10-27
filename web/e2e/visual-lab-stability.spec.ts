import { expect, test, type Page, type Route } from '@playwright/test'

/**
 * Phase 11.2B stability run (opt-in).
 *
 *   cd web
 *   $env:STRESS='1'; npx playwright test e2e/visual-lab-stability.spec.ts --reporter=list
 *
 * It repeats mount/unmount, theme, render, quality, resize, visibility and
 * context-loss cycles while watching for console errors, leaked renderer
 * instances, stale frame counters and JS heap growth. Headless Chrome is not a
 * device benchmark, so the numbers are recorded, not asserted as targets.
 */

test.setTimeout(600_000)

const CYCLES = 30
const TOGGLES = 20
const LOSSES = 5

async function installMocks(page: Page) {
  await page.route('**/api/v1/**', (route: Route) => {
    const path = new URL(route.request().url()).pathname
    const json = (data: unknown) => route.fulfill({ json: { data } })
    if (path === '/api/v1/auth/me') {
      return json({ id: 1, username: 'stress', display_name: '压测', role: 'admin', status: 'active', last_login_at: null, created_at: '2026-10-21T08:00:00Z' })
    }
    return json({})
  })
}

async function enterLab(page: Page) {
  await page.evaluate(() => {
    history.pushState({}, '', '/lab/webgl')
    dispatchEvent(new PopStateEvent('popstate', { state: {} }))
  })
  await expect(page.locator('.visual-lab-stage')).toBeVisible()
}

async function leaveLab(page: Page) {
  await page.evaluate(() => {
    history.pushState({}, '', '/login')
    dispatchEvent(new PopStateEvent('popstate', { state: {} }))
  })
  await expect(page.locator('.visual-lab-stage')).toHaveCount(0)
}

async function diagnostics(page: Page) {
  return page.evaluate(() => {
    const target = window as unknown as {
      __visualLab?: { frames: () => number; instances: () => number; programs: () => number }
    }
    const lab = target.__visualLab
    return lab ? { frames: lab.frames(), instances: lab.instances(), programs: lab.programs() } : null
  })
}

async function heapMb(page: Page): Promise<number | null> {
  return page.evaluate(() => {
    const memory = (performance as Performance & { memory?: { usedJSHeapSize: number } }).memory
    return memory ? Math.round((memory.usedJSHeapSize / 1024 / 1024) * 10) / 10 : null
  })
}

async function metric(page: Page, label: string): Promise<string> {
  return (await page.locator('.visual-lab-metrics div').filter({ hasText: label }).locator('dd').textContent()) ?? ''
}

test('稳定性压力测试：30 次进出、20 次主题/渲染/质量切换、尺寸、可见性、上下文丢失', async ({ page }) => {
  test.skip(!process.env.STRESS, 'set STRESS=1 to run the stability suite')

  const errors: string[] = []
  page.on('pageerror', (error) => errors.push(`pageerror: ${error.message}`))
  page.on('console', (message) => { if (message.type() === 'error') errors.push(`console: ${message.text()}`) })

  await page.setViewportSize({ width: 1440, height: 900 })
  await installMocks(page)
  await page.goto('/lab/webgl', { waitUntil: 'networkidle' })
  await expect(page.locator('.visual-lab-stage')).toBeVisible()

  const supported = await page.evaluate(() => Boolean((window as unknown as { __visualLab?: { supported: boolean } }).__visualLab?.supported))
  test.skip(!supported, 'WebGL2 is not available in this environment')

  const heapSamples: Array<{ stage: string; mb: number | null }> = []
  heapSamples.push({ stage: 'baseline', mb: await heapMb(page) })
  await page.waitForTimeout(1600)
  const fpsStart = await metric(page, '帧率')

  // 1. 30 mount/unmount cycles (SPA, same JS context).
  for (let cycle = 1; cycle <= CYCLES; cycle += 1) {
    await leaveLab(page)
    expect((await diagnostics(page))?.instances, `cycle ${cycle} after leave`).toBe(0)
    await enterLab(page)
    expect((await diagnostics(page))?.instances, `cycle ${cycle} after enter`).toBe(1)
    if (cycle % 10 === 0) heapSamples.push({ stage: `after ${cycle} cycles`, mb: await heapMb(page) })
  }

  // 2. 20 theme toggles.
  for (let index = 0; index < TOGGLES; index += 1) {
    await page.getByText(index % 2 === 0 ? '浅色' : '深色', { exact: true }).click()
    await page.waitForTimeout(60)
  }
  expect((await diagnostics(page))?.instances).toBe(1)
  heapSamples.push({ stage: `after ${TOGGLES} theme toggles`, mb: await heapMb(page) })

  // 3. 20 render toggles (ends enabled because the count is even).
  for (let index = 0; index < TOGGLES; index += 1) {
    await page.locator('.visual-lab-row').filter({ hasText: '渲染' }).locator('.el-switch').click()
    await page.waitForTimeout(60)
  }
  await expect(page.locator('.visual-lab-canvas')).toBeVisible()
  heapSamples.push({ stage: `after ${TOGGLES} render toggles`, mb: await heapMb(page) })

  // 4. 20 quality toggles.
  for (let index = 0; index < TOGGLES; index += 1) {
    await page.getByText(['高', '均衡', '静态'][index % 3]!, { exact: true }).click()
    await page.waitForTimeout(60)
  }
  await page.getByText('高', { exact: true }).click()
  heapSamples.push({ stage: `after ${TOGGLES} quality toggles`, mb: await heapMb(page) })

  // 5. Window resizes.
  for (const width of [1280, 1024, 1440, 800, 1440]) {
    await page.setViewportSize({ width, height: 900 })
    await page.waitForTimeout(150)
  }
  heapSamples.push({ stage: 'after 5 resizes', mb: await heapMb(page) })

  // 6. Visibility hide/show.
  for (let index = 0; index < 3; index += 1) {
    await page.evaluate(() => {
      Object.defineProperty(document, 'hidden', { configurable: true, get: () => true })
      Object.defineProperty(document, 'visibilityState', { configurable: true, get: () => 'hidden' })
      document.dispatchEvent(new Event('visibilitychange'))
    })
    await page.waitForTimeout(120)
    await page.evaluate(() => {
      Object.defineProperty(document, 'hidden', { configurable: true, get: () => false })
      Object.defineProperty(document, 'visibilityState', { configurable: true, get: () => 'visible' })
      document.dispatchEvent(new Event('visibilitychange'))
    })
    await page.waitForTimeout(120)
  }

  // 7. Context loss / restore cycles.
  for (let index = 0; index < LOSSES; index += 1) {
    await page.evaluate(() => {
      const canvas = document.querySelector<HTMLCanvasElement>('.visual-lab-canvas')
      const extension = canvas?.getContext('webgl2')?.getExtension('WEBGL_lose_context')
      if (canvas && extension) {
        (canvas as HTMLCanvasElement & { labLoseContext?: WEBGL_lose_context }).labLoseContext = extension
        extension.loseContext()
      }
    })
    await expect(page.locator('.visual-lab-state')).toHaveText('图形上下文已丢失 · 等待恢复')
    await page.evaluate(() => {
      const canvas = document.querySelector<HTMLCanvasElement & { labLoseContext?: WEBGL_lose_context }>('.visual-lab-canvas')
      canvas?.labLoseContext?.restoreContext()
    })
    await expect(page.locator('.visual-lab-state')).toContainText('运行中')
  }
  heapSamples.push({ stage: `after ${LOSSES} context losses`, mb: await heapMb(page) })

  // 8. Read the last live numbers *before* leaving; the panel is gone afterwards.
  await page.waitForTimeout(1600)
  const fpsEnd = await metric(page, '帧率')
  heapSamples.push({ stage: 'final (mounted)', mb: await heapMb(page) })

  // 9. Leave for good and confirm nothing keeps running.
  await leaveLab(page)
  const settled = await diagnostics(page)
  const framesAtExit = settled?.frames ?? -1
  await page.waitForTimeout(800)
  expect((await diagnostics(page))?.frames).toBe(framesAtExit)
  expect((await diagnostics(page))?.instances).toBe(0)
  const summary = {
    cycles: CYCLES,
    toggles: TOGGLES,
    contextLosses: LOSSES,
    fpsStart,
    fpsEnd,
    heapSamples,
    errors,
  }
  console.log('STABILITY_SUMMARY ' + JSON.stringify(summary))

  expect(errors, 'no console or page errors during the stress run').toEqual([])

  // Heap is a trend, never a single-snapshot verdict: allow generous headroom.
  const baseline = heapSamples[0]?.mb
  const final = heapSamples[heapSamples.length - 1]?.mb
  if (baseline !== null && final !== null && baseline !== undefined && final !== undefined) {
    expect(final, `heap ${final} MB vs baseline ${baseline} MB`).toBeLessThan(baseline * 2.5 + 20)
  }
})
