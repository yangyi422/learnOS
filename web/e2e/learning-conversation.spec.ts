import { expect, test, type Page } from '@playwright/test'

async function installConversationMocks(page: Page) {
  let lessonID = 101
  const conversations = new Map<number, { turns: Record<string, unknown>[]; completion_suggested: boolean; position: number; total: number }>()
  const savedKeys = new Set<string>()
  const state = { calls: 0, advances: 0, failNext: false }
  const getConversation = (id: number) => {
    if (!conversations.has(id)) conversations.set(id, { turns: [], completion_suggested: false, position: id - 100, total: 3 })
    return conversations.get(id)!
  }
  await page.route('**/api/v1/**', async route => {
    const req = route.request()
    const path = new URL(req.url()).pathname
    let data: unknown = {}
    if (path.endsWith('/auth/me')) data = { id: 1, username: 'test', role: 'admin' }
    else if (path.endsWith('/current-lesson')) data = {
      course: { id: 1, name: '营养学' }, unit: { id: 10, title: '能量消耗', objective: '区分能量消耗' },
      lesson: { id: lessonID, title: `课程 ${lessonID - 100}`, content: '基础代谢是维持生命活动的最低能量消耗。例如呼吸和心跳需要能量。运动增加的消耗属于活动消耗。', expected_understanding: '区分基础代谢和活动消耗', core_question: '运动消耗属于基础代谢吗？', status: 'learning', content_role: 'core', depth_level: 1 },
    }
    else if (path.endsWith('/conversation')) {
      const id = Number(path.split('/').at(-2))
      const conversation = getConversation(id)
      if (req.method() === 'POST') {
        state.calls++
        const body = req.postDataJSON() as { message: string; idempotency_key: string }
        if (!savedKeys.has(body.idempotency_key)) {
          savedKeys.add(body.idempotency_key)
          conversation.turns.push({ id: conversation.turns.length + 1, user_answer: body.message, feedback: '', result: 'pending', evaluation_source: 'ai', turn_kind: 'conversation' })
        }
        if (state.failNext) {
          state.failNext = false
          await route.fulfill({ status: 504, contentType: 'application/json', body: JSON.stringify({ error: { code: 'AI_TIMEOUT', message: '超时', retryable: true } }) })
          return
        }
        const turn = conversation.turns.find(turn => turn.user_answer === body.message)!
        turn.feedback = body.message.includes('维持生命') ? '你能区分两种消耗，本课目标基本达成。' : '活动消耗和基础代谢不同，我们可以用散步来解释。'
        turn.result = 'replied'
        if (body.message.includes('维持生命')) conversation.completion_suggested = true
      }
      data = conversation
    }
    else if (path.endsWith('/retry')) {
      state.calls++
      const parts = path.split('/')
      const conversation = getConversation(Number(parts.at(-4)))
      const turn = conversation.turns.find(turn => turn.id === Number(parts.at(-2)))!
      turn.feedback = '活动消耗和基础代谢不同，我们可以用散步来解释。'
      turn.result = 'replied'
      data = conversation
    }
    else if (path.endsWith('/advance')) { state.advances++; lessonID++; data = {} }
    else if (path.endsWith('/learning-turns') || path.endsWith('/misconceptions')) data = []
    else if (path.endsWith('/cognitive-state')) data = { state: { current_level: 'unseen', status: 'unknown' }, evidence: [], timeline: [] }
    else if (path.endsWith('/relations')) data = { prerequisites: [], extensions: [], applications: [], related: [] }
    await route.fulfill({ contentType: 'application/json', body: JSON.stringify({ data }) })
  })
  return state
}

test('自由提问、多轮对话、稳定完成建议与一键推进', async ({ page }) => {
  const state = await installConversationMocks(page)
  await page.goto('/courses/1/learn')
  await expect(page.getByText('基础代谢是维持生命活动的最低能量消耗。', { exact: false })).toBeVisible()
  const input = page.locator('#lesson-answer')
  await input.fill('我不理解，运动消耗算吗？')
  await page.getByRole('button', { name: '发送', exact: true }).click()
  await expect(page.locator('.conversation-stream')).toContainText('散步')
  await expect(page.getByRole('button', { name: '跳过并进入下一课' })).toBeVisible()
  await page.getByRole('button', { name: '展开课件' }).click()
  await expect(page.locator('.lesson-copy')).toBeVisible()
  await input.fill('基础代谢是维持生命所需的最低能量消耗，运动属于额外的活动消耗。')
  await page.getByRole('button', { name: '发送', exact: true }).click()
  await expect(page.getByRole('button', { name: '完成并进入下一课' })).toBeVisible()
  await input.fill('再讲讲其他相关知识')
  await page.getByRole('button', { name: '发送', exact: true }).click()
  await expect(page.getByRole('button', { name: '完成并进入下一课' })).toBeVisible()
  await page.reload()
  await expect(page.locator('.conversation-turn')).toHaveCount(3)
  await page.getByRole('button', { name: '完成并进入下一课' }).evaluate(button => { (button as HTMLButtonElement).click(); (button as HTMLButtonElement).click() })
  await expect(page.locator('h1')).toContainText('课程 2')
  expect(state.advances).toBe(1)
  await expect(page.locator('.conversation-turn')).toHaveCount(0)
})

test('超时保留消息，重试不重复写入，未验证时可以跳过', async ({ page }) => {
  const state = await installConversationMocks(page)
  await page.goto('/courses/1/learn')
  state.failNext = true
  await page.locator('#lesson-answer').fill('这个概念为什么重要？')
  await page.getByRole('button', { name: '发送', exact: true }).click()
  await expect(page.getByText('AI 回复未完成，你的输入已保留，可重试。')).toBeVisible()
  await expect(page.locator('.conversation-turn')).toHaveCount(1)
  page.once('dialog', dialog => dialog.accept())
  await page.reload()
  await expect(page.locator('#lesson-answer')).toHaveValue('这个概念为什么重要？')
  await page.getByRole('button', { name: '重试回复' }).click()
  await expect(page.locator('#lesson-answer')).toHaveValue('')
  await expect(page.locator('.conversation-turn')).toHaveCount(1)
  expect(state.calls).toBe(2)
  await page.getByRole('button', { name: '跳过并进入下一课' }).click()
  await expect(page.locator('h1')).toContainText('课程 2')
})

for (const width of [375, 820, 1440]) {
  test(`${width}px 输入与推进始终可见且没有横向溢出`, async ({ page }, testInfo) => {
    await installConversationMocks(page)
    await page.setViewportSize({ width, height: 800 })
    await page.goto('/courses/1/learn')
    await expect(page.locator('#lesson-answer')).toBeVisible()
    const button = page.getByRole('button', { name: '跳过并进入下一课' })
    await expect(button).toBeInViewport()
    await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight))
    await expect(button).toBeInViewport()
    expect(await page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1)
    if (process.env.CAPTURE_VISUALS) {
      await page.screenshot({ path: testInfo.outputPath('learning-light.png'), fullPage: true })
      await page.evaluate(() => document.documentElement.dataset.theme = 'dark')
      await page.screenshot({ path: testInfo.outputPath('learning-dark.png'), fullPage: true })
    }
  })
}
