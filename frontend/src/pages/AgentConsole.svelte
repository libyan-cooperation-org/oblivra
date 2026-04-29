<!--
  Agent Console — per-agent deep view: status, processes, quarantine, kill.
  Bound to AgentService + agentStore.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, Badge, Button, KPI, DataTable, PopOutButton } from '@components/ui';
  import { Cpu, Terminal as TerminalIcon, ShieldAlert, Skull, RefreshCw } from 'lucide-svelte';
  import { agentStore } from '@lib/stores/agent.svelte';
  import { appStore } from '@lib/stores/app.svelte';
  import { IS_BROWSER } from '@lib/context';
  import { push } from '@lib/router.svelte';

  let selectedID = $state<string | null>(null);
  let processes = $state<any[]>([]);
  let loadingProcs = $state(false);

  const selected = $derived(agentStore.agents.find((a) => a.id === selectedID) ?? null);

  async function refreshAgents() {
    if (typeof agentStore.init === 'function') await agentStore.init();
  }

  async function loadProcesses(id: string) {
    loadingProcs = true;
    try {
      if (IS_BROWSER) { processes = []; return; }
      const { RequestProcessInventory } = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/agentservice'
      );
      const list = (await RequestProcessInventory(id)) as any[];
      processes = list ?? [];
    } catch (e: any) {
      appStore.notify(`Process inventory failed: ${e?.message ?? e}`, 'error');
      processes = [];
    } finally { loadingProcs = false; }
  }

  // Phase 36.7: killProcess + quarantine handlers removed. The MCP
  // ForensicEngine response chain (KillProcess RPC, ToggleQuarantine RPC,
  // ActionIsolateNetwork agent action) was deleted with the broad scope cut.
  // Process inventory + agent status remain read-only signals; pair detection
  // events with an external SOAR for active response.

  $effect(() => {
    if (selectedID) void loadProcesses(selectedID);
  });

  onMount(() => {
    void refreshAgents();
    if (agentStore.agents.length > 0) selectedID = agentStore.agents[0].id;
  });
</script>

<PageLayout title="Agent Console" subtitle="Per-host deep inspection and tactical control">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" icon={RefreshCw} onclick={refreshAgents}>Refresh</Button>
    <PopOutButton route="/agents" title="Agent Console" />
  {/snippet}

  <div class="flex h-full gap-4 -m-2">
    <!-- Sidebar: agent list -->
    <aside class="w-64 shrink-0 bg-surface-1 border border-border-primary rounded-md flex flex-col">
      <div class="p-3 border-b border-border-primary">
        <span class="text-[10px] uppercase tracking-widest font-bold">Agents</span>
        <span class="text-[10px] text-text-muted ml-2">{agentStore.agents.length}</span>
      </div>
      <div class="flex-1 overflow-y-auto">
        {#each agentStore.agents as a (a.id)}
          <button
            class="w-full text-left px-3 py-2 border-b border-border-primary hover:bg-surface-2 {selectedID === a.id ? 'bg-surface-2' : ''}"
            onclick={() => (selectedID = a.id)}
          >
            <div class="flex items-center gap-2">
              <span class="w-1.5 h-1.5 rounded-full {a.status === 'online' ? 'bg-success' : 'bg-text-muted'}"></span>
              <span class="text-[11px] font-bold truncate">{a.hostname || a.id}</span>
            </div>
            <div class="text-[9px] font-mono text-text-muted truncate">{a.remote_address ?? '—'}</div>
          </button>
        {/each}
        {#if agentStore.agents.length === 0}
          <div class="p-4 text-center text-[11px] text-text-muted">No agents registered.</div>
        {/if}
      </div>
    </aside>

    <!-- Main -->
    <section class="flex-1 flex flex-col gap-4 min-w-0">
      {#if !selected}
        <div class="bg-surface-1 border border-border-primary rounded-md p-12 text-center text-sm text-text-muted">
          Pick an agent on the left to inspect.
        </div>
      {:else}
        <!-- KPIs -->
        <div class="grid grid-cols-1 md:grid-cols-4 gap-3 shrink-0">
          <KPI label="Status" value={selected.status ?? '—'} variant={selected.status === 'online' ? 'success' : 'muted'} />
          <KPI label="OS" value={selected.os ?? '—'} variant="muted" />
          <KPI label="Address" value={selected.remote_address ?? '—'} variant="muted" />
          <KPI label="Processes" value={processes.length.toString()} variant={loadingProcs ? 'muted' : 'accent'} />
        </div>

        <!-- Phase 36.7: Open Shell + Isolate + Release buttons removed. -->
        <div class="flex items-center gap-2 shrink-0">
          <Button variant="ghost"     size="sm" icon={RefreshCw} onclick={() => loadProcesses(selectedID!)}>{loadingProcs ? 'Loading…' : 'Refresh procs'}</Button>
        </div>

        <!-- Process table -->
        <div class="flex-1 min-h-0 bg-surface-1 border border-border-primary rounded-md overflow-hidden">
          <div class="flex items-center gap-2 p-3 border-b border-border-primary">
            <Cpu size={14} class="text-accent" />
            <span class="text-[10px] uppercase tracking-widest font-bold">Process Inventory</span>
          </div>
          {#if processes.length === 0}
            <div class="p-12 text-center text-sm text-text-muted">{loadingProcs ? 'Querying agent…' : 'No process data.'}</div>
          {:else}
            <DataTable
              data={processes}
              columns={[
                { key: 'pid',  label: 'PID',  width: '80px' },
                { key: 'name', label: 'Name' },
                { key: 'user', label: 'User', width: '120px' },
                { key: 'cpu',  label: '%CPU', width: '70px' },
                { key: 'mem',  label: '%MEM', width: '70px' },
                { key: 'kill', label: '',     width: '60px' },
              ]}
              compact
            >
              {#snippet render({ col, row })}
                {#if col.key === 'pid'}
                  <span class="font-mono text-[10px]">{row.pid ?? '—'}</span>
                {:else if col.key === 'name'}
                  <span class="font-mono text-[11px]">{row.name ?? row.command ?? '—'}</span>
                {:else if col.key === 'cpu' || col.key === 'mem'}
                  <span class="font-mono text-[10px] text-text-muted">{(row[col.key] ?? 0).toFixed?.(1) ?? row[col.key] ?? '—'}</span>
                {:else if col.key === 'kill'}
                  <!-- Phase 36.7: kill action removed (response chain deleted). -->
                  <span></span>
                {:else}
                  <span class="text-[11px]">{row[col.key] ?? '—'}</span>
                {/if}
              {/snippet}
            </DataTable>
          {/if}
        </div>
      {/if}
    </section>
  </div>
</PageLayout>
