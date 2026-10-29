// The only bridge between the renderer and uTools. `preload.js` assigns the same
// shape to window.lifeos inside uTools; the dev shim keeps the UI usable when the
// built page is opened in a plain browser (no uTools runtime).

export const STORAGE_KEYS = {
  baseUrl: 'lifeos.baseUrl',
  token: 'lifeos.token',
  draft: 'lifeos.draft',
} as const

export const DEFAULT_BASE_URL = 'http://127.0.0.1:8080'

export interface LifeosConfig {
  baseUrl: string
  token: string
}

export interface PluginEnterAction {
  code?: string
  type?: string
  payload?: unknown
  from?: string
}

export interface NodeRequestOptions {
  url: string
  method?: string
  headers?: Record<string, string>
  body?: string
  timeoutMs?: number
}

export interface NodeResponse {
  status: number
  body: string
}

export interface LifeosBridge {
  readonly isDev: boolean
  /**
   * 'utools'      preload.js exposed window.lifeos (Node transport available);
   * 'utools-page' preload unavailable but the page itself has the utools API;
   * 'browser'     plain browser dev fallback.
   */
  readonly runtime?: 'utools' | 'utools-page' | 'browser'
  getConfig(): LifeosConfig
  setConfig(config: LifeosConfig): void
  readClipboard(): string
  getDraft(): string
  saveDraft(text: string): void
  clearDraft(): void
  /** Diagnostic: the last uTools enter action (JSON string), empty when unknown. */
  getLastEnter?(): string
  /** Reads and clears an enter action buffered before the page registered. */
  consumeEnter?(): { code?: string; payload?: unknown } | null
  onEnter(callback: (action: PluginEnterAction) => void): void
  onOut(callback: () => void): void
  hideWindow(): void
  /**
   * Present inside uTools only: performs the HTTP call from the Node/preload
   * context, because the plugin page is cross-origin to LifeOS and the backend
   * intentionally ships no CORS headers.
   */
  request?(options: NodeRequestOptions): Promise<NodeResponse>
}

interface BridgeHost {
  lifeos?: LifeosBridge
}

declare global {
  interface Window extends BridgeHost {}
}

function devStorage(): Storage | null {
  try {
    return globalThis.localStorage ?? null
  } catch {
    return null
  }
}

function devShim(): LifeosBridge {
  const read = (key: string): string => {
    try {
      return devStorage()?.getItem(key) ?? ''
    } catch {
      return ''
    }
  }
  const write = (key: string, value: string): void => {
    try {
      devStorage()?.setItem(key, value)
    } catch {
      /* private mode: drafts simply are not persisted */
    }
  }
  return {
    isDev: true,
    getConfig: () => ({
      baseUrl: read(STORAGE_KEYS.baseUrl) || DEFAULT_BASE_URL,
      token: read(STORAGE_KEYS.token),
    }),
    setConfig: (config) => {
      write(STORAGE_KEYS.baseUrl, config.baseUrl)
      write(STORAGE_KEYS.token, config.token)
    },
    readClipboard: () => '',
    getDraft: () => read(STORAGE_KEYS.draft),
    saveDraft: (text) => write(STORAGE_KEYS.draft, text),
    clearDraft: () => {
      try {
        devStorage()?.removeItem(STORAGE_KEYS.draft)
      } catch {
        /* ignore */
      }
    },
    runtime: 'browser' as const,
    getLastEnter: () => '',
    consumeEnter: () => null,
    onEnter: () => {},
    onOut: () => {},
    hideWindow: () => {},
  }
}

/**
 * 兜底：某些 uTools 版本直接向页面注入 `utools`，但 preload 的 window 隔离不生效。
 * 此时用页面里的 utools API 拼一个等价桥接，至少让配置、进入事件和窗口行为可用。
 */
function pageUtoolsShim(): LifeosBridge | null {
  const api = (globalThis as unknown as { utools?: Record<string, any> }).utools
  if (!api || typeof api.dbStorage?.getItem !== 'function') return null
  const read = (key: string): string => {
    try {
      const value = api.dbStorage.getItem(key)
      return typeof value === 'string' ? value : ''
    } catch {
      return ''
    }
  }
  const write = (key: string, value: string): void => {
    try {
      api.dbStorage.setItem(key, value)
    } catch {
      /* ignore */
    }
  }
  return {
    runtime: 'utools-page',
    isDev: false,
    getConfig: () => ({ baseUrl: read(STORAGE_KEYS.baseUrl) || DEFAULT_BASE_URL, token: read(STORAGE_KEYS.token) }),
    setConfig: (config) => {
      write(STORAGE_KEYS.baseUrl, config.baseUrl)
      write(STORAGE_KEYS.token, config.token)
    },
    readClipboard: () => {
      try {
        return typeof api.getCopyedText === 'function' ? String(api.getCopyedText() ?? '') : ''
      } catch {
        return ''
      }
    },
    getDraft: () => read(STORAGE_KEYS.draft),
    saveDraft: (text) => write(STORAGE_KEYS.draft, text),
    clearDraft: () => {
      try {
        api.dbStorage.removeItem(STORAGE_KEYS.draft)
      } catch {
        /* ignore */
      }
    },
    getLastEnter: () => read('lifeos.lastEnter'),
    onEnter: (callback) => {
      try {
        if (typeof api.onPluginEnter === 'function') api.onPluginEnter(callback)
      } catch {
        /* ignore */
      }
    },
    onOut: (callback) => {
      try {
        if (typeof api.onPluginOut === 'function') api.onPluginOut(callback)
      } catch {
        /* ignore */
      }
    },
    hideWindow: () => {
      try {
        api.hideMainWindow?.()
      } catch {
        /* ignore */
      }
    },
  }
}

export function getBridge(): LifeosBridge {
  const host = globalThis as unknown as BridgeHost
  return host.lifeos ?? pageUtoolsShim() ?? devShim()
}
