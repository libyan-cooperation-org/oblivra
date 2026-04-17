<!--
  OBLIVRA — Fleet Dashboard (Svelte 5)
  Global agent telemetry and mission-critical endpoint orchestration.
-->
<script lang="ts">
  import { KPI, PageLayout, Badge, Button, DataTable } from '@components/ui';
  import { Globe, RefreshCw } from 'lucide-svelte';
  import { appStore } from '@lib/stores/app.svelte';
  import { agentStore } from '@lib/stores/agent.svelte';
  import { diagnosticsStore } from '@lib/stores/diagnostics.svelte';

  const stats = $derived({
    total: agentStore.agents.length,
    online: agentStore.agents.filter(a => a.status === 'online').length,
    offline: agentStore.agents.filter(a => a.status === 'offline').length,
  });

  const diag = $derived(diagnosticsStore.snapshot);

  const columns = [
    { key: 'hostname', label: 'Host Identifier' },
    { key: 'status', label: 'State', width: '100px' },
    { key: 'remote_address', label: 'Remote IP', width: '120px' },
    { key: 'version', label: 'Core', width: '80px' },
    { key: 'id', label: 'ID', width: '150px' },
  ] as const;
</script>

<PageLayout title="Fleet Command" subtitle="Global agent mesh and endpoint health monitoring">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm" onclick={() => agentStore.refresh()}>
        <RefreshCw size={14} class="mr-1 {agentStore.loading ? 'animate-spin' : ''}" />
        Refresh
      </Button>
      <Button variant="primary" size="sm">Deploy new agent</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <!-- Fleet KPI Grid -->
    <div class="grid grid-cols-1 md:grid-cols-5 gap-4">
      <KPI label="Managed Agents" value={stats.total} trend="stable" trendValue="Active Mesh" />
      <KPI label="Agent Availability" value={stats.total > 0 ? ((stats.online / stats.total) * 100).toFixed(1) + '%' : '—'} variant="success" trend="stable" trendValue="Optimal" />
      <KPI label="Platform Health" value={diagnosticsStore.healthGrade} trend="stable" trendValue={diag ? `${diag.ingest.current_eps} EPS` : 'Initializing...'} variant={diagnosticsStore.healthGrade === 'A' ? 'success' : diagnosticsStore.healthGrade === 'B' ? 'accent' : 'warning'} />
      <KPI label="Engine Version" value={diag?.runtime.go_version.split(' ')[0] || 'v2.4.1'} trend="stable" trendValue="Sovereign Core" variant="default" />
      <KPI label="Mesh Latency" value={diag ? `${diag.query.avg_query_ms.toFixed(1)}ms` : '—'} trend="stable" trendValue="Avg Query" variant={diag && diag.query.avg_query_ms < 100 ? 'success' : 'warning'} />
    </div>

    <div class="flex-1 min-h-0 grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Agent Inventory -->
      <div class="lg:col-span-2 bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col shadow-card">
         <div class="p-3 bg-surface-2 border-b border-border-primary text-[10px] font-bold uppercase tracking-widest text-text-muted">
            End-Point Inventory
         </div>
         <div class="flex-1 overflow-auto">
            <DataTable data={agentStore.agents} {columns} compact>
              {#snippet render({ col, row, value })}
                {#if col.key === 'status'}
                   <Badge variant={row.status === 'online' ? 'success' : 'critical'}>{row.status}</Badge>
                {:else if col.key === 'hostname'}
                   <div class="flex flex-col">
                      <span class="text-[11px] font-bold text-text-heading">{row.hostname}</span>
                      <span class="text-[9px] text-text-muted font-mono">{row.id}</span>
                   </div>
                 {:else if col.key === 'id'}
                    <div class="flex items-center gap-2">
                      <Button variant="ghost" size="sm" onclick={() => appStore.navigate('agent-console', {id: row.id})}>Inspect</Button>
                    </div>
                 {:else}
                  <span class="text-[11px] text-text-secondary">{value}</span>
                 {/if}
              {/snippet}
            </DataTable>
         </div>
      </div>

      <!-- Fleet Distribution -->
      <div class="flex flex-col gap-6">
         <div class="bg-surface-1 border border-border-primary rounded-md p-4 flex flex-col items-center justify-center text-center gap-4 flex-1 relative group">
            <Globe class="text-accent opacity-20 group-hover:opacity-40 transition-opacity" size={120} />
            <div class="absolute inset-0 flex flex-col items-center justify-center">
               <span class="text-3xl font-bold text-text-heading font-mono">14</span>
               <span class="text-[10px] text-text-muted uppercase tracking-widest font-bold">Active Geozones</span>
            </div>
         </div>

         <div class="bg-surface-1 border border-border-primary rounded-md p-4 space-y-3">
            <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border-primary pb-2">Mesh Reliability</div>
            <div class="flex items-center justify-between">
               <span class="text-[11px] text-text-secondary">Signature Integrity</span>
               <Badge variant="success">SECURE</Badge>
            </div>
            <div class="flex items-center justify-between">
               <span class="text-[11px] text-text-secondary">Drift Compensation</span>
               <Badge variant="accent">ON</Badge>
            </div>
            <div class="flex items-center justify-between">
               <span class="text-[11px] text-text-secondary">Agents Online</span>
               <Badge variant={stats.offline > 0 ? 'warning' : 'success'}>{stats.online}/{stats.total}</Badge>
            </div>
         </div>
      </div>
    </div>
  </div>
</PageLayout>
