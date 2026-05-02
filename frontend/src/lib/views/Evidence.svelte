<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import {
    forensicsList,
    forensicsGaps,
    auditLog,
    auditVerify,
    type EvidenceItem,
    type LogGap,
    type AuditEntry,
    type VerifyResult,
  } from '../bridge';
  import Tile from '../components/Tile.svelte';
  import { copy } from '../clipboard';

  let items = $state<EvidenceItem[]>([]);
  let gaps = $state<LogGap[]>([]);
  let audit = $state<AuditEntry[]>([]);
  let verify = $state<VerifyResult | null>(null);
  let loadError = $state<string | null>(null);
  let timer: ReturnType<typeof setInterval> | null = null;
  let inFlight = 0;

  async function refresh() {
    const seq = ++inFlight;
    try {
      const [i, g, a, v] = await Promise.all([
        forensicsList(),
        forensicsGaps(),
        auditLog(50),
        auditVerify(),
      ]);
      if (seq !== inFlight) return;
      items = i;
      gaps = g;
      audit = a;
      verify = v;
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
</script>

<div class="mx-auto max-w-7xl space-y-6 p-8">
  <header>
    <p class="text-xs uppercase tracking-widest text-night-300">Evidence</p>
    <h2 class="text-2xl font-semibold tracking-tight">Audit chain · Sealed evidence · Log gaps</h2>
  </header>

  <section class="grid grid-cols-2 gap-4 lg:grid-cols-4">
    <Tile label="Audit entries" value={verify?.entries ?? 0} hint={verify?.ok ? 'chain intact' : 'BROKEN'} />
    <Tile label="Sealed evidence" value={items.length} />
    <Tile label="Log gaps detected" value={gaps.length} />
    <button
      type="button"
      onclick={() => verify?.rootHash && copy(verify.rootHash, 'Root hash copied')}
      class="text-left transition hover:opacity-80"
      title="Click to copy full root hash"
      disabled={!verify?.rootHash}
    >
      <Tile
        label="Root hash"
        value={verify?.rootHash ? verify.rootHash.slice(0, 8) + '…' : '—'}
        hint="sha256 + hmac · click to copy"
      />
    </button>
  </section>

  {#if loadError}
    <p class="text-xs text-signal-error">Failed to load: {loadError}</p>
  {/if}

  <section class="rounded-xl border border-night-700 bg-night-900/70">
    <div class="border-b border-night-700 px-4 py-3 text-sm font-semibold tracking-wide text-slate-100">
      Merkle audit chain — {verify?.ok ? 'verified ✓' : 'broken at #' + (verify?.brokenAt ?? '?')}
    </div>
    {#if audit.length === 0}
      <div class="px-4 py-12 text-center text-sm text-night-300">Empty.</div>
    {:else}
      <ul class="divide-y divide-night-700/70 font-mono text-xs">
        {#each audit as e (e.seq)}
          <li class="flex items-start gap-3 px-4 py-2">
            <span class="w-12 text-night-300">#{e.seq}</span>
            <span class="text-night-300">{new Date(e.timestamp).toLocaleTimeString()}</span>
            <span class="text-night-200">{e.actor}</span>
            <span class="text-night-400">·</span>
            <span class="text-slate-100">{e.action}</span>
            <button
              type="button"
              onclick={() => copy(e.hash, 'Hash copied')}
              class="ml-auto truncate rounded px-1 text-night-300 transition hover:bg-night-700/60 hover:text-white"
              title="Click to copy full hash"
            >
              {e.hash.slice(0, 12)}…
            </button>
          </li>
        {/each}
      </ul>
    {/if}
  </section>

  <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
    <section class="rounded-xl border border-night-700 bg-night-900/70">
      <div class="border-b border-night-700 px-4 py-3 text-sm font-semibold tracking-wide text-slate-100">
        Sealed evidence packages
      </div>
      {#if items.length === 0}
        <div class="px-4 py-12 text-center text-sm text-night-300">No sealed packages.</div>
      {:else}
        <ul class="divide-y divide-night-700/70 font-mono text-xs">
          {#each items as it (it.id)}
            <li class="flex flex-col gap-1 px-4 py-2">
              <div class="flex items-center gap-2">
                <span class="text-slate-100">{it.title}</span>
                <span class="text-night-300">· {it.hostId}</span>
                <span class="ml-auto text-night-300">{new Date(it.sealedAt).toLocaleString()}</span>
              </div>
              <div class="text-night-300">
                {it.eventIds.length} events · sha256 {it.hash.slice(0, 12)}…
              </div>
            </li>
          {/each}
        </ul>
      {/if}
    </section>

    <section class="rounded-xl border border-night-700 bg-night-900/70">
      <div class="border-b border-night-700 px-4 py-3 text-sm font-semibold tracking-wide text-slate-100">
        Log gaps (telemetry interruptions)
      </div>
      {#if gaps.length === 0}
        <div class="px-4 py-12 text-center text-sm text-night-300">No gaps detected.</div>
      {:else}
        <ul class="divide-y divide-night-700/70 font-mono text-xs">
          {#each gaps as g}
            <li class="flex items-center gap-3 px-4 py-2">
              <span class="text-slate-100">{g.hostId}</span>
              <span class="text-night-300">silent {g.duration}</span>
              <span class="ml-auto text-night-300">{new Date(g.endedAt).toLocaleTimeString()}</span>
            </li>
          {/each}
        </ul>
      {/if}
    </section>
  </div>
</div>
