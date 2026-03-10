import { Component, createResource, For, Show } from 'solid-js';
import { GetRiskHistory } from '../../wailsjs/go/app/RiskService';

export const ConfigRisk: Component = () => {
    const [history, { refetch }] = createResource(() => GetRiskHistory());

    const getRiskColor = (score: number) => {
        if (score >= 75) return '#ef4444'; // Red
        if (score >= 50) return '#f59e0b'; // Amber
        if (score >= 25) return '#3b82f6'; // Blue
        return '#10b981'; // Green
    };

    return (
        <div class="config-risk-panel app-entry-animation" style={{ padding: '1rem', height: '100%', overflow: 'auto', background: '#0a0a0c' }}>
            <header class="tactical-header" style={{ display: 'flex', 'justify-content': 'space-between', 'align-items': 'center', 'margin-bottom': '2rem', 'border-bottom': '1px solid #1f2937', 'padding-bottom': '1rem' }}>
                <div>
                    <h2 style={{ color: '#e5e7eb', margin: 0, 'font-size': '1.25rem', 'font-family': 'monospace' }}>TACTICAL CONFIG RISK AUDIT</h2>
                    <p style={{ color: '#9ca3af', margin: 0, 'font-size': '0.75rem' }}>Evaluation of blast radius and security posture impact for all platform changes.</p>
                </div>
                <button
                    onClick={refetch}
                    class="tactical-button"
                    style={{
                        background: '#1f2937',
                        color: '#60a5fa',
                        border: '1px solid #3b82f6',
                        padding: '0.5rem 1rem',
                        'font-family': 'monospace',
                        'font-size': '12px',
                        cursor: 'pointer'
                    }}
                >
                    REFRESH SCORES
                </button>
            </header>

            <div class="risk-timeline" style={{ display: 'flex', 'flex-direction': 'column', gap: '1rem' }}>
                <For each={history()}>
                    {(risk) => (
                        <div class="risk-card" style={{
                            "border-left": `4px solid ${getRiskColor(risk.score)}`,
                            background: 'rgba(31, 41, 55, 0.3)',
                            padding: '1rem',
                            'border-radius': '2px',
                            border: '1px solid #1f2937',
                            'border-left-width': '4px'
                        }}>
                            <div class="risk-meta" style={{ display: 'flex', 'justify-content': 'space-between', 'font-family': 'monospace', 'font-size': '10px', 'margin-bottom': '0.75rem' }}>
                                <span style={{ color: '#6b7280' }}>ID: {risk.id}</span>
                                <span style={{ color: '#9ca3af' }}>{new Date(risk.timestamp.toString()).toLocaleString()}</span>
                                <span style={{ color: getRiskColor(risk.score), 'font-weight': 'bold' }}>{risk.level.toUpperCase()} (INTENSITY: {risk.score}/100)</span>
                            </div>
                            <div class="risk-body">
                                <h3 style={{ margin: '0 0 0.5rem 0', 'font-size': '14px', color: '#e5e7eb', 'font-family': 'monospace' }}>{risk.reason}</h3>
                                <div style={{ background: '#111827', padding: '0.5rem', 'border-radius': '2px', border: '1px solid #1f2937' }}>
                                    <span style={{ color: '#60a5fa', 'font-size': '10px', 'font-weight': 'bold', 'display': 'block', 'margin-bottom': '0.25rem' }}>IMPACT ANALYSIS</span>
                                    <p style={{ margin: 0, 'font-size': '12px', color: '#9ca3af' }}>{risk.impact}</p>
                                </div>
                            </div>
                        </div>
                    )}
                </For>
                <Show when={history()?.length === 0}>
                    <div style={{
                        opacity: 0.5,
                        "text-align": "center",
                        padding: "80px",
                        "font-family": "var(--font-mono)",
                        "font-size": "11px",
                        color: "var(--text-muted)"
                    }}>
                        NO RECENT CONFIGURATION RISK EVALUATIONS RECORDED IN THE TACTICAL BUFFER
                    </div>
                </Show>
            </div>
        </div>
    );
};
