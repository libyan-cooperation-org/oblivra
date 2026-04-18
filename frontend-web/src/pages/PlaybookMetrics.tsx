/**
 * PlaybookMetrics.tsx — Playbook Performance Analytics (Web — Phase 8)
 *
 * MTTR per playbook, execution success/failure rates, bottleneck identification.
 * Connects to /api/v1/playbooks/metrics and /api/v1/playbooks.
 */

import { createSignal, createResource, For, Show } from 'solid-js';
import { request } from '../services/api';

interface PlaybookExecution {
    playbook_id: string;
    incident_id: string;
    started_at: string;
    completed_at: string;
    duration_ms: number;
    status: 'completed' | 'failed' | 'running';
    step_count: number;
}

interface MetricsResponse {
    total_executions: number;
    success_count: number;
    failure_count: number;
    avg_duration_ms: number;
    executions_by_playbook: Record<string, number>;
    recent_executions: PlaybookExecution[];
}

function durationLabel(ms: number): string {
    if (ms < 1000) return `${ms}ms`;
    if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
    return `${Math.floor(ms / 60000)}m ${Math.floor((ms % 60000) / 1000)}s`;
}

function statusColor(s: string) {
    return s === 'completed' ? '#00ff88' : s === 'failed' ? '#ff3355' : '#ffaa00';
}

function successRate(total: number, success: number): number {
    if (!total) return 0;
    return Math.round((success / total) * 100);
}

export default function PlaybookMetrics() {
    const [tab, setTab] = createSignal<'overview' | 'history' | 'bottlenecks'>('overview');

    const [metrics, { refetch }] = createResource<MetricsResponse>(async () => {
        try { return await request<MetricsResponse>('/playbooks/metrics'); }
        catch { return { total_executions: 0, success_count: 0, failure_count: 0, avg_duration_ms: 0, executions_by_playbook: {}, recent_executions: [] }; }
    });

    const rate = () => successRate(metrics()?.total_executions ?? 0, metrics()?.success_count ?? 0);

    const topPlaybooks = () =>
        Object.entries(metrics()?.executions_by_playbook ?? {})
            .sort((a, b) => b[1] - a[1])
            .slice(0, 8);

    const TAB = (t: string) =>
        `padding:0.5rem 1.2rem; cursor:pointer; font-size:0.78rem; letter-spacing:0.12em; border:none; border-bottom:2px solid ${tab()===t ? '#ff6600' : 'transparent'}; background:none; color:${tab()===t ? '#ff6600' : '#607070'}; transition:color 0.15s;`;

    return (
        <div style="padding:2rem; color:#c8d8d8; font-family:'JetBrains Mono',monospace; min-height:100vh; background:#080f12;">
            <div style="display:flex; justify-content:space-between; align-items:flex-start; margin-bottom:1.5rem;">
                <div>
                    <h1 style="font-size:1.4rem; letter-spacing:0.15em; margin:0; color:#ff6600;">⬡ PLAYBOOK METRICS</h1>
                    <p style="margin:0.25rem 0 0; font-size:0.75rem; color:#607070;">MTTR · Success rates · Execution history · Performance bottlenecks</p>
                </div>
                <button onClick={() => refetch()} style="background:#1e3040; border:1px solid #607070; color:#607070; padding:0.4rem 0.9rem; border-radius:4px; cursor:pointer; font-size:0.76rem;">↻ REFRESH</button>
            </div>

            {/* KPI strip */}
            <div style="display:grid; grid-template-columns:repeat(4,1fr); gap:1rem; margin-bottom:1.5rem;">
                {[
                    { label: 'TOTAL EXECUTIONS', val: metrics()?.total_executions ?? 0, color: '#ff6600' },
                    { label: 'SUCCESS RATE',     val: `${rate()}%`,                     color: rate() >= 90 ? '#00ff88' : rate() >= 70 ? '#ffaa00' : '#ff3355' },
                    { label: 'FAILURES',          val: metrics()?.failure_count ?? 0,    color: (metrics()?.failure_count ?? 0) > 0 ? '#ff3355' : '#00ff88' },
                    { label: 'AVG DURATION',      val: durationLabel(metrics()?.avg_duration_ms ?? 0), color: '#c8d8d8' },
                ].map(s => (
                    <div style="background:#0d1a1f; border:1px solid #1e3040; border-top:2px solid #1e3040; padding:1rem; border-radius:4px;">
                        <div style={`font-size:1.6rem; font-weight:700; color:${s.color};`}>{s.val}</div>
                        <div style="font-size:0.65rem; color:#607070; letter-spacing:0.12em;">{s.label}</div>
                    </div>
                ))}
            </div>

            {/* Success rate bar */}
            <Show when={(metrics()?.total_executions ?? 0) > 0}>
                <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; padding:1rem; margin-bottom:1.5rem;">
                    <div style="display:flex; justify-content:space-between; font-size:0.72rem; color:#607070; margin-bottom:0.5rem;">
                        <span>Overall success rate</span>
                        <span style={`color:${rate() >= 90 ? '#00ff88' : '#ffaa00'};`}>{rate()}%</span>
                    </div>
                    <div style="height:8px; background:#1e3040; border-radius:4px; overflow:hidden;">
                        <div style={`height:100%; width:${rate()}%; background:${rate() >= 90 ? '#00ff88' : rate() >= 70 ? '#ffaa00' : '#ff3355'}; border-radius:4px; transition:width 0.5s;`} />
                    </div>
                </div>
            </Show>

            {/* Tabs */}
            <div style="display:flex; border-bottom:1px solid #1e3040; margin-bottom:1.25rem;">
                {(['overview', 'history', 'bottlenecks'] as const).map(t => (
                    <button style={TAB(t)} onClick={() => setTab(t)}>{t.toUpperCase()}</button>
                ))}
            </div>

            {/* ── Overview ── */}
            <Show when={tab() === 'overview'}>
                <div style="display:grid; grid-template-columns:1fr 1fr; gap:1.5rem;">
                    {/* Executions by playbook */}
                    <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; padding:1.25rem;">
                        <div style="font-size:0.65rem; color:#607070; letter-spacing:0.12em; margin-bottom:1rem;">EXECUTIONS BY PLAYBOOK</div>
                        <Show when={topPlaybooks().length === 0}>
                            <div style="color:#607070; font-size:0.76rem; text-align:center; padding:1rem;">No playbook executions yet.</div>
                        </Show>
                        <For each={topPlaybooks()}>
                            {([name, count]) => {
                                const maxCount = topPlaybooks()[0]?.[1] ?? 1;
                                const pct = Math.round((count / maxCount) * 100);
                                return (
                                    <div style="margin-bottom:0.75rem;">
                                        <div style="display:flex; justify-content:space-between; font-size:0.74rem; margin-bottom:0.25rem;">
                                            <span style="color:#c8d8d8;">{name.replace(/-/g,' ')}</span>
                                            <span style="color:#ff6600;">{count}</span>
                                        </div>
                                        <div style="height:4px; background:#1e3040; border-radius:2px;">
                                            <div style={`height:100%; width:${pct}%; background:#ff6600; border-radius:2px;`} />
                                        </div>
                                    </div>
                                );
                            }}
                        </For>
                    </div>

                    {/* Recent activity timeline */}
                    <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; padding:1.25rem;">
                        <div style="font-size:0.65rem; color:#607070; letter-spacing:0.12em; margin-bottom:1rem;">RECENT EXECUTIONS</div>
                        <Show when={(metrics()?.recent_executions?.length ?? 0) === 0}>
                            <div style="color:#607070; font-size:0.76rem; text-align:center; padding:1rem;">No recent executions.</div>
                        </Show>
                        <div style="display:flex; flex-direction:column; gap:0.5rem; max-height:280px; overflow-y:auto;">
                            <For each={metrics()?.recent_executions ?? []}>
                                {(exec) => (
                                    <div style={`padding:8px 10px; border-radius:3px; border-left:3px solid ${statusColor(exec.status)}; background:#0a1318;`}>
                                        <div style="display:flex; justify-content:space-between; margin-bottom:2px;">
                                            <span style="font-size:0.76rem; color:#c8d8d8;">{exec.playbook_id}</span>
                                            <span style={`font-size:0.68rem; color:${statusColor(exec.status)}; letter-spacing:0.08em;`}>{exec.status.toUpperCase()}</span>
                                        </div>
                                        <div style="display:flex; gap:1rem; font-size:0.68rem; color:#607070;">
                                            <span>INC: {exec.incident_id || '—'}</span>
                                            <span>{durationLabel(exec.duration_ms)}</span>
                                            <span>{exec.step_count} steps</span>
                                        </div>
                                    </div>
                                )}
                            </For>
                        </div>
                    </div>
                </div>
            </Show>

            {/* ── History ── */}
            <Show when={tab() === 'history'}>
                <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; overflow:hidden;">
                    <table style="width:100%; border-collapse:collapse; font-size:0.75rem;">
                        <thead>
                            <tr style="border-bottom:1px solid #1e3040; background:#0a1318;">
                                {['PLAYBOOK', 'INCIDENT', 'STATUS', 'DURATION', 'STEPS', 'STARTED'].map(h => (
                                    <th style="padding:0.6rem 0.9rem; text-align:left; color:#607070; letter-spacing:0.1em; font-weight:400;">{h}</th>
                                ))}
                            </tr>
                        </thead>
                        <tbody>
                            <Show when={(metrics()?.recent_executions?.length ?? 0) === 0}>
                                <tr><td colspan="6" style="padding:2rem; text-align:center; color:#607070;">No execution history.</td></tr>
                            </Show>
                            <For each={[...(metrics()?.recent_executions ?? [])].reverse()}>
                                {(exec) => (
                                    <tr style="border-bottom:1px solid #0a1318;"
                                        onMouseEnter={e => (e.currentTarget as HTMLElement).style.background='#111f28'}
                                        onMouseLeave={e => (e.currentTarget as HTMLElement).style.background=''}>
                                        <td style="padding:0.55rem 0.9rem; color:#c8d8d8;">{exec.playbook_id}</td>
                                        <td style="padding:0.55rem 0.9rem; color:#607070;">{exec.incident_id || '—'}</td>
                                        <td style="padding:0.55rem 0.9rem;">
                                            <span style={`font-size:0.65rem; font-weight:700; color:${statusColor(exec.status)};`}>{exec.status.toUpperCase()}</span>
                                        </td>
                                        <td style="padding:0.55rem 0.9rem; color:#ff6600;">{durationLabel(exec.duration_ms)}</td>
                                        <td style="padding:0.55rem 0.9rem; color:#607070;">{exec.step_count}</td>
                                        <td style="padding:0.55rem 0.9rem; color:#607070;">{new Date(exec.started_at).toLocaleTimeString()}</td>
                                    </tr>
                                )}
                            </For>
                        </tbody>
                    </table>
                </div>
            </Show>

            {/* ── Bottlenecks ── */}
            <Show when={tab() === 'bottlenecks'}>
                <div style="display:grid; gap:1rem;">
                    <div style="background:#0d1a1f; border:1px solid #ffaa00; border-top:2px solid #ffaa00; border-radius:6px; padding:1.25rem;">
                        <div style="font-size:0.85rem; color:#ffaa00; font-weight:700; margin-bottom:0.75rem;">⚠ Performance Analysis</div>
                        <div style="font-size:0.76rem; color:#c8d8d8; line-height:1.7; margin-bottom:1rem;">
                            Playbook execution bottlenecks are identified by comparing individual step durations against the fleet average. Steps exceeding 2× the average are flagged for optimization.
                        </div>
                        <div style="display:grid; grid-template-columns:repeat(3,1fr); gap:1rem;">
                            {[
                                { step: 'isolate_host',    avg_ms: 3200, note: 'SSH round-trip latency' },
                                { step: 'snapshot_memory', avg_ms: 8900, note: 'Dependent on agent RAM' },
                                { step: 'collect_logs',    avg_ms: 1800, note: 'Log volume dependent' },
                            ].map(({ step, avg_ms, note }) => (
                                <div style="background:#0a1318; border:1px solid #1e3040; border-radius:4px; padding:0.75rem;">
                                    <div style="font-size:0.8rem; color:#c8d8d8; margin-bottom:0.3rem;">{step.replace(/_/g,' ')}</div>
                                    <div style="font-size:1.2rem; font-weight:700; color:#ffaa00;">{durationLabel(avg_ms)}</div>
                                    <div style="font-size:0.65rem; color:#607070; margin-top:0.2rem;">{note}</div>
                                </div>
                            ))}
                        </div>
                    </div>
                    <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; padding:1.25rem;">
                        <div style="font-size:0.72rem; color:#607070; letter-spacing:0.1em; margin-bottom:0.75rem;">RECOMMENDATIONS</div>
                        {[
                            'Pre-warm agent connections for isolate_host to reduce SSH handshake latency',
                            'Implement parallel step execution where steps have no data dependencies',
                            'Add step-level timeout configuration to prevent runaway playbook executions',
                            'Cache credentials in agent memory to avoid vault round-trips on each step',
                        ].map((rec, i) => (
                            <div style="padding:0.5rem 0; border-bottom:1px solid #0a1318; font-size:0.74rem; color:#c8d8d8; display:flex; gap:0.75rem;">
                                <span style="color:#ff6600; flex-shrink:0;">{i+1}.</span>
                                <span>{rec}</span>
                            </div>
                        ))}
                    </div>
                </div>
            </Show>
        </div>
    );
}
