# LearnOS 架构

```text
Browser
  ↓
Vue 3 SPA
  ↓ /api/v1
  Go + Gin
  ├── Course Service
  ├── Learning Workflow Service
  │   └── AI Provider Interface
  │       ├── MockProvider
  │       └── DeepSeekProvider
  ├── Knowledge Graph Service
  │   └── Knowledge Graph Repository
  ├── Cognitive State Service
  │   └── Cognitive Repository
  ├── Challenge Service
  │   ├── Challenge Repository
  │   └── Challenge Provider (generator / evaluator)
  └── Misconception Service
      └── Misconception Repository
  ↓
SQLite
```

## 当前阶段

本骨架只实现：

- Go 服务启动；
- SQLite 初始化与自动迁移；
- 营养学示例课程、模块和当前知识点的幂等种子数据；
- 课程列表 API；
- 当前知识点、回答提交和最近学习记录 API；
- Vue 学习页与结构化 AI / Mock 反馈；
- Vue 首页读取并展示课程；
- 单用户 Basic Auth；
- Docker 与 Caddy 部署基础。

Phase 4 增加静态 Curriculum Graph：Lesson 直接作为知识节点，LessonRelation 表达 prerequisite、extends、application 和 related。Knowledge Graph Service 负责 DTO、关系约束、prerequisite DAG 校验、拓扑排序和图统计；它不读取掌握度来解锁节点，不调用 AI，也不自动切换当前 Lesson。

Phase 3/5 中，`POST /api/v1/courses/:id/answers` 先校验课程和当前知识点，再通过 Provider 评价回答。DeepSeek 的 v2 JSON Output 经过反序列化、AssessmentTargetLevel 和认知证据校验后，才会在同一事务中写入学习轮次、掌握度、误区、认知状态、认知证据、状态事件和 AI 调用审计。Provider 失败、超时、非法结构或认知数据写入失败不会留下成功学习记录。

`AIEvaluationRun` 只用于评价调用审计，不是 Agent 编排系统。Phase 6 的普通 Lesson 评价使用 `learnos-evaluator-v3`（保留 v2 的结构和认知校验，并增加 reasoning pattern），挑战生成和挑战评价分别使用独立 Prompt 版本；历史 v1/v2 记录保留原 Prompt 版本和评价快照，读取历史不会重新调用模型，也不会自动生成 CognitiveEvidence。

`Knowledge Graph Service` 只负责 World State。`Cognitive State Service` 负责 User State 的层级、状态、证据和事件演化；CurrentLevel 只升不降，低质量新回答通过 `needs_review` 表达风险，不直接覆盖既有高质量理解摘要。

非法 AI 评价会在服务端记录 Provider、模型、Prompt 版本、尝试次数和安全的校验原因；不记录 API Key、Authorization 或原始模型响应，也不把内部原因返回前端。

Phase 6 将挑战和普通 Lesson 评价分成不同的 `LearningTurn.TurnKind`。Challenge Service 先验证生成结果和挑战回答，再由同一事务写入 AssessmentChallenge、ChallengeAttempt、LearningTurn、CognitiveState / Evidence / Event、MisconceptionEvent、pattern link 和 AIEvaluationRun。Transfer 的 target level 独立于 Lesson 的 AssessmentTargetLevel；挑战失败默认不改变原认知层级。

Transfer Challenge 在调用 AI 前对精确匹配的 non-substantive 回答（如“ 不知道 ”）进行本地规则判定，保存 `insufficient` 的正式挑战结果但不生成认知证据、误区或 contradiction；包含实际判断信息的错误回答仍进入 Challenge Evaluator，Provider timeout 语义不变。

Misconception Service 只读取结构化误区、事件和固定 reasoning pattern link，返回课程范围的网络 DTO；它不做关键词聚类、不引入向量数据库，也不改变 Curriculum Graph。

营养学知识骨架由人工定义的幂等 seed 补充；原有 B1 Lesson 和 Phase 2/3 历史不会被重建或删除。

## 数据原则

- SQLite 保存原始记录和结构化状态；
- 对话原文用于追溯，不直接作为每次模型上下文；
- 上下文由当前任务、课程状态、相关误区和必要个人档案动态组装；
- Markdown 从数据库生成，默认不与数据库双向编辑。
