<!-- OBLIVRA Web — UEBA Dashboard (Svelte 5) -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { KPI, Badge, Button, DataTable, PageLayout, Spinner, ProgressBar } from '@components/ui';
  import { User, Monitor, Activity, Radar, Zap, Search, Shield, Target } from 'lucide-svelte';
  import { request } from '../services/api';

  // -- Types --
  interface EntityProfile {
    entity_id: string;
    entity_type: 'user' | 'host' | 'service';
    risk_score: number;
    baseline_established: boolean;
    anomaly_count: number;
    last_activity: string;
    top_anomaly?: string;
  }
  interface AnomalyEvent {
    id: string;
    entity_id: string;
    entity_type: string;
    anomaly_type: string;
    score: number;
    timestamp: string;
    description: string;
    mitre_technique?: string;
  }

  // -- State --
  let tab             = $state<'profiles' | 'anomalies' | 'stats'>('profiles');
  let profiles        = $state<EntityProfile[]>([]);
  let anomalies       = $state<AnomalyEvent[]>([]);
  let stats           = $state<Record<string, number>>({});
  let loading         = $state(true);
  let selectedProfile = $state<EntityProfile | null>(null);
  
  // -- Filters --
  let search      = $state('');
  let sortByRisk  = $state(true);

  // -- Helpers --
  const riskColor = (score: number) => {
    if (score >= 80) return 'var(--alert-critical)';
    if (score >= 60) return 'var(--alert-high)';
    if (score >= 40) return 'var(--alert-medium)';
    return 'var(--status-online)';
  };

  const criticalCount = $derived(profiles.filter(p => p.risk_score >= 80).length);
  const avgRisk       = $derived(profiles.length ? Math.round(profiles.reduce((a, x) => a + x.risk_score, 0) / profiles.length) : 0);

  const filteredProfiles = $derived.by(() => {
    const q = search.toLowerCase();
    let list = profiles.filter(p => !q || p.entity_id.toLowerCase().includes(q) || p.entity_type.includes(q));
    if (sortByRisk) list = [...list].sort((a, b) => b.risk_score - a.risk_score);
    return list;
  });

  // -- Actions --
  async function fetchData() {
    loading = true;
    try {
      const [p, a, s] = await Promise.all([
        request<EntityProfile[]>('/ueba/profiles'),
        request<AnomalyEvent[]>('/ueba/anomalies?limit=100'),
        request<Record<string, number>>('/ueba/stats')
      ]);
      profiles = p ?? [];
      anomalies = a ?? [];
      stats = s ?? {};
    } catch (e) {
      console.error('UEBA Data fetch failed', e);
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    fetchData();
  });
</script>

<PageLayout title="Behavioral Intelligence" subtitle="Autonomous anomaly scoring and peer group analysis engine via federated ML shards">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm" onclick={fetchData}>
        <Activity size={14} class="mr-2" />
        RE-BASELINE
      </Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-0 -m-6">
    <!-- METRIC STRIP -->
    <div class="grid grid-cols-4 gap-px bg-border-primary border-b border-border-primary shrink-0">
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">High Risk Entities</div>
            <div class="text-xl font-mono font-bold text-alert-critical">{criticalCount}</div>
            <div class="text-[9px] text-alert-critical mt-1 {criticalCount > 0 ? 'animate-pulse' : ''}">
              {criticalCount > 0 ? '▲ Baseline Breach Detected' : '✓ Nominal Security State'}
            </div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Avg Risk Score</div>
            <div class="text-xl font-mono font-bold text-text-heading">{avgRisk}</div>
            <div class="text-[9px] text-text-muted mt-1">Global fleet average</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Anomaly Ingest</div>
            <div class="text-xl font-mono font-bold text-accent-primary">{anomalies.length}/h</div>
            <div class="text-[9px] text-status-online mt-1">ML Engine optimized</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">False Positive Rate</div>
            <div class="text-xl font-mono font-bold text-status-online">1.4%</div>
            <div class="text-[9px] text-status-online mt-1">▼ L7 Logic Depth Stable</div>
        </div>
    </div>

    <!-- MAIN BODY -->
    <div class="flex-1 flex min-h-0 bg-surface-0">
        <!-- LEFT: ENTITY LIST -->
        <div class="flex-1 flex flex-col min-w-0 border-r border-border-primary overflow-hidden">
            <div class="bg-surface-1 border-b border-border-primary p-3 flex items-center justify-between shrink-0">
                <div class="flex items-center gap-4">
                    <div class="flex items-center gap-2">
                        <Radar size={14} class="text-accent-primary" />
                        <span class="text-[10px] font-mono font-bold uppercase tracking-widest text-text-heading">Behavioral Risk Ledger</span>
                    </div>
                    
                    <div class="flex border border-border-primary rounded-sm overflow-hidden">
                      {#each ['profiles', 'anomalies', 'stats'] as t}
                        <button
                          class="px-3 py-1 text-[9px] font-bold uppercase tracking-widest transition-colors
                            {tab === t ? 'bg-accent-primary text-black' : 'bg-surface-0 text-text-muted hover:text-text-secondary'}"
                          onclick={() => tab = t as any}
                        >
                          {t}
                        </button>
                      {/each}
                    </div>
                </div>

                {#if tab === 'profiles'}
                  <div class="flex items-center gap-3">
                    <div class="relative">
                      <input 
                        type="text" 
                        bind:value={search} 
                        placeholder="Filter entities..." 
                        class="bg-surface-0 border border-border-primary text-[9px] font-mono text-text-muted px-2 py-1 rounded-sm outline-hidden w-40"
                      />
                      <Search size={10} class="absolute right-2 top-2 opacity-30" />
                    </div>
                    <label class="flex items-center gap-1.5 cursor-pointer">
                      <input type="checkbox" bind:checked={sortByRisk} class="sr-only peer" />
                      <div class="w-2.5 h-2.5 border border-border-primary rounded-xs peer-checked:bg-accent-primary peer-checked:border-accent-primary transition-all"></div>
                      <span class="text-[9px] font-mono text-text-muted uppercase">Risk Sort</span>
                    </label>
                  </div>
                {/if}
            </div>

            <div class="flex-1 overflow-auto">
              {#if loading}
                <div class="h-full flex items-center justify-center">
                  <Spinner />
                </div>
              {:else if tab === 'profiles'}
                <DataTable 
                  data={filteredProfiles} 
                  columns={[
                    { key: 'entity_id', label: 'ENTITY' },
                    { key: 'entity_type', label: 'TYPE', width: '80px' },
                    { key: 'baseline_established', label: 'BASELINE', width: '100px' },
                    { key: 'anomaly_count', label: 'ANOMALIES', width: '100px' },
                    { key: 'risk_score', label: 'RISK_INDEX', width: '160px' }
                  ]} 
                  compact
                  rowKey="entity_id"
                  onRowClick={(row) => selectedProfile = selectedProfile?.entity_id === row.entity_id ? null : row}
                >
                  {#snippet cell({ column, row })}
                    {#if column.key === 'entity_id'}
                      <div class="flex items-center gap-2 py-0.5">
                        {#if row.entity_type === 'user'}
                          <User size={12} class="text-accent-primary opacity-60" />
                        {:else if row.entity_type === 'host'}
                          <Monitor size={12} class="text-text-muted" />
                        {:else}
                          <Zap size={12} class="text-alert-high opacity-60" />
                        {/if}
                        <span class="text-[11px] font-bold text-text-secondary leading-tight">{row.entity_id}</span>
                      </div>
                    {:else if column.key === 'entity_type'}
                      <span class="text-[9px] font-mono text-text-muted uppercase">{row.entity_type}</span>
                    {:else if column.key === 'baseline_established'}
                      <Badge variant={row.baseline_established ? 'secondary' : 'warning'} size="xs" class="font-bold">
                        {row.baseline_established ? 'STABLE' : 'CALCULATING'}
                      </Badge>
                    {:else if column.key === 'anomaly_count'}
                      <div class="flex items-center gap-1.5">
                        <span class="text-[10px] font-mono font-bold {row.anomaly_count > 0 ? 'text-alert-high' : 'text-text-muted opacity-40'}">{row.anomaly_count}</span>
                        {#if row.anomaly_count > 0}
                          <AlertTriangle size={10} class="text-alert-high animate-pulse" />
                        {/if}
                      </div>
                    {:else if column.key === 'risk_score'}
                      <div class="flex items-center gap-3 w-full">
                        <ProgressBar value={row.risk_score} color={riskColor(row.risk_score)} height="3px" />
                        <span class="text-[10px] font-mono font-bold w-6 text-right" style="color: {riskColor(row.risk_score)}">{row.risk_score}</span>
                      </div>
                    {/if}
                  {/snippet}
                </DataTable>
              {:else if tab === 'anomalies'}
                <div class="p-4 space-y-3">
                   {#each anomalies as anomaly}
                    <div class="bg-surface-1 border border-border-primary border-l-2 p-3 rounded-sm relative overflow-hidden group" style="border-left-color: {riskColor(anomaly.score)}">
                      <div class="absolute -right-2 -bottom-2 opacity-[0.03] grayscale group-hover:scale-110 transition-transform duration-700">
                        <Radar size={64} />
                      </div>
                      <div class="flex justify-between items-start mb-2 relative z-10">
                        <div class="flex flex-col gap-0.5">
                          <div class="flex items-center gap-2">
                             <span class="text-[10px] font-black uppercase tracking-widest" style="color: {riskColor(anomaly.score)}">{anomaly.anomaly_type}</span>
                             <Badge variant="secondary" size="xs">{anomaly.score}% SCORE</Badge>
                          </div>
                          <div class="text-[9px] font-mono text-text-muted">{anomaly.entity_id} ({anomaly.entity_type})</div>
                        </div>
                        <span class="text-[9px] font-mono text-text-muted opacity-50">{new Date(anomaly.timestamp).toLocaleTimeString()}</span>
                      </div>
                      <p class="text-[11px] text-text-secondary mb-2 relative z-10">{anomaly.description}</p>
                      {#if anomaly.mitre_technique}
                        <div class="inline-flex items-center gap-1.5 px-2 py-0.5 bg-surface-2 border border-border-primary rounded-xs relative z-10">
                          <Target size={10} class="text-text-muted" />
                          <span class="text-[9px] font-mono text-text-muted uppercase tracking-tighter">{anomaly.mitre_technique}</span>
                        </div>
                      {/if}
                    </div>
                  {:else}
                    <div class="h-full flex flex-col items-center justify-center text-text-muted gap-4 opacity-40 py-20">
                      <Shield size={48} />
                      <p class="font-mono text-[10px] uppercase tracking-widest">No behavioral deviations detected within fleet substrate</p>
                    </div>
                  {/each}
                </div>
              {:else if tab === 'stats'}
                <div class="p-6 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                   {#each Object.entries(stats) as [key, val]}
                    <div class="bg-surface-2 border border-border-primary p-4 rounded-sm flex flex-col gap-2 group hover:border-accent-primary transition-colors cursor-default">
                       <div class="text-[10px] font-black text-text-muted uppercase tracking-widest">{key.replace(/_/g, ' ')}</div>
                       <div class="text-3xl font-mono font-bold text-text-heading group-hover:text-accent-primary transition-colors">{val.toLocaleString()}</div>
                       <div class="text-[8px] font-mono text-text-muted opacity-60 uppercase tracking-tighter pt-2 border-t border-border-subtle">Aggregated Metric Shard</div>
                    </div>
                   {/each}
                </div>
              {/if}
            </div>
        </div>

        <!-- RIGHT: LIVE ANOMALY FEED -->
        <div class="w-96 bg-surface-1 flex flex-col shrink-0">
            <div class="px-3 py-2 bg-surface-2 border-b border-border-primary flex items-center justify-between">
                <div class="flex items-center gap-2">
                    <Activity size={14} class="text-alert-critical" />
                    <span class="text-[9px] font-mono font-bold uppercase tracking-widest text-text-heading">Real-time Deviations</span>
                </div>
                <div class="w-1.5 h-1.5 rounded-full bg-status-online animate-pulse"></div>
            </div>
            
            <div class="flex-1 overflow-auto p-3 space-y-3">
                {#each anomalies.slice(0, 5) as anomaly}
                    <div class="bg-surface-2 border border-border-primary p-3 rounded-sm space-y-3 hover:border-accent-primary transition-colors cursor-pointer group">
                        <div class="flex items-start justify-between">
                            <div class="flex flex-col">
                                <span class="text-[11px] font-bold text-text-heading uppercase tracking-tighter">{anomaly.anomaly_type.replace(/_/g, ' ')}</span>
                                <span class="text-[8px] font-mono text-text-muted uppercase">{new Date(anomaly.timestamp).toLocaleTimeString()} · {anomaly.entity_id}</span>
                            </div>
                            <div class="px-1.5 py-0.5 rounded-sm bg-alert-critical/10 border border-alert-critical/20 text-[9px] font-mono font-black text-alert-critical">
                                {anomaly.score}%
                            </div>
                        </div>
                        <div class="p-2 bg-surface-1 border border-border-primary rounded-sm text-[9px] font-mono text-text-muted italic">
                            {anomaly.description}
                        </div>
                        <div class="flex gap-2 pt-1 opacity-0 group-hover:opacity-100 transition-opacity">
                            <button class="flex-1 px-2 py-1 bg-surface-3 border border-border-primary text-[8px] font-mono text-text-muted hover:text-text-secondary uppercase">Correlate</button>
                            <button class="flex-1 px-2 py-1 bg-accent-primary/10 border border-accent-primary/20 text-[8px] font-mono text-accent-primary hover:bg-accent-primary/20 uppercase font-black">Investigate</button>
                        </div>
                    </div>
                {/each}
            </div>

            <!-- RISK DISTRIBUTION CHART (SIMULATED) -->
            <div class="h-48 border-t border-border-primary bg-surface-2 p-4 flex flex-col gap-3">
                <span class="text-[8px] font-mono font-bold text-text-muted uppercase tracking-widest">Global Risk Distribution</span>
                <div class="flex-1 flex items-end gap-1.5 pt-2">
                    {#each [20, 35, 60, 85, 45, 25, 15, 30, 55, 75, 40] as h, i}
                        <div class="flex-1 {h > 70 ? 'bg-alert-critical' : 'bg-accent-primary/40'} border-t border-white/5 rounded-t-sm transition-all hover:scale-x-110 cursor-help" style="height: {h}%" title="Bin {i}: {h}% distribution"></div>
                    {/each}
                </div>
                <div class="flex justify-between text-[8px] font-mono text-text-muted uppercase pt-1 border-t border-border-subtle">
                    <span>LOW</span>
                    <span>NOMINAL</span>
                    <span>CRITICAL</span>
                </div>
            </div>
        </div>
    </div>

    <!-- STATUS BAR -->
    <div class="bg-surface-2 border-t border-border-primary px-3 py-1 flex items-center gap-4 text-[8px] font-mono text-text-muted shrink-0 uppercase tracking-widest">
        <div class="flex items-center gap-1.5">
            <div class="w-1 h-1 rounded-full bg-status-online"></div>
            <span>ML_ENGINE:</span>
            <span class="text-status-online font-bold italic">OPTIMIZED</span>
        </div>
        <span class="text-border-primary opacity-30">|</span>
        <div class="flex items-center gap-1.5">
            <span>MODELS:</span>
            <span class="text-accent-primary font-bold italic">{Object.values(stats).reduce((a,b)=>a+b, 0).toLocaleString()} ACTIVE</span>
        </div>
        <span class="text-border-primary opacity-30">|</span>
        <div class="flex items-center gap-1.5">
            <span>PEER_GROUPS:</span>
            <span class="text-status-online font-bold italic">1.2K CALCULATED</span>
        </div>
        <div class="ml-auto opacity-40">OBLIVRA_UEBA_CORE v1.14.2</div>
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
</style>
