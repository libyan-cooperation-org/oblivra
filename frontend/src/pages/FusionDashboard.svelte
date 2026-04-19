<!--
  OBLIVRA — Fusion Dashboard (Svelte 5)
  Strategic campaign orchestration and multi-layer correlation.
-->
<script lang="ts">
  import { PageLayout, Badge, Button } from '@components/ui';
  import { Layers, Target, Filter } from 'lucide-svelte';

  const campaigns = [
    { id: 'FC-2026-004', name: 'FIN_EXFIL_TEMPEST', severity: 'critical', confidence: 98, stages: 4, assets: 12, actor: 'SILVER_VOXEL' },
    { id: 'FC-2026-003', name: 'RANSOM_PREP_BETA', severity: 'high', confidence: 82, stages: 2, assets: 42, actor: 'COBALT_MIRROR' },
    { id: 'FC-2026-002', name: 'AD_PERSISTENCE_CHART', severity: 'medium', confidence: 44, stages: 1, assets: 2, actor: 'NEON_TEMPEST' }
  ];

  let selectedCampaign = $state(campaigns[0]);
</script>

<PageLayout title="Fusion Intelligence" subtitle="Mapping tactical telemetry to strategic mission objectives">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm" icon={Filter}>LAYER FILTER</Button>
      <Button variant="primary" size="sm">NEW CAMPAIGN</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-0 -m-6">
    <!-- METRIC STRIP -->
    <div class="grid grid-cols-4 gap-px bg-border-primary border-b border-border-primary shrink-0">
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Active Clusters</div>
            <div class="text-xl font-mono font-bold text-error">14</div>
            <div class="text-[9px] text-error mt-1 animate-pulse">▲ High Density Detect</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Correlation Depth</div>
            <div class="text-xl font-mono font-bold text-accent">L4+</div>
            <div class="text-[9px] text-success mt-1">Cross-Platform sync</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Avg Confidence</div>
            <div class="text-xl font-mono font-bold text-text-heading">88%</div>
            <div class="text-[9px] text-success mt-1">Sovereign-grade ML</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Time to Correlate</div>
            <div class="text-xl font-mono font-bold text-text-heading">12ms</div>
            <div class="text-[9px] text-text-muted mt-1">Near real-time</div>
        </div>
    </div>

    <!-- MAIN BODY -->
    <div class="flex-1 flex min-h-0">
        <!-- LEFT: CAMPAIGN LIST -->
        <div class="w-96 bg-surface-2 border-r border-border-primary flex flex-col shrink-0 p-4 gap-4 overflow-auto">
            <div class="space-y-4">
                <span class="text-[9px] font-mono font-bold text-text-muted uppercase tracking-widest flex items-center gap-2">
                    <Layers size={12} class="text-accent" />
                    Active Threat Campaigns
                </span>
                {#each campaigns as campaign}
                    <button 
                        class="w-full text-left bg-surface-1 border {selectedCampaign.id === campaign.id ? 'border-accent' : 'border-border-primary'} p-4 rounded-sm space-y-4 hover:border-accent transition-all cursor-pointer group relative overflow-hidden"
                        onclick={() => selectedCampaign = campaign}
                    >
                        <div class="flex items-start justify-between">
                            <div class="flex flex-col">
                                <span class="text-[12px] font-bold text-text-heading tracking-tight leading-tight">{campaign.name}</span>
                                <span class="text-[8px] font-mono text-text-muted uppercase">{campaign.id} · {campaign.actor}</span>
                            </div>
                            <Badge variant={campaign.severity === 'critical' ? 'critical' : 'warning'} size="xs" class="px-2">
                                {campaign.severity.toUpperCase()}
                            </Badge>
                        </div>
                        <div class="grid grid-cols-2 gap-4">
                            <div class="space-y-1">
                                <span class="text-[8px] font-mono text-text-muted uppercase">Confidence</span>
                                <div class="flex items-center gap-2">
                                    <div class="h-1 flex-1 bg-surface-3 rounded-full overflow-hidden">
                                        <div class="h-full bg-accent" style="width: {campaign.confidence}%"></div>
                                    </div>
                                    <span class="text-[9px] font-mono text-text-secondary">{campaign.confidence}%</span>
                                </div>
                            </div>
                            <div class="flex justify-between items-end pb-0.5">
                                <div class="flex flex-col">
                                    <span class="text-[8px] font-mono text-text-muted uppercase tracking-tight">Assets</span>
                                    <span class="text-[10px] font-mono font-bold text-text-heading">{campaign.assets}</span>
                                </div>
                                <div class="flex flex-col items-end">
                                    <span class="text-[8px] font-mono text-text-muted uppercase tracking-tight">Stage</span>
                                    <span class="text-[10px] font-mono font-bold text-error">{campaign.stages}/5</span>
                                </div>
                            </div>
                        </div>
                        {#if selectedCampaign.id === campaign.id}
                            <div class="absolute inset-y-0 right-0 w-1 bg-accent"></div>
                        {/if}
                    </button>
                {/each}
            </div>
        </div>

        <!-- RIGHT: CAMPAIGN ORCHESTRATION -->
        <div class="flex-1 flex flex-col min-w-0">
            <div class="bg-surface-1 border-b border-border-primary p-3 flex items-center justify-between shrink-0">
                <div class="flex items-center gap-2">
                    <Target size={14} class="text-error" />
                    <span class="text-[10px] font-mono font-bold uppercase tracking-widest text-text-heading">Correlation Analysis: {selectedCampaign.name}</span>
                </div>
                <div class="flex gap-2">
                    <Button variant="secondary" size="xs">VIEW GRAPH</Button>
                    <Button variant="danger" size="xs">TERMINATE FLOWS</Button>
                </div>
            </div>

            <div class="flex-1 overflow-auto mask-fade-bottom p-6 space-y-8">
                <!-- ATTACK TIMELINE -->
                <div class="space-y-4">
                    <span class="text-[9px] font-mono font-bold text-text-muted uppercase tracking-widest">Multi-Stage Propagation Path</span>
                    <div class="relative pl-8 space-y-8 border-l border-border-primary ml-2">
                        {#each [
                            { stage: 'Reconnaissance', time: '10:12:04', desc: 'Scan detected from 104.1.2.4 on public ingress', status: 'completed', icon: 'search' },
                            { stage: 'Initial Access', time: '10:14:22', desc: 'Spear-phishing payload executed on WRK-FIN-09', status: 'completed', icon: 'zap' },
                            { stage: 'Persistence', time: '10:15:00', desc: 'Scheduled task "SystemUpdate" created on domain controller', status: 'completed', icon: 'shield' },
                            { stage: 'Lateral Movement', time: '10:42:15', desc: 'WMI execution detected on SRV-SQL-PROD', status: 'active', icon: 'share-2' },
                            { stage: 'Exfiltration', time: 'TBD', desc: 'Staging data for external transfer', status: 'pending', icon: 'wifi-off' }
                        ] as item}
                            <div class="relative">
                                <div class="absolute -left-10 w-4 h-4 rounded-full border-2 {item.status === 'completed' ? 'bg-success border-success' : item.status === 'active' ? 'bg-error border-error animate-pulse' : 'bg-surface-2 border-border-primary'} flex items-center justify-center z-10">
                                    {#if item.status === 'completed'}
                                        <div class="w-1.5 h-1.5 rounded-full bg-white"></div>
                                    {/if}
                                </div>
                                <div class="flex flex-col gap-1">
                                    <div class="flex items-center gap-2">
                                        <span class="text-[11px] font-bold {item.status === 'active' ? 'text-error' : 'text-text-heading'} uppercase tracking-tight">{item.stage}</span>
                                        <span class="text-[8px] font-mono text-text-muted">{item.time}</span>
                                        {#if item.status === 'active'}
                                            <Badge variant="critical" size="xs" class="animate-sla-blink">CRITICAL_ACTION</Badge>
                                        {/if}
                                    </div>
                                    <p class="text-[10px] font-mono text-text-secondary max-w-lg">{item.desc}</p>
                                </div>
                            </div>
                        {/each}
                    </div>
                </div>

                <!-- LINKED ASSETS -->
                <div class="space-y-4">
                    <span class="text-[9px] font-mono font-bold text-text-muted uppercase tracking-widest">Compromised Asset Mesh</span>
                    <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
                        {#each ['WRK-FIN-09', 'SRV-AD-DC01', 'SRV-SQL-PROD'] as host, i}
                            <div class="bg-surface-2 border border-error/20 p-3 rounded-sm flex items-center justify-between group hover:border-error transition-colors">
                                <div class="flex flex-col">
                                    <span class="text-[10px] font-bold text-text-heading uppercase tracking-tighter">{host}</span>
                                    <span class="text-[8px] font-mono text-text-muted">10.18.2.{44+i}</span>
                                </div>
                                <Button variant="ghost" size="xs" class="opacity-0 group-hover:opacity-100 h-6 px-1.5">ISOLATE</Button>
                            </div>
                        {/each}
                    </div>
                </div>
            </div>
        </div>
    </div>

    <!-- STATUS BAR -->
    <div class="bg-surface-2 border-t border-border-primary px-3 py-1 flex items-center gap-4 text-[8px] font-mono text-text-muted shrink-0">
        <div class="flex items-center gap-1.5">
            <span>FUSION:</span>
            <span class="text-success font-bold">READY</span>
        </div>
        <span class="text-border-primary">|</span>
        <div class="flex items-center gap-1.5">
            <span>LAYERS:</span>
            <span class="text-accent font-bold">EDR, SIEM, NDR, UEBA</span>
        </div>
        <span class="text-border-primary">|</span>
        <div class="flex items-center gap-1.5">
            <span>ORCHESTRATION:</span>
            <span class="text-success font-bold">AUTO</span>
        </div>
        <div class="ml-auto uppercase tracking-widest opacity-60">FUSION_ENGINE v2.4.1</div>
    </div>
  </div>
</PageLayout>

<style>
  .overflow-auto {
    mask-image: linear-gradient(to bottom, transparent 0px, black 12px, black calc(100% - 16px), transparent 100%);
  }
</style>
