/**
 * 视觉实验入口的可见性开关。
 *
 * 「实验」悬浮条与「视觉实验室」是开发期调特效用的工具，不该出现在正式界面里。
 * 规则：
 * - 开发环境（`import.meta.env.DEV`）始终可见；
 * - 正式构建默认隐藏，只有显式开启才出现：URL 带 `?fx=1`，或 localStorage
 *   里设了 `learnos:fx-enabled=1`（桌面端没有地址栏，用这个开关）。
 *
 * 只影响入口是否渲染；权限仍由 `/auth/me` 的 admin 角色决定（见 pilot.ts）。
 */
const ENABLE_KEY = 'learnos:fx-enabled'
const QUERY_KEY = 'fx'

function readFlag(): boolean {
  if (import.meta.env.DEV) return true
  try {
    if (new URLSearchParams(window.location.search).get(QUERY_KEY) === '1') {
      // 记到本地，避免每次手输；退出用 localStorage.removeItem('learnos:fx-enabled')
      window.localStorage.setItem(ENABLE_KEY, '1')
      return true
    }
    return window.localStorage.getItem(ENABLE_KEY) === '1'
  } catch {
    return false
  }
}

const labVisible = readFlag()

/** 实验工具入口是否应渲染（仍需 admin 角色才真正可用）。 */
export function labToolsVisible(): boolean {
  return labVisible
}
