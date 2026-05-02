<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { serviceHealthList, type ServiceHealthRow } from '../bridge';
  import Tile from '../components/Tile.svelte';

  let rows = $state<ServiceHealthRow[]>([]);
  let loadError = $state<string | null>(null);
  let timer: ReturnType<typeof setInterval> | null = null;
  let inFlight = 0;
  let filter = $state('');

  async function refresh() {
    const seq = ++inFlight;
    try {
      const list = await serviceHealthList();
      if (seq !== inFlight) return;
      rows = list;
      loadError = null;
    } catch (err) {
      if (seq !== inFlight) return;
      loadError = (err as Error).message;
    }
  }

  onMount(() => {
    void refresh();
    timer = setInterval(() => void refresh(), 4000);
  });
  onDestroy(() => { if (timer) clearInterval(timer); });

  const filtered = $derived(
    filter ? rows.filter((r) => r.sourceType.toLowerCase().includes(filter.toLowerCase())) : rows,
  );

  const counts = $derived.by(() => {
    const c = { healthy: 0, degraded: 0, silent: 0, unknown: 0 };
    for (const r of rows) {
      const k = r.status as keyof typeof c;
      if (k in c) c[k]++;
    }
    return c;
  });

  const statusColor: Record<string, string> = {
    healthy: 'bg-signal-success/20 text-signal-success',
    degraded: 'bg-signal-warn/20 text-signal-warn',
    silent: 'bg-signal-error/20 text-signal-error',
    unknown: 'bg-night-700 text-night-300',
  };

  function relTime(ts: string): string {
    if (!ts) return '—';
    const sec = Math.floor((Date.now() - new Date(ts).getTime()) / 1000);
    if (sec < 0) return 'just now';
    if (sec < 60) return `${sec}s ago`;
    if (sec < 3600) return `${Math.floor(sec / 60)}m ago`;
    if (sec < 86400) return `${Math.floor(sec / 3600)}h ago`;
    return `${Math.floor(sec / 86400)}d ago`;
  }

  function fmtPct(n: number): string {
    return `${(n * 100).toFixed(n < 0.01 ? 2 : 1)}%`;
  }
</script>

<div class="mx-auto max-w-7xl space-y-6 p-8">
  <header class="flex items-baseline justify-between">
    <div>
      <p class="text-xs uppercase tracking-widest text-night-300">Service health</p>
      <h2 class="text-2xl font-semibold tracking-tight">One row per logical service</h2>
    </div>
    <input
      bind:value={filter}
      placeholder="filter by sourceType…"
      class="w-56 rounded-md border border-night-600 bg-night-800/70 px-3 py-1.5 text-xs text-slate-100 placeholder:text-night-300 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500"
    />
  </header>

  <section class="grid grid-cols-2 gap-4 lg:grid-cols-4">
    <Tile label="Healthy" value={counts.healthy} hint="seen in last 5m, low unparsed" />
    <Tile label="Degraded" value={counts.degraded} hint=">5m or unparsed >20% or 5+ gaps" />
    <Tile label="Silent" value={counts.silent} hint=">30m without an event" />
    <Tile label="Total services" value={rows.length} hint="distinct sourceTypes" />
  </section>

  {#if loadError}
    <p class="text-xs text-signal-error">Failed to load: {loadError}</p>
  {/if}

  <section class="rounded-xl border border-night-700 bg-night-900/70">
    <div class="border-b border-night-700 px-4 py-3 text-sm font-semibold tracking-wide text-slate-100">
      Services ({filtered.length})
    </div>
    {#if filtered.length === 0}
      <div class="px-4 py-12 text-center text-sm text-night-300">
        {rows.length === 0 ? 'No events ingested yet.' : 'Filter matched nothing.'}
      </div>
    {:else}
      <ul class="divide-y divide-night-700/70 font-mono text-xs">
        {#each filtered as r (r.sourceType)}
          <li class="px-4 py-3">
            <div class="flex items-start gap-3">
              <span class="rounded px-1.5 py-0.5 text-[10px] {statusColor[r.status] ?? 'bg-night-700'}">{r.status}</span>
              <span class="w-56 truncate text-slate-100">{r.sourceType}</span>
              <span class="text-night-300">{r.hosts} host{r.hosts === 1 ? '' : 's'}</span>
              <span class="text-night-400">·</span>
              <span class="text-night-300">{r.events24h.toLocaleString()} ev lifetime</span>
              <span class="ml-auto text-night-300">{relTime(r.lastSeen)}</span>
            </div>
            <div class="mt-1 flex flex-wrap gap-x-4 gap-y-1 text-[11px] text-night-400 pl-2">
              <span>unparsed: <span class={r.unparsedRate > 0.20 ? 'text-signal-warn' : 'text-night-300'}>{fmtPct(r.unparsedRate)}</span></span>
              <span>gaps observed: <span class={r.gapsObserved > 5 ? 'text-signal-warn' : 'text-night-300'}>{r.gapsObserved}</span></span>
              <span>avg delay: <span class="text-night-300">{r.avgDelayMs}ms</span></span>
            </div>
            {#if r.topHosts && r.topHosts.length > 0}
              <div class="mt-1.5 flex flex-wrap gap-1.5 pl-2">
                {#each r.topHosts as h}
                  <span class="rounded-full border border-night-700 bg-night-800/70 px-2 py-0.5 text-[10px] text-night-300">{h}</span>
                {/each}
              </div>
            {/if}
          </li>
        {/each}
      </ul>
    {/if}
  </section>
</div>
