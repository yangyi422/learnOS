# LearnOS Phase 5：个人认知状态、掌握证据与认知演化

请在当前 LearnOS 仓库中实现 **Phase 5：Personal Cognitive State**。

已完成并验收：

- Phase 1：项目骨架
- Phase 2：持久化学习闭环
- Phase 3：DeepSeek AI 结构化认知评价
- Phase 4：课程知识世界 / LessonRelation / prerequisite DAG / 基础知识结构页

Phase 5 要回答：

> 世界地图已经有了，那么“我”现在站在哪里？
>
> 系统凭什么认为我理解了某个知识？
>
> 我的理解从第一次接触到现在发生了什么变化？

本阶段建立：

```text
World State
+
User Cognitive State
+
Cognitive Evidence
+
Cognitive Evolution
```

不要提前实现自动路线推荐、迁移测试生成器、误区网络或 Agent 导航。

---

## 一、开始前

先完整阅读：

- AGENTS.md
- README.md
- docs/PRODUCT.md
- docs/ARCHITECTURE.md
- docs/DATABASE.md
- docs/ROADMAP.md
- docs/prompts/phase3.md
- docs/prompts/phase4.md
- 当前所有 model / repository / service / handler / router
- Course / CourseUnit / Lesson / LessonRelation
- LearningTurn / MasteryRecord / Misconception / AIEvaluationRun
- Phase 3 AIProvider / DeepSeekProvider / Prompt
- KnowledgeGraphService
- LearningView
- /courses/:id/map
- tests
- Dockerfile / docker-compose.yml

先输出：

1. Phase 4 实际架构
2. Phase 5 修改文件清单
3. CognitiveState / Evidence / Event 模型
4. Evaluation Prompt v2 Schema
5. State Transition 规则
6. 旧 Phase 2 / 3 历史兼容策略
7. 简短实施顺序

确认兼容后再编码，不要大范围重构。

---

# 二、核心原则

## World State 与 User State 分离

Phase 4：

```text
Lesson A prerequisite Lesson B
Lesson C 是 application
Lesson D 是 extension
```

这是知识本身。

Phase 5：

```text
我是否接触过？
我能否识别？
我能否解释？
我有没有实际应用证据？
系统为什么这么判断？
我的理解是否稳定？
```

这是个人认知状态。

不要把：

```text
Lesson.Status
ContentRole
DepthLevel
```

直接当用户认知状态。

---

# 三、认知层级

统一：

```text
unseen
exposed
recognize
understand
apply
transfer
```

含义：

```text
unseen
LearnOS 没有真实认知证据。

exposed
已经接触，但尚无有效理解证据。

recognize
能识别概念、核心判断或基本区别。

understand
能用自己的语言解释核心含义、机制或重要边界。

apply
能把知识用于具体案例或真实场景。

transfer
能把知识迁移到明显陌生、跨场景的问题。
```

注意：

> Phase 5 支持 transfer 数据结构，但不实现迁移测试生成器。

Phase 6 才专门做 Transfer Challenge。

---

# 四、CurrentLevel 不是最近一次分数

定义：

> `CurrentLevel` = 目前已经被证据验证过的最高认知层级。

不能：

```text
昨天 Understand
今天回答“不知道”
→ 直接降回 Exposed
```

应该：

```text
CurrentLevel = understand
Status = needs_review
```

因此认知等级与当前稳定状态必须分开。

---

# 五、认知状态 Status

支持：

```text
unknown
developing
stable
needs_review
```

规则：

```text
unseen
→ unknown

exposed / recognize
→ developing

understand / apply / transfer
且近期证据继续支持
→ stable

已有较高 Level，
但近期 incorrect / insufficient / contradiction
→ needs_review
```

本阶段不做遗忘曲线和自动降级。

---

# 六、Lesson 增加 AssessmentTargetLevel

为了防止 AI 对普通概念题声称用户已经 Apply / Transfer：

```go
AssessmentTargetLevel string `gorm:"size:32;not null;default:'understand'"`
```

允许：

```text
recognize
understand
apply
transfer
```

含义：

> 当前 Lesson/CoreQuestion 最多能提供哪一级认知证据。

建议 Demo：

```text
水在人体中的基本作用
understand

体液平衡是如何维持的
understand

口渴是否是可靠的饮水依据
understand

日常饮水需求应该如何判断
understand

高温和运动为什么会改变补水需求
apply

大量出汗后为什么需要关注电解质
apply

年龄与疾病状态为什么会影响口渴信号
understand

如何综合判断不同场景下的补水策略
apply
```

本阶段不要 Seed transfer target。

---

# 七、新增 CognitiveState

建议：

```go
type CognitiveState struct {
    ID uint `gorm:"primaryKey"`

    CourseID uint `gorm:"not null;index"`
    LessonID uint `gorm:"not null;uniqueIndex"`

    CurrentLevel string `gorm:"size:32;not null;default:'unseen'"`
    Status       string `gorm:"size:32;not null;default:'unknown'"`

    UnderstandingSummary string `gorm:"type:text"`

    LastLearningTurnID *uint `gorm:"index"`
    LastEvaluatedAt *time.Time

    CreatedAt time.Time
    UpdatedAt time.Time
}
```

`UnderstandingSummary` 表示：

> 从真实学习证据中，可以确认用户当前是怎样理解该知识的。

不是 AI 的标准答案摘要。

---

# 八、新增 CognitiveEvidence

建议：

```go
type CognitiveEvidence struct {
    ID uint `gorm:"primaryKey"`

    CourseID uint `gorm:"not null;index"`
    LessonID uint `gorm:"not null;index"`

    LearningTurnID uint `gorm:"not null;index"`
    EvidenceIndex int `gorm:"not null"`

    EvidenceType string `gorm:"size:32;not null"`
    CognitiveLevel string `gorm:"size:32;not null"`
    Polarity string `gorm:"size:16;not null"`

    Description string `gorm:"type:text;not null"`
    Source string `gorm:"size:32;not null"`

    CreatedAt time.Time
}
```

建议组合唯一：

```text
LearningTurnID + EvidenceIndex
```

EvidenceType：

```text
recognition
concept_explanation
boundary_awareness
application
transfer
contradiction
```

Polarity：

```text
support
contradict
```

任何认知等级都必须可以追溯到具体 Evidence。

---

# 九、Evidence 上限

必须在 Go 层验证：

```text
recognition
最多支持 recognize

concept_explanation
最多支持 understand

boundary_awareness
最多支持 understand

application
最多支持 apply

transfer
最多支持 transfer
```

例如：

```json
{
  "evidence_type": "recognition",
  "cognitive_level": "transfer"
}
```

必须拒绝。

不要相信 AI 一定遵守 Prompt。

---

# 十、新增 CognitiveStateEvent

为了记录：

> 我的理解怎么改变的

建议：

```go
type CognitiveStateEvent struct {
    ID uint `gorm:"primaryKey"`

    CourseID uint `gorm:"not null;index"`
    LessonID uint `gorm:"not null;index"`
    LearningTurnID uint `gorm:"not null;index"`

    FromLevel string `gorm:"size:32"`
    ToLevel string `gorm:"size:32"`

    FromStatus string `gorm:"size:32"`
    ToStatus string `gorm:"size:32"`

    UnderstandingSummary string `gorm:"type:text"`
    Reason string `gorm:"type:text"`

    CreatedAt time.Time
}
```

每次成功将 AI Evaluation 应用到认知状态，都创建 Event。

即使 Level 没升级，只要：

```text
Status
UnderstandingSummary
Evidence
```

发生了变化，也可以记录。

这就是第一版认知演化足迹。

---

# 十一、Phase 3 Evaluation Schema 升级 v2

不要删除 Phase 3 已有字段：

```text
result
feedback
explanation
correct_parts
missing_parts
misconceptions
boundary_conditions
mastery_evidence
mastery_score
needs_review
```

新增：

```json
{
  "demonstrated_level": "understand",
  "user_understanding_summary": "用户已经认识到口渴不是唯一补水依据，并能结合环境和出汗进行判断，但尚未完整覆盖年龄与疾病等边界。",
  "cognitive_evidence": [
    {
      "evidence_type": "recognition",
      "cognitive_level": "recognize",
      "polarity": "support",
      "description": "能够识别没有口渴并不等于一定不缺水"
    },
    {
      "evidence_type": "boundary_awareness",
      "cognitive_level": "understand",
      "polarity": "support",
      "description": "能够指出大量出汗和环境温度会改变补水需求"
    }
  ]
}
```

Prompt Version：

```text
learnos-evaluator-v2
```

历史 AIEvaluationRun 的 v1 不能被修改。

---

# 十二、demonstrated_level

AI 可以输出：

```text
exposed
recognize
understand
apply
transfer
```

不能输出：

```text
unseen
```

AssessmentTargetLevel 是硬上限。

例如：

```text
Target=understand
AI 返回 apply
```

应视为非法 AI Response，并按 Phase 3 规则最多重试一次。

不要无日志静默升级或 clamp。

---

# 十三、Result 对 Level 的业务上限

Go 层再次限制：

```text
insufficient
→ 最多 exposed

incorrect
→ 最多 exposed

partially_correct
→ 最多 recognize

mostly_correct
→ 最多 AI demonstrated_level，同时不能超过 Target

correct
→ 最多 AI demonstrated_level，同时不能超过 Target
```

AI 不是最终规则制定者。

---

# 十四、AI Prompt v2 认知评价规则

System Prompt 必须明确：

### recognize

只有识别、指出、区分，但缺少解释。

### understand

能够解释：

```text
含义
原因
机制
关系
边界
```

### apply

必须实际对：

```text
具体案例
真实问题
具体场景
```

使用知识。

不能因为回答长就判 Apply。

### transfer

必须：

```text
题目本身具有明显陌生 / 跨场景性质
+
用户成功迁移知识
```

不能因为回答优秀就判 Transfer。

---

# 十五、user_understanding_summary

它回答：

> 从用户这次回答里，能够确认用户当前是怎么理解这件事的？

不是知识标准答案。

例如用户回答：

```text
不能，只看口渴不够，还需要结合出汗、温度和身体状态。
```

Summary 可以是：

```text
用户已经认识到口渴只是补水判断的一种信号，并能结合环境、出汗和身体状态考虑补水需求。
```

不要写：

```text
用户完全掌握了水与体液平衡。
```

必须克制。

---

# 十六、Summary 更新规则

所有合法 Evaluation Summary 都进入：

```text
CognitiveStateEvent
```

但 `CognitiveState.UnderstandingSummary` 只保存：

> 最近一次有实质认知内容的 Summary。

如果用户回答：

```text
不知道
```

Event 可以记录：

```text
本次没有提供足够理解信息。
```

但不能覆盖之前已经存在的高质量 UnderstandingSummary。

---

# 十七、AI 不得伪造长期证据

Phase 5 Prompt 明确禁止 AI 声称：

```text
7 天后仍能记住
长期稳定掌握
反复验证正确
已经可以迁移
```

除非输入真实提供了这种证据。

Retention / spaced review 属于后续阶段。

---

# 十八、State Transition

Level Rank：

```text
unseen = 0
exposed = 1
recognize = 2
understand = 3
apply = 4
transfer = 5
```

本次 TurnLevel 大于 CurrentLevel：

```text
升级
```

小于等于：

```text
默认不降级
```

Status 根据最新真实表现变化。

例如：

```text
历史：
understand / stable

本次：
insufficient

结果：
understand / needs_review
```

再次正确：

```text
understand / stable
```

规则必须是可测试 Go 逻辑，不要完全交给 AI。

---

# 十九、事务一致性

一次正式学习成功：

```text
AI Evaluation v2
↓
Schema Validation
↓
Cognitive Validation
↓
DB Transaction
↓
LearningTurn
MasteryRecord
Misconception
CognitiveState
CognitiveEvidence
CognitiveStateEvent
AIEvaluationRun linkage
↓
Commit
```

AI 请求仍在事务外。

如果 CognitiveState / Evidence / Event 写入失败：

> 不允许留下半成功 LearningTurn 或 MasteryRecord。

---

# 二十、旧历史策略

当前数据库已经有：

- Phase 2 Mock 测试
- Phase 3 DeepSeek 验收数据
- 故意输入的错误答案
- “不知道”等测试内容

不要为了让 Phase 5 看起来有数据，就自动把旧历史推断为：

```text
Understand
Apply
Transfer
```

Phase 5 默认：

> 从 Phase 5 上线后的新 Evaluation 开始建立 CognitiveState。

旧 LearningTurn：

```text
继续保留
继续显示
不删除
不自动 backfill 为 CognitiveEvidence
```

未创建 CognitiveState 的 Lesson，查询时逻辑返回：

```text
current_level = unseen
status = unknown
```

GET 不应产生数据库副作用。

---

# 二十一、Repository / Service

新增 CognitiveState Repository：

```text
GetByLessonID
Upsert
ListByCourse
```

CognitiveEvidence：

```text
CreateMany
ListByLesson
ListByLearningTurn
```

CognitiveStateEvent：

```text
Create
ListByLesson
```

新增 CognitiveStateService：

```text
GetCourseCognitiveStates
GetLessonCognitiveState
ApplyEvaluationToCognitiveState
CalculateTurnLevel
ValidateCognitiveEvidence
```

Repository 不承担状态升级规则。

Handler 不直接操作 GORM。

---

# 二十二、API：课程认知状态

新增：

```http
GET /api/v1/courses/:id/cognitive-states
```

返回所有 Lesson：

```json
{
  "data": {
    "course_id": 1,
    "states": [
      {
        "lesson_id": 1,
        "current_level": "unseen",
        "status": "unknown",
        "understanding_summary": "",
        "evidence_count": 0
      }
    ]
  }
}
```

所有 Lesson 必须出现。

没有数据库 State 的 Lesson 合成 unseen。

---

# 二十三、API：Lesson 认知详情

新增：

```http
GET /api/v1/courses/:id/lessons/:lessonId/cognitive-state
```

返回：

```json
{
  "data": {
    "lesson": {
      "id": 3,
      "title": "口渴是否是可靠的饮水依据"
    },
    "state": {
      "current_level": "understand",
      "status": "stable",
      "understanding_summary": "...",
      "last_evaluated_at": "..."
    },
    "evidence": [],
    "timeline": []
  }
}
```

Timeline 默认最新 20 条，倒序。

不返回 Raw AI Response。

---

# 二十四、POST answers 保持兼容

继续使用：

```http
POST /api/v1/courses/:id/answers
```

成功响应在 Phase 3 基础上新增：

```json
{
  "demonstrated_level": "understand",
  "user_understanding_summary": "...",
  "cognitive_evidence": [],
  "cognitive_state": {
    "current_level": "understand",
    "status": "stable"
  }
}
```

不要删除 Phase 3 旧字段。

---

# 二十五、LearningTurn 快照

建议新增：

```text
DemonstratedLevel
UserUnderstandingSummary
CognitiveEvidenceJSON
```

读取历史时绝不能重新调用 DeepSeek。

---

# 二十六、Knowledge Map Personal Overlay

Phase 4 `/courses/:id/map` 不重做。

每个 Lesson 卡片只增加轻量：

```text
我的状态：
未接触
```

或：

```text
我的状态：
理解 · 稳定
```

支持中文映射：

```text
unseen → 未接触
exposed → 已接触
recognize → 识别
understand → 理解
apply → 应用
transfer → 迁移

unknown → 未知
developing → 发展中
stable → 稳定
needs_review → 待复习
```

Personal State 只是 Overlay，不修改 World Graph。

---

# 二十七、Knowledge Map 右侧详情

Phase 4 已显示：

```text
前置
后续
深化
应用
相关
```

Phase 5 增加：

```text
我的认知

当前层级：
Understand

状态：
Stable

当前理解：
……

掌握证据：
3 条

理解变化：
……
```

不用做最终漂亮时间轴。

---

# 二十八、LearningView

学习页增加：

```text
我的认知状态
```

展示：

```text
当前层级
状态
当前理解
具体 CognitiveEvidence
```

MasteryScore 仍可显示，但视觉优先级低于证据。

原则：

> 不要只告诉用户“70%”，要告诉用户为什么。

---

# 二十九、认知演化基础展示

展示 CognitiveStateEvent：

```text
2026-08-07

Unseen → Understand
Unknown → Stable

当时的理解：
……
```

下一次：

```text
Understand → Understand
Stable → Needs Review

当时表现：
……
```

这只是第一版认知足迹，不做全局 Hero's Path 动画。

---

# 三十、Phase 4 两个收尾检查

进入 Phase 5 前顺手确认：

1. `年龄与疾病状态为什么会影响口渴信号`

应为：

```text
IsCore=false
ContentRole=extension
Depth=3
```

前端显示：

```text
可选 | 扩展 | Depth 3
```

如果 Seed 错误，做最小幂等修正。

2. Relation UI 不应把 extends / application 全叫“可延伸方向”。

尽量明确：

```text
深化自 / 深化内容
应用自 / 应用场景
相关知识
```

只做小型收尾，不大改 UI。

---

# 三十一、Phase 5 不做

不要实现：

- AI 自动生成知识图
- 自动解锁
- 推荐下一课
- Agent 导航
- 探索雷达
- 陌生知识按钮
- 问题池
- 误区网络聚类
- 跨 Lesson misconception pattern mining
- 迁移测试生成器
- 自动 Transfer Challenge
- 遗忘曲线
- 自动认知降级
- 间隔重复
- 来源可信度系统
- RAG / Vector DB
- 在线搜索 / Citation
- 最终全局认知地图
- Hero's Path 动画
- 多用户
- Markdown / Obsidian 同步
- 大范围 UI 重构

---

# 三十二、测试要求

至少覆盖：

## Level / Status

1. unseen → exposed
2. exposed → recognize
3. recognize → understand
4. understand → apply
5. apply → transfer
6. 低水平新回答不会自动降 CurrentLevel
7. understand 后 incorrect → understand/needs_review
8. needs_review 后正确证据 → stable

## Evidence

9. recognition 最大 recognize
10. concept_explanation 最大 understand
11. boundary_awareness 最大 understand
12. application 最大 apply
13. transfer 最大 transfer
14. duplicate turn evidence 不重复
15. contradiction 正确保存

## Target

16. Target=understand 时 AI=apply → invalid
17. Target=apply 时 AI=understand → 合法
18. partially_correct 不能升级到 apply

## State

19. 首次成功评价创建 CognitiveState
20. UnderstandingSummary 正确保存
21. insufficient 不覆盖已有高质量 summary
22. LastEvaluatedAt 正确

## Event

23. 成功更新产生 CognitiveStateEvent
24. before/after level/status 正确
25. timeline 倒序

## Transaction

26. Evidence 写失败时不产生半成功 LearningTurn
27. State 写失败时不留下半成功 MasteryRecord
28. invalid AI response 不写正式 Cognitive 数据

## API

29. course cognitive-states 返回全部 Lesson
30. 无状态 Lesson 返回 unseen/unknown
31. lesson cognitive-state 正确
32. lesson 不属于 course 被拒绝
33. answer response 包含 cognitive_state
34. 不返回 raw AI response

## Legacy

35. Phase 2/3 历史不会自动 backfill 为高级认知等级
36. 旧 LearningTurn 继续正常显示
37. Phase 4 Knowledge Graph 不被破坏

所有测试不得调用真实 DeepSeek。

---

# 三十三、迁移与兼容

使用 GORM AutoMigrate。

新增：

```text
cognitive_states
cognitive_evidences
cognitive_state_events
```

必须保证：

- Phase 2 Mock 历史在
- Phase 3 DeepSeek 历史在
- Phase 4 Knowledge Graph 在
- Lesson ID 不变
- LessonRelation 不变
- AIEvaluationRun 不丢失
- Misconception 不丢失

禁止清库。

---

# 三十四、文档

更新：

- README.md
- docs/PRODUCT.md
- docs/ARCHITECTURE.md
- docs/DATABASE.md
- docs/ROADMAP.md

明确：

```text
World Graph ≠ User Cognitive State
```

ROADMAP：

```text
Phase 1 ✅
Phase 2 ✅
Phase 3 ✅
Phase 4 ✅
Phase 5 ✅（完成后）
```

Phase 6 才重点处理：

```text
迁移测试
误区网络
错误 → 修正 → 再验证
```

---

# 三十五、Phase 5 完成标准

必须同时满足：

1. 每个 Lesson 能返回 CurrentLevel / Status / UnderstandingSummary。
2. 无数据时真实显示 unseen / unknown。
3. 任何认知等级都能追溯到 CognitiveEvidence。
4. DeepSeek Prompt v2 返回 demonstrated_level / user_understanding_summary / cognitive_evidence。
5. Go 对 Level / Evidence / Target 严格校验。
6. AI 不能凭空声称 Apply / Transfer / Retention。
7. 每次认知更新产生 CognitiveStateEvent。
8. Knowledge Map 能看到 Personal State Overlay。
9. LearningView 能看到认知状态和证据。
10. 可以看到第一版“理解变化”。
11. Phase 2 / 3 / 4 历史全部保留。

---

# 三十六、人工验收

Codex 完成后给出以下测试流程：

```text
1. docker compose up -d --build app

2. 打开 /courses/1/map

3. 没有 Phase 5 新证据的 Lesson 应显示：
   未接触

4. 不自动拿 Phase 3 测试历史推断 Understand

5. 进入当前学习页

6. 回答：
   “不能。口渴只是一个信号，还要结合大量出汗、环境温度和身体状态判断。”

7. 提交

8. 检查：
   demonstrated_level
   CurrentLevel
   Status
   UnderstandingSummary
   CognitiveEvidence

9. 回到知识结构页
   当前节点状态应发生变化

10. 再回答：
    “不知道”

11. 检查：
    CurrentLevel 不直接降低
    Status → needs_review
    已有高质量 UnderstandingSummary 不被“信息不足”覆盖

12. 再给出正确答案
    Status 可恢复 stable

13. 查看理解变化
    能看到每次 Event

14. Ctrl + F5
15. docker compose restart app

16. CognitiveState / Evidence / Event 仍存在
17. Phase 3 历史仍在
18. Phase 4 Knowledge Graph 仍在
```

---

# 三十七、构建验证

至少执行：

```bash
go test ./...
go build ./cmd/server
npm run build
docker compose build app
git diff --check
```

Go 代码执行 gofmt。

不得删除测试换取通过。

---

# 三十八、Codex 工作顺序

1. 阅读仓库
2. 汇报 Phase 4 实际结构
3. 列出 Phase 5 文件修改清单
4. Phase 4 两个小型收尾
5. AssessmentTargetLevel
6. CognitiveState
7. CognitiveEvidence
8. CognitiveStateEvent
9. migration
10. Evaluation Prompt v2
11. DeepSeek Schema / Validation
12. CognitiveStateService
13. 正式学习事务整合
14. Repository
15. API
16. 后端测试
17. 前端 API 类型
18. LearningView 状态卡
19. Knowledge Map Personal Overlay
20. 认知变化基础展示
21. 测试 / 构建 / 修复
22. 文档
23. 最终报告

最终报告必须包含：

```text
完成内容
修改文件
认知层级定义
Status 定义
Evidence 类型
State Transition 规则
Prompt v2 变化
AssessmentTargetLevel
数据库变化
API
UI
事务一致性
旧历史兼容策略
测试结果
Docker 构建结果
人工验收步骤
明确未实现的 Phase 6+ 内容
```

现在开始。

先不要立即编码。

请先阅读仓库并向我汇报：

1. Phase 4 当前实际架构
2. Phase 5 文件清单
3. CognitiveState / Evidence / Event 模型
4. Evaluation Prompt v2 Schema
5. State Transition 规则
6. 为什么旧 Phase 2 / 3 历史不会污染 Phase 5 CognitiveState
7. 简短实施顺序

确认与现有代码兼容后，再开始实现。
