# 本地开发与视觉评审笔记

记录这台工作机与 LifeOS 开发/评审相关的环境事实和固定流程，避免每次重新试错。

## 1. 环境事实

| 项 | 值 / 结论 |
| --- | --- |
| Windows Docker | **没有**。`docker` / `docker compose` 只在 WSL2 内可用 |
| WSL 发行版 | `Ubuntu-24.04`；既有部署目录 `/home/hp/projects/learnos-starter`，容器 `learnos-starter-app-1` |
| Windows 上的 8080 | 被 WSL2 localhost 转发占用（指向 WSL 内的部署实例）。**本地开发请用 8090**，Vite 用 5174 |
| Go | `D:\Project\Business\go1.27.0\bin\go.exe`（PATH 里的 go 是 1.19，不能编 go.mod） |
| cgo 编译器 | 无 gcc；用 `ziglang` 的 `zig.exe cc`（`python -m pip install ziglang` 安装） |
| Node / npm | Node 24 / npm 11；`web/node_modules` 已安装 |
| Playwright 浏览器 | 未下载，统一用系统 Chrome：`C:\Program Files\Google\Chrome\Application\chrome.exe` |
| DSH 沙箱 | 普通 bash/ops_bash 无法运行 Go/npm 子进程；必须 `pwsh` + `sandbox_permissions=danger-full-access` |
| Go 测试伪失败 | Windows 上部分测试因 `TempDir RemoveAll cleanup`（SQLite 文件占用）失败，断言本身是通的 |

`. .\scripts\dev-env.ps1` 已封装 Go/zig/GOCACHE/GOPATH/npm cache/Chrome 路径，优先使用。

## 2. 常用命令

### 2.1 一次完整验证（推荐）

```powershell
. .\scripts\dev-env.ps1
.\scripts\verify-local.ps1            # go vet/test + typecheck + vitest + build
.\scripts\verify-local.ps1 -E2E       # 追加 5 个目标 E2E spec
.\scripts\verify-local.ps1 -Visual    # 追加视觉评审截图（见下）
```

### 2.2 前端单独开发

```powershell
. .\scripts\dev-env.ps1
cd web
npm run dev                 # http://127.0.0.1:5173，代理到 8080（WSL 部署）
# 若本地 Go API 跑在 8090：
$env:VITE_API_TARGET='http://127.0.0.1:8090'; npm run dev -- --port 5174
```

### 2.3 真实数据视觉评审（可选）

需要真实 API 时（大多数首页评审用 2.4 的 mock 版本即可）：

```powershell
. .\scripts\dev-env.ps1
New-Item -ItemType Directory -Force data\dev | Out-Null
& $env:LEARNOS_GO build -buildvcs=false -trimpath -o data\dev\learnos.exe .\cmd\server

$env:APP_ENV='development'; $env:APP_ADDR='127.0.0.1:8090'; $env:APP_DATA_DIR='./data/dev'
$env:DEMO_SEED_ENABLED='true'; $env:AI_PROVIDER='mock'
Start-Process data\dev\learnos.exe -WorkingDirectory (Get-Location) -WindowStyle Hidden `
  -RedirectStandardOutput data\dev\server.log -RedirectStandardError data\dev\server.err.log

cd web
$env:VITE_API_TARGET='http://127.0.0.1:8090'; npm run dev -- --port 5174
```

清理：结束 Vite 与 `learnos.exe`，删除 `data/dev`。

### 2.4 视觉评审截图（mock，无需后端）

```powershell
cd web
$env:CAPTURE_VISUALS='1'
npx playwright test e2e/visual-audit.spec.ts
```

截图输出：`web/test-results/visual-audit-*/visual-*.png`。覆盖：

- 桌面浅色：全部 / 工作 / 生活
- 桌面深色：全部 / 生活
- 平板 1024、手机 390（浅色 / 深色）
- 空工作区、已完成焦点

不设置 `CAPTURE_VISUALS` 时同一 spec 作为结构冒烟测试运行（约 5 秒）。

## 3. 已知坑与处理

| 现象 | 原因 / 处理 |
| --- | --- |
| `listen tcp 127.0.0.1:8080: bind ...` | Windows 8080 已被 WSL 转发占用；开发服务改用 8090 |
| Vite 崩溃 `EBUSY ... .tmpdir` | 编辑器/代理的原子写入产生临时目录；`vite.config.ts` 已忽略 `**/.*.tmpdir/**` 等。若仍出现，检查是否把临时文件写进 `web/` |
| `spawn EPERM` / `pipe: Access is denied` | DSH 沙箱限制；用 `pwsh` + danger-full-access 运行 Go/npm/Playwright |
| Playwright 报没有浏览器 | 设置 `PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH` 指向系统 Chrome（dev-env 已设置） |
| Go 测试 107 个 FAIL | 全部是 TempDir 清理伪失败；`verify-local.ps1` 会区分真实断言失败 |
| 首页 E2E 大面积失败 | 先确认布局断言是否随信息架构更新；用 `--grep` 跑子集，最后再全量 |
| `regression.spec.ts:418` 移动端抽屉失败 | 既有环境问题（改动前 HEAD 同样失败），不是回归 |

## 4. E2E 分层执行

```powershell
cd web
$env:PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH='C:\Program Files\Google\Chrome\Application\chrome.exe'
# 只跑受影响的
npx playwright test e2e/workspace.spec.ts --grep "Prismatic|场景加载保留布局"
# 收尾跑完整目标集
npx playwright test e2e/daily-focus.spec.ts e2e/workspace.spec.ts e2e/planning.spec.ts e2e/regression.spec.ts e2e/ui-foundation.spec.ts e2e/visual-audit.spec.ts
```

## 5. 复用资产

| 文件 | 用途 |
| --- | --- |
| `scripts/dev-env.ps1` | 本机环境变量 |
| `scripts/verify-local.ps1` | Go/前端/可选 E2E/可选视觉评审 |
| `web/e2e/visual-audit.spec.ts` | 首页视觉评审截图（mock） |
| `docs/DESIGN_REVIEW_CHECKLIST.md` | 设计/交互/可访问性检查清单 |
| `scripts/deploy-local.ps1` + `deploy/deploy.sh` | 本地/云端部署 |
