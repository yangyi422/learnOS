// Real transport + lifecycle test for preload.js itself: it is the file uTools runs
// before the page, so a throw there silently degrades the whole plugin.
import { createRequire } from 'node:module'
import http from 'node:http'
import type { AddressInfo } from 'node:net'
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'

interface NodeResponse { status: number; body: string }
interface LifeosBridge {
  runtime: string
  isDev: boolean
  request(options: { url: string; method?: string; headers?: Record<string, string>; body?: string; timeoutMs?: number }): Promise<NodeResponse>
  getLastEnter(): string
  consumeEnter(): { code?: string; payload?: string } | null
  onEnter(callback: (action: { code?: string; payload?: string }) => void): void
  getConfig(): { baseUrl: string; token: string }
  setConfig(config: { baseUrl: string; token: string }): void
  saveDraft(text: string): void
  getDraft(): string
}

let server: http.Server
let base = ''
let seen: { method?: string; authorization?: string; body: string } | null = null
let store: Record<string, string> = {}
let enterCallback: ((action: unknown) => void) | null = null

beforeAll(async () => {
  // Simulate the uTools host and load preload.js exactly the way uTools does
  // (plain CommonJS, before the page). It runs once: uTools also loads it once per
  // plugin session, which is why a changed preload needs a plugin/uTools restart.
  const utools = {
    dbStorage: {
      getItem: (key: string) => store[key] ?? null,
      setItem: (key: string, value: string) => { store[key] = value },
      removeItem: (key: string) => { delete store[key] },
    },
    onPluginEnter: (cb: (action: unknown) => void) => { enterCallback = cb },
    onPluginOut: () => {},
    hideMainWindow: () => {},
    setExpendHeight: () => {},
  }
  vi.stubGlobal('utools', utools)
  createRequire(import.meta.url)('../../preload.js')

  server = http.createServer((req, res) => {
    const chunks: Buffer[] = []
    req.on('data', c => chunks.push(c as Buffer))
    req.on('end', () => {
      seen = { method: req.method, authorization: req.headers.authorization as string, body: Buffer.concat(chunks).toString('utf8') }
      res.writeHead(201, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify({ data: { id: 1, content: 'ok' } }))
    })
  })
  await new Promise<void>(resolve => server.listen(0, '127.0.0.1', resolve))
  base = `http://127.0.0.1:${(server.address() as AddressInfo).port}`
})

afterAll(async () => {
  await new Promise<void>(resolve => server.close(() => resolve()))
})

beforeEach(() => {
  store = {}
  seen = null
})

describe('preload bridge', () => {
  it('defines window.lifeos and reports the uTools runtime', () => {
    const bridge = (window as unknown as { lifeos: LifeosBridge }).lifeos
    expect(bridge).toBeTruthy()
    expect(bridge.runtime).toBe('utools')
    expect(bridge.isDev).toBe(false)
    expect(typeof bridge.request).toBe('function')
  })

  // 在 jsdom 里 `require` 由测试运行器接管，preload 会走 fetch 兜底分支；在真实
  // uTools preload 中走 Node http 分支。两条分支共用一个契约：状态码 + 原始 body。
  it('performs the capture request with bearer auth against a real server', async () => {
    const bridge = (window as unknown as { lifeos: LifeosBridge }).lifeos
    const response = await bridge.request({
      url: `${base}/api/v1/capture/inbox`,
      method: 'POST',
      headers: { Authorization: 'Bearer los_pat_x', 'Content-Type': 'application/json' },
      body: JSON.stringify({ content: '灵感', capture_key: 'cap-1' }),
    })
    expect(response.status).toBe(201)
    expect(seen?.method).toBe('POST')
    expect(seen?.authorization).toBe('Bearer los_pat_x')
    expect(JSON.parse(seen!.body)).toEqual({ content: '灵感', capture_key: 'cap-1' })
  })

  it('buffers an enter action that arrives before the page registers, then replays it', async () => {
    const bridge = (window as unknown as { lifeos: LifeosBridge }).lifeos
    expect(enterCallback).toBeTruthy()
    // uTools fires this while the page is still loading.
    enterCallback!({ code: 'lifeos-capture-text', type: 'over', payload: '测试写入' })

    expect(bridge.getLastEnter()).toContain('测试写入')
    expect(JSON.parse(bridge.getLastEnter()).payloadLength).toBe(4)

    const received: string[] = []
    bridge.onEnter(action => received.push(String(action.payload)))
    expect(received).toEqual(['测试写入'])
    expect(bridge.consumeEnter()).toBeNull()
  })

  it('hands the action straight to the handler when it is already registered', () => {
    const bridge = (window as unknown as { lifeos: LifeosBridge }).lifeos
    const received: string[] = []
    bridge.onEnter(action => received.push(String(action.payload)))
    enterCallback!({ code: 'lifeos-capture-text', type: 'over', payload: '迟到的文本' })
    expect(received).toEqual(['迟到的文本'])
  })

  it('persists config and drafts through dbStorage', () => {
    const bridge = (window as unknown as { lifeos: LifeosBridge }).lifeos
    bridge.setConfig({ baseUrl: 'http://127.0.0.1:8080', token: 'los_pat_y' })
    expect(bridge.getConfig()).toEqual({ baseUrl: 'http://127.0.0.1:8080', token: 'los_pat_y' })
    bridge.saveDraft('草稿')
    expect(bridge.getDraft()).toBe('草稿')
  })
})
