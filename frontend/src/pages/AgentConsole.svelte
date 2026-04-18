<!--
  OBLIVRA — Agent Console (Svelte 5)
  Deep inspection, process management and forensic control for specific agents.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { KPI, PageLayout, Badge, Button, DataTable, Toggle } from '@components/ui';
  import { Lock, Globe, RefreshCw, Activity } from 'lucide-svelte';
  import { appStore } from '@lib/stores/app.svelte';
  import { agentStore } from '@lib/stores/agent.svelte';

  // Read the agent ID injected by appStore.navigate('agent-console', { id: '...' })
  function readNavParam(key: string): string {
    try {
      const raw = sessionStorage.getItem('oblivra:nav_params');
      if (!raw) return '';
      return (JSON.parse(raw)[key] as string) ?? '';
    } catch { return ''; }
  }

  let agentID = $state(readNavParam('id'));
  const agent = $derived(agentStore.agents.find(a => a.id === agentID || a.hostname === agentID));

  // Process list — telemetry integration
  const processes = $derived(agentStore.processes);

  onMount(() => {
    const id = readNavParam('id');
    if (id) {
        agentID = id;
        agentStore.fetchProcessInventory(id);
    }
    agentStore.refresh();
  });

  let quarantine = $state(false);

  async function refreshAll() {
    agentStore.refresh();
    if (agentID) agentStore.fetchProcessInventory(agentID);
  }

  async function toggleQuarantine() {
    try {
      const targetState = !quarantine;
      await agentStore.toggleQuarantine(agentID, targetState);
      quarantine = targetState;
      appStore.notify(
        `Agent ${agentID} ${quarantine ? 'isolated' : 'restored'}`,
        quarantine ? 'warning' : 'success'
      );
    } catch (err: any) {
      appStore.notify(`Quarantine failed: ${err}`, 'error');
    }
  }

  async function killProc(pid: number) {
    if (!agentID) return;
    try {
      await agentStore.killProcess(agentID, pid);
      appStore.notify(`Kill directive sent for PID ${pid}`, 'info');
      // No need to manually filter processes, the next inventory update will handle it
    } catch (err: any) {
      appStore.notify(`Kill failed: ${err}`, 'error');
    }
  }

  const columns = [
    { key: 'pid', label: 'PID', width: '80px' },
    { key: 'name', label: 'Process / Thread' },
    { key: 'cpu', label: 'CPU', width: '80px' },
    { key: 'user', label: 'Identity', width: '100px' },
    { key: 'risk', label: 'Risk', width: '80px' },
    { key: 'actions', label: '', width: '60px' },
  ];
</script>

<PageLayout title={agent ? `Agent: ${agent.hostname}` : agentID ? `Agent: ${agentID}` : 'Agent Console'} subtitle="Real-time telemetry and atomic control for endpoint">
  {#snippet toolbar()}
    <div class="flex items-center gap-3">
        <div class="flex items-center gap-2 px-3 py-1 bg-accent/10 border border-accent/30 rounded-full">
            <div class="w-1.5 h-1.5 rounded-full {agent?.status === 'online' ? 'bg-accent animate-pulse' : 'bg-text-muted'}"></div>
            <span class="text-[9px] font-bold text-accent uppercase tracking-widest">{agent?.status === 'online' ? 'Live Stream Active' : 'Disconnected'}</span>
        </div>
        <Button variant="secondary" size="sm" onclick={refreshAll}>
            <RefreshCw size={14} class="mr-1 inline align-middle {agentStore.loading ? 'animate-spin' : ''}" />
            Refresh
        </Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
      <KPI label="Agent OS" value={agent?.version || 'N/A'} trend="stable" />
      <KPI label="Last Heartbeat" value={agent?.last_seen ? new Date(agent.last_seen).toLocaleTimeString() : 'Never'} trend="stable" variant="success" />
      <KPI label="Remote IP" value={agent?.remote_address || '0.0.0.0'} trend="stable" variant="accent" />
      <KPI label="Status" value={agent?.status || 'Offline'} variant={agent?.status === 'online' ? 'success' : 'critical'} />
    </div>

    <div class="flex-1 min-h-0 grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Process Monitor -->
      <div class="lg:col-span-2 bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col shadow-card">
         <div class="p-3 bg-surface-2 border-b border-border-primary text-[10px] font-bold uppercase tracking-widest text-text-muted font-mono">
            Process Inventory & Resource Attribution
         </div>
          <div class="flex-1 overflow-auto">
            {#if processes.length > 0}
               <DataTable data={processes} columns={columns as any} compact>
                 {#snippet render({ col, row, value })}
                   {#if col.key === 'risk'}
                      <Badge variant={row.risk === 'critical' ? 'critical' : row.risk === 'medium' ? 'warning' : 'muted'}>
                        {value}
                      </Badge>
                   {:else if col.key === 'name'}
                      <code class="text-[11px] font-bold text-text-heading">{value}</code>
                   {:else if (col.key as any) === 'actions'}
                      <Button variant="danger" size="sm" onclick={() => killProc(row.pid)}>Kill</Button>
                   {:else}
                     <span class="text-[11px] text-text-secondary">{value}</span>
                   {/if}
                 {/snippet}
               </DataTable>
            {:else}
               <div class="flex flex-col items-center justify-center h-full opacity-40 p-12 text-center">
                  <Activity size={32} class="mb-2 text-text-muted" />
                  <span class="text-[10px] uppercase font-bold tracking-widest">No Active Telemetry</span>
                  <span class="text-[9px] mt-1">Waiting for agent to stream process inventory...</span>
               </div>
            {/if}
          </div>
      </div>

      <!-- Control Sidebar -->
      <div class="flex flex-col gap-6">
         <div class="bg-surface-1 border border-border-primary rounded-md p-4 space-y-4">
            <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border-primary pb-2">Atomic Containment</div>
            <div class="flex justify-between items-center">
               <div class="flex flex-col">
                  <span class="text-xs font-bold text-text-heading">Quarantine Agent</span>
                  <span class="text-[9px] text-text-muted">Isolate from all network traffic</span>
               </div>
               <div onclick={toggleQuarantine} role="presentation">
                 <Toggle bind:checked={quarantine} />
               </div>
            </div>
            <Button variant="secondary" class="w-full flex items-center justify-center gap-2">
               <Lock size={12} class="text-accent" />
               Freeze Filesystem
            </Button>
         </div>

         <div class="flex-1 bg-surface-1 border border-border-primary rounded-md p-4 flex flex-col gap-3">
            <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border-primary pb-2">Network Topology (Edge)</div>
            <div class="flex-1 flex flex-col justify-center items-center opacity-30 gap-2">
               <Globe size={48} />
               <span class="text-[10px] uppercase font-bold tracking-widest">Scanning Meshes...</span>
            </div>
         </div>
      </div>
    </div>
  </div>
</PageLayout>
