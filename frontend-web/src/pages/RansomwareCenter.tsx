/**
 * RansomwareCenter.tsx — Ransomware Defense Center (Web — Phase 9)
 *
 * Fleet-wide ransomware status, active detections, and remediation controls.
 * Connects to /api/v1/ransomware/*.
 */

import { createSignal, createResource, For, Show } from 'solid-js';
import { request } from '../services/api';

interface RansomwareEvent {
    id: string;
    timestamp: string;
    host_id: string;
    type: 'entropy_spike' | 'canary_triggered' | 'shadow_copy_deleted' | 'mass_rename' | 'ransom_note';
    severity: 'critical' | 'high' | 'medium';
    details: string;
    isolated: boolean;
}

interface HostDefenseStatus {
    host_id: string;
    hostname: string;
    status: 'protected' | 'at_risk' | 'isolated' | 'active_threat';
    canary_count: number;
    last_scan: string;
    entropy_score?: number;
}

const SEV_COLOR: Record<string, string> = { critical: '#ff3355', high: '#ff6600', medium: '#ffaa00' };
const HOST_COLOR: Record<string, string> = { protected: '#00ff88', at_risk: '#ffaa00', isolated: '#ff6600', active_threat: '#ff3355' };

export default function RansomwareCenter() {
    const [tab, setTab] = createSignal<'overview' | 'events' | 'hosts'>('overview');
    const [isolatingHost, setIsolatingHost] = createSignal<string | null>(null);
    const [actionResult, setActionResult] = createSignal('');

    const [events, { refetch: refetchEvents }] = createResource<RansomwareEvent[]>(async () => {
        try { return await request<RansomwareEvent[]>('/ransomware/events?limit=50'); } catch { return []; }
    });

    const [hosts, { refetch: refetchHosts }] = createResource<HostDefenseStatus[]>(async () => {
        try { return await request<HostDefenseStatus[]>('/ransomware/hosts'); } catch { return []; }
    });

    const [stats] = createResource<Record<string, number>>(async () => {
        try { return await request<Record<string, number>>('/ransomware/stats'); } catch { return {}; }
    });

    const activeThreats = () => (events() ?? []).filter(e => !e.isolated).length;
    const atRiskHosts = () => (hosts() ?? []).filter(h => h.status === 'at_risk' || h.status === 'active_threat').length;

    const isolateHost = async (hostId: string) => {
        setIsolatingHost(hostId);
        try {
            await request('/ransomware/isolate', { method: 'POST', body: JSON.stringify({ host_id: hostId }) });
            setActionResult(`✓ Host ${hostId} isolated.`);
            refetchHosts();
        } catch (e: any) {
            setActionResult('✗ ' + (e?.message ?? e));
        }
        setIsolatingHost(null);
        setTimeout(() => setActionResult(''), 4000);
    };

    const TAB = (t: string) =>
        `padding:0.5rem 1.2rem; cursor:pointer; font-size:0.78rem; letter-spacing:0.12em; border:none; border-bottom:2px solid ${tab()===t ? '#ff3355' : 'transparent'}; background:none; color:${tab()===t ? '#ff3355' : '#607070'}; transition:color 0.15s;`;

    return (
        <div style="padding:2rem; color:#c8d8d8; font-family:'JetBrains Mono',monospace; min-height:100vh; background:#080f12;">
            <div style="display:flex; justify-content:space-between; align-items:flex-start; margin-bottom:1.5rem;">
                <div>
                    <h1 style="font-size:1.4rem; letter-spacing:0.15em; margin:0; color:#ff3355;">⬡ RANSOMWARE DEFENSE</h1>
                    <p style="margin:0.25rem 0 0; font-size:0.75rem; color:#607070;">Fleet-wide monitoring · Canary tripwires · Entropy analysis · Auto-isolation</p>
                </div>
                <div style="display:flex; gap:0.5rem;">
                    <button onClick={() => { refetchEvents(); refetchHosts(); }}
                        style="background:#1e3040; border:1px solid #607070; color:#607070; padding:0.4rem 0.9rem; border-radius:4px; cursor:pointer; font-size:0.76rem; letter-spacing:0.1em;">
                        ↻ REFRESH
                    </button>
                </div>
            </div>

            {/* Status strip */}
            <div style="display:grid; grid-template-columns:repeat(4,1fr); gap:1rem; margin-bottom:1.5rem;">
                {[
                    { label: 'ACTIVE THREATS', val: activeThreats(), color: activeThreats() > 0 ? '#ff3355' : '#00ff88' },
                    { label: 'AT-RISK HOSTS',  val: atRiskHosts(),   color: atRiskHosts() > 0 ? '#ffaa00' : '#00ff88' },
                    { label: 'TOTAL EVENTS',   val: (events() ?? []).length, color: '#ff6600' },
                    { label: 'PROTECTED HOSTS',val: (hosts() ?? []).filter(h => h.status === 'protected').length, color: '#00ff88' },
                ].map(s => (
                    <div style={`background:#0d1a1f; border:1px solid #1e3040; border-top:2px solid ${s.color}; padding:1rem; border-radius:4px;`}>
                        <div style={`font-size:1.6rem; font-weight:700; color:${s.color};`}>{s.val}</div>
                        <div style="font-size:0.65rem; color:#607070; letter-spacing:0.12em;">{s.label}</div>
                    </div>
                ))}
            </div>

            <Show when={actionResult()}>
                <div style={`padding:10px 14px; border-radius:4px; margin-bottom:1rem; font-size:0.78rem; background:${actionResult().startsWith('✓') ? '#002a1a' : '#2a0d15'}; border:1px solid ${actionResult().startsWith('✓') ? '#00ff88' : '#ff3355'}; color:${actionResult().startsWith('✓') ? '#00ff88' : '#ff3355'};`}>
                    {actionResult()}
                </div>
            </Show>

            {/* Tabs */}
            <div style="display:flex; border-bottom:1px solid #1e3040; margin-bottom:1.25rem;">
                {(['overview', 'events', 'hosts'] as const).map(t => (
                    <button style={TAB(t)} onClick={() => setTab(t)}>{t.toUpperCase()}</button>
                ))}
            </div>

            {/* ── Overview ── */}
            <Show when={tab() === 'overview'}>
                <div style="display:grid; grid-template-columns:repeat(auto-fill,minmax(220px,1fr)); gap:1rem; margin-bottom:1.5rem;">
                    {[
                        { name: 'Entropy Monitor',   status: 'active', detail: 'Real-time file write analysis' },
                        { name: 'Canary Tripwires',  status: 'active', detail: 'Deployed across all endpoints' },
                        { name: 'Shadow Copy Guard', status: 'active', detail: 'VSS deletion monitoring' },
                        { name: 'Network Isolation', status: 'ready',  detail: 'Auto-triggers on detection' },
                        { name: 'Process Terminator',status: 'ready',  detail: 'Kill suspicious encryptors' },
                        { name: 'Backup Trigger',    status: 'active', detail: 'Snapshot on anomaly detect' },
                    ].map(({ name, status, detail }) => (
                        <div style={`background:#0d1a1f; border:1px solid ${status === 'active' ? 'rgba(0,255,136,0.2)' : '#1e3040'}; border-radius:4px; padding:1rem;`}>
                            <div style="display:flex; align-items:center; gap:6px; margin-bottom:0.4rem;">
                                <span style={`font-size:8px; color:${status === 'active' ? '#00ff88' : '#00ffe7'};`}>●</span>
                                <span style="font-size:0.82rem; font-weight:600; color:#c8d8d8;">{name}</span>
                            </div>
                            <div style="font-size:0.7rem; color:#607070;">{detail}</div>
                        </div>
                    ))}
                </div>

                <Show when={(stats() && Object.keys(stats()!).length > 0)}>
                    <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; padding:1.25rem;">
                        <div style="font-size:0.65rem; color:#607070; letter-spacing:0.12em; margin-bottom:0.75rem;">FLEET STATISTICS</div>
                        <div style="display:grid; grid-template-columns:repeat(auto-fill,minmax(160px,1fr)); gap:1rem;">
                            <For each={Object.entries(stats() ?? {})}>
                                {([k, v]) => (
                                    <div>
                                        <div style="font-size:1.2rem; font-weight:700; color:#ff3355;">{v}</div>
                                        <div style="font-size:0.68rem; color:#607070; margin-top:2px;">{k.replace(/_/g,' ')}</div>
                                    </div>
                                )}
                            </For>
                        </div>
                    </div>
                </Show>
            </Show>

            {/* ── Events ── */}
            <Show when={tab() === 'events'}>
                <Show when={events.loading}><div style="color:#607070; padding:1rem;">Loading…</div></Show>
                <Show when={!events.loading && (events() ?? []).length === 0}>
                    <div style="color:#00ff88; padding:3rem; text-align:center; font-size:0.82rem;">✓ No ransomware activity detected across fleet.</div>
                </Show>
                <For each={events()}>
                    {(evt) => (
                        <div style={`background:#0d1a1f; border:1px solid ${SEV_COLOR[evt.severity] ?? '#1e3040'}22; border-left:3px solid ${SEV_COLOR[evt.severity] ?? '#1e3040'}; border-radius:4px; padding:1rem; margin-bottom:0.75rem;`}>
                            <div style="display:flex; justify-content:space-between; align-items:flex-start; margin-bottom:0.4rem;">
                                <div>
                                    <span style={`font-size:0.72rem; font-weight:700; color:${SEV_COLOR[evt.severity]}; letter-spacing:0.1em;`}>{evt.type.replace(/_/g,' ').toUpperCase()}</span>
                                    <span style="color:#607070; margin-left:0.75rem; font-size:0.72rem;">{evt.host_id}</span>
                                </div>
                                <span style="color:#607070; font-size:0.7rem;">{new Date(evt.timestamp).toLocaleTimeString()}</span>
                            </div>
                            <div style="font-size:0.76rem; color:#c8d8d8;">{evt.details}</div>
                            <Show when={evt.isolated}>
                                <span style="font-size:0.65rem; color:#ff6600; margin-top:0.3rem; display:inline-block;">🔒 HOST ISOLATED</span>
                            </Show>
                        </div>
                    )}
                </For>
            </Show>

            {/* ── Hosts ── */}
            <Show when={tab() === 'hosts'}>
                <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; overflow:hidden;">
                    <table style="width:100%; border-collapse:collapse; font-size:0.76rem;">
                        <thead>
                            <tr style="border-bottom:1px solid #1e3040; background:#0a1318;">
                                {['STATUS', 'HOSTNAME', 'CANARIES', 'ENTROPY', 'LAST SCAN', 'ACTION'].map(h => (
                                    <th style="padding:0.65rem 1rem; text-align:left; color:#607070; letter-spacing:0.1em; font-weight:400;">{h}</th>
                                ))}
                            </tr>
                        </thead>
                        <tbody>
                            <Show when={!hosts.loading} fallback={<tr><td colspan="6" style="padding:2rem; text-align:center; color:#607070;">Loading hosts…</td></tr>}>
                                <For each={hosts()} fallback={<tr><td colspan="6" style="padding:2rem; text-align:center; color:#607070;">No hosts registered.</td></tr>}>
                                    {(h) => {
                                        const color = HOST_COLOR[h.status] ?? '#607070';
                                        return (
                                            <tr style="border-bottom:1px solid #0a1318;"
                                                onMouseEnter={e => (e.currentTarget as HTMLElement).style.background='#111f28'}
                                                onMouseLeave={e => (e.currentTarget as HTMLElement).style.background=''}>
                                                <td style="padding:0.65rem 1rem;">
                                                    <span style={`font-size:0.68rem; color:${color}; letter-spacing:0.1em;`}>● {h.status.replace(/_/g,' ').toUpperCase()}</span>
                                                </td>
                                                <td style="padding:0.65rem 1rem; color:#c8d8d8;">{h.hostname || h.host_id}</td>
                                                <td style="padding:0.65rem 1rem; color:#607070;">{h.canary_count ?? '—'}</td>
                                                <td style="padding:0.65rem 1rem;">
                                                    <Show when={h.entropy_score !== undefined}>
                                                        <span style={`color:${h.entropy_score! > 7 ? '#ff3355' : h.entropy_score! > 5 ? '#ffaa00' : '#00ff88'};`}>
                                                            {h.entropy_score!.toFixed(2)}
                                                        </span>
                                                    </Show>
                                                </td>
                                                <td style="padding:0.65rem 1rem; color:#607070;">{h.last_scan ? new Date(h.last_scan).toLocaleString() : '—'}</td>
                                                <td style="padding:0.65rem 1rem;">
                                                    <Show when={h.status !== 'isolated'}>
                                                        <button
                                                            onClick={() => isolateHost(h.host_id)}
                                                            disabled={isolatingHost() === h.host_id}
                                                            style={`background:none; border:1px solid #ff3355; color:#ff3355; padding:0.25rem 0.75rem; border-radius:3px; cursor:pointer; font-size:0.7rem; letter-spacing:0.08em; opacity:${isolatingHost() === h.host_id ? 0.5 : 1};`}>
                                                            {isolatingHost() === h.host_id ? '…' : 'ISOLATE'}
                                                        </button>
                                                    </Show>
                                                </td>
                                            </tr>
                                        );
                                    }}
                                </For>
                            </Show>
                        </tbody>
                    </table>
                </div>
            </Show>
        </div>
    );
}
