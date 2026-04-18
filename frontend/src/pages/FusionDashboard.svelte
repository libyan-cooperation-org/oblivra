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

<script lang="ts">
  import { onMount } from 'svelte';
  import { KPI, PageLayout, Badge, Button } from '@components/ui';
  import { Shield, Target, Activity, Layers, Cpu, Radio, ChevronRight, User, Monitor } from 'lucide-svelte';
  import { campaignStore } from '@lib/stores/campaigns.svelte';
  import { GetActiveClusters } from '@wailsjs/github.com/kingknull/oblivrashell/internal/services/graphservice';
  import CampaignTimeline from '@lib/components/forensics/CampaignTimeline.svelte';

  async function refreshCampaigns() {
    try {
      const clusters = await GetActiveClusters();
      campaignStore.setCampaigns(clusters);
    } catch (err) {
      console.error('[fusion] Failed to fetch campaigns:', err);
    }
  }

  onMount(() => {
    refreshCampaigns();
    const interval = setInterval(refreshCampaigns, 5000);
    return () => clearInterval(interval);
  });

  // Calculate global stats from clusters
  const totalCampaigns = $derived($campaignStore.length);
  const avgConfidence = $derived($campaignStore.length > 0 
    ? ($campaignStore.reduce((acc, c) => acc + (c.edge_count > 5 ? 90 : 60), 0) / $campaignStore.length).toFixed(1)
    : "0.0"
  );

  let selectedCampaignID = $state(null);

  function openRecon(id: string) {
    selectedCampaignID = id;
  }

  function closeRecon() {
    selectedCampaignID = null;
  }
</script>

<PageLayout title="Fusion Intelligence" subtitle="Unified platform correlation: Mapping tactical telemetry to strategic mission objectives">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" onclick={refreshCampaigns}>Recalibrate Fusion</Button>
    <Button variant="primary" size="sm" icon="📡">Synchronize All Layers</Button>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
      <KPI label="Active Campaigns" value={totalCampaigns} trendValue="Live Tracking" variant="accent" />
      <KPI label="Fusion Confidence" value={avgConfidence + "%"} trendValue="Optimal" variant="success" />
      <KPI label="Platform Latency" value="12ms" trendValue="Low" variant="success" />
      <KPI label="Correlation Depth" value="L4" trendValue="Sovereign" variant="success" />
    </div>

    <div class="flex-1 min-h-0 grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Active Threat Campaigns -->
      <div class="lg:col-span-2 bg-surface-1 border border-border-primary rounded-md p-6 flex flex-col shadow-card gap-4 overflow-hidden">
         <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border-primary pb-2 flex justify-between items-center">
            Active Threat Campaigns
            <div class="flex items-center gap-2">
               <div class="w-1.5 h-1.5 bg-green-400 rounded-full animate-pulse"></div>
               <span class="text-[9px] font-mono text-green-400">CORRELATION ENGINE ACTIVE</span>
            </div>
         </div>
         
         <div class="flex-1 overflow-y-auto pr-1 space-y-3">
            {#if $campaignStore.length === 0}
               <div class="h-full flex flex-col items-center justify-center text-center opacity-40 py-12">
                  <Shield class="w-12 h-12 text-slate-600 mb-4" />
                  <div class="text-xs font-bold text-slate-500 uppercase tracking-widest">No Active Campaigns</div>
                  <div class="text-[10px] font-mono text-slate-600 mt-2">PLATFORM SIGNAL IS NOMINAL. NO MULTI-STAGE CORRELATIONS DETECTED.</div>
               </div>
            {:else}
               {#each $campaignStore as campaign}
                  <div class="p-4 bg-surface-2 border border-border-secondary rounded-md flex flex-col gap-3 group hover:border-accent/40 transition-all">
                     <div class="flex justify-between items-start">
                        <div class="flex flex-col gap-1">
                           <span class="text-xs font-bold text-text-heading font-mono">{campaign.cluster_id.substring(0, 24)}...</span>
                           <div class="flex items-center gap-2">
                              {#each campaign.entities.slice(0, 3) as entity}
                                 <Badge variant="info" size="xs" class="font-mono text-[8px]">{entity.split(':').pop()}</Badge>
                              {/each}
                              {#if campaign.entities.length > 3}
                                 <span class="text-[8px] text-text-muted">+{campaign.entities.length - 3} more</span>
                              {/if}
                           </div>
                        </div>
                        <div class="flex flex-col items-end gap-1">
                           <Badge variant={campaign.edge_count > 10 ? 'critical' : 'warning'} size="xs">
                              {campaign.edge_count > 10 ? 'CRITICAL' : 'SUSPICIOUS'}
                           </Badge>
                           <span class="text-[9px] font-mono text-text-muted uppercase">Edges: {campaign.edge_count}</span>
                        </div>
                     </div>
                     
                     <div class="flex items-center gap-2 mt-1">
                        {#each campaign.tactics as tactic}
                           <div class="px-1.5 py-0.5 bg-accent/10 border border-accent/20 rounded text-[9px] font-mono text-accent">
                              {tactic}
                           </div>
                        {/each}
                     </div>

                     <div class="flex justify-between items-center pt-2 border-t border-border-primary/50">
                        <div class="flex items-center gap-4">
                           <div class="flex flex-col">
                              <span class="text-[8px] text-text-muted uppercase font-bold">First Seen</span>
                              <span class="text-[9px] font-mono text-text-heading">{new Date(campaign.first_seen).toLocaleTimeString()}</span>
                           </div>
                           <div class="flex flex-col">
                              <span class="text-[8px] text-text-muted uppercase font-bold">Last Activity</span>
                              <span class="text-[9px] font-mono text-text-heading">{new Date(campaign.last_seen).toLocaleTimeString()}</span>
                           </div>
                        </div>
                        <button 
                           class="p-1.5 bg-surface-3 hover:bg-accent hover:text-white rounded transition-all"
                           onclick={() => openRecon(campaign.cluster_id)}
                        >
                           <ChevronRight size={14} />
                        </button>
                     </div>
                  </div>
               {/each}
            {/if}
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
                  <div class="flex-1 bg-accent/20 rounded-t-sm border-x border-t border-accent/10" style="height: {20 + Math.sin(i * 0.8 + (totalCampaigns * 0.5)) * 30 + 40}%"></div>
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

  {#if selectedCampaignID}
     <CampaignTimeline clusterID={selectedCampaignID} onClose={closeRecon} />
  {/if}
</PageLayout>
