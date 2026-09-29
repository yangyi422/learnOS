# LearnOS Personal OS：Workspace Home UX Polish
## 首页信息密度 / Continue Surfaces / Empty States / Quick Capture 收口

当前 Workspace Home 第一版的信息架构已经成立：

- Current Focus
- Today
- Active Projects
- Continue Learning
- Inbox

本任务不再做大规模页面重构，而是把它收口成适合每天打开的 Personal OS 首页。

核心目标：

- 提升 Current Focus 信息密度
- 让 Today 更可操作
- 让 Projects / Learning / Inbox 更像 continuation surface
- 统一 Empty / Loading / Error
- 强化快速记录
- 保持现代、简洁、克制、高级感

不要新增 Calendar、Weather、Habit、Data Dashboard、Notes、Activity Timeline、完整 Global Search、AI 推荐或统计图表。

---

## 一、总体原则

首页只回答：

- 现在做什么？
- 今天有什么？
- 从哪里继续？
- 有什么还没整理？

首页不负责完整管理项目、课程或 Inbox。

详细管理仍进入：

- `/projects`
- `/learn`
- `/inbox`

不要把首页重新做成 Dashboard 拼盘。

---

## 二、保留当前结构

保留：

```text
Current Focus
Today
Active Projects
Continue Learning
Inbox
```

允许调整：

- 内容密度
- 排版
- CTA
- Empty State
- 行为入口
- 小范围布局比例

---

## 三、Current Focus

这是首页视觉优先级最高的区域。

目标：

Current Focus 必须回答：

- 现在是什么？
- 为什么值得继续？
- 属于什么？
- 当前状态是什么？

Task Focus 示例：

```text
CURRENT FOCUS

LearnOS

修复 Knowledge Map 重复生成节点

进行中 · P1
最近更新 2 小时前

[继续处理 →]
```

逾期：

```text
像素团团

测试1

逾期 1 天 · 待整理

这是当前最需要处理的到期任务。

[打开任务 →]
```

Next：

```text
LearnOS

部署 v0.1 到云服务器

下一步 · High

这是当前优先级最高的可执行任务。

[开始处理 →]
```

---

## 四、Current Focus Reason

不要使用 AI。

根据 Focus 来源生成固定 reason：

```text
doing      → 当前正在推进
overdue    → 已到期，需要处理
due_today  → 今天到期
next       → 当前最高优先的下一步
learning   → 最近的学习位置
```

不要新增数据库字段。

---

## 五、布局

Desktop 建议：

```text
Current Focus 65~70%
Today         30~35%
```

减少 Current Focus 内部无意义留白。

---

## 六、Today

保留：

```text
Doing
Due / Overdue
Next
```

但增强可操作性。

```text
TODAY                                 查看全部 →

0        1        0
进行中   到期/逾期  下一步

需要关注
● 测试1                         逾期 1 天
```

最多展示 3 条。

优先级：

```text
overdue
→ due today
→ doing
→ next
```

状态颜色使用现有 token，逾期只用轻 warning / danger，不要大红块。

---

## 七、Active Projects

保持简洁列表，不改成大型 Card Grid。

```text
ACTIVE PROJECTS                         项目列表 →

▣ LearnOS
  1 进行中 · 4 下一步 · 5 未完成        →

▣ 像素团团
  0 进行中 · 2 下一步 · 3 未完成        →
```

要求：

- 最多 3~5 个 active project
- 整行可点击
- hover 有轻微反馈
- 不展示完整 Description
- 不展示多个按钮
- 复用现有 Project accent / icon

---

## 八、Continue Learning

有 Course：

```text
CONTINUE LEARNING                      学习空间 →

营养学

能量与能量平衡
能量摄入与消耗的三大出口

未接触 · unknown

[继续学习 →]
```

只显示 1 个主要 Course，可选显示：

```text
还有 9 个知识世界
```

不要展示完整 Course Grid。

选择逻辑优先：

1. 最近存在 LearningTurn 的 Course
2. CurrentLesson 最近更新的 Course
3. Course UpdatedAt 最近
4. Course CreatedAt 最近

不要 AI 推荐。

---

## 九、Learning Empty State

没有 Course：

```text
CONTINUE LEARNING                      学习空间 →

知识世界还是空的

创建第一个学习领域后，
这里会保存你最近的学习位置。

[开始一个学习领域 →]
```

不要只显示灰色说明文字。

---

## 十、Inbox

Inbox 应成为首页最快的输入口。

```text
INBOX                                  打开收集箱 →

[ 有什么先记下来……                 ][ + ]

最近
· 研究 SQLite WAL backup
· LearnOS 首页增加快速搜索
· 看看这个网页
```

---

## 十一、Inline Quick Capture

桌面端提供 inline input。

Placeholder：

```text
有什么先记下来……
```

行为：

```text
Enter
或点击 +
→ POST /inbox
```

成功后：

- 清空 input
- 更新 preview
- toast：已记录

不要打开 Dialog。

顶部原有“快速记录”按钮保留，二者复用同一个 createInbox service。

---

## 十二、Inbox Preview

最多显示 3 条：

```text
研究 SQLite WAL backup        09:12
```

如果是 URL，显示 domain / truncated URL 即可。

不要抓 metadata。

Inbox 空：

```text
收集箱是空的。

有想法时先记下来，
不需要马上决定它属于什么。
```

Inline Capture 仍然显示。

---

## 十三、首页层级

视觉顺序保持：

1. Current Focus
2. Today
3. Active Projects
4. Continue Learning
5. Inbox

Desktop 推荐：

```text
Row 1
Current Focus | Today

Row 2
Projects | Learning

Row 3
Inbox full width
```

---

## 十四、不要因为页面空就塞 Widget

当前数据少时留白正常。

禁止为了填空增加：

- 天气
- 时钟
- Quote
- CPU / Server 状态
- 日历
- 伪数据
- 随机统计卡

只优化 Empty State。

---

## 十五、统一 Empty / Loading / Error

Empty State：

```text
Title
1~2 行说明
Primary / Secondary action
```

Loading：

每个模块独立 Skeleton，不要整页 Spinner。

Error：

```text
项目数据暂时无法加载。

[重试]
```

不要显示裸 `Failed to fetch`。

---

## 十六、交互

要求：

```text
Current Focus CTA
→ 对应 Task / Learning

Today item
→ Task Drawer / Project view

Project row
→ 对应 Project Kanban

Continue Learning
→ CurrentLesson

Inbox item
→ Inbox

Quick Capture
→ 保存 InboxItem
```

---

## 十七、Sidebar

当前 Sidebar 基本正确，只确认：

```text
首页
学习
项目
收集箱
探索
设置
```

active route 正确。

不要重构 AppShell。

---

## 十八、Visual Style

继续沿用：

```text
Quiet Intelligence UI
Modern
Minimal
Premium
```

强调：

- typography
- subtle surface
- low shadow
- 轻 border
- 大留白
- 单一 primary blue
- muted status colors

不要做大量 Card / 高饱和 Bento / Dashboard 图表。

---

## 十九、Responsive

Desktop：

```text
Focus | Today
Projects | Learning
Inbox
```

Tablet：

```text
Focus
Today
Projects | Learning
Inbox
```

Mobile：

```text
Current Focus
Today
Continue Learning
Active Projects
Inbox
```

Mobile Inbox Capture 全宽。

---

## 二十、Accessibility

确保：

- button focus state
- input focus state
- Enter submit
- CTA 有清晰 aria / title
- 色彩不是唯一状态表达

---

## 二十一、性能

如果已有 `/api/v1/workspace/home`，优先增强现有 response。

如果当前多个 API 请求已经稳定，不要为了架构统一强制改成聚合 API。

先审计再决定。

---

## 二十二、测试

至少验证：

### Focus
1. doing task
2. overdue task
3. due today task
4. next task
5. learning fallback
6. empty focus

### Today
7. overdue priority
8. due today
9. doing
10. next
11. preview 上限

### Projects
12. active only
13. max 5
14. project click

### Learning
15. recent learning course
16. no history fallback
17. empty course CTA

### Inbox
18. inline capture
19. Enter submit
20. button submit
21. refresh persists
22. max 3 preview
23. empty state

### Error / Loading
24. module loading
25. partial module error
26. retry

### Regression
27. Project Kanban unaffected
28. Learning Home unaffected
29. Inbox page unaffected
30. Sidebar routing unaffected

---

## 二十三、构建验证

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

## 二十四、明确不做

本任务不做：

- 新业务模块
- Global Search 完整实现
- Notes
- Resources
- Calendar
- Habits
- Data Hub
- Automation
- Activity Timeline backend
- AI Focus 推荐
- AI Inbox 分类
- 新 Dashboard 图表

---

## 二十五、开始前先审计

先不要改代码。

请先汇报：

1. 当前 Workspace Home 组件结构
2. Current Focus 当前数据来源和选择规则
3. Today 当前数据来源
4. Project summary 当前数据来源
5. Continue Learning 当前选择逻辑
6. Inbox preview 当前接口
7. Quick Capture 当前实现
8. 是否需要改 workspace 聚合 API
9. 哪些组件可以复用
10. 哪些 Empty State 需要调整
11. 预计修改文件
12. 实施顺序

然后给出新的 Desktop ASCII wireframe。

确认后再开始实现。
