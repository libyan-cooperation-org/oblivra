<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { getSystemInfo, siemStats, type SystemInfo, type IngestStats } from '../bridge';
  import Tile from '../components/Tile.svelte';

  let info = $state<SystemInfo | null>(null);
  let stats = $state<IngestStats | null>(null);
  let loadError = $state<string | null>(null);
  let timer: ReturnType<typeof setInterval> | null = null;

  async function refresh() {
    try {
      const [i, s] = await Promise.all([getSystemInfo(), siemStats()]);
      info = i;
      stats = s;
      loadError = null;
    } catch (err) {
      loadError = (err as Error).message;
    }
  }

  onMount(() => {
    void refresh();
    timer = setInterval(() => void refresh(), 3000);
  });
  onDestroy(() => {
    if (timer) clearInterval(timer);
  });

  const tiles = $derived([
    { label: 'Events ingested', value: stats?.total ?? 0, hint: 'lifetime' },
    { label: 'Hot store rows',  value: stats?.hotCount ?? 0, hint: 'BadgerDB' },
    { label: 'Current EPS',     value: stats?.eps ?? 0, hint: 'rolling 1s window' },
    { label: 'WAL bytes',       value: stats ? `${(stats.wal.bytes / 1024).toFixed(1)} KiB` : '—', hint: `${stats?.wal.count ?? 0} entries` },
  ]);
</script>

<div class="mx-auto max-w-7xl space-y-8 p-8">
  <section class="space-y-2">
    <p class="text-xs uppercase tracking-widest text-night-300">Platform</p>
    <h2 class="text-2xl font-semibold tracking-tight">Sovereign Log Platform — Phase 0</h2>
    <p class="max-w-2xl text-sm text-night-200">
      Skeleton spine is live. Storage tiers, ingest pipeline, detection engine, and forensic
      timeline will land in the next phases. The Wails desktop shell and the headless web
      dashboard share the exact same UI source.
    </p>
  </section>

  <section class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
    {#each tiles as tile}
      <Tile label={tile.label} value={tile.value} hint={tile.hint} />
    {/each}
  </section>

  <section class="rounded-xl border border-night-700 bg-night-900/70 p-6">
    <div class="mb-4 flex items-center justify-between">
      <h3 class="text-sm font-semibold tracking-wide text-slate-100">Runtime</h3>
      <span class="text-[11px] uppercase tracking-widest text-night-300">live</span>
    </div>

    {#if loadError}
      <div class="text-sm text-signal-error">Failed to load: {loadError}</div>
    {:else if info}
      <dl class="grid grid-cols-2 gap-x-8 gap-y-3 text-sm sm:grid-cols-3">
        <div>
          <dt class="text-night-300">Version</dt>
          <dd class="font-mono text-slate-100">{info.version}</dd>
        </div>
        <div>
          <dt class="text-night-300">Go</dt>
          <dd class="font-mono text-slate-100">{info.goVersion}</dd>
        </div>
        <div>
          <dt class="text-night-300">OS / Arch</dt>
          <dd class="font-mono text-slate-100">{info.os} / {info.arch}</dd>
        </div>
        <div>
          <dt class="text-night-300">CPUs</dt>
          <dd class="font-mono text-slate-100">{info.numCpu}</dd>
        </div>
        <div>
          <dt class="text-night-300">Goroutines</dt>
          <dd class="font-mono text-slate-100">{info.goroutines}</dd>
        </div>
        <div>
          <dt class="text-night-300">Started</dt>
          <dd class="font-mono text-slate-100">{info.startedAt}</dd>
        </div>
      </dl>
    {:else}
      <div class="text-sm text-night-300">Loading runtime info…</div>
    {/if}
  </section>
</div>
