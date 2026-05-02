<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { fleetList, intelList, type Agent, type Indicator } from '../bridge';
  import Tile from '../components/Tile.svelte';

  let agents = $state<Agent[]>([]);
  let iocs = $state<Indicator[]>([]);
  let selectedId = $state<string | null>(null);
  let loadError = $state<string | null>(null);
  let timer: ReturnType<typeof setInterval> | null = null;
  let inFlight = 0;

  async function refresh() {
    const seq = ++inFlight;
    try {
      const [a, i] = await Promise.all([fleetList(), intelList()]);
      if (seq !== inFlight) return;
      agents = a;
      iocs = i;
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

  // Health classification for the per-agent status badge.
  function statusOf(a: Agent): { kind: 'healthy' | 'silent' | 'lagging' | 'dropping'; label: string } {
    const ageMs = Date.now() - new Date(a.lastSeen).getTime();
    if (ageMs > 15 * 60 * 1000) return { kind: 'silent', label: 'silent' };
    if ((a.droppedEvents ?? 0) > 0) return { kind: 'dropping', label: 'dropping' };
    if ((a.spillBytes ?? 0) > 50 * 1024 * 1024) return { kind: 'lagging', label: 'lagging' };
    if (ageMs > 5 * 60 * 1000) return { kind: 'lagging', label: 'stale' };
    return { kind: 'healthy', label: 'healthy' };
  }

  const statusClass: Record<string, string> = {
    healthy: 'bg-signal-success/20 text-signal-success',
    silent: 'bg-signal-error/20 text-signal-error',
    lagging: 'bg-signal-warn/20 text-signal-warn',
    dropping: 'bg-signal-error/20 text-signal-error',
  };

  function fmtBytes(n?: number): string {
    if (!n || n <= 0) return '0';
    const units = ['B', 'KiB', 'MiB', 'GiB'];
    let v = n;
    let i = 0;
    while (v >= 1024 && i < units.length - 1) { v /= 1024; i++; }
    return `${v.toFixed(v >= 100 ? 0 : 1)} ${units[i]}`;
  }

  function relTime(ts?: string): string {
    if (!ts) return '—';
    const d = new Date(ts);
    const sec = Math.floor((Date.now() - d.getTime()) / 1000);
    if (sec < 0) return 'in the future';
    if (sec < 60) return `${sec}s ago`;
    if (sec < 3600) return `${Math.floor(sec / 60)}m ago`;
    if (sec < 86400) return `${Math.floor(sec / 3600)}h ago`;
    return `${Math.floor(sec / 86400)}d ago`;
  }

  // Aggregations for the tile row.
  const aggs = $derived.by(() => {
    let healthy = 0, silent = 0, lagging = 0, dropping = 0;
    let spillTotal = 0;
    let droppedTotal = 0;
    const versions = new Map<string, number>();
    for (const a of agents) {
      const s = statusOf(a);
      if (s.kind === 'healthy') healthy++;
      else if (s.kind === 'silent') silent++;
      else if (s.kind === 'dropping') dropping++;
      else lagging++;
      spillTotal += a.spillBytes ?? 0;
      droppedTotal += a.droppedEvents ?? 0;
      const v = a.version || 'unknown';
      versions.set(v, (versions.get(v) ?? 0) + 1);
    }
    return { healthy, silent, lagging, dropping, spillTotal, droppedTotal, versions };
  });

  const selected = $derived(agents.find((a) => a.id === selectedId) ?? null);
  $effect(() => {
    if (!selectedId && agents.length > 0) selectedId = agents[0].id;
  });

  let copied = $state(false);
  async function copyPubkey() {
    if (!selected?.pubkeyB64) return;
    try {
      await navigator.clipboard.writeText(selected.pubkeyB64);
      copied = true;
      setTimeout(() => (copied = false), 1500);
    } catch {}
  }
</script>

<div class="mx-auto max-w-7xl space-y-6 p-8">
  <header class="flex items-baseline justify-between">
    <div>
      <p class="text-xs uppercase tracking-widest text-night-300">Fleet</p>
      <h2 class="text-2xl font-semibold tracking-tight">Agents · Health · Threat intel</h2>
    </div>
    <div class="text-[11px] text-night-300">
      {agents.length} agent{agents.length === 1 ? '' : 's'} · {iocs.length} indicator{iocs.length === 1 ? '' : 's'}
    </div>
  </header>

  <section class="grid grid-cols-2 gap-4 lg:grid-cols-5">
    <Tile label="Healthy" value={aggs.healthy} hint="seen in last 5m" />
    <Tile label="Lagging" value={aggs.lagging} hint=">5m or spill > 50 MiB" />
    <Tile label="Silent" value={aggs.silent} hint=">15m without check-in" />
    <Tile label="Spill backlog" value={fmtBytes(aggs.spillTotal)} hint="across all agents" />
    <Tile label="Dropped events" value={aggs.droppedTotal} hint="cumulative" />
  </section>

  {#if loadError}
    <p class="text-xs text-signal-error">Failed to load: {loadError}</p>
  {/if}

  <div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
    <!-- Agent list — clickable -->
    <section class="rounded-xl border border-night-700 bg-night-900/70">
      <div class="border-b border-night-700 px-4 py-3 text-sm font-semibold tracking-wide text-slate-100">
        Agents
      </div>
      {#if agents.length === 0}
        <div class="px-4 py-12 text-center text-sm text-night-300">
          No agents registered yet.<br />
          <span class="text-night-400">POST /api/v1/agent/register or run `oblivra-agent run`.</span>
        </div>
      {:else}
        <ul class="divide-y divide-night-700/70 font-mono text-xs max-h-[24rem] overflow-y-auto scrollbar-thin">
          {#each agents as a (a.id)}
            {@const s = statusOf(a)}
            <li>
              <button
                type="button"
                class="flex w-full items-start gap-2 px-4 py-2 text-left hover:bg-night-700/40"
                class:bg-night-700={selectedId === a.id}
                onclick={() => (selectedId = a.id)}
              >
                <span class="flex-1">
                  <div class="flex items-center gap-2">
                    <span class="truncate text-slate-100">{a.hostname}</span>
                    <span class="rounded px-1.5 py-0.5 text-[10px] {statusClass[s.kind]}">{s.label}</span>
                  </div>
                  <div class="mt-0.5 text-night-300">
                    {a.os}/{a.arch} · {a.events} events · {relTime(a.lastSeen)}
                  </div>
                </span>
              </button>
            </li>
          {/each}
        </ul>
      {/if}
    </section>

    <!-- Per-agent detail -->
    <section class="lg:col-span-2 rounded-xl border border-night-700 bg-night-900/70">
      <div class="border-b border-night-700 px-4 py-3 text-sm font-semibold tracking-wide text-slate-100">
        Detail · {selected?.hostname ?? '—'}
      </div>
      {#if !selected}
        <div class="px-4 py-12 text-center text-sm text-night-300">Pick an agent.</div>
      {:else}
        {@const s = statusOf(selected)}
        <div class="grid gap-4 p-4 text-xs sm:grid-cols-2 lg:grid-cols-3">
          <div>
            <div class="text-night-300">Status</div>
            <div class="mt-1"><span class="rounded px-1.5 py-0.5 text-[10px] {statusClass[s.kind]}">{s.label}</span></div>
          </div>
          <div>
            <div class="text-night-300">Last seen</div>
            <div class="font-mono text-slate-100">{relTime(selected.lastSeen)}</div>
          </div>
          <div>
            <div class="text-night-300">Registered</div>
            <div class="font-mono text-slate-100">{relTime(selected.registered)}</div>
          </div>
          <div>
            <div class="text-night-300">OS / Arch</div>
            <div class="font-mono text-slate-100">{selected.os ?? '—'} / {selected.arch ?? '—'}</div>
          </div>
          <div>
            <div class="text-night-300">Version</div>
            <div class="font-mono text-slate-100">{selected.version || '—'}</div>
          </div>
          <div>
            <div class="text-night-300">Inputs configured</div>
            <div class="font-mono text-slate-100">{selected.inputCount ?? '—'}</div>
          </div>
          <div>
            <div class="text-night-300">Events lifetime</div>
            <div class="font-mono text-slate-100">{selected.events.toLocaleString()}</div>
          </div>
          <div>
            <div class="text-night-300">Queue depth</div>
            <div class="font-mono text-slate-100">{selected.queueDepth ?? '—'}</div>
          </div>
          <div>
            <div class="text-night-300">Batch size</div>
            <div class="font-mono text-slate-100">{selected.batchSize ?? '—'}</div>
          </div>
          <div>
            <div class="text-night-300">Spill files</div>
            <div class="font-mono text-slate-100">{selected.spillFiles ?? 0}</div>
          </div>
          <div>
            <div class="text-night-300">Spill bytes</div>
            <div class="font-mono text-slate-100">{fmtBytes(selected.spillBytes)}</div>
          </div>
          <div>
            <div class="text-night-300">Dropped events</div>
            <div class="font-mono {(selected.droppedEvents ?? 0) > 0 ? 'text-signal-error' : 'text-slate-100'}">
              {selected.droppedEvents ?? 0}
            </div>
          </div>
        </div>

        <div class="border-t border-night-700 px-4 py-3 text-xs">
          <div class="text-night-300">Signing key (ed25519)</div>
          <div class="mt-1 flex items-center gap-2 font-mono text-slate-100">
            <span class="rounded bg-night-800 px-1.5 py-0.5 text-[10px]">
              fp: {selected.pubkeyFingerprint ?? '—'}
            </span>
            {#if selected.pubkeyB64}
              <button
                class="rounded-md border border-night-600 bg-night-800/70 px-2 py-1 text-[10px] text-slate-100 hover:bg-night-700"
                onclick={copyPubkey}
              >
                {copied ? 'copied!' : 'copy public key'}
              </button>
              <span class="text-night-400 truncate max-w-[28ch]">{selected.pubkeyB64.slice(0, 22)}…</span>
            {:else}
              <span class="text-night-400">(agent did not register a signing key)</span>
            {/if}
          </div>
          {#if selected.pubkeyB64}
            <p class="mt-2 text-[11px] text-night-400">
              Add to <code class="rounded bg-night-800 px-1">OBLIVRA_AGENT_PUBKEYS</code> on the server to verify
              this agent's per-event signatures.
            </p>
          {/if}
        </div>

        {#if selected.tags && selected.tags.length > 0}
          <div class="border-t border-night-700 px-4 py-3 text-xs">
            <div class="text-night-300 mb-1.5">Tags</div>
            <div class="flex flex-wrap gap-1.5">
              {#each selected.tags as t}
                <span class="rounded-full border border-night-600 bg-night-800/70 px-2 py-0.5 text-[10px] text-night-200">{t}</span>
              {/each}
            </div>
          </div>
        {/if}
      {/if}
    </section>
  </div>

  <!-- Threat-intel — separate concern, kept compact -->
  <section class="rounded-xl border border-night-700 bg-night-900/70">
    <div class="flex items-center justify-between border-b border-night-700 px-4 py-3 text-sm font-semibold tracking-wide text-slate-100">
      <span>Threat-intel indicators</span>
      <span class="text-[11px] text-night-300">{iocs.length} loaded</span>
    </div>
    {#if iocs.length === 0}
      <div class="px-4 py-8 text-center text-sm text-night-300">
        No indicators loaded. POST one to <code class="text-night-200">/api/v1/threatintel/indicator</code> or load a STIX feed.
      </div>
    {:else}
      <ul class="divide-y divide-night-700/70 font-mono text-xs max-h-64 overflow-y-auto scrollbar-thin">
        {#each iocs as i (i.value)}
          <li class="flex items-center gap-3 px-4 py-2">
            <span class="rounded border border-night-600 px-1.5 py-0.5 text-[10px] text-night-200">{i.type}</span>
            <span class="text-slate-100">{i.value}</span>
            <span class="text-night-300">· {i.source ?? 'manual'}</span>
            <span class="ml-auto text-night-300">{i.severity ?? '—'}</span>
          </li>
        {/each}
      </ul>
    {/if}
  </section>
</div>
