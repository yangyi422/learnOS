# LearnOS Phase 7：Exploration Engine / 探索系统

请在当前 LearnOS 仓库中实现 **Phase 7：Exploration Engine（探索系统）**。

本阶段的目标不是继续增加课程数量，也不是做通用推荐系统，而是第一次让 LearnOS 开始主动回答：

> **“基于我已经认识的世界，我接下来还有什么值得认识？”**

当前已经完成：

- Phase 1：项目骨架
- Phase 2：持久化学习闭环
- Phase 3：DeepSeek AI 结构化认知评价
- Phase 4：Knowledge World / Curriculum Graph
- Phase 5：Personal Cognitive State / Evidence / Evolution
- Phase 6：Transfer Challenge / Misconception Network / Correction Validation
- Phase 7 Preparation：测试知识世界扩充
  - 营养学：8 个可学习节点
  - 逻辑与科学思维：3 个可学习节点
  - 心理学：3 个可学习节点

当前测试世界已经具备已学习/未接触节点、CognitiveState、Misconception、Reasoning Pattern、Transfer Evidence、多 Course 和少量跨域测试素材。

现在开始正式 Phase 7。

---

## 一、开始前必须阅读

请完整阅读：

- `AGENTS.md`
- `README.md`
- `docs/PRODUCT.md`
- `docs/ARCHITECTURE.md`
- `docs/DATABASE.md`
- `docs/ROADMAP.md`
- `docs/prompts/phase4.md`
- `docs/prompts/phase5.md`
- `docs/prompts/phase6.md`
- `docs/prompts/phase7-prep.md`

重点阅读当前实现：

- Course / CourseUnit / Lesson
- LessonRelation
- CrossCourseLessonRelation（如果存在）
- KnowledgeGraphService
- CognitiveState / CognitiveEvidence / CognitiveStateEvent
- Misconception / MisconceptionEvent / MisconceptionPatternLink
- AssessmentChallenge / ChallengeAttempt
- AIProvider / DeepSeekProvider
- AIEvaluationRun
- Seed 逻辑
- 首页 / 课程档案 / Knowledge Map / LearningView / Misconception 页面
- API client / stores
- tests
- migrations / AutoMigrate
- Dockerfile / docker-compose.yml

开始编码前先汇报：

1. 当前知识图谱和认知状态可提供哪些 Exploration 输入
2. 当前是否已经存在跨 Course Relation
3. 当前首页 / Knowledge Map 最适合承载 Exploration UI 的位置
4. 本阶段预计新增的数据模型
5. 本阶段预计新增 API
6. 本阶段预计修改文件
7. 简短实施计划

不要立即编码。

---

# 二、Phase 7 核心目标

本阶段只实现三个能力：

```text
Exploration Radar
陌生知识入口
Question Pool
```

它们必须形成一个闭环，而不是三个孤立按钮。

目标链路：

```text
Knowledge World
+
Personal Cognitive State
+
Misconception / Reasoning Pattern
+
Transfer Evidence
        ↓
Exploration Engine
        ↓
产生少量“值得探索的方向”
        ↓
用户查看理由
        ↓
加入 Question Pool
        ↓
用户选择一个问题
        ↓
进入已有 Lesson / 形成探索入口
```

重点：

> **解释“为什么值得探索”，而不是只给推荐结果。**

---

# 三、Phase 7 不做什么

明确禁止提前实现：

- AI 自动创建完整 Course
- AI 自动创建正式 Lesson
- AI 自动修改 Knowledge Graph
- 自动扩张 Curriculum DAG
- RAG
- Vector DB
- Embedding
- 在线搜索
- Citation / Sources
- 知识可信度系统
- 多 Agent 编排
- 自动学习计划
- 自动排课
- Spaced Repetition
- 自动通知
- 推荐模型训练
- 用户画像系统
- 多用户
- Gamification
- 长期兴趣模型
- 自动课程生成器

本阶段只做：

```text
基于当前已有知识世界进行探索推荐
+
问题池管理
```

---

# 四、Exploration 的三类方向

Phase 7 只支持：

```text
adjacent
cross_domain
unknown
```

## adjacent

邻近探索，来源：

- downstream
- related
- extends
- application
- prerequisite gap
- 同 Course 内尚未接触但结构上接近的 Lesson

## cross_domain

跨领域探索，来源：

- CrossCourseLessonRelation（如果已实现）
- Reasoning Pattern
- Misconception Pattern
- 人工定义的跨域语义连接

必须解释为什么它与当前学习相关。

## unknown

陌生知识入口。

只从当前数据库已有 Lesson 中选择：

- 用户未学习
- 与当前领域足够不同
- 不依赖互联网
- 不要求与当前 Source 有强关系

其语义必须明确为“扩展认知边界”，不是“你学错了”。

---

# 五、ExplorationDirection 数据模型

新增：

```go
ExplorationDirection
```

建议字段：

```go
ID uint
CourseID uint
SourceLessonID *uint

TargetCourseID uint
TargetLessonID uint

DirectionType string
// adjacent / cross_domain / unknown

Title string
Summary string
WhyWorthExploring string

Score float64

ReasonCode string
// graph_neighbor
// prerequisite_gap
// downstream
// related
// application
// reasoning_pattern
// misconception_pattern
// cross_course_relation
// unfamiliar_boundary

ReasonDataJSON string

Status string
// active / saved / dismissed / opened

GeneratedBy string
// rule / ai

Provider string
Model string
PromptVersion string

CreatedAt time.Time
UpdatedAt time.Time
```

允许按现有风格调整，但必须保留：

```text
source
target
type
why
score
reason
status
provenance
```

---

# 六、ExplorationQuestion 数据模型

新增：

```go
ExplorationQuestion
```

建议字段：

```go
ID uint
CourseID uint

SourceDirectionID *uint
SourceLessonID *uint
TargetLessonID *uint

Question string
Context string
WhyThisQuestion string

QuestionType string
// deepen / connect / challenge / unfamiliar

Status string
// open / exploring / answered / archived

Origin string
// exploration_direction / manual / ai

CreatedAt
UpdatedAt
```

本阶段实际只需要完整实现：

```text
open
exploring
archived
```

---

# 七、Exploration Engine 基本原则

## 1. 规则优先，AI 负责表达

先由本地结构产生 candidate：

```text
LessonRelation
CrossCourse Relation
CognitiveState
Misconception Pattern
Reasoning Pattern
unseen nodes
```

AI 只用于：

- 自然化 summary
- why_worth_exploring
- question 文案增强

不要让 AI 决定最终 Target。

## 2. 必须可解释

每个 Direction 必须有：

```text
WhyWorthExploring
ReasonCode
ReasonData
```

禁止只显示“推荐给你”。

## 3. 避免知识茧房

候选足够时，一组 Radar 至少尽量包含：

```text
1 adjacent
1 cross_domain
1 unknown
```

不足则允许缺失，不能伪造。

## 4. 避免重复

目标已经 stable transfer 时默认不推荐。
同一 source + target + type 已 active/saved/opened 时不要重复创建。

## 5. 未接触优先

优先：

```text
unseen
exposed
developing
```

降低：

```text
stable understand
stable apply
stable transfer
```

---

# 八、Exploration Score

只做可解释 heuristic，不做 ML。

建议：

```text
graph direct neighbor      +30
cross-course explicit link +30
reasoning pattern match    +25
misconception pattern      +25
unseen target              +20
developing target          +10
same course                 +5
different course           +15
already stable             -30
already transfer           -50
dismissed recently         -20
```

最终 normalize 到 0~100。

要求：

- Score 可追踪
- ReasonData 说明分数来源
- 禁止 AI score

---

# 九、Radar API

新增：

```http
GET /api/v1/exploration/radar
```

参数：

```text
course_id
lesson_id
limit
```

默认 `limit=6`。

返回至少：

```json
{
  "directions": [
    {
      "id": 1,
      "direction_type": "cross_domain",
      "target_course": {},
      "target_lesson": {},
      "title": "...",
      "summary": "...",
      "why_worth_exploring": "...",
      "score": 82,
      "reason_code": "reasoning_pattern"
    }
  ]
}
```

---

# 十、Candidate Generation

如果请求带 `lesson_id`，以该 Lesson 为 Source。

否则优先：

```text
最近 LearningTurn 对应 Lesson
→ 当前 Course CurrentLesson
→ 无 Source 时基于 Knowledge World 生成陌生探索
```

### adjacent

从：

```text
prerequisite
downstream
related
extends
application
```

中找 target。

### cross-domain

优先：

```text
CrossCourseLessonRelation
```

如果不存在，则基于有限 taxonomy 做映射，不允许 AI 任意发明。

至少支持：

```text
single_factor_reasoning
→ 逻辑与科学思维 / 单因素解释的陷阱

correlation_causation
→ 逻辑与科学思维 / 相关不等于因果

unsupported_assumption
→ 逻辑与科学思维 / 如何判断一条证据有多可靠

overgeneralization
→ 逻辑与科学思维 / 如何判断一条证据有多可靠

binary_thinking
→ 逻辑与科学思维 / 单因素解释的陷阱
```

### unknown

从其他 Course 的 unseen Lesson 中选。

要求：

```text
不与当前 Source 直接相关
未被近期推荐
用户未学习
```

unknown 的 score 不宜超过明显高质量 adjacent/cross-domain。

---

# 十一、Diversity Pass

Radar 结果最终做 diversity pass。

例如 `limit=6` 时优先：

```text
2 adjacent
2 cross_domain
2 unknown
```

候选不足则按剩余 score 补齐。

不得为了凑类型伪造 target。

---

# 十二、Radar UI

首页新增：

```text
探索雷达
```

建议放在课程区域下方，不抢 Current Focus。

副标题：

```text
从你已经认识的地方出发，看看还有哪些值得继续认识。
```

每张卡显示：

```text
类型
目标 Lesson
所属 Course
为什么值得探索
```

中文类型：

```text
adjacent     → 邻近探索
cross_domain → 跨领域
unknown      → 陌生知识
```

按钮：

```text
去看看
加入问题池
暂不感兴趣
```

---

# 十三、Direction Status API

新增：

```http
POST /api/v1/exploration/directions/:id/save
POST /api/v1/exploration/directions/:id/dismiss
POST /api/v1/exploration/directions/:id/open
```

语义：

```text
save → saved
dismiss → dismissed
open → opened
```

`opened` 不代表已学习。

---

# 十四、陌生知识入口

增加：

```text
发现一个陌生知识
```

API：

```http
POST /api/v1/exploration/unfamiliar
```

返回一个 `DirectionType=unknown`。

要求：

- 只从数据库现有 unseen Lesson 选
- 不选 stable / transfer
- 不选当前 Lesson
- 不重复最近若干次 unfamiliar
- 不依赖 AI 才能返回

---

# 十五、Question Pool 页面

新增页面：

```text
/exploration/questions
```

侧边栏可新增：

```text
探索
```

探索页至少包含：

```text
探索雷达
问题池
```

Question Pool 显示：

```text
Question
Context
WhyThisQuestion
Source Course / Lesson
Status
```

操作：

```text
开始探索
归档
```

---

# 十六、加入问题池

从 Direction 点击：

```text
加入问题池
```

API：

```http
POST /api/v1/exploration/directions/:id/questions
```

规则：

- 同 Direction 默认只允许一个 open Question
- 已存在则返回已有，不重复生成

---

# 十七、Question 生成

优先本地模板。

### adjacent

直接复用：

```text
TargetLesson.CoreQuestion
```

### cross-domain

模板：

```text
{SourceLesson} 中的判断方式，和 {TargetLesson} 有什么联系？
```

### unknown

直接：

```text
TargetLesson.CoreQuestion
```

AI 可选增强，但 AI 失败不能阻塞 Question Pool。

---

# 十八、AI Exploration Prompt

新增：

```text
learnos-exploration-v1
```

输入仅：

```text
Source Lesson
Target Lesson
DirectionType
ReasonCode
ReasonData
Current CognitiveState summary
```

输出严格 JSON：

```json
{
  "summary": "...",
  "why_worth_exploring": "...",
  "question": "..."
}
```

AI 禁止决定：

- Score
- Target
- DirectionType
- CognitiveLevel
- 是否推荐

---

# 十九、AI 失败策略

Exploration 是辅助能力，不是核心 Evaluation。

AI 文案增强失败时：

```text
允许返回规则生成的本地文案
```

并记录：

```text
GeneratedBy=rule
```

成功：

```text
GeneratedBy=ai
Provider=deepseek
Model=...
PromptVersion=learnos-exploration-v1
```

不能把 fallback 伪装成 AI success。

---

# 二十、认知安全

以下操作全部不能修改 CognitiveState：

```text
查看 Radar
打开 Direction
加入问题池
发现陌生知识
```

也不能创建：

```text
LearningTurn
MasteryRecord
CognitiveEvidence
```

只有用户真正进入 LearningView 并提交 Answer 后，才走现有认知链路。

---

# 二十一、“去看看”

点击后：

1. `Direction.Status=opened`
2. 跳转到现有 Target Lesson 学习页

但不要自动创建学习记录。

---

# 二十二、Exploration History

新增轻量 API：

```http
GET /api/v1/exploration/history
```

返回最近：

```text
saved
dismissed
opened
```

本阶段不需要复杂历史页面。

---

# 二十三、Misconception / Reasoning Pattern 参与探索

如果 active Misconception 带：

```text
single_factor_reasoning
```

可提高：

```text
单因素解释的陷阱
```

score。

UI 不得使用羞辱性解释。

推荐表达：

```text
你在最近的学习中遇到了一个多因素判断问题，
这个知识点可以帮助你建立更通用的分析框架。
```

---

# 二十四、Transfer 参与探索

如果：

```text
CurrentLevel=transfer
Status=stable
```

则：

- 降低同 Lesson 邻近重复推荐
- 提高 cross-domain / unknown 权重

体现：

```text
尚未掌握 → 邻近深化
已经掌握 → 扩展边界
```

只做简单 heuristic。

---

# 二十五、刷新与持久化

GET radar 不应该每次完全随机。

建议：

- active Direction 持久化
- active 不足时补充
- 同 source + target + type 去重
- dismissed 不立即重现
- opened 不立即重现

简单冷却可采用：

```text
dismissed 7 天
opened 3 天
```

如现有架构不适合，可使用更简单的持久化去重策略。

---

# 二十六、数据库约束

Direction 建议业务唯一：

```text
source_lesson_id
target_lesson_id
direction_type
```

考虑 SQLite nullable unique 行为，可由 service 层辅助去重。

Question Pool：

同 Direction 只能有一个 `open` Question，由 service 保证。

---

# 二十七、API 汇总

至少实现：

```http
GET  /api/v1/exploration/radar
POST /api/v1/exploration/unfamiliar

POST /api/v1/exploration/directions/:id/save
POST /api/v1/exploration/directions/:id/dismiss
POST /api/v1/exploration/directions/:id/open

POST /api/v1/exploration/directions/:id/questions

GET  /api/v1/exploration/questions
POST /api/v1/exploration/questions/:id/start
POST /api/v1/exploration/questions/:id/archive

GET  /api/v1/exploration/history
```

按现有路由风格可微调。

---

# 二十八、日志与可观察性

Exploration AI 日志：

```text
exploration ai:
direction_type=...
source_lesson_id=...
target_lesson_id=...
provider=...
model=...
attempt=...
provider_latency_ms=...
prompt_chars=...
raw_content_chars=...
status=...
```

不记录 API Key、Authorization、完整敏感 Prompt。

---

# 二十九、性能要求

Radar 主请求不能完全依赖 AI。

规则 candidate + DB 查询必须能独立返回。

AI 文案可只增强 top N；AI timeout 时仍返回 rule result。

不要因为 DeepSeek 慢导致首页不可用。

---

# 三十、人工验收

## A. Adjacent

从营养学已学习节点出发，应出现类似：

```text
日常饮水需求应该如何判断
```

并解释它是当前知识的后续 / related / application。

## B. Cross-domain

若已有 `single_factor_reasoning` 或显式跨域 relation，应出现：

```text
逻辑与科学思维
单因素解释的陷阱
```

必须有解释理由。

## C. Unknown

点击：

```text
发现一个陌生知识
```

应返回心理学或其他 unseen Lesson。

要求：

- 不创建 CognitiveState
- 不修改进度
- 再次点击尽量不立即重复

## D. Question Pool

Direction → 加入问题池。

重复点击不重复创建。

## E. Open

点击“去看看”：

```text
Direction → opened
进入 Lesson
```

Lesson 仍然 unseen，直到真正提交 Answer。

## F. Dismiss

“暂不感兴趣”：

```text
Direction → dismissed
```

刷新后不立即出现。

## G. Persistence

```text
docker compose restart app
```

Direction / Question Pool 仍存在。

---

# 三十一、测试要求

至少覆盖：

### Engine
1. adjacent candidate
2. cross-domain candidate
3. unknown candidate
4. stable transfer 降权
5. unseen 优先
6. diversity pass
7. Direction 去重

### Explainability
8. ReasonCode
9. WhyWorthExploring
10. Score 可解释

### Status
11. save
12. dismiss
13. open
14. history

### Question Pool
15. 创建
16. 去重
17. archive
18. start

### Cognitive Safety
19. radar 不修改 CognitiveState
20. unfamiliar 不修改 CognitiveState
21. open 不创建 LearningTurn
22. question 不创建 CognitiveEvidence

### AI
23. AI JSON success
24. AI failure → rule fallback
25. AI 不决定 score
26. AI 不决定 target

### Persistence
27. restart 后 Direction 保留
28. restart 后 Question 保留

---

# 三十二、Phase 7 完成标准

只有满足：

```text
Radar 能从当前 Knowledge World 产生探索方向
+
区分 adjacent / cross-domain / unknown
+
每个方向有“为什么”
+
Question Pool 可持久化
+
陌生知识入口可用
+
Exploration 不污染 CognitiveState
+
AI 失败不阻塞探索
+
Docker restart 持久化
```

才算 Phase 7 完成。

---

# 三十三、文档更新

更新：

- README.md
- docs/PRODUCT.md
- docs/ARCHITECTURE.md
- docs/DATABASE.md
- docs/ROADMAP.md

Phase 7 完成人工验收后再标：

```text
Phase 7：Exploration Engine ✅
```

不要提前开始 Phase 8。

---

# 三十四、构建验证

至少执行：

```bash
go test ./...
go build ./cmd/server
npm run build
docker compose build app
git diff --check
```

Go 执行 `gofmt`。

不得删除测试换取通过。

---

# 三十五、最终报告

必须汇报：

```text
完成内容
新增模型
新增 API
Exploration Score 规则
ReasonCode
Candidate Generation
Diversity Strategy
Reasoning Pattern 映射
Question Pool
AI fallback
UI 修改
测试结果
构建结果
人工验收步骤
明确未实现哪些 Phase 8+ 能力
```

---

# 三十六、Codex 工作顺序

1. 阅读仓库
2. 汇报 Exploration 输入
3. 设计 ExplorationDirection
4. 设计 ExplorationQuestion
5. AutoMigrate
6. Rule-based candidate generation
7. Score
8. Diversity pass
9. ReasonCode / ReasonData
10. AI 文案增强
11. Rule fallback
12. Radar API
13. unfamiliar
14. Direction status
15. Question Pool
16. history
17. 首页 Radar
18. 探索页
19. tests
20. Docker build
21. docs
22. final report

现在开始。

先不要立即编码。

请先阅读仓库并汇报：

1. 当前 Knowledge Graph 可以提供哪些 Exploration candidate
2. CognitiveState / Misconception / Reasoning Pattern 可以如何参与 Exploration
3. 当前是否存在 CrossCourseLessonRelation
4. ExplorationDirection / ExplorationQuestion 应如何融入现有模型
5. Radar 最适合放在哪个页面
6. 本阶段预计新增 API
7. 修改文件清单
8. 简短实施顺序

确认与 Phase 1~6 兼容后再开始实现。
