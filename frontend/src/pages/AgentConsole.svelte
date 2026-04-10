<!--
  OBLIVRA — Agent Console (Svelte 5)
  Deep inspection, process management and forensic control for specific agents.
-->
<script lang="ts">
  import { KPI, PageLayout, Badge, Button, DataTable, Toggle } from '@components/ui';
  import { Shield, Zap, Search, Cpu, Database, Activity, Lock, Power } from 'lucide-svelte';
  import { appStore } from '@lib/stores/app.svelte';

  const processes = [
    { pid: 1421, name: 'nginx', cpu: '2.4%', mem: '142MB', user: 'www-data', risk: 'low' },
    { id: 2, pid: 4991, name: 'kworker/u:2', cpu: '12%', mem: '12KB', user: 'root', risk: 'medium' },
    { id: 3, pid: 8821, name: './sh -i', cpu: '0.1%', mem: '440KB', user: 'operator', risk: 'critical' },
  ];

  let quarantine = $state(false);
</script>

<PageLayout title="Agent Inspect: prod-web-01" subtitle="Real-time telemetry and atomic control for endpoint AG-01">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
       <div class="flex items-center gap-2 px-3 py-1 bg-accent/10 border border-accent/30 rounded-full">
          <div class="w-1.5 h-1.5 rounded-full bg-accent animate-pulse"></div>
          <span class="text-[9px] font-bold text-accent uppercase tracking-widest">Live Stream Active</span>
       </div>
       <Button variant="secondary" size="sm">Kernel Trace</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
      <KPI title="Agent CPU" value="14.2%" trend="Nominal" />
      <KPI title="Memory Pressure" value="Low" trend="12.4 GB Free" variant="success" />
      <KPI title="Integrity Check" value="PASSED" trend="Secure Boot" variant="success" />
      <KPI title="Risk Factor" value="0.12" trend="Elevated" variant="warning" />
    </div>

    <div class="flex-1 min-h-0 grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Process Monitor -->
      <div class="lg:col-span-2 bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col shadow-premium">
         <div class="p-3 bg-surface-2 border-b border-border-primary flex justify-between items-center text-[10px] font-bold uppercase tracking-widest text-text-muted font-mono">
            Process Inventory & Resource Attribution
         </div>
         <div class="flex-1 overflow-auto">
            <DataTable data={processes} columns={[
              { key: 'pid', label: 'PID', width: '80px' },
              { key: 'name', label: 'Process / Thread' },
              { key: 'cpu', label: 'CPU', width: '80px' },
              { key: 'user', label: 'Identity', width: '100px' },
              { key: 'risk', label: 'Risk', width: '80px' },
              { key: 'actions', label: '', width: '60px' }
            ]} density="compact">
              {#snippet cell({ column, row })}
                {#if column.key === 'risk'}
                   <Badge variant={row.risk === 'critical' ? 'error' : row.risk === 'medium' ? 'warning' : 'info'}>{row.risk}</Badge>
                {:else if column.key === 'name'}
                   <code class="text-[11px] font-bold text-text-heading">{row.name}</code>
                {:else if column.key === 'actions'}
                   <Button variant="ghost" size="xs" class="text-error">Kill</Button>
                {:else}
                  <span class="text-[11px] text-text-secondary">{row[column.key]}</span>
                {/if}
              {/snippet}
            </DataTable>
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
               <Toggle bind:checked={quarantine} onChange={() => appStore.notify(`Agent prod-web-01 ${quarantine ? 'isolated' : 'restored'}`, 'warning')} />
            </div>
            <Button variant="secondary" class="w-full text-xs py-2 flex items-center justify-center gap-2">
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
