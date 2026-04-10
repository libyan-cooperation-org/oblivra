<!--
  OBLIVRA — Fleet Dashboard (Svelte 5)
  Global agent telemetry and mission-critical endpoint orchestration.
-->
<script lang="ts">
  import { KPI, PageLayout, Badge, Button, Chart, DataTable } from '@components/ui';
  import { Shield, Zap, Globe, Activity, Server, Cpu } from 'lucide-svelte';
  import { appStore } from '@lib/stores/app.svelte';

  const agents = [
    { id: 'AG-01', name: 'prod-web-01', status: 'online', version: '2.4.1', load: '12%', country: 'US' },
    { id: 'AG-02', name: 'prod-db-02', status: 'online', version: '2.4.1', load: '45%', country: 'DE' },
    { id: 'AG-03', name: 'staging-k8s', status: 'warning', version: '2.3.8', load: '88%', country: 'CN' },
    { id: 'AG-04', name: 'edge-gateway-4', status: 'offline', version: '2.4.0', load: '0%', country: 'SG' },
  ];

  const stats = $derived({
    total: agents.length,
    online: agents.filter(a => a.status === 'online').length,
    offline: agents.filter(a => a.status === 'offline').length,
  });
</script>

<PageLayout title="Fleet Command" subtitle="Global agent mesh and endpoint health monitoring">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm">Deploy new agent</Button>
      <Button variant="primary" size="sm">Force Mesh Update</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <!-- Fleet KPI Grid -->
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
      <KPI title="Managed Agents" value={stats.total} trend="Active Mesh" />
      <KPI title="Agent Availability" value="{((stats.online / stats.total) * 100).toFixed(1)}%" variant="success" trend="Optimal" />
      <KPI title="Global Version" value="v2.4.1" trend="85% coverage" variant="accent" />
      <KPI title="Mesh Latency" value="14ms" trend="Nominal" variant="success" />
    </div>

    <div class="flex-1 min-h-0 grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Agent Inventory -->
      <div class="lg:col-span-2 bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col shadow-premium">
         <div class="p-3 bg-surface-2 border-b border-border-primary flex justify-between items-center text-[10px] font-bold uppercase tracking-widest text-text-muted">
            End-Point Inventory
         </div>
         <div class="flex-1 overflow-auto">
            <DataTable data={agents} columns={[
              { key: 'name', label: 'Host Identifier' },
              { key: 'status', label: 'State', width: '100px' },
              { key: 'load', label: 'Load', width: '80px' },
              { key: 'version', label: 'Core', width: '80px' },
              { key: 'actions', label: '', width: '60px' }
            ]} density="compact">
              {#snippet cell({ column, row })}
                {#if column.key === 'status'}
                   <Badge variant={row.status === 'online' ? 'success' : row.status === 'warning' ? 'warning' : 'error'}>{row.status}</Badge>
                {:else if column.key === 'name'}
                   <div class="flex flex-col">
                      <span class="text-[11px] font-bold text-text-heading">{row.name}</span>
                      <span class="text-[9px] text-text-muted font-mono">{row.id} • {row.country}</span>
                   </div>
                {:else if column.key === 'load'}
                   <div class="w-full bg-surface-3 h-1 rounded-full overflow-hidden mt-1">
                      <div class="bg-accent h-full" style="width: {row.load}"></div>
                   </div>
                {:else if column.key === 'actions'}
                   <Button variant="ghost" size="xs">Shell</Button>
                {:else}
                  <span class="text-[11px] text-text-secondary">{row[column.key]}</span>
                {/if}
              {/snippet}
            </DataTable>
         </div>
      </div>

      <!-- Fleet Distribution (Mock) -->
      <div class="flex flex-col gap-6">
         <div class="bg-surface-1 border border-border-primary rounded-md p-4 flex flex-col items-center justify-center text-center gap-4 h-full relative group">
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
         </div>
      </div>
    </div>
  </div>
</PageLayout>
