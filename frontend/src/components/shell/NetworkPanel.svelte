<!--
  NetworkPanel — listening sockets + routes for the selected host.
  Runs `ss -tlnp 2>/dev/null || netstat -tlnp 2>/dev/null` for sockets
  and `ip route 2>/dev/null || route -n 2>/dev/null` for routes — fallbacks
  cover both modern and old (BusyBox) Linux boxes.
-->
<script lang="ts">
  import { Radar } from 'lucide-svelte';
  import RemoteExecPanel from './RemoteExecPanel.svelte';

  type Tab = 'sockets' | 'routes';
  let tab = $state<Tab>('sockets');

  const SOCKET_CMD = "ss -tlnp 2>/dev/null || netstat -tlnp 2>/dev/null";
  const ROUTE_CMD = "ip route 2>/dev/null || route -n 2>/dev/null";

  // Reactive: which command we run is whichever tab is active.
  let cmd = $derived(tab === 'sockets' ? SOCKET_CMD : ROUTE_CMD);
</script>

<RemoteExecPanel title="Network" icon={Radar} command={cmd} pollIntervalMs={10_000}>
  {#snippet controls({ refresh: _refresh })}
    <div class="mr-2 flex items-center gap-0.5 rounded-md border border-[var(--b1)] bg-[var(--s2)] p-0.5">
      {#each ['sockets', 'routes'] as t}
        <button
          class="rounded px-2 py-0.5 text-[10px] uppercase tracking-wider {tab === t
            ? 'bg-cyan-400/15 text-cyan-200'
            : 'text-[var(--tx3)] hover:text-[var(--tx)]'}"
          onclick={() => (tab = t as Tab)}
        >{t}</button>
      {/each}
    </div>
  {/snippet}
  {#snippet children({ output })}
    <pre class="whitespace-pre p-3 font-mono text-[11px] leading-relaxed text-[var(--tx)]">{output || '(no output)'}</pre>
  {/snippet}
</RemoteExecPanel>
