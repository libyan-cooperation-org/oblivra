<!-- OBLIVRA Web — SOAR Dashboard (Svelte 5) -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { KPI, Badge, Button, DataTable, PageLayout, Spinner, ProgressBar } from '@components/ui';
  import { Activity, Zap, History, Clock, Target, AlertTriangle, ChevronRight, TrendingUp } from 'lucide-svelte';
  import { request } from '../services/api';

  // -- Types --
  interface PlaybookExecution {
    playbook_id: string;
    incident_id: string;
    started_at: string;
    completed_at: string;
    duration_ms: number;
    status: 'completed' | 'failed' | 'running';
    step_count: number;
  }
  interface MetricsResponse {
    total_executions: number;
    success_count: number;
    failure_count: number;
    avg_duration_ms: number;
    executions_by_playbook: Record<string, number>;
    recent_executions: PlaybookExecution[];
  }

  // -- State --
  let tab     = $state<'overview' | 'history' | 'bottlenecks'>('overview');
  let metrics = $state<MetricsResponse | null>(null);
  let loading = $state(true);

  // -- Helpers --
  const durationLabel = (ms: number) => {
    if (ms < 1000) return `${ms}ms`;
    if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
    return `${Math.floor(ms / 60000)}m ${Math.floor((ms % 60000) / 1000)}s`;
  };
  const statusColor = (s: string) => {
    if (s === 'completed') return 'var(--status-online)';
    if (s === 'failed') return 'var(--alert-critical)';
    return 'var(--alert-high)';
  };
  const successRate = $derived(metrics?.total_executions ? Math.round((metrics.success_count / metrics.total_executions) * 100) : 0);
  const topPlaybooks = $derived(metrics ? Object.entries(metrics.executions_by_playbook).sort((a,b) => b[1] - a[1]).slice(0, 8) : []);

  // -- Actions --
  async function fetchData() {
    loading = true;
    try {
      metrics = await request<MetricsResponse>('/playbooks/metrics');
    } catch (e) {
      console.error('SOAR Metrics fetch failed', e);
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    fetchData();
  });
</script>

<PageLayout title="Orchestration Metrics" subtitle="SOAR performance substrate: MTTR analysis, success telemetry, and execution shards">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm" onclick={fetchData}>
        <Activity size={14} class="mr-2" />
        METRICS RE-SYNC
      </Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-0 -m-6">
    <!-- METRIC STRIP -->
    <div class="grid grid-cols-4 gap-px bg-border-primary border-b border-border-primary shrink-0">
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Total Executions</div>
            <div class="text-xl font-mono font-bold text-accent-primary">{metrics?.total_executions ?? 0}</div>
            <div class="text-[9px] text-text-muted mt-1">Autonomous orchestration</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Success Rate</div>
            <div class="text-xl font-mono font-bold {successRate >= 90 ? 'text-status-online' : 'text-alert-high'}">{successRate}%</div>
            <div class="text-[9px] {successRate >= 90 ? 'text-status-online' : 'text-alert-high'} mt-1">
              {successRate >= 90 ? '▲ Baseline nominal' : '▼ Below SLA threshold'}
            </div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Avg Execution Time</div>
            <div class="text-xl font-mono font-bold text-text-heading">{durationLabel(metrics?.avg_duration_ms ?? 0)}</div>
            <div class="text-[9px] text-text-muted mt-1">Fleet MTTR optimization</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Active Failures</div>
            <div class="text-xl font-mono font-bold {metrics?.failure_count ? 'text-alert-critical' : 'text-text-heading'}">{metrics?.failure_count ?? 0}</div>
            <div class="text-[9px] text-text-muted mt-1">Requiring manual triage</div>
        </div>
    </div>

    <!-- MAIN BODY -->
    <div class="flex-1 flex min-h-0 bg-surface-0">
        <!-- LEFT: MAIN CONTENT -->
        <div class="flex-1 flex flex-col min-w-0 border-r border-border-primary overflow-hidden">
            <div class="bg-surface-1 border-b border-border-primary p-3 flex items-center justify-between shrink-0">
                <div class="flex items-center gap-4">
                    <div class="flex items-center gap-2">
                        <Zap size={14} class="text-accent-primary" />
                        <span class="text-[10px] font-mono font-bold uppercase tracking-widest text-text-heading">Playbook Performance Shard</span>
                    </div>
                    
                    <div class="flex border border-border-primary rounded-sm overflow-hidden">
                      {#each ['overview', 'history', 'bottlenecks'] as t}
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
            </div>

            <div class="flex-1 overflow-auto">
              {#if loading}
                <div class="h-full flex items-center justify-center">
                  <Spinner />
                </div>
              {:else if tab === 'overview'}
                <div class="p-6 space-y-8">
                  <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
                    <!-- Executions by Playbook -->
                    <div class="bg-surface-1 border border-border-primary rounded-sm overflow-hidden shadow-premium">
                       <div class="p-3 bg-surface-2 border-b border-border-primary text-[10px] font-bold uppercase tracking-widest text-text-muted">
                          Distribution by Playbook Identifier
                       </div>
                       <div class="p-4 space-y-4">
                          {#each topPlaybooks as [name, count]}
                             {@const maxCount = topPlaybooks[0][1]}
                             {@const pct = Math.round((count / maxCount) * 100)}
                             <div class="space-y-1.5">
                                <div class="flex justify-between items-center text-[10px] font-mono">
                                   <span class="text-text-secondary font-bold uppercase tracking-tighter">{name.replace(/-/g, ' ')}</span>
                                   <span class="text-accent-primary">{count} EXECS</span>
                                </div>
                                <ProgressBar value={pct} height="3px" color="var(--accent-primary)" />
                             </div>
                          {:else}
                             <div class="py-12 text-center text-[10px] font-mono text-text-muted uppercase tracking-widest">No execution data registered</div>
                          {/each}
                       </div>
                    </div>

                    <!-- Recent Timeline -->
                    <div class="bg-surface-1 border border-border-primary rounded-sm overflow-hidden shadow-premium">
                       <div class="p-3 bg-surface-2 border-b border-border-primary text-[10px] font-bold uppercase tracking-widest text-text-muted">
                          Real-time Execution Stream
                       </div>
                       <div class="p-2 space-y-1 max-h-[400px] overflow-auto">
                          {#each metrics?.recent_executions ?? [] as exec}
                             <div class="flex items-center gap-3 p-2 bg-surface-2 border border-border-subtle rounded-sm group hover:border-accent-primary transition-colors cursor-default">
                                <div class="w-1.5 h-1.5 rounded-full shrink-0" style="background: {statusColor(exec.status)}"></div>
                                <div class="flex-1 min-w-0">
                                   <div class="flex justify-between items-center">
                                      <span class="text-[11px] font-bold text-text-heading uppercase tracking-tighter truncate">{exec.playbook_id}</span>
                                      <span class="text-[9px] font-mono text-text-muted">{durationLabel(exec.duration_ms)}</span>
                                   </div>
                                   <div class="flex gap-2 text-[8px] font-mono text-text-muted uppercase opacity-60">
                                      <span>INC: {exec.incident_id || 'SYSTEM'}</span>
                                      <span>|</span>
                                      <span>{exec.step_count} STEPS</span>
                                   </div>
                                </div>
                                <ChevronRight size={12} class="text-text-muted opacity-20 group-hover:opacity-100 group-hover:text-accent-primary transition-all" />
                             </div>
                          {:else}
                             <div class="py-12 text-center text-[10px] font-mono text-text-muted uppercase tracking-widest">No recent timeline data</div>
                          {/each}
                       </div>
                    </div>
                  </div>
                </div>
              {:else if tab === 'history'}
                <DataTable 
                  data={metrics?.recent_executions ?? []} 
                  columns={[
                    { key: 'playbook_id', label: 'PLAYBOOK_ID' },
                    { key: 'incident_id', label: 'CASE_REF', width: '120px' },
                    { key: 'status', label: 'RESULT', width: '100px' },
                    { key: 'duration_ms', label: 'MTTR', width: '100px' },
                    { key: 'step_count', label: 'COMPLEXITY', width: '100px' },
                    { key: 'started_at', label: 'INGEST_TIME', width: '140px' }
                  ]} 
                  compact
                  rowKey="incident_id"
                >
                  {#snippet cell({ column, row })}
                    {#if column.key === 'playbook_id'}
                      <span class="text-[11px] font-bold text-text-heading uppercase">{row.playbook_id}</span>
                    {:else if column.key === 'incident_id'}
                      <span class="text-[9px] font-mono text-accent-primary">{row.incident_id || 'SYSTEM'}</span>
                    {:else if column.key === 'status'}
                      <Badge variant={row.status === 'completed' ? 'success' : 'danger'} size="xs" dot>{row.status}</Badge>
                    {:else if column.key === 'duration_ms'}
                      <span class="text-[10px] font-mono font-bold text-text-muted">{durationLabel(row.duration_ms)}</span>
                    {:else if column.key === 'step_count'}
                      <span class="text-[10px] font-mono text-text-muted">{row.step_count} SHARDS</span>
                    {:else if column.key === 'started_at'}
                      <span class="text-[9px] font-mono text-text-muted opacity-60 uppercase">{new Date(row.started_at).toLocaleTimeString()}</span>
                    {/if}
                  {/snippet}
                </DataTable>
              {:else if tab === 'bottlenecks'}
                <div class="p-6 space-y-6">
                   <div class="bg-surface-2 border border-alert-high p-4 rounded-sm flex flex-col gap-3 relative overflow-hidden group">
                      <div class="absolute -right-4 -bottom-4 opacity-[0.03] grayscale group-hover:scale-110 transition-transform duration-700">
                        <TrendingUp size={120} />
                      </div>
                      <div class="flex items-center gap-2">
                         <AlertTriangle size={16} class="text-alert-high" />
                         <span class="text-xs font-black uppercase tracking-widest text-alert-high">Performance Optimization Advisory</span>
                      </div>
                      <p class="text-[11px] text-text-secondary leading-relaxed max-w-2xl relative z-10">
                         Playbook execution bottlenecks are identified by comparing individual step durations against the global baseline. 
                         Steps exceeding 2× the cluster average are flagged for immediate logic sharding or credential pre-warming.
                      </p>
                      
                      <div class="grid grid-cols-1 md:grid-cols-3 gap-4 mt-2 relative z-10">
                         {#each [
                           { step: 'isolate_host', avg: 3200, note: 'SSH handshake latency' },
                           { step: 'snapshot_memory', avg: 8900, note: 'I/O dependent' },
                           { step: 'collect_logs', avg: 1800, note: 'Log volume variance' }
                         ] as b}
                            <div class="bg-surface-1 border border-border-primary p-3 rounded-sm flex flex-col gap-1">
                               <span class="text-[9px] font-mono text-text-muted uppercase tracking-tighter">{b.step}</span>
                               <span class="text-xl font-mono font-bold text-alert-high italic">{durationLabel(b.avg)}</span>
                               <span class="text-[8px] font-mono text-text-muted opacity-60 uppercase">{b.note}</span>
                            </div>
                         {/each}
                      </div>
                   </div>

                   <div class="bg-surface-1 border border-border-primary rounded-sm overflow-hidden shadow-premium">
                      <div class="p-3 bg-surface-2 border-b border-border-primary text-[10px] font-bold uppercase tracking-widest text-text-muted">
                        Tactical Optimization Roadmap
                      </div>
                      <div class="p-0">
                         {#each [
                           'Pre-warm agent connections for isolate_host to reduce SSH handshake latency',
                           'Implement parallel step execution for non-dependent data shards',
                           'Add step-level timeout configuration to prevent runaway playbook threads',
                           'Cache credentials in secure agent memory to avoid vault round-trips'
                         ] as rec, i}
                            <div class="p-4 border-b border-border-subtle last:border-0 flex gap-4 items-start group hover:bg-surface-2 transition-colors">
                               <span class="text-xs font-black text-accent-primary italic">0{i+1}</span>
                               <p class="text-[11px] text-text-secondary font-bold group-hover:text-text-heading transition-colors">{rec}</p>
                            </div>
                         {/each}
                      </div>
                   </div>
                </div>
              {/if}
            </div>
        </div>

        <!-- RIGHT: ADVISORY SIDEBAR -->
        <div class="w-80 bg-surface-1 flex flex-col shrink-0">
            <div class="px-3 py-2 bg-surface-2 border-b border-border-primary flex items-center gap-2">
                <Target size={14} class="text-status-online" />
                <span class="text-[9px] font-mono font-bold uppercase tracking-widest text-text-heading">SLA Compliance</span>
            </div>
            
            <div class="p-4 space-y-6">
                <div class="space-y-4">
                  <div class="text-[9px] font-mono font-bold text-text-muted uppercase tracking-widest border-b border-border-subtle pb-2">Cluster Metrics</div>
                  {#each [
                    { name: 'AUTO_REMEDIATION', val: '92%', color: 'status-online' },
                    { name: 'MTTR_P1_INCIDENTS', val: '14m 2s', color: 'status-online' },
                    { name: 'ORCHESTRATION_LOAD', val: 'LOW', color: 'accent-primary' },
                    { name: 'EXECUTOR_HEALTH', val: 'STABLE', color: 'status-online' }
                  ] as metric}
                    <div class="flex justify-between items-center text-[10px] font-mono">
                      <span class="text-text-muted uppercase tracking-tight">{metric.name}</span>
                      <span class="font-bold text-{metric.color} italic">{metric.val}</span>
                    </div>
                  {/each}
                </div>

                <div class="pt-4 border-t border-border-primary space-y-4">
                    <span class="text-[9px] font-mono font-bold text-text-muted uppercase tracking-widest">Active Orchestrators</span>
                    {#each [1,2,3] as i}
                       <div class="bg-surface-2 border border-border-primary p-3 rounded-sm space-y-1 group hover:border-accent-primary cursor-pointer transition-colors">
                          <div class="flex items-center justify-between">
                             <div class="flex items-center gap-2">
                                <Activity size={12} class="text-status-online" />
                                <span class="text-[10px] font-bold text-text-heading uppercase tracking-tighter">NODE_SOAR_0{i}</span>
                             </div>
                             <div class="w-1.5 h-1.5 rounded-full bg-status-online animate-pulse"></div>
                          </div>
                          <p class="text-[8px] text-text-muted font-mono leading-relaxed opacity-60">
                             Shard processing active. Currently managing {Math.floor(Math.random()*12)} threads.
                          </p>
                       </div>
                    {/each}
                </div>
            </div>

            <div class="mt-auto border-t border-border-primary p-4 bg-surface-2">
                 <div class="flex items-center justify-between mb-2">
                    <span class="text-[9px] font-mono font-bold text-text-muted uppercase tracking-widest">Logic Depth</span>
                    <Badge variant="accent" size="xs">L7_AWARE</Badge>
                 </div>
                 <div class="text-[8px] font-mono text-text-muted space-y-1 opacity-60">
                    <div>Engine: V8_ORCHESTRATOR</div>
                    <div>Schema: OBLIVRA_SOAR_v1.4</div>
                    <div>Integrations: 142 Active</div>
                 </div>
            </div>
        </div>
    </div>

    <!-- STATUS BAR -->
    <div class="bg-surface-2 border-t border-border-primary px-3 py-1 flex items-center gap-4 text-[8px] font-mono text-text-muted shrink-0 uppercase tracking-widest">
        <div class="flex items-center gap-1.5">
            <div class="w-1 h-1 rounded-full bg-status-online"></div>
            <span>EXECUTION_PLANE:</span>
            <span class="text-status-online font-bold italic">OPTIMIZED</span>
        </div>
        <span class="text-border-primary opacity-30">|</span>
        <div class="flex items-center gap-1.5">
            <span>ORCHESTRATOR_L7:</span>
            <span class="text-status-online font-bold italic">NOMINAL</span>
        </div>
        <span class="text-border-primary opacity-30">|</span>
        <div class="flex items-center gap-1.5">
            <span>PIPELINE_LATENCY:</span>
            <span class="text-accent-primary font-bold italic">14ms</span>
        </div>
        <div class="ml-auto opacity-40">OBLIVRA_SOAR_SUBSTRATE v1.4.2</div>
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
