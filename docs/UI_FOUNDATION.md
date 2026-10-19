# LifeOS UI Foundation V1

> 历史阶段文档。当前视觉以根目录 DESIGN.md 为准，宽度、主题与外壳已按 [DESIGN_IMPLEMENTATION.md](DESIGN_IMPLEMENTATION.md) 更新；下文保留此前实施记录。

## 审计与范围

已有 workspace-theme.css 统一颜色、状态、焦点、表面、圆角和 Element Plus 基础样式；PageHeader/SectionHeader 已存在。差异来自各页面的 1080/1100/1240/1320/1400/1440/1540px 宽度、独立标题字号与间距，以及资料专题自己的标题实现。

新增 PageShell，迁移 20 个业务页面外层。只调整前端视觉，不改变业务逻辑、API、数据库、路由语义、资料库 Wiki/Cards 状态或 Markdown 编辑器。Login 保留独立布局。

## 共享布局

PageShell 接收 variant、可选 title/description/eyebrow，支持 header、breadcrumbs、actions 和主体插槽；兼容原有 PageHeader。

| 变体 | 宽度 | 页面 |
| --- | --- | --- |
| standard | 最大 1320px | 工作台、Inbox、生活/目标、财务、回顾、资料库首页、设置、用户、新建领域、课程覆盖档案、来源列表 |
| wide | 最大 1600px | 学习首页、课程知识世界、知识地图、项目、探索 |
| reading | 外层 1600px，文章 860px | 资料专题/Wiki/菜谱、日常学习、误区、来源详情 |

同变体共享水平基线，超宽屏按变体居中。reading 只暴露文章宽度变量，不压缩目录、标题和操作栏；library-reader、日常学习正文使用该变量，证据面板及误区图保留业务结构。

AppShell 的 main-content 负责内边距和滚动，PageShell 不新增第二层 padding 或滚动区。桌面左右 clamp(24px,3vw,48px)、顶部 32px；820px 以下左右 20px，520px 以下左右 14px、顶部 24px。保留侧栏断点。滚动条占位只作用于桌面内容区，窄屏关闭；不在 HTML/body 预留滚动条或设置固定宽度，避免 Teleport 遮罩/抽屉偏移。

## 标题与 Token

PageHeader：面包屑 → 可选 eyebrow → 标题/操作同一行 → 描述。标题 26–32px/650 字重，描述 14px/1.7，面包屑 12px；面包屑间距 16px、主体间距 24px。小屏操作换行，无操作无占位。工作台保留“今天继续什么？”。中文标题、英文专有名词及用户专题名称不批量改名。

几何 Token 加入已有 workspace-theme.css：page-width-standard/wide、content-width-reading、page-gutter/padding/header-gap、font-page/section/card/body/caption、control-height；复用 space/radius/color/focus/shadow 系列，不新增主题。默认按钮及输入/选择控件统一 36px，小/大号、图标及链接控件保留尺寸。项目、生活和 Inbox 的 Tab 保留不同形式，共享高度、字体和基本间距。

课程、专题及探索卡片的基础圆角使用 radius-card；任务卡、课程进度及所有卡片信息结构保留。Markdown 继续使用专用样式，与 PageHeader 隔离。既有深色主题保持，无新 UI 依赖。

## 特殊布局与边界

保留工作台原网格；看板四列、横向滚动及任务抽屉；日常学习紧凑上下文标题、独立聊天滚动和固定输入；Wiki 目录、Cards 独立阅读；财务表格/趋势及来源证据面板。没有强行把这些模块改造成相同卡片。

各业务卡片的局部密度、精确内边距及专用标题仍有合理差异，未宣称逐像素全部统一。学习聊天与来源详情保留业务标题而非全面替换 PageHeader。Vite 原有大包提示不在本轮处理。

## 验证

ui-foundation.spec.ts 用合成数据巡检 15 个入口/状态，覆盖 1920/1440/1024/390px 的基线、宽度、阅读栏、溢出与运行时错误；CAPTURE_VISUALS=1 生成 60 张截图，位于被 Git 忽略的 web/test-results。现有浏览器用例继续验证业务操作。

测试维护：看板绝对位置断言改成工具栏相对间距；首页骨架以显式响应门闩代替 250ms 竞态；多页面响应式巡检使用 60 秒上限，单业务用例不放宽。探索切换测试显式控制第二次响应；学习草稿恢复以 DOM ready 和并行断言验证，避免短暂提示被串行等待错过。

真实 Windows 滚动条、触屏及正式数据密度仍建议人工体验。未部署、未修改私人数据。2026-10-13 最终验证：前端 typecheck、35 项单元测试、生产构建通过；全量 Playwright 147 项通过，虽允许一次重试，实际没有触发重试。独立四宽度专项另通过，并生成、抽查 60 张最终截图（ui-final-visuals）。`git diff --check` 通过。本轮仅前端，未重跑 Go 后端测试。
