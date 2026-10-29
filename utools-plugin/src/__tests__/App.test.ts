import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import App from '../App.vue'
import type { LifeosBridge } from '../bridge'

const ITEM = {
  id: 1,
  content: '想法',
  source_type: 'manual',
  source_url: '',
  status: 'inbox',
  created_at: '2026-10-29T14:00:00+08:00',
}

interface Harness {
  bridge: LifeosBridge & {
    setConfig: ReturnType<typeof vi.fn>
    readClipboard: ReturnType<typeof vi.fn>
    getDraft: ReturnType<typeof vi.fn>
    saveDraft: ReturnType<typeof vi.fn>
    clearDraft: ReturnType<typeof vi.fn>
    onOut: ReturnType<typeof vi.fn>
    onEnter: ReturnType<typeof vi.fn>
    hideWindow: ReturnType<typeof vi.fn>
  }
  store: { baseUrl: string; token: string; draft: string }
}

function setupBridge(draft = ''): Harness {
  const store = { baseUrl: 'http://127.0.0.1:8080', token: 'los_pat_test', draft }
  const bridge = {
    isDev: false,
    getConfig: vi.fn(() => ({ baseUrl: store.baseUrl, token: store.token })),
    setConfig: vi.fn((config: { baseUrl: string; token: string }) => {
      store.baseUrl = config.baseUrl
      store.token = config.token
    }),
    readClipboard: vi.fn(() => ''),
    getDraft: vi.fn(() => store.draft),
    saveDraft: vi.fn((text: string) => {
      store.draft = text
    }),
    clearDraft: vi.fn(() => {
      store.draft = ''
    }),
    onEnter: vi.fn(),
    onOut: vi.fn(),
    hideWindow: vi.fn(),
  }
  return { bridge, store }
}

function fakeResponse(status: number, body: unknown): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: async () => (body === undefined ? '' : JSON.stringify(body)),
  } as unknown as Response
}

const mounted: VueWrapper[] = []

async function mountApp(harness: Harness): Promise<VueWrapper> {
  ;(window as unknown as { lifeos: LifeosBridge }).lifeos = harness.bridge
  const wrapper = mount(App)
  mounted.push(wrapper)
  await flushPromises()
  return wrapper
}

function bodyOf(fetchMock: ReturnType<typeof vi.fn>, index: number): { content: string; capture_key: string } {
  return JSON.parse(fetchMock.mock.calls[index][1].body)
}

async function submit(wrapper: VueWrapper, text: string): Promise<void> {
  await wrapper.get('textarea').setValue(text)
  await wrapper.get('form.composer').trigger('submit')
  await flushPromises()
}

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))

afterEach(() => {
  mounted.splice(0).forEach((wrapper) => wrapper.unmount())
  vi.unstubAllGlobals()
  vi.unstubAllEnvs()
  delete (window as unknown as { lifeos?: LifeosBridge }).lifeos
})

describe('LifeOS capture composer', () => {
  it('sends one capture request, then clears the input and closes the window', async () => {
    const harness = setupBridge()
    const fetchMock = vi.fn().mockResolvedValue(fakeResponse(201, { data: ITEM }))
    vi.stubGlobal('fetch', fetchMock)

    const wrapper = await mountApp(harness)
    expect(wrapper.get('textarea').element.value).toBe('')

    await submit(wrapper, '先做困难的事')

    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('http://127.0.0.1:8080/api/v1/capture/inbox')
    expect(init.method).toBe('POST')
    expect(init.headers.Authorization).toBe('Bearer los_pat_test')
    expect(init.headers['Content-Type']).toBe('application/json')
    expect(bodyOf(fetchMock, 0)).toEqual({
      content: '先做困难的事',
      capture_key: expect.stringMatching(/^[A-Za-z0-9_-]{8,80}$/),
    })

    expect(wrapper.get('textarea').element.value).toBe('')
    expect(wrapper.text()).toContain('已进入收集箱')
    expect(harness.bridge.clearDraft).toHaveBeenCalled()

    await sleep(520)
    expect(harness.bridge.hideWindow).toHaveBeenCalledTimes(1)
  })

  it('keeps the input on 401 and does not retry by itself', async () => {
    const harness = setupBridge()
    const fetchMock = vi
      .fn()
      .mockResolvedValue(fakeResponse(401, { error: { code: 'UNAUTHORIZED', message: '认证失败' } }))
    vi.stubGlobal('fetch', fetchMock)

    const wrapper = await mountApp(harness)
    await submit(wrapper, '一条没发出去的想法')

    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(wrapper.get('textarea').element.value).toBe('一条没发出去的想法')
    expect(wrapper.text()).toContain('凭证')
    expect(wrapper.text()).not.toContain('已进入收集箱')
    expect(wrapper.get('button.composer__submit').text()).toBe('去配置')

    await sleep(520)
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it('keeps the input after a 5xx and reuses the same capture_key on retry', async () => {
    const harness = setupBridge()
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(fakeResponse(503, undefined))
      .mockResolvedValueOnce(fakeResponse(201, { data: ITEM }))
    vi.stubGlobal('fetch', fetchMock)

    const wrapper = await mountApp(harness)
    await submit(wrapper, '重试也要只存一次')

    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(wrapper.get('textarea').element.value).toBe('重试也要只存一次')
    expect(wrapper.text()).toContain('稍后重试')
    expect(wrapper.get('button.composer__submit').text()).toBe('重试')

    await wrapper.get('form.composer').trigger('submit')
    await flushPromises()

    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(bodyOf(fetchMock, 0).capture_key).toBe(bodyOf(fetchMock, 1).capture_key)
    expect(bodyOf(fetchMock, 1).content).toBe('重试也要只存一次')
    expect(wrapper.get('textarea').element.value).toBe('')
    expect(wrapper.text()).toContain('已进入收集箱')
  })

  it('keeps the input on a network error and lets the user retry', async () => {
    const harness = setupBridge()
    const fetchMock = vi
      .fn()
      .mockRejectedValueOnce(new TypeError('Failed to fetch'))
      .mockResolvedValueOnce(fakeResponse(201, { data: ITEM }))
    vi.stubGlobal('fetch', fetchMock)

    const wrapper = await mountApp(harness)
    await submit(wrapper, '断网时写下的想法')

    expect(wrapper.get('textarea').element.value).toBe('断网时写下的想法')
    expect(wrapper.text()).toContain('连不上 LifeOS')

    await wrapper.get('form.composer').trigger('submit')
    await flushPromises()

    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(bodyOf(fetchMock, 0).capture_key).toBe(bodyOf(fetchMock, 1).capture_key)
    expect(wrapper.get('textarea').element.value).toBe('')
  })

  it('mints a new capture_key when the user edits the draft after a failure', async () => {
    const harness = setupBridge()
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(fakeResponse(503, undefined))
      .mockResolvedValueOnce(fakeResponse(201, { data: ITEM }))
    vi.stubGlobal('fetch', fetchMock)

    const wrapper = await mountApp(harness)
    await submit(wrapper, '第一版想法')
    await submit(wrapper, '第一版想法，补充一句')

    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(bodyOf(fetchMock, 0).capture_key).not.toBe(bodyOf(fetchMock, 1).capture_key)
    expect(bodyOf(fetchMock, 1).content).toBe('第一版想法，补充一句')
  })

  it('does not replay a successful capture when the plugin opens again', async () => {
    const harness = setupBridge()
    const fetchMock = vi.fn().mockResolvedValue(fakeResponse(201, { data: ITEM }))
    vi.stubGlobal('fetch', fetchMock)

    const first = await mountApp(harness)
    await submit(first, '已经存好的想法')
    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(harness.store.draft).toBe('')

    first.unmount()
    mounted.length = 0

    const reopenFetch = vi.fn()
    vi.stubGlobal('fetch', reopenFetch)
    const second = await mountApp(harness)

    expect(reopenFetch).not.toHaveBeenCalled()
    expect(second.get('textarea').element.value).toBe('')
    expect(second.text()).not.toContain('已进入收集箱')
    expect(second.text()).not.toContain('已恢复上次未保存内容')
  })

  it('restores an unsaved draft on the next entry and marks it', async () => {
    const harness = setupBridge('上次没来得及存的想法')
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)

    const wrapper = await mountApp(harness)

    expect(wrapper.get('textarea').element.value).toBe('上次没来得及存的想法')
    expect(wrapper.text()).toContain('已恢复上次未保存内容')
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('writes the draft through the bridge when the plugin is about to close', async () => {
    const harness = setupBridge()
    vi.stubGlobal('fetch', vi.fn())

    const wrapper = await mountApp(harness)
    await wrapper.get('textarea').setValue('写了一半就关掉')

    const outHandler = harness.bridge.onOut.mock.calls[0][0] as () => void
    outHandler()

    expect(harness.bridge.saveDraft).toHaveBeenCalledWith('写了一半就关掉')
    expect(harness.store.draft).toBe('写了一半就关掉')
  })

  it('prefills a short clipboard text and selects it', async () => {
    const harness = setupBridge()
    harness.bridge.readClipboard.mockReturnValue('https://example.com/article')
    vi.stubGlobal('fetch', vi.fn())

    const wrapper = await mountApp(harness)
    const textarea = wrapper.get('textarea').element as HTMLTextAreaElement

    expect(textarea.value).toBe('https://example.com/article')
    expect(wrapper.text()).toContain('来自剪贴板')
    expect(textarea.selectionStart).toBe(0)
    expect(textarea.selectionEnd).toBe(textarea.value.length)
  })

  it('submits on Enter, keeps Shift+Enter for newlines and blocks empty input', async () => {
    const harness = setupBridge()
    const fetchMock = vi.fn().mockResolvedValue(fakeResponse(201, { data: ITEM }))
    vi.stubGlobal('fetch', fetchMock)

    const wrapper = await mountApp(harness)
    expect(wrapper.get('button.composer__submit').attributes('disabled')).toBeDefined()

    const textarea = wrapper.get('textarea')
    await textarea.setValue('回车提交')
    await textarea.trigger('keydown', { key: 'Enter', shiftKey: true })
    expect(fetchMock).not.toHaveBeenCalled()

    await textarea.trigger('keydown', { key: 'Enter' })
    await flushPromises()
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it('asks for the capture credential through the config form on first run', async () => {
    const harness = setupBridge()
    harness.store.token = ''
    vi.stubGlobal('fetch', vi.fn())

    const wrapper = await mountApp(harness)

    expect(wrapper.find('form.composer').exists()).toBe(false)
    expect(wrapper.find('form.config').exists()).toBe(true)
    expect(wrapper.text()).toContain('快速采集凭证')

    const fields = wrapper.findAll('input')
    await fields[0].setValue('http://192.168.1.9:8080')
    await fields[1].setValue('los_pat_new')
    await wrapper.get('form.config').trigger('submit')
    await flushPromises()

    expect(harness.bridge.setConfig).toHaveBeenCalledWith({
      baseUrl: 'http://192.168.1.9:8080',
      token: 'los_pat_new',
    })
    expect(wrapper.find('form.composer').exists()).toBe(true)
  })

  it('rejects a malformed API address in the config form', async () => {
    const harness = setupBridge()
    harness.store.token = ''
    vi.stubGlobal('fetch', vi.fn())

    const wrapper = await mountApp(harness)
    await wrapper.findAll('input')[0].setValue('127.0.0.1:8080')
    await wrapper.findAll('input')[1].setValue('los_pat_new')
    await wrapper.get('form.config').trigger('submit')
    await flushPromises()

    expect(wrapper.text()).toContain('http:// 或 https://')
    expect(harness.bridge.setConfig).not.toHaveBeenCalled()
    expect(wrapper.find('form.config').exists()).toBe(true)
  })
})

describe('uTools 匹配指令的 payload 预填', () => {
  it('把输入文本直接填进输入框，按一次 Enter 即保存', async () => {
    const harness = setupBridge()
    const fetchMock = vi.fn().mockResolvedValue(fakeResponse(201, { data: ITEM }))
    vi.stubGlobal('fetch', fetchMock)

    // mount 时 App 会通过 bridge.onEnter 注册回调，取回它模拟 uTools 的进入事件。
    const wrapper = await mountApp(harness)
    const onEnter = harness.bridge.onEnter.mock.calls[0][0] as (action: { code: string; payload: string }) => void
    onEnter({ code: 'lifeos-capture-text', payload: '123234' })
    await flushPromises()

    expect(wrapper.get('textarea').element.value).toBe('123234')
    expect(wrapper.text()).toContain('来自 uTools 输入')

    await wrapper.get('form.composer').trigger('submit')
    await flushPromises()
    expect(bodyOf(fetchMock, 0).content).toBe('123234')
  })

  it('payload 优先于草稿和剪贴板，且超长输入不会被带入', async () => {
    const harness = setupBridge('上次没保存的内容')
    harness.bridge.readClipboard.mockReturnValue('剪贴板里的旧文本')
    const wrapper = await mountApp(harness)
    const onEnter = harness.bridge.onEnter.mock.calls[0][0] as (action: { code: string; payload: string }) => void

    onEnter({ code: 'lifeos-capture-text', payload: '  新的待办  ' })
    await flushPromises()
    expect(wrapper.get('textarea').element.value).toBe('新的待办')

    onEnter({ code: 'lifeos-capture-text', payload: 'x'.repeat(10001) })
    await flushPromises()
    expect(wrapper.get('textarea').element.value).toBe('新的待办')
  })
})
