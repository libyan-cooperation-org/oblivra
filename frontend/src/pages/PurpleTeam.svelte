<!--
  OBLIVRA — Purple Team (Svelte 5)
  Adversary Simulation and Collaborative Defense Orchestration.
-->
<script lang="ts">
  import { KPI, PageLayout, Badge, Button, DataTable } from '@components/ui';
  import { Sword, Shield, Target, Activity, Zap, Play, Skull, Crosshair, Lock } from 'lucide-svelte';
  import { appStore } from '@lib/stores/app.svelte';

  const simulations = [
    { id: 'S-701', name: 'T1059.001 - PowerShell Execution', status: 'running', coverage: '82%', drift: 'Low' },
    { id: 'S-702', name: 'T1566.001 - Spearphishing Attachment', status: 'completed', coverage: '100%', drift: 'Zero' },
    { id: 'S-703', name: 'T1003.001 - LSASS Memory Dumping', status: 'blocked', coverage: '45%', drift: 'Critical' },
  ];
</script>

<PageLayout title="Purple Team Ops" subtitle="Collaborative adversary simulation and detection engineering verification: Bridging offensive testing and defensive posture">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm">MITRE Topology</Button>
      <Button variant="primary" size="sm" icon="🔥">Deploy Emulation</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <!-- Pulse Stats -->
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 shrink-0">
      <KPI label="Active Emulations" value={simulations.filter(s => s.status === 'running').length} trend="stable" trendValue="Prioritized" variant="accent" />
      <KPI label="Detection Coverage" value="88.2%" trend="up" trendValue="+4.1%" variant="success" />
      <KPI label="Atomic Scenarios" value="1,422" trend="stable" trendValue="Verified" variant="success" />
      <KPI label="Logic Drift" value="Minimal" trend="stable" trendValue="Hardened" variant="success" />
    </div>

    <div class="flex-1 min-h-0 grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Simulation Index -->
      <div class="lg:col-span-2 bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col shadow-premium">
         <div class="p-3 bg-surface-2 border-b border-border-primary flex justify-between items-center text-[10px] font-bold uppercase tracking-widest text-text-muted font-mono">
            Platform Adversary Emulation Registry
         </div>
         <div class="flex-1 overflow-auto">
            <DataTable data={simulations} columns={[
              { key: 'name', label: 'Tactical Mission / TTP' },
              { key: 'status', label: 'Runtime', width: '100px' },
              { key: 'coverage', label: 'Success Rate', width: '120px' },
              { key: 'action', label: '', width: '80px' }
            ]} compact>
              {#snippet render({ value, col, row })}
                {#if col.key === 'status'}
                   <Badge variant={value === 'running' ? 'accent' : value === 'blocked' ? 'critical' : 'success'} dot={value === 'running'}>
                      {value.toUpperCase()}
                   </Badge>
                {:else if col.key === 'coverage'}
                   <div class="flex items-center gap-3">
                      <div class="flex-1 bg-surface-3 h-1 rounded-full overflow-hidden min-w-[50px]">
                         <div class="bg-accent h-full shadow-glow-accent/20" style="width: {value}"></div>
                      </div>
                      <span class="text-[10px] font-mono font-bold text-text-heading">{value}</span>
                   </div>
                {:else if col.key === 'name'}
                   <div class="flex items-center gap-2">
                      <Skull size={14} class="text-error opacity-60" />
                      <span class="text-[11px] font-bold text-text-heading">{value}</span>
                   </div>
                {:else if col.key === 'action'}
                   <div class="flex gap-2">
                      <Button variant="ghost" size="xs"><Play size={12} /></Button>
                      <Button variant="ghost" size="xs"><Activity size={12} /></Button>
                   </div>
                {:else}
                   <span class="text-[11px] text-text-secondary">{value}</span>
                {/if}
              {/snippet}
            </DataTable>
         </div>
      </div>

      <!-- Simulation Controller -->
      <div class="flex flex-col gap-6">
         <div class="bg-surface-1 border border-border-primary rounded-md p-6 flex flex-col items-center justify-center text-center gap-4 relative overflow-hidden group shadow-premium hover:border-error/40 transition-all">
            <div class="absolute inset-0 bg-error/5 group-hover:bg-error/10 transition-colors pointer-events-none"></div>
            <Sword size={48} class="text-error relative z-10 group-hover:scale-110 transition-transform duration-500" />
            <div class="relative z-10">
               <h4 class="text-lg font-bold text-text-heading tracking-tight uppercase">Red-Cell Engagement</h4>
               <p class="text-[9px] text-text-muted mt-1 uppercase tracking-widest font-bold opacity-60">Authorize offensive logic deployment</p>
            </div>
            <Button variant="error" class="w-full text-[10px] font-bold py-3 relative z-10 shadow-glow-error/10">ARM OFFENSIVE SHELL</Button>
         </div>

         <div class="flex-1 bg-surface-1 border border-border-primary rounded-md p-5 flex flex-col gap-4 shadow-sm">
            <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border-primary pb-3 flex items-center justify-between">
               <span>Blue-Cell Feedback Loop</span>
               <Badge variant="success" size="xs">SECURE</Badge>
            </div>
            <div class="flex-1 space-y-5 overflow-y-auto pr-2 custom-scrollbar">
               {#each Array(4) as _, i}
                  <div class="flex gap-4 items-start group">
                     <div class="w-2 h-2 rounded-full bg-success mt-1.5 shadow-sm shadow-success/40 group-hover:scale-125 transition-transform"></div>
                     <div class="flex flex-col gap-0.5">
                        <span class="text-[11px] font-bold text-text-heading group-hover:text-success transition-colors">Alert V-{1000 + i*42}: Logic Breach Blocked</span>
                        <div class="flex items-center gap-2 text-[8px] text-text-muted font-mono uppercase tracking-widest font-bold">
                           <span>LATENCY: 12ms</span>
                           <span class="opacity-30">|</span>
                           <span>SVR: MESH-NODE-{i+1}</span>
                        </div>
                     </div>
                  </div>
               {/each}
            </div>
         </div>
      </div>
    </div>
  </div>
</PageLayout>
