# LearnOS 产品定义

## 一句话定位

一个以用户控制的数据存储作为长期记忆，通过知识地图、状态机和可配置 Agent 工作流组织 LLM，从而实现可持续学习的个人 AI 系统。

知识结构页同时提供路径列表与 Knowledge Map Graph。地图默认展示 prerequisite DAG，并可叠加 extends、application、related 关系；正式 Lesson、当前 Lesson、认知状态和尚未生成的待生成节点使用不同的轻量视觉状态。浏览节点不会改变学习主线，只有明确选择“开始学习 / 继续学习”才会切换 CurrentLesson。

## 要解决的问题

普通 LLM 长对话存在以下限制：

- 历史最终超过上下文窗口；
- 重要状态淹没在聊天记录里；
- 新会话或更换模型后难以延续；
- 课程容易重复、偏离或遗漏；
- 掌握度、误区、复习和实践缺少结构化管理。

LearnOS 将长期记忆、课程状态和流程控制放在程序中，按当前任务向模型提供精简上下文。

## MVP 验证目标

以现有营养学课程作为第一门真实课程，跑通：

1. 读取当前课程状态；
2. 展示当前问题；
3. 用户提交回答；
4. DeepSeek 返回结构化评价；
5. 保存正确点、缺失点、误区和掌握度；
6. 更新当前进度；
7. 下次进入时准确继续。

## Phase 4 知识世界

课程中的 Lesson 同时作为静态知识节点。课程结构可以表达基础、核心、应用和扩展，以及前置、深化、应用和关联关系。Phase 4 展示人工定义的知识世界结构。

## Phase 5 个人认知状态

个人认知状态独立于 Lesson 的静态角色、课程状态和 MasteryRecord。系统通过本次结构化评价中的 `demonstrated_level`、`user_understanding_summary` 和 `cognitive_evidence`，维护 `unseen` 到 `transfer` 的最高已验证层级、当前稳定状态和可追溯演化事件。没有真实证据的旧学习记录不会自动回填为高级认知状态。

本阶段提供认知状态与详情 API 和学习页展示，但不实现自动解锁、路线推荐、迁移测试生成器、误区网络或 Agent 导航。

## Phase 6 迁移与误区修正

Phase 6 将“理解”与“能迁移”分开验证。只有独立的新场景 Transfer Challenge 返回 `correct` / `mostly_correct`、`demonstrated_level=transfer`、transfer 支持证据且没有矛盾时，才会把 CognitiveState 提升到 transfer。迁移失败本身不会降低已有层级或自动设置待复习；明确矛盾或 active misconception 再次出现才会触发 `needs_review`。

Misconception 保留 active / resolved 生命周期，通过 `MisconceptionEvent` 记录 observed、resolved、reopened。普通 Lesson 正确不会直接解决误区，只有针对目标误区的 Recheck 在新任务中明确修正且无矛盾才允许 resolved。固定 reasoning pattern 以最多两个 taxonomy 关联到具体误区，不使用 Embedding 或聚类。

## Phase 7 preparation：测试知识世界扩充

正式 Phase 7 之前，系统增加三个用于验证未来探索能力的静态知识岛：营养学、逻辑与科学思维、心理学。此次准备只扩充人工定义的 Course、CourseUnit、Lesson 和同 Course 内 LessonRelation。

新增知识节点没有用户认知证据时显示为 `unseen / unknown`，Seed 不创建任何学习记录、掌握度、误区、Challenge 或 CognitiveEvidence。

## Phase 7 探索引擎

探索引擎从同课程关系、跨课程陌生节点、用户当前的 CognitiveState、active Misconception 和 reasoning pattern 生成候选方向。方向先由规则计算分数和多样性，再可由 AI 生成展示文案；AI 失败时保留规则候选和本地文案。

测试知识世界目前没有 CrossCourseLessonRelation，因此保留有限的人工 taxonomy bridge，让“口渴是否是可靠的饮水依据”可以连接到“如何判断一条证据有多可靠”，让“日常饮水需求应该如何判断”可以连接到“单因素解释的陷阱”。这些是静态知识关系，不代表用户已经产生任何认知或误区证据。

探索方向的“去看看”是支线浏览：只把 `ExplorationDirection.status` 更新为 `opened`，然后导航到目标 Lesson。它不会改变目标 Course 的 `CurrentLesson`，也不会创建 LearningTurn、CognitiveEvidence 或修改 CognitiveState。只有主线“继续学习”和正式提交行为才推进 CurrentLesson。

## Phase 8 课程构建与覆盖度

课程蓝图回答“这个学科应该覆盖什么”，与个人认知状态完全分离。系统展示蓝图到正式 Lesson 的覆盖度，并允许规则或 AI 生成待审核扩充草案。AI 不直接创建或发布课程；用户显式 Apply 后，系统才在事务中新增节点和关系，保留所有历史 Lesson ID 与主线位置。

## Phase 9 来源、可信度与知识校验

Phase 9 为课程蓝图和知识节点增加可审核的 Grounding 层。用户先登记来源，再手工录入来源中的证据片段，最后把证据连接到 Blueprint、Blueprint Lesson 或正式 Lesson。来源类型、证据定位、连接关系和审核事件都会被保留；冲突不会被隐藏或自动删除。

来源可信度表示该来源在当前目标上的质量判断，不等于真值。Grounding 状态只由审核后的支持/矛盾连接与来源可信度规则产生，状态可以是未校验、部分校验、已校验或存在冲突。Grounding Coverage 只回答“课程内容有多少可追溯来源”，不回答用户是否掌握，也不改变课程结构覆盖度、CurrentLesson、LearningTurn 或 CognitiveState。

当前 v0.1 暂不在主导航、课程档案和学习页展示来源校验入口；相关后端 API 与数据保留，待需要人工审核来源时再恢复界面。
