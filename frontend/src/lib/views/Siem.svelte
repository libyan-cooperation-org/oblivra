<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import {
    siemIngest,
    siemSearch,
    siemStats,
    liveTail,
    eventGet,
    siemSearchExportUrl,
    type OblivraEvent,
    type EventDetail,
    type IngestStats,
    type Severity,
    type SearchResponse,
    type LiveTailHandle,
  } from '../bridge';
  import Tile from '../components/Tile.svelte';

  let stats = $state<IngestStats | null>(null);
  let result = $state<SearchResponse | null>(null);
  let busy = $state(false);
  let error = $state<string | null>(null);

  // Event detail drill-through.
  let selected = $state<EventDetail | null>(null);
  let detailError = $state<string | null>(null);

  async function open(id: string) {
    detailError = null;
    selected = null;
    try {
      selected = await eventGet(id);
    } catch (err) {
      detailError = (err as Error).message;
    }
  }

  // Query state
  let query = $state('');
  let live = $state(true);
  let limit = $state(50);

  // Probe state
  let probeMessage = $state('Failed password for root from 10.0.0.42 port 22 ssh2');
  let probeSeverity = $state<Severity>('warning');

  let timer: ReturnType<typeof setInterval> | null = null;
  let tail: LiveTailHandle | null = null;
  let tailEvents = $state<OblivraEvent[]>([]);
  let tailDropped = $state(0);

  async function refresh() {
    try {
      const [s, q] = await Promise.all([
        siemStats(),
        siemSearch({ query, limit, newestFirst: true }),
      ]);
      stats = s;
      result = q;
      error = null;
    } catch (e) {
      error = (e as Error).message;
    }
  }

  function startTimer() {
    stopTimer();
    if (live) {
      timer = setInterval(() => void refresh(), 5000);
    }
  }
  function stopTimer() {
    if (timer) {
      clearInterval(timer);
      timer = null;
    }
  }

  function startTail() {
    stopTail();
    if (!live) return;
    tail = liveTail(
      (ev) => {
        tailEvents = [ev, ...tailEvents].slice(0, 200);
        if (tailEvents.length === 200) tailDropped++;
      },
      () => {
        // ignore — auto-reconnects
      },
    );
  }
  function stopTail() {
    if (tail) {
      tail.close();
      tail = null;
    }
  }

  $effect(() => {
    // re-evaluate timer + websocket when `live` changes
    startTimer();
    startTail();
  });

  async function probe() {
    busy = true;
    error = null;
    try {
      await siemIngest({
        source: 'rest',
        eventType: 'probe',
        severity: probeSeverity,
        hostId: 'phase1-probe',
        message: probeMessage,
      });
      await refresh();
    } catch (e) {
      error = (e as Error).message;
    } finally {
      busy = false;
    }
  }

  async function probeBurst() {
    busy = true;
    error = null;
    try {
      const targets = ['web-01', 'web-02', 'db-01', 'fw-edge', 'auth-svc'];
      const types: Severity[] = ['info', 'warning', 'error', 'notice'];
      const messages = [
        'sshd Failed password for root',
        'sshd Accepted publickey for admin',
        'kernel: nf_conntrack: table full',
        'auth.log new session opened for user analyst',
        'firewalld: dropped INPUT from 198.51.100.7',
      ];
      const promises = Array.from({ length: 25 }, (_, i) =>
        siemIngest({
          source: 'rest',
          eventType: 'burst',
          severity: types[i % types.length],
          hostId: targets[i % targets.length],
          message: messages[i % messages.length],
        }),
      );
      await Promise.all(promises);
      await refresh();
    } catch (e) {
      error = (e as Error).message;
    } finally {
      busy = false;
    }
  }

  onMount(() => {
    void refresh();
  });
  onDestroy(() => {
    stopTimer();
    stopTail();
  });

  function handleQueryKey(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      e.preventDefault();
      void refresh();
    }
  }

  function applyChip(value: string) {
    query = value;
    void refresh();
  }

  const sevColor: Record<Severity, string> = {
    debug: 'text-night-300',
    info: 'text-signal-info',
    notice: 'text-signal-info',
    warning: 'text-signal-warn',
    error: 'text-signal-error',
    critical: 'text-signal-error',
    alert: 'text-signal-error',
  };
</script>

<div class="mx-auto max-w-7xl space-y-6 p-8">
  <header class="flex items-baseline justify-between">
    <div>
      <p class="text-xs uppercase tracking-widest text-night-300">SIEM</p>
      <h2 class="text-2xl font-semibold tracking-tight">Live event stream</h2>
    </div>
    <div class="flex items-center gap-3 text-xs text-night-300">
      {#if result}
        <span class="rounded-md border border-night-600 px-2 py-0.5 font-mono">
          {result.mode}
        </span>
        <span>·</span>
        <span class="font-mono">{result.total} hits</span>
        <span>·</span>
        <span class="font-mono">{result.took}</span>
      {/if}
    </div>
  </header>

  <section class="grid grid-cols-2 gap-4 lg:grid-cols-4">
    <Tile label="Events ingested" value={stats?.total ?? '—'} hint="lifetime" />
    <Tile label="Hot store" value={stats?.hotCount ?? '—'} hint="BadgerDB rows" />
    <Tile label="EPS" value={stats?.eps ?? 0} hint="rolling 1s" />
    <Tile
      label="WAL"
      value={stats ? `${(stats.wal.bytes / 1024).toFixed(1)} KiB` : '—'}
      hint={`${stats?.wal.count ?? 0} lines`}
    />
  </section>

  <section class="rounded-xl border border-night-700 bg-night-900/70 p-4">
    <div class="flex flex-wrap items-center gap-2">
      <input
        bind:value={query}
        onkeydown={handleQueryKey}
        placeholder='search — e.g. severity:warning, message:sshd, hostId:web-01, "Failed password"'
        class="flex-1 min-w-[24ch] rounded-md border border-night-600 bg-night-800/70 px-3 py-1.5 font-mono text-xs text-slate-100 placeholder:text-night-300 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500"
      />
      <button
        class="rounded-md bg-accent-500 px-3 py-1.5 text-xs font-medium text-white shadow-sm hover:bg-accent-600"
        onclick={() => void refresh()}
      >
        Search
      </button>
      <button
        class="rounded-md border border-night-600 bg-night-800/70 px-3 py-1.5 text-xs text-slate-100 hover:bg-night-700"
        onclick={() => applyChip('')}
        title="Clear query"
      >
        Clear
      </button>
      <label class="ml-2 flex items-center gap-2 text-xs text-night-200">
        <input type="checkbox" bind:checked={live} class="accent-accent-500" />
        live
      </label>
      <select
        bind:value={limit}
        class="rounded-md border border-night-600 bg-night-800/70 px-2 py-1.5 text-xs text-slate-100 focus:border-accent-500 focus:outline-none"
      >
        <option value={25}>25</option>
        <option value={50}>50</option>
        <option value={100}>100</option>
        <option value={250}>250</option>
      </select>
    </div>

    <div class="mt-3 flex flex-wrap gap-2 text-[11px]">
      {#each [
        { label: 'errors', q: 'severity:error severity:critical' },
        { label: 'warnings', q: 'severity:warning' },
        { label: 'sshd', q: 'message:sshd' },
        { label: 'firewalld', q: 'message:firewalld' },
        { label: 'web tier', q: 'hostId:web-01 hostId:web-02' },
      ] as chip}
        <button
          class="rounded-full border border-night-600 bg-night-800/70 px-3 py-1 text-night-200 hover:bg-night-700"
          onclick={() => applyChip(chip.q)}
        >
          {chip.label}
        </button>
      {/each}
    </div>
  </section>

  <section class="rounded-xl border border-night-700 bg-night-900/70 p-4">
    <div class="mb-3 flex items-center justify-between">
      <h3 class="text-sm font-semibold tracking-wide text-slate-100">Ingest probe</h3>
      <span class="text-[11px] text-night-300">
        sends synthetic events through WAL → hot store → Bleve
      </span>
    </div>
    <div class="flex flex-wrap items-center gap-2">
      <input
        bind:value={probeMessage}
        class="flex-1 min-w-[24ch] rounded-md border border-night-600 bg-night-800/70 px-3 py-1.5 text-xs text-slate-100 placeholder:text-night-300 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500"
        placeholder="probe message"
      />
      <select
        bind:value={probeSeverity}
        class="rounded-md border border-night-600 bg-night-800/70 px-2 py-1.5 text-xs text-slate-100 focus:border-accent-500 focus:outline-none"
      >
        <option value="debug">debug</option>
        <option value="info">info</option>
        <option value="notice">notice</option>
        <option value="warning">warning</option>
        <option value="error">error</option>
        <option value="critical">critical</option>
      </select>
      <button
        class="rounded-md bg-accent-500 px-3 py-1.5 text-xs font-medium text-white shadow-sm hover:bg-accent-600 disabled:opacity-50"
        disabled={busy}
        onclick={probe}
      >
        Send 1
      </button>
      <button
        class="rounded-md border border-night-600 bg-night-800/70 px-3 py-1.5 text-xs text-slate-100 hover:bg-night-700 disabled:opacity-50"
        disabled={busy}
        onclick={probeBurst}
      >
        Burst x25
      </button>
    </div>
    {#if error}
      <p class="mt-2 text-xs text-signal-error">{error}</p>
    {/if}
  </section>

  {#if live && tailEvents.length > 0}
    <section class="rounded-xl border border-accent-500/40 bg-night-900/70">
      <div class="flex items-center justify-between border-b border-night-700 px-4 py-2">
        <h3 class="text-sm font-semibold tracking-wide text-slate-100">
          <span class="mr-1 inline-block h-2 w-2 rounded-full bg-accent-500 animate-pulse"></span>
          Live tail (WebSocket)
        </h3>
        <span class="text-[11px] text-night-300">
          {tailEvents.length} buffered{tailDropped ? ` · ${tailDropped} dropped` : ''}
        </span>
      </div>
      <ul class="divide-y divide-night-700/70 font-mono text-xs max-h-72 overflow-y-auto scrollbar-thin">
        {#each tailEvents.slice(0, 30) as ev (ev.id)}
          <li class="flex items-start gap-3 px-4 py-1.5">
            <span class="text-night-300">{new Date(ev.timestamp).toLocaleTimeString()}</span>
            <span class={sevColor[(ev.severity ?? 'info') as Severity]}>
              {(ev.severity ?? 'info').toUpperCase().padEnd(8)}
            </span>
            <span class="text-night-200">{ev.hostId ?? '—'}</span>
            <span class="flex-1 truncate text-slate-100">{ev.message}</span>
          </li>
        {/each}
      </ul>
    </section>
  {/if}

  <section class="rounded-xl border border-night-700 bg-night-900/70">
    <div class="flex items-center justify-between border-b border-night-700 px-4 py-3">
      <h3 class="text-sm font-semibold tracking-wide text-slate-100">Recent events</h3>
      <div class="flex items-center gap-2 text-[11px] text-night-300">
        <span>{result?.events.length ?? 0} shown · newest first</span>
        {#if (result?.events.length ?? 0) > 0}
          <a class="rounded-md border border-night-600 bg-night-800/70 px-2 py-0.5 text-slate-100 hover:bg-night-700"
             href={siemSearchExportUrl({ query, limit, newestFirst: true }, 'csv')} download>CSV</a>
          <a class="rounded-md border border-night-600 bg-night-800/70 px-2 py-0.5 text-slate-100 hover:bg-night-700"
             href={siemSearchExportUrl({ query, limit, newestFirst: true }, 'ndjson')} download>NDJSON</a>
        {/if}
      </div>
    </div>
    {#if !result || result.events.length === 0}
      <div class="px-4 py-12 text-center text-sm text-night-300">
        {query ? `No events matching "${query}"` : 'No events yet — fire the probe above.'}
      </div>
    {:else}
      <ul class="divide-y divide-night-700/70 font-mono text-xs">
        {#each result.events as ev (ev.id)}
          <li>
            <button
              type="button"
              class="flex w-full items-start gap-3 px-4 py-2 text-left hover:bg-night-700/30"
              onclick={() => open(ev.id)}
            >
              <span class="text-night-300">{new Date(ev.timestamp).toLocaleTimeString()}</span>
              <span class={sevColor[(ev.severity ?? 'info') as Severity]}>
                {(ev.severity ?? 'info').toUpperCase().padEnd(8)}
              </span>
              <span class="text-night-200">{ev.hostId ?? '—'}</span>
              <span class="text-night-400">·</span>
              <span class="text-night-300">{ev.eventType ?? ev.source}</span>
              <span class="flex-1 truncate text-slate-100">{ev.message}</span>
            </button>
          </li>
        {/each}
      </ul>
    {/if}
  </section>

  {#if selected || detailError}
    <section class="rounded-xl border border-accent-500/40 bg-night-900/70">
      <div class="flex items-center justify-between border-b border-night-700 px-4 py-3">
        <h3 class="text-sm font-semibold tracking-wide text-slate-100">Event detail</h3>
        <button class="rounded-md border border-night-600 bg-night-800/70 px-2 py-0.5 text-[11px] text-slate-100 hover:bg-night-700"
                onclick={() => { selected = null; detailError = null; }}>close</button>
      </div>
      {#if detailError}
        <div class="px-4 py-3 text-xs text-signal-error">{detailError}</div>
      {:else if selected}
        {@const ev = selected.event}
        <div class="grid gap-4 p-4 text-xs sm:grid-cols-2">
          <div><div class="text-night-300">ID</div><div class="font-mono text-slate-100 break-all">{ev.id}</div></div>
          <div><div class="text-night-300">Timestamp</div><div class="font-mono text-slate-100">{new Date(ev.timestamp).toISOString()}</div></div>
          <div><div class="text-night-300">Received</div><div class="font-mono text-slate-100">{new Date(ev.receivedAt).toISOString()}</div></div>
          <div><div class="text-night-300">Source</div><div class="font-mono text-slate-100">{ev.source}</div></div>
          <div><div class="text-night-300">Host</div><div class="font-mono text-slate-100">{ev.hostId ?? '—'}</div></div>
          <div><div class="text-night-300">Event type</div><div class="font-mono text-slate-100">{ev.eventType ?? '—'}</div></div>
          <div><div class="text-night-300">Severity</div><div class="font-mono text-slate-100">{ev.severity ?? '—'}</div></div>
          <div><div class="text-night-300">Tenant</div><div class="font-mono text-slate-100">{ev.tenantId}</div></div>
        </div>
        <div class="border-t border-night-700 px-4 py-3 text-xs">
          <div class="text-night-300 mb-1">Message</div>
          <div class="font-mono text-slate-100 whitespace-pre-wrap break-words">{ev.message}</div>
          {#if ev.raw && ev.raw !== ev.message}
            <div class="text-night-300 mb-1 mt-3">Raw</div>
            <div class="font-mono text-night-300 whitespace-pre-wrap break-words">{ev.raw}</div>
          {/if}
        </div>
        {#if ev.fields && Object.keys(ev.fields).length > 0}
          <div class="border-t border-night-700 px-4 py-3 text-xs">
            <div class="text-night-300 mb-1.5">Fields ({Object.keys(ev.fields).length})</div>
            <div class="grid gap-1 sm:grid-cols-2 lg:grid-cols-3">
              {#each Object.entries(ev.fields) as [k, v]}
                <div class="flex gap-2 truncate">
                  <span class="text-night-300">{k}:</span>
                  <span class="text-slate-100 truncate">{v}</span>
                </div>
              {/each}
            </div>
          </div>
        {/if}
        <div class="border-t border-night-700 px-4 py-3 text-xs">
          <div class="text-night-300 mb-1.5">
            Related — same host within ±60s ({selected.related.length})
          </div>
          {#if selected.related.length === 0}
            <div class="text-night-400">No nearby events on this host.</div>
          {:else}
            <ul class="divide-y divide-night-700/70 font-mono">
              {#each selected.related as r (r.id)}
                <li class="flex items-start gap-3 py-1">
                  <span class="text-night-300 w-20">{new Date(r.timestamp).toLocaleTimeString()}</span>
                  <span class="text-night-200 w-20 truncate">{r.eventType ?? r.source}</span>
                  <span class="flex-1 truncate text-slate-100">{r.message}</span>
                  <button class="rounded-md border border-night-600 bg-night-800/70 px-1.5 py-0.5 text-[10px] text-slate-100 hover:bg-night-700"
                          onclick={() => open(r.id)}>open</button>
                </li>
              {/each}
            </ul>
          {/if}
        </div>
      {/if}
    </section>
  {/if}
</div>
