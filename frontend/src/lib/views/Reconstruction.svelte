<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import {
    reconSessions, reconStateAt, reconCmdSus, reconAuthMulti,
    type Session, type ProcessSnapshot, type CmdLine, type AuthChain,
  } from '../bridge';
  import Tile from '../components/Tile.svelte';

  let host = $state('');
  let sessions = $state<Session[]>([]);
  let snap = $state<ProcessSnapshot | null>(null);
  let cmds = $state<CmdLine[]>([]);
  let chains = $state<AuthChain[]>([]);
  let loadError = $state<string | null>(null);
  let timer: ReturnType<typeof setInterval> | null = null;
  let inFlight = 0;

  async function refresh() {
    const seq = ++inFlight;
    try {
      const [s, c, m] = await Promise.all([reconSessions(host), reconCmdSus(), reconAuthMulti()]);
      if (seq !== inFlight) return;
      sessions = s; cmds = c; chains = m;
      // Process snapshot only meaningful with a host filter — clear it
      // when the user clears the filter so the panel doesn't show stale
      // process trees from a previously-selected host.
      if (host) {
        snap = await reconStateAt(host);
        if (seq !== inFlight) return;
      } else {
        snap = null;
      }
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
  onDestroy(() => { if (timer) clearInterval(timer); });
</script>

<div class="mx-auto max-w-7xl space-y-6 p-8">
  <header class="flex items-baseline justify-between">
    <div>
      <p class="text-xs uppercase tracking-widest text-night-300">Reconstruction</p>
      <h2 class="text-2xl font-semibold tracking-tight">Sessions · State · Cmdline · Cross-protocol auth</h2>
    </div>
    <input
      bind:value={host}
      placeholder="host filter (e.g. web-01)"
      class="rounded-md border border-night-600 bg-night-800/70 px-3 py-1.5 text-xs text-slate-100"
    />
  </header>

  <section class="grid grid-cols-2 gap-4 lg:grid-cols-4">
    <Tile label="Sessions tracked" value={sessions.length} hint={host ? `filtered: ${host}` : 'all hosts'} />
    <Tile label="Suspicious cmdlines" value={cmds.length} hint="all hosts" />
    <Tile label="Multi-protocol logins" value={chains.length} hint="all hosts · lateral-movement signal" />
    <Tile label="Processes running" value={snap?.running.length ?? '—'} hint={host || 'pick a host'} />
  </section>

  {#if loadError}
    <p class="text-xs text-signal-error">Failed to load: {loadError}</p>
  {/if}

  <section class="rounded-xl border border-night-700 bg-night-900/70">
    <div class="border-b border-night-700 px-4 py-3 text-sm font-semibold tracking-wide">Sessions</div>
    {#if sessions.length === 0}
      <div class="px-4 py-8 text-center text-sm text-night-300">No sessions reconstructed yet.</div>
    {:else}
      <ul class="divide-y divide-night-700/70 font-mono text-xs">
        {#each sessions.slice(0, 50) as s}
          <li class="flex items-center gap-3 px-4 py-2">
            <span class="w-32 truncate text-slate-100">{s.user}@{s.hostId}</span>
            <span class="text-night-300">{s.method ?? '—'}</span>
            <span class="rounded px-1.5 py-0.5 text-[10px]"
              class:bg-signal-success={s.state === 'closed'}
              class:bg-signal-warn={s.state === 'open'}
              class:bg-signal-error={s.state === 'failed'}>
              {s.state}{s.failedAttempts ? ` ×${s.failedAttempts}` : ''}
            </span>
            <span class="text-night-300">{s.sourceIp ?? '—'}</span>
            <span class="ml-auto text-night-300">{new Date(s.startedAt).toLocaleTimeString()}</span>
          </li>
        {/each}
      </ul>
    {/if}
  </section>

  <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
    <section class="rounded-xl border border-night-700 bg-night-900/70">
      <div class="border-b border-night-700 px-4 py-3 text-sm font-semibold tracking-wide">
        Suspicious cmdlines
      </div>
      <ul class="divide-y divide-night-700/70 font-mono text-xs max-h-96 overflow-y-auto scrollbar-thin">
        {#each cmds as c}
          <li class="px-4 py-2">
            <div class="flex items-center gap-2">
              <span class="text-night-200">{c.hostId}</span>
              <span class="text-night-400">·</span>
              <span class="text-night-300">{c.image ?? '—'}</span>
              <span class="ml-auto text-night-300">{new Date(c.timestamp).toLocaleTimeString()}</span>
            </div>
            <div class="mt-1 truncate text-slate-100">{c.command}</div>
          </li>
        {/each}
      </ul>
    </section>

    <section class="rounded-xl border border-night-700 bg-night-900/70">
      <div class="border-b border-night-700 px-4 py-3 text-sm font-semibold tracking-wide">
        Multi-protocol auth (lateral-movement signal)
      </div>
      {#if chains.length === 0}
        <div class="px-4 py-12 text-center text-sm text-night-300">None observed.</div>
      {:else}
        <ul class="divide-y divide-night-700/70 font-mono text-xs">
          {#each chains as ch}
            <li class="px-4 py-2">
              <div class="flex items-center gap-2">
                <span class="text-slate-100">{ch.user}</span>
                <span class="text-night-300">{ch.day}</span>
                <span class="ml-auto text-night-300">{ch.events.length} events</span>
              </div>
              <div class="mt-1 text-night-300">
                protocols: {ch.protocols.join(', ')} · hosts: {ch.hosts.join(', ')}
              </div>
            </li>
          {/each}
        </ul>
      {/if}
    </section>
  </div>

  {#if snap}
    <section class="rounded-xl border border-night-700 bg-night-900/70">
      <div class="border-b border-night-700 px-4 py-3 text-sm font-semibold tracking-wide">
        State at {new Date(snap.at).toLocaleTimeString()} on {snap.hostId}
        <span class="text-night-300">({snap.running.length} running, {snap.exited.length} exited)</span>
      </div>
      <table class="w-full font-mono text-xs">
        <thead class="text-night-300">
          <tr><th class="px-4 py-2 text-left">PID</th><th class="px-4 py-2 text-left">PPID</th><th class="px-4 py-2 text-left">Image</th><th class="px-4 py-2 text-left">Started</th></tr>
        </thead>
        <tbody class="divide-y divide-night-700/70">
          {#each snap.running as p}
            <tr><td class="px-4 py-1">{p.pid}</td><td class="px-4 py-1">{p.ppid ?? ''}</td><td class="px-4 py-1">{p.image ?? ''}</td><td class="px-4 py-1 text-night-300">{new Date(p.started).toLocaleTimeString()}</td></tr>
          {/each}
        </tbody>
      </table>
    </section>
  {/if}
</div>
