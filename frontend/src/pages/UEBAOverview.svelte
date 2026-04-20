<!--
  OBLIVRA — UEBA Overview (Svelte 5)
  Real-time behavioral intelligence and anomaly orchestration.
-->
<script lang="ts">
  import { PageLayout, Badge, Button, DataTable, ProgressBar } from '@components/ui';
  import { User, Monitor, Activity, Radar } from 'lucide-svelte';
  import { uebaStore } from '@lib/stores/ueba.svelte';
  import { onMount } from 'svelte';

  const highRiskEntities = $derived(uebaStore.profiles);
  const anomalies = $derived(uebaStore.anomalies);
  const stats = $derived(uebaStore.stats);

  onMount(() => {
    uebaStore.refresh();
  });
</script>

<PageLayout title="Behavioral Intelligence" subtitle="Autonomous anomaly scoring and peer group analysis engine">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm" onclick={() => uebaStore.refresh()} loading={uebaStore.loading}>BASELINE STATUS</Button>
      <Button variant="primary" size="sm">DOWNLOAD AUDIT</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-0 -m-6">
    <!-- METRIC STRIP -->
    <div class="grid grid-cols-4 gap-px bg-border-primary border-b border-border-primary shrink-0">
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">High Risk Entities</div>
            <div class="text-xl font-mono font-bold text-error">{stats.high_risk_entities}</div>
            <div class="text-[9px] text-error mt-1">▲ Critical baseline breach</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Total Entities</div>
            <div class="text-xl font-mono font-bold text-text-heading">{stats.total_entities}</div>
            <div class="text-[9px] text-text-muted mt-1">Global fleet total</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Anomaly Ingest</div>
            <div class="text-xl font-mono font-bold text-accent">{stats.anomalies_24h}/24h</div>
            <div class="text-[9px] text-success mt-1">ML Engine optimized</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Baselines Active</div>
            <div class="text-xl font-mono font-bold text-text-heading">{stats.baselines_active}</div>
            <div class="text-[9px] text-success mt-1">Continuous profiling</div>
        </div>
    </div>

    <!-- MAIN BODY -->
    <div class="flex-1 flex min-h-0">
        <!-- LEFT: ENTITY LIST -->
        <div class="flex-1 flex flex-col min-w-0">
            <div class="bg-surface-1 border-b border-border-primary p-3 flex items-center justify-between shrink-0">
                <div class="flex items-center gap-2">
                    <Radar size={14} class="text-accent" />
                    <span class="text-[10px] font-mono font-bold uppercase tracking-widest text-text-heading">Behavioral Risk Ledger</span>
                </div>
                <div class="flex bg-surface-2 border border-border-primary rounded-sm overflow-hidden h-6">
                    <button class="px-2 text-[8px] font-mono font-bold bg-surface-3 text-accent border-r border-border-primary uppercase">All</button>
                    <button class="px-2 text-[8px] font-mono font-bold text-text-muted uppercase">Users</button>
                    <button class="px-2 text-[8px] font-mono font-bold text-text-muted uppercase">Hosts</button>
                </div>
            </div>

            <div class="flex-1 overflow-auto mask-fade-bottom">
                <DataTable 
                    data={highRiskEntities} 
                    columns={[
                        { key: 'id', label: 'ENTITY' },
                        { key: 'peer_group', label: 'PEER GROUP', width: '120px' },
                        { key: 'deviations', label: 'ANOMALIES', width: '100px' },
                        { key: 'risk', label: 'SCORE', width: '140px' },
                        { key: 'actions', label: '', width: '80px' }
                    ]} 
                    compact
                >
                    {#snippet render({ col, row })}
                        {#if col.key === 'id'}
                            <div class="flex items-center gap-2 py-0.5">
                                {#if row.type === 'user'}
                                    <User size={12} class="text-accent" />
                                {:else}
                                    <Monitor size={12} class="text-text-muted" />
                                {/if}
                                <span class="text-[11px] font-bold text-text-secondary leading-tight">{row.id}</span>
                            </div>
                        {:else if col.key === 'peer_group'}
                            <span class="text-[10px] font-mono text-text-muted">{row.peer_group}</span>
                        {:else if col.key === 'deviations'}
                            <Badge variant={row.deviations > 5 ? 'warning' : 'muted'} size="xs" class="font-mono">{row.deviations}</Badge>
                        {:else if col.key === 'risk'}
                            <div class="flex items-center gap-2 w-full">
                                <ProgressBar value={row.risk} variant={row.risk > 80 ? 'error' : row.risk > 40 ? 'warning' : 'success'} size="xs" />
                                <span class="text-[10px] font-mono text-text-muted w-8">{row.risk}</span>
                            </div>
                        {:else if col.key === 'actions'}
                            <Button variant="ghost" size="xs" class="h-6 px-2 text-[8px] font-mono">PROFILE</Button>
                        {/if}
                    {/snippet}
                </DataTable>
            </div>
        </div>

        <!-- RIGHT: LIVE ANOMALY FEED -->
        <div class="w-96 bg-surface-2 border-l border-border-primary flex flex-col shrink-0">
            <div class="px-3 py-2 bg-surface-3 border-b border-border-primary flex items-center justify-between">
                <div class="flex items-center gap-2">
                    <Activity size={14} class="text-error" />
                    <span class="text-[9px] font-mono font-bold uppercase tracking-widest text-text-heading">Real-time Deviations</span>
                </div>
                <div class="w-1.5 h-1.5 rounded-full bg-success animate-pulse"></div>
            </div>
            
            <div class="flex-1 overflow-auto p-3 space-y-3">
                {#each anomalies as anomaly}
                    <div class="bg-surface-1 border border-border-primary p-3 rounded-sm space-y-3 hover:border-error transition-colors cursor-pointer group">
                        <div class="flex items-start justify-between">
                            <div class="flex flex-col">
                                <span class="text-[11px] font-bold text-text-heading uppercase tracking-tighter">{anomaly.signal}</span>
                                <span class="text-[8px] font-mono text-text-muted uppercase">{anomaly.time} · {anomaly.entity}</span>
                            </div>
                            <div class="px-1.5 py-0.5 rounded-sm bg-error/10 border border-error/20 text-[9px] font-mono font-black text-error">
                                {anomaly.score}%
                            </div>
                        </div>
                        <div class="p-2 bg-surface-2 border border-border-primary rounded-sm text-[9px] font-mono text-text-muted italic">
                            Evidence: {anomaly.evidence}
                        </div>
                        <div class="flex gap-2 pt-1 opacity-0 group-hover:opacity-100 transition-opacity">
                            <button class="flex-1 px-2 py-1 bg-surface-3 border border-border-primary text-[8px] font-mono text-text-muted hover:text-text-secondary">CORRELATE</button>
                            <button class="flex-1 px-2 py-1 bg-error/10 border border-error/20 text-[8px] font-mono text-error hover:bg-error/20">INVESTIGATE</button>
                        </div>
                    </div>
                {/each}
            </div>

            <!-- RISK DISTRIBUTION CHART (SIMULATED) -->
            <div class="h-48 border-t border-border-primary bg-surface-3 p-4 space-y-3">
                <span class="text-[8px] font-mono font-bold text-text-muted uppercase tracking-widest">Global Risk Distribution</span>
                <div class="flex-1 flex items-end gap-1.5 pt-4">
                    {#each [20, 35, 60, 85, 45, 25, 15] as h, i}
                        <div class="flex-1 {i === 3 ? 'bg-error' : 'bg-accent/40'} border-t border-white/5 rounded-t-sm" style="height: {h}%"></div>
                    {/each}
                </div>
                <div class="flex justify-between text-[8px] font-mono text-text-muted uppercase pt-1">
                    <span>LOW</span>
                    <span>NOMINAL</span>
                    <span>HIGH</span>
                </div>
            </div>
        </div>
    </div>

    <!-- STATUS BAR -->
    <div class="bg-surface-2 border-t border-border-primary px-3 py-1 flex items-center gap-4 text-[8px] font-mono text-text-muted shrink-0">
        <div class="flex items-center gap-1.5">
            <span>ENGINE:</span>
            <span class="text-success font-bold">OPTIMIZED</span>
        </div>
        <span class="text-border-primary">|</span>
        <div class="flex items-center gap-1.5">
            <span>MODELS:</span>
            <span class="text-accent font-bold">142 ACTIVE</span>
        </div>
        <span class="text-border-primary">|</span>
        <div class="flex items-center gap-1.5">
            <span>PEER_GROUPS:</span>
            <span class="text-success font-bold">1.2K CALCULATED</span>
        </div>
        <div class="ml-auto uppercase tracking-widest opacity-60">UEBA_CORE v1.14.2</div>
    </div>
  </div>
</PageLayout>

<style>
  .overflow-auto {
    mask-image: linear-gradient(to bottom, transparent 0px, black 12px, black calc(100% - 16px), transparent 100%);
  }
</style>
