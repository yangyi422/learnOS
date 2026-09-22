# LearnOS 运维手册

## 数据目录

SQLite 是事实来源。生产容器使用 `/app/data/learnos.db`，备份使用 `/app/backups/`。宿主机建议将两者分开挂载：

```env
APP_ENV=production
DATA_HOST_DIR=./data/prod
BACKUP_HOST_DIR=./backups
```

开发环境可使用 `DATA_HOST_DIR=./data`，不要把开发库目录和生产目录共用。应用启动时会执行可重复的 GORM AutoMigrate；检测到旧 schema 时先创建一次 SQLite 备份，再迁移并写入 `system_metadata.schema_version`。

本地开发模式固定使用回环地址：Go API 为 `127.0.0.1:8080`，Vite 开发服务器为 `127.0.0.1:5173`，Vite Preview 为 `127.0.0.1:4173`。`APP_ENV=development` 会跳过登录认证，且拒绝绑定 `0.0.0.0` 或其他非 `127.0.0.1` 地址；production 及其他环境使用数据库会话认证。

生产 Docker 部署使用端口映射 `宿主机 8888 → 容器 8080`：直接通过服务器 IP 访问时将 `APP_BIND_HOST` 配为 `0.0.0.0`，仅经 Caddy 访问时可配为 `127.0.0.1`；容器内 Docker healthcheck 和 Caddy 到应用的连接使用 `app:8080`。不要把容器内健康检查端口改为宿主机端口。

## 初始化与启动

```bash
go run ./cmd/learnos init
docker compose up -d --build app
curl --fail http://127.0.0.1:8888/health
```

`init` 只建立 schema 和 System Seed，不创建 Course、Unit、Lesson 或个人学习数据。生产 / Dogfooding 首次启动必须保持空知识世界；开发环境通过 `DEMO_SEED_ENABLED=true` 保留 Demo World。正常启动不会删除或重置既有课程、Lesson、学习记录、认知状态、探索记录或来源数据。

课程删除是用户主动操作，不会随启动或 Seed 自动发生。课程档案和学习首页的“删除课程”会二次确认，确认后调用删除 API，并在一个事务中删除该课程专属结构、学习/认知/探索数据和 Grounding 连接；页面会明确提示删除成功或失败，成功后刷新课程列表。跨课程复用的知识来源与来源证据保留。删除前建议先创建 SQLite 备份。

## 备份、导出与恢复

应用设置页可以创建备份、下载 JSON/Markdown 导出并运行一致性检查。API 为：

```bash
curl -c cookies.txt -H 'Content-Type: application/json' -d '{"username":"admin","password":"change-me"}' http://127.0.0.1:8888/api/v1/auth/login
curl -b cookies.txt -X POST http://127.0.0.1:8888/api/v1/system/backups
curl -b cookies.txt http://127.0.0.1:8888/api/v1/system/backups
curl -b cookies.txt -X POST 'http://127.0.0.1:8888/api/v1/system/export?format=json' -o learnos-export.json
curl -b cookies.txt -X POST 'http://127.0.0.1:8888/api/v1/system/export?format=markdown' -o learnos-export.md
```

设置页还可以选择服务端备份并恢复。该操作要求输入 `恢复 <备份文件名>`，确认后固定所选快照、优雅停止应用，并依赖 Docker `restart: unless-stopped` 或等价进程管理器重新启动。启动阶段会先备份当前数据库，再恢复固定快照；页面和接口不会接受任意文件路径。

恢复前停止应用，确认备份文件来自可信目录：

```bash
docker compose stop app
go run ./cmd/learnos restore --backup ./backups/learnos-YYYYMMDD-HHMMSS.db
docker compose up -d app
curl --fail http://127.0.0.1:8888/health
```

恢复命令会先验证源文件的 SQLite `integrity_check`，为当前数据库创建 pre-restore 备份，复制到临时文件并再次校验后原子替换。恢复不会改变任何 Lesson 的 ID；恢复完成后应人工检查课程列表、当前 Lesson、学习历史和认知状态。

备份保留数量由 `BACKUP_RETENTION_COUNT` 控制，默认 20。备份目录必须有持久化卷；不要只依赖容器可写层。

## 诊断与一致性

设置页或以下接口可查看：

```bash
curl -b cookies.txt http://127.0.0.1:8888/api/v1/system/diagnostics
curl -b cookies.txt http://127.0.0.1:8888/api/v1/system/consistency
```

`/health` 只执行轻量数据库连接检查，适合 Docker healthcheck；完整 SQLite `integrity_check` 在 diagnostics 中执行，避免每个探活请求扫描数据库。一致性检查是只读的，会检查 Course/Unit/Lesson 归属、关系端点与 DAG、节点类型、掌握分数/归属、认知证据与回答范围、误区/挑战、探索和来源关联；不会修复、删除或重写学习数据。

## AI 故障处理

AI 超时、空响应、结构校验失败、限流、网络错误和未配置分别返回统一错误对象：

```json
{"error":{"code":"AI_TIMEOUT","message":"AI 响应超时，本次数据未保存。","retryable":true}}
```

页面会显示可读说明和重试入口。AI 失败不会创建 LearningTurn、CognitiveEvidence 或更新 CognitiveState；网络错误优先检查容器出网、API 地址、超时配置和服务商状态。不要把 API Key 写入仓库、导出文件或日志。

## Docker 维护

应用使用 `restart: unless-stopped`，健康检查为 `/health`，Caddy 生产 profile 依赖应用健康后启动。查看日志：

```bash
docker compose logs -f --tail=200 app
docker compose ps
```

收到 SIGINT/SIGTERM 时，服务停止接收新请求，最多等待 10 秒完成已有请求，然后关闭 SQLite 连接。升级前建议手工创建一次备份，并保留最近一次可验证恢复的备份。

## GitHub Actions 自动部署

仓库的 `.github/workflows/deploy.yml` 在 `main` 推送或手动触发时运行。它会先执行 Go 测试、`go vet`、前端类型检查、Vitest 和生产构建；全部通过后，使用 SSH 在生产服务器执行：

```bash
git fetch origin main
git merge --ff-only origin/main
docker compose --profile production up -d --build
```

在 GitHub 仓库的 `Settings → Environments → production → Environment secrets` 中配置：

- `DEPLOY_HOST`：服务器域名或 IP
- `DEPLOY_PORT`：SSH 端口，可选，默认 `22`
- `DEPLOY_USER`：SSH 登录用户
- `DEPLOY_SSH_KEY`：对应用户的私钥，完整粘贴多行内容
- `DEPLOY_PATH`：服务器上的 LearnOS 仓库绝对路径，例如 `/opt/learnos`

服务器需要预先完成 Docker、Docker Compose、Git 和仓库访问权限配置；`.env`、SQLite 数据目录和备份目录应留在服务器上，不放入 Git。部署使用 `git merge --ff-only`，服务器存在未提交改动时会安全失败，不会覆盖这些改动。
