# LearnOS Roadmap

- [x] 登录区分凭据失败与服务故障，避免重复提交；输入框聚焦使用单层细边框

## Phase 1：可运行骨架（本次）

- [x] Go + Gin 服务
- [x] SQLite + GORM
- [x] Vue 3 + Element Plus
- [x] 课程列表 API
- [x] 营养学种子课程
- [x] Docker 部署骨架
- [x] 部署显式更新远端引用并校验 Actions 提交 SHA，兼容旧版 Git
- [x] 单用户 Basic Auth
- [x] 多用户会话登录与管理员创建用户（课程级数据隔离）
- [x] 按角色显示用户管理入口与退出登录

## Phase 2：不依赖 AI 的最小学习闭环（已完成）

- [x] course_units、lessons、learning_turns、mastery_records 与 misconceptions 数据模型
- [x] 当前课程、模块和知识点的幂等初始化
- [x] 当前问题页面与首页继续学习跳转
- [x] 提交用户回答与安全校验
- [x] 固定模拟评价与反馈展示
- [x] learning_turns 持久化
- [x] 在事务中创建或更新知识点掌握度
- [x] 最近回答记录查询与刷新后保留

本阶段不接入 DeepSeek，不实现 JSON Schema AI 输出校验；固定评价将在 Phase 3 替换。

## Phase 3：DeepSeek AI 结构化认知评价（本次完成）

- [x] AI_PROVIDER 配置与 Mock / DeepSeek Provider 抽象
- [x] DeepSeek OpenAI-compatible Chat Completions 接入
- [x] JSON Output、Prompt v1、严格业务结构校验
- [x] 超时、有限重试和安全错误处理
- [x] LearningTurn 结构化评价快照
- [x] AIEvaluationRun 调用审计
- [x] MasteryRecord 更新和 Misconception 精确去重
- [x] 学习页和历史页展示 AI 评价内容

本阶段不实现课程状态机、下一知识点选择或复习调度。

## 后续课程状态与复习能力

- [ ] 初始化、学习、追问、复习、实践、暂停、结课状态
- [ ] 下一知识点选择策略
- [ ] 阶段复习
- [ ] 到期复习

## Phase 4：知识世界与课程结构（已完成）

- [x] Lesson 静态角色、核心/扩展和 Depth 元数据
- [x] LessonRelation：prerequisite / extends / application / related
- [x] prerequisite DAG 校验、拓扑顺序和图统计
- [x] 营养学知识骨架的分支、平行节点和汇合点
- [x] Knowledge Graph API 与 Lesson 关系 API
- [x] 基础知识结构查看页 `/courses/:id/map`
- [x] 保留已有 Lesson ID 与 Phase 2/3 学习历史

Phase 4 不实现个人认知状态、自动解锁、路线推荐、AI 修改知识图或最终视觉地图。

## Phase 5：个人认知状态与认知演化（已完成）

- [x] CognitiveState / CognitiveEvidence / CognitiveStateEvent
- [x] Evaluation Prompt v2 与严格 Schema / Target 校验
- [x] Recognize / Understand / Apply / Transfer 层级和 State Transition
- [x] 认知状态、证据和演化时间线 API
- [x] LearningView / KnowledgeGraphView 认知状态展示
- [x] 保留旧 Phase 2/3 学习历史，不自动 backfill 认知等级

Phase 5 不实现自动解锁、路线推荐、迁移测试生成器、误区网络或 Agent 导航。

## Phase 6：迁移测试、误区网络与认知修正（已完成）

- [x] AssessmentChallenge / ChallengeAttempt 与独立挑战审计
- [x] transfer generator / challenge evaluator 及严格业务校验
- [x] Transfer Challenge 通过规则与 CognitiveState 影响规则
- [x] Misconception active / resolved / reopened 生命周期和事件
- [x] 固定 reasoning pattern taxonomy 与误区网络 API
- [x] 针对性 Misconception Recheck 与 Correction Validation
- [x] LearningView 挑战交互、误区网络页、知识结构迁移状态 overlay
- [x] AutoMigrate 和旧 Phase 2/3 数据兼容
- [x] non-substantive Transfer Answer 的本地正式判定与状态保持
- [x] Challenge Generation attempt/provider/validation 性能日志（脱敏，不输出 Prompt 或 RawResponse）

Phase 6 不实现探索雷达、问题池、Agent 导航、知识来源可信度或最终图形化认知地图。

## Phase 7 preparation：测试知识世界扩充（已完成）

- [x] 补全营养学现有 8 个 Lesson 的学习字段与测试题面
- [x] 新增逻辑与科学思维测试知识岛（3 个 Lesson）
- [x] 新增心理学测试知识岛（3 个 Lesson）
- [x] 静态 Course / Unit / Lesson / LessonRelation Seed 幂等
- [x] 保留已有 Lesson ID 与 Phase 1~6 历史，不伪造用户认知数据

本准备任务不实现探索雷达、陌生知识入口、问题池、推荐算法、Agent、AI 课程生成或 CrossCourseLessonRelation。

## Phase 7：Exploration Engine（已完成）

- [x] 基于 Knowledge Graph、CognitiveState、Misconception 和 reasoning pattern 的探索候选生成与评分
- [x] 探索雷达 API 与首页轻量入口
- [x] 陌生知识入口与跨课程候选
- [x] ExplorationDirection 保存、略过、打开和历史
- [x] ExplorationQuestion 问题池、启动和归档
- [x] AI 方向文案生成、结构校验与规则文案降级
- [x] 探索支线 Lesson 读取/回答，不改变课程 CurrentLesson
- [x] 不因打开探索方向创建 LearningTurn、CognitiveEvidence 或修改 CognitiveState

## Phase 8：Curriculum Construction & Coverage（进行中）

- [x] CurriculumBlueprint / BlueprintUnit / BlueprintLesson / BlueprintRelation 模型与 Nutrition v0 种子
- [x] 课程覆盖度 API；Coverage 只描述学科蓝图与正式 Lesson 的映射，不读取个人认知状态
- [x] CurriculumDraft ChangeSet、严格 AI 草稿协议与人工 Apply / Reject API
- [x] Apply 事务、DAG 校验、旧 Lesson ID / CurrentLesson / 学习历史安全边界
- [x] Course Archive 课程覆盖度与扩充草案审核界面
- [x] 主线 CurrentLesson 显式切换、规则型下一课推荐和 Knowledge Structure 学习入口
- [x] Blueprint Unit Expansion 与按 Unit 生成 Curriculum Draft 的渐进式课程生长入口
- [x] 知识结构页按 Unit 剩余重要性选择课程草案生成范围，避免继续生成节点的空草案请求
- [x] LearningView 下一步、首页继续/选择/展开知识世界联动
- [ ] 人工验收蓝图覆盖、草案 Apply 和回滚边界后再标记本阶段完成

本阶段不实现 AI 直接发布课程、自动删除或修改既有 Lesson、搜索、RAG、来源系统或自动扩图。

## Phase 9：Source, Credibility & Grounding（进行中）

- [x] KnowledgeSource Registry 与跨 Course 去重
- [x] SourceEvidence、GroundingLink 与三类 Grounding Target
- [x] 人工 Credibility Assessment，明确可信度不等于真值
- [x] Grounding 状态统一重算、冲突状态和审核事件
- [x] Grounding Coverage API 与课程档案 / Lesson 页面入口
- [x] 暂时隐藏来源校验前端入口；后端 API、数据和审核能力保留
- [x] 不触碰 CognitiveState、LearningTurn、CurrentLesson 和既有 Lesson 内容
- [ ] 人工验收来源、证据、连接审核和冲突边界后再标记本阶段完成

本阶段不实现搜索、抓取、RAG、向量、Embedding、OCR、Zotero 同步、AI 自动宣判真值或自动扩图。

## 后续长期使用能力

- [x] Phase 10 Dogfooding Readiness：诊断、一致性检查、SQLite 备份/恢复、JSON/Markdown 导出、优雅停机和 AI 失败可重试提示
- [ ] 人工完成 Phase 10 生产初始化、备份恢复演练和长时间运行验收后关闭稳定化阶段
- [ ] 模型切换
- [ ] 调用成本统计
- [x] v0.1 Quiet Intelligence UI 收口：统一 AppShell、PageHeader、状态 tokens 与响应式布局
- [x] v0.1 Visual Redesign：Current Focus、Compact Domain Tile、Discovery Card、Editorial Learning Flow 与 Vertical Knowledge Path
- [x] v0.1 Final UI Unification：Learning、Curriculum Coverage、Knowledge Map 与课程档案同步首页视觉语言
- [x] v0.1 Knowledge Map Graph：保留列表视图，新增 Vue Flow + dagre 地图、待生成节点、关系/Unit 筛选、CurrentLesson 与认知状态 overlay
- [x] Knowledge Map Graph Interaction Completion：节点选中与共享详情 Rail、显式开始/继续学习、Blueprint Unit Expansion / Draft 操作及移动端详情 Drawer
- [x] Scroll-driven Vertical Knowledge Tree：路径视图改为 DOM 纵向 DAG、IntersectionObserver 自动选中、关系定位、CurrentLesson 初始定位与滚动恢复
- [x] Unit Expansion 幂等保护：数据库状态抢占、并发请求隔离、完整结果事务写入与待审核 Curriculum Draft 防重复

## v0.1 Progressive Empty World Bootstrap（已完成基础实现）

- [x] production / dogfooding 空知识世界与 development Demo World 分离
- [x] Domain Skeleton、Starter Blueprint Expansion、Initial Knowledge World 三阶段独立持久化与 API
- [x] 阶段级超时、结构校验、失败保留前序结果、可重复重试与幂等 Apply
- [x] Apply 前不创建正式 Course / Blueprint / Lesson，Apply 后不创建个人认知数据
- [x] 首页空世界入口与 `/domains/new` 三阶段审核界面
- [x] Domain Initialization 可重复创建多个独立 Course，课程档案页始终提供新建入口
- [ ] 人工完成真实 DeepSeek、超时重试和 production 数据目录验收
- [x] 系统设置页支持 API Key 配置，并在保存后立即切换运行时 Provider
- [x] 课程档案和学习首页支持带二次确认的单课程删除
- [x] 修复课程菜单删除交互链路，确认后实际调用删除 API，并反馈成功或失败结果
- [x] 本地开发服务仅监听 `127.0.0.1`，development 跳过 Basic Auth，生产认证保持不变

## 重构前可验证测试基线（已完成）

- [x] 审计前端路由、页面本地状态、API 封装和课程/领域/知识节点模型
- [x] 记录生成进度、课程进度、掌握度、节点类型和探索归属的当前计算方式
- [x] 增加空领域、当前 Lesson、探索归属和 API Key 脱敏的 Go 回归测试
- [x] 增加 Vitest 知识路径数据一致性测试和 Playwright 核心页面回归矩阵
- [x] 补齐 Vue 类型检查、前端测试脚本和浏览器测试最小配置
- [x] 记录完整测试套件的既有阻断与尚未覆盖风险

本基线不开始状态管理、API、数据模型或页面业务重构。详细审计见 `docs/TEST_BASELINE.md`。

## 数据一致性与用户信任（已完成）

- [x] 探索 DTO 分离推荐上下文与候选节点真实来源，并校验 Course/Lesson 归属
- [x] 领域切换清空旧推荐、隔离异步响应，推荐请求禁用浏览器缓存
- [x] 统一五类知识节点类型，所有视图使用同一映射，Depth 不再参与类型推断
- [x] 拆分课程生成度、学习覆盖度和理解掌握度，旧 `progress` 仅保留兼容
- [x] 幂等修复“已有有效当前 Lesson 但仍初始化中”的旧状态
- [x] 增加来源归属、缓存隔离、类型映射、进度计算和跨视图一致性回归测试

本阶段不新增数据库列、不删除或重建学习数据，也不进行视觉重构。详细契约见 `docs/DATA_CONSISTENCY.md`。

## 关键交互可靠性（已完成）

- [x] 新建领域前后端统一校验必填、长度、空白和期望深度，提交互斥并提供可读 400 错误
- [x] 移动导航支持导航后收起、遮罩和 Escape 关闭、焦点进入/恢复，且不锁定 body 滚动
- [x] 探索选择器稳定显示当前领域名称和标签，领域切换期间清空旧结果并显示加载状态
- [x] 增加空表单、连续点击、抽屉关闭/焦点恢复、选择器多断点显示的浏览器回归

本阶段不改变业务数据模型，不删除或迁移现有学习数据，也不调整无关视觉样式。交互契约见 `docs/INTERACTION_RELIABILITY.md`。

## 核心学习闭环完善（已完成）

- [x] 空回答拦截、按 Lesson 自动保存/恢复草稿、未提交离开提示和具体回答建议
- [x] Evaluation Prompt v4：已理解内容、关键缺口、误区、使用证据、可信度/不确定性、下一步与迁移条件
- [x] 四类用户可读学习状态说明、认知证据时间与状态变化原因
- [x] 回答版本、评价变化、认知证据和掌握度前后快照的历史展示
- [x] 迁移挑战解锁条件、与原问题差异、独立作答记录及明确掌握度影响
- [x] 回答、挑战生成和挑战作答幂等键；AI 失败保持学习数据原子性
- [x] Schema 13 加法迁移、旧数据兼容和迁移前备份/回滚说明
- [x] 核心闭环服务、HTTP 集成和 Playwright 端到端回归

本阶段不重做知识地图或全局视觉系统。详细契约见 `docs/LEARNING_LOOP.md`。

## 可操作知识工具（已完成）

- [x] 知识区域稳定 ID 关联、旧数据无歧义回填、空/重复/错挂防护
- [x] 当前区域默认展开，其余区域概要折叠；统一八类筛选和用户可读知识层级
- [x] 查看节点与切换 CurrentLesson 分离，生成入口显示数量和草案影响
- [x] 探索推荐补齐类型、真实来源、理由和关系；问题池支持保存、撤销、领域/优先级/状态
- [x] 单次 AI 误区只作为推测；用户可确认、纠正、忽略，并查看回答/事件依据
- [x] 增加筛选、稳定关联、保存撤销、空状态和显式节点切换回归测试

本阶段不改变节点类型和来源口径，不自动发布课程或删除学习数据，不进行全局视觉重构。详细契约见 `docs/OPERABLE_LEARNING_TOOLS.md`。

## 视觉一致性、无障碍与系统设置（已完成）

- [x] 统一用户可见中文术语与六类语义状态色，提高辅助文字、空状态和未激活导航对比度
- [x] 首页当前学习重点补充最近学习、理解掌握度、建议时长和下一步目标
- [x] 桌面与移动导航改为真实链接，补齐 landmark、`aria-current`、字段错误关联、异步播报和全局键盘焦点样式
- [x] AI 服务商与模型分开设置，显示密钥配置状态、实际生效模型、最近成功调用并支持安全连接测试
- [x] 备份结果展示文件元数据；恢复要求逐字确认、固定快照、恢复前自动备份和无活动连接重启恢复
- [x] 一致性报告按课程结构、学习证据、历史、探索及来源分类并给出修复建议
- [x] 覆盖键盘导航、语义属性、密钥脱敏、安全恢复和 375/768/1024/1440 响应式回归

本阶段不修改学习状态、课程生成、推荐或掌握度计算逻辑，也不新增数据库迁移。详细说明见 `docs/ACCESSIBILITY_AND_SETTINGS.md`。

## 最终工程加固（自动化基线完成）

- [x] 已展开知识区域渲染、节点详情按需缓存，路径/列表切换不重读图数据
- [x] 探索雷达按领域/Lesson/数量隔离缓存，业务写入后主动失效
- [x] 知识、学习和探索页无效读请求取消，问题池每 20 条分段渲染
- [x] 课程草案超时/解析失败原子性回归，生成占位刷新轮询，领域草案刷新恢复
- [x] 一致性检查覆盖 Course/Unit/Lesson、节点类型、掌握记录与认知证据交叉归属
- [x] 补充缓存、取消、完整建域、生成恢复、长列表、误区审阅、备份/导出/恢复和密钥泄漏回归
- [ ] 统一所有 AI 写入的持久化 request ledger 和进程崩溃后自动续作
- [ ] 真实 Provider、真实备份回滚、屏幕阅读器和长时间运行人工验收

本轮无数据库 schema 变更，不删除、重置或自动修复学习数据。验收边界见 `docs/ENGINEERING_ACCEPTANCE.md`。
