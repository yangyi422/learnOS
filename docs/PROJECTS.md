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
