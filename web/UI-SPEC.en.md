# E-SP-Line2 admin UI specification

This document is the single source of truth for visuals and component conventions in the `web/` admin front end. Read it before adding a page, copy the structure of an existing page, and do not reinvent styling.

[简体中文](UI-SPEC.md) · **English**

Scope: `web/src/index.css`, `web/tailwind.config.js`, `web/src/components/**`, `web/src/lib/cn.ts`.

> **Design language: shadcn/ui (https://ui.shadcn.com, v4 "base").** This is not "borrow the palette" — it is the whole contract, with values taken straight from its `globals.css`:
> the neutral ramp is pure grey (`0 0% x`), body text is pure black, borders are `hsl(0 0% 90%)`;
> `primary` is **pure black** on light and near-white on dark — it supplies contrast, not hue;
> `--ring` is a **mid grey** (`0 0% 63%`), deliberately not the primary;
> there is exactly one radius, `--radius`, and every other step **multiplies** it; the only shadow is `xs`;
> focus is `border-ring` + 2px `ring-ring/50`, with **no offset gap**.
> Before changing any visual, look at the matching component on ui.shadcn.com and match its geometry and restraint.

---

## 1. Principles

1. **Colours come from tokens only.** No palette class names such as `bg-blue-600`, `text-gray-500` or `border-red-200` in a page. Need a colour? Find the token; if there is none, add the token first (see §2.6).
2. **One semantic, one component.** Status uses `Badge`, containers use `Card` (with `CardAction` for a top-right slot), tables use `TableShell`, dropdowns use `Select`, loading uses `Skeleton`, empty states use `EmptyState`, errors use `ErrorState`, headers use `PageHeader`, form fields use `FormField`.
3. **Density is a design decision, not an accident.** Pick the page's density tier (§5) before choosing component sizes; do not give every page the same spacing.
4. **Hierarchy comes from contrast, not stacked shadows.** Rank content by size, weight, the `--text-*` scale and whitespace; shadows are basically just `xs`, `sm` at most on cards.
5. **Dark mode is a first-class citizen.** Look at every visual change under `.dark`. If you cannot, do not merge it.

---

## 2. Tokens

All of them are defined in the `:root` and `.dark` blocks of `src/index.css` as `H S% L%` HSL components, exposed to Tailwind through `hsl(var(--token) / <alpha-value>)` (`tailwind.config.js`).

### 2.1 Surface ladder

| Token | Purpose |
| --- | --- |
| `--surface` | Inputs, the component itself |
| `--surface-raised` | Cards, sidebar, top bar |
| `--surface-sunken` | Page background, inset blocks |
| `--surface-inset` | Table headers, code blocks, skeleton base |
| `--surface-overlay` | Popovers |
| `--surface-hover` / `--surface-active` | Hover / pressed |

Rule: the page background is always `--surface-sunken`; a card is `--surface-raised`; nesting another layer inside a card uses `--surface-sunken` and never raises it again.

### 2.2 Text ladder

`--text-primary` (body and titles) → `--text-secondary` (secondary body) → `--text-tertiary` (captions, table headers) → `--text-disabled` (placeholders, separators). Four levels mean the information should be split up.

### 2.3 Borders

`--border` (default), `--border-strong` (interactive controls and boundaries that must be seen), `--border-subtle` (dividers inside a block).

### 2.4 Brand and status

- Brand *is* contrast: `--primary` is `0 0% 0%` (pure black) on light and `0 0% 92.2%` (near-white) on dark. Hover is `bg-primary/90`, never a second hue.
- `--accent` is the **neutral hover wash** (`0 0% 96.1%` light / `0 0% 22%` dark) — shadcn's `accent`, not a second brand colour.
- Status: each of `success` / `warning` / `danger` / `info` has four branches, `*-bg`, `*-fg`, `*-border` and `*-solid`.

Status semantics are fixed. Do not swap them on a hunch:

| Semantic | Used for |
| --- | --- |
| `success` | running / active / enabled / acked |
| `warning` | pending / degraded |
| `danger` | error / failed |
| `info` | delivered / in progress / informational |
| `neutral` | stopped / inactive / unknown |

### 2.5 Radii, shadows, rhythm

- **There is one real radius**, `--radius` (0.625rem); `xs/sm/md/lg/xl/2xl` **multiply** it by 0.4/0.6/0.8/1/1.4/1.8. Change `--radius`, not the individual steps.
- Usage: controls `md`, cards `xl`, badges `full` (pills), code `xs`.
- Shadows are barely there and there is only `xs` (inputs, outline buttons). Cards rely on the hairline, not on shadow.
- Rhythm variables: `--gutter` (page side padding), `--stack-gap` (gap between blocks), `--density-row` (row height), `--content-max` (max content width), `--sidebar-width*`, `--topbar-height`.
- The type ladder comes from `fontSize` in `tailwind.config.js`. Body is `text-sm`, captions `text-xs`, and `text-2xs` is reserved for dense uppercase labels like table headers.

### 2.6 Adding a token

1. Define it in `:root` and give it a counterpart in `.dark`.
2. Expose it in the `tailwind.config.js` `colors` with `withOpacity("--new-token")`.
3. Add a line to §2 here.
4. Do not write `hsl(var(--x))` directly in a page unless it is a gradient or a multi-value shadow.

### 2.7 Global baseline

`:focus-visible` is `border-ring` + `ring-2 ring-ring/50`, with **no offset gap**; scrollbars, `::selection` (a solid primary block), antialiasing, `color-scheme` and `prefers-reduced-motion` are all handled in `index.css`, so pages do not deal with them. Numbers always get `.tabular`.

---

## 3. Components

All of them live in `src/components/ui/` and are built on `cva` + `cn` (`clsx` + `tailwind-merge`). `cn` guarantees a later class overrides an earlier one.

### Button

`variant`: `primary` (default; near-black fill, one main action per screen) / `outline` (bordered, the first choice for routine secondary actions) / `secondary` (neutral fill, a step stronger than outline) / `ghost` (toolbars, icons) / `danger` (destructive) / `link` (inline text action).
`size`: `sm` / `md` / `icon` / `icon-sm`. `asChild` moves the styling onto a `<Link>`. Hover always uses opacity such as `bg-primary/90`, never a new hue.

```tsx
<Button variant="primary" size="sm"><Plus />{t('platforms.create')}</Button>
<Button variant="outline" size="sm">{t('common.cancel')}</Button>
<Button variant="ghost" size="icon-sm" aria-label={t('common.edit')}><Pencil /></Button>
```

Icon buttons must have an `aria-label`.

### Card

`variant`: `default` (hairline + `shadow-sm`) / `sunken` / `elevated` / `success` / `warning` / `danger` / `info` / `ghost`; `padding`: `none` / `tight` / `default` / `roomy`.
Sub-components: `CardHeader` (`p-6`, vertical stack) / `CardTitle` / `CardDescription` / `CardBody` (`p-6 pt-0`) / `CardFooter` (`p-6 pt-0`).
A card with sub-components uses `padding="none"` so the sub-components own their padding; table cards do the same. Card radius is fixed at `xl`.

### Badge

`variant` takes the semantic only (§2.4); `dot` adds a status dot. The component decides the colour, so the call site never builds a ternary. A badge is a **pill** (`rounded-full`, `text-xs`) with a tinted ground and a transparent border.

```tsx
<Badge variant={running ? 'success' : 'neutral'} dot>{label}</Badge>
```

### TableShell

Wraps a `<table>` and owns the card look, header styling, row hover, horizontal scrolling (`min-w` keeps narrow screens from collapsing) and the empty slot.
Sub-components: `TableHead`, `TableBody`, `TableRow`, `TableHeaderCell`, `TableCell`.
`empty` takes the empty or skeleton state; `toolbar` holds counts and filters; `striped` is for long lists only; `busy` passes loading through.

Every cell carries its own `min-w-[...]`; long text uses `.truncate-cell` and gets a `title`.

```tsx
<TableShell busy={loading} empty={empty} toolbar={<span className="tabular text-xs text-text-tertiary">{t('platforms.count', { count })}</span>}>
  <TableHead><tr><TableHeaderCell className="min-w-[11rem]">{t('common.name')}</TableHeaderCell></tr></TableHead>
  <TableBody>{rows}</TableBody>
</TableShell>
```

### PageHeader

`title` + `description` + `actions` (primary action rightmost, at most one primary) plus an optional `eyebrow`. It is the first block of every page and appears exactly once.

### EmptyState

`icon` + `title` + `description` plus an optional `action`; `size="page"` is for a whole empty page, the default `inline` is for inside a card. The copy states what is missing and what to do about it.

### ErrorState

`title` + `description` + `onRetry` (supplying it renders a retry button). Every error state uses it; do not hand-write a red box.

### FormField

`label` + `hint` / `error` + control, wiring `id`, `aria-describedby` and `aria-invalid` for you. The control is either a render prop or a single element, with className from the exported `controlClassName`.

```tsx
<FormField label={t('settings.siteName')} hint={t('settings.siteNameHint')} required>
  {(props) => <input {...props} className={controlClassName} value={v} onChange={...} />}
</FormField>
```

### Select

A thin wrapper over `@radix-ui/react-select` with the shadcn API: `Select` / `SelectTrigger` / `SelectValue` / `SelectContent` / `SelectItem`. The trigger is `h-9`, transparent, hairline-bordered. **Never hand-write a native `<select>` again.**

```tsx
<Select value={v} onValueChange={setV}>
  <SelectTrigger><SelectValue /></SelectTrigger>
  <SelectContent><SelectItem value="a">A</SelectItem></SelectContent>
</Select>
```

### Switch

`@radix-ui/react-switch` is not installed, so a native checkbox with `role="switch"` stands in: track `h-5 w-9 rounded-full bg-input`, checked `bg-primary`, thumb `size-4` translating. Only for boolean settings rows.

### Skeleton

`shape`: `block` / `line` / `circle`. `SkeletonText` and `SkeletonRows` are provided too. Loading a table uses `SkeletonRows`, loading a card area uses `Skeleton`, and the bare text "Loading…" is not allowed any more.

### StatCard / StatGrid

`label` + `value`, with optional `icon`, `tone`, `caption` and `loading`. `StatGrid` is fixed at 2 or 4 columns. One group per screen at most; more means the hierarchy is missing.

---

## 4. Light and dark

- `ThemeProvider` (`components/theme-provider.tsx`) reads `localStorage['espl.theme']`, follows `prefers-color-scheme` when unset, and adds/removes `.dark` on `<html>` plus writes `color-scheme`. It follows system preference changes live and syncs across tabs.
- Pages never read `localStorage` directly; use `toggleTheme` / `resolvedTheme` from `useTheme()`.
- In dark mode, build depth from token values, not from reduced opacity. `primary` flips to near-white in dark automatically (shadcn's behaviour), so the "primary button" works in both modes with no branching.
- The `Login` brand panel uses `--primary` as a large near-black field with a very faint `--primary-foreground` radial glow; no brand hue is introduced.

---

## 5. Page density

| Tier | Purpose | Shape |
| --- | --- | --- |
| Sparse | Login, empty pages, docs | Heavy whitespace, `text-3xl` titles, single column |
| Standard | Platforms / Instances / Routes | `PageHeader` (with rule) + table, cells at `py-3` |
| Compact | Messages / Monitoring | Filter pills, zebra stripes, `text-2xs` headers, metric strip |
| Split | Settings | Section nav on the left, one section `Card` on the right |
| Card grid | Adapters | `sm:2 / xl:3`, actions in the card footer |
| Editorial | Docs | Link cards plus a numbered architecture section, almost no tables |

Every tier shares the **same base rhythm** (`gap-4`, card `p-6`, `PageHeader` closing with a hairline); the difference is how much fits on one screen, not a second set of dimensions. shadcn is compact by nature — do not inflate spacing to look "rich".

No "one mould for ten pages": before picking a tier, ask whether the page is scanned or read. Scanned pages use compact or table tiers; read pages use sparse or editorial.

---

## 6. Adding a page

1. Create the file under `src/pages/`, default-exporting the component.
2. Mount the route under the `Layout` route in `App.tsx`; add an entry to `NAV_ITEMS` in `Layout.tsx` with a `group` (`business` / `ops` / `settings`) and the `nav.*` key.
3. The first block is always `PageHeader`, with the `title` and `description` going through i18n.
4. Pick the §5 density tier, assemble §3 components, wire loading to `Skeleton*`, the empty state to `EmptyState` and errors to `ErrorState`.
5. Every string goes into `zh-CN.json` and `en-US.json`, where **both files must have exactly the same keys**; never delete an existing key.
6. On a list page, give every cell a `min-w`, and give long text `.truncate-cell` plus a `title`.
7. Self-check: `grep -nE "(bg|text|border)-(gray|slate|blue|red|green|yellow|purple|orange|zinc|neutral)-[0-9]" src/pages/new-page.tsx` must return nothing.
8. Run through it in dark mode and with `prefers-reduced-motion`.
9. Dropdowns are always `Select`, toggles always `role="switch"` — do not mix in native controls.

### Structural skeleton

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

## 7. Acceptance checks

- No palette class names (apart from the `Login` brand glow).
- The focus ring is `ring-ring/50` with no offset.
- No hand-written `animate-spin` loaders and no bare "Loading" text.
- No inline ternaries choosing a status colour.
- New keys are symmetric in both locales.
- A table does not overflow the page at 320px wide — the table scrolls, the page does not.
- Every icon button has an `aria-label` and every `TableShell` gets `busy`.
