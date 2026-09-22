# LearnOS Phase 8：Curriculum Construction & Coverage
## 课程构建机制 / 核心覆盖 / 可审核扩充

请在当前 LearnOS 仓库中实现 **Phase 8：Curriculum Construction & Coverage（课程构建与完整性）**。

本阶段是在 Phase 7 Exploration Engine 之后，正式回到“课程本身怎么被构建和补全”的阶段。

核心问题不是：

> AI 能不能多生成几个 Lesson？

而是：

> **一门课程怎样拥有一个相对稳定、可检查、可扩展的知识骨架，并且在个性化学习与探索支线不断增长时，仍然知道核心知识有没有遗漏。**

---

## 一、当前前提

当前已完成：

- Phase 1：项目骨架
- Phase 2：持久化学习闭环
- Phase 3：AI 结构化认知评价
- Phase 4：Knowledge World / Curriculum Graph
- Phase 5：Personal Cognitive State / Evidence / Evolution
- Phase 6：Transfer Challenge / Misconception Network / Correction Validation
- Phase 7 Preparation：测试知识世界扩充
- Phase 7：Exploration Engine

当前 Course 已经可以包含：

```text
Course
→ Unit
→ Lesson
→ LessonRelation
```

并且 Lesson 已经参与：

```text
Learning
CognitiveState
Evidence
Misconception
Transfer
Exploration
```

但仍缺少一个更高层的问题：

```text
“这门课程完整来说应该包含什么？”
```

例如当前营养学虽然已经不止一个喝水问题，但仍然无法系统回答：

```text
蛋白质有没有覆盖？
脂肪有没有覆盖？
维生素矿物质有没有覆盖？
能量平衡有没有覆盖？
食品标签有没有覆盖？
```

Phase 8 就负责建立这个 Blueprint / Coverage 层。

---

# 二、核心理念

必须严格区分三层：

```text
Curriculum Blueprint
= 这个领域应该有哪些核心知识

Knowledge World
= 当前 LearnOS 实际已经存在的 Course / Unit / Lesson

Personal Cognitive State
= 用户当前理解到了哪里
```

禁止混淆。

例如：

```text
Blueprint 显示“蛋白质基础”缺失
```

只表示：

```text
当前 Course World 里还没有对应 Lesson
```

不表示：

```text
用户不懂蛋白质
```

同样：

```text
Lesson 已存在
```

也不表示：

```text
用户已掌握
```

---

# 三、本阶段只实现五件事

```text
1. Curriculum Blueprint
2. Curriculum Coverage
3. Curriculum Draft
4. Draft Review / Apply
5. Nutrition Blueprint v0 测试样本
```

最终形成：

```text
标准课程骨架
↓
和当前 Knowledge World 比较
↓
发现缺失核心内容
↓
生成扩充草案
↓
人工查看
↓
显式 Apply
↓
正式进入 Knowledge World
```

---

# 四、严格禁止

不要实现：

- AI 直接修改正式 Course Graph
- AI 自动新增 Lesson 后立即生效
- AI 自动删除 Lesson
- AI 自动重排 CurrentLesson
- AI 自动修改 CognitiveState
- AI 根据个人偏好删掉核心知识
- 在线搜索
- RAG
- Citation
- 来源可信度系统
- Vector DB
- Embedding
- 多 Agent
- 自动抓教材
- 自动抓论文
- 自动把 Exploration Question 转成正式课程节点
- 完整营养学自动生成
- 多用户课程模板

最重要规则：

> **所有 AI 课程生成结果必须先进入 Draft，禁止直接写入正式 Knowledge World。**

---

# 五、开始前必须阅读

请阅读：

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
- `docs/prompts/phase7.md`

重点检查：

- Course / CourseUnit / Lesson
- LessonRelation
- KnowledgeGraphService
- CognitiveState
- ExplorationDirection
- ExplorationQuestion
- AIProvider / DeepSeekProvider
- Seed
- AutoMigrate
- tests
- 课程档案 / Knowledge Map

开始编码前先汇报：

1. 当前 Course / Unit / Lesson 的唯一标识与更新方式
2. 当前营养学 8 个 hydration Lesson 的实际 ID / Title / Unit
3. 当前 LessonRelation 的约束
4. DAG validator 是否能复用
5. Blueprint 应采用哪些模型
6. 如何保证 Apply Draft 时不破坏已有 Lesson ID 和历史
7. 预计新增 API
8. 修改文件清单
9. 简短实施顺序

先不要立即编码。

---

# 六、CurriculumBlueprint

新增：

```go
CurriculumBlueprint
```

建议字段：

```go
ID uint
CourseID *uint

Name string
Domain string
Description string

LearningGoal string
Audience string
TargetDepth string

Version string

Status string
// draft
// active
// archived

CreatedBy string
// manual
// seed
// ai_assisted

GroundingStatus string
// ungrounded
// provisional
// grounded

CreatedAt
UpdatedAt
```

Phase 8 中默认：

```text
GroundingStatus=provisional
```

因为来源可信度还没实现。

UI 必须提示：

```text
当前课程蓝图为暂定结构，尚未经过来源校验。
```

---

# 七、CurriculumBlueprintUnit

新增：

```go
CurriculumBlueprintUnit
```

建议字段：

```go
ID
BlueprintID

Key
Title
Description
SortOrder

Importance
// core
// recommended
// optional

CreatedAt
UpdatedAt
```

`Key` 必须稳定，例如：

```text
nutrition.foundation
nutrition.macros
nutrition.micros
nutrition.hydration
```

不要用数据库 ID 充当语义 key。

---

# 八、CurriculumBlueprintLesson

新增：

```go
CurriculumBlueprintLesson
```

建议：

```go
ID
BlueprintID
BlueprintUnitID

Key
Title
Summary

Importance
// core
// recommended
// optional

ContentRole
DepthLevel
AssessmentTargetLevel
SortOrder

AppliedLessonID *uint

CreatedAt
UpdatedAt
```

关键字段：

```text
AppliedLessonID
```

表示这个 Blueprint 节点映射到正式 Knowledge World 中哪个 Lesson。

不要每次依赖 Title 猜映射。

---

# 九、CurriculumBlueprintRelation

新增：

```go
CurriculumBlueprintRelation
```

建议：

```go
ID
BlueprintID

FromLessonKey
ToLessonKey

RelationType
// prerequisite
// related
// extends
// application

CreatedAt
```

Blueprint 的 prerequisite 必须保持 DAG。

复用 Phase 4 validator 思路。

---

# 十、Curriculum Coverage

新增：

```go
CurriculumCoverageService
```

比较：

```text
Blueprint
VS
Current Course Knowledge World
```

每个 Blueprint Lesson 至少有：

```text
covered
missing
```

如果存在明显候选但尚未映射，可选：

```text
possible_match
```

但不能 AI 自动认定 covered。

最稳规则：

```text
AppliedLessonID != nil → covered
AppliedLessonID == nil → missing
```

---

# 十一、Coverage Metrics

至少计算：

```text
core_total
core_covered
core_missing

recommended_total
recommended_covered

optional_total
optional_covered

core_coverage_percent
overall_coverage_percent
```

这是：

```text
课程结构完整度
```

不是：

```text
用户学习进度
```

UI 必须明确区分。

---

# 十二、Coverage API

至少：

```http
GET /api/v1/courses/:courseId/curriculum
GET /api/v1/courses/:courseId/curriculum/coverage
```

返回：

- active blueprint
- blueprint units
- blueprint lessons
- coverage state
- metrics
- missing core lessons

---

# 十三、CurriculumDraft

新增：

```go
CurriculumDraft
```

建议：

```go
ID

CourseID
BlueprintID

Title
Summary

Status
// draft
// applied
// rejected

GeneratedBy
// rule
// ai

Provider
Model
PromptVersion

ChangeSetJSON

CreatedAt
UpdatedAt
AppliedAt *time.Time
```

Draft 只保存建议变更。

不得直接改正式 Lesson。

---

# 十四、ChangeSet 最小格式

建议：

```json
{
  "new_units": [],
  "new_lessons": [],
  "new_relations": [],
  "blueprint_mappings": []
}
```

例如 Lesson：

```json
{
  "temp_key": "protein_basics",
  "unit_temp_key": "unit_macros",
  "blueprint_lesson_key": "nutrition.protein.function",
  "title": "蛋白质为什么重要",
  "core_question": "...",
  "expected_understanding": "...",
  "content_role": "foundation",
  "depth_level": 1,
  "assessment_target_level": "understand",
  "is_core": true
}
```

---

# 十五、Draft Apply

实现：

```text
CurriculumDraftService.Apply
```

要求：

1. 必须事务执行
2. 校验 blueprint
3. 校验 key / title 冲突
4. 校验 prerequisite DAG
5. 创建 Unit
6. 创建 Lesson
7. 创建 Relation
8. 更新 BlueprintLesson.AppliedLessonID
9. Draft → applied

任何一步失败：

```text
全部 rollback
```

---

# 十六、历史保护规则

Apply 不得：

- 删除已有 Lesson
- 修改已有 Lesson ID
- 删除 LearningTurn
- 删除 CognitiveState
- 删除 Misconception
- 删除 Challenge
- 删除 Exploration 历史

Phase 8 不做已有 Lesson 自动 merge。

只允许：

```text
新增缺失 Lesson
+
建立 Blueprint mapping
```

已有 Lesson Revision 留以后。

---

# 十七、AI Curriculum Draft Generator

新增 prompt：

```text
learnos-curriculum-draft-v1
```

任务：

> **针对 Blueprint 中 missing 的节点，生成最小新增 Lesson 草案。**

输入：

```text
Course
Blueprint
Missing Blueprint Lessons
Existing Units
Existing Lessons
Relevant Existing Relations
```

不要传：

- LearningTurn 全历史
- CognitiveEvidence 全历史
- Misconception 全历史
- Exploration 全历史

这是 World Structure，不是 Personalization。

---

# 十八、AI 输出

严格 JSON：

```json
{
  "summary": "...",
  "new_units": [],
  "new_lessons": [],
  "new_relations": []
}
```

AI 不允许：

- 删除节点
- 修改 CognitiveState
- 修改 CurrentLesson
- 改学习进度
- 决定哪些 core 可以省略

---

# 十九、Draft Generation API

新增：

```http
POST /api/v1/courses/:courseId/curriculum/drafts
```

输入可选：

```json
{
  "scope": "missing_core",
  "limit": 5
}
```

Phase 8 只支持：

```text
missing_core
missing_recommended
```

默认：

```text
missing_core
```

知识结构页根据当前 Unit 的未覆盖蓝图节点选择生成范围：仍有核心节点时使用 `missing_core`，核心节点完成后使用 `missing_recommended`。仅剩 optional 节点时不再显示“生成节点”按钮，避免提交一个必然为空的草案请求。

从知识结构页发起的 Unit 草案审核完成后，课程档案页通过 `return_to=map` 返回对应课程的知识结构页；直接从课程档案进入的草案审核仍留在课程档案页。

限制：

```text
limit <= 5
```

不要一次生成几十个 Lesson。

---

# 二十、Draft API

至少：

```http
GET  /api/v1/courses/:courseId/curriculum/drafts
GET  /api/v1/courses/:courseId/curriculum/drafts/:draftId

POST /api/v1/courses/:courseId/curriculum/drafts
POST /api/v1/courses/:courseId/curriculum/drafts/:draftId/apply
POST /api/v1/courses/:courseId/curriculum/drafts/:draftId/reject
```

---

# 二十一、AI 失败策略

如果 AI：

```text
timeout
empty content
invalid JSON
```

则：

- 不修改正式 Course
- 不创建 applied changes
- 返回明确错误
- 可以保留 audit

若做 rule-only fallback，必须：

```text
GeneratedBy=rule
```

禁止 silent fake AI success。

---

# 二十二、Nutrition Blueprint v0

Seed：

```text
Nutrition Core Blueprint v0
```

属性：

```text
GroundingStatus=provisional
CreatedBy=seed
Version=v0
```

它不是权威营养学课程。

只用于验证：

```text
当前 hydration 内容只是完整营养学中的一个区域。
```

---

# 二十三、Nutrition Blueprint v0 一级结构

至少：

```text
1. 营养学基础与能量
2. 碳水化合物与膳食纤维
3. 蛋白质
4. 脂肪
5. 维生素
6. 矿物质与电解质
7. 水与体液平衡
8. 食品标签与饮食判断
9. 膳食结构与实际应用
```

不要扩成上百节点。

总 Blueprint Lesson 建议：

```text
24~30 个
```

---

# 二十四、Blueprint Lesson 建议

## 营养学基础与能量

```text
nutrition.foundation.energy_balance
能量摄入与消耗为什么需要平衡
core

nutrition.foundation.nutrient_density
热量充足为什么不等于营养均衡
core

nutrition.foundation.digestion_absorption
食物中的营养素如何被消化和吸收
recommended
```

## 碳水化合物与膳食纤维

```text
nutrition.carb.function
碳水化合物在身体中的主要作用
core

nutrition.carb.quality
不同碳水来源为什么对身体影响不同
core

nutrition.carb.fiber
膳食纤维为什么重要
core
```

## 蛋白质

```text
nutrition.protein.function
蛋白质为什么重要
core

nutrition.protein.quality
蛋白质来源和质量有什么区别
recommended

nutrition.protein.distribution
为什么蛋白质摄入不仅看一天总量
recommended
```

## 脂肪

```text
nutrition.fat.function
脂肪为什么是必需营养素
core

nutrition.fat.types
不同类型脂肪为什么不能一概而论
core

nutrition.fat.saturated
为什么需要关注饱和脂肪
recommended
```

## 维生素

```text
nutrition.vitamin.role
维生素为什么需要少量却不可缺少
core

nutrition.vitamin.solubility
脂溶性和水溶性维生素有什么区别
recommended

nutrition.vitamin.food_first
为什么一般优先从食物获得维生素
recommended
```

## 矿物质与电解质

```text
nutrition.mineral.role
矿物质在人体中主要做什么
core

nutrition.mineral.sodium_potassium
为什么需要同时理解钠和钾
core

nutrition.mineral.calcium_iron
为什么钙和铁的营养问题不能只看单次化验
recommended
```

## 水与体液平衡

把现有 8 个 Lesson 映射到：

```text
nutrition.hydration.water_roles
nutrition.hydration.fluid_balance
nutrition.hydration.thirst_signal
nutrition.hydration.daily_need
nutrition.hydration.heat_exercise
nutrition.hydration.electrolytes
nutrition.hydration.age_condition
nutrition.hydration.integrated_strategy
```

必须保留原 Lesson ID。

BlueprintLesson.AppliedLessonID 指向原 Lesson。

## 食品标签与饮食判断

```text
nutrition.label.ingredients
配料表能告诉我们什么
core

nutrition.label.nutrition_facts
营养成分表能告诉我们什么
core

nutrition.label.combined_reasoning
为什么需要把配料表和营养成分表结合判断
core
```

## 膳食结构与实际应用

```text
nutrition.meal.balance
一餐怎样同时考虑主食、蛋白质和蔬菜
core

nutrition.meal.pattern
为什么长期饮食模式比单顿饭更重要
core

nutrition.meal.optimization
怎样给一顿普通饮食做优先级优化
application
```

---

# 二十五、Blueprint Relation

只建必要 prerequisite。

示例：

```text
energy_balance
→ nutrient_density

carb.function
→ carb.quality

protein.function
→ protein.quality

fat.function
→ fat.types

mineral.role
→ sodium_potassium

water_roles
→ fluid_balance
→ thirst_signal

ingredients
→ combined_reasoning

nutrition_facts
→ combined_reasoning

meal.balance
→ meal.optimization
```

不要把整个课程强制成一条线。

允许并行分支。

---

# 二十六、现有 hydration 映射

Seed 时：

```text
AppliedLessonID
```

直接绑定当前营养学 8 个 Lesson。

不允许删除重建。

Coverage 预期：

```text
水与体液平衡：8 / 8 covered
其他 Unit：大部分 missing
```

因此总体课程覆盖率明显低于 100%。

这是正确结果。

---

# 二十七、UI：课程蓝图

在课程档案或 Knowledge Map 增加：

```text
课程蓝图
```

至少显示：

```text
核心覆盖率
总体覆盖率
```

按 Unit 展示：

```text
营养学基础与能量 0/3
蛋白质 0/3
水与体液平衡 8/8
```

每个 Blueprint Lesson 显示：

```text
已覆盖
缺失
```

---

# 二十八、UI：待补充核心知识

新增区域：

```text
待补充的核心知识
```

例如：

```text
蛋白质为什么重要
脂肪为什么是必需营养素
配料表能告诉我们什么
```

按钮：

```text
生成扩充草案
```

不要叫：

```text
立即创建课程
```

---

# 二十九、UI：Draft Review

Draft 页面或 Drawer 至少展示：

```text
为什么建议补充
新增 Unit
新增 Lesson
新增 Relation
Coverage 将发生什么变化
```

按钮：

```text
应用草案
拒绝草案
```

提示：

```text
应用后会扩充课程知识结构，但不会改变你的学习进度或当前认知状态。
```

---

# 三十、Apply 后语义

Draft Apply 后：

```text
新 Lesson 进入 Knowledge World
```

但：

- 不自动成为 CurrentLesson
- 不创建 CognitiveState
- 不创建 LearningTurn
- 不修改学习进度
- 不自动创建 ExplorationDirection

新 Lesson 默认：

```text
unseen / unknown
```

---

# 三十一、Blueprint 与个性化边界

Blueprint 不根据用户个人情况删除 core。

未来个性化只改变：

```text
学习顺序
讲解方式
案例
探索支线
深度
```

而不是随意删核心领域。

---

# 三十二、Blueprint 与 Exploration 边界

ExplorationDirection 不自动进入 Blueprint。

例如：

```text
我对确认偏误感兴趣
```

不意味着：

```text
确认偏误变成营养学 core Lesson
```

Blueprint 表达学科结构。

Exploration 表达个人探索路径。

继续独立。

---

# 三十三、Grounding 提示

Phase 8 尚未实现来源可信度。

所有 Blueprint 页面都提示：

```text
当前课程蓝图为暂定结构，后续将通过教材、指南与可靠来源进行校验。
```

禁止使用：

```text
权威课程
官方标准
科学证明完整
```

---

# 三十四、人工验收 A

打开营养学：

```text
课程蓝图
```

预期：

```text
水与体液平衡 8/8
其他 Unit 多数缺失
核心覆盖率明显低于 100%
```

---

# 三十五、人工验收 B

点击：

```text
生成缺失核心知识草案
```

limit 3~5。

预期：

```text
生成 Draft
```

但正式 Knowledge Map 不立即出现新节点。

---

# 三十六、人工验收 C

查看 Draft。

至少看到：

```text
新增 Lesson
CoreQuestion
ExpectedUnderstanding
Relation
Blueprint key
```

---

# 三十七、人工验收 D

点击：

```text
应用草案
```

预期：

```text
新 Lesson 正式进入 Course
Blueprint missing → covered
```

但：

```text
CognitiveState 不变
LearningTurn 不新增
CurrentLesson 不变
```

---

# 三十八、人工验收 E

重复 Apply：

```text
不能重复创建
```

已 applied Draft 再 apply：

```text
409 / equivalent domain error
```

---

# 三十九、人工验收 F

执行：

```bash
docker compose restart app
```

Blueprint / Mapping / Draft / Applied Lesson 仍存在。

---

# 四十、测试要求

至少覆盖：

## Blueprint
1. Seed 幂等
2. Key 唯一
3. active blueprint 唯一策略
4. prerequisite DAG

## Mapping
5. hydration Lesson ID 保持
6. AppliedLessonID 正确
7. restart 不重复 mapping

## Coverage
8. covered
9. missing
10. core coverage
11. overall coverage
12. coverage 不把 CognitiveState 当课程覆盖

## Draft
13. 创建 draft
14. AI JSON strict
15. draft 不直接修改 Lesson
16. reject
17. apply transaction
18. apply rollback
19. duplicate apply 防护

## Cognitive Safety
20. apply 不创建 CognitiveState
21. apply 不创建 LearningTurn
22. apply 不修改 CurrentLesson
23. apply 不删除历史

## Persistence
24. restart 后 blueprint 存在
25. draft 存在
26. mapping 存在

---

# 四十一、AI 可观察性

日志：

```text
curriculum draft ai:
course_id=...
blueprint_id=...
missing_count=...
provider=...
model=...
prompt_version=learnos-curriculum-draft-v1
attempt=...
provider_latency_ms=...
prompt_chars=...
raw_content_chars=...
status=...
```

禁止记录 API Key。

---

# 四十二、Timeout

新增独立：

```text
AI_CURRICULUM_DRAFT_TIMEOUT_SECONDS=60
```

不要复用普通 Evaluation timeout 或 Challenge timeout。

生成限制 <=5 Lesson。

---

# 四十三、文档

更新：

- README.md
- docs/PRODUCT.md
- docs/ARCHITECTURE.md
- docs/DATABASE.md
- docs/ROADMAP.md

人工验收完成前不要提前标 Phase 8 ✅。

---

# 四十四、完成标准

必须满足：

```text
存在 Curriculum Blueprint
+
能够计算 Course Coverage
+
能够识别 missing core
+
AI 只能生成 Draft
+
Draft 显式 Apply 才进入 Knowledge World
+
Apply 不污染 Personal Cognitive State
+
现有 hydration Lesson 被安全映射
+
Docker restart 持久化
```

---

# 四十五、后续明确留白

Phase 8 不解决：

```text
Blueprint 是否科学权威
```

下一阶段再处理：

```text
Source / Credibility / Grounding
```

届时 Blueprint 可以通过教材、指南与可靠来源进行校验和修订。

不要在 Phase 8 偷跑。

---

# 四十六、构建验证

至少执行：

```bash
go test ./...
go build ./cmd/server
npm run build
docker compose build app
git diff --check
```

Go 代码执行：

```bash
gofmt
```

---

# 四十七、实施顺序

1. 阅读仓库
2. 汇报 Blueprint 与现有模型兼容方案
3. CurriculumBlueprint
4. BlueprintUnit
5. BlueprintLesson
6. BlueprintRelation
7. DAG validator
8. Seed Nutrition Blueprint v0
9. 安全映射 hydration Lesson
10. CoverageService
11. Coverage API
12. CurriculumDraft
13. Draft Generator
14. AI strict JSON
15. Draft APIs
16. Apply transaction
17. Duplicate / rollback protection
18. 课程蓝图 UI
19. Coverage UI
20. Draft Review UI
21. tests
22. Docker build
23. docs
24. final report

---

# 四十八、现在开始前先汇报

先不要立即编码。

请先阅读仓库并汇报：

1. 当前 Course / Unit / Lesson 的实际模型与唯一标识方式
2. 当前营养学 8 个 hydration Lesson 的 ID / Title / Unit
3. LessonRelation / KnowledgeGraphService 的 DAG 校验能否复用
4. Blueprint 应采用哪些模型
5. Nutrition Blueprint v0 如何映射当前 8 个 hydration Lesson
6. Curriculum Coverage 如何和 Personal CognitiveState 完全独立
7. CurriculumDraft Apply 如何保证事务、ID 与历史安全
8. 本阶段预计新增 API
9. 修改文件清单
10. 简短实施顺序

确认兼容 Phase 1~7 后再开始实现。
