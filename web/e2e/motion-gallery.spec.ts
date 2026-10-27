import { expect, test, type Page, type Route } from '@playwright/test'

/** Phase 11.5B — Motion Gallery inside the admin Visual Effects Lab. */

const STAMP = '2026-10-21T08:00:00Z'

async function mock(page: Page, role: 'admin' | 'user' = 'admin') {
  await page.route('**/api/v1/**', (route: Route) => {
    const path = new URL(route.request().url()).pathname
    if (path === '/api/v1/auth/me') {
      return route.fulfill({ json: { data: { id: 1, username: role, display_name: role === 'admin' ? '管理员' : '普通用户', role, status: 'active', last_login_at: null, created_at: STAMP } } })
    }
    return route.fulfill({ json: { data: {} } })
  })
}

async function openMotion(page: Page, role: 'admin' | 'user' = 'admin') {
  await mock(page, role)
  await page.goto('/admin/visual-effects', { waitUntil: 'networkidle' })
  await expect(page.locator('.fx-lab')).toBeVisible()
  await page.locator('.fx-lab__modes').getByText('动效', { exact: true }).click()
  await expect(page.locator('.mg')).toBeVisible()
}

test('管理员可以打开动效分类，看到六个动效', async ({ page }) => {
  await openMotion(page)
  await expect(page.locator('[data-testid^="mg-motion-"]')).toHaveCount(6)
  await expect(page.locator('.ms-stage')).toHaveCount(1)
  expect(await page.locator('canvas').count()).toBe(0)
})

test('切回效果分类仍然只有四个效果', async ({ page }) => {
  await openMotion(page)
  await page.locator('.fx-lab__modes').getByText('效果', { exact: true }).click()
  await expect(page.locator('[data-testid^="fx-effect-"]')).toHaveCount(4)
})

test('时长参数真实作用到预览', async ({ page }) => {
  await openMotion(page)
  await page.getByTestId('mg-motion-content-reveal').click()
  await page.getByTestId('mg-duration').getByText('300ms').click()
  await expect.poll(async () => await page.locator('.ms-stage').getAttribute('style')).toContain('--motion-duration: 300ms')
})

test('重播触发进入动画，动画结束后移除状态', async ({ page }) => {
  await openMotion(page)
  await page.getByTestId('mg-motion-content-reveal').click()
  await page.getByTestId('mg-replay').click()
  await expect(page.locator('.motion-reveal.is-entering')).toHaveCount(1)
  await expect.poll(async () => await page.locator('.motion-reveal.is-entering').count(), { timeout: 3000 }).toBe(0)
})

test('模拟 Reduced Motion 关闭动画但保留内容', async ({ page }) => {
  await openMotion(page)
  await page.getByTestId('mg-motion-content-reveal').click()
  await page.getByTestId('mg-reduced').click()
  await expect(page.locator('.ms-stage')).toHaveAttribute('data-motion-reduced', '')
  await page.getByTestId('mg-replay').click()
  await expect(page.locator('.motion-reveal.is-entering')).toHaveCount(1)
  const animation = await page.locator('.motion-reveal > *').first().evaluate((element) => getComputedStyle(element).animationName)
  expect(animation).toBe('none')
  await expect(page.locator('.ms-list__row').first()).toBeVisible()
})

test('面板显隐使用真实过渡并正确移除', async ({ page }) => {
  await openMotion(page)
  await page.getByTestId('mg-motion-panel-presence').click()
  await page.getByTestId('mg-panel-toggle').click()
  await expect(page.getByTestId('mg-panel')).toBeVisible()
  await page.getByTestId('mg-panel-toggle').click()
  await expect(page.getByTestId('mg-panel')).toHaveCount(0)
})

test('复制的示例与实际参数一致', async ({ page, context }) => {
  await context.grantPermissions(['clipboard-read', 'clipboard-write'])
  await openMotion(page)
  await page.getByTestId('mg-motion-content-reveal').click()
  await page.getByTestId('mg-duration').getByText('180ms').click()
  const usage = await page.getByTestId('mg-usage').innerText()
  expect(usage).toContain('MotionReveal')
  expect(usage).toContain(':duration="180"')
  await page.getByTestId('mg-copy').click()
  const copied = await page.evaluate(() => navigator.clipboard.readText())
  const normalize = (text: string) => text.replace(/\s+$/gm, '').trim()
  expect(normalize(copied)).toBe(normalize(usage))
})

test('对比模式并排显示默认与当前参数', async ({ page }) => {
  await openMotion(page)
  await page.getByTestId('mg-compare').click()
  await expect(page.locator('.ms-stage')).toHaveCount(2)
})

test('恢复默认值', async ({ page }) => {
  await openMotion(page)
  await page.getByTestId('mg-motion-content-reveal').click()
  await page.getByTestId('mg-duration').getByText('300ms').click()
  await page.getByTestId('mg-distance').getByRole('slider').press('ArrowRight')
  await page.getByTestId('mg-reset').click()
  await expect.poll(async () => await page.locator('.ms-stage').getAttribute('style')).toContain('--motion-duration: 160ms')
  await expect.poll(async () => await page.locator('.ms-stage').getAttribute('style')).toContain('--motion-distance: 4px')
})

test('深浅色切换正常', async ({ page }) => {
  await openMotion(page)
  await page.getByTestId('mg-theme').getByText('浅色', { exact: true }).click()
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'light')
  await page.getByTestId('mg-theme').getByText('深色', { exact: true }).click()
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark')
})

test('窄屏无横向溢出', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  await openMotion(page)
  await expect(page.locator('.mg__side')).toBeVisible()
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth + 1)).toBe(true)
})

test('普通用户无法进入动效实验室', async ({ page }) => {
  await mock(page, 'user')
  await page.goto('/admin/visual-effects', { waitUntil: 'networkidle' })
  await expect(page).toHaveURL(/\/forbidden$/)
  await expect(page.locator('.mg')).toHaveCount(0)
})

test('切换动效与参数不产生写请求', async ({ page }) => {
  const writes: string[] = []
  page.on('request', (request) => {
    if (request.method() !== 'GET' && request.url().includes('/api/v1/')) writes.push(`${request.method()} ${request.url()}`)
  })
  await openMotion(page)
  for (const id of ['card-hover', 'tab-transition', 'content-reveal', 'panel-presence', 'timing-comparison']) {
    await page.getByTestId(`mg-motion-${id}`).click()
  }
  await page.getByTestId('mg-replay').click()
  expect(writes).toEqual([])
})
