# 本地与云服务器部署

本文说明 LearnOS 的两种部署路径、可复用脚本、验证方式与回滚步骤。

## 1. 运行环境对照

| 环境 | 运行方式 | 使用脚本 |
| --- | --- | --- |
| 本机 Windows | **原生 `bin\learnos.exe` + SQLite 文件**（不安装 WSL2，不使用 Docker） | `scripts\run-local.ps1` |
| 云服务器 Linux | 宿主机 Docker + compose 插件 | `deploy/deploy.sh`（POSIX，原生执行） |

> **本机为什么不再走 WSL2 + Docker？** 本机已决定不安装 WSL2，而 Docker Desktop 的两种后端（WSL2、Hyper-V）都要求它或 Hyper-V，因此本机改为原生二进制：一个 `bin\learnos.exe` + 一个 `data\<env>\learnos.db`，前端是编译进二进制的嵌入资源。`Dockerfile` 与 `docker-compose.yml` 保留给云端 Linux，云端行为不变。

## 2. 前置条件

### 2.1 本机 Windows（原生）

- Go 1.26+：`scripts\dev-env.ps1` 会优先使用 `D:\Project\Business\go1.27.0\bin\go.exe`，可用 `LEARNOS_GO` 覆盖；
- cgo 需要的 C 编译器：gcc，或 zig（`python -m pip install ziglang`，脚本自动识别）；
- Node ≥ 22 与既有的 `web/node_modules`（仅在需要重建前端时用到）；
- 可选 `.env`（与容器共用同一份，参考 `.env.example`）：`APP_USERNAME`、`APP_PASSWORD_HASH`、`APP_PORT`、`AI_*` 等；`APP_ADDR`、`APP_DATA_DIR`、`BACKUP_DIR`、`DATA_HOST_DIR` 这类容器路径由脚本接管，不会被带进原生运行。

### 2.2 云服务器 Linux（Docker）

- Docker + `docker compose` 插件；
- 仓库根目录存在 `.env`（参考 `.env.example`，至少配置 `APP_ENV`、`APP_USERNAME`、`APP_PASSWORD_HASH`）；
- 数据与备份使用宿主机目录挂载：`DATA_HOST_DIR`、`BACKUP_HOST_DIR`（默认 `./data/prod`、`./backups`）；
- 推荐安装 `curl` 与 `sqlite3`，用于部署后验证（缺少时脚本会跳过对应检查）。

## 3. 一键部署

### 3.1 本机 Windows（原生运行）

```powershell
. .\scripts\dev-env.ps1                      # 首次或换终端：Go / zig cgo / npm / Chrome
.\scripts\run-local.ps1                      # 缺失时先构建前端，再编译并前台启动
.\scripts\run-local.ps1 -Detached            # 后台启动，自检 health / 首页资源 / 未登录 401
.\scripts\run-local.ps1 -Environment production -Port 8080 -RebuildWeb
```

- 产物固定为 `bin\learnos.exe`（已忽略），数据落在 `data\dev` 或 `data\prod`，备份落在 `backups\`；
- 端口优先级：`-Port` > `.env` 的 `APP_PORT` > 8080；默认只绑定 `127.0.0.1`，需要局域网访问时加 `-BindHost 0.0.0.0`；
- `-Detached` 把输出写到 `.tmp\run-local.log`、pid 写到 `.tmp\run-local.pid`，60 秒内轮询 `/health`，然后打印首页引用的 `assets/index-*.js` 与生产环境未登录接口的 401。它启动的子进程随父 shell 一起结束，需要常驻请在自己的终端窗口里前台运行，或交给计划任务；
- 生产运行必须先在 `.env` 配置 `APP_USERNAME` 与 `APP_PASSWORD_HASH`，否则所有接口都会认证失败（脚本会给出告警）；
- 执行策略拒绝未签名脚本（报“未对文件进行数字签名”）时，改用 `scripts\run-local.cmd`，参数与 `run-local.ps1` 完全一致，例如 `scripts\run-local.cmd -NoBuild -Port 8080 -DataDir .data\prod`；它只对子进程加 `-ExecutionPolicy Bypass`，不修改本机策略。

### 3.2 云服务器（Linux）

```bash
cd /path/to/learnOS
bash deploy/deploy.sh --pull        # git pull --ff-only + 构建 + 部署 + 验证
# 或仅用当前代码构建：
bash deploy/deploy.sh
# 云上通过 Caddy 提供 HTTPS 时，同时拉起 production profile：
bash deploy/deploy.sh --pull --with-caddy
```

也可以在 CI/脚本中直接使用：

```bash
bash deploy/deploy.sh --no-verify && curl -fsS http://127.0.0.1:8888/health
```

## 4. 脚本做了什么

### 4.1 `scripts\run-local.ps1`（本机原生）

1. 读取 `.env`（没有就跳过），只导出应用需要的键，并丢弃容器专用键；
2. 解析环境（`APP_ENV` / `-Environment`）、端口（`APP_PORT` / `-Port`）与数据目录（`data\dev`、`data\prod` 或 `-DataDir`）；
3. 前端产物缺失或加 `-RebuildWeb` 时执行 `npm run build`，写入 `internal/webassets/dist`；
4. 用 cgo（gcc 或 zig cc）编译 `./cmd/server` 到 `bin\learnos.exe`；
5. 前台运行，或按 `-Detached` 后台运行并完成 health、首页资源、未登录 401 的自检。

### 4.2 `deploy/deploy.sh`（云端/容器）：

1. 校验 Docker、compose 插件与 `.env`；
2. 可选 `--pull`（要求工作区干净）；
3. `docker compose up -d --build app`，等待容器 health 通过（最长 90 秒，超时会打印日志并失败）；
4. 验证首页资源版本、bundle 是否包含「今日焦点」标记、未登录接口返回 401；
5. 宿主机有 `sqlite3` 时输出 `schema_version / app_version / daily_focuses` 行数，并列出最近备份；
6. 输出 `docker compose ps` 与访问地址。

部署成功时退出码为 0，任何一步失败返回非 0，可直接接 CI。

## 4.3 GitHub Actions 生产部署（push main 自动触发）

`.github/workflows/deploy.yml`：

1. `verify`：在 GitHub runner 上跑 Go 测试/vet、前端 typecheck/单测/构建；
2. `deploy`：
   - runner 以 `fetch-depth: 0` 检出完整历史，向服务器询问当前 HEAD；
   - 用 `git bundle` 打包**服务器缺失的提交**，通过 SSH 上传到服务器 `/tmp`；
   - 服务器从 bundle 做 `--ff-only` 更新，不再依赖服务器主动访问 GitHub（此前出现过服务器到 GitHub 的 SSH 被断开导致 `Could not read from remote repository` 的失败）；
   - bundle 拉取失败时才回退到 `git fetch origin`；
   - 构建失败自动重试一次；重启后等待容器 `healthy` 才判定成功。

因此生产部署只需要 runner → 服务器的 SSH 连通；服务器只需要能拉取 Docker 基础镜像（首次）与 Go/npm 依赖（构建缓存命中时更快）。

## 5. 本地验证脚本

```powershell
. .\scripts\dev-env.ps1
.\scripts\verify-local.ps1              # go vet + go test + typecheck + vitest + vite build
.\scripts\verify-local.ps1 -E2E         # 追加 Playwright 目标回归
.\scripts\verify-local.ps1 -GoPackages ./internal/model/ -SkipWeb   # 快速冒烟
```

- Go 测试在 Windows 上会出现 `TempDir RemoveAll cleanup` 失败（SQLite 文件句柄未释放），脚本将其识别为环境告警而非失败；其他断言失败会让脚本返回 1。
- `-E2E` 会跑 `daily-focus / workspace / planning / regression / ui-foundation` 五个 spec；已知既有环境失败（移动端抽屉 overlay 点击）可用 `-AllowEnvFailures` 标记为告警。

## 6. 回滚

1. 停止应用：本机 `Stop-Process -Id (Get-Content .tmp\run-local.pid)`（前台运行时直接 Ctrl+C），云端 `docker compose down`；数据是普通目录，停止不会删除数据；
2. 恢复迁移前备份：本机 `backups\`、云端部署目录 `backups/` 下最新 ZIP（完整备份，迁移前自动生成），取其数据库覆盖 `data\prod\learnos.db`；
3. 代码回退：`git checkout <上一个可用提交>`，本机再执行 `.\scripts\run-local.ps1`，云端执行 `bash deploy/deploy.sh --no-verify` 重建镜像；
4. 通过 `/health` 与首页登录确认。

## 7. 常见问题

| 现象 | 原因 / 处理 |
| --- | --- |
| 本机出现 `docker: command not found` | 预期：本机不安装 WSL2/Docker，使用 `scripts\run-local.ps1`；Docker 只在云端使用 |
| `run-local.ps1` 无法加载，提示“未对文件进行数字签名” | 该窗口的执行策略拒绝未签名脚本；用 `scripts\run-local.cmd`（参数相同），或在该窗口先执行 `Set-ExecutionPolicy -Scope Process Bypass -Force` |
| 在 DeepSeek Harness 会话里启动时报 `mkdir …\data\prod: Access is denied` | 会话沙箱不放行仓库内 `data\` 的写入；临时验证加 `-DataDir .tmp\data-probe`，真实运行请在自己的终端里执行 |
| 进程被回收、服务随即停止 | `-Detached` 的子进程随父 shell 结束；需要常驻请在自己的终端前台运行或注册计划任务 |
| 端口占用 / 容器起不来 | 检查 `.env` 的 `APP_BIND_HOST`、`APP_PORT`；`docker compose ps` 与 `docker compose logs app` |
| 首次构建慢 | 容器内 `npm ci` 与 Go 模块下载；后续有构建缓存 |
| 数据库迁移 | 启动时自动执行；迁移前会生成 `backups/learnos-<时间戳>.zip`，请勿删除 |
| 构建在 `apt-get update` 处失败（Broken pipe / 255） | 运行镜像已不再执行 apt（CA 证书从 Go 构建阶段复制，健康检查为 Go 二进制），该失败类型已消除；若仍遇到 Go 模块/npm 下载失败，CI 会自动重试一次构建 |
| 部署目录有未提交改动，`--pull` 被拒绝 | 部署目录是可丢弃的部署副本：先 `git checkout -- . && git clean -fd`（`.env`/`data/`/`backups` 是 ignore 或已被排除，不受影响），再执行 `--pull` |
| 云上 HTTPS/Caddy 没有更新 | 加 `--with-caddy`，或单独执行 `docker compose --profile production up -d --build` |
| 在 DeepSeek Harness 沙箱内跑 Go/npm | 需要 `pwsh` + `danger-full-access`（普通沙箱禁止工具链子进程管道与 node_modules 写入） |

## 8. 文件清单

| 文件 | 作用 |
| --- | --- |
| `scripts/dev-env.ps1` | Windows 本机环境（Go 1.27、zig cgo、缓存、npm、Chrome） |
| `scripts/run-local.ps1` | 本机原生入口：构建并运行 `bin\learnos.exe`（自检 health/首页资源/401） |
| `scripts/verify-local.ps1` | 本地验证（Go/前端/可选 E2E） |
| `scripts/deploy-local.ps1` | 备用：Windows → WSL 部署入口（本机不再有 WSL 时不可用） |
| `deploy/deploy.sh` | 云端 Linux 部署脚本 |
