// NDROverview.tsx — Phase 11 Web: flow visualization and NDR alerts
import { Component, createSignal, onMount, For, Show } from 'solid-js';
import * as NDRService from '../../../wailsjs/go/services/NDRService';

export const NDROverview: Component = () => {
    const [flows, setFlows] = createSignal<any[]>([]);
    const [anomalies, setAnomalies] = createSignal<any[]>([]);
    const [loading, setLoading] = createSignal(true);
    const [tab, setTab] = createSignal<'flows' | 'anomalies' | 'topology'>('flows');

    onMount(async () => {
        try {
            // GetLiveTraffic returns recent network flows
            const traffic = await (NDRService as any).GetLiveTraffic();
            const all = traffic ?? [];
            // Separate anomalous flows (those with a 'type' or 'anomaly' field) from normal flows
            setAnomalies(all.filter((f: any) => f.anomaly || f.type));
            setFlows(all.filter((f: any) => !f.anomaly && !f.type));
        } catch { }
        setLoading(false);
    });

    const anomalyColor = (type: string) => {
        if (type?.includes('lateral')) return '#f85149';
        if (type?.includes('beacon')) return '#d29922';
        if (type?.includes('exfil')) return '#f0883e';
        return '#6b7280';
    };

    return (
        <div style="padding: 0; height: 100%; overflow-y: auto; background: var(--bg-primary); color: var(--text-primary); font-family: var(--font-ui);">
            <div style="height: var(--header-height); border-bottom: 1px solid var(--glass-border); display: flex; justify-content: space-between; align-items: center; padding: 0 1.5rem; background: var(--bg-secondary);">
                <div style="display: flex; align-items: center; gap: 0.75rem;">
                    <span style="font-size: 16px;">🌐</span>
                    <h2 style="font-size: 13px; letter-spacing: 2px; font-weight: 700; margin: 0; text-transform: uppercase;">NDR Overview</h2>
                </div>
                <div style="display: flex; gap: 1.5rem; font-family: var(--font-mono); font-size: 10px;">
                    <span style="color: var(--text-muted);">{flows().length} FLOWS</span>
                    <span style={anomalies().length > 0 ? 'color: #f85149;' : 'color: #3fb950;'}>{anomalies().length} ANOMALIES</span>
                </div>
            </div>

            <div style="display: flex; gap: 0; border-bottom: 1px solid var(--glass-border); background: var(--bg-secondary);">
                {(['flows', 'anomalies', 'topology'] as const).map(t => (
                    <button onClick={() => setTab(t)}
                        style={`padding: 10px 20px; font-size: 11px; font-weight: 700; letter-spacing: 1px; text-transform: uppercase; font-family: var(--font-mono); border: none; cursor: pointer; background: transparent; border-bottom: 2px solid ${tab() === t ? 'var(--accent-primary)' : 'transparent'}; color: ${tab() === t ? 'var(--accent-primary)' : 'var(--text-muted)'};`}>
                        {t}
                    </button>
                ))}
            </div>

            <div style="padding: 1.5rem;">
                <Show when={loading()}>
                    <div style="padding: 3rem; text-align: center; color: var(--text-muted); font-family: var(--font-mono); font-size: 11px;">LOADING NDR DATA...</div>
                </Show>

                {/* Flows */}
                <Show when={!loading() && tab() === 'flows'}>
                    <Show when={flows().length === 0}>
                        <div style="padding: 4rem; text-align: center; color: var(--text-muted); font-family: var(--font-mono); font-size: 11px;">
                            <div style="font-size: 3rem; opacity: 0.2; margin-bottom: 1rem;">🌐</div>
                            NO FLOW DATA<br/><span style="font-size: 10px; opacity: 0.6; display: block; margin-top: 0.5rem;">Deploy NDR agents or configure NetFlow collectors to populate this view.</span>
                        </div>
                    </Show>
                    <Show when={flows().length > 0}>
                        <div style="border: 1px solid var(--glass-border); border-radius: 6px; overflow: hidden;">
                            <table style="width: 100%; border-collapse: collapse; font-size: 10px; font-family: var(--font-mono);">
                                <thead>
                                    <tr style="background: var(--bg-secondary); border-bottom: 1px solid var(--glass-border);">
                                        {['Timestamp', 'Src IP', 'Dst IP', 'Protocol', 'Bytes', 'Packets', 'Flags'].map(h => (
                                            <th style="padding: 9px 12px; text-align: left; color: var(--text-muted); font-weight: 600; letter-spacing: 0.5px;">{h}</th>
                                        ))}
                                    </tr>
                                </thead>
                                <tbody>
                                    <For each={flows().slice(0, 100)}>
                                        {(f: any) => (
                                            <tr style="border-bottom: 1px solid rgba(255,255,255,0.04);">
                                                <td style="padding: 7px 12px; color: var(--text-muted);">{f.timestamp?.slice(0, 19)?.replace('T', ' ') ?? '—'}</td>
                                                <td style="padding: 7px 12px; color: var(--text-primary);">{f.src_ip ?? '—'}</td>
                                                <td style="padding: 7px 12px; color: var(--text-primary);">{f.dst_ip ?? '—'}</td>
                                                <td style="padding: 7px 12px; color: var(--accent-primary);">{f.protocol ?? '—'}</td>
                                                <td style="padding: 7px 12px; color: var(--text-secondary);">{f.bytes?.toLocaleString?.() ?? '—'}</td>
                                                <td style="padding: 7px 12px; color: var(--text-secondary);">{f.packets ?? '—'}</td>
                                                <td style="padding: 7px 12px; color: var(--text-muted);">{f.flags ?? '—'}</td>
                                            </tr>
                                        )}
                                    </For>
                                </tbody>
                            </table>
                        </div>
                    </Show>
                </Show>

                {/* Anomalies */}
                <Show when={!loading() && tab() === 'anomalies'}>
                    <Show when={anomalies().length === 0}>
                        <div style="padding: 4rem; text-align: center; color: #3fb950; font-family: var(--font-mono); font-size: 12px;">✓ No network anomalies detected</div>
                    </Show>
                    <div style="display: flex; flex-direction: column; gap: 0.75rem;">
                        <For each={anomalies()}>
                            {(a: any) => {
                                const color = anomalyColor(a.type ?? '');
                                return (
                                    <div style={`background: var(--bg-secondary); border: 1px solid var(--glass-border); border-left: 3px solid ${color}; border-radius: 6px; padding: 1rem;`}>
                                        <div style="display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 6px;">
                                            <span style={`font-size: 12px; font-weight: 700; font-family: var(--font-mono); color: ${color};`}>{(a.type ?? 'UNKNOWN').toUpperCase().replace(/_/g, ' ')}</span>
                                            <span style="font-size: 10px; color: var(--text-muted); font-family: var(--font-mono);">{a.timestamp?.slice(0, 16)?.replace('T', ' ')}</span>
                                        </div>
                                        <div style="font-size: 11px; color: var(--text-secondary); margin-bottom: 6px;">{a.description ?? 'Anomalous network activity detected'}</div>
                                        <div style="display: flex; gap: 1rem; font-size: 10px; font-family: var(--font-mono); color: var(--text-muted);">
                                            <Show when={a.src_ip}><span>SRC: {a.src_ip}</span></Show>
                                            <Show when={a.dst_ip}><span>DST: {a.dst_ip}</span></Show>
                                            <Show when={a.confidence}><span>CONFIDENCE: {(a.confidence * 100).toFixed(0)}%</span></Show>
                                        </div>
                                    </div>
                                );
                            }}
                        </For>
                    </div>
                </Show>

                {/* Topology placeholder */}
                <Show when={!loading() && tab() === 'topology'}>
                    <div style="background: var(--bg-secondary); border: 1px solid var(--glass-border); border-radius: 6px; padding: 2rem; text-align: center; color: var(--text-muted);">
                        <div style="font-size: 2rem; margin-bottom: 1rem; opacity: 0.3;">🕸️</div>
                        <div style="font-family: var(--font-mono); font-size: 11px; letter-spacing: 1px; margin-bottom: 0.5rem;">NETWORK TOPOLOGY VISUALIZATION</div>
                        <div style="font-size: 10px; opacity: 0.6; max-width: 400px; margin: 0 auto; line-height: 1.5;">For interactive network topology with flow paths and lateral movement overlays, use the dedicated Network Map view at <code style="background: rgba(255,255,255,0.05); padding: 1px 4px; border-radius: 2px;">/ndr</code></div>
                    </div>
                </Show>
            </div>
        </div>
    );
};
