export interface APIErrorPayload {
  code: string
  message: string
  retryable: boolean
}

export class APIRequestError extends Error {
  code: string
  retryable: boolean
  status: number

  constructor(payload: APIErrorPayload, status = 0) {
    super(payload.message)
    this.name = 'APIRequestError'
    this.code = payload.code
    this.retryable = payload.retryable
    this.status = status
  }
}

export function isRequestAborted(reason: unknown): boolean {
  return reason instanceof Error && reason.name === 'AbortError'
}

export async function request<T>(path: string, init?: RequestInit): Promise<T> {
  let response: Response
  try {
    response = await fetch(path, {
      ...init,
      headers: { 'Content-Type': 'application/json', ...init?.headers },
    })
  } catch (reason) {
    // Cancellation is an expected lifecycle event (route switch / unmount), not a
    // network failure that should be surfaced to the learner.
    if (isRequestAborted(reason)) throw reason
    throw new APIRequestError({ code: 'NETWORK_ERROR', message: '无法连接 LearnOS 服务，请检查应用是否正在运行。', retryable: true })
  }

  if (!response.ok) {
    if (response.status === 401 && window.location.pathname !== '/login') {
      window.location.assign('/login')
    }
    let payload: unknown = null
    try { payload = await response.json() } catch { /* keep status fallback */ }
    const rawError = (payload as { error?: unknown } | null)?.error
    if (rawError && typeof rawError === 'object' && 'code' in rawError) {
      const error = rawError as Partial<APIErrorPayload>
      throw new APIRequestError({ code: error.code ?? 'REQUEST_ERROR', message: error.message ?? `请求失败：HTTP ${response.status}`, retryable: error.retryable ?? false }, response.status)
    }
    const legacyCode = typeof rawError === 'string' ? rawError : `HTTP_${response.status}`
    throw new APIRequestError({ code: legacyCode, message: readableLegacyError(legacyCode, response.status), retryable: response.status >= 500 }, response.status)
  }

  return response.json() as Promise<T>
}

function readableLegacyError(code: string, status: number): string {
  const messages: Record<string, string> = {
    AI_TIMEOUT: 'AI 响应超时，本次数据未保存。',
    AI_INVALID_RESPONSE: 'AI 返回内容无法验证，本次数据未保存。',
    AI_PROVIDER_ERROR: 'AI 服务暂时不可用，本次数据未保存。',
    AI_NOT_CONFIGURED: 'AI 尚未配置，本次数据未保存。',
    AI_RATE_LIMITED: 'AI 请求过于频繁，请稍后重试。本次数据未保存。',
    AI_NETWORK_ERROR: '暂时无法连接 AI 服务，本次数据未保存。',
    'invalid domain request': '请检查领域名称、学习原因和期望深度。',
    'invalid domain initialization request': '请检查领域名称、学习原因和期望深度。',
  }
  return messages[code] ?? `请求失败：HTTP ${status}`
}
