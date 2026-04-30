<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import {
    trustSummary, qualitySources, tamperFindings,
    type TrustSummary, type SourceProfile, type TamperFinding,
  } from '../bridge';
  import Tile from '../components/Tile.svelte';

  let trust = $state<TrustSummary | null>(null);
  let sources = $state<SourceProfile[]>([]);
  let tamper = $state<TamperFinding[]>([]);
  let timer: ReturnType<typeof setInterval> | null = null;

  async function refresh() {
    try {
      const [t, s, m] = await Promise.all([trustSummary(), qualitySources(), tamperFindings()]);
      trust = t; sources = s; tamper = m;
    } catch {}
  }
  onMount(() => { void refresh(); timer = setInterval(() => void refresh(), 5000); });
  onDestroy(() => { if (timer) clearInterval(timer); });
</script>

<div class="mx-auto max-w-7xl space-y-6 p-8">
  <header>
    <p class="text-xs uppercase tracking-widest text-night-300">Trust &amp; Quality</p>
    <h2 class="text-2xl font-semibold tracking-tight">Event provenance · Source reliability · Tamper signals</h2>
  </header>

  <section class="grid grid-cols-2 gap-4 lg:grid-cols-4">
    <Tile label="Verified" value={trust?.verified ?? 0} hint="agent-signed / mTLS" />
    <Tile label="Consistent" value={trust?.consistent ?? 0} hint="multi-source corroborated" />
    <Tile label="Suspicious" value={trust?.suspicious ?? 0} hint="anomaly attached" />
    <Tile label="Untrusted" value={trust?.untrusted ?? 0} hint="single anonymous source" />
  </section>

  <section class="rounded-xl border border-night-700 bg-night-900/70">
    <div class="border-b border-night-700 px-4 py-3 text-sm font-semibold tracking-wide">Source reliability (worst-first)</div>
    {#if sources.length === 0}
      <div class="px-4 py-8 text-center text-sm text-night-300">No sources observed yet.</div>
    {:else}
      <table class="w-full font-mono text-xs">
        <thead class="text-night-300"><tr>
          <th class="px-4 py-2 text-left">Host</th><th class="px-4 py-2 text-left">Source</th><th class="px-4 py-2 text-right">Total</th><th class="px-4 py-2 text-right">Unparsed</th><th class="px-4 py-2 text-right">Avg delay</th><th class="px-4 py-2 text-right">Gaps</th>
        </tr></thead>
        <tbody class="divide-y divide-night-700/70">
          {#each sources as s}
            <tr>
              <td class="px-4 py-1 text-slate-100">{s.host}</td>
              <td class="px-4 py-1 text-night-200">{s.source}</td>
              <td class="px-4 py-1 text-right">{s.total}</td>
              <td class="px-4 py-1 text-right" class:text-signal-warn={s.unparsedRate > 0.1}>{(s.unparsedRate * 100).toFixed(1)}%</td>
              <td class="px-4 py-1 text-right">{s.avgDelayMs}ms</td>
              <td class="px-4 py-1 text-right" class:text-signal-error={s.gapsObserved > 0}>{s.gapsObserved}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </section>

  <section class="rounded-xl border border-signal-error/40 bg-night-900/70">
    <div class="border-b border-night-700 px-4 py-3 text-sm font-semibold tracking-wide">
      Tamper findings (auditd disable / journal truncate / clock rollback)
    </div>
    {#if tamper.length === 0}
      <div class="px-4 py-8 text-center text-sm text-night-300">No tamper signals — chain looks clean.</div>
    {:else}
      <ul class="divide-y divide-night-700/70 font-mono text-xs">
        {#each tamper as f}
          <li class="flex items-center gap-3 px-4 py-2">
            <span class="rounded bg-signal-error/20 px-1.5 py-0.5 text-[10px] text-signal-error">{f.kind}</span>
            <span class="text-slate-100">{f.hostId}</span>
            <span class="ml-2 flex-1 text-night-200">{f.detail}</span>
            <span class="text-night-300">{new Date(f.timestamp).toLocaleTimeString()}</span>
          </li>
        {/each}
      </ul>
    {/if}
  </section>
</div>
