# 本地开发与视觉验证说明

流程入口：[AGENTS.md](../AGENTS.md#日常-ai-前端视觉流程默认执行)；风格约束：[design.md](../design.md)；检查清单：[DESIGN_REVIEW_CHECKLIST.md](DESIGN_REVIEW_CHECKLIST.md)。本文件只记录环境与命令，不另建视觉规范。

## 环境与作用域

| 项 | 本机事实 / 使用方式 |
| --- | --- |
| 正常运行环境 | WSL2 `Ubuntu-24.04`，部署副本 `/home/hp/projects/learnos-starter`；当前本机入口 `http://127.0.0.1:8080`，应以 `/health` 与实际资源为准 |
| Docker | 在 WSL 内执行；不要另起一套实例替代正常服务 |
| 前端 | Vue 3 / Element Plus / Vite，沿用当前 lockfile，不额外装框架 |
| Playwright 默认 | 自动启动 `127.0.0.1:4173` 的临时 Vite（可复用同地址已运行服务），不是已部署版本；测试完成后由 Playwright 回收其启动的服务 |
| 浏览器 | 配置优先读取 `PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH`，Windows 有标准路径系统 Chrome 时自动复用，否则使用 Playwright 已安装的 Chromium；不静默下载浏览器 |
| Go | Windows PATH 曾为 Go 1.19，无法编译当前 go.mod；Go 任务先查版本，使用符合要求的 SDK / WSL / 现有 Docker。纯前端验证不需要 Go |
| 执行权限 | 按当前会话实际工具/权限执行，优先可用的 bash 执行器；不要照搬历史“必须 PowerShell/自动提权”的结论，不为权限错误换壳绕过限制 |
| 嵌入资源 | Vite 默认构建会写 `internal/webassets/dist`；检查使用 `build:check`，保护已有嵌入产物与未提交修改 |

## 视觉工作流的日常命令

从仓库根目录开始，下面的 npm / Playwright 命令在 Windows、WSL、Linux 均复用现有依赖。

```bash
cd web
npm run typecheck
npm run test
npm run build:check
npm run test:ui
```

- `build:check` 输出到 `.tmp/dist-check/`，不修改 Go 嵌入资源；真正打包发布仍用已有 Docker 构建流程。
- `test:ui` 只组合现有 `visual-audit` 与 `home-material`，无需后端/数据库。前者默认生成首页首屏与整页截图，后者检查六卡、控件状态、路由隔离及静态降级。**它不是全站/所有浏览器/真实数据验收。**
- 首页截图固定合成数据、日期 `2026-10-21T08:00:00Z`、`Asia/Shanghai`、`zh-CN`、DPR=1，使用 Reduced Motion 与 CSS Ambient。定时器仍工作，不通过冻结整个页面掩盖加载或交互错误。
- 输出在 `web/test-results/visual-audit-*/visual-*.png`，同名 JSON 记录 URL、主题、视口、数据来源、固定时间、浏览器与 CSS/JS 地址，并附入 Playwright 结果。必须实际打开图片检查，不把存在文件等同验收通过。
- 覆盖桌面深浅/三场景、1024 平板、390 手机、空态与完成态；其他状态按目标任务选择对应 E2E。

### 保留基线与修改后证据

```bash
npm run test:ui -- --output=../.tmp/ui-before
# 在允许范围内修改，保持相同浏览器/字体/fixture；然后：
npm run test:ui -- --output=../.tmp/ui-after
```

Playwright 会清理自己的输出目录，所以两次必须不同目录；多个独立进程也不能共用一个目录。现有测试保存图片供 Agent/人工比较，没有把新截图自动认作 golden 的机制；不要无脑更新像素基线。需要像素断言时仅在同一浏览器、字体、DPR、数据和静态渲染条件下对受影响区域补最小断言。

### 非首页与共享改动

复用已有页面 spec，不用首页截图替代目标页验证。例如：

```bash
# 局部页：按任务挑选，不必每次全部跑
npm run test:e2e -- e2e/projects.spec.ts e2e/library.spec.ts
# 首页业务行为或布局：
npm run test:e2e -- e2e/workspace.spec.ts e2e/daily-focus.spec.ts e2e/planning.spec.ts
# 共享组件/主题改动：
npm run test:e2e -- e2e/ui-foundation.spec.ts e2e/regression.spec.ts
# Ambient / refraction / 动效相关改动：
npm run test:e2e -- e2e/home-ambient.spec.ts e2e/glass-refraction.spec.ts
```

`ui-foundation` 已检查首页、学习、项目、收集箱、生活、资料库/阅读等核心路由的宽度、基线与溢出；`regression` 已有双主题/自定义色系及多断点巡检与截图。局部页面还需自行以现有 fixture 采集目标页 light/dark × 桌面/手机的前后图；缺覆盖才在原 spec 中补最小 case，不另建测试平台。

### 验证实际运行的前端（不启动临时服务器）

Bash：

```bash
PLAYWRIGHT_BASE_URL=http://127.0.0.1:8080 npm run test:ui -- --output=../.tmp/ui-served
```

PowerShell：

```powershell
$env:PLAYWRIGHT_BASE_URL='http://127.0.0.1:8080'
npm run test:ui -- --output=../.tmp/ui-served
Remove-Item Env:PLAYWRIGHT_BASE_URL
```

设置 `PLAYWRIGHT_BASE_URL` 后，配置不启动/重启 webServer，只访问指定地址；临时 Vite 与 Go 嵌入版本必须明确区分。上述两个视觉 spec 仍拦截 API 使用 fixture，所以验证的是**实际服务前端在合成数据下的渲染**，不证明真实登录、个人数据、后端接口或写入通过。不要将真实凭据写入测试或截图，不对公网生产环境执行有副作用的测试。

若正常服务资源版本未变化，不能宣称部署更新成功；源码 UI 任务需按既有部署脚本更新并核对 HTML 引用、CSS/JS 与 `/health`。纯文档/测试工具初始化不需要重启正常服务。

## 现有整体验证与开发入口

- [verify-local.ps1](../scripts/verify-local.ps1)：Go / 前端 / `-E2E` / `-Visual` 集成入口，前端检查构建同样隔离输出；PowerShell 用户可配合 [dev-env.ps1](../scripts/dev-env.ps1) 使用。纯前端优先用上面的 npm 命令，避免依赖 Go/cgo 初始化。
- `-AllowEnvFailures` 是显式容忍告警，不等于测试已通过；E2E 启动异常、没有跑到断言也必须报告失败/未验证。Windows SQLite 清理问题要依据实际错误分类，不把所有 Go 失败概括为“伪失败”。
- 手动开发 `npm run dev` 默认 `127.0.0.1:5173`，API 目标由 `VITE_API_TARGET` 指定，默认 8080；若需独立开发 Go API，使用空闲端口（如 8090）及独立数据目录，不能占用正常运行端口或数据库。
- 浏览器缺失时，可设置 `PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH` 指向已有 Chrome。没有可用浏览器才按环境许可安装 Playwright Chromium；无法安装则报告未验证，不声称视觉通过。

## 常见失败的定位顺序

1. **页面没变化**：查当前 URL、实际 CSS/JS 地址、computed style / 命中规则与叠层，再查构建或部署；不直接归咎缓存。
2. **截图不稳定**：先查数据、日期、字体、DPR、主题、动态背景与 hover 状态是否一致；不要对漂移 WebGL 帧承诺逐像素等同。
3. **ENOENT / trace 丢失**：查独立 Playwright 进程是否共用并清理同一输出目录。
4. **旧布局断言失败**：对照实际已验收页面与当前规范，证明过期再更新，不能移动页面或降低正确断言凑通过。
5. **权限/启动/网络失败**：保留真实退出码和错误，修复明确原因后重试；无法取得浏览器证据就停止视觉微调并报告。

完整流程的自动修正预算是初次修改后的 **最多两轮**，不是无限循环。构建、测试、截图生成、实际看图与用户认可应分别汇报。
