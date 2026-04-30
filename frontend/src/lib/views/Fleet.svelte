<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { fleetList, intelList, type Agent, type Indicator } from '../bridge';
  import Tile from '../components/Tile.svelte';

  let agents = $state<Agent[]>([]);
  let iocs = $state<Indicator[]>([]);
  let timer: ReturnType<typeof setInterval> | null = null;

  async function refresh() {
    const [a, i] = await Promise.all([fleetList(), intelList()]);
    agents = a;
    iocs = i;
  }

  onMount(() => {
    void refresh();
    timer = setInterval(() => void refresh(), 4000);
  });
  onDestroy(() => {
    if (timer) clearInterval(timer);
  });
</script>

<div class="mx-auto max-w-7xl space-y-6 p-8">
  <header>
    <p class="text-xs uppercase tracking-widest text-night-300">Fleet</p>
    <h2 class="text-2xl font-semibold tracking-tight">Agents · Threat intel</h2>
  </header>

  <section class="grid grid-cols-2 gap-4 lg:grid-cols-4">
    <Tile label="Agents registered" value={agents.length} />
    <Tile label="Threat indicators" value={iocs.length} />
    <Tile
      label="Healthy (last 5m)"
      value={agents.filter((a) => Date.now() - new Date(a.lastSeen).getTime() < 5 * 60 * 1000).length}
    />
    <Tile
      label="Total events from agents"
      value={agents.reduce((s, a) => s + (a.events ?? 0), 0)}
    />
  </section>

  <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
    <section class="rounded-xl border border-night-700 bg-night-900/70">
      <div class="border-b border-night-700 px-4 py-3 text-sm font-semibold tracking-wide text-slate-100">
        Agents
      </div>
      {#if agents.length === 0}
        <div class="px-4 py-12 text-center text-sm text-night-300">
          No agents registered yet. POST to /api/v1/agent/register to add one.
        </div>
      {:else}
        <ul class="divide-y divide-night-700/70 font-mono text-xs">
          {#each agents as a (a.id)}
            <li class="flex items-center gap-3 px-4 py-2">
              <span class="w-32 truncate text-slate-100">{a.hostname}</span>
              <span class="text-night-300">{a.os}/{a.arch}</span>
              <span class="text-night-400">·</span>
              <span class="text-night-300">{a.events} events</span>
              <span class="ml-auto text-night-300">{new Date(a.lastSeen).toLocaleTimeString()}</span>
            </li>
          {/each}
        </ul>
      {/if}
    </section>

    <section class="rounded-xl border border-night-700 bg-night-900/70">
      <div class="border-b border-night-700 px-4 py-3 text-sm font-semibold tracking-wide text-slate-100">
        Threat-intel indicators
      </div>
      <ul class="divide-y divide-night-700/70 font-mono text-xs">
        {#each iocs as i (i.value)}
          <li class="flex items-center gap-3 px-4 py-2">
            <span class="rounded border border-night-600 px-1.5 py-0.5 text-[10px] text-night-200">
              {i.type}
            </span>
            <span class="text-slate-100">{i.value}</span>
            <span class="text-night-300">· {i.source ?? 'manual'}</span>
            <span class="ml-auto text-night-300">{i.severity ?? '—'}</span>
          </li>
        {/each}
      </ul>
    </section>
  </div>
</div>
