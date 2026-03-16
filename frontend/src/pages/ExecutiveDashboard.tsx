import { Component, createSignal, onMount, onCleanup, For } from 'solid-js';
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

// Format large numbers with appropriate suffix
function fmtCount(n: number): string {
    if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
    if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
    return String(n);
}

// Format uptime seconds as days/hours
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
    const [posture, setPosture] = createSignal<ThreatPosture>({ score: 0, level: 'LOADING…', color: '#8b949e' });
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
            const svc = (window as any).go?.app as any;
            if (!svc) return;

            // ── 1. Ingest metrics ─────────────────────────────────────────────
            const ingestMetrics = await svc.IngestService?.GetMetrics().catch(() => null);
            const totalProcessed: number = ingestMetrics?.total_processed ?? 0;
            const eventsStr = fmtCount(totalProcessed);

            // ── 2. Active hosts ───────────────────────────────────────────────
            const hosts = await svc.HostService?.ListHosts().catch(() => null);
            const hostCount: number = Array.isArray(hosts) ? hosts.length : 0;

            // ── 3. Open incidents ─────────────────────────────────────────────
            const incidents = await svc.AlertingService?.ListIncidents('New', 500).catch(() => null);
            const openCount: number = Array.isArray(incidents) ? incidents.length : 0;

            // ── 4. Detection rules ────────────────────────────────────────────
            const rules = await svc.AlertingService?.GetDetectionRules().catch(() => null);
            const ruleCount: number = Array.isArray(rules) ? rules.length : 0;

            // ── 5. Identity / MFA ─────────────────────────────────────────────
            const idStats = await svc.IdentityService?.GetSecurityStats().catch(() => null);
            let mfaStr = '—';
            if (idStats && typeof idStats.total_users === 'number' && idStats.total_users > 0) {
                const pct = Math.round((idStats.mfa_passive / idStats.total_users) * 100);
                mfaStr = `${pct}%`;
            } else if (idStats && idStats.total_users === 0) {
                mfaStr = 'N/A';
            }

            // ── 6. Trust / posture score ──────────────────────────────────────
            const trustMetrics = await svc.TrustService?.GetTrustDriftMetrics().catch(() => null);
            const trustScore: number = typeof trustMetrics?.current_score === 'number'
                ? Math.round(trustMetrics.current_score)
                : 0;

            // Map trust service pillars to posture component bars
            if (trustMetrics?.pillar_trends) {
                const pillarMap: Record<string, number> = {};
                for (const p of trustMetrics.pillar_trends as any[]) {
                    pillarMap[p.component] = Math.max(0, Math.min(100, Math.round(p.velocity * 100 + 70)));
                }
                // Fetch full pillar scores directly if available
                const pillarScores = await svc.TrustService?.GetPillarScores?.().catch(() => null);
                if (pillarScores) {
                    // Pillar scores are already 0–25/20/15 weighted values; normalise to %
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
            }

            const postureLevel = trustScore >= 90 ? 'SOVEREIGN CONFIDENCE'
                : trustScore >= 75 ? 'ELEVATED READINESS'
                : trustScore >= 50 ? 'CAUTIONARY POSTURE'
                : 'CRITICAL ALERT';
            const postureColor = trustScore >= 90 ? '#3fb950'
                : trustScore >= 75 ? '#d29922'
                : '#f85149';
            setPosture({ score: trustScore, level: postureLevel, color: postureColor });

            // ── 7. Compliance packs ───────────────────────────────────────────
            const packDefs = await svc.ComplianceService?.ListCompliancePacks().catch(() => null);
            if (Array.isArray(packDefs) && packDefs.length > 0) {
                // Evaluate all packs in parallel; cap at 6 for the badge row
                const packResults = await Promise.allSettled(
                    packDefs.slice(0, 6).map((p: any) =>
                        svc.ComplianceService.EvaluatePack(p.id).catch(() => null)
                    )
                );
                const badges = packDefs.slice(0, 6).map((p: any, i: number) => {
                    const res = packResults[i];
                    let status = 'unknown';
                    if (res.status === 'fulfilled' && res.value) {
                        status = res.value.pass_rate >= 0.95 ? 'pass'
                            : res.value.pass_rate >= 0.7 ? 'warn'
                            : 'fail';
                    }
                    return { name: p.name ?? p.id, status };
                });
                setComplianceResults(badges);
            }

            // Derive overall compliance score from badges
            const passCount = complianceResults().filter(b => b.status === 'pass').length;
            const totalBadges = complianceResults().length;
            const complianceScore = totalBadges > 0 ? Math.round((passCount / totalBadges) * 100) : 0;

            // ── 8. Uptime ─────────────────────────────────────────────────────
            const obsStatus = await svc.ObservabilityService?.GetObservabilityStatus().catch(() => null);
            const startIso: string = obsStatus?.start_time ?? '';
            const uptimeStr = startIso ? fmtUptime(startIso) : '—';

            // ── 9. Update metric cards ────────────────────────────────────────
            setMetrics([
                { label: 'PLATFORM UPTIME',  value: uptimeStr,              trend: 'stable', context: 'Since last restart' },
                { label: 'EVENTS INGESTED',  value: eventsStr,              trend: 'up',     context: 'Total processed' },
                { label: 'ACTIVE HOSTS',     value: String(hostCount),      trend: 'up',     context: 'In fleet' },
                { label: 'MEAN DETECT TIME', value: '~4s',                  trend: 'stable', context: 'Avg alert latency (est)' },
                { label: 'OPEN INCIDENTS',   value: String(openCount),      trend: openCount > 5 ? 'up' : 'down', context: 'Unresolved' },
                { label: 'COMPLIANCE SCORE', value: `${complianceScore}%`,  trend: 'up',     context: 'Across all frameworks' },
                { label: 'MFA COVERAGE',     value: mfaStr,                 trend: 'stable', context: 'Active users' },
                { label: 'DETECTION RULES',  value: String(ruleCount),      trend: 'up',     context: 'Active YAML rules' },
            ]);

            // ── 10. Lifecycle data ────────────────────────────────────────────
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
        // Refresh every 30 seconds
        refreshTimer = setInterval(loadAll, 30_000);
    });

    onCleanup(() => {
        if (refreshTimer !== null) clearInterval(refreshTimer);
    });

    const trendIcon  = (t: string) => t === 'up' ? '↑' : t === 'down' ? '↓' : '—';
    const trendColor = (t: string) => t === 'up' ? '#3fb950' : t === 'down' ? '#f85149' : '#8b949e';

    return (
        <div class="exec-dashboard">
            {/* Header */}
            <div class="exec-header">
                <div>
                    <h1>Executive Overview</h1>
                    <span class="exec-subtitle">SOVEREIGN TERMINAL — Platform Intelligence Summary</span>
                </div>
                <div class="exec-timestamp">
                    {new Date().toLocaleDateString('en-US', { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' })}
                </div>
            </div>

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
            <div class="exec-kpi-grid">
                <For each={metrics()}>
                    {(m) => (
                        <div class="kpi-card">
                            <div class="kpi-label">{m.label}</div>
                            <div class="kpi-value">
                                <span class={m.loading ? 'kpi-loading' : ''}>{m.value}</span>
                                <span class="kpi-trend" style={{ color: trendColor(m.trend) }}>
                                    {trendIcon(m.trend)}
                                </span>
                            </div>
                            <div class="kpi-context">{m.context}</div>
                        </div>
                    )}
                </For>
            </div>

            {/* Two Column Layout */}
            <div class="exec-columns">
                {/* Data Lifecycle */}
                <div class="exec-card">
                    <h3>Data Lifecycle & Retention</h3>
                    <div class="lifecycle-status">
                        <span class={`legal-hold-badge ${lifecycleStats()?.legal_hold_active ? 'active' : ''}`}>
                            {lifecycleStats()?.legal_hold_active ? '⚠ LEGAL HOLD ACTIVE' : '● Normal Operations'}
                        </span>
                        {lifecycleStats()?.last_run_at && (
                            <span class="lifecycle-last-run">
                                Last purge: {new Date(lifecycleStats()!.last_run_at).toLocaleDateString()}
                                {' '}({lifecycleStats()!.total_rows_purged.toLocaleString()} rows)
                            </span>
                        )}
                    </div>
                    <table class="retention-table">
                        <thead>
                            <tr>
                                <th>Category</th>
                                <th>Retention</th>
                                <th>Status</th>
                            </tr>
                        </thead>
                        <tbody>
                            <For each={policies()}>
                                {(p) => (
                                    <tr>
                                        <td>
                                            <div class="retention-category">{p.category.replace(/_/g, ' ')}</div>
                                            <div class="retention-desc">{p.description}</div>
                                        </td>
                                        <td class="retention-days">{p.retain_days > 0 ? `${p.retain_days}d` : '∞'}</td>
                                        <td>
                                            <span class={`retention-status ${p.is_enabled ? 'active' : 'disabled'}`}>
                                                {p.is_enabled ? 'ENFORCED' : 'DISABLED'}
                                            </span>
                                        </td>
                                    </tr>
                                )}
                            </For>
                        </tbody>
                    </table>
                </div>

                {/* Security Posture Breakdown */}
                <div class="exec-card">
                    <h3>Security Posture Breakdown</h3>
                    <div class="posture-grid">
                        <For each={postureComponents()}>
                            {(component) => (
                                <div class="posture-row">
                                    <div class="posture-name">{component.name}</div>
                                    <div class="posture-bar-container">
                                        <div
                                            class="posture-bar"
                                            style={{
                                                width: `${component.score}%`,
                                                background: component.score >= 90 ? '#3fb950'
                                                    : component.score >= 70 ? '#d29922'
                                                    : '#f85149',
                                                transition: 'width 0.6s ease',
                                            }}
                                        />
                                    </div>
                                    <div class="posture-score-label">{component.score}%</div>
                                    <div class="posture-weight">×{component.weight}</div>
                                </div>
                            )}
                        </For>
                    </div>

                    <h3 style={{ 'margin-top': '24px' }}>Compliance Status</h3>
                    <div class="compliance-badges">
                        <For each={complianceResults()}>
                            {(fw) => (
                                <span class={`compliance-badge ${fw.status}`}>
                                    {fw.status === 'pass' ? '✓' : fw.status === 'warn' ? '!' : fw.status === 'fail' ? '✗' : '?'} {fw.name}
                                </span>
                            )}
                        </For>
                    </div>
                </div>
            </div>
        </div>
    );
};
