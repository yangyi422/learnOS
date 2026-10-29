# LifeOS Desktop (Wails v3 + Vue 3)

Windows 桌面客户端。只做一件事：把现有的 LifeOS Web 前端装进一个原生窗口，并让它继续调用**远端** Go API。
没有第二套前端源码，没有本地业务数据库，也没有任何后端业务逻辑。

```
desktop/
├── go.mod                      # 独立 module（lifeos/desktop），不引用 learnos 后端
├── main.go                     # Wails v3 入口：embed 前端 + 建窗口
├── internal/bridge/
│   ├── bridge.go               # 127.0.0.1 本地服务：静态资源 + /api 反代 + SPA fallback
│   ├── config.go               # apiBase 解析与校验
│   ├── bridge_test.go          # guard / 代理 / Cookie 重写的单元测试
│   └── bridge_live_test.go     # 可选：对真实 API 的端到端检查
├── frontend/dist/              # 由 web/ 构建产生（gitignore，仅保留 .gitkeep）
├── build/                      # Wails 构建配置（图标、manifest、Taskfile）
├── bin/LifeOS.exe              # 构建产物（gitignore）
├── desktop.config.example.json # 配置模板
└── README.md
```

前端源码仍在 `web/`，桌面端只消费它的构建产物。

## 为什么是「本地服务 + 反代」

Web 端所有请求都是相对路径（`/api/v1/...`），会话是 `HttpOnly` + `SameSite=Lax` 的 Cookie，后端没有 CORS 中间件。
如果让 WebView 直接请求远端地址，就必须放宽 CORS 或改 Cookie 策略——这两件事都被明确禁止。

因此桌面进程在 `127.0.0.1` 上起一个本地 HTTP 服务：

- `/`、`/assets/*` 等：先前端构建产物（`go:embed`，生产不依赖 dev server）；
- `/api/v1/*`：反向代理到 `apiBase`（真实远端）；
- `/healthz`：本地健康检查；
- 其余无扩展名路径：回退 `index.html`，Vue Router history 的深链与刷新可用。

页面与 API 同源，因此相对路径、Cookie、401 跳转 `/login` 全部保持原样，前端零改动、后端零改动。

安全约束：只监听 `127.0.0.1`（随机端口，禁止 `0.0.0.0`）；只代理 `/api/v1/`；方法仅 GET/HEAD/POST/PUT/PATCH/DELETE；校验 Host 与 Origin，其他站点无法驱动本地服务；`apiBase` 只能是 http/https origin，且不能带 path、query 或账号密码，请求参数无法改变上游地址。

### Cookie 说明（重要）

代理转发 `Set-Cookie` 时做两件事：**去掉 `Domain`**（远端域名在 `127.0.0.1` 上会被浏览器判为域不匹配）、**去掉 `Secure`**（本地是 http）。
`HttpOnly`、`SameSite=Lax`、`Path`、`Expires`、`Max-Age` 原样保留，认证逻辑没有放宽。
当前远端是纯 HTTP，服务端本身也不会下发 `Secure`；如果将来远端改 HTTPS，需要重新评估这一处（例如让本地服务也走 TLS）。

## WSL 工具链

| 组件 | 本机使用 | 说明 |
| --- | --- | --- |
| Go | **1.26.10** | `desktop/go.mod` 声明 `go 1.26.0` + `toolchain go1.26.10`，由 `GOTOOLCHAIN=auto` 自动下载到用户缓存；不改 Windows Go、不动 `/usr/local/go` |
| Wails CLI | **v3.0.0** | 安装在 WSL 用户级 `~/go/bin` |
| Node / npm | 22.23.3 / 10.9.9 | 只用于构建前端 |
| CGO | **关闭** | Windows 交叉编译官方默认即不需要 CGO，未安装 MinGW |

安装 Wails CLI（WSL）：

```bash
CGO_ENABLED=0 go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0
export PATH="$HOME/go/bin:$PATH"
wails3 version   # v3.0.0
wails3 doctor
```

> CLI 用 `CGO_ENABLED=0` 安装：Linux 的 GTK/WebKitGTK 后端只在本机跑 GUI 时才需要，交叉编译 Windows 不受影响。

## 配置 API 地址

**默认地址已编译进 EXE**（构建时 `go:embed desktop.config.json`），所以分发只需要一个 `LifeOS.exe`。
运行时仍可按以下优先级覆盖，无需重新编译：

```text
LIFEOS_API_BASE 环境变量  >  EXE 同目录的 desktop.config.json  >  当前目录的 desktop.config.json  >  内置默认
```

- 内置默认值来自仓库里的 `desktop/desktop.config.json`（构建时读取；改动它需要重新构建才会生效）。
- 覆盖文件示例：`{ "apiBase": "http://10.0.0.5:8888" }`，只允许 `http(s)://host:port`，不能带路径或账号密码。
- 启动日志会打印 `config: <来源>`，可确认实际用的是哪一个。
- 当前验证过的远端入口：`http://43.138.202.56:8888`（`GET /` 200，`GET /api/v1/auth/me` 401 JSON，说明它本身同时提供前端与 API）。

## 构建

前端产物（在**与 `node_modules` 匹配的操作系统**上执行；本机是 Windows）：

```powershell
cd web
npm run build:desktop      # → desktop/frontend/dist（不会碰 internal/webassets/dist）
```

Windows EXE（WSL）：

```bash
export PATH="$HOME/go/bin:$PATH"
cd desktop
wails3 build GOOS=windows GOARCH=amd64
```

产物：**`desktop/bin/LifeOS.exe`**（PE32+ GUI, x86-64）。

说明：
- `wails3 build` 里的 `build:frontend` 只校验 `frontend/dist/index.html` 是否存在，**不会**在 WSL 里跑 `npm`：本机 `web/node_modules` 是 Windows 版，WSL 无法加载其原生绑定（rolldown）。Vue 改动后请先在 Windows 重新执行 `npm run build:desktop`。
- 只构建 Windows 目标，没有引入 macOS / Linux / iOS / Android 的构建配置。
- 不覆盖 Web 端：`internal/webassets/dist` 保持不变，Web 构建与部署流程未改。

## 站内链接与外部浏览器

WebView2 没有标签页：页面里 `target="_blank"` 的链接会被交给操作系统，弹出系统浏览器、离开应用。桌面端因此把入口 URL 标成 `http://127.0.0.1:<port>/?shell=desktop`，前端 `web/src/shell.ts` 记住这个标记（sessionStorage，跨整页跳转保留），并把 `/library-tools/...` 这类工具入口改为**同窗口打开**；网页端行为不变（仍是新标签页）。工具页自带「返回资料库」，所以返回路径是完整的。

只有真正的站外链接（例如资料里的 http/https 原文链接）才应该继续交给系统浏览器，这部分行为未改动。

## 全屏与标题栏

**默认以全屏启动**（`StartState: WindowStateFullscreen`）：没有原生标题栏，也不显示任务栏，页面顶部的 LifeOS 导航就是最上面一行。

- **F11**：在窗口化与全屏之间切换；
- **Alt+F4**：退出程序（全屏下没有关闭按钮）；
- **`set LIFEOS_WINDOWED=1`**：以普通 1440×900 窗口启动（保留标题栏），适合调试或双屏并排。

若想“最大化而非全屏”时也去掉标题栏，需要无边框窗口 + 自绘拖拽区与窗口按钮，属于独立改动，本阶段未做。

## 启动不起来时怎么查

`LifeOS.exe` 是 GUI 子系统，双击失败时不会弹控制台。按顺序做三步：

1. **看同目录日志**：`lifeos-desktop.log`（首次运行自动创建，追加写入）记录了配置来源、上游地址、监听端口、WebView2 版本和全部错误。
2. **跑控制台诊断版**：同目录执行
   ```powershell
   .\LifeOS-debug.exe -check     # 只启动本地桥接并自检，打印 apiBase 与 /healthz、/、/api/v1/auth/me 的结果
   .\LifeOS-debug.exe            # 正常启动 GUI，但保留控制台日志
   ```
   `-check` 成功会打印 `apiBase`、每个请求的状态码和耗时；失败会直接给出原因。
3. **确认 WebView2**：缺失时程序会弹窗提示并给下载地址（Windows 10/11 通常自带；精简版/Server 可能没有）。安装后重跑即可。

常见原因与处理：

| 现象 | 原因 | 处理 |
| --- | --- | --- |
| 双击毫无反应，无窗口 | 缺 WebView2 运行时，或启动早期失败 | 跑 `LifeOS-debug.exe`，看控制台/日志；按提示装 WebView2 |
| 弹窗「缺少 WebView2 运行时」 | 同上 | 安装 [WebView2 Runtime](https://developer.microsoft.com/microsoft-edge/webview2/) |
| 弹窗「无法启动本地服务」 | 端口/权限问题，错误原文在弹窗与日志里 | 按错误处理；日志会给出具体 listen 错误 |
| 窗口出现但显示「无法连接到服务」 | `apiBase` 错或服务器不可达 | 修正 `desktop.config.json`，或 `LifeOS-debug.exe -check` 看代理结果 |
| 窗口显示连接错误但 `/healthz` 正常 | 上游 401/404 等业务响应 | 属正常鉴权行为：先登录 |
| 日志出现 `error creating controller with 8000ffff` | WebView2 控制器创建失败（发生在任何导航之前）：用户数据目录不可写、RDP 下的托管模式、杀软/EDR 拦拦截子进程，或运行时损坏 | 已默认改用 `%LOCALAPPDATA%\LifeOS\WebView2` 并开启 visual hosting；仍失败时按下面的三步定位 |

`0x8000FFFF` 的定位顺序：

```powershell
.\LifeOS-debug.exe -minimal                 # 只测 WebView2：出现「Wails + WebView2 OK」说明环境没问题
$env:LIFEOS_DISABLE_GPU='1'; .\LifeOS-debug.exe   # 显卡/驱动相关时使用
$env:LIFEOS_VISUAL_HOSTING='1'; .\LifeOS-debug.exe  # RDP / 混合 DPI 场景
.\LifeOS-debug.exe -check                   # 只测桥接与远端 API
```

cmd.exe 里对应写法是 `set LIFEOS_DISABLE_GPU=1`（PowerShell 是 `$env:...='1'`）。

**用户数据目录**：WebView2 需要一个可写目录，否则 `CreateCoreWebView2Controller` 会以 `0x8000FFFF` 失败（页面还没开始加载）。
程序按顺序探测，取第一个可写的：`LIFEOS_WEBVIEW_DATA` → `%LOCALAPPDATA%\LifeOS\WebView2` → `%APPDATA%\LifeOS\WebView2` → **EXE 同级的 `webview-data\`** → `%TEMP%\LifeOS-WebView2`。
每个候选是否可用都会写进日志，例如 `webview: user-data="…\bin\webview-data"`。某些安全策略会拒绝在 `AppData` 下新建目录，此时会自动落到 EXE 同级目录。

- `-minimal` 也失败 → 属于机器环境：查看日志里的 `webview data dir … unavailable` 行，或手动指定 `LIFEOS_WEBVIEW_DATA`；再不行修复/重装 [WebView2 Runtime](https://developer.microsoft.com/microsoft-edge/webview2/)、检查安全软件拦截。
- `-minimal` 成功、正式模式失败 → 把 `lifeos-desktop.log` 发出来，问题在我们的窗口/资源参数。

## 在 Windows 中运行

1. 把 `desktop/bin/LifeOS.exe` 复制到任意目录；
2. 同目录放 `desktop.config.json`（内容见上）；
3. 双击运行，应出现 LifeOS 窗口：
   - 地址栏/来源是 `http://127.0.0.1:<随机端口>`，不是远端地址；
   - 登录页可正常提交，登录后进入 `/`；
   - 刷新与直接访问 `/projects`、`/explore/library` 等深链仍可用；
   - 任意页面操作写入的数据，Web 端刷新即可看到；Web 端的改动，桌面端刷新即可看到（两端共用同一 API 与 SQLite）。
4. 排查：
   - 窗口显示「LifeOS 无法连接到服务」→ `apiBase` 写错或服务器不可达；
   - 本地健康检查：`curl http://127.0.0.1:<端口>/healthz`（端口在启动日志里）。

## 尚未验证（如实记录）

- **EXE 未在 Windows 上实际运行**：WSL 无法运行 Windows GUI，本轮只验证了交叉编译产物是 PE32+。
- **WebView2 运行时**：Windows 10/11 通常自带；未在干净机器上验证缺失时的表现（本阶段未内嵌 bootstrapper）。
- **真实登录与 Cookie 落盘**：单元测试覆盖了 `Set-Cookie` 重写规则，但浏览器实际存留、`SameSite=Lax` 下的发送、登出清除，仍需在 Windows 中手工确认。
- **深层路由与刷新**：本地服务已用测试验证 SPA fallback；WebView2 中的实际表现待确认。
- **TLS / 公网 HTTP 风险**：当前远端是明文 HTTP，凭据与数据在公网可被窃听或篡改。仅建议在可信网络内使用，或后续为服务器启用 HTTPS。
- 未做代码签名、安装包（NSIS/MSIX）、自动更新、托盘、快捷键、离线缓存——按计划留给后续阶段。
- `wails3 dev` 热重载未接入：`web/` 是用外部 dev server 的独立项目，本机 WSL 也没有 GTK4/WebKitGTK。UI 迭代建议用浏览器 `npm run dev`，界面确认后重建 EXE。

## 测试

```bash
# 单元测试（Guard / 代理 / Cookie），Linux 原生运行，不需要 GTK
cd desktop && go test ./internal/bridge

# 对真实远端做端到端代理检查（可选，需要网络）
LIFEOS_E2E_UPSTREAM=http://43.138.202.56:8888 go test ./internal/bridge -run RealUpstream -v
```

前端回归（保持不变）：`cd web && npm run typecheck && npm run test && npm run build:check && npm run test:ui`。
