<!--
  OBLIVRA — Lineage Explorer (Svelte 5)
  Data provenance and forensic lineage: Tracing the origin and transformation of security telemetry.
-->
<script lang="ts">
  import { KPI, PageLayout, Badge, Button, Chart } from '@components/ui';
  import { Network, Share2, Activity, Zap, Database, Search, ShieldCheck } from 'lucide-svelte';
  import { appStore } from '@lib/stores/app.svelte';

  const lineageNodes = [
    { id: 'L-01', type: 'Ingress', source: 'edge-gw-01', time: '12:00:01' },
    { id: 'L-02', type: 'Transform', logic: 'Entropy-Filter', time: '12:00:02' },
    { id: 'L-03', type: 'Enrich', logic: 'Geo-IP', time: '12:00:03' },
    { id: 'L-04', type: 'Persistence', sink: 'BadgerDB', time: '12:00:04' },
  ];
</script>

<PageLayout title="Data Lineage Explorer" subtitle="Forensic provenance tracking: Tracing telemetry from raw ingestion to persistent state">
  {#snippet toolbar()}
     <Button variant="secondary" size="sm">Trace New Hash</Button>
     <Button variant="primary" size="sm" icon="🔍">Provenance Audit</Button>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
      <KPI title="Tracked Events" value="1.4M" trend="Historical" />
      <KPI title="Lineage Depth" value="L12" trend="Millisecond" variant="accent" />
      <KPI title="Integrity Proof" value="VERIFIED" trend="Signed" variant="success" />
      <KPI title="Audit Stability" value="100%" trend="Optimal" variant="success" />
    </div>

    <div class="flex-1 min-h-0 grid grid-cols-1 lg:grid-cols-4 gap-6">
       <!-- Lineage Visualization (Mock) -->
       <div class="lg:col-span-3 bg-surface-1 border border-border-primary rounded-md relative overflow-hidden flex flex-col shadow-premium group">
          <div class="absolute inset-0 opacity-[0.03] pointer-events-none grayscale flex items-center justify-center">
             <Share2 size={600} />
          </div>
          
          <div class="p-3 bg-surface-2 border-b border-border-primary flex justify-between items-center text-[10px] font-bold uppercase tracking-widest text-text-muted">
             Evidence Provenance Chain
          </div>
          
          <div class="flex-1 relative p-10 flex flex-col justify-center items-center gap-6">
             {#each lineageNodes as node, i}
                <div class="flex flex-col items-center">
                   <div class="p-4 bg-surface-2 border border-border-primary rounded-md shadow-lg flex items-center gap-4 group-hover:scale-105 transition-transform cursor-pointer relative z-10 min-w-[200px]">
                      <div class="w-10 h-10 rounded-full bg-accent/20 flex items-center justify-center border border-accent/40 shrink-0">
                         {#if node.type === 'Ingress'}<Database size={16} class="text-accent" />{/if}
                         {#if node.type === 'Transform'}<Zap size={16} class="text-accent" />{/if}
                         {#if node.type === 'Enrich'}<Search size={16} class="text-accent" />{/if}
                         {#if node.type === 'Persistence'}<ShieldCheck size={16} class="text-accent" />{/if}
                      </div>
                      <div class="flex flex-col">
                         <span class="text-[9px] font-bold text-accent uppercase tracking-widest">{node.type}</span>
                         <span class="text-[11px] font-bold text-text-heading">{node.source || node.logic || node.sink}</span>
                         <span class="text-[8px] font-mono text-text-muted">{node.time}</span>
                      </div>
                   </div>
                   {#if i < lineageNodes.length - 1}
                      <div class="h-8 w-px bg-border-primary relative">
                         <div class="absolute bottom-0 left-1/2 -translate-x-1/2 w-1.5 h-1.5 rounded-full bg-accent"></div>
                      </div>
                   {/if}
                </div>
             {/each}
          </div>
       </div>

       <!-- Lineage Metadata -->
       <div class="flex flex-col gap-6">
          <div class="bg-surface-1 border border-border-primary rounded-md p-6 flex flex-col items-center justify-center text-center gap-3 border-dashed shadow-sm">
             <Network size={32} class="text-accent opacity-40" />
             <h4 class="text-xs font-bold text-text-heading uppercase tracking-widest">Temporal Consistency</h4>
             <p class="text-[10px] text-text-muted max-w-[180px]">Each transition in the lineage chain is cryptographically hashed and linked to the previous state block.</p>
          </div>

          <div class="flex-1 bg-surface-1 border border-border-primary rounded-md p-4 space-y-4">
             <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border-primary pb-2 flex items-center gap-2">
                <Activity size={12} />
                Transformation Entropy
             </div>
             <div class="space-y-4">
                {#each Array(3) as _, i}
                   <div class="flex justify-between items-center text-[10px]">
                      <span class="text-text-secondary">Logic Block {i+1}</span>
                      <span class="font-bold text-success">NOMINAL</span>
                   </div>
                {/each}
             </div>
          </div>
       </div>
    </div>
  </div>
</PageLayout>
