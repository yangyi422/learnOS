# LearnOS Phase 9：Source, Credibility & Grounding

请在当前 LearnOS 仓库中实现 **Phase 9：Source, Credibility & Grounding（来源、可信度与知识校验）**。

Phase 8 已经建立：

```text
Curriculum Blueprint
→ Coverage
→ Missing Core
→ Curriculum Draft
→ Human Review
→ Apply
→ Knowledge World
```

现在新增第四层：

```text
Grounding / Source Layer
= “为什么认为这些课程结构和知识内容值得相信？”
```

本阶段的核心不是“给 Lesson 随便挂几个链接”，而是让 LearnOS 能明确记录：
- 哪些来源支持课程结构或知识节点
- 来源本身是什么性质
- 支持到什么程度
- 是否存在限制或冲突
- 当前多少内容仍只是 provisional

## 1. 当前状态

已完成：
- Phase 1 项目骨架
- Phase 2 持久化学习闭环
- Phase 3 AI 结构化认知评价
- Phase 4 Knowledge World
- Phase 5 Personal Cognitive State
- Phase 6 Transfer / Misconception
- Phase 7 Exploration Engine
- Phase 8 Curriculum Blueprint / Coverage / Draft / Apply

Phase 8 中 Blueprint 当前应保持：

```text
GroundingStatus=provisional
```

Phase 9 要让：

```text
provisional
→ partially_grounded
→ grounded
```

成为有证据、有审计轨迹的过程。

---

## 2. Phase 9 只实现五个能力

```text
1. Source Registry
2. Source Evidence
3. Grounding Links
4. Credibility Assessment
5. Grounding Coverage / Review
```

目标链路：

```text
Source
↓
Evidence
↓
GroundingLink
↓
Credibility Assessment
↓
Grounding Coverage
↓
Human Review
↓
Grounding Status
```

---

## 3. 严格不做

不要实现：
- 自动网页搜索
- 自动联网抓取网页
- RAG
- Vector DB
- Embedding
- 自动全文索引
- 自动论文检索
- 自动教材抓取
- 浏览器插件
- Zotero 双向同步
- PDF OCR
- 多 Agent research
- AI 自动宣判“科学真理”
- AI 自动把 Blueprint 改成 grounded
- AI 自动删除冲突知识
- 自动修改 CognitiveState
- 自动修改已有 Lesson 内容

本阶段只建立：

> **来源和知识之间的结构化、可审核连接。**

---

## 4. 开始前必须阅读

请阅读：
- AGENTS.md
- README.md
- docs/PRODUCT.md
- docs/ARCHITECTURE.md
- docs/DATABASE.md
- docs/ROADMAP.md
- docs/prompts/phase4.md
- docs/prompts/phase5.md
- docs/prompts/phase6.md
- docs/prompts/phase7.md
- docs/prompts/phase8.md

重点检查：
- Course / CourseUnit / Lesson / LessonRelation
- CurriculumBlueprint / Unit / Lesson / Relation
- CurriculumDraft
- CognitiveState
- KnowledgeGraphService
- CurriculumCoverageService
- CurriculumDraftService
- AIProvider / DeepSeekProvider
- AutoMigrate / tests / Docker / API routes
- 课程档案 / Knowledge Map / Blueprint Coverage UI

开始编码前先汇报：
1. 当前 Blueprint / Lesson 是否已有 GroundingStatus
2. 哪些实体适合作为 Grounding Target
3. 当前是否已有 Source / Citation / Evidence 模型可复用
4. Source Registry 应如何与 Course 解耦
5. Grounding Coverage 如何与 Curriculum Coverage、Personal CognitiveState 独立
6. 预计新增模型
7. 预计新增 API
8. 修改文件清单
9. 简短实施顺序

先不要立即编码。

---

## 5. Grounding Target

Phase 9 只支持：

```text
curriculum_blueprint
blueprint_lesson
lesson
```

暂不支持：
- CognitiveState
- Misconception
- ExplorationDirection
- ChallengeAttempt
- LearningTurn

外部知识证据与个人认知证据必须分开。

---

## 6. KnowledgeSource

新增：

```go
KnowledgeSource
```

建议字段：

```go
ID uint

Title string
Authors string
Organization string

SourceType string
// textbook
// guideline
// consensus
// systematic_review
// review
// research_paper
// official_web
// reference_work
// course_material
// user_note
// other

Publisher string
PublicationYear *int
PublishedAt *time.Time
UpdatedAtSource *time.Time

URL string
DOI string
ISBN string

Language string
Description string

AccessStatus string
// metadata_only
// excerpt_available
// fulltext_available

VerificationStatus string
// unverified
// metadata_verified
// reviewed

CreatedBy string
// manual
// seed
// import
// ai_assisted

CreatedAt
UpdatedAt
```

要求：
- Source 与 Course 解耦
- 同一 Source 可支持多个 Course / Lesson
- URL 不必填
- DOI / ISBN / URL 只是 metadata
- 不因 URL 存在就自动联网抓取

---

## 7. Source 去重

业务级去重优先：
1. DOI exact
2. ISBN exact
3. normalized URL exact
4. Title + Organization + Year

不要做向量模糊去重。

重复创建时返回 409 或现有项目等价错误。

---

## 8. SourceEvidence

新增：

```go
SourceEvidence
```

建议字段：

```go
ID uint
SourceID uint

EvidenceType string
// excerpt
// summary
// table
// figure
// recommendation
// definition
// finding
// methodology

Locator string
Quote string
Summary string

Language string

ExtractionMethod string
// manual
// ai_assisted

VerificationStatus string
// unverified
// reviewed

CreatedAt
UpdatedAt
```

要求：

```text
Quote / Summary 至少一个非空
```

不要存整篇文章或整本书的大段正文。

---

## 9. GroundingLink

新增：

```go
GroundingLink
```

建议字段：

```go
ID uint
EvidenceID uint

TargetType string
// curriculum_blueprint
// blueprint_lesson
// lesson

TargetID uint

Relation string
// supports
// contradicts
// contextualizes
// limits
// defines

Strength string
// weak
// moderate
// strong

Rationale string

Status string
// proposed
// reviewed
// rejected

CreatedBy string
// manual
// ai_assisted

ReviewedAt *time.Time

CreatedAt
UpdatedAt
```

关键规则：

```text
Source 本身不能直接让知识 grounded。
必须经过：
Source → Evidence → GroundingLink → Review
```

---

## 10. 冲突与边界

必须支持：
- supports
- contradicts
- limits
- contextualizes

不要假设所有来源一致。

例如一个来源支持一般人群结论，另一个来源指出特殊人群例外，可以用：

```text
limits
```

而不是粗暴判谁对谁错。

---

## 11. SourceCredibilityAssessment

新增：

```go
SourceCredibilityAssessment
```

建议字段：

```go
ID uint
SourceID uint

AuthorityScore int
MethodologyScore int
DirectnessScore int
RecencyScore int
IndependenceScore int

OverallScore int

AuthorityReason string
MethodologyReason string
DirectnessReason string
RecencyReason string
IndependenceReason string

AssessmentMethod string
// manual
// rule
// ai_assisted

Status string
// draft
// reviewed
// stale

CreatedAt
UpdatedAt
ReviewedAt *time.Time
```

每项 0~20，总分 0~100。

---

## 12. Credibility ≠ Truth

UI 必须明确：

```text
可信度评分评价来源质量和适用性，
不等于自动证明来源中每个结论都正确。
```

并保持：

```text
Credibility
≠ Grounding Strength
```

---

## 13. Credibility baseline

Phase 9 只做可解释 heuristic。

例如：
- guideline / consensus：authority 通常较高
- systematic_review：methodology 通常较高
- research_paper：不能因类型直接给高 methodology
- textbook：educational directness 可较高，但 recency 独立评估
- official_web：authority 取决于 organization
- user_note：不能作为高 authority 来源

规则生成：

```text
draft assessment
```

人工 review 后才：

```text
reviewed
```

---

## 14. AI Credibility Assistant

可新增：

```text
learnos-source-credibility-v1
```

输入仅 Source metadata。

输出严格 JSON：

```json
{
  "authority_score": 0,
  "methodology_score": 0,
  "directness_score": 0,
  "recency_score": 0,
  "independence_score": 0,
  "reasons": {
    "authority": "...",
    "methodology": "...",
    "directness": "...",
    "recency": "...",
    "independence": "..."
  }
}
```

但：
- AI 只能生成 draft
- 不得自动 reviewed
- 不得自动创建 GroundingLink
- 不得自动让 Blueprint grounded

---

## 15. Grounding Status

统一支持：

```text
ungrounded
partially_grounded
grounded
conflicted
```

BlueprintLesson 最小规则建议：

```text
ungrounded:
没有 reviewed supports

partially_grounded:
已有 reviewed supports，但未满足 grounded 条件

grounded:
至少 1 个 reviewed supports
+ SourceCredibilityAssessment reviewed
+ OverallScore >= 60
+ Strength >= moderate

conflicted:
同时存在 reviewed supports 与 reviewed contradicts
```

文档明确：

```text
这不是科学真理判定器。
```

---

## 16. GroundingCoverageService

新增：

```go
GroundingCoverageService
```

针对 Blueprint 计算：

```text
core_total
core_grounded
core_partial
core_ungrounded
core_conflicted

recommended_total
recommended_grounded

grounding_percent
```

UI 要能同时显示：

```text
课程结构覆盖率
来源校验覆盖率
我的学习进度
```

三个数字必须分离。

---

## 17. Lesson Grounding

正式 Lesson 页面允许显示：

```text
知识来源
```

至少显示：
- GroundingStatus
- reviewed source count
- 主要 Evidence / Locator

不要挤占 LearningView 主问题区。

---

## 18. Blueprint Grounding 与 Lesson Grounding 分离

BlueprintLesson：

```text
为什么这个知识应该属于课程
```

Lesson：

```text
这个正式知识节点中的内容由什么支撑
```

同一 Evidence 可连接两类 target，但语义不同。

---

## 19. Source API

至少：

```http
GET  /api/v1/sources
POST /api/v1/sources
GET  /api/v1/sources/:id
PATCH /api/v1/sources/:id
```

本阶段不强制 delete。

---

## 20. Evidence API

至少：

```http
POST /api/v1/sources/:id/evidence
GET  /api/v1/sources/:id/evidence
PATCH /api/v1/evidence/:id
```

---

## 21. Grounding API

至少：

```http
POST /api/v1/grounding/links
GET  /api/v1/grounding/links?target_type=...&target_id=...

POST /api/v1/grounding/links/:id/review
POST /api/v1/grounding/links/:id/reject
```

---

## 22. Credibility API

至少：

```http
GET  /api/v1/sources/:id/credibility
POST /api/v1/sources/:id/credibility
POST /api/v1/sources/:id/credibility/review
```

---

## 23. Grounding Coverage API

至少：

```http
GET /api/v1/courses/:courseId/grounding/coverage
```

返回：
- Blueprint grounding metrics
- BlueprintLesson grounding status
- Lesson grounding summary

---

## 24. UI

新增全局或课程内入口：

```text
知识来源
```

如果做全局页面：

```text
/sources
```

至少支持：
- Source 列表
- 新增 Source
- Source Detail
- 添加 Evidence
- Credibility
- Review credibility

Source Detail 至少显示：

```text
Title
SourceType
Authors / Organization
Year
URL / DOI / ISBN
VerificationStatus
Credibility
Evidence
Grounding Links
```

---

## 25. Blueprint Grounding UI

在课程蓝图页面增加：

```text
来源校验
```

每个 Blueprint Lesson 显示：

```text
已校验
部分校验
未校验
存在冲突
```

点击后可查看：
- 支持来源
- 限制条件
- 冲突来源

---

## 26. Phase 8 UI 提示升级

Phase 9 后，原“临时 / 未完全校验”提示升级为：

```text
课程结构覆盖率：X%
来源校验覆盖率：Y%
GroundingStatus：...
```

仍然不能使用：
- 权威课程
- 官方标准
- 已科学证明完整

---

## 27. Draft Apply 后默认

Phase 8 Apply 的新 Lesson：

```text
GroundingStatus=ungrounded
```

AI 生成 Lesson 不等于 grounded。

Draft Review 可提示：

```text
应用后仍需来源校验。
```

---

## 28. 不污染 Personal Cognitive State

以下操作不得修改：
- CognitiveState
- CognitiveEvidence
- LearningTurn
- Mastery
- Misconception
- Transfer
- CurrentLesson

包括：
- 新增 Source
- 新增 Evidence
- Review Credibility
- Review GroundingLink
- GroundingStatus recalculation

---

## 29. Conflict

如果同一 target 同时存在：

```text
reviewed supports
+
reviewed contradicts
```

则：

```text
GroundingStatus=conflicted
```

UI 显示：

```text
存在来源分歧，等待人工判断。
```

不要自动删除、选边或修改 Lesson。

---

## 30. GroundingReviewEvent

建议新增：

```go
GroundingReviewEvent
```

字段：

```go
ID
TargetType
TargetID
FromStatus
ToStatus
Reason
TriggeredBy
// manual / rule
CreatedAt
```

如果现有通用 audit 能复用，也可以复用，但必须保留状态变化历史。

---

## 31. AI 评价不是 Source

Phase 3 / 6 的 DeepSeek Evaluation：

```text
不能进入 KnowledgeSource
```

AI 是 evaluation / reasoning tool，不是外部知识依据。

用户回答、理解总结、Misconception 同样不是 KnowledgeSource。

---

## 32. 测试数据原则

不要 Seed 虚假权威来源。

禁止类似：

```text
WHO Nutrition Guide 2026
OpenAI Nutrition Standard
```

除非仓库中已经有真实确认的数据来源。

可以不 Seed Source。

测试 fixture 若必要，明确用：

```text
Title=Demo Nutrition Reference
SourceType=course_material
VerificationStatus=unverified
```

不能冒充真实权威资料。

---

## 33. 人工验收 A：Source

手动新增：

```text
Demo Nutrition Reference
course_material
unverified
```

预期：
- 成功进入 Source Registry
- 重复唯一 metadata 不重复创建

---

## 34. 人工验收 B：Evidence

给 Source 添加：

```text
Locator=section 1
Summary=用于说明“蛋白质是人体重要营养素之一”的测试摘要。
```

预期：
- Evidence created
- Lesson 不变

---

## 35. 人工验收 C：Grounding

选择：

```text
nutrition.protein.function
蛋白质为什么重要
```

建立：

```text
supports
strength=moderate
status=proposed
```

预期：

```text
不能立即 grounded
```

---

## 36. 人工验收 D：Credibility

创建 draft credibility，人工 review。

如果 OverallScore >= 60，再 review GroundingLink。

预期：

```text
ungrounded
→ partially_grounded / grounded
```

按最终规则执行。

---

## 37. 人工验收 E：Conflict

再添加另一条 reviewed contradicts。

预期：

```text
GroundingStatus=conflicted
```

UI 显示来源分歧。

---

## 38. 人工验收 F：Coverage

打开营养学课程档案：

必须同时看到：

```text
课程结构覆盖率
来源校验覆盖率
```

二者不是同一个数字。

---

## 39. Persistence

执行：

```bash
docker compose restart app
```

Source / Evidence / Credibility / GroundingLink / GroundingStatus / Review history 全部仍存在。

---

## 40. Grounding status recalculation

统一实现：

```go
GroundingService.RecalculateTarget(...)
```

所有：
- Link review
- Link reject
- Credibility review
- Assessment stale

之后都走同一套状态重算。

不要把状态逻辑散在 Controller。

---

## 41. Grounding 与 Curriculum Coverage 完全独立

Curriculum Coverage 看：

```text
AppliedLessonID
```

Grounding Coverage 看：

```text
reviewed GroundingLink + Credibility
```

BlueprintLesson 即使尚未 Apply，也可以有 Grounding Evidence，用来证明：

```text
“这个主题为什么应该属于课程”
```

所以两套覆盖率必须独立。

---

## 42. AI observability

如启用 AI credibility：

```text
source credibility ai:
source_id=...
provider=...
model=...
prompt_version=learnos-source-credibility-v1
attempt=...
provider_latency_ms=...
prompt_chars=...
raw_content_chars=...
status=...
```

不得记录 API Key。

---

## 43. Timeout

新增独立：

```text
AI_SOURCE_CREDIBILITY_TIMEOUT_SECONDS=45
```

不要复用 evaluation / challenge / curriculum draft timeout。

---

## 44. 测试要求

至少覆盖：

### Source
1. create
2. update
3. dedupe
4. 与 Course 解耦

### Evidence
5. create
6. quote/summary 至少一个
7. source relation
8. review

### GroundingLink
9. supports
10. limits
11. contradicts
12. contextualizes
13. proposed 不参与 grounded
14. reviewed 才参与状态

### Credibility
15. draft
16. reviewed
17. overall score
18. AI draft 不自动 reviewed
19. stale（若实现）

### GroundingStatus
20. ungrounded
21. partially_grounded
22. grounded
23. conflicted
24. recalculation

### Coverage
25. blueprint grounding coverage
26. curriculum coverage 独立
27. cognitive progress 独立

### Safety
28. grounding 不修改 CognitiveState
29. 不创建 LearningTurn
30. 不修改 CurrentLesson
31. 不修改 Mastery

### Persistence
32. restart 后 Source 存在
33. Evidence 存在
34. GroundingLink 存在
35. Credibility 存在
36. Review history 存在

---

## 45. Phase 9 完成标准

只有同时满足：

```text
Source Registry
+
Source Evidence
+
GroundingLink
+
Credibility Review
+
GroundingStatus
+
Conflict 状态
+
Grounding Coverage
+
课程覆盖 / 来源覆盖 / 学习进度三者分离
+
Grounding 不污染 Personal Cognitive State
+
Docker restart 持久化
```

才算 Phase 9 完成。

---

## 46. Phase 10 留白

Phase 9 不做：

```text
自动获取来源
```

下一阶段再考虑：

```text
Source Ingestion / Retrieval / Search / RAG
```

Phase 9 先把：

```text
来源存在哪里
如何评价
如何连接知识
如何审计
```

做正确。

---

## 47. 构建验证

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

## 48. Codex 实施顺序

1. 阅读仓库
2. 汇报 Grounding 与现有架构兼容方案
3. KnowledgeSource
4. SourceEvidence
5. GroundingLink
6. SourceCredibilityAssessment
7. GroundingReviewEvent / audit
8. Source repository/service
9. GroundingService
10. Grounding status recalculation
11. GroundingCoverageService
12. Source API
13. Evidence API
14. Grounding API
15. Credibility API
16. Coverage API
17. Source Registry UI
18. Source Detail UI
19. Blueprint Grounding UI
20. Lesson Source UI
21. AI Credibility Assistant
22. tests
23. Docker build
24. docs
25. final report

---

## 49. 现在开始前先汇报

先不要立即编码。

请先阅读仓库并汇报：

1. 当前 Blueprint / Lesson 是否已有 GroundingStatus 字段
2. 当前是否已有 Source / Citation / Evidence 模型可以复用
3. curriculum_blueprint / blueprint_lesson / lesson 三类 Grounding Target 应如何统一实现
4. KnowledgeSource 应如何与 Course 解耦并避免重复
5. SourceEvidence 与 GroundingLink 为什么应该分层
6. SourceCredibilityAssessment 应如何保持“可信度 ≠ 真值”
7. GroundingStatus 的统一 recalculation 应放在哪里
8. Grounding Coverage 如何与 Curriculum Coverage、Personal Cognitive State 完全独立
9. 本阶段预计新增 API
10. 修改文件清单
11. 简短实施顺序

确认与 Phase 1~8 兼容后再开始实现。
