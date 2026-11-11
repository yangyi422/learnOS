# 页面风格盘点（2026-11-06）

目标：在统一全站页面风格之前，先列出**当前真实差异**——每页的标题尺度、表面材质、间距、区块标题、按钮高度。本文只记录测量结果与取舍，不含尚未实施的方案。

## 测量方法

- 环境：`npm run dev`（源码）+ 固定 mock 数据，1440×1000，浅色主题；
- 取值：`getComputedStyle` 读实际渲染值，不用设计稿推测；
- 覆盖 17 条路由（首页、学习、项目、收集箱、生活/财务/月度回顾、探索、资料库/资料专题、正在学习、课程档案、知识结构、误区网络、创建领域、系统设置、用户管理）；
- 结构层补充：扫描 `glass-system.css` 的统一材质选择器，确认哪些卡片类被覆盖、哪些靠页面自己写样式。

> 局限：项目 / 生活 / 财务 / 月度回顾 / 知识结构 / 课程档案 / 用户管理在 mock 空数据下没有渲染出卡片，其"表面"结论来自 CSS 选择器覆盖分析，不是逐像素渲染值。

## 一、结构层：三套表面配方并存

| 配方 | Token | 用在 | 视觉特征 |
|---|---|---|---|
| **A 首页雾面** | `--material-mist-*` → `--home-surface-*` | 仅首页六卡 | 描边 `.07`、**上沿白高光 + 内圈白线**、柔和落影、`blur(8px) saturate(1.03)`、四边 22px 羽化 |
| **B 业务表面** | `--material-surface-*` | `el-card`、生活卡、项目卡、任务卡、资料库卡、知识图谱画布、创建领域卡、误区网络卡、设置卡… | 顶/左边是**白线**（`.55` / `.32`）、右边是深线（`.20`）、三层渐变填充、inset 阴影、`blur(20px) saturate(1.3)` |
| **C 阅读面板** | `--material-reading-*` | `inbox-list`、`finance-content`、`library-reader`、`editorial-flow` | 实底（浅色 `rgba(244,249,255,.54)`）、`.20` 边、无羽化 |

实测确认的差异（同一视口、同一主题）：

```
工作台  卡片边框顶色 rgba(74,96,122,.07)   模糊 blur(8px)  saturate(1.03)
学习    卡片边框顶色 rgba(255,255,255,.55) 模糊 blur(20px) saturate(1.3)
收集箱  reading 边框 rgba(64,99,145,.2)     模糊 blur(20px) saturate(1.3)
```

也就是说：**首页的"边框"是一条淡灰线，业务页的"边框"是一条白亮线**——并排看是两种材质（对比图见 `.tmp/material-compare.png`）。

未被 B 配方覆盖的卡片类（需要单独确认）：
`life-goal-card`、`finance-panel`、`learning-main`。

## 二、逐页矩阵（1440，浅色）

| 页面 | 外壳 | 宽度 | h1 | 描述 | 内容内边距 | 表面 | 区块标题 | 主按钮高 |
|---|---|---|---|---|---|---|---|---|
| 工作台 | wide | 1353 | 44 / 500 / -.88px | 15px | 48 36 28 | home-section + focus-card（mist） | 14 / 600 | 36 |
| 学习 | wide | 1361 | 28 / 500 / -.56px | — | 32 32 48 | current-focus + domain-tile（surface） | 18 / 700 | **40** |
| 项目 | wide | 1361 | 28 / 500 | — | 32 32 48 | project-list-card + task-card（surface） | — | 36 |
| 收集箱 | standard | 1248 | 28 / 500 | 16px | 32 32 48 | inbox-list（reading） | — | 36 |
| 生活 | standard | 1248 | 28 / 500 | 16px | 32 32 48 | life-event-card✓ / **life-goal-card✗** | — | 36 |
| 财务档案 | standard | 1248 | 28 / 500 | 16px | 32 32 48 | finance-totals✓ / **finance-panel✗** | — | 36 |
| 月度回顾 | standard | 1248 | 28 / 500 | 16px | 32 32 48 | review-sections（surface） | — | 36 |
| 探索 | wide | 1361 | 28 / 500 | 16px | 32 32 48 | el-card（surface） | — | 36 |
| 资料库 | standard | 1248 | 28 / 500 | 16px | 32 32 48 | library-collection-card（surface） | — | 36 |
| 资料专题 | reading | 1361 | 28 / 500 | — | 32 32 48 | library-reader（reading） | — | — |
| 正在学习 | lesson | 1361 | — | — | 32 32 48 | **learning-main✗** + editorial-flow | — | 36 |
| 课程档案 | standard | 1248 | 28 / 500 | 16px | 32 32 48 | editorial-flow（surface） | **21 / 700** | 36 |
| 知识结构 | wide | 1361 | 28 / 500 | 16px | 32 32 48 | knowledge-map-canvas（surface） | — | — |
| 误区网络 | reading | 1361 | 28 / 500 | 16px | 32 32 48 | el-card（surface） | — | — |
| 创建领域 | standard | 1248 | 28 / 500 | 16px | 32 32 48 | el-card（surface） | — | 36 |
| 系统设置 | standard | 1248 | 28 / 500 | 16px | 32 32 48 | el-card（surface） | — | 36 |
| 用户管理 | standard | 1248 | 28 / 500 | 16px | 32 32 48 | el-table | — | 36 |

## 三、差异清单（按影响从大到小）

### P0 材质不统一（最明显）

1. **边框语义相反**：首页 = 淡灰线 + 白色上沿高光（rim）；业务页 = 白亮顶/左线 + 深灰右边（border 三色）。
2. **模糊强度不同**：首页卡片 `blur(8px) saturate(1.03)`，业务卡片 `blur(20px) saturate(1.3)`。同样的背景光场，首页更"薄"、业务页更"厚"。
3. **阴影配方不同**：首页落影 `0 10px 26px -18px .22`；业务页 `inset` 内阴影 + 很弱的偏移阴影，几乎不"浮起来"。
4. **羽化只属于首页**：22px 四边羽化仅首页卡片有，业务卡片是硬边裁切。

### P1 层级尺度不统一

5. **区块标题三级跳**：首页 `14/600`、学习 `18/700`、课程档案 `21/700`。同样的"区块标题"出现三种字号两种字重。
6. **描述文字**：首页 `15px`，其余 `16px`。
7. **内容上边距**：首页 `48px`，其余 `32px`；首页底边距 `28px`，其余 `48px`。
8. **按钮高度**：学习页首个主按钮 `40px`（`size="small"`/自定义高度混入），其余 `36px`。

### P2 结构细节

9. **宽页宽度不固定**：wide 外壳 `max-width: none`，实测 1353–1361 之间浮动（受侧栏/滚动条影响），而 standard 固定 1248。宽页之间也可能差几像素。
10. **三个卡片类未进统一材质**：`life-goal-card`、`finance-panel`、`learning-main`。
11. **首页是唯一使用羽化 mask 的页面**，`@supports` 降级分支也只在首页卡片上生效。

## 四、统一方案（三选一）

### 方案 A（推荐）· 把首页雾面提升为全站表面

- 把 `--material-mist-*` 提升为业务卡片的唯一来源，B 配方退化为"同一组 token 的另一处使用"；
- 业务卡片获得 rim + 落影 + 羽化；模糊统一到 8px；
- 优点：全站一套材质，首页与业务页并排看不出接缝；
- 代价：所有业务页视觉会变"更轻、更浮"，需要重跑全部截图基线（`visual-audit`、`ui-foundation` 16 路由 × 5 视口、`regression` 玻璃/深色用例），并逐页确认对比度。

### 方案 B（保守）· 只统一"边缘 + 模糊"，不动填充与尺寸

- 业务卡片的 border 三色改为一色 + 白色 rim；模糊统一；填充保持；
- 优点：改动集中在 `glass-system.css` 一处选择器，风险最低，几乎不影响布局与截图几何；
- 代价：首页与业务页仍会有细微差别（羽化只有首页有、阴影仍不同）。

### 方案 C · 统一层级尺度（与材质解耦）

- 只统一 P1 的 5–8 项：区块标题一个尺度、描述 15/16 二选一、内容内边距统一、按钮高度统一；
- 优点：风险最低、立刻减轻"每页长得不一样"的感觉；
- 代价：材质差异仍在。

建议顺序：**先 C（尺度）→ 再 B（边缘/模糊）→ 最后评估是否要 A（完整雾面）**。理由是尺度统一靠改 token 即可、可增量验收；材质统一会牵动全部截图基线，适合单独一轮。

## 五、统一后的验收清单

1. `--material-*` 只在一处定义（`glass-system.css`），页面级 CSS 不得再覆盖 `--material-surface-*` / `--material-mist-*`；
2. 同一视口下，任意两页的卡片：圆角、边框颜色、模糊、阴影必须完全一致（可用本文的测量脚本复查）；
3. 区块标题全站一个字号/字重；描述文字一个字号；主按钮一个高度；
4. 深浅两套都要过：文字对比度（WCAG AA）、`prefers-reduced-motion`、无 backdrop-filter/mask 时的静态降级；
5. 截图基线更新范围明确写在提交说明里，避免"改了但基线没更新"的假通过。

## 附：复查方式

本文数据由 `.tmp/audit-page-style.cjs`（本地脚本，已 gitignore）采集；需要复核时重新打开 dev server 跑一遍即可。

## 六、已实施：C 层（层级尺度）2026-11-06

| 项 | 改前 | 改后 |
|---|---|---|
| 区块标题 | 14/600（首页）、18/700（学习）、21/700（课程档案） | **18px / 600**（首页保留 14/600 的"路标"设计，见 HOME_DESIGN） |
| 内容小节标题 | 20px（课程档案 coverage 单位、反馈区、节点详情、草案行） | `var(--font-section-title)` = 18px |
| 卡片标题 | 17px（探索卡、项目卡）、21px（资料专题卡） | `var(--font-card-title)` = 16px / 600 |
| 三级标题 | 17px（资料档案、记录面板）、18px（下一课、挑战框） | 16px / 600 |
| 抽屉标题 | 17px / 680 | 18px / 600 |
| 描述文字 | 首页 15px、其他 16px | 全部 `--font-body`（16px） |
| 主按钮 | 学习页「打开知识结构」40px | `var(--control-height)`（36px） |
| 页面级字号覆盖 | `.coverage-content > .section-header h2 { 21px }`、`.dashboard-section .section-header h2 { 18px }` | 删除，统一由 `SectionHeader` 组件决定 |

实施后实测（1440 浅色）：区块标题 学习 / 课程档案均为 `18px/600`，主按钮全部 `36px`，描述全部 `16px`。
验证：typecheck、79 项单测、`ui-foundation`(5 视口×16 路由) + `dashboard` + `records` + `regression`(含四视口溢出、深浅玻璃) + `learning-conversation` + `library` 共 75 项 E2E 全通过。

仍未处理（下一轮）：P0 材质（边框语义、模糊、阴影、羽化）与 P2 结构项（wide 宽度浮动、三个未纳入材质配方的卡片类）。

## 七、已实施：B 层（边缘 + 模糊）2026-11-06

只动"边缘与模糊"，不动填充、尺寸与几何。

| 项 | 改前 | 改后 |
|---|---|---|
| 业务卡片边框 | 三色：顶/左白线 `.55`/`.32` + 右侧深线 `.20` | **单色** `--material-surface-edge` = 浅 `rgba(74,96,122,.07)` / 深 `rgba(167,142,225,.10)`（与首页同一取值） |
| 上沿接光 | 靠 border-top 白色线模拟 | `--material-surface-rim`：`inset 0 1px 0` 白 `.72` + 内圈白 `.34`（深色为淡紫白），与首页同读法 |
| 背景模糊 | `blur(20px) saturate(1.3)` | 浅色 `blur(8px) saturate(1.03)`、深色 `blur(10px) saturate(1.1)`（对齐首页卡片） |
| 已废弃令牌 | `--material-surface-edge-top` / `--material-surface-edge-left` 及 `--glass-card-edge-top/left` 别名 | 删除 |

副作用说明：`--material-surface-edge` 同时被阅读面板（`inbox-list`、`finance-content`、`library-reader`、`editorial-flow`）引用，因此这些面板的边缘也一并变淡；其填充仍是不透明底（`.54` / 深色 `.76`），实测边框仍可辨识。

验证：1440 浅色下 学习/探索/资料库/收集箱 卡片实测 `边框 rgba(74,96,122,.07)`、`模糊 blur(8px) saturate(1.03)`；深色 `rgba(167,142,225,.1)`、`blur(10px) saturate(1.1)`；首页卡片无变化（仍为 mist 配方）。E2E：`ui-foundation` + `home-material` 16 项、`regression` + `workspace` 110 项全通过。

仍未处理：P2 结构项（wide 外壳宽度浮动 1353–1361、`life-goal-card` / `finance-panel` / `learning-main` 的材质归属）、A 层（业务卡片是否也加羽化与落影）。

### 构建陷阱：前缀顺序会让 `backdrop-filter` 在生产构建里失效

首轮 B 层部署后发现：开发环境卡片 `backdrop-filter: blur(8px) saturate(1.03)` 正常，**生产构建里 computed 变成 `none`**，边缘与填充仍在。原因是配方里写成了

```css
backdrop-filter: var(--material-surface-filter);
-webkit-backdrop-filter: var(--material-surface-filter);   /* 前缀在后 */
```

打包压缩后只保留了带前缀的那条（`-webkit-backdrop-filter`），无前缀声明被丢弃，于是 Chromium 计算值为 `none`。

**约定**：前缀写在前面、标准属性写在后面（与 `workspace-theme.css` 的控件配方一致）：

```css
-webkit-backdrop-filter: var(--material-surface-filter);
backdrop-filter: var(--material-surface-filter);
```

核对方式：`npm run build:check` 后在产物 CSS 里确认两条都在，再用静态托管 + Playwright 读 `getComputedStyle(el).backdropFilter`。

## 八、已实施：A 层（材质收口：落影 + 羽化）2026-11-11

| 项 | 改前 | 改后 |
|---|---|---|
| 业务卡落影 | 只有 `--material-surface-inset`（内阴影），几乎不"浮起来" | 新增 `--material-surface-shadow`：浅 `0 10px 26px -18px rgba(40,64,90,.22), 0 2px 6px -4px rgba(40,64,90,.10)`、深 `0 8px 24px -20px rgba(0,0,0,.32)`，与首页 `--material-mist-shadow` 同值 |
| 羽化 | 只有首页卡片有 22px 四边羽化 | 业务卡改为"膜层在内容之下 + 22px 四边羽化"：元素本体 `background: none`，`::before` 承载 `--material-surface-film` 并用 `mask-composite: intersect` 做双轴羽化；描边与落影留在本体，所以内容永远清晰、边缘靠光与影过渡 |
| 悬停 | 换成实底 hover 色，会把膜层盖住 | 羽化卡悬停改为 `background-color: transparent` + `--material-surface-shadow-hover`，与首页同一读法 |
| 降级 | 首页有 `@supports` 实底分支 | 业务卡补齐同一降级：不支持滤镜/蒙版合成时用 `--material-surface-fallback` 实底并隐藏膜层 |
| 不参与羽化 | — | `.task-card`（自带 `::before` 高光线）、`.knowledge-map-canvas`（内部绝对定位图例）、`.finance-totals > button`、`.library-page-link` 保留硬边，避免伪元素冲突与定位漂移 |
| 材质归属修正 | 盘点称 `life-goal-card` / `finance-panel` / `learning-main` 未纳入材质 | 实测：`learning-main`（即 `editorial-flow`）与 `.finance-content` 已在 reading 配方内；`finance-panel` 类已不存在（现为 `finance-section` / `finance-rank-row` / `finance-import`）；本次把 `life-goal-card` 补进 surface 配方 |
| 宽页宽度 | wide `max-width: none`，1440 实测 1353–1361（随滚动条出现与否） | 新增 `--width-wide`（默认 1360，2200px/3200px 档 1600/1840），`page-shell--wide` 用它；首页 `.workspace-home` 例外保留全宽构图 |

验证：`npm run typecheck`、79 项单测、`vite build` 通过；E2E `ui-foundation`（5 视口 × 16 路由）+ `regression`（含四视口横向溢出、深浅共享玻璃）共 40 项通过。截图对比在 `.tmp/style-before` 与 `.tmp/style-after`（1440/390 × 浅/深 × /learn /projects /library /discover /life /settings，含 `metrics.json` 的宽度与计算样式记录）。

仍未处理：三条"保留硬边"的卡片若也要羽化，需先解决伪元素占用与图例锚点问题；截图基线仍是对比式（无像素断言），未改基线文件。

## 九、已实施：背景与顶栏全站统一（2026-11-11）

用户在 2026-11-11 指出「工作台的背景和其他页面不一致、头部也不一致」，实测确认差异来自两处历史分叉：

| 项 | 首页（改前） | 其他页（改前） | 改后（全站） |
|---|---|---|---|
| 环境底 / 光场 | 宿主 `.home-ambient-host.is-home` 局部 token：`linear-gradient(#f2f6fc,#eaf1fb 50%,#f5f7fb)` + 冷白入射 .70 | 根级旧 token：三段径向渐变（蓝/紫/暖） | 统一为首页那套，写在根级 token |
| 画布底色 | 首页覆盖为 `#eef4fc` | `#f2f6fc` | `#f2f6fc` |
| 顶栏膜 | 浅 `rgba(242,246,252,.12)` / 深 `.72` | 浅 `rgba(245,250,255,.58)` / 深 `.92` | 浅 `.12` / 深 `.72` |
| 顶栏下边框 | 透明 + 无阴影 | `1px --home-edge` 可见细线 | 透明 + 无阴影 |
| 光场起点 | 视口顶端（含顶栏后方） | 顶栏下方 76px | 视口顶端 |

实现：把 `HomeAmbient.vue` 里 `.home-ambient-host.is-home` 的整套光场 token（含深色与 ≤700px 变体）上移到 `workspace-theme.css` 根级，删除组件内局部定义与 `.app-shell--home` 对画布/环境/顶栏膜的覆盖，把「顶栏底边透明 + 无阴影」从首页局部规则放宽到 `.app-shell` 全局。卡片材质（`--home-surface-*` 的膜层羽化）仍只属于首页，未扩散。

验证：`measure-bg-header.cjs` 实测 5 条路由 × 浅深两主题，环境底计算值、画布底色、顶栏 `background-color`（浅 `rgba(242,246,252,.12)` / 深 `rgba(10,6,27,.72)`）、`border-bottom-color`（透明）、高度 76px 完全一致；`ui-foundation` + `regression` 40 项、`home-material` 8 项（断言已改写为「离开首页后顶栏膜色与光场必须与首页完全一致」，比原来更强）全部通过；截图见 `.tmp/style-unified`（1440/390 × 浅/深 × 首页 + 6 条业务路由）。

### 两次踩到的坑（2026-11-11 同日修正）

1. 先用 `html { scrollbar-gutter: stable }` 固定宽页宽度：宽度确实不再浮动，但 `position: fixed` 的抽屉/浮层会以「视口减 gutter」为基准，任务抽屉整体内移 15px（`projects.spec` 的 1280/1600 用例当场失败）。改为 `--width-wide` token，撤掉全局 gutter；首页原有的 `html:has(.workspace-home)` gutter 规则保持不变。
2. 项目看板原本用 `variant="wide"`（1440 实测 1361，四列各 328px），用户反馈过宽；改为 `variant="standard"`（1248，四列各 300px），390px 下仍由 `.kanban-scroll` 本地横向滚动，页面级无横向溢出。

