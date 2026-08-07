# LearnOS 产品定义

## 一句话定位

一个以用户控制的数据存储作为长期记忆，通过知识地图、状态机和可配置 Agent 工作流组织 LLM，从而实现可持续学习的个人 AI 系统。

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
