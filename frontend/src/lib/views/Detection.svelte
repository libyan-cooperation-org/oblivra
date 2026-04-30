<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import {
    listAlerts,
    listRules,
    mitreHeatmap,
    reloadRules,
    type Alert,
    type Rule,
  } from '../bridge';
  import Tile from '../components/Tile.svelte';

  let alerts = $state<Alert[]>([]);
  let rules = $state<Rule[]>([]);
  let heatmap = $state<{ technique: string; count: number }[]>([]);
  let busy = $state(false);
  let error = $state<string | null>(null);
  let timer: ReturnType<typeof setInterval> | null = null;

  async function refresh() {
    try {
      const [a, r, h] = await Promise.all([listAlerts(50), listRules(), mitreHeatmap()]);
      alerts = a;
      rules = r;
      heatmap = h.sort((x, y) => y.count - x.count);
      error = null;
    } catch (e) {
      error = (e as Error).message;
    }
  }

  async function reload() {
    busy = true;
    try {
      await reloadRules();
      await refresh();
    } catch (e) {
      error = (e as Error).message;
    } finally {
      busy = false;
    }
  }

  onMount(() => {
    void refresh();
    timer = setInterval(() => void refresh(), 3000);
  });
  onDestroy(() => {
    if (timer) clearInterval(timer);
  });

  const sev: Record<string, string> = {
    low: 'bg-night-700 text-night-200',
    medium: 'bg-signal-info/20 text-signal-info',
    high: 'bg-signal-warn/20 text-signal-warn',
    critical: 'bg-signal-error/20 text-signal-error',
  };
</script>

<div class="mx-auto max-w-7xl space-y-6 p-8">
  <header class="flex items-baseline justify-between">
    <div>
      <p class="text-xs uppercase tracking-widest text-night-300">Detection</p>
      <h2 class="text-2xl font-semibold tracking-tight">Rules · Alerts · MITRE coverage</h2>
    </div>
    <button
      class="rounded-md border border-night-600 bg-night-800/70 px-3 py-1.5 text-xs text-slate-100 hover:bg-night-700 disabled:opacity-50"
      disabled={busy}
      onclick={reload}
    >
      Reload rules
    </button>
  </header>

  <section class="grid grid-cols-2 gap-4 lg:grid-cols-4">
    <Tile label="Active rules" value={rules.filter((r) => !r.disabled).length} hint={`${rules.length} total`} />
    <Tile label="Alerts (recent)" value={alerts.length} hint="last 50" />
    <Tile label="MITRE coverage" value={heatmap.length} hint="techniques observed" />
    <Tile
      label="Top technique"
      value={heatmap[0]?.technique ?? '—'}
      hint={heatmap[0] ? `${heatmap[0].count} hits` : 'no detections yet'}
    />
  </section>

  {#if error}
    <p class="text-xs text-signal-error">{error}</p>
  {/if}

  <section class="rounded-xl border border-night-700 bg-night-900/70">
    <div class="border-b border-night-700 px-4 py-3 text-sm font-semibold tracking-wide text-slate-100">
      Recent alerts
    </div>
    {#if alerts.length === 0}
      <div class="px-4 py-12 text-center text-sm text-night-300">No alerts yet.</div>
    {:else}
      <ul class="divide-y divide-night-700/70 font-mono text-xs">
        {#each alerts as a (a.id)}
          <li class="flex items-start gap-3 px-4 py-2 hover:bg-night-700/30">
            <span class="text-night-300">{new Date(a.triggered).toLocaleTimeString()}</span>
            <span class="rounded px-1.5 py-0.5 text-[10px] {sev[a.severity] ?? 'bg-night-700'}">
              {a.severity}
            </span>
            <span class="text-night-200">{a.hostId ?? '—'}</span>
            <span class="text-night-400">·</span>
            <span class="text-slate-100">{a.ruleName}</span>
            <span class="ml-auto text-night-300 truncate max-w-[40ch]">{a.message}</span>
          </li>
        {/each}
      </ul>
    {/if}
  </section>

  <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
    <section class="rounded-xl border border-night-700 bg-night-900/70">
      <div class="border-b border-night-700 px-4 py-3 text-sm font-semibold tracking-wide text-slate-100">
        Rules ({rules.length})
      </div>
      <ul class="divide-y divide-night-700/70 font-mono text-xs">
        {#each rules as r (r.id)}
          <li class="flex items-start gap-3 px-4 py-2">
            <span class="rounded px-1.5 py-0.5 text-[10px] {sev[r.severity] ?? 'bg-night-700'}">
              {r.severity}
            </span>
            <div class="flex-1">
              <div class="text-slate-100">{r.name}</div>
              <div class="text-night-300">
                {r.id} · MITRE: {r.mitre?.join(', ') ?? '—'} · source: {r.source ?? 'builtin'}
              </div>
            </div>
          </li>
        {/each}
      </ul>
    </section>

    <section class="rounded-xl border border-night-700 bg-night-900/70">
      <div class="border-b border-night-700 px-4 py-3 text-sm font-semibold tracking-wide text-slate-100">
        MITRE heatmap
      </div>
      {#if heatmap.length === 0}
        <div class="px-4 py-12 text-center text-sm text-night-300">No techniques observed yet.</div>
      {:else}
        <ul class="divide-y divide-night-700/70 font-mono text-xs">
          {#each heatmap as h}
            <li class="flex items-center gap-4 px-4 py-2">
              <span class="w-24 text-slate-100">{h.technique}</span>
              <div class="relative h-1.5 flex-1 overflow-hidden rounded bg-night-700">
                <div
                  class="h-full rounded bg-accent-500"
                  style:width={`${Math.min(100, (h.count / Math.max(1, heatmap[0].count)) * 100)}%`}
                ></div>
              </div>
              <span class="w-12 text-right text-night-300">{h.count}</span>
            </li>
          {/each}
        </ul>
      {/if}
    </section>
  </div>
</div>
