# LearnOS Phase 10：Dogfooding Readiness & Stabilization

请在当前 LearnOS 仓库中实现 **Phase 10：Dogfooding Readiness & Stabilization（自用准备 / 稳定化 / 数据安全 / v0.1 收口）**。

这是 LearnOS v0.1 正式投入长期自用前的最后一个功能阶段。

当前已经完成：

- Phase 1：项目骨架
- Phase 2：持久化学习闭环
- Phase 3：AI 结构化认知评价
- Phase 4：Knowledge World
- Phase 5：Personal Cognitive State
- Phase 6：Transfer / Misconception
- Phase 7：Exploration Engine
- Phase 8：Curriculum Blueprint / Coverage / Draft / Apply
- Phase 9：Source / Credibility / Grounding 基础结构

Phase 9 的人工 Source/Credibility workflow 当前保留为可选能力，不再要求作为日常高频工作流继续扩展。

---

## 一、Phase 10 核心目标

本阶段不增加新的学习能力，只解决：

1. 数据安全
2. AI 可靠性
3. 错误与加载体验
4. 数据一致性
5. 导出与恢复
6. v0.1 产品收口

完成后目标：

```text
可以初始化一份干净生产数据
可以部署到服务器
可以连续使用数周
AI 偶发失败也不会破坏学习数据
升级版本前后可以备份、迁移、回滚
```

---

## 二、严格不做

不要新增：

- 新 Agent
- 新推荐算法
- RAG
- 自动搜索
- 自动找文献
- Spaced Repetition
- Gamification
- Notification
- 长期兴趣模型
- 知识压缩
- 新 Cognitive Level
- 新 Challenge 类型
- 新 Exploration 类型
- 多用户
- 权限系统重构
- Vector DB
- 自动内容抓取

这一步是收口，不是继续扩张。

---

## 三、开始前必须阅读

请先阅读：

- AGENTS.md
- README.md
- docs/PRODUCT.md
- docs/ARCHITECTURE.md
- docs/DATABASE.md
- docs/ROADMAP.md
- docs/prompts/phase7.md
- docs/prompts/phase8.md
- docs/prompts/phase9.md

重点检查：

### 数据
- SQLite 文件位置
- AutoMigrate
- Seed
- LearningTurn
- CognitiveState / CognitiveEvidence
- Misconception
- Challenge
- Exploration
- Curriculum Blueprint / Draft
- Source / Grounding

### AI
- Evaluation timeout
- Challenge timeout
- Curriculum Draft timeout
- Source Credibility timeout
- retry
- empty response
- AI_INVALID_RESPONSE
- AI_TIMEOUT

### UI
- 首页
- LearningView
- Knowledge Map
- Exploration
- Curriculum
- Source
- 系统设置

### Deployment
- Dockerfile
- docker-compose.yml
- volumes
- environment
- healthcheck
- logs

开始编码前先汇报：

1. 当前 SQLite 数据路径
2. 当前 Docker volume / bind mount
3. 当前 AI timeout / retry 配置
4. 哪些接口会直接显示 Failed to fetch
5. 当前是否已有 backup / export / restore
6. 当前 Seed 是否适合全新生产库
7. 当前是否已有 healthcheck
8. 预计修改文件
9. 简短实施顺序

先不要立即编码。

---

## 四、SQLite 备份

新增应用级 BackupService。

建议目录：

```text
data/
├─ learnos.db
└─ backups/
```

至少支持：

```text
manual backup
pre-migration backup
```

可选：

```text
startup safety backup
```

文件命名：

```text
learnos-YYYYMMDD-HHMMSS.db
```

---

## 五、备份 API

至少：

```http
POST /api/v1/system/backups
GET  /api/v1/system/backups
```

Phase 10 不要求网页直接 restore。

---

## 六、备份实现要求

SQLite 运行中备份必须使用安全方式。

禁止简单复制正在写入中的 DB。

优先：

- SQLite backup API
- VACUUM INTO
- 当前驱动支持的安全备份方案

如果技术栈不方便，可做短暂写锁后安全复制，但必须说明。

备份失败不能破坏现有 DB。

---

## 七、备份保留策略

简单实现：

```text
BACKUP_RETENTION_COUNT=20
```

保留最近 N 个备份。

---

## 八、Migration Safety

当前如果使用 AutoMigrate：

```text
启动
↓
检测需要 migration
↓
已有数据库时先创建 pre-migration backup
↓
执行 migration
```

Migration 失败：

```text
应用启动失败
保留原 DB
日志打印 backup 路径
```

禁止半迁移状态继续运行。

---

## 九、Schema Version

新增轻量 SchemaVersion 或等价机制：

```text
schema_version
last_migrated_at
app_version
```

目标：知道生产库当前是什么 schema 版本。

---

## 十、生产数据库初始化

新增明确 production init 流程。

形式可按现有架构选：

```bash
./learnos init
```

或等价命令。

目标：

```text
创建全新 SQLite
→ migration
→ Seed 静态 Knowledge World
→ Seed Blueprint
→ 不创建任何个人学习历史
```

---

## 十一、Production Seed 边界

生产初始化保留：

- Course
- CourseUnit
- Lesson
- LessonRelation
- CrossCourseRelation（如有）
- CurriculumBlueprint
- BlueprintUnit
- BlueprintLesson
- BlueprintRelation
- 静态配置

不要 Seed：

- LearningTurn
- CognitiveState
- CognitiveEvidence
- CognitiveStateEvent
- Misconception
- MisconceptionEvent
- AssessmentChallenge
- ChallengeAttempt
- ExplorationDirection
- ExplorationQuestion
- CurriculumDraft 测试数据
- AI Evaluation Run 测试数据
- Demo KnowledgeSource
- Demo Evidence
- Demo Credibility
- Demo GroundingLink

---

## 十二、开发库 / 生产库分离

增加明确环境：

```text
APP_ENV=development
APP_ENV=production
```

建议：

```text
data/dev/learnos.db
data/prod/learnos.db
```

原则：

> 生产库不能误用开发测试库。

---

## 十三、个人数据导出

增加最小导出：

```text
JSON
Markdown
```

---

## 十四、JSON Export

新增等价 API：

```http
POST /api/v1/system/export?format=json
```

至少包含：

```text
export_version
exported_at

courses
lessons
learning_turns
cognitive_states
cognitive_evidence
cognitive_state_events
misconceptions
misconception_events
challenge_attempts
exploration_questions
curriculum coverage summary
sources / grounding（如有）
```

不要导出：

- API Key
- Authorization
- 密码 hash
- secrets

---

## 十五、Markdown Export

目标：

> 即使 LearnOS 不再运行，也能读懂自己的学习记录。

建议：

```text
export/
├─ README.md
├─ courses/
│  └─ nutrition/
│     ├─ overview.md
│     ├─ lessons/
│     ├─ misconceptions.md
│     ├─ cognitive-history.md
│     └─ questions.md
```

不需要非常漂亮，但必须可读。

---

## 十六、Restore

Phase 10 只要求 SQLite backup restore。

提供安全 CLI 或管理命令：

```bash
learnos restore <backup-file>
```

Restore 必须：

1. 停止写入
2. 备份当前 DB
3. 校验 backup 可打开
4. 替换 DB
5. integrity check

不要求 JSON / Markdown restore。

---

## 十七、SQLite Integrity Check

新增 DatabaseHealthService，支持：

```text
PRAGMA integrity_check
```

用于：

```text
manual diagnostics
restore
```

不要每请求执行。

---

## 十八、System Diagnostics

新增：

```http
GET /api/v1/system/diagnostics
```

至少返回：

```text
app_version
schema_version
database_status
database_path
database_size
last_backup_at

ai_provider
ai_model
ai_timeout_config summary

course_count
lesson_count
learning_turn_count
cognitive_state_count
```

不能返回 secrets。

---

## 十九、AI Reliability

不重写 AI 架构，只统一错误模型。

至少定义：

```text
timeout
empty_response
invalid_response
provider_error
rate_limited
network_error
```

可以实现统一 AIError。

---

## 二十、统一 AI 错误响应

API 不要再只让前端看到：

```text
Failed to fetch
```

统一返回例如：

```json
{
  "error": {
    "code": "AI_TIMEOUT",
    "message": "AI 响应超时，本次数据未保存。",
    "retryable": true
  }
}
```

其他可包括：

```text
AI_EMPTY_RESPONSE
AI_INVALID_RESPONSE
AI_PROVIDER_ERROR
AI_RATE_LIMITED
AI_NETWORK_ERROR
```

---

## 二十一、前端 AI Error UX

统一组件：

```text
AIRequestError
```

例如：

```text
AI 响应超时
本次操作没有修改你的学习数据。

[重试]
```

不要继续显示裸 `Failed to fetch`。

至少覆盖：

- Answer Evaluation
- Challenge Generation
- Challenge Answer
- Curriculum Draft
- Source Credibility

---

## 二十二、Retry UX

Provider layer 保持最多 retry once（按当前语义）。

前端失败后允许手动重试。

手动重试必须幂等，不能重复创建：

- Draft
- Challenge
- Question

---

## 二十三、Loading UX

AI 操作至少显示：

```text
正在评价…
正在生成迁移挑战…
正在生成课程草案…
正在生成可信度草案…
```

超过 10 秒可显示：

```text
AI 正在处理，这一步可能需要一些时间。
```

不要伪造百分比。

---

## 二十四、ConsistencyCheckService

新增只读一致性检查。

至少检查：

### Course / Lesson
- CurrentLesson 属于 Course
- LessonRelation 两端存在
- prerequisite DAG 有效

### Cognitive
- CognitiveState.LastLearningTurnID 存在
- CognitiveEvidence 对应 LearningTurn 存在

### Misconception
- Target Lesson 存在
- resolved/reopened 状态与 event 基本一致

### Challenge
- ChallengeAttempt 的 challenge 存在

### Exploration
- Direction target lesson 存在
- Question source direction 存在

### Curriculum
- Blueprint AppliedLessonID 存在
- Draft applied 与 mapping 一致

### Grounding
- GroundingLink Evidence 存在
- Evidence Source 存在

---

## 二十五、Consistency API

新增：

```http
GET /api/v1/system/consistency
```

返回：

```text
healthy
warnings
errors
```

只读，不自动修复。

---

## 二十六、Healthcheck

后端新增：

```http
GET /health
```

至少检查：

```text
process alive
DB connection alive
```

不要调用 DeepSeek。

Docker 增加 healthcheck。

---

## 二十七、Graceful Shutdown

容器停止时保证：

- HTTP graceful shutdown
- SQLite connection 正常关闭
- 不粗暴截断事务

---

## 二十八、日志

关键日志统一：

```text
request_id
operation
latency
result
```

高风险 DB 操作记录：

```text
backup
migration
restore
export
```

AI 保留现有 observability。

禁止记录 secrets。

---

## 二十九、首页 v0.1 收口

首页需要回答：

```text
1. 我现在的主线在哪里？
2. 我今天可以继续什么？
3. 有没有值得探索的支线？
```

保留：
- Course Card
- Exploration Radar

Current Focus 改为真实业务语义。

---

## 三十、移除 Phase 开发标签

用户 UI 中移除：

```text
MVP · Phase 4
MVP · Phase 7
```

统一显示：

```text
LearnOS
```

如果需要版本：

```text
v0.1
```

---

## 三十一、导航收口

侧边栏统一：

```text
学习首页
课程档案
探索空间
知识来源
系统设置
```

不要增加更多一级导航。

---

## 三十二、系统设置页

至少显示：

```text
版本
环境
AI Provider
AI Model
数据库状态
最后备份
```

提供：

```text
立即备份
导出数据
系统诊断
一致性检查
```

不显示 API Key 明文。

---

## 三十三、部署配置准备

准备生产 compose 基础：

```text
docker-compose.yml
docker-compose.prod.yml（如适合）
.env.example
```

要求：

- SQLite volume 持久化
- backups volume 持久化
- DeepSeek Key env
- restart policy
- healthcheck
- log rotation（如适合）
- 非必要端口不暴露

真正服务器部署步骤放下一份 deployment 文档。

---

## 三十四、`.env.example`

至少：

```text
APP_ENV=production
PORT=8080

AI_PROVIDER=deepseek
AI_MODEL=deepseek-v4-flash
DEEPSEEK_API_KEY=

AI_TIMEOUT_SECONDS=45
AI_CHALLENGE_GENERATION_TIMEOUT_SECONDS=60
AI_CURRICULUM_DRAFT_TIMEOUT_SECONDS=60
AI_SOURCE_CREDIBILITY_TIMEOUT_SECONDS=45

DATA_DIR=/app/data
BACKUP_DIR=/app/backups
BACKUP_RETENTION_COUNT=20
```

按项目当前真实变量名调整。

---

## 三十五、版本

新增：

```text
APP_VERSION=0.1.0
```

或 build-time version。

Diagnostics / UI 可读。

---

## 三十六、v0.1 干净初始化验收

创建全新 production DB 后：

应该存在：

```text
Course
Blueprint
Lesson
Relations
```

但：

```text
LearningTurn=0
CognitiveState=0
CognitiveEvidence=0
Misconception=0
ChallengeAttempt=0
ExplorationDirection=0
ExplorationQuestion=0
CurriculumDraft=0
Demo Source=0
```

保证个人认知历史从 0 开始。

---

## 三十七、开发数据保留

当前开发 SQLite 不删除。

可以迁移到：

```text
data/dev/
```

生产使用：

```text
data/prod/
```

开发库继续用于回归测试。

---

## 三十八、Dogfooding Event（可选）

只有现有 audit 很容易复用时才做轻量事件记录：

```text
lesson_opened
answer_submitted
exploration_opened
question_saved
curriculum_draft_applied
ai_error
```

只存本地 SQLite。

如果实现成本明显增加，跳过。

---

## 三十九、不做第三方遥测

不接入：

- Google Analytics
- Mixpanel
- PostHog
- 其他第三方 analytics

Dogfooding 数据保持本地。

---

## 四十、人工验收

### A. Backup

点击：

```text
立即备份
```

预期备份文件创建，DB 继续可用。

### B. AI timeout UX

通过测试 provider 或临时低 timeout。

预期：

```text
AI_TIMEOUT
可重试
明确“本次数据未保存”
```

不能显示裸 `Failed to fetch`。

### C. Export

执行 JSON / Markdown export。

确认：
- 数据可读
- 不包含 API Key

### D. Consistency

正常数据返回：

```text
healthy
```

测试坏引用时能报告 error，但不自动修复。

### E. Restart

```bash
docker compose restart app
```

DB / backup 仍在，healthcheck 恢复 healthy。

### F. 全新生产初始化

使用全新目录：

```text
data/prod-test/
```

执行 production init。

确认：
- 静态知识存在
- 个人历史全为 0

### G. Restore

仅测试环境：

1. 创建 backup A
2. 新增测试学习记录
3. restore A
4. 测试记录消失
5. integrity_check=ok

---

## 四十一、测试要求

至少覆盖：

### Backup
1. backup success
2. retention
3. failure 不破坏 DB
4. pre-migration backup

### Migration
5. schema version
6. migration failure safe abort

### Init
7. production init
8. seed idempotent
9. personal history zero

### Export
10. JSON
11. Markdown
12. secrets excluded

### Restore
13. validation
14. pre-restore backup
15. restore
16. integrity check

### AI Error
17. timeout
18. empty response
19. invalid response
20. provider error
21. retryable flag
22. frontend readable message

### Consistency
23. healthy
24. broken reference detection
25. no auto repair

### Health
26. /health
27. DB unavailable → unhealthy

### Persistence
28. restart after backup
29. restart after export
30. production data persists

---

## 四十二、文档更新

更新：

- README.md
- docs/PRODUCT.md
- docs/ARCHITECTURE.md
- docs/DATABASE.md
- docs/ROADMAP.md

新增：

```text
docs/OPERATIONS.md
```

至少记录：

- backup
- restore
- diagnostics
- export
- production init
- migration safety

---

## 四十三、ROADMAP

人工验收完成后标记：

```text
Phase 10：Dogfooding Readiness & Stabilization ✅
```

之后进入：

```text
LearnOS v0.1 Dogfooding
```

不要立即定义 Phase 11。

---

## 四十四、完成标准

只有同时满足：

```text
生产初始化安全
+
开发库 / 生产库分离
+
SQLite backup 可用
+
restore 可验证
+
migration 前有 backup
+
JSON / Markdown export 可用
+
AI 错误不再只显示 Failed to fetch
+
Consistency Check 可用
+
Docker healthcheck 可用
+
用户 UI 去掉 Phase 开发标记
+
系统可以长期使用
```

才算 Phase 10 完成。

---

## 四十五、Phase 10 后冻结大功能

Phase 10 完成后进入：

```text
4~8 周真实 Dogfooding
```

期间只处理：

- P0 数据安全
- P1 功能错误
- 高频 UX 摩擦
- AI 可靠性
- 性能问题

其他新想法进入：

```text
Future Ideas
```

不立即开发。

---

## 四十六、构建验证

至少执行：

```bash
go test ./...
go build ./cmd/server
npm run build
docker compose build app
git diff --check
```

Go 执行 `gofmt`。

---

## 四十七、最终报告

必须汇报：

```text
完成内容
数据目录设计
backup / restore
migration safety
production init
export
AI error model
frontend retry UX
consistency check
healthcheck
v0.1 UI 收口
production seed 策略
测试结果
Docker build
人工验收步骤
已知技术债
明确说明未新增学习功能
```

---

## 四十八、Codex 实施顺序

1. 阅读仓库
2. 汇报当前生产风险
3. 环境 / 数据目录分离
4. SchemaVersion
5. BackupService
6. Migration pre-backup
7. Production Init
8. Export
9. Restore CLI
10. DB integrity check
11. Diagnostics
12. ConsistencyCheck
13. AIError 统一
14. frontend error/retry UX
15. loading UX
16. healthcheck
17. graceful shutdown
18. 系统设置页
19. 首页 / 导航 v0.1 收口
20. `.env.example`
21. prod compose 基础
22. tests
23. docs/OPERATIONS.md
24. Docker build
25. final report

---

## 四十九、现在开始前先汇报

先不要立即编码。

请先阅读仓库并汇报：

1. 当前 SQLite 文件和 volume 实际位置
2. 当前开发库中有哪些测试 / 个人数据
3. production 初始化是否会误 Seed 测试数据
4. 当前所有 AI timeout / retry 配置
5. 哪些接口当前可能显示 `Failed to fetch`
6. 当前 backup / restore / export 能力
7. AutoMigrate 当前启动顺序
8. Docker healthcheck / restart policy
9. Phase 10 预计新增模型 / service / API
10. 修改文件清单
11. 简短实施顺序

确认兼容 Phase 1~9 后再开始实现。
