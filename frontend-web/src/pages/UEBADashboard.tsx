/**
 * UEBADashboard.tsx — User & Entity Behavior Analytics (Web — Phase 10)
 *
 * Risk-ranked entity profiles, anomaly feed, and behavioral baseline overview.
 * Connects to /api/v1/ueba/*.
 */

import { createSignal, createResource, For, Show } from 'solid-js';
import { request } from '../services/api';

interface EntityProfile {
    entity_id: string;
    entity_type: 'user' | 'host' | 'service';
    risk_score: number;
    baseline_established: boolean;
    anomaly_count: number;
    last_activity: string;
    top_anomaly?: string;
}

interface AnomalyEvent {
    id: string;
    entity_id: string;
    entity_type: string;
    anomaly_type: string;
    score: number;
    timestamp: string;
    description: string;
    mitre_technique?: string;
}

function riskColor(score: number) {
    if (score >= 80) return '#ff3355';
    if (score >= 60) return '#ff6600';
    if (score >= 40) return '#ffaa00';
    return '#00ff88';
}

function riskLabel(score: number) {
    if (score >= 80) return 'CRITICAL';
    if (score >= 60) return 'HIGH';
    if (score >= 40) return 'MEDIUM';
    return 'LOW';
}

type Tab = 'profiles' | 'anomalies' | 'stats';

export default function UEBADashboard() {
    const [tab, setTab] = createSignal<Tab>('profiles');
    const [search, setSearch] = createSignal('');
    const [sortByRisk, setSortByRisk] = createSignal(true);
    const [selectedProfile, setSelectedProfile] = createSignal<EntityProfile | null>(null);

    const [profiles, { refetch: refetchProfiles }] = createResource<EntityProfile[]>(async () => {
        try { return await request<EntityProfile[]>('/ueba/profiles'); } catch { return []; }
    });

    const [anomalies] = createResource<AnomalyEvent[]>(async () => {
        try { return await request<AnomalyEvent[]>('/ueba/anomalies?limit=100'); } catch { return []; }
    });

    const [stats] = createResource<Record<string, number>>(async () => {
        try { return await request<Record<string, number>>('/ueba/stats'); } catch { return {}; }
    });

    const filtered = () => {
        const q = search().toLowerCase();
        let list = (profiles() ?? []).filter(p =>
            !q || p.entity_id.toLowerCase().includes(q) || p.entity_type.includes(q)
        );
        if (sortByRisk()) list = [...list].sort((a, b) => b.risk_score - a.risk_score);
        return list;
    };

    const criticalCount = () => (profiles() ?? []).filter(p => p.risk_score >= 80).length;
    const avgRisk = () => {
        const p = profiles() ?? [];
        if (!p.length) return 0;
        return Math.round(p.reduce((a, x) => a + x.risk_score, 0) / p.length);
    };

    const TAB = (t: Tab) =>
        `padding:0.5rem 1.2rem; cursor:pointer; font-size:0.78rem; letter-spacing:0.12em; border:none; border-bottom:2px solid ${tab()===t ? '#ff6600' : 'transparent'}; background:none; color:${tab()===t ? '#ff6600' : '#607070'}; transition:color 0.15s;`;

    return (
        <div style="padding:2rem; color:#c8d8d8; font-family:'JetBrains Mono',monospace; min-height:100vh; background:#080f12;">
            <div style="display:flex; justify-content:space-between; align-items:flex-start; margin-bottom:1.5rem;">
                <div>
                    <h1 style="font-size:1.4rem; letter-spacing:0.15em; margin:0; color:#ffaa00;">⬡ UEBA DASHBOARD</h1>
                    <p style="margin:0.25rem 0 0; font-size:0.75rem; color:#607070;">Behavioral baselines · Anomaly detection · Entity risk scoring</p>
                </div>
                <button onClick={() => refetchProfiles()} style="background:#1e3040; border:1px solid #607070; color:#607070; padding:0.4rem 0.9rem; border-radius:4px; cursor:pointer; font-size:0.76rem; letter-spacing:0.1em;">↻ REFRESH</button>
            </div>

            {/* Stats strip */}
            <div style="display:grid; grid-template-columns:repeat(4,1fr); gap:1rem; margin-bottom:1.5rem;">
                {[
                    { label: 'TOTAL ENTITIES', val: (profiles() ?? []).length, color: '#ffaa00' },
                    { label: 'CRITICAL RISK',  val: criticalCount(), color: '#ff3355' },
                    { label: 'AVG RISK SCORE', val: avgRisk(), color: riskColor(avgRisk()) },
                    { label: 'ANOMALIES (24H)', val: (anomalies() ?? []).length, color: '#ff6600' },
                ].map(s => (
                    <div style={`background:#0d1a1f; border:1px solid #1e3040; border-top:2px solid ${s.color}; padding:1rem; border-radius:4px;`}>
                        <div style={`font-size:1.6rem; font-weight:700; color:${s.color};`}>{s.val}</div>
                        <div style="font-size:0.65rem; color:#607070; letter-spacing:0.12em;">{s.label}</div>
                    </div>
                ))}
            </div>

            {/* Tabs */}
            <div style="display:flex; border-bottom:1px solid #1e3040; margin-bottom:1.25rem;">
                {(['profiles', 'anomalies', 'stats'] as Tab[]).map(t => (
                    <button style={TAB(t)} onClick={() => setTab(t)}>{t.toUpperCase()}</button>
                ))}
            </div>

            {/* ── Profiles ── */}
            <Show when={tab() === 'profiles'}>
                <div style="display:grid; grid-template-columns:1fr 300px; gap:1rem; align-items:start;">
                    <div>
                        <div style="display:flex; gap:0.75rem; margin-bottom:1rem; align-items:center;">
                            <input type="text" value={search()} onInput={e => setSearch(e.currentTarget.value)}
                                placeholder="Filter entities…"
                                style="background:#0d1a1f; border:1px solid #1e3040; color:#c8d8d8; padding:0.4rem 0.75rem; border-radius:3px; font-size:0.78rem; width:220px;" />
                            <label style="font-size:0.74rem; color:#607070; display:flex; align-items:center; gap:0.4rem; cursor:pointer;">
                                <input type="checkbox" checked={sortByRisk()} onChange={e => setSortByRisk(e.currentTarget.checked)} />
                                Sort by risk
                            </label>
                        </div>
                        <Show when={profiles.loading}>
                            <div style="color:#607070; padding:2rem; text-align:center;">Loading profiles…</div>
                        </Show>
                        <div style="display:grid; grid-template-columns:repeat(auto-fill,minmax(260px,1fr)); gap:0.75rem;">
                            <For each={filtered()} fallback={
                                <div style="color:#607070; padding:2rem; grid-column:1/-1; text-align:center; font-size:0.78rem;">
                                    No behavioral profiles. Profiles build automatically from agent activity (24–48h baseline window).
                                </div>
                            }>
                                {(p) => {
                                    const color = riskColor(p.risk_score);
                                    const isSelected = () => selectedProfile()?.entity_id === p.entity_id;
                                    return (
                                        <div onClick={() => setSelectedProfile(isSelected() ? null : p)}
                                            style={`background:#0d1a1f; border:1px solid ${isSelected() ? '#ff6600' : '#1e3040'}; border-left:3px solid ${color}; border-radius:4px; padding:1rem; cursor:pointer; transition:border-color 0.15s;`}>
                                            <div style="display:flex; justify-content:space-between; align-items:flex-start; margin-bottom:0.5rem;">
                                                <div>
                                                    <div style="font-size:0.82rem; color:#c8d8d8; font-weight:600;">{p.entity_id}</div>
                                                    <div style="font-size:0.65rem; color:#607070;">{p.entity_type}</div>
                                                </div>
                                                <div style="text-align:right;">
                                                    <div style={`font-size:1.3rem; font-weight:700; color:${color}; line-height:1;`}>{p.risk_score}</div>
                                                    <div style={`font-size:0.6rem; color:${color}; letter-spacing:0.08em;`}>{riskLabel(p.risk_score)}</div>
                                                </div>
                                            </div>
                                            <div style={`height:3px; background:#1e3040; border-radius:2px; overflow:hidden;`}>
                                                <div style={`height:100%; width:${Math.min(100, p.risk_score)}%; background:${color};`} />
                                            </div>
                                            <Show when={p.anomaly_count > 0}>
                                                <div style="font-size:0.68rem; color:#607070; margin-top:0.4rem;">{p.anomaly_count} anomalies</div>
                                            </Show>
                                        </div>
                                    );
                                }}
                            </For>
                        </div>
                    </div>

                    {/* Detail panel */}
                    <Show when={selectedProfile()}>
                        {(p) => (
                            <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; padding:1.25rem; position:sticky; top:1rem;">
                                <div style="font-size:0.65rem; color:#607070; letter-spacing:0.12em; margin-bottom:0.75rem;">ENTITY DETAIL</div>
                                <div style={`font-size:1.8rem; font-weight:700; color:${riskColor(p().risk_score)}; margin-bottom:0.25rem;`}>{p().risk_score}</div>
                                <div style="font-size:0.82rem; color:#c8d8d8; margin-bottom:1rem;">{p().entity_id}</div>
                                {[
                                    ['Type', p().entity_type],
                                    ['Baseline', p().baseline_established ? '✓ Established' : '⏳ Building...'],
                                    ['Anomalies', String(p().anomaly_count)],
                                    ['Last Activity', p().last_activity ? new Date(p().last_activity).toLocaleString() : '—'],
                                    ['Top Anomaly', p().top_anomaly ?? '—'],
                                ].map(([k, v]) => (
                                    <div style="display:flex; justify-content:space-between; margin-bottom:0.5rem; font-size:0.74rem; border-bottom:1px solid #1e3040; padding-bottom:0.5rem;">
                                        <span style="color:#607070;">{k}</span>
                                        <span style="color:#c8d8d8;">{v}</span>
                                    </div>
                                ))}
                            </div>
                        )}
                    </Show>
                </div>
            </Show>

            {/* ── Anomalies ── */}
            <Show when={tab() === 'anomalies'}>
                <Show when={anomalies.loading}><div style="color:#607070;">Loading…</div></Show>
                <For each={anomalies()} fallback={<div style="color:#607070; padding:2rem; text-align:center;">No anomalies detected. All entities within baseline.</div>}>
                    {(a) => (
                        <div style={`background:#0d1a1f; border:1px solid #1e3040; border-left:3px solid ${riskColor(a.score)}; border-radius:4px; padding:1rem; margin-bottom:0.75rem;`}>
                            <div style="display:flex; justify-content:space-between; align-items:flex-start; margin-bottom:0.5rem;">
                                <div>
                                    <span style={`font-size:0.72rem; font-weight:700; color:${riskColor(a.score)}; letter-spacing:0.1em;`}>{a.anomaly_type.replace(/_/g,' ').toUpperCase()}</span>
                                    <span style="color:#607070; margin-left:0.75rem; font-size:0.72rem;">{a.entity_id}</span>
                                </div>
                                <span style="color:#607070; font-size:0.7rem;">{new Date(a.timestamp).toLocaleTimeString()}</span>
                            </div>
                            <div style="font-size:0.76rem; color:#c8d8d8; margin-bottom:0.25rem;">{a.description}</div>
                            <Show when={a.mitre_technique}>
                                <span style="background:#1e3040; color:#607070; padding:0.1rem 0.4rem; border-radius:2px; font-size:0.65rem;">{a.mitre_technique}</span>
                            </Show>
                        </div>
                    )}
                </For>
            </Show>

            {/* ── Stats ── */}
            <Show when={tab() === 'stats'}>
                <div style="display:grid; grid-template-columns:repeat(auto-fill,minmax(200px,1fr)); gap:1rem;">
                    <For each={Object.entries(stats() ?? {})}>
                        {([k, v]) => (
                            <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; padding:1.25rem;">
                                <div style="font-size:1.6rem; font-weight:700; color:#ffaa00;">{v}</div>
                                <div style="font-size:0.72rem; color:#607070; margin-top:0.25rem;">{k.replace(/_/g,' ')}</div>
                            </div>
                        )}
                    </For>
                </div>
            </Show>
        </div>
    );
}
