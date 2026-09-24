# LearnOS 架构

> 重构前的路由、模型、进度计算、关键数据流和回归测试矩阵见 [`TEST_BASELINE.md`](TEST_BASELINE.md)。
> 探索来源、节点类型、三类进度和旧课程状态修复的现行契约见 [`DATA_CONSISTENCY.md`](DATA_CONSISTENCY.md)。
> 回答版本、评价可信度、迁移挑战和掌握证据的现行契约见 [`LEARNING_LOOP.md`](LEARNING_LOOP.md)。

```text
Browser
  ↓
Vue 3 SPA
  ↓ /api/v1
  Go + Gin
  ├── User Service
  │   └── Session authentication / administrator user management
  ├── Course Service (user-scoped)
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
  ├── Exploration Service
  │   ├── Exploration Repository
  │   └── optional Exploration Provider
  ├── Curriculum Service
  │   └── Curriculum Repository
  ├── Next Lesson Service
  │   ├── Knowledge Graph Repository
  │   └── Cognitive / Learning Repository
  ├── Grounding Service
  │   └── Grounding Repository
  ↓
SQLite
```

项目工作台使用独立的 Project handler → service → repository 链路。Project 保存用户归属；Task 通过 Project 继承访问边界。看板移动在 repository 事务内校验相邻任务、计算状态列排序值并维护完成时间。前端通过 /projects 的查询参数保存项目与视图选择，任务事实只在 SQLite；迁移和 API 见 [PROJECTS.md](PROJECTS.md)。

## 当前阶段

本骨架只实现：

- Go 服务启动；
- SQLite 初始化与自动迁移；
- 营养学、逻辑与科学思维、心理学示例课程、模块和当前知识点的幂等种子数据；
- 课程列表 API；
- 当前知识点、回答提交和最近学习记录 API；
- Vue 学习页与结构化 AI / Mock 反馈；
- Vue 首页读取并展示课程；
- production 及非 development 环境使用数据库用户会话认证；首个管理员由 `APP_USERNAME` / `APP_PASSWORD_HASH` 引导创建；管理员手动创建普通用户；development 仅监听 `127.0.0.1` 并跳过认证；
- Course 保存 `user_id` 作为个人知识世界的边界；课程下的 Lesson、学习记录、掌握度、认知状态、挑战和探索记录通过 `course_id` 继承隔离；旧课程首次迁移时归属引导管理员；
- Docker 与 Caddy 部署基础。

Phase 4 增加静态 Curriculum Graph：Lesson 直接作为知识节点，LessonRelation 表达 prerequisite、extends、application 和 related。Knowledge Graph Service 负责 DTO、关系约束、prerequisite DAG 校验、拓扑排序和图统计；它不读取掌握度来解锁节点，不调用 AI，也不自动切换当前 Lesson。

Knowledge Map Graph 在现有 `GET /api/v1/courses/:id/knowledge-graph` 只读 DTO 上补充待生成节点、`AppliedLessonID` 映射、稳定的 `lesson:<id>` / `blueprint:<id>` 节点键、BlueprintRelation 和 CurrentLesson 标记。正式 Lesson 与待生成节点均按 Course 隔离。当前主路径视图使用普通 DOM 按 prerequisite DAG 的 topological rank 纵向排列，IntersectionObserver 根据视口中心更新选中节点；Vue Flow 实现保留为暂不启用的回退组件。点击节点只更新详情，`POST /api/v1/courses/:id/current-lesson` 仍是唯一的显式主线切换入口。列表视图继续保留，Blueprint Unit Expansion 和 Curriculum Draft Apply 仍复用原有服务，不在路径视图中新增生成逻辑。

知识区域不再按标题合并：CourseUnit 与 BlueprintUnit 使用稳定数据库 ID 关联，旧数据只从 AppliedLesson 映射做无歧义回填。前端默认只挂载当前区域的节点，其余区域保留概要并按需展开；筛选逻辑在路径和列表视图复用同一组节点 DTO。误区网络仅聚合重复观察或用户确认的模式，单次 AI 结果保留为可审阅推测。

Phase 3/5 中，`POST /api/v1/courses/:id/answers` 先校验课程和当前知识点，再通过 Provider 评价回答。DeepSeek 的结构化 JSON Output 经过反序列化、AssessmentTargetLevel 和认知证据校验后，才会在同一事务中写入学习轮次、掌握度、误区、认知状态、认知证据、状态事件和 AI 调用审计。Provider 失败、超时、非法结构或认知数据写入失败不会留下成功学习记录。

`AIEvaluationRun` 只用于评价调用审计，不是 Agent 编排系统。普通 Lesson 评价使用 `learnos-evaluator-v4`：保留 v3 的认知与 reasoning pattern 校验，并增加评价所用证据、可信度、不确定性、下一步行动和迁移资格；挑战生成和挑战评价继续使用独立 Prompt 版本。历史 v1/v2/v3 记录保留原 Prompt 版本和评价快照，读取历史不会重新调用模型，也不会自动生成 CognitiveEvidence。

`Knowledge Graph Service` 只负责 World State。`Cognitive State Service` 负责 User State 的层级、状态、证据和事件演化；CurrentLevel 只升不降，低质量新回答通过 `needs_review` 表达风险，不直接覆盖既有高质量理解摘要。

非法 AI 评价会在服务端记录 Provider、模型、Prompt 版本、尝试次数和安全的校验原因；不记录 API Key、Authorization 或原始模型响应，也不把内部原因返回前端。Challenge Generation 额外按每个 attempt 记录 provider latency、Prompt/内容字符数、校验状态和累计耗时，并在最终成功时记录总 attempt 数和总耗时；日志不输出完整 Prompt 或 RawResponse。

Phase 6 将挑战和普通 Lesson 评价分成不同的 `LearningTurn.TurnKind`。Challenge Service 先验证生成结果和挑战回答，再由同一事务写入 AssessmentChallenge、ChallengeAttempt、LearningTurn、CognitiveState / Evidence / Event、MisconceptionEvent、pattern link 和 AIEvaluationRun。Transfer 的 target level 独立于 Lesson 的 AssessmentTargetLevel；挑战失败默认不改变原认知层级。

核心学习写操作使用客户端生成的幂等键：普通回答、挑战生成、挑战回答重放时读取原记录，同一键对应不同内容时拒绝为冲突。普通回答的累计掌握分不再由单次评价直接覆盖，而按回答版本累计聚合；通过迁移挑战会追加独立 transfer 证据并将累计分提高 0.10（最高 1.00）。每个 LearningTurn 保存本次评价分和累计分前后快照。

Transfer Challenge 在调用 AI 前对精确匹配的 non-substantive 回答（如“ 不知道 ”）进行本地规则判定，保存 `insufficient` 的正式挑战结果但不生成认知证据、误区或 contradiction；包含实际判断信息的错误回答仍进入 Challenge Evaluator，Provider timeout 语义不变。

Misconception Service 只读取结构化误区、事件和固定 reasoning pattern link，返回课程范围的网络 DTO；它不做关键词聚类、不引入向量数据库，也不改变 Curriculum Graph。

营养学、逻辑与科学思维、心理学知识骨架由人工定义的幂等 seed 补充。Seed 按课程名、课程内 Unit 标题和 Lesson 标题查找或创建，按关系端点和类型幂等插入；原有 B1 Lesson、Lesson ID、当前课程状态和 Phase 1~6 历史不会被重建或删除。新课程只初始化首个当前 Lesson，不写入任何用户学习或认知记录。

本次测试世界扩充暂不引入 `CrossCourseLessonRelation`。现有 `LessonRelation` 继续只表达单 Course 内结构，跨课程连接留给正式 Phase 7 的独立设计，避免绕过同 Course prerequisite DAG 校验。

Exploration Service 将 Knowledge Graph 的相邻关系、跨课程陌生节点、CognitiveState、active Misconception 和 reasoning pattern 组合成 ExplorationDirection，并保存状态、评分、原因和生成来源。当前没有 `CrossCourseLessonRelation` 表，因此对测试知识世界使用有限、人工维护的跨课程 taxonomy fallback（饮水判断 ↔ 多因素推理/证据可靠性）；它只描述知识世界，不创建或推断用户 CognitiveState / Misconception。ExplorationQuestion 从方向生成可追问的问题池；规则生成是稳定降级路径，AI 只负责受校验的展示文案。

探索方向打开使用独立的支线 Lesson API。该操作只更新 ExplorationDirection 为 `opened` 并导航到目标 Lesson，不更新任何 Course 的 CurrentLesson，也不创建 LearningTurn、CognitiveEvidence 或 CognitiveState 变化。支线回答使用独立接口，主线 CurrentLesson 仍只由正常学习行为修改。

Phase 8 增加独立的 Curriculum Blueprint 与 Coverage 层。Blueprint 描述学科应覆盖的 Unit、Lesson 和关系；Coverage 只根据 BlueprintLesson 到正式 Lesson 的映射计算，不读取 CognitiveState、MasteryRecord 或学习历史。AI 只能生成 CurriculumDraft ChangeSet，显式 Apply 在单事务中新增节点、关系和映射，并复用 Knowledge Graph 的关系/DAG 校验；Apply 不删除或修改既有 Lesson，不改 CurrentLesson，也不写入个人认知数据。

学习主线的下一步由 Next Lesson Service 按规则计算，不调用 AI：优先当前 Unit 的后续正式 Lesson，再考虑相邻 Core Unit，并结合 prerequisite 是否已有认知证据、Core/Extension、Depth 和是否未接触排序。未满足前置知识只作为提示，不构成硬锁。`POST /api/v1/courses/:id/current-lesson` 是唯一的显式主线切换入口；查看节点、打开 Knowledge Structure 和 Exploration Open 都不会修改 CurrentLesson，也不会创建学习记录。

Blueprint Unit 的后续展开分两步：Unit Expansion 只通过既有领域蓝图 Provider 生成并持久化 5～10 个 BlueprintLesson；随后 CurriculumService 按目标 Unit 生成 3～5 个 CurriculumDraft，仍须 Review 后 Apply 才会创建正式 Lesson。Expansion、Draft 和 Apply 都不写入 LearningTurn、CognitiveState、CognitiveEvidence 或 Misconception。Expansion 通过数据库条件更新原子地从 `unexpanded` 抢占到 `expanding`，只有持有者可以调用 Provider；完整响应校验后才在单事务内写入 BlueprintLesson、BlueprintRelation 并标记 `expanded`，失败只复位自己的进行中状态。待审核 Draft 通过按 Course、Blueprint 和目标 Unit（或全局范围）生成的 PendingKey 及数据库唯一索引防止重复生成。

Phase 9 增加独立的 Source / Evidence / Grounding 层。KnowledgeSource 是跨课程可复用的来源登记，不属于任何单一 Course；SourceEvidence 记录人工录入的定位、摘录或摘要；GroundingLink 把证据审核后连接到 `curriculum_blueprint`、`blueprint_lesson` 或 `lesson`。可信度评估是对来源质量与适用性的可审计判断，不等于事实真值。Grounding Service 在来源连接或可信度审核后统一重算 `ungrounded`、`partially_grounded`、`grounded`、`conflicted`，并记录 GroundingReviewEvent；它不调用搜索、不自动生成证据，也不读取或写入 CognitiveState、CognitiveEvidence、LearningTurn、MasteryRecord、Misconception 或 CurrentLesson。

Phase 10 增加独立的运维层：Backup Service 负责 SQLite 快照、保留策略和恢复前保护； Export Service 只从结构化业务表生成 JSON / Markdown； Database Health Service 区分轻量 `/health` 探活和 diagnostics 的完整 integrity check； Consistency Service 只读检查外键引用、课程图、探索记录、蓝图映射和 Grounding 连接。迁移仍由 `database.Open` 统一执行，优先写入 pre-migration backup，再 AutoMigrate 和更新 `system_metadata`。应用收到终止信号后优雅停止 HTTP server 并关闭 SQLite 连接。

系统设置页通过 `/api/v1/system/ai-config` 管理单用户 AI Provider。API Key 仅保存于 SQLite 的系统配置单例并以“已配置”状态返回；RuntimeProvider 代理允许保存配置后立即切换 Mock / DeepSeek，领域初始化和其它 AI 服务共享同一运行时 Provider。

设置页恢复流程不在活动连接上替换 SQLite：确认请求先把所选备份固定为隐藏的待恢复快照，再通知应用优雅关闭 HTTP 和数据库连接；受 Docker 或进程管理器重启后，启动阶段在 `database.Open` 前创建当前数据库的安全备份并原子恢复。这样即使普通备份保留策略淘汰了原文件，确认过的恢复快照仍保持稳定。失败标记会归档且不会循环重启。

前端使用统一语义状态色和中文术语；桌面与移动导航都是带 `aria-label` 的真实链接导航，当前链接使用 `aria-current="page"`。异步设置反馈通过 `aria-live` 播报，字段错误通过 `aria-describedby` 关联，所有核心控件使用一致的 `focus-visible` 外框。实现和验收范围见 `docs/ACCESSIBILITY_AND_SETTINGS.md`。

当前已完成 `Phase 7 Exploration Engine`；后续长期能力仍不包括自动路线切换或 Agent 自主修改知识图。

## 数据原则

- SQLite 保存原始记录和结构化状态；
- 对话原文用于追溯，不直接作为每次模型上下文；
- 上下文由当前任务、课程状态、相关误区和必要个人档案动态组装；
- Markdown 从数据库生成，默认不与数据库双向编辑。
