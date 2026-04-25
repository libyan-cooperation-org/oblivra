<!--
  OBLIVRA — Fleet Dashboard (Svelte 5)
  Real-time visibility into the sovereign agent fleet.
-->
<script lang="ts">
  import { PageLayout, Badge, Button, DataTable, Input, Tabs, PopOutButton } from '@components/ui';
  import { Activity, Terminal, ShieldAlert, MoreHorizontal, Monitor, Clock, ShieldCheck } from 'lucide-svelte';
  import { agentStore } from '@lib/stores/agent.svelte';

  let searchQuery = $state('');
  let activeTab = $state('ALL HOSTS');

  const stats = $derived({
    total: agentStore.agents.length,
    online: agentStore.agents.filter(a => a.status === 'online').length,
    critical: agentStore.agents.filter(a => a.severity === 'critical').length,
    health: '98.2%'
  });

  const filteredAgents = $derived(agentStore.agents.filter(a => {
    const matchesSearch = a.hostname?.toLowerCase().includes(searchQuery.toLowerCase()) || 
                         a.remote_address?.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesTab = activeTab === 'ALL HOSTS' || a.status?.toUpperCase() === activeTab;
    return matchesSearch && matchesTab;
  }));  const tabItems = [
    { id: 'ALL HOSTS', label: 'ALL HOSTS' },
    { id: 'ONLINE', label: 'ONLINE' },
    { id: 'OFFLINE', label: 'OFFLINE' }
  ];

  function handleAction(agentId: string, action: string) {
    console.log(`Executing ${action} on ${agentId}`);
  }
</script>

<PageLayout title="Fleet Management" subtitle="Real-time orchestration of the sovereign agent mesh">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Input variant="search" placeholder="Filter agents..." bind:value={searchQuery} class="w-64" />
      <Button variant="secondary" size="sm">EXPORT LIST</Button>
      <Button variant="primary" size="sm">DEPLOY AGENT</Button>
      <PopOutButton route="/fleet" title="Fleet Management" />
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-0 -m-6">
    <!-- METRIC STRIP -->
    <div class="grid grid-cols-4 gap-px bg-border-primary border-b border-border-primary shrink-0">
        <div class="bg-surface-2 p-3 group hover:bg-surface-3 transition-colors cursor-pointer">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Total Fleet</div>
            <div class="text-xl font-mono font-bold text-text-heading">{stats.total}</div>
            <div class="text-[9px] text-text-muted mt-1">Managed endpoints</div>
        </div>
        <div class="bg-surface-2 p-3 group hover:bg-surface-3 transition-colors cursor-pointer">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Live Nodes</div>
            <div class="text-xl font-mono font-bold text-success">{stats.online}</div>
            <div class="text-[9px] text-success mt-1">Active heartbeats</div>
        </div>
        <div class="bg-surface-2 p-3 group hover:bg-surface-3 transition-colors cursor-pointer">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Health Score</div>
            <div class="text-xl font-mono font-bold text-accent">{stats.health}</div>
            <div class="text-[9px] text-accent mt-1">Fleet stability nominal</div>
        </div>
        <div class="bg-surface-2 p-3 group hover:bg-surface-3 transition-colors cursor-pointer border-r-0">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Critical Issues</div>
            <div class="text-xl font-mono font-bold text-error">{stats.critical}</div>
            <div class="text-[9px] text-error mt-1 uppercase animate-pulse">Action required</div>
        </div>
    </div>

    <!-- MAIN TABLE AREA -->
    <div class="flex-1 flex flex-col min-h-0 bg-surface-1">
        <div class="px-4 py-2 border-b border-border-primary flex items-center justify-between shrink-0">
            <Tabs tabs={tabItems} bind:active={activeTab} />
            <div class="flex items-center gap-4 text-[8px] font-mono text-text-muted">
>
                <span>Last Sync: 14s ago</span>
                <span class="w-2 h-2 rounded-full bg-success animate-pulse"></span>
            </div>
        </div>

        <div class="flex-1 overflow-auto mask-fade-bottom">
            <DataTable 
                data={filteredAgents} 
                columns={[
                    { key: 'status', label: 'STATUS', width: '80px' },
                    { key: 'hostname', label: 'ENDPOINT_NAME' },
                    { key: 'remote_address', label: 'IP_ADDRESS', width: '120px' },
                    { key: 'os', label: 'PLATFORM', width: '100px' },
                    { key: 'arch', label: 'ARCH', width: '80px' },
                    { key: 'version', label: 'VERSION', width: '80px' },
                    { key: 'id', label: 'ACTIONS', width: '120px' }
                ]} 
                compact
            >
                {#snippet render({ col, row })}
                    {#if col.key === 'status'}
                        <div class="flex items-center gap-2">
                            <div class="w-1.5 h-1.5 rounded-full {row.status === 'online' ? 'bg-success shadow-[0_0_4px_rgba(34,197,94,0.6)]' : 'bg-text-muted'}"></div>
                            <Badge variant={row.status === 'online' ? 'success' : 'muted'} size="xs">{row.status}</Badge>
                        </div>
                    {:else if col.key === 'hostname'}
                        <div class="flex items-center gap-2 py-1">
                            <Monitor size={12} class="text-text-muted" />
                            <div class="flex flex-col">
                                <span class="text-[10px] font-bold text-text-heading uppercase">{row.hostname}</span>
                                <span class="text-[8px] font-mono text-text-muted opacity-60 tabular-nums">{row.id}</span>
                            </div>
                        </div>
                    {:else if col.key === 'remote_address'}
                        <span class="text-[9px] font-mono text-accent tabular-nums">{row.remote_address}</span>
                    {:else if col.key === 'os'}
                        <div class="flex items-center gap-1.5">
                           <span class="text-[9px] font-mono text-text-muted uppercase">{row.os || 'Linux'}</span>
                        </div>
                    {:else if col.key === 'arch'}
                        <span class="text-[8px] font-mono text-text-muted uppercase opacity-60">{row.arch || 'x64'}</span>
                    {:else if col.key === 'version'}
                        <span class="text-[8px] font-mono text-text-muted">v{row.version || '1.0'}</span>
                    {:else if col.key === 'id'}
                        <div class="flex items-center gap-1.5 opacity-0 group-hover:opacity-100 transition-opacity">
                            <button 
                                class="p-1 hover:bg-surface-3 rounded-sm text-accent border border-accent/20 transition-colors"
                                title="Terminal"
                                onclick={() => handleAction(row.id, 'terminal')}
                            >
                                <Terminal size={12} />
                            </button>
                            <button 
                                class="p-1 hover:bg-surface-3 rounded-sm text-error border border-error/20 transition-colors"
                                title="Isolate"
                                onclick={() => handleAction(row.id, 'isolate')}
                            >
                                <ShieldAlert size={12} />
                            </button>
                            <button 
                                class="p-1 hover:bg-surface-3 rounded-sm text-text-muted border border-border-primary transition-colors"
                                title="More"
                            >
                                <MoreHorizontal size={12} />
                            </button>
                        </div>
                    {/if}
                {/snippet}
            </DataTable>
        </div>
    </div>

    <!-- STATUS BAR -->
    <div class="bg-surface-2 border-t border-border-primary px-3 py-1.5 flex items-center justify-between shrink-0">
        <div class="flex items-center gap-6">
            <div class="flex items-center gap-2 text-[8px] font-mono text-text-muted uppercase">
                <Activity size={10} class="text-success" />
                <span>Sync Pipeline: Nominal</span>
            </div>
            <div class="flex items-center gap-2 text-[8px] font-mono text-text-muted uppercase">
                <Clock size={10} class="text-accent" />
                <span>Next Heartbeat: 4s</span>
            </div>
            <div class="flex items-center gap-2 text-[8px] font-mono text-text-muted uppercase">
                <ShieldCheck size={10} class="text-success" />
                <span>Integrity: 100%</span>
            </div>
        </div>
        <div class="text-[8px] font-mono text-text-muted uppercase tracking-[0.2em] opacity-40">
            Fleet_Core v1.4.2 — Sovereign Mesh
        </div>
    </div>
  </div>
</PageLayout>
