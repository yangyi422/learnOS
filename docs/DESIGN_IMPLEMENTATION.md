# DESIGN.md：Wope 参考适配

## 审计与范围

根目录 DESIGN.md 已从上一轮中性浅色规范替换为 Wope 深紫界面参考。本次读取的是更新后的全文，而非沿用旧设计结论。现有 Vue 3 / Element Plus、AppShell、PageShell、PageHeader、SectionHeader 和语义变量已经可以复用。styles.css 负责业务结构，workspace-theme.css 负责共享主题并映射 Element Plus；路由与 API 无需改动。

首页面向任务聚焦和捕获，看板面向任务扫描与拖动，学习页面向阅读和对话，探索面向发现，生活/财务面向回顾，资料库使用目录和独立文章阅读。因此不把营销官网的 56/72px hero 标题、108px 分区间距和124px上下留白直接套到每个业务页。

## 实际规范

- 默认主题深色：画布 #0a0118，文字 #ffffff / #d2d0dd / #9b96b0，强调 #713dff。卡片与悬停使用接近画布的 #10071f / #160d25 / #211631；边界为10%白色，符合接近画布的材质层次。
- 主要填充操作使用紫色和白字；暗色链接与描边按钮使用 #b7a4fb，避免饱和紫色小字在深色背景上难读。辅助语义颜色仅用于实际成功、警告、危险状态，不承担主品牌角色。
- 胶囊按钮与标签、16px 卡片，浮层保留明确边界而不使用黑色投影。输入采用直角、底边与键盘聚焦反馈，登录页与业务页共用规则。
- 首页任务焦点具有轻微静态紫色下照光，其余内容保持普通表面；顶栏允许8px模糊，不为每张卡片添加模糊或光晕。不新增动画与图片服务。
- 标题28px/700，区域18px/700，界面正文16px、辅助14px，中文保持正常字距。Inter V / Inter 与 Rebond Grotesque / Sora 仅作为字体栈，未引入远程字体请求；没有这些字体时使用系统回退。长文本维持16px/1.8。
- standard最大1248px，wide使用可用宽度，reading文章760px，lesson800px。侧栏224/64px与移动抽屉、顶栏56px、32/24/16px页边距继续保持产品布局；普通控件36px、手机主要按钮44px。
- 原有明确选择的 light/system/dark 保持有效。新用户默认dark，首屏脚本、运行时及theme-color一致；不重置localStorage。

所有页面通过同源语义变量获得主题，旧bg/text/color-primary名称仅是兼容别名，不形成另一套设计系统。业务卡片的内容结构、四列看板、学习固定输入、Wiki目录与Cards独立阅读、Markdown安全渲染和原始正文均保持。没有修改数据库、服务端API、身份认证、财务计算或路由语义。

## 主要文件

workspace-theme.css、styles.css、theme.ts、index.html；PageHeader、SectionHeader、TaskDrawer；WorkspaceHomeView、LoginView、SettingsView、library.css。此前已落地的PageShell、AppShell与页面统一继续复用。相关视觉断言更新为当前规范，业务交互断言保留。docs/ui-design.md与ROADMAP同步更新。

## 验证与剩余边界

测试使用现有合成API数据，不操作正式数据库。复用既有Vitest与Playwright；页面布局覆盖1920/1440/1280/1024/390px，保留基线、单一h1、动作区域、阅读宽度与溢出检测，并检查浅深色文字、按钮及焦点对比度。最终结果：类型检查通过；16个文件35项单元测试通过；生产构建通过，仍有既有大包警告；git diff --check通过。全站161项首轮160通过、1项因修改代码触发Vite热更新而在填写表单时中断；代码稳定后的项目、归档、生活与共享布局专项41项全部通过，包含该中断用例和最新按钮/标题/画布断言。当前无遗留测试失败。生成75张五种宽度巡检截图，并实际抽查首页暗色、桌面Wiki和390px画廊。仅前端修改，未重跑Go测试；未部署本次更新，也未提交推送。

未新增Tailwind、UI框架、字体服务或营销页面；未安装Rebond Grotesque/Inter V商业字体。Windows字体、触屏、非Chromium及真实大量私人内容仍需人工验收。专业知识图的节点形状、用户自定义项目颜色以及业务状态色保留，不复制官网展示性的logo rail、宣传badge或截图装饰。


## 后续视觉样板

用户后续要求明显重组 AppShell 与首页，而非仅调整主题。当前首页结构、局部紫色环境光与玻璃面板的实施范围、截图和验证见 [WORKSPACE_CONSOLE.md](WORKSPACE_CONSOLE.md)。阅读页及密集业务页暂不推广该样板。
