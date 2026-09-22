# LearnOS v0.1 Post-Phase 10：Progressive Empty World Bootstrap
## 空知识世界初始化 / 渐进式领域生成 / Dogfooding 正式起点

Phase 10 已完成。

现在不要继续新增新的“大功能 Phase”。

本任务是 LearnOS v0.1 正式 Dogfooding 前的最后一次产品语义修正与初始化收口。

最重要的产品原则是：

> **生产环境第一次打开 LearnOS 时，不预装任何具体知识库。**

真正的初始状态应是：

```text
System
✅ Schema
✅ System taxonomy
✅ Prompt / AI config
✅ 必要系统规则

Knowledge World
0 Course
0 Unit
0 Lesson
0 Blueprint

Personal Cognitive World
0 LearningTurn
0 CognitiveState
0 Misconception
0 Challenge
0 Exploration
...
```

然后由用户主动输入：

```text
“我想学习营养学”
```

LearnOS 再逐步建立这个领域。

同时必须避免：

> 一次 AI 请求生成整套 Blueprint + 几十个节点 + 初始 Lesson，导致长延迟、空响应、超时和整条流程回滚。

因此本任务必须采用：

# 三阶段渐进式 Domain Initialization

```text
Stage 1：Domain Skeleton
        ↓
Stage 2：Starter Blueprint Expansion
        ↓
Stage 3：Initial Knowledge World
        ↓
正式学习
```

每个阶段：
- 独立 API
- 独立 AI timeout
- 独立持久化
- 可单独重试
- 幂等
- 前一步成功后，后一步失败不能丢失前一步结果

---

# 一、当前项目前提

当前已经完成：

- Phase 1：项目骨架
- Phase 2：持久化学习闭环
- Phase 3：AI 结构化认知评价
- Phase 4：Knowledge World
- Phase 5：Personal Cognitive State
- Phase 6：Transfer / Misconception
- Phase 7：Exploration Engine
- Phase 8：Curriculum Blueprint / Coverage / Draft / Apply
- Phase 9：Source / Credibility / Grounding 基础能力
- Phase 10：Dogfooding Readiness & Stabilization
  - backup / restore
  - migration safety
  - production init
  - export
  - diagnostics
  - consistency check
  - AI error UX
  - healthcheck
  - dev / prod 数据隔离
  - v0.1 UI 收口

本任务基于 Phase 10 之后继续。

---

# 二、先修正 Production Init 语义

如果当前 production init 会 Seed：

```text
营养学
心理学
逻辑与科学思维
Nutrition Blueprint
Demo Lesson
Demo Relation
```

需要改掉。

生产 / dogfooding 初始化只能 Seed：

```text
System Seed
```

例如：

- Reasoning Pattern taxonomy
- Cognitive Level / Status 所需静态配置（若数据库持久化）
- Relation type 所需静态配置（若数据库持久化）
- Prompt / System config（若数据库持久化）
- SchemaVersion
- 必要系统设置

生产初始状态必须满足：

```text
Course = 0
CourseUnit = 0
Lesson = 0
LessonRelation = 0
CrossCourseLessonRelation = 0

CurriculumBlueprint = 0
CurriculumBlueprintUnit = 0
CurriculumBlueprintLesson = 0
CurriculumBlueprintRelation = 0

CurriculumDraft = 0

LearningTurn = 0
CognitiveState = 0
CognitiveEvidence = 0
CognitiveStateEvent = 0

Misconception = 0
MisconceptionEvent = 0

AssessmentChallenge = 0
ChallengeAttempt = 0

ExplorationDirection = 0
ExplorationQuestion = 0

KnowledgeSource = 0
SourceEvidence = 0
GroundingLink = 0
SourceCredibilityAssessment = 0
```

开发环境 Demo World 保留，不删除。

---

# 三、Development / Production 严格分离

建议：

```text
APP_ENV=development
DEMO_SEED_ENABLED=true
```

开发环境可以继续初始化：

```text
营养学
心理学
逻辑与科学思维
现有 Phase 1~9 回归测试数据
```

生产 / dogfooding：

```text
APP_ENV=production
DEMO_SEED_ENABLED=false
```

只能 System Seed。

不要删除当前 dev DB。

---

# 四、首页必须支持 0 Course

第一次打开：

```text
LearnOS

你的知识世界还是空的。

从一个你真正想了解的领域开始。

[创建第一个学习领域]
```

辅助文案：

```text
你不需要先知道完整课程应该长什么样。
告诉 LearnOS 你想认识什么，它会先建立一张领域地图，
再逐步展开最适合你开始学习的区域。
```

不要默认推荐营养学。

---

# 五、新模型：DomainInitializationDraft

新增：

```go
DomainInitializationDraft
```

如现有 Draft 基础设施能安全复用，可以复用部分通用字段。

建议字段：

```go
ID uint

DomainName string
LearningGoal string
TargetDepth string

Status string
// skeleton_draft
// skeleton_confirmed
// starter_expanded
// world_ready
// applied
// rejected

GeneratedBy string
Provider string
Model string

SkeletonPromptVersion string
StarterPromptVersion string
WorldPromptVersion string

SkeletonJSON string
StarterBlueprintJSON string
InitialWorldJSON string

CreatedAt
UpdatedAt
AppliedAt *time.Time
```

核心要求：

> Draft 在正式 Apply 前不能创建正式 Course / Lesson / Blueprint。

---

# 六、Domain 输入

UI：

```text
创建学习领域
```

至少：

```text
领域名称
为什么想学
期望深度
```

示例：

```text
领域：
营养学

学习目标：
改善日常饮食，并系统理解营养学基础

期望深度：
系统学习
```

深度枚举：

```text
overview     快速了解
foundation   基础入门
systematic   系统学习
```

不要做复杂层级。

---

# 七、Stage 1：Domain Skeleton

## 目标

只回答：

> 这个领域大致由哪些主要区域组成？

Stage 1 禁止生成：

- 20~30 个 Blueprint Lesson
- CoreQuestion
- ExpectedUnderstanding
- 正式 Lesson
- 个人认知数据

只生成：

```text
Course metadata
+
6~10 个 Blueprint Unit Skeleton
```

例如：

```text
营养学
├─ 营养学基础与能量
├─ 碳水化合物
├─ 蛋白质
├─ 脂肪
├─ 维生素
├─ 矿物质
├─ 水与体液
├─ 食品标签
└─ 膳食实践
```

---

# 八、Stage 1 Prompt

新增：

```text
learnos-domain-skeleton-v1
```

输入：

```text
DomainName
LearningGoal
TargetDepth
```

输出严格 JSON：

```json
{
  "course": {
    "name": "...",
    "description": "..."
  },
  "blueprint": {
    "name": "...",
    "domain": "...",
    "learning_goal": "...",
    "target_depth": "...",
    "units": [
      {
        "key": "...",
        "title": "...",
        "description": "...",
        "importance": "core"
      }
    ]
  },
  "recommended_starter_unit_keys": [
    "..."
  ]
}
```

要求：

```text
6 <= units <= 10
1 <= recommended_starter_unit_keys <= 2
```

不要输出 Blueprint Lesson。

---

# 九、Stage 1 API

新增：

```http
POST /api/v1/domains/drafts
GET  /api/v1/domains/drafts/:id
```

POST 创建 Skeleton Draft。

请求：

```json
{
  "domain_name": "营养学",
  "learning_goal": "改善日常饮食，并系统理解营养学基础",
  "target_depth": "systematic"
}
```

成功后：

```text
DomainInitializationDraft 持久化
Course 仍然 = 0
Blueprint 仍然 = 0
Lesson 仍然 = 0
```

---

# 十、Stage 1 Review UI

生成后展示：

```text
营养学

领域地图：
- 营养学基础与能量
- 碳水化合物
- 蛋白质
...
```

标出：

```text
建议从以下区域开始：
营养学基础与能量
```

按钮：

```text
继续展开起步区域
重新生成领域地图
取消
```

---

# 十一、Stage 1 Regenerate

API：

```http
POST /api/v1/domains/drafts/:id/regenerate-skeleton
```

旧 Skeleton Draft 可以覆盖当前 draft 内容或保留版本历史，按现有项目风格决定。

但：

```text
正式 Course 仍为 0
```

---

# 十二、Stage 2：Starter Blueprint Expansion

## 目标

只展开最适合作为学习起点的：

```text
1~2 个 Unit
```

生成总计：

```text
约 5~10 个 Blueprint Lesson
```

Stage 2 不是完整领域 Blueprint。

其他 Unit 保持：

```text
collapsed / unexpanded
```

---

# 十三、Stage 2 Prompt

新增：

```text
learnos-domain-starter-blueprint-v1
```

输入：

```text
DomainName
LearningGoal
TargetDepth
Skeleton Units
Selected Starter Unit Keys
```

输出严格 JSON：

```json
{
  "expanded_units": [
    {
      "key": "...",
      "lessons": [
        {
          "key": "...",
          "title": "...",
          "summary": "...",
          "importance": "core",
          "content_role": "foundation",
          "depth_level": 1,
          "assessment_target_level": "understand"
        }
      ]
    }
  ],
  "relations": [
    {
      "from_lesson_key": "...",
      "to_lesson_key": "...",
      "relation_type": "prerequisite"
    }
  ]
}
```

要求：

```text
5 <= total lessons <= 10
```

---

# 十四、Stage 2 API

新增：

```http
POST /api/v1/domains/drafts/:id/expand-starter
```

可以允许用户选择 1~2 个 starter unit keys。

成功后：

```text
StarterBlueprintJSON 持久化
Draft.Status = starter_expanded
```

正式：

```text
Course = 0
Blueprint = 0
Lesson = 0
```

---

# 十五、Stage 2 UI

展示：

```text
领域地图
✅ 营养学基础与能量（已展开）
✅ 碳水化合物（已展开）
○ 蛋白质（以后展开）
○ 脂肪（以后展开）
...
```

展开 Unit 中显示 Blueprint Lesson。

按钮：

```text
准备第一批课程
重新展开
返回
```

---

# 十六、Stage 3：Initial Knowledge World

## 目标

从已展开 Blueprint Lesson 中生成：

```text
3~5 个正式学习候选 Lesson
```

但仍先保存到 Draft。

每个 Lesson 必须具备：

```text
Title
CoreQuestion
ExpectedUnderstanding
ContentRole
DepthLevel
AssessmentTargetLevel
IsCore
BlueprintLessonKey
```

---

# 十七、Stage 3 Prompt

新增：

```text
learnos-domain-initial-world-v1
```

输入：

```text
Domain
LearningGoal
TargetDepth
Expanded Starter Blueprint
```

输出：

```json
{
  "initial_lessons": [
    {
      "blueprint_lesson_key": "...",
      "title": "...",
      "core_question": "...",
      "expected_understanding": "...",
      "content_role": "foundation",
      "depth_level": 1,
      "assessment_target_level": "understand",
      "is_core": true
    }
  ],
  "relations": [
    {
      "from_blueprint_lesson_key": "...",
      "to_blueprint_lesson_key": "...",
      "relation_type": "prerequisite"
    }
  ],
  "recommended_first_lesson_key": "..."
}
```

要求：

```text
3 <= initial_lessons <= 5
```

优先：

```text
foundation
core
depth 1
prerequisite 少
```

---

# 十八、Stage 3 API

新增：

```http
POST /api/v1/domains/drafts/:id/generate-initial-world
```

成功后：

```text
InitialWorldJSON 持久化
Draft.Status = world_ready
```

正式 Course / Lesson 仍然不存在。

---

# 十九、三阶段 timeout 必须完全独立

不要使用一个共享 60s Context 覆盖三步。

建议新增：

```text
AI_DOMAIN_SKELETON_TIMEOUT_SECONDS=45
AI_DOMAIN_STARTER_BLUEPRINT_TIMEOUT_SECONDS=60
AI_DOMAIN_INITIAL_WORLD_TIMEOUT_SECONDS=60
```

每个 API 请求拥有独立 timeout。

---

# 二十、三阶段失败语义

例如：

```text
Stage 1 success
Stage 2 success
Stage 3 timeout
```

必须保持：

```text
SkeletonJSON ✅
StarterBlueprintJSON ✅
```

用户只需要：

```text
重试 Stage 3
```

不要重新从 Stage 1 开始。

---

# 二十一、AI Error UX

复用 Phase 10：

```text
AI_TIMEOUT
AI_EMPTY_RESPONSE
AI_INVALID_RESPONSE
AI_PROVIDER_ERROR
AI_NETWORK_ERROR
```

UI 示例：

```text
第一批课程生成超时。

你已经确认的领域地图和起步区域不会丢失。

[重新生成第一批课程]
```

禁止显示裸：

```text
Failed to fetch
```

---

# 二十二、Provider Retry

继续沿用：

```text
provider layer 最多 retry once
```

不要无限 retry。

每个 Stage 自己重试，不跨 Stage 重试。

---

# 二十三、Stage AI 可观察性

日志：

```text
domain initialization ai:
draft_id=...
stage=skeleton|starter_blueprint|initial_world
provider=...
model=...
prompt_version=...
attempt=...
provider_latency_ms=...
prompt_chars=...
raw_content_chars=...
status=...
failure_type=...
total_latency_ms=...
```

不记录 API Key。

---

# 二十四、用户最终 Review

Stage 3 成功后，显示：

```text
营养学知识世界准备完成

领域地图：
9 个区域

当前已展开：
2 个区域

准备创建：
4 个初始 Lesson

建议第一课：
能量摄入与消耗为什么需要平衡
```

按钮：

```text
确认并创建
重新生成第一批课程
取消
```

---

# 二十五、最终 Apply

API：

```http
POST /api/v1/domains/drafts/:id/apply
```

事务中：

1. 校验 Draft.Status=world_ready
2. 防止同名 Domain 已存在
3. 创建 Course
4. 创建 CurriculumBlueprint
5. 创建 Skeleton Blueprint Units
6. 对已展开 Unit 创建 Blueprint Lessons
7. 创建 Blueprint Relations
8. 创建 3~5 个正式 Lesson
9. 创建必要 LessonRelation
10. 更新 BlueprintLesson.AppliedLessonID
11. Course.CurrentLesson = recommended first lesson
12. Draft.Status=applied

任何一步失败：

```text
全部 rollback
```

---

# 二十六、未展开 Blueprint Unit 怎么处理

Skeleton 中未展开 Unit 仍然应该创建为：

```text
CurriculumBlueprintUnit
```

但其中：

```text
BlueprintLesson = 0
```

或者用显式：

```text
ExpansionStatus=unexpanded
```

如果现有 BlueprintUnit 没有此字段，可新增：

```text
ExpansionStatus
// unexpanded
// expanded
```

避免把“Unit 存在但没 Lesson”误判为结构错误。

---

# 二十七、后续展开不要新建第二套逻辑

以后用户学到附近时，未展开 Unit 可以继续通过现有 Curriculum 系统扩展。

本任务可只预留：

```text
Expand Unit
```

的 service boundary。

如果实现成本低，可以新增：

```http
POST /api/v1/courses/:courseId/curriculum/units/:blueprintUnitId/expand
```

但不是 v0.1 必须项。

不要为了它延长本任务。

---

# 二十八、Apply 后 Personal Cognitive 数据必须仍为 0

创建知识世界后：

```text
Course > 0
Blueprint > 0
Lesson = 3~5
```

但：

```text
LearningTurn = 0
CognitiveState = 0
CognitiveEvidence = 0
CognitiveStateEvent = 0
Misconception = 0
AssessmentChallenge = 0
ChallengeAttempt = 0
ExplorationDirection = 0
ExplorationQuestion = 0
```

Lesson 在 UI 中：

```text
unseen / unknown
```

---

# 二十九、第一次真实回答

用户点击：

```text
开始学习
```

只是进入 CurrentLesson。

仍然不创建 LearningTurn。

只有正式提交第一次 Answer：

```text
LearningTurn = 1
```

这才是 Dogfooding 个人认知历史正式起点。

---

# 三十、同名领域防重复

如果已经存在：

```text
营养学
```

再次创建相同 domain：

```text
DOMAIN_ALREADY_EXISTS
```

UI：

```text
这个学习领域已经存在。

[进入课程]
```

不要 merge。

---

# 三十一、Apply 幂等

如果用户点击 Apply 后网络断开，但服务器已经成功：

再次调用同 Draft Apply：

```text
不能创建第二套 Course
```

返回：

```text
already_applied
```

或现有项目等价语义。

---

# 三十二、Empty World API 兼容

Course=0 时以下都必须正常：

- 首页
- Course list
- Current Focus
- Exploration Radar
- Question Pool
- Curriculum
- Sources
- Diagnostics
- Consistency

不得 500。

Exploration：

```text
directions=[]
```

UI：

```text
建立第一个学习领域后，LearnOS 会根据你的学习轨迹发现新的探索方向。
```

---

# 三十三、Source / Grounding 不阻塞 Domain Initialization

Domain 创建时：

```text
KnowledgeSource = 0
GroundingStatus = provisional / ungrounded
```

不要要求用户先录 Source。

Phase 9 保留为后续可选能力。

---

# 三十四、Blueprint 的 Grounding Status

AI 生成：

```text
GroundingStatus=provisional
```

或现有模型中等价值。

不要：

```text
grounded
```

因为没有来源校验。

---

# 三十五、Production Bootstrap

Phase 10 的：

```text
production init
```

修改后应得到：

```text
System ready
Knowledge World empty
Personal World empty
```

启动过程中禁止自动调用 DeepSeek 创建默认领域。

---

# 三十六、Dogfooding 初始化流程

本任务完成后提供一套明确命令。

目标：

```text
创建全新 dogfood DB
↓
production bootstrap
↓
Course=0
↓
启动 Docker
↓
浏览器打开
↓
“你的知识世界还是空的”
↓
用户手动创建第一个领域
```

不要导入当前开发 DB。

---

# 三十七、Baseline Backup

在：

```text
Knowledge World = 0
```

时创建：

```text
v0.1-empty-world-baseline
```

用户创建第一个领域 Apply 成功后再创建：

```text
v0.1-first-domain-baseline
```

---

# 三十八、Development Demo World 保留

当前开发 DB：

```text
营养学 Demo
心理学 Demo
逻辑与科学思维 Demo
测试认知状态
测试 Challenge
Exploration
Curriculum Draft
Grounding
```

全部保留。

这是 regression world，不是 production seed。

---

# 三十九、UI 进度展示

Domain 初始化过程可以用：

```text
① 建立领域地图        ✓
② 展开起步区域        ✓
③ 准备第一批课程      …
```

不要显示虚假百分比。

如果失败：

```text
③ 准备第一批课程      !

前两步已经保存。
```

---

# 四十、不要引入任务队列

v0.1 不需要：

- Redis
- RabbitMQ
- Celery
- background job queue

采用：

```text
小 AI 请求
独立 timeout
阶段持久化
可重试
幂等
```

即可。

---

# 四十一、人工验收 Scenario A：Empty World

全新 DB：

```text
Course=0
Lesson=0
Blueprint=0
Personal data=0
```

首页显示：

```text
你的知识世界还是空的
```

---

# 四十二、Scenario B：Stage 1

输入：

```text
营养学
改善日常饮食，并系统理解营养学基础
系统学习
```

预期：

```text
6~10 Unit Skeleton
```

数据库正式：

```text
Course=0
```

---

# 四十三、Scenario C：Stage 2

点击：

```text
继续展开起步区域
```

预期：

```text
1~2 Unit expanded
5~10 Blueprint Lesson Draft
```

正式：

```text
Course=0
```

---

# 四十四、Scenario D：Stage 3

点击：

```text
准备第一批课程
```

预期：

```text
3~5 Initial Lesson Draft
```

即使本阶段 timeout：

```text
Stage 1 / Stage 2 数据仍存在
```

重试只重试 Stage 3。

---

# 四十五、Scenario E：Apply

点击：

```text
确认并创建
```

预期：

```text
Course=1
Blueprint=1
Lesson=3~5
CurrentLesson valid
```

Personal Cognitive：

```text
全部 0
```

---

# 四十六、Scenario F：第一次学习

进入 CurrentLesson：

```text
unseen / unknown
```

提交正式回答：

```text
LearningTurn=1
```

这条成为 Dogfooding 起点。

---

# 四十七、Scenario G：第二领域

之后点击：

```text
+ 创建学习领域
```

例如：

```text
心理学
```

应生成独立 Draft / Blueprint / Knowledge World。

---

# 四十八、Scenario H：AI timeout

测试任意一个 Stage timeout。

预期：

```text
明确 AI_TIMEOUT
可重试
前一阶段结果保留
不创建半成品 Course
```

---

# 四十九、测试要求

至少覆盖：

## Production Init
1. Course=0
2. Lesson=0
3. Blueprint=0
4. Personal data=0
5. System Seed 正常

## Development
6. Demo seed 保留
7. regression tests 正常

## Skeleton
8. 6~10 units
9. strict JSON
10. timeout
11. empty response
12. regenerate

## Starter Expansion
13. 1~2 units
14. total 5~10 blueprint lessons
15. independent timeout
16. Stage 1 persistence

## Initial World
17. 3~5 lessons
18. CoreQuestion
19. ExpectedUnderstanding
20. Relation valid
21. independent timeout
22. Stage 2 persistence

## Apply
23. transaction
24. idempotency
25. duplicate domain protection
26. Blueprint mapping
27. CurrentLesson valid
28. Personal data remains 0

## Empty UI
29. homepage
30. exploration
31. source
32. diagnostics
33. no 500

## Persistence
34. restart after Stage 1
35. restart after Stage 2
36. restart after Stage 3
37. restart after Apply

---

# 五十、文档更新

更新：

- README.md
- docs/PRODUCT.md
- docs/ARCHITECTURE.md
- docs/DATABASE.md
- docs/ROADMAP.md
- docs/OPERATIONS.md

新增：

```text
docs/DOMAIN_INITIALIZATION.md
```

明确记录：

```text
Empty World
→ Domain Skeleton
→ Starter Blueprint Expansion
→ Initial Knowledge World
→ Review
→ Apply
→ Learning
```

---

# 五十一、构建验证

至少执行：

```bash
go test ./...
go build ./cmd/server
npm run build
docker compose build app
git diff --check
```

Go 执行：

```bash
gofmt
```

---

# 五十二、最终报告

必须汇报：

```text
Production Init 修改
System Seed / Demo Seed 分离
Empty World UI
DomainInitializationDraft
Stage 1 / 2 / 3 实现
每阶段 timeout
每阶段 persistence
AI error / retry
Apply transaction
幂等
Personal Cognitive Safety
测试结果
Docker build
Dogfooding 初始化命令
已知技术债
```

最后明确告诉我：

```text
现在是否可以创建一份全新的 Dogfooding DB，
第一次打开 LearnOS 后从“我想学习营养学”开始真实使用。
```

---

# 五十三、实施顺序

1. 阅读 Phase 10 当前实现
2. 识别 production seed / demo seed
3. 拆分 System Seed / Demo Knowledge Seed
4. 修正 production init
5. 验证 Course=0 backend
6. Empty Home UI
7. DomainInitializationDraft
8. Stage 1 Skeleton
9. Stage 1 Review
10. Stage 2 Starter Expansion
11. Stage 2 Review
12. Stage 3 Initial World
13. Final Review
14. Apply transaction
15. 幂等 / duplicate domain
16. Empty Exploration / Source
17. independent timeout / error UX
18. tests
19. docs
20. Docker build
21. clean dogfood bootstrap
22. final report

---

# 五十四、开始前先汇报

现在先不要删除或修改当前开发数据库。

请先阅读 Phase 10 完成后的仓库，并汇报：

1. 当前 production init 实际 Seed 哪些表
2. 哪些属于 System Seed
3. 哪些属于 Demo Knowledge World
4. 当前 Course=0 时后端是否能正常运行
5. 首页 / Exploration / Curriculum 哪些地方假设一定有 Course
6. Phase 8 的 CurriculumDraft / Apply 哪些逻辑可以复用
7. DomainInitializationDraft 最适合新增还是复用现有 Draft 基础
8. Stage 1 / 2 / 3 各自应该使用什么独立 timeout
9. 如何保证 Stage 3 timeout 时 Stage 1 / 2 不丢失
10. 如何保证 Apply 前 Course 始终为 0
11. 如何保证 Apply 后 Personal Cognitive 数据仍全部为 0
12. 预计修改文件
13. 简短实施顺序

确认方案后再开始实现。
