# Progressive Domain Initialization

LearnOS 的 production / Dogfooding 首次启动保持空知识世界。用户可以从首页或课程档案页重复创建新的学习领域；每个领域都按三个可以独立重试的阶段逐步准备内容：

1. Domain Skeleton：生成 6–10 个 Unit，只保存领域骨架。
2. Starter Blueprint Expansion：用户选择 1–2 个 Unit，生成 5–10 个 Blueprint Lesson。
3. Initial Knowledge World：生成 3–5 个正式 Lesson 草案，供用户确认。

在最后的 Apply 前，数据库不会创建正式 Course、CurriculumBlueprint 或 Lesson。Apply 使用一个事务创建新的课程结构、蓝图和首批 Lesson，并把该 Course 的当前 Lesson 指向建议第一课；不会创建 LearningTurn、CognitiveState、CognitiveEvidence、Misconception 或其他个人学习记录。重复 Apply 返回同一个已创建课程，不会复制课程。已有其他 Course 不会阻止新的学习领域创建，也不会被新的 Draft 修改。

学习领域名称按去除首尾空白、英文大小写不敏感的规则去重。重复领域会返回 `DOMAIN_ALREADY_EXISTS`，前端提供进入已有课程或返回课程档案的操作。

创建表单与服务端使用同一组输入边界：领域名称去除首尾空白后为 2–50 个字符，学习原因为必填且至少 10 个字符（最多 4000 个字符），期望深度只能是 `overview`、`foundation` 或 `systematic`。前端无效输入只显示字段级错误，不发请求；服务端仍独立校验并以 `DOMAIN_INPUT_INVALID` 返回可读错误。提交期间按钮锁定，同一页面上的连续点击不会重复创建 Draft。

## 环境边界

```env
APP_ENV=production
DEMO_SEED_ENABLED=false
```

production 的 `init` 和首次启动只执行 schema / system metadata 初始化。开发环境默认 `DEMO_SEED_ENABLED=true`，继续提供既有 Demo World。两种环境应使用不同的 `APP_DATA_DIR` 或 Docker volume；不要把生产库切换到开发数据目录。

## API

```text
POST /api/v1/domains/drafts
GET  /api/v1/domains/drafts/:draftId
POST /api/v1/domains/drafts/:draftId/regenerate-skeleton
POST /api/v1/domains/drafts/:draftId/expand-starter
POST /api/v1/domains/drafts/:draftId/generate-initial-world
POST /api/v1/domains/drafts/:draftId/apply
```

三个 AI 阶段分别使用以下 timeout，并由各自的持久化字段记录 prompt/provider/model：

```text
AI_DOMAIN_SKELETON_TIMEOUT_SECONDS=120
AI_DOMAIN_STARTER_BLUEPRINT_TIMEOUT_SECONDS=60
AI_DOMAIN_INITIAL_WORLD_TIMEOUT_SECONDS=180
```

AI 输出必须通过严格 JSON 和业务范围校验；服务端拒绝未知字段、错误枚举、错误数量和无效关系。对于模型偶尔返回的 ` ```json ` 代码围栏，会先去除展示包装，再执行同样的严格校验。阶段失败不会覆盖已经保存的前序阶段；对已完成的 Stage 3 重复请求直接返回已保存结果。Apply 前可以关闭页面，之后通过 draft ID 继续。

## 网页验收

生产环境打开 `http://127.0.0.1:8888/`。空生产库会显示“创建第一个学习领域”；已有课程时可从“课程档案 → + 新建学习领域”进入同一个 `/domains/new` 流程。开发环境仍使用 `http://127.0.0.1:8080/domains/new` 验证流程。建议依次确认：

- Stage 1 显示 6–10 个 Unit，数据库仍为零 Course / Blueprint / Lesson。
- Stage 2 只展开 1–2 个 Unit，刷新页面后结果仍在。
- Stage 3 显示 3–5 个 Lesson 草案，超时或刷新不会丢失 Stage 1/2。
- Apply 后只出现一个新 Course，个人学习记录和认知数据仍为零。
- 再次 GET / Apply 同一个 draft 不会新增 ID 或复制课程。
- 已有 Course 时创建第二、第三个不同领域，每个 Course 都有独立 CurrentLesson、Knowledge Map、Coverage 和学习记录。

`AI_PROVIDER=mock` 可用于无外网的页面验收；切换为 DeepSeek 前请设置 API Key，并保留阶段独立 timeout。

也可以在“系统设置 → AI 配置”中选择 DeepSeek 并保存 API Key。保存后无需重启即可用于三阶段领域初始化；未配置真实 Key 时可继续使用 Mock。
