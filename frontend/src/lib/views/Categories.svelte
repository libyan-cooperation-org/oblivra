<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { categoriesList, type CategoryStat } from '../bridge';
  import Tile from '../components/Tile.svelte';

  let cats = $state<CategoryStat[]>([]);
  let loadError = $state<string | null>(null);
  let timer: ReturnType<typeof setInterval> | null = null;
  let inFlight = 0;
  let filter = $state('');

  async function refresh() {
    const seq = ++inFlight;
    try {
      const all = await categoriesList();
      if (seq !== inFlight) return;
      cats = all;
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
  onDestroy(() => {
    if (timer) clearInterval(timer);
  });

  const filtered = $derived(
    filter ? cats.filter((c) => c.sourceType.toLowerCase().includes(filter.toLowerCase())) : cats,
  );
  const totalEvents = $derived(cats.reduce((s, c) => s + c.count, 0));
  const topMax = $derived(filtered.length > 0 ? filtered[0].count : 1);

  function relTime(ts: string): string {
    if (!ts) return '—';
    const sec = Math.floor((Date.now() - new Date(ts).getTime()) / 1000);
    if (sec < 60) return `${sec}s ago`;
    if (sec < 3600) return `${Math.floor(sec / 60)}m ago`;
    if (sec < 86400) return `${Math.floor(sec / 3600)}h ago`;
    return `${Math.floor(sec / 86400)}d ago`;
  }
</script>

<div class="mx-auto max-w-7xl space-y-6 p-8">
  <header class="flex items-baseline justify-between">
    <div>
      <p class="text-xs uppercase tracking-widest text-night-300">Categories</p>
      <h2 class="text-2xl font-semibold tracking-tight">Log breakdown by sourceType</h2>
    </div>
    <input
      bind:value={filter}
      placeholder="filter…"
      class="w-48 rounded-md border border-night-600 bg-night-800/70 px-3 py-1.5 text-xs text-slate-100 placeholder:text-night-300 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500"
    />
  </header>

  <section class="grid grid-cols-2 gap-4 lg:grid-cols-4">
    <Tile label="Categories observed" value={cats.length} />
    <Tile label="Total events" value={totalEvents.toLocaleString()} hint="lifetime" />
    <Tile label="Top category" value={cats[0]?.sourceType ?? '—'} hint={cats[0] ? `${cats[0].count.toLocaleString()} events` : 'none yet'} />
    <Tile
      label="Most-recent"
      value={cats.length > 0 ? [...cats].sort((a, b) => new Date(b.lastSeen).getTime() - new Date(a.lastSeen).getTime())[0].sourceType : '—'}
      hint={cats.length > 0 ? relTime([...cats].sort((a, b) => new Date(b.lastSeen).getTime() - new Date(a.lastSeen).getTime())[0].lastSeen) : 'idle'}
    />
  </section>

  {#if loadError}
    <p class="text-xs text-signal-error">Failed to load: {loadError}</p>
  {/if}

  <section class="rounded-xl border border-night-700 bg-night-900/70">
    <div class="border-b border-night-700 px-4 py-3 text-sm font-semibold tracking-wide text-slate-100">
      Categories ({filtered.length})
    </div>
    {#if filtered.length === 0}
      <div class="px-4 py-12 text-center text-sm text-night-300">
        {cats.length === 0 ? 'No events ingested yet.' : 'Filter matched nothing.'}
      </div>
    {:else}
      <ul class="divide-y divide-night-700/70 font-mono text-xs">
        {#each filtered as c (c.sourceType)}
          <li class="px-4 py-3">
            <div class="flex items-center gap-3">
              <span class="w-48 truncate text-slate-100">{c.sourceType}</span>
              <div class="relative h-2 flex-1 overflow-hidden rounded bg-night-700">
                <div class="h-full rounded bg-accent-500"
                     style:width={`${Math.min(100, (c.count / Math.max(1, topMax)) * 100)}%`}></div>
              </div>
              <span class="w-24 text-right text-slate-100">{c.count.toLocaleString()}</span>
              <span class="w-20 text-right text-night-300">{relTime(c.lastSeen)}</span>
            </div>
            {#if c.topHosts && c.topHosts.length > 0}
              <div class="mt-1.5 flex flex-wrap gap-1.5 pl-48">
                {#each c.topHosts as h}
                  <span class="rounded-full border border-night-700 bg-night-800/70 px-2 py-0.5 text-[10px] text-night-300">
                    {h.host} · {h.count.toLocaleString()}
                  </span>
                {/each}
              </div>
            {/if}
          </li>
        {/each}
      </ul>
    {/if}
  </section>
</div>
