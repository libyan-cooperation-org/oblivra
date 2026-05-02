<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import {
    savedSearchesList, savedSearchesSave, savedSearchesRun, savedSearchesDelete,
    type SavedSearch,
  } from '../bridge';
  import Tile from '../components/Tile.svelte';

  let saved = $state<SavedSearch[]>([]);
  let loadError = $state<string | null>(null);
  let busy = $state(false);
  let timer: ReturnType<typeof setInterval> | null = null;
  let inFlight = 0;

  // Add-form state.
  let name = $state('');
  let query = $state('');
  let queryKind = $state<'bleve' | 'oql'>('bleve');
  let interval = $state(0);
  let alertAt = $state(0);
  let severity = $state<'low' | 'medium' | 'high' | 'critical'>('high');
  let formError = $state<string | null>(null);
  let runResult = $state<Record<string, string>>({});

  async function refresh() {
    const seq = ++inFlight;
    try {
      const list = await savedSearchesList();
      if (seq !== inFlight) return;
      saved = list;
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

  async function add() {
    busy = true;
    formError = null;
    try {
      await savedSearchesSave({
        name, query, queryKind,
        intervalMinutes: interval || undefined,
        alertOnAtLeast: alertAt || undefined,
        severity: alertAt > 0 ? severity : undefined,
      });
      name = ''; query = ''; interval = 0; alertAt = 0;
      await refresh();
    } catch (err) {
      formError = (err as Error).message;
    } finally {
      busy = false;
    }
  }

  async function run(id: string) {
    runResult = { ...runResult, [id]: 'running…' };
    try {
      const r = await savedSearchesRun(id);
      runResult = { ...runResult, [id]: `${r.hits} hit${r.hits === 1 ? '' : 's'}` };
      await refresh();
    } catch (err) {
      runResult = { ...runResult, [id]: 'error: ' + (err as Error).message };
    }
    setTimeout(() => {
      runResult = Object.fromEntries(Object.entries(runResult).filter(([k]) => k !== id));
    }, 8000);
  }

  async function remove(id: string) {
    if (!confirm('Delete this saved search?')) return;
    try {
      await savedSearchesDelete(id);
      await refresh();
    } catch (err) {
      loadError = (err as Error).message;
    }
  }

  function relTime(ts?: string): string {
    if (!ts) return 'never';
    const sec = Math.floor((Date.now() - new Date(ts).getTime()) / 1000);
    if (sec < 60) return `${sec}s ago`;
    if (sec < 3600) return `${Math.floor(sec / 60)}m ago`;
    if (sec < 86400) return `${Math.floor(sec / 3600)}h ago`;
    return `${Math.floor(sec / 86400)}d ago`;
  }
</script>

<div class="mx-auto max-w-7xl space-y-6 p-8">
  <header>
    <p class="text-xs uppercase tracking-widest text-night-300">Saved searches</p>
    <h2 class="text-2xl font-semibold tracking-tight">Reusable queries · Scheduled checks</h2>
  </header>

  <section class="grid grid-cols-2 gap-4 lg:grid-cols-4">
    <Tile label="Saved" value={saved.length} />
    <Tile label="Scheduled" value={saved.filter((s) => (s.intervalMinutes ?? 0) > 0).length} hint="auto-run on interval" />
    <Tile label="With alerting" value={saved.filter((s) => (s.alertOnAtLeast ?? 0) > 0).length} hint="raise alert if hit ≥ N" />
    <Tile label="Recent runs" value={saved.filter((s) => s.lastRunAt).length} />
  </section>

  {#if loadError}<p class="text-xs text-signal-error">Failed to load: {loadError}</p>{/if}

  <!-- Add-form -->
  <section class="rounded-xl border border-night-700 bg-night-900/70 p-4">
    <h3 class="mb-3 text-sm font-semibold tracking-wide text-slate-100">New saved search</h3>
    <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
      <label class="text-xs">
        <div class="text-night-300 mb-1">Name</div>
        <input bind:value={name} class="w-full rounded-md border border-night-600 bg-night-800/70 px-3 py-1.5 text-xs text-slate-100" />
      </label>
      <label class="text-xs">
        <div class="text-night-300 mb-1">Query kind</div>
        <select bind:value={queryKind} class="w-full rounded-md border border-night-600 bg-night-800/70 px-2 py-1.5 text-xs text-slate-100">
          <option value="bleve">Bleve</option>
          <option value="oql">OQL</option>
        </select>
      </label>
      <label class="text-xs lg:col-span-3">
        <div class="text-night-300 mb-1">Query</div>
        <input bind:value={query} placeholder='severity:error AND host:web-01' class="w-full font-mono rounded-md border border-night-600 bg-night-800/70 px-3 py-1.5 text-xs text-slate-100" />
      </label>
      <label class="text-xs">
        <div class="text-night-300 mb-1">Interval (min, ≥5; 0 = manual)</div>
        <input type="number" min="0" bind:value={interval} class="w-full rounded-md border border-night-600 bg-night-800/70 px-3 py-1.5 text-xs text-slate-100" />
      </label>
      <label class="text-xs">
        <div class="text-night-300 mb-1">Alert when hits ≥</div>
        <input type="number" min="0" bind:value={alertAt} class="w-full rounded-md border border-night-600 bg-night-800/70 px-3 py-1.5 text-xs text-slate-100" />
      </label>
      <label class="text-xs">
        <div class="text-night-300 mb-1">Alert severity</div>
        <select bind:value={severity} disabled={alertAt <= 0} class="w-full rounded-md border border-night-600 bg-night-800/70 px-2 py-1.5 text-xs text-slate-100">
          <option value="low">low</option>
          <option value="medium">medium</option>
          <option value="high">high</option>
          <option value="critical">critical</option>
        </select>
      </label>
    </div>
    {#if formError}<p class="mt-3 text-xs text-signal-error">{formError}</p>{/if}
    <div class="mt-4 flex items-center gap-3">
      <button class="rounded-md bg-accent-500 px-3 py-1.5 text-xs font-medium text-white shadow-sm hover:bg-accent-600 disabled:opacity-50"
              disabled={busy || !name || !query} onclick={add}>Save</button>
      <span class="text-[11px] text-night-300">Scheduled queries run via the platform scheduler. Bleve query syntax: <code class="rounded bg-night-800 px-1">field:value</code></span>
    </div>
  </section>

  <section class="rounded-xl border border-night-700 bg-night-900/70">
    <div class="border-b border-night-700 px-4 py-3 text-sm font-semibold tracking-wide text-slate-100">Saved</div>
    {#if saved.length === 0}
      <div class="px-4 py-8 text-center text-sm text-night-300">No saved searches yet.</div>
    {:else}
      <ul class="divide-y divide-night-700/70 font-mono text-xs">
        {#each saved as s (s.id)}
          <li class="flex flex-col gap-1.5 px-4 py-3">
            <div class="flex items-center gap-2">
              <span class="rounded bg-night-800 px-1.5 py-0.5 text-[10px] uppercase text-night-200">{s.queryKind ?? 'bleve'}</span>
              <span class="text-slate-100">{s.name}</span>
              {#if (s.intervalMinutes ?? 0) > 0}
                <span class="text-night-300">· every {s.intervalMinutes}m</span>
              {/if}
              {#if (s.alertOnAtLeast ?? 0) > 0}
                <span class="text-night-300">· alert ≥ {s.alertOnAtLeast} ({s.severity ?? 'high'})</span>
              {/if}
              <span class="ml-auto text-night-300">last: {relTime(s.lastRunAt)}{s.lastHitCount !== undefined ? ` · ${s.lastHitCount} hits` : ''}</span>
              <button class="rounded-md border border-night-600 bg-night-800/70 px-2 py-0.5 text-[10px] text-slate-100 hover:bg-night-700" onclick={() => run(s.id)}>run</button>
              <button class="rounded-md border border-signal-error/50 px-2 py-0.5 text-[10px] text-signal-error hover:bg-night-700" onclick={() => remove(s.id)}>delete</button>
            </div>
            <div class="text-night-300 truncate">{s.query}</div>
            {#if s.lastError}
              <div class="text-signal-error">last error: {s.lastError}</div>
            {/if}
            {#if runResult[s.id]}
              <div class="rounded bg-night-800/60 px-2 py-1 text-[11px] text-slate-100">{runResult[s.id]}</div>
            {/if}
          </li>
        {/each}
      </ul>
    {/if}
  </section>
</div>
