# 数据一致性契约

本文件固定探索来源、知识节点类型、课程进度和当前学习位置的口径，供 API、前端视图和回归测试共同使用。

## 探索来源

`ExplorationDirection` 数据表中的 `course_id` / `source_lesson_id` 是产生推荐时的上下文；候选知识节点由 `target_course_id` / `target_lesson_id` 标识。对外 Direction DTO 明确拆分为：

- `context_course_id` / `context_lesson_id`：当前筛选和推荐上下文；`course_id` 暂时作为上下文课程兼容别名。
- `source_domain_id` / `source_course_id` / `source_lesson_id`：候选节点的真实归属。当前产品中 Domain 与 Course 是同一实体，因此前两者相同。
- `target_course_id` / `target_lesson_id`：导航目标兼容字段，与候选来源指向同一个节点。

服务端返回前校验候选 Lesson 的 `course_id` 与候选 Course 一致，也校验上下文 Lesson 属于上下文 Course。卡片只读取 `source_course` / `source_lesson`，不允许用当前筛选课程覆盖。雷达请求使用 `no-store`，领域切换清空当前结果，并用请求版本阻止旧异步响应回写。

## 知识节点类型

唯一枚举是 `foundation`、`core`、`deepening`、`application`、`extension`。后端 `ContentRole`、知识图 DTO、路径视图、列表视图、节点详情、图视图和学习侧栏使用同一字段和前端映射函数。

`depth_level` 只表示知识层级；`importance` 只表示蓝图生成优先级；旧 `is_core` 只保留兼容。三者都不得隐式推断或覆盖节点类型。旧数据中空的 `content_role` 仍按既有兼容规则读取为 `core`，但新写入必须通过枚举校验。

## 三类状态与进度

课程列表 API 保留旧 `status` / `progress` 以兼容已有客户端，但 UI 不再使用 `progress`。新增字段均由结构化事实实时派生，不写入新的缓存列：

- `generation_status` / `generation_progress`（课程生成度）：有效 `BlueprintLesson.applied_lesson_id` 映射数除以 BlueprintLesson 总数；无蓝图但已有正式 Lesson 时为 100%。
- `learning_status` / `coverage_progress`（学习覆盖度）：有 LearningTurn，或 CognitiveState 不再为 `unseen` 的不同 Lesson 数，除以正式 Lesson 总数。
- `mastery_status` / `mastery_progress`（理解掌握度）：按 `unseen/exposed/recognize/understand/apply/transfer = 0/20/40/60/80/100` 给正式 Lesson 汇总，未产生 CognitiveState 的 Lesson 为 0；任何节点为 `needs_review` 时聚合状态优先为 `needs_review`。

所有百分比四舍五入并限制在 0 到 100。生成、覆盖和掌握互不推导。

## 当前 Lesson 与旧数据修复

课程存在同课程、同 Unit 的有效 `current_lesson_id` 与 `current_unit_id` 时，`initializing` 是旧的不一致状态。课程列表读取会执行幂等修复，将这种记录改为 `learning`；缺少内容或指针无效的课程保持 `initializing`。显式切换当前 Lesson 时也只在旧状态为 `initializing` 的情况下同步改为 `learning`，不会覆盖 `paused` 或 `completed`。

该阶段没有数据库结构迁移，不删除、不重建、不移动任何 Course、Unit、Lesson 或学习记录，旧 `progress` 列继续保留。若需要回滚应用版本，旧版本仍可读取全部原列；状态修复唯一可能的持久化影响是把具备有效当前 Lesson 的 `initializing` 改为 `learning`。
