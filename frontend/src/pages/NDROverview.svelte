<!--
  OBLIVRA — NDR Overview (Svelte 5)
  Network Detection and Response: Deep packet inspection and traffic flow analysis.
-->
<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { KPI, PageLayout, Badge, Button, DataTable } from '@components/ui';
  import { Shield, Zap, Activity, Crosshair, Network, Router } from 'lucide-svelte';
  import { GetLiveTraffic } from '@wailsjs/github.com/kingknull/oblivrashell/internal/services/ndrservice';
  import { subscribe } from '@lib/bridge';

  let trafficFlows = $state<any[]>([]);
  let loading = $state(true);

  // Compute stats based on the flow array
  let monitoredFlowsCount = $derived(trafficFlows.length);
  let totalBytes = $derived(trafficFlows.reduce((acc, f) => acc + (f.bytes_sent || 0) + (f.bytes_recv || 0), 0));
  let totalBytesStr = $derived((totalBytes / 1024 / 1024).toFixed(2) + ' MB');
  let uniqueDests = $derived(new Set(trafficFlows.map(f => f.dest_ip)).size);

  let unsubFlow: () => void;

  async function loadData() {
    loading = true;
    try {
      const flows = await GetLiveTraffic();
      trafficFlows = (flows || []).reverse(); // Newest first
    } catch (err) {
      console.error('[ndr] failed to load traffic flows:', err);
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    loadData();
    unsubFlow = subscribe('ndr:flow', (flow) => {
      trafficFlows = [flow, ...trafficFlows].slice(0, 1000); 
    });
  });

  onDestroy(() => {
    unsubFlow?.();
  });

  const columns = [
    { key: 'src_ip', label: 'Origin Shard' },
    { key: 'dest_ip', label: 'Exit Node' },
    { key: 'protocol', label: 'Protocol', width: '100px' },
    { key: 'app_name', label: 'Application', width: '120px' },
    { key: 'size', label: 'Transfer Size', width: '120px' },
    { key: 'action', label: '', width: '80px' }
  ];

</script>

<PageLayout title="Network Operations" subtitle="Deep packet inspection and protocol-level traffic analysis: Native NDR orchestration">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm" onclick={loadData}>Refresh Buffer</Button>
      <Button variant="primary" size="sm"><Shield size={14} class="mr-1.5 inline align-middle"/> Engage IDS Filter</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <!-- Pulse Stats -->
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 shrink-0">
      <KPI label="Mesh Throughput" value={totalBytesStr} variant="accent" />
      <KPI label="Monitored Flows" value={monitoredFlowsCount.toString()} variant="success" />
      <KPI label="Unique Endpoints" value={uniqueDests.toString()} variant="info" />
      <KPI label="DPI Status" value="ACTIVE" variant="success" />
    </div>

    <div class="flex-1 min-h-0 grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Traffic Inventory -->
      <div class="lg:col-span-2 bg-slate-900/40 border border-white/5 rounded-lg overflow-hidden flex flex-col shadow-2xl backdrop-blur-md">
         <div class="p-3 bg-white/5 border-b border-white/5 flex justify-between items-center text-[10px] font-bold uppercase tracking-widest text-slate-400 font-mono">
            <span>Live Traffic Flow Analytics (L7 Deep Inspection)</span>
            <div class="flex items-center gap-2 px-2 py-0.5 bg-green-500/10 border border-green-500/20 rounded-full">
                <div class="w-1.5 h-1.5 bg-green-400 rounded-full animate-pulse"></div>
                <span class="text-[10px] font-mono text-green-400 uppercase">Streaming</span>
            </div>
         </div>
         <div class="flex-1 overflow-auto">
            <DataTable data={trafficFlows} {columns} compact>
              {#snippet render({ value, col, row })}
                {#if col.key === 'protocol'}
                   <span class="text-[10px] font-bold text-slate-300 uppercase tracking-widest">{value}</span>
                {:else if col.key === 'app_name'}
                   <Badge variant={value === 'DNS' || value === 'HTTP' ? 'info' : 'muted'}>{value || 'UNKNOWN'}</Badge>
                {:else if col.key === 'src_ip' || col.key === 'dest_ip'}
                   <div class="flex items-center gap-2">
                      <div class="w-1.5 h-1.5 rounded-full bg-blue-500/40"></div>
                      <code class="text-[11px] font-bold text-slate-100">{value}</code>
                   </div>
                {:else if col.key === 'size'}
                   <span class="text-[10px] font-mono whitespace-nowrap">{(((row.bytes_sent || 0) + (row.bytes_recv || 0)) / 1024).toFixed(1)} KB</span>
                {:else if col.key === 'action'}
                   <div class="flex gap-2">
                      <Button variant="ghost" size="xs"><Activity size={12} /></Button>
                      <Button variant="ghost" size="xs"><Crosshair size={12} /></Button>
                   </div>
                {:else}
                  <span class="text-[11px] text-slate-400">{value}</span>
                {/if}
              {/snippet}
            </DataTable>
         </div>
      </div>

      <!-- Protocol Distribution -->
      <div class="flex flex-col gap-6">
         <div class="bg-slate-900/40 border border-white/5 rounded-lg p-6 flex flex-col gap-5 shadow-2xl backdrop-blur-md">
            <div class="text-[10px] font-bold text-slate-400 uppercase tracking-widest border-b border-white/5 pb-3 flex items-center justify-between">
               <span>Tactical Distribution</span>
               <Network size={12} />
            </div>
            <div class="space-y-5">
               <div>
                  <div class="flex justify-between text-[10px] mb-1.5 font-bold uppercase tracking-widest">
                     <span class="text-slate-300">HTTPS/TLS (Sovereign)</span>
                     <span class="text-blue-400">64%</span>
                  </div>
                  <div class="w-full bg-black/40 h-1.5 rounded-full overflow-hidden shadow-inner">
                     <div class="bg-blue-500 h-full shadow-glow-accent/20" style="width: 64%"></div>
                  </div>
               </div>
               <div>
                  <div class="flex justify-between text-[10px] mb-1.5 font-bold uppercase tracking-widest">
                     <span class="text-slate-300">Encrypted P2P Mesh</span>
                     <span class="text-green-400">22%</span>
                  </div>
                  <div class="w-full bg-black/40 h-1.5 rounded-full overflow-hidden shadow-inner">
                     <div class="bg-green-500 h-full" style="width: 22%"></div>
                  </div>
               </div>
               <div>
                  <div class="flex justify-between text-[10px] mb-1.5 font-bold uppercase tracking-widest">
                     <span class="text-slate-300">Unknown/Entropy</span>
                     <span class="text-pink-400">4%</span>
                  </div>
                  <div class="w-full bg-black/40 h-1.5 rounded-full overflow-hidden shadow-inner">
                     <div class="bg-pink-500 h-full" style="width: 4%"></div>
                  </div>
               </div>
            </div>
         </div>

         <div class="flex-1 bg-slate-900/40 border border-white/5 rounded-lg p-8 flex flex-col items-center justify-center text-center gap-4 relative overflow-hidden group shadow-2xl border-dashed hover:border-blue-500/40 transition-all backdrop-blur-md">
            <div class="absolute inset-0 bg-blue-500/5 group-hover:bg-blue-500/10 transition-colors pointer-events-none"></div>
            <Router size={40} class="text-blue-500 relative z-10 opacity-60 group-hover:scale-110 transition-transform duration-500" />
            <div class="relative z-10">
               <h4 class="text-xs font-bold text-white uppercase tracking-widest">Logic Eviction</h4>
               <p class="text-[10px] text-slate-400 mt-2 max-w-[180px] leading-relaxed font-bold opacity-60">
                 DPI logic is analyzing distinct protocols across all mesh shards.
               </p>
            </div>
         </div>
      </div>
    </div>
  </div>
</PageLayout>
