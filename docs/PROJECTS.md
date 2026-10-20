# Project Kanban Workspace

## 定位

项目表示正在推进的目标；任务表示具体事项；四列看板表示任务当前状态。/projects 默认显示全部未归档项目的看板，URL 的 project=all|项目ID 和 view=board|today|list 保留筛选与视图。SQLite 是项目任务状态的唯一事实来源，浏览器只读取和提交 API；Markdown 是单向导出。

## Project 与 Task

Project 使用既有 ID、用户归属和 title 列作为名称，新增目标说明、图标、低饱和强调色和归档时间。状态为 active / paused / completed / archived。归档可恢复，不删除任务。默认总看板排除已归档项目；选中某个已归档项目仍可查看任务。

Task 属于一个 Project，包含名称、可选说明、状态、排序值、high / normal / low 优先级、可选日期、创建/修改/完成时间。状态固定为：

| 值 | 页面名称 | 含义 |
| --- | --- | --- |
| inbox | 待整理 | 尚未决定何时推进 |
| next | 下一步 | 已明确且可执行 |
| doing | 进行中 | 正在做 |
| done | 已完成 | 已完成，可重新打开 |

每个项目可有多项 next 任务。原 is_next_action 列保留为迁移历史，新界面和新业务规则只读 status。

## 看板与拖动

排序作用域为同一用户的同一状态列，允许“全部项目”中不同项目的任务交错排序。移动请求提交任务 ID、目标状态、相邻任务 ID 和客户端看到的 updated_at；服务端校验当前用户、相邻任务与版本，在事务中计算整数间隔排序值。通常只更新被移动任务；间隔耗尽时重排该列。拖动不会更改 project_id。从非完成列进入 done 设置 completed_at；离开 done 清除它。客户端先乐观更新，串行提交移动请求；失败先恢复原快照，再重新读取看板并提示重试。项目切换和过期读请求不会覆盖当前视图，同位置放置不提交，保留原有列内排序。

Done 列默认读取最近 20 条；其余通过分页加载。列头总数来自服务端项目统计，不依赖已加载的历史条数。移动端允许横向滑动四列，同时可在任务抽屉中用状态选择器修改任务。

## Today

Today 独立于看板项目筛选，聚合所有 active 项目的未完成任务。前端复用原 Today API，以逾期、今天到期、进行中、下一步的优先级重新互斥分组；同一任务只显示一次。下一步默认展示六条，可展开完整列表，不会自动标记为今天必须完成。支持直接完成、打开详情和返回所属项目；跨日时刷新本地日期。到期日存为 YYYY-MM-DD 日历日期，Today 使用浏览器本地日期。不同设备处于不同时区时，Today 分组可能在跨日时不同，但数据库中的任务状态和日期一致。

任务 API 创建、列表和编辑响应中的 `due_date` 均保持 `YYYY-MM-DD`；从 SQLite 的 DATE 列读回时间戳形式时，服务端恢复为日历日期，避免再次编辑任务说明时把时间戳当作无效到期日提交。

## Task Drawer 与项目列表

任务抽屉默认阅读：标题、紧凑属性、完整说明和可折叠时间记录。阅读模式底部提供编辑、完成（或重新打开）及弱化的删除入口。点击编辑复用原表单，底部只显示取消与保存；保存成功回到阅读，失败保留输入。取消、关闭、切换任务或离开页面前检查未保存修改。创建任务仍直接进入表单。抽屉正文独立滚动，底部操作固定，手机适配全屏及安全区。到期日选择器使用简体中文界面，提交值仍为 `YYYY-MM-DD`。记录区展示创建、最近修改及完成时间。当前没有任务事件表，因此不显示不可推导的状态变化历史。项目列表负责创建、编辑、暂停、恢复和归档项目；不执行项目硬删除。

## API

保留 GET/POST /api/v1/projects、PATCH /api/v1/projects/:id 和旧项目内嵌任务接口。看板使用 GET/POST /api/v1/tasks、PATCH/DELETE /api/v1/tasks/:id、PATCH /api/v1/tasks/:id/move。任务列表支持 project_id、status、due、limit、offset。所有读取和写入校验当前用户的项目归属。

## Schema 17 迁移

启动检测旧 schema 后先通过 Phase 10 流程创建 SQLite 备份。迁移添加字段，再将旧 todo 映射为 inbox；其中 is_next_action=1 的 todo 映射为 next。doing 与 done 保持原状态；旧 doing + is_next_action=1 保留为 doing。旧 done 无真实完成时间，使用旧 updated_at 近似回填 completed_at。旧任务按创建时间分配初始间隔排序值。迁移最后创建看板排序索引并删除旧的每项目唯一下一步索引。所有语句对重复启动安全；项目和任务 ID 不变。

## 本轮兼容性

本轮只调整前端交互和分组，复用现有拖拽依赖与任务 API，不增加数据库迁移、依赖或配置。工作台的 Today API 原分组协议保持不变。任务卡片标题和摘要各最多两行，四列保持可读宽度，窄屏只在看板内部横向滚动。触屏使用延迟拖拽，也可在详情编辑状态。

## Inbox 与轻量记录

Inbox 是统一捕获入口，首页、Inbox 页面和快速记录弹层共用输入组件：Enter 保存、Shift+Enter 换行，空内容不能提交，失败保留内容；捕获请求标识在重试期间保持不变，后端按用户与标识去重。整理时选择「转为任务」或「保存为记录」，无须先填写标签、类型或其他属性。任务转换复用原有创建事务，默认最近使用的有效活动项目；扩展任务属性按需展开。

轻量记录使用独立 `lightweight_records` 表，不进入任务状态列。Inbox 原始内容不随记录编辑改变，`processed_to_type=record` 与目标 ID 保存去向；来源唯一索引和条件状态认领防止重复转换，直接新建记录也按用户与请求标识去重，目标创建与 Inbox 更新同一事务提交。重复转换返回明确冲突，刷新后可查看原结果，不创建第二个目标。

Inbox 的「记录」入口展示所有记录（包括未关联项目、已归档项目的记录），其中「已归档记录」可查看和恢复。已处理 Inbox 可查看关联任务或记录；删除目标后保留来源的真实处理状态，再次打开提示不可用。选定项目看板下方的「项目记录」折叠区展示最近八条，可展开全部，支持新增与阅读；编辑内容、项目与链接复用同一弹层。

外部链接可保存显示名称及原始 URI。仅允许 `http://`、`https://` 和 `obsidian://open?vault=...&file=...`，后者要求唯一的 vault/file 参数，禁止其他动作、额外参数、用户凭据、控制字符和危险协议。含空格等特殊字符的参数应百分号编码，完整原始链接不会被重新编码。网页在新标签打开并设置 noopener/noreferrer；Obsidian 交给浏览器唤起本地应用，每个外部链接均提供复制入口。正文以纯文本安全展示，允许链接可点击，不渲染 HTML。

本次不引入全局搜索、Obsidian 同步、文件读取或完整笔记系统。Obsidian 是否能打开取决于本机客户端、Vault 名称、文件位置与浏览器协议支持；打开失败不修改数据。JSON / Markdown 导出包含记录；SQLite 备份完整保留全部数据。

### 本轮实现文件

- 后端模型与迁移：`internal/model/record.go`、`internal/model/inbox.go`、`internal/model/system.go`、`internal/database/database.go`、`internal/database/database_test.go`。
- 持久化与业务：`internal/repository/record_repository.go`、`internal/repository/inbox_repository.go`、`internal/service/record_service.go`、`internal/service/inbox_service.go`、`internal/service/export_service.go`。
- API 与装配：`internal/httpapi/record_handler.go`、`internal/httpapi/workspace_handler.go`、`internal/httpapi/handler.go`、`internal/httpapi/router.go`、`internal/app/app.go`。
- 链接与后端验证：`internal/links/external.go`、`internal/links/external_test.go`、`internal/service/record_service_test.go`、`internal/service/inbox_service_test.go`、`internal/service/export_service_test.go`、`internal/httpapi/record_handler_test.go`。
- 前端页面与输入：`web/src/views/InboxView.vue`、`web/src/views/WorkspaceHomeView.vue`、`web/src/views/ProjectsView.vue`、`web/src/components/InboxCaptureForm.vue`、`web/src/components/QuickCaptureDialog.vue`、`web/src/components/ConvertInboxDialog.vue`。
- 记录组件与 API：`web/src/components/RecordsPanel.vue`、`web/src/components/RecordDialog.vue`、`web/src/components/ExternalLink.vue`、`web/src/components/LinkedText.vue`、`web/src/api/records.ts`、`web/src/api/inbox.ts`。
- 前端工具与验证：`web/src/utils/recentProject.ts`、`web/src/utils/externalLinks.ts`、`web/src/utils/externalLinks.test.ts`、`web/src/api/records.test.ts`、`web/e2e/records.spec.ts`。
- 设计及进度：`docs/PROJECTS.md`、`docs/DATABASE.md`、`docs/ui-design.md`、`docs/ROADMAP.md`。

## 手动任务归档（Schema 27）

任务的 `archived_at` 是独立于业务状态的可空时间戳，旧任务默认不归档。仅 `done` 任务可归档；归档/恢复只更新归档时间及修改时间，不改变完成时间、可靠完成时间、排序、内容、项目或下一步标记，不生成新的完成事件。

已完成任务阅读抽屉提供归档，已完成列提供按需展开的多选和数量确认。批量操作最多 500 项，校验所有 ID 的归属及完成状态后在同一事务写入；重复请求保持已有归档时间。归档任务详情只读，先取消归档才能编辑或重新打开。默认任务查询、看板、今天与首页不包含归档任务；项目列表中的“已归档任务”按项目筛选、每页 50 项并可恢复，URL `view=list&tasks=archived&project=ID` 保存上下文。归档列表也可访问已归档项目中的任务。

新增认证接口 `POST /api/v1/tasks/archive`，请求 `{ "task_ids": [1, 2], "archived": true }`；`false` 取消归档。`GET /api/v1/tasks?archived=true` 返回归档任务，省略或 false 返回未归档任务。归档仍是正常任务记录，单个 ID 详情和原始来源引用继续有效。项目 `done_task_count` 保留所有已完成任务的历史总数，`archived_task_count` 是其中归档项；看板已完成数量为两者之差。月度回顾继续按 `actual_completed_at` 查询，包含归档任务。整库备份涵盖新字段。

## Phase 11.1a 项目场景

Project.context 为 work、life、unclassified。创建省略时默认 unclassified，更新省略时保留原值，其他值拒绝；项目创建/编辑复用原弹窗。场景只过滤信息，不改变用户所有权、状态、排序、截止日期或归档语义。

任务 effective_context = 所属项目 context，不新增任务场景列，不改变 ProjectID 非空约束。修改项目场景后，后续首页查询立即反映所属任务的新归属。全部视图保留未分类项目，并提供 `/projects?view=list&classify=unclassified` 整理入口。

`GET /api/v1/workspace/home?date=YYYY-MM-DD&context=all|work|life` 缺省等价 all，非法场景返回 400。响应新增 context、unclassified_count；projects 仍最多五个 active 项目，今日任务仍排除非 active 项目。context 条件在数据库读取及任务分页/限量之前应用。原项目/任务接口默认范围不变；POST/PATCH projects 接受 context。

自动 focus 是继续处理建议，不是每日焦点；work/life 只使用当前场景任务，all 保留原学习回退。learning、recent、资料与收集箱维持全局内容，不按名称推测分类。

## Phase 11.1b：计划日期与首页计划视图

`plan_date` 是可空 YYYY-MM-DD，表示用户选择的执行日期；`due_date` 仍是截止日期，二者互不推导。POST /api/v1/tasks 可传 plan_date；PATCH /api/v1/tasks/:id 省略 plan_date 保留旧值、传 null 清除、传日期设置，空字符串/时间戳/非法日期拒绝。TaskDrawer 使用 value-format 日历字符串，可编辑或清空计划日期；清空不删除任务、不改变状态或截止日期。

任务 PATCH 可携带 expected_updated_at（RFC3339），仓储在事务中核对且在 UPDATE 条件中校验，冲突返回既有 409。省略该字段兼容旧客户端。重复标记 done 保留 completed_at 与 actual_completed_at，重新打开才清除完成事实。跨项目移动在同一事务校验目标所有权，并触碰旧、新项目更新时间。

GET /api/v1/tasks 新增可选 context=all|work|life、plan_date、actionable=true|false、q；actionable 表示 active 项目、未完成、未归档任务。过滤均先于 limit/offset；默认行为不变，actionable 与 archived=true 互斥。首页选择器按 20 项分页（请求 21 项判断下一页），搜索已有任务，不创建副本。

GET /api/v1/workspace/home 保留旧 today，新增 planning：items、due_today、overdue、previous_plan 和 summary。只读当前用户 active 项目下的未完成、未归档任务，场景继承项目。互斥归位顺序：plan_date=查询日 → due_date<查询日 → due_date=查询日 → plan_date<查询日。主列表只显示 items；其余为提醒。summary 的 planned/due_today/overdue/previous_plan 对应这些互斥桶，total 为去重总数；同时计划且到期的任务只计 planned，在行内保留截止标记。过去计划不滚入今天，未来计划不改截止日期。

继续处理建议与 DailyFocus 无关：七天内的进行中 → 今日主动计划 → 今天到期 → 未逾期且无未来/过去计划的高优先 next。长期逾期只保留提醒，除非用户安排到今天；全部场景仍可回退全局学习，分类场景不会把全局学习当任务。

任务变更通知只传失效令牌，不传任务内容；同标签、storage 跨标签与窗口恢复触发重新读取。首页按浏览器本地午夜刷新日历日期，场景偏好不随时间改变；网络失败不乐观改变任务，409 要求刷新后重新操作。长列表先折叠展开，聚合计数保持完整。
