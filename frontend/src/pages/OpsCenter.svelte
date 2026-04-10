<!--
  OBLIVRA — Ops Center (Svelte 5)
  The Command and Control Hub: Managing clusters, active operations and host fleet.
-->
<script lang="ts">
  import { KPI, Badge, DataTable, PageLayout, Button, Input, Tabs } from '@components/ui';
  import { Server, Activity, Terminal, Shield, Cpu, Filter, Search, Zap } from 'lucide-svelte';
  import { appStore } from '@lib/stores/app.svelte';

  let searchQuery = $state('');
  let activeTab = $state('hosts');

  const hosts = $derived(appStore.hosts.filter(h => 
    h.label.toLowerCase().includes(searchQuery.toLowerCase()) || 
    h.address.includes(searchQuery)
  ));

  const sessions = $derived(appStore.sessions);
</script>

<PageLayout title="Operations Center" subtitle="Master command & control of all managed endpoints and active mission-sets">
  {#snippet toolbar()}
     <div class="flex items-center gap-2">
        <Input variant="search" placeholder="Filter fleet..." bind:value={searchQuery} class="w-64" />
        <Button variant="primary" size="sm">New Mission</Button>
     </div>
  {/snippet}

  <Tabs bind:active={activeTab} items={[
    { id: 'hosts', label: 'Fleet Inventory' },
    { id: 'sessions', label: 'Active Sessions' },
    { id: 'performance', label: 'Cluster Metrics' },
    { id: 'logs', label: 'Audit Trail' }
  ]} />

  <div class="flex flex-col h-full gap-5 mt-4">
    {#if activeTab === 'hosts'}
       <div class="grid grid-cols-1 md:grid-cols-4 gap-4 shrink-0">
          <KPI label="Total Fleet" value={appStore.hosts.length} trend="stable" trendValue="Active" />
          <KPI label="Available Slots" value="24" trend="stable" trendValue="Nominal" variant="success" />
          <KPI label="Encryption Level" value="v3.1" trend="stable" trendValue="Hardened" variant="accent" />
          <KPI label="Cluster Sync" value="100%" trend="stable" trendValue="Zero Lag" variant="success" />
       </div>

       <div class="flex-1 min-h-0 bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col shadow-premium">
          <div class="p-3 bg-surface-2 border-b border-border-primary flex justify-between items-center text-[10px] font-bold uppercase tracking-widest text-text-muted">
             Global Host Registry
          </div>
          <div class="flex-1 overflow-auto">
             <DataTable data={hosts} columns={[
               { key: 'label', label: 'Entity Label' },
               { key: 'address', label: 'Access Path', width: '200px' },
               { key: 'port', label: 'Port', width: '80px' },
               { key: 'platform', label: 'Platform', width: '100px' },
               { key: 'action', label: '', width: '120px' }
             ]} compact>
               {#snippet render({ value, col, row })}
                 {#if col.key === 'label'}
                    <div class="flex items-center gap-2">
                       <Server size={14} class="text-accent" />
                       <span class="text-[11px] font-bold text-text-heading">{value}</span>
                    </div>
                 {:else if col.key === 'address'}
                    <code class="text-[10px] font-mono text-text-secondary opacity-70">{value}</code>
                 {:else if col.key === 'action'}
                    <div class="flex gap-2">
                       <Button variant="ghost" size="xs">Shell</Button>
                       <Button variant="ghost" size="xs">DPI</Button>
                    </div>
                 {:else}
                    <span class="text-[11px] text-text-secondary">{value}</span>
                 {/if}
               {/snippet}
             </DataTable>
          </div>
       </div>
    {:else if activeTab === 'sessions'}
       <div class="grid grid-cols-1 md:grid-cols-3 gap-4 shrink-0">
          <KPI label="Live Pipelines" value={sessions.filter(s => s.status === 'active').length} trend="stable" trendValue="Encrypted" variant="accent" />
          <KPI label="Transfers Active" value={appStore.transfers.filter(t => t.status === 'active').length} trend="up" trendValue="In-flight" variant="warning" />
          <KPI label="Mean Session Age" value="1.4h" trend="stable" trendValue="Stable" />
       </div>

       <div class="flex-1 min-h-0 bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col">
          <div class="p-3 bg-surface-2 border-b border-border-primary text-[10px] font-bold uppercase tracking-widest text-text-muted">Active Tunnel Orchestration</div>
          <div class="flex-1 overflow-auto p-4 space-y-3">
             {#if sessions.length === 0}
                <div class="h-full flex flex-col items-center justify-center text-center opacity-20 py-20 grayscale">
                   <Terminal size={48} />
                   <span class="text-xs font-bold mt-4 uppercase tracking-widest">No Active War-heads</span>
                </div>
             {:else}
                {#each sessions as session}
                   <div class="flex items-center justify-between p-3 bg-surface-2 border border-border-secondary rounded-sm">
                      <div class="flex items-center gap-4">
                         <div class="w-2 h-2 rounded-full {session.status === 'active' ? 'bg-success animate-pulse' : 'bg-text-muted'}"></div>
                         <div class="flex flex-col">
                            <span class="text-[11px] font-bold text-text-heading">{session.hostLabel}</span>
                            <span class="text-[9px] text-text-muted font-mono">{session.id} • {session.protocol}</span>
                         </div>
                      </div>
                      <div class="flex gap-2">
                         <Button variant="secondary" size="xs">Attach</Button>
                         <Button variant="ghost" size="xs" class="text-error">Kill</Button>
                      </div>
                   </div>
                {/each}
             {/if}
          </div>
       </div>
    {/if}
  </div>
</PageLayout>
