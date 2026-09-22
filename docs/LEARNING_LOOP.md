# 核心学习闭环

本文记录 LearnOS 当前“回答—评价—修正—迁移—掌握证据”的可验证契约。它只描述个人学习状态，不改变知识地图、课程蓝图或全局视觉系统。

## 一次完整流程

1. 学习页按 Course 和 Lesson 将未提交回答保存在浏览器 `localStorage`；空白回答不发请求，离开页面前提示，提交成功后删除对应草稿。
2. `POST /api/v1/courses/:course_id/answers`（探索支线使用 Lesson answer 接口）携带 `idempotency_key`。服务端验证回答，再调用评价 Provider。
3. `learnos-evaluator-v4` 必须返回已理解内容、关键缺口、可能误区、使用证据、可信度、不确定性、下一步行动和迁移条件。服务端完成 JSON、长度、枚举、目标层级和证据校验后才允许持久化。
4. 每次修正回答创建新的 `LearningTurn`；旧回答、旧评价、认知证据和状态事件不更新、不删除。
5. 达到 `understand` 后可生成迁移挑战。挑战题面说明新场景与原问题的区别；挑战回答写入独立 `ChallengeAttempt` 和 `transfer_challenge` 类型的 `LearningTurn`。
6. 迁移通过新增 `transfer` 支持证据，并将累计掌握度增加 0.10（最高 1.00）；未通过不扣除掌握度。误区复测只追加修正证据，不直接增加迁移掌握分。

## 状态机

底层认知层级保持既有六级模型，产品界面合并为四种说明：

| 底层层级 | 页面状态 | 判定含义 |
| --- | --- | --- |
| `unseen` | 未接触 | 没有可验证的回答证据 |
| `exposed` / `recognize` | 学习中 | 已接触或能识别概念，但解释证据不足 |
| `understand` | 初步理解 | 能解释核心机制或重要边界，可开始迁移挑战 |
| `apply` / `transfer` | 掌握 | 已在案例或陌生场景中使用知识 |

`current_level` 只升不降。后续低质量回答或矛盾证据不会删除历史高质量证据，也不会直接降低层级，而会将 `status` 置为 `needs_review`。达到当前层级且没有负面证据时为 `stable`，较低层级的成长过程为 `developing`。每次成功评价同时追加 `CognitiveEvidence` 和 `CognitiveStateEvent`，页面展示证据内容、时间与状态变化原因。

单次回答的 `mastery_score` 是该次 AI 评价快照；普通回答用“当前累计分 × 既有回答数 + 本次评价分”再除以新回答数更新 `MasteryRecord.mastery_score`。每个 `LearningTurn` 保存 `mastery_score_before` / `mastery_score_after`，迁移成功再基于当前累计分增加 0.10。因而新评价可影响当前汇总，但不能覆盖历史回答、历史分数或证据。

## 原子性与幂等

- 普通回答只有在 AI 结果通过结构与业务校验后，才在一个 SQLite 事务中写入 LearningTurn、MasteryRecord、认知状态、证据、状态事件、误区和成功审计。
- 挑战生成先完成 AI 校验，再在一个事务中写入 Challenge 和成功审计；挑战作答在另一个事务中写入 Attempt、独立 LearningTurn、掌握度、认知证据/事件、误区变化和审计。
- AI 超时、Provider 错误或非法输出只写入失败审计，不创建成功学习轮次，不更新掌握度和认知状态。挑战作答超时时挑战仍为 `pending`。
- 回答、挑战生成和挑战作答均接受最长 128 字符的 `idempotency_key`。同一键和同一业务内容重放返回原结果；同一键配不同内容返回 HTTP 409。
- 前端在请求成功前复用同一键，并用提交中状态阻止连续点击；网络失败重试不会形成重复记录。

## 评价展示契约

评价结果至少包含：

- `correct_parts`：已理解的部分；
- `missing_parts`：关键知识缺口；
- `misconceptions`：可能存在的误区，可为空数组但字段必须存在；
- `evidence_used`：只能引用用户本次回答中实际出现的内容；
- `confidence` 与 `uncertainty`：结论可信度和仍不能判断的部分；
- `recommended_next_action`：可执行的下一步；
- `transfer_challenge_eligible`：服务端按评价结果和认知层级重新计算，不能只相信模型布尔值。

Mock Provider 使用固定且标注为 `mock` 的结构化结果，不伪装为真实模型评价。

## Schema 13 迁移与回滚

迁移只增加可兼容列和唯一索引，不删除、重写或回填历史学习数据：

- `learning_turns`：增加 `idempotency_key`、评价证据/可信度/不确定性/下一步/迁移条件和掌握度前后快照；
- `assessment_challenges`、`challenge_attempts`：增加 `idempotency_key`；
- `challenge_attempts`：增加 `explanation`。

旧行的新字符串/JSON字段保持空值，新数值字段保持 0，前端读取时按空数组和兼容默认值处理。应用检测到旧 schema 后会先生成 pre-migration SQLite 备份，再执行可重复的 `AutoMigrate` 并把 `system_metadata.schema_version` 更新为 13。

升级前已有的 `MasteryRecord` 不做破坏性重算；它的当前分数和 `answer_count` 作为后续累计计算的历史基线。历史 LearningTurn 和 CognitiveEvidence 仍是逐条审计事实，避免迁移根据旧快照猜测并改写用户状态。

回滚到旧二进制时，旧代码可忽略这些新增列；若必须精确恢复旧 schema，停止写入后使用迁移前自动备份执行现有 Restore 流程。不要手工删除列或清理学习历史。

## 仍然存在的边界

- 草稿只保存在当前浏览器，不跨设备同步，也不进入服务端备份。
- 当前认知状态是历史证据的汇总快照；完整审计以 LearningTurn、CognitiveEvidence 和 CognitiveStateEvent 为准。
- 掌握分是产品化辅助指标，不是统计置信区间；可信度字段描述单次 AI 结论的不确定性。
