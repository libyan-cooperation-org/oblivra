<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { mitreHeatmap, listAlerts, type Alert } from '../bridge';
  import { MITRE_TACTICS } from '../mitre';
  import Tile from '../components/Tile.svelte';

  let counts = $state<Record<string, number>>({});
  let alerts = $state<Alert[]>([]);
  let loadError = $state<string | null>(null);
  let timer: ReturnType<typeof setInterval> | null = null;
  let inFlight = 0;
  let hovered = $state<{ tactic: string; technique: string; name: string } | null>(null);

  async function refresh() {
    const seq = ++inFlight;
    try {
      const [h, a] = await Promise.all([mitreHeatmap(), listAlerts(500)]);
      if (seq !== inFlight) return;
      const map: Record<string, number> = {};
      for (const e of h) map[e.technique] = e.count;
      counts = map;
      alerts = a;
      loadError = null;
    } catch (err) {
      if (seq !== inFlight) return;
      loadError = (err as Error).message;
    }
  }

  onMount(() => {
    void refresh();
    timer = setInterval(() => void refresh(), 5000);
  });
  onDestroy(() => { if (timer) clearInterval(timer); });

  // Compute matrix-wide stats so the tile bar shows real coverage
  // numbers, not empty placeholders.
  const stats = $derived.by(() => {
    let totalTechniques = 0;
    let coveredTechniques = 0;
    let totalFires = 0;
    for (const t of MITRE_TACTICS) {
      for (const k of t.techniques) {
        totalTechniques++;
        const n = counts[k.id] ?? 0;
        if (n > 0) {
          coveredTechniques++;
          totalFires += n;
        }
      }
    }
    return { totalTechniques, coveredTechniques, totalFires };
  });

  const hoveredAlerts = $derived.by(() => {
    if (!hovered) return [] as Alert[];
    return alerts.filter((a) => (a.mitre ?? []).some((m) => m === hovered!.technique)).slice(0, 8);
  });

  // Bucket coverage into 5 visual intensities — same idea as a GitHub
  // contributions calendar so it scans at a glance.
  function intensityClass(n: number): string {
    if (n === 0) return 'bg-night-800 border-night-700';
    if (n < 3)   return 'bg-accent-500/15 border-accent-500/40';
    if (n < 10)  return 'bg-accent-500/35 border-accent-500/60';
    if (n < 50)  return 'bg-accent-500/60 border-accent-500/80';
    return            'bg-accent-500/90 border-accent-500';
  }
</script>

<div class="mx-auto max-w-7xl space-y-6 p-8">
  <header class="flex items-baseline justify-between">
    <div>
      <p class="text-xs uppercase tracking-widest text-night-300">MITRE ATT&amp;CK</p>
      <h2 class="text-2xl font-semibold tracking-tight">Detection coverage matrix</h2>
    </div>
    <div class="text-[11px] text-night-300">techniques in this view: {stats.totalTechniques}</div>
  </header>

  <section class="grid grid-cols-2 gap-4 lg:grid-cols-4">
    <Tile label="Tactics" value={MITRE_TACTICS.length} hint="enterprise matrix" />
    <Tile label="Covered techniques" value={stats.coveredTechniques} hint={`of ${stats.totalTechniques}`} />
    <Tile label="Coverage" value={`${Math.round((stats.coveredTechniques / Math.max(1, stats.totalTechniques)) * 100)}%`} />
    <Tile label="Total fires" value={stats.totalFires.toLocaleString()} hint="lifetime" />
  </section>

  {#if loadError}
    <p class="text-xs text-signal-error">Failed to load: {loadError}</p>
  {/if}

  <!-- Matrix legend -->
  <div class="flex items-center gap-3 text-[11px] text-night-300">
    <span>Coverage:</span>
    <span class="inline-block h-3 w-3 rounded border bg-night-800 border-night-700"></span> 0
    <span class="inline-block h-3 w-3 rounded border bg-accent-500/15 border-accent-500/40"></span> 1–2
    <span class="inline-block h-3 w-3 rounded border bg-accent-500/35 border-accent-500/60"></span> 3–9
    <span class="inline-block h-3 w-3 rounded border bg-accent-500/60 border-accent-500/80"></span> 10–49
    <span class="inline-block h-3 w-3 rounded border bg-accent-500/90 border-accent-500"></span> 50+
  </div>

  <!-- Tactic columns — horizontally scrollable on small screens -->
  <section class="overflow-x-auto rounded-xl border border-night-700 bg-night-900/70 p-4">
    <div class="grid gap-3" style:grid-template-columns="repeat({MITRE_TACTICS.length}, minmax(180px, 1fr))">
      {#each MITRE_TACTICS as tactic}
        <div>
          <div class="mb-2 flex items-center justify-between">
            <span class="text-[11px] font-semibold uppercase tracking-widest text-night-200">{tactic.name}</span>
            <span class="text-[10px] text-night-400">{tactic.id}</span>
          </div>
          <div class="space-y-1.5">
            {#each tactic.techniques as t}
              {@const n = counts[t.id] ?? 0}
              <button
                type="button"
                class="block w-full rounded border px-2 py-1.5 text-left text-[11px] transition {intensityClass(n)}"
                onmouseenter={() => (hovered = { tactic: tactic.name, technique: t.id, name: t.name })}
                onmouseleave={() => (hovered = null)}
                title="{t.id} · {t.name}"
              >
                <div class="flex items-baseline justify-between gap-2">
                  <span class="font-mono text-[10px] text-night-100">{t.id}</span>
                  <span class="text-[10px] text-night-100 font-semibold">{n}</span>
                </div>
                <div class="truncate text-night-100">{t.name}</div>
              </button>
            {/each}
          </div>
        </div>
      {/each}
    </div>
  </section>

  <!-- Hover detail — shows alerts for the technique under the cursor -->
  {#if hovered}
    <section class="rounded-xl border border-accent-500/40 bg-night-900/70 p-4">
      <div class="flex items-baseline justify-between">
        <div>
          <p class="text-xs uppercase tracking-widest text-night-300">{hovered.tactic}</p>
          <h3 class="text-sm font-semibold text-slate-100">
            <span class="font-mono text-night-200">{hovered.technique}</span>
            <span class="ml-2">{hovered.name}</span>
          </h3>
        </div>
        <span class="text-[11px] text-night-300">recent alerts: {hoveredAlerts.length}</span>
      </div>
      {#if hoveredAlerts.length === 0}
        <p class="mt-3 text-xs text-night-400">No alerts have referenced this technique yet.</p>
      {:else}
        <ul class="mt-3 divide-y divide-night-700/70 font-mono text-[11px]">
          {#each hoveredAlerts as a (a.id)}
            <li class="flex items-start gap-2 py-1.5">
              <span class="text-night-400">{new Date(a.triggered).toLocaleTimeString()}</span>
              <span class="text-slate-100">{a.ruleName}</span>
              <span class="text-night-400">·</span>
              <span class="text-night-300">{a.hostId ?? '—'}</span>
              <span class="ml-auto truncate text-night-300 max-w-[40ch]">{a.message}</span>
            </li>
          {/each}
        </ul>
      {/if}
    </section>
  {/if}
</div>
