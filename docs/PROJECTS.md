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

排序作用域为同一用户的同一状态列，允许“全部项目”中不同项目的任务交错排序。移动请求提交任务 ID、目标状态、相邻任务 ID 和客户端看到的 updated_at；服务端校验当前用户、相邻任务与版本，在事务中计算整数间隔排序值。通常只更新被移动任务；间隔耗尽时重排该列。拖动不会更改 project_id。从非完成列进入 done 设置 completed_at；离开 done 清除它。请求失败时客户端重新读取看板并显示恢复提示。

Done 列默认读取最近 20 条；其余通过分页加载。列头总数来自服务端项目统计，不依赖已加载的历史条数。移动端允许横向滑动四列，同时可在任务抽屉中用状态选择器修改任务。

## Today

Today 对所选项目范围中的 active 项目展示三组未完成任务：doing、今天到期或逾期、next 候选。任务只显示在最靠前的一组；next 不会自动标记为今天必须完成。到期日存为 YYYY-MM-DD 日历日期，Today 使用浏览器本地日期。不同设备处于不同时区时，Today 分组可能在跨日时不同，但数据库中的任务状态和日期一致。

## Task Drawer 与项目列表

任务抽屉负责创建、编辑、完成和确认删除。记录区展示创建、最近修改及完成时间。当前没有任务事件表，因此不显示不可推导的状态变化历史。项目列表负责创建、编辑、暂停、恢复和归档项目；不执行项目硬删除。

## API

保留 GET/POST /api/v1/projects、PATCH /api/v1/projects/:id 和旧项目内嵌任务接口。看板使用 GET/POST /api/v1/tasks、PATCH/DELETE /api/v1/tasks/:id、PATCH /api/v1/tasks/:id/move。任务列表支持 project_id、status、due、limit、offset。所有读取和写入校验当前用户的项目归属。

## Schema 17 迁移

启动检测旧 schema 后先通过 Phase 10 流程创建 SQLite 备份。迁移添加字段，再将旧 todo 映射为 inbox；其中 is_next_action=1 的 todo 映射为 next。doing 与 done 保持原状态；旧 doing + is_next_action=1 保留为 doing。旧 done 无真实完成时间，使用旧 updated_at 近似回填 completed_at。旧任务按创建时间分配初始间隔排序值。迁移最后创建看板排序索引并删除旧的每项目唯一下一步索引。所有语句对重复启动安全；项目和任务 ID 不变。
