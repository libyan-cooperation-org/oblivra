<!--
  OBLIVRA — SOC Dashboard (Svelte 5)
  Tactical Command Hub: High-density platform status and mission critical telemetry.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { RefreshCw, Terminal, AlertTriangle, Zap, Database, Server, ShieldCheck } from 'lucide-svelte';
  import { alertStore } from '@lib/stores/alerts.svelte';
  import { diagnosticsStore } from '@lib/stores/diagnostics.svelte';
  import { KPI, Badge, Button, PageLayout, DataTable } from '@components/ui';

  // Stats derived from stores
  const stats = $derived({
    total: alertStore.alerts.length,
    critical: alertStore.alerts.filter(a => a.severity === 'critical').length,
    high: alertStore.alerts.filter(a => a.severity === 'high').length,
    eps: diagnosticsStore.snapshot?.ingest.current_eps ?? 0,
    health: diagnosticsStore.snapshot?.health_grade ?? 'PENDING',
    mttr: '0.0m',
    uptime: '00:00:00'
  });

  onMount(() => {
    diagnosticsStore.init();
    alertStore.init();
  });
</script>

<PageLayout title="Tactical Command Hub" subtitle="Real-time sovereign security posture and telemetry orchestration">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm" icon={RefreshCw}>SYNC MESH</Button>
      <Button variant="primary" size="sm" icon={Terminal}>OQL TERMINAL</Button>
      <Button variant="cta" size="sm" icon={Zap}>WAR MODE</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-4">
    <!-- CORE KPI STRIP -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4 shrink-0">
      <KPI label="Global Risk Score" value={stats.critical > 5 ? '8.4' : '2.1'} trend={stats.critical > 5 ? 'up' : 'down'} variant={stats.critical > 5 ? 'critical' : 'success'} />
      <KPI label="Active Threats" value={stats.total.toString()} trend="up" trendValue="+14%" />
      <KPI label="Ingest Rate (EPS)" value={stats.eps.toLocaleString()} sublabel="Peak: 1.8M EPS" />
      <KPI label="Platform Health" value={stats.health} variant={stats.health.startsWith('A') ? 'success' : 'warning'} />
      <KPI label="Sovereign Uptime" value={stats.uptime} sublabel="99.9999% SLA" />
    </div>

    <!-- MAIN GRID -->
    <div class="flex-1 grid grid-cols-12 gap-4 min-h-0">
        <!-- ALERT FEED -->
        <div class="col-span-12 lg:col-span-8 flex flex-col bg-surface-1 border border-border-primary rounded-sm min-h-0 shadow-premium">
            <div class="flex items-center justify-between p-3 border-b border-border-primary bg-surface-2 shrink-0">
                <div class="flex items-center gap-2">
                    <AlertTriangle size={14} class="text-warning" />
                    <span class="text-[10px] font-bold text-text-heading uppercase tracking-widest">Priority Alert Feed</span>
                </div>
                <div class="flex gap-2">
                    <Button variant="danger" size="xs">CRITICAL ({stats.critical})</Button>
                    <Button variant="secondary" size="xs">ALL EVENTS</Button>
                </div>
            </div>
            
            <div class="flex-1 overflow-auto mask-fade-bottom">
                <DataTable 
                    data={alertStore.alerts} 
                    columns={[
                        { key: 'timestamp', label: 'TIMESTAMP', width: '140px' },
                        { key: 'severity', label: 'SEV', width: '80px' },
                        { key: 'title', label: 'DETECTION_LOGIC' },
                        { key: 'host', label: 'SOURCE', width: '120px' },
                        { key: 'status', label: 'STATUS', width: '100px' }
                    ]}
                    compact
                >
                    {#snippet render({ col, row })}
                        {#if col.key === 'timestamp'}
                            <span class="text-[9px] font-mono text-text-muted tabular-nums">{row.timestamp}</span>
                        {:else if col.key === 'severity'}
                            <Badge variant={row.severity === 'critical' ? 'critical' : row.severity === 'high' ? 'warning' : 'info'} size="xs" class="w-full justify-center">
                                {row.severity}
                            </Badge>
                        {:else if col.key === 'title'}
                            <span class="text-[10px] font-bold text-text-secondary line-clamp-1">{row.title}</span>
                        {:else if col.key === 'host'}
                            <span class="text-[10px] font-mono text-accent">{row.host}</span>
                        {:else if col.key === 'status'}
                            <Badge variant="muted" size="xs" dot>{row.status}</Badge>
                        {/if}
                    {/snippet}
                </DataTable>
            </div>
        </div>

        <!-- SIDEBAR: SYSTEM STATUS -->
        <div class="col-span-12 lg:col-span-4 flex flex-col gap-4 min-h-0">
            <!-- MESH STATUS -->
            <div class="bg-surface-2 border border-border-primary rounded-sm p-4 space-y-4 shadow-premium">
                <div class="flex items-center justify-between border-b border-border-primary pb-2">
                    <div class="flex items-center gap-2">
                        <ShieldCheck size={14} class="text-success" />
                        <span class="text-[9px] font-mono font-bold text-text-muted uppercase tracking-widest">Mesh Integrity</span>
                    </div>
                    <Badge variant="success" size="xs">SECURE</Badge>
                </div>
                
                <div class="space-y-3">
                    <div class="flex items-center justify-between">
                        <div class="flex items-center gap-2">
                            <Server size={12} class="text-text-muted" />
                            <span class="text-[10px] font-mono text-text-secondary">Core Nodes</span>
                        </div>
                        <span class="text-[10px] font-mono font-bold text-success">14/14</span>
                    </div>
                    <div class="flex items-center justify-between">
                        <div class="flex items-center gap-2">
                            <Database size={12} class="text-text-muted" />
                            <span class="text-[10px] font-mono text-text-secondary">Vault Shards</span>
                        </div>
                        <span class="text-[10px] font-mono font-bold text-success">102/102</span>
                    </div>
                    <div class="flex items-center justify-between">
                        <div class="flex items-center gap-2">
                            <RefreshCw size={12} class="text-text-muted animate-spin-slow" />
                            <span class="text-[10px] font-mono text-text-secondary">Peer Sync</span>
                        </div>
                        <span class="text-[10px] font-mono font-bold text-accent">Active</span>
                    </div>
                </div>
            </div>

            <!-- ENGINE TELEMETRY -->
            <div class="bg-surface-2 border border-border-primary rounded-sm flex-1 flex flex-col min-h-0 shadow-premium overflow-hidden">
                <div class="flex items-center justify-between p-3 border-b border-border-primary">
                    <div class="flex items-center gap-2">
                        <Zap size={14} class="text-accent" />
                        <span class="text-[9px] font-mono font-bold text-text-muted uppercase tracking-widest">Engine Load</span>
                    </div>
                </div>
                <div class="p-4 flex flex-col gap-6 flex-1">
                    <div class="space-y-2">
                        <div class="flex justify-between text-[8px] font-mono uppercase text-text-muted">
                            <span>Correlation Engine</span>
                            <span>42%</span>
                        </div>
                        <div class="h-1 bg-surface-3 rounded-full overflow-hidden">
                            <div class="h-full bg-accent" style="width: 42%"></div>
                        </div>
                    </div>
                    <div class="space-y-2">
                        <div class="flex justify-between text-[8px] font-mono uppercase text-text-muted">
                            <span>Ingest Pipeline</span>
                            <span>68%</span>
                        </div>
                        <div class="h-1 bg-surface-3 rounded-full overflow-hidden">
                            <div class="h-full bg-warning" style="width: 68%"></div>
                        </div>
                    </div>
                    <div class="space-y-2">
                        <div class="flex justify-between text-[8px] font-mono uppercase text-text-muted">
                            <span>Query Parallelism</span>
                            <span>14%</span>
                        </div>
                        <div class="h-1 bg-surface-3 rounded-full overflow-hidden">
                            <div class="h-full bg-success" style="width: 14%"></div>
                        </div>
                    </div>

                    <!-- MIN-GRAPH PLACEHOLDER -->
                    <div class="mt-auto border border-border-primary rounded-sm bg-black/20 h-24 flex items-center justify-center">
                        <div class="flex gap-1 items-end h-12">
                            {#each [40, 60, 45, 90, 65, 30, 50, 80, 40, 60] as h}
                                <div class="w-2 bg-accent/40 rounded-t-xs" style="height: {h}%"></div>
                            {/each}
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>
  </div>
</PageLayout>
