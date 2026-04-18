<!--
  OBLIVRA — Fusion Dashboard (Svelte 5)
  Unified mission intelligence: Correlating fleet telemetry with strategic risk.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { KPI, PageLayout, Badge, Button } from '@components/ui';
  import { Shield, Activity, ChevronRight } from 'lucide-svelte';
  import { campaignStore } from '@lib/stores/campaigns.svelte';
  import { GetActiveClusters } from '@wailsjs/github.com/kingknull/oblivrashell/internal/services/graphservice';
  import CampaignTimeline from '@lib/components/forensics/CampaignTimeline.svelte';

  async function refreshCampaigns() {
    try {
      const clusters = await GetActiveClusters();
      campaignStore.setCampaigns(clusters as any);
    } catch (err) {
      console.error("Failed to refresh campaigns:", err);
    }
  }

  onMount(async () => {
    await refreshCampaigns();
  });

  // Calculate global stats from clusters
  const totalCampaigns = $derived($campaignStore.length);
  const avgConfidence = $derived($campaignStore.length > 0 
    ? ($campaignStore.reduce((acc, c) => acc + (c.edge_count > 5 ? 90 : 60), 0) / $campaignStore.length).toFixed(1)
    : "0.0"
  );

  let selectedCampaignID = $state<string | null>(null);

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
    <Button variant="primary" size="sm">
      <Activity size={14} class="mr-2" />
      Synchronize All Layers
    </Button>
  {/snippet}

  <div class="flex h-full gap-6">
    <div class="flex-1 flex flex-col gap-6">
      <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
        <KPI label="Active Campaigns" value={totalCampaigns} trendValue="Live Tracking" variant="accent" />
        <KPI label="Fusion Confidence" value={avgConfidence + "%"} trendValue="Optimal" variant="success" />
        <KPI label="Platform Latency" value="12ms" trendValue="Low" variant="success" />
        <KPI label="Correlation Depth" value="L4" trendValue="Sovereign" variant="success" />
      </div>

      <div class="bg-surface-1 border border-border-primary rounded-lg overflow-hidden flex flex-col min-h-0">
        <div class="p-4 bg-surface-2 border-b border-border-primary flex items-center justify-between">
          <h2 class="text-[11px] font-bold uppercase tracking-widest text-text-heading">Strategic Threat Clusters</h2>
          <Badge variant="info">Real-time Correlation</Badge>
        </div>
        
        <div class="flex-1 overflow-auto p-4">
          <div class="grid grid-cols-1 gap-4">
            {#each $campaignStore as campaign}
              <div 
                class="p-4 bg-surface-2 border border-border-primary rounded-lg hover:border-accent transition-all cursor-pointer group"
                onclick={() => openRecon(campaign.cluster_id)}
                role="button"
                tabindex="0"
                onkeydown={(e) => e.key === 'Enter' && openRecon(campaign.cluster_id)}
              >
                <div class="flex items-center justify-between mb-3">
                  <div class="flex items-center gap-3">
                    <div class="p-2 bg-accent/10 rounded-md">
                      <Shield size={18} class="text-accent" />
                    </div>
                    <div class="flex flex-col">
                      <span class="text-[13px] font-bold text-text-heading">{campaign.cluster_id}</span>
                      <span class="text-[10px] text-text-muted">{campaign.entities.length} nodes involved</span>
                    </div>
                  </div>
                  <ChevronRight size={16} class="text-text-muted group-hover:text-accent transition-colors" />
                </div>

                <div class="grid grid-cols-2 gap-4">
                   <div class="flex flex-col gap-1">
                      <span class="text-[9px] uppercase font-bold text-text-muted">Persistence Vectors</span>
                      <div class="flex flex-wrap gap-1">
                        {#each campaign.tactics as tactic}
                          <Badge variant="muted" size="sm">{tactic}</Badge>
                        {/each}
                      </div>
                   </div>
                   <div class="flex flex-col gap-1 items-end">
                      <span class="text-[9px] uppercase font-bold text-text-muted">Attack Path Density</span>
                      <div class="flex items-center gap-2">
                        <div class="w-24 h-1.5 bg-surface-3 rounded-full overflow-hidden">
                           <div class="h-full bg-accent" style="width: {Math.min(campaign.edge_count * 10, 100)}%"></div>
                        </div>
                        <span class="text-[10px] font-bold text-text-heading">{campaign.edge_count} paths</span>
                      </div>
                   </div>
                </div>
              </div>
            {/each}
          </div>
        </div>
      </div>
    </div>

    {#if selectedCampaignID}
      <div class="w-96 bg-surface-1 border-l border-border-primary p-6 flex flex-col gap-6 shadow-2xl">
        <div class="flex items-center justify-between">
          <h2 class="text-[12px] font-bold uppercase tracking-widest text-text-heading">Campaign Analysis</h2>
          <Button variant="ghost" size="sm" onclick={closeRecon}>Close</Button>
        </div>
        
        <CampaignTimeline clusterID={selectedCampaignID} onClose={closeRecon} />
      </div>
    {/if}
  </div>
</PageLayout>
