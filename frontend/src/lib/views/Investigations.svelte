<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { listAlerts, uebaProfiles, type Alert, type UebaProfile } from '../bridge';
  import Tile from '../components/Tile.svelte';

  let alerts = $state<Alert[]>([]);
  let profiles = $state<UebaProfile[]>([]);
  let loadError = $state<string | null>(null);
  let timer: ReturnType<typeof setInterval> | null = null;
  let inFlight = 0;

  async function refresh() {
    const seq = ++inFlight;
    try {
      const [a, p] = await Promise.all([listAlerts(100), uebaProfiles()]);
      if (seq !== inFlight) return; // a newer request already won
      alerts = a;
      profiles = p;
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
  onDestroy(() => {
    if (timer) clearInterval(timer);
  });

  // Group alerts by host so analysts can pick a target.
  const byHost = $derived.by(() => {
    const map = new Map<string, Alert[]>();
    for (const a of alerts) {
      const k = a.hostId || 'unknown';
      if (!map.has(k)) map.set(k, []);
      map.get(k)!.push(a);
    }
    return Array.from(map.entries()).map(([host, list]) => ({
      host,
      list,
      severity: list.some((a) => a.severity === 'critical')
        ? 'critical'
        : list.some((a) => a.severity === 'high')
        ? 'high'
        : list.some((a) => a.severity === 'medium')
        ? 'medium'
        : 'low',
    }));
  });

  let selectedHost = $state<string | null>(null);
  $effect(() => {
    if (!selectedHost && byHost.length > 0) {
      selectedHost = byHost[0].host;
    }
  });

  const selectedAlerts = $derived(byHost.find((h) => h.host === selectedHost)?.list ?? []);
  const selectedProfile = $derived(profiles.find((p) => p.entity === selectedHost) ?? null);

  // Top-spike tile: only meaningful when a real anomaly fired. Without a
  // threshold the tile would name a host every time UEBA had any profile,
  // implying an anomaly that didn't happen. z >= 2 matches our internal
  // alerting threshold; below that we show '—'.
  const topSpike = $derived.by(() => {
    const sorted = [...profiles].sort((a, b) => b.lastSpike - a.lastSpike);
    const top = sorted[0];
    if (!top || top.lastSpike < 2) return null;
    return top;
  });

  const sev: Record<string, string> = {
    low: 'bg-night-700 text-night-200',
    medium: 'bg-signal-info/20 text-signal-info',
    high: 'bg-signal-warn/20 text-signal-warn',
    critical: 'bg-signal-error/20 text-signal-error',
  };
</script>

<div class="mx-auto max-w-7xl space-y-6 p-8">
  <header>
    <p class="text-xs uppercase tracking-widest text-night-300">Investigations</p>
    <h2 class="text-2xl font-semibold tracking-tight">Per-host triage</h2>
  </header>

  <section class="grid grid-cols-2 gap-4 lg:grid-cols-4">
    <Tile label="Hosts under fire" value={byHost.length} />
    <Tile label="Open alerts" value={alerts.filter((a) => a.state === 'open').length} />
    <Tile label="UEBA profiles" value={profiles.length} />
    <Tile
      label="Top spike (z-score)"
      value={topSpike ? topSpike.lastSpike.toFixed(2) : '—'}
      hint={topSpike ? topSpike.entity : 'no spikes ≥ 2σ'}
    />
  </section>

  {#if loadError}
    <p class="text-xs text-signal-error">Failed to load: {loadError}</p>
  {/if}

  <div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
    <section class="rounded-xl border border-night-700 bg-night-900/70">
      <div class="border-b border-night-700 px-4 py-3 text-sm font-semibold tracking-wide text-slate-100">
        Hosts
      </div>
      {#if byHost.length === 0}
        <div class="px-4 py-12 text-center text-sm text-night-300">No alerted hosts.</div>
      {:else}
        <ul class="divide-y divide-night-700/70">
          {#each byHost as h}
            <li>
              <button
                type="button"
                class="flex w-full items-center justify-between px-4 py-2 text-left hover:bg-night-700/40"
                class:bg-night-700={selectedHost === h.host}
                onclick={() => (selectedHost = h.host)}
              >
                <span class="font-mono text-xs text-slate-100">{h.host}</span>
                <span class="rounded px-1.5 py-0.5 text-[10px] {sev[h.severity]}">
                  {h.list.length} · {h.severity}
                </span>
              </button>
            </li>
          {/each}
        </ul>
      {/if}
    </section>

    <section class="rounded-xl border border-night-700 bg-night-900/70 lg:col-span-2">
      <div class="border-b border-night-700 px-4 py-3 text-sm font-semibold tracking-wide text-slate-100">
        Detail · {selectedHost ?? '—'}
      </div>
      {#if !selectedHost}
        <div class="px-4 py-12 text-center text-sm text-night-300">Pick a host to investigate.</div>
      {:else}
        <div class="grid grid-cols-2 gap-4 px-4 py-3 text-xs">
          {#if selectedProfile}
            <div>
              <div class="text-night-300">Mean EPM</div>
              <div class="font-mono text-slate-100">{selectedProfile.mean.toFixed(2)}</div>
            </div>
            <div>
              <div class="text-night-300">σ</div>
              <div class="font-mono text-slate-100">{selectedProfile.stdDev.toFixed(2)}</div>
            </div>
            <div>
              <div class="text-night-300">Last EPM</div>
              <div class="font-mono text-slate-100">{selectedProfile.lastEpm}</div>
            </div>
            <div>
              <div class="text-night-300">Last spike (z)</div>
              <div class="font-mono text-slate-100">{selectedProfile.lastSpike.toFixed(2)}</div>
            </div>
          {:else}
            <div class="col-span-2 text-night-300">No UEBA profile yet.</div>
          {/if}
        </div>
        <ul class="divide-y divide-night-700/70 font-mono text-xs">
          {#each selectedAlerts as a (a.id)}
            <li class="flex items-start gap-3 px-4 py-2">
              <span class="text-night-300">{new Date(a.triggered).toLocaleTimeString()}</span>
              <span class="rounded px-1.5 py-0.5 text-[10px] {sev[a.severity]}">{a.severity}</span>
              <span class="text-slate-100">{a.ruleName}</span>
              <span class="ml-auto truncate text-night-300 max-w-[42ch]">{a.message}</span>
            </li>
          {/each}
        </ul>
      {/if}
    </section>
  </div>
</div>
