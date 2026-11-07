# E-SP-Line2 后台 UI 规范

本文是 `web/` 后台前端的唯一视觉与组件约定。新增页面前先读一遍，照抄现有页面的结构，不要重新发明样式。

**简体中文** · [English](UI-SPEC.en.md)

范围：`web/src/index.css`、`web/tailwind.config.js`、`web/src/components/**`、`web/src/lib/cn.ts`。

> **设计语言：shadcn/ui（https://ui.shadcn.com，v4「base」）。** 这不是「参考一下配色」，而是整套约定，取值直接对齐其 `globals.css`：
> 中性盘是纯灰（`0 0% x`），正文纯黑、边框 `hsl(0 0% 90%)`；
> `primary` 浅色下**纯黑**、深色下近白——它提供对比，不提供色相；
> `--ring` 是**中灰**（`0 0% 63%`），故意不等于主色；
> 圆角只有一档 `--radius`，其余按比例**相乘**推导；阴影只有 `xs`；
> 焦点是 `border-ring` + 2px `ring-ring/50`，**没有 offset 间隙**。
> 改任何视觉前先看 ui.shadcn.com 的对应组件，照它的几何与克制程度来。

---

## 1. 原则

1. **颜色只从令牌来。** 页面里不出现 `bg-blue-600`、`text-gray-500`、`border-red-200` 这类调色板类名。要什么颜色，先找令牌；找不到就先加令牌（见 §2.6）。
2. **一个语义只有一个组件。** 状态用 `Badge`，容器用 `Card`（右上角用 `CardAction`），表格用 `TableShell`，下拉用 `Select`，加载用 `Skeleton`，空态用 `EmptyState`，错误用 `ErrorState`，标题区用 `PageHeader`，表单域用 `FormField`。
3. **密度是设计决策，不是巧合。** 每个页面先选密度档位（§5），再挑组件尺寸，不要所有页面都用同一套间距。
4. **层级靠对比，不靠堆阴影。** 主次由字号、字重、`--text-*` 色阶和留白区分；阴影基本只有 `xs`，卡片最多 `sm`。
5. **深色是一等公民。** 任何视觉改动都要在 `.dark` 下看一眼。做不到就别合。

---

## 2. 令牌

全部定义在 `src/index.css` 的 `:root` 与 `.dark` 两个块中，单位为 `H S% L%` 的 HSL 分量，通过 `hsl(var(--token) / <alpha-value>)` 暴露到 Tailwind（`tailwind.config.js`）。

### 2.1 表面阶梯（surface）

| 令牌 | 用途 |
| --- | --- |
| `--surface` | 输入框、组件本体底色 |
| `--surface-raised` | 卡片、侧栏、顶栏 |
| `--surface-sunken` | 页面背景、内嵌区块 |
| `--surface-inset` | 表头、代码块、骨架屏基色 |
| `--surface-overlay` | 弹层 |
| `--surface-hover` / `--surface-active` | 悬停 / 按下 |

规则：页面背景永远是 `--surface-sunken`；卡片是 `--surface-raised`；卡片里再套一层只用 `--surface-sunken`，不再抬升。

### 2.2 文字阶梯（text）

`--text-primary`（正文与标题）→ `--text-secondary`（次要正文）→ `--text-tertiary`（辅助说明、表头）→ `--text-disabled`（占位符、分隔符）。超过三级就说明信息该拆。

### 2.3 边框

`--border`（默认）、`--border-strong`（可交互控件、需要抓眼的边界）、`--border-subtle`（区块内分隔）。

### 2.4 品牌与状态

- 品牌就是对比：`--primary` 浅色下 `0 0% 0%`（纯黑）、深色下 `0 0% 92.2%`（近白）。hover 用 `bg-primary/90` 降透明度，不要另造一个色相。
- `--accent` 是**中性悬停底色**（浅 `0 0% 96.1%` / 深 `0 0% 22%`），即 shadcn 的 `accent`，不是第二个品牌色。
- 状态：`success` / `warning` / `danger` / `info` 各四支 `*-bg`、`*-fg`、`*-border`、`*-solid`。

状态语义固定映射，不要按感觉换：

| 语义 | 用于 |
| --- | --- |
| `success` | running / active / enabled / acked |
| `warning` | pending / 降级 |
| `danger` | error / 失败 |
| `info` | delivered / 进行中 / 提示 |
| `neutral` | stopped / inactive / 未知 |

### 2.5 半径、阴影、节奏

- **半径只有一档真值** `--radius`（0.625rem），其余档位按 0.4/0.6/0.8/1/1.4/1.8 **相乘**推导。改圆角只改 `--radius`，不要逐档改。
- 用法：控件 `md`、卡片 `xl`、徽标 `full`（药丸）、代码 `xs`。
- 阴影极轻且只有一档 `xs`（输入框、outline 按钮）。卡片靠 hairline 而非阴影。
- 节奏变量：`--gutter`（页面横向留白）、`--stack-gap`（区块间距）、`--density-row`（行高）、`--content-max`（内容最大宽度）、`--sidebar-width*`、`--topbar-height`。
- 字号阶梯由 `tailwind.config.js` 的 `fontSize` 给到。正文 `text-sm`，辅助说明 `text-xs`，`text-2xs` 只留给表头这类全大写密排标签。

### 2.6 加一个新令牌

1. 在 `:root` 定义，并在 `.dark` 给出对应值。
2. 在 `tailwind.config.js` 的 `colors` 里用 `withOpacity("--新令牌")` 暴露。
3. 在本文 §2 补一行说明。
4. 不要在页面里直接写 `hsl(var(--x))`，除非是渐变或多值阴影。

### 2.7 全局基线

`:focus-visible` 统一 `border-ring` + `ring-2 ring-ring/50`，**不设 offset 间隙**；滚动条、`::selection`（纯主色块）、抗锯齿、`color-scheme`、`prefers-reduced-motion` 都在 `index.css` 里统一处理，页面不用管。数字一律加 `.tabular`。

---

## 3. 组件

全部位于 `src/components/ui/`，基于 `cva` + `cn`（`clsx` + `tailwind-merge`）。`cn` 保证后写的类能覆盖先写的。

### Button

`variant`：`primary`（默认；近黑填充，每屏一个主操作）/ `outline`（描边，日常次操作首选）/ `secondary`（中性填充，需要比 outline 强一点）/ `ghost`（工具栏、图标）/ `danger`（破坏性）/ `link`（行内文字操作）。
`size`：`sm` / `md` / `icon` / `icon-sm`。`asChild` 可把样式挂到 `<Link>` 上。hover 一律用 `bg-primary/90` 这类透明度，不新增色相。

```tsx
<Button variant="primary" size="sm"><Plus />{t('platforms.create')}</Button>
<Button variant="outline" size="sm">{t('common.cancel')}</Button>
<Button variant="ghost" size="icon-sm" aria-label={t('common.edit')}><Pencil /></Button>
```

图标按钮必须有 `aria-label`。

### Card

`variant`：`default`（hairline + `shadow-sm`）/ `sunken` / `elevated` / `success` / `warning` / `danger` / `info` / `ghost`；`padding`：`none` / `tight` / `default` / `roomy`。
子件：`CardHeader`（`p-6`，纵向栈）/ `CardTitle` / `CardDescription` / `CardBody`（`p-6 pt-0`）/ `CardFooter`（`p-6 pt-0`）。
带子件的卡片用 `padding="none"` 让子件自己管内边距；表格类卡片同理。卡片圆角固定 `xl`。

### Badge

`variant` 只传语义（§2.4），`dot` 加状态点。颜色由组件决定，调用点不拼三元。徽标是**药丸形** `rounded-full` + `text-xs`，只有淡底色、边框为 transparent。

```tsx
<Badge variant={running ? 'success' : 'neutral'} dot>{label}</Badge>
```

### TableShell

包住 `<table>`，负责卡片外观、表头样式、行 hover、横向滚动（`min-w` 保证窄屏不塌）、空态插槽。
子件：`TableHead`、`TableBody`、`TableRow`、`TableHeaderCell`、`TableCell`。
`empty` 接空态/骨架；`toolbar` 放计数、筛选；`striped` 仅用于长列表；`busy` 传加载中。

每个单元格自带 `min-w-[...]`；长文本用 `.truncate-cell` 并给 `title`。

```tsx
<TableShell busy={loading} empty={empty} toolbar={<span className="tabular text-xs text-text-tertiary">{t('platforms.count', { count })}</span>}>
  <TableHead><tr><TableHeaderCell className="min-w-[11rem]">{t('common.name')}</TableHeaderCell></tr></TableHead>
  <TableBody>{rows}</TableBody>
</TableShell>
```

### PageHeader

`title` + `description` + `actions`（主操作放最右，最多一个 primary）+ 可选 `eyebrow`。每个页面第一块就是它，且唯一。

### EmptyState

`icon` + `title` + `description` + 可选 `action`；`size="page"` 用于整页空，默认 `inline` 用于卡内。文案说清「缺什么 + 怎么办」。

### ErrorState

`title` + `description` + `onRetry`（有就出「重试」按钮）。错误态一律用它，不要手写红框。

### FormField

`label` + `hint` / `error` + 控件，自动接 `id`、`aria-describedby`、`aria-invalid`。控件用 render-prop 或单元素，className 用导出的 `controlClassName`。

```tsx
<FormField label={t('settings.siteName')} hint={t('settings.siteNameHint')} required>
  {(props) => <input {...props} className={controlClassName} value={v} onChange={...} />}
</FormField>
```

### Select

`@radix-ui/react-select` 封装，与 shadcn 同 API：`Select` / `SelectTrigger` / `SelectValue` / `SelectContent` / `SelectItem`。触发器 `h-9`、透明底、hairline 边框。**禁止再写原生 `<select>`。**

```tsx
<Select value={v} onValueChange={setV}>
  <SelectTrigger><SelectValue /></SelectTrigger>
  <SelectContent><SelectItem value="a">A</SelectItem></SelectContent>
</Select>
```

### 开关键（Switch）

`@radix-ui/react-switch` 未安装，用 `role="switch"` 的原生 checkbox 伪装：轨道 `h-5 w-9 rounded-full bg-input`，选中 `bg-primary`，滑块 `size-4` 平移。只用于设置页的布尔项。

### Skeleton

`shape`：`block` / `line` / `circle`。另有 `SkeletonText`、`SkeletonRows`。表格加载用 `SkeletonRows`，卡片区用 `Skeleton`，禁止再出现「加载中…」纯文字。

### StatCard / StatGrid

`label` + `value`，可选 `icon`、`tone`、`caption`、`loading`。`StatGrid` 固定为 2/4 列。一屏最多一组，超过就是没排层级。

---

## 4. 深浅色

- `ThemeProvider`（`components/theme-provider.tsx`）读懂 `localStorage['espl.theme']`，无值时跟随 `prefers-color-scheme`，并给 `<html>` 加/去 `.dark` + 写 `color-scheme`。系统偏好变化时实时跟随，跨标签页同步。
- 页面不直接读 `localStorage`，用 `useTheme()` 的 `toggleTheme` / `resolvedTheme`。
- 深色下不靠降透明度做层次；用 `.dark` 的令牌值。`primary` 在深色下自动翻成近白（shadcn 的做法），所以"主按钮"两种模式都自然成立，不需要分支。
- `Login` 品牌区用 `--primary` 做大面积近黑色块，叠一层极淡的 `--primary-foreground` 径向光，不引入品牌色相。

---

## 5. 页面密度

| 档 | 用途 | 表现 |
| --- | --- | --- |
| 稀疏 | 登录、空态页、文档 | 大留白、`text-3xl` 标题、单列 |
| 标准 | Platforms / Instances / Routes | `PageHeader`（带分割线）+ 表格，单元格 `py-3` |
| 紧凑 | Messages / Monitoring | 筛选药丸、斑马纹、`text-2xs` 表头、指标条 |
| 分栏 | Settings | 左侧分区导航 + 右侧单分区 `Card` |
| 卡片网格 | Adapters | `sm:2 / xl:3`，卡内操作在 footer |
| 编辑式 | Docs | 链接卡 + 编号架构区，几乎无表格 |

所有档位的**基础节奏一致**（`gap-4`、卡片 `p-6`、`PageHeader` 收一条 hairline），差异来自"一屏放多少东西"，不是来自另一套尺寸。shadcn 本身就偏紧凑，别为了"显得丰富"把间距放大。

禁止「十个页面一个模子」：选档位时先问这一页是「扫」还是「读」，扫用紧凑/表格，读用稀疏/编辑式。

---

## 6. 新增一个页面

1. 在 `src/pages/` 建文件，默认导出组件。
2. 在 `App.tsx` 的 `Layout` 路由下挂路由；在 `Layout.tsx` 的 `NAV_ITEMS` 加条目，选 `group`（`business` / `ops` / `settings`），补 `nav.*` key。
3. 第一块固定 `PageHeader`，`title`/`description` 都走 i18n。
4. 选 §5 的密度档，用 §3 的组件拼，加载态接 `Skeleton*`，空态接 `EmptyState`，错误态接 `ErrorState`。
5. 所有文案进 `zh-CN.json` 与 `en-US.json`，**两个文件 key 必须完全一致**；不删既有 key。
6. 列表页确保每个单元格有 `min-w`，长文本 `.truncate-cell` + `title`。
7. 自检：`grep -nE "(bg|text|border)-(gray|slate|blue|red|green|yellow|purple|orange|zinc|neutral)-[0-9]" src/pages/新页面.tsx` 应为空。
8. 深色下过一遍；`prefers-reduced-motion` 下过一遍。
9. 下拉一律 `Select`，开关一律 `role="switch"`，不要混用原生控件。

### 结构骨架

```tsx
export default function Example() {
  const { t } = useTranslation()
  return (
    <>
      <PageHeader title={t('example.title')} description={t('example.subtitle')}
        actions={<Button variant="primary" size="sm"><Plus />{t('common.create')}</Button>} />
      {error ? <ErrorState title={t('state.loadFailed')} description={error} onRetry={load} retryLabel={t('common.retry')} />
        : <TableShell busy={loading} empty={loading ? <SkeletonRows /> : <EmptyState icon={Icon} title={...} description={...} />}>
            {/* TableHead / TableBody */}
          </TableShell>}
    </>
  )
}
```

---

## 7. 验收自检

- 无调色板类名（`Login` 品牌区的径向光除外）。
- 焦点环是 `ring-ring/50` 且无 offset。
- 无手写 `animate-spin` 加载、无纯文字「加载中」。
- 无内联三元拼状态颜色。
- 新 key 中英对称。
- 表格在 320px 宽不横向溢出页面（表格自身可滚，页面不滚）。
- 所有图标按钮有 `aria-label`，`TableShell` 有 `busy`。
