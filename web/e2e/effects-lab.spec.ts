import { expect, test, type Page, type Route } from '@playwright/test'

/** Phase 11.4B — admin-only Visual Effects Lab. */

const STAMP = '2026-10-21T08:00:00Z'

async function mock(page: Page, role: 'admin' | 'user' | null) {
  await page.route('**/api/v1/**', (route: Route) => {
    const path = new URL(route.request().url()).pathname
    if (path === '/api/v1/auth/me') {
      if (role === null) return route.fulfill({ status: 401, json: { error: 'authentication required' } })
      return route.fulfill({ json: { data: { id: 1, username: role, display_name: role === 'admin' ? '管理员' : '普通用户', role, status: 'active', last_login_at: null, created_at: STAMP } } })
    }
    return route.fulfill({ json: { data: {} } })
  })
}

async function openLab(page: Page, role: 'admin' | 'user' = 'admin') {
  await mock(page, role)
  await page.goto('/admin/visual-effects', { waitUntil: 'networkidle' })
}

test('管理员可见入口并进入实验室', async ({ page }) => {
  await mock(page, 'admin')
  await page.goto('/', { waitUntil: 'networkidle' })
  await page.locator('.home-account__button').click()
  await expect(page.getByRole('menuitem', { name: '视觉实验室' })).toBeVisible()
  await page.getByRole('menuitem', { name: '视觉实验室' }).click()
  await expect(page).toHaveURL(/\/admin\/visual-effects$/)
  await expect(page.locator('.fx-lab')).toBeVisible()
  await expect(page.locator('[data-testid^="fx-effect-"]')).toHaveCount(4)

  // Direct refresh of the admin route must keep working (history fallback).
  await page.reload({ waitUntil: 'networkidle' })
  await expect(page.locator('.fx-lab')).toBeVisible()
})

test('普通用户看不到入口', async ({ page }) => {
  await mock(page, 'user')
  await page.goto('/', { waitUntil: 'networkidle' })
  await page.locator('.home-account__button').click()
  await expect(page.getByRole('menuitem', { name: '视觉实验室' })).toHaveCount(0)
})

test('普通用户直接访问实验室被拒绝', async ({ page }) => {
  await openLab(page, 'user')
  await expect(page).toHaveURL(/\/forbidden$/)
  await expect(page.locator('.fx-lab')).toHaveCount(0)
  await expect(page.getByRole('heading', { name: '没有访问权限' })).toBeVisible()
})

test('普通用户无法通过查询参数或本地存储绕过', async ({ page }) => {
  await page.addInitScript(() => {
    localStorage.setItem('learnos:lab:effects', JSON.stringify({ effect: 'ambient-gradient', enabled: true }))
    localStorage.setItem('learnos-ambient', 'webgl')
  })
  await mock(page, 'user')
  await page.goto('/admin/visual-effects?effect=ambient-gradient&enabled=true', { waitUntil: 'networkidle' })
  await expect(page).toHaveURL(/\/forbidden$/)
  await expect(page.locator('.fx-lab')).toHaveCount(0)
})

test('普通用户直接访问设置与用户管理被拒绝', async ({ page }) => {
  await mock(page, 'user')
  await page.goto('/settings', { waitUntil: 'networkidle' })
  await expect(page).toHaveURL(/\/forbidden$/)
  await page.goto('/users', { waitUntil: 'networkidle' })
  await expect(page).toHaveURL(/\/forbidden$/)
})

test('未登录用户被送到登录页', async ({ page }) => {
  await mock(page, null)
  await page.goto('/admin/visual-effects', { waitUntil: 'domcontentloaded' })
  // The 401 handler performs a full navigation; wait for the destination instead of the URL event.
  await expect(page.locator('input[type="password"]')).toBeVisible({ timeout: 15000 })
  await expect(page).toHaveURL(/\/login$/)
})

test('管理员默认预览 Spotlight，且卡片带反光层', async ({ page }) => {
  await openLab(page)
  const card = page.locator('.fx-card').first()
  await expect(card).toBeVisible()
  await expect(card).toHaveClass(/fx-spotlight/)
  expect(await card.getAttribute('style')).toContain('--fx-intensity')
})

test('切换 Ambient 后进入 WebGL 并显示技术类型', async ({ page }) => {
  await openLab(page)
  await page.getByRole('button', { name: /Ambient Gradient/ }).click()
  await expect(page.locator('.fx-ambient')).toHaveAttribute('data-state', 'webgl', { timeout: 10000 })
  await expect(page.locator('.fx-lab__tech')).toContainText('WebGL')
})

test('对比模式并排显示开关两个版本', async ({ page }) => {
  await openLab(page)
  await page.getByTestId('fx-compare').click()
  await expect(page.locator('.fx-lab__panel')).toHaveCount(2)
})

test('模拟减少动态效果时预览进入静态模式', async ({ page }) => {
  await openLab(page)
  await page.getByTestId('fx-reduced').click()
  await expect(page.locator('.fx-preview').first()).toHaveAttribute('data-fx-reduced', '')
})

test('关闭动画后 Ambient 回退到 CSS 光场', async ({ page }) => {
  await openLab(page)
  await page.getByRole('button', { name: /Ambient Gradient/ }).click()
  await expect(page.locator('.fx-ambient')).toHaveAttribute('data-state', 'webgl', { timeout: 10000 })
  await page.getByTestId('fx-animate').click()
  await expect(page.locator('.fx-ambient')).toHaveAttribute('data-state', 'css')
  await expect(page.locator('.fx-ambient canvas')).toHaveCount(0)
  await expect(page.locator('.fx-preview')).toHaveCount(1)
})

test('离开实验室后不残留渲染上下文', async ({ page }) => {
  await openLab(page)
  await page.getByRole('button', { name: /Ambient Gradient/ }).click()
  await expect(page.locator('.fx-ambient')).toHaveAttribute('data-state', 'webgl', { timeout: 10000 })
  await page.goto('/projects', { waitUntil: 'networkidle' })
  await expect(page.locator('.fx-ambient')).toHaveCount(0)
})

test('四种效果都在目录中，且 CSS 效果不创建画布', async ({ page }) => {
  await openLab(page)
  await expect(page.locator('[data-testid^="fx-effect-"]')).toHaveCount(4)
  await page.getByTestId('fx-effect-surface-depth').click()
  await expect(page.getByTestId('fx-preview').locator('.fx-depth').first()).toBeVisible()
  expect(await page.locator('canvas').count()).toBe(0)
  await page.getByTestId('fx-effect-soft-reveal').click()
  expect(await page.locator('canvas').count()).toBe(0)
})

test('切换效果会清理上一个效果的渲染资源', async ({ page }) => {
  await openLab(page)
  await page.getByTestId('fx-effect-ambient-gradient').click()
  await expect(page.locator('.fx-ambient')).toHaveAttribute('data-state', 'webgl', { timeout: 10000 })
  await page.getByTestId('fx-effect-spotlight-surface').click()
  await expect(page.locator('.fx-ambient')).toHaveCount(0)
  expect(await page.locator('canvas').count()).toBe(0)
})

test('参数调整立即改变真实效果并可恢复默认', async ({ page }) => {
  await openLab(page)
  const preview = page.getByTestId('fx-preview')
  const config = page.getByTestId('fx-config')
  const slider = page.getByTestId('fx-param-radius').getByRole('slider')
  await slider.focus()
  await slider.press('ArrowRight')
  await expect.poll(async () => await config.innerText()).toContain('"radius": 190')
  await expect.poll(async () => await preview.getAttribute('style')).toContain('--fx-spotlight-radius: 190px')

  await page.getByTestId('fx-reset').click()
  await expect.poll(async () => await config.innerText()).toContain('"radius": 180')
  await expect.poll(async () => await preview.getAttribute('style')).toContain('--fx-spotlight-radius: 180px')
})

test('使用示例与实际组件 Props 一致', async ({ page }) => {
  await openLab(page)
  const usage = page.getByTestId('fx-usage')
  await expect(usage).toContainText('SpotlightSurface')
  await expect(usage).toContainText(':radius="180"')
  await page.getByTestId('fx-effect-soft-reveal').click()
  await expect(usage).toContainText('SoftReveal')
  await expect(usage).toContainText(':duration="220"')
  await page.getByTestId('fx-effect-ambient-gradient').click()
  await expect(usage).toContainText('AmbientSurface')
  await expect(usage).toContainText('quality="high"')
})

test('复制配置与实际参数一致', async ({ page, context }) => {
  await context.grantPermissions(['clipboard-read', 'clipboard-write'])
  await openLab(page)
  await page.getByTestId('fx-copy-config').click()
  const copied = await page.evaluate(() => navigator.clipboard.readText())
  const config = JSON.parse(await page.getByTestId('fx-config').innerText())
  expect(JSON.parse(copied)).toEqual(config)
  expect(JSON.parse(copied)).toMatchObject({ intensity: 0.6, radius: 180, speed: 160, tone: 'neutral' })
})

test('深浅色切换作用于预览与页面', async ({ page }) => {
  await openLab(page)
  await page.locator('.fx-lab__side').getByText('浅色', { exact: true }).click()
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'light')
  await page.locator('.fx-lab__side').getByText('深色', { exact: true }).click()
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark')
})

test('窄屏无横向溢出', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  await openLab(page)
  await expect(page.locator('.fx-lab__side')).toBeVisible()
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth + 1)).toBe(true)
})
