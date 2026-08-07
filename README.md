# LearnOS Starter

LearnOS 是一个单用户、自托管的个人 AI 学习系统。本仓库是第一阶段可运行骨架。

## 当前能力

- Go + Gin 后端
- SQLite + GORM 持久化
- Vue 3 + TypeScript + Element Plus 前端
- 自动创建一门带知识结构和当前知识点的营养学示例课程
- `/api/v1/courses` 课程列表接口
- `/api/v1/courses/:id/knowledge-graph` 静态知识结构接口
- `/api/v1/courses/:id/lessons/:lessonId/relations` 知识点关系接口
- `/api/v1/courses/:id/cognitive-states` 课程认知状态接口
- `/api/v1/courses/:id/lessons/:lessonId/cognitive-state` Lesson 认知详情接口
- 支持 Mock / DeepSeek 的结构化学习评价、学习记录和掌握度持久化
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

访问：`http://localhost:5173`

开发环境中如设置了 Basic Auth，浏览器会弹出账号密码框。

## Docker 运行

```bash
cp .env.example .env
# 部署前修改用户名和密码哈希
docker compose up -d --build app
```

服务仅绑定到宿主机回环地址：`http://127.0.0.1:8080`。

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
curl http://127.0.0.1:8080/healthz
curl -u admin:change-me http://127.0.0.1:8080/api/v1/courses
curl -u admin:change-me http://127.0.0.1:8080/api/v1/courses/1/current-lesson
```

## Phase 3 AI 配置

默认使用本地 MockProvider，适合测试和无 API Key 的本地调试：

```env
AI_PROVIDER=mock
AI_TIMEOUT_SECONDS=45
```

使用真实 DeepSeek 评价时，在后端环境配置：

```env
AI_PROVIDER=deepseek
DEEPSEEK_API_KEY=your-deepseek-api-key
DEEPSEEK_BASE_URL=https://api.deepseek.com
DEEPSEEK_MODEL=deepseek-v4-flash
AI_TIMEOUT_SECONDS=45
```

API Key 只在后端使用，不会返回给浏览器，也不会写入评价错误日志。选择 `AI_PROVIDER=deepseek` 但未配置 API Key 时，应用启动会失败并提示配置错误。

## Learning API

- `GET /api/v1/courses/:id/current-lesson`：读取课程当前模块和知识点。
- `POST /api/v1/courses/:id/answers`：提交当前知识点回答并保存掌握度。
- `GET /api/v1/courses/:id/learning-turns?limit=10`：读取最近回答记录，`limit` 最大为 50。
- `GET /api/v1/courses/:id/knowledge-graph`：读取课程的 Lesson 节点、静态关系和图统计。
- `GET /api/v1/courses/:id/lessons/:lessonId/relations`：读取指定 Lesson 的前置、后续、扩展、应用和相关知识。
- `POST /api/v1/courses/:id/lessons/:lessonId/challenges`：在达到 `understand` 后生成 transfer challenge，或为 active misconception 生成 recheck。
- `POST /api/v1/courses/:id/challenges/:challengeId/answers`：提交一次独立挑战；成功迁移才会产生 transfer 证据。
- `GET /api/v1/courses/:id/misconception-network`：读取误区、固定 reasoning pattern 和事件网络。
- `GET /api/v1/courses/:id/lessons/:lessonId/misconceptions`：读取当前 Lesson 的误区生命周期和验证历史。

回答成功后会返回判断结果、反馈、解释、正确点、缺失点、误区、边界条件、掌握证据、掌握度、认知层级、认知证据和当前认知状态。历史读取不会重新请求 DeepSeek。

认知状态使用 `unseen`、`exposed`、`recognize`、`understand`、`apply`、`transfer` 六级，以及 `unknown`、`developing`、`stable`、`needs_review` 四种状态。它与 Lesson 的静态结构、课程状态和 MasteryRecord 分离；没有真实认知证据的历史 Lesson 不会被自动回填为高级认知等级。

知识结构页：`/courses/:id/map`。它只展示人工定义的静态课程结构，并叠加已验证的认知层级；不计算解锁条件、不推荐下一课，也不调用 AI 修改知识图。学习页在达到 `understand` 后提供迁移挑战和 active misconception 的针对性重新测试；误区网络页为 `/courses/:id/misconceptions`。

## 下一步

后续阶段再实现复习调度和路线推荐；Phase 6 不实现探索雷达、问题池、Agent 导航或知识来源可信度。

```text
读取当前知识点
→ 展示问题
→ 提交回答
→ 使用已保存的结构化评价
→ 保存结构化评价
→ 更新掌握度与误区
```
