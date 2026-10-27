import { expect, test, type Page, type Route } from '@playwright/test'

/**
 * Visual lab acceptance (Phase 11.2A).
 *
 * The route only exists in dev builds, and Playwright runs against the Vite dev
 * server, so `/lab/webgl` is reachable here and absent from production. All API
 * calls are intercepted, so no Go server is needed.
 *
 * GPU availability is not a precondition: tests that need real pixels skip
 * themselves when the environment cannot create a WebGL2 context.
 *
 * Screenshots are written only with CAPTURE_VISUALS=1, like visual-audit.spec.ts.
 */

async function installMocks(page: Page) {
  await page.route('**/api/v1/**', async (route: Route) => {
    const path = new URL(route.request().url()).pathname
    const json = (data: unknown) => route.fulfill({ json: { data } })
    if (path === '/api/v1/auth/me') {
      return json({ id: 1, username: 'visual', display_name: '视觉评审', role: 'admin', status: 'active', last_login_at: null, created_at: '2026-10-21T08:00:00Z' })
    }
    return json({})
  })
}

async function openLab(page: Page, width = 1440, height = 900) {
  await page.setViewportSize({ width, height })
  await installMocks(page)
  await page.goto('/lab/webgl', { waitUntil: 'networkidle' })
  await expect(page.locator('.visual-lab-stage')).toBeVisible()
}

async function webglAvailable(page: Page): Promise<boolean> {
  return page.evaluate(() => Boolean((window as unknown as { __visualLab?: { supported: boolean } }).__visualLab?.supported))
}

async function frameCount(page: Page): Promise<number> {
  return page.evaluate(() => (window as unknown as { __visualLab?: { frames: () => number } }).__visualLab?.frames() ?? -1)
}

interface LabDiagnostics {
  frames: number
  instances: number
  programs: number
}

async function diagnostics(page: Page): Promise<LabDiagnostics | null> {
  return page.evaluate(() => {
    const target = window as unknown as {
      __visualLab?: { frames: () => number; instances: () => number; programs: () => number }
    }
    const lab = target.__visualLab
    return lab ? { frames: lab.frames(), instances: lab.instances(), programs: lab.programs() } : null
  })
}

/** Copies a native-resolution block out of the stage canvas for pixel comparison. */
async function sampleBlock(page: Page, x: number, y: number, size = 32): Promise<number[]> {
  return page.evaluate(([sx, sy, s]) => {
    const canvas = document.querySelector<HTMLCanvasElement>('.visual-lab-canvas')
    const context = canvas?.getContext('webgl2')
    if (!canvas || !context) return []
    const probe = document.createElement('canvas')
    probe.width = s
    probe.height = s
    const ctx = probe.getContext('2d')
    if (!ctx) return []
    ctx.drawImage(canvas, sx, sy, s, s, 0, 0, s, s)
    const data = ctx.getImageData(0, 0, s, s).data
    const values: number[] = []
    for (let i = 0; i < data.length; i += 4) values.push(data[i]!, data[i + 1]!, data[i + 2]!)
    return values
  }, [x, y, size] as const)
}

function meanDelta(a: number[], b: number[]): number {
  const length = Math.min(a.length, b.length)
  if (!length) return 0
  let total = 0
  for (let i = 0; i < length; i += 1) total += Math.abs(a[i]! - b[i]!)
  return total / length
}

async function capture(page: Page, name: string) {
  if (!process.env.CAPTURE_VISUALS) return
  // Let the 1 Hz sampler and the first frames land before capturing.
  await page.waitForTimeout(1400)
  await page.evaluate(() => window.scrollTo(0, 0))
  await page.screenshot({ path: `test-results/visual-lab-${name}.png`, fullPage: true })
}

test('实验页打开，默认场景为 A，指标来自真实画布', async ({ page }) => {
  const errors: string[] = []
  page.on('pageerror', (error) => errors.push(error.message))
  await openLab(page)

  await expect(page.locator('.page-header h1')).toHaveText('WebGL Visual Lab')
  await expect(page.getByText('仅本地开发 · 未注册正式路由')).toBeVisible()
  await expect(page.locator('.visual-lab-stage-note')).toContainText('方案 A · Spatial Ambient')

  if (!(await webglAvailable(page))) test.skip(true, 'WebGL2 is not available in this environment')

  const canvas = page.locator('.visual-lab-canvas')
  await expect(canvas).toBeVisible()
  await expect.poll(async () => canvas.evaluate((element) => (element as HTMLCanvasElement).width)).toBeGreaterThan(0)
  await expect(page.locator('.visual-lab-state')).toContainText('运行中')

  const metric = (label: string) => page.locator('.visual-lab-metrics div').filter({ hasText: label }).locator('dd')
  await expect(metric('画布')).not.toHaveText('—')
  await expect(metric('宽高比')).not.toHaveText('—')
  await expect(metric('上下文')).toHaveText('正常')
  await expect(metric('GPU 计时')).toHaveText(/不可用|测量中|\d+\.\d+ ms/)

  expect(errors).toEqual([])
  await capture(page, 'dark')
})

test('A/B 切换改变画布输出', async ({ page }) => {
  await openLab(page)
  if (!(await webglAvailable(page))) test.skip(true, 'WebGL2 is not available in this environment')

  const before = await sampleBlock(page, 600, 120, 48)
  await page.getByText('B · 材质', { exact: true }).click()
  await expect(page.locator('.visual-lab-stage-note')).toContainText('方案 B · Material Depth')
  await page.waitForTimeout(120)
  const after = await sampleBlock(page, 600, 120, 48)

  expect(meanDelta(before, after)).toBeGreaterThan(0.5)
  await capture(page, 'material')
})

test('强度控件真实改变光效', async ({ page }) => {
  await openLab(page)
  if (!(await webglAvailable(page))) test.skip(true, 'WebGL2 is not available in this environment')

  const corner = { x: 20, y: 40 }
  const before = await sampleBlock(page, corner.x, corner.y, 48)
  await page.getByRole('slider', { name: '强度' }).focus()
  await page.getByRole('slider', { name: '强度' }).press('Home')
  await expect(page.locator('.visual-lab-row__value')).toHaveText('0.00')
  await page.waitForTimeout(120)
  const after = await sampleBlock(page, corner.x, corner.y, 48)

  expect(meanDelta(before, after)).toBeGreaterThan(0.5)
})

test('质量「静态」停止持续动画', async ({ page }) => {
  await openLab(page)
  if (!(await webglAvailable(page))) test.skip(true, 'WebGL2 is not available in this environment')

  await page.getByText('静态', { exact: true }).click()
  await expect(page.locator('.visual-lab-state')).toHaveText('静态模式 · 只渲染一帧')
  await expect(page.locator('.visual-lab-canvas')).toBeVisible()

  const first = await frameCount(page)
  await page.waitForTimeout(500)
  expect(await frameCount(page)).toBe(first)
})

test('关闭渲染后画面停在最后一帧，且不再产生帧', async ({ page }) => {
  await openLab(page)
  if (!(await webglAvailable(page))) test.skip(true, 'WebGL2 is not available in this environment')

  await page.locator('.visual-lab-row').filter({ hasText: '渲染' }).locator('.el-switch').click()
  await expect(page.locator('.visual-lab-state')).toHaveText('已暂停 · 停留在最后一帧')
  await expect(page.locator('.visual-lab-canvas')).toBeVisible()

  const first = await frameCount(page)
  await page.waitForTimeout(500)
  expect(await frameCount(page)).toBe(first)
})

test('主题切换后场景与页面一起换色', async ({ page }) => {
  await openLab(page)
  if (!(await webglAvailable(page))) test.skip(true, 'WebGL2 is not available in this environment')

  const dark = await sampleBlock(page, 20, 40, 48)
  await page.getByText('浅色', { exact: true }).click()
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'light')
  await expect.poll(async () => page.evaluate(() => localStorage.getItem('learnos-theme'))).toBe('light')
  await page.waitForTimeout(120)
  const light = await sampleBlock(page, 20, 40, 48)

  expect(meanDelta(dark, light)).toBeGreaterThan(10)
  await capture(page, 'light')
})

test('离开页面后不再有渲染循环', async ({ page }) => {
  await openLab(page)
  if (!(await webglAvailable(page))) test.skip(true, 'WebGL2 is not available in this environment')

  await expect.poll(() => frameCount(page)).toBeGreaterThan(3)
  await page.locator('.home-account__button').click()
  await page.getByText('退出登录').click()
  await expect(page).toHaveURL(/\/login$/)
  await expect(page.locator('.visual-lab-canvas')).toHaveCount(0)

  const afterUnmount = await frameCount(page)
  await page.waitForTimeout(600)
  expect(await frameCount(page)).toBe(afterUnmount)
  // The renderer itself must be gone, not merely idle.
  expect((await diagnostics(page))?.instances).toBe(0)
})

test('上下文丢失后恢复', async ({ page }) => {
  await openLab(page)
  if (!(await webglAvailable(page))) test.skip(true, 'WebGL2 is not available in this environment')

  await page.evaluate(() => {
    const canvas = document.querySelector<HTMLCanvasElement>('.visual-lab-canvas')
    const extension = canvas?.getContext('webgl2')?.getExtension('WEBGL_lose_context')
    if (canvas && extension) {
      // A lost context no longer exposes extensions, so keep the handle around.
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
  await expect(page.locator('.visual-lab-metrics div').filter({ hasText: '上下文' }).locator('dd')).toHaveText('正常')
})

test('不支持 WebGL2 时回退到现有 CSS 光场', async ({ page }) => {
  await page.addInitScript(() => {
    const prototype = HTMLCanvasElement.prototype as unknown as {
      getContext: (type: string, ...rest: unknown[]) => unknown
    }
    const original = prototype.getContext
    prototype.getContext = function (this: HTMLCanvasElement, type: string, ...rest: unknown[]) {
      if (type === 'webgl2') return null
      return original.apply(this, [type, ...rest])
    }
  })
  await openLab(page)

  await expect(page.locator('.visual-lab-state')).toHaveText('当前浏览器不支持 WebGL2 · 显示静态画面')
  await expect(page.locator('.visual-lab-canvas')).toBeHidden()
  await expect(page.locator('.visual-lab-static .home-light-field')).toBeVisible()
  await expect(page.locator('.visual-lab-metrics div').filter({ hasText: '上下文' }).locator('dd')).toHaveText('不可用')
  await expect(page.locator('.visual-lab-metrics div').filter({ hasText: 'GPU 计时' }).locator('dd')).toHaveText('不可用')
  await capture(page, 'unsupported')
})

test('减少动态效果时默认暂停，且可手动开启动画', async ({ page }) => {
  await page.emulateMedia({ reducedMotion: 'reduce' })
  await openLab(page)

  await expect(page.locator('.visual-lab-state')).toHaveText('已暂停 · 停留在最后一帧')
  await expect(page.locator('.visual-lab-canvas')).toBeVisible()

  if (!(await webglAvailable(page))) return
  await page.locator('.visual-lab-row').filter({ hasText: '渲染' }).locator('.el-switch').click()
  await expect(page.locator('.visual-lab-state')).toContainText('已手动开启动画')
})

test('关键文字由 DOM 承载，画布只做装饰', async ({ page }) => {
  await openLab(page)

  await expect(page.locator('.visual-lab-canvas')).toHaveAttribute('aria-hidden', 'true')
  await expect(page.locator('.visual-lab-canvas')).toHaveAttribute('tabindex', '-1')
  await expect(page.locator('.visual-lab-state')).toBeVisible()
  for (const label of ['渲染', '场景', '强度', '质量', '主题', '帧率', '绘制', '宽高比', '画布', '像素比', 'GPU 计时', '上下文']) {
    await expect(page.getByText(label, { exact: true }).first()).toBeVisible()
  }
})

test('390px 无横向溢出', async ({ page }) => {
  await openLab(page, 390, 844)
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth + 1)).toBe(true)
  await expect(page.locator('.visual-lab-stage')).toBeVisible()
  await expect(page.locator('.visual-lab-metrics')).toBeVisible()
  await capture(page, 'mobile')
})

test('主题切换不重建渲染器', async ({ page }) => {
  await openLab(page)
  if (!(await webglAvailable(page))) test.skip(true, 'WebGL2 is not available in this environment')

  const before = await diagnostics(page)
  await page.getByText('浅色', { exact: true }).click()
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'light')
  await page.waitForTimeout(600)
  const after = await diagnostics(page)

  expect(after?.instances).toBe(1)
  expect(after?.programs).toBe(1)
  expect(after!.frames).toBeGreaterThan(before!.frames)
})

test('场景切换后只保留一个 program', async ({ page }) => {
  await openLab(page)
  if (!(await webglAvailable(page))) test.skip(true, 'WebGL2 is not available in this environment')

  await page.getByText('B · 材质', { exact: true }).click()
  await page.waitForTimeout(200)
  expect((await diagnostics(page))?.programs).toBe(1)
  await page.getByText('A · 空间光场', { exact: true }).click()
  await page.waitForTimeout(200)
  expect((await diagnostics(page))?.programs).toBe(1)
})

test('页面隐藏时停止绘制并清空实时帧率，恢复后继续', async ({ page }) => {
  await openLab(page)
  if (!(await webglAvailable(page))) test.skip(true, 'WebGL2 is not available in this environment')

  await expect.poll(() => frameCount(page)).toBeGreaterThan(3)
  // Simulation: a test cannot background the page, so the read-only visibility
  // getters are overridden and the real handler is exercised.
  await page.evaluate(() => {
    Object.defineProperty(document, 'hidden', { configurable: true, get: () => true })
    Object.defineProperty(document, 'visibilityState', { configurable: true, get: () => 'hidden' })
    document.dispatchEvent(new Event('visibilitychange'))
  })
  const hiddenAt = await frameCount(page)
  await page.waitForTimeout(500)
  expect(await frameCount(page)).toBe(hiddenAt)
  await expect(page.locator('.visual-lab-metrics div').filter({ hasText: '帧率' }).locator('dd')).toHaveText('—')

  await page.evaluate(() => {
    Object.defineProperty(document, 'hidden', { configurable: true, get: () => false })
    Object.defineProperty(document, 'visibilityState', { configurable: true, get: () => 'visible' })
    document.dispatchEvent(new Event('visibilitychange'))
  })
  await expect.poll(() => frameCount(page)).toBeGreaterThan(hiddenAt + 2)

  // A user pause must survive a hide/show cycle: returning must not resume.
  await page.locator('.visual-lab-row').filter({ hasText: '渲染' }).locator('.el-switch').click()
  await page.waitForTimeout(150)
  const pausedAt = await frameCount(page)
  await page.evaluate(() => {
    Object.defineProperty(document, 'hidden', { configurable: true, get: () => true })
    document.dispatchEvent(new Event('visibilitychange'))
  })
  await page.waitForTimeout(150)
  await page.evaluate(() => {
    Object.defineProperty(document, 'hidden', { configurable: true, get: () => false })
    document.dispatchEvent(new Event('visibilitychange'))
  })
  await page.waitForTimeout(300)
  expect(await frameCount(page)).toBe(pausedAt)
})

test('反复进入退出实验页不残留渲染器', async ({ page }) => {
  await openLab(page)
  if (!(await webglAvailable(page))) test.skip(true, 'WebGL2 is not available in this environment')

  for (let cycle = 0; cycle < 3; cycle += 1) {
    // SPA navigation keeps the JS context, so instance/frame counters stay meaningful.
    await page.evaluate(() => {
      history.pushState({}, '', '/login')
      dispatchEvent(new PopStateEvent('popstate', { state: {} }))
    })
    await expect(page.locator('.visual-lab-stage')).toHaveCount(0)
    expect((await diagnostics(page))?.instances).toBe(0)

    await page.evaluate(() => {
      history.pushState({}, '', '/lab/webgl')
      dispatchEvent(new PopStateEvent('popstate', { state: {} }))
    })
    await expect(page.locator('.visual-lab-stage')).toBeVisible()
    expect((await diagnostics(page))?.instances).toBe(1)
  }
})

test('无关业务页面不会初始化 WebGL2', async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 })
  await installMocks(page)
  await page.goto('/', { waitUntil: 'networkidle' })

  expect(await page.evaluate(() => (window as unknown as { __visualLab?: unknown }).__visualLab === undefined)).toBe(true)
  await expect(page.locator('canvas')).toHaveCount(0)
})

test('容器尺寸变化后画布跟随', async ({ page }) => {
  await openLab(page)
  if (!(await webglAvailable(page))) test.skip(true, 'WebGL2 is not available in this environment')

  const canvasRow = page.locator('.visual-lab-metrics div').filter({ hasText: '画布' }).locator('dd')
  const before = await canvasRow.textContent()
  await page.setViewportSize({ width: 1100, height: 900 })
  await expect.poll(async () => canvasRow.textContent()).not.toBe(before)
  await expect(page.locator('.visual-lab-canvas')).toBeVisible()
})
