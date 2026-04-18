/**
 * FusionDashboard.tsx — Multi-Stage Attack Fusion Engine (Web — Phase 10.6)
 *
 * Kill chain progression, campaign clustering, probabilistic scoring.
 * Connects to /api/v1/fusion/*.
 */

import { createSignal, createResource, For, Show } from 'solid-js';
import { request } from '../services/api';

interface FusionCampaign {
    id: string;
    entities: string[];
    alert_count: number;
    tactic_stages: string[];
    stage_count: number;
    confidence: number;
    first_seen: string;
    last_seen: string;
    status: 'active' | 'closed' | 'investigating';
    kill_chain_progress: number; // 0-100
}

interface KillChainStage {
    tactic_id: string;
    tactic_name: string;
    hit_count: number;
    techniques: string[];
    first_seen?: string;
}

async function fetchCampaigns(): Promise<FusionCampaign[]> {
    try {
        const r = await request<{ campaigns: FusionCampaign[] }>('/fusion/campaigns');
        return r.campaigns ?? [];
    } catch {
        return [];
    }
}

async function fetchKillChain(campaignId: string): Promise<KillChainStage[]> {
    try {
        const r = await request<{ stages: KillChainStage[] }>(`/fusion/campaigns/${campaignId}/kill-chain`);
        return r.stages ?? [];
    } catch {
        return [];
    }
}

const TACTIC_ORDER = [
    'Initial Access', 'Execution', 'Persistence', 'Privilege Escalation',
    'Defense Evasion', 'Credential Access', 'Discovery', 'Lateral Movement',
    'Collection', 'Command & Control', 'Exfiltration', 'Impact',
];

function confidenceColor(c: number): string {
    if (c >= 0.8) return '#ff3355';
    if (c >= 0.6) return '#ff6600';
    if (c >= 0.4) return '#ffaa00';
    return '#00ff88';
}

function statusColor(s: string): string {
    if (s === 'active') return '#ff3355';
    if (s === 'investigating') return '#ffaa00';
    return '#607070';
}

export default function FusionDashboard() {
    const [selected, setSelected] = createSignal<FusionCampaign | null>(null);
    const [killChain, setKillChain] = createSignal<KillChainStage[]>([]);
    const [loadingKC, setLoadingKC] = createSignal(false);

    const [campaigns, { refetch }] = createResource(fetchCampaigns);

    const selectCampaign = async (c: FusionCampaign) => {
        setSelected(selected()?.id === c.id ? null : c);
        if (selected()?.id !== c.id) {
            setLoadingKC(true);
            const kc = await fetchKillChain(c.id);
            setKillChain(kc);
            setLoadingKC(false);
        }
    };

    const activeCampaigns = () => (campaigns() ?? []).filter(c => c.status === 'active').length;
    const highConfidence  = () => (campaigns() ?? []).filter(c => c.confidence >= 0.7).length;

    return (
        <div style="padding:2rem; color:#c8d8d8; font-family:'JetBrains Mono',monospace; min-height:100vh; background:#080f12;">
            <div style="display:flex; justify-content:space-between; align-items:flex-start; margin-bottom:1.5rem;">
                <div>
                    <h1 style="font-size:1.4rem; letter-spacing:0.15em; margin:0; color:#ff3355;">⬡ FUSION ENGINE</h1>
                    <p style="margin:0.25rem 0 0; font-size:0.75rem; color:#607070;">
                        Kill chain correlation · Campaign clustering · Probabilistic attack scoring
                    </p>
                </div>
                <button onClick={() => refetch()} style="background:#1e3040; border:1px solid #607070; color:#607070; padding:0.4rem 0.9rem; border-radius:4px; cursor:pointer; font-size:0.76rem;">↻ REFRESH</button>
            </div>

            {/* Stats */}
            <div style="display:grid; grid-template-columns:repeat(4,1fr); gap:1rem; margin-bottom:1.5rem;">
                {[
                    { label: 'TOTAL CAMPAIGNS',   val: campaigns()?.length ?? 0,  color: '#c8d8d8' },
                    { label: 'ACTIVE',             val: activeCampaigns(),          color: '#ff3355' },
                    { label: 'HIGH CONFIDENCE',    val: highConfidence(),            color: '#ff6600' },
                    { label: 'MAX STAGE COVERAGE', val: `${Math.max(0, ...(campaigns()??[]).map(c=>c.stage_count))}/12`, color: '#ffaa00' },
                ].map(s => (
                    <div style="background:#0d1a1f; border:1px solid #1e3040; border-top:2px solid #1e3040; padding:1rem; border-radius:4px;">
                        <div style={`font-size:1.6rem; font-weight:700; color:${s.color};`}>{s.val}</div>
                        <div style="font-size:0.65rem; color:#607070; letter-spacing:0.12em;">{s.label}</div>
                    </div>
                ))}
            </div>

            <Show when={campaigns.loading}><div style="color:#607070; padding:2rem; text-align:center;">Loading fusion data…</div></Show>

            <Show when={!campaigns.loading && (campaigns()??[]).length === 0}>
                <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; padding:4rem; text-align:center; color:#607070;">
                    <div style="font-size:2.5rem; margin-bottom:1rem; opacity:0.3;">🔗</div>
                    <div style="font-size:0.82rem; letter-spacing:0.1em; margin-bottom:0.5rem;">NO CAMPAIGNS DETECTED</div>
                    <div style="font-size:0.72rem; max-width:480px; margin:0 auto; line-height:1.6;">
                        The Fusion Engine correlates alerts sharing entities (IPs, users, hosts) across tactic stages. Campaigns appear when 3+ tactic stages are observed on the same entity cluster within a sliding time window.
                    </div>
                </div>
            </Show>

            <Show when={(campaigns()??[]).length > 0}>
                <div style="display:grid; grid-template-columns:360px 1fr; gap:1.5rem; align-items:start;">
                    {/* Campaign list */}
                    <div style="display:flex; flex-direction:column; gap:0.75rem;">
                        <For each={campaigns()}>
                            {(c) => {
                                const isSelected = () => selected()?.id === c.id;
                                const conf = Math.round(c.confidence * 100);
                                return (
                                    <div onClick={() => selectCampaign(c)}
                                        style={`background:#0d1a1f; border:1px solid ${isSelected() ? '#ff3355' : '#1e3040'}; border-left:3px solid ${statusColor(c.status)}; border-radius:4px; padding:1rem; cursor:pointer; transition:border-color 0.15s;`}>
                                        <div style="display:flex; justify-content:space-between; align-items:flex-start; margin-bottom:0.4rem;">
                                            <span style="font-size:0.78rem; font-weight:700; color:#c8d8d8; font-family:monospace;">{c.id.slice(0,12)}…</span>
                                            <span style={`font-size:0.65rem; font-weight:700; color:${statusColor(c.status)}; letter-spacing:0.1em;`}>{c.status.toUpperCase()}</span>
                                        </div>

                                        {/* Kill chain progress */}
                                        <div style="display:flex; gap:2px; margin-bottom:0.5rem;">
                                            {TACTIC_ORDER.slice(0, 12).map((t) => {
                                                const active = c.tactic_stages.some(s => s.toLowerCase().includes(t.toLowerCase().split(' ')[0]));
                                                return (
                                                    <div title={t} style={`flex:1; height:6px; border-radius:1px; background:${active ? confidenceColor(c.confidence) : '#1e3040'};`} />
                                                );
                                            })}
                                        </div>

                                        <div style="display:flex; gap:1rem; font-size:0.68rem; color:#607070;">
                                            <span>{c.stage_count}/12 stages</span>
                                            <span>{c.alert_count} alerts</span>
                                            <span style={`color:${confidenceColor(c.confidence)};`}>{conf}% confidence</span>
                                        </div>
                                        <div style="font-size:0.68rem; color:#607070; margin-top:0.25rem;">
                                            Entities: {c.entities.slice(0,3).join(', ')}{c.entities.length > 3 ? ` +${c.entities.length-3}` : ''}
                                        </div>
                                    </div>
                                );
                            }}
                        </For>
                    </div>

                    {/* Kill chain detail */}
                    <div>
                        <Show when={!selected()}>
                            <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; padding:3rem; text-align:center; color:#607070; font-size:0.78rem; letter-spacing:0.1em;">
                                <div style="font-size:2.5rem; margin-bottom:1rem; opacity:0.3;">🔗</div>
                                SELECT A CAMPAIGN TO VIEW KILL CHAIN
                            </div>
                        </Show>

                        <Show when={selected()}>
                            {(c) => (
                                <div style="display:flex; flex-direction:column; gap:1rem;">
                                    {/* Campaign header */}
                                    <div style="background:#0d1a1f; border:1px solid #1e3040; border-top:2px solid #ff3355; border-radius:6px; padding:1.25rem;">
                                        <div style="display:flex; justify-content:space-between; align-items:flex-start;">
                                            <div>
                                                <div style="font-size:0.88rem; color:#ff3355; letter-spacing:0.08em; margin-bottom:0.4rem;">{c().id}</div>
                                                <div style="font-size:0.72rem; color:#607070;">
                                                    First seen: {new Date(c().first_seen).toLocaleString()} &nbsp;·&nbsp;
                                                    Last activity: {new Date(c().last_seen).toLocaleString()}
                                                </div>
                                            </div>
                                            <div style="text-align:right;">
                                                <div style={`font-size:1.5rem; font-weight:700; color:${confidenceColor(c().confidence)};`}>{Math.round(c().confidence*100)}%</div>
                                                <div style="font-size:0.65rem; color:#607070;">confidence</div>
                                            </div>
                                        </div>
                                        <div style="margin-top:0.75rem; display:flex; flex-wrap:wrap; gap:0.4rem;">
                                            <For each={c().entities}>
                                                {(e) => <span style="background:#1e3040; color:#c8d8d8; padding:0.15rem 0.5rem; border-radius:2px; font-size:0.68rem;">{e}</span>}
                                            </For>
                                        </div>
                                    </div>

                                    {/* Kill chain visualization */}
                                    <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; padding:1.25rem;">
                                        <div style="font-size:0.65rem; color:#607070; letter-spacing:0.12em; margin-bottom:1rem;">KILL CHAIN PROGRESSION</div>
                                        <Show when={loadingKC()}><div style="color:#607070; font-size:0.76rem;">Loading kill chain…</div></Show>
                                        <Show when={!loadingKC() && killChain().length === 0}>
                                            {/* Render from campaign data */}
                                            <div style="display:flex; align-items:center; gap:4px; flex-wrap:nowrap; overflow-x:auto; padding-bottom:0.5rem;">
                                                {TACTIC_ORDER.map((tactic, i) => {
                                                    const active = c().tactic_stages.some(s => s.toLowerCase().includes(tactic.toLowerCase().split(' ')[0]));
                                                    return (
                                                        <>
                                                            <div style={`flex-shrink:0; padding:8px 10px; border-radius:3px; font-size:0.65rem; letter-spacing:0.06em; text-align:center; min-width:80px; background:${active ? '#2a0d15' : '#0a1318'}; border:1px solid ${active ? '#ff3355' : '#1e3040'}; color:${active ? '#ff3355' : '#3a5060'};`}>
                                                                <div style="font-weight:700; margin-bottom:2px;">{active ? '●' : '○'}</div>
                                                                {tactic}
                                                            </div>
                                                            {i < 11 && <div style={`font-size:14px; color:${active ? '#ff3355' : '#1e3040'}; flex-shrink:0;`}>→</div>}
                                                        </>
                                                    );
                                                })}
                                            </div>
                                        </Show>
                                        <Show when={!loadingKC() && killChain().length > 0}>
                                            <div style="display:flex; align-items:center; gap:4px; flex-wrap:nowrap; overflow-x:auto; padding-bottom:0.5rem;">
                                                <For each={TACTIC_ORDER}>
                                                    {(tactic, i) => {
                                                        const stage = killChain().find(s => s.tactic_name.toLowerCase().includes(tactic.toLowerCase().split(' ')[0]));
                                                        return (
                                                            <>
                                                                <div title={stage ? `${stage.hit_count} hits: ${stage.techniques.join(', ')}` : 'Not observed'}
                                                                    style={`flex-shrink:0; padding:8px 10px; border-radius:3px; font-size:0.65rem; text-align:center; min-width:80px; background:${stage ? '#2a0d15' : '#0a1318'}; border:1px solid ${stage ? '#ff3355' : '#1e3040'}; color:${stage ? '#ff3355' : '#3a5060'}; cursor:${stage ? 'help' : 'default'};`}>
                                                                    <div style="font-weight:700; margin-bottom:2px;">{stage ? stage.hit_count : '○'}</div>
                                                                    {tactic}
                                                                </div>
                                                                {i() < 11 && <div style={`font-size:14px; color:${stage ? '#ff3355' : '#1e3040'}; flex-shrink:0;`}>→</div>}
                                                            </>
                                                        );
                                                    }}
                                                </For>
                                            </div>
                                        </Show>
                                    </div>
                                </div>
                            )}
                        </Show>
                    </div>
                </div>
            </Show>
        </div>
    );
}
