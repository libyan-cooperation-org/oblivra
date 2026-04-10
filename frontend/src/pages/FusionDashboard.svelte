<!--
  OBLIVRA — Fusion Dashboard (Svelte 5)
  Unified mission intelligence: Correlating fleet telemetry with strategic risk.
-->
<script lang="ts">
  import { KPI, PageLayout, Badge, Button, Chart } from '@components/ui';
  import { Shield, Target, Activity, Zap, Layers, Globe, Cpu } from 'lucide-svelte';
  import { appStore } from '@lib/stores/app.svelte';

  const fusionLayers = [
    { name: 'Fleet Telemetry', status: 'synced', density: 'High' },
    { name: 'Adversary Profiles', status: 'synced', density: 'Moderate' },
    { name: 'Cryptographic Audit', status: 'verified', density: 'Critical' },
    { name: 'Network Protocol L7', status: 'warning', density: 'Low' },
  ];
</script>

<PageLayout title="Fusion Intelligence" subtitle="Unified platform correlation: Mapping tactical telemetry to strategic mission objectives">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm">Recalibrate Fusion</Button>
    <Button variant="primary" size="sm" icon="📡">Synchronize All Layers</Button>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
      <KPI title="Correlation Depth" value="L4" trend="Deep Scan" variant="accent" />
      <KPI title="Fusion Confidence" value="98.2%" trend="Optimal" variant="success" />
      <KPI title="Platform Latency" value="12ms" trend="Low" variant="success" />
      <KPI title="Active Witnesses" value="4" trend="Distributed" variant="success" />
    </div>

    <div class="flex-1 min-h-0 grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Fusion Layer Overview -->
      <div class="lg:col-span-2 bg-surface-1 border border-border-primary rounded-md p-6 flex flex-col shadow-premium gap-6">
         <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border-primary pb-2 flex justify-between items-center">
            Mission Correlation Layers
            <Badge variant="info" size="xs">Auto-Syncing</Badge>
         </div>
         
         <div class="flex-1 grid grid-cols-1 md:grid-cols-2 gap-4 content-start">
            {#each fusionLayers as layer}
               <div class="p-4 bg-surface-2 border border-border-secondary rounded-md flex flex-col gap-3 group hover:border-accent transition-colors">
                  <div class="flex justify-between items-center">
                     <span class="text-xs font-bold text-text-heading">{layer.name}</span>
                     <Badge variant={layer.status === 'verified' || layer.status === 'synced' ? 'success' : 'warning'} size="xs">
                        {layer.status.toUpperCase()}
                     </Badge>
                  </div>
                  <div class="flex justify-between items-end">
                     <div class="flex flex-col">
                        <span class="text-[9px] text-text-muted uppercase tracking-widest font-bold">Data Density</span>
                        <span class="text-lg font-bold font-mono text-accent">{layer.density}</span>
                     </div>
                     <Layers class="text-text-muted opacity-20 group-hover:opacity-40 transition-opacity" size={24} />
                  </div>
               </div>
            {/each}
         </div>
      </div>

      <!-- Tactical Visuals -->
      <div class="flex flex-col gap-6">
         <div class="bg-surface-1 border border-border-primary rounded-md p-6 flex flex-col items-center justify-center text-center gap-4 relative overflow-hidden group">
            <Cpu class="text-accent opacity-20 animate-pulse" size={64} />
            <div class="relative z-10">
               <h4 class="text-xs font-bold text-text-heading uppercase tracking-widest">Distributed Synthesis</h4>
               <p class="text-[10px] text-text-muted mt-2">Correlation logic is distributed across the fleet to minimize central processing latency.</p>
            </div>
         </div>

         <div class="flex-1 bg-surface-1 border border-border-primary rounded-md p-4 space-y-4">
            <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border-primary pb-2 flex items-center gap-2">
               <Activity size={12} />
               Platform Signal Entropy
            </div>
            <div class="flex-1 h-32 flex items-end justify-between px-2 gap-1">
               {#each Array(12) as _, i}
                  <div class="flex-1 bg-accent/20 rounded-t-sm border-x border-t border-accent/10" style="height: {20 + Math.random() * 80}%"></div>
               {/each}
            </div>
            <div class="flex justify-between text-[8px] text-text-muted font-bold uppercase">
               <span>Raw Ingest</span>
               <span>Synthesized Output</span>
            </div>
         </div>
      </div>
    </div>
  </div>
</PageLayout>
