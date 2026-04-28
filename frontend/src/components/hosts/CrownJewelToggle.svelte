<!--
  OBLIVRA — Crown-Jewel toggle (Phase 32 follow-up).

  Compact pill that operators click on a HostDetail / FleetDashboard
  row to mark the host as crown-jewel (tier-1, business-critical).
  Drives the NBA recommender's `IsFromCrownJewel` fact-flag — alerts
  on tagged hosts will:
    • bias recommendations away from autonomous quarantine
    • bias toward evidence-capture + escalation

  Storage is localStorage-backed (see crownJewels.svelte.ts). Cross-
  device / cross-tenant persistence is a Phase 33 follow-up.
-->
<script lang="ts">
  import { crownJewels } from '@lib/stores/crownJewels.svelte';
  import { Crown } from 'lucide-svelte';

  interface Props {
    hostId: string;
    /** When true, render only the icon (for dense table cells). */
    compact?: boolean;
  }

  let { hostId, compact = false }: Props = $props();

  let active = $derived(crownJewels.has(hostId));
</script>

<button
  class="inline-flex items-center gap-1 rounded border transition-colors duration-fast {active
    ? 'bg-warning/10 border-warning/40 text-warning hover:bg-warning/20'
    : 'border-border-primary text-text-muted hover:border-border-hover hover:text-text-secondary'} {compact
    ? 'px-1 py-0.5'
    : 'px-2 py-0.5 text-[var(--fs-micro)]'}"
  onclick={(e) => { e.stopPropagation(); crownJewels.toggle(hostId); }}
  aria-pressed={active}
  title={active
    ? 'Crown-jewel asset — alerts bias toward evidence + escalation. Click to remove.'
    : 'Mark as crown-jewel (tier-1 asset). Recommender biases against autonomous quarantine.'}
>
  <Crown size={compact ? 10 : 11} class={active ? 'fill-warning/30' : ''} />
  {#if !compact}
    <span class="font-mono uppercase tracking-widest">
      {active ? 'crown-jewel' : 'mark crown-jewel'}
    </span>
  {/if}
</button>
