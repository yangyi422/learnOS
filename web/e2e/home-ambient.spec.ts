import { expect, test, type Page, type Route } from '@playwright/test'

/**
 * Phase 11.3A — home ambient gray release.
 *
 * Default home must be the existing CSS light field. WebGL2 is opt-in through
 * `?ambient=webgl` on `/` only, and every failure path must fall back to CSS.
 * Assertions read real DOM, renderer instance counts and frame counters.
 */

const DATE = '2026-10-21'
const STAMP = '2026-10-21T08:00:00Z'

function emptyHome() {
  return {
    date: DATE,
    context: 'all',
    unclassified_count: 0,
    focus: { kind: 'empty', title: '' },
    daily_focus: { work: null, life: null },
    today: { date: DATE, doing: [], due: [], next: [] },
    planning: {
      items: [], due_today: [], overdue: [], previous_plan: [],
      summary: { planned: 0, due_today: 0, overdue: 0, previous_plan: 0, total: 0 },
    },
    projects: [],
    learning: [],
    inbox: { count: 0, items: [] },
    errors: {},
  }
}

const EMPTY_RECENT = { library: [], events: [], goals: [], asset_date: '', errors: {} }

async function installMocks(page: Page) {
  await page.route('**/api/v1/**', async (route: Route) => {
    const path = new URL(route.request().url()).pathname
    const json = (data: unknown) => route.fulfill({ json: { data } })
    if (path === '/api/v1/auth/me') {
      return json({ id: 1, username: 'ambient', display_name: '灰度', role: 'admin', status: 'active', last_login_at: null, created_at: STAMP })
    }
    if (path === '/api/v1/workspace/home') return json(emptyHome())
    if (path === '/api/v1/workspace/recent') return json(EMPTY_RECENT)
    return json({})
  })
}

async function openHome(page: Page, path = '/') {
  await installMocks(page)
  await page.goto(path, { waitUntil: 'networkidle' })
  await expect(page.locator('.home-ambient')).toBeVisible()
}

interface AmbientDiagnostics {
  state: string
  instances: number
  frames: number
}

async function ambient(page: Page): Promise<AmbientDiagnostics | null> {
  return page.evaluate(() => {
    const target = window as unknown as {
      __homeAmbient?: { state: () => string; instances: () => number; frames: () => number }
    }
    const probe = target.__homeAmbient
    return probe ? { state: probe.state(), instances: probe.instances(), frames: probe.frames() } : null
  })
}

async function loseContext(page: Page): Promise<void> {
  await page.evaluate(() => {
    const canvas = document.querySelector<HTMLCanvasElement>('.home-ambient-canvas')
    const extension = canvas?.getContext('webgl2')?.getExtension('WEBGL_lose_context')
    if (canvas && extension) {
      (canvas as HTMLCanvasElement & { labLoseContext?: WEBGL_lose_context }).labLoseContext = extension
      extension.loseContext()
    }
  })
}

async function restoreContext(page: Page): Promise<void> {
  await page.evaluate(() => {
    const canvas = document.querySelector<HTMLCanvasElement & { labLoseContext?: WEBGL_lose_context }>('.home-ambient-canvas')
    canvas?.labLoseContext?.restoreContext()
  })
}

async function openAppearance(page: Page) {
  await page.locator('.home-account__button').click()
  // The drawer title also contains 外观设置, so target the menu item by role.
  await page.getByRole('menuitem', { name: '外观设置' }).click()
  await expect(page.getByRole('button', { name: '仅 CSS' })).toBeVisible()
}

test('默认首页不初始化 WebGL2，保持现有 CSS 光场', async ({ page }) => {
  await openHome(page)

  await expect(page.locator('.page-header h1')).toHaveText('今天，按自己的节奏')
  await expect(page.locator('.home-light-field')).toBeVisible()
  await expect(page.locator('.home-ambient-canvas')).toHaveCount(0)
  const probe = await ambient(page)
  expect(probe?.state).toBe('css')
  expect(probe?.instances).toBe(0)
})

test('?ambient=webgl 在首页启用 WebGL2', async ({ page }) => {
  await openHome(page, '/?ambient=webgl')

  await expect(page.locator('.home-ambient-canvas')).toHaveClass(/is-visible/)
  await expect(page.locator('.home-light-field')).toBeHidden()
  await expect.poll(async () => (await ambient(page))?.state).toBe('webgl')
  const probe = await ambient(page)
  expect(probe?.instances).toBe(1)
  await expect.poll(async () => (await ambient(page))!.frames).toBeGreaterThan(2)
})

test('?ambient=css 强制回退到 CSS 光场', async ({ page }) => {
  await openHome(page, '/?ambient=css')

  await expect(page.locator('.home-light-field')).toBeVisible()
  await expect(page.locator('.home-ambient-canvas')).toHaveCount(0)
  const probe = await ambient(page)
  expect(probe?.state).toBe('css')
  expect(probe?.instances).toBe(0)
})

test('非首页路由忽略 ?ambient=webgl', async ({ page }) => {
  await installMocks(page)
  await page.goto('/projects?ambient=webgl', { waitUntil: 'networkidle' })
  await expect(page.locator('.home-ambient')).toBeVisible()

  await expect.poll(async () => (await ambient(page))?.state).toBe('css')
  expect((await ambient(page))?.instances).toBe(0)
  await expect(page.locator('.home-ambient-canvas')).toHaveCount(0)
})

test('Reduced Motion 下不创建 WebGL 上下文', async ({ page }) => {
  await page.emulateMedia({ reducedMotion: 'reduce' })
  await openHome(page, '/?ambient=webgl')

  await expect(page.locator('.home-light-field')).toBeVisible()
  const probe = await ambient(page)
  expect(probe?.state).toBe('static')
  expect(probe?.instances).toBe(0)
})

test('WebGL2 不可用时回退 CSS 且无异常', async ({ page }) => {
  const errors: string[] = []
  page.on('pageerror', (error) => errors.push(error.message))
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
  await openHome(page, '/?ambient=webgl')

  await expect(page.locator('.home-light-field')).toBeVisible()
  await expect.poll(async () => (await ambient(page))?.state).toBe('degraded')
  expect((await ambient(page))?.instances).toBe(0)
  await expect(page.locator('.home-ambient-canvas')).toHaveCount(0)
  expect(errors).toEqual([])
})

test('着色器初始化失败时回退 CSS', async ({ page }) => {
  await page.addInitScript(() => {
    const context = (window as unknown as { WebGL2RenderingContext?: { prototype: Record<string, unknown> } }).WebGL2RenderingContext
    if (context?.prototype) {
      context.prototype.getShaderParameter = () => false
    }
  })
  await openHome(page, '/?ambient=webgl')

  await expect(page.locator('.home-light-field')).toBeVisible()
  await expect.poll(async () => (await ambient(page))?.state).toBe('degraded')
  expect((await ambient(page))?.instances).toBe(0)
  await expect(page.locator('.home-ambient-canvas')).toHaveCount(0)
})

test('Context Lost 后恢复成功，不进入降级', async ({ page }) => {
  await openHome(page, '/?ambient=webgl')
  await expect.poll(async () => (await ambient(page))!.frames).toBeGreaterThan(2)

  await loseContext(page)
  const lost = await ambient(page)
  expect(lost?.state).toBe('webgl')

  await restoreContext(page)
  await expect.poll(async () => (await ambient(page))!.frames).toBeGreaterThan((lost?.frames ?? 0) + 2)
  const recovered = await ambient(page)
  expect(recovered?.state).toBe('webgl')
  expect(recovered?.instances).toBe(1)
  await expect(page.locator('.home-ambient-canvas')).toHaveClass(/is-visible/)
})

test('恢复超时后永久降级，CSS 可见且不再绘制', async ({ page }) => {
  await openHome(page, '/?ambient=webgl&ambientRestoreTimeout=100')
  await expect.poll(async () => (await ambient(page))!.frames).toBeGreaterThan(2)

  await loseContext(page)
  await expect.poll(async () => (await ambient(page))!.state, { timeout: 3000 }).toBe('degraded')

  await expect(page.locator('.home-light-field')).toBeVisible()
  await expect(page.locator('.home-ambient-canvas')).toHaveCount(0)

  const frozen = await ambient(page)
  expect(frozen?.instances).toBe(0)
  await page.waitForTimeout(500)
  expect((await ambient(page))!.frames).toBe(frozen!.frames)
})

test('恢复在超时前到达时正常恢复，不进入降级', async ({ page }) => {
  await openHome(page, '/?ambient=webgl&ambientRestoreTimeout=1000')
  await expect.poll(async () => (await ambient(page))!.frames).toBeGreaterThan(2)

  await loseContext(page)
  await page.waitForTimeout(300)
  await restoreContext(page)
  await expect.poll(async () => (await ambient(page))?.state).toBe('webgl')
  const recovered = await ambient(page)
  await page.waitForTimeout(1200)
  // The restored frame must not be overwritten by the late timeout.
  expect((await ambient(page))?.state).toBe('webgl')
  expect((await ambient(page))!.frames).toBeGreaterThan(recovered!.frames)
})

test('超时后卸载，迟到的超时回调不产生副作用', async ({ page }) => {
  await openHome(page, '/?ambient=webgl&ambientRestoreTimeout=100')
  await expect.poll(async () => (await ambient(page))!.frames).toBeGreaterThan(2)

  await loseContext(page)
  // Leave before the deadline: disposal must cancel the pending timeout.
  await page.getByRole('link', { name: '项目' }).click()
  await expect(page).toHaveURL(/\/projects$/)
  await page.waitForTimeout(400)

  const after = await ambient(page)
  expect(after?.state).toBe('css')
  expect(after?.instances).toBe(0)
  await page.waitForTimeout(400)
  expect((await ambient(page))!.frames).toBe(after!.frames)
})

test('主题属性变化不重建渲染器', async ({ page }) => {
  await openHome(page, '/?ambient=webgl')
  const before = await ambient(page)

  await page.evaluate(() => {
    document.documentElement.dataset.theme = 'light'
    document.documentElement.style.colorScheme = 'light'
  })
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'light')
  await page.waitForTimeout(600)

  const after = await ambient(page)
  expect(after?.state).toBe('webgl')
  expect(after?.instances).toBe(1)
  expect(after!.frames).toBeGreaterThan(before!.frames)
})

test('CSS 与 WebGL 连续切换不泄漏实例', async ({ page }) => {
  await openHome(page, '/?ambient=webgl')
  expect((await ambient(page))?.instances).toBe(1)

  for (let index = 0; index < 4; index += 1) {
    await page.evaluate(() => {
      history.pushState({}, '', '/?ambient=css')
      dispatchEvent(new PopStateEvent('popstate', { state: {} }))
    })
    await expect.poll(async () => (await ambient(page))!.instances).toBe(0)
    expect((await ambient(page))?.state).toBe('css')

    await page.evaluate(() => {
      history.pushState({}, '', '/?ambient=webgl')
      dispatchEvent(new PopStateEvent('popstate', { state: {} }))
    })
    await expect.poll(async () => (await ambient(page))!.instances).toBe(1)
    expect((await ambient(page))?.state).toBe('webgl')
  }
})

test('离开首页后无残留渲染循环', async ({ page }) => {
  await openHome(page, '/?ambient=webgl')
  await expect.poll(async () => (await ambient(page))!.frames).toBeGreaterThan(3)

  await page.getByRole('link', { name: '项目' }).click()
  await expect(page).toHaveURL(/\/projects$/)
  // The host releases the canvas on leaving the home route.
  await expect(page.locator('.home-ambient-canvas')).toHaveCount(0)
  await expect(page.locator('.home-light-field')).toBeVisible()

  const left = await ambient(page)
  expect(left?.instances).toBe(0)
  await page.waitForTimeout(600)
  expect((await ambient(page))!.frames).toBe(left!.frames)
})

test('WebGL 画布不拦截交互，首页文字层级不受影响', async ({ page }) => {
  await openHome(page, '/?ambient=webgl')

  const canvas = page.locator('.home-ambient-canvas')
  await expect(canvas).toHaveAttribute('aria-hidden', 'true')
  await expect(canvas).toHaveAttribute('tabindex', '-1')
  const pointerEvents = await canvas.evaluate((element) => getComputedStyle(element).pointerEvents)
  expect(pointerEvents).toBe('none')

  await expect(page.getByRole('link', { name: '工作台' })).toBeVisible()
  await expect(page.locator('.page-header h1')).toHaveText('今天，按自己的节奏')
  await expect(page.getByRole('button', { name: /快速记录/ }).first()).toBeVisible()
})

test('外观设置选择 WebGL 后刷新仍生效', async ({ page }) => {
  await openHome(page)
  await openAppearance(page)
  await page.getByRole('button', { name: 'WebGL' }).click()
  await expect.poll(async () => (await ambient(page))?.state).toBe('webgl')
  expect(await page.evaluate(() => localStorage.getItem('learnos-ambient'))).toBe('webgl')

  await page.reload({ waitUntil: 'networkidle' })
  await expect.poll(async () => (await ambient(page))?.state).toBe('webgl')
  expect((await ambient(page))?.instances).toBe(1)
})

test('选择「仅 CSS」后主题切换与路由往返都不会重新开启', async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem('learnos-ambient', 'webgl'))
  await openHome(page)
  await expect.poll(async () => (await ambient(page))?.state).toBe('webgl')

  await openAppearance(page)
  await page.getByRole('button', { name: '仅 CSS' }).click()
  await expect.poll(async () => (await ambient(page))?.state).toBe('css')
  await page.getByRole('button', { name: '完成' }).click()

  await page.evaluate(() => { document.documentElement.dataset.theme = 'light' })
  await page.waitForTimeout(400)
  expect((await ambient(page))?.state).toBe('css')

  await page.getByRole('link', { name: '项目' }).click()
  await expect(page).toHaveURL(/\/projects$/)
  await page.goBack()
  await page.waitForTimeout(400)
  expect((await ambient(page))?.state).toBe('css')
  expect((await ambient(page))?.instances).toBe(0)
})

test('URL 强制 CSS 覆盖已保存的 WebGL 偏好', async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem('learnos-ambient', 'webgl'))
  await openHome(page, '/?ambient=css')
  expect((await ambient(page))?.state).toBe('css')
  expect((await ambient(page))?.instances).toBe(0)

  await page.goto('/', { waitUntil: 'networkidle' })
  await expect.poll(async () => (await ambient(page))?.state).toBe('webgl')
})

test('URL 的 WebGL 只对本次访问生效，不改写已保存偏好', async ({ page }) => {
  await openHome(page, '/?ambient=webgl')
  await expect.poll(async () => (await ambient(page))?.state).toBe('webgl')
  expect(await page.evaluate(() => localStorage.getItem('learnos-ambient'))).toBeNull()

  await page.goto('/', { waitUntil: 'networkidle' })
  expect((await ambient(page))?.state).toBe('css')
})

test('自动模式在无减少动态效果时启用 WebGL', async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem('learnos-ambient', 'auto'))
  await openHome(page)
  await expect.poll(async () => (await ambient(page))?.state).toBe('webgl')
})

test('自动模式在减少动态效果时退回静态 CSS', async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem('learnos-ambient', 'auto'))
  await page.emulateMedia({ reducedMotion: 'reduce' })
  await openHome(page)

  expect((await ambient(page))?.state).toBe('static')
  expect((await ambient(page))?.instances).toBe(0)
})

test('不支持 WebGL2 时外观设置禁用 WebGL 选项', async ({ page }) => {
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
  await openHome(page)
  await openAppearance(page)
  await expect(page.getByRole('button', { name: 'WebGL' })).toBeDisabled()
  await expect(page.getByText('当前浏览器不支持 WebGL2')).toBeVisible()
})

for (const width of [1440, 390]) {
  test(`Soft Mist V2 光场可见且 CSS / WebGL 不重复叠加 ${width}px`, async ({ page }) => {
    await page.setViewportSize({ width, height: width === 390 ? 844 : 960 })
    await page.addInitScript(() => localStorage.setItem('learnos-theme', 'light'))
    await openHome(page, '/?ambient=css')
    const field = page.locator('.home-light-field')
    const card = page.locator('.glass-hero')
    const box = await card.boundingBox()
    await expect(field).toBeVisible()
    await expect(page.locator('.home-ambient-canvas')).toHaveCount(0)
    const withField = await page.screenshot({ animations: 'disabled' })
    const cardWithField = await card.screenshot({ animations: 'disabled' })
    await field.evaluate((el) => { (el as HTMLElement).style.visibility = 'hidden' })
    const withoutField = await page.screenshot({ animations: 'disabled' })
    const cardWithoutField = await card.screenshot({ animations: 'disabled' })
    expect(withField.equals(withoutField)).toBe(false)
    // Verify transmission inside the card too, not just the exposed gutters.
    expect(cardWithField.equals(cardWithoutField)).toBe(false)
    expect(await card.boundingBox()).toEqual(box)
    await field.evaluate((el) => { (el as HTMLElement).style.visibility = '' })

    await page.goto('/?ambient=webgl', { waitUntil: 'networkidle' })
    await expect.poll(async () => (await ambient(page))?.state).toBe('webgl')
    const canvas = page.locator('.home-ambient-canvas')
    await expect(field).toBeHidden()
    await expect(canvas).toHaveCount(1)
    await expect(canvas).toHaveCSS('opacity', '1')
    const withWebgl = await page.screenshot({ animations: 'disabled' })
    await canvas.evaluate((el) => { (el as HTMLElement).style.visibility = 'hidden' })
    const withoutWebgl = await page.screenshot({ animations: 'disabled' })
    expect(withWebgl.equals(withoutWebgl)).toBe(false)
    expect(await card.boundingBox()).toEqual(box)
    await canvas.evaluate((el) => { (el as HTMLElement).style.visibility = '' })
  })

  test(`首页局部地平线随主题与尺寸同步，不泄漏到业务页 ${width}px`, async ({ page }) => {
    await page.setViewportSize({ width, height: width === 390 ? 844 : 960 })
    await page.addInitScript(() => localStorage.setItem('learnos-theme', 'dark'))
    await openHome(page, '/?ambient=webgl')
    await expect.poll(async () => (await ambient(page))?.state).toBe('webgl')
    const floorUniform = () => page.evaluate(() => {
      const gl = document.querySelector<HTMLCanvasElement>('.home-ambient-canvas')!.getContext('webgl2')!
      const program = gl.getParameter(gl.CURRENT_PROGRAM) as WebGLProgram
      const location = gl.getUniformLocation(program, 'uFloorField')
      if (!location) throw new Error('Ambient floor geometry uniform is missing')
      return Array.from(gl.getUniform(program, location) as Float32Array)
    })
    await expect.poll(async () => (await floorUniform()).map((v) => Math.round(v * 100))).toEqual(width === 390 ? [50, 78, 72, 18] : [52, 78, 40, 16])
    expect(await page.locator('.glass-hero').evaluate((el) => getComputedStyle(el, '::before').backgroundImage)).toBe('none')
    await page.evaluate(() => { document.documentElement.dataset.theme = 'light' })
    await expect.poll(async () => (await floorUniform()).map((v) => Math.round(v * 100))).toEqual(width === 390 ? [46, 112, 130, 58] : [34, 112, 66, 52])
    expect((await ambient(page))?.instances).toBe(1)
    // Uniforms change immediately; the existing CSS surface transition settles later.
    const lightCard = page.locator('.glass-hero')
    // The element itself carries no fill; the feathered ::before layer does.
    await expect(lightCard).toHaveCSS('background-color', 'rgba(0, 0, 0, 0)')
    await expect(lightCard).toHaveCSS('background-image', 'none')
    const feather = await lightCard.evaluate((el) => {
      const style = getComputedStyle(el, '::before')
      return {
        mask: style.maskImage || (style as CSSStyleDeclaration & { webkitMaskImage: string }).webkitMaskImage,
        composite: style.maskComposite || (style as CSSStyleDeclaration & { webkitMaskComposite: string }).webkitMaskComposite,
        layers: (style.backgroundImage.match(/radial-gradient|linear-gradient/g) || []).length,
        inset: style.inset,
        zIndex: style.zIndex,
      }
    })
    // Two per-axis feather masks intersect, so every edge fades instead of ending on a line.
    expect(feather.mask.match(/linear-gradient/g)?.length).toBe(2)
    expect(feather.mask).toContain('rgba(0, 0, 0, 0)')
    expect(feather.composite).toMatch(/intersect|source-in/)
    expect(feather.layers).toBeGreaterThanOrEqual(3)
    expect(feather.inset).toBe('0px')
    // The surface layer must sit behind the card content, never over it.
    expect(feather.zIndex).toBe('-1')
    await expect(lightCard).toHaveCSS('isolation', 'isolate')
    await expect(page.locator('.home-ambient')).toHaveCSS('top', '0px')
    await expect(page.locator('.home-topbar')).toHaveCSS('position', 'sticky')
    await expect(page.locator('.home-topbar')).toHaveCSS('background-color', 'rgba(237, 242, 251, 0.1)')
    // Chromium serializes these faint alpha values through 8-bit rounding.
    await expect(lightCard).toHaveCSS('border-top-color', /rgba\(74, 96, 122, 0\.02[45]\)/)
    await expect(lightCard).toHaveCSS('backdrop-filter', 'blur(6px) saturate(1.06)')
    await expect.poll(() => lightCard.evaluate((el) => getComputedStyle(el).boxShadow)).toMatch(/rgba\(44, 64, 92, 0\.02\)/)
    await expect(page.locator('.home-topbar')).toHaveCSS('border-bottom-color', 'rgba(0, 0, 0, 0)')
    await lightCard.hover()
    await expect(lightCard).toHaveCSS('border-top-color', 'rgba(74, 96, 122, 0.08)')
    await page.locator('.capture-entry').focus()
    await expect(page.locator('.capture-entry')).toHaveCSS('outline-width', '2px')
    await expect(page.locator('.capture-entry')).toHaveCSS('outline-style', 'solid')
    await page.locator('.capture-entry').evaluate((el) => (el as HTMLElement).blur())
    await page.setViewportSize({ width: width === 390 ? 1440 : 390, height: 960 })
    await expect.poll(async () => (await floorUniform()).map((v) => Math.round(v * 100))).toEqual(width === 390 ? [34, 112, 66, 52] : [46, 112, 130, 58])
    // Trigger the existing router link even at the collapsed-nav breakpoint.
    await page.locator('a[href="/projects"]').first().evaluate((el) => (el as HTMLAnchorElement).click())
    await expect(page).toHaveURL(/\/projects$/)
    await expect(page.locator('.home-ambient-host')).not.toHaveClass(/is-home/)
    await expect(page.locator('.home-ambient')).toHaveCSS('top', width === 390 ? '76px' : '68px')
    await expect(page.locator('.home-topbar')).toHaveCSS('background-color', 'rgba(245, 250, 255, 0.58)')
    expect(await page.locator('.home-ambient-host').evaluate((el) => getComputedStyle(el).getPropertyValue('--home-floor-center').trim())).toBe('')
    await expect.poll(async () => (await ambient(page))?.instances).toBe(0)
  })
}
