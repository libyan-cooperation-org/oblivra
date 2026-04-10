---
description: Generate a new OBLIVRA page — follow these steps to create a consistent SOC page using Svelte 5 and Tailwind CSS v4
---

# Generate OBLIVRA Page (Svelte 5)

## Prerequisites

1.  **Read the design system first**:
    Load `.agents/workflows/design-system.md` for full token reference and rules.

2.  **Understand the requirements**:
    What data is being visualized? What actions are being taken? What Wails services are required?

---

## Steps

### Step 1: Create the Component
Create the page component at `frontend/src/pages/PageName.svelte`.

Use the standard Svelte 5 boiler-plate with runes:
```svelte
<script lang="ts">
  import { KPI, Badge, DataTable, PageLayout, Button } from '@components/ui';
  import { Shield, Zap, Activity } from 'lucide-svelte';
  import { appStore } from '@lib/stores/app.svelte';

  // State using runes
  let loading = $state(true);
  let error = $state<string | null>(null);

  // Derived state
  const isHardened = $derived(appStore.systemStatus === 'hardened');
</script>

<PageLayout title="Page Title" subtitle="High-density description of the tactical module">
  {#snippet toolbar()}
    <Button variant="primary" size="sm">Primary Action</Button>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <!-- Pulse Stats -->
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 shrink-0">
      <KPI title="Metric" value="0.0" trend="Nominal" />
    </div>

    <!-- Main Content Grid -->
    <div class="flex-1 min-h-0 bg-surface-1 border border-border-primary rounded-md overflow-hidden">
      <!-- Content here (DataTable, Charts, etc.) -->
    </div>
  </div>
</PageLayout>
```

### Step 2: Register the Route
Add to `frontend/src/App.svelte` in the `routes` array:
```typescript
import PageName from '@pages/PageName.svelte';

const routes = [
  // ...
  { path: '/page-name', component: PageName },
];
```

### Step 3: Link in Sidebar/Navigation
Update `frontend/src/components/shell/Sidebar.svelte` to include the new navigation item in the appropriate tactical section.

---

## Design Rules

1.  **High Density**: Use `compact` variants for DataTables. Avoid excessive padding.
2.  **Monochrome-Accent**: Use `bg-surface-1/2/3`, `border-border-primary/secondary`, and `text-text-heading/secondary/muted`. Use `accent` or `error` sparingly for emphasis.
3.  **Typography**: Use `font-mono` for all technical data, IDs, and timestamps. Use `uppercase tracking-widest` for section headers.
4.  **No Placeholders**: Never use generic placeholders. Use `lucide-svelte` icons and generate high-fidelity mock data if backend bindings are pending.
5.  **Reactivity**: Use Svelte 5 runes (`$state`, `$derived`, `$effect`) exclusively.

---

## Component Checklist

- [ ] Uses `PageLayout` with title and subtitle.
- [ ] Includes `toolbar` snippet for primary actions.
- [ ] Uses `KPI` components for top-level metrics.
- [ ] Uses `DataTable` for list-based data.
- [ ] Technical data is in `font-mono`.
- [ ] Adheres to the monochrome-accent color palette.
- [ ] Includes `IS_BROWSER` guards for Wails bridge calls.
