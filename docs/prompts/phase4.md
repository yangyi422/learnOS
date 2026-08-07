# LearnOS Phase 4：知识世界与课程结构

请在当前 LearnOS 仓库中实现 **Phase 4：知识世界与课程结构（Knowledge World / Curriculum Graph）**。

已完成并验收：
- Phase 1：项目骨架
- Phase 2：持久化学习闭环
- Phase 3：DeepSeek AI 结构化认知评价

本阶段解决的问题是：

> 知识本身是什么结构？哪些知识存在前置、深化、应用和关联关系？如何让课程从线性 Lesson 列表变成存在分支、平行路线和汇合点的知识世界？

本阶段只构建 **知识世界的静态结构和基础查看能力**。不要提前实现个人认知状态、自动解锁、Agent 导航或最终知识地图视觉效果。

---

## 1. 开始前

先完整阅读并总结：
- AGENTS.md
- README.md
- docs/PRODUCT.md
- docs/ARCHITECTURE.md
- docs/DATABASE.md
- docs/ROADMAP.md
- 当前 model / repository / service / handler / router
- database migration / seed
- Phase 3 AI Provider / Evaluation 实现
- Vue Router、LearningView、课程档案页
- Dockerfile、docker-compose.yml
- 当前测试

先输出：
1. Phase 3 当前实际架构
2. Phase 4 预计新增/修改文件
3. 数据模型变化
4. Graph Service 设计
5. Seed 如何保护已有 Lesson 与历史数据
6. 简短实施顺序

确认与现有架构兼容后再编码。不要大范围重构。

---

# 2. 核心边界：World State 与 User State 分离

Phase 4 只处理 World State：

```text
这个知识是什么？
属于哪个 Unit？
是核心还是扩展？
需要哪些前置知识？
会通向哪些知识？
它是基础、深化还是应用？
课程 prerequisite 图是否存在环？
```

Phase 5 才处理 User Cognitive State：

```text
用户是否理解？
有什么掌握证据？
Recognize / Understand / Apply / Transfer 到哪一级？
是否具备进入下一知识的条件？
```

不要在 Phase 4 用 MasteryScore 阈值实现“解锁”。

---

# 3. 复用 Lesson 作为知识节点

当前已有：

```text
Course
CourseUnit
Lesson
```

Phase 4 优先让 **Lesson 直接成为知识图中的节点**。

不要新建一套与 Lesson 平行的 KnowledgeNode / ConceptNode，除非现有代码确实强烈要求。

---

# 4. 扩展 Lesson 静态结构元数据

根据现有 Lesson 增加少量字段；如果已有等价字段则复用：

```go
IsCore      bool   `gorm:"not null;default:true"`
ContentRole string `gorm:"size:32;not null;default:'core'"`
DepthLevel  int    `gorm:"not null;default:1"`
```

ContentRole 只允许：

```text
foundation
core
application
extension
```

含义：
- foundation：基础底座
- core：领域核心
- application：真实场景应用
- extension：扩展/深化/可选探索

注意：
`ContentRole` 是知识在课程中的角色，不代表用户掌握层级。

DepthLevel 暂定 1~5，仅作为课程设计层面的相对深度，不参与自动打分。

---

# 5. 新增 LessonRelation

建议模型：

```go
type LessonRelation struct {
	ID uint `gorm:"primaryKey"`

	CourseID uint `gorm:"not null;index"`

	FromLessonID uint `gorm:"not null;index"`
	ToLessonID   uint `gorm:"not null;index"`

	RelationType string `gorm:"size:32;not null;index"`

	CreatedAt time.Time
}
```

组合唯一：

```text
CourseID + FromLessonID + ToLessonID + RelationType
```

支持关系：

```text
prerequisite
extends
application
related
```

可以预留 cross_domain 枚举，但 Phase 4 不实现跨 Course 关系。

方向统一：

```text
FromLessonID = 起点 / 前置知识
ToLessonID   = 后续 / 被依赖知识
```

例如：

```text
A prerequisite B
```

表示学习 B 前，在课程结构上应该先理解 A。

---

# 6. Relation 约束

必须保证：
1. FromLesson != ToLesson
2. 两个 Lesson 都存在
3. 两个 Lesson 属于同一 Course
4. CourseID 一致
5. RelationType 是允许枚举
6. 完全相同 Relation 不重复
7. prerequisite 子图不可成环

不要允许：

```text
A → A
A → B → A
A → B → C → A
```

related / extends / application 不参与 prerequisite cycle 检查。

---

# 7. Curriculum Graph Validator

实现轻量图验证，不引入图数据库或大型算法库。

建议能力：

```go
ValidatePrerequisiteDAG(...)
TopologicalOrder(...)
```

DFS 或 Kahn 均可。

必须明确：

> Topological Order 只是合法顺序之一，不是唯一学习路线。

例如 A → B 与 A → C 时，A-B-C 和 A-C-B 都可能合法。

---

# 8. 营养学 Phase 4 Demo Knowledge Skeleton

不要一次生成完整营养学，只做足够验证：
- 分支
- 平行节点
- 汇合
- 核心/扩展
- 不同 Depth
- prerequisite / related / extends / application

必须复用当前真实 Lesson：

```text
口渴是否是可靠的饮水依据
```

不要重新创建它，不允许破坏现有 Phase 2 / 3 LearningTurn、MasteryRecord、Misconception、AIEvaluationRun 的关联。

建议结构：

## Unit A：水与体液基础

A1：
```text
水在人体中的基本作用
foundation
Depth 1
Core
```

A2：
```text
体液平衡是如何维持的
foundation
Depth 1
Core
```

关系：

```text
A1 prerequisite A2
```

## Unit B：饮水信号与日常判断

B1（复用现有）：
```text
口渴是否是可靠的饮水依据
core
Depth 2
Core
```

B2：
```text
日常饮水需求应该如何判断
core
Depth 2
Core
```

关系：

```text
A2 prerequisite B1
A2 prerequisite B2
B1 related B2
```

## Unit C：场景应用与扩展

C1：
```text
高温和运动为什么会改变补水需求
application
Depth 2
Core
```

C2：
```text
大量出汗后为什么需要关注电解质
application
Depth 3
Core
```

C3：
```text
年龄与疾病状态为什么会影响口渴信号
extension
Depth 3
Optional
```

C4：
```text
如何综合判断不同场景下的补水策略
application
Depth 4
Core
```

关系：

```text
A2 prerequisite C1
C1 prerequisite C2

B1 prerequisite C3

B1 prerequisite C4
B2 prerequisite C4
C2 prerequisite C4

C3 extends B1
C2 application A2
C4 application B1
C4 application B2
```

结构大致：

```text
A1
↓
A2
├────────────┬────────────┐
↓            ↓            ↓
B1           B2           C1
│            │            ↓
│            │            C2
├→ C3        │            │
│            │            │
└────────────┴──────┬─────┘
                    ↓
                    C4
```

---

# 9. Seed 必须保护历史

要求：
- 幂等
- 重启不重复插入 Unit / Lesson / Relation
- 不删除旧数据
- 不改变已有 Lesson ID
- 不重新创建 B1
- 不覆盖用户已有学习历史
- Relation 幂等插入
- 只补充缺失结构

禁止：

```text
清空 lessons
删除 data/
重新 seed 整库
人为创建假 LearningTurn
人为创建假 MasteryRecord
```

Phase 4 Seed 只负责 **世界结构**，不能伪造用户已经掌握 A1/A2。

---

# 10. Knowledge Graph API

新增：

```http
GET /api/v1/courses/:id/knowledge-graph
```

返回 DTO，不要直接暴露 GORM Model。

建议：

```json
{
  "data": {
    "course": {
      "id": 1,
      "name": "营养学"
    },
    "units": [],
    "nodes": [
      {
        "id": 1,
        "unit_id": 1,
        "title": "水在人体中的基本作用",
        "is_core": true,
        "content_role": "foundation",
        "depth_level": 1,
        "status": "pending"
      }
    ],
    "edges": [
      {
        "id": 1,
        "from_lesson_id": 1,
        "to_lesson_id": 2,
        "relation_type": "prerequisite"
      }
    ],
    "stats": {
      "node_count": 8,
      "edge_count": 12,
      "core_node_count": 7,
      "optional_node_count": 1,
      "root_node_count": 1,
      "leaf_node_count": 2,
      "max_depth_level": 4
    }
  }
}
```

要求：
- Course 不存在 → 404
- Node / Edge 排序 deterministic
- 不调用 AI
- 不动态生成知识结构
- status 可以兼容返回现有 Lesson.Status，但不要用它推断 mastery

root / leaf 以 prerequisite 子图计算。

---

# 11. Lesson Relation API

新增：

```http
GET /api/v1/courses/:id/lessons/:lessonId/relations
```

返回当前 Lesson 的结构关系，例如：

```json
{
  "data": {
    "lesson": {
      "id": 3,
      "title": "口渴是否是可靠的饮水依据"
    },
    "prerequisites": [],
    "next_lessons": [],
    "extensions": [],
    "applications": [],
    "related": []
  }
}
```

目的：

> 点击一个节点后知道“它从哪里来，会通向哪里”。

本阶段不要计算 `is_unlocked` / `can_learn`。

---

# 12. Service / Repository 分层

新增独立 KnowledgeGraphService，职责建议：

```text
GetCourseKnowledgeGraph
GetLessonRelations
ValidateCourseGraph
TopologicalOrder
CalculateGraphStats
```

Repository 负责：
```text
ListLessonsByCourse
ListRelationsByCourse
ListRelationsForLesson
```

Repository 不做 DFS / graph stats / 业务解释。

Handler 不直接操作 GORM。

---

# 13. 基础知识结构页面

新增：

```text
/courses/:id/map
```

这是 **Phase 4 的结构验收页**，不是最终知识地图。

禁止引入：
- D3
- Cytoscape
- Vue Flow
- Canvas / WebGL 图布局
- 大型图可视化依赖

Phase 9 才做真正《荒野之息》式的知识世界视觉地图与足迹。

Phase 4 页面只需清楚展示结构。

顶部示例：

```text
营养学 · 知识结构

8 个知识节点
7 个核心
1 个扩展
12 条关系
```

按 Unit 分组展示 Lesson 卡片：

```text
口渴是否是可靠的饮水依据

core
核心
Depth 2

前置：
体液平衡是如何维持的

后续：
年龄与疾病状态为什么会影响口渴信号
如何综合判断不同场景下的补水策略

相关：
日常饮水需求应该如何判断
```

页面应该至少能看出：
- A2 有多个分支
- C4 有多个 prerequisite 汇合

可以用列表、列布局、badge、简单 CSS connector，但不要追求最终地图视觉。

---

# 14. 增加入口

首页或课程档案合适位置增加：

```text
查看知识结构
```

跳转：

```text
/courses/:id/map
```

当前 LearningView 可增加轻量：

```text
知识位置

前置：
体液平衡是如何维持的

可延伸方向：
年龄与疾病状态为什么会影响口渴信号
如何综合判断不同场景下的补水策略

[查看知识结构]
```

不要写“下一课”，因为路线可能不唯一。

---

# 15. Phase 4 严禁实现“自动解锁”

本阶段不要增加：

```text
is_unlocked
is_available
can_learn
next_best_lesson
recommended_next_lesson
```

不要写：

```text
MasteryScore >= 0.7 → unlock
```

因为“用户是否具备进入某节点的条件”属于 Phase 5。

本阶段只把 prerequisite 明确表示出来。

---

# 16. 不自动切换当前 Lesson

禁止：
- AI 自动决定下一课
- Graph 自动切 CurrentLessonID
- 提交答案后自动前进

Phase 3 学习闭环保持原行为。

---

# 17. 不让 DeepSeek 生成知识图

虽然 Phase 3 已接 DeepSeek，但 Phase 4 Knowledge Skeleton 必须是：
- 明确 Seed
- 人工定义
- 可验证
- deterministic

不要：
- 让 DeepSeek 自动生成营养学全图
- 自动生成 prerequisite
- 自动新增 Unit / Lesson

AI 扩展知识世界属于后续阶段。

---

# 18. 测试

至少覆盖：

## Relation / Repository
1. Relation 持久化
2. 重复 Relation 不增加
3. ListRelationsByCourse 正确
4. ListRelationsForLesson 正确

## DAG Validator
5. A→B, A→C, B→D, C→D 通过
6. A→B→A 检测为环
7. A→B→C→A 检测为环
8. A→A 被拒绝
9. related / extends / application 不参与 prerequisite cycle 判断

## Graph Service
10. 返回全部 nodes / edges
11. deterministic 排序
12. root / leaf 统计正确
13. core / optional 统计正确
14. max depth 正确
15. Lesson relation 查询正确
16. Course 不存在正确报错

## Seed
17. Seed 两次 Lesson 数不增加
18. Seed 两次 Relation 数不增加
19. 现有“口渴是否是可靠的饮水依据” ID 不变化
20. 旧 LearningTurn / MasteryRecord / Misconception / AIEvaluationRun 不被删除

## Handler
21. GET knowledge-graph 成功
22. Course 不存在 404
23. GET lesson relations 成功
24. lesson 不属于 course 被拒绝

---

# 19. 前端

继续：
- Vue 3
- TypeScript
- Element Plus

API 放在 `web/src/api/`。

定义明确类型：
- KnowledgeGraph
- KnowledgeGraphNode
- KnowledgeGraphEdge
- KnowledgeGraphStats
- LessonRelations

不要无理由使用 any。

旧 Phase 2/3 数据缺少新字段时不能导致页面崩溃。

---

# 20. 数据兼容

继续 GORM AutoMigrate。

必须保证升级后：
- Course 在
- 旧 Lesson ID 不变
- Phase 2 Mock 历史在
- Phase 3 DeepSeek 历史在
- LearningTurn 在
- MasteryRecord 在
- Misconception 在
- AIEvaluationRun 在

严禁清库。

---

# 21. 文档

更新：
- README.md
- docs/PRODUCT.md
- docs/ARCHITECTURE.md
- docs/DATABASE.md
- docs/ROADMAP.md

ROADMAP 标记：

```text
Phase 1 ✅
Phase 2 ✅
Phase 3 ✅
Phase 4 ✅（完成后）
```

明确 Phase 5 才处理：
- 个人认知状态
- 掌握证据聚合
- Recognize / Understand / Apply / Transfer
- 是否具备进入某节点的条件

---

# 22. Phase 4 明确不做

不要实现：
- AI 自动生成课程
- AI 修改知识图
- AI 自动下一课
- 自动解锁
- 个性化路线推荐
- 个人认知状态机
- 长期 Mastery 算法
- 认知演化
- 认知足迹
- 误区网络
- 迁移测试生成器
- 探索雷达
- 陌生知识按钮
- 问题池
- 来源可信度系统
- RAG / Vector DB
- 在线搜索 / Citation
- Agent orchestration
- 复习调度
- 最终 Graph Canvas
- Markdown / Obsidian 同步
- 多用户
- 大范围 UI 重构

---

# 23. Phase 4 完成标准

必须同时满足：

1. LessonRelation 可表达 prerequisite / extends / application / related。
2. prerequisite 图保证 DAG，并有自动测试。
3. 营养学 Demo World 存在分支、平行节点、汇合、核心/扩展、不同 Depth。
4. 复用现有真实 B1 Lesson，不破坏 Phase 2 / 3 历史。
5. GET knowledge-graph 可读取完整结构。
6. GET lesson relations 可读取上下游关系。
7. `/courses/:id/map` 可以人工查看基础结构。
8. Phase 3 DeepSeek 学习评价继续正常。
9. Phase 4 不调用 AI 修改世界结构。
10. 不伪造用户学习历史。

---

# 24. 人工验收

完成后应给出测试步骤：

```text
1. docker compose up -d --build app

2. 打开 LearnOS
3. 确认 Phase 3 历史仍然存在

4. 点击“查看知识结构”
5. 进入 /courses/1/map

6. 确认可以看到：
   水在人体中的基本作用
   ↓
   体液平衡是如何维持的

7. 确认 A2 后出现多个分支：
   口渴
   日常饮水
   高温运动

8. 确认 C4 有多个前置：
   B1
   B2
   C2

9. 查看 B1：
   能看到 prerequisite / related / 后续关系

10. 返回 /courses/1/learn
11. DeepSeek 回答评价仍然正常

12. 刷新 / 重启容器
13. 确认历史和 Knowledge Graph 都存在
```

---

# 25. 构建验证

至少执行：

```bash
go test ./...
go build ./cmd/server
npm run build
docker compose build app
git diff --check
```

Go 代码执行 gofmt。

不得删测试换取通过。

---

# 26. 最终工作顺序

1. 阅读仓库
2. 总结 Phase 3
3. 列修改文件与方案
4. 扩展 Lesson
5. 新增 LessonRelation + migration
6. 幂等 Seed
7. Repository
8. Graph Validator
9. KnowledgeGraphService
10. DTO / Handler / Router
11. 后端测试
12. Vue API 类型
13. `/courses/:id/map`
14. 增加入口
15. LearningView 增加轻量知识位置
16. 测试 / 构建 / 修复
17. 更新文档
18. 最终汇报

最终报告必须列出：
- 完成内容
- 修改文件
- LessonRelation 设计
- Relation 方向定义
- DAG 校验方法
- Seed Knowledge Skeleton
- API
- UI
- 测试结果
- Docker 构建结果
- 历史数据兼容结果
- 人工验收步骤
- 明确未实现的 Phase 5+ 内容

现在开始。

先不要立即编码。先阅读仓库并汇报 Phase 3 实际结构、Phase 4 文件清单、模型变化、Graph Service 设计和数据保护方案，确认兼容后再实现。
