/**
 * NDRDashboard.tsx — Network Detection & Response (Web — Phase 11)
 *
 * Live flow visualization, anomaly feed, and protocol inspection.
 * Connects to /api/v1/ndr/*.
 */

import { createSignal, createResource, For, Show } from 'solid-js';
import { request } from '../services/api';

interface NetFlow {
    id: string;
    timestamp: string;
    src_ip: string;
    dst_ip: string;
    src_port: number;
    dst_port: number;
    protocol: string;
    bytes: number;
    packets: number;
    flags: string;
    ja3?: string;
    anomaly?: boolean;
    anomaly_type?: string;
}

interface NDRAlert {
    id: string;
    timestamp: string;
    type: string;
    src_ip: string;
    dst_ip: string;
    confidence: number;
    description: string;
    mitre_technique?: string;
}

function anomalyColor(type?: string): string {
    if (!type) return '#607070';
    if (type.includes('lateral')) return '#ff3355';
    if (type.includes('c2') || type.includes('beacon')) return '#ff6600';
    if (type.includes('exfil') || type.includes('dns')) return '#ffaa00';
    return '#c8d8d8';
}

type Tab = 'flows' | 'alerts' | 'protocols';

export default function NDRDashboard() {
    const [tab, setTab] = createSignal<Tab>('flows');
    const [protoFilter, setProtoFilter] = createSignal('all');
    const [showAnomalyOnly, setShowAnomalyOnly] = createSignal(false);

    const [flows, { refetch: refetchFlows }] = createResource<NetFlow[]>(async () => {
        try { return await request<NetFlow[]>('/ndr/flows?limit=200'); } catch { return []; }
    });

    const [alerts] = createResource<NDRAlert[]>(async () => {
        try { return await request<NDRAlert[]>('/ndr/alerts?limit=50'); } catch { return []; }
    });

    const [protocolStats] = createResource<Record<string, number>>(async () => {
        try { return await request<Record<string, number>>('/ndr/protocols'); } catch { return {}; }
    });

    const filteredFlows = () => {
        let list = flows() ?? [];
        if (protoFilter() !== 'all') list = list.filter(f => f.protocol === protoFilter());
        if (showAnomalyOnly()) list = list.filter(f => f.anomaly);
        return list;
    };

    const protocols = () => ['all', ...new Set((flows() ?? []).map(f => f.protocol).filter(Boolean))];
    const anomalyCount = () => (flows() ?? []).filter(f => f.anomaly).length;

    const TAB = (t: Tab) =>
        `padding:0.5rem 1.2rem; cursor:pointer; font-size:0.78rem; letter-spacing:0.12em; border:none; border-bottom:2px solid ${tab()===t ? '#00ffe7' : 'transparent'}; background:none; color:${tab()===t ? '#00ffe7' : '#607070'}; transition:color 0.15s;`;

    return (
        <div style="padding:2rem; color:#c8d8d8; font-family:'JetBrains Mono',monospace; min-height:100vh; background:#080f12;">
            <div style="display:flex; justify-content:space-between; align-items:flex-start; margin-bottom:1.5rem;">
                <div>
                    <h1 style="font-size:1.4rem; letter-spacing:0.15em; margin:0; color:#00ffe7;">⬡ NDR DASHBOARD</h1>
                    <p style="margin:0.25rem 0 0; font-size:0.75rem; color:#607070;">Network flows · Lateral movement · Protocol analysis · C2 detection</p>
                </div>
                <button onClick={() => refetchFlows()} style="background:#1e3040; border:1px solid #00ffe7; color:#00ffe7; padding:0.4rem 0.9rem; border-radius:4px; cursor:pointer; font-size:0.76rem; letter-spacing:0.1em;">↻ REFRESH</button>
            </div>

            {/* Stats strip */}
            <div style="display:grid; grid-template-columns:repeat(4,1fr); gap:1rem; margin-bottom:1.5rem;">
                {[
                    { label: 'TOTAL FLOWS', val: (flows() ?? []).length, color: '#00ffe7' },
                    { label: 'ANOMALOUS',   val: anomalyCount(), color: '#ff3355' },
                    { label: 'NDR ALERTS',  val: (alerts() ?? []).length, color: '#ff6600' },
                    { label: 'PROTOCOLS',   val: Object.keys(protocolStats() ?? {}).length, color: '#00ff88' },
                ].map(s => (
                    <div style={`background:#0d1a1f; border:1px solid #1e3040; border-top:2px solid ${s.color}; padding:1rem; border-radius:4px;`}>
                        <div style={`font-size:1.6rem; font-weight:700; color:${s.color};`}>{s.val}</div>
                        <div style="font-size:0.65rem; color:#607070; letter-spacing:0.12em;">{s.label}</div>
                    </div>
                ))}
            </div>

            {/* Tabs */}
            <div style="display:flex; border-bottom:1px solid #1e3040; margin-bottom:1.25rem;">
                {(['flows', 'alerts', 'protocols'] as Tab[]).map(t => (
                    <button style={TAB(t)} onClick={() => setTab(t)}>{t.toUpperCase()}</button>
                ))}
            </div>

            {/* ── Flows ── */}
            <Show when={tab() === 'flows'}>
                <div style="display:flex; gap:0.75rem; margin-bottom:1rem; align-items:center;">
                    <select value={protoFilter()} onChange={e => setProtoFilter(e.currentTarget.value)}
                        style="background:#0d1a1f; border:1px solid #1e3040; color:#c8d8d8; padding:0.4rem 0.75rem; border-radius:3px; font-size:0.78rem;">
                        <For each={protocols()}>{p => <option value={p}>{p === 'all' ? 'All Protocols' : p}</option>}</For>
                    </select>
                    <label style="font-size:0.74rem; color:#607070; display:flex; align-items:center; gap:0.4rem; cursor:pointer;">
                        <input type="checkbox" checked={showAnomalyOnly()} onChange={e => setShowAnomalyOnly(e.currentTarget.checked)} />
                        Anomalies only
                    </label>
                    <span style="color:#607070; font-size:0.74rem; margin-left:auto;">{filteredFlows().length} flows</span>
                </div>
                <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; overflow:hidden;">
                    <table style="width:100%; border-collapse:collapse; font-size:0.74rem;">
                        <thead>
                            <tr style="border-bottom:1px solid #1e3040; background:#0a1318;">
                                {['TIME', 'SRC', 'DST', 'PROTO', 'BYTES', 'FLAGS', 'ANOMALY'].map(h => (
                                    <th style="padding:0.6rem 0.9rem; text-align:left; color:#607070; letter-spacing:0.1em; font-weight:400;">{h}</th>
                                ))}
                            </tr>
                        </thead>
                        <tbody>
                            <Show when={!flows.loading} fallback={<tr><td colspan="7" style="padding:1.5rem; text-align:center; color:#607070;">Loading flows…</td></tr>}>
                                <For each={filteredFlows().slice(0, 150)} fallback={
                                    <tr><td colspan="7" style="padding:2rem; text-align:center; color:#607070;">
                                        No flow data. Deploy NDR agents or configure NetFlow/IPFIX collectors.
                                    </td></tr>
                                }>
                                    {(f) => (
                                        <tr style={`border-bottom:1px solid #0a1318; ${f.anomaly ? 'background:rgba(255,51,85,0.04);' : ''}`}
                                            onMouseEnter={e => (e.currentTarget as HTMLElement).style.background='#111f28'}
                                            onMouseLeave={e => (e.currentTarget as HTMLElement).style.background= f.anomaly ? 'rgba(255,51,85,0.04)' : ''}>
                                            <td style="padding:0.55rem 0.9rem; color:#607070; white-space:nowrap;">{new Date(f.timestamp).toLocaleTimeString()}</td>
                                            <td style="padding:0.55rem 0.9rem; color:#c8d8d8;">{f.src_ip}:{f.src_port}</td>
                                            <td style="padding:0.55rem 0.9rem; color:#c8d8d8;">{f.dst_ip}:{f.dst_port}</td>
                                            <td style="padding:0.55rem 0.9rem; color:#00ffe7;">{f.protocol}</td>
                                            <td style="padding:0.55rem 0.9rem; color:#607070;">{f.bytes ? `${(f.bytes/1024).toFixed(1)}K` : '—'}</td>
                                            <td style="padding:0.55rem 0.9rem; color:#607070;">{f.flags || '—'}</td>
                                            <td style="padding:0.55rem 0.9rem;">
                                                <Show when={f.anomaly}>
                                                    <span style={`font-size:0.65rem; color:${anomalyColor(f.anomaly_type)}; font-weight:700; letter-spacing:0.08em;`}>
                                                        ⚠ {(f.anomaly_type ?? 'ANOMALY').replace(/_/g,' ').toUpperCase()}
                                                    </span>
                                                </Show>
                                            </td>
                                        </tr>
                                    )}
                                </For>
                            </Show>
                        </tbody>
                    </table>
                </div>
            </Show>

            {/* ── Alerts ── */}
            <Show when={tab() === 'alerts'}>
                <Show when={alerts.loading}><div style="color:#607070; padding:1rem;">Loading…</div></Show>
                <For each={alerts()} fallback={<div style="color:#00ff88; padding:2rem; text-align:center;">✓ No NDR alerts — network activity within baseline.</div>}>
                    {(a) => (
                        <div style={`background:#0d1a1f; border:1px solid #1e3040; border-left:3px solid ${anomalyColor(a.type)}; border-radius:4px; padding:1rem; margin-bottom:0.75rem;`}>
                            <div style="display:flex; justify-content:space-between; align-items:flex-start; margin-bottom:0.5rem;">
                                <div>
                                    <span style={`font-size:0.78rem; font-weight:700; color:${anomalyColor(a.type)};`}>{a.type.replace(/_/g,' ').toUpperCase()}</span>
                                    <span style="color:#607070; margin-left:0.75rem; font-size:0.72rem;">{a.src_ip} → {a.dst_ip}</span>
                                </div>
                                <div style="text-align:right;">
                                    <div style="color:#607070; font-size:0.7rem;">{new Date(a.timestamp).toLocaleTimeString()}</div>
                                    <div style="color:#ffaa00; font-size:0.68rem; margin-top:2px;">{Math.round(a.confidence * 100)}% confidence</div>
                                </div>
                            </div>
                            <div style="font-size:0.76rem; color:#c8d8d8; margin-bottom:0.35rem;">{a.description}</div>
                            <Show when={a.mitre_technique}>
                                <span style="background:#1e3040; color:#607070; padding:0.1rem 0.4rem; border-radius:2px; font-size:0.65rem;">{a.mitre_technique}</span>
                            </Show>
                        </div>
                    )}
                </For>
            </Show>

            {/* ── Protocols ── */}
            <Show when={tab() === 'protocols'}>
                <div style="display:grid; grid-template-columns:repeat(auto-fill,minmax(180px,1fr)); gap:1rem;">
                    <Show when={protocolStats.loading}><div style="color:#607070;">Loading…</div></Show>
                    <For each={Object.entries(protocolStats() ?? {}).sort((a, b) => b[1] - a[1])}>
                        {([proto, count]) => {
                            const total = Object.values(protocolStats() ?? {}).reduce((a, b) => a + b, 0) || 1;
                            const pct = Math.round(count / total * 100);
                            return (
                                <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; padding:1.25rem;">
                                    <div style="font-size:1.4rem; font-weight:700; color:#00ffe7;">{count.toLocaleString()}</div>
                                    <div style="font-size:0.78rem; color:#c8d8d8; margin:0.25rem 0;">{proto}</div>
                                    <div style="height:4px; background:#1e3040; border-radius:2px;">
                                        <div style={`height:100%; border-radius:2px; background:#00ffe7; width:${pct}%;`}></div>
                                    </div>
                                    <div style="font-size:0.65rem; color:#607070; margin-top:0.3rem;">{pct}% of traffic</div>
                                </div>
                            );
                        }}
                    </For>
                </div>
            </Show>
        </div>
    );
}
