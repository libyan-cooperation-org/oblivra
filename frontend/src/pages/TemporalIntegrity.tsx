import { Component, createSignal, For, Show, onMount, onCleanup } from 'solid-js';
import { Violation } from '../wails';

const typeColor = (t: string) => {
    switch (t) {
        case 'clock_drift':   return 'var(--alert-medium)';
        case 'future_event':  return 'var(--alert-critical)';
        case 'stale_event':   return 'var(--accent-primary)';
        default:              return 'var(--text-muted)';
    }
};

const driftColor = (ms: number) => {
    const abs = Math.abs(ms);
    if (abs > 5000) return 'var(--alert-critical)';
    if (abs > 2000) return 'var(--alert-medium)';
    return 'var(--alert-low)';
};

export const TemporalIntegrity: Component = () => {
    const [violations, setViolations] = createSignal<Violation[]>([]);
    const [agentDrift, setAgentDrift] = createSignal<Record<string, number>>({});
    const [loading, setLoading]       = createSignal(true);
    const [loadErr, setLoadErr]       = createSignal('');

    const svc = () => (window as any).go?.app?.TemporalService;

    const loadData = async () => {
        setLoadErr('');
        try {
            const s = svc();
            if (!s) { setLoadErr('TemporalService not available'); return; }
            const [vios, drift] = await Promise.all([
                s.GetViolations(),
                s.GetAgentDrift(),
            ]);
            setViolations(vios || []);
            setAgentDrift(drift || {});
        } catch (err: any) {
            setLoadErr(err?.message ?? String(err));
        } finally {
            setLoading(false);
        }
    };

    onMount(() => {
        loadData();
        const t = setInterval(loadData, 3000);
        onCleanup(() => clearInterval(t));
    });

    const driftEntries = () => Object.entries(agentDrift());

    return (
        <div style={{
            display: 'flex',
            'flex-direction': 'column',
            height: '100%',
            overflow: 'hidden',
            background: 'var(--surface-0)',
            padding: '28px 32px 0 32px',
        }}>
            {/* ── Header ── */}
            <div style={{
                display: 'flex',
                'justify-content': 'space-between',
                'align-items': 'flex-end',
                'margin-bottom': '20px',
                'padding-bottom': '16px',
                'border-bottom': '1px solid var(--border-primary)',
                'flex-shrink': 0,
            }}>
                <div>
                    <div style={{ 'font-family': 'var(--font-mono)', 'font-size': '10px', 'font-weight': 700, color: 'var(--text-muted)', 'text-transform': 'uppercase', 'letter-spacing': '2px', 'margin-bottom': '4px' }}>AUDIT TRAIL</div>
                    <h1 style={{ 'font-family': 'var(--font-mono)', 'font-size': '20px', 'font-weight': 800, color: 'var(--text-primary)', margin: 0, 'letter-spacing': '-0.5px' }}>
                        Temporal Integrity
                    </h1>
                </div>
                <div style={{ display: 'flex', gap: '16px', 'font-family': 'var(--font-mono)', 'font-size': '10px', color: 'var(--text-muted)', 'align-items': 'center' }}>
                    <span style={{ border: '1px solid var(--border-primary)', padding: '4px 10px' }}>FUTURE SKEW: 1H</span>
                    <span style={{ border: '1px solid var(--border-primary)', padding: '4px 10px' }}>MAX AGE: 30D</span>
                    <span style={{ border: '1px solid var(--border-primary)', padding: '4px 10px' }}>DRIFT ALERT: 5000MS</span>
                </div>
            </div>

            {/* ── Load error ── */}
            <Show when={loadErr()}>
                <div style={{ padding: '10px 16px', 'margin-bottom': '16px', border: '1px solid var(--alert-critical)', 'font-family': 'var(--font-mono)', 'font-size': '11px', color: 'var(--alert-critical)', 'flex-shrink': 0 }}>
                    {loadErr()}
                </div>
            </Show>

            {/* ── Grid ── */}
            <div style={{ display: 'grid', 'grid-template-columns': '1fr 1fr', gap: '16px', flex: 1, 'min-height': 0, 'padding-bottom': '28px' }}>

                {/* Clock drift */}
                <div style={{ background: 'var(--surface-1)', border: '1px solid var(--border-primary)', display: 'flex', 'flex-direction': 'column', overflow: 'hidden' }}>
                    <div style={{ padding: '12px 16px', 'border-bottom': '1px solid var(--border-primary)', 'font-family': 'var(--font-mono)', 'font-size': '9px', 'font-weight': 800, color: 'var(--text-muted)', 'text-transform': 'uppercase', 'letter-spacing': '1.5px', 'flex-shrink': 0 }}>
                        FLEET CLOCK DRIFT — {driftEntries().length} HOSTS
                    </div>
                    <div style={{ flex: 1, 'overflow-y': 'auto', padding: '12px 16px', display: 'flex', 'flex-direction': 'column', gap: '10px' }}>
                        <Show when={loading()}>
                            <div style={{ 'font-family': 'var(--font-mono)', 'font-size': '11px', color: 'var(--text-muted)', padding: '16px 0' }}>LOADING...</div>
                        </Show>
                        <Show when={!loading() && driftEntries().length === 0}>
                            <div style={{ 'font-family': 'var(--font-mono)', 'font-size': '11px', color: 'var(--text-muted)', 'text-transform': 'uppercase', padding: '24px 0', 'text-align': 'center' }}>
                                NO DRIFT DATA
                            </div>
                        </Show>
                        <For each={driftEntries()}>
                            {([host, drift]) => {
                                const color = driftColor(drift);
                                const barWidth = Math.min(Math.abs(drift) / 100, 100);
                                return (
                                    <div style={{ display: 'grid', 'grid-template-columns': '140px 1fr 72px', gap: '0 12px', 'align-items': 'center' }}>
                                        <span style={{ 'font-family': 'var(--font-mono)', 'font-size': '10px', color: 'var(--text-secondary)', overflow: 'hidden', 'text-overflow': 'ellipsis', 'white-space': 'nowrap' }}>
                                            {host}
                                        </span>
                                        <div style={{ height: '4px', background: 'var(--surface-2)', overflow: 'hidden' }}>
                                            <div style={{ height: '100%', width: `${barWidth}%`, background: color, 'min-width': '2px', 'margin-left': drift < 0 ? 'auto' : '0', transition: 'width 0.4s' }} />
                                        </div>
                                        <span style={{ 'font-family': 'var(--font-mono)', 'font-size': '10px', 'font-weight': 800, color, 'text-align': 'right' }}>
                                            {drift > 0 ? '+' : ''}{drift}ms
                                        </span>
                                    </div>
                                );
                            }}
                        </For>
                    </div>
                </div>

                {/* Violations */}
                <div style={{ background: 'var(--surface-1)', border: '1px solid var(--border-primary)', display: 'flex', 'flex-direction': 'column', overflow: 'hidden' }}>
                    <div style={{ padding: '12px 16px', 'border-bottom': '1px solid var(--border-primary)', display: 'flex', 'justify-content': 'space-between', 'align-items': 'center', 'font-family': 'var(--font-mono)', 'font-size': '9px', 'font-weight': 800, color: 'var(--text-muted)', 'text-transform': 'uppercase', 'letter-spacing': '1.5px', 'flex-shrink': 0 }}>
                        <span>TEMPORAL VIOLATIONS</span>
                        <span style={{ color: violations().length > 0 ? 'var(--alert-critical)' : 'var(--alert-low)' }}>{violations().length} ACTIVE</span>
                    </div>
                    <div style={{ flex: 1, 'overflow-y': 'auto', padding: '8px', display: 'flex', 'flex-direction': 'column', gap: '4px' }}>
                        <Show when={!loading() && violations().length === 0}>
                            <div style={{ padding: '32px', 'text-align': 'center', 'font-family': 'var(--font-mono)', 'font-size': '11px', color: 'var(--alert-low)', 'text-transform': 'uppercase', 'letter-spacing': '0.5px' }}>
                                ALL CLEAR — NO VIOLATIONS
                            </div>
                        </Show>
                        <For each={violations()}>
                            {(v) => {
                                const color = typeColor(v.type);
                                return (
                                    <div style={{
                                        padding: '10px 12px',
                                        background: 'var(--surface-0)',
                                        border: '1px solid var(--border-primary)',
                                        'border-left': `3px solid ${color}`,
                                    }}>
                                        <div style={{ display: 'flex', 'justify-content': 'space-between', 'margin-bottom': '4px' }}>
                                            <span style={{ 'font-family': 'var(--font-mono)', 'font-size': '9px', 'font-weight': 800, color, 'text-transform': 'uppercase', 'letter-spacing': '0.5px' }}>
                                                {v.type.replace('_', ' ')}
                                            </span>
                                            <span style={{ 'font-family': 'var(--font-mono)', 'font-size': '9px', color: 'var(--text-muted)' }}>
                                                {new Date(v.timestamp).toLocaleTimeString()}
                                            </span>
                                        </div>
                                        <div style={{ 'font-family': 'var(--font-mono)', 'font-size': '10px', color: 'var(--text-muted)', 'margin-bottom': '4px' }}>
                                            HOST: <span style={{ color: 'var(--text-secondary)' }}>{v.host_id}</span>
                                        </div>
                                        <div style={{ 'font-family': 'var(--font-ui)', 'font-size': '11px', color: 'var(--text-secondary)', 'line-height': 1.4 }}>
                                            {v.detail}
                                        </div>
                                    </div>
                                );
                            }}
                        </For>
                    </div>
                </div>
            </div>
        </div>
    );
};
