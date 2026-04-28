<!--
  OBLIVRA — Delta Tile (Phase 32, "what's different from yesterday").

  Drop-in component for the top-right of every dashboard. Compares
  today's snapshot to yesterday's and surfaces the deltas in two
  seconds of glance. The most-requested incumbent feature.

  Snapshot mechanism:
    • On mount, capture (alerts.length, agents.length, criticals,
      compliance breaches, …) and stamp them with today's date.
    • Look up yesterday's snapshot from localStorage.
    • Compute deltas; render arrows + counts.

  This is intentionally LOCALSTORAGE-only for v1: no backend snapshot
  store needed. Cross-device sync is a Phase 33 follow-up.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { alertStore } from '@lib/stores/alerts.svelte';
  import { agentStore } from '@lib/stores/agent.svelte';
  import { ArrowUp, ArrowDown, Minus } from 'lucide-svelte';

  const SNAP_KEY = 'oblivra:dailySnapshots';
  const KEEP_DAYS = 7;

  type Snapshot = {
    date: string;             // YYYY-MM-DD
    alerts: number;
    critical: number;
    agents: number;
    online: number;
  };

  let yesterday = $state<Snapshot | null>(null);

  function todayKey(): string {
    const d = new Date();
    return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
  }

  function yesterdayKey(): string {
    const d = new Date();
    d.setDate(d.getDate() - 1);
    return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
  }

  function captureCurrent(): Snapshot {
    return {
      date: todayKey(),
      alerts: alertStore.alerts.length,
      critical: alertStore.alerts.filter((a) => a.severity?.toLowerCase() === 'critical').length,
      agents: agentStore.agents.length,
      online: agentStore.agents.filter((a) => a.status === 'online' || a.status === 'healthy').length,
    };
  }

  function loadSnapshots(): Record<string, Snapshot> {
    if (typeof localStorage === 'undefined') return {};
    try {
      const raw = localStorage.getItem(SNAP_KEY);
      if (!raw) return {};
      return JSON.parse(raw) ?? {};
    } catch { return {}; }
  }

  function persistSnapshots(snaps: Record<string, Snapshot>) {
    if (typeof localStorage === 'undefined') return;
    // Trim to most recent KEEP_DAYS entries.
    const keys = Object.keys(snaps).sort().slice(-KEEP_DAYS);
    const trimmed: Record<string, Snapshot> = {};
    for (const k of keys) trimmed[k] = snaps[k];
    try { localStorage.setItem(SNAP_KEY, JSON.stringify(trimmed)); } catch { /* quota */ }
  }

  onMount(() => {
    const snaps = loadSnapshots();
    yesterday = snaps[yesterdayKey()] ?? null;
    // Persist today's snapshot the first time the tile mounts each day.
    if (!snaps[todayKey()]) {
      snaps[todayKey()] = captureCurrent();
      persistSnapshots(snaps);
    } else {
      // Refresh today's snapshot so the comparison reflects the latest
      // data — but only if our values are higher (alerts grow over a
      // day; rolling them backward would be misleading).
      const cur = captureCurrent();
      const prev = snaps[todayKey()];
      snaps[todayKey()] = {
        ...prev,
        alerts:   Math.max(prev.alerts, cur.alerts),
        critical: Math.max(prev.critical, cur.critical),
        agents:   cur.agents,        // count, can shrink legitimately
        online:   cur.online,
      };
      persistSnapshots(snaps);
    }
  });

  // Live deltas — re-derive whenever the underlying stores tick.
  let deltas = $derived.by(() => {
    if (!yesterday) return null;
    const cur = captureCurrent();
    return {
      alerts:   cur.alerts - yesterday.alerts,
      critical: cur.critical - yesterday.critical,
      agents:   cur.agents - yesterday.agents,
      online:   cur.online - yesterday.online,
    };
  });

  function arrow(d: number) {
    if (d > 0) return ArrowUp;
    if (d < 0) return ArrowDown;
    return Minus;
  }

  /**
   * Polarity per metric — up = bad for alerts/critical (more = worse);
   * up = good for online (more = better); agents count is neutral.
   */
  function colour(metric: 'alerts' | 'critical' | 'agents' | 'online', d: number): string {
    if (d === 0) return 'text-text-muted';
    const upBad = metric === 'alerts' || metric === 'critical';
    const upGood = metric === 'online';
    if (upBad)  return d > 0 ? 'text-error'   : 'text-success';
    if (upGood) return d > 0 ? 'text-success' : 'text-error';
    return 'text-text-muted'; // neutral
  }
</script>

<div class="bg-surface-2 border border-border-primary rounded-md p-3 flex flex-col gap-2 min-w-[200px]">
  <header class="flex items-center justify-between">
    <span class="text-[var(--fs-micro)] font-bold uppercase tracking-widest text-text-muted">vs yesterday</span>
    {#if !yesterday}
      <span class="text-[var(--fs-micro)] font-mono text-text-muted opacity-60">no baseline</span>
    {/if}
  </header>

  {#if !deltas}
    <p class="text-[var(--fs-label)] text-text-muted leading-relaxed">First-day baseline — deltas appear tomorrow once the snapshot stack catches up.</p>
  {:else}
    <ul class="grid grid-cols-2 gap-x-2 gap-y-1.5">
      {#each [
        { key: 'alerts',   label: 'Alerts',    d: deltas.alerts },
        { key: 'critical', label: 'Critical',  d: deltas.critical },
        { key: 'online',   label: 'Online',    d: deltas.online },
        { key: 'agents',   label: 'Agents',    d: deltas.agents },
      ] as row}
        {@const Arrow = arrow(row.d)}
        {@const klass = colour(row.key as any, row.d)}
        <li class="flex items-center gap-1.5">
          <Arrow size={10} class={klass} />
          <span class="font-mono text-[var(--fs-label)] {klass}">{row.d > 0 ? '+' : ''}{row.d}</span>
          <span class="text-[var(--fs-micro)] text-text-muted ml-auto">{row.label}</span>
        </li>
      {/each}
    </ul>
  {/if}
</div>
