<!-- OBLIVRA Web — Fusion Dashboard (Svelte 5) -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { Badge, Button, PageLayout, Spinner } from '@components/ui';
  import { Layers, Target, Activity, Activity as Pulse, ShieldAlert, Share2, ChevronRight } from 'lucide-svelte';
  import { request } from '../services/api';

  // -- Types --
  interface FusionCampaign {
    id: string;
    entities: string[];
    alert_count: number;
    tactic_stages: string[];
    stage_count: number;
    confidence: number;
    first_seen: string;
    last_seen: string;
    status: 'active' | 'closed' | 'investigating';
    kill_chain_progress: number;
  }
  interface KillChainStage {
    tactic_id: string;
    tactic_name: string;
    hit_count: number;
    techniques: string[];
    first_seen?: string;
  }

  // -- Constants --
  const TACTIC_ORDER = [
    'Initial Access', 'Execution', 'Persistence', 'Privilege Escalation',
    'Defense Evasion', 'Credential Access', 'Discovery', 'Lateral Movement',
    'Collection', 'Command & Control', 'Exfiltration', 'Impact',
  ];

  // -- State --
  let campaigns      = $state<FusionCampaign[]>([]);
  let selected       = $state<FusionCampaign | null>(null);
  let killChain      = $state<KillChainStage[]>([]);
  let loading        = $state(true);

  // -- Helpers --
  const confidenceColor = (c: number) => {
    if (c >= 0.8) return 'var(--alert-critical)';
    if (c >= 0.6) return 'var(--alert-high)';
    if (c >= 0.4) return 'var(--alert-medium)';
    return 'var(--status-online)';
  };


  const activeCount = $derived(campaigns.filter(c => c.status === 'active').length);
  const avgConf     = $derived(campaigns.length ? Math.round(campaigns.reduce((a,b)=>a+b.confidence, 0) / campaigns.length * 100) : 0);
  const maxStages   = $derived(campaigns.length ? Math.max(...campaigns.map(c => c.stage_count)) : 0);

  // -- Actions --
  async function fetchData() {
    loading = true;
    try {
      const res = await request<{ campaigns: FusionCampaign[] }>('/fusion/campaigns');
      campaigns = res.campaigns ?? [];
      if (campaigns.length > 0 && !selected) {
        selectCampaign(campaigns[0]);
      }
    } catch (e) {
      console.error('Fusion Data fetch failed', e);
    } finally {
      loading = false;
    }
  }

  async function selectCampaign(c: FusionCampaign) {
    selected = c;

    try {
      const res = await request<{ stages: KillChainStage[] }>(`/fusion/campaigns/${c.id}/kill-chain`);
      killChain = res.stages ?? [];
    } catch {
      killChain = [];
    } finally {

    }
  }

  onMount(() => {
    fetchData();
  });
</script>

<PageLayout title="Fusion Intelligence" subtitle="Mapping tactical telemetry to strategic mission objectives via multi-layer correlation engine">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm" onclick={fetchData}>
        <Pulse size={14} class="mr-2" />
        RE-CORRELATE
      </Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-0 -m-6 overflow-hidden">
    <!-- METRIC STRIP -->
    <div class="grid grid-cols-4 gap-px bg-border-primary border-b border-border-primary shrink-0">
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Active Clusters</div>
            <div class="text-xl font-mono font-bold text-alert-critical">{activeCount}</div>
            <div class="text-[9px] text-alert-critical mt-1 {activeCount > 0 ? 'animate-pulse' : ''}">
              {activeCount > 0 ? '▲ High Density Detect' : '✓ Nominal Security State'}
            </div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Correlation Depth</div>
            <div class="text-xl font-mono font-bold text-accent-primary">L7+</div>
            <div class="text-[9px] text-status-online mt-1">Cross-Platform Sync</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Avg Confidence</div>
            <div class="text-xl font-mono font-bold text-text-heading">{avgConf}%</div>
            <div class="text-[9px] text-status-online mt-1">Sovereign-grade ML</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Max Coverage</div>
            <div class="text-xl font-mono font-bold text-text-heading">{maxStages}/12</div>
            <div class="text-[9px] text-text-muted mt-1">Kill-chain penetration</div>
        </div>
    </div>

    <!-- MAIN BODY -->
    <div class="flex-1 flex min-h-0 bg-surface-0">
        <!-- LEFT: CAMPAIGN LIST -->
        <div class="w-96 bg-surface-1 border-r border-border-primary flex flex-col shrink-0 overflow-hidden">
            <div class="p-3 bg-surface-2 border-b border-border-primary flex items-center gap-2 shrink-0">
                <Layers size={14} class="text-accent-primary" />
                <span class="text-[10px] font-mono font-bold uppercase tracking-widest text-text-heading">Active Threat Campaigns</span>
            </div>
            
            <div class="flex-1 overflow-auto p-3 space-y-3">
              {#if loading}
                <div class="py-12 flex justify-center"><Spinner /></div>
              {:else}
                {#each campaigns as campaign}
                    <button 
                        class="w-full text-left bg-surface-2 border {selected?.id === campaign.id ? 'border-accent-primary' : 'border-border-primary'} p-4 rounded-sm space-y-4 hover:border-accent-primary transition-all cursor-pointer group relative overflow-hidden"
                        onclick={() => selectCampaign(campaign)}
                    >
                        <div class="flex items-start justify-between">
                            <div class="flex flex-col">
                                <span class="text-[12px] font-black text-text-heading tracking-tight leading-tight uppercase italic">{campaign.id.slice(0, 16)}</span>
                                <span class="text-[8px] font-mono text-text-muted uppercase opacity-60">INGESTED: {new Date(campaign.first_seen).toLocaleDateString()}</span>
                            </div>
                            <Badge variant={campaign.status === 'active' ? 'danger' : 'warning'} size="xs" class="font-bold">
                                {campaign.status.toUpperCase()}
                            </Badge>
                        </div>
                        
                        <div class="space-y-1.5">
                           <div class="flex justify-between items-center text-[9px] font-mono text-text-muted">
                              <span>KILL CHAIN PROGRESSION</span>
                              <span>{campaign.stage_count}/12</span>
                           </div>
                           <div class="flex gap-0.5 h-1.5">
                              {#each TACTIC_ORDER as tactic}
                                 {@const active = campaign.tactic_stages.some(s => s.toLowerCase().includes(tactic.toLowerCase().split(' ')[0]))}
                                 <div class="flex-1 rounded-xs transition-colors" style="background: {active ? confidenceColor(campaign.confidence) : 'var(--surface-0)'}"></div>
                              {/each}
                           </div>
                        </div>

                        <div class="flex justify-between items-end border-t border-border-subtle pt-3">
                            <div class="flex flex-col">
                                <span class="text-[8px] font-mono text-text-muted uppercase tracking-tight">Confidence</span>
                                <span class="text-[11px] font-mono font-bold" style="color: {confidenceColor(campaign.confidence)}">{Math.round(campaign.confidence * 100)}%</span>
                            </div>
                            <div class="flex flex-col items-end">
                                <span class="text-[8px] font-mono text-text-muted uppercase tracking-tight">Entities</span>
                                <span class="text-[10px] font-mono font-bold text-text-heading">{campaign.entities.length} SHARDS</span>
                            </div>
                        </div>
                        
                        {#if selected?.id === campaign.id}
                            <div class="absolute inset-y-0 right-0 w-1 bg-accent-primary"></div>
                        {/if}
                    </button>
                {:else}
                  <div class="py-20 text-center opacity-40 flex flex-col items-center gap-4">
                     <Layers size={48} />
                     <p class="text-[10px] font-mono uppercase tracking-widest">No strategic campaigns detected</p>
                  </div>
                {/each}
              {/if}
            </div>
        </div>

        <!-- RIGHT: CORRELATION ANALYSIS -->
        <div class="flex-1 flex flex-col min-w-0 overflow-hidden">
            {#if !selected}
               <div class="flex-1 flex flex-col items-center justify-center text-text-muted opacity-20 gap-6 p-20 text-center">
                  <Target size={120} />
                  <p class="font-mono text-xs uppercase tracking-widest max-w-sm leading-relaxed">
                    Select a detected threat campaign to initialize deep-layer correlation analysis and kill-chain orchestration.
                  </p>
               </div>
            {:else}
              <div class="bg-surface-1 border-b border-border-primary p-3 flex items-center justify-between shrink-0">
                  <div class="flex items-center gap-2">
                      <Target size={14} class="text-alert-critical" />
                      <span class="text-[10px] font-mono font-bold uppercase tracking-widest text-text-heading">Correlation Analysis: {selected.id}</span>
                  </div>
                  <div class="flex gap-2">
                      <Button variant="secondary" size="xs" icon={Share2}>VIEW GRAPH</Button>
                      <Button variant="danger" size="xs" icon={ShieldAlert}>SHARD ISOLATION</Button>
                  </div>
              </div>

              <div class="flex-1 overflow-auto p-6 space-y-10">
                  <!-- CAMPAIGN METADATA HEADER -->
                  <div class="bg-surface-2 border border-border-primary p-5 rounded-sm relative overflow-hidden group">
                     <div class="absolute -right-4 -bottom-4 opacity-[0.03] grayscale group-hover:scale-110 transition-transform duration-700">
                        <Pulse size={160} />
                     </div>
                     <div class="grid grid-cols-1 md:grid-cols-3 gap-8 relative z-10">
                        <div class="space-y-1">
                           <span class="text-[9px] font-mono text-text-muted uppercase tracking-widest">Temporal Context</span>
                           <div class="text-[11px] font-bold text-text-secondary">First: {new Date(selected.first_seen).toLocaleString()}</div>
                           <div class="text-[11px] font-bold text-text-secondary">Last: {new Date(selected.last_seen).toLocaleString()}</div>
                        </div>
                        <div class="space-y-1">
                           <span class="text-[9px] font-mono text-text-muted uppercase tracking-widest">Alert Density</span>
                           <div class="text-2xl font-mono font-black text-text-heading italic">{selected.alert_count} TRIGGERED</div>
                        </div>
                        <div class="space-y-1">
                           <span class="text-[9px] font-mono text-text-muted uppercase tracking-widest">Probabilistic Score</span>
                           <div class="text-2xl font-mono font-black italic" style="color: {confidenceColor(selected.confidence)}">
                              {Math.round(selected.confidence * 100)}% CONF
                           </div>
                        </div>
                     </div>
                     <div class="mt-6 flex flex-wrap gap-2 relative z-10">
                        {#each selected.entities as entity}
                           <div class="px-2 py-1 bg-surface-1 border border-border-subtle rounded-xs flex items-center gap-2">
                              <div class="w-1.5 h-1.5 rounded-full bg-accent-primary/40"></div>
                              <span class="text-[9px] font-mono font-bold text-text-secondary uppercase">{entity}</span>
                           </div>
                        {/each}
                     </div>
                  </div>

                  <!-- ATTACK TIMELINE / KILL CHAIN -->
                  <div class="space-y-6">
                      <span class="text-[10px] font-mono font-bold text-text-muted uppercase tracking-widest flex items-center gap-2">
                        <Activity size={12} class="text-accent-primary" />
                        Multi-Stage Propagation Path
                      </span>
                      
                      <div class="flex overflow-x-auto pb-4 gap-2 scrollbar-hide">
                         {#each TACTIC_ORDER as tactic, i}
                            {@const stage = killChain.find(s => s.tactic_name.toLowerCase().includes(tactic.toLowerCase().split(' ')[0]))}
                            <div class="flex items-center gap-2 shrink-0">
                               <div class="w-32 bg-surface-1 border border-border-primary rounded-sm p-3 flex flex-col gap-2 transition-all
                                 {stage ? 'border-alert-critical bg-alert-critical/5' : 'opacity-40'}">
                                  <div class="flex justify-between items-start">
                                     <span class="text-[8px] font-black text-text-muted italic">0{i+1}</span>
                                     {#if stage}<Badge variant="danger" size="xs">{stage.hit_count}</Badge>{:else}<div class="w-3 h-3 border border-border-subtle rounded-full"></div>{/if}
                                  </div>
                                  <div class="text-[10px] font-black uppercase tracking-tighter leading-tight {stage ? 'text-alert-critical' : 'text-text-muted'}">{tactic}</div>
                                  {#if stage}
                                     <div class="text-[8px] font-mono text-alert-critical/60 truncate uppercase">{stage.techniques[0] || 'Unknown Technique'}</div>
                                  {/if}
                               </div>
                               {#if i < TACTIC_ORDER.length - 1}
                                  <ChevronRight size={14} class="text-border-primary" />
                               {/if}
                            </div>
                         {/each}
                      </div>
                  </div>

                  <!-- DETAILED HITS (If available from killChain) -->
                  {#if killChain.length > 0}
                    <div class="space-y-4">
                       <span class="text-[10px] font-mono font-bold text-text-muted uppercase tracking-widest">Observed Technique Clusters</span>
                       <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                          {#each killChain as stage}
                             <div class="bg-surface-2 border border-border-primary p-4 rounded-sm space-y-3">
                                <div class="flex justify-between items-center border-b border-border-subtle pb-2">
                                   <span class="text-[10px] font-black text-text-heading uppercase tracking-widest">{stage.tactic_name}</span>
                                   <span class="text-[9px] font-mono text-text-muted">{stage.hit_count} HITS</span>
                                </div>
                                <div class="flex flex-wrap gap-1.5">
                                   {#each stage.techniques as tech}
                                      <span class="text-[8px] font-mono bg-surface-1 px-1.5 py-0.5 border border-border-subtle text-text-muted uppercase">{tech}</span>
                                   {/each}
                                </div>
                             </div>
                          {/each}
                       </div>
                    </div>
                  {/if}
              </div>
            {/if}
        </div>
    </div>

    <!-- STATUS BAR -->
    <div class="bg-surface-2 border-t border-border-primary px-3 py-1 flex items-center gap-4 text-[8px] font-mono text-text-muted shrink-0 uppercase tracking-widest">
        <div class="flex items-center gap-1.5">
            <div class="w-1 h-1 rounded-full bg-status-online"></div>
            <span>FUSION_CORE:</span>
            <span class="text-status-online font-bold italic">STABLE</span>
        </div>
        <span class="text-border-primary opacity-30">|</span>
        <div class="flex items-center gap-1.5">
            <span>LAYERS:</span>
            <span class="text-accent-primary font-bold italic">EDR, SIEM, NDR, UEBA, CTI</span>
        </div>
        <span class="text-border-primary opacity-30">|</span>
        <div class="flex items-center gap-1.5">
            <span>PIPELINE:</span>
            <span class="text-status-online font-bold italic">AUTO_CORRELATE</span>
        </div>
        <div class="ml-auto opacity-40">OBLIVRA_FUSION_ENGINE v2.4.1</div>
    </div>
  </div>
</PageLayout>

<style>
  :global(.flex-1::-webkit-scrollbar) {
    width: 6px;
    height: 6px;
  }
  :global(.flex-1::-webkit-scrollbar-track) {
    background: var(--surface-0);
  }
  :global(.flex-1::-webkit-scrollbar-thumb) {
    background: var(--border-primary);
    border-radius: 3px;
  }
  .scrollbar-hide::-webkit-scrollbar {
    display: none;
  }
</style>
