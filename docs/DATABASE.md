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

所有表通过 GORM `AutoMigrate` 初始化。营养学示例课程、模块和知识点在事务中按课程名/模块标题/知识点标题幂等创建或修复；当前 ID 以整数外键字段保存，不使用字符串记录当前知识点。

Phase 4 的 prerequisite 子图由 Knowledge Graph Service 做 DAG 校验；root、leaf 和拓扑顺序只描述课程结构，不代表用户已掌握或自动解锁条件。原 Phase 2 Unit ID 保留并作为饮水信号 Unit，现有 `口渴是否是可靠的饮水依据` Lesson ID 保持不变。

提交回答时，先完成 Provider 返回值的 JSON 解析、目标层级和证据业务校验，再在同一数据库事务中写入 `learning_turns`、`mastery_records`、`misconceptions`、`cognitive_states`、`cognitive_evidences`、`cognitive_state_events` 和成功的 `ai_evaluation_runs`。AI 失败或认知数据写入失败时只保留 failed 审计，不创建成功学习记录，也不更新掌握度。

Phase 2 旧记录的新字段允许为空；历史读取会将缺失数组规范为空数组，不会重新调用 AI。

## Phase 6 新增

- `assessment_challenges`：保存 `transfer` / `misconception_recheck` 的独立题面、场景、评价标准、目标层级、目标误区、状态和生成 Provider 审计。
- `challenge_attempts`：每个 Challenge 最多一个回答，保存回答、结果、demonstrated level、是否通过、反馈和误区验证快照；通过 `challenge_id` 唯一约束防止重复提交。
- `misconception_events`：保存误区的 `observed`、`resolved`、`reopened` 生命周期事件，并可绑定 LearningTurn / Challenge。
- `misconception_pattern_links`：保存误区到固定 reasoning pattern 的关联；`misconception_id + pattern_key` 唯一，单次 AI 评价最多两个 pattern。

`learning_turns` 增加 `turn_kind` 和可空 `challenge_id`；`ai_evaluation_runs` 增加 `run_type`，区分 lesson evaluation、challenge generation 和 challenge evaluation。所有新增表由 `AutoMigrate` 可重复执行创建，不回填旧挑战、旧误区 pattern 或旧认知证据；现有 Lesson ID、LearningTurn、Misconception 和 AI 审计记录保留不变。

## 后续计划

- practice_tasks：现实实践任务
- agent_runs：模型调用审计与原始输出
- app_settings：DeepSeek 与系统配置
