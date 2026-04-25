<!--
  OBLIVRA — Tasks & Automation (Svelte 5)
  Managing platform automation jobs, scheduled tasks and orchestration scripts.
-->
<script lang="ts">
  import { KPI, PageLayout, Badge, Button, DataTable, PopOutButton} from '@components/ui';
  import { CheckSquare, Activity, Play, RefreshCw, Trash2, Calendar } from 'lucide-svelte';

  const tasks = [
    { id: 'T-01', name: 'Identity Sync (Global)', schedule: 'Hourly', status: 'active', lastRun: '12m ago' },
    { id: 'T-02', name: 'Forensic Block Audit', schedule: 'Daily', status: 'active', lastRun: '4h ago' },
    { id: 'T-03', name: 'BGP Trust Recalibration', schedule: 'Weekly', status: 'standby', lastRun: '2d ago' },
    { id: 'T-04', name: 'Auto-Purge Expired Logs', schedule: 'Daily', status: 'error', lastRun: '14h ago' },
  ];
</script>

<PageLayout title="Automation & Tasks" subtitle="Platform orchestration: Managing scheduled automation jobs, distributed tasks and logic cleanup scripts">
  {#snippet toolbar()}
     <Button variant="secondary" size="sm">Pause All Jobs</Button>
     <Button variant="primary" size="sm" icon="+">New Task</Button>
      <PopOutButton route="/tasks" title="Tasks" />
    {/snippet}

  <div class="flex flex-col h-full gap-6">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
      <KPI label="Active Jobs" value="2" trend="stable" trendValue="Nominal" />
      <KPI label="Success Rate" value="94.2%" trend="stable" trendValue="Stable" variant="success" />
      <KPI label="Pending Tasks" value="0" trend="stable" trendValue="Zero-Queued" variant="success" />
      <KPI label="Orchestrator" value="READY" trend="stable" trendValue="Hardened" variant="success" />
    </div>

    <div class="flex-1 min-h-0 grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Task Ledger -->
      <div class="lg:col-span-2 bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col shadow-premium">
         <div class="p-3 bg-surface-2 border-b border-border-primary flex justify-between items-center text-[10px] font-bold uppercase tracking-widest text-text-muted">
            Automation Strategy Ledger
         </div>
         <div class="flex-1 overflow-auto">
            <DataTable data={tasks} columns={[
              { key: 'name', label: 'Job Identity' },
              { key: 'schedule', label: 'Frequency', width: '100px' },
              { key: 'lastRun', label: 'Last Execution', width: '120px' },
              { key: 'status', label: 'State', width: '100px' },
              { key: 'id', label: 'Actions', width: '120px' }
            ]} compact striped>
              {#snippet render({ col, row, value })}
                {#if col.key === 'status'}
                   <Badge variant={value === 'active' ? 'success' : value === 'error' ? 'critical' : 'muted'} dot={value === 'active'}>{value.toUpperCase()}</Badge>
                {:else if col.key === 'name'}
                   <div class="flex items-center gap-2">
                      <CheckSquare size={14} class="text-accent opacity-70" />
                      <span class="text-[11px] font-bold text-text-heading">{value}</span>
                   </div>
                {:else if col.key === 'id'}
                   <div class="flex gap-2">
                      <Button variant="ghost" size="sm"><Play size={12} /></Button>
                      <Button variant="ghost" size="sm"><RefreshCw size={12} /></Button>
                      <Button variant="ghost" size="sm" class="text-error"><Trash2 size={12} /></Button>
                   </div>
                {:else if col.key === 'lastRun'}
                   <span class="text-[10px] font-mono text-text-muted">{value}</span>
                {:else}
                   <span class="text-[11px] text-text-secondary">{value}</span>
                {/if}
              {/snippet}
            </DataTable>
         </div>
      </div>

      <!-- Orchestration Insights -->
      <div class="flex flex-col gap-6">
         <div class="bg-surface-1 border border-border-primary rounded-md p-6 flex flex-col items-center justify-center text-center gap-4 relative overflow-hidden border-dashed shadow-sm">
            <Calendar size={32} class="text-accent opacity-40" />
            <h4 class="text-xs font-bold text-text-heading uppercase tracking-widest">Temporal Sharding</h4>
            <p class="text-[10px] text-text-muted max-w-[150px]">OBLIVRA shards task execution across the fleet to avoid logic-contention spikes.</p>
         </div>

         <div class="flex-1 bg-surface-1 border border-border-primary rounded-md p-4 space-y-4">
            <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border-primary pb-2 flex items-center gap-2">
               <Activity size={12} />
               Job Execution Entropy
            </div>
            <div class="flex-1 h-32 flex items-end justify-between px-2 gap-1 font-mono">
               {#each Array(12) as _}
                  <div class="flex-1 bg-accent/20 rounded-t-sm border-x border-accent/5" style="height: {20 + Math.random() * 70}%"></div>
               {/each}
            </div>
         </div>
      </div>
    </div>
  </div>
</PageLayout>
