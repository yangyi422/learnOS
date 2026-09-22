# LearnOS Starter

LearnOS 是一个单用户、自托管的个人 AI 学习系统。本仓库是第一阶段可运行骨架。

## 当前能力

- Go + Gin 后端
- SQLite + GORM 持久化
- Vue 3 + TypeScript + Element Plus 前端
- 开发环境可自动创建营养学、逻辑与科学思维、心理学三门测试课程；生产 / Dogfooding 首次启动保持空知识世界
- `/api/v1/courses` 课程列表接口
- `/api/v1/courses/:id/knowledge-graph` 静态知识结构接口
- `/api/v1/courses/:id/lessons/:lessonId/relations` 知识点关系接口
- `/api/v1/courses/:id/cognitive-states` 课程认知状态接口
- `/api/v1/courses/:id/lessons/:lessonId/cognitive-state` Lesson 认知详情接口
- 支持 Mock / DeepSeek 的结构化学习评价、学习记录和掌握度持久化
- Phase 8 Curriculum Blueprint、课程覆盖度、AI/规则扩充草案与事务化人工 Apply
- Phase 9 Source Registry、Source Evidence、Credibility Assessment、Grounding Review 与来源覆盖度
- Phase 10 运行诊断、SQLite 备份/恢复、JSON/Markdown 导出、数据库一致性检查与优雅停机
- Progressive Empty World Bootstrap：从空世界分三阶段建立领域骨架、起步蓝图和首批 Lesson
- 认知层级、掌握证据和认知演化时间线持久化
- 单用户 Basic Auth
- Docker Compose 与可选 Caddy HTTPS

## 环境要求

- 推荐 Go 1.26+
- 推荐 Node.js 22.18+
- Docker 部署不要求宿主机安装 Go 或 Node

Go 1.26 是当前稳定主版本；Gin 当前文档要求 Go 1.25 或以上。Vue 官方脚手架使用 Vite，当前文档推荐 Node.js 22.18 或更高。

## 本地开发

```bash
cp .env.example .env

# 后端
make dev-api

# 另一个终端启动前端
cd web
npm install
npm run dev
```

访问：`http://127.0.0.1:5173`

开发环境的 Vite 与 Go 服务分别只监听 `127.0.0.1:5173` 和 `127.0.0.1:8080`，并跳过 Basic Auth，便于本机预览与自动化检查。配置为 `APP_ENV=development` 时，`APP_ADDR` 若不是 `127.0.0.1` 会拒绝启动。生产及其他环境仍启用已配置的 Basic Auth。

## Docker 运行

```bash
cp .env.example .env
# 部署前修改用户名和密码哈希
docker compose up -d --build app
```

生产 / Dogfooding 建议显式使用独立数据目录，并保持 `APP_ENV=production`、`DEMO_SEED_ENABLED=false`。首次打开后从首页的“创建第一个学习领域”进入三阶段初始化；开发环境默认保留 Demo World。

初始化页面也可直接打开：`http://127.0.0.1:8888/domains/new`

生产服务默认绑定到宿主机回环地址：`http://127.0.0.1:8888`；容器内部仍监听 `8080`。

### 生成新密码哈希

本地 Go 环境：

```bash
make hash-password PASSWORD='your-strong-password'
```

也可以使用 Apache `htpasswd`：

```bash
htpasswd -bnBC 12 '' 'your-strong-password' | tr -d ':\n'
```

将结果填入 `.env` 的 `APP_PASSWORD_HASH`。

## 使用域名和 HTTPS

填写 `.env`：

```env
APP_DOMAIN=learn.example.com
```

启动生产 profile：

```bash
docker compose --profile production up -d --build
```

Caddy 将反向代理到应用并自动处理域名证书。

## 验证

```bash
curl http://127.0.0.1:8888/health
curl -u admin:change-me http://127.0.0.1:8888/api/v1/courses
curl -u admin:change-me http://127.0.0.1:8888/api/v1/courses/1/current-lesson
```

## Phase 3 AI 配置

默认使用本地 MockProvider，适合测试和无 API Key 的本地调试：

```env
AI_PROVIDER=mock
AI_TIMEOUT_SECONDS=45
AI_CHALLENGE_GENERATION_TIMEOUT_SECONDS=60
AI_DOMAIN_SKELETON_TIMEOUT_SECONDS=120
AI_DOMAIN_STARTER_BLUEPRINT_TIMEOUT_SECONDS=60
AI_DOMAIN_INITIAL_WORLD_TIMEOUT_SECONDS=180
```

使用真实 DeepSeek 评价时，在后端环境配置：

```env
AI_PROVIDER=deepseek
DEEPSEEK_API_KEY=your-deepseek-api-key
DEEPSEEK_BASE_URL=https://api.deepseek.com
DEEPSEEK_MODEL=deepseek-v4-flash
AI_TIMEOUT_SECONDS=45
AI_CHALLENGE_GENERATION_TIMEOUT_SECONDS=60
```

API Key 只在后端使用，不会返回给浏览器，也不会写入评价错误日志。选择 `AI_PROVIDER=deepseek` 但未配置 API Key 时，应用启动会失败并提示配置错误。

也可以登录应用后进入“系统设置 → AI 配置”填写 DeepSeek API Key。保存后配置立即生效，不需要重启；页面只显示 Key 是否已配置。保存的配置优先于环境变量，未保存系统配置时才使用环境变量。

## Learning API

- `GET /api/v1/courses/:id/current-lesson`：读取课程当前模块和知识点。
- `DELETE /api/v1/courses/:id`：在确认后删除单个课程及其课程专属学习数据；共享知识来源会保留。
- `GET /api/v1/courses/:id/lessons/:lessonId/learning`：读取探索支线目标 Lesson，不改变课程主线位置。
- `POST /api/v1/courses/:id/answers`：提交当前知识点回答并保存掌握度。
- `POST /api/v1/courses/:id/lessons/:lessonId/answers`：提交支线 Lesson 回答；读取和回答本身不推进 Course CurrentLesson。
- `GET /api/v1/courses/:id/learning-turns?limit=10`：读取最近回答记录，`limit` 最大为 50。
- `GET /api/v1/courses/:id/knowledge-graph`：读取课程的 Lesson 节点、静态关系和图统计。
- `GET /api/v1/courses/:id/lessons/:lessonId/relations`：读取指定 Lesson 的前置、后续、扩展、应用和相关知识。
- `POST /api/v1/courses/:id/lessons/:lessonId/challenges`：在达到 `understand` 后生成 transfer challenge，或为 active misconception 生成 recheck。
- `POST /api/v1/courses/:id/challenges/:challengeId/answers`：提交一次独立挑战；成功迁移才会产生 transfer 证据。
- `GET /api/v1/courses/:id/misconception-network`：读取误区、固定 reasoning pattern 和事件网络。
- `GET /api/v1/courses/:id/lessons/:lessonId/misconceptions`：读取当前 Lesson 的误区生命周期和验证历史。
- `GET /api/v1/exploration/radar?course_id=:id`：读取探索方向；可选 `lesson_id` 指定源 Lesson。
- `POST /api/v1/exploration/unfamiliar`：读取一个陌生知识候选。
- `POST /api/v1/exploration/directions/:id/save|dismiss|open`：保存、略过或打开方向；open 只更新方向状态并导航到目标 Lesson。
- `POST /api/v1/exploration/directions/:id/questions`、`GET /api/v1/exploration/questions`：生成和读取问题池。
- `POST /api/v1/exploration/questions/:id/start|archive`、`GET /api/v1/exploration/history`：启动/归档问题和读取方向历史。
- `GET|POST /api/v1/sources`、`GET|PATCH /api/v1/sources/:sourceId`：登记和维护跨课程复用的知识来源。
- `POST|GET /api/v1/sources/:sourceId/evidence`、`PATCH /api/v1/evidence/:evidenceId`：人工录入和维护来源证据片段。
- `POST|GET /api/v1/grounding/links`：把证据连接到 Blueprint、Blueprint Lesson 或 Lesson；review/reject 后重算 Grounding 状态。
- `GET|POST /api/v1/sources/:sourceId/credibility`、`POST /api/v1/sources/:sourceId/credibility/:assessmentId/review`：记录并审核来源可信度，不将其当作事实真值。
- `GET /api/v1/grounding/targets/:targetType/:targetId`、`GET /api/v1/courses/:id/grounding/coverage`：读取目标详情和 Grounding Coverage。
- `POST|GET /api/v1/system/backups`：创建或列出 SQLite 备份；应用会按保留数量清理旧备份。
- `POST /api/v1/system/backups/restore`：使用文件名与逐字确认请求安全恢复；服务会固定所选快照、优雅停机，并在重启阶段先备份当前数据再恢复。
- `POST /api/v1/system/export?format=json|markdown`：导出不含密钥的结构化学习数据。
- `GET /api/v1/system/diagnostics`、`GET /api/v1/system/consistency`：查看运行诊断和只读数据一致性报告。
- `GET|PATCH /api/v1/system/ai-config`：读取 AI 配置状态或更新 Provider、API Key、Base URL 和 Model；响应不会返回 API Key。
- `POST /api/v1/system/ai-config/test`：测试当前已生效 AI 连接，仅返回服务商、模型、检查时间和耗时。

回答成功后会返回判断结果、反馈、解释、正确点、缺失点、误区、边界条件、掌握证据、掌握度、认知层级、认知证据和当前认知状态。历史读取不会重新请求 DeepSeek。

认知状态使用 `unseen`、`exposed`、`recognize`、`understand`、`apply`、`transfer` 六级，以及 `unknown`、`developing`、`stable`、`needs_review` 四种状态。它与 Lesson 的静态结构、课程状态和 MasteryRecord 分离；没有真实认知证据的历史 Lesson 不会被自动回填为高级认知等级。

知识结构页：`/courses/:id/map`。它只展示人工定义的静态课程结构，并叠加已验证的认知层级；不计算解锁条件、不推荐下一课，也不调用 AI 修改知识图。学习页在达到 `understand` 后提供迁移挑战和 active misconception 的针对性重新测试；误区网络页为 `/courses/:id/misconceptions`。

## 下一步

当前已完成 `Phase 7 Exploration Engine`：营养学的 8 个静态 Lesson 已补全，并增加逻辑与科学思维、心理学两个最小知识岛。探索方向使用规则候选和可选 AI 文案，AI 失败会回退到规则文案；没有 `CrossCourseLessonRelation` 时使用有限静态 taxonomy bridge 产生可解释的跨域候选。探索打开不会改变课程主线，也不会产生学习或认知证据。

Phase 9 已建立来源、证据、可信度和 Grounding 审核链路。它与 Curriculum Coverage、Personal Cognitive State 分离，不搜索或抓取外部内容，不修改 Lesson 内容、CurrentLesson、LearningTurn 或认知状态。

Phase 10 的运维流程见 [docs/OPERATIONS.md](docs/OPERATIONS.md)。开发数据默认放在 `data/dev`，生产部署建议显式使用独立的 `DATA_HOST_DIR=./data/prod`；备份放在独立的 `backups` 目录。AI 失败统一返回可读的 `code/message/retryable`，不会把失败请求写入学习状态。

```text
读取当前知识点
→ 展示问题
→ 提交回答
→ 使用已保存的结构化评价
→ 保存结构化评价
→ 更新掌握度与误区
```
