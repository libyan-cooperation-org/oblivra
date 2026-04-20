<!--
  OBLIVRA — OQL Dashboard (Svelte 5)
  Oblivra Query Language (OQL) orchestration: Managing autonomous hunting logic and search modules.
-->
<script lang="ts">
  import { KPI, PageLayout, Badge, Button, DataTable, Input } from '@components/ui';
  import { Zap, Code, Play, Activity, Terminal } from 'lucide-svelte';
  import { siemStore } from '@lib/stores/siem.svelte';
  import { onMount } from 'svelte';

  let filter = $state('');

  const stats = $derived(siemStore.stats || {
    TotalEvents: 0,
    EventsPerSecond: 0,
    ActiveAgents: 0,
    StorageUsed: 0,
    OQLVelocity: 1.2
  });

  const queries: Record<string, string>[] = [
    { id: 'Q-01', name: 'Identity Lateral Jump', complexity: '0.82', category: 'Hunt', status: 'ready' },
    { id: 'Q-02', name: 'Entropy Spike Ingress', complexity: '0.45', category: 'Audit', status: 'ready' },
    { id: 'Q-03', name: 'Kernel Syscall Anomaly', complexity: '0.94', category: 'Forensic', status: 'running' },
  ];

  onMount(() => {
    siemStore.refreshStats();
  });
</script>

<PageLayout title="OQL Orchestration" subtitle="Managing Oblivra Query Language (OQL) modules for autonomous threat hunting and deep forensic search">
  {#snippet toolbar()}
     <div class="flex items-center gap-2">
        <Input variant="search" placeholder="Filter OQL modules..." class="w-64" bind:value={filter} />
        <Button variant="primary" size="sm" icon="< >">New Module</Button>
     </div>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
      <KPI label="Total Events" value={stats.TotalEvents.toLocaleString()} trend="stable" trendValue="Archive" />
      <KPI label="Query Velocity" value={stats.EventsPerSecond + " eps"} trend="up" trendValue="Real-time" variant="accent" />
      <KPI label="Active Agents" value={stats.ActiveAgents} trend="up" trendValue="MITRE Sync" variant="success" />
      <KPI label="Storage Used" value={(stats.StorageUsed / 1024 / 1024).toFixed(2) + "MB"} trend="stable" trendValue="Hardened" variant="success" />
    </div>

    <div class="flex-1 min-h-0 grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- OQL Registry -->
      <div class="lg:col-span-2 bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col shadow-premium">
         <div class="p-3 bg-surface-2 border-b border-border-primary flex justify-between items-center text-[10px] font-bold uppercase tracking-widest text-text-muted font-mono">
            Hunting Logic Warehouse (OQL)
         </div>
         <div class="flex-1 overflow-auto">
            <DataTable data={queries} columns={[
              { key: 'name', label: 'Module Identity' },
              { key: 'complexity', label: 'Complexity', width: '100px' },
              { key: 'category', label: 'Intent', width: '100px' },
              { key: 'status', label: 'State', width: '100px' },
              { key: 'action', label: '', width: '80px' }
            ]} compact>
              {#snippet render({ col: column, row })}
                {#if column.key === 'status'}
                   <Badge variant={row.status === 'running' ? 'accent' : 'success'} dot={row.status === 'running'}>{row.status.toUpperCase()}</Badge>
                {:else if column.key === 'complexity'}
                   <div class="flex items-center gap-2">
                      <div class="flex-1 h-1 bg-surface-3 rounded-full overflow-hidden w-12">
                         <div class="h-full bg-accent" style="width: {parseFloat(row.complexity) * 100}%"></div>
                      </div>
                      <span class="text-[10px] font-mono opacity-60">{row.complexity}</span>
                   </div>
                {:else if column.key === 'name'}
                   <div class="flex items-center gap-2">
                      <Terminal size={14} class="text-accent opacity-70" />
                      <span class="text-[11px] font-bold text-text-heading">{row.name}</span>
                   </div>
                {:else if column.key === 'action'}
                   <div class="flex gap-2">
                      <Button variant="ghost" size="xs"><Play size={12} /></Button>
                      <Button variant="ghost" size="xs"><Code size={12} /></Button>
                   </div>
                {:else}
                  <span class="text-[11px] text-text-secondary">{row[column.key]}</span>
                {/if}
              {/snippet}
            </DataTable>
         </div>
      </div>

      <!-- Engine Insights -->
      <div class="flex flex-col gap-6">
         <div class="bg-surface-1 border border-border-primary rounded-md p-6 flex flex-col items-center justify-center text-center gap-3 border-dashed shadow-sm">
            <Zap size={32} class="text-accent opacity-40" />
            <h4 class="text-xs font-bold text-text-heading uppercase tracking-widest">JIT Logic Synthesis</h4>
            <p class="text-[10px] text-text-muted max-w-[150px]">OQL modules are compiled Just-In-Time to distributed byte-code for maximum fleet execution efficiency.</p>
         </div>

         <div class="flex-1 bg-surface-1 border border-border-primary rounded-md p-4 space-y-4">
            <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border-primary pb-2 flex items-center gap-2">
               <Activity size={12} />
               Logic Throughput
            </div>
            <div class="space-y-4">
                {#each Array(3) as _, i}
                   <div>
                       <div class="flex justify-between text-[10px] mb-1">
                          <span class="text-text-secondary">Core {i} Affinity</span>
                          <span class="font-bold">{(80 + i * 5)}%</span>
                       </div>
                       <div class="w-full bg-surface-3 h-1 rounded-full overflow-hidden">
                          <div class="bg-accent h-full" style="width: {80 + i * 5}%"></div>
                       </div>
                   </div>
                {/each}
            </div>
         </div>
      </div>
    </div>
  </div>
</PageLayout>
