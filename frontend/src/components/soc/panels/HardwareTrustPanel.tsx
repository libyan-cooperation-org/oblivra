import { Component, For, Show, createResource, onMount, onCleanup } from 'solid-js';
import { GetAggregatedStatus, GetTrustDriftMetrics } from '../../../../wailsjs/go/services/RuntimeTrustService';

const statusColor = (s: string) => {
    if (s === 'TRUSTED')   return 'var(--status-online)';
    if (s === 'WARNING')   return 'var(--alert-medium)';
    if (s === 'UNTRUSTED') return 'var(--alert-critical)';
    return 'var(--text-muted)';
};

const indexColor = (n: number) => n > 90 ? 'var(--status-online)' : n > 70 ? 'var(--alert-medium)' : 'var(--alert-critical)';

export const HardwareTrustPanel: Component = () => {
    const [trustData, { refetch }] = createResource(async () => {
        try {
            const [status, metrics] = await Promise.all([
                GetAggregatedStatus(),
                GetTrustDriftMetrics(),
            ]);
            return { status: status || [], index: metrics?.current_score ?? 0, metrics };
        } catch (e) {
            console.error('Failed to fetch trust status:', e);
            return { status: [], index: 0, metrics: null };
        }
    });

    onMount(() => {
        const interval = setInterval(refetch, 30000);
        onCleanup(() => clearInterval(interval));
    });

    const idx = () => trustData()?.index ?? 0;
    const metrics = () => trustData()?.metrics;

    return (
        <div style={{ display: 'flex', 'flex-direction': 'column', height: '100%', background: 'var(--surface-0)', 'font-family': 'var(--font-mono)', 'font-size': '11px' }}>
            {/* Header */}
            <div style={{ padding: '8px 12px', 'border-bottom': '1px solid var(--border-primary)', display: 'flex', 'justify-content': 'space-between', 'align-items': 'center', background: 'var(--surface-1)', 'flex-shrink': 0 }}>
                <span style={{ color: 'var(--text-muted)', 'font-weight': 800, 'letter-spacing': '2px', 'text-transform': 'uppercase', 'font-size': '10px' }}>Platform Integrity</span>
                <div style={{ display: 'flex', 'align-items': 'center', gap: '12px' }}>
                    <Show when={metrics()?.is_bleeding}>
                        <span style={{ 'font-size': '9px', color: 'var(--alert-critical)', 'font-weight': 800, 'text-transform': 'uppercase' }}>
                            ETTF: {metrics()!.estimated_failure_time}
                        </span>
                    </Show>
                    <div style={{ 'font-size': '9px', color: 'var(--text-muted)' }}>
                        INDEX: <span style={{ 'font-weight': 800, 'font-size': '13px', color: indexColor(idx()) }}>{idx().toFixed(1)}%</span>
                    </div>
                </div>
            </div>

            {/* Index bar */}
            <div style={{ padding: '8px 12px', 'flex-shrink': 0 }}>
                <div style={{ height: '4px', background: 'var(--surface-2)', 'border-radius': '2px', overflow: 'hidden', border: '1px solid var(--border-primary)' }}>
                    <div style={{ height: '100%', width: `${idx()}%`, background: indexColor(idx()), transition: 'width 0.8s ease, background 0.4s' }} />
                </div>
                <Show when={metrics()}>
                    <div style={{ 'margin-top': '4px', 'font-size': '9px', color: 'var(--text-muted)', 'text-align': 'right' }}>
                        Drift: {metrics()!.velocity_per_hour > 0 ? '+' : ''}{metrics()!.velocity_per_hour?.toFixed(2)}/hr
                    </div>
                </Show>
            </div>

            {/* Component list */}
            <div style={{ flex: 1, 'overflow-y': 'auto', padding: '0 8px 8px 8px', display: 'flex', 'flex-direction': 'column', gap: '6px' }}>
                <For each={trustData()?.status || []}>
                    {(item: any) => (
                        <div style={{ padding: '8px 10px', background: 'var(--surface-1)', border: '1px solid var(--border-primary)', 'border-radius': '3px', display: 'flex', 'flex-direction': 'column', gap: '3px' }}>
                            <div style={{ display: 'flex', 'justify-content': 'space-between', 'align-items': 'center' }}>
                                <span style={{ color: 'var(--text-secondary)', 'font-weight': 800, 'text-transform': 'uppercase', 'letter-spacing': '0.5px' }}>{item.component}</span>
                                <span style={{ 'font-size': '9px', 'font-weight': 800, color: statusColor(item.status), 'text-transform': 'uppercase' }}>{item.status}</span>
                            </div>
                            <div style={{ 'font-size': '9px', color: 'var(--text-muted)', 'font-style': 'italic', 'text-transform': 'uppercase', opacity: '0.7' }}>{item.detail}</div>
                            <div style={{ 'font-size': '8px', color: 'var(--text-muted)', opacity: '0.5', 'margin-top': '2px' }}>
                                Last checked: {new Date(item.last_check).toLocaleTimeString()}
                            </div>
                        </div>
                    )}
                </For>
                <Show when={!trustData.loading && (!trustData()?.status || trustData()!.status.length === 0)}>
                    <div style={{ 'padding-top': '40px', 'text-align': 'center', opacity: '0.25', 'font-size': '10px', 'text-transform': 'uppercase' }}>
                        No attestation data
                    </div>
                </Show>
            </div>

            {/* Footer */}
            <div style={{ padding: '6px 12px', 'border-top': '1px solid var(--border-primary)', background: 'var(--surface-1)', display: 'flex', 'justify-content': 'space-between', 'align-items': 'center', 'flex-shrink': 0 }}>
                <span style={{ 'font-size': '9px', color: 'var(--text-muted)', 'text-transform': 'uppercase' }}>
                    ATTESTATION: <span style={{ color: 'var(--accent-primary)' }}>HARDWARE-BOUND</span>
                </span>
                <button
                    onClick={refetch}
                    style={{ 'font-size': '9px', color: 'var(--accent-primary)', background: 'none', border: 'none', cursor: 'pointer', 'font-weight': 800, 'font-family': 'var(--font-mono)', 'text-transform': 'uppercase' }}
                >
                    ↻ RE-VERIFY
                </button>
            </div>
        </div>
    );
};
