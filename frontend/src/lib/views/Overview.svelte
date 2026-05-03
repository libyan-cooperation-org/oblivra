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
    { label: 'Events ingested', value: stats?.total ?? 0,     hint: 'LIFETIME TOTAL' },
    { label: 'Hot store rows',  value: stats?.hotCount ?? 0,  hint: 'BADGERDB' },
    { label: 'Current EPS',     value: stats?.eps ?? 0,       hint: 'ROLLING 1s WINDOW' },
    { label: 'WAL',             value: stats ? `${(stats.wal.bytes / 1024).toFixed(1)} KiB` : '—', hint: `${stats?.wal.count ?? 0} ENTRIES` },
  ]);
</script>

<div class="p-6 space-y-6" style="max-width: 1280px; margin: 0 auto;">

  <!-- Page header -->
  <section>
    <div class="flex items-center gap-3 mb-3">
      <div style="width:3px; height:24px; background:var(--color-cyan-500); box-shadow:0 0 8px var(--color-cyan-500);"></div>
      <div>
        <p style="font-family:'Share Tech Mono',monospace; font-size:9px; letter-spacing:3px; color:var(--color-cyan-500); text-transform:uppercase; margin-bottom:2px;">Platform Status</p>
        <h2 style="font-family:'Rajdhani',sans-serif; font-weight:700; font-size:22px; letter-spacing:2px; text-transform:uppercase; color:#e8f4f8;">Sovereign Log Platform — Phase 0</h2>
      </div>
    </div>
    <p style="font-family:'Share Tech Mono',monospace; font-size:11px; line-height:1.7; color:var(--color-base-200); max-width:680px; letter-spacing:0.3px;">
      Skeleton spine is live. Storage tiers, ingest pipeline, detection engine, and forensic
      timeline will land in the next phases. The Wails desktop shell and the headless web
      dashboard share the exact same UI source.
    </p>
  </section>

  <!-- Metric tiles -->
  <section class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
    {#each tiles as tile}
      <Tile label={tile.label} value={tile.value} hint={tile.hint} />
    {/each}
  </section>

  <!-- Runtime info -->
  <section style="border:1px solid var(--color-base-700); background:linear-gradient(135deg, rgba(11,16,23,0.95), rgba(14,20,30,0.95)); position:relative; overflow:hidden;">
    <!-- Top bar -->
    <div class="flex items-center justify-between px-5 py-3 border-b border-base-700">
      <div class="flex items-center gap-2">
        <span style="font-family:'Rajdhani',sans-serif; font-weight:600; font-size:13px; letter-spacing:2px; text-transform:uppercase; color:#e8f4f8;">Runtime</span>
      </div>
      <div class="flex items-center gap-2">
        <span class="animate-glow" style="display:inline-block; width:6px; height:6px; border-radius:50%; background:var(--color-sig-ok);"></span>
        <span style="font-family:'Share Tech Mono',monospace; font-size:9px; letter-spacing:2px; color:var(--color-sig-ok);">LIVE</span>
      </div>
    </div>

    {#if loadError}
      <div class="px-5 py-4" style="font-family:'Share Tech Mono',monospace; font-size:11px; color:var(--color-sig-error); letter-spacing:0.5px;">
        ✗ FAILED TO LOAD: {loadError}
      </div>
    {:else if info}
      <dl class="grid grid-cols-2 gap-px sm:grid-cols-3" style="font-size:12px;">
        {#each [
          { key: 'VERSION',    val: info.version },
          { key: 'GO',         val: info.goVersion },
          { key: 'OS / ARCH',  val: `${info.os} / ${info.arch}` },
          { key: 'CPUS',       val: String(info.numCpu) },
          { key: 'GOROUTINES', val: String(info.goroutines) },
          { key: 'STARTED',    val: info.startedAt },
        ] as row}
          <div class="px-5 py-3" style="border-right:1px solid var(--color-base-700); border-bottom:1px solid var(--color-base-700);">
            <dt style="font-family:'Share Tech Mono',monospace; font-size:9px; letter-spacing:2px; color:var(--color-base-300); margin-bottom:4px;">{row.key}</dt>
            <dd style="font-family:'JetBrains Mono',monospace; font-size:13px; color:var(--color-cyan-400); letter-spacing:0.5px;">{row.val}</dd>
          </div>
        {/each}
      </dl>
    {:else}
      <div class="px-5 py-6 flex items-center gap-2">
        <span class="animate-glow-cyan" style="display:inline-block; width:6px; height:6px; border-radius:50%; background:var(--color-cyan-500);"></span>
        <span style="font-family:'Share Tech Mono',monospace; font-size:11px; letter-spacing:1px; color:var(--color-base-300);">LOADING RUNTIME INFO…</span>
      </div>
    {/if}

    <!-- Background grid -->
    <div style="position:absolute; inset:0; pointer-events:none; opacity:0.02;
                background-image:linear-gradient(var(--color-base-200) 1px,transparent 1px),linear-gradient(90deg,var(--color-base-200) 1px,transparent 1px);
                background-size:24px 24px;"></div>
  </section>

</div>
