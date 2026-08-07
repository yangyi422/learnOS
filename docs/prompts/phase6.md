# LearnOS Phase 6：迁移测试、误区网络与认知修正验证

请在当前 LearnOS 仓库中实现 **Phase 6：Transfer Challenge + Misconception Network + Correction Validation**。

当前已经完成并通过人工验收：

- Phase 1：项目骨架
- Phase 2：持久化学习闭环
- Phase 3：DeepSeek AI 结构化认知评价
- Phase 4：知识世界 / Curriculum Graph
- Phase 5：个人认知状态 / Cognitive Evidence / Cognitive Evolution

Phase 5 已经能够记录：

```text
World State
+
Current Cognitive Level
+
Cognitive Status
+
Understanding Summary
+
Cognitive Evidence
+
Cognitive State Event
```

并已经真实验证：

```text
未接触 → 理解 · 稳定
理解 · 稳定 → 理解 · 待复习
理解 · 待复习 → 理解 · 稳定
```

Phase 6 开始解决三个新的问题：

> **我是真的理解，还是只会原来的问题？**

> **不同错误背后是否存在重复出现的底层思维模式？**

> **一个已经发现的误区，什么时候才可以认为真的被修正？**

本阶段形成：

```text
发现错误
↓
记录误区
↓
识别底层模式
↓
针对性重新测试
↓
确认修正
↓
陌生场景迁移
```

不要提前实现探索雷达、问题池、Agent 导航、知识来源可信度或最终图形化认知地图。

---

## 一、开始前必须阅读当前项目

请完整阅读：

- `AGENTS.md`
- `README.md`
- `docs/PRODUCT.md`
- `docs/ARCHITECTURE.md`
- `docs/DATABASE.md`
- `docs/ROADMAP.md`
- `docs/prompts/phase4.md`
- `docs/prompts/phase5.md`
- 当前所有 model
- Course / CourseUnit / Lesson / LessonRelation
- LearningTurn
- MasteryRecord
- Misconception
- AIEvaluationRun
- CognitiveState
- CognitiveEvidence
- CognitiveStateEvent
- AIProvider / DeepSeekProvider
- evaluator-v2 Prompt 与 Validation
- KnowledgeGraphService
- CognitiveStateService
- 当前事务实现
- LearningView
- Knowledge Map
- tests
- Dockerfile / docker-compose.yml

先输出：

1. Phase 5 当前实际架构
2. Phase 6 预计新增 / 修改文件
3. Challenge 数据模型
4. Misconception Network 数据模型
5. Correction Validation 规则
6. Transfer Challenge 如何影响 CognitiveState
7. Prompt 变化
8. 旧数据兼容方案
9. 简短实施顺序

确认兼容当前架构后再编码。

不要大范围重构。

---

# 二、Phase 6 核心原则

## 1. 迁移能力必须通过独立的新任务证明

普通 Lesson CoreQuestion 只能证明当前题目允许证明的层级。

例如：

```text
口渴是否是可靠的饮水依据
AssessmentTargetLevel = understand
```

即使回答非常优秀，也不能凭空升级到 `transfer`。

Phase 6 新增独立的 `Transfer Challenge`。

只有成功完成真正陌生的新场景任务，才允许产生：

```text
CognitiveEvidence:
type = transfer
level = transfer
```

---

## 2. 迁移测试失败不等于原本不理解

例如：

```text
CurrentLevel = understand
Status = stable
```

用户第一次 Transfer Challenge 失败：

```text
不能自动降低 CurrentLevel
不能默认 needs_review
```

因为：

> 尚未证明能迁移 ≠ 已经证明原理解错误。

只有 Challenge 中出现：

```text
明确 contradiction
核心概念错误
原有误区再次出现
```

才可以：

```text
Status = needs_review
```

否则保持原认知状态，只记录 Transfer 尚未验证。

---

## 3. 误区不能因为后来答对一次就自动消失

Misconception 为 `active` 时，不能因为普通 Lesson 下一次回答正确就立即 `resolved`。

真正修正必须通过：

```text
针对这个误区设计的新问题
+
正确回答
+
没有再次出现同一误区
```

之后才允许 `resolved`。

---

## 4. 误区网络不是关键词列表

目标是从：

```text
“天然 = 健康”
“无糖 = 健康”
“高蛋白 = 健康”
```

看到它们可能共同连接到：

```text
binary_thinking
二元化判断
```

Phase 6 记录：

```text
具体 Misconception
↕
Underlying Reasoning Pattern
```

但不要使用向量数据库、Embedding 或复杂聚类。

---

# 三、新增 AssessmentChallenge

建议：

```go
type AssessmentChallenge struct {
    ID uint `gorm:"primaryKey"`

    CourseID uint `gorm:"not null;index"`
    LessonID uint `gorm:"not null;index"`

    ChallengeType string `gorm:"size:32;not null;index"`

    TargetMisconceptionID *uint `gorm:"index"`

    Prompt string `gorm:"type:text;not null"`
    ScenarioContext string `gorm:"type:text"`
    EvaluationCriteriaJSON string `gorm:"type:text;not null"`

    TargetLevel string `gorm:"size:32;not null"`

    Status string `gorm:"size:32;not null;default:'pending'"`

    Provider string `gorm:"size:64"`
    Model string `gorm:"size:128"`
    PromptVersion string `gorm:"size:128"`

    CreatedAt time.Time
    UpdatedAt time.Time
}
```

ChallengeType：

```text
transfer
misconception_recheck
```

Status：

```text
pending
answered
cancelled
```

一个 Challenge 成功提交一次正式答案后变为 `answered`。重试应生成新的 Challenge。

---

# 四、新增 ChallengeAttempt

建议：

```go
type ChallengeAttempt struct {
    ID uint `gorm:"primaryKey"`

    CourseID uint `gorm:"not null;index"`
    LessonID uint `gorm:"not null;index"`
    ChallengeID uint `gorm:"not null;uniqueIndex"`

    LearningTurnID uint `gorm:"not null;index"`

    UserAnswer string `gorm:"type:text;not null"`

    Result string `gorm:"size:32;not null"`
    DemonstratedLevel string `gorm:"size:32;not null"`

    Passed bool `gorm:"not null;default:false"`

    Feedback string `gorm:"type:text"`
    EvidenceJSON string `gorm:"type:text"`
    MisconceptionValidationJSON string `gorm:"type:text"`

    CreatedAt time.Time
}
```

建议给 LearningTurn 增加：

```text
TurnKind
ChallengeID
```

TurnKind：

```text
lesson_answer
transfer_challenge
misconception_recheck
```

如果现有结构已有等价字段则复用。

---

# 五、Challenge 生成 API

新增：

```http
POST /api/v1/courses/:id/lessons/:lessonId/challenges
```

请求：

```json
{
  "challenge_type": "transfer"
}
```

或：

```json
{
  "challenge_type": "misconception_recheck",
  "misconception_id": 12
}
```

要求：

- Course 存在
- Lesson 属于 Course
- transfer 不要求 TargetMisconceptionID
- misconception_recheck 必须带 active misconception
- misconception 必须属于相同 Course / Lesson
- 不允许为 resolved misconception 直接生成 correction challenge
- 不改变 CurrentLessonID
- 不改变 Curriculum Graph

---

# 六、Transfer Challenge 生成原则

Transfer Challenge 必须是：

> **使用同一个核心知识，但放到明显不同于原 CoreQuestion 的新情境中。**

不能只是原题同义改写。

例如原问题：

```text
只要不口渴，是否说明身体不缺水？
```

新的 Transfer Challenge 可以是：

```text
一名老人在高温天气散步后表示自己并不口渴，
家人因此认为无需关注饮水。

这个判断有什么问题？你会结合哪些信息判断？
```

要求：

- 新场景
- 不直接泄露答案
- 不要求课程中没有提供的专业背景
- 主要测试目标 Lesson 的核心知识
- 可利用 prerequisite，但不要依赖未学 extension
- 单次回答 2~5 分钟
- 不做医学诊断
- 不生成高风险个体化处方

---

# 七、Transfer Challenge TargetLevel

正式 Transfer Challenge：

```text
TargetLevel = transfer
```

它是独立 Assessment，因此不受 `Lesson.AssessmentTargetLevel=understand` 的上限限制。

普通 Lesson Answer 仍受 Lesson AssessmentTargetLevel 限制。

---

# 八、Transfer Generator Prompt

新增：

```text
learnos-transfer-generator-v1
```

AI 输入至少包括：

```text
Course
Unit
Lesson Title
CoreQuestion
ExpectedUnderstanding
Prerequisite Lesson Titles
必要的 Relation Context
Current Cognitive Level
Current Understanding Summary
Active Misconceptions
```

输出：

```json
{
  "prompt": "……",
  "scenario_context": "……",
  "target_level": "transfer",
  "evaluation_criteria": [
    "能够识别不能只用单一信号判断",
    "能够主动结合场景条件判断",
    "能够说明结论边界"
  ],
  "why_this_is_transfer": "该任务把原知识迁移到不同人群和环境场景中。",
  "source_concepts": [
    "口渴信号的局限",
    "场景因素"
  ]
}
```

Go 校验：

- prompt 非空
- target_level=transfer
- evaluation_criteria 1~6
- why_this_is_transfer 非空
- source_concepts 数量合理
- 不允许 AI 修改 course/lesson metadata
- 非法结果最多重试一次
- 最终非法则不创建 Challenge

---

# 九、Challenge Answer API

新增：

```http
POST /api/v1/courses/:id/challenges/:challengeId/answers
```

请求：

```json
{
  "answer": "……"
}
```

要求：

- Challenge 属于 Course
- status=pending
- answer 非空
- 最大长度沿用当前 Answer 限制
- 不允许重复成功提交
- 成功后 Challenge → answered

---

# 十、Challenge Evaluator Prompt

新增：

```text
learnos-challenge-evaluator-v1
```

不要直接复用普通 Lesson evaluator。

Transfer 输出示例：

```json
{
  "result": "mostly_correct",
  "feedback": "...",
  "explanation": "...",
  "demonstrated_level": "transfer",
  "cognitive_evidence": [
    {
      "evidence_type": "transfer",
      "cognitive_level": "transfer",
      "polarity": "support",
      "description": "..."
    }
  ],
  "misconceptions": [],
  "misconception_validation": null
}
```

Misconception Recheck 输出：

```json
{
  "result": "correct",
  "demonstrated_level": "understand",
  "cognitive_evidence": [],
  "misconception_validation": {
    "target_misconception_id": 12,
    "status": "corrected",
    "evidence": "用户在新问题中明确否定了原先的绝对化判断，并给出了正确边界。"
  }
}
```

---

# 十一、Transfer Pass 规则

只有同时满足：

```text
result = correct 或 mostly_correct
demonstrated_level = transfer
至少一条 transfer/support evidence
不存在 contradiction evidence
```

才：

```text
Passed = true
```

Passed 由 Go 计算，不由 AI 直接决定。

如果 Passed：

```text
CurrentLevel → transfer（如原等级更低）
Status → stable
```

并保存真实 transfer CognitiveEvidence。

如果 Failed：

```text
CurrentLevel 默认不降级
Status 默认不变
```

只有出现 contradiction 或 active misconception 再次出现时才 `needs_review`。

---

# 十二、Misconception Recheck

生成时必须输入：

```text
OriginalUnderstanding
CorrectUnderstanding
BoundaryNotes
Current UnderstandingSummary
```

但给用户的问题不能直接泄露 CorrectUnderstanding。

Validation 只允许：

```text
corrected
persists
unclear
```

只有同时满足：

```text
ChallengeType = misconception_recheck
TargetMisconceptionID 匹配
result = correct 或 mostly_correct
validation.status = corrected
不存在同一目标误区 contradiction
```

才允许：

```text
Misconception.Status = resolved
```

并保存：

```text
ResolvedAt
ResolvedByLearningTurnID
ResolvedByChallengeID
```

如果 `persists`：

```text
Status 保持 active
CognitiveState.Status = needs_review
```

如果 `unclear`：

```text
Misconception 仍 active
CurrentLevel 不降
CognitiveState.Status 默认保持
```

---

# 十三、Misconception 再次出现要 reopen

当新 Evaluation 发现 Misconception：

按：

```text
CourseID
LessonID
trimmed OriginalUnderstanding
trimmed CorrectUnderstanding
```

查历史，不区分 status。

如果 active：复用。

如果 resolved：

```text
Status → active
ResolvedAt / ResolvedBy... 清空
```

并记录 reopen。

不要创建重复 Misconception Row。

---

# 十四、新增 MisconceptionEvent

建议：

```go
type MisconceptionEvent struct {
    ID uint `gorm:"primaryKey"`

    CourseID uint `gorm:"not null;index"`
    LessonID uint `gorm:"not null;index"`
    MisconceptionID uint `gorm:"not null;index"`

    LearningTurnID *uint `gorm:"index"`
    ChallengeID *uint `gorm:"index"`

    EventType string `gorm:"size:32;not null"`
    Notes string `gorm:"type:text"`

    CreatedAt time.Time
}
```

EventType：

```text
observed
resolved
reopened
```

---

# 十五、Reasoning Pattern Taxonomy

Phase 6 不做 Embedding 聚类，使用有限可解释 taxonomy：

```text
binary_thinking
single_factor_reasoning
overgeneralization
boundary_neglect
dose_neglect
correlation_causation
category_confusion
unsupported_assumption
```

中文：

```text
二元化判断
单因素归因
过度泛化
忽略适用边界
忽略剂量 / 程度
相关与因果混淆
概念 / 类别混淆
无依据假设
```

每个 Misconception：

```text
允许 0~2 个 Pattern
```

不强迫分类。

---

# 十六、新增 MisconceptionPatternLink

建议：

```go
type MisconceptionPatternLink struct {
    ID uint `gorm:"primaryKey"`

    MisconceptionID uint `gorm:"not null;index"`
    PatternKey string `gorm:"size:64;not null;index"`

    Explanation string `gorm:"type:text"`

    Source string `gorm:"size:32;not null"`
    PromptVersion string `gorm:"size:128"`

    CreatedAt time.Time
}
```

唯一：

```text
MisconceptionID + PatternKey
```

---

# 十七、普通 Lesson Evaluator 升级 v3

现有：

```text
learnos-evaluator-v2
```

升级：

```text
learnos-evaluator-v3
```

保留 Phase 5 全部字段，只在新 Misconception 中增加：

```json
{
  "reasoning_patterns": [
    {
      "pattern_key": "binary_thinking",
      "explanation": "用户把多因素问题判断成绝对的是/否。"
    }
  ]
}
```

Go 严格校验 PatternKey。

旧 Phase 3/5 misconception 不自动 AI backfill。

---

# 十八、误区网络 API

新增：

```http
GET /api/v1/courses/:id/misconception-network
```

返回：

```json
{
  "data": {
    "patterns": [
      {
        "key": "binary_thinking",
        "name": "二元化判断",
        "active_count": 2,
        "resolved_count": 1
      }
    ],
    "misconceptions": [
      {
        "id": 12,
        "lesson_id": 3,
        "lesson_title": "口渴是否是可靠的饮水依据",
        "original_understanding": "...",
        "correct_understanding": "...",
        "status": "active",
        "pattern_keys": ["binary_thinking"],
        "occurrence_count": 2
      }
    ],
    "edges": [
      {
        "pattern_key": "binary_thinking",
        "misconception_id": 12
      }
    ]
  }
}
```

OccurrenceCount 可由 MisconceptionEvent 计算。

如当前没有 Lesson misconception API，再增加：

```http
GET /api/v1/courses/:id/lessons/:lessonId/misconceptions
```

---

# 十九、误区网络页面

新增：

```text
/courses/:id/misconceptions
```

Phase 6 只做逻辑网络，不做最终 Graph Canvas。

按 Pattern 分组：

```text
二元化判断
2 个活跃误区

├─ 口渴：不口渴 = 一定不缺水
└─ ...
```

点击 Misconception 显示：

```text
原理解
正确理解
边界
出现次数
状态
底层模式
事件历史

[验证修正]
```

---

# 二十、LearningView：迁移挑战

当前 Lesson 页面增加：

```text
迁移验证
```

若：

```text
CurrentLevel < understand
```

显示：

```text
建议先形成稳定理解后再做迁移测试
```

可禁用按钮。

当：

```text
CurrentLevel >= understand
```

允许：

```text
生成迁移挑战
```

这只是 Assessment Eligibility，不是自动解锁课程路线。

---

# 二十一、Transfer UI

生成后显示：

```text
迁移挑战

新场景：
……

[回答输入框]
[提交挑战]
```

提交后：

```text
迁移已验证
```

或：

```text
迁移尚未验证
```

展示：

```text
Feedback
Explanation
CognitiveEvidence
```

`why_this_is_transfer` 建议在答题后展示，避免提示过多。

---

# 二十二、误区修正 UI

Active Misconception 显示：

```text
待修正

原理解：
……

正确方向：
……

[验证修正]
```

点击后生成 `misconception_recheck`。

通过后：

```text
已修正 ✓
```

事件历史仍保留。

---

# 二十三、Knowledge Map Overlay

Phase 5 已有：

```text
理解 · 稳定
```

如果 CurrentLevel=transfer：

```text
迁移 · 稳定 ★
```

或：

```text
迁移已验证
```

不要引入复杂图形库。

---

# 二十四、事务一致性

合法 Challenge Evaluation 后事务写入：

```text
LearningTurn
ChallengeAttempt
Challenge.Status
CognitiveEvidence
CognitiveState
CognitiveStateEvent
Misconception update（如适用）
MisconceptionEvent
MisconceptionPatternLink（如有新误区）
AI audit linkage
```

任意正式写失败：

> 不允许留下半成功 Attempt。

AI 调用仍在事务外。

---

# 二十五、AI 审计

不要破坏现有 AIEvaluationRun。

Challenge Generation 至少应可追踪：

```text
provider
model
prompt_version
latency
status
validation_error
```

可以新增 AIChallengeRun，或对现有审计体系增加清晰 RunType。

选择侵入最小、语义清楚的设计。

不得记录 API Key / Authorization。

---

# 二十六、严格禁止错误升级

只有：

```text
正式 transfer Challenge
+
Passed
```

才能产生 `transfer` CognitiveState。

普通 Lesson evaluator-v3 仍受 AssessmentTargetLevel 限制。

Curriculum Graph 中的：

```text
related
extends
application
```

也不能自动产生 transfer evidence。

---

# 二十七、测试要求

所有测试不得调用真实 DeepSeek。

至少覆盖：

## Transfer Generator
1. 合法 Challenge
2. TargetLevel=transfer
3. criteria 1~6
4. invalid AI response 不持久化
5. AI failure 不创建 Challenge

## Challenge API
6. lesson 不属于 course
7. pending 可回答
8. answered 不可重复提交
9. empty answer 拒绝

## Transfer Pass
10. correct + transfer evidence → pass
11. mostly_correct + transfer evidence → pass
12. correct 但只有 understand evidence → fail
13. partially_correct → fail
14. contradiction → fail

## CognitiveState
15. understand + transfer pass → transfer/stable
16. transfer fail → level/status 默认不变
17. transfer fail + contradiction → needs_review
18. fail 不自动降级

## Recheck
19. corrected + correct → resolved
20. corrected + incorrect → 不 resolved
21. persists → active + needs_review
22. unclear → active，level 不降
23. misconception id mismatch → invalid

## Lifecycle
24. new misconception → observed event
25. active 再出现 → 不重复 row
26. resolved 再出现 → reopen 原 row
27. reopen event
28. resolved event

## Pattern
29. allowed PatternKey
30. unknown PatternKey 拒绝
31. 同 link 不重复
32. 允许 0 pattern
33. 最多 2 pattern

## Network
34. active/resolved count
35. occurrence_count
36. edges
37. course 隔离

## Transaction
38. Attempt 写失败整事务回滚
39. State 写失败不标 Challenge answered
40. resolve 写失败整事务回滚

## Legacy
41. Phase 1~5 历史继续存在
42. 旧 misconception 不自动 backfill
43. 普通 Phase 5 lesson answer 正常
44. 普通 Lesson 不能升级 transfer

---

# 二十八、旧数据兼容

继续 GORM AutoMigrate。

不得删除现有：

```text
Course
Lesson
LessonRelation
LearningTurn
MasteryRecord
Misconception
AIEvaluationRun
CognitiveState
CognitiveEvidence
CognitiveStateEvent
```

新增建议：

```text
assessment_challenges
challenge_attempts
misconception_events
misconception_pattern_links
```

以及必要字段。

禁止清空 SQLite。

---

# 二十九、文档更新

更新：

- README.md
- docs/PRODUCT.md
- docs/ARCHITECTURE.md
- docs/DATABASE.md
- docs/ROADMAP.md

ROADMAP：

```text
Phase 1 ✅
Phase 2 ✅
Phase 3 ✅
Phase 4 ✅
Phase 5 ✅
Phase 6 ✅（完成后）
```

---

# 三十、Phase 6 明确不做

不要实现：

- 探索雷达
- 陌生知识按钮
- 问题池
- AI 自动选择下一 Lesson
- Agent 导航
- 自动课程路径
- 自动 Curriculum 修改
- AI 自动新增知识节点
- 知识来源可信度 / Citation
- 在线搜索
- RAG
- Vector DB
- Embedding misconception clustering
- 自由生成 reasoning taxonomy
- 遗忘曲线
- 间隔重复
- 自动复习调度
- 全局 Hero's Path
- 最终 Graph Canvas
- D3 / Cytoscape / Vue Flow
- 多用户
- Obsidian 同步
- 大范围 UI 重构

---

# 三十一、Phase 6 完成标准

必须同时满足：

1. 能生成真正新场景的 Transfer Challenge。
2. 能回答并经过 AI 结构化评价。
3. Go 规则决定 Pass，不由 AI 直接决定。
4. 只有通过的 Transfer Challenge 才产生 transfer evidence。
5. 迁移失败不会自动抹掉已有 understand。
6. Misconception 有 observed / resolved / reopened 生命周期。
7. 普通后来答对不会直接 resolved。
8. targeted recheck 成功才能 resolved。
9. 多个具体误区可以连接到底层 Reasoning Pattern。
10. 页面可以查看误区网络和修正状态。
11. Phase 1~5 历史与功能全部继续正常。

---

# 三十二、人工验收流程

Codex 完成后必须给出：

```text
A. Transfer

1. docker compose up -d --build app
2. 打开“口渴是否是可靠的饮水依据”
3. 确认 CognitiveState >= understand
4. 生成迁移挑战
5. 确认不是原问题同义改写
6. 正确回答
7. 确认 Passed
8. 确认 transfer evidence
9. 确认 CurrentLevel=transfer / stable
10. Knowledge Map 显示迁移已验证

B. Transfer Failure

11. 再生成一个 Challenge
12. 回答“不知道”
13. 确认 CurrentLevel 不下降
14. 无 contradiction 时 Status 不被无理由改 needs_review

C. Misconception

15. 普通 Lesson 故意回答：
    “只要不口渴就说明一定不缺水。”
16. 确认 active misconception
17. 打开误区网络
18. 确认连接到合理 Reasoning Pattern

D. Correction

19. 点击“验证修正”
20. 生成 targeted recheck
21. 正确回答
22. Misconception → resolved
23. resolved event 存在

E. Reopen

24. 普通 Lesson 再次明确表达同一误区
25. 不创建重复 Misconception
26. 原记录 → active
27. reopened event 存在

F. Persistence

28. Ctrl + F5
29. docker compose restart app
30. Challenge / Attempt / transfer evidence / misconception / pattern / event 全部仍在
```

---

# 三十三、构建验证

至少执行：

```bash
go test ./...
go build ./cmd/server
npm run build
docker compose build app
git diff --check
```

Go 执行 gofmt。

不得删除测试换取通过。

---

# 三十四、Codex 工作顺序

1. 阅读仓库
2. 汇报 Phase 5 实际结构
3. 列 Phase 6 文件清单与方案
4. AssessmentChallenge / ChallengeAttempt
5. Misconception 生命周期字段
6. MisconceptionEvent
7. MisconceptionPatternLink
8. migration
9. evaluator-v3
10. transfer-generator-v1
11. challenge-evaluator-v1
12. Challenge Validation
13. Transfer Pass Logic
14. Correction Validation
15. Misconception reopen / resolve
16. Repository / Service
17. API
18. 后端测试
19. LearningView Transfer UI
20. Misconception Network 页面
21. Knowledge Map Transfer Overlay
22. 完整测试 / 构建
23. 修复
24. 文档
25. 最终报告

最终报告必须包含：

```text
Phase 6 完成内容
修改文件
Challenge 模型
Transfer 生成规则
Transfer Pass 规则
失败 Challenge 对 CognitiveState 的规则
Misconception Lifecycle
Correction Validation
Reasoning Pattern taxonomy
误区网络数据结构
Prompt Versions
数据库变化
API
UI
事务设计
旧数据兼容
测试结果
Docker 构建结果
人工验收步骤
明确未实现的 Phase 7+ 内容
```

现在开始。

先不要立即编码。

请先阅读仓库，并向我汇报：

1. Phase 5 当前实际架构
2. Phase 6 文件修改清单
3. AssessmentChallenge / ChallengeAttempt 模型
4. Misconception 生命周期设计
5. Reasoning Pattern 网络设计
6. Transfer Challenge 生成与判定规则
7. Correction Validation 规则
8. 对 CognitiveState 的影响规则
9. 旧数据兼容方案
10. 简短实施顺序

确认与现有代码兼容后，再开始实现。
