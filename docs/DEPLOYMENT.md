# 本地与云服务器部署

本文说明 LearnOS 的两种部署路径、可复用脚本、验证方式与回滚步骤。

## 1. 运行环境对照

| 环境 | Docker 位置 | 使用脚本 |
| --- | --- | --- |
| 本机 Windows | **WSL2 Ubuntu-24.04** 内的 Docker（Windows 侧没有 Docker） | `scripts\deploy-local.ps1`（内部调用 `deploy/deploy.sh`） |
| 云服务器 Linux | 宿主机 Docker + compose 插件 | `deploy/deploy.sh`（POSIX，原生执行） |

> **Windows 是否要装 Docker Desktop？** 不需要，也不会影响云端。镜像和 `docker-compose.yml` 都是 Linux 容器定义；Docker Desktop 只不过是在 Windows 上再提供一套 Linux 容器运行时（虚拟机），与云端 `docker compose` 的行为一致。保持当前 WSL Docker 即可；脚本本身也是 Linux 通用脚本，云端可直接复用。

## 2. 前置条件

- Docker + `docker compose` 插件；
- 仓库根目录存在 `.env`（参考 `.env.example`，至少配置 `APP_ENV`、`APP_USERNAME`、`APP_PASSWORD_HASH`；Windows 本地沿用部署目录已有的 `.env`）；
- 数据与备份使用宿主机目录挂载：`DATA_HOST_DIR`、`BACKUP_HOST_DIR`（默认 `./data/prod`、`./backups`）；
- 推荐安装 `curl` 与 `sqlite3`，用于部署后验证（缺少时脚本会跳过对应检查）。

## 3. 一键部署

### 3.1 Windows（经 WSL2）

```powershell
. .\scripts\dev-env.ps1            # 配置 Go/zig/npm/Chrome 等本机环境
.\scripts\deploy-local.ps1 -SyncChangedFiles   # 未提交改动：先同步到 WSL 部署目录再构建
# 或（推荐，已提交并推送后）
.\scripts\deploy-local.ps1 -Pull
```

- 默认 WSL 发行版 `Ubuntu-24.04`、部署目录 `/home/hp/projects/learnos-starter`，可用 `-Distro` / `-DeployPath` 覆盖。
- `-SyncChangedFiles` 只复制 `git status` 列出的文件（跳过 `.env`、`data/`、`backups/`、`node_modules`），并自动处理 Windows→Linux 的 CRLF 与 fileMode 差异。
- 部署结束会输出访问地址（默认 `http://127.0.0.1:8080`）。

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

## 4. 部署脚本做了什么

`deploy/deploy.sh`：

1. 校验 Docker、compose 插件与 `.env`；
2. 可选 `--pull`（要求工作区干净）；
3. `docker compose up -d --build app`，等待容器 health 通过（最长 90 秒，超时会打印日志并失败）；
4. 验证首页资源版本、bundle 是否包含「今日焦点」标记、未登录接口返回 401；
5. 宿主机有 `sqlite3` 时输出 `schema_version / app_version / daily_focuses` 行数，并列出最近备份；
6. 输出 `docker compose ps` 与访问地址。

部署成功时退出码为 0，任何一步失败返回非 0，可直接接 CI。

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

1. 停止应用：`docker compose down`（数据卷为宿主机目录，不会被删除）；
2. 恢复迁移前备份：部署目录 `backups/` 下最新 ZIP（完整备份，迁移前自动生成），取其数据库覆盖 `data/prod/learnos.db`；
3. 代码回退：`git checkout <上一个可用提交>`，再执行 `bash deploy/deploy.sh --no-verify` 重建镜像；
4. 通过 `/health` 与首页登录确认。

## 7. 常见问题

| 现象 | 原因 / 处理 |
| --- | --- |
| `docker: command not found`（Windows） | 必须通过 WSL 执行；使用 `scripts\deploy-local.ps1` |
| WSL 内构建后 git 显示大量文件被修改 | Windows 侧同步带入 CRLF/fileMode；脚本已 `sed` 去 CRLF 并设置 `core.fileMode false` |
| 端口占用 / 容器起不来 | 检查 `.env` 的 `APP_BIND_HOST`、`APP_PORT`；`docker compose ps` 与 `docker compose logs app` |
| 首次构建慢 | 容器内 `npm ci` 与 Go 模块下载；后续有构建缓存 |
| 数据库迁移 | 启动时自动执行；迁移前会生成 `backups/learnos-<时间戳>.zip`，请勿删除 |
| 部署目录有未提交改动，`--pull` 被拒绝 | 部署目录是可丢弃的部署副本：先 `git checkout -- . && git clean -fd`（`.env`/`data/`/`backups` 是 ignore 或已被排除，不受影响），再执行 `--pull` |
| 云上 HTTPS/Caddy 没有更新 | 加 `--with-caddy`，或单独执行 `docker compose --profile production up -d --build` |
| 在 DeepSeek Harness 沙箱内跑 Go/npm | 需要 `pwsh` + `danger-full-access`（普通沙箱禁止工具链子进程管道与 node_modules 写入） |

## 8. 文件清单

| 文件 | 作用 |
| --- | --- |
| `scripts/dev-env.ps1` | Windows 本机环境（Go 1.27、zig cgo、缓存、npm、Chrome） |
| `scripts/verify-local.ps1` | 本地验证（Go/前端/可选 E2E） |
| `scripts/deploy-local.ps1` | Windows → WSL 部署入口（可选同步未提交文件） |
| `deploy/deploy.sh` | Linux/云端通用部署脚本（本地 WSL 也复用它） |
