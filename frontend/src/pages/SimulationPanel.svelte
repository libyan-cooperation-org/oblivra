<!--
  OBLIVRA — Simulation Panel (Svelte 5)
  Orchestrating autonomous adversary simulations and resilience drills.
-->
<script lang="ts">
  import { KPI, PageLayout, Badge, Button, DataTable } from '@components/ui';
  import { Skull, Zap, Play, Activity, Target, ShieldCheck, RefreshCw, Layers } from 'lucide-svelte';
  import { appStore } from '@lib/stores/app.svelte';

  const simulations = [
    { id: 'S-201', name: 'Ransomware Beaconing', agent: 'L-01', state: 'active', coverage: '94%' },
    { id: 'S-202', name: 'Credential Harvesting', agent: 'M-04', state: 'standby', coverage: '82%' },
    { id: 'S-203', name: 'Identity Lateral Mov', agent: 'M-02', state: 'complete', coverage: '100%' },
  ];
</script>

<PageLayout title="Simulation Orchestration" subtitle="Adversary emulation: Running autonomous resilience drills and validating containment logic against the OBLIVRA sovereign fleet">
  {#snippet toolbar()}
     <Button variant="secondary" size="sm">Recalibrate Tactics</Button>
     <Button variant="primary" size="sm" icon="💀">Deploy Scenario</Button>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
      <KPI title="Active Drills" value="1" trend="Nominal" />
      <KPI title="Logic Validation" value="98%" trend="Verified" variant="success" />
      <KPI title="Simulation Depth" value="L9" trend="Advanced" variant="accent" />
      <KPI title="Fleet Coverage" value="100%" trend="Optimal" variant="success" />
    </div>

    <div class="flex-1 min-h-0 grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Simulation Registry -->
      <div class="lg:col-span-2 bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col shadow-premium">
         <div class="p-3 bg-surface-2 border-b border-border-primary flex justify-between items-center text-[10px] font-bold uppercase tracking-widest text-text-muted">
            Adversary Emulation Ledger
         </div>
         <div class="flex-1 overflow-auto">
            <DataTable data={simulations} columns={[
              { key: 'name', label: 'Scenario Identity' },
              { key: 'agent', label: 'Emu-Agent', width: '100px' },
              { key: 'coverage', label: 'Logic Coverage', width: '120px' },
              { key: 'state', label: 'Runtime', width: '100px' },
              { key: 'action', label: '', width: '80px' }
            ]} density="compact">
              {#snippet cell({ column, row })}
                {#if column.key === 'state'}
                   <Badge variant={row.state === 'active' ? 'accent' : row.state === 'complete' ? 'success' : 'secondary'} dot={row.state === 'active'}>{row.state.toUpperCase()}</Badge>
                {:else if column.key === 'name'}
                   <div class="flex items-center gap-2">
                      <Layers size={14} class="text-accent opacity-70" />
                      <span class="text-[11px] font-bold text-text-heading">{row.name}</span>
                   </div>
                {:else if column.key === 'action'}
                   <div class="flex gap-2">
                      <Button variant="ghost" size="xs"><Play size={12} /></Button>
                      <Button variant="ghost" size="xs"><RefreshCw size={12} /></Button>
                   </div>
                {:else}
                  <span class="text-[11px] text-text-secondary">{row[column.key]}</span>
                {/if}
              {/snippet}
            </DataTable>
         </div>
      </div>

      <!-- Simulation Insights -->
      <div class="flex flex-col gap-6">
         <div class="bg-surface-1 border border-border-primary rounded-md p-6 flex flex-col items-center justify-center text-center gap-4 relative overflow-hidden group border-dashed shadow-sm">
            <Target size={48} class="text-error opacity-40 animate-pulse" />
            <div class="relative z-10">
               <h4 class="text-xs font-bold text-text-heading uppercase tracking-widest">Resilience Threshold</h4>
               <p class="text-[10px] text-text-muted mt-2 max-w-[150px]">OBLIVRA validates your SOAR logic by emulating advanced TTPs in isolated memory blocks.</p>
            </div>
         </div>

         <div class="flex-1 bg-surface-1 border border-border-primary rounded-md p-4 space-y-4">
            <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border-primary pb-2 flex items-center gap-2">
               <Activity size={12} />
               Tactical Success Velocity
            </div>
            <div class="flex-1 h-32 flex items-end justify-between px-2 gap-1 font-mono">
               {#each Array(10) as _, i}
                  <div class="flex-1 bg-accent/20 rounded-t-sm border-x border-t border-accent/10" style="height: {30 + Math.random() * 60}%"></div>
               {/each}
            </div>
         </div>
      </div>
    </div>
  </div>
</PageLayout>
