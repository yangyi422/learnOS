# 可操作学习工具契约

本文记录知识结构、探索空间和误区网络从展示页变为可操作工具后的边界。

## 知识结构

- CourseUnit 与 CurriculumBlueprintUnit 通过 `course_units.blueprint_unit_id` 稳定关联；不再按标题合并。旧库启动时只根据 `BlueprintLesson.AppliedLessonID -> Lesson.UnitID` 的无歧义一对一关系回填，冲突关联保持为空并由服务拒绝错误挂载。
- 区域、正式节点和待生成节点分别使用 `course-unit:<id>` / `blueprint-unit:<id>`、`lesson:<id>`、`blueprint:<id>`。空且无扩展/生成动作的区域不返回。
- 默认只渲染当前 Lesson 所属区域的节点。折叠区域只渲染概要，展开时才挂载节点 DOM 和关系线；对大图减少首屏节点和观察器数量。
- 筛选口径固定为：当前 prerequisite 路径、正式已生成、蓝图待生成、未接触、学习中（exposed/recognize/understand）、已掌握（apply/transfer）、application 类型。
- Depth 只显示为入门/基础/进阶/深入/专题层，不推断节点类型。节点选择只读；只有“继续学习/切换到此节点”调用 CurrentLesson 写接口。
- 节点生成入口显示本次最多生成数量。生成的是待审核 CurriculumDraft，不直接创建 Lesson，也不切换 CurrentLesson。

## 探索空间

- 卡片来源使用候选节点的真实 source domain/course/lesson DTO，并展示推荐类型、理由和与当前知识的关系。
- “保存到问题池”将问题创建与方向 `saved` 状态置换放在同一事务；“撤销”归档该方向未归档的问题并恢复方向为 `active`。
- 问题状态为 `open / exploring / later / resolved / archived`，优先级为 `low / normal / high`。列表支持目标领域、状态和优先级的服务端筛选。
- 陌生知识从其他 Course 的未接触正式 Lesson 中选择，排除当前 Course；连续结果尽量不重复。探索导航仍不改变主线 CurrentLesson。

## 误区网络

- 新观察默认是 `ai_inferred`。单次 AI 观察保留为误区线索，但不聚合成稳定 reasoning pattern。
- 用户可 `confirm / correct / ignore`；审阅状态和事件在同一事务保存。用户确认或至少两次观察才参与模式聚合，纠正和忽略不参与。
- 事件 DTO 在可用时带回绑定 LearningTurn 的问题、回答和类型，用于说明形成判断的证据；历史事件不覆盖。

## 数据兼容与回滚

Schema 14 增加 `course_units.blueprint_unit_id`、`exploration_questions.priority`、`misconceptions.review_status/user_note`，并移除旧版 `blueprint_unit_id` 的隐式唯一索引（一个蓝图区域可拆成多个课程区域）。旧问题优先级读取为空时按 `normal` 展示；旧误区按 `ai_inferred` 展示。迁移前沿用系统自动 SQLite 备份；回滚应用时新增列可以保留，旧版本会忽略它们。不会删除、重建或按标题重写现有课程与学习历史。
