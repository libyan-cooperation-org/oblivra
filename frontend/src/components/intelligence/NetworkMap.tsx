import { Component, createSignal, onMount, For, Show } from 'solid-js';
import { GetLiveTraffic } from '../../../wailsjs/go/services/NDRService';

interface Flow {
    timestamp: string;
    src_ip: string;
    src_port: number;
    dest_ip: string;
    dest_port: number;
    protocol: string;
    bytes_sent: number;
    bytes_recv: number;
    app_name?: string;
}

export const NetworkMap: Component = () => {
    const [flows, setFlows] = createSignal<Flow[]>([]);
    const [anomalies, setAnomalies] = createSignal<any[]>([]);

    onMount(async () => {
        try {
            const data = await GetLiveTraffic();
            setFlows(data || []);

            // Mock anomalies for Phase 11 visualization
            setAnomalies([
                {
                    id: '1',
                    type: 'DGA_DETECTED',
                    source: '192.168.1.10',
                    target: 'ajksdhfkjah.com',
                    severity: 'HIGH',
                    evidence: [
                        { key: 'entropy', value: 4.8, threshold: 4.0, description: 'Domain name character entropy' },
                        { key: 'nxdomain_ratio', value: 0.85, description: 'Ratio of failed DNS queries' }
                    ]
                },
                {
                    id: '2',
                    type: 'LATERAL_MOVEMENT',
                    source: '192.168.1.15',
                    target: '10 Internal Hosts',
                    severity: 'CRITICAL',
                    evidence: [
                        { key: 'unique_destinations', value: 10, threshold: 5, description: 'Count of unique internal destinations' },
                        { key: 'window', value: '1m', description: 'Correlation time window' }
                    ]
                }
            ]);
        } catch (err) {
            console.error("Failed to load network traffic:", err);
        }
    });

    const getAppColor = (app?: string) => {
        switch (app) {
            case 'DNS': return '#fbbf24';
            case 'HTTPS': return '#10b981';
            case 'SSH': return '#3b82f6';
            default: return '#6b7280';
        }
    };

    return (
        <div class="network-map-container" style={{
            height: 'calc(100vh - 120px)',
            background: '#0a0a0c',
            color: '#e5e7eb',
            padding: '1.5rem',
            display: 'flex',
            gap: '1.5rem'
        }}>
            {/* Main Traffic View */}
            <div class="traffic-main" style={{ flex: 1, display: 'flex', 'flex-direction': 'column' }}>
                <div class="map-header" style={{ 'margin-bottom': '1.5rem' }}>
                    <h2 style={{ margin: 0, 'font-size': '1.25rem', 'letter-spacing': '1px' }}>NETWORK NDR FLOWS</h2>
                    <p style={{ color: '#9ca3af', 'font-size': '0.875rem' }}>Real-time deep packet inspection & behavioral correlation</p>
                </div>

                <div class="flow-grid" style={{
                    display: 'grid',
                    'grid-template-columns': 'repeat(auto-fill, minmax(320px, 1fr))',
                    gap: '1rem',
                    overflow: 'auto',
                    'padding-right': '0.5rem'
                }}>
                    <For each={flows()}>
                        {(flow) => (
                            <div class="flow-card app-entry-animation" style={{
                                background: '#111827',
                                border: `1px solid ${getAppColor(flow.app_name)}33`,
                                'border-left': `4px solid ${getAppColor(flow.app_name)}`,
                                padding: '1rem',
                                'border-radius': '4px',
                                position: 'relative'
                            }}>
                                <div style={{ display: 'flex', 'justify-content': 'space-between', 'margin-bottom': '0.75rem' }}>
                                    <span style={{ 'font-family': 'monospace', 'font-weight': 'bold', 'font-size': '0.8rem' }}>{flow.app_name || 'GENERIC'}</span>
                                    <span style={{ color: '#4b5563', 'font-size': '0.7rem', 'font-family': 'monospace' }}>{flow.protocol}</span>
                                </div>

                                <div style={{ display: 'flex', 'flex-direction': 'column', gap: '0.25rem' }}>
                                    <div style={{ display: 'flex', 'align-items': 'center', gap: '0.5rem', 'font-size': '0.9rem', 'font-family': 'monospace' }}>
                                        <span style={{ color: '#3b82f6' }}>{flow.src_ip}</span>
                                        <span style={{ color: '#1f2937' }}>→</span>
                                        <span style={{ color: '#ef4444' }}>{flow.dest_ip}</span>
                                    </div>
                                    <div style={{ 'font-size': '0.7rem', color: '#6b7280' }}>
                                        PORT {flow.src_port} → {flow.dest_port}
                                    </div>
                                </div>

                                <div style={{ 'margin-top': '1rem', display: 'flex', 'justify-content': 'space-between', 'font-size': '0.75rem' }}>
                                    <span style={{ color: '#10b981' }}>↑ {flow.bytes_sent} B</span>
                                    <span style={{ color: '#60a5fa' }}>↓ {flow.bytes_recv} B</span>
                                </div>
                                <div style={{ 'margin-top': '0.5rem', height: '2px', width: '100%', background: '#1f2937' }}>
                                    <div style={{
                                        height: '100%',
                                        width: `${Math.min((flow.bytes_sent / 512) * 100, 100)}%`,
                                        background: getAppColor(flow.app_name)
                                    }}></div>
                                </div>
                            </div>
                        )}
                    </For>
                </div>

                {/* Tactical Pulse Box */}
                <div class="tactical-visual" style={{
                    'margin-top': '2rem',
                    height: '200px',
                    background: 'rgba(59, 130, 246, 0.03)',
                    border: '1px dashed #3b82f633',
                    position: 'relative',
                    overflow: 'hidden',
                    'border-radius': '4px'
                }}>
                    <div style={{
                        position: 'absolute',
                        top: '10px',
                        left: '10px',
                        'font-family': 'monospace',
                        'font-size': '10px',
                        color: '#3b82f6'
                    }}>LIVE_SEGMENT_SCANNER_v2.1</div>

                    <For each={Array(8).fill(0)}>
                        {() => (
                            <div class="radar-pulse" style={{
                                position: 'absolute',
                                left: `${5 + Math.random() * 90}%`,
                                top: `${5 + Math.random() * 90}%`,
                                width: '4px',
                                height: '4px',
                                background: '#3b82f6',
                                'border-radius': '50%',
                                opacity: 0.4,
                                'box-shadow': '0 0 15px #3b82f6'
                            }}></div>
                        )}
                    </For>
                </div>
            </div>

            {/* Anomaly Sidebar */}
            <aside class="anomaly-sidebar" style={{
                width: '320px',
                background: '#0e1015',
                'border-left': '1px solid #1f2937',
                padding: '1.5rem',
                display: 'flex',
                'flex-direction': 'column',
                gap: '1.5rem'
            }}>
                <header>
                    <h3 style={{ margin: 0, 'font-size': '0.9rem', 'letter-spacing': '2px', color: '#f87171' }}>NETWORK ANOMALIES</h3>
                    <p style={{ margin: '0.25rem 0 0', 'font-size': '0.75rem', color: '#4b5563' }}>High-fidelity behavioral flags</p>
                </header>

                <div class="anomaly-list" style={{ display: 'flex', 'flex-direction': 'column', gap: '1rem' }}>
                    <For each={anomalies()}>
                        {(anomaly) => (
                            <div class="anomaly-item" style={{
                                background: 'rgba(239, 68, 68, 0.05)',
                                border: '1px solid #ef444433',
                                'border-left': '3px solid #ef4444',
                                padding: '1rem',
                                'border-radius': '2px'
                            }}>
                                <div style={{ display: 'flex', 'justify-content': 'space-between', 'margin-bottom': '0.5rem' }}>
                                    <span style={{ 'font-size': '0.75rem', 'font-weight': 'bold', color: '#ef4444' }}>{anomaly.type}</span>
                                    <span style={{ 'font-size': '0.65rem', color: '#fca5a5', background: '#991b1b', padding: '1px 4px', 'border-radius': '2px' }}>{anomaly.severity}</span>
                                </div>
                                <div style={{ 'font-size': '0.85rem', 'font-family': 'monospace', color: '#e5e7eb' }}>
                                    SRC: {anomaly.source}
                                </div>
                                <div style={{ 'font-size': '0.75rem', color: '#9ca3af', 'margin-top': '0.25rem' }}>
                                    Target: {anomaly.target}
                                </div>

                                <Show when={anomaly.evidence}>
                                    <div class="evidence-box" style={{ 'margin-top': '1rem', 'padding-top': '0.5rem', 'border-top': '1px solid rgba(255,255,255,0.05)' }}>
                                        <For each={anomaly.evidence}>
                                            {(ev: any) => (
                                                <div style={{ 'font-size': '0.65rem', 'margin-bottom': '0.25rem' }}>
                                                    <span style={{ color: '#9ca3af' }}>{ev.key}:</span>
                                                    <span style={{ 'font-family': 'monospace', color: '#fff', 'margin-left': '4px' }}>{ev.value}</span>
                                                </div>
                                            )}
                                        </For>
                                    </div>
                                </Show>
                            </div>
                        )}
                    </For>

                    <Show when={anomalies().length === 0}>
                        <div style={{ 'text-align': 'center', padding: '2rem', color: '#374151', 'font-style': 'italic', 'font-size': '0.85rem' }}>
                            No active anomalies detected in current window.
                        </div>
                    </Show>
                </div>

                <div class="ndr-stats" style={{ 'margin-top': 'auto', padding: '1rem', background: '#111827', 'border-radius': '4px' }}>
                    <div style={{ 'font-size': '0.75rem', color: '#6b7280', 'margin-bottom': '0.75rem' }}>PIPELINE METRICS</div>
                    <div style={{ display: 'flex', 'justify-content': 'space-between', 'margin-bottom': '0.5rem' }}>
                        <span style={{ 'font-size': '0.7rem', color: '#9ca3af' }}>PPS (IN/OUT)</span>
                        <span style={{ 'font-size': '0.8rem', 'font-family': 'monospace' }}>12.4k / 8.1k</span>
                    </div>
                    <div style={{ display: 'flex', 'justify-content': 'space-between' }}>
                        <span style={{ 'font-size': '0.7rem', color: '#9ca3af' }}>ACTIVE_FLOWS</span>
                        <span style={{ 'font-size': '0.8rem', 'font-family': 'monospace', color: '#10b981' }}>402</span>
                    </div>
                </div>
            </aside>
        </div>
    );
};
