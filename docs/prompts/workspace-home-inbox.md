# LearnOS Personal OS：Workspace Home + Inbox
## 个人工作台首页 / 快速收集箱 / 学习与项目统一入口

当前 LearnOS 已经逐步从单一学习系统扩展为一个自托管的个人工作台。

目前已经有两块核心能力：

```text
Learning / LearnOS = 我正在认识什么
Project Kanban     = 我正在推进什么
```

下一步先补齐两个最关键的入口能力：

```text
Workspace Home = 我今天最值得继续什么？
Inbox          = 我突然想到的东西先放哪里？
```

本任务只做 Workspace Home + Inbox。
不要顺手开发 Notes、Resources、Calendar、Habits、Data Hub、Automation、复杂 AI Agent。

---

# 一、产品原则

## Workspace Home

首页不负责“管理所有东西”，只负责：

```text
现在做什么？
今天有什么？
从哪里继续？
有没有还没整理的输入？
```

管理仍然进入对应模块：

```text
学习 → /learn
项目 → /projects
收集箱 → /inbox
```

首页不要做成 ERP Dashboard，也不要把完整 Kanban 和全部 Course Card 搬进来。

## Inbox

原则：

> Capture first, classify later.

用户记录一个想法时，不应该先被迫判断它是 Task、Question、Note 还是 Resource。
第一步只负责快速记录，之后再整理。

---

# 二、开始前先审计

现在先不要编码，也不要修改数据库。

请阅读：

- AGENTS.md
- README.md
- docs/PRODUCT.md
- docs/ARCHITECTURE.md
- docs/DATABASE.md
- 当前 Router
- 当前 AppShell / Sidebar
- 当前 Learning Home
- 当前 Project Kanban / Today View
- Course / CurrentLesson API
- Project / Task API
- 当前 UI Design System
- 当前是否已有 Inbox / Capture / Activity 类模型

先汇报：

1. 当前 `/` 对应哪个 View
2. 当前 Learning Home 路由
3. 当前 Sidebar 结构
4. 当前如何取得 Course.CurrentLesson
5. Project Kanban / Today 当前如何取数据
6. 是否已经有 Inbox / Capture 类模型
7. 是否已有全局 Header / Quick Action
8. 哪些旧首页组件可复用
9. 哪些 Learning Home 组件需要迁移到 `/learn`
10. 是否需要新增首页聚合 API
11. Inbox 推荐模型字段
12. Migration 风险
13. 预计修改文件
14. 实施顺序

确认后再开始。

---

# 三、路由目标

目标语义：

```text
/               Workspace Home
/learn           Learning Home
/projects        Project Kanban
/inbox           Inbox
/explore         Exploration
/settings        Settings
```

按当前 Router 风格调整，但必须明确：

```text
/ 不再等于 Learning Home
```

原有“继续你的学习”页面完整保留，只迁移为 Learning Home，不要重做一遍。

---

# 四、Sidebar

建议调整为：

```text
首页

学习
项目
收集箱

探索

系统设置
```

保持现有 Modern / Minimal / Premium Shell，不要为了新增 Inbox 重做整个侧栏。

---

# 五、Workspace Home 第一版只放 5 块

```text
1. Current Focus
2. Today
3. Active Projects
4. Continue Learning
5. Inbox
```

如果当前已有可靠 Activity 数据，可以加一个非常轻的 Recent Activity。
如果没有，不要为了首页新建完整 Activity Event 系统。

---

# 六、Desktop Wireframe

```text
┌────────────────────────────────────────────────────────────┐
│ 今天继续什么？                              [快速记录 +]  │
│ 9月29日 · 星期二                                          │
├───────────────────────────────┬────────────────────────────┤
│ CURRENT FOCUS                 │ TODAY                      │
│                               │                            │
│ LearnOS                       │ 进行中        2             │
│ 修复 Knowledge Map 交互       │ 今天到期      1             │
│                               │ 下一步        4             │
│ 项目 · 进行中                  │                            │
│ [继续处理 →]                  │ [查看今天 →]               │
├───────────────────────────────┴────────────────────────────┤
│ ACTIVE PROJECTS                                           │
│ LearnOS          1 Doing · 4 Next      [打开]             │
│ 像素团团         1 Doing · 2 Next      [打开]             │
├────────────────────────────────────────────────────────────┤
│ CONTINUE LEARNING                                         │
│ 营养学                                                    │
│ 能量摄入与消耗的三大出口                                  │
│ 未接触 · unknown                          [继续学习 →]     │
├────────────────────────────────────────────────────────────┤
│ INBOX                                                      │
│ 3 条待整理                                                │
│ · 研究 SQLite WAL backup                                  │
│ · LearnOS 首页增加全局搜索                                │
│ · 某个网页链接                                            │
│                                         [打开收集箱 →]     │
└────────────────────────────────────────────────────────────┘
```

首页通过 Typography、spacing 和少量 Surface 建立层级。
不要做成十几个小统计卡。

---

# 七、Current Focus

这是首页最高优先级模块。

不要使用 AI 选择。
使用确定性规则。

建议优先级：

```text
1. status=doing 的 Task
2. 今天到期且未完成的 Task
3. status=next 的高优先 Task
4. 最近活跃 Course 的 CurrentLesson
5. 都没有 → Empty State
```

多个 Doing Task 时优先：

```text
Priority 高
→ DueDate 更近
→ UpdatedAt 更新
```

按当前真实 Priority schema 调整。

不要新增 CurrentFocus 表。
Current Focus 由现有数据实时推导。

Task Focus 示例：

```text
CURRENT FOCUS

LearnOS

修复 Knowledge Map 重复生成节点

项目 · 进行中 · P1

[继续处理 →]
```

Learning Focus 示例：

```text
CURRENT FOCUS

营养学

能量摄入与消耗的三大出口

学习 · 未接触

[继续学习 →]
```

---

# 八、Today Widget

直接复用当前 Project Kanban 的 Today 逻辑。

只展示摘要：

```text
Doing count
Due today count
Next count
```

最多展示 3~5 条相关 Task。

不要在 Home 重写另一套 Today 查询规则。
如果当前逻辑只在前端，请抽成共享 composable / service。

---

# 九、Active Projects

首页只显示 active projects，最多 3~5 个。

每个只展示：

```text
Project Name
Doing count
Next count
Open count
```

例如：

```text
LearnOS
1 进行中 · 4 下一步

[打开]
```

不要复制完整 Kanban，也不要显示很长的项目说明。

---

# 十、Continue Learning

复用现有 Course / CurrentLesson。

首页只显示 1~2 个最近活跃学习领域，不显示全部 Course Grid。

展示：

```text
Course Name
Current Unit
Current Lesson
Cognitive Level
Cognitive Status
```

选择规则：

```text
优先最近有 LearningTurn 的 Course
否则 UpdatedAt / CreatedAt 最近的 Course
```

不要用 AI。

原本全部知识世界继续放 `/learn`。

---

# 十一、Inbox 模型

新增轻量 `InboxItem`。

建议字段：

```go
ID

Content string

Status string
// inbox
// processed
// archived

SourceType string
// manual
// url
// system

SourceURL string

ProcessedToType string
// task
// learning_question
// future: note/resource

ProcessedToID *uint

CreatedAt
UpdatedAt
ProcessedAt *time.Time
ArchivedAt *time.Time
```

按当前项目 schema 约定调整。

如果系统已有用户模型，必须按 UserID scope。
如果当前明确是单用户 self-hosted，则沿用现有单用户架构，不要本阶段新建多租户体系。

---

# 十二、Inbox Capture

Capture 时只要求：

```text
Content
```

可选：

```text
URL
```

不要要求用户同时选择：

- Project
- Tag
- Priority
- Course
- Type

输入越快越好。

全站至少在 Workspace Home Header 提供：

```text
+ 快速记录
```

点击打开轻量 Dialog / Popover：

```text
快速记录

[________________________________]

[保存]
```

支持：

```text
Ctrl/Cmd + Enter
```

不要为了它新建完整 Command Palette。

---

# 十三、URL Capture

如果输入内容本身是 URL：

```text
SourceType = url
SourceURL = value
```

v0.1 不做：

- 网页抓取
- 自动摘要
- favicon
- metadata parser
- Readability
- RAG

先可靠保存即可。

---

# 十四、Inbox Page

`/inbox`

结构：

```text
收集箱

12 条待整理

[快速记录...]

────────────────────────
今天

研究 SQLite WAL backup                 09:12
LearnOS 首页应该增加全局搜索           08:42
https://...                             08:10

────────────────────────
昨天
...
```

默认只展示：

```text
Status = inbox
```

可以切：

```text
待整理 / 已处理 / 已归档
```

但不要复杂筛选器。

---

# 十五、Inbox Item 操作

v0.1 优先提供：

```text
转为任务
归档
删除
```

如果 Phase 7 Question Pool 当前已经有安全、简单的手工新增接口，则可以增加：

```text
转为学习问题
```

如果没有，不要为了 Inbox 重构 Phase 7。

Notes / Resource 转换本阶段不做。

---

# 十六、Convert to Task

点击：

```text
转为任务
```

打开轻量 Dialog：

```text
标题
Project
Status
Priority
DueDate
```

默认：

```text
Content → Task.Title
Status = inbox 或当前选择
```

如果 Content 很长，可放 Description，并让用户确认标题。
不要 AI 自动改写。

转换成功后：

```text
InboxItem.Status = processed
ProcessedToType = task
ProcessedToID = Task.ID
ProcessedAt = now
```

---

# 十七、转换必须事务安全 + 幂等

必须保证：

```text
Task 创建成功
+
Inbox 标记 processed
```

要么一起成功，要么一起失败。

不能出现 Task 已创建，但 Inbox 仍显示待处理，从而重复转换。

已 processed 的 InboxItem 再次 convert：

```text
INBOX_ITEM_ALREADY_PROCESSED
```

前端提供：

```text
查看已创建任务
```

---

# 十八、Archive / Delete

Archive：

```text
Status = archived
ArchivedAt = now
```

不要 hard delete。

Delete 可保留，但放在 More Menu / destructive secondary action，不做醒目的红色主按钮。

---

# 十九、Home Inbox Widget

首页最多显示 3 条：

```text
INBOX

3 条待整理

研究 SQLite WAL backup
LearnOS 首页增加搜索
某个网页链接

[打开收集箱 →]
```

如果实现简单，Sidebar Inbox 可显示未处理数量 Badge。
不要做复杂通知中心。

---

# 二十、首页聚合 API

优先评估现有 API 是否足够。

如果 Workspace Home 需要 5~8 个独立请求，可以新增只读：

```http
GET /api/v1/workspace/home
```

返回例如：

```json
{
  "focus": {},
  "today": {},
  "projects": [],
  "learning": {},
  "inbox": {
    "count": 3,
    "items": []
  }
}
```

要求：

- 只读
- 不修改状态
- 不创建 CurrentFocus
- 不写数据库
- 单个模块缺失时不要整页 500

---

# 二十一、Loading / Error / Empty

Home 模块允许独立 Skeleton。

如果某一块失败：

```text
项目数据暂时无法加载
[重试]
```

其他模块继续展示。

不要整页裸 `Failed to fetch`。

Empty State：

## No Projects

```text
还没有项目。
[新建项目]
```

## No Courses

```text
知识世界还是空的。
[创建学习领域]
```

## Inbox Empty

```text
收集箱是空的。
有想法时先记下来，不必马上分类。
```

---

# 二十二、UI 风格

继续沿用当前：

```text
Quiet Intelligence UI
Modern
Minimal
Premium
```

首页视觉主次：

```text
Current Focus
    ↓
Today
    ↓
Projects / Learning
    ↓
Inbox
```

不要把所有模块做成一样大的 Card。

Desktop 推荐：

```text
Current Focus  2/3
Today          1/3

Projects       1/2
Learning       1/2

Inbox          full
```

Mobile：

```text
Current Focus
Today
Learning
Projects
Inbox
```

---

# 二十三、必须保留现有 Learning / Projects

当前“继续你的学习”页面：

```text
只迁移到 /learn
```

不要删除，不要大改。

当前 Project Kanban：

```text
继续留在 /projects
```

首页只聚合摘要。

---

# 二十四、本阶段明确不做

不要做：

- Notes Editor
- Resource Library
- Calendar
- Habits
- Data Hub
- Automation
- 完整 Global Search
- AI 自动分类 Inbox
- AI 自动推荐 Focus
- 网页内容抓取
- RAG
- Activity Event System
- Obsidian Sync

Global Search 只需要在 AppShell 预留未来入口位置即可。

---

# 二十五、多设备语义

Inbox 数据必须存 SQLite。

不能只放 localStorage。

电脑、手机、平板访问同一个自托管服务器时必须看到同一份 Inbox。

localStorage 只允许保存：

```text
last selected view
UI preferences
```

---

# 二十六、API 建议

按当前项目风格调整。

Inbox 至少：

```http
GET    /api/v1/inbox
POST   /api/v1/inbox
PATCH  /api/v1/inbox/:id
POST   /api/v1/inbox/:id/convert-to-task
POST   /api/v1/inbox/:id/archive
DELETE /api/v1/inbox/:id
```

可选：

```http
GET /api/v1/workspace/home
```

---

# 二十七、测试要求

## Routing

1. `/` = Workspace Home
2. `/learn` = 原 Learning Home
3. Sidebar Home / Learn 正确
4. 原学习相关导航不丢失

## Home

5. Focus from doing Task
6. fallback to due Task
7. fallback to next Task
8. fallback to CurrentLesson
9. empty focus
10. Today summary
11. Active Projects
12. Learning summary
13. Inbox summary

## Inbox

14. create item
15. restart 后仍存在
16. URL item
17. archive
18. delete
19. default 只展示 inbox

## Convert

20. convert to Task
21. transaction safety
22. duplicate conversion blocked
23. ProcessedToID 正确
24. Task 属于正确 Project

## Regression

25. Project Kanban unaffected
26. Today View unaffected
27. Learning Home unaffected
28. CurrentLesson unaffected
29. Domain Initialization unaffected

---

# 二十八、构建验证

执行：

```bash
go test ./...
go build ./cmd/server
npm run build
docker compose build app
git diff --check
```

如存在：

```bash
npm run lint
```

也执行。

---

# 二十九、文档

更新：

```text
README.md
docs/PRODUCT.md
docs/ARCHITECTURE.md
docs/DATABASE.md
docs/ROADMAP.md
```

新增：

```text
docs/WORKSPACE_HOME.md
docs/INBOX.md
```

明确：

```text
Home = continuation surface
Inbox = capture first, classify later
```

---

# 三十、完成标准

必须满足：

```text
/ 已成为 Personal Workspace Home
原 Learning Home 完整保留在 /learn
首页能看到 Current Focus
首页能看到 Today
首页能看到 Active Projects
首页能继续 CurrentLesson
首页能看到 Inbox 摘要
全站至少有一个 Quick Capture 入口
Inbox 数据存 SQLite
Inbox 可转换为 Task
转换事务安全且幂等
手机 / 电脑访问同一数据
UI 与当前 Design System 一致
```

---

# 三十一、最终开始前汇报

在真正编码前，请最终给我：

1. `/` 与 Learning Home 的安全迁移方案
2. Current Focus 的具体计算规则
3. Today 数据复用方案
4. Active Projects 数据复用方案
5. Continue Learning Course 选择规则
6. InboxItem 最终字段
7. Convert-to-Task 的事务方案
8. 是否需要 `/workspace/home` 聚合 API
9. Sidebar 修改方案
10. Desktop ASCII wireframe
11. Mobile ASCII wireframe
12. Inbox ASCII wireframe
13. 预计修改文件
14. Migration 风险
15. 实施顺序

确认后再开始实现。
