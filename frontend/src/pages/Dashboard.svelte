<!--
  OBLIVRA — SOC Dashboard (Svelte 5)
  Tactical Command Hub. All numbers are computed from real stores —
  alertStore, diagnosticsStore, agentStore. No hardcoded SLAs or
  fabricated risk scores.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { RefreshCw, Terminal, AlertTriangle, Zap, Database, Server, ShieldCheck } from 'lucide-svelte';
  import { alertStore } from '@lib/stores/alerts.svelte';
  import { diagnosticsStore } from '@lib/stores/diagnostics.svelte';
  import { agentStore } from '@lib/stores/agent.svelte';
  import { KPI, Badge, Button, PageLayout, DataTable, PopOutButton, ActivityFeed } from '@components/ui';
  import { push } from '@lib/router.svelte';

  // Frontend-tracked uptime — backend has no per-session uptime RPC, so we
  // measure from the moment Dashboard mounted. Resets on hard reload.
  let mountedAt = $state<Date | null>(null);
  let now = $state(new Date());
  let peakEPS = $state(0);

  // Risk score: weighted blend of severity counts. Caps at 10.0.
  // critical=4, high=2, medium=1; we divide by 5 so 5 critical incidents
  // saturate near 4.0 and a single critical reads as 0.8 — these scales
  // were chosen to align with operator intuition about "code red" thresholds.
  const riskScore = $derived.by(() => {
    const a = alertStore.alerts ?? [];
    const c = a.filter((x) => x.severity === 'critical').length;
    const h = a.filter((x) => x.severity === 'high').length;
    const m = a.filter((x) => x.severity === 'medium').length;
    const raw = (4 * c + 2 * h + m) / 5;
    return Math.min(raw, 10);
  });

  const stats = $derived({
    total:    (alertStore.alerts ?? []).length,
    critical: (alertStore.alerts ?? []).filter((a) => a.severity === 'critical').length,
    high:     (alertStore.alerts ?? []).filter((a) => a.severity === 'high').length,
    eps:      diagnosticsStore.snapshot?.ingest.current_eps ?? 0,
    health:   diagnosticsStore.snapshot?.health_grade ?? 'PENDING',
  });

  // Track peak EPS over the lifetime of this Dashboard instance.
  $effect(() => {
    const cur = diagnosticsStore.snapshot?.ingest.current_eps ?? 0;
    if (cur > peakEPS) peakEPS = cur;
  });

  function fmtUptime(): string {
    if (!mountedAt) return '00:00:00';
    const diff = Math.floor((now.getTime() - mountedAt.getTime()) / 1000);
    const h = String(Math.floor(diff / 3600)).padStart(2, '0');
    const m = String(Math.floor((diff % 3600) / 60)).padStart(2, '0');
    const s = String(diff % 60).padStart(2, '0');
    return `${h}:${m}:${s}`;
  }
  function fmtEPS(n: number): string {
    if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
    if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
    return n.toLocaleString();
  }

  // Engine-load gauges driven by real diagnostics:
  //   correlation = buffer fill (proxy for back-pressure on detection rules)
  //   ingest      = ingest target hit rate
  //   query       = clamp(p99_query_ms / 1000) so a 1s P99 reads as 100%.
  const engineLoad = $derived.by(() => {
    const s = diagnosticsStore.snapshot;
    if (!s) return { correlation: 0, ingest: 0, query: 0 };
    return {
      correlation: Math.min(100, Math.round(s.ingest.buffer_fill_pct ?? 0)),
      ingest: Math.min(100, Math.round((s.ingest.percent_of_target ?? 0) * 100)),
      query: Math.min(100, Math.round(((s.query.p99_query_ms ?? 0) / 1000) * 100)),
    };
  });

  // Mesh integrity — fleet count drives "core nodes". Vault shards we
  // can't enumerate yet (no Wails RPC); show "—" rather than fake "102/102".
  const fleetCounts = $derived.by(() => {
    const total = agentStore.agents?.length ?? 0;
    const online = (agentStore.agents ?? []).filter((a) => a.status === 'online' || a.status === 'active').length;
    return { online, total };
  });

  let tickTimer: ReturnType<typeof setInterval> | null = null;

  onMount(() => {
    mountedAt = new Date();
    diagnosticsStore.init();
    if (typeof alertStore.init === 'function') alertStore.init();
    if (typeof agentStore.init === 'function') agentStore.init();
    tickTimer = setInterval(() => (now = new Date()), 1000);
    return () => {
      if (tickTimer) clearInterval(tickTimer);
    };
  });

  // Risk variant: cosmetic colouring from the computed score.
  const riskVariant = $derived.by(() => {
    if (riskScore >= 7) return 'critical';
    if (riskScore >= 4) return 'warning';
    return 'success';
  });
</script>

<PageLayout title="Tactical Command Hub" subtitle="Real-time sovereign security posture and telemetry orchestration">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm" icon={RefreshCw} onclick={() => { diagnosticsStore.init(); alertStore.init?.(); }}>SYNC MESH</Button>
      <Button variant="primary"   size="sm" icon={Terminal} onclick={() => push('/siem-search')}>OQL TERMINAL</Button>
      <Button variant="cta"       size="sm" icon={Zap}      onclick={() => push('/war-mode')}>WAR MODE</Button>
    </div>
    <PopOutButton route="/" title="Dashboard" />
  {/snippet}

  <div class="flex flex-col h-full gap-4">
    <!-- CORE KPI STRIP -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4 shrink-0">
      <KPI label="Global Risk Score"
           value={riskScore.toFixed(1)}
           trend={riskScore >= 5 ? 'up' : 'down'}
           variant={riskVariant} />
      <KPI label="Active Threats"
           value={stats.total.toString()}
           sublabel="{stats.critical} critical · {stats.high} high" />
      <KPI label="Ingest Rate (EPS)"
           value={fmtEPS(stats.eps)}
           sublabel={peakEPS > 0 ? `Peak: ${fmtEPS(peakEPS)} EPS` : 'Peak: tracking…'} />
      <KPI label="Platform Health"
           value={stats.health}
           variant={stats.health.startsWith('A') ? 'success' : stats.health === 'PENDING' ? 'muted' : 'warning'} />
      <KPI label="Session Uptime"
           value={fmtUptime()}
           sublabel="Since dashboard mount" />
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
                    <Button variant="danger"    size="xs" onclick={() => push('/alerts?severity=critical')}>CRITICAL ({stats.critical})</Button>
                    <Button variant="secondary" size="xs" onclick={() => push('/alerts')}>ALL EVENTS</Button>
                </div>
            </div>

            <div class="flex-1 overflow-auto mask-fade-bottom">
                <DataTable
                    data={alertStore.alerts}
                    columns={[
                        { key: 'timestamp', label: 'TIMESTAMP', width: '140px' },
                        { key: 'severity',  label: 'SEV',       width: '80px' },
                        { key: 'title',     label: 'DETECTION_LOGIC' },
                        { key: 'host',      label: 'SOURCE',    width: '120px' },
                        { key: 'status',    label: 'STATUS',    width: '100px' }
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
                    <Badge variant={stats.health.startsWith('A') ? 'success' : 'muted'} size="xs">
                        {stats.health.startsWith('A') ? 'SECURE' : stats.health}
                    </Badge>
                </div>

                <div class="space-y-3">
                    <div class="flex items-center justify-between">
                        <div class="flex items-center gap-2">
                            <Server size={12} class="text-text-muted" />
                            <span class="text-[10px] font-mono text-text-secondary">Fleet Online</span>
                        </div>
                        <span class="text-[10px] font-mono font-bold {fleetCounts.online === fleetCounts.total ? 'text-success' : 'text-warning'}">
                            {fleetCounts.online}/{fleetCounts.total || '—'}
                        </span>
                    </div>
                    <div class="flex items-center justify-between">
                        <div class="flex items-center gap-2">
                            <Database size={12} class="text-text-muted" />
                            <span class="text-[10px] font-mono text-text-secondary">Buffer Fill</span>
                        </div>
                        <span class="text-[10px] font-mono font-bold {(diagnosticsStore.snapshot?.ingest.buffer_fill_pct ?? 0) > 80 ? 'text-warning' : 'text-success'}">
                            {Math.round(diagnosticsStore.snapshot?.ingest.buffer_fill_pct ?? 0)}%
                        </span>
                    </div>
                    <div class="flex items-center justify-between">
                        <div class="flex items-center gap-2">
                            <RefreshCw size={12} class="text-text-muted {diagnosticsStore.connected ? 'animate-spin-slow' : ''}" />
                            <span class="text-[10px] font-mono text-text-secondary">Diag Stream</span>
                        </div>
                        <span class="text-[10px] font-mono font-bold {diagnosticsStore.connected ? 'text-accent' : 'text-text-muted'}">
                            {diagnosticsStore.connected ? 'Active' : 'Pending'}
                        </span>
                    </div>
                </div>
            </div>

            <!-- ENGINE TELEMETRY (real %s from diagnostics snapshot) -->
            <div class="bg-surface-2 border border-border-primary rounded-sm flex flex-col min-h-0 shadow-premium overflow-hidden shrink-0">
                <div class="flex items-center justify-between p-3 border-b border-border-primary">
                    <div class="flex items-center gap-2">
                        <Zap size={14} class="text-accent" />
                        <span class="text-[9px] font-mono font-bold text-text-muted uppercase tracking-widest">Engine Load</span>
                    </div>
                </div>
                <div class="p-4 flex flex-col gap-6">
                    <div class="space-y-2">
                        <div class="flex justify-between text-[8px] font-mono uppercase text-text-muted">
                            <span>Correlation Buffer</span>
                            <span>{engineLoad.correlation}%</span>
                        </div>
                        <div class="h-1 bg-surface-3 rounded-full overflow-hidden">
                            <div class="h-full {engineLoad.correlation > 80 ? 'bg-warning' : 'bg-accent'}" style="width: {engineLoad.correlation}%"></div>
                        </div>
                    </div>
                    <div class="space-y-2">
                        <div class="flex justify-between text-[8px] font-mono uppercase text-text-muted">
                            <span>Ingest Pipeline</span>
                            <span>{engineLoad.ingest}%</span>
                        </div>
                        <div class="h-1 bg-surface-3 rounded-full overflow-hidden">
                            <div class="h-full {engineLoad.ingest > 90 ? 'bg-success' : engineLoad.ingest > 50 ? 'bg-warning' : 'bg-accent'}" style="width: {engineLoad.ingest}%"></div>
                        </div>
                    </div>
                    <div class="space-y-2">
                        <div class="flex justify-between text-[8px] font-mono uppercase text-text-muted">
                            <span>Query P99</span>
                            <span>{engineLoad.query}%</span>
                        </div>
                        <div class="h-1 bg-surface-3 rounded-full overflow-hidden">
                            <div class="h-full {engineLoad.query > 50 ? 'bg-warning' : 'bg-success'}" style="width: {engineLoad.query}%"></div>
                        </div>
                    </div>
                </div>
            </div>

            <div class="flex-1 min-h-0">
                <ActivityFeed limit={40} />
            </div>
        </div>
    </div>
  </div>
</PageLayout>
