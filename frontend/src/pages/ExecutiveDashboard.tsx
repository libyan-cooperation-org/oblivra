import { Component, createSignal, onMount, For } from 'solid-js';
import '../styles/executive.css';

interface PlatformMetric {
    label: string;
    value: string;
    trend: 'up' | 'down' | 'stable';
    context: string;
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

export const ExecutiveDashboard: Component = () => {
    const [metrics, setMetrics] = createSignal<PlatformMetric[]>([]);
    const [posture, setPosture] = createSignal<ThreatPosture>({ score: 0, level: 'UNKNOWN', color: '#8b949e' });
    const [policies, setPolicies] = createSignal<RetentionPolicy[]>([]);
    const [lifecycleStats, setLifecycleStats] = createSignal<LifecycleStats | null>(null);

    onMount(async () => {
        try {
            // @ts-ignore - Wails runtime bindings
            const svc = (window as any).go?.app as any;

            // Fetch lifecycle data
            if (svc?.DataLifecycleService) {
                const p = await svc.DataLifecycleService.GetPolicies();
                if (p) setPolicies(p);
                const s = await svc.DataLifecycleService.GetStats();
                if (s) setLifecycleStats(s);
            } else {
                setPolicies([
                    { category: 'audit_logs', description: 'Audit trail entries', retain_days: 365, is_enabled: true },
                    { category: 'host_events', description: 'SIEM/host security events', retain_days: 180, is_enabled: true },
                    { category: 'sessions', description: 'SSH session metadata', retain_days: 90, is_enabled: true },
                    { category: 'incidents', description: 'Incident records', retain_days: 730, is_enabled: true },
                    { category: 'recording_frames', description: 'Terminal recording data', retain_days: 30, is_enabled: true },
                ]);
            }
        } catch (e) {
            console.error('[EXEC] Failed to load executive data:', e);
        }

        // Build KPI metrics
        setMetrics([
            { label: 'PLATFORM UPTIME', value: '99.97%', trend: 'stable', context: 'Last 30 days' },
            { label: 'EVENTS INGESTED', value: '12.4M', trend: 'up', context: 'This month' },
            { label: 'ACTIVE HOSTS', value: '847', trend: 'up', context: 'Connected fleet' },
            { label: 'MEAN DETECT TIME', value: '4.2s', trend: 'down', context: 'Avg alert latency' },
            { label: 'OPEN INCIDENTS', value: '3', trend: 'down', context: 'Unresolved' },
            { label: 'COMPLIANCE SCORE', value: '96%', trend: 'up', context: 'Across all frameworks' },
            { label: 'MFA COVERAGE', value: '100%', trend: 'stable', context: 'Active users' },
            { label: 'DETECTION RULES', value: '54', trend: 'up', context: 'Active YAML rules' },
        ]);

        setPosture({ score: 87, level: 'ELEVATED READINESS', color: '#3fb950' });
    });

    const trendIcon = (t: string) => t === 'up' ? '↑' : t === 'down' ? '↓' : '—';
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
                    <div class="posture-desc">Platform Trust Index — composite of vault integrity, detection health, policy verification, and audit ledger consistency</div>
                </div>
            </div>

            {/* KPI Grid */}
            <div class="exec-kpi-grid">
                <For each={metrics()}>
                    {(m) => (
                        <div class="kpi-card">
                            <div class="kpi-label">{m.label}</div>
                            <div class="kpi-value">
                                {m.value}
                                <span class="kpi-trend" style={{ color: trendColor(m.trend) }}>{trendIcon(m.trend)}</span>
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
                        {[
                            { name: 'Vault Integrity', score: 100, weight: 0.25 },
                            { name: 'Detection Engine', score: 92, weight: 0.15 },
                            { name: 'Policy Verification', score: 88, weight: 0.15 },
                            { name: 'Audit Ledger', score: 78, weight: 0.25 },
                            { name: 'Runtime Attestation', score: 72, weight: 0.20 },
                        ].map((component) => (
                            <div class="posture-row">
                                <div class="posture-name">{component.name}</div>
                                <div class="posture-bar-container">
                                    <div class="posture-bar" style={{
                                        width: `${component.score}%`,
                                        background: component.score >= 90 ? '#3fb950' : component.score >= 70 ? '#d29922' : '#f85149'
                                    }} />
                                </div>
                                <div class="posture-score-label">{component.score}%</div>
                                <div class="posture-weight">×{component.weight}</div>
                            </div>
                        ))}
                    </div>

                    <h3 style={{ 'margin-top': '24px' }}>Compliance Status</h3>
                    <div class="compliance-badges">
                        {[
                            { name: 'PCI-DSS', status: 'pass' },
                            { name: 'NIST 800-53', status: 'pass' },
                            { name: 'ISO 27001', status: 'pass' },
                            { name: 'GDPR', status: 'warn' },
                            { name: 'HIPAA', status: 'pass' },
                            { name: 'SOC 2', status: 'pass' },
                        ].map((fw) => (
                            <span class={`compliance-badge ${fw.status}`}>
                                {fw.status === 'pass' ? '✓' : '!'} {fw.name}
                            </span>
                        ))}
                    </div>
                </div>
            </div>
        </div>
    );
};
