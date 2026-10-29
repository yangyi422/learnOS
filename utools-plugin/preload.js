// uTools preload. Runs before the page with Node.js + Electron access and must
// never throw: one failure here leaves the page without window.lifeos, which
// silently degrades the plugin to its browser fallback (config in localStorage,
// no uTools payload). Keep it self-contained — no local require — and guard every
// privileged access.
'use strict'

const STORAGE_KEYS = {
  baseUrl: 'lifeos.baseUrl',
  token: 'lifeos.token',
  draft: 'lifeos.draft',
  lastEnter: 'lifeos.lastEnter',
}

const DEFAULT_BASE_URL = 'http://127.0.0.1:8080'
const PLUGIN_HEIGHT = 220
const REQUEST_TIMEOUT_MS = 15000

// `utools` / `require` availability differs between host builds; probe instead of
// assuming, so window.lifeos is always defined.
const utoolsApi = typeof utools !== 'undefined' && utools ? utools : null

function nodeModule(name) {
  try {
    if (typeof require === 'function') return require(name)
  } catch (error) {
    // not reachable in this host build
  }
  return null
}

const http = nodeModule('http')
const https = nodeModule('https')
const electron = nodeModule('electron')
const clipboardApi = electron && electron.clipboard ? electron.clipboard : null

let enterHandler = null
let outHandler = null
// uTools can deliver onPluginEnter before the page registers its handler.
let pendingEnter = null
let lastEnter = ''

function readItem(key) {
  if (!utoolsApi || !utoolsApi.dbStorage) return ''
  try {
    const value = utoolsApi.dbStorage.getItem(key)
    return typeof value === 'string' ? value : ''
  } catch (error) {
    return ''
  }
}

function writeItem(key, value) {
  if (!utoolsApi || !utoolsApi.dbStorage) return
  try {
    utoolsApi.dbStorage.setItem(key, value)
  } catch (error) {
    // storage failure must never break capturing
  }
}

function removeItem(key) {
  if (!utoolsApi || !utoolsApi.dbStorage) return
  try {
    utoolsApi.dbStorage.removeItem(key)
  } catch (error) {
    // ignore
  }
}

/** One JSON request from the Node context: the page is cross-origin to LifeOS. */
function requestJson(options) {
  const opts = options || {}
  return new Promise(function (resolve, reject) {
    let target
    try {
      target = new URL(opts.url)
    } catch (error) {
      reject(new Error('invalid url: ' + opts.url))
      return
    }
    const transport = target.protocol === 'https:' ? https : http
    if (!transport) {
      // Host without Node http (or a sandboxed preload): fall back to fetch. Inside
      // uTools this branch is normally not taken, because `http` is available.
      if (typeof fetch !== 'function') {
        reject(new Error('no HTTP transport available'))
        return
      }
      fetch(target.toString(), {
        method: opts.method || 'GET',
        headers: opts.headers || {},
        body: opts.body || undefined,
      })
        .then(function (response) {
          return response.text().then(function (text) {
            resolve({ status: response.status, body: text })
          })
        })
        .catch(reject)
      return
    }
    const req = transport.request(
      {
        protocol: target.protocol,
        hostname: target.hostname,
        port: target.port || (target.protocol === 'https:' ? 443 : 80),
        path: target.pathname + target.search,
        method: opts.method || 'GET',
        headers: opts.headers || {},
      },
      function (response) {
        const chunks = []
        response.on('data', function (chunk) {
          chunks.push(chunk)
        })
        response.on('end', function () {
          resolve({ status: response.statusCode || 0, body: Buffer.concat(chunks).toString('utf8') })
        })
        response.on('error', reject)
      },
    )
    req.setTimeout(opts.timeoutMs || REQUEST_TIMEOUT_MS, function () {
      req.destroy(new Error('request timed out'))
    })
    req.on('error', reject)
    if (opts.body) req.write(opts.body)
    req.end()
  })
}

function recordEnter(action) {
  try {
    const payload = action && typeof action.payload === 'string' ? action.payload : ''
    lastEnter = JSON.stringify({
      at: new Date().toISOString(),
      code: action && action.code ? String(action.code) : '',
      type: action && action.type ? String(action.type) : '',
      from: action && action.from ? String(action.from) : '',
      payloadLength: payload.length,
      payloadPreview: payload.slice(0, 80),
    })
    writeItem(STORAGE_KEYS.lastEnter, lastEnter)
  } catch (error) {
    // diagnosis must never break entering
  }
}

const bridge = {
  // 'utools' proves this preload ran; the UI surfaces it in the config panel.
  runtime: utoolsApi ? 'utools' : 'browser',
  isDev: !utoolsApi,
  getConfig: function () {
    return {
      baseUrl: readItem(STORAGE_KEYS.baseUrl) || DEFAULT_BASE_URL,
      token: readItem(STORAGE_KEYS.token),
    }
  },
  setConfig: function (config) {
    writeItem(STORAGE_KEYS.baseUrl, String((config && config.baseUrl) || ''))
    writeItem(STORAGE_KEYS.token, String((config && config.token) || ''))
  },
  readClipboard: function () {
    try {
      return clipboardApi ? clipboardApi.readText() || '' : ''
    } catch (error) {
      return ''
    }
  },
  request: requestJson,
  getDraft: function () {
    return readItem(STORAGE_KEYS.draft)
  },
  saveDraft: function (text) {
    if (typeof text === 'string' && text.trim()) {
      writeItem(STORAGE_KEYS.draft, text)
      return
    }
    removeItem(STORAGE_KEYS.draft)
  },
  clearDraft: function () {
    removeItem(STORAGE_KEYS.draft)
  },
  getLastEnter: function () {
    return lastEnter || readItem(STORAGE_KEYS.lastEnter)
  },
  // Reads and clears a buffered action, so the page can recover the payload even
  // when its onEnter registration lost the race with uTools.
  consumeEnter: function () {
    const action = pendingEnter
    pendingEnter = null
    return action
  },
  onEnter: function (callback) {
    enterHandler = typeof callback === 'function' ? callback : null
    if (enterHandler && pendingEnter) {
      const action = pendingEnter
      pendingEnter = null
      try {
        enterHandler(action)
      } catch (error) {
        // ignore
      }
    }
  },
  onOut: function (callback) {
    outHandler = typeof callback === 'function' ? callback : null
  },
  hideWindow: function () {
    if (!utoolsApi) return
    try {
      utoolsApi.setExpendHeight(PLUGIN_HEIGHT)
    } catch (error) {
      // ignore
    }
    try {
      utoolsApi.hideMainWindow()
    } catch (error) {
      // ignore
    }
  },
}

// Expose the bridge to the *page* world. Newer uTools builds run preload with
// contextIsolation, where a plain `window.lifeos = ...` only lands in the isolated
// world and the page never sees it; contextBridge is the supported channel there.
let exposed = false
try {
  const electronApi = nodeModule('electron')
  if (electronApi && electronApi.contextBridge && typeof electronApi.contextBridge.exposeInMainWorld === 'function') {
    electronApi.contextBridge.exposeInMainWorld('lifeos', bridge)
    exposed = true
  }
} catch (error) {
  exposed = false
}
if (!exposed) {
  try {
    window.lifeos = bridge
    exposed = true
  } catch (error) {
    // last resort: nothing else to try
  }
}

try {
  if (utoolsApi && typeof utoolsApi.onPluginEnter === 'function') {
    utoolsApi.onPluginEnter(function (action) {
      recordEnter(action)
      if (enterHandler) {
        try {
          enterHandler(action)
        } catch (error) {
          // ignore
        }
        return
      }
      pendingEnter = action
    })
  }
} catch (error) {
  // never block the page on event registration
}

try {
  if (utoolsApi && typeof utoolsApi.onPluginOut === 'function') {
    utoolsApi.onPluginOut(function (isKill) {
      if (!outHandler) return
      try {
        outHandler(isKill)
      } catch (error) {
        // ignore
      }
    })
  }
} catch (error) {
  // ignore
}
