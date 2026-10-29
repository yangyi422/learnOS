# LifeOS 收集（uTools 插件）

极简快速采集插件：把想法、待办和链接送进 LifeOS 收集箱。uTools 里搜「LifeOS 收集」或「快速记录」，回车即存。

## 功能

| 能力 | 行为 |
| --- | --- |
| 进入即聚焦 | 打开插件后光标已在输入框；有剪贴板文本时预填并全选，可直接回车 |
| 键盘 | `Enter` 保存；`Shift + Enter` 换行；中文输入法回车确认候选词不会误提交 |
| 空输入 | 提交按钮禁用，回车无效 |
| 提交中 | 输入框只读、按钮显示「保存中…」并禁用，避免重复提交 |
| 成功 | 显示「已进入收集箱」，清空输入，约 420ms 后 `utools.hideMainWindow()` 关闭主窗口 |
| 失败 | 保留输入，显示可理解的中文原因；401 主按钮变「去配置」，5xx / 网络错误变「重试」 |
| 未保存内容 | 退出到后台或被结束运行前，通过 `utools.onPluginOut` 把草稿写入 `dbStorage`，下次进入恢复并提示「已恢复上次未保存内容」 |
| 首次使用 | 展示紧凑配置区（API 地址 + 采集凭证），保存到 `dbStorage` |
| 幂等 | 每次提交生成一个 `capture_key`；同一内容的网络重试复用同一个 key |

## 前置条件

1. LifeOS 服务已运行（默认 `http://127.0.0.1:8080`）。
2. 主仓库已提供 `POST /api/v1/capture/inbox`（`internal/httpapi/api_token_handler.go`，中间件 `TokenOrSession` + `RequireScope("capture:inbox")`）。
3. 一枚采集凭证，scope 为 `capture:inbox`，形如 `los_pat_…`。凭证由 LifeOS Web 端设置页的「快速采集凭证」面板生成（该面板由主仓库同步开发）。入口未就绪时，可先登录 Web 后调用 `POST /api/v1/api-tokens` 签发。

插件把 token 只存在本机 uTools 数据库（`utools.dbStorage`），不会写进仓库或日志。

## 目录结构

```
utools-plugin/
├─ plugin.json              # main / logo / preload / pluginSetting.height=220 / features[].cmds
├─ preload.js               # 唯一使用 Node + Electron 的文件；挂 window.lifeos 桥接
├─ logo.png                 # 256×256，由 scripts/generate-logo.mjs 生成
├─ index.html               # Vite 入口
├─ vite.config.ts           # base './'，输出 dist/，构建后拷入三个运行时文件
├─ src/
│  ├─ main.ts               # 挂载 Vue
│  ├─ App.vue               # 采集卡片：书写态 / 配置态 / 成功 / 错误
│  ├─ api.ts                # 冻结契约的 fetch 封装 + 错误分类 + 中文文案映射
│  ├─ captureKey.ts         # 8-80 字符 [A-Za-z0-9_-] 的随机 key
│  ├─ bridge.ts             # window.lifeos 访问层 + 浏览器 dev 兜底
│  ├─ styles.css            # 设计 token（冷白/浅蓝灰、16/9/999 圆角、单一强调色）
│  └─ __tests__/            # vitest：api 契约 + 界面状态机
└─ scripts/generate-logo.mjs
```

## 开发

```bash
cd utools-plugin
npm install
npm run dev        # Vite dev server，浏览器打开可调样式（走 dev 兜底）
npm run build      # 产出 dist/
npm test           # vitest run
npm run typecheck  # vue-tsc --noEmit
npm run logo       # 重新生成 logo.png
```

浏览器里直接打开时没有 `window.lifeos`，`bridge.ts` 会退化为 localStorage 版：配置与草稿可用，剪贴板读取返回空，`hideWindow` 为空操作。这只用于调样式，不是真实运行环境。

### preload 必须自包含

`preload.js` 是 uTools 在页面之前加载的唯一特权文件。它一旦抛错，页面就拿不到 `window.lifeos`，会**静默退化**成浏览器兜底（配置写 localStorage、没有 uTools 事件与剪贴板），从界面上很难看出来。

因此现在的写法是：不 import 任何本地文件，`require('http')`/`require('electron')` 全部包在 try/catch 里，取不到就退回 `fetch`；事件注册也包在 try/catch 中，保证 `window.lifeos` 一定存在。插件「连接设置」里会显示**运行模式**：

- `uTools 桥接` → preload 正常；
- `浏览器兜底（preload 未生效）` → preload 没跑起来（旧缓存或加载失败）。

改过 `preload.js` 后，uTools 的「重载」不一定重新加载 preload，需要**完全退出 uTools 再打开**（或移除开发项目后重新接入）。这是排查"输入没带进来"的第一个检查点。

### 为什么请求走 preload 而不是页面 fetch

插件页由 uTools 以本地文件方式加载，对 `http://127.0.0.1:8080` 属于**跨源请求**；LifeOS 后端有意不发送任何 CORS 头（见 `internal/httpapi/router.go` 与 `docs/CAPTURE_API.md`）。因此真实运行时 `src/api.ts` 优先使用 `window.lifeos.request()`——它由 `preload-request.cjs` 在 Node 上下文里发 `http/https` 请求，不受浏览器同源策略限制；只有在浏览器 dev 兜底环境下才退回 `fetch`。

运输层与生命周期由 `src/__tests__/preloadBridge.test.ts` 覆盖：用 `dbStorage`/`onPluginEnter` 桩加载真实 `preload.js`，对着本地真实 http 服务断言 method / Authorization / body / 状态码，并验证「早到的事件被缓冲、注册后补发」。

## 在 uTools 中加载

1. 打开 uTools 的「uTools 开发者工具」。
2. 「新建项目」，填写应用信息。
3. 点「选择工程 plugin.json」，选中 **`utools-plugin/dist/plugin.json`**。`dist/` 已经是完整可加载目录（`plugin.json`、`index.html`、`preload.js`、`logo.png`、`assets/` 齐全）。
4. 点「接入开发」，再点「打开」。
5. 在 uTools 搜索框输入「LifeOS 收集」或「快速记录」进入；首次进入会看到配置区。
6. 改完源码重新 `npm run build`，再在开发者工具里重载/重新进入。

`plugin.json` 中 `pluginSetting.height = 220` 决定初始高度；宽度由 uTools 主窗口决定，卡片内容 `max-width: 620px` 居中。

## 直接在 uTools 搜索结果里出现（匹配指令）

`plugin.json` 有两个功能：

| code | 触发方式 | 行为 |
| --- | --- | --- |
| `lifeos-capture` | 搜索「LifeOS 收集」「快速记录」 | 打开空白采集卡片（剪贴板有内容时预填） |
| `lifeos-capture-text` | **匹配指令**：选中任意文本或在搜索框输入 ≥2 个字符（`over` 类型） | 搜索结果中出现「保存到 LifeOS 收集箱」，回车后把该文本作为 `payload` 带进插件并自动填入输入框 |

也就是：在 uTools 里输入 `123234`，候选里会多一条「保存到 LifeOS 收集箱」；选中它按回车，插件打开时输入框已填好 `123234` 且全选，再按一次 Enter 就进收集箱（不会自动提交，避免误存）。

预填优先级：**uTools payload > 未保存草稿 > 剪贴板**。payload 超过 10000 字符会被忽略，不会带入。`onPluginEnter` 可能早于页面加载触发，`preload.js` 会缓存最后一次 action 并在渲染层注册回调时补发。

注意：只用 `over`，**不要**用 `regex` 写“任意匹配”。uTools 官方明确会忽略任意匹配的正则（`/.*/`、`/[\s\S]*/` 之类），这类规则不会生效；`over` 才是为任意文本设计的那条。

排查入口：插件「连接设置」面板最下面有一行「最近一次进入」，显示 uTools 实际传来的 `code · type · 字符数 · 文本预览`。如果这里显示 `0 字符`，说明文本没有随进入事件传进来；如果一行都为空，说明 `onPluginEnter` 没有触发。

## 存储键

| 键 | 内容 | 默认值 |
| --- | --- | --- |
| `lifeos.baseUrl` | LifeOS API 基地址 | `http://127.0.0.1:8080` |
| `lifeos.token` | 采集凭证 | 空（空则展示配置区） |
| `lifeos.draft` | 未保存的草稿文本 | 空；保存成功后删除 |

## 接口契约

```
POST {baseUrl}/api/v1/capture/inbox
Authorization: Bearer <lifeos.token>
Content-Type: application/json

{"content": "<纯文本>", "capture_key": "<8-80 字符 [A-Za-z0-9_-]>"}
```

| 状态 | 插件处理 |
| --- | --- |
| `201 {data:{…}}` | 成功：提示、清空、删除草稿、关闭窗口 |
| `401` | 「凭证无效，到设置页「快速采集凭证」重新生成」，主按钮变「去配置」，输入保留，不自动重试 |
| `400` | 「内容为空或超过 10000 字」；客户端也会在超长时禁用提交 |
| `409` | 丢弃旧 `capture_key`，提示可重试；下次提交换新 key |
| `5xx` | 「LifeOS 暂时没响应，稍后重试一次」，主按钮「重试」，复用同一 key |
| 网络失败 | 「连不上 LifeOS，确认服务已启动后重试」，复用同一 key |

后端错误里的 `error.message` 只作为 `CaptureFailure.detail` 保留，界面展示的是插件自己的中文文案，不会把服务端实现细节抖到界面上。

### 幂等策略

- 首次提交为当前内容生成 `cap-<32 位十六进制>`（满足 8-80 / `[A-Za-z0-9_-]`）。
- 同一内容重试（网络、5xx、手动「重试」）复用同一个 key，服务端据此只落一条。
- 用户修改内容后 key 作废并重新生成，避免不同内容撞同一个 key 触发 409。
- 提交成功或收到 409 后清空 key；重新进入插件不会重放上一次已成功的提交。

## 测试

`npm test`：19 个用例，全部对 `fetch` 打桩，不依赖真实服务器。

- `src/__tests__/api.test.ts`（7）
  - 成功：URL、方法、`Authorization`、`Content-Type`、body 的 `content` / `capture_key`
  - 401 映射与「去配置」动作；服务端 message 只进 detail 不进文案
  - 5xx / 网络错误可重试；409 冲突；缺配置时不发请求；201 响应体异常
  - baseUrl 归一化与 http(s) 校验
- `src/__tests__/App.test.ts`（12）
  - 成功提交后清空输入、显示「已进入收集箱」、删除草稿、延时关闭窗口
  - 401 保留输入、无自动重试、主按钮变「去配置」
  - 5xx / 网络错误保留输入，重试复用同一 `capture_key`
  - 修改内容后换新 key
  - 重新打开插件不重放、无残留草稿
  - 草稿恢复提示；`onPluginOut` 写草稿
  - 剪贴板预填并全选
  - `Enter` 提交、`Shift+Enter` 不提交、空输入禁用
  - 首次配置保存到 `bridge.setConfig`；非法地址被拒

## 设计说明

- Token 取自根目录 `design.md` 与 `web/src/workspace-theme.css` 的语义：`--radius-surface 16px` / `--radius-field 9px` / `--radius-pill 999px`，状态色沿用 `--color-success` / `--color-danger` 的实际取值；强调色按本次任务指定用 `#4a5cf5`。
- 背景是冷白到浅蓝灰的渐变，卡片为白色面 + 弱边界 + 单一偏移阴影；只有主按钮使用强调色，错误/成功各用自己的语义色。
- 字体使用系统中文无衬线（Inter → PingFang SC → Microsoft YaHei），不加载网络字体，离线可用。
- 书写态与配置态同为 220px 高，切换不跳动；动效只有 120-160ms 的状态过渡，`prefers-reduced-motion` 下关闭。
- 键盘提示用 `<kbd>` 呈现而不是说明文字堆叠；错误文案占位在提示行，不会把卡片撑高。

## 已知限制与未验证事项

- **未在真机 uTools 中加载验证**：本机没有安装 uTools，验证到的是 `dist/` 产物结构、浏览器渲染截图与自动化测试；`utools.*` 的真实行为（`onPluginOut` 触发时机、`hideMainWindow`、`dbStorage` 落库）未实测。
- 剪贴板预填依赖 preload 里的 `require('electron').clipboard.readText()`；浏览器 dev 兜底不读剪贴板。
- `utools.hideMainWindow()` 只隐藏 uTools 主窗口。插件被「分离为独立窗口」时不会关闭该独立窗口，成功提示会继续停留（这是按规范使用 `hideMainWindow` 的结果）。
- 凭证生成入口在 Web 设置页的「快速采集凭证」面板（主仓库同步开发）；插件与真实凭证的端到端联调未验证（未启动 LifeOS 服务联调）。
- 插件固定浅色外观（uTools 主窗口为浅色），未做深色主题适配；粘贴超长文本时输入框内部滚动，未做字数计数器。
