# LearnOS Roadmap

## Windows 本地播放器桥接（Player Bridge，2026-11-17）

- [x] P1 独立 Go 程序 `tools/lifeos-player-bridge`：`lifeos-player://play?itemId=` 解析（仅 play + 32 位 itemId）、配置校验、Jellyfin PlaybackInfo、Windows 凭据管理器读取、日志脱敏
- [x] P2 路径映射：多规则前缀边界（`/media/movies/` 不匹配 `/media/movies2/`）、大小写不敏感、拒绝 `..` 与越界、扩展名白名单、文件/盘符存在性检查并区分「文件缺失」与「映射盘不可用」
- [x] P3 代码完成 PotPlayer 检测与 `exec.Command` 启动（独立传参、不经 shell、启动后退出）；**真实播放未验证**（见下）
- [x] P4 LifeOS 前端：电影详情新增「本地播放」区块（桌面端显示、移动端隐藏），含协议 URL、状态说明、检查清单、复制链接；后端以增量字段 `external_id` 暴露 Jellyfin itemId（认证体系未改）
- [x] 构建与自检：`player-bridge.exe -verify` 逐项检查配置/凭据/Jellyfin/PotPlayer/映射盘；`install.ps1`（HKCU 协议注册 + 回读校验 + 自检）、`uninstall.ps1`（只删协议）
- [x] 目标 Windows 机器打通：`-verify` 5 项全 OK（config / credential / jellyfin 10.10.7 / PotPlayer / mapping），
  `-resolve` 把 `/media/电影/…mkv` 映射到 `V:\…\media\电影\…mkv`（10330139819 bytes 与 Jellyfin 一致），
  `-play` 成功启动 PotPlayer 播放 NAS 原始文件（详见 `tools/lifeos-player-bridge/docs/TEST-REPORT.md`）
- [x] 浏览器链路打通：云端页面点「使用 PotPlayer 播放」→ 浏览器确认框 → PotPlayer 播放 NAS 原始文件（真实截图已归档到测试报告）
- [ ] 电视剧单集播放：延期（Bridge 按单个 itemId 工作，后续从 Jellyfin 取 Season/Episode）

## 媒体中心重构：内容类型入口页 + 分类详情页（2026-11-17）

- [x] 媒体中心首页改为四类内容入口（2×2 大卡片）：电子书 / 电影 / 电视剧 / 游戏；卡片含预览拼图、标题、来源（Calibre · OPDS / Jellyfin / Steam · Windows）、状态标签（已接入 / 即将接入）、数量
- [x] 首页不再承担筛选：概览分区、chips、底部统一筛选栏已移除，整页只有页头 + 四张入口卡
- [x] 新增分类详情页 `/explore/library/media/{books|movies|series|games}`：搜索、来源筛选、排序（最近新增 / 标题 / 年份）、R18 开关（仅电影/电视剧）、画廊与列表视图切换、分页、详情弹窗
- [x] 游戏分类页为空状态：游戏库即将接入 + 未来支持 Steam 游戏收藏、封面与游玩时间；无工具栏、无 Mock 数据、无请求
- [x] 路由复用现有 `/explore/library` 前缀，新增路径排在 `:collection` 之前，未改动既有媒体查询接口与数据库
- [x] 资料库首页媒体卡保持三块入口之一，封面缩略、来源说明与进入操作与新入口页风格一致
- [x] 排序为分类页当前页内排序（接口仍按 id DESC 分页），未新增后端参数
- [ ] **Steam 接入：延期（本阶段不做）** — Steam API / API Key / SteamID64 / Connector / Windows 采集端 / 数据库迁移 / 同步任务均未实施

## 资料库与媒体中心改版：内容总览 + 浏览页（2026-11-17）

- [x] 资料库首页重构为个人内容总览：页头（我的资料库 / 整理常用资料、个人专题与数字媒体收藏 / 新建专题）+ 三块内容入口（专题 / 工具 / 媒体中心），媒体卡面积与信息量最大
- [x] 媒体中心入口卡：数量统计（电子书 / 电影 / 电视剧 / 游戏即将接入）、最近封面缩略、最近新增名称、进入按钮、来源说明（Calibre / Jellyfin / 后续 Steam）；无数据时整卡改为「媒体中心尚未同步内容」空状态
- [x] 媒体中心内页改为概览页：统计 chips（全部 / 电子书 / 电影 / 电视剧 / 游戏）、总览分区（最近新增 / 电子书 / 电影 / 电视剧 / 游戏预留）+ 查看全部、工具栏、画廊与详情分页保持不变
- [x] 统计与预览复用现有 `GET /api/v1/library/media`（每种类型一次请求），新增 `web/src/api/media.ts` 与 `utils/mediaHub.ts` 纯函数；请求失败时降级为「—」而不是 0，不新增后端接口
- [x] 游戏入口仍为 UI 预留：`load()` 短路、不发不支持的类型；空状态文案「游戏库即将接入 / 未来支持 Steam 游戏收藏、封面与游玩时间」
- [ ] **Steam 接入：延期（本阶段不做）** — Steam API / API Key / SteamID64 / Connector / Windows 采集端 / 数据库迁移 / 同步任务均未实施

## 媒体中心：预留游戏入口（2026-11-17）

- [x] 内容类型筛选加入「游戏」：全部类型 / 电子书 / 电影 / 电视剧 / 游戏，选项来自 `web/src/utils/mediaHub.ts`
- [x] 选择游戏时不发起任何后端查询（`load()` 在 `isReservedMediaKind` 处短路），后端无 game 类型也不受影响
- [x] 游戏空状态独立渲染：游戏图标 + 「游戏库尚未接入」+「未来支持 Steam 游戏收藏、封面、游玩时间。」+ 阶段说明，无 Mock 卡片
- [x] 游戏模式下「媒体来源」「R18」控件禁用（与游戏无关）；切回其它类型恢复正常请求
- [x] 顺手修正媒体卡片表面：`--material-surface-border` / `--material-surface-bg` 两个不存在的 token 改为共享柔雾配方（淡边 + 上沿接光 + 落影 + 同一滤镜），与资料库/生活卡片一致
- [ ] **Steam 接入：延期（本阶段不做）** — Steam API / API Key / SteamID64 / Connector / Windows 采集端 / 数据库迁移 / 同步任务均未实施

## 任务日期修复：空日期不再变成 1901-01-01（2026-11-16）

- [x] 根因：`validDueDate("")` 放行空字符串 → DATE 列存 `""` → 读回 Go 零值 `0001-01-01` → 前端日期面板落到 1901 年
- [x] 新建任务时「计划执行日期」「到期日」默认今天（本地日历日期），编辑既有任务保持原值
- [x] 前端 `taskDueDate()` 把 `""` / `0001-01-01` / `1901-01-01` 统一视为未设置；提交时按此归一化
- [x] 后端创建任务时空白日期转 `nil`；`ProjectTask.AfterFind` 把零日期读成 `nil`（历史脏数据一并修好）
- [x] 测试：`TestTaskZeroDatesReadAsUnset`、`TestCreateTaskTreatsEmptyDatesAsUnset`；E2E `projects` + `regression` 48 项

## 生活笔记：长篇书写（2026-11-11）

- [x] 生活页新增「生活笔记」标签（`/life?view=notes`），左侧笔记列表 + 右侧长篇编辑区
- [x] 复用 `records` 表与 `/api/v1/records`：无迁移、无新接口，`content` 上限 10000 字
- [x] 编辑区加入 glass-system 的 reading 配方（`.life-note-editor`），正文铺满面板、底部操作贴底、高度按视口计算
- [x] 交互：未保存切换确认、`Ctrl/Cmd+S` 保存、字数与上限提示、空状态用 `EmptyState`
- [ ] 待定：笔记目前与通用「记录」共用一张表，会同时出现在项目页/收集箱侧记录列表；若要完全分开需新增 notes 表 + 接口 + 迁移

## 页面帧与内边距全站统一（2026-11-11）

- [x] `standard` 与 `wide` 取同一上限 `--width-wide`（1360 / 1600 / 1840），`--home-max-width: var(--width-wide)`，首页不再独立成帧
- [x] 看板改用 `--width-board`（1248）显式声明，是唯一按业务收窄的例外
- [x] 首页上下内边距并入 32 / 48，`--home-gutter` 与 `--page-gutter` 同值（32 / 40 / 56）
- [x] 验证：1440/1920 逐路由测量一致；导航逐帧采样无中间态；`ui-foundation`+`regression`+`workspace`+`projects` 134 项、`test:ui` 13 项通过
- [x] 文档：design.md 页面帧与 4K 档、HOME_DESIGN.md 首页宽度表同步

## 看板宽度收窄（2026-11-11）

- [x] 项目页从 `variant="wide"`（1440 实测 1361）改为 `variant="standard"`：看板 1248，四列各 300px
- [x] 390px 仍由 `.kanban-scroll` 本地横向滚动（grid min-width 1020），页面级无横向溢出；深浅两主题一致
- [x] 验证：`ui-foundation` + `regression` + `projects` 共 56 项、`test:ui` 13 项全部通过；截图 `.tmp/projects-width`

## 背景与顶栏全站统一（2026-11-11）

- [x] 光场与环境底提升为根级唯一基准：`--home-environment` 与三个光区的中心/半径/光色（含深色、≤700px 变体）从 `HomeAmbient.vue` 的 `.is-home` 局部块上移到 `workspace-theme.css`
- [x] 删除 `.app-shell--home` 对画布/环境/顶栏膜的覆盖，首页不再有独立背景
- [x] 顶栏统一：浅 `rgba(242,246,252,.12)` / 深 `rgba(10,6,27,.72)`，底边透明且无阴影，光场起点回到视口顶端
- [x] 卡片材质（`--home-surface-*`）仍限首页，不随背景扩散
- [x] 验证：`measure-bg-header.cjs` 5 路由 × 浅深实测一致；`ui-foundation`+`regression` 40 项、`home-material` 8 项通过；截图 `.tmp/style-unified`
- [ ] 待做：首页 `.main-content` 上下内边距仍为 48/28（其它页 32/48），属 P1 间距项，未纳入本次

## 页面风格统一 A 层：材质收口（2026-11-11）

- [x] 业务卡落影与首页同值（`--material-surface-shadow`），补齐"靠光与影立起"，不再只有内阴影
- [x] 业务卡 22px 四边羽化：膜层下移到 `::before` 并 `mask-composite: intersect`，描边/落影留在本体；补 `@supports` 实底降级与"悬停不盖膜"读法
- [x] 材质归属修正：`learning-main` 与 `finance-content` 已由 reading 配方覆盖，`finance-panel` 类已不存在，`life-goal-card` 补进 surface 配方
- [x] 宽页宽度固定：改为 `--width-wide` token（默认 1360 / 2200px 档 1600 / 3200px 档 1840），首页 `.workspace-home` 例外保留全宽；曾用全局 `scrollbar-gutter: stable`，因抽屉 fixed 定位内移 15px 已撤回
- [x] 验证：typecheck、79 单测、`vite build`；`ui-foundation`（5 视口 × 16 路由）+ `regression`（四视口溢出、深浅玻璃）40 项 E2E 通过；截图对比 `.tmp/style-before` ↔ `.tmp/style-after`
- [ ] 保留硬边：`.task-card`、`.knowledge-map-canvas`、`.finance-totals > button`、`.library-page-link`（伪元素占用 / 绝对定位锚点），如需羽化要单独设计
- [ ] 待做：若后续要像素级基线，需按同一浏览器/字体/DPR 条件补最小断言；目前仍是截图式对比

## 页面风格统一 B 层：边缘与模糊（2026-11-06）

- [x] 修复生产构建 `backdrop-filter` 失效：配方前缀顺序改为「前缀在前、标准在后」，产物 CSS 已确认两条并存，静态托管实测 `blur(8px) saturate(1.03)`
- [x] **部署恢复（2026-11-11）**：本机确认不安装 WSL2，Docker Desktop 因此不可用；改用原生 `bin\learnos.exe` + SQLite（`scripts/run-local.ps1`），8080 已更新为最新前端产物（`index-C2GAIdlo.js`）。云端 Linux 的 `docker compose` 路径保留在 `deploy/deploy.sh`

- [x] 业务卡片从"三色边框（白顶/白左/深右）"改为单色淡边 + `--material-surface-rim` 上沿接光，取值与首页雾面一致
- [x] 背景模糊对齐首页卡片：浅色 `blur(8px) saturate(1.03)`、深色 `blur(10px) saturate(1.1)`（原 20px/18px）
- [x] 删除废弃令牌 `--material-surface-edge-top/left` 与 `--glass-card-edge-top/left`
- [x] 影响面确认：阅读面板同用 `--material-surface-edge`，边缘一并变淡但填充仍不透明，实测可辨识
- [x] 验证：浅/深实测边缘与模糊取值；`ui-foundation`+`home-material` 16 项、`regression`+`workspace` 110 项 E2E 全通过；首页材质未变
- [ ] 待做：P2（wide 宽度浮动、三个未归属材质的卡片类）与 A 层（业务卡是否加羽化/落影）

## 页面风格统一 C 层：标题尺度（2026-11-06）

- [x] 区块标题统一 18px / 600：`SectionHeader` 默认字重 700→600；删除 `.coverage-content > .section-header h2{21px}`、`.dashboard-section .section-header h2{18px}` 两处页面级覆盖
- [x] 内容小节标题（课程档案 coverage、反馈区、节点详情、扩充草案）20px → `--font-section-title`(18)
- [x] 卡片标题（探索卡、项目卡 17；资料专题卡 21）→ `--font-card-title`(16/600)；资料档案、记录面板三级标题 17 → 16/600；下一课/挑战框 h3 18 → 16/600
- [x] 抽屉标题 17/680 → 18/600
- [x] 描述文字统一 `--font-body`(16)：首页 15px 覆盖移除
- [x] 主按钮统一 `--control-height`(36)：学习页「打开知识结构」40px → 36
- [x] 验证：1440 实测区块标题 学习/课程档案均为 18px/600、按钮 36、描述 16；typecheck、79 单测、75 项 E2E（ui-foundation 5 视口×16 路由、regression 含四视口溢出与深浅玻璃、dashboard、records、learning-conversation、library）全通过；本地 8080 已部署
- [ ] 待做：P0 材质统一（边框语义 / 模糊 8 vs 20 / 阴影 / 羽化）与 P2 结构项（wide 宽度浮动、`life-goal-card`、`finance-panel`、`learning-main` 未纳入材质配方）

## 页面风格盘点（2026-11-06）

- [x] 新增 [docs/PAGE_STYLE_AUDIT.md](PAGE_STYLE_AUDIT.md)：17 条路由的实测矩阵（h1/描述/内边距/表面/区块标题/按钮高）+ 结构层的三套表面配方对照
- [x] 关键发现：首页雾面（边框 .07 + 白 rim + 落影 + blur 8px）与业务表面（白顶线 .55 + 深右边 .20 + blur 20px）**边框语义相反**；区块标题存在 14/600、18/700、21/700 三种；首页描述 15px、其他 16px；内容上边距 48 vs 32
- [x] 三个卡片类未纳入统一材质：`life-goal-card`、`finance-panel`、`learning-main`
- [x] 给出三档统一方案（A 完整雾面 / B 边缘+模糊 / C 层级尺度）与建议顺序 C→B→A，以及统一后的验收清单

## 页面冗余清理：P0 实验入口收口 + P1 学习页合并（2026-11-06）

- [x] P0：「实验」悬浮条与账号菜单「视觉实验室」在生产构建默认隐藏，仅 `import.meta.env.DEV` 或 `?fx=1`（记 localStorage）可见；权限仍由 admin 角色决定。新增 `web/src/lab-visibility.ts`
- [x] P1：`/learn` 增加「卡片 / 列表」视图切换（常驻，query 驱动），列表视图取代原 `/courses`；`/courses` 改为重定向 `/learn?view=list`，删除 `CoursesView.vue`（44 行）
- [x] 入口收敛：学习页去掉「课程档案」按钮；领域初始化完成后的「返回课程档案」指向列表视图
- [x] 测试更新：regression 的课程卡片用例改写为「卡片↔列表切换 + /courses 重定向」；深色主题用例改断言列表行；ui-foundation 16 条路由基线、四个视口横向溢出、dashboard 用例全部通过
- [x] 生产构建实测：`/courses → /learn?view=list`、切换器渲染、切回卡片无 query，页面零错误

## 首页设计规范文档化（2026-11-06）

- [x] 新增 [docs/HOME_DESIGN.md](HOME_DESIGN.md)：首页设计理念、三栏结构与层级手段、柔雾玻璃材质全量取值（浅/深）、边界"靠光与影"规则、状态与反馈、动效与键盘、响应式断点、无障碍、验收命令与已知边界
- [x] 文档数值逐项对照代码核对：网格 1fr/1.3fr/1fr、栏距 22px、宽度 1680/2000/2480、断点 700/1099/2200/3200、标签 14px·600·.04em、标题 44px·500·-.02em
- [x] 修正 `design.md` §3 的过期取值（边界 .025→.07），新增「卡片边界：靠光与影，不靠描边」修订段并回链 HOME_DESIGN.md
- [x] 修正文档中曾写错的断点：过渡档是 701–1099px 两栏、≤700px 单栏（`order` 指定顺序），不是 820px

## 挖角手帐：面板重排与得意技能适配（2026-10-29）

- [x] 角色详情重排：职业路线与交涉条件上移，招募情报与四线规划下移；移除固定的「基本职」行
- [x] 每个职业的转职条件改为紧跟在下拉框右侧显示（选中即见），不再另起一行
- [x] 新增得意技能数据 `data-proficiencies.js`（50/50 覆盖，来源 fire-emblem-fw.site，角色译名以 wiki 为准（茉莉→嘉丝敏））：职业条件命中任一得意技能即标 ★、选中后描边并提示命中项
- [x] 浏览器验证（源码目录与部署后的 8080 各一次）：段落顺序、5 个阶段、内联需求、50 个适配选项、选中高亮与提示均符合预期

## 桌面端回环端口固定（2026-10-29）

- [x] 问题：桥接用 `127.0.0.1:0` 随机端口，origin 每次启动都变，而 localStorage 按 origin 隔离 → 挖角手帐规划、主题、工作区场景"重启即丢"
- [x] 改为优先固定端口 `38080`（`LIFEOS_BRIDGE_PORT` 可改），被占用时退回随机端口并在日志中说明该次启动不保留本地存储
- [x] 新增两项 bridge 测试：连续两次 Start 得到同一 origin；端口被占时确实退回临时端口
- [x] 文档写明"本机缓存 vs 服务端持久化"的区别：真正落库依赖服务端 `tool_states` / `inbox_items`，旧服务端只能退回 localStorage

## uTools 匹配指令：搜索结果直接采集（2026-10-29）

- [x] `plugin.json` 新增 `lifeos-capture-text` 功能：`over`（选中文本）+ `regex`（搜索框任意 ≥2 字符）两条匹配指令，候选文案「保存到 LifeOS 收集箱」
- [x] payload 预填：`preload.js` 缓存早到的 `onPluginEnter` action 并在注册时补发；`App.vue` 按 payload > 草稿 > 剪贴板优先填入并全选，标注「来自 uTools 输入」，不自动提交
- [x] 测试：插件 vitest 27/27（新增 2 项：payload 直接预填并可一次 Enter 提交、payload 优先于草稿/剪贴板且超长忽略）；构建产物浏览器实测 payload 链路（textarea=123234、备注正确、Enter 后成功）
- [x] 修正：uTools 官方明确会**忽略任意匹配的正则**（`/.*/`、`/[\s\S]*/`），原来的 catch-all `regex` 不会生效；改为只保留 `over` 匹配指令，并去掉会误导人的 `regex` 条目
- [x] 诊断：preload 记录最后一次进入事件（code/type/payload 长度与预览，不含凭证）到 `dbStorage`，连接设置面板显示「最近一次进入」，用于区分“没有 payload”与“onPluginEnter 未触发”
- [x] 真机反馈修正（这一轮的真因）：preload 一旦抛错，页面就静默退化为浏览器兜底（配置落 localStorage、无事件无剪贴板）。`preload.js` 改为**完全自包含**：不引用本地文件、`require` 全包 try/catch、取不到 Node http 时退回 fetch、事件注册也包 try/catch，保证 `window.lifeos` 一定存在
- [x] 增加「运行模式」指示（uTools 桥接 / 浏览器兜底）与「最近一次进入」诊断，避免再用截图猜 preload 是否生效
- [x] 测试更贴近真实：`preloadBridge.test.ts` 用 dbStorage/onPluginEnter 桩加载真实 `preload.js`，对本地真实 http 服务断言 method/Authorization/body/status，并验证早到事件被缓冲、注册后补发（28 项全绿）
- [x] 文档写明：改过 preload 后 uTools「重载」不一定生效，需完全退出 uTools 再开
- [x] 第二轮真机定位：面板显示「浏览器兜底」证明 preload 没能把 `window.lifeos` 送到页面世界——新版 uTools 以 `contextIsolation` 运行 preload，隔离世界的赋值页面看不到。改为优先 `contextBridge.exposeInMainWorld('lifeos', …)`，失败才回退 `window.lifeos = …`
- [x] 再补一层页面兜底：若页面本身注入了 `utools`（无 preload），`bridge.ts` 直接用页面 `utools` API 拼等价桥接，并在「运行模式」标注 `uTools 页面兜底（preload 隔离）`
- [ ] 未验证：真机 uTools 中匹配指令的候选排序与 `over` 取值（需在装了 uTools 的机器上确认）

## uTools 快速采集插件 + 采集凭证（2026-10-29）

- [x] 后端最小凭证：`api_tokens` 表（sha256、`capture:inbox` 单一 scope、必填有效期、可撤销、`last_used_at`），schema 32→33
- [x] 接口：`GET/POST/DELETE /api/v1/api-tokens`（仅会话）+ `POST /api/v1/capture/inbox`（Bearer token 或会话，复用 `InboxService.CreateWithKey`）；凭证 principal 强制 `Role=user` + 单一 scope，其他 `/api/v1/*` 仍只认会话
- [x] Web 设置页新增「快速采集凭证」面板：生成（一次性明文 + 复制）、列表（前缀/有效期/最近使用）、撤销
- [x] 独立插件工程 `utools-plugin/`（Vue 3 + Vite，独立 node_modules，不改 web/、desktop/、不新增数据库）：`plugin.json` 指令「LifeOS 收集」「快速记录」、preload 桥接、草稿保留、剪贴板预填、Enter/Shift+Enter、幂等 capture_key
- [x] 跨源处理：插件页对 LifeOS 属跨源，后端不放宽 CORS；改为 preload 的 Node http 请求（`preload-request.cjs`），页面 fetch 仅作浏览器 dev 兜底
- [x] 测试：Go 服务/处理器/迁移共 5 组新增用例；真实服务端到端（登录→发凭证→采集→撤销→401）；插件 25 项 vitest（含真实 http 运输层）；设置页 1 项 Playwright；构建产物浏览器冒烟
- [x] 部署：本地 8080 schema 33 + `api_tokens` 表与索引存在 + 迁移前备份 `learnos-20261029-062332.zip`
- [ ] 未验证：真机 uTools 加载（onPluginOut / dbStorage / clipboard / hideMainWindow）；远端 43.138 需重新部署才有该接口

## 桌面端站内链接不再交给系统浏览器（2026-10-29）

- [x] 原因：WebView2 没有标签页，`target="_blank"` 由系统浏览器处理；桌面端点挖角手帐会跳出应用
- [x] 桌面窗口入口 URL 增加 `?shell=desktop`，新增 `web/src/shell.ts` 用 sessionStorage 记住标记（跨 401→/login 等整页跳转保留）；桌面端工具入口改为同窗口打开，网页端仍为新标签页
- [x] 资料库工具区文案同步为「规划随账号保存（Web 与桌面端共用）」
- [x] 验证：新增 2 项 E2E（桌面 shell 同窗口跳转并可进入工具页、web 仍 `target=_blank`），与既有 library 用例共 3 项通过；typecheck、79 项单测通过；本地 8080 实测 `?shell=desktop` 下链接无 target、纯 web 下为 `_blank`
- [x] EXE 已重建（`bin/LifeOS.exe`、`LifeOS-debug.exe` 均含 `shell=desktop`），便携包同步更新
- [ ] 未验证：Windows 实机点击工具后是否确实留在应用内（需在桌面端点一次）

## 挖角手帐接入账号存档（2026-10-29）

- [x] 后端：新增 `tool_states` 表（`user_id + tool_key` 唯一、`state_json`、`revision` 乐观锁），schema 31→32，随既有 AutoMigrate 迁移
- [x] 接口：`GET/PUT /api/v1/library-tools/:key/state`，复用会话认证；未登录 401、未知工具 404、无存档 404（`TOOL_STATE_NOT_FOUND`）、revision 冲突 409、格式/超限 400（上限 512 KiB）
- [x] 前端：静态工具页启动先读账号存档，404 时提示「上传本机到账号」，编辑后 900ms debounce 写回并带 revision；409 停止自动保存并提示载入账号版本；未登录/离线退回 localStorage 并明示状态
- [x] 保留 localStorage 作为离线缓存与迁移来源，导出/导入 JSON 不变；不写资料三表、不进专题 ZIP
- [x] 测试：service 3 项、handler 3 项（含生产路由 401）、迁移 1 项（建表 + 唯一索引 + 重开幂等）、新增 `library-tool-sync.spec.ts` 3 项 E2E；`go test ./...`、typecheck、79 项单测全通过
- [x] 部署验证：容器 healthy、`schema_version=32`、真实库中 `tool_states` 与 `idx_tool_state_user_key` 存在、迁移前自动备份 `learnos-20261029-053607.zip`；未登录访问工具页显示「未登录 · 仅保存在本机」且本地缓存仍写入
- [x] 修正部署脚本：此前只同步 `web/`，导致后端改动不进镜像；改为按子树同步 `internal/ cmd/ deploy/ web/ …`（不使用 `--delete`，保护镜像侧 `.env`、`data/`、`backups/`）
- [ ] 未验证：已登录用户的真实读写在浏览器中的完整往返（本地无该账号密码，仅以 Go 集成测试与 mock E2E 覆盖）

## 4K 大屏适配（2026-10-29）

- [x] 定位：3840 视口下首页仍被 1680px、标准页被 1248px 上限卡住，内容缩在左上角（实测 shell 宽 1680/1248，右侧空白过半）
- [x] 新增两级视口尺度步进（≥2200px、≥3200px）：标准页宽 1248→1440→1640，首页 1680→2000→2480，页面边距 32→40→56，卡片内边距 28→32→36，阅读 760→880、课件 800→920，首页栏距 24→28→32
- [x] 首页/顶栏横向留白改为 `--home-gutter`（36→40→56），消除 4K 下顶栏与页面对不齐
- [x] 字体与控件高度不随视口放大：物理字号交给操作系统缩放，避免与 150/200% 缩放叠加
- [x] 等价性：1440/1920 四路由构建产物逐像素完全一致；2560/3840 按预期变化且无横向溢出
- [x] typecheck、94 项 E2E（ui-foundation/workspace/home-material）通过（1 项并发抖动重跑通过）；本地 8080 与桌面 EXE 均已更新
- [ ] 未验证：你那台 4K 显示器在 100% 与 150% 系统缩放下的实际观感

## 桌面端单文件分发（2026-10-29）

- [x] `desktop/desktop.config.json` 通过 `go:embed` 编入 EXE，分发只需一个 `LifeOS.exe`；运行时仍支持 `LIFEOS_API_BASE` > 同目录/当前目录配置文件 > 内置默认
- [x] 配置解析顺序与来源写入 `LoadConfig(embedded []byte)`，新增单测覆盖「内嵌默认生效」与「环境变量覆盖内嵌」
- [x] 该配置改为纳入版本管理（此前 gitignore），保证任何机器 `go build` 都能编译通过；地址为作者自有服务器，不含任何凭据
- [x] 重新构建并确认 EXE 内含地址与 `embedded default` 标记；便携包更新为 `LifeOS.exe` + 可选调试版
- [ ] Windows 实机确认：删除同名配置文件后仍能用内置地址连上服务器

## 桌面端默认全屏（2026-10-29）

- [x] 启动状态改为 `StartState: WindowStateFullscreen`：默认无原生标题栏、不显示任务栏；`LIFEOS_WINDOWED=1` 可回到 1440×900 窗口
- [x] F11 仍可切换全屏/窗口；全屏下用 Alt+F4 退出（无关闭按钮）
- [x] 说明取舍：仅"最大化"隐藏标题栏需要无边框窗口 + 自绘拖拽区与窗口按钮，属独立改动，未在本轮实现
- [x] `bin/LifeOS-debug.exe` 与 `bin/LifeOS-new.exe` 已含新逻辑；`bin/LifeOS.exe` 运行中被占用，关闭应用后改名替换即可
- [ ] Windows 实机确认：全屏启动是否正常、F11 是否被网页抢键、全屏下滚动与尺寸

## 默认主题改为浅色（2026-10-29）

- [x] `web/src/theme.ts` 兜底值与 `savedPreference()` 回退改为 `light`；`web/index.html` 首屏内联脚本默认浅色，`meta[theme-color]` 缺省 `#e1eaf5`
- [x] 已保存的 `learnos-theme` 始终优先：`dark` 仍为深色、`system` 仍跟随系统；不清空既有偏好
- [x] 更新 `ui-foundation` 用例为「默认浅色 + system/dark 偏好仍生效」，并补 `theme-color` 断言；117 项 E2E（ui-foundation/workspace/home-ambient/visual-audit）、79 项单测、typecheck 通过
- [x] 本地 8080 重新部署后实测：OS 深浅两种设置下新会话都是浅色，`saved=dark` 为深色、`saved=system` 跟随 OS
- [x] 桌面端重新构建：`bin/LifeOS.exe` 与 `bin/LifeOS-debug.exe` 均含新默认（`desktop/frontend/dist` 已重建）
- [x] `design.md` 与 `docs/ui-design.md` 中的「默认深色」表述同步更新

## 首屏"无限加载"兜底与桌面诊断（2026-10-29）

- [x] 定位：Web 与桌面共有的首屏问题来自历史代码——`load()` 一旦请求不返回，`loading` 永远为真，骨架没有人任何出口（HEAD 版本即如此，非本轮引入）
- [x] 前端：`load()` 增加 20s 上限（超时显示「加载超时」+ 重试）、`currentUser()` 启动等待 10s 上限、骨架 6s 后显示「仍在加载…」提示；不再出现无出口的骨架
- [x] 桌面桥接：记录每个被代理的 `/api/v1/*` 调用的状态码与耗时，日志可直接指出页面停在哪个请求
- [x] 验证：mock 挂起 `/workspace/home` 时，6.9s 出现提示、20.5s 切换为错误+重试（骨架消失）；86 项 workspace/home-material E2E、79 项单测、typecheck 通过
- [ ] 待确认：真实环境首屏卡住的具体请求（需查看新的 `lifeos-desktop.log` 代理耗时行，或 Web 端 DevTools Network）；远端 Web 需重新部署才生效

## 桌面端接入：Wails v3 + Vue 3（2026-10-29）

- [x] 新增独立 `desktop/` Go 模块（`lifeos/desktop`，`go 1.26.0` + `toolchain go1.26.10`），不引用后端代码、不改现有 go.mod
- [x] Loopback 桥接：`127.0.0.1` 随机端口提供 `go:embed` 前端，`/api/v1/*` 反代到配置的远端，`/healthz` 本地健康检查，history 深链回退 index.html
- [x] 安全边界：只绑回环、只代理 `/api/v1/`、方法白名单、Host/Origin 校验、`apiBase` 仅允许 http(s) origin；`Set-Cookie` 去除 `Domain`/`Secure` 以适配 loopback，保留 HttpOnly/SameSite/Path/Max-Age
- [x] 前端复用：新增 `web/vite.desktop.config.ts` 与 `build:desktop`，产物写入 `desktop/frontend/dist`；`internal/webassets/dist` 与现有 Web 构建/部署流程未改（index.html SHA 未变）
- [x] 工具链：Wails CLI v3.0.0 装入 WSL 用户级 `~/go/bin`；`wails3 doctor` 通过；Windows 交叉编译 `CGO_ENABLED=0`，未安装 MinGW
- [x] 构建产出 `desktop/bin/LifeOS.exe`（PE32+ GUI x86-64），前端资源已嵌入
- [x] 验证：desktop 单元测试（Guard/代理/Cookie）通过；对真实远端 `http://43.138.202.56:8888` 的端到端代理检查通过；前端 typecheck + 79 项单测 + build:check 通过
- [ ] Windows 实机：运行 EXE、WebView2、登录与 Cookie 落盘、深层路由刷新待手测

## Phase 2A：PageHeader 视觉层级（2026-10-29）

- [x] 业务页 `PageHeader h1` 字重 700→500、字距 `-0.02em`，尺寸仍为 28px；首页 44px display 标题由自身 `--font-home-display` 规则覆盖，未受影响
- [x] 描述文字改用 `--color-text-secondary`，色阶下沉一级；字号、行高、间距、actions 布局与面包屑未改
- [x] 实测描述字号当前为 `--font-body`(16px) 而非 14px，按"不改高度/间距"约束保持不动，差异已在交付说明中标出
- [x] 构建产物 A/B：10 条业务路由 × 双主题 × 1440/390 下 PageHeader/描述/actions 几何与行数**完全不变**，仅标题文字像素变化；无 PageHeader 的路由 0 差异
- [x] typecheck、79 项单测、隔离构建、40 项核心 E2E（ui-foundation + regression）通过；本地已更新为 `index-DDiPQ18a.css` 并在 8080 复核 computed 字重/字距/描述色

## Phase 1：方向 B Token 语义收敛（2026-10-29）

- [x] 标题尺度分层：业务页固定 `--font-page-title` 28px，新增 `--font-home-display` 44px 仅首页使用，首页 h1 改为引用 token
- [x] 圆角语义化：`--radius-surface`(16) / `--radius-field`(9，实际生效值) / `--radius-pill`(999)；`--radius-card/-panel/-control/-button` 降为兼容别名，消除 `--radius-control` 0px 与 9px 的双重定义
- [x] 材质收敛为同一家族三配方：`--material-surface-*`（操作卡片）、`--material-mist-*`（首页羽化，`--home-surface-*` 为别名）、`--material-reading-*`（阅读面板）；`--glass-*` 全部降为别名
- [x] 明确 `--color-accent` 为唯一品牌强调色，success / warning / danger 语义色保留不受约束
- [x] 等价性验证：改前/改后构建产物在 5 条路由 × 双主题下逐像素完全一致；首页 20 张基准 18 张逐位相同、2 张最大差 1 级；跨页 44 张 40 张逐位相同
- [x] typecheck、79 项单测、隔离构建、40 项核心 E2E（ui-foundation + regression）、13 项视觉检查通过；本地服务已更新至 `index-BHkur234.css` 并复核 computed token
- [ ] 遗留：`--home-*` 调色名仍承担全局角色、Element `--el-border-radius-small: 0px` 与 field 9px 不一致、表格/learning-composer 12px 未语义化、业务组件 4 处 eyebrow 待 Phase 2 决策

## 轻量 AI 前端视觉工作流初始化（2026-10-29）

- [x] 在既有 AGENTS / 设计规范 / 评审清单中建立参考读取、差异提炼、组件限制判断、局部实施、截图复核与最多两轮自动修正；优先级为结构/信息层次 → 组件语言 → 装饰
- [x] 区分全局与局部 Token，固定已验收浅色冷白蓝灰/轻玻璃及现有深紫方向；Skill / MCP 缺失不伪造研究或强制安装
- [x] 复用 Playwright 与现有 visual-audit / home-material：`test:ui` 默认保存固定时间/数据的首屏、整页 PNG 与来源 JSON；支持 `PLAYWRIGHT_BASE_URL` 验证已运行前端，不新增浏览器平台或依赖
- [x] 新增 `build:check` 隔离检查构建；现有 PowerShell 验证入口保留真实 E2E 失败退出码、区分显式告警与通过，截图阶段不覆盖前一阶段输出
- [x] 源码与实际 8080 前端各 13 项视觉检查通过，已核验 10 组 PNG/JSON 证据并抽查双主题桌面/手机；非首页沿用既有核心回归，未修改页面与业务源码
- [x] 对齐已上线首页控件的 3px 焦点外扩断言，保留 2px 轮廓与对比度检查；不为了旧断言重画 UI
- [x] Typecheck、79 项单测、40 项核心 E2E 与隔离构建通过；验证脚本语法、启动失败退出码及显式告警分支通过模拟验证，未重启正常运行环境

## 首页层级与扫描节奏（2026-10-28，参考 Refero Styles）

- [x] 参考 dope.security / Authkit / Dimension：主方向取 Dimension 的"工作台玻璃面板 + 克制 500 字重 + 细边无阴影"，仅借 Authkit 四级前景阶梯与 dope.security 数字 stamped 处理
- [x] 首页模块标题降为标签层（14px/600、次级色、.04em 字距，六卡共用 `--home-label-*`），任务/项目/文章标题成为第一层
- [x] 模块计数（`N 个项目` / `N 条`）以 tabular numerals 呈现，作为扫描锚点；不新增装饰性 eyebrow
- [x] 行内更紧（项目行 16px、最近行 11px、行内 gap 4–5px），模块间更松（header 下 20px），列位置与宽度不变
- [x] 大标题 600→500、-0.02em；输入区圆角 12→10px，与卡片 16px、控件 999px 组成半径家族
- [x] 修正 `workspace.spec.ts` 中依赖旧 `background-clip: padding-box` 的过期断言（卡片已无元素级填充）
- [x] 按上线现状统一文档与测试：继续学习固定在中栏焦点卡下方（与 `docs/ui-design.md` 早先记录一致），右栏只保留快速记录与最近内容；`design.md` Phase 11.1d 与两处 E2E 断言同步更正

## 首页双主题材质统一精修（2026-10-28）

- [x] 删除首页 Hero / Primary / Secondary / Reading 视觉分支；六卡统一圆角、薄膜、弱边界、羽化、Hover / Focus，不改内容、布局和尺寸
- [x] `--home-surface-*` / `--home-control-*` 仅首页生效；导航、搜索、快捷记录、头像、场景筛选与记录入口统一轻玻璃基调
- [x] 浅色抬升基底、减弱灰蓝沉降与局部白斑；深色保留深紫光场，统一低对比卡面，不新增 Canvas / Shader / 动画框架
- [x] 六卡一致性、双主题 390–1920px、键盘焦点、选中/禁用状态、静态降级声明与路由隔离自动化验证
- [x] 相同接口验收数据下，1440/390px 全部卡片与顶部控件的 bounds、字体、网格尺寸前后一致；项目页材质不变
- [x] 逐视口与滚动位置抽样文字对比度：浅色最低 5.20:1，深色最低 5.38:1；文字与内容层不使用材质遮罩
- [ ] Safari / Firefox 的原生遮罩表现与真实触屏设备视觉复核

当前规则以 [设计规范](../design.md#3-光场与空间) 为准，以下为历史阶段记录。

## 首页浅色 Soft Mist Glass 视觉可见性纠偏（2026-10-27）

- [x] 按最终截图采用冷白蓝灰柔雾背景、弱边界冷白膜与极低接触阴影，移除 Hero 亮面
- [x] 首页浅色导航融入画布，保留原高度、选中态和全部操作；不推广到其他业务页
- [x] 浅色仅保留克制 Ambient，停用实验折射与 Spotlight 亮斑；深色行为保留
- [x] 取证实际8080资源、命中规则、computed style与临时条纹：未发现旧CSS覆盖或嵌入产物失效；此前差异被平坦光照和较厚冷白膜削弱
- [x] 重新组织内容后方蓝灰漫射与上下冷白区域；既有光场在浅色从视口顶部开始，导航不再与环境断开
- [x] 量化定位：旧全宽巨型渐变虽然 CSS 生效，但整屏可见亮度差仅约20级，低于可察觉阈值；不是覆盖或缓存问题
- [x] 重排为「左上冷白高光 + 右侧蓝灰漫射 + 左下蓝灰沉降」三段中尺度光区，可见亮度差提升到32–36级
- [x] 卡片材质改为独立羽化层：元素不再绘制矩形填充，`::before` 承载材质并四边羽化 26px，边缘不再终止于描线
- [x] 边缘渐变带 30px→58px、最大相邻跳变下降；填充提高到 56%/44%/48% 后卡片仍有柔和存在感；深色逐像素无回归
- [x] 验证 CSS / WebGL 光场互斥且可见，关闭光场后像素变化、卡片几何不变，无额外 Context
- [x] 1440px / 390px 深色截图逐像素无差异；浅色文字采样对比度最低4.95:1，交互状态及 Reduced Motion 回归
- [ ] Safari / Firefox 与真实触屏设备复核

定稿规则见 [设计规范](../design.md#3-光场与空间)，前序光场记录作为历史保留。

## 首页 Wope / Dimension 光场精修（2026-10-27）

- [x] 移除深色焦点卡宽幅顶光，使用首页单一共享底部紫色反照；浅色减弱均匀染色并加入极弱暖反照
- [x] CSS / WebGL 共享局部光场参数，主题与尺寸同步；其他业务页保留原光场
- [x] 桌面1440px、手机390px、Reduced Motion、主题切换与光场生命周期回归
- [x] 最终收尾：深色光域压扁为定向 Horizon，浅色仅增强冷灰轮廓与接触阴影；到此停止玻璃与光场微调
- [ ] Safari / Firefox 与真实触屏设备人工复核

当前规则见 [设计规范](../design.md#3-光场与空间)。

## B 方案首页：棱光学习工作台（2026-10-16）

- [x] 按选定 B 构图实现首页横向导航、不等宽三列与唯一共享紫色光场
- [x] 保留真实焦点、项目、今日任务、学习位置、最近内容与快速记录入口
- [x] 搜索仅过滤当前已加载工作区条目；项目进度使用真实任务总数，不虚构日程时间
- [x] 中文字体与桌面/手机字阶校准；手机顺序为重点、今天、学习、项目、最近内容
- [x] 保留深浅主题和其他业务页侧栏；不修改后端、不部署
- [ ] Safari/Firefox 与真实触屏设备复核

实现边界、数据语义与验证记录见 [WORKSPACE_PRISMATIC_HOME.md](WORKSPACE_PRISMATIC_HOME.md)。本节为首页当前规则，前序精修记录保留作历史。

## 首页 critique 后的舒展精修

- [x] 探索内容字阶微调，手机课程图标独占引导行，正文恢复卡片内完整宽度
- [x] 手机日期与快速记录同行，主任务提前出现，保留44px触控区域
- [x] 模块间距略放宽，普通/弱化玻璃表面区分，保留紫光与深浅主题
- [ ] Safari/Firefox及真实触屏设备的材质和排版复核

范围与验证见[首页最终精修文档](WORKSPACE_HOME_FINAL_POLISH.md#critique-后的舒展精修)。

## LifeOS 生活档案 V0.1

### 第二轮体验完善

- [x] 目标概况分组、动机/当前判断空提示与真实更新时间
- [x] 阶段记录编辑、日期调整、确认删除，状态前后值保留
- [x] 阶段变更与目标更新时间事务提交，归属校验与失败回滚
- [x] 目标摘要与更新时间、时间线真实目标/来源名称
- [x] 保存后列表即时更新、阅读优先、Esc 与焦点返回、未保存保护


- [x] 独立生活入口，时间线与长期目标两个视图
- [x] 事件持久化、补录、阅读/编辑/删除、双领域和里程碑筛选
- [x] 长期目标状态、主动阶段记录与判断历史、项目/事件关联
- [x] Inbox 转生活事件的原子处理与来源/请求去重
- [x] 完成项目与正式结业课程手动收录，删除来源保留档案快照
- [x] 安全 URL/Obsidian 与复制能力复用，当前用户认证保护
- [x] Schema 21 可重复迁移、整库备份与生活数据导出
- [ ] 用户真实设备 Obsidian 唤起与长期使用体验反馈

使用、数据契约、迁移和回退见 [LIFE_ARCHIVE.md](LIFE_ARCHIVE.md)。

## 轻量课件与自然对话

- [x] 现有学习页整合课件、自由多轮对话和固定输入/课程推进
- [x] 单模型调用、先保存消息、失败重试及历史恢复
- [x] 校验用户原文证据、稳定完成建议与用户确认
- [x] 完成/跳过与下一课定位原子提交，进度与掌握度独立
- [x] 保留知识来源、地图、详细评价、挑战和历史记录，按需展开
- [x] Schema 19 兼容新增课件正文，保留所有旧数据
- [ ] 使用真实模型与个人课程检查教学质量、证据判断及长时间使用体验

契约、迁移与回滚见 [LEARNING_LOOP.md](LEARNING_LOOP.md)。

## Workspace Home UX Polish（已完成）

- [x] Current Focus 显示真实归属、状态、固定原因和最近活动时间，不调用 AI
- [x] Today 按逾期、今天到期、进行中、下一步排列最多三条可直接打开的任务
- [x] Active Projects 保持轻量列表并突出真实项目图标、强调色和任务数量
- [x] Continue Learning 突出最近课程；空知识世界提供明确的首次学习入口
- [x] Inbox 首页增加 Enter / 按钮行内快速记录，并保留顶部快速记录对话框
- [x] 首页各区统一骨架加载、真实空状态、可重试错误和隔离浏览器回归

本轮复用 Workspace 聚合 API 与 Inbox 创建接口，不增加业务模块、后端字段或数据库迁移。

## Personal Workspace Home + Inbox（本次）

- [x] `/` 改为真实工作区首页，原学习首页完整迁移到 `/learn`
- [x] 首页按确定性规则推导 Current Focus，并复用 Today 服务端分组
- [x] 汇总最多 5 个活动项目、最多 2 个真实学习领域和 3 条待整理 Inbox 内容
- [x] Schema 18 `InboxItem` 用户隔离、SQLite 持久化、URL 来源记录与可重复迁移
- [x] 快速记录、编辑、待整理/已处理/已归档列表、归档与删除
- [x] Inbox 转任务使用单事务、重复转换保护和失败回滚
- [x] Home/Inbox 路由与轻量窄导航入口，保留完整看板和学习模块
- [x] JSON/Markdown 导出包含 Inbox；补充服务、迁移、事务与前端回归检查
- [ ] 使用个人真实数据进行最终人工验收并观察首次 Schema 18 启动备份

实现规则见 [PRODUCT.md](PRODUCT.md)、[ARCHITECTURE.md](ARCHITECTURE.md) 与 [DATABASE.md](DATABASE.md)。

## 个人数字工作空间视觉基线（本次）

- [x] 建立长期界面设计规范，并在 `AGENTS.md` 引用
- [x] 学习首页按首次使用、待开始与可继续学习三种真实状态呈现
- [x] 悬浮式应用外壳、独立导航轨道与工作区主题材质精修
- [x] 首次使用页把欢迎标题并入唯一创建引导区
- [x] 窄图标导航、工作区顶栏和项目看板的比例与层级调整
- [x] 深色、浅色、跟随系统主题及本机偏好持久化
- [x] 现有页面基础主题、Element Plus 控件和减少动态效果支持
- [x] 第二轮收紧项目页顶部结构，开放式看板分列与深浅主题卡片材质
- [x] 看板任务卡片摘要排版与任务详情编辑抽屉精修，覆盖深浅主题浮层
- [x] 任务抽屉边界、灰阶、遮罩、长说明与键盘焦点交互精修
- [x] 看板整卡拖动与设置页主题切换
- [x] 导航选中指示线与原生悬停提示，消除提示造成的侧栏横向滚动
- [ ] 逐页细化旧学习模块与知识图的主题细节

长期设计规范见 [ui-design.md](ui-design.md)；现有实现说明见 [WORKSPACE_VISUAL.md](WORKSPACE_VISUAL.md)。

## Project Kanban Workspace（本次）

- [x] 默认全部项目的四列看板、单项目切换与 URL 状态
- [x] 项目列表、项目目标/外观、暂停与归档
- [x] 任务描述、优先级、到期日、Task Drawer 和 Today
- [x] 跨列拖动、列内排序、失败恢复与后端持久化
- [x] Schema 17 保数迁移、迁移前备份和 Done 历史分页
- [x] 项目/任务 API、归属校验和迁移回归测试
- [x] 修复到期任务再次编辑说明时的日期格式错误，并补充 API 回归测试
- [x] 看板到期日选择器使用简体中文界面
- [ ] 真实移动端拖动、多设备和生产升级人工验收

现行语义见 [PROJECTS.md](PROJECTS.md)。原“项目执行中心初版”设计见 [PROJECT_EXECUTION.md](PROJECT_EXECUTION.md)。

## 项目执行中心初版（历史）

- [x] 独立于课程的个人项目、任务、状态与唯一下一步行动
- [x] 项目/任务增改、任务删除与按用户隔离的 API
- [x] 项目执行页面和导航入口
- [x] Schema 16 可重复迁移、迁移前备份和项目数据导出
- [x] 项目流程、输入校验与跨用户隔离回归测试
- [x] 修复 Schema 14 旧领域草稿升级到用户系统时的非空列迁移失败
- [ ] 人工验收多用户页面、移动端布局和生产环境升级

本阶段不实现笔记、Obsidian 同步、提醒、日历、AI 自动修改或项目协作。详细契约见 [`PROJECT_EXECUTION.md`](PROJECT_EXECUTION.md)。

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