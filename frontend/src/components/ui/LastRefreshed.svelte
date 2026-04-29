<!--
  LastRefreshed — small inline indicator showing when the parent page
  last successfully fetched data. Drop into any page that polls a
  REST or Wails endpoint on an interval.

  Usage:
    <script>
      let lastSync = $state<Date | null>(null);
      async function refresh() {
        ...
        lastSync = new Date();
      }
    </script>
    <LastRefreshed time={lastSync} />

  Why this exists:
  Operators auditing a SOC dashboard need to know whether they're
  looking at fresh data or a stale cache. "Last refreshed at 14:02:18"
  is the difference between trusting a tile and ignoring it. Audit
  finding (UI/UX punch list, 2026-04-29).

  Renders a relative-time string ("12s ago", "2m ago", "stale 5m+") that
  ticks every second so the operator sees freshness drift in real time.
-->
<script lang="ts">
  import { onMount, onDestroy } from 'svelte';

  interface Props {
    /** When the page last successfully fetched data. null = never. */
    time: Date | null;
    /**
     * Threshold (seconds) above which the indicator turns warning.
     * Defaults to 60s — most SOC dashboards poll every 30s, so 60s
     * means "we missed at least one cycle."
     */
    staleThresholdSec?: number;
    /** Optional class hook for the wrapper. */
    class?: string;
  }

  let { time, staleThresholdSec = 60, class: cls = '' }: Props = $props();

  let now = $state(Date.now());
  let tick: ReturnType<typeof setInterval> | null = null;

  onMount(() => {
    tick = setInterval(() => { now = Date.now(); }, 1000);
  });
  onDestroy(() => {
    if (tick) clearInterval(tick);
  });

  const ageSec = $derived(time ? Math.max(0, Math.floor((now - time.getTime()) / 1000)) : -1);

  const label = $derived.by(() => {
    if (!time) return 'never refreshed';
    if (ageSec < 5) return 'just now';
    if (ageSec < 60) return `${ageSec}s ago`;
    if (ageSec < 3600) return `${Math.floor(ageSec / 60)}m ago`;
    return `${Math.floor(ageSec / 3600)}h ago`;
  });

  const isStale = $derived(ageSec > staleThresholdSec);
  const tooltip = $derived(
    time ? `Last refreshed: ${time.toLocaleTimeString()}` : 'No successful refresh yet',
  );
</script>

<span
  class="inline-flex items-center gap-1 text-[9px] font-mono uppercase tracking-wider {isStale ? 'text-warning' : 'text-text-muted'} {cls}"
  title={tooltip}
  aria-label={tooltip}
>
  <span
    class="w-1.5 h-1.5 rounded-full {isStale ? 'bg-warning animate-pulse' : 'bg-success/60'}"
    aria-hidden="true"
  ></span>
  <span>{label}</span>
</span>
