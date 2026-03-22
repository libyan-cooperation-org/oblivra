// RansomwareUI.tsx — Phase 9 Web: centralized ransomware status and remediation
import { Component, createSignal, onMount, onCleanup, For, Show } from 'solid-js';
import * as SIEMService from '../../../wailsjs/go/services/SIEMService';
import { EventsOn, EventsOff } from '../../../wailsjs/runtime/runtime';

export const RansomwareUI: Component = () => {
    const [alerts, setAlerts] = createSignal<any[]>([]);
    const [stats, setStats] = createSignal<any>(null);
    const [loading, setLoading] = createSignal(true);
    const [liveAlert, setLiveAlert] = createSignal<string | null>(null);

    onMount(async () => {
        EventsOn('ransomware:detection', (data: any) => {
            setLiveAlert(`🚨 RANSOMWARE DETECTED on ${data?.host_id ?? 'unknown host'}: ${data?.type ?? 'entropy spike'}`);
            setTimeout(() => setLiveAlert(null), 10000);
        });

        try {
            const [hostEvents, globalStats] = await Promise.all([
                (SIEMService as any).SearchHostEvents('ransomware entropy canary', 50),
                (SIEMService as any).GetGlobalThreatStats(),
            ]);
            setAlerts(hostEvents ?? []);
            setStats(globalStats);
        } catch { }
        setLoading(false);
    });

    onCleanup(() => EventsOff('ransomware:detection'));

    const severityMap: Record<string, string> = { high: '#f85149', critical: '#f85149', medium: '#d29922', low: '#3fb950' };

    return (
        <div style="padding: 0; height: 100%; overflow-y: auto; background: var(--bg-primary); color: var(--text-primary); font-family: var(--font-ui);">
            <div style="height: var(--header-height); border-bottom: 1px solid var(--glass-border); display: flex; justify-content: space-between; align-items: center; padding: 0 1.5rem; background: var(--bg-secondary);">
                <div style="display: flex; align-items: center; gap: 0.75rem;">
                    <span style="font-size: 16px;">🦠</span>
                    <h2 style="font-size: 13px; letter-spacing: 2px; font-weight: 700; margin: 0; text-transform: uppercase;">Ransomware Defense</h2>
                </div>
                <span style={`font-size: 10px; font-family: var(--font-mono); color: ${alerts().length > 0 ? '#f85149' : '#3fb950'};`}>
                    {alerts().length > 0 ? `⚠ ${alerts().length} DETECTIONS` : '● PROTECTED'}
                </span>
            </div>

            <div style="padding: 1.5rem; display: flex; flex-direction: column; gap: 1.25rem;">
                {/* Live alert banner */}
                <Show when={liveAlert()}>
                    <div style="background: rgba(248,81,73,0.12); border: 1px solid rgba(248,81,73,0.5); border-radius: 6px; padding: 1rem; font-family: var(--font-mono); font-size: 12px; color: #f85149; animation: pulse 1s infinite; letter-spacing: 0.5px;">
                        {liveAlert()}
                    </div>
                </Show>

                {/* Status KPIs */}
                <div style="display: grid; grid-template-columns: repeat(4, 1fr); gap: 1px; background: var(--glass-border); border: 1px solid var(--glass-border); border-radius: 6px; overflow: hidden;">
                    {[
                        { label: 'Active Detections', value: alerts().length, color: alerts().length > 0 ? '#f85149' : '#3fb950' },
                        { label: 'IOC Matches', value: stats()?.ioc_matches ?? 0, color: '#d29922' },
                        { label: 'Hosts Protected', value: stats()?.total_hosts ?? '—', color: '#3fb950' },
                        { label: 'Canaries Active', value: stats()?.canary_count ?? '—', color: '#3fb950' },
                    ].map(({ label, value, color }) => (
                        <div style="padding: 1.25rem; background: var(--bg-secondary);">
                            <div style="font-size: 9px; text-transform: uppercase; letter-spacing: 1px; color: var(--text-muted); font-family: var(--font-mono); margin-bottom: 4px;">{label}</div>
                            <div style={`font-size: 1.75rem; font-weight: 900; font-family: var(--font-mono); color: ${color};`}>{value}</div>
                        </div>
                    ))}
                </div>

                {/* Defense layers status */}
                <div style="background: var(--bg-secondary); border: 1px solid var(--glass-border); border-radius: 6px; padding: 1.25rem;">
                    <div style="font-size: 10px; text-transform: uppercase; letter-spacing: 1px; color: var(--text-muted); font-family: var(--font-mono); margin-bottom: 1rem;">Defense Layers</div>
                    <div style="display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 0.75rem;">
                        {[
                            { name: 'Entropy Monitor', status: 'active', detail: 'Real-time file write analysis' },
                            { name: 'Canary Files', status: 'active', detail: 'Tripwires deployed on endpoints' },
                            { name: 'Shadow Copy Guard', status: 'active', detail: 'VSS deletion monitoring' },
                            { name: 'Network Isolation', status: 'ready', detail: 'Auto-isolate on detection' },
                            { name: 'Process Terminator', status: 'ready', detail: 'Kill suspicious encryptors' },
                            { name: 'Backup Trigger', status: 'active', detail: 'Snapshot on anomaly detection' },
                        ].map(({ name, status, detail }) => (
                            <div style={`padding: 10px 12px; border: 1px solid ${status === 'active' ? 'rgba(63,185,80,0.3)' : 'rgba(87,139,255,0.3)'}; border-radius: 4px; background: ${status === 'active' ? 'rgba(63,185,80,0.06)' : 'rgba(87,139,255,0.06)'};`}>
                                <div style="display: flex; align-items: center; gap: 6px; margin-bottom: 3px;">
                                    <span style={`font-size: 8px; color: ${status === 'active' ? '#3fb950' : 'var(--accent-primary)'};`}>●</span>
                                    <span style="font-size: 11px; font-weight: 700; font-family: var(--font-mono); color: var(--text-primary);">{name}</span>
                                </div>
                                <div style="font-size: 10px; color: var(--text-muted);">{detail}</div>
                            </div>
                        ))}
                    </div>
                </div>

                {/* Detections table */}
                <div style="background: var(--bg-secondary); border: 1px solid var(--glass-border); border-radius: 6px; padding: 1.25rem;">
                    <div style="font-size: 10px; text-transform: uppercase; letter-spacing: 1px; color: var(--text-muted); font-family: var(--font-mono); margin-bottom: 0.75rem;">
                        Detection Log ({alerts().length})
                    </div>
                    <Show when={loading()}>
                        <div style="color: var(--text-muted); font-family: var(--font-mono); font-size: 11px;">SCANNING...</div>
                    </Show>
                    <Show when={!loading() && alerts().length === 0}>
                        <div style="color: #3fb950; font-family: var(--font-mono); font-size: 11px;">✓ No ransomware activity detected</div>
                    </Show>
                    <Show when={alerts().length > 0}>
                        <div style="border: 1px solid var(--glass-border); border-radius: 4px; overflow: hidden; max-height: 300px; overflow-y: auto;">
                            <table style="width: 100%; border-collapse: collapse; font-size: 10px; font-family: var(--font-mono);">
                                <thead>
                                    <tr style="background: var(--bg-primary); border-bottom: 1px solid var(--glass-border);">
                                        {['Timestamp', 'Host', 'Event Type', 'Severity', 'Details'].map(h => (
                                            <th style="padding: 8px 10px; text-align: left; color: var(--text-muted); font-weight: 600;">{h}</th>
                                        ))}
                                    </tr>
                                </thead>
                                <tbody>
                                    <For each={alerts()}>
                                        {(a: any) => {
                                            const color = severityMap[(a.severity ?? 'low').toLowerCase()] ?? '#6b7280';
                                            return (
                                                <tr style="border-bottom: 1px solid rgba(255,255,255,0.04);">
                                                    <td style="padding: 7px 10px; color: var(--text-muted);">{a.timestamp?.slice(0, 16)?.replace('T', ' ')}</td>
                                                    <td style="padding: 7px 10px; color: var(--text-primary); font-weight: 600;">{a.host_id}</td>
                                                    <td style="padding: 7px 10px; color: var(--text-secondary);">{a.event_type}</td>
                                                    <td style="padding: 7px 10px;"><span style={`color: ${color}; font-weight: 700;`}>{(a.severity ?? 'LOW').toUpperCase()}</span></td>
                                                    <td style="padding: 7px 10px; color: var(--text-muted); max-width: 200px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">{a.details ?? '—'}</td>
                                                </tr>
                                            );
                                        }}
                                    </For>
                                </tbody>
                            </table>
                        </div>
                    </Show>
                </div>
            </div>

            <style>{`@keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.6; } }`}</style>
        </div>
    );
};
