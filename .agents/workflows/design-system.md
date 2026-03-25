---
description: OBLIVRA Master Design System — load this first as context before generating any UI
---

# OBLIVRA Design System — Master Reference

> **PURPOSE**: This is the single source of truth for all UI generation.
> Load this file FIRST before writing any component, page, or layout.

---

## 1. Platform Context

**OBLIVRA** is a SOC (Security Operations Center) platform — a Wails desktop app with a SolidJS frontend.
It runs in two contexts: **desktop** (Wails) and **browser** (web deployment).

- **Framework**: SolidJS 1.8+ (NOT React — no `useState`, no `useEffect`)
- **Styling**: CSS custom properties in `variables.css` + vanilla CSS files in `src/styles/`
- **Routing**: `@solidjs/router` with `HashRouter`
- **State**: Custom store via `@core/store` → `useApp()`
- **Icons**: Inline SVG (no icon library) — 24×24 viewBox, stroke-based, `stroke-width="1.6"`
- **Backend**: Wails — Go services imported from `wailsjs/go/services/`
- **Build**: Vite 7+ with `vite-plugin-solid`

---

## 2. Color System (from `variables.css`)

### Surfaces (dark → lighter)
```
--surface-0:  #1a1c20   (page background)
--surface-1:  #212327   (panels, nav, cards)
--surface-2:  #2b2d31   (elevated panels, headers)
--surface-3:  #33363c   (hover states)
--surface-4:  #3d4148   (active/pressed states)
```

### Borders
```
--border-subtle:    rgba(255,255,255,0.04)
--border-primary:   #3a3d44    (default borders)
--border-secondary: #4a4d55    (stronger borders)
--border-hover:     #52555e    (hover state borders)
--border-active:    #0099e0    (active/focus borders)
```

### Semantic Status — NEVER use raw colors, always use these tokens
```
CRITICAL: --alert-critical (#e04040)  |  RED — broken, failed, attack
HIGH:     --alert-high     (#f58b00)  |  ORANGE — degraded, warning
MEDIUM:   --alert-medium   (#f5c518)  |  YELLOW — needs attention
LOW:      --alert-low      (#5cc05c)  |  GREEN — healthy, resolved
INFO:     --alert-info     (#0099e0)  |  BLUE — informational

ONLINE:   --status-online  (#5cc05c)
DEGRADED: --status-degraded(#f58b00)
OFFLINE:  --status-offline (#e04040)
```

### Accent
```
--accent-primary:   #0099e0   (links, active states, focus rings)
--accent-cta:       #f58b00   (primary action buttons — the "Splunk orange")
--accent-cta-hover: #d97a00
--accent-danger:    #e04040
```

### Text
```
--text-primary:   #d4d5d8   (body text)
--text-secondary: #9b9ea4   (labels, descriptions)
--text-muted:     #6b6e76   (hints, disabled, section headers)
--text-heading:   #e8e9eb   (titles, headings)
--text-accent:    #0099e0   (links, interactive text)
```

---

## 3. Typography

### Fonts
```
--font-ui:   "Inter", system-ui, sans-serif    → ALL UI labels, text, buttons
--font-mono: "JetBrains Mono", monospace       → logs, IDs, timestamps, code, data values
```

### Sizes (intentionally small for density)
```
--font-size-2xs:  9px    → micro labels, status bar
--font-size-xs:   11px   → secondary info, badges
--font-size-sm:   12px   → body text, table cells, buttons
--font-size-base: 13px   → default body (used sparingly)
--font-size-md:   14px   → section titles
--font-size-lg:   16px   → page titles only
--font-size-xl:   20px   → hero numbers (KPI values)
--font-size-2xl:  24px   → large KPI values
```

### Rules
- Body text: `12-13px`, `font-weight: 400-500`
- Section headers: `10px`, `font-weight: 700`, `uppercase`, `letter-spacing: 0.7-0.8px`
- Button text: `11-12px`, `font-weight: 500-600`
- Monospace for: IDs, timestamps, hashes, metrics, log lines, code
- NEVER go above 16px for anything except KPI numbers

---

## 4. Spacing System

### Gaps
```
--gap-xs:  4px    --gap-sm:  8px    --gap-md:  12px
--gap-lg:  16px   --gap-xl:  24px   --gap-2xl: 32px
```

### Padding
```
--padding-xs: 4px 8px     → tiny elements
--padding-sm: 6px 12px    → buttons, badges
--padding-md: 8px 16px    → inputs, table cells
--padding-lg: 12px 20px   → panel headers
--padding-xl: 16px 24px   → page sections
```

### Radius — keep it sharp
```
--radius-xs:   2px    → badges, small elements
--radius-sm:   3px    → buttons, inputs
--radius-md:   4px    → cards, panels  ← MAX for most elements
--radius-lg:   6px    → modals
--radius-full: 9999px → status dots, pills
```

**RULE**: Never use `border-radius` > 6px except for circular elements.

---

## 5. Layout Constants

```
--header-height:       40px    → top title bar
--nav-collapsed-width: 48px    → CommandRail (icon-only sidebar)
--sidebar-width:       260px   → DrawerPanel (host/snippet list)
--status-bar-height:   24px    → bottom status bar
```

The app shell is:
```
┌──────────────── TitleBar (40px) ────────────────────┐
│ Logo │ Tabs │ Search Pill │ User │ Window Controls   │
├──────┬───────────────────────────────────────────────┤
│ C    │ Drawer │       Main Content                   │
│ R    │ Panel  │       (routed pages)                 │
│ a    │ (260px)│                                      │
│ i    │        │                                      │
│ l    │        │                                      │
│ 64px │        │                                      │
├──────┴───────────────────────────────────────────────┤
│ StatusBar (24px)                                     │
└──────────────────────────────────────────────────────┘
```

---

## 6. Existing CSS Classes (USE THESE, don't reinvent)

### Buttons
- `.ob-btn` → default button
- `.ob-btn-primary` → orange CTA
- `.ob-btn-blue` → blue secondary
- `.ob-btn-danger` → red destructive
- `.ob-btn-ghost` → transparent
- `.ob-btn-sm` / `.ob-btn-lg` → sizes

### Inputs
- `.ob-input` → text input
- `.ob-input-mono` → monospace input
- `.ob-select` → dropdown select
- `.ob-textarea` → multiline
- `.ob-search-bar` → Splunk-style search with button

### Layout
- `.ob-page` → full-height scrollable page container
- `.ob-page-header` → page header with title + actions
- `.ob-page-title` / `.ob-page-subtitle`
- `.ob-toolbar` → horizontal toolbar strip
- `.ob-panel-header` / `.ob-panel-title` → inner panel chrome

### Data
- `.ob-table-wrap` + `.ob-table` → dense data table
- `.ob-kpi` → KPI metric card
- `.ob-stat-grid` + `.ob-stat-grid-4` → grid of KPI cards
- `.ob-badge` + `.ob-badge-green/red/yellow/blue/gray` → status pills
- `.sev-critical/high/medium/low/info` → severity badges

### Cards & Containers
- `.ob-card` → bordered card
- `.ob-card-flat` → no side borders
- `.ob-code` → code/log block
- `.ob-notice` + `.ob-notice-info/warn/error/success` → notification strips

### Navigation
- `.ob-tabs` + `.ob-tab` → horizontal tab bar
- `.ob-breadcrumb` → breadcrumb trail

### State
- `.ob-skeleton` + `.ob-skeleton-text/block` → loading skeletons
- `.ob-empty-state` → empty state placeholder
- `.ob-loading` → loading indicator
- `.ob-load-error` → error state

### Modals
- `.ob-modal-overlay` → backdrop
- `.ob-modal` → modal container  
- `.ob-modal-header/body/actions` → modal sections

---

## 7. Component Architecture Rules

### SolidJS Patterns (NOT React)
```tsx
// ✅ CORRECT — SolidJS
import { Component, createSignal, Show, For, onMount } from 'solid-js';

const MyComponent: Component = () => {
    const [value, setValue] = createSignal('');
    return <div>{value()}</div>;  // Note: value() — call the signal!
};

// ❌ WRONG — React patterns
const MyComponent = () => {
    const [value, setValue] = useState('');  // NO useState
    return <div>{value}</div>;  // Signals must be called
};
```

### Import Aliases
```tsx
import { useApp } from '@core/store';      // App state
import { IS_DESKTOP, IS_BROWSER } from '@core/context';
import { SomeComponent } from '@components/ui/SomeComponent';
```

### Backend Calls (desktop only)
```tsx
// Always lazy-import Wails services
const fetchData = async () => {
    if (IS_BROWSER) return;  // Guard for browser context
    try {
        const { ServiceMethod } = await import('../../../wailsjs/go/services/ServiceName');
        const result = await ServiceMethod(args);
    } catch (err) {
        setError(String(err));
    }
};
```

### Component File Structure
```
src/components/{domain}/
  ComponentName.tsx        → component code
src/styles/
  {domain}.css             → matching CSS file
src/pages/
  PageName.tsx             → full page components
```

### Export Patterns
```tsx
// Named exports for components
export const MyComponent: Component = () => { ... };

// Default exports for pages (for lazy loading)
export default MyComponent;
// OR
export { MyComponent };
```

---

## 8. UX Laws — ABSOLUTE RULES

### Rule 1: DENSITY OVER DECORATION
- No empty space waste. Every pixel must serve a purpose.
- Max padding on any element: `16px 24px`.
- No large hero sections. No centered marketing-style layouts.
- Tables, grids, and lists dominate. Cards are tight.

### Rule 2: EVERY SCREEN ANSWERS TWO QUESTIONS
```
1. "What is broken?"  → Status indicators, severity badges, error counts
2. "Where do I act?"  → Clear primary action, highlighted rows, contextual buttons
```

### Rule 3: SPEED > BEAUTY
- Zero lag interactions. Instant state feedback.
- No decorative animations. Only state-conveying animations:
  - `pulse` → live/updating data
  - `spin` → loading
  - `fade-in` (0.15-0.2s max) → panel appearance
- No glassmorphism, no gradients (except accent-gradient for rare CTAs).

### Rule 4: CONSISTENCY > CREATIVITY
- Same layout patterns everywhere.
- Same spacing tokens. Same border colors. Same font sizes.
- No one-off styling. If it's not in `variables.css` or `components.css`, don't use it.

### Rule 5: MONOSPACE FOR DATA
- All IDs, hashes, timestamps, log lines, metrics → `font-family: var(--font-mono)`
- All labels, titles, descriptions, buttons → `font-family: var(--font-ui)`

### Rule 6: UPPERCASE FOR SECTION HEADERS
- Section dividers: `10px`, `uppercase`, `font-weight: 700`, `letter-spacing: 0.7px`, `color: var(--text-muted)`
- Page titles: `var(--font-size-lg)`, `font-weight: 700`, normal case

### Rule 7: NO VAGUE LABELS
- ❌ "Overview", "Dashboard", "Stats"
- ✅ "Active Threats", "Failed Logins (24h)", "Endpoint Compliance"
- Every label must tell the analyst exactly what they're looking at.

---

## 9. Page Template

Every new page should follow this skeleton:

```tsx
import { Component, createSignal, onMount, Show, For } from 'solid-js';
import { useApp } from '@core/store';
import { IS_BROWSER } from '@core/context';
import '../../styles/{domain}.css';

export const PageName: Component = () => {
    const [state] = useApp();
    const [loading, setLoading] = createSignal(true);
    const [error, setError] = createSignal<string | null>(null);
    const [data, setData] = createSignal<DataType[]>([]);

    const fetchData = async () => {
        setLoading(true);
        setError(null);
        if (IS_BROWSER) { setLoading(false); return; }
        try {
            const { Method } = await import('../../../wailsjs/go/services/Service');
            const res = await Method();
            setData(res || []);
        } catch (err) {
            setError(String(err));
        } finally {
            setLoading(false);
        }
    };

    onMount(() => fetchData());

    return (
        <div class="ob-page">
            {/* Page Header */}
            <div class="ob-page-header">
                <div>
                    <div class="ob-page-title">Page Title</div>
                    <div class="ob-page-subtitle">OPERATIONAL_CONTEXT</div>
                </div>
                <div style="display: flex; gap: 8px;">
                    <button class="ob-btn" onClick={fetchData}>Refresh</button>
                    <button class="ob-btn ob-btn-primary">Primary Action</button>
                </div>
            </div>

            {/* Error State */}
            <Show when={error()}>
                <div class="ob-notice ob-notice-error">{error()}</div>
            </Show>

            {/* Loading State */}
            <Show when={loading()}>
                <div class="ob-loading">LOADING...</div>
            </Show>

            {/* Content */}
            <Show when={!loading() && data().length > 0}>
                {/* Dense grid/table layout here */}
            </Show>

            {/* Empty State */}
            <Show when={!loading() && data().length === 0 && !error()}>
                <div class="ob-empty-state">
                    <div class="ob-empty-title">NO DATA</div>
                    <div class="ob-empty-desc">Description of what's missing</div>
                </div>
            </Show>
        </div>
    );
};
```

---

## 10. File Naming Conventions

| Type | Pattern | Example |
|------|---------|---------|
| Component | `PascalCase.tsx` | `AlertDashboard.tsx` |
| Page | `PascalCase.tsx` | `OpsCenter.tsx` |
| CSS | `kebab-case.css` | `ops-center.css` |
| Store/Util | `camelCase.ts` | `bridge.ts` |
| Types | `camelCase.ts` | `types.ts` |

---

## 11. Checklist Before Generating UI

- [ ] Uses only tokens from `variables.css` (no hardcoded colors)
- [ ] Uses existing CSS classes from `components.css` (no reinventing)
- [ ] SolidJS patterns (not React) — signals called with `()`, `Show`/`For`/`Switch`
- [ ] All data values use `font-mono`
- [ ] All section headers use `uppercase`, `10px`, `font-weight: 700`
- [ ] No `border-radius` > 6px (except pills)
- [ ] No decorative animations
- [ ] Every label is specific (no "Overview")
- [ ] Answers "What is broken?" and "Where do I act?"
- [ ] Guards for `IS_BROWSER` on Wails service calls
- [ ] Matching CSS file created in `src/styles/`
