# LearnOS 重构前测试基线

本文件记录 2026-09-20 的实现审计结果。范围仅包含现有行为、数据流和回归保护；没有调整产品规则，也没有迁移、删除或重置现有学习数据。

## 项目结构与测试栈

| 层 | 位置 | 当前实现 |
| --- | --- | --- |
| 启动与路由 | `cmd/server`、`internal/httpapi` | Go、Gin，统一 `/api/v1`；handler → service → repository |
| 领域服务 | `internal/service` | 课程、学习、认知、误区、探索、课程蓝图、来源和系统诊断 |
| 事实与持久化 | `internal/model`、`internal/repository` | GORM + SQLite；Markdown/JSON 是导出，不是事实来源 |
| SPA | `web/src` | Vue 3、TypeScript、Vue Router、Element Plus；没有 Pinia/Vuex |
| 页面状态 | 各 `views/*.vue` | `ref` / `computed` 和路由参数组成的页面本地状态；跨页事实每次从 API 重读 |
| 前端 API 层 | `web/src/api` | 统一 `request()`，按业务域拆分封装；DTO 在 `web/src/types` |
| 后端测试 | `internal/**/*_test.go` | Go `testing`，临时 SQLite 或内存 SQLite |
| 前端单元测试 | `web/src/**/*.test.ts` | Vitest；当前保护知识路径布局的节点一致性 |
| 浏览器回归 | `web/e2e` | Playwright + Chromium；API 全部拦截，不接触真实数据库 |

测试命令：

```bash
go test ./...
cd web
npm run test
npm run typecheck
npm run build
npm run test:e2e
```

## 路由、页面与接口

| 功能 | 前端路由 / 源码 | 主要 API |
| --- | --- | --- |
| 学习首页 | `/`，`DashboardView.vue` | `GET /courses`、`GET /courses/:id/current-lesson`、`GET /courses/:id/cognitive-states`、探索雷达 |
| 课程档案入口 | `/courses`，`CoursesView.vue` | `GET /courses`、`DELETE /courses/:id` |
| 单课程档案 | `/courses/:id/archive`，`CourseArchiveView.vue` | curriculum coverage、draft list/apply/reject、grounding coverage |
| 当前 Lesson 学习页 | `/courses/:id/learn`，`LearningView.vue` | current/指定 Lesson、answers、turns、cognitive state、relations、misconceptions、challenges、next lesson |
| 知识路径/列表 | `/courses/:id/map`，`KnowledgeGraphView.vue`；`KnowledgeTreePath.vue` / `KnowledgePath.vue` | `GET /courses/:id/knowledge-graph`、cognitive states、lesson relations；显式开始学习才 POST current-lesson |
| 误区网络 | `/courses/:id/misconceptions`，`MisconceptionNetworkView.vue` | `GET /courses/:id/misconception-network` |
| 探索空间 | `/exploration/questions`，`ExplorationView.vue` | radar、unfamiliar、directions、questions、history |
| 新建学习领域 | `/domains/new`，`DomainInitializationView.vue` | domain draft 的 create/get/regenerate/expand/generate/apply |
| 系统设置 | `/settings`，`SettingsView.vue` | diagnostics、consistency、backup、export、AI config |

桌面和移动导航都由 `AppShell.vue` 驱动。桌面使用侧栏，移动端使用 Element Plus Drawer；导航开关是组件本地状态，切页由 Vue Router 完成。

## 关键数据模型和数据流

### 课程、领域与知识节点

- `Course` 是已应用的学习领域，持有 `CurrentUnitID`、`CurrentLessonID`、展示用 `Progress` 和最近学习时间。
- `CourseUnit` / `Lesson` / `LessonRelation` 是正式、可学习的课程图；关系类型包含 prerequisite、extends、application、related。
- `DomainInitializationDraft` 保存新领域三阶段中间结果：Skeleton → Starter Expansion → Initial World → Apply。Apply 前不创建正式 Course/Lesson，也不创建个人认知数据。
- `CurriculumBlueprint` / Unit / Lesson / Relation 描述学科应覆盖的结构；`CurriculumDraft` 是需人工审核的正式节点增量。
- Knowledge Graph DTO 将正式 Lesson 映射为 `node_type=lesson`、稳定键 `lesson:<id>`；尚未映射到有效正式 Lesson 的 BlueprintLesson 映射为 `node_type=blueprint`、稳定键 `blueprint:<id>`。`content_role`（foundation/core/application/extension）与节点类型相互独立。
- 路径视图与列表视图读取同一个 `KnowledgeGraph.nodes` 数组。路径视图只通过 `knowledgePathLayout.ts` 按 prerequisite DAG 排列 rank，不创建第二份业务数据。

### 学习和认知状态

```text
LearningView 提交回答
  → Course Service 校验 Course / Lesson
  → AI Provider 返回结构化评价并通过 Schema/目标层级校验
  → 同一事务写 LearningTurn、MasteryRecord、Misconception、CognitiveEvidence/Event、AIEvaluationRun
  → CognitiveState Service 汇总当前认知层级与状态
```

- `MasteryRecord.mastery_score` 是最近一次通过校验的评价分数，不是多次回答的平均值；回答数/错误数累计，`needs_review` 跟随最近一次评价。
- 认知层级依次为 `unseen → exposed → recognize → understand → apply → transfer`，层级只升不降。负面或部分结果会限制本次可证明的最高层级；矛盾通过 `needs_review` 表达。
- `CognitiveState.status` 使用 unknown、developing、stable、needs_review。学习历史保留当次分数和评价快照。
- 首页先按最近学习时间、是否有 CurrentLesson 和 progress 选取 focus course，再调用该课程的 current-lesson。学习页同样从 current-lesson 端点读取，因此两页应共享同一个 Course 指针。

### 进度与覆盖度

- 课程生成阶段由 `DomainInitializationDraft.status` 表示：`skeleton_confirmed → starter_expanded → world_ready → applied`。
- Blueprint Unit 展开状态独立记录为 unexpanded / expanding / expanded；正式节点生成仍需 CurriculumDraft 审核和 Apply。
- Curriculum coverage 只按 `BlueprintLesson.AppliedLessonID` 是否指向实际存在的正式 Lesson 计算 covered。核心覆盖率是 `core_covered / core_total × 100`，总体覆盖率是全部重要性 covered / total；Unit 展开进度同理为 applied/blueprint lesson 数量。
- `Course.progress` 当前不是实时学习进度：示例营养学种子固定为 48，新应用课程为 0，代码中没有随回答或认知状态更新该字段。这是现状风险，重构时不能把它误当成由掌握度动态计算的指标。
- 掌握度与课程覆盖度是不同维度：前者属于个人认知状态，后者属于学科蓝图和正式 Lesson 的结构映射。

### 探索来源归属

- Exploration candidate 的 target course 取自 target Lesson 的 `CourseID`；API 同时返回扁平的 `target_course_id` 和嵌套 course/lesson DTO。
- question 的 `course_id` 是来源领域，`source_lesson_id` 是来源节点；target 字段表示探索目标。
- 新增回归测试把 API 中的来源/目标 ID 与 SQLite 中 Lesson 的真实 `CourseID` 逐项比对。当前 `directionView`/`questionView` 会检查引用存在，但没有独立拒绝“存在但 course_id 与 lesson.CourseID 不一致”的损坏记录；数据库也没有跨表约束保证这点，属于后续一致性风险。

### API Key 边界

- AI 配置写入后，API 只返回 `api_key_configured`，不返回 key；设置页保存后清空输入框。
- 回归测试使用哨兵 key 检查 PATCH/GET 响应、Gin 输出、标准应用日志、浏览器 console 和页面文本。
- 当前 key 仍由单用户自托管实例保存在本地 SQLite 配置记录中；本基线验证的是“不进入接口响应、日志或页面明文”，并不等同于数据库静态加密。

## 回归用例

| 需求 | 保护层 |
| --- | --- |
| 空的新建领域表单不产生有效数据 | Go service：空白输入返回 `ErrDomainInputInvalid`，Draft/Course/Blueprint/Lesson 均为 0；Playwright：字段下显示本地错误且不发请求 |
| 移动导航可开、关、切页 | Playwright 375px：按钮打开后焦点进入菜单，Escape/遮罩关闭并恢复按钮焦点，切页后自动收起并聚焦页面标题 |
| 路径/列表节点一致 | Vitest：路径布局保留原节点对象/字段；Playwright：同一节点两种视图的详情文本一致 |
| 探索来源领域与节点归属一致 | Go handler：API direction 的 source/target ID、嵌套 DTO 与 SQLite Lesson/Course 逐项一致 |
| 首页和课程详情当前 Lesson 一致 | Go handler：course list 指针与 current-lesson 端点一致；Playwright：首页标题与学习页标题一致 |
| 375/768/1024/1440 无横向溢出 | Playwright：逐档访问 9 个核心路由，比较 document scrollWidth/clientWidth |
| API Key 不泄漏 | Go handler + Playwright：响应、Gin/应用日志、console、页面文本均不得出现哨兵值 |
| 探索选择器可见且隔离切换 | Playwright：375/768/1440px 均显示标签和当前领域；切换时显示加载状态、清空旧卡片并忽略过期响应 |

## 当前已知问题与未覆盖风险

1. 完整 Go service 测试目前有既有阻断：`CurriculumDraft.ChangeSetJSON` 的 GORM 默认列名是 `change_set_json`，但 `CurriculumRepository.FinalizeDraftGeneration` 更新 `change_set`，4 个 curriculum draft 测试报 `no such column: change_set`。本轮未修改生产映射。
2. GORM 测试默认 logger 会大量打印预期的 `record not found`，降低失败信号可读性；后续可在测试夹具统一设为 silent，但需避免遮蔽真实 SQL 错误。
3. Playwright 使用 API contract mocks，适合稳定验证页面行为且不会污染现有数据；它不替代连接临时 SQLite 的完整前后端集成测试。
4. `Course.progress` 是静态字段；重构进度展示前必须先明确产品语义和唯一计算来源。
5. Exploration 的字段一致性主要由候选构造保证，损坏的历史行缺少数据库外键/服务层交叉校验。
6. API Key 未进入响应和已覆盖日志，但没有对反向代理、崩溃转储、操作系统交换区或本地 SQLite 静态加密做自动化验证。
7. 前端生产包目前约 1.09MB（未压缩 JS），构建有大 chunk 警告；本轮不做路由懒加载或拆包。

## 重构准入建议

- 先修复 curriculum draft 列名/迁移一致性，让 `go test ./...` 全绿，并增加针对真实迁移后 schema 的断言。
- 保持本文件中的回归矩阵为最小门禁；重构路由、状态管理或 API DTO 时同步更新 contract mocks 和 Go handler 测试。
- 若要改变 progress/mastery/coverage 的定义，应先增加新旧公式的明确验收样例，不能把三者合并为一个百分比。

## 本次验证结果

- `npm run test`：通过，7 个 Vitest 用例。
- `npm run test:e2e`：通过，15 个 Playwright 用例。
- `npm run typecheck`：通过。
- `npm run build`：通过；保留大于 500kB 的 chunk 警告。
- `go test ./...`：除上述 `change_set` 列名导致的 4 个既有 curriculum draft 失败外，其余包通过；本轮新增的 service/handler 回归全部通过。
