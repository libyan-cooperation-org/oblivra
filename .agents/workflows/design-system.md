---
description: OBLIVRA Master Design System — Master reference for Svelte 5 + Tailwind CSS v4 UI development
---

# OBLIVRA Design System — Master Reference

> **PURPOSE**: This is the single source of truth for all UI generation.
> Load this file FIRST before writing any component, page, or layout.

---

## 1. Platform Context

**OBLIVRA** is a SOC (Security Operations Center) platform — a Wails desktop app with a Svelte 5 frontend.

- **Framework**: Svelte 5 (Runes mode: `$state`, `$derived`, `$effect`)
- **Styling**: Tailwind CSS v4 (Theme-first utility architecture)
- **Routing**: `svelte-routing` (for SPA navigation)
- **State**: Custom Svelte 5 runes store via `@lib/stores/app.svelte.ts`
- **Icons**: `lucide-svelte` (24x24, stroke-based)
- **Backend**: Wails 3 (Go services → TypeScript bindings)
- **Build**: Vite 7+ with `@sveltejs/vite-plugin-svelte`

---

## 2. Aesthetics Table (Tailwind v4 tokens)

### Color Palette (Sovereign Monochrome-Accent)
OBLIVRA uses a high-density dark theme with specific semantic meanings.

| Token | Class | Rationale |
| :--- | :--- | :--- |
| **Background** | `bg-background` | Deep charcoal for reduced eye strain during 12h shifts. |
| **Surface 1** | `bg-surface-1` | Primary container background (panels, sidebars). |
| **Surface 2** | `bg-surface-2` | Elevated surface (headers, selected items). |
| **Accent** | `text-accent` / `bg-accent` | Core interaction color (Primary Blue). |
| **Success** | `text-success` / `bg-success` | Healthy state, resolved incidents. |
| **Error** | `text-error` / `bg-error` | ACTIVE ATTACK, broken services, critical alerts. |
| **Warning** | `text-warning` / `bg-warning` | Degraded state, medium-priority alerts. |

### Typography
- **Heading**: `font-sans` (Inter/Outfit) — For labels, UI chrome, and headlines.
- **Data**: `font-mono` (JetBrains Mono) — MANDATORY for all log data, IDs, hashes, IPs, and timestamps.
- **Micro**: `text-[10px] font-bold uppercase tracking-widest` — Standard for section headers and meta-labels.

---

## 3. UI Components (Standardized Library)

Always prefer these pre-built components located in `@components/ui/`:

1.  **KPI**: Metric cards for top-of-page status.
2.  **Badge**: Semantic status indicators.
3.  **DataTable**: High-density data grids with compact row height.
4.  **PageLayout**: Standard page wrapper with title, subtitle, and toolbar slot.
5.  **Button**: Standardized variants (`primary`, `secondary`, `error`, `ghost`).
6.  **Toggle / Input**: Controlled form elements.

---

## 4. UX Immutable Rules

1.  **DENSITY OVER DECORATION**: No wasted whitespace. SOC analysts need information density.
2.  **TWO-QUESTION TEST**: Every screen MUST answer:
    *   "What is broken?" (Severity/Status)
    *   "Where do I act?" (Primary actions/Expansion)
3.  **MONO FOR DATA**: Any value that is a machine-readable ID, IP, or Timestamp MUST be `font-mono`.
4.  **REACTION SPEED**: Every interaction must feel instant. Avoid transition delays > 150ms.
5.  **DARK MODE ONLY**: The "Sovereign" aesthetic does not support a light mode. Hardcode for low-light environments.

---

## 5. Implementation Pattern (Svelte 5)

```svelte
<script lang="ts">
  import { KPI, PageLayout, Button } from '@components/ui';
  import { Shield, Activity } from 'lucide-svelte';
  import { appStore } from '@lib/stores/app.svelte';

  // Use runes for all state
  let data = $state([]);
  let active = $state(false);

  // Derived state for automatic reactivity
  const totalCount = $derived(data.length);

  // Effect for side-effects
  $effect(() => {
    console.log('State changed:', active);
  });
</script>

<PageLayout title="Tactical Title" subtitle="High-density micro-copy">
  <div class="grid grid-cols-4 gap-4">
    <KPI title="Active" value={totalCount} variant="accent" />
  </div>
  <!-- Main content using Tailwind utility classes -->
</PageLayout>
```

---

## 6. Layout Constants

- **Sidebar Width**: `w-64`
- **Header Height**: `h-12`
- **Border Radius**: `rounded-md` (Standard 4-6px, keep it sharp)
- **Border Style**: `border border-border-primary`
