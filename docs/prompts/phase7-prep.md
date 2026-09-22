# LearnOS Phase 7 前置：测试知识世界扩充

请在当前 LearnOS 仓库中完成一个 **Phase 7 前置任务：测试知识世界扩充**。

这不是新的正式 Phase，不改变现有 ROADMAP，也不要把它标记为 Phase 7 已完成。

当前已完成：
- Phase 1：项目骨架
- Phase 2：持久化学习闭环
- Phase 3：DeepSeek AI 结构化认知评价
- Phase 4：知识世界 / Curriculum Graph
- Phase 5：个人认知状态 / Evidence / Cognitive Evolution
- Phase 6：迁移测试 / 误区生命周期 / Reasoning Pattern / 修正验证

Phase 7 将正式实现：
- 探索雷达
- 陌生知识入口
- 问题池

但当前测试世界过于单一。如果直接做 Phase 7，很容易退化成“喝水 → 更多喝水知识”，无法验证真正的邻近探索、跨域连接和未知知识发现。

本任务只做一件事：

> **扩充测试知识世界，让 Phase 7 有足够丰富的测试素材。**

---

## 一、开始前

请先阅读：

- AGENTS.md
- README.md
- docs/PRODUCT.md
- docs/ARCHITECTURE.md
- docs/DATABASE.md
- docs/ROADMAP.md
- docs/prompts/phase4.md
- docs/prompts/phase5.md
- docs/prompts/phase6.md
- 当前 Course / CourseUnit / Lesson / LessonRelation
- KnowledgeGraphService
- CognitiveState / Misconception / Challenge
- Seed 逻辑
- Vue Knowledge Map / LearningView
- tests
- Dockerfile / docker-compose.yml

先汇报：
1. 当前 Course / Unit / Lesson 实际 Seed 情况
2. 当前营养学 8 个 Lesson 哪些已经具备完整学习字段
3. 当前 Seed 如何保证幂等
4. 本任务预计修改文件
5. 如何保护现有 Lesson ID 和 Phase 1~6 历史
6. 简短实施计划

确认兼容后再编码。

---

# 二、任务目标

完成后至少存在三个可用于 Phase 7 测试的知识岛：

```text
营养学
逻辑与科学思维
心理学
```

要求：

```text
营养学：现有 8 个节点全部真正可学习
逻辑与科学思维：3 个最小测试节点
心理学：3 个最小测试节点
```

不要构建完整课程，只需要形成多领域、多关系、可跨域连接的测试世界。

---

# 三、严格边界

只扩充静态课程内容：

```text
Course
CourseUnit
Lesson
LessonRelation
CoreQuestion
ExpectedUnderstanding
AssessmentTargetLevel
ContentRole
DepthLevel
IsCore
```

不要实现：
- 探索雷达
- 陌生知识按钮
- 问题池
- 推荐算法
- Agent
- AI 自动生成课程
- AI 修改 Knowledge Graph
- RAG / Vector DB
- 来源可信度 / Citation
- 在线搜索
- 新 CognitiveState 规则
- 新 Challenge 类型
- UI 大改

---

# 四、补全现有营养学 8 个 Lesson

必须复用已有 Lesson，禁止删除后重建。

目标 Lesson：

1. 水在人体中的基本作用
2. 体液平衡是如何维持的
3. 口渴是否是可靠的饮水依据
4. 日常饮水需求应该如何判断
5. 高温和运动为什么会改变补水需求
6. 大量出汗后为什么需要关注电解质
7. 年龄与疾病状态为什么会影响口渴信号
8. 如何综合判断不同场景下的补水策略

确保每个 Lesson 至少具备：

```text
Title
CoreQuestion
ExpectedUnderstanding
AssessmentTargetLevel
ContentRole
DepthLevel
IsCore
Status
```

建议内容如下。

### N1 水在人体中的基本作用

```text
ContentRole=foundation
DepthLevel=1
IsCore=true
AssessmentTargetLevel=understand
```

CoreQuestion：

```text
为什么人体不能简单把水理解成“解渴用的饮料”？水在身体里主要承担哪些作用？
```

ExpectedUnderstanding：

```text
水不仅用于缓解口渴，还参与体温调节、体液与血液运输、代谢反应、营养物质和代谢产物运输，以及维持细胞和组织正常环境。应把水理解为人体内部环境的重要组成部分，而不是单一饮用品。
```

### N2 体液平衡是如何维持的

```text
foundation / Depth 1 / Core / Target=understand
```

CoreQuestion：

```text
人体每天都在摄入和排出水分，为什么体内水分通常还能维持在相对稳定的范围？
```

ExpectedUnderstanding：

```text
人体通过摄入、尿液、汗液、呼吸等途径持续发生水分交换，并通过口渴、肾脏调节和相关激素机制维持动态平衡。体液平衡不是静止状态，而是持续调节。
```

### N3 口渴是否是可靠的饮水依据

保留当前成熟 CoreQuestion / ExpectedUnderstanding 和历史关联，不要无必要覆盖。

```text
Target=understand
```

### N4 日常饮水需求应该如何判断

```text
core / Depth 2 / Core / Target=understand
```

CoreQuestion：

```text
日常生活中，能不能用一个固定饮水量适用于所有人？判断自己是否需要补水时应该考虑哪些因素？
```

ExpectedUnderstanding：

```text
饮水需求受体型、饮食、环境温湿度、活动量、出汗和身体状态等多因素影响，不存在适用于所有人所有场景的唯一固定量。判断应综合口渴、实际摄入、排尿、活动和环境等信息。
```

### N5 高温和运动为什么会改变补水需求

```text
application / Depth 2 / Core / Target=apply
```

CoreQuestion：

```text
一个人在炎热天气运动后，即使和平时喝了同样多的水，为什么仍可能出现水分不足？
```

ExpectedUnderstanding：

```text
高温和运动会提高体温调节需求并增加出汗，使水分损失增加，因此日常静息状态下合适的饮水量可能不足，需要结合环境、强度和出汗动态调整。
```

### N6 大量出汗后为什么需要关注电解质

```text
application / Depth 3 / Core / Target=apply
```

CoreQuestion：

```text
大量出汗以后为什么有时只补大量白水并不是最完整的补水思路？
```

ExpectedUnderstanding：

```text
汗液不仅带走水，也会带走钠等电解质。在长时间、大量出汗场景中，仅补水可能不足以恢复水和电解质平衡；但普通日常活动并不意味着必须额外补电解质，应结合实际出汗程度和持续时间判断。
```

### N7 年龄与疾病状态为什么会影响口渴信号

确保：

```text
IsCore=false
ContentRole=extension
DepthLevel=3
AssessmentTargetLevel=understand
```

CoreQuestion：

```text
为什么同样处于水分不足状态，不同年龄或不同身体状态的人可能表现出不同程度的口渴？
```

ExpectedUnderstanding：

```text
口渴感受和水分调节会受到年龄、身体状态以及部分疾病或药物等因素影响，因此其敏感程度并非所有人都相同。重点是理解口渴有价值但存在个体和情境边界。
```

### N8 如何综合判断不同场景下的补水策略

```text
application / Depth 4 / Core / Target=apply
```

CoreQuestion：

```text
如果一个人既不明显口渴，又处在高温、运动或长时间低饮水等特殊场景中，应该怎样综合判断是否需要补水？
```

ExpectedUnderstanding：

```text
应综合环境、活动量、出汗、实际饮水、饮食、口渴、排尿和身体状态等多种信息，避免依赖单一指标，也避免从“不能只看口渴”走向“必须机械大量喝水”的另一个极端。
```

---

# 五、新增课程：逻辑与科学思维

Course：

```text
Name=逻辑与科学思维
Description=帮助识别常见推理错误、理解证据与因果关系，并建立更可靠的日常判断框架。
```

Unit：

```text
日常推理基础
```

只建 3 个 Lesson。

### L1 单因素解释的陷阱

```text
foundation / Depth 1 / Core / Target=understand
```

CoreQuestion：

```text
当一个现象发生时，为什么“找到一个可能原因”并不等于“已经解释了这个现象”？
```

ExpectedUnderstanding：

```text
现实问题通常可能受多个变量共同影响，一个因素与结果有关并不意味着它能单独解释全部结果。可靠判断需要考虑其他因素、条件和替代解释，避免把复杂问题简化成单一原因。
```

### L2 相关不等于因果

```text
core / Depth 2 / Core / Target=understand
```

CoreQuestion：

```text
如果两件事情经常同时出现，为什么不能直接认为其中一件导致了另一件？
```

ExpectedUnderstanding：

```text
相关只能说明变量共同变化，不能自动证明因果。可能存在反向因果、共同原因、选择偏差或偶然关系，因果判断需要额外证据。
```

### L3 如何判断一条证据有多可靠

```text
application / Depth 2 / Core / Target=apply
```

CoreQuestion：

```text
面对“有人亲身试过有效”和“多个较高质量研究得到一致结果”两类信息时，应该如何比较它们的可信度？
```

ExpectedUnderstanding：

```text
证据可靠性与样本数量、研究设计、偏差控制、可重复性、来源独立性和证据一致性有关。个人经验可提供线索，但更容易受偶然性和认知偏差影响。
```

Relation：

```text
L1 prerequisite L2
L1 prerequisite L3
L2 related L3
```

---

# 六、新增课程：心理学

Course：

```text
Name=心理学
Description=从认知、判断与行为角度理解人的心理过程，以及这些过程如何影响日常选择。
```

Unit：

```text
认知与行为基础
```

只建 3 个 Lesson。

### P1 确认偏误

```text
foundation / Depth 1 / Core / Target=understand
```

CoreQuestion：

```text
为什么人在已经相信一个观点以后，往往更容易注意到支持它的信息，而忽略反对它的信息？
```

ExpectedUnderstanding：

```text
确认偏误指人更容易寻找、注意、解释和记住支持既有信念的信息，同时低估冲突证据。它是常见认知倾向，并不意味着人是在故意欺骗自己。
```

### P2 情绪如何影响判断

```text
core / Depth 2 / Core / Target=understand
```

CoreQuestion：

```text
为什么人在焦虑、愤怒或兴奋时，面对同一个问题可能做出和平静时不同的判断？
```

ExpectedUnderstanding：

```text
情绪会影响注意、风险感知、信息解释和决策权重，因此同一信息在不同情绪状态下可能被赋予不同意义。情绪包含信息，但不能自动等同于客观事实。
```

### P3 习惯为什么会自动发生

```text
application / Depth 2 / Core / Target=apply
```

CoreQuestion：

```text
为什么有些行为明明没有经过认真决定，却会在固定时间或场景中自动发生？
```

ExpectedUnderstanding：

```text
重复行为会逐渐与环境线索、时间、情绪或上下文形成稳定联系，使行为启动越来越依赖线索。改变习惯不仅依靠意志力，也可通过调整线索、环境和替代行为实现。
```

Relation：

```text
P1 related P2
P2 related P3
```

不要人为制造严格 prerequisite。

---

# 七、跨域测试关系

Phase 7 需要测试跨域连接，但不要破坏 Phase 4 的 Course 内 DAG。

如果现有 `LessonRelation` 严格限制同一 Course，则评估是否新增轻量：

```text
CrossCourseLessonRelation
```

建议字段：

```text
ID
FromCourseID
FromLessonID
ToCourseID
ToLessonID
RelationType
Reason
CreatedAt
```

只允许：

```text
related
reasoning_link
```

禁止跨 Course prerequisite。

最多 Seed 以下 3 条：

### 关系 1

```text
营养学：日常饮水需求应该如何判断
→
逻辑与科学思维：单因素解释的陷阱

type=reasoning_link
```

Reason：

```text
饮水需求受多个因素影响，单纯依赖口渴或固定饮水量属于典型单因素判断风险。
```

### 关系 2

```text
营养学：口渴是否是可靠的饮水依据
→
逻辑与科学思维：如何判断一条证据有多可靠

type=related
```

Reason：

```text
判断口渴、尿色等身体信号时，需要理解单一观察指标的证据边界。
```

### 关系 3

```text
逻辑与科学思维：如何判断一条证据有多可靠
→
心理学：确认偏误

type=reasoning_link
```

Reason：

```text
人在评价证据时可能优先接受支持既有观点的信息，因此证据判断与确认偏误存在直接联系。
```

如果新增 CrossCourseLessonRelation 会引发明显架构重构，则本任务可以暂不实现，只在最终报告说明由 Phase 7 正式设计。

---

# 八、Seed 必须幂等并保护历史

要求：

- 三门 Course 不重复
- Unit 不重复
- Lesson 不重复
- Relation 不重复
- 不改变已有营养学 Lesson ID
- 特别保护“口渴是否是可靠的饮水依据”
- 不删除任何 Phase 1~6 数据
- 不创建假学习记录

禁止：

```text
DELETE
TRUNCATE
重建数据库
清空 data/
```

不得 Seed：

```text
LearningTurn
MasteryRecord
CognitiveState
CognitiveEvidence
Misconception
Challenge
```

新知识节点应该自然显示：

```text
unseen / unknown
```

---

# 九、UI 最小兼容

完成后课程列表至少能看到：

```text
营养学
逻辑与科学思维
心理学
```

新 Course 默认显示未开始 / 0% 或现有等价状态，不伪造进度。

每个 Course 都应该能进入：

```text
/courses/:id/map
```

每个新增 Lesson 都必须可以进入现有 LearningView：

```text
显示 CoreQuestion
提交回答
AI Evaluation
CognitiveState
History
```

不需要用户现在真的学习。

新 Course 可以把第一个 Lesson 设为 CurrentLesson，但不得修改营养学当前 Lesson。

---

# 十、测试要求

至少覆盖：

### Nutrition
1. 8 个现有 Lesson 都有 CoreQuestion
2. 都有 ExpectedUnderstanding
3. 都有 AssessmentTargetLevel
4. 原 Lesson ID 不变
5. 原历史不删除

### Logic
6. Course/Unit/3 Lessons 幂等
7. Relation 正确

### Psychology
8. Course/Unit/3 Lessons 幂等
9. Relation 正确

### Cross-domain（如果实现）
10. Relation 幂等
11. 禁止 cross-course prerequisite
12. Reason 非空

### Cognitive Safety
13. 新 Lesson 无 CognitiveState 时为 unseen/unknown
14. Seed 不创建假 LearningTurn
15. Seed 不创建假 MasteryRecord
16. Seed 不创建假 CognitiveEvidence

### Compatibility
17. Phase 6 Challenge 数据仍在
18. Phase 5 CognitiveState 仍在
19. Phase 4 Knowledge Graph 正常
20. Phase 3 AI Evaluation 正常

---

# 十一、文档

更新：

- README.md
- docs/PRODUCT.md
- docs/ARCHITECTURE.md
- docs/DATABASE.md
- docs/ROADMAP.md

不要标记：

```text
Phase 7 ✅
```

记录为：

```text
Phase 7 preparation / test knowledge world expansion ✅
Phase 7 Exploration Engine：待开始
```

---

# 十二、人工验收

完成后给出测试步骤：

```text
1. docker compose up -d --build app

2. 打开课程列表
3. 确认出现：
   营养学
   逻辑与科学思维
   心理学

4. 打开营养学 Knowledge Map
5. 确认原 8 个 Lesson 和历史不变
6. 逐个查看，确认都有 CoreQuestion

7. 打开逻辑课程
8. 确认 3 个节点及关系

9. 打开心理学
10. 确认 3 个节点及关系

11. 新节点全部显示“未接触”

12. 随便进入一个新 Lesson
13. 确认可以使用现有学习闭环

14. Ctrl + F5
15. docker compose restart app

16. 三门 Course 与结构仍然存在
17. Phase 1~6 历史仍然存在
```

---

# 十三、完成标准

本任务只有满足以下条件才算完成：

```text
营养学 8 个节点全部可学习
+
逻辑与科学思维 3 个节点
+
心理学 3 个节点
+
Seed 幂等
+
不伪造认知数据
+
不破坏 Phase 1~6 历史
```

Phase 7 Exploration Engine 尚未开始。

---

# 十四、构建验证

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

# 十五、Codex 工作顺序

1. 阅读仓库
2. 汇报当前 Seed / Course / Lesson 状态
3. 列修改文件
4. 补全营养学 8 Lesson
5. 新增逻辑 Course / Unit / Lessons / Relations
6. 新增心理学 Course / Unit / Lessons / Relations
7. 评估 CrossCourseLessonRelation 是否值得在本任务实现
8. 完成幂等 Seed
9. 后端测试
10. 确认现有 UI 自动兼容
11. 必要时最小 UI 修正
12. 完整测试 / 构建
13. 更新文档
14. 最终报告

最终报告必须包含：

```text
完成内容
修改文件
三门课程结构
营养学补全情况
逻辑课程内容
心理课程内容
跨域 Relation 是否实现及原因
Seed 幂等策略
旧数据保护策略
测试结果
Docker 构建结果
人工验收步骤
明确说明 Phase 7 尚未开始
```

现在开始。

先不要立即编码。

请先阅读仓库并汇报：

1. 当前实际 Course / Unit / Lesson 数据结构
2. 营养学 8 个 Lesson 当前哪些字段缺失
3. 新增逻辑与心理 Course 的 Seed 方案
4. 是否有必要在本任务引入 CrossCourseLessonRelation
5. 如何保证现有 Lesson ID 与 Phase 1~6 历史不受影响
6. 修改文件清单
7. 简短实施顺序

确认兼容后再开始实现。
