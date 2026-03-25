import { Component, createSignal, onMount, Show, createMemo } from 'solid-js';
import { 
    PageLayout, 
    KPIGrid, 
    KPI, 
    Table, 
    Badge, 
    Panel, 
    SectionHeader,
    LoadingState, 
    ErrorState,
    Histogram,
    Button,
    normalizeSeverity,
    formatTimestamp,
    Column
} from '@components/ui';
import { IS_BROWSER } from '@core/context';
import '../styles/credential-intel.css';

interface Anomaly {
    id?: string;
    type: string;
    severity: string;
    details: string;
    timestamp: string;
}

export const CredentialIntel: Component = () => {
    const [heatmap, setHeatmap] = createSignal<Record<string, number>>({});
    const [anomalies, setAnomalies] = createSignal<Anomaly[]>([]);
    const [riskScore, setRiskScore] = createSignal(100);
    const [loading, setLoading] = createSignal(true);
    const [error, setError] = createSignal<string | null>(null);

    const fetchData = async () => {
        setLoading(true);
        setError(null);
        if (IS_BROWSER) { 
            // Demo data for browser context
            setAnomalies([
                { type: 'Multiple Vault Access', severity: 'HIGH', details: 'Unusual access frequency from IP 192.168.1.45', timestamp: new Date().toISOString() },
                { type: 'Service Account Usage', severity: 'MEDIUM', details: 'Automated login outside maintenance window', timestamp: new Date().toISOString() }
            ]);
            setHeatmap({
                "00:00": 10, "01:00": 5, "02:00": 2, "03:00": 1, "04:00": 3, "05:00": 8,
                "06:00": 15, "07:00": 30, "08:00": 45, "09:00": 60, "10:00": 55, "11:00": 40,
                "12:00": 35, "13:00": 42, "14:00": 50, "15:00": 58, "16:00": 65, "17:00": 70,
                "18:00": 45, "19:00": 30, "20:00": 25, "21:00": 20, "22:00": 15, "23:00": 12
            });
            setLoading(false); 
            return; 
        }

        try {
            const { GetHeatmapData, GetAnomalies, GetRiskScore } = await import('../../wailsjs/go/services/CredentialIntelService');
            const [h, a, s] = await Promise.all([
                GetHeatmapData(),
                GetAnomalies(),
                GetRiskScore()
            ]);
            setHeatmap(h || {});
            setAnomalies(a || []);
            setRiskScore(s ?? 100);
        } catch (err) {
            console.error("Failed to load credential intel:", err);
            setError(String(err));
        } finally {
            setLoading(false);
        }
    };

    onMount(() => fetchData());

    const histogramData = createMemo(() => {
        const h = heatmap();
        const maxVal = Math.max(...Object.values(h), 1);
        
        return Object.entries(h).map(([time, count]) => ({
            count,
            heightPct: (count / maxVal) * 100,
            timeLabel: time
        }));
    });

    const getRiskVariant = (score: number) => {
        if (score > 80) return { color: 'var(--status-online)', label: 'OPTIMAL' };
        if (score > 50) return { color: 'var(--status-degraded)', label: 'WARNING' };
        return { color: 'var(--status-offline)', label: 'CRITICAL' };
    };

    const columns: Column<Anomaly>[] = [
        { 
            key: 'type', 
            label: 'Behavioral Pattern',
            render: (a) => (
                <div>
                    <div style="font-weight: 700; font-size: 13px; color: var(--text-primary);">{a.type}</div>
                    <div style="font-size: 11px; color: var(--text-muted);">{a.details}</div>
                </div>
            )
        },
        { 
            key: 'severity', 
            label: 'Risk Level',
            width: '120px',
            render: (a) => (
                <Badge severity={normalizeSeverity(a.severity)}>
                    {a.severity}
                </Badge>
            )
        },
        { 
            key: 'timestamp', 
            label: 'Detected At',
            width: '180px',
            mono: true,
            render: (a) => formatTimestamp(a.timestamp)
        }
    ];

    return (
        <PageLayout
            title="Credential Lifecycle Intelligence"
            subtitle="BEHAVIORAL_ANALYTICS_&_VAULT_CORELLATION"
            actions={
                <Button variant="ghost" onClick={fetchData}>
                    REFRESH_STATE
                </Button>
            }
        >
            <Show when={error()}>
                <ErrorState message={error()!} onRetry={fetchData} />
            </Show>

            <Show when={loading() && anomalies().length === 0}>
                <LoadingState message="CONSTRUCTING_BEHAVIORAL_GRAPH..." />
            </Show>

            <Show when={!loading() && !error()}>
                <KPIGrid cols={4} class="mb-6">
                    <KPI 
                        label="VAULT_TRUST_SCORE" 
                        value={riskScore()} 
                        color={getRiskVariant(riskScore()).color} 
                        delta={-2} 
                        deltaLabel="pts" 
                    />
                    <KPI 
                        label="STALE_SECRETS" 
                        value={12} 
                        sparkData={[10, 8, 11, 12, 12]} 
                        sparkColor="var(--status-degraded)" 
                    />
                    <KPI 
                        label="ACTIVE_SESSIONS" 
                        value={4} 
                        sparkData={[4, 5, 4, 3, 4]} 
                        sparkColor="var(--status-online)" 
                    />
                    <KPI 
                        label="EXPIRED_LEASES" 
                        value={8} 
                        color="var(--alert-high)" 
                    />
                </KPIGrid>

                <div class="intel-layout">
                    <div style="display: flex; flex-direction: column; gap: var(--gap-lg);">
                        <Panel title="VAULT ACCESS HISTOGRAM" subtitle="LAST_24H_AGGREGATED_HEATMAP">
                            <Histogram data={histogramData()} height={120} />
                        </Panel>

                        <Panel title="DETECTION_ANOMALIES" noPadding>
                            <Table 
                                columns={columns} 
                                data={anomalies()} 
                                emptyText="NO BEHAVIORAL ANOMALIES DETECTED"
                                striped
                            />
                        </Panel>
                    </div>

                    <div style="display: flex; flex-direction: column; gap: var(--gap-lg);">
                        <Panel title="HEURISTIC ASSESSMENT">
                            <div class="trust-score-wrap" style="background: var(--surface-1); border: 1px solid var(--border-primary); padding: 24px; border-radius: var(--radius-sm); display: flex; flex-direction: column; align-items: center; gap: 12px; margin-bottom: var(--gap-lg);">
                                <div style="font-size: 10px; font-weight: 800; color: var(--text-muted); text-transform: uppercase; letter-spacing: 1px;">OVERALL TRUST</div>
                                <div 
                                    style={{ 
                                        'color': getRiskVariant(riskScore()).color,
                                        'font-size': '48px',
                                        'font-weight': '900',
                                        'font-family': 'var(--font-mono)',
                                        'line-height': '1'
                                    }}
                                >
                                    {riskScore()}
                                </div>
                                <Badge severity={normalizeSeverity(getRiskVariant(riskScore()).label)} size="lg">
                                    {getRiskVariant(riskScore()).label}
                                </Badge>
                            </div>
                            
                            <SectionHeader>VAULT HIGHLIGHTS</SectionHeader>
                            <div style="display: flex; flex-direction: column; gap: 8px;">
                                <div style="display: flex; justify-content: space-between; font-size: 12px; padding: 6px 0; border-bottom: 1px solid var(--border-subtle);">
                                    <span style="color: var(--text-secondary); font-weight: 600;">Rotated Secrets</span>
                                    <span style="color: var(--status-online); font-family: var(--font-mono); font-weight: 700;">84%</span>
                                </div>
                                <div style="display: flex; justify-content: space-between; font-size: 12px; padding: 6px 0; border-bottom: 1px solid var(--border-subtle);">
                                    <span style="color: var(--text-secondary); font-weight: 600;">MFA Enforced</span>
                                    <span style="color: var(--status-online); font-family: var(--font-mono); font-weight: 700;">100%</span>
                                </div>
                                <div style="display: flex; justify-content: space-between; font-size: 12px; padding: 6px 0; border-bottom: 1px solid var(--border-subtle);">
                                    <span style="color: var(--text-secondary); font-weight: 600;">Active Leases</span>
                                    <span style="color: var(--status-online); font-family: var(--font-mono); font-weight: 700;">142</span>
                                </div>
                                <div style="display: flex; justify-content: space-between; font-size: 12px; padding: 6px 0;">
                                    <span style="color: var(--text-secondary); font-weight: 600;">Expired Leases</span>
                                    <span style="color: var(--status-offline); font-family: var(--font-mono); font-weight: 700;">3</span>
                                </div>
                            </div>
                        </Panel>

                        <Panel title="HEURISTIC_TELEMETRY">
                            <div style="font-size: 11px; color: var(--text-muted); line-height: 1.6; font-family: var(--font-mono);">
                                Monitoring active for 1,248 secrets across 14 vaults. 
                                Cross-correlation with SIEM behavioral baseline enabled.
                                Last scan: {formatTimestamp(new Date())}.
                            </div>
                        </Panel>
                    </div>
                </div>
            </Show>
        </PageLayout>
    );
};
