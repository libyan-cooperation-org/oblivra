import { Component, createSignal, onMount, For, Show } from 'solid-js';
import { GetHeatmapData, GetAnomalies, GetRiskScore } from '../../wailsjs/go/services/CredentialIntelService';

interface Anomaly {
    type: string;
    severity: string;
    details: string;
    timestamp: string;
}

export const CredentialIntel: Component = () => {
    const [heatmap, setHeatmap] = createSignal<Record<string, number>>({});
    const [anomalies, setAnomalies] = createSignal<Anomaly[]>([]);
    const [riskScore, setRiskScore] = createSignal(100);

    onMount(async () => {
        try {
            const [h, a, s] = await Promise.all([
                GetHeatmapData(),
                GetAnomalies(),
                GetRiskScore()
            ]);
            setHeatmap(h);
            setAnomalies(a);
            setRiskScore(s);
        } catch (err) {
            console.error("Failed to load credential intel:", err);
        }
    });

    const getRiskColor = (score: number) => {
        if (score > 80) return '#10b981';
        if (score > 50) return '#fbbf24';
        return '#ef4444';
    };

    const getHeatLevel = (hour: number) => {
        // Find if any key in heatmap ends with this hour
        const hourStr = hour < 10 ? ` 0${hour}:00` : ` ${hour}:00`;
        const entry = Object.entries(heatmap()).find(([key]) => key.endsWith(hourStr));
        return entry ? Math.min(entry[1] * 20, 100) : 0;
    };

    const getSeverityColor = (sev: string) => {
        switch (sev) {
            case 'CRITICAL': return '#ef4444';
            case 'HIGH': return '#f87171';
            case 'MEDIUM': return '#fbbf24';
            default: return '#60a5fa';
        }
    };

    return (
        <div class="credential-intel-container" style={{
            padding: '2rem',
            background: '#0a0a0c',
            color: '#e5e7eb',
            height: 'calc(100vh - 80px)',
            overflow: 'auto'
        }}>
            <header style={{ "margin-bottom": '2rem' }}>
                <h1 style={{ margin: 0, "font-size": '1.5rem', 'letter-spacing': '2px', color: '#60a5fa' }}>CREDENTIAL LIFECYCLE INTELLIGENCE</h1>
                <p style={{ color: '#9ca3af', 'font-size': '0.875rem' }}>Behavioral analytics & vault usage correlation (Phase 10.5)</p>
            </header>

            <div class="intel-grid" style={{
                display: 'grid',
                'grid-template-columns': '1fr 350px',
                gap: '2rem'
            }}>
                <div class="main-content" style={{ display: 'flex', 'flex-direction': 'column', gap: '2rem' }}>
                    {/* Usage Heatmap Section */}
                    <section style={{
                        background: '#111827',
                        padding: '1.5rem',
                        'border-radius': '8px',
                        border: '1px solid #1f2937'
                    }}>
                        <h3 style={{ margin: '0 0 1rem', 'font-size': '0.9rem', 'font-family': 'monospace' }}>VAULT ACCESS HEATMAP (LAST 24H)</h3>
                        <div style={{
                            display: 'grid',
                            'grid-template-columns': 'repeat(24, 1fr)',
                            gap: '4px',
                            height: '100px'
                        }}>
                            {Array.from({ length: 24 }).map((_, hour) => (
                                <div style={{
                                    background: '#1f2937',
                                    'border-radius': '2px',
                                    height: '100%',
                                    position: 'relative',
                                }}>
                                    <div style={{
                                        position: 'absolute',
                                        bottom: 0,
                                        width: '100%',
                                        height: `${getHeatLevel(hour) || (0.1 + Math.random() * 0.2) * 100}%`,
                                        background: '#3b82f6',
                                        'box-shadow': '0 0 10px #3b82f644',
                                        'border-radius': '2px',
                                        opacity: getHeatLevel(hour) > 0 ? 1 : 0.2
                                    }}></div>
                                </div>
                            ))}
                        </div>
                        <div style={{ display: 'flex', 'justify-content': 'space-between', 'margin-top': '0.5rem', 'font-size': '0.7rem', color: '#4b5563' }}>
                            <span>00:00</span>
                            <span>12:00</span>
                            <span>23:59</span>
                        </div>
                    </section>

                    {/* Anomalies List */}
                    <section style={{
                        background: '#111827',
                        padding: '1.5rem',
                        'border-radius': '8px',
                        border: '1px solid #1f2937'
                    }}>
                        <h3 style={{ margin: '0 0 1rem', 'font-size': '0.9rem', 'font-family': 'monospace', color: '#ef4444' }}>BEHAVIORAL ANOMALIES</h3>
                        <div style={{ display: 'flex', 'flex-direction': 'column', gap: '1rem' }}>
                            <For each={anomalies()}>
                                {(anomaly) => (
                                    <div style={{
                                        padding: '1rem',
                                        background: '#0e1015',
                                        border: '1px solid #1f2937',
                                        'border-left': `4px solid ${getSeverityColor(anomaly.severity)}`,
                                        'border-radius': '4px'
                                    }}>
                                        <div style={{ display: 'flex', 'justify-content': 'space-between', 'margin-bottom': '0.5rem' }}>
                                            <span style={{ 'font-weight': 'bold', 'font-size': '0.85rem' }}>{anomaly.type}</span>
                                            <span style={{
                                                'font-size': '0.7rem',
                                                background: `${getSeverityColor(anomaly.severity)}22`,
                                                color: getSeverityColor(anomaly.severity),
                                                padding: '2px 6px',
                                                'border-radius': '4px'
                                            }}>{anomaly.severity}</span>
                                        </div>
                                        <p style={{ margin: 0, 'font-size': '0.85rem', color: '#9ca3af' }}>{anomaly.details}</p>
                                        <div style={{ 'margin-top': '0.5rem', 'font-size': '0.7rem', color: '#4b5563', 'font-family': 'monospace' }}>
                                            {anomaly.timestamp}
                                        </div>
                                    </div>
                                )}
                            </For>
                            <Show when={anomalies().length === 0}>
                                <div style={{ padding: '2rem', 'text-align': 'center', color: '#374151', 'font-style': 'italic' }}>
                                    No behavioral anomalies detected.
                                </div>
                            </Show>
                        </div>
                    </section>
                </div>

                <aside style={{ display: 'flex', 'flex-direction': 'column', gap: '2rem' }}>
                    {/* Risk Score Card */}
                    <div style={{
                        background: '#111827',
                        padding: '2rem',
                        'border-radius': '8px',
                        border: '1px solid #1f2937',
                        'text-align': 'center'
                    }}>
                        <div style={{ 'font-size': '0.75rem', color: '#6b7280', 'margin-bottom': '1rem', 'letter-spacing': '1px' }}>OVERALL TRUST SCORE</div>
                        <div style={{
                            'font-size': '4rem',
                            'font-weight': 'bold',
                            color: getRiskColor(riskScore()),
                            'text-shadow': `0 0 20px ${getRiskColor(riskScore())}44`
                        }}>{riskScore()}</div>
                        <div style={{ 'margin-top': '1rem', 'font-size': '0.85rem', color: '#9ca3af' }}>
                            {riskScore() > 80 ? 'EXCELLENT' : riskScore() > 50 ? 'WARNING' : 'CRITICAL'}
                        </div>
                    </div>

                    {/* Stats Box */}
                    <div style={{
                        background: '#111827',
                        padding: '1.5rem',
                        'border-radius': '8px',
                        border: '1px solid #1f2937'
                    }}>
                        <h4 style={{ margin: '0 0 1rem', 'font-size': '0.8rem', color: '#6b7280' }}>VAULT HIGHLIGHTS</h4>
                        <div style={{ display: 'flex', 'flex-direction': 'column', gap: '0.75rem' }}>
                            <div style={{ display: 'flex', 'justify-content': 'space-between', 'font-size': '0.85rem' }}>
                                <span style={{ color: '#9ca3af' }}>Stale Credentials</span>
                                <span style={{ color: '#fbbf24' }}>12</span>
                            </div>
                            <div style={{ display: 'flex', 'justify-content': 'space-between', 'font-size': '0.85rem' }}>
                                <span style={{ color: '#9ca3af' }}>Active Sessions</span>
                                <span style={{ color: '#10b981' }}>4</span>
                            </div>
                            <div style={{ display: 'flex', 'justify-content': 'space-between', 'font-size': '0.85rem' }}>
                                <span style={{ color: '#9ca3af' }}>Shared Secrets</span>
                                <span style={{ color: '#60a5fa' }}>8</span>
                            </div>
                        </div>
                    </div>
                </aside>
            </div>
        </div>
    );
};
