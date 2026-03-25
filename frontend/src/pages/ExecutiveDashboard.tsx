import { Component, createSignal, onMount, onCleanup, For, Show } from 'solid-js';
import { 
    PageLayout, 
    KPIGrid, 
    KPI, 
    Table, 
    Badge, 
    Panel, 
    SectionHeader, 
    Notice,
    Progress,
    LoadingState,
    normalizeSeverity,
    Column
} from '@components/ui';
import '../styles/executive.css';

interface PlatformMetric {
    label: string;
    value: string;
    trend: 'up' | 'down' | 'stable';
    context: string;
    loading?: boolean;
}

interface ThreatPosture {
    score: number;
    level: string;
    color: string;
}

interface RetentionPolicy {
    category: string;
    description: string;
    retain_days: number;
    is_enabled: boolean;
}

interface LifecycleStats {
    last_run_at: string;
    next_run_at: string;
    total_rows_purged: number;
    legal_hold_active: boolean;
}

function fmtCount(n: number): string {
    if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
    if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
    return String(n);
}

function fmtUptime(startIso: string): string {
    if (!startIso) return '—';
    const ms = Date.now() - new Date(startIso).getTime();
    const days = Math.floor(ms / 86_400_000);
    const hours = Math.floor((ms % 86_400_000) / 3_600_000);
    if (days > 0) return `${days}d ${hours}h`;
    const mins = Math.floor((ms % 3_600_000) / 60_000);
    return hours > 0 ? `${hours}h ${mins}m` : `${mins}m`;
}

export const ExecutiveDashboard: Component = () => {
    const [metrics, setMetrics] = createSignal<PlatformMetric[]>([
        { label: 'PLATFORM UPTIME', value: '—', trend: 'stable', context: 'Since last restart', loading: true },
        { label: 'EVENTS INGESTED', value: '—', trend: 'up', context: 'Total processed', loading: true },
        { label: 'ACTIVE HOSTS', value: '—', trend: 'up', context: 'Connected fleet', loading: true },
        { label: 'MEAN DETECT TIME', value: '~4s', trend: 'stable', context: 'Avg alert latency (est)' },
        { label: 'OPEN INCIDENTS', value: '—', trend: 'down', context: 'Unresolved', loading: true },
        { label: 'COMPLIANCE SCORE', value: '—', trend: 'up', context: 'Across all frameworks', loading: true },
        { label: 'MFA COVERAGE', value: '—', trend: 'stable', context: 'Active users', loading: true },
        { label: 'DETECTION RULES', value: '—', trend: 'up', context: 'Active YAML rules', loading: true },
    ]);
    const [posture, setPosture] = createSignal<ThreatPosture>({ score: 0, level: 'LOADING…', color: 'var(--text-muted)' });
    const [policies, setPolicies] = createSignal<RetentionPolicy[]>([]);
    const [lifecycleStats, setLifecycleStats] = createSignal<LifecycleStats | null>(null);
    const [postureComponents, setPostureComponents] = createSignal<{ name: string; score: number; weight: number }[]>([
        { name: 'Vault Integrity',      score: 0, weight: 0.25 },
        { name: 'Detection Engine',     score: 0, weight: 0.15 },
        { name: 'Policy Verification',  score: 0, weight: 0.15 },
        { name: 'Audit Ledger',         score: 0, weight: 0.25 },
        { name: 'Runtime Attestation',  score: 0, weight: 0.20 },
    ]);
    const [complianceResults, setComplianceResults] = createSignal<{ name: string; status: string }[]>([
        { name: 'PCI-DSS',    status: 'unknown' },
        { name: 'NIST 800-53',status: 'unknown' },
        { name: 'ISO 27001',  status: 'unknown' },
        { name: 'GDPR',       status: 'unknown' },
        { name: 'HIPAA',      status: 'unknown' },
        { name: 'SOC 2',      status: 'unknown' },
    ]);

    let refreshTimer: ReturnType<typeof setInterval> | null = null;

    async function loadAll() {
        try {
            const svc = (window as any).go?.services as any;
            if (!svc) return;

            const ingestMetrics = await svc.IngestService?.GetMetrics().catch(() => null);
            const eventsStr = fmtCount(ingestMetrics?.total_processed ?? 0);

            const hosts = await svc.HostService?.ListHosts().catch(() => null);
            const hostCount = Array.isArray(hosts) ? hosts.length : 0;

            const incidents = await svc.AlertingService?.ListIncidents('New', 500).catch(() => null);
            const openCount = Array.isArray(incidents) ? incidents.length : 0;

            const rules = await svc.AlertingService?.GetDetectionRules().catch(() => null);
            const ruleCount = Array.isArray(rules) ? rules.length : 0;

            const idStats = await svc.IdentityService?.GetSecurityStats().catch(() => null);
            let mfaStr = '—';
            if (idStats && idStats.total_users > 0) {
                mfaStr = `${Math.round((idStats.mfa_passive / idStats.total_users) * 100)}%`;
            }

            const trustMetrics = await svc.TrustService?.GetTrustDriftMetrics().catch(() => null);
            const trustScore = Math.round(trustMetrics?.current_score ?? 0);

            const pillarScores = await svc.TrustService?.GetPillarScores?.().catch(() => null);
            if (pillarScores) {
                const weights: Record<string, number> = {
                    'Vault Integrity': 25, 'Attestation State': 20, 'Detection Rules': 15,
                    'Policy Engine': 15, 'Audit Trail': 25,
                };
                const nameMap: Record<string, string> = {
                    'Vault Integrity': 'Vault Integrity',
                    'Attestation State': 'Runtime Attestation',
                    'Detection Rules': 'Detection Engine',
                    'Policy Engine': 'Policy Verification',
                    'Audit Trail': 'Audit Ledger',
                };
                setPostureComponents(prev => prev.map(comp => {
                    const backendKey = Object.keys(nameMap).find(k => nameMap[k] === comp.name);
                    if (!backendKey) return comp;
                    const raw: number = pillarScores[backendKey] ?? 0;
                    const max = weights[backendKey] ?? 25;
                    return { ...comp, score: Math.round((raw / max) * 100) };
                }));
            }

            const postureLevel = trustScore >= 90 ? 'SOVEREIGN CONFIDENCE'
                : trustScore >= 75 ? 'ELEVATED READINESS'
                : trustScore >= 50 ? 'CAUTIONARY POSTURE'
                : 'CRITICAL ALERT';
            const postureColor = trustScore >= 90 ? 'var(--status-online)'
                : trustScore >= 75 ? 'var(--status-degraded)'
                : 'var(--alert-critical)';
            setPosture({ score: trustScore, level: postureLevel, color: postureColor });

            const packDefs = await svc.ComplianceService?.ListCompliancePacks().catch(() => null);
            if (Array.isArray(packDefs)) {
                const packResults = await Promise.allSettled(
                    packDefs.slice(0, 6).map((p: any) => svc.ComplianceService.EvaluatePack(p.id).catch(() => null))
                );
                const badges = packDefs.slice(0, 6).map((p: any, i: number) => {
                    const res = packResults[i];
                    let status = 'unknown';
                    if (res.status === 'fulfilled' && res.value) {
                        status = res.value.pass_rate >= 0.95 ? 'pass' : res.value.pass_rate >= 0.7 ? 'warn' : 'fail';
                    }
                    return { name: p.name ?? p.id, status };
                });
                setComplianceResults(badges);
            }

            const passCount = complianceResults().filter(b => b.status === 'pass').length;
            const complianceScore = complianceResults().length > 0 ? Math.round((passCount / complianceResults().length) * 100) : 0;

            const obsStatus = await svc.ObservabilityService?.GetObservabilityStatus().catch(() => null);
            const uptimeStr = obsStatus?.start_time ? fmtUptime(obsStatus.start_time) : '—';

            setMetrics([
                { label: 'PLATFORM UPTIME',  value: uptimeStr,              trend: 'stable', context: 'Since last restart' },
                { label: 'EVENTS INGESTED',  value: eventsStr,              trend: 'up',     context: 'Total processed' },
                { label: 'ACTIVE HOSTS',     value: String(hostCount),      trend: 'up',     context: 'In fleet' },
                { label: 'MEAN DETECT TIME', value: '~4s',                  trend: 'stable', context: 'Avg latency' },
                { label: 'OPEN INCIDENTS',   value: String(openCount),      trend: openCount > 5 ? 'up' : 'down', context: 'Unresolved' },
                { label: 'COMPLIANCE SCORE', value: `${complianceScore}%`,  trend: 'up',     context: 'Global frameworks' },
                { label: 'MFA COVERAGE',     value: mfaStr,                 trend: 'stable', context: 'Active users' },
                { label: 'DETECTION RULES',  value: String(ruleCount),      trend: 'up',     context: 'Active YAML' },
            ]);

            const lPolicies = await svc.DataLifecycleService?.GetPolicies().catch(() => null);
            if (Array.isArray(lPolicies)) setPolicies(lPolicies);
            const lStats = await svc.DataLifecycleService?.GetStats().catch(() => null);
            if (lStats) setLifecycleStats(lStats);

        } catch (e) {
            console.error('[EXEC] Failed to load executive data:', e);
        }
    }

    onMount(() => {
        loadAll();
        refreshTimer = setInterval(loadAll, 30_000);
    });

    onCleanup(() => {
        if (refreshTimer !== null) clearInterval(refreshTimer);
    });

    const retentionColumns: Column<RetentionPolicy>[] = [
        { 
            key: 'category', 
            label: 'Category',
            render: (p) => (
                <div>
                    <div class="retention-category">{p.category.replace(/_/g, ' ')}</div>
                    <div class="retention-desc">{p.description}</div>
                </div>
            )
        },
        { 
            key: 'retain_days', 
            label: 'Retention',
            width: '80px',
            mono: true,
            render: (p) => p.retain_days > 0 ? `${p.retain_days}d` : '∞'
        },
        { 
            key: 'is_enabled', 
            label: 'Status',
            width: '100px',
            render: (p) => (
                <Badge color={p.is_enabled ? 'green' : 'gray'}>
                    {p.is_enabled ? 'ENFORCED' : 'DISABLED'}
                </Badge>
            )
        }
    ];

    return (
        <PageLayout
            title="Executive Overview"
            subtitle="SOVEREIGN TERMINAL — PLATFORM_INTELLIGENCE_SUMMARY"
            actions={
                <div class="exec-timestamp">
                    {new Date().toLocaleDateString('en-US', { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' })}
                </div>
            }
        >
            {/* Threat Posture Banner */}
            <div class="posture-banner" style={{ 'border-left-color': posture().color }}>
                <div class="posture-score" style={{ color: posture().color }}>{posture().score}</div>
                <div class="posture-info">
                    <div class="posture-level" style={{ color: posture().color }}>{posture().level}</div>
                    <div class="posture-desc">
                        Platform Trust Index — composite of vault integrity, detection health,
                        policy verification, and audit ledger consistency
                    </div>
                </div>
            </div>

            {/* KPI Grid */}
            <KPIGrid cols={4}>
                <For each={metrics()}>
                    {(m) => (
                        <KPI 
                            label={m.label} 
                            value={m.value} 
                            delta={m.trend === 'up' ? 1 : m.trend === 'down' ? -1 : undefined}
                            deltaLabel=""
                            color={m.loading ? 'var(--text-muted)' : undefined}
                            class={m.loading ? 'kpi-loading' : ''}
                        />
                    )}
                </For>
            </KPIGrid>

            <div class="exec-columns">
                <Panel title="DATA LIFECYCLE & RETENTION" noPadding>
                    <div style="padding: 12px 16px; display: flex; align-items: center; gap: 12px;">
                        <Show when={lifecycleStats()?.legal_hold_active}>
                            <Badge color="orange">⚠ LEGAL HOLD ACTIVE</Badge>
                        </Show>
                        <Show when={!lifecycleStats()?.legal_hold_active}>
                            <Badge color="green">● NORMAL OPERATIONS</Badge>
                        </Show>
                        <Show when={lifecycleStats()?.last_run_at}>
                            <span class="lifecycle-last-run">
                                Last purge: {new Date(lifecycleStats()!.last_run_at).toLocaleDateString()}
                                {' '}({fmtCount(lifecycleStats()!.total_rows_purged)} rows)
                            </span>
                        </Show>
                    </div>
                    <Table columns={retentionColumns} data={policies()} striped />
                </Panel>

                <Panel title="SECURITY POSTURE BREAKDOWN">
                    <div class="posture-grid">
                        <For each={postureComponents()}>
                            {(comp) => (
                                <div class="posture-row">
                                    <div class="posture-name">{comp.name}</div>
                                    <Progress 
                                        value={comp.score} 
                                        color={comp.score >= 90 ? 'green' : comp.score >= 70 ? 'orange' : 'red'}
                                        class="posture-bar-wrap"
                                    />
                                    <div class="posture-score-label">{comp.score}%</div>
                                    <div class="posture-weight">×{comp.weight}</div>
                                </div>
                            )}
                        </For>
                    </div>

                    <SectionHeader class="compliance-header">GLOBAL COMPLIANCE STATUS</SectionHeader>
                    <div class="compliance-badges">
                        <For each={complianceResults()}>
                            {(fw) => (
                                <Badge color={fw.status === 'pass' ? 'green' : fw.status === 'warn' ? 'orange' : fw.status === 'fail' ? 'red' : 'gray'}>
                                    {fw.status === 'pass' ? '✓' : fw.status === 'warn' ? '!' : fw.status === 'fail' ? '✗' : '?'} {fw.name}
                                </Badge>
                            )}
                        </For>
                    </div>
                </Panel>
            </div>
        </PageLayout>
    );
};
