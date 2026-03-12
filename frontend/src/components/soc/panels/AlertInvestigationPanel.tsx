import { Component, For, Show, createResource, onMount, onCleanup } from 'solid-js';
import { GetAlertHistory } from '../../../../wailsjs/go/app/AlertingService';
import { ConnectToSession } from '../../../../wailsjs/go/app/SSHService';
import { useToast } from '../../../core/toast';

export const AlertInvestigationPanel: Component = () => {
    const { addToast } = useToast();
    const [alerts, { refetch }] = createResource(async () => {
        try {
            const data = await GetAlertHistory();
            return (data || []).map((item: any) => ({
                id: item.ID || Math.random(),
                title: item.Title || item.Name || 'Unknown Alert',
                host: item.Hostname || item.Host || 'N/A',
                user: item.User || 'N/A',
                risk: item.RiskScore || item.Severity || 50,
                time: item.Timestamp ? new Date(item.Timestamp).toLocaleTimeString() : 'N/A',
            }));
        } catch (e) {
            console.error('Failed to fetch alerts:', e);
            return [];
        }
    });

    onMount(() => {
        const interval = setInterval(refetch, 10000);
        onCleanup(() => clearInterval(interval));
    });

    const riskColor = (r: number) =>
        r > 80 ? 'var(--alert-critical)' : r > 50 ? 'var(--alert-medium)' : 'var(--alert-low)';

    return (
        <div style={{ display: 'flex', 'flex-direction': 'column', height: '100%', background: 'var(--surface-0)', 'font-family': 'var(--font-mono)', 'font-size': '11px' }}>
            {/* Header */}
            <div style={{ padding: '8px 12px', 'border-bottom': '1px solid var(--border-primary)', display: 'flex', 'align-items': 'center', 'justify-content': 'space-between', background: 'var(--surface-1)', 'flex-shrink': 0 }}>
                <div style={{ display: 'flex', 'align-items': 'center', gap: '8px' }}>
                    <span style={{ color: 'var(--text-muted)', 'font-weight': 800, 'letter-spacing': '2px', 'text-transform': 'uppercase', 'font-size': '10px' }}>Active Ingress Alerts</span>
                    <span style={{ 'font-size': '8px', padding: '1px 5px', background: 'rgba(240,64,64,0.12)', color: 'var(--alert-critical)', border: '1px solid rgba(240,64,64,0.3)', 'font-weight': 800 }}>LIVE</span>
                </div>
                <Show when={alerts.loading}>
                    <span style={{ 'font-size': '9px', color: 'var(--accent-primary)' }}>REFRESHING…</span>
                </Show>
            </div>

            {/* List */}
            <div style={{ flex: 1, 'overflow-y': 'auto', padding: '8px', display: 'flex', 'flex-direction': 'column', gap: '6px' }}>
                <For each={alerts() || []}>
                    {(item) => (
                        <div style={{
                            padding: '10px 12px',
                            border: '1px solid var(--border-primary)',
                            background: 'var(--surface-1)',
                            'border-radius': '3px',
                            position: 'relative',
                            overflow: 'hidden',
                        }}>
                            {/* Left risk accent */}
                            <div style={{ position: 'absolute', left: 0, top: 0, bottom: 0, width: '3px', background: riskColor(item.risk) }} />

                            <div style={{ display: 'flex', 'justify-content': 'space-between', 'margin-bottom': '4px', 'padding-left': '8px' }}>
                                <span style={{ color: riskColor(item.risk), 'font-weight': 800, 'font-size': '10px', 'text-transform': 'uppercase', overflow: 'hidden', 'white-space': 'nowrap', 'text-overflow': 'ellipsis', 'max-width': '70%' }}>{item.title}</span>
                                <span style={{ color: 'var(--text-muted)', 'font-size': '9px' }}>{item.time}</span>
                            </div>

                            <div style={{ display: 'flex', gap: '16px', color: 'var(--text-muted)', 'font-size': '9px', 'padding-left': '8px', 'margin-bottom': '8px' }}>
                                <span>HOST: <span style={{ color: 'var(--text-secondary)' }}>{item.host}</span></span>
                                <span>USER: <span style={{ color: 'var(--text-secondary)' }}>{item.user}</span></span>
                            </div>

                            {/* Risk bar */}
                            <div style={{ display: 'flex', 'align-items': 'center', gap: '8px', 'padding-left': '8px' }}>
                                <div style={{ flex: 1, height: '3px', background: 'var(--surface-3)', 'border-radius': '2px', overflow: 'hidden' }}>
                                    <div style={{ height: '100%', width: `${item.risk}%`, background: riskColor(item.risk), transition: 'width 0.4s' }} />
                                </div>
                                <span style={{ 'font-weight': 800, 'font-size': '10px', color: riskColor(item.risk), 'min-width': '24px', 'text-align': 'right' }}>{item.risk}</span>
                            </div>

                            {/* Pivot button */}
                            <div style={{ display: 'flex', 'justify-content': 'flex-end', 'margin-top': '8px', 'padding-left': '8px' }}>
                                <button
                                    style={{
                                        'font-size': '8px', background: 'rgba(240,64,64,0.1)',
                                        border: '1px solid rgba(240,64,64,0.35)', padding: '2px 8px',
                                        color: 'var(--alert-critical)', 'font-family': 'var(--font-mono)',
                                        'font-weight': 800, cursor: 'pointer', 'text-transform': 'uppercase',
                                        'letter-spacing': '0.5px', 'border-radius': '2px',
                                        transition: 'all 0.1s',
                                    }}
                                    onMouseOver={(e) => { (e.currentTarget as HTMLElement).style.background = 'var(--alert-critical)'; (e.currentTarget as HTMLElement).style.color = '#000'; }}
                                    onMouseOut={(e) => { (e.currentTarget as HTMLElement).style.background = 'rgba(240,64,64,0.1)'; (e.currentTarget as HTMLElement).style.color = 'var(--alert-critical)'; }}
                                    onClick={async (e) => {
                                        e.stopPropagation();
                                        try {
                                            await ConnectToSession(item.host, 'soc-session');
                                            addToast({ type: 'info', title: 'Tactical Pivot', message: `Establishing SSH to ${item.host}…` });
                                        } catch (err) {
                                            addToast({ type: 'error', title: 'Pivot Failed', message: `Could not connect to ${item.host}: ${err}` });
                                        }
                                    }}
                                >
                                    ⇒ PIVOT TO TERMINAL
                                </button>
                            </div>
                        </div>
                    )}
                </For>

                <Show when={!alerts.loading && (!alerts() || alerts()!.length === 0)}>
                    <div style={{ flex: 1, display: 'flex', 'align-items': 'center', 'justify-content': 'center', 'flex-direction': 'column', gap: '8px', opacity: '0.3', 'padding-top': '40px' }}>
                        <div style={{ width: '28px', height: '28px', border: '1px dashed var(--border-secondary)', 'border-radius': '50%' }} />
                        <span style={{ 'font-size': '10px', 'text-transform': 'uppercase', 'letter-spacing': '1px' }}>No active alerts</span>
                    </div>
                </Show>
            </div>

            {/* Footer */}
            <div style={{ padding: '6px 12px', 'border-top': '1px solid var(--border-primary)', background: 'var(--surface-1)', display: 'flex', 'justify-content': 'space-between', 'align-items': 'center', 'flex-shrink': 0 }}>
                <span style={{ 'font-size': '9px', color: 'var(--text-muted)', 'text-transform': 'uppercase' }}>TOTAL: {alerts()?.length ?? 0}</span>
                <button
                    onClick={refetch}
                    style={{ 'font-size': '9px', color: 'var(--accent-primary)', 'font-weight': 800, background: 'none', border: 'none', cursor: 'pointer', 'text-transform': 'uppercase', 'font-family': 'var(--font-mono)' }}
                >
                    ↻ REFRESH
                </button>
            </div>
        </div>
    );
};
