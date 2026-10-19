# Wope — Style Reference
> Wope - violet horizon interface. Think of a dark observatory table with a purple light bloom rising from below the glass.

**Theme:** dark

Source measurements are normalized; roles and recommendations are interpreted. Font summary lists are independent, not paired by position. HTML examples are reconstructions, not source components.

Wope - violet horizon interface. The page feels like a control deck lit from beneath by ultraviolet light: a nearly black-violet canvas, pale gray body copy, and concentrated purple glow under the product UI. The signature move is contrast by illumination rather than by layering — surfaces stay close to #0a0118 and depth comes from translucent white borders, blur, and radiant gradients instead of heavy shadow. Rebond Grotesque appears only where the brand wants scale and impact, while Inter V handles everything else with tight negative tracking in larger sizes and occasional wide-tracked utility text. Buttons, badges, and small controls are all full pills at 999px radius, set against mostly 16px cards, so soft controls float inside a more architectural content frame.

## Tokens — Colors

| Name | Value | Token | Role |
|------|-------|-------|------|
| Night Violet | `#0a0118` | `--color-night-violet` | Page background, dark card surfaces, footer and section canvas |
| Void Plum | `#0c0616` | `--color-void-plum` | Inverse text on light treatments, occasional dark text on pale surfaces |
| Soft Quartz | `#ffffff` | `--color-soft-quartz` | Primary headings, buttons, borders, icons, input text, dominant foreground |
| Muted Steel | `#d2d0dd` | `--color-muted-steel` | Secondary heading tone, brighter muted copy over Night Violet |
| Ash Lilac | `#9b96b0` | `--color-ash-lilac` | Body copy, navigation links, supporting labels, low-emphasis UI text |
| Dim Fog | `#85808c` | `--color-dim-fog` | Inactive icons, subtle separators, tertiary interface details |
| Ultraviolet Core | `#713dff` | `--color-ultraviolet-core` | Primary accent, active links, luminous highlights — saturated violet against the black-violet canvas creates the site’s underlit glow |
| Lilac Beam | `#b7a4fb` | `--color-lilac-beam` | Gradient flare, line effects, ambient glow edges — used as the cooler top note inside purple light blooms |
| Horizon Glow | `#8562ff` | `--color-horizon-glow` | Gradient midpoint and visual hotspot — the brighter violet that makes hero illumination feel energized instead of flat |
| Glass White 04 | `#ffffff` | `--color-glass-white-04` | Tinted fills for ghost pills, badge backgrounds, translucent overlays |
| Glass White 10 | `#ffffff` | `--color-glass-white-10` | Hairline borders on pills and glass panels |
| Ultraviolet Horizon | `linear-gradient(180deg, rgba(183, 164, 251, 0) 0%, rgba(183, 164, 251, 0.5) 50%, rgba(133, 98, 255, 0.5) 75%, rgba(133, 98, 255, 0) 100%)` | `--color-ultraviolet-horizon` | Hero light field, illuminated dividers, ambient section glow |
| Violet Beam | `linear-gradient(180deg, rgba(183, 164, 251, 0) 0%, rgb(183, 164, 251) 50%, rgb(133, 98, 255) 75%, rgba(133, 98, 255, 0) 100%)` | `--color-violet-beam` | Brighter line and halo treatments where the glow should read crisp instead of misty |
| Deep Fade Veil | `linear-gradient(180deg, rgba(11, 2, 23, 0) 22.69%, rgba(22, 9, 42, 0.5) 100%)` | `--color-deep-fade-veil` | Section overlays and bottom fades that darken content without breaking the purple atmosphere |

## Tokens — Typography

### Inter V — System font for navigation, body copy, links, buttons, badges, inputs, and smaller headings. The custom build is doing more than a stock Inter install: disabled standard ligature behavior plus alternate stylistic sets make the text feel slightly engineered rather than default SaaS. Use 400 for body, 500 for controls and links, 700 for compact emphasis; apply tighter tracking at larger UI sizes and the wide tracking only for utility-style labels. · `--font-inter-v`
- **Substitute:** Inter
- **Weights:** 400, 500, 700
- **Sizes:** 14px, 16px, 18px, 20px, 24px, 28px
- **Line height:** 1.15, 1.33, 1.40, 1.43, 1.50, 1.56, 1.71
- **Letter spacing:** -0.28px at 14px, -0.16px at 16px, -0.36px at 18px, -0.40px at 20px, -0.48px at 24px, 2.24px at 28px
- **OpenType features:** `"calt" 0, "cv06", "liga" 0; "cv01", "dlig", "salt", "ss01", "ss03", "ss14"; "cv01", "cv02", "cv03", "cv04", "cv09", "cv10", "ss14"; "calt" 0, "cv06", "dlig", "liga" 0; "cv01", "salt", "ss01", "ss03", "ss14"`
- **Role:** System font for navigation, body copy, links, buttons, badges, inputs, and smaller headings. The custom build is doing more than a stock Inter install: disabled standard ligature behavior plus alternate stylistic sets make the text feel slightly engineered rather than default SaaS. Use 400 for body, 500 for controls and links, 700 for compact emphasis; apply tighter tracking at larger UI sizes and the wide tracking only for utility-style labels.

### Rebond Grotesque — Display and headline font for hero statements and major section titles. The heavier serifless grotesque gives Wope its large-scale voice: broad, compact, and slightly formal, which keeps the glowing purple environment from feeling playful. Use only for headline moments, not for buttons or dense UI. · `--font-rebond-grotesque`
- **Substitute:** Sora
- **Weights:** 700
- **Sizes:** 20px, 24px, 56px, 72px
- **Line height:** 1.11, 1.14, 1.33, 1.60
- **Letter spacing:** -0.40px at 20px, -0.48px at 24px, -1.12px at 56px, -1.44px at 72px
- **OpenType features:** `"cv01", "salt", "ss01", "ss03", "ss14"`
- **Role:** Display and headline font for hero statements and major section titles. The heavier serifless grotesque gives Wope its large-scale voice: broad, compact, and slightly formal, which keeps the glowing purple environment from feeling playful. Use only for headline moments, not for buttons or dense UI.

### Type Scale

| Role | Family | Weight | Size | Line Height | Letter Spacing | Token |
|------|--------|--------|------|-------------|----------------|-------|
| caption | — | — | 14px | 16 | -0.28px | `--text-caption` |
| body | — | — | 16px | 21 | -0.16px | `--text-body` |
| subheading | — | — | 18px | 28 | -0.36px | `--text-subheading` |
| heading-sm | — | — | 24px | 36 | -0.48px | `--text-heading-sm` |
| heading | — | — | 28px | 48 | 2.24px | `--text-heading` |
| heading-lg | — | — | 56px | 74 | -1.12px | `--text-heading-lg` |
| display | — | — | 72px | 80 | -1.44px | `--text-display` |

## Tokens — Spacing & Shapes

**Base unit:** 4px

**Density:** comfortable

### Spacing Scale

| Name | Value | Token |
|------|-------|-------|
| 4 | 4px | `--spacing-4` |
| 8 | 8px | `--spacing-8` |
| 12 | 12px | `--spacing-12` |
| 16 | 16px | `--spacing-16` |
| 20 | 20px | `--spacing-20` |
| 24 | 24px | `--spacing-24` |
| 28 | 28px | `--spacing-28` |
| 32 | 32px | `--spacing-32` |
| 40 | 40px | `--spacing-40` |
| 64 | 64px | `--spacing-64` |
| 80 | 80px | `--spacing-80` |
| 96 | 96px | `--spacing-96` |
| 108 | 108px | `--spacing-108` |
| 112 | 112px | `--spacing-112` |
| 120 | 120px | `--spacing-120` |
| 124 | 124px | `--spacing-124` |

### Border Radius

| Element | Value |
|---------|-------|
| cards | 16px |
| badges | 999px |
| inputs | 0px |
| buttons | 999px |
| overlayCards | 10px |

### Layout

- **Page max-width:** 1248px
- **Section gap:** 108px
- **Card padding:** 28px
- **Element gap:** 16px

## Components

### Primary Glass CTA Pill
**Role:** Main action button in hero and high-attention areas

Use a 999px pill with background rgba(255,255,255,0.04), 1px solid rgba(255,255,255,0.1), white label text, and padding 7px 24px. Keep the fill translucent so the button reads like lit glass floating above #0a0118 rather than a solid chip.

### Header Outline Pill Button
**Role:** Secondary header actions such as Log in

Transparent background, 1px solid rgba(255,255,255,0.1), 999px radius, padding 4px 16px, label in white or Ash Lilac depending on emphasis. This variant should feel lighter and smaller than the primary CTA.

### Text Navigation Pill
**Role:** Minimal nav actions and tertiary controls

Transparent background, no visible border, 999px radius, padding 4px 24px, white text. Use when the interface wants pill spacing without a framed container.

### Hero Trial Badge
**Role:** Compact promotional label above or below hero copy

Pill badge with background rgba(255,255,255,0.04), white text, 999px radius, and padding 4px 14px. Border may be omitted or kept at 1px rgba(255,255,255,0.1) when placed over a strong glow.

### Feature Surface Card
**Role:** Primary content card for feature summaries and product panels

Use Night Violet #0a0118 background, 16px radius, no shadow, and internal padding of 28px when content sits inside. Separate from the page with 1px translucent white borders or slight glow edges rather than elevation.

### Frosted Info Card
**Role:** Secondary glass panel for smaller callouts

Use background rgba(255,255,255,0.02), 16px radius, no shadow, 28px padding, and optional backdrop blur(4px). This reads as a lighter film over the dark canvas.

### Overlay Mini Card
**Role:** Floating stat, tooltip, or decorative UI inset

Use background rgba(10,1,24,0.2), 10px radius, no shadow. Combine with blur(8px) and a thin rgba(255,255,255,0.1) border when it sits on top of product visuals.

### Product Screenshot Frame
**Role:** Large hero UI showcase container

Large 16px rounded frame in #0a0118 with subtle internal glow, no shadow, and an underlighting treatment using the Ultraviolet Horizon gradient. The screenshot should appear embedded into the purple light field, not hovering above it.

### Header Bar
**Role:** Top navigation container

Full-width bar over the dark canvas, content centered within 1248px, low-contrast links in Ash Lilac, and action pills aligned right. Use 16px horizontal grouping gaps and a restrained 78px height zone.

### Borderless Input Field
**Role:** Email or search input inside dark callout forms

Transparent background, no visible border, 0px radius, white text, padding 14px 130px 14px 20px. Pair with a surrounding card or underline rather than boxing the field itself.

### Logo Rail
**Role:** Social-proof brand strip

Contained horizontal row of monochrome partner logos in white or Dim Fog on the dark canvas. Keep the rail low contrast and evenly spaced so it sits as a quiet separator between hero and feature content.

### Section Block
**Role:** Standard vertical content band

Transparent section wrapper with 124px top and bottom padding when used as a major page band. Inside, center content and keep internal stacks at 16px or 24px gaps.

## Do's and Don'ts

### Do
- Use Night Violet #0a0118 as the dominant canvas and keep most sections within that same dark-violet family.
- Use Rebond Grotesque 56px or 72px with -1.12px to -1.44px tracking for hero and section headlines.
- Set body and navigation text in Inter V using Ash Lilac #9b96b0 for default copy and Soft Quartz #ffffff for emphasis.
- Use 999px radius for buttons and badges; reserve 16px radius for cards and 10px for floating overlays.
- Build glow with Ultraviolet Core #713dff, Lilac Beam #b7a4fb, and Horizon Glow #8562ff gradients instead of heavy drop shadows.
- Use 1px borders in rgba(255,255,255,0.1) on pills, glass cards, and framed surfaces.
- Keep vertical section rhythm large: 108px section gaps and 124px padding on major content bands.

### Don't
- Do not introduce bright non-violet accents like teal, green, or orange into primary UI actions.
- Do not use square or lightly rounded buttons; controls are pills at 999px radius.
- Do not place strong opaque white cards on the page; surfaces stay dark or translucent over #0a0118.
- Do not add conventional black box-shadows; depth comes from blur, borders, and glow gradients.
- Do not use Rebond Grotesque for body copy, labels, or buttons.
- Do not collapse muted text and primary text into one tone; keep #9b96b0 for support text and #ffffff for primary content.
- Do not use bordered rectangular inputs; the input itself is borderless and relies on its container treatment.

## Surfaces

| Level | Name | Value | Purpose |
|-------|------|-------|---------|
| 0 | Night Canvas | `#0a0118` | Global page background and default section base |
| 1 | Glass Film | `#ffffff` | Translucent fills at 2% to 4% opacity for badges, CTAs, and frosted cards |
| 2 | Dark Panel | `#0a0118` | Primary cards and screenshot frames with 16px radius |
| 3 | Overlay Veil | `#16092a` | Gradient fades and blurred overlays above the base canvas |
| 4 | Ultraviolet Glow | `#713dff` | Illuminated accent field beneath hero imagery and active focal points |

## Elevation

Wope avoids conventional elevation. Cards and UI frames sit nearly flush with the #0a0118 canvas, then separate through 1px translucent borders, backdrop blur(4px), blur(8px), and purple underlighting gradients. Even large product panels feel embedded in the page, as if backlit from below rather than raised above it.

## Imagery

The page is product-visual first: large contained screenshots of the SEO interface, framed with rounded corners and immersed in purple glow. Graphics are abstract and atmospheric rather than illustrative — grid lines, light rays, and radiant horizon effects create the scene around the UI. Logos are monochrome and quiet. The layout is text-dominant at the top, then gives substantial visual area to the product frame, so imagery works as proof of the software rather than decoration.

## Agent Prompt Guide

Quick Color Reference: background #0a0118; primary text #ffffff; body text #9b96b0; border rgba(255,255,255,0.1); CTA accent #713dff; glow highlight #8562ff. Example component prompts: 1) Create a hero section with background #0a0118, centered Rebond Grotesque headline at 72px weight 700 line-height 1.11 letter-spacing -1.44px in #ffffff, supporting Inter text at 18px weight 400 line-height 1.56 in #9b96b0, and a glass CTA pill with rgba(255,255,255,0.04) fill, 1px rgba(255,255,255,0.1) border, 999px radius, 7px 24px padding, white text. 2) Create a dark product screenshot frame: 16px radius card in #0a0118, no shadow, thin rgba(255,255,255,0.1) border, purple underglow using linear-gradient(180deg, rgba(183,164,251,0) 0%, rgba(183,164,251,0.5) 50%, rgba(133,98,255,0.5) 75%, rgba(133,98,255,0) 100%). 3) Create a header bar inside 1248px max width with Inter 14px/16px links in #9b96b0, 16px item gaps, a transparent outline login pill with 4px 16px padding and 999px radius, plus a stronger glass signup pill. 4) Create a feature card using #0a0118 background, 16px radius, 28px padding, headline in Rebond Grotesque 24px #ffffff, body in Inter 16px #9b96b0, and no drop shadow. 5) Create a promo badge with rgba(255,255,255,0.04) fill, 999px radius, padding 4px 14px, white 14px Inter text, placed above a dark section title.

## Similar Brands

- **Linear** — Shared dark product-first presentation with restrained neutrals and concentrated violet illumination
- **Raycast** — Similar black-purple glow field and glassy, pill-based control styling
- **Arc** — Uses atmospheric gradients and soft-luminous dark surfaces instead of traditional shadows
- **Warp** — Comparable dark developer-style canvas with bright accent lighting and centered hero statements

## Quick Start

### CSS Custom Properties

```css
:root {
  /* Colors */
  --color-night-violet: #0a0118;
  --color-void-plum: #0c0616;
  --color-soft-quartz: #ffffff;
  --color-muted-steel: #d2d0dd;
  --color-ash-lilac: #9b96b0;
  --color-dim-fog: #85808c;
  --color-ultraviolet-core: #713dff;
  --color-lilac-beam: #b7a4fb;
  --color-horizon-glow: #8562ff;
  --color-glass-white-04: #ffffff;
  --color-glass-white-10: #ffffff;
  --color-ultraviolet-horizon: #8562ff;
  --gradient-ultraviolet-horizon: linear-gradient(180deg, rgba(183, 164, 251, 0) 0%, rgba(183, 164, 251, 0.5) 50%, rgba(133, 98, 255, 0.5) 75%, rgba(133, 98, 255, 0) 100%);
  --color-violet-beam: #8562ff;
  --gradient-violet-beam: linear-gradient(180deg, rgba(183, 164, 251, 0) 0%, rgb(183, 164, 251) 50%, rgb(133, 98, 255) 75%, rgba(133, 98, 255, 0) 100%);
  --color-deep-fade-veil: #16092a;
  --gradient-deep-fade-veil: linear-gradient(180deg, rgba(11, 2, 23, 0) 22.69%, rgba(22, 9, 42, 0.5) 100%);

  /* Typography — Font Families */
  --font-inter-v: 'Inter V', ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
  --font-rebond-grotesque: 'Rebond Grotesque', ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;

  /* Typography — Scale */
  --text-caption: 14px;
  --leading-caption: 16;
  --tracking-caption: -0.28px;
  --text-body: 16px;
  --leading-body: 21;
  --tracking-body: -0.16px;
  --text-subheading: 18px;
  --leading-subheading: 28;
  --tracking-subheading: -0.36px;
  --text-heading-sm: 24px;
  --leading-heading-sm: 36;
  --tracking-heading-sm: -0.48px;
  --text-heading: 28px;
  --leading-heading: 48;
  --tracking-heading: 2.24px;
  --text-heading-lg: 56px;
  --leading-heading-lg: 74;
  --tracking-heading-lg: -1.12px;
  --text-display: 72px;
  --leading-display: 80;
  --tracking-display: -1.44px;

  /* Typography — Weights */
  --font-weight-regular: 400;
  --font-weight-medium: 500;
  --font-weight-bold: 700;

  /* Spacing */
  --spacing-unit: 4px;
  --spacing-4: 4px;
  --spacing-8: 8px;
  --spacing-12: 12px;
  --spacing-16: 16px;
  --spacing-20: 20px;
  --spacing-24: 24px;
  --spacing-28: 28px;
  --spacing-32: 32px;
  --spacing-40: 40px;
  --spacing-64: 64px;
  --spacing-80: 80px;
  --spacing-96: 96px;
  --spacing-108: 108px;
  --spacing-112: 112px;
  --spacing-120: 120px;
  --spacing-124: 124px;

  /* Layout */
  --page-max-width: 1248px;
  --section-gap: 108px;
  --card-padding: 28px;
  --element-gap: 16px;

  /* Border Radius */
  --radius-lg: 10px;
  --radius-2xl: 16px;
  --radius-full: 999px;

  /* Named Radii */
  --radius-cards: 16px;
  --radius-badges: 999px;
  --radius-inputs: 0px;
  --radius-buttons: 999px;
  --radius-overlaycards: 10px;

  /* Surfaces */
  --surface-night-canvas: #0a0118;
  --surface-glass-film: #ffffff;
  --surface-dark-panel: #0a0118;
  --surface-overlay-veil: #16092a;
  --surface-ultraviolet-glow: #713dff;
}
```

### Tailwind v4

```css
@theme {
  /* Colors */
  --color-night-violet: #0a0118;
  --color-void-plum: #0c0616;
  --color-soft-quartz: #ffffff;
  --color-muted-steel: #d2d0dd;
  --color-ash-lilac: #9b96b0;
  --color-dim-fog: #85808c;
  --color-ultraviolet-core: #713dff;
  --color-lilac-beam: #b7a4fb;
  --color-horizon-glow: #8562ff;
  --color-glass-white-04: #ffffff;
  --color-glass-white-10: #ffffff;
  --color-ultraviolet-horizon: #8562ff;
  --color-violet-beam: #8562ff;
  --color-deep-fade-veil: #16092a;

  /* Typography */
  --font-inter-v: 'Inter V', ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
  --font-rebond-grotesque: 'Rebond Grotesque', ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;

  /* Typography — Scale */
  --text-caption: 14px;
  --leading-caption: 16;
  --tracking-caption: -0.28px;
  --text-body: 16px;
  --leading-body: 21;
  --tracking-body: -0.16px;
  --text-subheading: 18px;
  --leading-subheading: 28;
  --tracking-subheading: -0.36px;
  --text-heading-sm: 24px;
  --leading-heading-sm: 36;
  --tracking-heading-sm: -0.48px;
  --text-heading: 28px;
  --leading-heading: 48;
  --tracking-heading: 2.24px;
  --text-heading-lg: 56px;
  --leading-heading-lg: 74;
  --tracking-heading-lg: -1.12px;
  --text-display: 72px;
  --leading-display: 80;
  --tracking-display: -1.44px;

  /* Spacing */
  --spacing-4: 4px;
  --spacing-8: 8px;
  --spacing-12: 12px;
  --spacing-16: 16px;
  --spacing-20: 20px;
  --spacing-24: 24px;
  --spacing-28: 28px;
  --spacing-32: 32px;
  --spacing-40: 40px;
  --spacing-64: 64px;
  --spacing-80: 80px;
  --spacing-96: 96px;
  --spacing-108: 108px;
  --spacing-112: 112px;
  --spacing-120: 120px;
  --spacing-124: 124px;

  /* Border Radius */
  --radius-lg: 10px;
  --radius-2xl: 16px;
  --radius-full: 999px;
}
```


## LearnOS implementation rules — 首页样板与后续逐页适配

以下为产品内已落地的规则，优先于上文营销页面的示例尺寸；业务保护、响应式和可访问性要求继续保留。共享变量位于`web/src/workspace-theme.css`的现有主题定义中。`--console-*`为按需采用的控制台材质语义，不会自动覆盖旧业务页；不要另建主题文件或整站全局覆盖。

| 角色 | 现有变量/组件 | 使用规则 |
|---|---|---|
| 画布与环境光 | `--console-canvas`、`--console-ambient`；AppShell | 深色#0A0118、浅色#F5F6FB；固定环境层不接收事件，位于内容之后，不能产生额外滚动。阅读页按需降低光照，不自动套首页背景。 |
| 普通表面 | `--console-surface`、`--console-border` | 深色rgba(18,10,32,.50)、浅色rgba(255,255,255,.50)；仅外层14px模糊，不嵌套模糊、不对整块设置opacity。 |
| 重点表面 | `--console-surface-focus`、`--console-border-focus` | 极轻透明渐变和较清晰边缘，用于当前唯一重点，不让每张卡发光。 |
| 弱化表面 | `--console-surface-muted`、`--console-border-muted` | 用于活动记录或次级分组，比普通表面更透明、边框更弱。 |
| 边缘与悬停 | `--console-border`、`--console-border-focus/secondary/muted`、`--console-sheen`、`--console-surface-hover` | 四边连续同色1px基础边框；高光独立放在内侧，不能代替轮廓。填色止于padding-box，hover仅180ms表面过渡，不改变边框，不上浮；遵守reduced-motion。 |
| 圆角/间距 | 既有`--radius-panel`、`--radius-button`、`--space-*` | 面板16px、按钮胶囊；普通面板24px内边距，密集焦点/今日区域可20px上下、24px左右。 |
| 文字 | `--console-text/secondary/muted`与现有字体层级 | 深色#F0EAF8/#CBC0D9/#C4B8D1；浅色#2D2637/#62596C/#72657E。正文14–16px、辅助12–13px、模块18–20px，中文正常字距。不得通过父容器opacity降低文字清晰度。 |
| 操作 | 既有Element Plus Button、WorkspaceIcon、QuickCaptureDialog | 主要动作使用`--console-action`紫色+白字；次要入口用`--console-link`，不重复造按钮组件。复用现有focus-visible焦点环，键盘操作必须保留。 |
| 图标容器 | `--console-icon-size/radius` | 32px、8px圆角；项目优先原图标，空/空白/占位圆点用名称首个Unicode字符，空标题兜底“项”。 |
| 页面骨架 | PageShell、PageHeader、SectionHeader、EmptyState | 复用原路由、公共组件和宽度变体，后续按页面任务逐步适配，不统一成一类万能卡。 |

首页局部主光只用于当前旅程：椭圆中心锚定主卡下沿，中心亮、两端消散；不把下沿主光添加到普通表面。主卡不使用硬高度，常规任务场景第一排约230–240px，长标题和学习问题可自然增高；第二排在真实内容宽度>=960px时1:1自然拉伸等高、内容顶部对齐，单栏恢复各自自然高度。“今天”使用三个只读统计分组，逾期仅通过克制语义红表达，不把数量变成按钮。本轮其他业务页面不迁移。

首页保持舒展阅读：桌面模块纵向间距32px、横向24px，手机模块间距28px。学习内容标题19px/1.6行高；手机图标与课程名称同行，其余学习内容使用卡片内完整宽度。手机日期与快速记录同行，页头至主卡28px。普通表面与高光退后，弱化表面浅色alpha .26、深色.30，hover仍保持弱化层级；旅程透明膜、紫光、连续细边与两主题保留。


## LearnOS B homepage — scoped implementation update (2026-10-16)

本段仅记录 `/` 的已实现 B 首页，优先于上文旧首页构图、字阶和卡片主光说明；原始风格参考及其他业务页规则保留。选定构图为 `design/home-concepts-2026-10-16/violet-study/b-1440x1080.png`。不把首页三列或光场提升为整站强制布局。

**The Shared Light Rule.** 首页只有一个 `HomeLightField`，通过根主题中已有 `--home-*` 变量控制光源、透明面板和边缘反光。深色画布为 #09051B，浅色为 #F4F3FA；面板16px圆角、14px背景模糊，不嵌套模糊、不用父级opacity压暗文字。

**The Chinese Hierarchy Rule.** 中文字体沿用 `--font-ui`：PingFang SC、Microsoft YaHei、Noto Sans CJK SC、WenQuanYi Zen Hei 与系统回退。首页/重点/课题标题桌面44/36/28px，手机28/26/21px；中文保持正常字距。此处不沿用营销参考中的拉丁展示字体。

**The Home Boundary Rule.** 桌面横向导航只属于首页，三列比例1:1.3:1，列间24px；手机按重点、今天、学习、项目、最近内容排序。其他页面沿用侧栏，深浅主题选择保留。项目进度与今日状态以真实API事实为准，不为构图补造数据。

实现和验证记录见 `docs/WORKSPACE_PRISMATIC_HOME.md`。本次为已授权的局部追加，不重写现有参考或另建全局主题体系；未将旧参考的营销组件、旧首页尺寸或未验证的跨浏览器表现固化为 B 规则。


### 首页背光修订（2026-10-16）
首页保持 B 的导航、三列、字阶与真实内容，但光照改为安静的漫反射背光。删除夹缝白核、星芒、汇聚弧线及焦点/学习两侧的独立反光伪元素。唯一光域位于今日重点后方偏下：桌面650×230px扁椭圆，中心18%为低强度平台并向外缓慢衰减；两列时随重点移到25%横向位置，手机缩为330×190px。深色主背光为rgba(104,78,145,.22)，不得恢复白核或高饱和聚光；卡片保留既有透明表面、轻微斜向明度变化与统一细边。关闭光层时，文字、按钮和布局仍须独立成立。


### 2026-10-19：采用较强地平线背光

用户选择 horizon-preview 的 strong 档。正式首页使用同色三层照明：淡紫窄亮带、饱和紫过渡和宽范围环境扩散，亮带直径为重点卡宽度96%，中心始终藏在该卡下沿后方2px。光层锚定 focus-card-frame 实际高度，不依赖固定页面坐标，长任务、加载态及手机布局随卡片变化。重点卡上部遮光、底部少量透光与细反光；不加白核、星芒、弧线或网格。浅色主题沿用结构，单独校准紫光和白色面板透光值。保留原布局、字阶、数据与全部操作。本次仅同步实现，部署单独执行。


### 2026-10-19：浅色玻璃与冷色霓虹

浅色首页以 #EAF0F8 冰蓝画布、青蓝与淡紫环境折射替代旧浅紫底色。普通面板采用rgba(255,255,255,.32)，重点表面从.58渐变至.28，保留14px背板模糊、16px圆角、白色细边与上沿内反射。重点下沿地平线使用青蓝亮带与蓝紫过渡；文字区域用 #172840 / #384F69 / #4D617A 分层，不使用半透明文字。主按钮 #4653DF，悬停 #3542C9，配小范围有偏移的蓝色投影。背景灯色透过面板，霓虹不包围每张卡片；无网格、粒子或新增装饰组件。布局、业务数据及行为保留，仅首页浅色主题改变；深色颜色、较强地平线与主按钮值显式保持原值。


### 2026-10-19：玻璃材质与学习/最近内容互换

根据用户提供的 Snipaste_2026-10-19_11-49-51.png 修正浅色材质：取消重点卡下沿强青色窄亮带，保留低强度透光过渡；普通面板白色透明度.22，重点表面.34→.22，配白色上沿、轻微底沿折射和有偏移的柔和投影，背景加深少许以形成玻璃前后层次。深色较强背光保持原参数。

桌面中栏为重点→继续学习，右栏为最近留下的；平板学习跟随重点、最近内容在下一行，手机仍按重点→今天→学习→项目→最近内容顺序。学习卡在701px以上将几何图示排到摘要/状态旁，标题与行动横跨卡片，减少垂直占用。加载态同步将学习骨架放在重点下方；业务数据、跳转和按钮行为保留。


### 2026-10-19：玻璃反射与导航中轴

浅色首页降低普通面板填充至.14，重点表面.24→.12；背景增加宽斜向亮区及局部青蓝/淡紫色场，配非对称内边缘反射，避免将白边浅色填充当作透明玻璃。正文仍为深墨色，深色主题材质不变。

首页顶部导航以重点卡实际横向中轴为锚：通过 ResizeObserver 观测重点容器/主内容尺寸，将卡片中心相对顶栏的位置写入首页局部变量 --home-focus-axis；加载态与内容态切换重新绑定，离开首页清理监听及变量。宽度不足时继续使用抽屉导航；1599px以下收起账号文字，1399px以下压缩搜索宽度，避免控件重叠。浏览器测试直接测量导航与卡片中心偏差小于1px，同时检查导航不侵入右侧操作区。


### 2026-10-19：移除白线与白膜材质

依据用户最新截图 Snipaste_2026-10-19_13-34-49.png，移除浅色环境中的斜向白色高亮带；取消白色整圈描边与高亮内边，改为rgba(64,99,145,.13)的柔和冷色轮廓。普通表面为rgba(235,246,255,.10)，重点为冷色透明膜.14→.07，保留现有背景模糊与弱局部反射；不通过继续加亮白线制造玻璃感。该方案是透明界面材质表达，不宣称真实光学折射。布局、导航实际中轴、深色主题与业务交互保持不变。


### 2026-10-19：首页色系自定义

首页个人菜单加入「首页外观」，所有登录用户均可打开无背景遮罩的侧边设置，支持深/浅/系统模式、默认及冰川蓝/极光紫/玫瑰/松石绿预设与原生取色器。即时应用，只保存当前浏览器 localStorage `learnos-home-accent`；不写学习数据或后端。色系只影响首页环境色、背光、链接及主按钮，不改透明材质、布局、导航中轴或其他业务页面。恢复默认清除自定义变量并还原深浅主题原设计。

持久化只接受完整六位十六进制颜色，非法存储回退默认；禁用存储时仍即时预览并显示无法保存提示。同源多标签同步storage事件。自定义主按钮按相对亮度选择黑/白前景，单元测试覆盖4096种颜色，保证正常状态文本对比度≥4.5:1。无彩色自定义保留零饱和度，避免灰白颜色错误生成红色背景。浅色/深色各自使用固定亮度、透明度范围映射环境色，保留正文对比度。


### 2026-10-19：自然边缘环境光

用户要求色光不再集中于屏幕中间。取消重点卡下沿灯带、底部独立透光伪元素及其无用tokens，将唯一 HomeLightField 移到 AppShell 的整页固定背景层。主色光从左上视口外漫入，右下为较弱的邻近色反射，底部仅留低强度反照；三层以不同范围和衰减混合，光源中心均在画布外，无白核、弧线、网格或动画。背景光不再跟随重点卡高度，也不随任务密度改变中心位置，不出现内容网格矩形裁切。玻璃透明参数、正文、布局、导航对齐及色系持久化保留。自定义色系同步映射三处环境受光，默认配色和自定义配色共用同一空间规则。


### 2026-10-19：通透玻璃与整卡导航

进一步降低浅色普通玻璃填充至.06、重点表面.10→.05，背板使用blur(18px) saturate(1.2)，保留透明边缘和分布式环境色；不增加白色描边或中心发光。深色材质维持blur(14px)。

移除首页「今日重点」底部主操作按钮、「继续学习」底部行动和重复学习空间按钮；两张有真实目的地的卡片使用覆盖整个表面的原生RouterLink，右上以轻量箭头提示可进入。重点任务仍打开对应项目任务详情，学习仍进入当前课程/知识地图；按Tab聚焦与Enter打开、Ctrl/Cmd新标签、真实href均保留。空重点无跳转，学习空状态保留创建入口，学习出错保留重试且不放链接遮挡。悬停使用主题色轮廓/标题提示，焦点使用清晰外轮廓。色系调节继续影响环境光和交互色，不改变玻璃透明度或布局。


### 2026-10-19：五卡一致悬停与玻璃厚度

首页五张卡片统一使用180ms边缘反光、轻微受光变化和柔和投影，悬停时分区标题采用主题链接色；不位移、不缩放、不添加白色发光框。键盘focus-within使用相同材质反馈，整卡链接保留独立焦点轮廓。项目、今日安排和最近动态仍以内部条目作为操作目标，外层材质反馈不改变点击语义。

材质增加低强度斜向表面反射和背向冷色暗面，通过内边缘、偏移软投影及前后透色建立厚度；浅色悬停不再切换到.55白色底，改为.10冷色透明受光，避免鼠标经过变成实心白块。深色普通面板降至.26，重点表面由.82→.58渐变，接收整页环境光；原布局、色系设置、导航中轴与数据行为保留。


## 全站采用首页材质（2026-10-19）

用户授权将首页设计理念推广至全部页面，取代早期首页专属限制。根级 --home-* 保留命名并成为共享材质；glass-system.css 负责角色映射。所有页面共享自然边缘光、玻璃反射与180ms一致悬停，保留具体业务拓扑。阅读、图谱节点、输入与弹层按可读性分级，不机械套用同一透明度。全局色系兼容原 learnos-home-accent 偏好。详见 docs/GLASS_DESIGN_SYSTEM.md。


2026-10-19 导航修正：顶部玻璃导航扩展到全部已登录路由，取消业务页全局侧栏；首页继续按重点卡片中轴对齐，业务页默认内容视口居中，窄屏沿用顶部菜单按钮与抽屉。现有模块内部导航不受影响。
