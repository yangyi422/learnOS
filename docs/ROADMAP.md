# LearnOS Roadmap

## Phase 1：可运行骨架（本次）

- [x] Go + Gin 服务
- [x] SQLite + GORM
- [x] Vue 3 + Element Plus
- [x] 课程列表 API
- [x] 营养学种子课程
- [x] Docker 部署骨架
- [x] 单用户 Basic Auth

## Phase 2：不依赖 AI 的最小学习闭环（已完成）

- [x] course_units、lessons、learning_turns、mastery_records 与 misconceptions 数据模型
- [x] 当前课程、模块和知识点的幂等初始化
- [x] 当前问题页面与首页继续学习跳转
- [x] 提交用户回答与安全校验
- [x] 固定模拟评价与反馈展示
- [x] learning_turns 持久化
- [x] 在事务中创建或更新知识点掌握度
- [x] 最近回答记录查询与刷新后保留

本阶段不接入 DeepSeek，不实现 JSON Schema AI 输出校验；固定评价将在 Phase 3 替换。

## Phase 3：DeepSeek AI 结构化认知评价（本次完成）

- [x] AI_PROVIDER 配置与 Mock / DeepSeek Provider 抽象
- [x] DeepSeek OpenAI-compatible Chat Completions 接入
- [x] JSON Output、Prompt v1、严格业务结构校验
- [x] 超时、有限重试和安全错误处理
- [x] LearningTurn 结构化评价快照
- [x] AIEvaluationRun 调用审计
- [x] MasteryRecord 更新和 Misconception 精确去重
- [x] 学习页和历史页展示 AI 评价内容

本阶段不实现课程状态机、下一知识点选择或复习调度。

## 后续课程状态与复习能力

- [ ] 初始化、学习、追问、复习、实践、暂停、结课状态
- [ ] 下一知识点选择策略
- [ ] 阶段复习
- [ ] 到期复习

## Phase 4：知识世界与课程结构（已完成）

- [x] Lesson 静态角色、核心/扩展和 Depth 元数据
- [x] LessonRelation：prerequisite / extends / application / related
- [x] prerequisite DAG 校验、拓扑顺序和图统计
- [x] 营养学知识骨架的分支、平行节点和汇合点
- [x] Knowledge Graph API 与 Lesson 关系 API
- [x] 基础知识结构查看页 `/courses/:id/map`
- [x] 保留已有 Lesson ID 与 Phase 2/3 学习历史

Phase 4 不实现个人认知状态、自动解锁、路线推荐、AI 修改知识图或最终视觉地图。

## Phase 5：个人认知状态与认知演化（已完成）

- [x] CognitiveState / CognitiveEvidence / CognitiveStateEvent
- [x] Evaluation Prompt v2 与严格 Schema / Target 校验
- [x] Recognize / Understand / Apply / Transfer 层级和 State Transition
- [x] 认知状态、证据和演化时间线 API
- [x] LearningView / KnowledgeGraphView 认知状态展示
- [x] 保留旧 Phase 2/3 学习历史，不自动 backfill 认知等级

Phase 5 不实现自动解锁、路线推荐、迁移测试生成器、误区网络或 Agent 导航。

## Phase 6：迁移测试、误区网络与认知修正（已完成）

- [x] AssessmentChallenge / ChallengeAttempt 与独立挑战审计
- [x] transfer generator / challenge evaluator 及严格业务校验
- [x] Transfer Challenge 通过规则与 CognitiveState 影响规则
- [x] Misconception active / resolved / reopened 生命周期和事件
- [x] 固定 reasoning pattern taxonomy 与误区网络 API
- [x] 针对性 Misconception Recheck 与 Correction Validation
- [x] LearningView 挑战交互、误区网络页、知识结构迁移状态 overlay
- [x] AutoMigrate 和旧 Phase 2/3 数据兼容
- [x] non-substantive Transfer Answer 的本地正式判定与状态保持

Phase 6 不实现探索雷达、问题池、Agent 导航、知识来源可信度或最终图形化认知地图。

## 后续长期使用能力

- [ ] Markdown 导出
- [ ] 自动备份
- [ ] 模型切换
- [ ] 调用成本统计
- [ ] 移动端体验优化
