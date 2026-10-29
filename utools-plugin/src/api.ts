// Contract for the frozen capture endpoint: POST {baseUrl}/api/v1/capture/inbox
// with a capture-scope bearer token. Everything here is pure so it can be tested
// without a server.
import { getBridge } from './bridge'

export interface InboxItem {
  id: number
  content: string
  source_type: string
  source_url: string
  status: string
  created_at: string
}

export type CaptureFailureKind =
  | 'unauthorized'
  | 'invalid'
  | 'conflict'
  | 'server'
  | 'network'
  | 'misconfigured'

export interface CaptureFailureOptions {
  status?: number
  retryable?: boolean
  detail?: string
}

/** Typed failure so the UI can pick copy and the next action without parsing strings. */
export class CaptureFailure extends Error {
  readonly kind: CaptureFailureKind
  readonly status: number
  readonly retryable: boolean
  readonly detail: string

  constructor(kind: CaptureFailureKind, message: string, options: CaptureFailureOptions = {}) {
    super(message)
    this.name = 'CaptureFailure'
    this.kind = kind
    this.status = options.status ?? 0
    this.retryable = options.retryable ?? false
    this.detail = options.detail ?? ''
  }
}

export interface CaptureRequest {
  baseUrl: string
  token: string
  content: string
  captureKey: string
  fetchImpl?: typeof fetch
  signal?: AbortSignal
  timeoutMs?: number
}

export const CAPTURE_PATH = '/api/v1/capture/inbox'

/**
 * Inside uTools the call must leave from preload (Node): the plugin page is a
 * local file, so a page fetch() would be cross-origin and blocked because LifeOS
 * serves no CORS headers. In a plain browser (dev preview, unit tests) the bridge
 * has no request() and the regular fetch is used.
 */
function resolveTransport(): typeof fetch | undefined {
  const bridge = getBridge()
  if (bridge.request) {
    return (async (input: RequestInfo | URL, init?: RequestInit) => {
      const nodeResponse = await bridge.request!({
        url: String(input),
        method: init?.method ?? 'GET',
        headers: (init?.headers ?? {}) as Record<string, string>,
        body: typeof init?.body === 'string' ? init.body : '',
      })
      return new Response(nodeResponse.body, { status: nodeResponse.status })
    }) as typeof fetch
  }
  return globalThis.fetch
}
export const DEFAULT_TIMEOUT_MS = 15000

/** Trims whitespace and trailing slashes so '/capture/inbox' never doubles up. */
export function normalizeBaseUrl(value: string): string {
  return value.trim().replace(/\/+$/, '')
}

export function isHttpUrl(value: string): boolean {
  try {
    const url = new URL(value)
    return url.protocol === 'http:' || url.protocol === 'https:'
  } catch {
    return false
  }
}

interface CaptureResponseBody {
  data?: InboxItem
  error?: { code?: string; message?: string; retryable?: boolean }
}

function readServerMessage(payload: unknown): string {
  if (payload && typeof payload === 'object' && 'error' in payload) {
    const error = (payload as CaptureResponseBody).error
    if (error && typeof error.message === 'string') return error.message
  }
  return ''
}

async function readJson(response: Response): Promise<CaptureResponseBody | null> {
  try {
    const text = await response.text()
    if (!text) return null
    return JSON.parse(text) as CaptureResponseBody
  } catch {
    return null
  }
}

function failureForStatus(status: number, detail: string): CaptureFailure {
  if (status === 401 || status === 403) {
    return new CaptureFailure('unauthorized', '采集凭证无效或已过期。', { status, retryable: false, detail })
  }
  if (status === 400 || status === 422) {
    return new CaptureFailure('invalid', '这条内容不符合收集箱的要求。', { status, retryable: false, detail })
  }
  if (status === 409) {
    return new CaptureFailure('conflict', '同一个提交编号已经保存过其他内容。', { status, retryable: false, detail })
  }
  if (status === 429) {
    return new CaptureFailure('server', 'LifeOS 正在限流。', { status, retryable: true, detail })
  }
  if (status >= 500) {
    return new CaptureFailure('server', 'LifeOS 服务暂时不可用。', { status, retryable: true, detail })
  }
  return new CaptureFailure('server', 'LifeOS 返回了意外的响应。', { status, retryable: false, detail })
}

function withTimeout(request: Promise<Response>, timeoutMs: number, signal?: AbortSignal): Promise<Response> {
  if (!timeoutMs || timeoutMs <= 0 || signal) return request
  const controller = new AbortController()
  const timer = setTimeout(() => controller.abort(), timeoutMs)
  return Promise.race([
    request,
    new Promise<Response>((_, reject) => {
      controller.signal.addEventListener('abort', () => reject(new Error('capture request timed out')))
    }),
  ]).finally(() => clearTimeout(timer))
}

/** Sends one inbox item. Throws CaptureFailure for every failure path. */
export async function captureInbox(input: CaptureRequest): Promise<InboxItem> {
  const baseUrl = normalizeBaseUrl(input.baseUrl)
  if (!baseUrl || !isHttpUrl(baseUrl)) {
    throw new CaptureFailure('misconfigured', '还没有填写可用的 API 地址。')
  }
  const token = input.token.trim()
  if (!token) {
    throw new CaptureFailure('misconfigured', '还没有配置采集凭证。')
  }

  const doFetch = input.fetchImpl ?? resolveTransport()
  if (typeof doFetch !== 'function') {
    throw new CaptureFailure('misconfigured', '当前环境不支持网络请求。')
  }

  let response: Response
  try {
    response = await withTimeout(
      doFetch(`${baseUrl}${CAPTURE_PATH}`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ content: input.content, capture_key: input.captureKey }),
        signal: input.signal,
      }),
      input.timeoutMs ?? DEFAULT_TIMEOUT_MS,
      input.signal,
    )
  } catch (error) {
    throw new CaptureFailure('network', '无法连接 LifeOS。', {
      retryable: true,
      detail: error instanceof Error ? error.message : String(error),
    })
  }

  const payload = await readJson(response)
  if (response.ok) {
    if (!payload || !payload.data) {
      throw new CaptureFailure('server', 'LifeOS 返回了无法识别的内容。', {
        status: response.status,
        retryable: false,
      })
    }
    return payload.data
  }
  throw failureForStatus(response.status, readServerMessage(payload))
}

export interface FailureGuidance {
  message: string
  action: 'config' | 'retry' | null
}

/** Chinese UI copy plus the single recovery action for each failure kind. */
export function describeCaptureFailure(failure: CaptureFailure): FailureGuidance {
  switch (failure.kind) {
    case 'unauthorized':
      return { message: '凭证无效，到设置页「快速采集凭证」重新生成', action: 'config' }
    case 'misconfigured':
      return { message: '还没连上 LifeOS，先填写地址和凭证', action: 'config' }
    case 'network':
      return { message: '连不上 LifeOS，确认服务已启动后重试', action: 'retry' }
    case 'server':
      return { message: 'LifeOS 暂时没响应，稍后重试一次', action: 'retry' }
    case 'conflict':
      return { message: '内容已按旧编号存过，重试会重新保存', action: 'retry' }
    case 'invalid':
      return { message: '内容为空或超过 10000 字', action: null }
    default:
      return { message: '提交失败，稍后重试一次', action: 'retry' }
  }
}
