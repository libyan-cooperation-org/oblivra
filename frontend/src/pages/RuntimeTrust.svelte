<!--
  OBLIVRA — Runtime Trust (Svelte 5)
  Real-time process verification and execution trust orchestration.
-->
<script lang="ts">
  import { KPI, PageLayout, Badge, Button, DataTable } from '@components/ui';
  import { ShieldCheck, Activity, Cpu, Eye } from 'lucide-svelte';

  const processes: Record<string, any>[] = [
    { pid: 1422, name: 'oblivra-agent', trust: 1.0, status: 'verified', memory: '42MB' },
    { pid: 8821, name: 'kworker/u16:1', trust: 0.98, status: 'baseline', memory: '0MB' },
    { pid: 9001, name: 'unknown-binary', trust: 0.12, status: 'quarantine', memory: '128MB' },
  ];
</script>

<PageLayout title="Runtime Trust" subtitle="Real-time process verification: Monitoring binary integrity and execution trust scores">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm">Recalibrate Baselines</Button>
    <Button variant="primary" size="sm" icon="🛡️">Scan Active Memory</Button>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
      <KPI label="Verified Load" value="94%" trend="stable" trendValue="Secure" variant="success" />
      <KPI label="Quarantined" value="1" trend="stable" trendValue="Active" variant="critical" />
      <KPI label="Memory Integrity" value="High" trend="stable" trendValue="NOMINAL" variant="success" />
      <KPI label="Kernel Trust" value="Ring 0" trend="stable" trendValue="Hardened" variant="accent" />
    </div>

    <div class="flex-1 min-h-0 grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Process Registry -->
      <div class="lg:col-span-2 bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col shadow-premium">
         <div class="p-3 bg-surface-2 border-b border-border-primary flex justify-between items-center text-[10px] font-bold uppercase tracking-widest text-text-muted">
            Process Execution Trust Ledger
         </div>
         <div class="flex-1 overflow-auto">
            <DataTable data={processes} columns={[
              { key: 'name', label: 'Process / Binary' },
              { key: 'pid', label: 'PID', width: '80px' },
              { key: 'trust', label: 'Trust Score', width: '100px' },
              { key: 'status', label: 'State', width: '120px' },
              { key: 'action', label: '', width: '60px' }
            ]} compact>
              {#snippet render({ col: column, row })}
                {#if column.key === 'status'}
                   <Badge variant={row.status === 'verified' ? 'success' : row.status === 'quarantine' ? 'critical' : 'muted'}>{row.status.toUpperCase()}</Badge>
                {:else if column.key === 'trust'}
                   <div class="flex items-center gap-2">
                      <div class="flex-1 h-1 bg-surface-3 rounded-full overflow-hidden w-12">
                         <div class="h-full {row.trust > 0.8 ? 'bg-success' : row.trust > 0.4 ? 'bg-warning' : 'bg-error'}" style="width: {row.trust * 100}%"></div>
                      </div>
                      <span class="text-[10px] font-mono">{row.trust.toFixed(2)}</span>
                   </div>
                {:else if column.key === 'name'}
                   <div class="flex items-center gap-2">
                      <Cpu size={14} class="text-accent opacity-70" />
                      <span class="text-[11px] font-bold text-text-heading">{row.name}</span>
                   </div>
                {:else if column.key === 'action'}
                   <Button variant="ghost" size="xs"><Eye size={12} /></Button>
                {:else}
                  <span class="text-[11px] text-text-secondary">{row[column.key]}</span>
                {/if}
              {/snippet}
            </DataTable>
         </div>
      </div>

      <!-- Trust Engine Insights -->
      <div class="flex flex-col gap-6">
         <div class="bg-surface-1 border border-border-primary rounded-md p-6 flex flex-col items-center justify-center text-center gap-4 border-dashed shadow-sm">
            <ShieldCheck size={48} class="text-success opacity-40" />
            <h4 class="text-xs font-bold text-text-heading uppercase tracking-widest">Cognitive Trust Bridge</h4>
            <p class="text-[10px] text-text-muted max-w-[150px]">OBLIVRA uses behavioral AI to calculate trust scores based on syscall patterns and entropy shifts.</p>
         </div>

         <div class="flex-1 bg-surface-1 border border-border-primary rounded-md p-4 space-y-4">
            <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border-primary pb-2 flex items-center gap-2">
               <Activity size={12} />
               Trust Drift (24H)
            </div>
            <div class="flex-1 h-24 flex items-end justify-between px-1 gap-1">
               {#each Array(15) as _, i}
                  <div class="flex-1 {i === 12 ? 'bg-error/40' : 'bg-accent/20'} rounded-t-sm" style="height: {30 + Math.random() * 70}%"></div>
               {/each}
            </div>
            <div class="flex justify-between text-[8px] text-text-muted uppercase font-bold">
               <span>Secure</span>
               <span class="text-error">Anomalous Spike</span>
            </div>
         </div>
      </div>
    </div>
  </div>
</PageLayout>
