# LearnOS v0.1 UI Polish
## Quiet Intelligence UI / 全站视觉统一与学习体验收口

Phase 10、Progressive Domain Initialization、多课程、Next Lesson、Knowledge World Expansion 等核心能力已经完成或基本可用。

现在进入 LearnOS v0.1 正式 Dogfooding 前的最后一轮 UI 收尾。

当前实现已完成第一轮 UI 收口：AppShell、移动端导航、PageHeader、SectionHeader、DomainTile、StatusTag、EmptyState、ProgressBar 与学习状态摘要已提取为共享组件；课程档案、首页、领域初始化、探索、设置、知识结构和误区网络已接入统一页面层级。来源校验入口继续按产品决定保持隐藏，业务接口与数据模型不变。

Visual Redesign 实施记录：首页已重构为 Current Focus / Knowledge Worlds / Exploration 三层构图；课程展示改为 Compact Domain Tile；探索改为 Discovery Card；LearningView 改为 Editorial Flow + Knowledge Rail；Knowledge Structure 改为 CSS Vertical Knowledge Path。以上仅重组前端 Composition，未修改 API、课程状态、认知状态或探索算法。

Final UI Unification 实施记录：Curriculum Coverage 已改为 Coverage Summary / Unit Section / Lesson Row / Draft Feed；Knowledge Map 的统计和 Node Detail 已同步轻量 Rail；Learning History、错误提示、标签和空状态已统一到首页的 Editorial surface。未修改后端业务逻辑、API 契约或数据语义。

本任务只做：

- Design System
- 全站 App Shell
- 页面信息层级统一
- LearningView 学习体验收口
- Knowledge Structure 视觉收口
- Course Library 视觉收口
- Exploration / Sources / Settings 一致化
- Responsive

不要修改核心学习业务逻辑。

---

# 1. 设计方向

统一为：

## Quiet Intelligence UI

关键词：

- Calm
- Focused
- Structured
- Dense but breathable
- Knowledge-first
- Low visual noise
- Editorial + productivity tool

中文：

> 安静、理性、有层次的个人认知工作台。

不要做成：

- 在线教育平台
- 游戏化学习 App
- 企业后台
- AI 聊天网站
- 默认 Element Plus Demo

参考气质：

- Linear 的克制
- Notion 的知识感
- Obsidian 的个人世界感
- LearnOS 自己的认知状态体系

---

# 2. 核心原则

1. 内容优先于装饰
2. 学习主流程优先于元数据
3. 状态颜色只表达状态
4. 少用阴影
5. 少用 Card 套 Card
6. 少用巨大 Header
7. 高信息密度但保留呼吸
8. 同类交互视觉一致
9. 同一状态颜色一致
10. 桌面端优先，移动端可用

---

# 3. 全站 App Shell

所有主页面统一使用同一 Sidebar Shell。

左侧：

```text
LearnOS
个人学习系统

学习首页
课程档案
探索空间
知识来源
系统设置

v0.1
```

要求：

- Sidebar 在 Home / Courses / Learning / Map / Exploration / Sources / Settings 都存在
- 当前路由高亮
- 不再每页重复巨大 PERSONAL LEARNING SYSTEM Header
- 内容区顶部只保留紧凑 Page Header
- Sidebar desktop width 约 220~240px

---

# 4. Design Tokens

新增统一 theme tokens，例如：

```css
--bg-app: #f6f7f9;
--bg-surface: #ffffff;
--bg-subtle: #f8fafc;

--text-primary: #111827;
--text-secondary: #64748b;
--text-tertiary: #94a3b8;

--border-default: #e5e7eb;
--border-subtle: #edf0f4;

--primary: #3b82f6;
--primary-hover: #2563eb;
--primary-soft: #eff6ff;

--success: #16a34a;
--success-soft: #f0fdf4;

--warning: #d97706;
--warning-soft: #fffbeb;

--danger: #dc2626;
--danger-soft: #fef2f2;

--sidebar-bg: #111827;
--sidebar-active: #25344f;
```

可按现有品牌色微调，但必须集中定义。

---

# 5. 状态颜色

统一语义：

```text
蓝色   当前 / 主操作 / 当前学习
绿色   stable / 已掌握 / 已完成
黄色   needs_review / misconception / warning
红色   系统错误 / destructive
灰色   unseen / unknown / inactive
```

同一状态不能跨页面换颜色。

---

# 6. Typography

统一建议：

```text
Eyebrow
11px / 600 / uppercase / letter-spacing .12em

Page Title
28~30px / 700

Section Title
20~22px / 650~700

Card Title
16~18px / 600

Body
14~15px

Metadata
12~13px
```

---

# 7. Spacing / Radius / Shadow

Spacing 使用统一尺度：

```text
4 / 8 / 12 / 16 / 20 / 24 / 32 / 40 / 48
```

Radius：

```text
大 Surface 12px
Card 10px
Input 8px
Button 6px
Tag 4~6px
Dialog / Drawer 12px
```

Card 默认：

```text
1px border
no shadow
```

只在 Dialog / Drawer / Dropdown / hover interactive card 使用明显 shadow。

---

# 8. Button 体系

Primary：

- 继续学习
- 提交回答
- + 新建学习领域
- 确认并创建

Secondary：

- 查看知识结构
- 查看课程档案
- 重新生成

Text / Tertiary：

- 刷新
- 返回
- 查看详情

Danger：

- 删除课程

删除课程不要作为 Card 常驻主按钮。

---

# 9. Page Header

取消巨大重复顶部 Header。

统一：

```text
Breadcrumb / Eyebrow

页面标题                       页面操作
一行说明
```

例如：

```text
课程档案                       [+ 新建学习领域]

管理你的知识世界与学习进度。
```

v0.1 版本只放 Sidebar bottom / Settings，不要每页右上角重复。

---

# 10. Home

首页目标：

> 第一眼知道“今天继续什么”。

建议结构：

```text
Page Header

继续学习
Course Current Focus

你的知识世界
Course overview

值得看看
Exploration Radar

最近变化（可选）
```

弱化当前营销式大 Hero。

---

# 11. Course Card

首页和课程档案尽量复用同一个 CourseCard。

结构：

```text
个人心理运行学                  12%

系统学习情绪与行为之间的关系

当前
情绪基础 · 情绪是什么

━━━━━━━━━━━━ 12%

[继续学习]                    ···
```

更多菜单：

```text
查看课程档案
查看知识结构
删除课程
```

如果没有 CurrentLesson：

```text
尚未选择学习节点
[选择节点]
```

---

# 12. Courses 页面

Header：

```text
课程档案                       [+ 新建学习领域]

管理你的知识世界与学习进度。
```

桌面使用 responsive grid：

```text
repeat(auto-fit, minmax(280px, 1fr))
```

避免一门课程时过窄、多门课程时过挤。

---

# 13. LearningView

这是最高优先级页面。

Desktop：

```text
Main 72%
Sidebar 28%
```

Main 顺序严格：

```text
Lesson Header
→ Core Question
→ User Answer
→ AI Evaluation
→ Next Action
→ Misconception / Transfer
→ Recent History
```

右侧：

```text
Cognitive State
Knowledge Position
Source Summary
Recent Cognitive Change
```

---

# 14. Learning Header

紧凑：

```text
个人心理运行学 / 情绪基础

情绪是什么：定义与核心功能       当前学习
```

Core Question 用浅色 surface，不要巨大边框。

---

# 15. Answer 与 AI Evaluation

Answer textarea 约 140~180px 高。

提交后 AI Evaluation 必须直接出现在 Answer 下方。

推荐结构：

```text
AI 反馈

理解正确
✓ ...

还可以补充
△ ...

需要修正
! ...

为什么
...

边界条件
...
```

correct / missing / misconception 使用轻量 status block，不要每条一个 Card。

---

# 16. Next Action

AI Evaluation 下方固定有：

```text
下一步
```

主动作：

```text
[学习下一课]
```

可选：

```text
[继续补充]
[针对性重测]
[迁移挑战]
[查看知识结构]
```

主线继续不能被 Misconception / Transfer 淹没。

---

# 17. Learning Sidebar

减少多个独立 Card。

优先合并成一个：

```text
学习状态

认知
理解 · 发展中

Evidence
2 条

知识位置
个人心理运行学
/ 情绪基础
/ 情绪是什么

来源
暂无已验证来源

[查看知识结构]
```

Recent Change 可以放底部轻量显示。

---

# 18. Recent History

默认折叠，只显示最近一条摘要。

```text
最近学习记录 · 3

昨天 · 需要补充
“你已经理解……”
```

提供：

```text
展开全部
```

不要默认完整展开全部 AI response。

---

# 19. Knowledge Structure

保留左侧结构 + 右侧详情，但降低后台列表感。

Unit Header：

```text
情绪基础：理解情绪的来源与功能

3 / 8 已展开
━━━━━━━━━━━━░░░░░
```

---

# 20. Lesson Node

不要每个 Lesson 都是巨大矩形。

建议：

```text
● 情绪是什么：定义与核心功能
  核心 · 基础 · Depth 1
  当前学习

● 情绪的分类与生理基础
  核心 · Depth 2
  未接触

○ 情绪理论概览
  核心 · Depth 2
  未接触
```

可以使用轻微 timeline / connector，但不要做复杂 Canvas Graph。

---

# 21. Unit Expansion 状态

未展开：

```text
工程化实践与模块管理
0 个正式节点
[展开知识区域]
```

部分生成：

```text
工程化实践与模块管理
3 / 8 已生成
[继续生成节点]
```

全部：

```text
8 / 8 已生成
```

统一按钮样式。

---

# 22. Node Detail

右栏展示：

```text
Lesson Title
Role · Depth

[开始学习 / 继续学习]

前置知识
后续知识
深化
应用
相关
```

没有内容显示“暂无”，使用 secondary text。

---

# 23. Exploration

探索页不要做推荐 Feed。

每条方向：

```text
邻近探索

情绪分类与生理基础

为什么值得探索
...

来自
个人心理运行学 · 情绪基础

[去看看] [保存问题]
```

DirectionType 用统一 Tag：

- 邻近
- 跨领域
- 陌生知识

---

# 24. Sources

知识来源保持次级工作区。

Source list 信息：

```text
Title
Type
Verification
Evidence count
Grounding count
```

不要让 Source/Grounding 在 LearningView 抢占视觉中心。

---

# 25. Settings

按 Section 而不是大量大 Card：

```text
系统信息
AI 配置摘要
数据安全
诊断
```

操作：

```text
立即备份
导出数据
一致性检查
系统诊断
```

危险操作放 Danger Zone。

---

# 26. Empty / Loading / Error

统一 EmptyState 组件：

```text
icon
title
description
primary action
```

统一 AI Loading 文案：

- 正在建立领域地图…
- 正在展开起步区域…
- 正在准备第一批课程…
- 正在评价你的理解…

超过 10 秒：

```text
AI 正在处理，这一步可能需要一些时间。
```

统一错误：

```text
AI 响应超时

本次操作没有修改你的学习数据。

[重试]
```

不得出现裸 Failed to fetch。

---

# 27. Responsive

>= 1280：

- Full Sidebar
- Learning 72/28
- Course 3~5 columns

768~1279：

- compact sidebar
- Learning 65/35
- Course 2~3 columns

<768：

- Sidebar Drawer
- 全部单列

Learning mobile 顺序：

```text
Lesson
Question
Answer
AI Evaluation
Next Action
Misconception
Cognitive State
Knowledge Position
Source
History
```

---

# 28. Element Plus

可以继续使用 Element Plus，但统一覆盖：

- Button
- Tag
- Card
- Input
- Textarea
- Dialog
- Drawer
- Dropdown
- Skeleton

不要保留默认 Demo 风格。

---

# 29. 删除开发感 UI

用户 UI 中不要显示：

- MVP
- Phase 7 / Phase 10
- STATIC STRUCTURE
- DEBUG
- TEST

允许保留作为 eyebrow 的产品词：

- KNOWLEDGE WORLD
- EXPLORATION RADAR
- PERSONAL LEARNING SYSTEM

但不要出现开发术语。

---

# 30. 禁止修改业务语义

本任务不要修改：

- Cognitive State rules
- Evidence rules
- Transfer rules
- Misconception logic
- Exploration recommendation
- Curriculum generation
- Domain Initialization
- NextLesson
- CurrentLesson
- Grounding
- Backup / Restore

只有 UI 展示确实需要 readonly API adjustment 时，先汇报再改。

---

# 31. 优先提取共享组件

先审计并按需要提取：

```text
AppShell
PageHeader
CourseCard
StatusTag
EmptyState
AIRequestError
SectionHeader
LearningStateSummary
ProgressBar
```

不要建立巨大 design framework。

---

# 32. 视觉验收页面

必须逐页 self-review：

1. Home
2. Courses
3. Course detail
4. LearningView before answer
5. LearningView after AI evaluation
6. Knowledge Structure
7. Knowledge Structure selected node
8. Domain Initialization Stage 1
9. Domain Initialization Stage 2
10. Domain Initialization Stage 3
11. Exploration
12. Sources
13. Settings
14. Empty World
15. AI Error

---

# 33. 功能回归

UI 修改后回归：

- 创建学习领域
- Stage 1 / 2 / 3
- Apply
- 创建第二门课程
- 继续学习
- 切换 Lesson
- 下一课
- 展开 Unit
- 生成 Curriculum Draft
- Apply Lesson
- 提交 Answer
- AI Evaluation
- Misconception Recheck
- Transfer Challenge
- Exploration Open
- Backup
- Diagnostics

---

# 34. Build

执行：

```bash
npm run build
go test ./...
go build ./cmd/server
docker compose build app
git diff --check
```

如存在 lint：

```bash
npm run lint
```

---

# 35. 开始前先审计

先不要立即大面积修改。

请先汇报：

1. 当前全站 Layout / AppShell 结构
2. 哪些页面没有使用 Sidebar
3. 当前有哪些重复 Page Header
4. 当前 CourseCard 有几套实现
5. 当前状态颜色分别定义在哪里
6. 当前 LearningView 主要组件结构
7. 当前 Knowledge Structure 主要组件结构
8. 哪些 Element Plus 默认样式最明显
9. 推荐新增哪些 design tokens
10. 推荐提取哪些共享组件
11. 预计修改文件
12. 实施顺序

确认后再开始。

---

# 36. 推荐实施顺序

1. Audit
2. Design Tokens
3. AppShell
4. PageHeader
5. Buttons / Tags / Inputs
6. CourseCard
7. Home
8. Courses
9. LearningView
10. Knowledge Structure
11. Domain Initialization
12. Exploration
13. Sources
14. Settings
15. Empty / Loading / Error
16. Responsive
17. Visual regression
18. Functional regression
19. Build
20. Final report

---

# 37. 完成标准

必须满足：

```text
全站 Sidebar 一致
Page Header 一致
Course Card 一致
状态颜色一致
按钮层级一致
Typography 一致
圆角一致
Border 一致
Learning 主流程视觉优先
Knowledge Structure 不再像后台表单
没有大量 Card 套 Card
没有巨大无意义留白
没有开发 Phase 标签
```

最终请明确回答：

> LearnOS v0.1 是否已经具备统一、稳定、适合长期 Dogfooding 的 UI。
