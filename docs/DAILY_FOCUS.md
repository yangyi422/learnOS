# LifeOS Phase 11.1c：每日焦点（Daily Focus）

状态：已实现、已验证（2026-10-21），未部署线上环境。范围：工作 / 生活场景各自每天最多一个用户主动选择的焦点任务；全部视图只展示两者。不包含自动推荐写入、专注计时、积分、周回顾。

## 1. 产品规则

1. 每位用户每个自然日、每个场景（work / life）最多一个焦点；`all` 视图没有独立焦点。
2. 焦点只引用已有 `ProjectTask`（`task_id`），不复制任务、不新建任务。
3. 可被选为焦点的任务必须同时满足：
   - 属于当前用户；
   - 所属项目 `context` 等于目标场景（`unclassified` 不能被选为工作或生活焦点）；
   - 所属项目 `status = active`（暂停、已完成、归档项目不可选）；
   - 任务 `archived_at IS NULL`；
   - 任务 `status <> 'done'`。
4. 设置/更换/清空焦点不修改任务本身的 `status`、`plan_date`、`due_date`、排序或所属项目。
5. 更换焦点不修改旧任务的任何业务状态，只把当天该场景的焦点指向新任务。
6. 完成焦点复用任务完成逻辑（`PATCH /api/v1/tasks/:id` 且 `status=done`）：任务状态、项目看板、项目统计、今日计划、每日焦点同时以真实任务状态为准。焦点行保留到当天结束，读取时显示 `completed`。重复完成不会覆盖 `completed_at`。
7. 第二天不继承：新日期没有记录就是没有焦点，历史焦点不自动复制。
8. 允许完全不设置焦点，尤其是生活场景。
9. 长期逾期任务不自动成为焦点。首页「继续处理建议」仍可展示建议，但绝不写入每日焦点。
10. 历史数据不自动生成焦点（迁移只建表，不回填）。

## 2. 数据模型

表 `daily_focuses`（GORM AutoMigrate，Schema 31）：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | integer PK | |
| `user_id` | integer not null | 归属用户 |
| `focus_date` | date not null | 工作台日历日期 `YYYY-MM-DD`，不使用 UTC 时间戳推断自然日 |
| `context` | varchar(10) not null | 仅 `work` / `life` |
| `task_id` | integer null | 引用 `project_tasks.id`；为空表示引用失效（任务被删除后的安全化结果） |
| `revision` | integer not null default 1 | 乐观并发版本，每次实际变更 +1 |
| `created_at` / `updated_at` | datetime | |

唯一索引：`idx_daily_focus_scope (user_id, focus_date, context)`。

约束策略：

- 不建立 SQLite 外键级联：任务删除不能静默改变焦点历史。读取时按用户与场景重新校验引用，任务不存在时返回 `state=missing`，不会越权展示他人任务。
- 迁移幂等：`AutoMigrate` + 唯一索引可重复执行；不修改 `projects`、`project_tasks`（含 `sort_order`、`plan_date`、`due_date`）。
- 历史数据不回填；迁移前备份沿用现有 `migrationNeedsBackup` 机制。

## 3. API

日期格式统一 `YYYY-MM-DD`；场景参数 `all | work | life`；更新接口只接受 `work | life`。

### 3.1 查询

`GET /api/v1/workspace/daily-focus?date=YYYY-MM-DD&context=all`

- `date` 必填，否则 400 `DAILY_FOCUS_DATE_INVALID`。
- `context` 省略等价 `all`；非法值 400 `DAILY_FOCUS_CONTEXT_INVALID`。

```json
{
  "data": {
    "date": "2026-10-21",
    "context": "all",
    "focuses": {
      "work": {
        "id": 7,
        "date": "2026-10-21",
        "context": "work",
        "task_id": 42,
        "revision": 3,
        "state": "ok",
        "title": "完成提案初稿",
        "task_status": "doing",
        "priority": "high",
        "plan_date": "2026-10-21",
        "due_date": "2026-10-23",
        "completed_at": null,
        "task_updated_at": "2026-10-21T02:08:00Z",
        "project_id": 5,
        "project_title": "正职项目",
        "project_context": "work",
        "project_status": "active",
        "state_reason": "",
        "updated_at": "2026-10-21T02:10:00Z"
      },
      "life": null
    }
  }
}
```

`state` 取值（按检查顺序）：

| state | 含义 | 界面行为 |
| --- | --- | --- |
| `missing` | 任务已不存在 | 显示失效说明与更换/清空，不提供完成/跳转 |
| `completed` | 任务已完成 | 保留完成状态，可更换 |
| `context_changed` | 项目场景已不等于焦点场景 | 失效说明，不自动移动焦点，可更换/清空 |
| `project_inactive` | 项目暂停/完成/归档 | 保留标题与历史引用，不提供完成/跳转，可更换/清空 |
| `archived` | 任务已归档且未完成 | 失效说明，可更换/清空 |
| `ok` | 可操作 | 可完成、更换、清空、打开任务 |

`state_reason` 为可直接展示的中文说明（例如「原任务已被删除」）。

### 3.2 设置 / 更换 / 清空

`PUT /api/v1/workspace/daily-focus/:date/:context`

```json
{ "task_id": 42, "expected_revision": 3 }
```

- `task_id` 为数字：设置/更换焦点。
- `"task_id": null`：清空当天该场景焦点（删除记录）。
- `expected_revision` 可选：`>0` 时必须等于当前 `revision`，否则 409 `DAILY_FOCUS_CONFLICT`；`0`/省略表示不做版本校验。
- 相同任务重复设置：成功且不增加 `revision`（幂等）。
- 候选校验失败：400 `DAILY_FOCUS_TASK_INVALID`（不存在/无权限/跨场景/项目不可用/已归档/已完成统一返回，错误消息不泄漏其他用户数据）。
- 清空不存在的焦点：成功，返回 `focus: null`。

响应：

```json
{ "data": { "focus": { "...": "同上视图" }, "revision": 4 } }
```

清空成功：`{ "data": { "focus": null, "revision": 0 } }`。

### 3.3 工作台首页集成

`GET /api/v1/workspace/home` 增加字段（不新建第二套数据来源）：

```json
{
  "daily_focus": { "work": { "...": "焦点视图" }, "life": null }
}
```

- `context=work` 只返回 `work`，`context=life` 只返回 `life`，`context=all` 返回两者。
- 焦点读取失败不阻断首页其他区块：写入 `errors.daily_focus`，`daily_focus` 保持空对象；首页仅在原有五个核心区块全部失败时才返回 503。

候选任务列表复用现有 `GET /api/v1/tasks?context=&actionable=true&plan_date=&q=&limit=&offset=`，默认展示当前场景「今天已计划且未完成」，搜索时展示该场景全部可操作任务。不新增任务副本。

## 4. 前端交互

- `WorkspaceHomeView` 三栏布局、导航、环境光、主题与自定义色系不变；中间主卡片从「继续处理建议」升级为「今日焦点」。
- 工作 / 生活场景：展示该场景焦点，支持选择、更换、清空、完成；未设置时给出选择入口。
- 全部场景：一张卡片内展示工作与生活两份紧凑摘要，各自可操作；两者都为空时只渲染一个轻量空状态和两个入口，不渲染两个大空白卡。
- 原自动推荐降级为焦点卡片下方的次级「继续处理建议」行，不占据主焦点位置，不自动写入焦点。
- 焦点选择器复用现有任务抽屉/列表风格：默认今天已计划的未完成任务，可搜索其他可操作任务，显示项目、计划日期、截止日期；已完成/归档不可选；无任务时自然空状态；成功立即更新；失败保留用户选择并可重试。
- 任务标题是真实链接，可打开任务详情；「更换」「清空」「完成」是独立按钮，不触发卡片跳转。
- 状态：加载骨架、已完成、无焦点、失效（missing/archived/project_inactive/context_changed）、revision 冲突、保存失败均有明确提示；失败时页面保留最后确认状态。
- 深浅主题、自定义色系、移动端顺序与现有玻璃材质一致；不新增光效、不改顶部导航、不改环境背景。

## 5. 测试要求

Go：每人每天每场景唯一、工作/生活独立、未分类不可选、设置不改任务、更换不改旧任务、完成同步与重复完成幂等、次日不继承、归档/删除/跨场景移动、重复设置幂等、旧 revision 冲突、迁移幂等与历史数据保留。

前端：`dailyFocus` API 封装与状态推导单元测试；Playwright 覆盖选择/更换/清空/完成、全部视图双焦点、空状态、失效状态、冲突、失败重试、跨日与场景切换、移动端与深浅主题。

## 6. 明确不做

周回顾、AI 推荐、专注计时器/番茄钟、任务积分与排行榜、日历模块、独立任务、DailyFocus 自动生成或继承、线上部署。

## 7. 验证结果（2026-10-21）

- Go：`go vet ./internal/... ./cmd/...` 通过；全量 `go test` 无断言失败，新增 DailyFocus service / handler / schema 30→31 迁移测试全部通过（Windows 上 107 个测试仅因 TempDir 清理文件占用报伪失败，属既有环境限制）。
- 前端：`npm run typecheck` 0 错误；`npm run test` 20 文件 / 50 用例通过；`npm run build` 通过并更新 `internal/webassets/dist`。
- Playwright（全 API mock）：新增 `web/e2e/daily-focus.spec.ts` 18 项全部通过；首页/计划/回归/UI 基础合计 139 项中 138 项通过。唯一失败为移动端导航抽屉 overlay 点击，已在改动前 HEAD 独立 worktree 复现，属既有环境问题。
- 未执行真实数据库迁移、真实后端联调与线上部署；部署与真实数据验收需人工完成。
