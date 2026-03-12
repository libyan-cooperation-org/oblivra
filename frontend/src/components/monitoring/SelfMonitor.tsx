import { Component, createSignal, onMount, For } from 'solid-js';
import * as MetricsService from '../../../wailsjs/go/app/MetricsService';
import * as ObservabilityService from '../../../wailsjs/go/app/ObservabilityService';
import { monitoring } from '../../../wailsjs/go/models';

export const SelfMonitor: Component = () => {
    const [metrics, setMetrics] = createSignal<monitoring.Metric[]>([]);
    const [status, setStatus] = createSignal<any>(null);
    const [lastUpdate, setLastUpdate] = createSignal<Date>(new Date());

    const fetchData = async () => {
        try {
            const [m, s] = await Promise.all([
                (MetricsService as any).GetAllMetrics(),
                (ObservabilityService as any).GetObservabilityStatus()
            ]);
            setMetrics(m || []);
            setStatus(s);
            setLastUpdate(new Date());
        } catch (e) {
            console.error('Failed to fetch monitoring data:', e);
        }
    };

    onMount(() => {
        fetchData();
        const interval = setInterval(fetchData, 2000);
        return () => clearInterval(interval);
    });

    const timeSinceStartup = (startStr: string): number => {
        if (!startStr) return 0;
        const start = new Date(startStr);
        const now = new Date();
        return (now.getTime() - start.getTime()) / (1000 * 60);
    };

    const formatValue = (metric: monitoring.Metric) => {
        if (metric.name.includes('bytes') && !metric.name.includes('slope')) {
            const mb = metric.value / (1024 * 1024);
            return `${mb.toFixed(2)} MB`;
        }
        if (metric.name.includes('slope_bytes')) {
            const kb = metric.value / 1024;
            const sign = kb >= 0 ? '+' : '';
            return `${sign}${kb.toFixed(2)} KB/min`;
        }
        if (metric.name.includes('slope')) {
            const sign = metric.value >= 0 ? '+' : '';
            return `${sign}${metric.value.toFixed(2)} / min`;
        }
        if (metric.name.includes('latency')) {
            return `${metric.value.toFixed(1)} ms`;
        }
        return metric.value.toLocaleString();
    };

    return (
        <div class="monitor-container">
            <header class="monitor-header">
                <div class="monitor-title">
                    <span class="monitor-icon">📡</span>
                    <h2>PLATFORM SELF-MONITOR</h2>
                </div>
                <div class="last-sync">
                    LAST REFRESH: {lastUpdate().toLocaleTimeString()}
                </div>
            </header>

            <div class="monitor-grid">
                {/* System Health Card */}
                <div class="monitor-card wide">
                    <div class="card-header">
                        <h3>SYSTEM RUNTIME</h3>
                        <div class="status-indicator">
                            <span class="dot" style={{ background: 'var(--status-online)' }}></span>
                            OPERATIONAL
                        </div>
                    </div>
                    <div class="runtime-stats">
                        <div class="stat-box">
                            <span class="label">GOROUTINES</span>
                            <span class="value">{status()?.goroutines || '-'}</span>
                            <span class="sub">{status()?.goroutine_peak ? `PEAK: ${status().goroutine_peak}` : ''}</span>
                        </div>
                        <div class="stat-box">
                            <span class="label">HEAP MEMORY</span>
                            <span class="value">{status()?.heap_alloc_mb?.toFixed(2) || '-'} MB</span>
                            <span class="sub">SYS: {status()?.heap_sys_mb?.toFixed(2)} MB</span>
                        </div>
                        <div class="stat-box">
                            <span class="label">GC PAUSE</span>
                            <span class="value">{(status()?.gc_pause_ns / 1000000)?.toFixed(3) || '-'} ms</span>
                            <span class="sub">COUNT: {status()?.gc_count}</span>
                        </div>
                        <div class="stat-box">
                            <span class="label">CPU CORES</span>
                            <span class="value">{status()?.num_cpu || '-'}</span>
                            <span class="sub">{status()?.go_version}</span>
                        </div>
                    </div>
                </div>

                {/* Database Stats */}
                <div class="monitor-card">
                    <div class="card-header">
                        <h3>STORAGE (BADGERDB)</h3>
                    </div>
                    <div class="list-metrics">
                        <For each={metrics().filter(m => m.name.startsWith('badger_'))}>
                            {(metric) => (
                                <div class="metric-row">
                                    <span class="m-name">{metric.name.replace('badger_', '').replace(/_/g, ' ')}</span>
                                    <span class="m-value">{formatValue(metric)}</span>
                                </div>
                            )}
                        </For>
                    </div>
                </div>

                {/* Ingestion & SIEM */}
                <div class="monitor-card">
                    <div class="card-header">
                        <h3>INGESTION PIPELINE</h3>
                    </div>
                    <div class="list-metrics">
                        <For each={metrics().filter(m => !m.name.startsWith('badger_') && !m.name.startsWith('ssh_') && !m.name.startsWith('stability_'))}>
                            {(metric) => (
                                <div class="metric-row">
                                    <span class="m-name">{metric.name.replace(/_/g, ' ')}</span>
                                    <span class="m-value">{formatValue(metric)}</span>
                                </div>
                            )}
                        </For>
                    </div>
                </div>

                {/* Stability & Soak Metrics */}
                <div class="monitor-card stability">
                    <div class="card-header">
                        <h3>STABILITY & SOAK METRICS</h3>
                        {(() => {
                            const rssSlope = metrics().find(m => m.name === 'stability_rss_slope_bytes_min')?.value || 0;
                            const goSlope = metrics().find(m => m.name === 'stability_goroutine_slope_min')?.value || 0;
                            let grade = "EXCEPTIONAL";
                            let color = "var(--status-online)";
                            
                            if (rssSlope > 1024 * 10 || goSlope > 0.5) { // >10KB/min or >0.5 goroutine/min
                                grade = "STABLE";
                                color = "var(--status-warning)";
                            }
                            if (rssSlope > 1024 * 100 || goSlope > 2) { // >100KB/min or >2 goroutines/min
                                grade = "DEGRADING";
                                color = "var(--status-danger)";
                            }
                            
                            return (
                                <div class="stability-grade" style={{ color }}>
                                    {grade}
                                </div>
                            );
                        })()}
                    </div>
                    <div class="list-metrics">
                        <For each={metrics().filter(m => m.name.startsWith('stability_'))}>
                            {(metric) => (
                                <div class="metric-row">
                                    <span class="m-name">{metric.name.replace('stability_', '').replace(/_/g, ' ')}</span>
                                    <span class="m-value" style={{ 
                                        color: metric.value > 0 ? 'var(--status-warning)' : 'var(--status-online)' 
                                    }}>
                                        {formatValue(metric)}
                                    </span>
                                </div>
                            )}
                        </For>
                        <div class="stability-note">
                            * Measured over {status() ? (timeSinceStartup(status().start_time)).toFixed(1) : '-'} minutes
                        </div>
                    </div>
                </div>

                {/* SSH & Tunnels */}
                <div class="monitor-card">
                    <div class="card-header">
                        <h3>SSH / CONNECTIVITY</h3>
                    </div>
                    <div class="list-metrics">
                        <For each={metrics().filter(m => m.name.startsWith('ssh_') || m.name.startsWith('tunnel_'))}>
                            {(metric) => (
                                <div class="metric-row">
                                    <span class="m-name">{metric.name.replace(/_/g, ' ')}</span>
                                    <span class="m-value">{formatValue(metric)}</span>
                                </div>
                            )}
                        </For>
                    </div>
                </div>
            </div>

            <style>{`
                .monitor-container {
                    padding: 0;
                    height: 100%;
                    background: var(--bg-primary);
                    color: var(--text-primary);
                    font-family: var(--font-ui);
                }
                .monitor-header {
                    height: var(--header-height);
                    border-bottom: 1px solid var(--glass-border);
                    display: flex;
                    justify-content: space-between;
                    align-items: center;
                    padding: 0 1.5rem;
                    background: var(--bg-secondary);
                }
                .monitor-title { display: flex; align-items: center; gap: 0.75rem; }
                .monitor-title h2 { font-size: 14px; letter-spacing: 2px; font-weight: 700; margin: 0; }
                .last-sync { font-family: var(--font-mono); font-size: 10px; color: var(--text-muted); }
                
                .monitor-grid {
                    display: grid;
                    grid-template-columns: repeat(auto-fill, minmax(400px, 1fr));
                    gap: 1px;
                    background: var(--glass-border);
                    padding: 1px;
                }
                .monitor-card {
                    background: var(--bg-secondary);
                    padding: 1.5rem;
                    display: flex;
                    flex-direction: column;
                    gap: 1.5rem;
                }
                .monitor-card.wide { grid-column: 1 / -1; }
                
                .card-header { display: flex; justify-content: space-between; align-items: center; }
                .card-header h3 { font-size: 11px; letter-spacing: 1px; color: var(--text-secondary); margin: 0; }
                
                .status-indicator { display: flex; align-items: center; gap: 0.5rem; font-family: var(--font-mono); font-size: 11px; }
                .status-indicator .dot { width: 8px; height: 8px; border-radius: 50%; }
                
                .runtime-stats {
                    display: grid;
                    grid-template-columns: repeat(4, 1fr);
                    gap: 1.5rem;
                }
                .stat-box { display: flex; flex-direction: column; gap: 0.25rem; }
                .stat-box .label { font-size: 9px; color: var(--text-muted); letter-spacing: 1px; }
                .stat-box .value { font-family: var(--font-mono); font-size: 1.5rem; font-weight: 700; color: var(--text-primary); }
                .stat-box .sub { font-size: 10px; color: var(--text-muted); }
                
                .list-metrics { display: flex; flex-direction: column; gap: 0.75rem; }
                .metric-row {
                    display: flex;
                    justify-content: space-between;
                    align-items: center;
                    padding-bottom: 0.5rem;
                    border-bottom: 1px solid rgba(255,255,255,0.05);
                }
                .m-name { font-size: 11px; color: var(--text-secondary); text-transform: uppercase; }
                .m-value { font-family: var(--font-mono); font-size: 12px; color: var(--text-primary); }

                .stability { border-left: 2px solid var(--status-online); }
                .stability-grade { font-family: var(--font-mono); font-size: 11px; font-weight: 700; letter-spacing: 1px; }
                .stability-note { font-size: 9px; color: var(--text-muted); font-style: italic; margin-top: 0.5rem; }
            `}</style>
        </div>
    );
};
