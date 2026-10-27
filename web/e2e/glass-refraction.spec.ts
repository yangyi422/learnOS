import { expect, test, type Page, type Route } from '@playwright/test'

test.setTimeout(60_000)

/** Phase 11.6C — admin-only WebGL2 refraction on the focus card. */

const STAMP = '2026-10-21T08:00:00Z'
const focus = {
  id: 11, date: '2026-10-26', context: 'work', task_id: 1, revision: 1, state: 'ok',
  title: '完成首页信息架构评审并给出修改清单', task_status: 'doing', priority: 'high',
  plan_date: '2026-10-26', due_date: '2026-10-30', completed_at: null,
  project_id: 1, project_title: '产品官网重构', project_context: 'work', project_status: 'active',
  state_reason: '', updated_at: STAMP, task_updated_at: STAMP,
}

async function mock(page: Page, role: 'admin' | 'user' = 'admin') {
  await page.route('**/api/v1/**', (route: Route) => {
    const path = new URL(route.request().url()).pathname
    const json = (data: unknown) => route.fulfill({ json: { data } })
    if (path === '/api/v1/auth/me') return json({ id: 1, username: role, display_name: role === 'admin' ? '管理员' : '普通用户', role, status: 'active', last_login_at: null, created_at: STAMP })
    if (path === '/api/v1/workspace/home') return json({
      date: '2026-10-26', context: 'all', unclassified_count: 0, focus: null,
      daily_focus: { work: focus, life: null },
      today: { date: '2026-10-26', doing: [], due: [], next: [] },
      planning: { items: [], due_today: [], overdue: [], previous_plan: [], summary: { planned: 0, due_today: 0, overdue: 0, previous_plan: 0, total: 0 } },
      projects: [], learning: [], inbox: { count: 0, items: [] }, errors: {},
    })
    if (path === '/api/v1/workspace/recent') return json({ library: [], events: [], goals: [], asset_date: '2026-10-01', errors: {} })
    return json([])
  })
}

async function openHome(page: Page) {
  await mock(page)
  await page.goto('/?ambient=webgl', { waitUntil: 'networkidle' })
  await page.waitForFunction(() => (window as unknown as { __homeAmbient?: { state: () => string } }).__homeAmbient?.state() === 'webgl')
  await page.getByTestId('fx-pilot-toggle').click()
  await expect(page.getByTestId('fx-pilot-refraction')).toBeVisible()
}

async function toggle(page: Page, on: boolean) {
  const active = (await page.locator('.focus-card.glass-refract').count()) > 0
  if (active !== on) await page.getByTestId('fx-pilot-refraction').click()
  await page.waitForTimeout(360)
}

async function sample(page: Page) {
  return page.evaluate(() => {
    const canvas = document.querySelector('.home-ambient-canvas') as HTMLCanvasElement | null
    const card = document.querySelector('.focus-card') as HTMLElement | null
    if (!canvas || !card) return null
    const copy = document.createElement('canvas')
    copy.width = canvas.width
    copy.height = canvas.height
    const context = copy.getContext('2d')!
    context.drawImage(canvas, 0, 0)
    const canvasRect = canvas.getBoundingClientRect()
    const rect = card.getBoundingClientRect()
    const scaleX = canvas.width / canvasRect.width
    const scaleY = canvas.height / canvasRect.height
    const x = Math.round((rect.left - canvasRect.left) * scaleX)
    const y = Math.round((rect.top - canvasRect.top) * scaleY)
    const w = Math.round(rect.width * scaleX)
    const h = Math.round(rect.height * scaleY)
    return { w, h, data: Array.from(context.getImageData(x, y, w, h).data) }
  })
}

function compare(a: { w: number; h: number; data: number[] }, b: { w: number; h: number; data: number[] }) {
  const band = 24
  let edge = 0, edgeN = 0, centre = 0, centreN = 0
  for (let y = 0; y < a.h; y++) {
    for (let x = 0; x < a.w; x++) {
      const i = (y * a.w + x) * 4
      const delta = Math.abs(a.data[i]! - b.data[i]!) + Math.abs(a.data[i + 1]! - b.data[i + 1]!) + Math.abs(a.data[i + 2]! - b.data[i + 2]!)
      if (x < band || y < band || x > a.w - band || y > a.h - band) { edge += delta; edgeN += 1 }
      if (x > a.w * .3 && x < a.w * .7 && y > a.h * .3 && y < a.h * .7) { centre += delta; centreN += 1 }
    }
  }
  return { edge: edge / edgeN / 3, centre: centre / centreN / 3 }
}

test('管理员开启折射后，边缘背景被采样位移，中心保持稳定', async ({ page }) => {
  await openHome(page)
  await toggle(page, false)
  const off = await sample(page)
  await toggle(page, true)
  await expect(page.locator('.focus-card')).toHaveClass(/glass-refract/)
  let on = await sample(page)
  expect(off && on).toBeTruthy()
  let result = compare(off!, on!)
  // The ambient loop can be a frame behind under load; take the best of a few samples.
  for (let attempt = 0; attempt < 4 && result.edge <= 1; attempt += 1) {
    await page.waitForTimeout(200)
    on = await sample(page)
    if (on) result = compare(off!, on)
  }
  expect(result.edge).toBeGreaterThan(1)
  expect(result.edge).toBeGreaterThan(result.centre * 2)
  await toggle(page, false)
  await expect(page.locator('.focus-card')).not.toHaveClass(/glass-refract/)
})

test('Soft Mist 浅色只绘制 Ambient，切回深色才恢复实验折射', async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem('learnos-theme', 'light'))
  await openHome(page)
  await toggle(page, true)
  const compositeActive = () => page.evaluate(() => {
    const gl = document.querySelector<HTMLCanvasElement>('.home-ambient-canvas')!.getContext('webgl2')!
    const program = gl.getParameter(gl.CURRENT_PROGRAM) as WebGLProgram
    return gl.getUniformLocation(program, 'uBg') !== null
  })
  await expect.poll(compositeActive).toBe(false)
  await expect(page.locator('.focus-card')).toHaveCSS('background-color', 'rgba(0, 0, 0, 0)')
  await expect(page.locator('.home-ambient-canvas')).toHaveCount(1)
  await page.evaluate(() => { document.documentElement.dataset.theme = 'dark' })
  await expect.poll(compositeActive).toBe(true)
  await expect(page.locator('.home-ambient-canvas')).toHaveCount(1)
})

test('折射开启时只创建一个 Ambient 画布，关闭后回到 CSS', async ({ page }) => {
  await mock(page)
  await page.goto('/', { waitUntil: 'networkidle' })
  await page.waitForSelector('.focus-card')
  await page.getByTestId('fx-pilot-toggle').click()
  await toggle(page, true)
  await page.waitForSelector('.home-ambient-canvas', { timeout: 15000 })
  expect(await page.locator('.home-ambient-canvas').count()).toBe(1)
  await expect(page.locator('.focus-card')).toHaveClass(/glass-refract/)
  await toggle(page, false)
  await expect(page.locator('.home-ambient-canvas')).toHaveCount(0)
})

test('Reduced Motion 下保留静态玻璃，不创建折射画布', async ({ page }) => {
  await page.emulateMedia({ reducedMotion: 'reduce' })
  await mock(page)
  await page.goto('/', { waitUntil: 'networkidle' })
  await page.waitForSelector('.focus-card')
  await page.getByTestId('fx-pilot-toggle').click()
  await toggle(page, true)
  await expect(page.locator('.focus-card')).toHaveClass(/glass-refract/)
  await expect(page.locator('.home-ambient-canvas')).toHaveCount(0)
})

test('普通用户看不到折射开关', async ({ page }) => {
  await mock(page, 'user')
  await page.goto('/', { waitUntil: 'networkidle' })
  await expect(page.getByTestId('fx-pilot-bar')).toHaveCount(0)
  await expect(page.getByTestId('fx-pilot-refraction')).toHaveCount(0)
  await expect(page.locator('.focus-card.glass-refract')).toHaveCount(0)
})
