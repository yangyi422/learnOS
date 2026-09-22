# LearnOS v0.1 Knowledge Map Graph
## 地图 / 列表双视图 · 可视化知识世界 · 渐进式点亮

当前 LearnOS v0.1 的核心功能已经基本完成：

- 多学习领域
- Progressive Domain Initialization
- Curriculum Blueprint / Coverage / Draft / Apply
- Knowledge World Expansion
- CurrentLesson / NextLesson
- Personal Cognitive State
- Misconception / Transfer
- Exploration
- Source / Grounding
- Dogfooding Readiness
- 全站 UI 正在做最终统一

现在希望在正式部署到云服务器之前，
完成 LearnOS v0.1 最后一个视觉 / 交互增强：

# Knowledge Map Graph

目标：

> 把当前“知识结构”页面从纯纵向列表，升级为真正可视化的知识世界地图。

但必须保留现有列表视图，并且不能重写现有知识模型。

最终形态：

```text
知识结构

[地图视图] [列表视图]

地图视图：
- 可拖动
- 可缩放
- 节点可点击
- 显示 prerequisite / extends / application / related
- 显示当前学习状态
- 显示 Blueprint 尚未生成节点
- 点击节点右侧显示详情
- Blueprint-only 节点可触发“展开知识区域”
```

---

# 一、核心原则

本任务只做：

```text
Knowledge Structure Visualization
+
Graph Interaction
+
Map/List 双视图
```

不要修改：

- Course 业务语义
- Curriculum 生成逻辑
- Blueprint 数据结构
- LessonRelation 业务定义
- Cognitive State 更新规则
- NextLesson 规则
- CurrentLesson 规则
- Exploration 逻辑
- Grounding 逻辑

可以新增：

- 只读 Graph DTO / API
- 前端 Graph mapping
- Layout service / helper
- Graph-specific UI state

---

# 二、技术方案

前端优先使用：

```text
@vue-flow/core
@vue-flow/background
@vue-flow/controls
```

可选：

```text
@vue-flow/minimap
```

默认不显示 minimap，除非图很大时确实有帮助。

自动布局建议：

```text
dagre
```

或：

```text
elkjs
```

优先选当前仓库更轻、更稳定的方案。

默认布局方向：

```text
Left → Right
```

表达：

```text
基础
→ 理解
→ 深化
→ 应用
```

不要自己实现 Graph Engine。

---

# 三、地图 / 列表双视图

当前 Knowledge Structure 页面保留。

顶部新增：

```text
[地图] [列表]
```

默认建议：

```text
地图
```

如果用户上一次选择列表，可以 localStorage 保存：

```text
knowledge-map-view=graph|list
```

不要写数据库。

---

# 四、页面结构

推荐：

```text
Breadcrumb

个人心理运行学
知识结构

5 个正式节点 · 27 个规划节点

[地图] [列表]              [返回学习]
```

地图模式主体：

```text
┌──────────────────────────────────────┬─────────────────┐
│                                      │ NODE DETAIL     │
│                                      │                 │
│            GRAPH CANVAS              │ 情绪是什么      │
│                                      │                 │
│                                      │ [继续学习]      │
│                                      │                 │
│                                      │ 前置知识        │
│                                      │ 后续知识        │
│                                      │ 应用            │
│                                      │ 相关            │
└──────────────────────────────────────┴─────────────────┘
```

Graph：

```text
70~75%
```

Detail Rail：

```text
25~30%
```

---

# 五、节点类型

至少区分：

## Applied Lesson
正式 Lesson，来自 Lesson。

## Blueprint-only Lesson
存在于 CurriculumBlueprintLesson，但：

```text
AppliedLessonID = null
```

表示地图里已经知道这里存在一个知识点，但它还没有正式生成。

## Current Lesson
正式 Lesson 且：

```text
Lesson.ID == Course.CurrentLessonID
```

---

# 六、节点视觉

保持克制，不做巨大彩色卡片。

建议：

```text
width 200~240px
```

示例：

```text
● 情绪是什么：定义与核心功能

基础 · Depth 1

当前学习
```

状态建议：

```text
● current
● learned / stable
○ unseen
◇ blueprint only
△ needs review
```

颜色：

- 当前：蓝色边框 + 浅蓝背景
- stable：低饱和绿色
- developing：蓝色
- needs_review：amber
- unseen / unknown：灰色
- blueprint-only：虚线边框、低对比度、空心符号

不要使用高饱和 Skill Tree 风格。

---

# 七、Node Content

Applied Lesson 至少可映射：

```text
Title
ContentRole
DepthLevel
Cognitive Level
Cognitive Status
```

但节点本身只展示关键内容，例如：

```text
基础 · Depth 1
理解 · 发展中
```

不要塞很多 Tag。

---

# 八、Blueprint-only Node

显示：

```text
◇ 情绪调节策略

Blueprint
尚未生成
```

点击右侧：

```text
情绪调节策略

这个知识节点已存在于课程蓝图中，
但尚未生成正式学习内容。

[展开知识区域]
```

如果现有生成机制只支持按 Unit batch：

禁止新增单节点生成系统。

直接复用：

```text
Unit Expansion
+
Curriculum Draft
```

---

# 九、Unit Grouping

Graph 不能完全忽略 Unit。

推荐轻量：

```text
swimlane / group background / lane label
```

例如：

```text
┌ 情绪基础 ──────────────────────┐
  ● 情绪是什么 → ○ 情绪分类
└───────────────────────────────┘
```

不要强边框。

如果 Vue Flow group node 成本高，v0.1 可先用：

```text
lane label + layout spacing
```

但必须看得出 Unit 分区。

---

# 十、关系类型

支持：

```text
prerequisite
extends
application
related
```

默认不要全部显示。

默认：

```text
☑ prerequisite
☐ extends
☐ application
☐ related
```

顶部提供 Relation Filter。

---

# 十一、Edge Visual

建议：

## prerequisite
实线 + 箭头

## extends
细线 / 轻蓝

## application
虚线

## related
浅灰虚线

不要制造蜘蛛网。

---

# 十二、关系方向

必须严格遵循现有 LessonRelation 语义。

如果语义是：

```text
A prerequisite of B
```

Graph：

```text
A → B
```

开始前必须审计：

```text
FromLessonID
ToLessonID
RelationType
```

以及 BlueprintRelation 的方向定义。

---

# 十三、默认布局

优先：

```text
Left → Right
```

同 Unit 内：

```text
Depth 1 → Depth 2 → Depth 3
```

如果 prerequisite DAG 更合理，以 DAG 决定 rank。

不要按数据库 ID 简单排序。

不同 Unit 从上到下排列。

---

# 十四、Cross-course Relations

如果存在 CrossCourseLessonRelation：

v0.1 默认不要全部显示。

可预留：

```text
显示跨领域关联 [off]
```

如果实现成本大，本次只在 Node Detail 显示：

```text
3 个跨领域关联
```

不要阻塞主任务。

---

# 十五、Node Click

点击节点：

```text
selectedNode = node
```

右侧详情更新。

不能自动修改 CurrentLesson。

只有显式：

```text
开始学习
继续学习
```

才切换 CurrentLesson。

---

# 十六、Node Detail Rail

复用现有右侧详情。

Applied Lesson：

```text
NODE DETAIL

情绪是什么：定义与核心功能

基础 · Depth 1

认知
理解 · 发展中

[继续学习]

前置知识
后续知识
深化内容
应用场景
相关知识
```

Blueprint-only：

```text
BLUEPRINT NODE

情绪调节策略

核心 · Depth 2

尚未生成正式 Lesson

[展开知识区域]
```

---

# 十七、Fit View

进入地图：

```text
fitView
```

如果存在 CurrentLesson，优先让 CurrentLesson 附近区域进入视口。

不要默认 zoom 太近。

Controls：

```text
Zoom In
Zoom Out
Fit View
```

Canvas 背景：

```text
very subtle dots/grid
```

不要赛博深色背景。

---

# 十八、Large Graph Strategy

当节点较多时，至少提供：

```text
Unit Filter
```

顶部：

```text
全部区域
情绪基础
情绪识别
情绪调节
...
```

选择 Unit 后只显示相关区域。

关系 Filter 也必须有。

Status Filter 可延后。

---

# 十九、Blueprint Nodes Toggle

顶部：

```text
☑ 显示规划节点
```

默认：

```text
on
```

这是 LearnOS 和普通课程树的关键区别：

> 可以看到尚未走到、甚至尚未正式生成的世界边界。

---

# 二十、Cognitive Overlay

Applied Lesson 轻量叠加 CognitiveState。

不要做大面积 Heatmap。

Knowledge Map 仍然首先是结构地图。

Grounding 不在 Graph 节点上展示，只在详情里显示摘要。

---

# 二十一、Graph API

如果当前 Knowledge Map API 已经足够，前端直接转换，不新增 API。

如果不足，可新增只读：

```http
GET /api/v1/courses/:courseId/knowledge-graph
```

建议 DTO：

```json
{
  "course": {},
  "units": [],
  "nodes": [
    {
      "id": "lesson:123",
      "node_type": "lesson",
      "lesson_id": 123,
      "blueprint_lesson_id": 88,
      "unit_id": 10,
      "title": "...",
      "content_role": "foundation",
      "depth_level": 1,
      "is_core": true,
      "is_current": true,
      "cognitive_level": "understand",
      "cognitive_status": "developing"
    },
    {
      "id": "blueprint:99",
      "node_type": "blueprint",
      "blueprint_lesson_id": 99,
      "unit_id": 11,
      "title": "...",
      "depth_level": 2,
      "is_core": true
    }
  ],
  "edges": [
    {
      "id": "...",
      "source": "...",
      "target": "...",
      "relation_type": "prerequisite"
    }
  ]
}
```

后端只返回领域语义，不返回：

```text
x / y
VueFlowNode
CSS class
Handle position
```

Layout 属于前端。

---

# 二十二、Blueprint / Lesson 去重

如果：

```text
BlueprintLesson.AppliedLessonID != null
```

Graph 只显示正式 Lesson node。

不要同时出现 Blueprint node + Lesson node。

Blueprint 信息 merge 到 Lesson node。

如果 BlueprintRelation 和 LessonRelation 表达同一条已 Applied 关系：

只显示一条 Edge。

优先 LessonRelation。

Node ID：

```text
lesson:<id>
blueprint:<id>
```

---

# 二十三、地图状态

只存前端：

```text
selectedNode
selectedUnit
enabledRelations
showBlueprintNodes
viewport（可选）
```

不要写数据库。

---

# 二十四、列表视图保留

保留当前已经做好的：

```text
Unit
Vertical Node Path
Right Detail Rail
```

地图是增强，不是替代。

两种视图必须共用：

- Node Detail
- Start Learning
- CurrentLesson
- Unit Expansion
- Cognitive status mapping

不要维护两套业务规则。

---

# 二十五、Responsive

Desktop：

```text
Graph + Rail
```

Tablet：

```text
Graph full width
Node Detail Drawer
```

Mobile：

默认列表视图。

用户仍可切地图。

---

# 二十六、Performance

目标：

```text
100 nodes
```

仍能基本流畅拖动 / 缩放。

避免：

- mousemove 请求 API
- select node 重新自动 layout
- 大量深层 watch

Layout 只在：

```text
graph data change
filter change
```

时运行。

---

# 二十七、Loading / Error

Loading：

```text
正在构建知识地图…
```

失败：

```text
知识地图加载失败。

[重试]
[切换列表视图]
```

Graph 失败不能影响 List View 和学习流程。

---

# 二十八、Empty State

Course 存在但 Lesson / Blueprint 为空：

```text
这个知识世界还没有展开。

[展开第一个知识区域]
```

---

# 二十九、视觉方向

继续沿用当前已经确认的：

```text
Quiet Intelligence UI
Editorial Knowledge Workspace
```

不要：

- neon
- cyberpunk
- 游戏技能树
- 高饱和 mind map
- 复杂 Canvas 装饰

它应该像：

> 一张安静、可以工作的认知地图。

---

# 三十、测试要求

至少覆盖：

## DTO
1. Applied Lesson node
2. Blueprint-only node
3. Applied Blueprint 不重复
4. CurrentLesson flag
5. Cognitive overlay
6. LessonRelation
7. BlueprintRelation
8. Edge deduplication

## Graph
9. render nodes
10. render edges
11. fit view
12. select node
13. detail rail
14. relation filter
15. unit filter
16. blueprint toggle
17. current node style

## Actions
18. Start Learning
19. Continue Learning
20. Blueprint-only expansion
21. click node does not change CurrentLesson
22. list/map switch

## Regression
23. existing list view
24. NextLesson
25. Unit Expansion
26. Curriculum Draft
27. Course isolation

---

# 三十一、构建验证

执行：

```bash
npm run build
go test ./...
go build ./cmd/server
docker compose build app
git diff --check
```

如果引入 npm dependency：

确保 package-lock.json 更新。

---

# 三十二、文档

更新：

```text
docs/PRODUCT.md
docs/ARCHITECTURE.md
docs/ROADMAP.md
```

可新增：

```text
docs/KNOWLEDGE_MAP.md
```

记录：

- Map View
- List View
- Node types
- Edge types
- Cognitive overlay
- Blueprint-only semantics

---

# 三十三、明确不做

不做：

- 3D Knowledge World
- 自研 Canvas 引擎
- Force simulation
- Graph DB / Neo4j
- Vector DB
- RAG
- AI 自动重新排图
- 自动关系推断
- 用户拖动后永久保存位置
- 多人协作

---

# 三十四、开始前先审计

现在先不要编码。

请阅读：

- 当前 Knowledge Map / Knowledge Structure 实现
- LessonRelation
- CurriculumBlueprintRelation
- CurrentLesson
- CognitiveState
- Unit Expansion
- Curriculum Draft / Apply

然后汇报：

1. 当前知识结构页的数据 API
2. LessonRelation 数据结构与方向语义
3. BlueprintRelation 数据结构与方向语义
4. AppliedLessonID mapping 机制
5. 当前是否已经能拿到 CognitiveState
6. 当前列表视图组件结构
7. Node Detail 是否可直接复用
8. 是否需要新增 knowledge-graph API
9. Vue Flow + dagre / elk 哪个更适合当前仓库
10. Blueprint-only node 如何与 Applied Lesson 去重
11. Unit grouping 推荐方案
12. 预计新增 npm 依赖
13. 预计修改文件
14. 实施顺序

请用 ASCII wireframe 给出：

- Graph View
- Node Detail
- Relation Filter
- Unit Filter

确认后再开始实现。
