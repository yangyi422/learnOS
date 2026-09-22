# 数据库演进草案

## 当前已实现

### courses

- id
- name
- description
- goal
- status
- progress
- current_unit
- current_unit_id
- current_lesson_id
- last_studied_at
- created_at
- updated_at

## Phase 2 已实现

- `course_units`：课程模块，包含目标、排序和 `pending` / `learning` / `completed` 状态。
- `lessons`：最小可提问知识点，包含核心问题、期望理解、状态和 `assessment_target_level`；后者限制一次评价最多能证明的认知层级。
- `lessons` 的静态结构字段：`is_core`、`content_role`、`depth_level`，分别表示核心/扩展、课程内容角色和课程设计深度；不表示用户掌握层级。
- `lesson_relations`：保存同一 Course 内 Lesson 之间的 `prerequisite`、`extends`、`application`、`related` 关系，并以 Course、起点、终点和关系类型组合唯一。
- `learning_turns`：问题、回答、结构化评价快照、Provider、模型、Prompt 版本、掌握度和复习标记；数组和误区以 JSON 字符串保存。
- `mastery_records`：知识点掌握度、回答次数、错误次数、复习标记和下次复习时间。
- `misconceptions`：保存 AI 明确识别的 active 误区，并按课程、Lesson 和理解文本精确去重。
- `ai_evaluation_runs`：记录 Provider、模型、Prompt 版本、调用状态、重试次数、延迟、Token、原始模型 content 和安全错误代码。
- `cognitive_states`：每个 Lesson 的个人认知层级、状态、理解摘要和最近评价时间。
- `cognitive_evidences`：绑定 LearningTurn 的认知证据，记录证据类型、认知层级、正负极性和来源；`learning_turn_id + evidence_index` 组合唯一。
- `cognitive_state_events`：记录认知状态每次成功更新的前后层级、状态、摘要和原因，详情 API 默认按最新 20 条倒序返回。

`learning_turns` 额外保存 `demonstrated_level`、`user_understanding_summary` 和 `cognitive_evidence` JSON 快照；旧 Phase 2/3 记录这些字段为空是合法的。

所有表通过 GORM `AutoMigrate` 初始化。营养学、逻辑与科学思维、心理学示例课程、模块和知识点在同一事务中按课程名/模块标题/知识点标题幂等创建或修复；LessonRelation 按 Course、起点、终点和关系类型幂等插入。当前 ID 以整数外键字段保存，不使用字符串记录当前知识点。

测试知识世界扩充只写入静态课程结构。已有营养学 Lesson（尤其是 `口渴是否是可靠的饮水依据`）按现有标题复用 ID，已有 Course 的当前 Lesson 和课程状态不被重置；新 Course 以 0% 和首个 Lesson 初始化。Seed 不创建 `learning_turns`、`mastery_records`、`misconceptions`、`cognitive_states`、`cognitive_evidences`、`assessment_challenges` 或其它用户历史数据。

Phase 4 的 prerequisite 子图由 Knowledge Graph Service 做 DAG 校验；root、leaf 和拓扑顺序只描述课程结构，不代表用户已掌握或自动解锁条件。原 Phase 2 Unit ID 保留并作为饮水信号 Unit，现有 `口渴是否是可靠的饮水依据` Lesson ID 保持不变。

提交回答时，先完成 Provider 返回值的 JSON 解析、目标层级和证据业务校验，再在同一数据库事务中写入 `learning_turns`、`mastery_records`、`misconceptions`、`cognitive_states`、`cognitive_evidences`、`cognitive_state_events` 和成功的 `ai_evaluation_runs`。AI 失败或认知数据写入失败时只保留 failed 审计，不创建成功学习记录，也不更新掌握度。

Phase 2 旧记录的新字段允许为空；历史读取会将缺失数组规范为空数组，不会重新调用 AI。

## Schema 13：核心学习闭环审计

- `learning_turns.idempotency_key`：可空、唯一；旧记录保持 NULL，新写入用于回答、迁移挑战和误区复测的幂等重放。
- `learning_turns` 增加 `evidence_used`、`confidence`、`uncertainty`、`recommended_next_action`、`transfer_challenge_eligible`，保存评价当时的完整解释快照。
- `learning_turns.mastery_score_before` / `mastery_score_after`：保存该轮写入前后的累计掌握度；原 `mastery_score` 继续表示本次评价分。
- `assessment_challenges.idempotency_key` 与 `challenge_attempts.idempotency_key`：可空、唯一；防止挑战生成和挑战作答因重试重复写入。
- `challenge_attempts.explanation`：保存挑战评价解释，不再只依赖 LearningTurn 读取。

所有变化均为增加列/索引的 `AutoMigrate`，不删除、不重排、不回填既有行。旧行按零值兼容，完整迁移与回滚策略见 [`LEARNING_LOOP.md`](LEARNING_LOOP.md)。

## Schema 14：可操作知识工具

- `course_units.blueprint_unit_id`：可空、非唯一的稳定 ID 关联；一个蓝图区域可以拆成多个正式课程区域。启动时仅从已应用 BlueprintLesson 到正式 Lesson 的无歧义关系回填，不使用标题；迁移会移除旧版隐式唯一索引。
- `exploration_questions.priority`：`low / normal / high`，旧空值按 `normal` 兼容；状态增加 `later / resolved`。
- `misconceptions.review_status` / `user_note`：保存 AI 推测、用户确认、用户纠正或忽略，以及用户说明；审阅动作追加 MisconceptionEvent，不覆盖旧事件。

迁移只增列、索引和安全回填，不删除 Course、Lesson、LearningTurn 或认知证据。迁移前自动备份，回滚时旧应用可忽略新增列。详细契约见 [`OPERABLE_LEARNING_TOOLS.md`](OPERABLE_LEARNING_TOOLS.md)。

## Phase 7 Exploration Engine

- `exploration_directions`：保存源/目标 Lesson、相邻/跨域/陌生方向、评分、规则原因、状态和 AI/规则生成 provenance。
- `exploration_questions`：保存从方向产生的深化、连接、挑战或陌生知识问题，以及 open / exploring / archived 状态。

两张表由 `AutoMigrate` 可重复执行创建，不修改既有 Course、Lesson、LessonRelation 或 Phase 1~6 历史。方向打开只更新自身状态；目标 Course 的 `current_lesson_id` 不在探索流程中写入。探索导航也不创建 LearningTurn、CognitiveEvidence 或 CognitiveState。

当前没有 `CrossCourseLessonRelation` 表。为保证测试知识世界在没有用户误区记录时仍能验证 `cross_domain`，服务层使用有限的静态跨课程 taxonomy bridge；它不写入用户状态、误区或学习记录，也不替代未来独立的跨课程关系模型。

## Phase 6 新增

- `assessment_challenges`：保存 `transfer` / `misconception_recheck` 的独立题面、场景、评价标准、目标层级、目标误区、状态和生成 Provider 审计。
- `challenge_attempts`：每个 Challenge 最多一个回答，保存回答、结果、demonstrated level、是否通过、反馈和误区验证快照；通过 `challenge_id` 唯一约束防止重复提交。
- `misconception_events`：保存误区的 `observed`、`resolved`、`reopened` 生命周期事件，并可绑定 LearningTurn / Challenge。
- `misconception_pattern_links`：保存误区到固定 reasoning pattern 的关联；`misconception_id + pattern_key` 唯一，单次 AI 评价最多两个 pattern。

`learning_turns` 增加 `turn_kind` 和可空 `challenge_id`；`ai_evaluation_runs` 增加 `run_type`，区分 lesson evaluation、challenge generation 和 challenge evaluation。所有新增表由 `AutoMigrate` 可重复执行创建，不回填旧挑战、旧误区 pattern 或旧认知证据；现有 Lesson ID、LearningTurn、Misconception 和 AI 审计记录保留不变。

## Phase 10 Dogfooding Readiness

- `system_metadata`：保存当前 schema version、最近迁移时间和应用版本；启动迁移前若检测到旧 schema，会先执行 SQLite `VACUUM INTO` 备份。
- SQLite 备份文件位于独立 backup 目录，文件名带 UTC 时间戳，按 `BACKUP_RETENTION_COUNT` 保留最近备份。
- JSON / Markdown 导出从结构化表生成，不包含 API Key、密码哈希或系统元数据；AI 调用原始响应不进入个人导出快照。
- Restore CLI 和设置页恢复都只接受通过 `integrity_check` 的 SQLite 文件；设置页会先固定所选快照并要求逐字确认，重启后在无活动连接时创建 pre-restore 备份，再通过临时文件原子替换目标库。失败请求不会删除现有学习数据，也不会自动重试形成重启循环。
- Diagnostics 和 Consistency Report 都是只读能力；一致性检查不自动修复任何数据。
- `DELETE /api/v1/courses/:id` 的课程删除在单事务中清理 Course、CourseUnit、Lesson、蓝图、课程草案、学习/认知/挑战/误区/探索记录和 Grounding 连接；KnowledgeSource 与 SourceEvidence 是跨课程共享资产，不随课程删除。
- `ai_configurations` 是单例系统配置，保存当前 AI Provider、Base URL、Model 和 API Key。API Key 不进入 JSON/Markdown 导出，也不会由 API 返回，只返回是否已配置；配置 API 更新运行时 Provider，不需要重启。

后续仍不新增 Agent、搜索、RAG、向量或自动扩图能力。

## Phase 8 Curriculum Construction & Coverage

- `curriculum_blueprints`：课程蓝图版本、领域、学习目标和临时核验状态。
- `curriculum_blueprint_units` / `curriculum_blueprint_lessons`：蓝图节点及其 core / recommended / optional 重要性；`applied_lesson_id` 只建立蓝图到正式 Lesson 的映射。
- `curriculum_blueprint_units.expansion_status`：记录 Unit 是 `unexpanded`、`expanding` 还是 `expanded`。展开只增加 BlueprintLesson 和 BlueprintRelation，不直接创建正式 Lesson。
- `curriculum_blueprint_relations`：蓝图内部关系，Apply 前验证 prerequisite DAG。
- `curriculum_drafts`：待人工审核的 ChangeSet、生成来源、Provider/Model/Prompt 版本和 Apply 状态。

`curriculum_blueprint_lessons` 继续以 `(blueprint_id, key)` 保证稳定语义 Key 唯一。`curriculum_drafts` 的 `blueprint_unit_id` 与内部 `pending_key` 用于记录目标区域；处于 `generating` 或 `draft` 状态的相同生成范围由部分唯一索引保护，避免双击、重试或并发请求产生多份待审核草案。

Coverage 是学科结构事实，不使用 `CognitiveState`、`CognitiveEvidence`、`MasteryRecord` 或 `LearningTurn`。AI 只写 Draft；Apply 使用事务新增正式 CourseUnit/Lesson/Relation 并更新映射，既有 ID、CurrentLesson、学习历史和个人认知数据保持不变。

显式切换 CurrentLesson 更新 `courses.current_unit`、`current_unit_id` 和 `current_lesson_id`；若旧状态仍是 `initializing`，同时修复为 `learning`，但不覆盖 `paused` / `completed`。下一课计算只读课程结构、LessonRelation 和已有学习/认知证据，不因推荐或切换创建任何个人学习记录。

课程列表的生成度、学习覆盖度和理解掌握度不新增持久化列，而是从 Blueprint 映射、LearningTurn/CognitiveState 和认知层级分别计算。旧 `courses.progress` 仅保留 API/旧客户端兼容，当前前端不再展示它。详细契约与旧数据修复边界见 `docs/DATA_CONSISTENCY.md`。

## Phase 9 Source, Credibility & Grounding

- `knowledge_sources`：跨课程复用的来源登记，包括来源类型、书目信息、URL/DOI/ISBN、访问状态和人工核验状态；通过 DOI、ISBN、规范化 URL 或标题/机构/年份避免重复。
- `source_evidences`：来源中的人工录入证据片段，保存定位、可选短摘录、摘要、语言、提取方式和证据核验状态。
- `grounding_links`：把 SourceEvidence 连接到有限的 Grounding Target（`curriculum_blueprint`、`blueprint_lesson`、`lesson`），记录 supports / contradicts / qualifies、强度、理由、审核状态和审核时间。
- `source_credibility_assessments`：按权威性、方法质量、直接性、时效性和独立性保存来源评估；评分表示“用于当前目标的可信度判断”，不表示事实真值，也不自动替代人工审核。
- `grounding_review_events`：保存目标状态重算前后的审计事件。

Grounding 状态由统一服务按审核后的连接和来源评估重算：存在审核后的矛盾连接为 `conflicted`；有审核后的支持连接但缺少合格来源评估为 `partially_grounded`；有中/高强度支持、审核后的连接和来源总体评分至少 60 才为 `grounded`；没有审核支持为 `ungrounded`。Blueprint 的状态由其直接连接与子节点聚合，不覆盖 Phase 8 的个人认知或课程主线状态。

Grounding Coverage 是来源覆盖事实，只统计 BlueprintLesson / 正式 Lesson 的审核连接、证据、来源和冲突；Curriculum Coverage 只统计蓝图到 Lesson 的结构映射；Personal Cognitive State 只统计用户学习证据。三者不互相推导。

## Phase 7 preparation

当前已完成测试知识世界扩充和 Exploration Engine：营养学 8 个 Lesson、逻辑与科学思维 3 个 Lesson、心理学 3 个 Lesson。跨课程候选由 Exploration Service 计算，仍不新增 `CrossCourseLessonRelation`；`LessonRelation` 继续只表达同 Course 内结构。
