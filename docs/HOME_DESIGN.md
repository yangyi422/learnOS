# 首页设计规范 · Soft Mist Glass Workbench

首页（`/`，工作台）是 LifeOS 的入口页。本文记录它**当前已实现**的设计理念、取值与验收方法；数值以仓库代码为准，不描述未实现的设想。

- 实现：[`web/src/views/WorkspaceHomeView.vue`](../web/src/views/WorkspaceHomeView.vue)、[`web/src/workspace-theme.css`](../web/src/workspace-theme.css)、[`web/src/components/AppShell.vue`](../web/src/components/AppShell.vue)、[`web/src/components/ambient/HomeAmbient.vue`](../web/src/components/ambient/HomeAmbient.vue)
- 全局规范见根目录 [`design.md`](../design.md)；本文只把首页这一层的规则讲细，不重复全站条款。
- 回归入口：`web/e2e/workspace.spec.ts`、`web/e2e/home-material.spec.ts`、`web/e2e/home-ambient.spec.ts`

---

## 1. 设计理念

四句话概括首页的取舍：

1. **一天一件事**：中间「今日焦点」是唯一的主行动；左右两栏是上下文，不抢视线。
2. **雾面玻璃，不是卡片泥潭**：六个模块共享**同一种材质**，靠留白和字号分层，不靠加粗描边、不靠彩色块。
3. **边界靠光，不靠线**：卡片轮廓由「上沿接光 + 柔和落影」建立，描边只有一丝（7% 不透明度）。
4. **安静但有信息**：默认状态下几乎无动效；只有状态变化（场景切换、保存、加载）才动，且幅度小、时间短。

不做的事：首页不放图表仪表盘、不放统计大数字、不放装饰插画；空状态用一句话说明可做的动作，不虚构示例数据。

---

## 2. 结构与信息层级

### 2.1 页面骨架

```
AppShell（雾面顶栏：品牌 / 导航 / 搜索 / 快速记录 / 账号）
└─ PageShell variant="wide"
   └─ PageHeader（h1「今天，按自己的节奏」+ 日期；右侧场景切换 全部/工作/生活）
   └─ 三栏网格
      ├─ 左栏：今日计划 / 正在推进
      ├─ 中栏：今日焦点（主卡）→ 继续上次 → 继续学习·全局
      └─ 右栏：快速记录 / 最近留下的·全局
      └─ 页脚快捷入口（个人档案 / 收集箱 / 我的资料库 / 生活档案 / 财务档案）
```

- 中栏固定为主行动区，宽度比两侧大约 30%：`grid-template-columns: minmax(0,1fr) minmax(0,1.3fr) minmax(0,1fr)`。
- 栏内间距 `22px`，栏间用 `--home-grid-gap`（24 / 28 / 32px 随视口递增）。
- 首页内容宽度 `--home-max-width` 已与全站页面帧取同一值（`var(--width-wide)`：1360 / 1600 / 1840px 三档，2026-11-11 起），左右留白 `--home-gutter` 与 `--page-gutter` 相同（32 / 40 / 56px）。切换路由时页面不再变宽或变窄。
- **701–1099px** 退化为两栏：中栏（焦点）与左栏同排，右栏（快速记录 / 最近 / 学习）横排到第二行（`repeat(auto-fit, minmax(250px,1fr))`）。
- **≤ 700px** 单栏，顺序由显式 `order` 指定：今日焦点 → 今日计划 → 正在推进 → 快速记录 → 最近内容 → 继续学习 → 页脚快捷入口（`order: 1..7`），首屏标题降到 28px、卡片内边距降到 20–22px。

### 2.2 层级手段（按优先级）

| 层级 | 手段 | 取值 |
|---|---|---|
| 页面标题 | 44px / 500 / `-0.02em` | `--font-home-display` |
| 日期、副标题 | 16px（`--font-body`）、次要色 | `--home-muted` |
| 模块标签 | 14px / 600 / `0.04em` 字距，像"路标"而非标题 | `--home-label-*` |
| 卡片内主文案 | 16–28px（焦点问题最大），加深 | `--home-ink` |
| 次要说明 | 13px，次级色 | `--home-secondary` / `--home-muted` |
| 计数、角标 | 与标签同排、右对齐，作为扫描锚点 | — |

原则：**先内容、再字阶、再分组**；不用底色块或描边来分主次。

---

## 3. 材质：Soft Mist Glass

六个模块与顶部控件共用同一套材质 token，任何一处改动必须六卡同时生效（`home-material.spec.ts` 会校验）。

### 3.1 卡面配方

| 组成 | Token | 浅色 | 深色 |
|---|---|---|---|
| 玻璃电影层 | `--material-mist-film` / `--home-surface-film` | `rgba(250,252,255,.30)` | `rgba(22,15,43,.46)` |
| 尾部渐变 | `--material-mist-tail` | `rgba(244,249,255,.14)` | `rgba(16,10,35,.30)` |
| 顶部微光 | `--material-mist-sheen` | `rgba(255,255,255,.12)` | `rgba(193,172,232,.055)` |
| 描边 | `--material-mist-edge` | `rgba(74,96,122,.07)` | `rgba(167,142,225,.10)` |
| 上沿高光 | `--material-mist-rim` | `inset 0 1px 0 rgba(255,255,255,.72), inset 0 0 0 1px rgba(255,255,255,.34)` | `inset 0 1px 0 rgba(214,198,255,.14), inset 0 0 0 1px rgba(214,198,255,.06)` |
| 落影 | `--material-mist-shadow` | `0 10px 26px -18px rgba(40,64,90,.22), 0 2px 6px -4px rgba(40,64,90,.10)` | `0 8px 24px -20px rgba(0,0,0,.32)` |
| 背景模糊 | `--material-mist-filter` | `blur(20px) saturate(1.3)` | `blur(18px) saturate(1.2)` |
| 圆角 | `--material-mist-radius` | `16px` | 同 |

**关键点：边界由 rim + shadow 建立，描边只做"一丝"提示。** 曾经把描边加到 12% 会立刻失去玻璃感——不要再走这条路；需要更清楚的轮廓时，优先加深落影（`-.22 → -.28`）而不是加粗边框。

卡片底部还有一层羽化（`mask-image` 渐变），让"雾"在卡片下缘自然收进去，而不是被硬边裁断。

### 3.2 环境光场

- 浅色：三层径向/线性渐变（冷蓝 → 淡紫 → 暖米），`--home-environment`；另有 `--home-light-entry / --home-light-reflection / --home-light-floor` 三个光斑参数。
- 深色：`#09051b` 底 + 紫色光斑，克制不发光。
- WebGL 环境层是可选项（`HomeAmbient`）：`prefers-reduced-motion`、WebGL2 不可用、上下文丢失时**自动回退**到静态 CSS 光场，且两者从不叠加；切走路由后停止渲染循环。
- 页面 body 底色与卡片画布可能不同，组件一律使用语义 token，不直接写死颜色。

### 3.3 文字与控件色

| Token | 浅色 | 深色 | 用途 |
|---|---|---|---|
| `--home-ink` | `#172840` | `#f2edf9` | 主文字 |
| `--home-secondary` | `#384f69` | `#d1c5e3` | 次级文字、模块标签 |
| `--home-muted` | `#4d617a` | `#bcb0d1` | 说明文字 |
| `--home-link` | `#354db1` | `#c9b3ff` | 链接、主操作 |
| `--home-edge` | `rgba(64,99,145,.13)` | `rgba(167,142,225,.22)` | 控件边框（比卡片描边略强，控件需要更明确的边界） |
| `--home-tint` | `rgba(75,126,225,.07)` | — | 控件底色 |

对比度按 WCAG AA 校准；深浅两套独立取色，不靠一个值反转。

---

## 4. 状态与反馈

| 状态 | 表现 | 规则 |
|---|---|---|
| 首次加载 | 六处骨架屏，保持最终布局的占位几何 | 骨架高度与真实卡片一致，避免加载完成后跳动 |
| 慢加载 | 6s 后骨架区出现「仍在加载…」提示 | 不让用户怀疑页面卡死 |
| 加载超时 | 20s 后换成错误 + 重试按钮 | 任何请求都必须有出口；不允许无限骨架 |
| 分区失败 | 该区显示错误与重试，**不遮挡其它区域** | 次要模块失败不阻塞焦点卡 |
| 切换场景 | 隐藏旧内容内容、保留节点与几何（`inert` + 最小高度锁定） | 切换失败保留全局区域，旧任务不可操作 |
| 空状态 | 一句话说明能做什么，不放假数据 | 空工作区不虚构项目/任务 |
| 成功保存 | 顶部快速记录清空 + 计数即时更新 | 乐观更新后以服务端聚合回填为准 |

---

## 5. 交互与动效

- 动效 token：`--motion-base: 180ms`、`--motion-slow: 300ms`、`--motion-ease-standard: cubic-bezier(.2,.7,.3,1)`。
- 只有三类东西会动：边框/阴影的 hover 过渡、场景切换的内容淡入、保存后的状态提示。
- `prefers-reduced-motion: reduce` 时所有时长降到 `0.01ms`（含骨架动画），保留状态变化但去掉位移。
- 键盘：所有交互元素保留 `:focus-visible` 轮廓；首页搜索支持 Escape 关闭、方向键在结果间移动；快速记录 Enter 保存、Shift+Enter 换行。
- 鼠标 hover 只在 `@media (hover: hover)` 下生效，避免触屏粘住 hover 态。

---

## 6. 响应式与高分屏

| 视口 | 首页宽度 | 栏距 | 页面留白 | 说明 |
|---|---|---|---|---|
| ≤ 700px | 100% | 18px | 16px | 单栏，`order` 指定顺序，控件高度 ≥44px，标题 28px |
| 701–1099px | 100% | 22px | 24px | 两栏：焦点+今日计划同排，右栏三块横排换行 |
| 1100–1279px | 100% | 24px | 24px | 三栏，侧栏图标化 |
| 1280–2199px | 1360px（= 全站 `--width-wide`） | 24px | 32px | 桌面基准（三栏） |
| ≥ 2200px | 1600px | 28px | 40px | 2560 视口 / 4K@150% |
| ≥ 3200px | 1840px | 32px | 56px | 4K@100% |

字体与控件高度**不随视口放大**：物理字号交给操作系统缩放，避免与 150%/200% 叠加。1920 及以下已用构建产物逐像素比对确认未被大屏规则影响。

---

## 7. 无障碍

- 每张卡片是带 `aria-labelledby` 的 `<section>`，模块标签即标题；骨架区用 `aria-busy` + `aria-label` 说明正在加载的内容。
- 场景切换用 `el-radio-group`（原生 radio 语义），不用自定义 tab。
- 状态提示用 `role="status"`，错误用 `role="alert"`。
- 数字角标带 `aria-label`（如「2 条待整理」），不只靠视觉呈现。
- 顶部搜索框 `aria-controls` / `aria-expanded` 指向结果列表。

---

## 8. 验收方法

改首页后按顺序跑：

```bash
cd web
npm run typecheck
npm run test          # 单元测试（含 api/主题/首页偏好）
npm run build:check
npm run test:ui       # Playwright：截图 + 材质/边框/响应式回归
```

`test:ui` 覆盖的关键断言：

- 六卡同材质、内容不被遮罩吞掉、材质不泄漏到其他路由（light/dark × 1440/390）
- 边框四边颜色一致、hover 与常态不同（`Home border continuity`）
- 首页光场可见且 CSS/WebGL 不重复叠加、离开首页无残留渲染循环
- 骨架 → 内容、错误 → 重试、场景切换保留几何
- 1920/1440/1280/1024/390 五档布局与色系自定义

**视觉复核**不等同于测试通过：仍需用真实浏览器看 1440 与 390 的浅/深两套截图，确认层级、留白和边框观感。

---

## 9. 已知边界

- 首页材质是"雾面玻璃"的克制版本：不做折射/色散，不用 WebGL 做卡片本身。
- 深色主题的落影比浅色弱，因为深色背景上落影几乎不可见，边界主要靠 rim 与微光。
- 大屏规则只调整宽度、栏距与留白，不动字号；这是有意为之，见 §6。
- 若未来要加"更厚玻璃"，调 `--material-mist-film` 的奶白度即可，不需要改卡片 CSS。
