---
description: Generate a new OBLIVRA page — follow these steps to create a consistent SOC page
---

# Generate OBLIVRA Page

## Prerequisites

// turbo-all

1. **Read the design system first**:
   Load `.agents/workflows/design-system.md` for full token reference, CSS classes, and rules.

2. **Understand the user's page requirements**:
   - What is this page for? (What problem does it solve for a SOC analyst?)
   - What data does it display?
   - What actions can the user take?
   - Does it need backend (Wails) service calls?

## Steps

### Step 1: Create the CSS file

Create `frontend/src/styles/{page-name}.css` with styles specific to this page.

Rules:
- Use ONLY tokens from `variables.css`
- Use existing `.ob-*` classes where possible
- Only add new classes for truly unique patterns
- Follow the naming pattern: `.{page-name}-{element}`

### Step 2: Create the Component

Create the page component at either:
- `frontend/src/pages/PageName.tsx` — for standalone pages
- `frontend/src/components/{domain}/PageName.tsx` — for domain-grouped components

Follow this structure:
```tsx
import { Component, createSignal, onMount, Show, For } from 'solid-js';
import { useApp } from '@core/store';
import { IS_BROWSER } from '@core/context';
import '../../styles/{page-name}.css';

export const PageName: Component = () => {
    const [state] = useApp();
    const [loading, setLoading] = createSignal(true);
    const [error, setError] = createSignal<string | null>(null);

    return (
        <div class="ob-page">
            {/* ob-page-header, then dense content */}
        </div>
    );
};
```

### Step 3: Register the Route

Add to `frontend/src/index.tsx`:
```tsx
import { PageName } from './pages/PageName';
// ...
<Route path="/page-name" component={PageName} />
```

### Step 4: Add to CommandRail

Add to `frontend/src/components/layout/CommandRail.tsx`:
1. Add route to `routeMap`
2. Add nav item to the appropriate section (observe/operate/intel/govern/system)
3. Add lazy panel import to `panelImports`

### Step 5: Verify

- Check: Does every label answer "What is broken?" or "Where do I act?"
- Check: Is all data in `font-mono`? Are section headers `uppercase 10px 700`?
- Check: No hardcoded colors? No border-radius > 6px?
- Check: `IS_BROWSER` guard on all Wails calls?
- Check: Using existing `.ob-*` CSS classes?

## Design Patterns by Page Type

### Data Table Page (SIEM Search, Fleet, Logs)
```
┌─ Page Header (title + filters/actions) ─────────────┐
├─ Toolbar (search bar + filter pills) ────────────────┤
├─ Dense Table (.ob-table) ────────────────────────────┤
│  th: timestamp | hostname | severity | message       │
│  td: rows with severity color strips                 │
│  onclick: expand detail panel (right side)           │
├──────────────────────────────────────────────────────┤
│ Status: "Showing 1,247 of 89,312 events"            │
└──────────────────────────────────────────────────────┘
```

### Dashboard Page (Alerts, Executive, Health)
```
┌─ Page Header ────────────────────────────────────────┐
├─ KPI Grid (.ob-stat-grid-4) ─────────────────────────┤
│  [Critical: 3] [High: 12] [Resolved: 847] [MTTD: 4m]│
├─ Tab Bar (.ob-tabs) ─────────────────────────────────┤
│  Active | Investigating | Resolved                    │
├─ Content Grid ───────────────────────────────────────┤
│  [Alert Cards with severity strips]                   │
│  [Each card: title, host, timestamp, actions]         │
└──────────────────────────────────────────────────────┘
```

### Detail/Inspector Page (Entity View, Decision Log)
```
┌─ Page Header + Breadcrumb ───────────────────────────┐
├──────────────────┬───────────────────────────────────┤
│ Sidebar List     │ Detail Panel                       │
│ (scrollable)     │ ┌─ Header (entity name + status) ─┤
│                  │ ├─ Metadata Grid ─────────────────┤
│ [item] ◄ active  │ ├─ Related Items Table ───────────┤
│ [item]           │ ├─ Action Bar ────────────────────┤
│ [item]           │ └─────────────────────────────────┤
└──────────────────┴───────────────────────────────────┘
```

### Form/Config Page (Settings, Vault)
```
┌─ Page Header ────────────────────────────────────────┐
├─ Section Header (UPPERCASE) ─────────────────────────┤
├─ Form Group (.ob-form) ──────────────────────────────┤
│  Label + Input rows                                   │
├─ Section Header (UPPERCASE) ─────────────────────────┤
├─ Form Group ─────────────────────────────────────────┤
├─ Action Bar (bottom, sticky) ────────────────────────┤
│  [Cancel] [Save Changes]                              │
└──────────────────────────────────────────────────────┘
```
