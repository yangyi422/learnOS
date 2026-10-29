# LearnOS Roadmap

## uTools 匹配指令：搜索结果直接采集（2026-10-29）

- [x] `plugin.json` 新增 `lifeos-capture-text` 功能：`over`（选中文本）+ `regex`（搜索框任意 ≥2 字符）两条匹配指令，候选文案「保存到 LifeOS 收集箱」
- [x] payload 预填：`preload.js` 缓存早到的 `onPluginEnter` action 并在注册时补发；`App.vue` 按 payload > 草稿 > 剪贴板优先填入并全选，标注「来自 uTools 输入」，不自动提交
- [x] 测试：插件 vitest 27/27（新增 2 项：payload 直接预填并可一次 Enter 提交、payload 优先于草稿/剪贴板且超长忽略）；构建产物浏览器实测 payload 链路（textarea=123234、备注正确、Enter 后成功）
- [x] 修正：uTools 官方明确会**忽略任意匹配的正则**（`/.*/`、`/[\s\S]*/`），原来的 catch-all `regex` 不会生效；改为只保留 `over` 匹配指令，并去掉会误导人的 `regex` 条目
- [x] 诊断：preload 记录最后一次进入事件（code/type/payload 长度与预览，不含凭证）到 `dbStorage`，连接设置面板显示「最近一次进入」，用于区分“没有 payload”与“onPluginEnter 未触发”
- [x] 真机反馈修正（这一轮的真因）：preload 一旦抛错，页面就静默退化为浏览器兜底（配置落 localStorage、无事件无剪贴板）。`preload.js` 改为**完全自包含**：不引用本地文件、`require` 全包 try/catch、取不到 Node http 时退回 fetch、事件注册也包 try/catch，保证 `window.lifeos` 一定存在
- [x] 增加「运行模式」指示（uTools 桥接 / 浏览器兜底）与「最近一次进入」诊断，避免再用截图猜 preload 是否生效
- [x] 测试更贴近真实：`preloadBridge.test.ts` 用 dbStorage/onPluginEnter 桩加载真实 `preload.js`，对本地真实 http 服务断言 method/Authorization/body/status，并验证早到事件被缓冲、注册后补发（28 项全绿）
- [x] 文档写明：改过 preload 后 uTools「重载」不一定生效，需完全退出 uTools 再开
- [x] 第二轮真机定位：面板显示「浏览器兜底」证明 preload 没能把 `window.lifeos` 送到页面世界——新版 uTools 以 `contextIsolation` 运行 preload，隔离世界的赋值页面看不到。改为优先 `contextBridge.exposeInMainWorld('lifeos', …)`，失败才回退 `window.lifeos = …`
- [x] 再补一层页面兜底：若页面本身注入了 `utools`（无 preload），`bridge.ts` 直接用页面 `utools` API 拼等价桥接，并在「运行模式」标注 `uTools 页面兜底（preload 隔离）`
- [ ] 未验证：真机 uTools 中匹配指令的候选排序与 `over` 取值（需在装了 uTools 的机器上确认）

## uTools 快速采集插件 + 采集凭证（2026-10-29）

- [x] 后端最小凭证：`api_tokens` 表（sha256、`capture:inbox` 单一 scope、必填有效期、可撤销、`last_used_at`），schema 32→33
- [x] 接口：`GET/POST/DELETE /api/v1/api-tokens`（仅会话）+ `POST /api/v1/capture/inbox`（Bearer token 或会话，复用 `InboxService.CreateWithKey`）；凭证 principal 强制 `Role=user` + 单一 scope，其他 `/api/v1/*` 仍只认会话
- [x] Web 设置页新增「快速采集凭证」面板：生成（一次性明文 + 复制）、列表（前缀/有效期/最近使用）、撤销
- [x] 独立插件工程 `utools-plugin/`（Vue 3 + Vite，独立 node_modules，不改 web/、desktop/、不新增数据库）：`plugin.json` 指令「LifeOS 收集」「快速记录」、preload 桥接、草稿保留、剪贴板预填、Enter/Shift+Enter、幂等 capture_key
- [x] 跨源处理：插件页对 LifeOS 属跨源，后端不放宽 CORS；改为 preload 的 Node http 请求（`preload-request.cjs`），页面 fetch 仅作浏览器 dev 兜底
- [x] 测试：Go 服务/处理器/迁移共 5 组新增用例；真实服务端到端（登录→发凭证→采集→撤销→401）；插件 25 项 vitest（含真实 http 运输层）；设置页 1 项 Playwright；构建产物浏览器冒烟
- [x] 部署：本地 8080 schema 33 + `api_tokens` 表与索引存在 + 迁移前备份 `learnos-20261029-062332.zip`
- [ ] 未验证：真机 uTools 加载（onPluginOut / dbStorage / clipboard / hideMainWindow）；远端 43.138 需重新部署才有该接口

## 桌面端站内链接不再交给系统浏览器（2026-10-29）

- [x] 原因：WebView2 没有标签页，`target="_blank"` 由系统浏览器处理；桌面端点挖角手帐会跳出应用
- [x] 桌面窗口入口 URL 增加 `?shell=desktop`，新增 `web/src/shell.ts` 用 sessionStorage 记住标记（跨 401→/login 等整页跳转保留）；桌面端工具入口改为同窗口打开，网页端仍为新标签页
- [x] 资料库工具区文案同步为「规划随账号保存（Web 与桌面端共用）」
- [x] 验证：新增 2 项 E2E（桌面 shell 同窗口跳转并可进入工具页、web 仍 `target=_blank`），与既有 library 用例共 3 项通过；typecheck、79 项单测通过；本地 8080 实测 `?shell=desktop` 下链接无 target、纯 web 下为 `_blank`
- [x] EXE 已重建（`bin/LifeOS.exe`、`LifeOS-debug.exe` 均含 `shell=desktop`），便携包同步更新
- [ ] 未验证：Windows 实机点击工具后是否确实留在应用内（需在桌面端点一次）

## 挖角手帐接入账号存档（2026-10-29）

- [x] 后端：新增 `tool_states` 表（`user_id + tool_key` 唯一、`state_json`、`revision` 乐观锁），schema 31→32，随既有 AutoMigrate 迁移
- [x] 接口：`GET/PUT /api/v1/library-tools/:key/state`，复用会话认证；未登录 401、未知工具 404、无存档 404（`TOOL_STATE_NOT_FOUND`）、revision 冲突 409、格式/超限 400（上限 512 KiB）
- [x] 前端：静态工具页启动先读账号存档，404 时提示「上传本机到账号」，编辑后 900ms debounce 写回并带 revision；409 停止自动保存并提示载入账号版本；未登录/离线退回 localStorage 并明示状态
- [x] 保留 localStorage 作为离线缓存与迁移来源，导出/导入 JSON 不变；不写资料三表、不进专题 ZIP
- [x] 测试：service 3 项、handler 3 项（含生产路由 401）、迁移 1 项（建表 + 唯一索引 + 重开幂等）、新增 `library-tool-sync.spec.ts` 3 项 E2E；`go test ./...`、typecheck、79 项单测全通过
- [x] 部署验证：容器 healthy、`schema_version=32`、真实库中 `tool_states` 与 `idx_tool_state_user_key` 存在、迁移前自动备份 `learnos-20261029-053607.zip`；未登录访问工具页显示「未登录 · 仅保存在本机」且本地缓存仍写入
- [x] 修正部署脚本：此前只同步 `web/`，导致后端改动不进镜像；改为按子树同步 `internal/ cmd/ deploy/ web/ …`（不使用 `--delete`，保护镜像侧 `.env`、`data/`、`backups/`）
- [ ] 未验证：已登录用户的真实读写在浏览器中的完整往返（本地无该账号密码，仅以 Go 集成测试与 mock E2E 覆盖）

## 4K 大屏适配（2026-10-29）

- [x] 定位：3840 视口下首页仍被 1680px、标准页被 1248px 上限卡住，内容缩在左上角（实测 shell 宽 1680/1248，右侧空白过半）
- [x] 新增两级视口尺度步进（≥2200px、≥3200px）：标准页宽 1248→1440→1640，首页 1680→2000→2480，页面边距 32→40→56，卡片内边距 28→32→36，阅读 760→880、课件 800→920，首页栏距 24→28→32
- [x] 首页/顶栏横向留白改为 `--home-gutter`（36→40→56），消除 4K 下顶栏与页面对不齐
- [x] 字体与控件高度不随视口放大：物理字号交给操作系统缩放，避免与 150/200% 缩放叠加
- [x] 等价性：1440/1920 四路由构建产物逐像素完全一致；2560/3840 按预期变化且无横向溢出
- [x] typecheck、94 项 E2E（ui-foundation/workspace/home-material）通过（1 项并发抖动重跑通过）；本地 8080 与桌面 EXE 均已更新
- [ ] 未验证：你那台 4K 显示器在 100% 与 150% 系统缩放下的实际观感

## 桌面端单文件分发（2026-10-29）

- [x] `desktop/desktop.config.json` 通过 `go:embed` 编入 EXE，分发只需一个 `LifeOS.exe`；运行时仍支持 `LIFEOS_API_BASE` > 同目录/当前目录配置文件 > 内置默认
- [x] 配置解析顺序与来源写入 `LoadConfig(embedded []byte)`，新增单测覆盖「内嵌默认生效」与「环境变量覆盖内嵌」
- [x] 该配置改为纳入版本管理（此前 gitignore），保证任何机器 `go build` 都能编译通过；地址为作者自有服务器，不含任何凭据
- [x] 重新构建并确认 EXE 内含地址与 `embedded default` 标记；便携包更新为 `LifeOS.exe` + 可选调试版
- [ ] Windows 实机确认：删除同名配置文件后仍能用内置地址连上服务器

## 桌面端默认全屏（2026-10-29）

- [x] 启动状态改为 `StartState: WindowStateFullscreen`：默认无原生标题栏、不显示任务栏；`LIFEOS_WINDOWED=1` 可回到 1440×900 窗口
- [x] F11 仍可切换全屏/窗口；全屏下用 Alt+F4 退出（无关闭按钮）
- [x] 说明取舍：仅"最大化"隐藏标题栏需要无边框窗口 + 自绘拖拽区与窗口按钮，属独立改动，未在本轮实现
- [x] `bin/LifeOS-debug.exe` 与 `bin/LifeOS-new.exe` 已含新逻辑；`bin/LifeOS.exe` 运行中被占用，关闭应用后改名替换即可
- [ ] Windows 实机确认：全屏启动是否正常、F11 是否被网页抢键、全屏下滚动与尺寸

## 默认主题改为浅色（2026-10-29）

- [x] `web/src/theme.ts` 兜底值与 `savedPreference()` 回退改为 `light`；`web/index.html` 首屏内联脚本默认浅色，`meta[theme-color]` 缺省 `#e1eaf5`
- [x] 已保存的 `learnos-theme` 始终优先：`dark` 仍为深色、`system` 仍跟随系统；不清空既有偏好
- [x] 更新 `ui-foundation` 用例为「默认浅色 + system/dark 偏好仍生效」，并补 `theme-color` 断言；117 项 E2E（ui-foundation/workspace/home-ambient/visual-audit）、79 项单测、typecheck 通过
- [x] 本地 8080 重新部署后实测：OS 深浅两种设置下新会话都是浅色，`saved=dark` 为深色、`saved=system` 跟随 OS
- [x] 桌面端重新构建：`bin/LifeOS.exe` 与 `bin/LifeOS-debug.exe` 均含新默认（`desktop/frontend/dist` 已重建）
- [x] `design.md` 与 `docs/ui-design.md` 中的「默认深色」表述同步更新

## 首屏"无限加载"兜底与桌面诊断（2026-10-29）

- [x] 定位：Web 与桌面共有的首屏问题来自历史代码——`load()` 一旦请求不返回，`loading` 永远为真，骨架没有人任何出口（HEAD 版本即如此，非本轮引入）
- [x] 前端：`load()` 增加 20s 上限（超时显示「加载超时」+ 重试）、`currentUser()` 启动等待 10s 上限、骨架 6s 后显示「仍在加载…」提示；不再出现无出口的骨架
- [x] 桌面桥接：记录每个被代理的 `/api/v1/*` 调用的状态码与耗时，日志可直接指出页面停在哪个请求
- [x] 验证：mock 挂起 `/workspace/home` 时，6.9s 出现提示、20.5s 切换为错误+重试（骨架消失）；86 项 workspace/home-material E2E、79 项单测、typecheck 通过
- [ ] 待确认：真实环境首屏卡住的具体请求（需查看新的 `lifeos-desktop.log` 代理耗时行，或 Web 端 DevTools Network）；远端 Web 需重新部署才生效

## 桌面端接入：Wails v3 + Vue 3（2026-10-29）

- [x] 新增独立 `desktop/` Go 模块（`lifeos/desktop`，`go 1.26.0` + `toolchain go1.26.10`），不引用后端代码、不改现有 go.mod
- [x] Loopback 桥接：`127.0.0.1` 随机端口提供 `go:embed` 前端，`/api/v1/*` 反代到配置的远端，`/healthz` 本地健康检查，history 深链回退 index.html
- [x] 安全边界：只绑回环、只代理 `/api/v1/`、方法白名单、Host/Origin 校验、`apiBase` 仅允许 http(s) origin；`Set-Cookie` 去除 `Domain`/`Secure` 以适配 loopback，保留 HttpOnly/SameSite/Path/Max-Age
- [x] 前端复用：新增 `web/vite.desktop.config.ts` 与 `build:desktop`，产物写入 `desktop/frontend/dist`；`internal/webassets/dist` 与现有 Web 构建/部署流程未改（index.html SHA 未变）
- [x] 工具链：Wails CLI v3.0.0 装入 WSL 用户级 `~/go/bin`；`wails3 doctor` 通过；Windows 交叉编译 `CGO_ENABLED=0`，未安装 MinGW
- [x] 构建产出 `desktop/bin/LifeOS.exe`（PE32+ GUI x86-64），前端资源已嵌入
- [x] 验证：desktop 单元测试（Guard/代理/Cookie）通过；对真实远端 `http://43.138.202.56:8888` 的端到端代理检查通过；前端 typecheck + 79 项单测 + build:check 通过
- [ ] Windows 实机：运行 EXE、WebView2、登录与 Cookie 落盘、深层路由刷新待手测

## Phase 2A：PageHeader 视觉层级（2026-10-29）

- [x] 业务页 `PageHeader h1` 字重 700→500、字距 `-0.02em`，尺寸仍为 28px；首页 44px display 标题由自身 `--font-home-display` 规则覆盖，未受影响
- [x] 描述文字改用 `--color-text-secondary`，色阶下沉一级；字号、行高、间距、actions 布局与面包屑未改
- [x] 实测描述字号当前为 `--font-body`(16px) 而非 14px，按"不改高度/间距"约束保持不动，差异已在交付说明中标出
- [x] 构建产物 A/B：10 条业务路由 × 双主题 × 1440/390 下 PageHeader/描述/actions 几何与行数**完全不变**，仅标题文字像素变化；无 PageHeader 的路由 0 差异
- [x] typecheck、79 项单测、隔离构建、40 项核心 E2E（ui-foundation + regression）通过；本地已更新为 `index-DDiPQ18a.css` 并在 8080 复核 computed 字重/字距/描述色

## Phase 1：方向 B Token 语义收敛（2026-10-29）

- [x] 标题尺度分层：业务页固定 `--font-page-title` 28px，新增 `--font-home-display` 44px 仅首页使用，首页 h1 改为引用 token
- [x] 圆角语义化：`--radius-surface`(16) / `--radius-field`(9，实际生效值) / `--radius-pill`(999)；`--radius-card/-panel/-control/-button` 降为兼容别名，消除 `--radius-control` 0px 与 9px 的双重定义
- [x] 材质收敛为同一家族三配方：`--material-surface-*`（操作卡片）、`--material-mist-*`（首页羽化，`--home-surface-*` 为别名）、`--material-reading-*`（阅读面板）；`--glass-*` 全部降为别名
- [x] 明确 `--color-accent` 为唯一品牌强调色，success / warning / danger 语义色保留不受约束
- [x] 等价性验证：改前/改后构建产物在 5 条路由 × 双主题下逐像素完全一致；首页 20 张基准 18 张逐位相同、2 张最大差 1 级；跨页 44 张 40 张逐位相同
- [x] typecheck、79 项单测、隔离构建、40 项核心 E2E（ui-foundation + regression）、13 项视觉检查通过；本地服务已更新至 `index-BHkur234.css` 并复核 computed token
- [ ] 遗留：`--home-*` 调色名仍承担全局角色、Element `--el-border-radius-small: 0px` 与 field 9px 不一致、表格/learning-composer 12px 未语义化、业务组件 4 处 eyebrow 待 Phase 2 决策

## 轻量 AI 前端视觉工作流初始化（2026-10-29）

- [x] 在既有 AGENTS / 设计规范 / 评审清单中建立参考读取、差异提炼、组件限制判断、局部实施、截图复核与最多两轮自动修正；优先级为结构/信息层次 → 组件语言 → 装饰
- [x] 区分全局与局部 Token，固定已验收浅色冷白蓝灰/轻玻璃及现有深紫方向；Skill / MCP 缺失不伪造研究或强制安装
- [x] 复用 Playwright 与现有 visual-audit / home-material：`test:ui` 默认保存固定时间/数据的首屏、整页 PNG 与来源 JSON；支持 `PLAYWRIGHT_BASE_URL` 验证已运行前端，不新增浏览器平台或依赖
- [x] 新增 `build:check` 隔离检查构建；现有 PowerShell 验证入口保留真实 E2E 失败退出码、区分显式告警与通过，截图阶段不覆盖前一阶段输出
- [x] 源码与实际 8080 前端各 13 项视觉检查通过，已核验 10 组 PNG/JSON 证据并抽查双主题桌面/手机；非首页沿用既有核心回归，未修改页面与业务源码
- [x] 对齐已上线首页控件的 3px 焦点外扩断言，保留 2px 轮廓与对比度检查；不为了旧断言重画 UI
- [x] Typecheck、79 项单测、40 项核心 E2E 与隔离构建通过；验证脚本语法、启动失败退出码及显式告警分支通过模拟验证，未重启正常运行环境

## 首页层级与扫描节奏（2026-10-28，参考 Refero Styles）

- [x] 参考 dope.security / Authkit / Dimension：主方向取 Dimension 的"工作台玻璃面板 + 克制 500 字重 + 细边无阴影"，仅借 Authkit 四级前景阶梯与 dope.security 数字 stamped 处理
- [x] 首页模块标题降为标签层（14px/600、次级色、.04em 字距，六卡共用 `--home-label-*`），任务/项目/文章标题成为第一层
- [x] 模块计数（`N 个项目` / `N 条`）以 tabular numerals 呈现，作为扫描锚点；不新增装饰性 eyebrow
- [x] 行内更紧（项目行 16px、最近行 11px、行内 gap 4–5px），模块间更松（header 下 20px），列位置与宽度不变
- [x] 大标题 600→500、-0.02em；输入区圆角 12→10px，与卡片 16px、控件 999px 组成半径家族
- [x] 修正 `workspace.spec.ts` 中依赖旧 `background-clip: padding-box` 的过期断言（卡片已无元素级填充）
- [x] 按上线现状统一文档与测试：继续学习固定在中栏焦点卡下方（与 `docs/ui-design.md` 早先记录一致），右栏只保留快速记录与最近内容；`design.md` Phase 11.1d 与两处 E2E 断言同步更正

## 首页双主题材质统一精修（2026-10-28）

- [x] 删除首页 Hero / Primary / Secondary / Reading 视觉分支；六卡统一圆角、薄膜、弱边界、羽化、Hover / Focus，不改内容、布局和尺寸
- [x] `--home-surface-*` / `--home-control-*` 仅首页生效；导航、搜索、快捷记录、头像、场景筛选与记录入口统一轻玻璃基调
- [x] 浅色抬升基底、减弱灰蓝沉降与局部白斑；深色保留深紫光场，统一低对比卡面，不新增 Canvas / Shader / 动画框架
- [x] 六卡一致性、双主题 390–1920px、键盘焦点、选中/禁用状态、静态降级声明与路由隔离自动化验证
- [x] 相同接口验收数据下，1440/390px 全部卡片与顶部控件的 bounds、字体、网格尺寸前后一致；项目页材质不变
- [x] 逐视口与滚动位置抽样文字对比度：浅色最低 5.20:1，深色最低 5.38:1；文字与内容层不使用材质遮罩
- [ ] Safari / Firefox 的原生遮罩表现与真实触屏设备视觉复核

当前规则以 [设计规范](../design.md#3-光场与空间) 为准，以下为历史阶段记录。

## 首页浅色 Soft Mist Glass 视觉可见性纠偏（2026-10-27）

- [x] 按最终截图采用冷白蓝灰柔雾背景、弱边界冷白膜与极低接触阴影，移除 Hero 亮面
- [x] 首页浅色导航融入画布，保留原高度、选中态和全部操作；不推广到其他业务页
- [x] 浅色仅保留克制 Ambient，停用实验折射与 Spotlight 亮斑；深色行为保留
- [x] 取证实际8080资源、命中规则、computed style与临时条纹：未发现旧CSS覆盖或嵌入产物失效；此前差异被平坦光照和较厚冷白膜削弱
- [x] 重新组织内容后方蓝灰漫射与上下冷白区域；既有光场在浅色从视口顶部开始，导航不再与环境断开
- [x] 量化定位：旧全宽巨型渐变虽然 CSS 生效，但整屏可见亮度差仅约20级，低于可察觉阈值；不是覆盖或缓存问题
- [x] 重排为「左上冷白高光 + 右侧蓝灰漫射 + 左下蓝灰沉降」三段中尺度光区，可见亮度差提升到32–36级
- [x] 卡片材质改为独立羽化层：元素不再绘制矩形填充，`::before` 承载材质并四边羽化 26px，边缘不再终止于描线
- [x] 边缘渐变带 30px→58px、最大相邻跳变下降；填充提高到 56%/44%/48% 后卡片仍有柔和存在感；深色逐像素无回归
- [x] 验证 CSS / WebGL 光场互斥且可见，关闭光场后像素变化、卡片几何不变，无额外 Context
- [x] 1440px / 390px 深色截图逐像素无差异；浅色文字采样对比度最低4.95:1，交互状态及 Reduced Motion 回归
- [ ] Safari / Firefox 与真实触屏设备复核

定稿规则见 [设计规范](../design.md#3-光场与空间)，前序光场记录作为历史保留。

## 首页 Wope / Dimension 光场精修（2026-10-27）

- [x] 移除深色焦点卡宽幅顶光，使用首页单一共享底部紫色反照；浅色减弱均匀染色并加入极弱暖反照
- [x] CSS / WebGL 共享局部光场参数，主题与尺寸同步；其他业务页保留原光场
- [x] 桌面1440px、手机390px、Reduced Motion、主题切换与光场生命周期回归
- [x] 最终收尾：深色光域压扁为定向 Horizon，浅色仅增强冷灰轮廓与接触阴影；到此停止玻璃与光场微调
- [ ] Safari / Firefox 与真实触屏设备人工复核

当前规则见 [设计规范](../design.md#3-光场与空间)。

## B 方案首页：棱光学习工作台（2026-10-16）

- [x] 按选定 B 构图实现首页横向导航、不等宽三列与唯一共享紫色光场
- [x] 保留真实焦点、项目、今日任务、学习位置、最近内容与快速记录入口
- [x] 搜索仅过滤当前已加载工作区条目；项目进度使用真实任务总数，不虚构日程时间
- [x] 中文字体与桌面/手机字阶校准；手机顺序为重点、今天、学习、项目、最近内容
- [x] 保留深浅主题和其他业务页侧栏；不修改后端、不部署
- [ ] Safari/Firefox 与真实触屏设备复核

实现边界、数据语义与验证记录见 [WORKSPACE_PRISMATIC_HOME.md](WORKSPACE_PRISMATIC_HOME.md)。本节为首页当前规则，前序精修记录保留作历史。

## 首页 critique 后的舒展精修

- [x] 探索内容字阶微调，手机课程图标独占引导行，正文恢复卡片内完整宽度
- [x] 手机日期与快速记录同行，主任务提前出现，保留44px触控区域
- [x] 模块间距略放宽，普通/弱化玻璃表面区分，保留紫光与深浅主题
- [ ] Safari/Firefox及真实触屏设备的材质和排版复核

范围与验证见[首页最终精修文档](WORKSPACE_HOME_FINAL_POLISH.md#critique-后的舒展精修)。

## LifeOS 生活档案 V0.1

### 第二轮体验完善

- [x] 目标概况分组、动机/当前判断空提示与真实更新时间
- [x] 阶段记录编辑、日期调整、确认删除，状态前后值保留
- [x] 阶段变更与目标更新时间事务提交，归属校验与失败回滚
- [x] 目标摘要与更新时间、时间线真实目标/来源名称
- [x] 保存后列表即时更新、阅读优先、Esc 与焦点返回、未保存保护


- [x] 独立生活入口，时间线与长期目标两个视图
- [x] 事件持久化、补录、阅读/编辑/删除、双领域和里程碑筛选
- [x] 长期目标状态、主动阶段记录与判断历史、项目/事件关联
- [x] Inbox 转生活事件的原子处理与来源/请求去重
- [x] 完成项目与正式结业课程手动收录，删除来源保留档案快照
- [x] 安全 URL/Obsidian 与复制能力复用，当前用户认证保护
- [x] Schema 21 可重复迁移、整库备份与生活数据导出
- [ ] 用户真实设备 Obsidian 唤起与长期使用体验反馈

使用、数据契约、迁移和回退见 [LIFE_ARCHIVE.md](LIFE_ARCHIVE.md)。

## 轻量课件与自然对话

- [x] 现有学习页整合课件、自由多轮对话和固定输入/课程推进
- [x] 单模型调用、先保存消息、失败重试及历史恢复
- [x] 校验用户原文证据、稳定完成建议与用户确认
- [x] 完成/跳过与下一课定位原子提交，进度与掌握度独立
- [x] 保留知识来源、地图、详细评价、挑战和历史记录，按需展开
- [x] Schema 19 兼容新增课件正文，保留所有旧数据
- [ ] 使用真实模型与个人课程检查教学质量、证据判断及长时间使用体验

契约、迁移与回滚见 [LEARNING_LOOP.md](LEARNING_LOOP.md)。

## Workspace Home UX Polish（已完成）

- [x] Current Focus 显示真实归属、状态、固定原因和最近活动时间，不调用 AI
- [x] Today 按逾期、今天到期、进行中、下一步排列最多三条可直接打开的任务
- [x] Active Projects 保持轻量列表并突出真实项目图标、强调色和任务数量
- [x] Continue Learning 突出最近课程；空知识世界提供明确的首次学习入口
- [x] Inbox 首页增加 Enter / 按钮行内快速记录，并保留顶部快速记录对话框
- [x] 首页各区统一骨架加载、真实空状态、可重试错误和隔离浏览器回归

本轮复用 Workspace 聚合 API 与 Inbox 创建接口，不增加业务模块、后端字段或数据库迁移。

## Personal Workspace Home + Inbox（本次）

- [x] `/` 改为真实工作区首页，原学习首页完整迁移到 `/learn`
- [x] 首页按确定性规则推导 Current Focus，并复用 Today 服务端分组
- [x] 汇总最多 5 个活动项目、最多 2 个真实学习领域和 3 条待整理 Inbox 内容
- [x] Schema 18 `InboxItem` 用户隔离、SQLite 持久化、URL 来源记录与可重复迁移
- [x] 快速记录、编辑、待整理/已处理/已归档列表、归档与删除
- [x] Inbox 转任务使用单事务、重复转换保护和失败回滚
- [x] Home/Inbox 路由与轻量窄导航入口，保留完整看板和学习模块
- [x] JSON/Markdown 导出包含 Inbox；补充服务、迁移、事务与前端回归检查
- [ ] 使用个人真实数据进行最终人工验收并观察首次 Schema 18 启动备份

实现规则见 [PRODUCT.md](PRODUCT.md)、[ARCHITECTURE.md](ARCHITECTURE.md) 与 [DATABASE.md](DATABASE.md)。

## 个人数字工作空间视觉基线（本次）

- [x] 建立长期界面设计规范，并在 `AGENTS.md` 引用
- [x] 学习首页按首次使用、待开始与可继续学习三种真实状态呈现
- [x] 悬浮式应用外壳、独立导航轨道与工作区主题材质精修
- [x] 首次使用页把欢迎标题并入唯一创建引导区
- [x] 窄图标导航、工作区顶栏和项目看板的比例与层级调整
- [x] 深色、浅色、跟随系统主题及本机偏好持久化
- [x] 现有页面基础主题、Element Plus 控件和减少动态效果支持
- [x] 第二轮收紧项目页顶部结构，开放式看板分列与深浅主题卡片材质
- [x] 看板任务卡片摘要排版与任务详情编辑抽屉精修，覆盖深浅主题浮层
- [x] 任务抽屉边界、灰阶、遮罩、长说明与键盘焦点交互精修
- [x] 看板整卡拖动与设置页主题切换
- [x] 导航选中指示线与原生悬停提示，消除提示造成的侧栏横向滚动
- [ ] 逐页细化旧学习模块与知识图的主题细节

长期设计规范见 [ui-design.md](ui-design.md)；现有实现说明见 [WORKSPACE_VISUAL.md](WORKSPACE_VISUAL.md)。

## Project Kanban Workspace（本次）

- [x] 默认全部项目的四列看板、单项目切换与 URL 状态
- [x] 项目列表、项目目标/外观、暂停与归档
- [x] 任务描述、优先级、到期日、Task Drawer 和 Today
- [x] 跨列拖动、列内排序、失败恢复与后端持久化
- [x] Schema 17 保数迁移、迁移前备份和 Done 历史分页
- [x] 项目/任务 API、归属校验和迁移回归测试
- [x] 修复到期任务再次编辑说明时的日期格式错误，并补充 API 回归测试
- [x] 看板到期日选择器使用简体中文界面
- [ ] 真实移动端拖动、多设备和生产升级人工验收

现行语义见 [PROJECTS.md](PROJECTS.md)。原“项目执行中心初版”设计见 [PROJECT_EXECUTION.md](PROJECT_EXECUTION.md)。

## 项目执行中心初版（历史）

- [x] 独立于课程的个人项目、任务、状态与唯一下一步行动
- [x] 项目/任务增改、任务删除与按用户隔离的 API
- [x] 项目执行页面和导航入口
- [x] Schema 16 可重复迁移、迁移前备份和项目数据导出
- [x] 项目流程、输入校验与跨用户隔离回归测试
- [x] 修复 Schema 14 旧领域草稿升级到用户系统时的非空列迁移失败
- [ ] 人工验收多用户页面、移动端布局和生产环境升级

本阶段不实现笔记、Obsidian 同步、提醒、日历、AI 自动修改或项目协作。详细契约见 [`PROJECT_EXECUTION.md`](PROJECT_EXECUTION.md)。

- [x] 登录区分凭据失败与服务故障，避免重复提交；输入框聚焦使用单层细边框

## Phase 1：可运行骨架（本次）

- [x] Go + Gin 服务
- [x] SQLite + GORM
- [x] Vue 3 + Element Plus
- [x] 课程列表 API
- [x] 营养学种子课程
- [x] Docker 部署骨架
- [x] 部署显式更新远端引用并校验 Actions 提交 SHA，兼容旧版 Git
- [x] 单用户 Basic Auth
- [x] 多用户会话登录与管理员创建用户（课程级数据隔离）
- [x] 按角色显示用户管理入口与退出登录

## Phase 2：不依赖 AI 的最小学习闭环（已完成）

- [x] course_units、lessons、learning_turns、mastery_records 与 misconceptions 数据模型
- [x] 当前课程、模块和知识点的幂等初始化
- [x] 当前问题页面与首页继续学习跳转
- [x] 提交用户回答与安全校验
- [x] 固定模拟评价与反馈展示
- [x] learning_turns 持久化
- [x] 在事务中创建或更新知识点掌握度
- [x] 最近回答记录查询与刷新后保留

本阶段不接入 DeepSeek，不实现 JSON Schema AI 输出校验；固定评价将在 Phase 3 替换。

## Phase 3：DeepSeek AI 结构化认知评价（本次完成）

- [x] AI_PROVIDER 配置与 Mock / DeepSeek Provider 抽象
- [x] DeepSeek OpenAI-compatible Chat Completions 接入
- [x] JSON Output、Prompt v1、严格业务结构校验
- [x] 超时、有限重试和安全错误处理
- [x] LearningTurn 结构化评价快照
- [x] AIEvaluationRun 调用审计
- [x] MasteryRecord 更新和 Misconception 精确去重
- [x] 学习页和历史页展示 AI 评价内容

本阶段不实现课程状态机、下一知识点选择或复习调度。

## 后续课程状态与复习能力

- [ ] 初始化、学习、追问、复习、实践、暂停、结课状态
- [ ] 下一知识点选择策略
- [ ] 阶段复习
- [ ] 到期复习

## Phase 4：知识世界与课程结构（已完成）

- [x] Lesson 静态角色、核心/扩展和 Depth 元数据
- [x] LessonRelation：prerequisite / extends / application / related
- [x] prerequisite DAG 校验、拓扑顺序和图统计
- [x] 营养学知识骨架的分支、平行节点和汇合点
- [x] Knowledge Graph API 与 Lesson 关系 API
- [x] 基础知识结构查看页 `/courses/:id/map`
- [x] 保留已有 Lesson ID 与 Phase 2/3 学习历史

Phase 4 不实现个人认知状态、自动解锁、路线推荐、AI 修改知识图或最终视觉地图。

## Phase 5：个人认知状态与认知演化（已完成）

- [x] CognitiveState / CognitiveEvidence / CognitiveStateEvent
- [x] Evaluation Prompt v2 与严格 Schema / Target 校验
- [x] Recognize / Understand / Apply / Transfer 层级和 State Transition
- [x] 认知状态、证据和演化时间线 API
- [x] LearningView / KnowledgeGraphView 认知状态展示
- [x] 保留旧 Phase 2/3 学习历史，不自动 backfill 认知等级

Phase 5 不实现自动解锁、路线推荐、迁移测试生成器、误区网络或 Agent 导航。

## Phase 6：迁移测试、误区网络与认知修正（已完成）

- [x] AssessmentChallenge / ChallengeAttempt 与独立挑战审计
- [x] transfer generator / challenge evaluator 及严格业务校验
- [x] Transfer Challenge 通过规则与 CognitiveState 影响规则
- [x] Misconception active / resolved / reopened 生命周期和事件
- [x] 固定 reasoning pattern taxonomy 与误区网络 API
- [x] 针对性 Misconception Recheck 与 Correction Validation
- [x] LearningView 挑战交互、误区网络页、知识结构迁移状态 overlay
- [x] AutoMigrate 和旧 Phase 2/3 数据兼容
- [x] non-substantive Transfer Answer 的本地正式判定与状态保持
- [x] Challenge Generation attempt/provider/validation 性能日志（脱敏，不输出 Prompt 或 RawResponse）

Phase 6 不实现探索雷达、问题池、Agent 导航、知识来源可信度或最终图形化认知地图。

## Phase 7 preparation：测试知识世界扩充（已完成）

- [x] 补全营养学现有 8 个 Lesson 的学习字段与测试题面
- [x] 新增逻辑与科学思维测试知识岛（3 个 Lesson）
- [x] 新增心理学测试知识岛（3 个 Lesson）
- [x] 静态 Course / Unit / Lesson / LessonRelation Seed 幂等
- [x] 保留已有 Lesson ID 与 Phase 1~6 历史，不伪造用户认知数据

本准备任务不实现探索雷达、陌生知识入口、问题池、推荐算法、Agent、AI 课程生成或 CrossCourseLessonRelation。

## Phase 7：Exploration Engine（已完成）

- [x] 基于 Knowledge Graph、CognitiveState、Misconception 和 reasoning pattern 的探索候选生成与评分
- [x] 探索雷达 API 与首页轻量入口
- [x] 陌生知识入口与跨课程候选
- [x] ExplorationDirection 保存、略过、打开和历史
- [x] ExplorationQuestion 问题池、启动和归档
- [x] AI 方向文案生成、结构校验与规则文案降级
- [x] 探索支线 Lesson 读取/回答，不改变课程 CurrentLesson
- [x] 不因打开探索方向创建 LearningTurn、CognitiveEvidence 或修改 CognitiveState

## Phase 8：Curriculum Construction & Coverage（进行中）

- [x] CurriculumBlueprint / BlueprintUnit / BlueprintLesson / BlueprintRelation 模型与 Nutrition v0 种子
- [x] 课程覆盖度 API；Coverage 只描述学科蓝图与正式 Lesson 的映射，不读取个人认知状态
- [x] CurriculumDraft ChangeSet、严格 AI 草稿协议与人工 Apply / Reject API
- [x] Apply 事务、DAG 校验、旧 Lesson ID / CurrentLesson / 学习历史安全边界
- [x] Course Archive 课程覆盖度与扩充草案审核界面
- [x] 主线 CurrentLesson 显式切换、规则型下一课推荐和 Knowledge Structure 学习入口
- [x] Blueprint Unit Expansion 与按 Unit 生成 Curriculum Draft 的渐进式课程生长入口
- [x] 知识结构页按 Unit 剩余重要性选择课程草案生成范围，避免继续生成节点的空草案请求
- [x] LearningView 下一步、首页继续/选择/展开知识世界联动
- [ ] 人工验收蓝图覆盖、草案 Apply 和回滚边界后再标记本阶段完成

本阶段不实现 AI 直接发布课程、自动删除或修改既有 Lesson、搜索、RAG、来源系统或自动扩图。

## Phase 9：Source, Credibility & Grounding（进行中）

- [x] KnowledgeSource Registry 与跨 Course 去重
- [x] SourceEvidence、GroundingLink 与三类 Grounding Target
- [x] 人工 Credibility Assessment，明确可信度不等于真值
- [x] Grounding 状态统一重算、冲突状态和审核事件
- [x] Grounding Coverage API 与课程档案 / Lesson 页面入口
- [x] 暂时隐藏来源校验前端入口；后端 API、数据和审核能力保留
- [x] 不触碰 CognitiveState、LearningTurn、CurrentLesson 和既有 Lesson 内容
- [ ] 人工验收来源、证据、连接审核和冲突边界后再标记本阶段完成

本阶段不实现搜索、抓取、RAG、向量、Embedding、OCR、Zotero 同步、AI 自动宣判真值或自动扩图。

## 后续长期使用能力

- [x] Phase 10 Dogfooding Readiness：诊断、一致性检查、SQLite 备份/恢复、JSON/Markdown 导出、优雅停机和 AI 失败可重试提示
- [ ] 人工完成 Phase 10 生产初始化、备份恢复演练和长时间运行验收后关闭稳定化阶段
- [ ] 模型切换
- [ ] 调用成本统计
- [x] v0.1 Quiet Intelligence UI 收口：统一 AppShell、PageHeader、状态 tokens 与响应式布局
- [x] v0.1 Visual Redesign：Current Focus、Compact Domain Tile、Discovery Card、Editorial Learning Flow 与 Vertical Knowledge Path
- [x] v0.1 Final UI Unification：Learning、Curriculum Coverage、Knowledge Map 与课程档案同步首页视觉语言
- [x] v0.1 Knowledge Map Graph：保留列表视图，新增 Vue Flow + dagre 地图、待生成节点、关系/Unit 筛选、CurrentLesson 与认知状态 overlay
- [x] Knowledge Map Graph Interaction Completion：节点选中与共享详情 Rail、显式开始/继续学习、Blueprint Unit Expansion / Draft 操作及移动端详情 Drawer
- [x] Scroll-driven Vertical Knowledge Tree：路径视图改为 DOM 纵向 DAG、IntersectionObserver 自动选中、关系定位、CurrentLesson 初始定位与滚动恢复
- [x] Unit Expansion 幂等保护：数据库状态抢占、并发请求隔离、完整结果事务写入与待审核 Curriculum Draft 防重复

## v0.1 Progressive Empty World Bootstrap（已完成基础实现）

- [x] production / dogfooding 空知识世界与 development Demo World 分离
- [x] Domain Skeleton、Starter Blueprint Expansion、Initial Knowledge World 三阶段独立持久化与 API
- [x] 阶段级超时、结构校验、失败保留前序结果、可重复重试与幂等 Apply
- [x] Apply 前不创建正式 Course / Blueprint / Lesson，Apply 后不创建个人认知数据
- [x] 首页空世界入口与 `/domains/new` 三阶段审核界面
- [x] Domain Initialization 可重复创建多个独立 Course，课程档案页始终提供新建入口
- [ ] 人工完成真实 DeepSeek、超时重试和 production 数据目录验收
- [x] 系统设置页支持 API Key 配置，并在保存后立即切换运行时 Provider
- [x] 课程档案和学习首页支持带二次确认的单课程删除
- [x] 修复课程菜单删除交互链路，确认后实际调用删除 API，并反馈成功或失败结果
- [x] 本地开发服务仅监听 `127.0.0.1`，development 跳过 Basic Auth，生产认证保持不变

## 重构前可验证测试基线（已完成）

- [x] 审计前端路由、页面本地状态、API 封装和课程/领域/知识节点模型
- [x] 记录生成进度、课程进度、掌握度、节点类型和探索归属的当前计算方式
- [x] 增加空领域、当前 Lesson、探索归属和 API Key 脱敏的 Go 回归测试
- [x] 增加 Vitest 知识路径数据一致性测试和 Playwright 核心页面回归矩阵
- [x] 补齐 Vue 类型检查、前端测试脚本和浏览器测试最小配置
- [x] 记录完整测试套件的既有阻断与尚未覆盖风险

本基线不开始状态管理、API、数据模型或页面业务重构。详细审计见 `docs/TEST_BASELINE.md`。

## 数据一致性与用户信任（已完成）

- [x] 探索 DTO 分离推荐上下文与候选节点真实来源，并校验 Course/Lesson 归属
- [x] 领域切换清空旧推荐、隔离异步响应，推荐请求禁用浏览器缓存
- [x] 统一五类知识节点类型，所有视图使用同一映射，Depth 不再参与类型推断
- [x] 拆分课程生成度、学习覆盖度和理解掌握度，旧 `progress` 仅保留兼容
- [x] 幂等修复“已有有效当前 Lesson 但仍初始化中”的旧状态
- [x] 增加来源归属、缓存隔离、类型映射、进度计算和跨视图一致性回归测试

本阶段不新增数据库列、不删除或重建学习数据，也不进行视觉重构。详细契约见 `docs/DATA_CONSISTENCY.md`。

## 关键交互可靠性（已完成）

- [x] 新建领域前后端统一校验必填、长度、空白和期望深度，提交互斥并提供可读 400 错误
- [x] 移动导航支持导航后收起、遮罩和 Escape 关闭、焦点进入/恢复，且不锁定 body 滚动
- [x] 探索选择器稳定显示当前领域名称和标签，领域切换期间清空旧结果并显示加载状态
- [x] 增加空表单、连续点击、抽屉关闭/焦点恢复、选择器多断点显示的浏览器回归

本阶段不改变业务数据模型，不删除或迁移现有学习数据，也不调整无关视觉样式。交互契约见 `docs/INTERACTION_RELIABILITY.md`。

## 核心学习闭环完善（已完成）

- [x] 空回答拦截、按 Lesson 自动保存/恢复草稿、未提交离开提示和具体回答建议
- [x] Evaluation Prompt v4：已理解内容、关键缺口、误区、使用证据、可信度/不确定性、下一步与迁移条件
- [x] 四类用户可读学习状态说明、认知证据时间与状态变化原因
- [x] 回答版本、评价变化、认知证据和掌握度前后快照的历史展示
- [x] 迁移挑战解锁条件、与原问题差异、独立作答记录及明确掌握度影响
- [x] 回答、挑战生成和挑战作答幂等键；AI 失败保持学习数据原子性
- [x] Schema 13 加法迁移、旧数据兼容和迁移前备份/回滚说明
- [x] 核心闭环服务、HTTP 集成和 Playwright 端到端回归

本阶段不重做知识地图或全局视觉系统。详细契约见 `docs/LEARNING_LOOP.md`。

## 可操作知识工具（已完成）

- [x] 知识区域稳定 ID 关联、旧数据无歧义回填、空/重复/错挂防护
- [x] 当前区域默认展开，其余区域概要折叠；统一八类筛选和用户可读知识层级
- [x] 查看节点与切换 CurrentLesson 分离，生成入口显示数量和草案影响
- [x] 探索推荐补齐类型、真实来源、理由和关系；问题池支持保存、撤销、领域/优先级/状态
- [x] 单次 AI 误区只作为推测；用户可确认、纠正、忽略，并查看回答/事件依据
- [x] 增加筛选、稳定关联、保存撤销、空状态和显式节点切换回归测试

本阶段不改变节点类型和来源口径，不自动发布课程或删除学习数据，不进行全局视觉重构。详细契约见 `docs/OPERABLE_LEARNING_TOOLS.md`。

## 视觉一致性、无障碍与系统设置（已完成）

- [x] 统一用户可见中文术语与六类语义状态色，提高辅助文字、空状态和未激活导航对比度
- [x] 首页当前学习重点补充最近学习、理解掌握度、建议时长和下一步目标
- [x] 桌面与移动导航改为真实链接，补齐 landmark、`aria-current`、字段错误关联、异步播报和全局键盘焦点样式
- [x] AI 服务商与模型分开设置，显示密钥配置状态、实际生效模型、最近成功调用并支持安全连接测试
- [x] 备份结果展示文件元数据；恢复要求逐字确认、固定快照、恢复前自动备份和无活动连接重启恢复
- [x] 一致性报告按课程结构、学习证据、历史、探索及来源分类并给出修复建议
- [x] 覆盖键盘导航、语义属性、密钥脱敏、安全恢复和 375/768/1024/1440 响应式回归

本阶段不修改学习状态、课程生成、推荐或掌握度计算逻辑，也不新增数据库迁移。详细说明见 `docs/ACCESSIBILITY_AND_SETTINGS.md`。

## 最终工程加固（自动化基线完成）

- [x] 已展开知识区域渲染、节点详情按需缓存，路径/列表切换不重读图数据
- [x] 探索雷达按领域/Lesson/数量隔离缓存，业务写入后主动失效
- [x] 知识、学习和探索页无效读请求取消，问题池每 20 条分段渲染
- [x] 课程草案超时/解析失败原子性回归，生成占位刷新轮询，领域草案刷新恢复
- [x] 一致性检查覆盖 Course/Unit/Lesson、节点类型、掌握记录与认知证据交叉归属
- [x] 补充缓存、取消、完整建域、生成恢复、长列表、误区审阅、备份/导出/恢复和密钥泄漏回归
- [ ] 统一所有 AI 写入的持久化 request ledger 和进程崩溃后自动续作
- [ ] 真实 Provider、真实备份回滚、屏幕阅读器和长时间运行人工验收

本轮无数据库 schema 变更，不删除、重置或自动修复学习数据。验收边界见 `docs/ENGINEERING_ACCEPTANCE.md`。

## 项目看板交互优化（已完成）

- [x] 任务详情默认阅读，显式进入编辑，保存返回阅读并保护未保存修改
- [x] 跨列拖动乐观更新、失败回滚、同位置放置无请求，保留既有排序与版本校验
- [x] 卡片标题与摘要各两行，收敛元数据与拖放高亮
- [x] Today 聚合全部活动项目，逾期 / 今天到期 / 进行中 / 下一步互斥分组，下一步可展开
- [x] 四列看板局部横向滚动、移动抽屉与固定底部操作，覆盖四种屏宽

复用现有 API 和拖拽依赖，本轮不涉及数据库迁移或其他工作台业务变更。设计见 `docs/PROJECTS.md`。

## 统一 Inbox、项目记录与 Obsidian 链接

- [x] 首页、Inbox、快速捕获共用输入，Enter 保存 / Shift+Enter 换行，失败保留并安全重试
- [x] 复用任务转换事务，默认最近有效项目，防止重复创建及半完成状态
- [x] Inbox 保存为轻量记录，保留原文及来源关联；记录支持阅读、编辑、项目调整、归档、恢复、删除
- [x] 项目折叠记录区与 Inbox 全部/已归档记录入口，保留未关联及已归档项目记录
- [x] 网页 / 受控 Obsidian open URI 白名单、原链接保存及复制，不读取本地 Vault
- [x] Schema 20 可重复迁移、迁移前备份、来源唯一约束和捕获去重；导出包含记录

本轮不新增一级导航、搜索系统或依赖，不修改学习业务。详细设计与回退见 `docs/PROJECTS.md`、`docs/DATABASE.md`。

## LifeOS V0.2-A 财务档案

- [x] 生活下财务子页，概览、分类/标签、只读分页明细与原始详情。
- [x] 钱迹 18 列 CSV 后端预览、BOM、精确整数分、行号错误。
- [x] 来源账本 / 交易 ID 去重、更新、批次事务、陈旧预览保护。
- [x] 明确完整范围和稳定 ID 后才移除缺失数据，顺序撤销恢复前后快照。
- [x] 认证、上传限制、私密日志保护、Schema 22 加法迁移及完整备份。
- [ ] 用户真实 1340 条样本核验（等待本地 CSV 路径）。

设计与计算契约见 [FINANCE_ARCHIVE.md](FINANCE_ARCHIVE.md)。


## LifeOS V0.2-A 财务档案第二轮

- [x] 复核整数分统计、交易 ID 幂等更新、事务回滚和增量导出保留历史；补充月份/分类/标签一致性合成测试。
- [x] 区分全部时间、未结束期间与已结束期间，不由自然月推断记录完整；展示实际覆盖和最新导入。
- [x] 月度共同比例尺、差额与负差额表达；分类展开、统一占比及明细追溯。
- [x] 常用/高级筛选、条件提示、一键清除、URL 恢复和异步旧响应保护。
- [x] 生活/财务轻量面包屑、原生活视图返回和内存滚动恢复。
- [ ] 用户真实账本金额与长历史界面人工验收（未读取私人 CSV）。

本轮不涉及 API、数据库或全局导航变更，计算与界面契约见 [FINANCE_ARCHIVE.md](FINANCE_ARCHIVE.md)。


## LifeOS V0.2-B 资产档案

- [x] 独立资产类别/日期快照，CNY 整数分、NULL 与零区分、自动判断完整范围。
- [x] 新增、只读详情、编辑、确认删除、同日唯一与版本冲突保护。
- [x] 最新历史资产、类别构成、实际日期净资产趋势与可比变化；部分范围独立展示。
- [x] 上次快照手动填表模板，不自动保存，不修改历史。
- [x] 财务第五视图、既有认证、Schema 23 加法迁移、整库备份/恢复与导出。
- [ ] 真实资产数据人工验收（测试仅用合成数据）。

设计与统计规则见 [ASSET_ARCHIVE.md](ASSET_ARCHIVE.md)。V0.2-A 收支数据与计算口径保持独立。

## LifeOS V0.2-C：财务参考

- 长期目标关联已有资产快照、多对多去重、判断日期与可编辑说明，解除关联保留原始记录。
- 资产详情反向打开既有目标详情；被引用快照阻止删除，目标删除清理关联。
- Schema 24、认证与归属校验、备份/导出覆盖；不自动更新阶段记录、目标判断或资金进度。
- 数据契约和交互见 [GOAL_FINANCIAL_REFERENCES.md](GOAL_FINANCIAL_REFERENCES.md)。

## LifeOS V0.3-A：月度生活回顾

- 生活内新增月度阅读页，按真实日期聚合行动、学习活动、经历、阶段变化和财务记录。
- 复用收支/资产统计；新增未来真实完成时间，不把旧更新时间当完成日期。
- Schema 25：月度正式档案及独立事实版本，冻结事实、个人心得、显式重汇总确认和原事实留档。
- 原来源删除保留历史摘要、认证与隐私日志保护、备份恢复和导出兼容。
- 设计、口径与限制见 [MONTHLY_REVIEW.md](MONTHLY_REVIEW.md)。

## LifeOS 我的资料库 V0.1

- [x] 专题、单层分组与根目录页面，通用/Wiki/菜谱可编辑模板。
- [x] Markdown 阅读/编辑与安全过滤，多来源与标签、URL 图片。
- [x] Wiki 目录与卡片清单切换，专题内中文搜索与分页。
- [x] 稳定 ID 内部链接、来源失效提示、事务安全删除与版本校验。
- [x] Schema 26、整库备份与专题 Markdown/JSON ZIP 导出。
- 后续：本地附件、可选 Inbox 整理入口；本轮不实现 AI 或自动抓取。

### 已实现：项目任务手动归档

- 已完成任务详情归档、已完成列批量选择和数量确认。
- 项目列表内按项目查看已归档任务，查看归档时间并取消归档。
- 独立 archived_at 字段，默认看板/今天隐藏，保留完成时间与月度回顾历史统计。
- Schema 27 可重复迁移、原始记录保留和整库备份兼容。

### 已实现：资料库 V0.1.1 本地图片

- 私有 PNG/JPEG/WebP 上传与稳定引用，数据卷持久化、内容/大小/像素校验和存储配额。
- Markdown 点击选择、截图粘贴、拖拽上传，失败重试与正文保留，阅读/预览大图和加载失败提示。
- 专题导出本地图片并改写离线相对路径，缺失图片阻止不完整导出。
- 数据库和图片完整 ZIP 备份/下载/校验恢复，兼容旧数据库备份；显式孤立图片清理。

### 已实现：资料库页面封面统一图片交互

- 封面上传/即时预览、选择正文已有本地图片、清除与外部 URL 兼容。
- 复用稳定图片引用及现有清理、导出、完整备份，无新数据库结构。

### 已实现：资料库卡片图片性能优化

- 卡片使用按需生成的 640px JPEG，原图、稳定引用和数据库保持不变。
- 私有 ETag 条件缓存，登录与归属检查后 304，不再重复读取图片正文。
- 单一版本、并发去重、原子写入、独立缓存配额与历史图片兼容；懒加载及异步解码。
- 原图导出与完整备份保持不变，恢复后可重新生成小图；测量见 [LIBRARY_IMAGE_PERFORMANCE.md](LIBRARY_IMAGE_PERFORMANCE.md)。

### 已修复：资料库 Cards / Wiki 导航

- URL 保存视图模式，Cards 详情独立阅读、显式返回画廊、重复点击 Cards 仍可返回。
- Wiki 分栏保留，切换、刷新、前进后退与编辑保存保持当前专题及筛选上下文。
- 无新 API 或迁移；资料库及图片浏览器回归 20 项通过。

### UI Foundation V1

- 新增 PageShell standard/wide/reading，主要页面外层渐进迁移。
- PageHeader、面包屑、标题/操作基线、主题几何 Token 与默认控件高度统一。
- 保留工作台网格、看板、学习聊天、Wiki/Cards 及 Markdown 专用布局。
- 四种窗口宽度的合成数据巡检与截图；35 项单元测试、147 项浏览器测试及构建通过。设计及范围见 [UI_FOUNDATION.md](UI_FOUNDATION.md)。


### LifeOS 资料库 V0.1.2

- Cards 使用定宽上限、4:3 封面的响应式画廊，省略不可靠正文摘要并提供缺图回退。
- Wiki 保留目录阅读与稳定链接，统一目录层级及资料库专用 Markdown 排版。
- 阅读封面避免与正文重复展示，沿用 PageShell、缩略图、视图查询参数与编辑/导出能力；详见 [LIBRARY.md](LIBRARY.md)。


### LifeOS 工作台首页 V0.2

- 保留原 Focus/Today/项目/学习口径，移除重复首页 Inbox 表单和历史列表。
- 顶部快速记录复用 Inbox，并增加关闭/导航时未保存保护；待整理数量作为轻量入口。
- 新增受认证保护的限量只读最近内容查询，串联资料库、生活事件和真实目标阶段；财务仅返回快照日期。
- 继续使用 PageShell 与现有私有缩略图和原始详情路由，详见 [WORKSPACE_HOME_V02.md](WORKSPACE_HOME_V02.md)。


### LearnOS Design System 1.0

- 按根目录 DESIGN.md 收敛全站主题为中性表面与靛紫强调，默认浅色，保留显式系统/深色选择。
- 单一语义 token 来源兼容既有变量，统一 AppShell、PageHeader、SectionHeader、控件与焦点；移除常驻阴影、高光与旧主题定义。
- PageShell 支持 standard 1200px、wide 可用宽度、reading 760px、lesson 800px；侧栏 224/64px 与手机抽屉，保留看板与固定课程输入行为。
- 资料库移动目录可折叠，阅读层级不修改 Markdown 原文；中文首页标题、学习错误重试与课程卡片键盘操作统一。
- 无数据库/API/业务模型变化；实施与验证见 [DESIGN_IMPLEMENTATION.md](DESIGN_IMPLEMENTATION.md)。


### DESIGN.md Wope 参考适配

- 按更新后的深紫暗色参考迁移共享语义 tokens、默认主题、胶囊按钮、16px 卡片、直角底边输入与 1248px standard 容器。
- 页面沿用 PageShell；工作型页面保留紧凑间距，文章与看板保留专用布局。不复制宣传官网的大字号、超大分区留白或营销内容。
- 保留显式浅色/系统主题设置及所有业务路由、数据与操作。相关验证见 docs/DESIGN_IMPLEMENTATION.md。


### Wope 个人控制台视觉样板（AppShell + 首页）

- 重组首页12列布局：8/4主焦点与Today、7/5项目与学习、三列最近内容与胶囊档案入口；窄屏保留焦点→Today→学习→项目→最近内容顺序。
- 主焦点边缘光晕、首页局部紫色环境光、半透明面板与细亮边框。正文背后使用稳定底色。导航与内容使用独立色层。
- 项目为小卡网格、学习为主题阅读卡、最近内容为来源卡；真实数据与原入口、快速记录、错误重试不变。
- 本轮只建立视觉样板，不进一步迁移其他业务页；使用相同合成数据进行四种宽度前后截图与现有业务回归。详见 docs/WORKSPACE_CONSOLE.md。


### Editorial Home：关注、创造、探索与痕迹

- 仅改首页，保留原焦点选择、真实API、Inbox捕获和来源跳转；“当前旅程”为呈现名称，不新建业务对象。
- 去除焦点/Today/项目/课程/最近内容的围合卡片、大号计数和环境光，改用开放分栏、阅读式学习主题、按真实时间排列的痕迹。
- Light暖纸 #F7F6F2 / Dark深紫灰 #19161F，首页局部语义色、克制强调和无阴影；共享外壳与其他页面不变。
- 两种主题四种宽度截图、文字对比度、跨路由样式隔离与现有交互回归。详见 docs/WORKSPACE_EDITORIAL_HOME.md。


### 首页品牌恢复：Editorial + Panel Hybrid

- 保留当前旅程、Today、正在创造、正在探索、最近痕迹及原数据/入口。
- 恢复紫黑画布、深色surface、柔和描边、局部微光与 #713DFF 主操作；浅色对应暖底与淡紫轻面板。
- 使用整区容器，内部条目仍开放；项目按内容收束，不人为填充等高大卡片。
- 仅首页样式与相关测试文档修改，不改共享组件、全局token、数据库、API或其他业务页。详见 docs/WORKSPACE_LAYERED_HOME.md。


### 首页样板：Wope 局部背光与透明薄膜

- 定位首页实色来源：局部home-surface/home-focus/home-exploration与旅程遮光层；PageShell仅管理宽度，不修改共享组件。
- 深色 #0A0118、普通面板2.5%白色薄膜、细边框与16px圆角；主光源锚定旅程下沿中心，文字区使用接近画布的稳定底色。
- 冷浅主题保持同一光源位置与布局，用独立文字变量与轻透白膜呈现。
- 创造与探索改为全宽连续的目录/继续学习区域，按真实内容自然占高，消除并排造成的左侧空洞。
- 本轮仅首页及相关验证文档，无后端/数据库/API/其他页面改动；见 docs/WORKSPACE_BACKLIT_HOME.md。

### 首页持续环境光与透明玻璃材质

- 首页专属固定环境层：偏右/左下两处弱光，复用侧栏断点，主卡下沿局部光随内容滚动。
- 普通面板与旅程采用半透明背景、单层14px模糊、上下沿明暗及轻量悬停；布局、数据和操作保持不变。
- 使用合成数据验证深浅主题顶部/中部/底部固定光层、点击、无溢出、路由隔离与截图。实施及验证见WORKSPACE_AMBIENT_GLASS.md。

### 首页比例、内容密度与操作位置

- 不改内容宽度、光照与业务；按PageShell实际可用宽度使用容器查询，第一排2:1、第二排1:1并在不足960px时回到单栏。
- 项目改紧凑纵向行和近邻箭头，学习操作移到课程文字下，标题换行、摘要最多两行。
- 痕迹使用整行约67px活动列表，窄屏日期换行，边框/表面更安静；深色辅助文字稍提亮。
- 合成数据覆盖多项目/单项目/空状态、长课程、深浅主题和桌面/手机截图；结果见WORKSPACE_HOME_DENSITY.md。

### 首页最终精修与可复用材质规范

- 第二排桌面自然拉伸等高、单栏自然占高；第一排减少padding/gap与最小高度，长内容不裁切。
- 今天使用三个只读数量分组；逾期语义红沿用现有日期判定，不改变数量与任务入口。
- 项目空/占位图标使用名称首字，图标容器统一32/8px；深色辅助文字统一变量提亮。
- 成熟材质收敛到现有workspace-theme.css可选变量，更新根DESIGN.md；其他业务页保持原样。
- 验证与限制见WORKSPACE_HOME_FINAL_POLISH.md。

### 首页边框专项收尾

- 保留并验证工作区已有边框修复；将五张首页卡片的1px基础边框收敛为统一规则，高光独立于轮廓，主卡光晕位于透明表面后方。
- 默认与hover保持同一边框和圆角，保留透明材质及背景光，不增加实色底。
- 深浅主题2倍截图及四种宽度回归，实施与验证见WORKSPACE_HOME_FINAL_POLISH.md的“首页边框专项收尾”。

### frontend-design 首页边缘与背光复核

- 按DESIGN.md现有约束落实完整轮廓、玻璃后方背光、双主题稳定层次，范围仍限首页边框及相关光效。
- 截图发现窄屏主卡底边被负层光晕覆盖，隔离上下文移至透明外层；普通表面采用padding-box，微调痕迹边框可辨度。
- 同步边缘规范与层叠测试，验证记录见WORKSPACE_HOME_FINAL_POLISH.md。


### 2026-10-16：首页安静背光

- 保留 B 首页布局及真实业务内容，删除夹缝白核、汇聚弧线和独立接缝反光。
- 改为重点卡后方偏下的宽扁低饱和背光，弱化学习卡染色；保留透明膜与基础细边。
- 用深浅主题及桌面/平板/手机实页截图和关闭光效对照验收。


### 2026-10-19：采用较强地平线背光

用户选择 horizon-preview 的 strong 档。正式首页使用同色三层照明：淡紫窄亮带、饱和紫过渡和宽范围环境扩散，亮带直径为重点卡宽度96%，中心始终藏在该卡下沿后方2px。光层锚定 focus-card-frame 实际高度，不依赖固定页面坐标，长任务、加载态及手机布局随卡片变化。重点卡上部遮光、底部少量透光与细反光；不加白核、星芒、弧线或网格。浅色主题沿用结构，单独校准紫光和白色面板透光值。保留原布局、字阶、数据与全部操作。本次仅同步实现，部署单独执行。


### 2026-10-19：浅色玻璃与冷色霓虹

浅色首页以 #EAF0F8 冰蓝画布、青蓝与淡紫环境折射替代旧浅紫底色。普通面板采用rgba(255,255,255,.32)，重点表面从.58渐变至.28，保留14px背板模糊、16px圆角、白色细边与上沿内反射。重点下沿地平线使用青蓝亮带与蓝紫过渡；文字区域用 #172840 / #384F69 / #4D617A 分层，不使用半透明文字。主按钮 #4653DF，悬停 #3542C9，配小范围有偏移的蓝色投影。背景灯色透过面板，霓虹不包围每张卡片；无网格、粒子或新增装饰组件。布局、业务数据及行为保留，仅首页浅色主题改变；深色颜色、较强地平线与主按钮值显式保持原值。


### 2026-10-19：玻璃材质与学习/最近内容互换

根据用户提供的 Snipaste_2026-10-19_11-49-51.png 修正浅色材质：取消重点卡下沿强青色窄亮带，保留低强度透光过渡；普通面板白色透明度.22，重点表面.34→.22，配白色上沿、轻微底沿折射和有偏移的柔和投影，背景加深少许以形成玻璃前后层次。深色较强背光保持原参数。

桌面中栏为重点→继续学习，右栏为最近留下的；平板学习跟随重点、最近内容在下一行，手机仍按重点→今天→学习→项目→最近内容顺序。学习卡在701px以上将几何图示排到摘要/状态旁，标题与行动横跨卡片，减少垂直占用。加载态同步将学习骨架放在重点下方；业务数据、跳转和按钮行为保留。


### 2026-10-19：玻璃反射与导航中轴

浅色首页降低普通面板填充至.14，重点表面.24→.12；背景增加宽斜向亮区及局部青蓝/淡紫色场，配非对称内边缘反射，避免将白边浅色填充当作透明玻璃。正文仍为深墨色，深色主题材质不变。

首页顶部导航以重点卡实际横向中轴为锚：通过 ResizeObserver 观测重点容器/主内容尺寸，将卡片中心相对顶栏的位置写入首页局部变量 --home-focus-axis；加载态与内容态切换重新绑定，离开首页清理监听及变量。宽度不足时继续使用抽屉导航；1599px以下收起账号文字，1399px以下压缩搜索宽度，避免控件重叠。浏览器测试直接测量导航与卡片中心偏差小于1px，同时检查导航不侵入右侧操作区。


### 2026-10-19：移除白线与白膜材质

依据用户最新截图 Snipaste_2026-10-19_13-34-49.png，移除浅色环境中的斜向白色高亮带；取消白色整圈描边与高亮内边，改为rgba(64,99,145,.13)的柔和冷色轮廓。普通表面为rgba(235,246,255,.10)，重点为冷色透明膜.14→.07，保留现有背景模糊与弱局部反射；不通过继续加亮白线制造玻璃感。该方案是透明界面材质表达，不宣称真实光学折射。布局、导航实际中轴、深色主题与业务交互保持不变。


### 2026-10-19：首页色系自定义

首页个人菜单加入「首页外观」，所有登录用户均可打开无背景遮罩的侧边设置，支持深/浅/系统模式、默认及冰川蓝/极光紫/玫瑰/松石绿预设与原生取色器。即时应用，只保存当前浏览器 localStorage `learnos-home-accent`；不写学习数据或后端。色系只影响首页环境色、背光、链接及主按钮，不改透明材质、布局、导航中轴或其他业务页面。恢复默认清除自定义变量并还原深浅主题原设计。

持久化只接受完整六位十六进制颜色，非法存储回退默认；禁用存储时仍即时预览并显示无法保存提示。同源多标签同步storage事件。自定义主按钮按相对亮度选择黑/白前景，单元测试覆盖4096种颜色，保证正常状态文本对比度≥4.5:1。无彩色自定义保留零饱和度，避免灰白颜色错误生成红色背景。浅色/深色各自使用固定亮度、透明度范围映射环境色，保留正文对比度。


### 2026-10-19：自然边缘环境光

用户要求色光不再集中于屏幕中间。取消重点卡下沿灯带、底部独立透光伪元素及其无用tokens，将唯一 HomeLightField 移到 AppShell 的整页固定背景层。主色光从左上视口外漫入，右下为较弱的邻近色反射，底部仅留低强度反照；三层以不同范围和衰减混合，光源中心均在画布外，无白核、弧线、网格或动画。背景光不再跟随重点卡高度，也不随任务密度改变中心位置，不出现内容网格矩形裁切。玻璃透明参数、正文、布局、导航对齐及色系持久化保留。自定义色系同步映射三处环境受光，默认配色和自定义配色共用同一空间规则。


### 2026-10-19：通透玻璃与整卡导航

进一步降低浅色普通玻璃填充至.06、重点表面.10→.05，背板使用blur(18px) saturate(1.2)，保留透明边缘和分布式环境色；不增加白色描边或中心发光。深色材质维持blur(14px)。

移除首页「今日重点」底部主操作按钮、「继续学习」底部行动和重复学习空间按钮；两张有真实目的地的卡片使用覆盖整个表面的原生RouterLink，右上以轻量箭头提示可进入。重点任务仍打开对应项目任务详情，学习仍进入当前课程/知识地图；按Tab聚焦与Enter打开、Ctrl/Cmd新标签、真实href均保留。空重点无跳转，学习空状态保留创建入口，学习出错保留重试且不放链接遮挡。悬停使用主题色轮廓/标题提示，焦点使用清晰外轮廓。色系调节继续影响环境光和交互色，不改变玻璃透明度或布局。


### 2026-10-19：五卡一致悬停与玻璃厚度

首页五张卡片统一使用180ms边缘反光、轻微受光变化和柔和投影，悬停时分区标题采用主题链接色；不位移、不缩放、不添加白色发光框。键盘focus-within使用相同材质反馈，整卡链接保留独立焦点轮廓。项目、今日安排和最近动态仍以内部条目作为操作目标，外层材质反馈不改变点击语义。

材质增加低强度斜向表面反射和背向冷色暗面，通过内边缘、偏移软投影及前后透色建立厚度；浅色悬停不再切换到.55白色底，改为.10冷色透明受光，避免鼠标经过变成实心白块。深色普通面板降至.26，重点表面由.82→.58渐变，接收整页环境光；原布局、色系设置、导航中轴与数据行为保留。


### 2026-10-19：全站玻璃设计统一

- 将首页批准的玻璃、自然边缘环境光、共享悬停推广至所有页面；保留各业务模块布局及交互。
- 统一侧栏、登录、卡片、阅读/编辑、表格、表单和弹层的材质角色，业务语义色保留。
- 色系偏好扩展为全局外观设置，兼容已有浏览器保存值；新增侧栏、手机与设置入口。
- 执行全站浏览器回归与深浅/桌面/手机截图验收，检查通过后自动部署。详见 docs/GLASS_DESIGN_SYSTEM.md。


## 全站顶部导航统一（2026-10-19）

所有已登录页面复用工作台的玻璃顶部导航，不再在进入学习、项目、生活、收集箱或资料库后切换为左侧栏。桌面当前栏目跟随路由高亮，课程详情归属学习，资料库详情归属资料库；探索、系统设置与用户管理保留个人菜单入口。窄屏继续使用顶部菜单按钮和可键盘操作的抽屉，并提供资料库入口。首页导航仍按今日重点中轴对齐，其他页面按内容视口居中。色调、材质和模块功能保持。


## 当前设计规范整理（2026-10-19）

将实际落地的玻璃材质、分布式环境光、全站顶部导航、深浅主题、自定义色系及交互/验收要求统一整理至根目录 design.md。DESIGN.md 保留兼容入口，原 Wope 参考和多轮演进归档，避免旧方案覆盖当前设计。仅文档变更。

## Phase 11.1a：场景分类与工作台切换（2026-10-20）

- 项目新增 work / life / unclassified 场景；历史数据默认未分类，不自动猜测。
- 任务不存重复场景，继续强制属于项目；Workspace 项目、今日任务和自动建议在数据库查询层按项目场景过滤。
- 首页增加全部 / 工作 / 生活分段切换，按当前用户保存浏览器偏好；保留三栏、主题、自定义色及全局学习与最近内容。
- 自动重点更名为继续处理建议；分类场景无任务时不回退到全局学习。未分类入口复用原项目列表与编辑弹窗。
- schema 29 通过原迁移前备份 + AutoMigrate 添加默认字段与用户/场景索引，测试覆盖重复迁移和历史任务事实保留。
- 本轮不部署。没有独立任务、plan_date 或 DailyFocus。
- 11.1d 待办：学习、资料、recent 缺少场景元数据，当前明确为全局内容；本阶段不推断或重构其分类。

### Phase 11.1a 场景控件对齐修正

场景切换移入首页页头右侧 actions，手机换行后仍右对齐；改为共享透明玻璃胶囊与轻受色选中态，去除实色分段主按钮外观。未分类项目为零、未读取或项目分区失败时不显示整理入口。保留原场景过滤、记忆和键盘操作。

### Phase 11.1a 场景切换稳定性修正

保留三栏 DOM，切换期间仅遮住旧场景任务/项目并禁用交互，不再清空首页重建骨架；全局学习和近期动态不闪烁。保留当前视口场景卡片的最小高度、预留首页滚动条空间，宽度变化时重置测量；整理入口采用页头右侧浮动定位，不影响页头高度。保留取消请求与身份校验，失败可重试，旧场景内容不会冒充新场景。

## Phase 11.1b：今日计划与任务安排（已验收，未部署）

- ProjectTask 增加可空 plan_date，schema 30；历史保持 NULL，不从 due_date 回填。
- 任务创建/更新/查询支持计划日期；PATCH 区分省略、null 和日历日期，支持可选 expected_updated_at 条件更新；任务仍必须属于项目。
- 首页新增 planning 聚合：今日主动计划、今天到期、已逾期、此前计划未完成；主列表和摘要按互斥桶去重，重叠截止/计划信息保留在任务行。
- 左栏今日计划复用原任务，支持选择已有任务、移出、改期、完成及 TaskDrawer 详情；提醒默认折叠。旧 Today 接口与返回字段保留。
- 继续处理建议按进行中、今日计划、今天到期、高优先下一步取候选；超过七天的逾期不因截止日期被推荐，明确安排到今天后可重新进入建议。
- 首页/看板任务写入统一刷新，跨标签只广播无个人数据的失效令牌；跨本地午夜或恢复页面时重读日期与数据。保留三场景和防抖布局。
- 不实现独立任务、DailyFocus、计时器、AI 规划、每日复盘或日历模块；本轮不部署运行环境。

### Phase 11.1b 验证结果

`go test ./internal/... ./cmd/...`、`go vet ./internal/... ./cmd/...` 与服务端构建通过；前端 typecheck、41 项单元测试、生产构建通过。相关首页/项目/计划 Playwright 完整回归 97 项通过，手机触控尺寸补充回归 2 项通过。首次回归中的旧定位器/布局断言、跨日测试时钟缓冲及迁移测试夹具 ID 冲突已修正并重跑通过。未执行其他业务模块的全套 E2E、生产数据库迁移或部署。

## Phase 11.1c：每日焦点（Daily Focus，2026-10-21）

- 新增 `daily_focuses` 表（schema 31）：`(user_id, focus_date, context)` 唯一，引用既有 ProjectTask，`revision` 乐观并发；不建外键、不复制任务、不回填历史焦点。
- 首页工作 / 生活场景各自最多一个用户主动选择的焦点；全部视图只展示两者摘要、不拥有独立焦点；允许不设置，生活场景尤其不强制安排。
- 新 API：`GET /api/v1/workspace/daily-focus`、`PUT /api/v1/workspace/daily-focus/:date/:context`；`workspace/home` 增加 `daily_focus` 字段，与 PUT 共用同一数据来源。
- 设置 / 更换 / 清空只改焦点引用，不修改任务 status、plan_date、due_date、排序或所属项目；完成焦点复用任务完成逻辑，保留当天已完成状态；次日不自动继承。
- 焦点状态：`ok`、`completed`、`missing`、`context_changed`、`project_inactive`、`archived`；失效引用按当前用户与场景重新校验，不越权、不悬空。
- 首页中间主卡从「继续处理建议」升级为「今日焦点」，原自动建议降级为次级行且绝不自动写入焦点；选择器默认今日已计划未完成任务，可搜索其他可操作任务，已完成 / 归档 / 未分类不可选。
- 设计、API 与状态机见 [DAILY_FOCUS.md](DAILY_FOCUS.md)；本轮不部署运行环境。

### Phase 11.1c 验证结果

- 后端：`go vet ./internal/... ./cmd/...` 通过；`go test -count=1 ./internal/... ./cmd/...` 无断言失败、无构建失败（107 个测试仅因 Windows TempDir 清理时 sqlite 文件句柄占用报伪失败，属既有环境限制，与基线同类）。新增 DailyFocus service/handler/迁移测试全部通过：每人每天每场景唯一、work/life 独立、未分类拒绝、设置/更换不改任务字段、完成保留且重复完成不覆盖 completed_at、次日不继承、项目暂停/归档/跨场景、任务归档/删除→missing、同任务幂等、旧 revision 409、清空幂等、跨用户隔离、schema 30→31 重复迁移幂等且既有任务数据不变。
- 前端：`npm run typecheck` 0 错误；`npm run test` 20 文件 / 50 用例通过（新增 dailyFocus 9 项）；`npm run build` 通过并更新内嵌前端。
- 浏览器：新增 `daily-focus` 18 项全部通过；`daily-focus`、`workspace`、`planning`、`regression`、`ui-foundation` 合计 139 项中 138 项通过。唯一失败为 `regression.spec.ts` 的移动端导航抽屉 overlay 点击，在改动前 HEAD（v0.1.1-12-g789bf91）独立 worktree 中同样失败，属既有环境问题，不是本轮回归；跨标签同步慢用例已加 60s 超时。
- 未执行真实数据库迁移与线上部署；E2E 全部为 API 路由拦截，权限与事务由 Go 测试覆盖。

## Phase 11.1d：工作台体验整合与优化（2026-10-21）

- 首页按「计划 → 行动 → 沉淀」重组三栏：左栏今日计划（主要操作入口）与默认折叠的待处理提醒、正在推进的项目；中栏今日焦点与「继续上次」恢复入口；右栏快速记录、最近活动和全局继续学习。
- 全部场景的今日计划仍是互斥列表，但摘要分别给出工作/生活已安排数量，任务行保留场景标签；焦点任务在计划行标记「今日焦点」，避免同一任务在多处重复同权重展示。
- 原自动建议改为「继续上次」，仅作恢复入口，不写入焦点；建议与当天焦点为同一任务时隐藏；学习位置移到右栏全局入口，中栏不再承载学习卡片。
- 标题随场景变化：全部「今天，按自己的节奏」、工作「今天，专注重要的事」、生活「留一点时间给自己」；日期统一保留，语气不随任务数量变化。
- 密度与层级：焦点卡空态降低最小高度；最近条目与全局学习收紧字阶与间距；无数据的最近/学习不渲染空卡片；桌面三栏、701–1099px 两列、700px 以下单列（焦点→计划→项目→快速记录→最近→学习）。
- 复用 QuickCaptureDialog 增加右栏入口；不改后端、API、数据库与任务写入逻辑；保留场景切换竞态、跨日刷新、跨标签同步和几何最小高度。
- 设计与验收语义见 [design.md](../design.md) 的「Phase 11.1d 首页体验整合」。

### Phase 11.1d 验证结果

- 前置检查：`daily-focus`、`workspace`、`planning`、`ui-foundation` 及 11.1b 计划相关 E2E 全部通过；Go 后端测试与前端类型/单测通过，确认 11.1a～c 能力仍在。
- Go：`go vet ./internal/... ./cmd/...` 通过；全量 `go test` 无断言失败（仅 Windows TempDir 清理伪失败，由 `scripts/verify-local.ps1` 识别为环境告警）。本轮未改后端、API 或数据库。
- 前端：`npm run typecheck` 0 错误；`npm run test` 20 文件 / 50 用例通过；`npm run build` 通过并更新内嵌前端。
- 浏览器：`workspace`、`planning`、`regression`、`daily-focus`、`ui-foundation` 合计 **144 项，143 通过**；唯一失败为 `regression.spec.ts` 的移动端导航抽屉 overlay 点击，已在改动前 HEAD 复现，属既有环境问题。新增 11.1d 用例覆盖场景标题、全部场景计划数量、继续上次去重、右栏快速记录入口与全局内容位置；所有原有 Prismatic/密度/边框/几何用例已按新信息结构更新并通过。
- 性能：真实开发接口下首页聚合 19–34ms、最近内容 4–6ms；每次加载 1 次 home + 1 次 recent，场景切换只重读 home，无重复查询。
- 视觉：1440/1024/390、深浅主题截图核对无横向溢出；右栏不再出现被拉伸的巨大空卡片，中栏空白不由装饰填充。未部署、未提交，E2E 全部为 API 路由拦截。
- 复用流程：新增 `web/e2e/visual-audit.spec.ts`（mock 截图，无需后端）、`scripts/verify-local.ps1 -Visual`、`docs/LOCAL_DEV_NOTES.md`（本机环境与已知坑）和 `docs/DESIGN_REVIEW_CHECKLIST.md`；Vite 支持 `VITE_API_TARGET` 并忽略原子写入临时目录。

## 资料库工具入口：万缕千丝 · 挖角手帐（2026-10-26）

- 资料库首页新增「工具」分区，分区内提供独立静态手帐页面 `/library-tools/fefw-planner/planner.html`；手帐与 LifeOS 同源运行，规划数据保存在浏览器 localStorage。
- 手帐沿用自身的浅色手帐界面，不写入资料库专题 / 页面模型，也不进入专题 ZIP 或整库 Markdown 导出；从手帐可返回 `/explore/library`。
- 仅新增 `web/public/library-tools/fefw-planner/` 静态资源、资料库入口卡片与样式；无数据库、API、认证与 schema 变更。

### 验证结果（2026-10-26）

- 前端 `npm run typecheck` 0 错误；`npm run test` 26 文件 / 78 用例通过；`npm run build` 通过并更新内嵌前端。
- 新增 `web/e2e/library.spec.ts` 用例「资料库首页提供挖角手帐工具入口」通过。
- 用独立开发实例（8084 + 临时数据目录）验证 `/explore/library` 工具入口、`/library-tools/fefw-planner/planner.html?from=library` 页面和「返回资料库」按钮；桌面与移动端截图通过。
- 已部署当前 8080 运行实例（2026-10-26 16:43）：容器 healthy，`/library-tools/fefw-planner/planner.html` 及样式、脚本、头像均返回 200，前端 bundle 含「万缕千丝 · 挖角手帐」。本次因 `web/public` 为未跟踪目录，使用完整工作区同步后执行 `deploy/deploy.sh`。
