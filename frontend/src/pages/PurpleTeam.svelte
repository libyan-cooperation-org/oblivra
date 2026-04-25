<script lang="ts">
  import { KPI, PageLayout, Badge, Button, DataTable, PopOutButton} from '@components/ui';
  import { Sword, Activity, Skull } from 'lucide-svelte';

  const simulations = $state<any[]>([]);

  const columns = [
    { key: 'name', label: 'Tactical Mission / TTP' },
    { key: 'status', label: 'Runtime', width: '100px' },
    { key: 'coverage', label: 'Success Rate', width: '120px' },
    { key: 'action', label: '', width: '80px' },
  ];
</script>

<PageLayout title="Purple Team Ops" subtitle="Collaborative adversary simulation and detection engineering verification">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm">MITRE Topology</Button>
      <Button variant="primary" size="sm" icon="🔥">Deploy Emulation</Button>
    </div>
      <PopOutButton route="/purple-team" title="Purple Team" />
    {/snippet}

  <div class="flex flex-col h-full gap-6">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 shrink-0">
      <KPI label="Active Emulations" value={simulations.filter(s => s.status === 'running').length} trend="stable" trendValue="Nominal" variant="accent" />
      <KPI label="Detection Coverage" value="0%" trend="stable" trendValue="PENDING" variant="success" />
      <KPI label="Atomic Scenarios" value="0" trend="stable" trendValue="Verified" variant="success" />
      <KPI label="Logic Drift" value="Zero" trend="stable" trendValue="Hardened" variant="success" />
    </div>

    <div class="flex-1 min-h-0 grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Simulation Index -->
      <div class="lg:col-span-2 bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col shadow-card">
         <div class="p-3 bg-surface-2 border-b border-border-primary text-[10px] font-bold uppercase tracking-widest text-text-muted font-mono">
            Platform Adversary Emulation Registry
         </div>
         <div class="flex-1 overflow-auto">
            {#if simulations.length === 0}
               <div class="flex flex-col items-center justify-center h-full opacity-20 py-24 gap-4">
                  <Skull size={48} />
                  <span class="text-[10px] font-mono font-bold uppercase tracking-[0.2em]">No simulations orchestrated</span>
               </div>
            {:else}
               <DataTable data={simulations} {columns} compact>
                 {#snippet render()}
                   <!-- Render logic -->
                 {/snippet}
               </DataTable>
            {/if}
         </div>
      </div>

      <!-- Simulation Controller -->
      <div class="flex flex-col gap-6">
         <div class="bg-surface-1 border border-border-primary rounded-md p-6 flex flex-col items-center justify-center text-center gap-4 relative overflow-hidden group shadow-card hover:border-error/40 transition-all">
            <div class="absolute inset-0 bg-error/5 group-hover:bg-error/10 transition-colors pointer-events-none"></div>
            <Sword size={48} class="text-error relative z-10 group-hover:scale-110 transition-transform duration-500" />
            <div class="relative z-10">
               <h4 class="text-lg font-bold text-text-heading tracking-tight uppercase">Red-Cell Engagement</h4>
               <p class="text-[9px] text-text-muted mt-1 uppercase tracking-widest font-bold opacity-60">Authorize offensive logic deployment</p>
            </div>
            <Button variant="danger" class="w-full relative z-10">ARM OFFENSIVE SHELL</Button>
         </div>

         <div class="flex-1 bg-surface-1 border border-border-primary rounded-md p-5 flex flex-col gap-4 shadow-card">
            <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border-primary pb-3 flex items-center justify-between">
               <span>Blue-Cell Feedback Loop</span>
               <Badge variant="success" size="xs">SECURE</Badge>
            </div>
            <div class="flex-1 flex flex-col items-center justify-center opacity-10 gap-4">
               <Activity size={32} />
               <span class="text-[9px] font-mono font-bold uppercase tracking-widest">Awaiting engagement signals</span>
            </div>
         </div>
      </div>
    </div>
  </div>
</PageLayout>
