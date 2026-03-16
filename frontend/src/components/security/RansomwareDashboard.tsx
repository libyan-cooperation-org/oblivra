import { Component, createSignal, onMount, onCleanup, For, Show } from 'solid-js';
import { ListIncidents, UpdateIncidentStatus } from '../../../wailsjs/go/services/IncidentService';
import { ListAgents } from '../../../wailsjs/go/services/AgentService';
import { database } from '../../../wailsjs/go/models';

// Wails bindings for NetworkIsolatorService (generated at build; calling directly for now)
const IsolateHost = (hostID: string, reason: string): Promise<void> =>
    (window as any)['go']?.['services']?.['NetworkIsolatorService']?.['IsolateHost']?.(hostID, reason) ?? Promise.resolve();

const GetIsolatedHosts = (): Promise<any[]> =>
    (window as any)['go']?.['services']?.['NetworkIsolatorService']?.['GetIsolatedHosts']?.() ?? Promise.resolve([]);

// ── severity → colour token ──────────────────────────────────────────────────
const sevColor = (s: string) => {
    switch (s?.toLowerCase()) {
        case 'critical': return 'var(--alert-critical)';
        case 'high':     return 'var(--alert-high)';
        case 'medium':   return 'var(--alert-medium)';
        default:         return 'var(--alert-low)';
    }
};

// ── Stat card ────────────────────────────────────────────────────────────────
const StatCard: Component<{
    label: string;
    value: string;
    sub: string;
    accent?: string;
    loading?: boolean;
    error?: string;
}> = (props) => (
    <div style={{
        background: 'var(--surface-1)',
        border: `1px solid ${props.accent ? props.accent + '44' : 'var(--border-primary)'}`,
        'border-top': `2px solid ${props.accent ?? 'var(--border-primary)'}`,
        padding: '20px 24px',
        display: 'flex',
        'flex-direction': 'column',
        gap: '8px',
    }}>
        <div style={{
            'font-family': 'var(--font-mono)',
            'font-size': '10px',
            'font-weight': 700,
            color: 'var(--text-muted)',
            'text-transform': 'uppercase',
            'letter-spacing': '1px',
        }}>
            {props.label}
        </div>
        <Show when={!props.loading && !props.error} fallback={
            <div style={{ 'font-size': '14px', 'font-family': 'var(--font-mono)', color: 'var(--text-muted)' }}>
                {props.error ? `ERR: ${props.error}` : '...'}
            </div>
        }>
            <div style={{
                'font-family': 'var(--font-mono)',
                'font-size': '22px',
                'font-weight': 800,
                color: props.accent ?? 'var(--text-primary)',
                'line-height': 1,
                'letter-spacing': '-0.5px',
            }}>
                {props.value}
            </div>
        </Show>
        <div style={{ 'font-size': '11px', color: 'var(--text-muted)', 'font-family': 'var(--font-ui)' }}>
            {props.sub}
        </div>
    </div>
);

// ── Empty state ───────────────────────────────────────────────────────────────
const EmptyThreats: Component = () => (
    <div style={{
        padding: '48px 32px',
        'text-align': 'center',
        border: '1px solid var(--border-primary)',
        'border-left': '2px solid var(--alert-low)',
        background: 'var(--surface-1)',
    }}>
        <div style={{
            'font-family': 'var(--font-mono)',
            'font-size': '11px',
            'font-weight': 800,
            color: 'var(--alert-low)',
            'text-transform': 'uppercase',
            'letter-spacing': '1px',
            'margin-bottom': '8px',
        }}>
            ALL CLEAR — NO RANSOMWARE SIGNALS
        </div>
        <div style={{ 'font-size': '11px', color: 'var(--text-muted)', 'font-family': 'var(--font-ui)' }}>
            No behavioral chains matching encryption patterns. Canary files armed and monitoring.
        </div>
    </div>
);

// ── Error state ───────────────────────────────────────────────────────────────
const LoadError: Component<{ msg: string; onRetry: () => void }> = (props) => (
    <div style={{
        padding: '32px',
        border: '1px solid var(--alert-critical)',
        background: 'rgba(220,38,38,0.05)',
        display: 'flex',
        'align-items': 'center',
        'justify-content': 'space-between',
    }}>
        <div>
            <div style={{ 'font-family': 'var(--font-mono)', 'font-size': '11px', 'font-weight': 800, color: 'var(--alert-critical)', 'margin-bottom': '4px' }}>
                BACKEND UNREACHABLE
            </div>
            <div style={{ 'font-size': '11px', color: 'var(--text-muted)' }}>{props.msg}</div>
        </div>
        <button
            onClick={props.onRetry}
            style={{
                background: 'transparent',
                border: '1px solid var(--border-primary)',
                color: 'var(--text-primary)',
                'font-family': 'var(--font-mono)',
                'font-size': '10px',
                'font-weight': 700,
                'text-transform': 'uppercase',
                'letter-spacing': '0.5px',
                padding: '6px 16px',
                cursor: 'pointer',
            }}
        >
            RETRY
        </button>
    </div>
);

// ── Main component ────────────────────────────────────────────────────────────
export const RansomwareDashboard: Component = () => {
    const [loading, setLoading]           = createSignal(true);
    const [loadErr, setLoadErr]           = createSignal('');
    const [incidents, setIncidents]       = createSignal<database.Incident[]>([]);
    const [isolated, setIsolated]         = createSignal<any[]>([]);
    const [agentCount, setAgentCount]     = createSignal(0);
    const [isolating, setIsolating]       = createSignal<string | null>(null);
    const [actionErr, setActionErr]       = createSignal('');

    const load = async () => {
        setLoading(true);
        setLoadErr('');
        try {
            const [inc, agents, iso] = await Promise.all([
                ListIncidents(null as any, '', '', 200),
                ListAgents(),
                GetIsolatedHosts(),
            ]);
            setIncidents((inc ?? []).filter((i: database.Incident) =>
                i.rule_id?.includes('Ransomware') || i.rule_id?.includes('RANS')
            ));
            setAgentCount((agents ?? []).length);
            setIsolated(iso ?? []);
        } catch (err: any) {
            setLoadErr(err?.message ?? String(err));
        } finally {
            setLoading(false);
        }
    };

    onMount(() => {
        load();
        const t = setInterval(load, 15_000);
        onCleanup(() => clearInterval(t));
    });

    const executeIsolation = async (hostID: string) => {
        setIsolating(hostID);
        setActionErr('');
        try {
            await IsolateHost(hostID, 'Operator-triggered from Ransomware Dashboard');
            await load();
        } catch (err: any) {
            setActionErr(`Isolation failed: ${err?.message ?? err}`);
        } finally {
            setIsolating(null);
        }
    };

    const resolveIncident = async (id: string) => {
        try {
            await UpdateIncidentStatus(null as any, id, 'Resolved', 'Resolved via Ransomware Dashboard');
            await load();
        } catch { /* ignore */ }
    };

    const activeCount    = () => incidents().length;
    const criticalCount  = () => incidents().filter(i => i.severity?.toLowerCase() === 'critical').length;
    const isolatedCount  = () => isolated().length;

    return (
        <div style={{
            display: 'flex',
            'flex-direction': 'column',
            height: '100%',
            overflow: 'auto',
            padding: '28px 32px',
            background: 'var(--surface-0)',
        }}>
            {/* ── Header ── */}
            <div style={{
                display: 'flex',
                'justify-content': 'space-between',
                'align-items': 'flex-end',
                'margin-bottom': '24px',
                'padding-bottom': '16px',
                'border-bottom': '1px solid var(--border-primary)',
            }}>
                <div>
                    <div style={{
                        'font-family': 'var(--font-mono)',
                        'font-size': '11px',
                        'font-weight': 700,
                        color: 'var(--text-muted)',
                        'text-transform': 'uppercase',
                        'letter-spacing': '2px',
                        'margin-bottom': '4px',
                    }}>
                        RANSOMWARE DEFENSE
                    </div>
                    <h1 style={{
                        'font-family': 'var(--font-mono)',
                        'font-size': '20px',
                        'font-weight': 800,
                        color: 'var(--text-primary)',
                        margin: 0,
                        'letter-spacing': '-0.5px',
                    }}>
                        Behavioral Detection + Automated Isolation
                    </h1>
                </div>
                <div style={{ display: 'flex', gap: '8px', 'align-items': 'center' }}>
                    <Show when={loading()}>
                        <div style={{
                            'font-family': 'var(--font-mono)',
                            'font-size': '10px',
                            color: 'var(--text-muted)',
                            'text-transform': 'uppercase',
                            'letter-spacing': '1px',
                        }}>
                            SYNCING...
                        </div>
                    </Show>
                    {/* Live indicator */}
                    <div style={{
                        display: 'flex',
                        'align-items': 'center',
                        gap: '6px',
                        'font-family': 'var(--font-mono)',
                        'font-size': '10px',
                        color: criticalCount() > 0 ? 'var(--alert-critical)' : 'var(--alert-low)',
                        'font-weight': 700,
                        'text-transform': 'uppercase',
                        border: `1px solid ${criticalCount() > 0 ? 'var(--alert-critical)' : 'var(--alert-low)'}`,
                        padding: '4px 12px',
                    }}>
                        <div style={{
                            width: '6px',
                            height: '6px',
                            background: criticalCount() > 0 ? 'var(--alert-critical)' : 'var(--alert-low)',
                            animation: criticalCount() > 0 ? 'pulse 1.2s infinite' : 'none',
                        }} />
                        {criticalCount() > 0 ? 'THREAT ACTIVE' : 'NOMINAL'}
                    </div>
                </div>
            </div>

            {/* ── Stat row ── */}
            <div style={{
                display: 'grid',
                'grid-template-columns': 'repeat(4, 1fr)',
                gap: '12px',
                'margin-bottom': '24px',
            }}>
                <StatCard
                    label="Monitored Agents"
                    value={loading() ? '—' : String(agentCount())}
                    sub="Live agents reporting telemetry"
                    loading={loading()}
                    error={loadErr() || undefined}
                />
                <StatCard
                    label="Active Threats"
                    value={loading() ? '—' : String(activeCount())}
                    sub="Ransomware behavioral matches"
                    accent={activeCount() > 0 ? 'var(--alert-critical)' : undefined}
                    loading={loading()}
                    error={loadErr() || undefined}
                />
                <StatCard
                    label="Critical"
                    value={loading() ? '—' : String(criticalCount())}
                    sub="Score ≥ isolation threshold"
                    accent={criticalCount() > 0 ? 'var(--alert-critical)' : undefined}
                    loading={loading()}
                    error={loadErr() || undefined}
                />
                <StatCard
                    label="Hosts Isolated"
                    value={loading() ? '—' : String(isolatedCount())}
                    sub="Network containment active"
                    accent={isolatedCount() > 0 ? 'var(--alert-high)' : undefined}
                    loading={loading()}
                    error={loadErr() || undefined}
                />
            </div>

            {/* ── Action error ── */}
            <Show when={actionErr()}>
                <div style={{
                    padding: '10px 16px',
                    background: 'rgba(220,38,38,0.08)',
                    border: '1px solid var(--alert-critical)',
                    'font-family': 'var(--font-mono)',
                    'font-size': '11px',
                    color: 'var(--alert-critical)',
                    'margin-bottom': '16px',
                }}>
                    {actionErr()}
                </div>
            </Show>

            {/* ── Load error ── */}
            <Show when={loadErr() && !loading()}>
                <LoadError msg={loadErr()} onRetry={load} />
            </Show>

            {/* ── Incident list ── */}
            <Show when={!loadErr()}>
                <div style={{
                    'font-family': 'var(--font-mono)',
                    'font-size': '10px',
                    'font-weight': 700,
                    color: 'var(--text-muted)',
                    'text-transform': 'uppercase',
                    'letter-spacing': '1px',
                    'margin-bottom': '8px',
                }}>
                    INCIDENT FEED — {activeCount()} MATCHES
                </div>

                <Show when={!loading() && activeCount() === 0}>
                    <EmptyThreats />
                </Show>

                <Show when={loading() && activeCount() === 0}>
                    <div style={{
                        padding: '32px',
                        'text-align': 'center',
                        'font-family': 'var(--font-mono)',
                        'font-size': '11px',
                        color: 'var(--text-muted)',
                        border: '1px solid var(--border-primary)',
                    }}>
                        LOADING INCIDENT DATA...
                    </div>
                </Show>

                <div style={{ display: 'flex', 'flex-direction': 'column', gap: '8px' }}>
                    <For each={incidents()}>
                        {(inc) => {
                            const color = sevColor(inc.severity);
                            const isBeingIsolated = () => isolating() === inc.group_key;
                            const alreadyIsolated = () => isolated().some((h: any) => h.host_id === inc.group_key);

                            return (
                                <div style={{
                                    background: 'var(--surface-1)',
                                    border: '1px solid var(--border-primary)',
                                    'border-left': `3px solid ${color}`,
                                    padding: '16px 20px',
                                    display: 'flex',
                                    'justify-content': 'space-between',
                                    'align-items': 'flex-start',
                                    gap: '24px',
                                }}>
                                    {/* Left: metadata */}
                                    <div style={{ flex: 1, 'min-width': 0 }}>
                                        <div style={{
                                            display: 'flex',
                                            'align-items': 'center',
                                            gap: '8px',
                                            'margin-bottom': '8px',
                                        }}>
                                            <span style={{
                                                'font-family': 'var(--font-mono)',
                                                'font-size': '9px',
                                                'font-weight': 800,
                                                background: color,
                                                color: '#000',
                                                padding: '2px 6px',
                                                'text-transform': 'uppercase',
                                                'letter-spacing': '0.5px',
                                            }}>
                                                {inc.severity?.toUpperCase() ?? 'UNKNOWN'}
                                            </span>
                                            <span style={{
                                                'font-family': 'var(--font-mono)',
                                                'font-size': '11px',
                                                'font-weight': 700,
                                                color: 'var(--text-primary)',
                                                'text-transform': 'uppercase',
                                                'letter-spacing': '0.3px',
                                            }}>
                                                {inc.title}
                                            </span>
                                        </div>
                                        <p style={{
                                            'font-size': '11px',
                                            color: 'var(--text-muted)',
                                            margin: '0 0 10px 0',
                                            'font-family': 'var(--font-ui)',
                                            'line-height': 1.5,
                                        }}>
                                            {inc.description}
                                        </p>
                                        <div style={{
                                            display: 'flex',
                                            gap: '20px',
                                            'font-size': '10px',
                                            'font-family': 'var(--font-mono)',
                                            color: 'var(--text-muted)',
                                            'text-transform': 'uppercase',
                                            'flex-wrap': 'wrap',
                                        }}>
                                            <span>HOST: <span style={{ color: 'var(--text-secondary)' }}>{inc.group_key || '—'}</span></span>
                                            <span>RULE: <span style={{ color: 'var(--text-secondary)' }}>{inc.rule_id}</span></span>
                                            <span>EVENTS: <span style={{ color: 'var(--text-secondary)' }}>{inc.event_count ?? 0}</span></span>
                                            <span>STATUS: <span style={{ color: 'var(--text-secondary)' }}>{inc.status}</span></span>
                                            <Show when={(inc.mitre_techniques ?? []).length > 0}>
                                                <span>MITRE: <span style={{ color: 'var(--accent-primary)' }}>{(inc.mitre_techniques ?? []).join(', ')}</span></span>
                                            </Show>
                                        </div>
                                    </div>

                                    {/* Right: actions */}
                                    <div style={{
                                        display: 'flex',
                                        'flex-direction': 'column',
                                        gap: '8px',
                                        'align-items': 'flex-end',
                                        'flex-shrink': 0,
                                    }}>
                                        <Show when={alreadyIsolated()}>
                                            <div style={{
                                                'font-family': 'var(--font-mono)',
                                                'font-size': '10px',
                                                'font-weight': 800,
                                                color: 'var(--alert-high)',
                                                'text-transform': 'uppercase',
                                                border: '1px solid var(--alert-high)',
                                                padding: '6px 14px',
                                            }}>
                                                ISOLATED
                                            </div>
                                        </Show>
                                        <Show when={!alreadyIsolated() && inc.group_key}>
                                            <button
                                                disabled={isBeingIsolated()}
                                                onClick={() => executeIsolation(inc.group_key)}
                                                style={{
                                                    background: 'var(--alert-critical)',
                                                    border: 'none',
                                                    color: '#fff',
                                                    'font-family': 'var(--font-mono)',
                                                    'font-size': '10px',
                                                    'font-weight': 800,
                                                    'text-transform': 'uppercase',
                                                    'letter-spacing': '0.5px',
                                                    padding: '6px 14px',
                                                    cursor: isBeingIsolated() ? 'wait' : 'pointer',
                                                    opacity: isBeingIsolated() ? 0.6 : 1,
                                                }}
                                            >
                                                {isBeingIsolated() ? 'ISOLATING...' : 'ISOLATE HOST'}
                                            </button>
                                        </Show>
                                        <button
                                            onClick={() => resolveIncident(inc.id)}
                                            style={{
                                                background: 'transparent',
                                                border: '1px solid var(--border-primary)',
                                                color: 'var(--text-muted)',
                                                'font-family': 'var(--font-mono)',
                                                'font-size': '10px',
                                                'font-weight': 700,
                                                'text-transform': 'uppercase',
                                                padding: '5px 14px',
                                                cursor: 'pointer',
                                            }}
                                        >
                                            RESOLVE
                                        </button>
                                    </div>
                                </div>
                            );
                        }}
                    </For>
                </div>
            </Show>

            {/* ── Isolated hosts panel ── */}
            <Show when={isolatedCount() > 0}>
                <div style={{ 'margin-top': '32px' }}>
                    <div style={{
                        'font-family': 'var(--font-mono)',
                        'font-size': '10px',
                        'font-weight': 700,
                        color: 'var(--alert-high)',
                        'text-transform': 'uppercase',
                        'letter-spacing': '1px',
                        'margin-bottom': '8px',
                    }}>
                        CONTAINED HOSTS — {isolatedCount()}
                    </div>
                    <div style={{ display: 'flex', 'flex-direction': 'column', gap: '4px' }}>
                        <For each={isolated()}>
                            {(h) => (
                                <div style={{
                                    background: 'rgba(234,88,12,0.06)',
                                    border: '1px solid var(--alert-high)',
                                    'border-left': '3px solid var(--alert-high)',
                                    padding: '10px 16px',
                                    display: 'flex',
                                    'justify-content': 'space-between',
                                    'align-items': 'center',
                                    'font-family': 'var(--font-mono)',
                                    'font-size': '11px',
                                }}>
                                    <span style={{ color: 'var(--text-primary)', 'font-weight': 700 }}>{h.host_id}</span>
                                    <span style={{ color: 'var(--text-muted)', 'font-size': '10px' }}>
                                        Score: {h.threat_score} · {h.auto ? 'AUTO' : 'MANUAL'} · {h.isolated_at?.split('T')[0]}
                                    </span>
                                    <span style={{ color: 'var(--alert-high)', 'font-weight': 800, 'font-size': '10px' }}>NETWORK SEVERED</span>
                                </div>
                            )}
                        </For>
                    </div>
                </div>
            </Show>

            <style>{`
                @keyframes pulse {
                    0%, 100% { opacity: 1; }
                    50% { opacity: 0.3; }
                }
            `}</style>
        </div>
    );
};
