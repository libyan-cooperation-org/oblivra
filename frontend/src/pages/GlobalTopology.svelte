<!--
  OBLIVRA — Global Topology (Svelte 5)
  Macro-level platform architecture visualization: Mapping distributed mesh nodes and cross-border data flows.
-->
<script lang="ts">
  import { KPI, PageLayout, Badge, Button } from '@components/ui';
  import { Shield, Globe, Activity, Share2, Server } from 'lucide-svelte';
  import { appStore } from '@lib/stores/app.svelte';

  const regions = [
    { name: 'North America', nodes: 42, health: 'optimal' },
    { name: 'Europe Central', nodes: 28, health: 'optimal' },
    { name: 'Asia Pacific', nodes: 15, health: 'warning' },
  ];
</script>

<PageLayout title="Global Topology" subtitle="Distributed mesh orchestration: Visualizing the macro-level relational architecture of the OBLIVRA sovereign fleet">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm">Relocate Mesh Center</Button>
      <Button variant="primary" size="sm" icon="🌐">Broadcast Global Pulse</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
      <KPI label="Global Nodes" value="85" trend="up" trendValue="+3 New" variant="success" />
      <KPI label="Mesh Latency" value="142ms" trend="stable" trendValue="P99" />
      <KPI label="Data Gravity" value="High" trend="stable" trendValue="Regional" variant="accent" />
      <KPI label="Tunnel Coverage" value="100%" trend="stable" trendValue="E2EE" variant="success" />
    </div>

    <div class="flex-1 min-h-0 grid grid-cols-1 lg:grid-cols-4 gap-6">
       <!-- Global Mesh Visualization (Mock) -->
       <div class="lg:col-span-3 bg-surface-1 border border-border-primary rounded-md relative overflow-hidden flex flex-col shadow-premium group">
          <div class="absolute inset-0 opacity-[0.05] pointer-events-none grayscale flex items-center justify-center">
             <Globe size={600} class="animate-slow-spin" />
          </div>
          
          <div class="p-3 bg-surface-2 border-b border-border-primary flex justify-between items-center text-[10px] font-bold uppercase tracking-widest text-text-muted font-mono">
             Live Sovereign Fleet Mesh (Macro)
          </div>
          
          <div class="flex-1 relative p-12 overflow-auto grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 content-start">
             {#each regions as region}
                <div class="p-6 bg-surface-2/50 backdrop-blur-md border border-border-primary rounded-md flex flex-col gap-4 group/card hover:border-accent transition-all cursor-pointer">
                   <div class="flex justify-between items-start">
                      <div class="flex flex-col">
                         <span class="text-xs font-bold text-text-heading group-hover/card:text-accent transition-colors">{region.name}</span>
                         <span class="text-[10px] text-text-muted uppercase tracking-widest font-bold">Region Cluster</span>
                      </div>
                      <Badge variant={region.health === 'optimal' ? 'success' : 'warning'} size="xs">{region.health.toUpperCase()}</Badge>
                   </div>
                   
                   <div class="flex justify-between items-end">
                      <div class="flex flex-col">
                         <span class="text-2xl font-bold font-mono text-text-heading">{region.nodes}</span>
                         <span class="text-[9px] text-text-muted uppercase font-bold">Active Nodes</span>
                      </div>
                      <Server size={32} class="text-text-muted opacity-20 group-hover/card:opacity-50 transition-all" />
                   </div>

                   <div class="mt-2 h-1 bg-surface-3 rounded-full overflow-hidden">
                      <div class="h-full bg-accent animate-pulse" style="width: {region.health === 'optimal' ? '100%' : '60%'}"></div>
                   </div>
                </div>
             {/each}
          </div>
       </div>

       <!-- Topology Intelligence -->
       <div class="flex flex-col gap-6">
          <div class="bg-surface-1 border border-border-primary rounded-md p-6 flex flex-col items-center justify-center text-center gap-4 relative overflow-hidden border-dashed shadow-sm">
             <Share2 size={32} class="text-accent opacity-40" />
             <h4 class="text-xs font-bold text-text-heading uppercase tracking-widest">Relational Sharding</h4>
             <p class="text-[10px] text-text-muted max-w-[150px]">OBLIVRA shards topology state across regional witnesses to ensure 0-Downtime visibility.</p>
          </div>

          <div class="flex-1 bg-surface-1 border border-border-primary rounded-md p-4 space-y-4">
             <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border-primary pb-2 flex items-center gap-2">
                <Activity size={12} />
                Regional Sink Pressure
             </div>
             <div class="space-y-6">
                {#each regions as region}
                   <div>
                       <div class="flex justify-between text-[9px] mb-1 font-bold">
                          <span class="text-text-muted uppercase tracking-widest">{region.name}</span>
                          <span class="text-text-heading">{Math.floor(Math.random() * 40 + 40)}% Ingest</span>
                       </div>
                       <div class="w-full bg-surface-3 h-1 rounded-full overflow-hidden">
                          <div class="bg-accent h-full" style="width: {Math.floor(Math.random() * 40 + 40)}%"></div>
                       </div>
                   </div>
                {/each}
             </div>
          </div>
       </div>
    </div>
  </div>
</PageLayout>
